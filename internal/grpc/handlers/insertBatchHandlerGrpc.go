package handlers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Dorrrke/shortener-url/internal/config"
	shortenergrpcv1 "github.com/Dorrrke/shortener-url/internal/grpc/gen/shortenergrpc.v1"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/internal/models"
	"github.com/Dorrrke/shortener-url/internal/service"
	"github.com/Dorrrke/shortener-url/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func InsertBatchHandlerGrpc(ctx context.Context, cfg config.AppConfig, sService service.ShortenerService, URLsJSON string) (*shortenergrpcv1.InsertBatchResponce, error) {
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

	var modelURL []models.RequestBatchURLModel
	err := json.Unmarshal([]byte(URLsJSON), &modelURL)
	if err != nil {
		logger.Log.Debug("cannot decod boby json", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal error")
	}

	if len(modelURL) == 0 {
		return nil, status.Error(codes.NotFound, "No data")
	}

	var bantchValues []models.BantchURL
	var resBatchValues []models.ResponseBatchURLModel
	for _, v := range modelURL {
		if utils.ValidationURL(v.OriginalURL) {
			urlID := strings.Split(uuid.New().String(), "-")[0]
			var shortURL string
			if cfg.BaseURL == "" {
				shortURL = "http://" + cfg.ServerAddress + "/" + urlID
			} else {
				shortURL = "http://" + cfg.BaseURL + "/" + urlID
			}
			bantchValues = append(bantchValues, models.BantchURL{
				OriginalURL: v.OriginalURL,
				ShortURL:    shortURL,
				UserID:      userID,
			})
			resBatchValues = append(resBatchValues, models.ResponseBatchURLModel{
				CorrID:      v.CorrID,
				OriginalURL: shortURL,
			})
		} else {
			logger.Log.Error("Bad request, no valid url")
			return nil, status.Error(codes.InvalidArgument, "Bad request")
		}
	}

	if err := sService.SaveURLBatch(bantchValues); err != nil {
		logger.Log.Error("Error while save batch", zap.Error(err))
		return nil, status.Error(codes.Internal, "Save data error")
	}

	jsonUrls, err := json.Marshal(resBatchValues)
	if err != nil {
		logger.Log.Debug("cannot decod boby json", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	return &shortenergrpcv1.InsertBatchResponce{ShortUrlsJson: string(jsonUrls)}, nil
}
