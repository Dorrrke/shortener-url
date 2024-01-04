package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func ExampleServer_GetAllUrls() {
	r := chi.NewRouter()
	var server Server

	r.Route("/", func(r chi.Router) {
		r.Get("/api/user/urls", server.GetAllUrls)
	})
	// Создадим тестовый сервер
	srv := httptest.NewServer(r)
	// Подключимся к бд
	pool, err := pgxpool.New(context.Background(), "postgres://postgres:6406655@localhost:5432/postgres")
	if err != nil {
		logger.Log.Error("Error wile init db driver: " + err.Error())
		panic(err)
	}
	server.AddStorage(&storage.DBStorage{DB: pool})

	// Создаем jwt токен с id пользвователя
	token, err := createJWTToken("asgds-ryew24-nbf45")
	if err != nil {
		logger.Log.Info("cannot create token", zap.Error(err))
	}
	//Создаем запрос с соответсвующими полями
	getReq := resty.New().R()
	getReq.Method = http.MethodGet
	getReq.URL = srv.URL + "/api/user/urls"
	getReq.Cookies = append(getReq.Cookies, &http.Cookie{Name: "auth",
		Value: token,
		Path:  "/"})
	// Отпарвляем запрос, тк url запроса соответсвует url хендлера, то выполится хендлер GetAllUrls
	_, err = getReq.Send()
}

func ExampleServer_GetOriginalURLHandler() {
	r := chi.NewRouter()

	var URLServer Server
	URLServer.AddStorage(&storage.MemStorage{URLMap: make(map[string]string)})

	r.Route("/", func(r chi.Router) {
		r.Post("/", URLServer.ShortenerURLHandler)
		r.Get("/{id}", URLServer.GetOriginalURLHandler)
	})
	srv := httptest.NewServer(r)

	getReq := resty.New().R()
	getReq.Method = http.MethodGet
	getReq.URL = srv.URL + "5c55d18e "
	_, err := getReq.Send()
	if err != nil {
		logger.Log.Info("Error")
	}
}

func ExampleServer_ShortenerURLHandler() {
	var URLServer Server
	URLServer.AddStorage(&storage.MemStorage{URLMap: make(map[string]string)})

	body := strings.NewReader("https://www.youtube.com/")
	request := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()
	URLServer.ShortenerURLHandler(w, request)
}

func ExampleServer_ShortenerJSONURLHandler() {
	var URLServer Server
	URLServer.AddStorage(&storage.MemStorage{URLMap: make(map[string]string)})

	body := strings.NewReader(`{"url":"https://www.youtube.com/"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/shorten", body)
	w := httptest.NewRecorder()
	URLServer.ShortenerJSONURLHandler(w, request)
}

func ExampleServer_InsertBatchHandler() {
	r := chi.NewRouter()
	var server Server

	r.Route("/", func(r chi.Router) {
		r.Post("/api/user/urls", server.InsertBatchHandler)
	})

	srv := httptest.NewServer(r)
	userID := "asgds-ryew24-nbf45"

	token, err := createJWTToken(userID)
	if err != nil {
		logger.Log.Info("cannot create token", zap.Error(err))
	}

	pool, err := pgxpool.New(context.Background(), "postgres://postgres:6406655@localhost:5432/postgres")
	if err != nil {
		logger.Log.Error("Error wile init db driver: " + err.Error())
		panic(err)
	}
	server.AddStorage(&storage.DBStorage{DB: pool})

	getReq := resty.New().R()
	getReq.Method = http.MethodPost
	getReq.URL = srv.URL + "/api/user/urls"
	getReq.Body = `[{"correlation_id": "dfas1","original_url": "https://music.yandex.ru/home"},{"correlation_id": "asfd2","original_url": "https://www.youtube.com/"},{"correlation_id": "3gda","original_url": "https://github.com/golang/mock"}]`
	getReq.Cookies = append(getReq.Cookies, &http.Cookie{Name: "auth",
		Value: token,
		Path:  "/"})
	_, err = getReq.Send()
}

func ExampleServer_DeleteURLHandler() {
	r := chi.NewRouter()
	var server Server

	r.Route("/", func(r chi.Router) {
		r.Delete("/api/user/urls", server.DeleteURLHandler)
		r.Get("/{id}", server.GetOriginalURLHandler)
	})

	srv := httptest.NewServer(r)
	userID := "asgds-ryew24-nbf45"
	token, err := createJWTToken(userID)
	if err != nil {
		logger.Log.Info("cannot create token", zap.Error(err))
	}

	pool, err := pgxpool.New(context.Background(), "postgres://postgres:6406655@localhost:5432/postgres")
	if err != nil {
		logger.Log.Error("Error wile init db driver: " + err.Error())
		panic(err)
	}
	server.AddStorage(&storage.DBStorage{DB: pool})
	getReq := resty.New().R()
	getReq.Method = http.MethodDelete
	getReq.URL = srv.URL + "/api/user/urls"
	getReq.Cookies = append(getReq.Cookies, &http.Cookie{Name: "auth",
		Value: token,
		Path:  "/"})
	_, err = getReq.Send()
}
