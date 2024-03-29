// Модуль config - структуры для конфигурации сервиса из файла и в следствии всей конфигурации.
package config

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"strings"

	"go.uber.org/zap"

	"github.com/Dorrrke/shortener-url/internal/logger"
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
	TrustedSubnet   string `json:"trusted_subnet" env:"TRUSTED_SUBNET,required"`
}

// MustLoad - обязательная к запуску функция создающая файл конфига.
// Функция парсит переменные оркужения, флаги и данные из файла конфига.
func MustLoad() (*AppConfig, bool) {
	var cfg AppConfig

	var cfgFilePath string
	flag.StringVar(&cfgFilePath, "config", "", "config file path")
	flag.StringVar(&cfg.ServerAddress, "a", "", "address and port to run server")
	flag.StringVar(&cfg.BaseURL, "b", "", "address and port to run short URL")
	flag.StringVar(&cfg.FileStoragePath, "f", "", "storage file path")
	flag.StringVar(&cfg.DatabaseDsn, "d", "", "databse addr")
	flag.StringVar(&cfg.TrustedSubnet, "t", "", "trusted subnet")
	httpsFlag := flag.Bool("s", false, "use https server")
	grpcEnable := flag.Bool("g", false, "use https server")
	flag.Parse()
	cfg.EnableHTTPS = *httpsFlag

	if strings.Contains(cfg.BaseURL, "http://") {
		correctURL := strings.Replace(cfg.BaseURL, "http://", "", -1)
		cfg.BaseURL = correctURL
	}

	logger.Log.Info("config from flags", zap.Any("cfg", cfg))
	logger.Log.Info("config file path", zap.String("cfg file", cfgFilePath))

	if cfg.BaseURL == "" {
		cfg.BaseURL = os.Getenv("BASE_URL")
	}
	if cfg.DatabaseDsn == "" {
		cfg.DatabaseDsn = os.Getenv("DATABASE_DSN")
	}
	if cfg.TrustedSubnet == "" {
		cfg.TrustedSubnet = os.Getenv("TRUSTED_SUBNET")
	}
	if cfg.FileStoragePath == "" {
		cfg.FileStoragePath = os.Getenv("FILE_STORAGE_PATH")
		if cfg.FileStoragePath == "" {
			cfg.FileStoragePath = FilePath
		}
	}
	if cfg.ServerAddress == "" {
		cfg.ServerAddress = os.Getenv("SERVER_ADDRESS")
	}

	if cfg.TrustedSubnet == "" {
		cfg.TrustedSubnet = os.Getenv("TRUSTED_SUBNET")
	}

	if cfg.ServerAddress == "" && cfg.BaseURL == "" && cfg.DatabaseDsn == "" {
		logger.Log.Info("Check config file")
		fileConfig, err := uploadConfigFromFile(cfgFilePath)
		if err != nil {
			logger.Log.Error("config parsing from file error", zap.Error(err))
			return &cfg, *grpcEnable
		}
		return &fileConfig, *grpcEnable
	}

	return &cfg, *grpcEnable
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
