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

	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"

	"github.com/Dorrrke/shortener-url/internal/config"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/pkg/models"
	"github.com/Dorrrke/shortener-url/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const SECRET_KEY = "Secret123Key345Super"

type Server struct {
	storage    storage.Storage
	ServerConf config.Config
	filePath   string
}

type restorURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

func (s *Server) GetOriginalURLHandler(res http.ResponseWriter, req *http.Request) {
	URLId := chi.URLParam(req, "id")
	if URLId != "" {
		var shortURL string
		if s.ServerConf.ShortURLHostConfig.Host == "" {
			shortURL = "http://" + req.Host + "/" + URLId
		} else {
			shortURL = "http://" + s.ServerConf.ShortURLHostConfig.String() + "/" + URLId
		}
		url, err := s.getURLByShortURL(shortURL)
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
}

func (s *Server) ShortenerURLHandler(res http.ResponseWriter, req *http.Request) {

	var userID string
	reqCookie, err := req.Cookie("auth")
	if err != nil {
		logger.Log.Info("Cookie false")
		userID = uuid.New().String()
		token, err := CreateJWTToken(userID)
		if err != nil {
			logger.Log.Error("cannot create token", zap.Error(err))
		}
		cookie := http.Cookie{
			Name:  "auth",
			Value: token,
			Path:  "/",
		}
		log.Printf("new uuid" + userID)
		http.SetCookie(res, &cookie)
	} else {
		logger.Log.Info("Cookie true")
		log.Printf("uuid from cookie: " + userID)
		userID = GetUID(reqCookie.Value)
		if userID == "" {
			http.Error(res, "User unauth", http.StatusUnauthorized)
			return
		}
		http.SetCookie(res, reqCookie)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, err.Error(), 500)
		return
	}
	if !validationURL(string(body)) {
		http.Error(res, "Не корректный запрос", http.StatusBadRequest)
		return
	}
	urlID := strings.Split(uuid.New().String(), "-")[0]
	var result string
	if s.ServerConf.ShortURLHostConfig.Host == "" {
		result = "http://" + req.Host + "/" + urlID
	} else {
		result = "http://" + s.ServerConf.ShortURLHostConfig.String() + "/" + urlID
	}

	if err := s.saveURL(string(body), result, userID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if !pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				logger.Log.Info("cannot save URL in file", zap.Error(err))
				http.Error(res, "Не корректный запрос", http.StatusBadRequest)
				return
			}

			shortURL, err := s.getURLByOriginalURL(string(body))
			if err != nil {
				logger.Log.Error("Error when read from base: ", zap.Error(err))
				http.Error(res, "Не корректный запрос", http.StatusBadRequest)
				return
			}
			result = shortURL
			res.Header().Set("content-type", "text/plain")
			res.WriteHeader(http.StatusConflict)
			res.Write([]byte(result))
			return
		}
	}
	res.Header().Set("content-type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(result))

}

