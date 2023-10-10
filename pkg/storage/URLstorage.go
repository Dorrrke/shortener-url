package storage

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

type URLCreatorGetter interface {
	CreateURL(URLId string, origURL string)
	GetOrigURL(URLId string) string
	CheckMapKey(URLId string) bool
}

type URLStorage struct {
	URLMap map[string]string
	DB     *pgx.Conn
}

func (storage *URLStorage) CreateURL(URLId string, origURL string) {
	storage.URLMap[URLId] = origURL
}

func (storage URLStorage) GetOrigURL(URLId string) string {
	// ToDo: Сделать отправку ошибки при пустой мапе
	return storage.URLMap[URLId]
}

func (storage URLStorage) CheckMapKey(URLId string) bool {
	if _, ok := storage.URLMap[URLId]; ok {
		return ok
	} else {
		return false
	}
}

func (storage URLStorage) CheckDBConnect(ctx context.Context) error {

	if err := storage.DB.Ping(ctx); err != nil {
		return errors.Wrap(err, "Error while checking connection")
	}
	return nil
}
