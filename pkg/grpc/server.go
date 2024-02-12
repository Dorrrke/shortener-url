package grpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/Dorrrke/shortener-url/internal/config"
	"github.com/Dorrrke/shortener-url/internal/logger"
	shortenergrpcv1 "github.com/Dorrrke/shortener-url/pkg/grpc/gen/shortenergrpc.v1"
	"github.com/Dorrrke/shortener-url/pkg/models"
	"github.com/Dorrrke/shortener-url/pkg/service"
	"github.com/Dorrrke/shortener-url/pkg/storage"
)

// SecretKey - Секретный ключ для создания JWT токена.
const SecretKey = "Secret123Key345Super"

type ShortenerGRPCServer struct {
	shortenergrpcv1.UnimplementedShortenerServer
	sService *service.ShortenerService
	cfg      *config.AppConfig
}

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

func RegisterGrpcService(gRPC *grpc.Server, sService *service.ShortenerService, cfg *config.AppConfig) {
	shortenergrpcv1.RegisterShortenerServer(gRPC, &ShortenerGRPCServer{sService: sService, cfg: cfg})
}

func (s *ShortenerGRPCServer) GetOriginalURL(ctx context.Context, req *shortenergrpcv1.GetOriginalURLRequest) (*shortenergrpcv1.GetOriginalURLResponce, error) {
	var shortURL string
	if s.cfg.BaseURL == "" {
		shortURL = "http://" + s.cfg.ServerAddress + "/" + req.GetShortUrl()
	} else {
		shortURL = "http://" + s.cfg.BaseURL + "/" + req.GetShortUrl()
	}

	url, delete, err := s.sService.GetOriginalURL(shortURL)
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

func (s *ShortenerGRPCServer) ShortenerURL(ctx context.Context, req *shortenergrpcv1.ShortenerURLRequest) (*shortenergrpcv1.ShortenerURLResponce, error) {
	var userID string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get("auth")
		if len(values) > 0 {
			token := values[0]
			userID := GetUID(token)
			if userID == "" {
				logger.Log.Error("User id from token is empty")
				return nil, status.Error(codes.Unauthenticated, "User id from token is empty")
			}
			header := metadata.Pairs("auth", token)
			grpc.SetHeader(ctx, header)
		} else {
			userID := uuid.New().String()
			token, err := createJWTToken(userID)
			if err != nil {
				logger.Log.Error("cannot create token", zap.Error(err))
				return nil, status.Error(codes.Internal, "Create token error")
			}
			header := metadata.Pairs("auth", token)
			grpc.SetHeader(ctx, header)
		}
	} else {
		userID := uuid.New().String()
		token, err := createJWTToken(userID)
		if err != nil {
			logger.Log.Error("cannot create token", zap.Error(err))
			return nil, status.Error(codes.Internal, "Create token error")
		}
		header := metadata.Pairs("auth", token)
		grpc.SetHeader(ctx, header)
	}
	originalURL := req.GetOriginalUrl()
	if !validationURL(originalURL) {
		logger.Log.Error("Bad request, no valid url")
		return nil, status.Error(codes.InvalidArgument, "Bad request")
	}
	urlID := strings.Split(uuid.New().String(), "-")[0]
	var shortURL string
	if s.cfg.BaseURL == "" {
		shortURL = "http://" + s.cfg.ServerAddress + "/" + urlID
	} else {
		shortURL = "http://" + s.cfg.BaseURL + "/" + urlID
	}

	if err := s.sService.SaveURL(originalURL, shortURL, userID); err != nil {
		if errors.Is(err, storage.ErrMemStorageError) {
			shortDBURL, err := s.sService.GetShortByOriginal(originalURL)
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

			shortDBURL, err := s.sService.GetShortByOriginal(originalURL)
			if err != nil {
				logger.Log.Error("Error when read from base: ", zap.Error(err))
				return nil, status.Error(codes.Internal, "Error when read from base")
			}
			return &shortenergrpcv1.ShortenerURLResponce{ShortUrl: shortDBURL}, nil
		}
	}
	return &shortenergrpcv1.ShortenerURLResponce{ShortUrl: shortURL}, nil
}

