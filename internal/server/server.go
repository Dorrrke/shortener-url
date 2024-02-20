// Пакет server содержит в себе основную логику работы hendler-ов свервиса, а так же промежуточные функции для общения со storage.
package server

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/Dorrrke/shortener-url/internal/config"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/internal/models"
	"github.com/Dorrrke/shortener-url/internal/service"
	"github.com/Dorrrke/shortener-url/internal/storage"
)

// SecretKey - Секретный ключ для создания JWT токена.
const SecretKey = "Secret123Key345Super"

// структура сервера, с данными о хранилище, конфиге, логгере и каналом для удаления url.
type Server struct {
	Config   *config.AppConfig
	sService service.ShortenerService
}

// структура Claims используется для созадния JWT Token.
type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

// New - метод создание экземпляра типа Server.
func New(cfg *config.AppConfig, service *service.ShortenerService) *Server {
	server := Server{
		Config:   cfg,
		sService: *service,
	}
	return &server
}

// GetOriginalURLHandler - хендлер для перехода на оригинальный адресс по сокращенной ссылке.
// В качестве ответа, хендлер находит в хранилище оригинальый url соответсвующий полученному сокращенному url и возвращает его в теле ответа с статус кодом 307 (StatusTemporaryRedirect).
// В том случае, если адрес удален, возвращается ошибка с кодм 410 (StatusGone).
func (s *Server) GetOriginalURLHandler(res http.ResponseWriter, req *http.Request) {
	URLId := chi.URLParam(req, "id")
	if URLId != "" {
		var shortURL string
		if s.Config.BaseURL == "" {
			shortURL = "http://" + req.Host + "/" + URLId
		} else {
			shortURL = "http://" + s.Config.BaseURL + "/" + URLId
		}
		url, deteted, err := s.sService.GetOriginalURL(shortURL)

		if err != nil {
			logger.Log.Error("Error when read from base: ", zap.Error(err))
			http.Error(res, "Не корректный запрос", http.StatusBadRequest)
			return
		}
		if deteted {
			res.WriteHeader(http.StatusGone)
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

// ShortenerURLHandler - хендлер для сокращения url.
// Хендлер получает в теле запроса url аддрес, создает случайню строку посредствам пакета uuid.
//
//	urlID := strings.Split(uuid.New().String(), "-")[0]
//
// После чего сохраняет полученный адррес в базу данных и возварщает его в теле ответа пользователю со статусом 210 (StatusCreated).
// В том случае если аддрес уже сохраняли, хендлер вернет сокращенный url со статусом 409 (StatusConflict).
func (s *Server) ShortenerURLHandler(res http.ResponseWriter, req *http.Request) {

	var userID string
	reqCookie, err := req.Cookie("auth")
	if err != nil {
		logger.Log.Info("Cookie false")
		userID = uuid.New().String()
		token, err := createJWTToken(userID)
		if err != nil {
			logger.Log.Error("cannot create token", zap.Error(err))
		}
		cookie := http.Cookie{
			Name:  "auth",
			Value: token,
			Path:  "/",
		}

		http.SetCookie(res, &cookie)
	} else {
		logger.Log.Info("Cookie true")

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
	if s.Config.BaseURL == "" {
		result = "http://" + req.Host + "/" + urlID
	} else {
		result = "http://" + s.Config.BaseURL + "/" + urlID
	}

	if err := s.sService.SaveURL(string(body), result, userID); err != nil {
		if errors.Is(err, storage.ErrMemStorageError) {
			shortURL, err := s.sService.GetShortByOriginal(string(body))
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if !pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				logger.Log.Info("cannot save URL in file", zap.Error(err))
				http.Error(res, "Не корректный запрос", http.StatusBadRequest)
				return
			}

			shortURL, err := s.sService.GetShortByOriginal(string(body))
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

// ShortenerJSONURLHandler - работатет аналогично ShortenerURLHandler только в теле запроса получает url в формате json.
// Хендлер получает в теле запроса url аддрес в формате json, десириализует полученную строку и создает случайню строку посредствам пакета uuid.
//
//	urlID := strings.Split(uuid.New().String(), "-")[0]
//
// После чего сохраняет полученный адррес в базу данных и возварщает его в теле ответа пользователю со статусом 210 (StatusCreated).
// В том случае если аддрес уже сохраняли, хендлер вернет сокращенный url со статусом 409 (StatusConflict).
func (s *Server) ShortenerJSONURLHandler(res http.ResponseWriter, req *http.Request) {

	var userID string
	reqCookie, err := req.Cookie("auth")
	if err != nil {
		userID = uuid.New().String()
		token, err := createJWTToken(userID)
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
	if s.Config.BaseURL == "" {
		result = "http://" + req.Host + "/" + urlID
	} else {
		result = "http://" + s.Config.BaseURL + "/" + urlID
	}
	if err := s.sService.SaveURL(modelURL.URLAddres, result, userID); err != nil {
		if errors.Is(err, storage.ErrMemStorageError) {
			shortURL, err := s.sService.GetShortByOriginal(modelURL.URLAddres)
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
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				shortURL, err := s.sService.GetShortByOriginal(modelURL.URLAddres)
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

// CheckDBConnectionHandler - хендлер для проверки подключения к базе данных.
// Если подключение есть, веренет статус код 200 (StatusOK).
// В случае если подключния нет, вернет статус код 500 (StatusInternalServerError).
func (s *Server) CheckDBConnectionHandler(res http.ResponseWriter, req *http.Request) {
	err := s.sService.CheckDBConnection()
	if err != nil {
		logger.Log.Error("Error check db connect", zap.Error(err))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}

// GetAllUrls - хендлер для получения всех сокращенных пользователем url.
// Сервис проверяет id пользователся из jwt токена хранящегося в cookie, если такого пользователя нет или id путое возвращает ошибку со статусом 401 (StatusUnauthorized).
// В случае если id существует, вернет все сокращенные пользователем url в формате json.
// Если пользователь не сократил ни одного url вернет ошибку со статусом 204 (StatusNoContent).
func (s *Server) GetAllUrls(res http.ResponseWriter, req *http.Request) {
	var userID string
	reqCookie, err := req.Cookie("auth")
	if err != nil {
		userID = uuid.New().String()
		token, err := createJWTToken(userID)
		if err != nil {
			logger.Log.Info("cannot create token", zap.Error(err))
		}
		cookie := http.Cookie{
			Name:  "auth",
			Value: token,
			Path:  "/",
		}

		http.SetCookie(res, &cookie)
		http.Error(res, "User unauth", http.StatusUnauthorized)
		return
	} else {
		userID = GetUID(reqCookie.Value)
		if userID == "" {
			http.Error(res, "User unauth", http.StatusUnauthorized)
			return
		}

		http.SetCookie(res, reqCookie)
	}
	urls, err := s.sService.GetAllURLsByID(userID)
	if err != nil {
		http.Error(res, "Не корректный запрос", http.StatusInternalServerError)
		return
	}
	if len(urls) == 0 {
		http.Error(res, "Нет сохраненных адресов", http.StatusNoContent)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(res)
	if err := enc.Encode(urls); err != nil {
		logger.Log.Debug("error encoding responce", zap.Error(err))
		http.Error(res, "Не корректный запрос", http.StatusInternalServerError)
	}

}

// InsertBatchHandler - хендлер для сохранения нескольок url за раз.
// Сервис проверяет id пользователся из jwt токена хранящегося в cookie, если такого пользователя нет или id путое возвращает ошибку со статусом 401 (StatusUnauthorized).
// В случае если id существует, десериализует данные из json, сокращает все адреса, сохраняет их в бд и возвращает пользователю список новых сокращенных адресов.
func (s *Server) InsertBatchHandler(res http.ResponseWriter, req *http.Request) {
	var userID string
	reqCookie, err := req.Cookie("auth")
	if err != nil {
		userID = uuid.New().String()
		token, err := createJWTToken(userID)
		if err != nil {
			logger.Log.Info("cannot create token", zap.Error(err))
			http.Error(res, "Cannot create token", http.StatusInternalServerError)
			return
		}
		cookie := http.Cookie{
			Name:  "auth",
			Value: token,
			Path:  "/",
		}

		http.SetCookie(res, &cookie)
	} else {
		userID = GetUID(reqCookie.Value)
		if userID == "" {
			http.Error(res, "User unauth", http.StatusUnauthorized)
			return
		}

		http.SetCookie(res, reqCookie)
	}

	dec := json.NewDecoder(req.Body)
	var modelURL []models.RequestBatchURLModel
	if err := dec.Decode(&modelURL); err != nil {
		logger.Log.Error("cannot decod boby json", zap.Error(err))
		http.Error(res, "Ошибка при разборе данных", http.StatusInternalServerError)
		return
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
			if s.Config.BaseURL == "" {
				shortURL = "http://" + req.Host + "/" + urlID
			} else {
				shortURL = "http://" + s.Config.BaseURL + "/" + urlID
			}
			bantchValues = append(bantchValues, models.BantchURL{
				OriginalURL: v.OriginalURL,
				ShortURL:    shortURL,
				UserID:      userID,
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

	if err := s.sService.SaveURLBatch(bantchValues); err != nil {
		logger.Log.Error("Error while save batch", zap.Error(err))
		http.Error(res, "Ошибка при сохарнении данных", http.StatusInternalServerError)
		return
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

// DeleteURLHandler - хендлер для удаления url.
// Сервис проверяет id пользователся из jwt токена хранящегося в cookie, если такого пользователя нет или id путое возвращает ошибку со статусом 401 (StatusUnauthorized).
// В случае если id существует, десериализует данные из json и отправляет их в канал для удаления, после чего, не дожидаясь окончания удаления возвращает статус 202 (StatusAccepted).
// Процесс удаления происходит в другом потоке, что бы не тормозить работу сервиса.
func (s *Server) DeleteURLHandler(res http.ResponseWriter, req *http.Request) {
	var userID string
	reqCookie, err := req.Cookie("auth")
	if err != nil {
		userID = uuid.New().String()
		token, err := createJWTToken(userID)
		if err != nil {
			logger.Log.Info("cannot create token", zap.Error(err))
			http.Error(res, "Cannot create token", http.StatusInternalServerError)
			return
		}
		cookie := http.Cookie{
			Name:  "auth",
			Value: token,
			Path:  "/",
		}

		http.SetCookie(res, &cookie)
	} else {
		userID = GetUID(reqCookie.Value)
		if userID == "" {
			http.Error(res, "User unauth", http.StatusUnauthorized)
			return
		}

		http.SetCookie(res, reqCookie)
	}

	dec := json.NewDecoder(req.Body)
	var moodel []string
	if err := dec.Decode(&moodel); err != nil {
		logger.Log.Error("cannot decod boby json", zap.Error(err))
	}
	go s.sService.DeleteURL(moodel, req.Host)
	res.WriteHeader(http.StatusAccepted)
}

// GetServiceStats - хендлер возвращающий статистику сервиса: количество пользователей и количество сокращенных URL.
// Хендлрер работает тольок в том случае, если при конфигурации сервиса было указанно строковое представление бесскалссовой адресации.
// Если при запросе хендлера, переданный в заглоловке X-Real-IP не в ходит в доврененную подсеть, хендлер возвращает статус 403.
// Если подсеть не указана вообще, то доступ к хендлеру запрещен вовсе.
func (s *Server) GetServiceStats(res http.ResponseWriter, req *http.Request) {
	if s.Config.TrustedSubnet == "" {
		http.Error(res, "Access is denied", http.StatusForbidden)
		return
	}
	realIP := req.Header.Get("X-Real-IP")
	headerIP := net.ParseIP(realIP)
	_, IPnet, _ := net.ParseCIDR(s.Config.TrustedSubnet)
	if !IPnet.Contains(headerIP) {
		http.Error(res, "Access is denied", http.StatusForbidden)
		return
	}

	statModel, err := s.sService.GetServiceStat()
	if err != nil {
		logger.Log.Error("Get stat error", zap.Error(err))
		http.Error(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(res)
	if err := enc.Encode(statModel); err != nil {
		logger.Log.Debug("error encoding responce", zap.Error(err))
		http.Error(res, "Не корректный запрос", http.StatusInternalServerError)
	}
}

// validationURL - метод валидации адреса.
func validationURL(URL string) bool {
	if strings.HasPrefix(URL, "http://") || strings.HasPrefix(URL, "https://") {
		return true
	}
	return false
}

// createJWTToken - функция создания JWT token.
func createJWTToken(uuid string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 3)),
		},
		UserID: uuid,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// GetUID - функция получения id пользвателя из jwt токена.
func GetUID(tokenString string) string {
	claim := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claim, func(t *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})
	if err != nil {
		return ""
	}

	if !token.Valid {
		return ""
	}

	return claim.UserID
}
