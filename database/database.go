package database

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go_server_framework/config"
	"go_server_framework/loghandle"

	// MySQL 드라이버
	_ "github.com/go-sql-driver/mysql"
)

// DB는 데이터베이스 연결 객체입니다
var (
	db   *sql.DB
	once sync.Once
	mu   sync.Mutex
)

// Connect는 데이터베이스에 연결합니다
func Connect() error {
	var err error
	once.Do(func() {
		err = connect()
		if err != nil {
			loghandle.Warn("데이터베이스 연결 실패: %v", err)
		}
	})
	return err
}

// connect는 실제 데이터베이스 연결을 수행합니다 (내부 함수)
func connect() error {
	// 설정 로드
	cfg := config.GetConfig()

	// MySQL 연결 문자열 생성
	dsn := cfg.GetDSN()
	if dsn == "" {
		return fmt.Errorf("데이터베이스 연결 문자열을 생성할 수 없습니다")
	}

	// 데이터베이스 연결
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("MySQL 연결 실패: %w", err)
	}

	// 연결 설정
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)

	// 연결 테스트
	if err := db.Ping(); err != nil {
		return fmt.Errorf("MySQL 연결 테스트 실패: %w", err)
	}

	loghandle.Info("MySQL 데이터베이스 연결 성공: %s", cfg.Database.DBName)
	return nil
}

// GetDB는 데이터베이스 연결을 반환합니다
func GetDB() *sql.DB {
	if db == nil {
		mu.Lock()
		defer mu.Unlock()
		if db == nil {
			if err := Connect(); err != nil {
				loghandle.Warn("데이터베이스 연결 실패: %v - nil 반환", err)
				return nil
			}
		}
	}
	return db
}

// IsConnected는 데이터베이스 연결이 활성화되어 있는지 확인합니다
func IsConnected() bool {
	if db == nil {
		return false
	}

	// 연결 상태 확인
	if err := db.Ping(); err != nil {
		loghandle.Warn("데이터베이스 연결 상태 확인 실패: %v", err)
		return false
	}

	return true
}

// Close는 데이터베이스 연결을 종료합니다
func Close() {
	mu.Lock()
	defer mu.Unlock()

	if db != nil {
		if err := db.Close(); err != nil {
			loghandle.Error("데이터베이스 연결 종료 실패: %v", err)
		} else {
			loghandle.Info("데이터베이스 연결 종료 성공")
		}
		db = nil
	} else {
		loghandle.Debug("데이터베이스 연결이 없어 종료할 필요가 없습니다")
	}
}
