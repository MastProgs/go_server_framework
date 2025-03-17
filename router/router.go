package router

import (
	"net/http"

	workerpool "go_server_framework/core"
)

var (
	Pool    *workerpool.WorkerPool
	Manager *RouterManager
)

// SetupRouter는 라우터를 설정하고 반환합니다
func SetupRouter() *http.ServeMux {
	// 라우터 관리자 생성
	Manager = NewRouterManager()

	return Manager.MainRouter
}

// RegisterServices는 서비스 등록 함수를 호출합니다
// 이 함수는 init 패키지에서 호출됩니다
func RegisterServices(registerFunc func(*RouterManager)) {
	if Manager != nil {
		registerFunc(Manager)
	}
}
