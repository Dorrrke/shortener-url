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

func (storage URLStorage) CreateTable(ctx context.Context) error {
	createTableStr := `CREATE TABLE IF NOT EXISTS url_database.short_urls
	(
		url_id serial PRIMARY KEY,
		original character(255) NOT NULL,
		short character(255) NOT NULL
	)`
	_, err := storage.DB.Exec(ctx, createTableStr)
	if err != nil {
		return errors.Wrap(err, "Error whitle creating table")
	}
	return nil
}

func (storage URLStorage) InsertURL(ctx context.Context, originalURL string, shortURL string) error {
	_, err := storage.DB.Exec(ctx, "INSERT INTO url_database.short_urls (original, short) values ($1, $2)", originalURL, shortURL)
	if err != nil {
		return errors.Wrap(err, "Error while inserting row in db")
	}
	return nil
}

func (storage URLStorage) GetURLByShortURL(ctx context.Context, shotURL string) (string, error) {
	rows := storage.DB.QueryRow(ctx, "SELECT original FROM url_database.short_urls where short = $1", shotURL)
	// if err != nil {
	// 	return "", errors.Wrap(err, "Error when getting row from db")
	// }
	var result string

	if err := rows.Scan(&result); err != nil {
		return "", errors.Wrap(err, "Error parsing db info")
	}

	return result, nil

}
