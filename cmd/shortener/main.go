package main

import (
	"flag"
	"net/http"

	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/pkg/server"
	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type ValueConfig struct {
	serverCfg ServerAdrConfig
	URLCfg    BaseURLConfig
}

type ServerAdrConfig struct {
	Addr string `env:"SERVER_ADDRESS,required"`
}
type BaseURLConfig struct {
	Addr string `env:"BASE_URL,required"`
}

func main() {

	var URLServer server.Server
	URLServer.New()
	var cfg ValueConfig

	flag.Var(&URLServer.ServerConf.HostConfig, "a", "address and port to run server")
	flag.Var(&URLServer.ServerConf.ShortURLHostConfig, "b", "address and port to run short URL")
	flag.Parse()

	servErr := env.Parse(&cfg.serverCfg)
	if servErr == nil {
		URLServer.ServerConf.HostConfig.Set(cfg.serverCfg.Addr)
	}
	URLErr := env.Parse(&cfg.URLCfg)
	if URLErr == nil {
		URLServer.ServerConf.ShortURLHostConfig.Set(cfg.URLCfg.Addr)
	}
	if err := run(URLServer); err != nil {
		panic(err)
	}

}

func run(serv server.Server) error {

	if err := logger.Initialize(zap.InfoLevel.String()); err != nil {
		return err
	}

	logger.Log.Info("Running server")
	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Post("/", logger.WithLogging(server.GzipMiddleware(serv.ShortenerURLHandler)))
		r.Get("/{id}", logger.WithLogging(server.GzipMiddleware(serv.GetOriginalURLHandler)))
		r.Post("/api/shorten", logger.WithLogging(server.GzipMiddleware(serv.ShortenerJSONURLHandler)))
	})

	if serv.ServerConf.HostConfig.Host == "" {
		return http.ListenAndServe(":8080", r)
	} else {
		return http.ListenAndServe(serv.ServerConf.HostConfig.String(), r)
	}
}
