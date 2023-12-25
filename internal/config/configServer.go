// Пакет config - пакет хранящий в себе информацию для конфигурации сервера.
package config

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// Структура конфига с информацией об адрессе сервера.
type ConfigServer struct {
	// Адрес сервера.
	Host string
	// Порт сервера.
	Port int
}

// Метод возвращающий строку адреса типом host + port.
func (config ConfigServer) String() string {
	return config.Host + ":" + strconv.Itoa(config.Port)
}

// Метод установки значения в переменную типа ConfigServer.
// На вход поступает строка с адресом, после чего она разделяется на адрес и порт и значения устанавливаются в соответсвующий поля.
func (config *ConfigServer) Set(s string) error {

	matched, err := regexp.MatchString(`^[-a-zA-Z0-9+&@#/%?=~_|!:,.;]*[-a-zA-Z0-9+&@#/%=~_|]`, s)
	if err != nil {
		return err
	}
	if matched {
		if strings.Contains(s, "http://") {
			fullURL := strings.Replace(s, "http://", "", -1)
			fullURLSplit := strings.Split(fullURL, ":")
			port, err := strconv.Atoi(fullURLSplit[1])
			if err != nil {
				return err
			}
			config.Host = fullURLSplit[0]
			config.Port = port
			return nil
		} else {
			fullURL := strings.Split(s, ":")
			port, err := strconv.Atoi(fullURL[1])
			if err != nil {
				return err
			}
			config.Host = fullURL[0]
			config.Port = port
			return nil
		}
	} else {
		if s == "" || s == " " {
			config.Host = "localhost"
			config.Port = 8080
			return nil
		} else {
			return errors.New("need address in a form host:port")
		}
	}
}

// Структура конфига с информацией об базовом адресе сокращенных url.
type ConfigShortURL struct {
	// Адрес сервера.
	Host string
	// Порт сервера.
	Port int
}

// Метод возвращающий строку адреса типом host + port.
func (config ConfigShortURL) String() string {
	return config.Host + ":" + strconv.Itoa(config.Port)
}

// Метод установки значения в переменную типа ConfigShortURL.
// На вход поступает строка с адресом, после чего она разделяется на адрес и порт и значения устанавливаются в соответсвующий поля.
func (config *ConfigShortURL) Set(s string) error {
	matched, err := regexp.MatchString(`^[-a-zA-Z0-9+&@#/%?=~_|!:,.;]*[-a-zA-Z0-9+&@#/%=~_|]`, s)
	if err != nil {
		return err
	}
	if matched {
		if strings.Contains(s, "http://") {
			fullURL := strings.Replace(s, "http://", "", -1)
			fullURLSplit := strings.Split(fullURL, ":")
			port, err := strconv.Atoi(fullURLSplit[1])
			if err != nil {
				return err
			}
			config.Host = fullURLSplit[0]
			config.Port = port
			return nil
		} else {
			fullURL := strings.Split(s, ":")
			port, err := strconv.Atoi(fullURL[1])
			if err != nil {
				return err
			}
			config.Host = fullURL[0]
			config.Port = port
			return nil
		}
	} else {
		if s == "" || s == " " {
			config.Host = "localhost"
			config.Port = 8080
			return nil
		} else {
			return errors.New("need address in a form host:port")
		}
	}
}

// Структура хранящаяя в себе две config структуры для конфигурации сервера.
type Config struct {
	// Структура конфига с информацией об адрессе сервера.
	HostConfig ConfigServer
	// Структура конфига с информацией об базоыом адрессе сокращенных url.
	ShortURLHostConfig ConfigShortURL
}
