package init

import (
	"go_server_framework/database"
	"go_server_framework/loghandle"
)

// InitDatabase는 데이터베이스 연결을 초기화합니다
func InitDatabase() error {
	// 데이터베이스 연결
	if err := database.Connect(); err != nil {
		return err
	}

	loghandle.Info("데이터베이스 초기화 완료")
	return nil
}

// GetDB는 데이터베이스 연결을 반환합니다
func GetDB() interface{} {
	return database.GetDB()
}

// ShutdownDatabase는 데이터베이스 연결을 종료합니다
func ShutdownDatabase() {
	database.Close()
}
