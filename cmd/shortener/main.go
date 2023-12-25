// Модуль main - основная точка входа в систему.
// В пакете происходит подключение к базе данных, если имеется ссылка для подключения, создание storage, и инициализация logger и server.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"net/http/pprof"

	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/pkg/server"
	"github.com/Dorrrke/shortener-url/pkg/storage"
	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// FilePath — константа с названием файла для хранения данных при отсутствии подключения к бд.
const FilePath string = "short-url-db.json"

// ValueConfig структура хранящая сруктуры для парсинга пременных окуржения по средствам пакета env.
type ValueConfig struct {
	serverCfg     ServerAdrConfig
	URLCfg        BaseURLConfig
	storageRestor StorageRestor
	dataBaseDsn   DataBaseConf
}

// Структура для получения переменной окружения SERVER_ADDRESS.
// SERVER_ADDRESS - переменная окружения хранящая в себе адресс для запуска сервера.
type ServerAdrConfig struct {
	Addr string `env:"SERVER_ADDRESS,required"`
}

// Структура для получения переменной окружения BASE_URL.
// BASE_URL - переменная окружения хранящаяя в себе базовый адресс для сокращенных url.
type BaseURLConfig struct {
	Addr string `env:"BASE_URL,required"`
}

// Структура для получения переменной окружения FILE_STORAGE_PATH.
// FILE_STORAGE_PATH - переменная окружения хранящаяя в себе путь к файлу для хранения сокращенных url.
type StorageRestor struct {
	FilePathString string `env:"FILE_STORAGE_PATH,required"`
}

// Структура для получения переменной окружения DATABASE_DSN.
// DATABASE_DSN переменная окружения хранящаяя в себе адресс базы данных для подключения к ней.
type DataBaseConf struct {
	DBDSN string `env:"DATABASE_DSN,required"`
}

func main() {

	if err := logger.Initialize(zap.InfoLevel.String()); err != nil {
		panic(err)
	}
	var URLServer server.Server
	var cfg ValueConfig
	var fileName string
	var DBaddr string

	URLServer.New()

	flag.Var(&URLServer.ServerConf.HostConfig, "a", "address and port to run server")
	flag.Var(&URLServer.ServerConf.ShortURLHostConfig, "b", "address and port to run short URL")
	flag.StringVar(&fileName, "f", "", "storage file path")
	flag.StringVar(&DBaddr, "d", "", "databse addr")
	flag.Parse()
	URLServer.AddFilePath(fileName)

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
		conn := initDB(cfg.dataBaseDsn.DBDSN)
		URLServer.AddStorage(&storage.DBStorage{DB: conn})
		defer conn.Close()
	}
	logger.Log.Info("DataBase URL env: " + cfg.dataBaseDsn.DBDSN)
	logger.Log.Info("DataBase URL flag: " + DBaddr)

	if cfg.dataBaseDsn.DBDSN == "" && DBaddr != "" {
		conn := initDB(DBaddr)
		URLServer.AddStorage(&storage.DBStorage{DB: conn})
		defer conn.Close()
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
	if err := run(URLServer); err != nil {
		panic(err)
	}

}

func run(serv server.Server) error {

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

	if serv.ServerConf.HostConfig.Host == "" {
		return http.ListenAndServe(":8080", r)
	} else {
		return http.ListenAndServe(serv.ServerConf.HostConfig.String(), r)
	}
}

func initDB(DBAddr string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), DBAddr)
	if err != nil {
		logger.Log.Error("Error wile init db driver: " + err.Error())
		panic(err)
	}
	return pool

}
