package handlers

import (
	"net/http"
	"strings"

	"github.com/Dorrrke/shortener-url/cmd/storage"
)

func GetOriginalURLHandler(res http.ResponseWriter, req *http.Request, s storage.URLStorage) {
	if req.Method == http.MethodGet {
		URLId := strings.Split(req.URL.String(), "/")[1]
		if s.CheckMapKey(URLId) {
			url := s.GetOrigURL(URLId)
			res.Header().Add("Location", url)
			res.WriteHeader(http.StatusTemporaryRedirect)
		} else {
			http.Error(res, "Не корректный запрос", http.StatusBadRequest)
		}
		return
	} else {
		http.Error(res, "Не корректный запрос", http.StatusBadRequest)
	}
}
