package service

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/Dorrrke/shortener-url/internal/config"
	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/internal/models"
	"github.com/Dorrrke/shortener-url/internal/storage"
)

// type Service interface {
// 	GetOriginalURL(short string) (string, bool, error)
// 	GetShortByOriginal(original string) (string, error)
// 	CheckDBConnection() error
// 	GetAllURLsByID(userID string) ([]models.URLModel, error)
// 	GetServiceStat() (models.StatModel, error)
// 	SaveURL(original string, short string, userID string) error
// 	SaveURLBatch(batch []models.BantchURL) error
// 	DeleteURL(moodel []string, host string)
// 	RestorStorage() error
// }

type ShortenerService struct {
	Config        *config.AppConfig
	storage       storage.Storage
	deleteQuereCh chan string
}

func NewService(stor storage.Storage, cfg *config.AppConfig) *ShortenerService {
	deleteCh := make(chan string, 5)
	service := ShortenerService{
		Config:        cfg,
		storage:       stor,
		deleteQuereCh: deleteCh,
	}
	go service.deleteUrls()

	return &service
}

func (ss *ShortenerService) GetOriginalURL(short string) (string, bool, error) {
	logger.Log.Info("Get from db")
	ctx := context.Background()
	originalURL, deleted, err := ss.storage.GetOriginalURLByShort(ctx, short)
	if err != nil {
		return "", false, err
	}

	return originalURL, deleted, nil
}

func (ss *ShortenerService) GetShortByOriginal(original string) (string, error) {
	logger.Log.Info("Get from db")
	ctx := context.Background()
	originalURL, err := ss.storage.GetShortByOriginalURL(ctx, original)
	if err != nil {
		return "", err
	}
	return originalURL, nil
}

func (ss *ShortenerService) CheckDBConnection() error {
	ctx := context.Background()
	if err := ss.storage.CheckDBConnect(ctx); err != nil {
		logger.Log.Error("Error check db connection", zap.Error(err))
		return err
	}
	return nil
}

func (ss *ShortenerService) GetAllURLsByID(userID string) ([]models.URLModel, error) {
	ctx := context.Background()
	userURL, err := ss.storage.GetAllUrls(ctx, userID)
	if err != nil {
		return nil, err
	}
	return userURL, nil
}

func (ss *ShortenerService) GetServiceStat() (models.StatModel, error) {
	logger.Log.Info("Get from db")
	ctx := context.Background()
	URLs, users, err := ss.storage.GetStats(ctx)
	if err != nil {
		return models.StatModel{}, err
	}
	return models.StatModel{
		URLsCount:  URLs,
		UsercCount: users,
	}, nil
}

func (ss *ShortenerService) SaveURL(original string, short string, userID string) error {
	logger.Log.Info("Save into db")
	ctx := context.Background()
	if err := ss.storage.InsertURL(ctx, original, short, userID); err != nil {
		return err
	}
	if ss.Config.FileStoragePath != "" {
		logger.Log.Info("Save into file")
		if err := writeURL(ss.Config.FileStoragePath, models.RestorURL{ShortURL: short, OriginalURL: original}); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (ss *ShortenerService) SaveURLBatch(batch []models.BantchURL) error {
	ctx := context.Background()
	if err := ss.storage.InsertBanchURL(ctx, batch); err != nil {
		return err
	}
	if ss.Config.FileStoragePath != "" {
		logger.Log.Info("Save batch into file")
		for _, v := range batch {
			if err := writeURL(ss.Config.FileStoragePath, models.RestorURL{ShortURL: v.ShortURL, OriginalURL: v.OriginalURL}); err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

func (ss *ShortenerService) DeleteURL(moodel []string, host string) {
	for _, data := range moodel {
		var deleteURL string
		if ss.Config.BaseURL == "" {
			deleteURL = "http://" + host + "/" + data
		} else {
			deleteURL = "http://" + ss.Config.BaseURL + "/" + data
		}
		ss.deleteQuereCh <- deleteURL
	}
}

func writeURL(fileName string, lastURL models.RestorURL) error {
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(file)
	data, err := json.Marshal(&lastURL)
	if err != nil {
		return errors.Wrap(err, "encode last url")
	}
	if _, err := writer.Write(data); err != nil {
		return errors.Wrap(err, "write if file last url")
	}
	if err := writer.WriteByte('\n'); err != nil {
		return errors.Wrap(err, "write in file '\n'")
	}
	writer.Flush()
	file.Close()
	return nil
}

func (ss *ShortenerService) deleteUrls() {

	var deleteQueue []string
	ctx := context.Background()
	for {
		select {
		case row := <-ss.deleteQuereCh:
			logger.Log.Info("Add url in delete quere", zap.String("url", row))
			deleteQueue = append(deleteQueue, row)
		default:
			if deleteQueue != nil {
				logger.Log.Info("Set delete status in db", zap.Any("delete quere", deleteQueue))
				if err := ss.storage.SetDeleteURLStatus(ctx, deleteQueue); err != nil {
					logger.Log.Error("Dlete status", zap.Error(err))
					continue
				}
				deleteQueue = nil
			}
		}
	}
}

// RestorStorage - функция для восстановления харнилища после перезапуска сервиса.
func (ss *ShortenerService) RestorStorage() error {
	if err := ss.storage.CheckDBConnect(context.Background()); err == nil {
		if err := ss.createTable(); err != nil {
			logger.Log.Info("Error when create table: " + err.Error())
			return errors.Wrap(err, "Error when create table: ")
		}
	}
	if ss.Config.FileStoragePath != "" {
		file, err := os.OpenFile(ss.Config.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			data := models.RestorURL{}
			err := json.Unmarshal(scanner.Bytes(), &data)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			ss.storage.InsertURL(ctx, data.ShortURL, data.OriginalURL, "")
		}
		file.Close()
		return nil
	}
	return nil
}

// CreateTable - функция создания таблиц в базе данных.
// Функция запускается при успещном подключении к базе данных.
func (ss *ShortenerService) createTable() error {
	ctx := context.Background()
	if err := ss.storage.CreateTable(ctx); err != nil {
		return err
	}
	return nil
}
