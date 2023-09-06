package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Dorrrke/shortener-url/cmd/storage"
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
			name: "Test negative request from Post hadler #2",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				shortURL:    "http://localhost:8080/",
			},
			request: "/",
			body:    "https://www.youtube.com/",
			method:  http.MethodDelete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stor := storage.URLStorage{
				URLMap: make(map[string]string),
			}
			body := strings.NewReader(tt.body)
			request := httptest.NewRequest(tt.method, tt.request, body)
			w := httptest.NewRecorder()
			ShortenerURLHandler(w, request, stor)

			result := w.Result()

			assert.Equal(t, tt.want.code, result.StatusCode)
			assert.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))

			result.Body.Close()
		})
	}
}
