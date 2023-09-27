package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/Dorrrke/shortener-url/internal/config"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type RequestURLJson struct {
	URLAddres string `json:"url"`
}
type ResponseURLJson struct {
	URLAddres string `json:"result"`
}

type Server struct {
	storage    storage.URLStorage
	ServerConf config.Config
}

func (s *Server) GetOriginalURLHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		URLId := chi.URLParam(req, "id")
		if URLId != "" {
			if s.storage.CheckMapKey(URLId) {
				url := s.storage.GetOrigURL(URLId)
				res.Header().Add("Location", url)
				res.WriteHeader(http.StatusTemporaryRedirect)
			} else {
				http.Error(res, "Не корректный запрос", http.StatusBadRequest)
			}
			return
		}
		http.Error(res, "Не корректный запрос", http.StatusBadRequest)
	} else {
		http.Error(res, "Не корректный запрос", http.StatusBadRequest)
	}
}

func (s *Server) ShortenerURLHandler(res http.ResponseWriter, req *http.Request) {

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), 500)
		return
	}
	matched, err := regexp.MatchString(`^(https?|ftp|file)://[-a-zA-Z0-9+&@#/%?=~_|!:,.;]*[-a-zA-Z0-9+&@#/%=~_|]`, string(body))
	if matched && err == nil {
		urlID := strings.Split(uuid.New().String(), "-")[0]
		var result string
		if s.ServerConf.ShortURLHostConfig.Host == "" {
			result = "http://" + req.Host + "/" + urlID
		} else {
			result = "http://" + s.ServerConf.ShortURLHostConfig.String() + "/" + urlID
		}
		s.storage.CreateURL(urlID, string(body))
		res.Header().Set("content-type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(result))
		return
	}
	http.Error(res, "Не корректный запрос", http.StatusBadRequest)

}

func (s *Server) ShortenerJsonURLHandler(res http.ResponseWriter, req *http.Request) {

	dec := json.NewDecoder(req.Body)
	var modelURL RequestURLJson

	if err := dec.Decode(&modelURL); err != nil {
		logger.Log.Debug("cannot decod boby json", zap.Error(err))
	}
	matched, err := regexp.MatchString(`^(https?|ftp|file)://[-a-zA-Z0-9+&@#/%?=~_|!:,.;]*[-a-zA-Z0-9+&@#/%=~_|]`, modelURL.URLAddres)
	if matched && err == nil {
		urlID := strings.Split(uuid.New().String(), "-")[0]
		var result string
		if s.ServerConf.ShortURLHostConfig.Host == "" {
			result = "http://" + req.Host + "/" + urlID
		} else {
			result = "http://" + s.ServerConf.ShortURLHostConfig.String() + "/" + urlID
		}
		s.storage.CreateURL(urlID, modelURL.URLAddres)
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusCreated)
		enc := json.NewEncoder(res)
		resultJson := ResponseURLJson{
			result,
		}
		if err := enc.Encode(resultJson); err != nil {
			logger.Log.Debug("error encoding responce", zap.Error(err))
		}
		return
	}
	log.Print(matched)
	http.Error(res, "Не корректный запрос", http.StatusBadRequest)

}

func (s *Server) New() {
	s.storage.URLMap = make(map[string]string)
}

func (s *Server) AddStorage(stor storage.URLStorage) {
	s.storage = stor
}

func (s *Server) GetStorage() {
	log.Println(s.storage.URLMap)
}
