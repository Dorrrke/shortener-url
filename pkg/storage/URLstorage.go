package storage

import (
	"context"

	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/pkg/models"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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

func (storage URLStorage) CreateTable(ctx context.Context) error {
	createTableStr := `CREATE TABLE IF NOT EXISTS short_urls
	(
		url_id serial PRIMARY KEY,
		original character(255) NOT NULL,
		short character(255) NOT NULL
	);
	
	create UNIQUE INDEX IF NOT EXISTS original_id ON short_urls (original)`
	_, err := storage.DB.Exec(ctx, createTableStr)
	if err != nil {
		return errors.Wrap(err, "Error whitle creating table")
	}
	return nil
}

func (storage URLStorage) InsertURL(ctx context.Context, originalURL string, shortURL string) error {
	_, err := storage.DB.Exec(ctx, "INSERT INTO short_urls (original, short) values ($1, $2)", originalURL, shortURL)
	if err != nil {
		return errors.Wrap(err, "Error while inserting row in db")
	}
	return nil
}

func (storage URLStorage) GetURLByShortURL(ctx context.Context, shotURL string) (string, error) {
	logger.Log.Info("Serach shortURL: ", zap.String("1", shotURL))
	rows := storage.DB.QueryRow(ctx, "SELECT original FROM short_urls where short = $1", shotURL)
	// if err != nil {
	// 	return "", errors.Wrap(err, "Error when getting row from db")
	// }
	var result string

	if err := rows.Scan(&result); err != nil {
		return "", errors.Wrap(err, "Error parsing db info")
	}

	return result, nil

}

func (storage URLStorage) GetURLByOriginalURL(ctx context.Context, original string) (string, error) {

	rows := storage.DB.QueryRow(ctx, "SELECT short FROM short_urls where original = $1", original)
	// if err != nil {
	// 	return "", errors.Wrap(err, "Error when getting row from db")
	// }
	var result string

	if err := rows.Scan(&result); err != nil {
		return "", errors.Wrap(err, "Error parsing db info")
	}

	return result, nil

}

func (storage URLStorage) InsertBanchURL(ctx context.Context, value []models.BantchURL) error {
	tx, err := storage.DB.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	if _, err := tx.Prepare(ctx, "insert bantch", "INSERT INTO short_urls (original, short) values ($1, $2)"); err != nil {
		return err
	}

	for _, v := range value {
		if _, err := tx.Exec(ctx, "insert bantch", v.OriginalURL, v.ShortURL); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
