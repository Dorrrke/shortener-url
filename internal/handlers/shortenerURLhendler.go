package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/Dorrrke/shortener-url/cmd/storage"
	"github.com/google/uuid"
)

func ShortenerURLHandler(res http.ResponseWriter, req *http.Request, s storage.URLStorage) {
	if req.Method == http.MethodPost {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, err.Error(), 500)
			return
		}
		urlID := strings.Split(uuid.New().String(), "-")[0]
		s.CreateURL(urlID, string(body))
		resultURL := "http://localhost:8080/" + urlID
		res.Header().Set("content-type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(resultURL))
		return
	} else {
		http.Error(res, "Не корректный запрос", http.StatusBadRequest)
	}
}
