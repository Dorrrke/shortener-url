// Модуль config - структуры для конфигурации сервиса из файла и в следствии всей конфигурации.
package config

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"strings"

	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/caarlos0/env/v10"
	"go.uber.org/zap"
)

// FilePath — константа с названием файла для хранения данных при отсутствии подключения к бд.
const FilePath string = "short-url-db.json"

// AppConfig - сттруктура для хранения конфигураци и конфигурации сервиса.
type AppConfig struct {
	ServerAddress   string `json:"server_address" env:"SERVER_ADDRESS,required"`
	BaseURL         string `json:"base_url" env:"BASE_URL,required"`
	FileStoragePath string `json:"file_storage_path" env:"FILE_STORAGE_PATH,required"`
	DatabaseDsn     string `json:"database_dsn" env:"DATABASE_DSN,required"`
	EnableHTTPS     bool   `json:"enable_https"`
}

// MustLoad - обязательная к запуску функция создающая файл конфига.
// Функция парсит переменные оркужения, флаги и данные из файла конфига.
func MustLoad() *AppConfig {
	var cfg AppConfig

	var cfgFilePath string
	flag.StringVar(&cfgFilePath, "config", "", "config file path")
	flag.StringVar(&cfg.ServerAddress, "a", "", "address and port to run server")
	flag.StringVar(&cfg.BaseURL, "b", "", "address and port to run short URL")
	flag.StringVar(&cfg.FileStoragePath, "f", "", "storage file path")
	flag.StringVar(&cfg.DatabaseDsn, "d", "", "databse addr")
	httpsFlag := flag.Bool("s", false, "use https server")
	flag.Parse()
	cfg.EnableHTTPS = *httpsFlag

	if strings.Contains(cfg.BaseURL, "http://") {
		correctURL := strings.Replace(cfg.BaseURL, "http://", "", -1)
		cfg.BaseURL = correctURL
	}

	logger.Log.Info("config from flags", zap.Any("cfg", cfg))
	logger.Log.Info("config file path", zap.String("cfg file", cfgFilePath))

	if cfg.FileStoragePath == "" {
		cfg.FileStoragePath = FilePath
	}

	var tempCfg AppConfig
	if err := env.Parse(&tempCfg); err == nil {
		logger.Log.Info("env does not err;")
		logger.Log.Info("parsed cfg from env", zap.Any("cfg", tempCfg))
		if tempCfg.FileStoragePath == "" {
			if cfg.FileStoragePath == "" {
				tempCfg.FileStoragePath = FilePath
				return &tempCfg
			}
			tempCfg.FileStoragePath = cfg.FileStoragePath
			return &tempCfg
		}
		return &tempCfg
	}

	if cfg.ServerAddress == "" && cfg.BaseURL == "" && cfg.DatabaseDsn == "" && cfg.FileStoragePath == "" {
		logger.Log.Info("Check config file")
		fileConfig, err := uploadConfigFromFile(cfgFilePath)
		if err != nil {
			logger.Log.Error("config parsing from file error", zap.Error(err))
			return &AppConfig{
				ServerAddress: ":8080",
				BaseURL:       "",
				DatabaseDsn:   "",
				EnableHTTPS:   cfg.EnableHTTPS,
			}
		}
		return &fileConfig
	}

	return &cfg
}

// uploadConfigFromFile - функция составления конфига из файла.
func uploadConfigFromFile(cfgPath string) (AppConfig, error) {
	var config AppConfig
	if cfgPath == "" {
		return AppConfig{}, errors.New("config path is empty")
	}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		logger.Log.Error("Config file is not exist", zap.Error(err))
		panic(err)
	}
	f, err := os.Open(cfgPath)
	if err != nil {
		logger.Log.Error("Error open file", zap.Error(err))
		panic(err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	if err := dec.Decode(&config); err != nil {
		logger.Log.Error("error parse config file")
		panic(err)
	}
	logger.Log.Info("config from json", zap.Any("config", config))
	return config, nil
}
