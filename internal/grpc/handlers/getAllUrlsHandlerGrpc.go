package handlers

import (
	"context"
	"encoding/json"

	"github.com/Dorrrke/shortener-url/internal/config"
	shortenergrpcv1 "github.com/Dorrrke/shortener-url/internal/grpc/gen/shortenergrpc.v1"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/internal/service"
	"github.com/Dorrrke/shortener-url/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func GetAllURLsHandlerGrpc(ctx context.Context, cfg config.AppConfig, sService service.ShortenerService) (*shortenergrpcv1.GetAllURLsResponce, error) {
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

	urls, err := sService.GetAllURLsByID(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "Internal error")
	}
	if len(urls) == 0 {
		return nil, status.Error(codes.NotFound, "No data")
	}
	jsonURLs, err := json.Marshal(urls)
	if err != nil {
		logger.Log.Debug("cannot encode to json", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	return &shortenergrpcv1.GetAllURLsResponce{AllUrlsJson: string(jsonURLs)}, nil
}
