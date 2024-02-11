// В пакете storage харнится интерфейс хранилища (Storage) и две реализации интерфейса.
package storage

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/Dorrrke/shortener-url/pkg/models"
)

// ErrMemStorageError ошибка при попыттке записать уже существующий url в MemStorage.
var ErrMemStorageError = errors.New("url is alredy shorted")

// Storage - итерфейс хранилища с необходимыми методами.
type Storage interface {
	InsertURL(ctx context.Context, originalURL string, shortURL string, userID string) error
	GetAllUrls(ctx context.Context, userID string) ([]models.URLModel, error)
	GetOriginalURLByShort(ctx context.Context, shotURL string) (string, bool, error)
	GetShortByOriginalURL(ctx context.Context, original string) (string, error)
	CheckDBConnect(ctx context.Context) error
	CreateTable(ctx context.Context) error
	InsertBanchURL(ctx context.Context, value []models.BantchURL) error
	SetDeleteURLStatus(ctx context.Context, value []string) error
	GetStats(ctx context.Context) (int, int, error)
	Clear(ctx context.Context) error
}

// MemStorage - реализация интерфейса Storage без базы данных, при помощи map - MemStorage
type MemStorage struct {
	URLMap map[string]string
}

// InsertURL - метод сохранения url в map.
func (s *MemStorage) InsertURL(ctx context.Context, originalURL string, shortURL string, userID string) error {
	if s.URLMap == nil {
		return errors.New("Map is not init")
	}
	conflict := false
	for _, v := range s.URLMap {
		if v == originalURL {
			conflict = true
		}
	}
	if conflict {
		return ErrMemStorageError
	}
	s.URLMap[shortURL] = originalURL
	return nil
}

// GetOriginalURLByShort - метод получения оригинального url из map по сокращенному url.
func (s *MemStorage) GetOriginalURLByShort(ctx context.Context, shotURL string) (string, bool, error) {
	if len(s.URLMap) == 0 {
		return "", false, errors.New("Mem Storage is empty")
	}
	return s.URLMap[shotURL], false, nil
}

// GetShortByOriginalURL - метод получения сокращенного url из map по оригинальному url.
func (s *MemStorage) GetShortByOriginalURL(ctx context.Context, original string) (string, error) {
	var key string
	for k, v := range s.URLMap {
		if v == original {
			key = k
		}
	}
	if key == "" {
		return "", errors.New("Short url not find")
	}
	return key, nil
}

// CheckDBConnect - метод проверки подключения к базе данных.
// Так как это MemStorage возвращает ошибку, что бд не подключена.
func (s *MemStorage) CheckDBConnect(ctx context.Context) error {
	return errors.New("DataBase is not init")
}

// CreateTable - метод создания таблицы в базе данных.
// Так как это MemStorage возвращает ошибку, что бд не подключена.
func (s *MemStorage) CreateTable(ctx context.Context) error {
	return errors.New("DataBase is not init")
}

// SetDeleteURLStatus - метод установки статуса Delete в базе данных.
// Так как это MemStorage возвращает ошибку, что бд не подключена.
func (s *MemStorage) SetDeleteURLStatus(ctx context.Context, value []string) error {
	return errors.New("DataBase is not init")
}

// GetAllUrls - метод получения количества пользователей сервиса и количество всех сокращенных URL.
// Так как это MemStorage возвращает ошибку, что бд не подключена.
func (s *MemStorage) GetStats(ctx context.Context) (int, int, error) {
	return -1, -1, errors.New("DataBase is not init")
}

// GetAllUrls - метод получения всех сокращенных url пользвателя из бд.
// Так как это MemStorage возвращает ошибку, что бд не подключена.
func (s *MemStorage) GetAllUrls(ctx context.Context, userID string) ([]models.URLModel, error) {
	return nil, errors.New("DataBase is not init")
}

// InsertBanchURL - метод сохраниения нескольких url в map.
func (s *MemStorage) InsertBanchURL(ctx context.Context, value []models.BantchURL) error {
	if len(s.URLMap) == 0 {
		return errors.New("Mem Storage is empty")
	}
	for _, v := range value {
		s.URLMap[v.ShortURL] = v.OriginalURL
	}
	return nil
}

// Clear - метод очистки всех таблиц в бд.
// Так как это MemStorage возвращает ошибку, что бд не подключена.
func (s *MemStorage) Clear(ctx context.Context) error {
	return errors.New("DataBase is not init")
}

// DBStorage - реализация интерфейса Storage с базой данных - DBStorage.
// Использутся база данных PostgreSQL, драйвер pgx.
type DBStorage struct {
	// DB - ссылка на пул подключений к postgre.
	DB *pgxpool.Pool
}

