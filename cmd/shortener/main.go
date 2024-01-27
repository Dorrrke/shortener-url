// Модуль main - основная точка входа в систему.
// В пакете происходит подключение к базе данных, если имеется ссылка для подключения, создание storage, и инициализация logger и server.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"net/http/pprof"

	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"

	"github.com/Dorrrke/shortener-url/internal/config"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/pkg/server"
	"github.com/Dorrrke/shortener-url/pkg/storage"
)

// FilePath — константа с названием файла для хранения данных при отсутствии подключения к бд.
const FilePath string = "short-url-db.json"

// Глобальные переменные для вывода при запуске.
var (
	// buildVersion - версия сборки.
	buildVersion string
	// buildDate - дата сборки.
	buildDate string
	// buildCommit - комментарии к сборке.
	buildCommit string
)

// ValueConfig структура хранящая сруктуры для парсинга пременных окуржения по средствам пакета env.
type ValueConfig struct {
	serverCfg     ServerAdrConfig
	URLCfg        BaseURLConfig
	storageRestor StorageRestor
	dataBaseDsn   DataBaseConf
}

// ServerAdrConfig - структура для получения переменной окружения SERVER_ADDRESS.
// SERVER_ADDRESS - переменная окружения хранящая в себе адресс для запуска сервера.
type ServerAdrConfig struct {
	Addr string `env:"SERVER_ADDRESS,required"`
}

// BaseURLConfig - структура для получения переменной окружения BASE_URL.
// BASE_URL - переменная окружения хранящаяя в себе базовый адресс для сокращенных url.
type BaseURLConfig struct {
	Addr string `env:"BASE_URL,required"`
}

// StorageRestor - структура для получения переменной окружения FILE_STORAGE_PATH.
// FILE_STORAGE_PATH - переменная окружения хранящаяя в себе путь к файлу для хранения сокращенных url.
type StorageRestor struct {
	FilePathString string `env:"FILE_STORAGE_PATH,required"`
}

// DataBaseConf - структура для получения переменной окружения DATABASE_DSN.
// DATABASE_DSN переменная окружения хранящаяя в себе адресс базы данных для подключения к ней.
type DataBaseConf struct {
	DBDSN string `env:"DATABASE_DSN,required"`
}

func main() {
	if buildVersion == "" {
		buildVersion = "N/A"
	} else {
		fmt.Printf("Build version: %s\n", buildVersion)
	}

	if buildDate == "" {
		buildDate = "N/A"
	} else {
		fmt.Printf("Build date: %s\n", buildDate)
	}

	if buildCommit == "" {
		buildCommit = "N/A"
	} else {
		fmt.Printf("Build commit: %s\n", buildCommit)
	}

	if err := logger.Initialize(zap.InfoLevel.String()); err != nil {
		panic(err)
	}
	var URLServer server.Server
	var cfg ValueConfig
	var fileName string
	var DBaddr string
	var cfgPath string
	var config config.AppConfig
	var dbConnection pgxpool.Conn

	// TODO: Перенести конфигурацию сервиса в другой пакет
	URLServer.New()
	flag.StringVar(&cfgPath, "config", "", "config file path")
	flag.Var(&URLServer.ServerConf.HostConfig, "a", "address and port to run server")
	flag.Var(&URLServer.ServerConf.ShortURLHostConfig, "b", "address and port to run short URL")
	flag.StringVar(&fileName, "f", "", "storage file path")
	flag.StringVar(&DBaddr, "d", "", "databse addr")
	httpsFlag := flag.Bool("s", false, "use https server")
	flag.Parse()
	URLServer.AddFilePath(fileName)

	if cfgPath != "" {
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			logger.Log.Error("Config file is not exist", zap.Error(err))
			panic(err)
		}
		f, err := os.Open(cfgPath)
		if err != nil {
			logger.Log.Error("Error open file", zap.Error(err))
			panic(err)
		}
		defer f.Close()
		dec := json.NewDecoder(f)
		if err := dec.Decode(&config); err != nil {
			logger.Log.Error("error parse config file")
			panic(err)
		}
		logger.Log.Info("config from json", zap.Any("config", config))
		URLServer.ServerConf.HostConfig.Set(config.ServerAddress)
		URLServer.ServerConf.ShortURLHostConfig.Set(config.BaseURL)
		DBaddr = config.DatabaseDsn
		fileName = config.FileStoragePath
	}

	if *httpsFlag {
		log.Print("https")
	}
	servErr := env.Parse(&cfg.serverCfg)
	if servErr == nil {
		URLServer.ServerConf.HostConfig.Set(cfg.serverCfg.Addr)
	}
	URLErr := env.Parse(&cfg.URLCfg)
	if URLErr == nil {
		URLServer.ServerConf.ShortURLHostConfig.Set(cfg.URLCfg.Addr)
	}
	dbDsnErr := env.Parse(&cfg.dataBaseDsn)
	if dbDsnErr == nil {
		dbConnection := initDB(cfg.dataBaseDsn.DBDSN)
		URLServer.AddStorage(&storage.DBStorage{DB: dbConnection})
		defer dbConnection.Close()
	}
	logger.Log.Info("DataBase URL env: " + cfg.dataBaseDsn.DBDSN)
	logger.Log.Info("DataBase URL flag: " + DBaddr)

	if cfg.dataBaseDsn.DBDSN == "" && DBaddr != "" {
		dbConnection := initDB(DBaddr)
		URLServer.AddStorage(&storage.DBStorage{DB: dbConnection})
		defer dbConnection.Close()
	}

	if cfg.dataBaseDsn.DBDSN == "" && DBaddr == "" {
		URLServer.AddStorage(&storage.MemStorage{URLMap: make(map[string]string)})
	}

	filePathErr := env.Parse(&cfg.storageRestor)
	if filePathErr == nil {
		log.Print("env")
		URLServer.AddFilePath(cfg.storageRestor.FilePathString)
	}
	// if URLServer.GetFilePath() == "" {
	// 	log.Print("default")
	// 	URLServer.AddFilePath(FilePath)
	// }

	if err := URLServer.RestorStorage(); err != nil {
		logger.Log.Error("Error restor storage: ", zap.Error(err))
	}
	server := &http.Server{}
	go run(URLServer, server, httpsFlag, config)
	// if err := run(URLServer, server, httpsFlag, config); err != nil {
	// 	panic(err)
	// }
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	signal := <-stop
	logger.Log.Info("stopping server", zap.String("signal", signal.String()))
	stopService(server, &dbConnection)

}

