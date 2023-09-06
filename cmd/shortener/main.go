package main

import (
	"net/http"

	"github.com/Dorrrke/shortener-url/cmd/storage"
	"github.com/Dorrrke/shortener-url/internal/handlers"
)

var stor storage.URLStorage

func handleRoot(res http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost {
		handlers.ShortenerURLHandler(res, req, stor)
	}
	if req.Method == http.MethodGet {
		handlers.GetOriginalURLHandler(res, req, stor)
	}
}

func main() {

	stor = storage.URLStorage{
		URLMap: make(map[string]string),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(`/`, handleRoot)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
