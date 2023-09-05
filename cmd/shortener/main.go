package main

import (
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

var urlMap map[string]string

func shortenerUrl(res http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, err.Error(), 500)
			return
		}
		urlId := strings.Split(uuid.New().String(), "-")[0]
		urlMap[urlId] = string(body)
		resultUrl := "http://localhost:8080/" + urlId
		res.Header().Set("content-type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(resultUrl))
		return
	} else {
		if req.Method == http.MethodGet {
			urlId := strings.Split(req.URL.String(), "/")[1]
			if _, ok := urlMap[urlId]; ok {
				url := urlMap[urlId]
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
	mux.HandleFunc(`/`, shortenerUrl)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
