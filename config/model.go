package config

// Config는 애플리케이션 설정을 정의합니다
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Log      LogConfig      `toml:"log"`
	Database DatabaseConfig `toml:"database"`
	JWT      JWTConfig      `toml:"jwt"`
}

// ServerConfig는 서버 관련 설정을 정의합니다
type ServerConfig struct {
	Port     int    `toml:"port"`
	Host     string `toml:"host"`
	Timeout  int    `toml:"timeout"` // 초 단위
	Debug    bool   `toml:"debug"`
	Certfile string `toml:"certfile"`
	Keyfile  string `toml:"keyfile"`
}

// LogConfig는 로깅 관련 설정을 정의합니다
type LogConfig struct {
	Level        string        `toml:"level"`      // debug, info, warn, error
	Format       string        `toml:"format"`     // text, json
	Output       []string      `toml:"output"`     // console, file
	AddSource    bool          `toml:"add_source"` // 소스 정보 추가 여부
	File         FileLogConfig `toml:"file"`
	AsyncEnabled bool          `toml:"async_enabled"` // 비동기 로깅 활성화 여부
	QueueSize    int           `toml:"queue_size"`    // 비동기 로깅 큐 크기
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