// InsertURL - метод сохранинеия данных в бд.
func (s *DBStorage) InsertURL(ctx context.Context, originalURL string, shortURL string, userID string) error {
	_, err := s.DB.Exec(ctx, "INSERT INTO short_urls (original, short, uid) values ($1, $2, $3)", originalURL, shortURL, userID)
	if err != nil {
		return errors.Wrap(err, "Error while inserting row in db")
	}
	return nil
}

// GetOriginalURLByShort - метод получения оригинального url по сокращенному из базы данных.
func (s *DBStorage) GetOriginalURLByShort(ctx context.Context, shotURL string) (string, bool, error) {
	logger.Log.Info("Serach shortURL: ", zap.String("1", shotURL))
	rows := s.DB.QueryRow(ctx, "SELECT original, deleted FROM short_urls where short = $1", shotURL)
	// if err != nil {
	// 	return "", errors.Wrap(err, "Error when getting row from db")
	// }
	var original string
	var deleted bool

	if err := rows.Scan(&original, &deleted); err != nil {
		return "", false, errors.Wrap(err, "Error parsing db info")
	}

	return original, deleted, nil
}

// GetShortByOriginalURL - метод получения сокращенного url по оригинальному из бд.
func (s *DBStorage) GetShortByOriginalURL(ctx context.Context, original string) (string, error) {
	rows := s.DB.QueryRow(ctx, "SELECT short FROM short_urls where original = $1", original)
	// if err != nil {
	// 	return "", errors.Wrap(err, "Error when getting row from db")
	// }
	var result string

	if err := rows.Scan(&result); err != nil {
		return "", errors.Wrap(err, "Error parsing db info")
	}

	return strings.TrimSpace(result), nil
}

// CheckDBConnect - метод для проверки подключения к базе данных.
func (s *DBStorage) CheckDBConnect(ctx context.Context) error {
	if err := s.DB.Ping(ctx); err != nil {
		return errors.Wrap(err, "Error while checking connection")
	}
	return nil
}

// GetAllUrls - метод получения всех сокращенных url пользователем из бд.
func (s *DBStorage) GetAllUrls(ctx context.Context, userID string) ([]models.URLModel, error) {
	rows, err := s.DB.Query(ctx, "SELECT original, short FROM short_urls where uid = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var urls []models.URLModel

	for rows.Next() {
		var url models.URLModel
		err = rows.Scan(&url.OriginalID, &url.ShortID)
		if err != nil {
			return nil, err
		}
		url.OriginalID = strings.TrimSpace(url.OriginalID)
		url.ShortID = strings.TrimSpace(url.ShortID)
		urls = append(urls, url)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return urls, nil
}

// GetAllUrls - метод получения количества пользователей сервиса и количество всех сокращенных URL.
func (s *DBStorage) GetStats(ctx context.Context) (int, int, error) {
	getStatStr := `SELECT COUNT(short), COUNT(DISTINCT uid) FROM short_urls`

	row := s.DB.QueryRow(ctx, getStatStr)
	var URLCount int
	var usersCount int

	if err := row.Scan(&URLCount, &usersCount); err != nil {
		return -1, -1, err
	}

	return URLCount, usersCount, nil

}

// CreateTable - метод создания таблицы в базе данных, если ее не существует.
func (s *DBStorage) CreateTable(ctx context.Context) error {
	createTableStr := `CREATE TABLE IF NOT EXISTS short_urls
	(
		url_id serial PRIMARY KEY,
		original character(255) NOT NULL,
		short character(255) NOT NULL,
		uid character(255) NOT NULL,
		deleted boolean NOT NULL DEFAULT false
	);
	
	create UNIQUE INDEX IF NOT EXISTS original_id ON short_urls (original)`
	_, err := s.DB.Exec(ctx, createTableStr)
	if err != nil {
		return errors.Wrap(err, "Error whitle creating table")
	}
	return nil
}

// InsertBanchURL - метод сохраниения нескольких url в базу данных.
// Используются транзакции.
func (s *DBStorage) InsertBanchURL(ctx context.Context, value []models.BantchURL) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	if _, err := tx.Prepare(ctx, "insert bantch", "INSERT INTO short_urls (original, short, uid) values ($1, $2, $3)"); err != nil {
		return err
	}

	for _, v := range value {
		if _, err := tx.Exec(ctx, "insert bantch", v.OriginalURL, v.ShortURL, v.UserID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// SetDeleteURLStatus - метод установки статуса Deleted в базе данных.
// В реализации используются транзакции.
func (s *DBStorage) SetDeleteURLStatus(ctx context.Context, value []string) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	if _, err := tx.Prepare(ctx, "delete", "UPDATE short_urls SET deleted=true WHERE short=$1"); err != nil {
		return err
	}

	for _, v := range value {
		if _, err := tx.Exec(ctx, "delete", v); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// Clear - метод очистки таблицы в базе данных.
func (s *DBStorage) Clear(ctx context.Context) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM short_urls`)
	if err != nil {
		return errors.Wrap(err, "users table err")
	}

	return tx.Commit(ctx)
}