func run(serv server.Server, serverHTTP *http.Server, httpsFlag *bool, cfg config.AppConfig) error {

	logger.Log.Info("Running server")
	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Post("/", logger.WithLogging(server.GzipMiddleware(serv.ShortenerURLHandler)))
		r.Get("/{id}", logger.WithLogging(server.GzipMiddleware(serv.GetOriginalURLHandler)))
		r.Route("/api", func(r chi.Router) {
			r.Get("/user/urls", logger.WithLogging(server.GzipMiddleware(serv.GetAllUrls)))
			r.Delete("/user/urls", logger.WithLogging(server.GzipMiddleware(serv.DeleteURLHandler)))
			r.Route("/shorten", func(r chi.Router) {
				r.Post("/", logger.WithLogging(server.GzipMiddleware(serv.ShortenerJSONURLHandler)))
				r.Post("/batch", logger.WithLogging(server.GzipMiddleware(serv.InsertBatchHandler)))
			})
		})
		r.Get("/ping", logger.WithLogging(server.GzipMiddleware(serv.CheckDBConnectionHandler)))
	})
	r.HandleFunc("/debug/pprof", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	// Регистрация обработчиков pprof для различных типов профилирования
	r.HandleFunc("/debug/pprof/heap", pprof.Index)
	r.HandleFunc("/debug/pprof/goroutine", pprof.Index)
	r.HandleFunc("/debug/pprof/block", pprof.Index)
	r.HandleFunc("/debug/pprof/threadcreate", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	serverHTTP.Handler = r
	if serv.ServerConf.HostConfig.Host != "" {
		serverHTTP.Addr = serv.ServerConf.HostConfig.String()
	} else {
		serverHTTP.Addr = ":8080"
	}
	if *httpsFlag || cfg.EnableHTTPS {
		manager := &autocert.Manager{
			Cache:      autocert.DirCache("cache-dir"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist("short.ru"),
		}
		serverHTTP.TLSConfig = manager.TLSConfig()
		return serverHTTP.ListenAndServeTLS("", "")
	}
	return serverHTTP.ListenAndServe()
}

func initDB(DBAddr string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), DBAddr)
	if err != nil {
		logger.Log.Error("Error wile init db driver: " + err.Error())
		panic(err)
	}
	return pool

}

func stopService(serverHTTP *http.Server, dbConnection *pgxpool.Conn) {
	err := dbConnection.Conn().Close(context.Background())
	serverHTTP.Shutdown(context.Background())
	logger.Log.Info("Service stop", zap.Error(err))
}