func (s *Server) ShortenerJSONURLHandler(res http.ResponseWriter, req *http.Request) {

	var userID string
	reqCookie, err := req.Cookie("auth")
	if err != nil {
		userID = uuid.New().String()
		token, err := CreateJWTToken(userID)
		if err != nil {
			logger.Log.Info("cannot create token", zap.Error(err))
		}
		cookie := http.Cookie{
			Name:  "auth",
			Value: token,
			Path:  "/",
		}
		http.SetCookie(res, &cookie)
	} else {
		userID = uuid.New().String()
		http.SetCookie(res, reqCookie)
	}

	dec := json.NewDecoder(req.Body)
	var modelURL models.RequestURLJson

	if err := dec.Decode(&modelURL); err != nil {
		logger.Log.Debug("cannot decod boby json", zap.Error(err))
	}
	if !validationURL(string(modelURL.URLAddres)) {
		http.Error(res, "Не корректный запрос", http.StatusBadRequest)
		return
	}
	urlID := strings.Split(uuid.New().String(), "-")[0]
	var result string
	if s.ServerConf.ShortURLHostConfig.Host == "" {
		result = "http://" + req.Host + "/" + urlID
	} else {
		result = "http://" + s.ServerConf.ShortURLHostConfig.String() + "/" + urlID
	}
	if err := s.saveURL(modelURL.URLAddres, result, userID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				shortURL, err := s.getURLByOriginalURL(modelURL.URLAddres)
				if err != nil {
					logger.Log.Error("Error when read from base: ", zap.Error(err))
					http.Error(res, "Не корректный запрос", http.StatusBadRequest)
					return
				}
				result = shortURL
				res.Header().Set("Content-Type", "application/json")
				res.WriteHeader(http.StatusConflict)

				enc := json.NewEncoder(res)
				resultJSON := models.ResponseURLJson{
					URLAddres: shortURL,
				}
				if err := enc.Encode(resultJSON); err != nil {
					logger.Log.Debug("error encoding responce", zap.Error(err))
					http.Error(res, "Не корректный запрос", http.StatusInternalServerError)
				}
				return

			} else {
				logger.Log.Info("cannot save URL in file", zap.Error(err))
			}
		}
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(res)
	resultJSON := models.ResponseURLJson{
		URLAddres: result,
	}
	if err := enc.Encode(resultJSON); err != nil {
		logger.Log.Debug("error encoding responce", zap.Error(err))
		http.Error(res, "Не корректный запрос", http.StatusInternalServerError)
	}

}

