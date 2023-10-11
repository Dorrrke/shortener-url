package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/pkg/server"
	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi/v5"
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
	URLServer.New()
	var cfg ValueConfig
	var fileName string
	var DBaddr string

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
		if err := URLServer.InitBD(cfg.dataBaseDsn.DBDSN); err != nil {
			log.Printf("Error wile init db driver: %v", err.Error())
			URLServer.AddDB(nil)
		}
	}
	logger.Log.Info("DataBase URL env: " + cfg.dataBaseDsn.DBDSN)
	logger.Log.Info("DataBase URL flag: " + DBaddr)

	if cfg.dataBaseDsn.DBDSN == "" {
		if err := URLServer.InitBD(DBaddr); err != nil {
			logger.Log.Error("Error wile init db driver: " + err.Error())
			URLServer.AddDB(nil)
		}
	}

	filePathErr := env.Parse(&cfg.storageRestor)
	if filePathErr == nil {
		log.Print("env")
		URLServer.AddFilePath(cfg.storageRestor.FilePathString)
	}
	if URLServer.GetFilePath() == "" {
		log.Print("default")
		URLServer.AddFilePath(FilePath)
	}

	URLServer.RestorStorage()
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
		r.Post("/api/shorten", logger.WithLogging(server.GzipMiddleware(serv.ShortenerJSONURLHandler)))
		r.Get("/ping", logger.WithLogging(server.GzipMiddleware(serv.CheckDBConnectionHandler)))
	})

	if serv.ServerConf.HostConfig.Host == "" {
		return http.ListenAndServe(":8080", r)
	} else {
		return http.ListenAndServe(serv.ServerConf.HostConfig.String(), r)
	}
}
