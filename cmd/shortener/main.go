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
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const FilePath string = "short-url-db.json" // Вынести эту константу в конфиг

type ValueConfig struct {
	serverCfg     ServerAdrConfig
	URLCfg        BaseURLConfig
	storageRestor StorageRestor
	dataBaseDsn   DataBaseConf
}

type ServerAdrConfig struct {
	Addr string `env:"SERVER_ADDRESS,required"`
}
type BaseURLConfig struct {
	Addr string `env:"BASE_URL,required"`
}
type StorageRestor struct {
	FilePathString string `env:"FILE_STORAGE_PATH,required"`
}
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
		r.Use(middleware.Compress(5, "gzip"))
		r.Post("/", logger.WithLogging(serv.ShortenerURLHandler))
		r.Get("/{id}", logger.WithLogging(serv.GetOriginalURLHandler))
		r.Route("/api", func(r chi.Router) {
			r.Get("/user/urls", logger.WithLogging(serv.GetAllUrls))
			r.Delete("/user/urls", logger.WithLogging(serv.DeleteURLHandler))
			r.Route("/shorten", func(r chi.Router) {
				r.Post("/", logger.WithLogging(serv.ShortenerJSONURLHandler))
				r.Post("/batch", logger.WithLogging(serv.InsertBatchHandler))
			})
		})
		r.Get("/ping", logger.WithLogging(serv.CheckDBConnectionHandler))
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
