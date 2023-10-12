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
			url, err := s.getURL(URLId)
			if err != nil {
				logger.Log.Error("Error when read from base: ", zap.Error(err))
				http.Error(res, "Не корректный запрос", http.StatusBadRequest)
				return
			}
			if url != "" {
				res.Header().Add("Location", url)
				res.WriteHeader(http.StatusTemporaryRedirect)
				return
			}
			http.Error(res, "Не корректный запрос", http.StatusBadRequest)
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

		if err := s.saveURL(string(body), urlID); err != nil {
			logger.Log.Info("cannot save URL in file", zap.Error(err))
		}

		// s.storage.CreateURL(urlID, string(body))
		// if err := writeURL(s.filePath, restorURL{urlID, string(body)}); err != nil {
		// 	logger.Log.Debug("cannot save URL in file", zap.Error(err)) //прокидывать экзепляр логера в сервер, что бы не пользоваться стандартным
		// 	// http.Error(res, "Не корректный запрос", http.StatusInternalServerError) нужно передавать ошибку пользователю
		// }
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

		if err := s.saveURL(modelURL.URLAddres, urlID); err != nil {
			logger.Log.Info("cannot save URL in file", zap.Error(err))
		}

		// s.storage.CreateURL(urlID, modelURL.URLAddres)
		// if err := writeURL(s.filePath, restorURL{urlID, modelURL.URLAddres}); err != nil {
		// 	logger.Log.Debug("cannot save URL in file", zap.Error(err)) //прокидывать экзепляр логера в сервер, что бы не пользоваться стандартным
		// 	// http.Error(res, "Не корректный запрос", http.StatusInternalServerError) нужно передавать ошибку пользователю
		// }
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
	if s.storage.DB == nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.storage.CheckDBConnect(ctx); err != nil {
		log.Printf("Error check connection: %v", err.Error())
		res.WriteHeader(http.StatusInternalServerError)
		return
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

	if s.storage.DB != nil {
		if err := s.CreateTable(); err != nil {
			logger.Log.Info("Error when create table: " + err.Error())
			return errors.Wrap(err, "Error when create table: ")
		}
		return nil
	} else {
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

func (s *Server) saveURL(original string, short string) error {

	if s.storage.DB != nil {
		logger.Log.Info("Save into db")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.storage.InsertURL(ctx, original, short); err != nil {
			return err
		}
		return nil
	} else {
		if s.filePath != "" {
			logger.Log.Info("Save into file")
			if err := writeURL(s.filePath, restorURL{short, original}); err != nil {
				logger.Log.Debug("cannot save URL in file", zap.Error(err)) //прокидывать экзепляр логера в сервер, что бы не пользоваться стандартным
				return err
			}
			s.storage.CreateURL(short, original)
			return nil
		} else {
			logger.Log.Debug("Save into map")
			s.storage.CreateURL(short, original)
			return nil
		}
	}
}

func (s *Server) getURL(short string) (string, error) {
	if s.storage.DB != nil {
		logger.Log.Info("Get from db")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		originalURL, err := s.storage.GetURLByShortURL(ctx, short)
		if err != nil {
			return "", err
		}
		return originalURL, nil
	} else {
		logger.Log.Info("Get from map")
		if s.storage.CheckMapKey(short) {
			url := s.storage.GetOrigURL(short)
			return url, nil
		}
		return "", nil // Тут нужна ошибка
	}
}

func (s *Server) CreateTable() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.storage.CreateTable(ctx); err != nil {
		return err
	}
	return nil
}
