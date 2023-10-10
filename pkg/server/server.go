package server

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"

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
	filePath   string
}

type restorURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
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

	logger.Log.Info("Test logger in handler")
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), 500)
		return
	}
	if strings.HasPrefix(string(body), "http://") || strings.HasPrefix(string(body), "https://") {
		urlID := strings.Split(uuid.New().String(), "-")[0]
		var result string
		if s.ServerConf.ShortURLHostConfig.Host == "" {
			result = "http://" + req.Host + "/" + urlID
		} else {
			result = "http://" + s.ServerConf.ShortURLHostConfig.String() + "/" + urlID
		}
		s.storage.CreateURL(urlID, string(body))
		if err := writeURL(s.filePath, restorURL{urlID, string(body)}); err != nil {
			logger.Log.Debug("cannot save URL in file", zap.Error(err)) //прокидывать экзепляр логера в сервер, что бы не пользоваться стандартным
			// http.Error(res, "Не корректный запрос", http.StatusInternalServerError) нужно передавать ошибку пользователю
		}
		res.Header().Set("content-type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(result))
		return
	}
	http.Error(res, "Не корректный запрос", http.StatusBadRequest)

}

func (s *Server) ShortenerJSONURLHandler(res http.ResponseWriter, req *http.Request) {

	dec := json.NewDecoder(req.Body)
	var modelURL RequestURLJson

	if err := dec.Decode(&modelURL); err != nil {
		logger.Log.Debug("cannot decod boby json", zap.Error(err))
	}
	if strings.HasPrefix(string(modelURL.URLAddres), "http://") || strings.HasPrefix(string(modelURL.URLAddres), "https://") {
		urlID := strings.Split(uuid.New().String(), "-")[0]
		var result string
		if s.ServerConf.ShortURLHostConfig.Host == "" {
			result = "http://" + req.Host + "/" + urlID
		} else {
			result = "http://" + s.ServerConf.ShortURLHostConfig.String() + "/" + urlID
		}
		s.storage.CreateURL(urlID, modelURL.URLAddres)
		if err := writeURL(s.filePath, restorURL{urlID, modelURL.URLAddres}); err != nil {
			logger.Log.Debug("cannot save URL in file", zap.Error(err)) //прокидывать экзепляр логера в сервер, что бы не пользоваться стандартным
			// http.Error(res, "Не корректный запрос", http.StatusInternalServerError) нужно передавать ошибку пользователю
		}
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusCreated)
		enc := json.NewEncoder(res)
		resultJSON := ResponseURLJson{
			result,
		}
		if err := enc.Encode(resultJSON); err != nil {
			logger.Log.Debug("error encoding responce", zap.Error(err))
			http.Error(res, "Не корректный запрос", http.StatusInternalServerError)
		}
		return
	}
	http.Error(res, "Не корректный запрос", http.StatusBadRequest)

}

func (s *Server) CheckDBConnectionHandler(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := s.storage.CheckDBConnect(ctx); err != nil {
		logger.Log.Debug("Error check connection")
		res.WriteHeader(http.StatusInternalServerError)
	}
	res.WriteHeader(http.StatusOK)
}

func (s *Server) New() {
	s.storage.URLMap = make(map[string]string)
}

func (s *Server) AddStorage(stor storage.URLStorage) {
	s.storage = stor
}

func (s *Server) InitBD(DBaddr string) error {
	conn, err := pgx.Connect(context.Background(), DBaddr)
	if err != nil {
		log.Printf("Error wile init db driver: %v", err.Error())
		return errors.Wrap(err, "Error to connect db")
	}
	s.storage.DB = conn
	defer conn.Close(context.Background())
	return nil
}

func (s *Server) GetStorage() {
	log.Println(s.storage.URLMap)
}

func (s *Server) AddFilePath(fileName string) {
	s.filePath = fileName
}

func (s *Server) GetFilePath() string {
	return s.filePath
}

func (s *Server) RestorStorage() error {
	file, err := os.OpenFile(s.filePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		data := restorURL{}
		err := json.Unmarshal(scanner.Bytes(), &data)
		if err != nil {
			return err
		}
		s.storage.CreateURL(data.ShortURL, data.OriginalURL)
	}
	file.Close()
	return nil
}

func writeURL(fileName string, lastURL restorURL) error {
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(file)
	data, err := json.Marshal(&lastURL)
	if err != nil {
		return errors.Wrap(err, "encode last url")
	}
	if _, err := writer.Write(data); err != nil {
		return errors.Wrap(err, "write if file last url")
	}
	if err := writer.WriteByte('\n'); err != nil {
		return errors.Wrap(err, "write in file '\n'")
	}
	writer.Flush()
	file.Close()
	return nil
}

func (s *Server) AddDB(db *pgx.Conn) {
	s.storage.DB = db
}
