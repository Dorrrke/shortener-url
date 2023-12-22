package server

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var db = "postgres://postgres:6406655@localhost:5432/postgres"

func TestInsertBatchHandler(t *testing.T) {

	r := chi.NewRouter()
	var server Server

	r.Route("/", func(r chi.Router) {
		r.Post("/api/user/urls", server.InsertBatchHandler)
	})

	srv := httptest.NewServer(r)
	pool, err := pgxpool.New(context.Background(), db)
	if err != nil {
		logger.Log.Error("Error wile init db driver: " + err.Error())
		panic(err)
	}
	server.AddStorage(&storage.DBStorage{DB: pool})
	if err := server.RestorStorage(); err != nil {
		logger.Log.Error("Error restor storage: ", zap.Error(err))
	}
	err = server.storage.ClearTables(context.Background())
	if err != nil {
		log.Println(err.Error())
	}

	type want struct {
		code        int
		contentType string
	}

	tests := []struct {
		name    string
		userID  string
		request string
		method  string
		value   string
		want    want
	}{
		{
			name:    "Test insert batch urls #1 Correct request",
			userID:  "asgds-ryew24-nbf45",
			request: "/api/user/urls",
			method:  http.MethodPost,
			value:   `[{"correlation_id": "dfas1","original_url": "https://music.yandex.ru/home"},{"correlation_id": "asfd2","original_url": "https://www.youtube.com/"},{"correlation_id": "3gda","original_url": "https://github.com/golang/mock"}]`,
			want: want{
				code:        http.StatusCreated,
				contentType: "application/json",
			},
		},
		{
			name:    "Test get all urls #2 Without userID",
			userID:  "",
			request: "/api/user/urls",
			method:  http.MethodPost,
			value:   `[{"correlation_id": "dfs1","original_url": "https://music.yandex.ru/home"},{"correlation_id": "fd1","original_url": "https://www.youtube.com/"},{"correlation_id": "fd3","original_url": "https://github.com/golang/mock"}]`,
			want: want{
				code:        http.StatusUnauthorized,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := tt.userID

			token, err := createJWTToken(userID)
			if err != nil {
				logger.Log.Info("cannot create token", zap.Error(err))
			}

			getReq := resty.New().R()
			getReq.Method = tt.method
			getReq.URL = srv.URL + tt.request
			getReq.Body = tt.value
			getReq.Cookies = append(getReq.Cookies, &http.Cookie{Name: "auth",
				Value: token,
				Path:  "/"})
			restGet, err := getReq.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.code, restGet.StatusCode())
			assert.Equal(t, tt.want.contentType, restGet.Header().Get("Content-Type"))
		})

	}
	srv.Close()
}
