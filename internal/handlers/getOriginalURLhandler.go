package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Dorrrke/shortener-url/cmd/storage"
)

func GetOriginalURLHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		URLId := chi.URLParam(req, "id")
		if URLId != "" {
			if storage.MapURL.CheckMapKey(URLId) {
				url := storage.MapURL.GetOrigURL(URLId)
				res.Header().Add("Location", url)
				res.WriteHeader(http.StatusTemporaryRedirect)
			} else {
				http.Error(res, "Не корректный запрос", http.StatusBadRequest)
			}
			return
		} else {
			http.Error(res, "Не корректный запрос", http.StatusBadRequest)
		}
	} else {
		http.Error(res, "Не корректный запрос", http.StatusBadRequest)
	}
}
