package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"

	"github.com/Dorrrke/shortener-url/internal/config"
	"github.com/Dorrrke/shortener-url/internal/service"
	"github.com/Dorrrke/shortener-url/internal/storage"
)

func TestGetOriginalURLHandler(t *testing.T) {

	r := chi.NewRouter()

	var URLServer Server

	r.Route("/", func(r chi.Router) {
		r.Post("/", URLServer.ShortenerURLHandler)
		r.Get("/{id}", URLServer.GetOriginalURLHandler)
	})
	srv := httptest.NewServer(r)

	cfg := config.AppConfig{
		ServerAddress:   srv.Config.Addr,
		BaseURL:         "",
		FileStoragePath: "",
		DatabaseDsn:     "",
		EnableHTTPS:     false,
	}

	sService := service.NewService(&storage.MemStorage{URLMap: make(map[string]string)}, &cfg)
	URLServer = *New(&cfg, sService)

	type want struct {
		code     int
		location string
	}

	tests := []struct {
		name    string
		request string
		method  string
		want    want
	}{
		{
			name: "Test Get hadler #1",
			want: want{
				code:     http.StatusOK,
				location: "https://www.youtube.com/",
			},
			request: "/",
			method:  http.MethodGet,
		},
		{
			name: "Test negative request from Get hadler #2",
			want: want{
				code:     http.StatusBadRequest,
				location: "",
			},
			request: "/",
			method:  http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postReq := resty.New().R()
			postReq.Method = http.MethodPost
			postReq.URL = srv.URL + tt.request
			postReq.Body = tt.want.location
			respPost, err := postReq.Send()
			assert.NoError(t, err, "error making HTTP request")
			var request string
			if strings.HasPrefix(string(respPost.Body()), "http://") {
				request = string(respPost.Body())
			} else {
				request = srv.URL + "/"
			}

			getReq := resty.New().R()
			getReq.Method = tt.method
			getReq.URL = request
			resp, err := getReq.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.code, resp.StatusCode())
		})
	}

	srv.Close()
}

func BenchmarkGetOriginalURLHandler(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		r := chi.NewRouter()

		var URLServer Server

		r.Route("/", func(r chi.Router) {
			r.Post("/", URLServer.ShortenerURLHandler)
			r.Get("/{id}", URLServer.GetOriginalURLHandler)
		})
		srv := httptest.NewServer(r)

		cfg := config.AppConfig{
			ServerAddress:   srv.Config.Addr,
			BaseURL:         "",
			FileStoragePath: "",
			DatabaseDsn:     "",
			EnableHTTPS:     false,
		}
		sService := service.NewService(&storage.MemStorage{URLMap: make(map[string]string)}, &cfg)
		URLServer = *New(&cfg, sService)

		postReq := resty.New().R()
		postReq.Method = http.MethodPost
		postReq.URL = srv.URL + "/"
		postReq.Body = "https://www.youtube.com/"
		respPost, err := postReq.Send()
		assert.NoError(b, err, "error making HTTP request")
		var request string
		if strings.HasPrefix(string(respPost.Body()), "http://") {
			request = string(respPost.Body())
		} else {
			request = srv.URL + "/"
		}

		getReq := resty.New().R()
		getReq.Method = http.MethodGet
		getReq.URL = request
		b.StartTimer()
		_, err = getReq.Send()
		assert.NoError(b, err, "error making HTTP request")
		srv.Close()
	}
}
