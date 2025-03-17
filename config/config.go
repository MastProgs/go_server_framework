package config

import (
	"fmt"
	"sync"

	"github.com/BurntSushi/toml"
)

// Config는 애플리케이션 설정을 정의합니다
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Log      LogConfig      `toml:"log"`
	Database DatabaseConfig `toml:"database"`
	JWT      JWTConfig      `toml:"jwt"`
}

// ServerConfig는 서버 관련 설정을 정의합니다
type ServerConfig struct {
	Port    int    `toml:"port"`
	Host    string `toml:"host"`
	Timeout int    `toml:"timeout"` // 초 단위
}

// LogConfig는 로깅 관련 설정을 정의합니다
type LogConfig struct {
	Level     string        `toml:"level"`      // debug, info, warn, error
	Format    string        `toml:"format"`     // text, json
	Output    []string      `toml:"output"`     // console, file
	AddSource bool          `toml:"add_source"` // 소스 정보 추가 여부
	File      FileLogConfig `toml:"file"`
}

// FileLogConfig는 파일 로깅 관련 설정을 정의합니다
type FileLogConfig struct {
	Filename   string `toml:"filename"`
	MaxSize    int    `toml:"max_size"` // MB
	MaxBackups int    `toml:"max_backups"`
	MaxAge     int    `toml:"max_age"` // 일
	Compress   bool   `toml:"compress"`
}

// DatabaseConfig는 데이터베이스 관련 설정을 정의합니다
type DatabaseConfig struct {
	Type            string `toml:"type"` // mysql
	Host            string `toml:"host"`
	Port            int    `toml:"port"`
	User            string `toml:"user"`
	Password        string `toml:"password"`
	DBName          string `toml:"dbname"`
	MaxOpenConns    int    `toml:"max_open_conns"`
	MaxIdleConns    int    `toml:"max_idle_conns"`
	ConnMaxLifetime int    `toml:"conn_max_lifetime"` // 초 단위
}

// JWTConfig는 JWT 관련 설정을 정의합니다
type JWTConfig struct {
	Secret            string `toml:"secret"`
	Expiration        int    `toml:"expiration"`         // 초 단위
	RefreshExpiration int    `toml:"refresh_expiration"` // 초 단위
}

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
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			c.Database.User, c.Database.Password, c.Database.Host,
			c.Database.Port, c.Database.DBName)
	}
	return ""
}
