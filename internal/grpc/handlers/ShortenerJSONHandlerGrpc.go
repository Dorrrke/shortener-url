package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Dorrrke/shortener-url/internal/config"
	shortenergrpcv1 "github.com/Dorrrke/shortener-url/internal/grpc/gen/shortenergrpc.v1"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/internal/models"
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

func ShortenerJSONHandlerGrpc(ctx context.Context, cfg config.AppConfig, sService service.ShortenerService, orignalURL string) (*shortenergrpcv1.ShortenerJSONResponce, error) {
	var userID string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get("auth")
		if len(values) > 0 {
			token := values[0]
			userID := utils.GetUID(token)
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

	var modelURL models.RequestURLJson
	err := json.Unmarshal([]byte(orignalURL), &modelURL)
	if err != nil {
		logger.Log.Debug("cannot decod boby json", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal error")
	}

	if !utils.ValidationURL(string(modelURL.URLAddres)) {
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

	if err := sService.SaveURL(modelURL.URLAddres, shortURL, userID); err != nil {
		if errors.Is(err, storage.ErrMemStorageError) {
			shortDBURL, err := sService.GetShortByOriginal(modelURL.URLAddres)
			if err != nil {
				logger.Log.Error("Error when read from base: ", zap.Error(err))
				return nil, status.Error(codes.Internal, "Error when read from base")
			}
			jsonShortURL, err := json.Marshal(shortDBURL)
			if err != nil {
				logger.Log.Debug("cannot decod boby json", zap.Error(err))
				return nil, status.Error(codes.Internal, "Internal error")
			}

			return &shortenergrpcv1.ShortenerJSONResponce{ShortUrlJson: string(jsonShortURL)}, nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if !pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				logger.Log.Info("cannot save URL", zap.Error(err))
				return nil, status.Error(codes.Aborted, "Cannot save url")
			}

			shortDBURL, err := sService.GetShortByOriginal(modelURL.URLAddres)
			if err != nil {
				logger.Log.Error("Error when read from base: ", zap.Error(err))
				return nil, status.Error(codes.Internal, "Error when read from base")
			}
			jsonShortURL, err := json.Marshal(shortDBURL)
			if err != nil {
				logger.Log.Debug("cannot decod boby json", zap.Error(err))
				return nil, status.Error(codes.Internal, "Internal error")
			}
			return &shortenergrpcv1.ShortenerJSONResponce{ShortUrlJson: string(jsonShortURL)}, nil
		}
	}

	return &shortenergrpcv1.ShortenerJSONResponce{}, nil
}
