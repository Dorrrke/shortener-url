package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/Dorrrke/shortener-url/internal/config"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/internal/models"
	"github.com/Dorrrke/shortener-url/internal/service"
	mock_storage "github.com/Dorrrke/shortener-url/mocks"
)

func TestGetAllUrls(t *testing.T) {

	r := chi.NewRouter()
	var server Server

	r.Route("/", func(r chi.Router) {
		r.Get("/api/user/urls", server.GetAllUrls)
	})

	srv := httptest.NewServer(r)

	type want struct {
		code        int
		contentType string
		body        string
	}

	tests := []struct {
		name    string
		userID  string
		request string
		method  string
		dbCall  bool
		value   []models.URLModel
		want    want
	}{
		{
			name:    "Test get all urls #1 Correct request",
			userID:  "asgds-ryew24-nbf45",
			request: "/api/user/urls",
			method:  http.MethodGet,
			dbCall:  true,
			value: []models.URLModel{
				{
					ShortID:    "http://aaa",
					OriginalID: "http://afdsafasdfadf",
				},
				{
					ShortID:    "http://bbb",
					OriginalID: "http://adfbvdshfdha",
				},
				{
					ShortID:    "http://ccc",
					OriginalID: "http://trytrukjtyj",
				},
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
				body:        `[{"short_url":"http://aaa","original_url":"http://afdsafasdfadf"},{"short_url":"http://bbb","original_url":"http://adfbvdshfdha"},{"short_url":"http://ccc","original_url":"http://trytrukjtyj"}]`,
			},
		},
		{
			name:    "Test get all urls #2 Without userID",
			userID:  "",
			request: "/api/user/urls",
			method:  http.MethodGet,
			dbCall:  false,
			want: want{
				code:        http.StatusUnauthorized,
				contentType: "text/plain; charset=utf-8",
				body:        `User unauth`,
			},
		},
		{
			name:    "Test get all urls #3 No value",
			userID:  "fdsfdsaa-gfgfg-hggh",
			request: "/api/user/urls",
			method:  http.MethodGet,
			dbCall:  true,
			value:   []models.URLModel{},
			want: want{
				code:        http.StatusNoContent,
				contentType: "text/plain; charset=utf-8",
				body:        ``,
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
				m.EXPECT().GetAllUrls(context.Background(), userID).Return(tt.value, nil)
			}
			token, err := createJWTToken(userID)
			if err != nil {
				logger.Log.Info("cannot create token", zap.Error(err))
			}

			var cfg config.AppConfig
			sService := service.NewService(m, &cfg)
			server = *New(&cfg, sService)
			getReq := resty.New().R()
			getReq.Method = tt.method
			getReq.URL = srv.URL + tt.request
			getReq.Cookies = append(getReq.Cookies, &http.Cookie{Name: "auth",
				Value: token,
				Path:  "/"})
			restGet, err := getReq.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.code, restGet.StatusCode())
			assert.Equal(t, tt.want.body, strings.Trim(string(restGet.Body()), "\n"))
			assert.Equal(t, tt.want.contentType, restGet.Header().Get("Content-Type"))
		})

	}
	srv.Close()
}

func BenchmarkGetAllUrls(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		r := chi.NewRouter()
		var server Server

		r.Route("/", func(r chi.Router) {
			r.Get("/api/user/urls", server.GetAllUrls)
		})

		srv := httptest.NewServer(r)
		ctrl := gomock.NewController(b)
		defer ctrl.Finish()

		m := mock_storage.NewMockStorage(ctrl)
		userID := "asgds-ryew24-nbf45"
		value := []models.URLModel{
			{
				ShortID:    "http://aaa",
				OriginalID: "http://afdsafasdfadf",
			},
			{
				ShortID:    "http://bbb",
				OriginalID: "http://adfbvdshfdha",
			},
			{
				ShortID:    "http://ccc",
				OriginalID: "http://trytrukjtyj",
			},
		}

		m.EXPECT().GetAllUrls(context.Background(), userID).Return(value, nil)

		token, err := createJWTToken(userID)
		if err != nil {
			logger.Log.Info("cannot create token", zap.Error(err))
		}

		var cfg config.AppConfig
		sService := service.NewService(m, &cfg)
		server = *New(&cfg, sService)
		getReq := resty.New().R()
		getReq.Method = http.MethodGet
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
