package handlers

import (
	"context"
	"encoding/json"
	"net"

	"github.com/Dorrrke/shortener-url/internal/config"
	shortenergrpcv1 "github.com/Dorrrke/shortener-url/internal/grpc/gen/shortenergrpc.v1"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func ServiceStatHandlerGrpc(ctx context.Context, cfg config.AppConfig, sService service.ShortenerService) (*shortenergrpcv1.ServiceStatResponce, error) {
	if cfg.TrustedSubnet == "" {
		return nil, status.Error(codes.PermissionDenied, "Permission denied")
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.PermissionDenied, "Permission denied")
	}
	realIP := md.Get("X-Real-IP")
	headerIP := net.ParseIP(realIP[0])
	_, IPnet, _ := net.ParseCIDR(cfg.TrustedSubnet)
	if !IPnet.Contains(headerIP) {
		return nil, status.Error(codes.PermissionDenied, "Permission denied")
	}

	statModel, err := sService.GetServiceStat()
	if err != nil {
		logger.Log.Error("Get stat error", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal error")
	}

	statJSON, err := json.Marshal(statModel)
	if err != nil {
		logger.Log.Debug("cannot encode to json", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal error")
	}

	return &shortenergrpcv1.ServiceStatResponce{Stat: string(statJSON)}, nil
}
