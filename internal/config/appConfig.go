// Модуль config - структуры для конфигурации сервиса из файла и в следствии всей конфигурации.
package config

// AppConfig - сттруктура для хранения конфигураци и конфигурации сервиса.
type AppConfig struct {
	ServerAddress   string `json:"server_address"`
	BaseURL         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DatabaseDsn     string `json:"database_dsn"`
	EnableHTTPS     bool   `json:"enable_https"`
}
