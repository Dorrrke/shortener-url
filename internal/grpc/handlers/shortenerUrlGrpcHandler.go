package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/Dorrrke/shortener-url/internal/config"
	shortenergrpcv1 "github.com/Dorrrke/shortener-url/internal/grpc/gen/shortenergrpc.v1"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/internal/service"
	"github.com/Dorrrke/shortener-url/internal/storage"
	"github.com/Dorrrke/shortener-url/internal/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func ShortenerURLHandlerGrpc(ctx context.Context, cfg config.AppConfig, sService service.ShortenerService, originalURL string) (*shortenergrpcv1.ShortenerURLResponce, error) {
	var userID string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get("auth")
		if len(values) > 0 {
			token := values[0]
			userID := utils.GetUID(userID)
			if userID == "" {
				logger.Log.Error("User id from token is empty")
				return nil, status.Error(codes.Unauthenticated, "User id from token is empty")
			}
			header := metadata.Pairs("auth", token)
			grpc.SetHeader(ctx, header)
		} else {
			userID := uuid.New().String()
			token, err := utils.CreateJWTToken(userID)
			if err != nil {
				logger.Log.Error("cannot create token", zap.Error(err))
				return nil, status.Error(codes.Internal, "Create token error")
			}
			header := metadata.Pairs("auth", token)
			grpc.SetHeader(ctx, header)
		}
	} else {
		userID := uuid.New().String()
		token, err := utils.CreateJWTToken(userID)
		if err != nil {
			logger.Log.Error("cannot create token", zap.Error(err))
			return nil, status.Error(codes.Internal, "Create token error")
		}
		header := metadata.Pairs("auth", token)
		grpc.SetHeader(ctx, header)
	}
	original := originalURL
	if !utils.ValidationURL(original) {
		logger.Log.Error("Bad request, no valid url")
		return nil, status.Error(codes.InvalidArgument, "Bad request")
	}
	urlID := strings.Split(uuid.New().String(), "-")[0]
	var shortURL string
	if cfg.BaseURL == "" {
		shortURL = "http://" + cfg.ServerAddress + "/" + urlID
	} else {
		shortURL = "http://" + cfg.BaseURL + "/" + urlID
	}

	if err := sService.SaveURL(original, shortURL, userID); err != nil {
		if errors.Is(err, storage.ErrMemStorageError) {
			shortDBURL, err := sService.GetShortByOriginal(original)
			if err != nil {
				logger.Log.Error("Error when read from base: ", zap.Error(err))
				return nil, status.Error(codes.Internal, "Error when read from base")
			}
			return &shortenergrpcv1.ShortenerURLResponce{ShortUrl: shortDBURL}, nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if !pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				logger.Log.Info("cannot save URL", zap.Error(err))
				return nil, status.Error(codes.Aborted, "Cannot save url")
			}

			shortDBURL, err := sService.GetShortByOriginal(original)
			if err != nil {
				logger.Log.Error("Error when read from base: ", zap.Error(err))
				return nil, status.Error(codes.Internal, "Error when read from base")
			}
			return &shortenergrpcv1.ShortenerURLResponce{ShortUrl: shortDBURL}, nil
		}
	}
	return &shortenergrpcv1.ShortenerURLResponce{ShortUrl: shortURL}, nil

}
