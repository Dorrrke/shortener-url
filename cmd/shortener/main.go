package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/Dorrrke/shortener-url/pkg/server"
	"github.com/go-chi/chi/v5"
)

func main() {

	var URLServer server.Server
	URLServer.New()

	flag.Var(&URLServer.ServerConf.HostConfig, "a", "address and port to run server")
	flag.Var(&URLServer.ServerConf.ShortURLHostConfig, "b", "address and port to run short URL")
	flag.Parse()
	URLServer.GetStorage()
	if err := run(URLServer); err != nil {
		panic(err)
	}

}

func run(serv server.Server) error {
	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Post("/", serv.ShortenerURLHandler)
		r.Get("/{id}", serv.GetOriginalURLHandler)
	})

	log.Print(serv.ServerConf.HostConfig.String())
	if serv.ServerConf.HostConfig.Host == "" {
		return http.ListenAndServe(":8080", r)
	} else {
		return http.ListenAndServe(serv.ServerConf.HostConfig.String(), r)
	}
}
