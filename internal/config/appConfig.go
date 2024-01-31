// Модуль config - структуры для конфигурации сервиса из файла и в следствии всей конфигурации.
package config

import (
	"encoding/json"
	"errors"
	"flag"
	"os"

	"github.com/Dorrrke/shortener-url/internal/logger"
	"github.com/caarlos0/env/v10"
	"go.uber.org/zap"
)

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

	if err := env.Parse(&cfg); err != nil {
		logger.Log.Error("env is empty or not exist", zap.Error(err))
	}
	logger.Log.Info("parsed cfg from env", zap.Any("cfg", cfg))

	if cfg.ServerAddress == "" && cfg.BaseURL == "" && cfg.DatabaseDsn == "" && cfg.FileStoragePath == "" {
		fileConfig, err := uploadConfigFromFile(cfgFilePath)
		if err != nil {
			logger.Log.Error("config parsing from file error", zap.Error(err))
			return &AppConfig{
				ServerAddress:   ":8080",
				BaseURL:         "",
				FileStoragePath: "",
				DatabaseDsn:     "",
				EnableHTTPS:     cfg.EnableHTTPS,
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
		return AppConfig{}, errors.New("Config path is empty")
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
