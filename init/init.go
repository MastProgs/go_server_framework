package init

import (
	"net/http"

	"go_server_framework/loghandle"
	"go_server_framework/loop"
)

var Router *http.ServeMux

// InitAll은 애플리케이션의 모든 컴포넌트를 초기화합니다
func InitAll() (err error) {
	// 설정 초기화 (가장 먼저 초기화해야 함)
	InitConfig()

	// 로거 초기화
	InitLogger()

	loghandle.Info("애플리케이션 초기화 시작")

	// 데이터베이스 초기화
	if err := InitDatabase(); err != nil {
		loghandle.Warn("데이터베이스 초기화 실패: %v - 데이터베이스 없이 계속 실행합니다", err)
	} else {
		loghandle.Info("데이터베이스 초기화 성공")
	}

	// 라우터 초기화
	Router = InitRouter()

	// 여기에 다른 컴포넌트 초기화 함수 호출
	// 예: InitCache(), InitMessageQueue() 등

	loghandle.Info("애플리케이션 초기화 완료")
	return nil
}

func PreInit() (err error) {
	loop.SetupCronJobs()
	return nil
}

// ShutdownAll은 애플리케이션의 모든 컴포넌트를 정리합니다
func ShutdownAll() (err error) {
	loghandle.Info("애플리케이션 종료 시작")

	// 라우터 종료 (워커 풀 포함)
	ShutdownRouter()

	// 데이터베이스 종료
	ShutdownDatabase()

	// 여기에 다른 컴포넌트 종료 함수 호출
	// 예: ShutdownCache(), ShutdownMessageQueue() 등

	// 마지막 로그 메시지
	loghandle.Info("애플리케이션 종료 완료")

	// 로거 종료 (반드시 마지막에 실행)
	loghandle.GetLogger().Close()

	return nil
}
