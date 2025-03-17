package init

import (
	"net/http"

	"go_server_framework/loghandle"
	"go_server_framework/router"
	"go_server_framework/services"
)

// InitRouter는 HTTP 라우터를 초기화합니다
func InitRouter() *http.ServeMux {
	// 라우터 설정
	r := router.SetupRouter()

	// 로그 추가
	loghandle.Info("라우터 설정 완료, 서비스 등록 시작")

	// 모든 서비스 등록
	router.RegisterServices(services.RegisterAllServices)

	// 로그 추가
	loghandle.Info("서비스 등록 완료, 라우터 관리자 시작")

	// 라우터 관리자 시작
	router.Manager.Start()

	loghandle.Info("라우터 초기화 완료")

	return r
}

// ShutdownRouter는 라우터 관련 리소스를 정리합니다
func ShutdownRouter() {
	// 라우터 관리자 종료
	router.Manager.Stop()

	loghandle.Info("라우터 종료 완료")
}
