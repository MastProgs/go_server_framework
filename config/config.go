package config

import (
	"fmt"
	"sync"

	"github.com/BurntSushi/toml"
)

var (
	instance *Config
	once     sync.Once
)

// GetConfig는 싱글턴 패턴으로 설정 인스턴스를 반환합니다
func GetConfig() *Config {
	once.Do(func() {
		instance = &Config{}
		if err := instance.loadConfig("config.toml"); err != nil {
			panic(err)
		}
	})
	return instance
}

// loadConfig는 설정 파일을 로드합니다
func (c *Config) loadConfig(filename string) error {
	_, err := toml.DecodeFile(filename, c)
	return err
}

// GetDSN은 데이터베이스 연결 문자열을 반환합니다
func (c *Config) GetDSN() string {
	if c.Database.Type == "mysql" {
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Asia%%2FSeoul",
			c.Database.User, c.Database.Password, c.Database.Host,
			c.Database.Port, c.Database.DBName)
	}
	return ""
}
