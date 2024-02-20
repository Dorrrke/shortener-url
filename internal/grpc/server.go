package grpcserver

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Dorrrke/shortener-url/internal/config"
	shortenergrpcv1 "github.com/Dorrrke/shortener-url/internal/grpc/gen/shortenergrpc.v1"
	"github.com/Dorrrke/shortener-url/internal/grpc/handlers"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/internal/service"
)

type ShortenerGRPCServer struct {
	shortenergrpcv1.UnimplementedShortenerServer
	sService *service.ShortenerService
	cfg      *config.AppConfig
}

func RegisterGrpcService(gRPC *grpc.Server, sService *service.ShortenerService, cfg *config.AppConfig) {
	shortenergrpcv1.RegisterShortenerServer(gRPC, &ShortenerGRPCServer{sService: sService, cfg: cfg})
}

func (s *ShortenerGRPCServer) GetOriginalURL(ctx context.Context, req *shortenergrpcv1.GetOriginalURLRequest) (*shortenergrpcv1.GetOriginalURLResponce, error) {
	return handlers.GetOriginalURLHandlerGrpc(ctx, *s.cfg, *s.sService, req.GetShortUrl())
}

func (s *ShortenerGRPCServer) ShortenerURL(ctx context.Context, req *shortenergrpcv1.ShortenerURLRequest) (*shortenergrpcv1.ShortenerURLResponce, error) {
	return handlers.ShortenerURLHandlerGrpc(ctx, *s.cfg, *s.sService, req.GetOriginalUrl())
}

func (s *ShortenerGRPCServer) ShortenerJSON(ctx context.Context, req *shortenergrpcv1.ShortenerJSONRequest) (*shortenergrpcv1.ShortenerJSONResponce, error) {
	return handlers.ShortenerJSONHandlerGrpc(ctx, *s.cfg, *s.sService, req.GetOrignalUrl())
}

func (s *ShortenerGRPCServer) CheckDBConnection(ctx context.Context, req *shortenergrpcv1.CheckDBConnectionRequest) (*shortenergrpcv1.CheckDBConnectionResponce, error) {
	err := s.sService.CheckDBConnection()
	if err != nil {
		logger.Log.Error("Error check db connect", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	return &shortenergrpcv1.CheckDBConnectionResponce{}, nil
}

func (s *ShortenerGRPCServer) GetAllURLs(ctx context.Context, req *shortenergrpcv1.GetAllURLsRequest) (*shortenergrpcv1.GetAllURLsResponce, error) {
	return handlers.GetAllURLsHandlerGrpc(ctx, *s.cfg, *s.sService)
}

func (s *ShortenerGRPCServer) InsertBatch(ctx context.Context, req *shortenergrpcv1.InsertBatchRequest) (*shortenergrpcv1.InsertBatchResponce, error) {
	return handlers.InsertBatchHandlerGrpc(ctx, *s.cfg, *s.sService, req.GetUrlsJson())
}

func (s *ShortenerGRPCServer) DeleteURL(ctx context.Context, req *shortenergrpcv1.DeleteURLRequest) (*shortenergrpcv1.DeleteURLResponce, error) {
	return handlers.DeleteURLHandlerGrpc(ctx, *s.cfg, *s.sService, req.GetUrls())
}

func (s *ShortenerGRPCServer) ServiceStat(ctx context.Context, req *shortenergrpcv1.ServiceStatRequest) (*shortenergrpcv1.ServiceStatResponce, error) {
	return handlers.ServiceStatHandlerGrpc(ctx, *s.cfg, *s.sService)
}
