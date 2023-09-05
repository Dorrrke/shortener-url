package main

import (
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

var urlMap map[string]string

func shortenerURL(res http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, err.Error(), 500)
			return
		}
		urlID := strings.Split(uuid.New().String(), "-")[0]
		urlMap[urlID] = string(body)
		resultURL := "http://localhost:8080/" + urlID
		res.Header().Set("content-type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(resultURL))
		return
	} else {
		if req.Method == http.MethodGet {
			urlID := strings.Split(req.URL.String(), "/")[1]
			if _, ok := urlMap[urlID]; ok {
				url := urlMap[urlID]
				res.Header().Add("Location", url)
				res.WriteHeader(http.StatusTemporaryRedirect)
			}
			return
		} else {
			http.Error(res, "Не корректный запрос", http.StatusBadRequest)
		}
	}
}

func main() {

	urlMap = make(map[string]string)

	mux := http.NewServeMux()
	mux.HandleFunc(`/`, shortenerURL)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
