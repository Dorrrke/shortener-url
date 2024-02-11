package server

import (
	"context"
	"log"
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
	mock_storage "github.com/Dorrrke/shortener-url/mocks"
)

func TestDeleteURLHandler(t *testing.T) {

	r := chi.NewRouter()
	var server Server

	r.Route("/", func(r chi.Router) {
		r.Delete("/api/user/urls", server.DeleteURLHandler)
		r.Get("/{id}", server.GetOriginalURLHandler)
	})

	srv := httptest.NewServer(r)

	cfg := config.AppConfig{
		ServerAddress:   srv.Config.Addr,
		BaseURL:         "",
		FileStoragePath: "",
		DatabaseDsn:     "",
		EnableHTTPS:     false,
	}
	server.Config = &cfg

	type want struct {
		code    int
		getCode int
	}

	tests := []struct {
		name    string
		userID  string
		request string
		method  string
		dbCall  bool
		value   string
		want    want
	}{
		{
			name:    "Test get all urls #1 Correct request",
			userID:  "asgds-ryew24-nbf45",
			request: "/api/user/urls",
			method:  http.MethodDelete,
			dbCall:  true,
			value:   "6qxTVvsy",
			want: want{
				code:    http.StatusAccepted,
				getCode: http.StatusGone,
			},
		},
		{
			name:    "Test get all urls #2 Without userID",
			userID:  "",
			request: "/api/user/urls",
			method:  http.MethodDelete,
			dbCall:  false,
			value:   "6qxTVvsy",
			want: want{
				code: http.StatusUnauthorized,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock_storage.NewMockStorage(ctrl)
			userID := tt.userID

			if tt.dbCall {
				// m.EXPECT().SetDeleteURLStatus(context.Background(), tt.value).Return(nil)
				m.EXPECT().GetOriginalURLByShort(context.Background(), srv.URL+"/"+tt.value).Return("url1", true, nil)
			}
			token, err := createJWTToken(userID)
			if err != nil {
				logger.Log.Info("cannot create token", zap.Error(err))
			}

			server.AddStorage(m)
			getReq := resty.New().R()
			getReq.Method = tt.method
			getReq.URL = srv.URL + tt.request
			getReq.Cookies = append(getReq.Cookies, &http.Cookie{Name: "auth",
				Value: token,
				Path:  "/"})
			restGet, err := getReq.Send()
			if !tt.dbCall {
				assert.NoError(t, err, "error making HTTP request")
				assert.Equal(t, tt.want.code, restGet.StatusCode())
				return
			}
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.code, restGet.StatusCode())
			log.Println("Value: ")
			log.Println(tt.value)

			log.Println("delete test: deleted url: " + tt.value)
			req := resty.New().R()
			req.Method = http.MethodGet
			req.URL = srv.URL + "/" + tt.value
			rest, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.getCode, rest.StatusCode())
		})

	}

	srv.Close()
}

func BenchmarkDeleteURLHandler(b *testing.B) {
	for i := 0; i < b.N; i++ {
		r := chi.NewRouter()
		var server Server

		r.Route("/", func(r chi.Router) {
			r.Delete("/api/user/urls", server.DeleteURLHandler)
			r.Get("/{id}", server.GetOriginalURLHandler)
		})

		srv := httptest.NewServer(r)
		b.StopTimer()
		ctrl := gomock.NewController(b)
		defer ctrl.Finish()

		m := mock_storage.NewMockStorage(ctrl)
		userID := "asgds-ryew24-nbf45"

		token, err := createJWTToken(userID)
		if err != nil {
			logger.Log.Info("cannot create token", zap.Error(err))
		}

		server.AddStorage(m)
		getReq := resty.New().R()
		getReq.Method = http.MethodDelete
		getReq.URL = srv.URL + "/api/user/urls"
		getReq.Cookies = append(getReq.Cookies, &http.Cookie{Name: "auth",
			Value: token,
			Path:  "/"})
		b.StartTimer()
		_, err = getReq.Send()

		assert.NoError(b, err, "error making HTTP request")
		srv.Close()
	}
}
