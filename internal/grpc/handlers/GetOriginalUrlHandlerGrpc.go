package handlers

import (
	"context"

	"github.com/Dorrrke/shortener-url/internal/config"
	shortenergrpcv1 "github.com/Dorrrke/shortener-url/internal/grpc/gen/shortenergrpc.v1"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetOriginalURLHandlerGrpc(ctx context.Context, cfg config.AppConfig, sService service.ShortenerService, shortUrl string) (*shortenergrpcv1.GetOriginalURLResponce, error) {
	var shortURL string
	if cfg.BaseURL == "" {
		shortURL = "http://" + cfg.ServerAddress + "/" + shortUrl
	} else {
		shortURL = "http://" + cfg.BaseURL + "/" + shortUrl
	}

	url, delete, err := sService.GetOriginalURL(shortURL)
	if err != nil {
		logger.Log.Error("Error when read from base: ", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal error")
	}

	if delete {
		return nil, status.Error(codes.NotFound, "Url was deleted")
	}
	if url != "" {
		return &shortenergrpcv1.GetOriginalURLResponce{OriginalUrl: url}, nil
	}
	return nil, status.Error(codes.InvalidArgument, "Bad request")
}
