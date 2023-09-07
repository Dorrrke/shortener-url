package handlers

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/Dorrrke/shortener-url/cmd/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

func TestGetOriginalURLHandler(t *testing.T) {

	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Post("/", ShortenerURLHandler)
		r.Get("/{id}", GetOriginalURLHandler)
	})
	srv := httptest.NewServer(r)

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
			name: "Test Get hadler #2",
			want: want{
				code:     http.StatusOK,
				location: "https://music.yandex.ru/home",
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

			storage.MapURL = storage.URLStorage{
				URLMap: make(map[string]string),
			}

			postReq := resty.New().R()
			postReq.Method = http.MethodPost
			postReq.URL = srv.URL + tt.request
			postReq.Body = tt.want.location
			respPost, err := postReq.Send()
			assert.NoError(t, err, "error making HTTP request")
			var request string
			matched, err := regexp.MatchString(`^(https?|ftp|file)://[-a-zA-Z0-9+&@#/%?=~_|!:,.;]*[-a-zA-Z0-9+&@#/%=~_|]`, string(respPost.Body()))
			if matched && err == nil {
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
			// assert.Equal(t, tt.want.location, resp.Header().Get("Location"))

			// body := strings.NewReader(tt.want.location)
			// requestPost := httptest.NewRequest(http.MethodPost, tt.request, body)
			// wPost := httptest.NewRecorder()

			// ShortenerURLHandler(wPost, requestPost)
			// resultPost := wPost.Result()

			// getResult, err := ioutil.ReadAll(resultPost.Body)
			// require.NoError(t, err)
			// err = resultPost.Body.Close()
			// require.NoError(t, err)

			// tt.request = tt.request + string(getResult)

			// log.Println(tt.request)
			// log.Println(storage.MapURL)

			// requestGet := httptest.NewRequest(http.MethodGet, tt.request, nil)
			// wGet := httptest.NewRecorder()
			// GetOriginalURLHandler(wGet, requestGet)

			// result := wGet.Result()

			// assert.Equal(t, tt.want.code, result.StatusCode)
			// assert.Equal(t, tt.want.location, result.Header.Get("Location"))

			// result.Body.Close()
		})
	}

	srv.Close()
}
