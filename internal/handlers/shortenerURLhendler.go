package handlers

import (
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/Dorrrke/shortener-url/cmd/storage"
	"github.com/google/uuid"
)

func ShortenerURLHandler(res http.ResponseWriter, req *http.Request) {

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), 500)
		return
	}
	matched, err := regexp.MatchString(`^(https?|ftp|file)://[-a-zA-Z0-9+&@#/%?=~_|!:,.;]*[-a-zA-Z0-9+&@#/%=~_|]`, string(body))
	if matched && err == nil {
		urlID := strings.Split(uuid.New().String(), "-")[0]
		storage.MapURL.CreateURL(urlID, string(body))
		resultURL := "http://" + req.Host + "/" + urlID
		res.Header().Set("content-type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(resultURL))
		return
	} else {
		http.Error(res, "Не корректный запрос", http.StatusBadRequest)
	}

}
