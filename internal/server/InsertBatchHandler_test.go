package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/Dorrrke/shortener-url/internal/config"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/internal/service"
	mock_storage "github.com/Dorrrke/shortener-url/mocks"
)

var db = "postgres://postgres:6406655@localhost:5432/postgres"

func TestInsertBatchHandler(t *testing.T) {

	r := chi.NewRouter()
	var server Server

	r.Route("/", func(r chi.Router) {
		r.Post("/api/user/urls", server.InsertBatchHandler)
	})

	srv := httptest.NewServer(r)

	cfg := config.AppConfig{
		ServerAddress:   srv.Config.Addr,
		BaseURL:         "",
		FileStoragePath: "",
		DatabaseDsn:     "",
		EnableHTTPS:     false,
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
		dbCall  bool
		want    want
	}{
		{
			name:    "Test insert batch urls #1 Correct request",
			userID:  "asgds-ryew24-nbf45",
			request: "/api/user/urls",
			method:  http.MethodPost,
			value:   `[{"correlation_id": "dfas1","original_url": "https://music.yandex.ru/home"},{"correlation_id": "asfd2","original_url": "https://www.youtube.com/"},{"correlation_id": "3gda","original_url": "https://github.com/golang/mock"}]`,
			dbCall:  true,
			want: want{
				code:        http.StatusCreated,
				contentType: "application/json",
			},
		},
		{
			name:    "Test insert batch urls #2 Without userID",
			userID:  "",
			request: "/api/user/urls",
			method:  http.MethodPost,
			value:   `[{"correlation_id": "dfs1","original_url": "https://music.yandex.ru/home"},{"correlation_id": "fd1","original_url": "https://www.youtube.com/"},{"correlation_id": "fd3","original_url": "https://github.com/golang/mock"}]`,
			dbCall:  false,
			want: want{
				code:        http.StatusUnauthorized,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := tt.userID

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock_storage.NewMockStorage(ctrl)
			if tt.dbCall {
				m.EXPECT().InsertBanchURL(context.Background(), gomock.All()).Return(nil)
			}
			token, err := createJWTToken(userID)
			if err != nil {
				logger.Log.Info("cannot create token", zap.Error(err))
			}

			sService := service.NewService(m, &cfg)
			server = *New(&cfg, sService)

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

func BenchmarkInsertBatchHandler(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		r := chi.NewRouter()
		var server Server

		r.Route("/", func(r chi.Router) {
			r.Post("/api/user/urls", server.InsertBatchHandler)
		})

		srv := httptest.NewServer(r)
		ctrl := gomock.NewController(b)
		defer ctrl.Finish()

		m := mock_storage.NewMockStorage(ctrl)
		m.EXPECT().InsertBanchURL(context.Background(), gomock.All()).Return(nil)

		cfg := config.AppConfig{
			ServerAddress:   srv.Config.Addr,
			BaseURL:         "",
			FileStoragePath: "",
			DatabaseDsn:     "",
			EnableHTTPS:     false,
		}
		sService := service.NewService(m, &cfg)
		server = *New(&cfg, sService)

		userID := "asgds-ryew24-nbf45"

		token, err := createJWTToken(userID)
		if err != nil {
			logger.Log.Info("cannot create token", zap.Error(err))
		}

		getReq := resty.New().R()
		getReq.Method = http.MethodPost
		getReq.URL = srv.URL + "/api/user/urls"
		getReq.Body = `[{"correlation_id": "dfas1","original_url": "https://music.yandex.ru/home"},{"correlation_id": "asfd2","original_url": "https://www.youtube.com/"},{"correlation_id": "3gda","original_url": "https://github.com/golang/mock"}]`
		getReq.Cookies = append(getReq.Cookies, &http.Cookie{Name: "auth",
			Value: token,
			Path:  "/"})
		b.StartTimer()
		_, err = getReq.Send()
		assert.NoError(b, err, "error making HTTP request")
		srv.Close()
	}
}
