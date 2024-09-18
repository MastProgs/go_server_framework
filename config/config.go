package config

import (
	"encoding/json"
	"os"
	"sync"
)

type Config struct {
	Server struct {
		Port int `json:"port"`
	} `json:"server"`
	Database struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		DBName   string `json:"dbname"`
	} `json:"database"`
	// 추가 설정 필드들...
}

var (
	instance *Config
	once     sync.Once
)

func GetConfig() *Config {
	once.Do(func() {
		instance = &Config{}
		if err := instance.loadConfig("config.json"); err != nil {
			panic(err)
		}
	})
	return instance
}

func (c *Config) loadConfig(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(c)
}
