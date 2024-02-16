package handlers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Dorrrke/shortener-url/internal/config"
	"github.com/Dorrrke/shortener-url/internal/service"
	"github.com/Dorrrke/shortener-url/internal/storage"
)

func TestShortenerURLHandlerGrpc(t *testing.T) {
	type want struct {
		shortURL bool
	}

	tests := []struct {
		name        string
		originalURL string
		want        want
	}{
		{
			name: "Test Post hadler #1",
			want: want{
				shortURL: true,
			},
			originalURL: "https://www.youtube.com/",
		},
		{
			name: "Test Post hadler #2",
			want: want{
				shortURL: true,
			},
			originalURL: "https://www.iana.org/assignments/http-status-codes/http-status-codes.xhtml",
		},
		{
			name: "Test negative request from Post hadler #3",
			want: want{
				shortURL: false,
			},
			originalURL: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			cfg := config.AppConfig{
				ServerAddress:   "localhost:8080",
				BaseURL:         "",
				FileStoragePath: "",
				DatabaseDsn:     "",
				EnableHTTPS:     false,
			}
			sService := service.NewService(&storage.MemStorage{URLMap: make(map[string]string)}, &cfg)

			res, err := ShortenerURLHandlerGrpc(ctx, cfg, *sService, tt.originalURL)

			assert.NoError(t, err)
			assert.Equal(t, tt.want.shortURL, res)
		})
	}
}
