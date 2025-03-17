package router

import (
	"net/http"
	"strings"

	workerpool "go_server_framework/core"
	"go_server_framework/loghandle"
	"go_server_framework/middleware"
)

// RouterManager는 여러 서브 라우터를 관리합니다
type RouterManager struct {
	MainRouter *http.ServeMux
	SubRouters map[string]*SubRouter
	Pool       *workerpool.WorkerPool
}

// NewRouterManager는 새로운 라우터 관리자를 생성합니다
func NewRouterManager() *RouterManager {
	if Pool == nil {
		Pool = workerpool.NewWorkerPool()
	}

	return &RouterManager{
		MainRouter: http.NewServeMux(),
		SubRouters: make(map[string]*SubRouter),
		Pool:       Pool,
	}
}

// RegisterSubRouter는 새로운 서브 라우터를 등록합니다
func (rm *RouterManager) RegisterSubRouter(name string, basePath string) *SubRouter {
	if _, exists := rm.SubRouters[name]; exists {
		loghandle.Warn("이미 존재하는 라우터 이름입니다: %s", name)
		return rm.SubRouters[name]
	}

	// 경로가 슬래시로 시작하지 않으면 슬래시를 추가합니다.
	if basePath != "" && !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	subRouter := NewSubRouter(name, basePath, rm.Pool)

	// 기본 미들웨어 적용 - 요청 로깅
	subRouter.Use(middleware.RequestLoggerMiddleware)

	rm.SubRouters[name] = subRouter

	// 디버그 로그 추가
	loghandle.Info("서브 라우터 등록: 이름=%s, 경로=%s", name, basePath)

	// 경로 끝에 슬래시를 추가하여 모든 하위 경로를 매칭
	// Go의 http.ServeMux는 /test/ 패턴으로 등록하면 /test/로 시작하는 모든 경로를 매칭합니다.
	rm.MainRouter.Handle(basePath+"/", http.StripPrefix(basePath, subRouter))

	// 경로 자체도 등록 (예: /test)
	rm.MainRouter.Handle(basePath, http.StripPrefix(basePath, subRouter))

	return subRouter
}

// GetSubRouter는 이름으로 서브 라우터를 가져옵니다
func (rm *RouterManager) GetSubRouter(name string) *SubRouter {
	router, exists := rm.SubRouters[name]
	if !exists {
		loghandle.Warn("존재하지 않는 라우터 이름입니다: %s", name)
		return nil
	}
	return router
}

// Start는 워커 풀을 시작합니다
func (rm *RouterManager) Start() {
	rm.Pool.Start()
	loghandle.Info("라우터 관리자 시작됨")
}

// Stop은 워커 풀을 중지합니다
func (rm *RouterManager) Stop() {
	rm.Pool.Stop()
	loghandle.Info("라우터 관리자 중지됨")
}

// Handler는 메인 라우터의 핸들러를 반환합니다
func (rm *RouterManager) Handler() http.Handler {
	return rm.MainRouter
}
