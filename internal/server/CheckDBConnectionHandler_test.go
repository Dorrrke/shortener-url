package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/Dorrrke/shortener-url/internal/config"
	"github.com/Dorrrke/shortener-url/internal/service"
	mock_storage "github.com/Dorrrke/shortener-url/mocks"
)

func TestCheckDBConnectionHandler(t *testing.T) {

	r := chi.NewRouter()
	var serverHTTP Server

	r.Route("/", func(r chi.Router) {
		r.Get("/ping", serverHTTP.CheckDBConnectionHandler)
	})

	srv := httptest.NewServer(r)

	type want struct {
		code int
	}

	tests := []struct {
		name      string
		request   string
		method    string
		dbConnect bool
		value     context.Context
		want      want
	}{
		{
			name:      "Test db check #1",
			request:   "/ping",
			method:    http.MethodGet,
			dbConnect: true,
			value:     context.Background(),
			want: want{
				code: http.StatusOK,
			},
		},
		{
			name:      "Test db check #2 No connect",
			request:   "/ping",
			method:    http.MethodGet,
			dbConnect: false,
			value:     context.Background(),
			want: want{
				code: http.StatusInternalServerError,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock_storage.NewMockStorage(ctrl)

			if !tt.dbConnect {
				m.EXPECT().CheckDBConnect(tt.value).Return(errors.New("no connect"))
			} else {
				m.EXPECT().CheckDBConnect(tt.value).Return(nil)
			}

			var cfg config.AppConfig
			sService := service.NewService(m, &cfg)
			serverHTTP = *New(&cfg, sService)
			getReq := resty.New().R()
			getReq.Method = tt.method
			getReq.URL = srv.URL + tt.request
			restGet, err := getReq.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tt.want.code, restGet.StatusCode())
		})

	}
	srv.Close()
}

func BenchmarkCheckDBConnectionHandler(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		r := chi.NewRouter()
		var serverHTTP Server

		r.Route("/", func(r chi.Router) {
			r.Get("/ping", serverHTTP.CheckDBConnectionHandler)
		})

		srv := httptest.NewServer(r)
		ctrl := gomock.NewController(b)
		defer ctrl.Finish()

		m := mock_storage.NewMockStorage(ctrl)

		m.EXPECT().CheckDBConnect(context.Background()).Return(nil)

		var cfg config.AppConfig
		sService := service.NewService(m, &cfg)
		serverHTTP = *New(&cfg, sService)
		getReq := resty.New().R()
		getReq.Method = http.MethodGet
		getReq.URL = srv.URL + "/ping"
		b.StartTimer()

		_, err := getReq.Send()
		assert.NoError(b, err, "error making HTTP request")
		srv.Close()
	}

}
