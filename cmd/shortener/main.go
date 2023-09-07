package main

import (
	"net/http"

	"github.com/Dorrrke/shortener-url/cmd/storage"
	"github.com/Dorrrke/shortener-url/internal/handlers"
	"github.com/go-chi/chi/v5"
)

func main() {

	storage.MapURL = storage.URLStorage{
		URLMap: make(map[string]string),
	}

	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Post("/", handlers.ShortenerURLHandler)
		r.Get("/{id}", handlers.GetOriginalURLHandler)
	})

	err := http.ListenAndServe(`:8080`, r)
	if err != nil {
		panic(err)
	}
}
