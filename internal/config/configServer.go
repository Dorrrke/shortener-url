package config

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

type ConfigServer struct {
	Host string
	Port int
}

func (config ConfigServer) String() string {
	return config.Host + ":" + strconv.Itoa(config.Port)
}

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

type ConfigShortURL struct {
	Host string
	Port int
}

func (config ConfigShortURL) String() string {
	return config.Host + ":" + strconv.Itoa(config.Port)
}

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

type Config struct {
	HostConfig         ConfigServer
	ShortURLHostConfig ConfigShortURL
}
