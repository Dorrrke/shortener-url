package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Dorrrke/shortener-url/internal/config"
	"github.com/Dorrrke/shortener-url/pkg/storage"
	"github.com/stretchr/testify/assert"
)

func TestShortenerURLHandler(t *testing.T) {
	type want struct {
		code        int
		contentType string
		shortURL    string
	}

	tests := []struct {
		name    string
		body    string
		request string
		method  string
		want    want
	}{
		{
			name: "Test Post hadler #1",
			want: want{
				code:        http.StatusCreated,
				contentType: "text/plain",
				shortURL:    "http://localhost:8080/",
			},
			request: "/",
			body:    "https://www.youtube.com/",
			method:  http.MethodPost,
		},
		{
			name: "Test Post hadler #2",
			want: want{
				code:        http.StatusCreated,
				contentType: "text/plain",
				shortURL:    "http://localhost:8080/",
			},
			request: "/",
			body:    "https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml",
			method:  http.MethodPost,
		},
		{
			name: "Test negative request from Post hadler #3",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				shortURL:    "http://localhost:8080/",
			},
			request: "/",
			body:    "/",
			method:  http.MethodPost,
		},
		{
			name: "Test negative request from Post hadler #4",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				shortURL:    "http://localhost:8080/",
			},
			request: "/",
			body:    "www.youtube.com",
			method:  http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var URLServer Server
			URLServer.AddStorage(&storage.MemStorage{URLMap: make(map[string]string)})

			cfg := config.AppConfig{
				ServerAddress:   "localhost:8080",
				BaseURL:         "",
				FileStoragePath: "",
				DatabaseDsn:     "",
				EnableHTTPS:     false,
			}
			URLServer.Config = &cfg

			body := strings.NewReader(tt.body)
			request := httptest.NewRequest(tt.method, tt.request, body)
			w := httptest.NewRecorder()
			URLServer.ShortenerURLHandler(w, request)

			result := w.Result()

			assert.Equal(t, tt.want.code, result.StatusCode)
			assert.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))

			result.Body.Close()
		})
	}
}

func BenchmarkShortenerURLHandler(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var URLServer Server
		URLServer.AddStorage(&storage.MemStorage{URLMap: make(map[string]string)})
		cfg := config.AppConfig{
			ServerAddress:   "localhost:8080",
			BaseURL:         "",
			FileStoragePath: "",
			DatabaseDsn:     "",
			EnableHTTPS:     false,
		}
		URLServer.Config = &cfg

		body := strings.NewReader("https://www.youtube.com/")
		request := httptest.NewRequest(http.MethodPost, "/", body)
		w := httptest.NewRecorder()
		b.StartTimer()
		URLServer.ShortenerURLHandler(w, request)
	}
}