func (s *ShortenerGRPCServer) ShortenerJSON(ctx context.Context, req *shortenergrpcv1.ShortenerJSONRequest) (*shortenergrpcv1.ShortenerJSONResponce, error) {
	var userID string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get("auth")
		if len(values) > 0 {
			token := values[0]
			userID := GetUID(token)
			if userID == "" {
				logger.Log.Error("User id from token is empty")
				return nil, status.Error(codes.Unauthenticated, "User id from token is empty")
			}
			header := metadata.Pairs("auth", token)
			grpc.SetHeader(ctx, header)
		} else {
			userID := uuid.New().String()
			token, err := createJWTToken(userID)
			if err != nil {
				logger.Log.Error("cannot create token", zap.Error(err))
				return nil, status.Error(codes.Internal, "Create token error")
			}
			header := metadata.Pairs("auth", token)
			grpc.SetHeader(ctx, header)
		}
	} else {
		userID := uuid.New().String()
		token, err := createJWTToken(userID)
		if err != nil {
			logger.Log.Error("cannot create token", zap.Error(err))
			return nil, status.Error(codes.Internal, "Create token error")
		}
		header := metadata.Pairs("auth", token)
		grpc.SetHeader(ctx, header)
	}

	var modelURL models.RequestURLJson
	err := json.Unmarshal([]byte(req.OrignalUrl), &modelURL)
	if err != nil {
		logger.Log.Debug("cannot decod boby json", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal error")
	}

	if !validationURL(string(modelURL.URLAddres)) {
		logger.Log.Error("Bad request, no valid url")
		return nil, status.Error(codes.InvalidArgument, "Bad request")
	}

	urlID := strings.Split(uuid.New().String(), "-")[0]
	var shortURL string
	if s.cfg.BaseURL == "" {
		shortURL = "http://" + s.cfg.ServerAddress + "/" + urlID
	} else {
		shortURL = "http://" + s.cfg.BaseURL + "/" + urlID
	}

	if err := s.sService.SaveURL(modelURL.URLAddres, shortURL, userID); err != nil {
		if errors.Is(err, storage.ErrMemStorageError) {
			shortDBURL, err := s.sService.GetShortByOriginal(modelURL.URLAddres)
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

			shortDBURL, err := s.sService.GetShortByOriginal(modelURL.URLAddres)
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

func (s *ShortenerGRPCServer) CheckDBConnection(ctx context.Context, req *shortenergrpcv1.CheckDBConnectionRequest) (*shortenergrpcv1.CheckDBConnectionResponce, error) {
	err := s.sService.CheckDBConnection()
	if err != nil {
		logger.Log.Error("Error check db connect", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal error")
	}
	return &shortenergrpcv1.CheckDBConnectionResponce{}, nil
}

func (s *ShortenerGRPCServer) GetAllURLs(ctx context.Context, req *shortenergrpcv1.GetAllURLsRequest) (*shortenergrpcv1.GetAllURLsResponce, error) {
	var userID string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get("auth")
		if len(values) > 0 {
			token := values[0]
			userID := GetUID(token)
			if userID == "" {
				logger.Log.Error("User id from token is empty")
				return nil, status.Error(codes.Unauthenticated, "User id from token is empty")
			}
			header := metadata.Pairs("auth", token)
			grpc.SetHeader(ctx, header)
		} else {
			userID := uuid.New().String()
			token, err := createJWTToken(userID)
			if err != nil {
				logger.Log.Error("cannot create token", zap.Error(err))
				return nil, status.Error(codes.Internal, "Create token error")
			}
			header := metadata.Pairs("auth", token)
			grpc.SetHeader(ctx, header)
		}
	} else {
		userID := uuid.New().String()
		token, err := createJWTToken(userID)
		if err != nil {
			logger.Log.Error("cannot create token", zap.Error(err))
			return nil, status.Error(codes.Internal, "Create token error")
		}
		header := metadata.Pairs("auth", token)
		grpc.SetHeader(ctx, header)
	}

	urls, err := s.sService.GetAllURLsByID(userID)
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

func (s *ShortenerGRPCServer) InsertBatch(ctx context.Context, req *shortenergrpcv1.InsertBatchRequest) (*shortenergrpcv1.InsertBatchResponce, error) {
	var userID string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get("auth")
		if len(values) > 0 {
			token := values[0]
			userID := GetUID(token)
			if userID == "" {
				logger.Log.Error("User id from token is empty")
				return nil, status.Error(codes.Unauthenticated, "User id from token is empty")
			}
			header := metadata.Pairs("auth", token)
			grpc.SetHeader(ctx, header)
		} else {
			userID := uuid.New().String()
			token, err := createJWTToken(userID)
			if err != nil {
				logger.Log.Error("cannot create token", zap.Error(err))
				return nil, status.Error(codes.Internal, "Create token error")
			}
			header := metadata.Pairs("auth", token)
			grpc.SetHeader(ctx, header)
		}
	} else {
		userID := uuid.New().String()
		token, err := createJWTToken(userID)
		if err != nil {
			logger.Log.Error("cannot create token", zap.Error(err))
			return nil, status.Error(codes.Internal, "Create token error")
		}
		header := metadata.Pairs("auth", token)
		grpc.SetHeader(ctx, header)
	}

	var modelURL []models.RequestBatchURLModel
	err := json.Unmarshal([]byte(req.UrlsJson), &modelURL)
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
		if validationURL(v.OriginalURL) {
			urlID := strings.Split(uuid.New().String(), "-")[0]
			var shortURL string
			if s.cfg.BaseURL == "" {
				shortURL = "http://" + s.cfg.ServerAddress + "/" + urlID
			} else {
				shortURL = "http://" + s.cfg.BaseURL + "/" + urlID
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

	if err := s.sService.SaveURLBatch(bantchValues); err != nil {
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

func (s *ShortenerGRPCServer) DeleteURL(ctx context.Context, req *shortenergrpcv1.DeleteURLRequest) (*shortenergrpcv1.DeleteURLResponce, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get("auth")
		if len(values) > 0 {
			token := values[0]
			userID := GetUID(token)
			if userID == "" {
				logger.Log.Error("User id from token is empty")
				return nil, status.Error(codes.Unauthenticated, "User id from token is empty")
			}
			header := metadata.Pairs("auth", token)
			grpc.SetHeader(ctx, header)
		} else {
			userID := uuid.New().String()
			token, err := createJWTToken(userID)
			if err != nil {
				logger.Log.Error("cannot create token", zap.Error(err))
				return nil, status.Error(codes.Internal, "Create token error")
			}
			header := metadata.Pairs("auth", token)
			grpc.SetHeader(ctx, header)
		}
	} else {
		userID := uuid.New().String()
		token, err := createJWTToken(userID)
		if err != nil {
			logger.Log.Error("cannot create token", zap.Error(err))
			return nil, status.Error(codes.Internal, "Create token error")
		}
		header := metadata.Pairs("auth", token)
		grpc.SetHeader(ctx, header)
	}

	var moodel []string
	if err := json.Unmarshal([]byte(req.Urls), &moodel); err != nil {
		logger.Log.Debug("cannot decod boby json", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal error")
	}

	go s.sService.DeleteURL(moodel, s.cfg.ServerAddress)
	return &shortenergrpcv1.DeleteURLResponce{}, nil
}
func (s *ShortenerGRPCServer) ServiceStat(ctx context.Context, req *shortenergrpcv1.ServiceStatRequest) (*shortenergrpcv1.ServiceStatResponce, error) {
	if s.cfg.TrustedSubnet == "" {
		return nil, status.Error(codes.PermissionDenied, "Permission denied")
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.PermissionDenied, "Permission denied")
	}
	realIP := md.Get("X-Real-IP")
	headerIP := net.ParseIP(realIP[0])
	_, IPnet, _ := net.ParseCIDR(s.cfg.TrustedSubnet)
	if !IPnet.Contains(headerIP) {
		return nil, status.Error(codes.PermissionDenied, "Permission denied")
	}

	statModel, err := s.sService.GetServiceStat()
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

func createJWTToken(uuid string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 3)),
		},
		UserID: uuid,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func GetUID(tokenString string) string {
	claim := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claim, func(t *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})
	if err != nil {
		return ""
	}

	if !token.Valid {
		return ""
	}

	return claim.UserID
}

func validationURL(URL string) bool {
	if strings.HasPrefix(URL, "http://") || strings.HasPrefix(URL, "https://") {
		return true
	}
	return false
}
