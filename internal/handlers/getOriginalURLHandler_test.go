package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dorrrke/shortener-url/cmd/storage"
	"github.com/stretchr/testify/assert"
)

func TestGetOriginalURLHandler(t *testing.T) {
	type want struct {
		code     int
		location string
	}

	tests := []struct {
		name    string
		request string
		URLmap  map[string]string
		want    want
	}{
		{
			name: "Test Get hadler #1",
			want: want{
				code:     http.StatusTemporaryRedirect,
				location: "https://www.youtube.com/",
			},
			request: "/qerttyAbC",
			URLmap: map[string]string{
				"qerttyAbC": "https://www.youtube.com/",
			},
		},
		{
			name: "Test Get hadler #2",
			want: want{
				code:     http.StatusTemporaryRedirect,
				location: "https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml",
			},
			request: "/progpgod",
			URLmap: map[string]string{
				"progpgod": "https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml",
			},
		},
		{
			name: "Test negative request from Get hadler #2",
			want: want{
				code:     http.StatusBadRequest,
				location: "",
			},
			request: "/",
			URLmap: map[string]string{
				"DGFdfgGD": "https://www.youtube.com/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stor := storage.URLStorage{
				URLMap: tt.URLmap,
			}
			request := httptest.NewRequest(http.MethodGet, tt.request, nil)
			w := httptest.NewRecorder()
			GetOriginalURLHandler(w, request, stor)

			result := w.Result()

			assert.Equal(t, tt.want.code, result.StatusCode)
			assert.Equal(t, tt.want.location, result.Header.Get("Location"))

			result.Body.Close()
		})
	}
}