func (s *Server) CheckDBConnectionHandler(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.storage.CheckDBConnect(ctx); err != nil {
		log.Printf("Error check connection: %v", err.Error())
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}

func (s *Server) GetAllUrls(res http.ResponseWriter, req *http.Request) {
	var userID string
	reqCookie, err := req.Cookie("auth")
	if err != nil {
		userID = uuid.New().String()
		token, err := CreateJWTToken(userID)
		if err != nil {
			logger.Log.Info("cannot create token", zap.Error(err))
		}
		cookie := http.Cookie{
			Name:  "auth",
			Value: token,
			Path:  "/",
		}
		log.Printf("new uuid" + userID)
		http.SetCookie(res, &cookie)
	} else {
		userID = GetUID(reqCookie.Value)
		if userID == "" {
			http.Error(res, "User unauth", http.StatusUnauthorized)
			return
		}
		log.Printf("uuid from cookie: " + userID)
		http.SetCookie(res, reqCookie)
	}
	urls, err := s.getAllURLs(userID)
	if err != nil {
		http.Error(res, "Не корректный запрос", http.StatusInternalServerError)
		return
	}
	if len(urls) == 0 {
		http.Error(res, "Нет сохраненных адресов", http.StatusNoContent)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(res)
	if err := enc.Encode(urls); err != nil {
		logger.Log.Debug("error encoding responce", zap.Error(err))
		http.Error(res, "Не корректный запрос", http.StatusInternalServerError)
	}

}

func (s *Server) InsertBatchHandler(res http.ResponseWriter, req *http.Request) {
	var userID string
	reqCookie, err := req.Cookie("auth")
	if err != nil {
		userID = uuid.New().String()
	} else {
		userID = GetUID(reqCookie.Value)
	}
	token, err := CreateJWTToken(userID)
	if err != nil {
		logger.Log.Info("cannot create token", zap.Error(err))
	}
	cookie := http.Cookie{
		Name:  "auth",
		Value: token,
		Path:  "/",
	}
	http.SetCookie(res, &cookie)

	dec := json.NewDecoder(req.Body)
	var modelURL []models.RequestBatchURLModel
	if err := dec.Decode(&modelURL); err != nil {
		logger.Log.Error("cannot decod boby json", zap.Error(err))
	}
	if len(modelURL) == 0 {
		http.Error(res, "Не корректный запрос", http.StatusBadRequest)
		return
	}
	var bantchValues []models.BantchURL
	var resBatchValues []models.ResponseBatchURLModel
	for _, v := range modelURL {
		if validationURL(v.OriginalURL) {
			urlID := strings.Split(uuid.New().String(), "-")[0]
			var shortURL string
			if s.ServerConf.ShortURLHostConfig.Host == "" {
				shortURL = "http://" + req.Host + "/" + urlID
			} else {
				shortURL = "http://" + s.ServerConf.ShortURLHostConfig.String() + "/" + urlID
			}
			bantchValues = append(bantchValues, models.BantchURL{
				OriginalURL: v.OriginalURL,
				ShortURL:    shortURL,
				UserId:      userID,
			})
			resBatchValues = append(resBatchValues, models.ResponseBatchURLModel{
				CorrID:      v.CorrID,
				OriginalURL: shortURL,
			})
		} else {
			http.Error(res, "Не корректный запрос", http.StatusBadRequest)
			return
		}
	}

	if err := s.SaveURLBatch(bantchValues); err != nil {
		logger.Log.Error("Error while save batch", zap.Error(err))
		http.Error(res, "Ошибка при сохарнении данных", http.StatusInternalServerError)
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(res)
	// resultJSON := models.ResponseURLJson{
	// 	URLAddres: result,
	// }
	if err := enc.Encode(resBatchValues); err != nil {
		logger.Log.Debug("error encoding responce", zap.Error(err))
		http.Error(res, "Не корректный запрос", http.StatusInternalServerError)
	}
}

func (s *Server) AddStorage(stor storage.Storage) {
	s.storage = stor
}

func (s *Server) AddFilePath(fileName string) {
	s.filePath = fileName
}

func (s *Server) GetFilePath() string {
	return s.filePath
}

func (s *Server) RestorStorage() error {
	if err := s.CreateTable(); err != nil {
		logger.Log.Info("Error when create table: " + err.Error())
		return errors.Wrap(err, "Error when create table: ")
	}
	if s.filePath != "" {
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
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			s.storage.InsertURL(ctx, data.ShortURL, data.OriginalURL, "")
		}
		file.Close()
		return nil
	}
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

// func (s *Server) AddDB(dataBase *pgx.Conn) {
// 	s.storage.DB = dataBase
// }

func (s *Server) saveURL(original string, short string, userID string) error {
	logger.Log.Info("Save into db")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.storage.InsertURL(ctx, original, short, userID); err != nil {
		return err
	}
	if s.filePath != "" {
		logger.Log.Info("Save into file")
		if err := writeURL(s.filePath, restorURL{short, original}); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (s *Server) SaveURLBatch(batch []models.BantchURL) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.storage.InsertBanchURL(ctx, batch); err != nil {
		return err
	}
	if s.filePath != "" {
		logger.Log.Info("Save batch into file")
		for _, v := range batch {
			if err := writeURL(s.filePath, restorURL{v.ShortURL, v.OriginalURL}); err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

func (s *Server) getURLByShortURL(short string) (string, error) {
	logger.Log.Info("Get from db")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	originalURL, err := s.storage.GetOriginalURLByShort(ctx, short)
	if err != nil {
		return "", err
	}
	return originalURL, nil
}

func (s *Server) getAllURLs(userID string) ([]models.URLModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	userURL, err := s.storage.GetAllUrls(ctx, userID)
	if err != nil {
		return nil, err
	}
	return userURL, nil
}

func (s *Server) getURLByOriginalURL(original string) (string, error) {
	logger.Log.Info("Get from db")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	originalURL, err := s.storage.GetShortByOriginalURL(ctx, original)
	if err != nil {
		return "", err
	}
	return originalURL, nil
}

func (s *Server) CreateTable() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.storage.CreateTable(ctx); err != nil {
		return err
	}
	return nil
}

func validationURL(URL string) bool {
	if strings.HasPrefix(URL, "http://") || strings.HasPrefix(URL, "https://") {
		return true
	}
	return false
}

func CreateJWTToken(uuid string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 3)),
		},
		UserID: uuid,
	})

	tokenString, err := token.SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func GetUID(tokenString string) string {
	claim := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claim, func(t *jwt.Token) (interface{}, error) {
		return []byte(SECRET_KEY), nil
	})
	if err != nil {
		return ""
	}

	if !token.Valid {
		return ""
	}
	log.Printf(claim.UserID)
	return claim.UserID
}
