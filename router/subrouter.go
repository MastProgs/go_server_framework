package router

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	workerpool "go_server_framework/core"
	"go_server_framework/loghandle"
	"go_server_framework/middleware"
)

// SubRouter는 특정 경로에 대한 라우터를 정의합니다
type SubRouter struct {
	Name       string                            // 라우터 이름
	BasePath   string                            // 기본 경로
	Mux        *http.ServeMux                    // 서브 라우터의 ServeMux
	Pool       *workerpool.WorkerPool            // 워커 풀
	Middleware []func(http.Handler) http.Handler // 미들웨어 체인
}

// NewSubRouter는 새로운 서브 라우터를 생성합니다
func NewSubRouter(name string, basePath string, pool *workerpool.WorkerPool) *SubRouter {
	return &SubRouter{
		Name:       name,
		BasePath:   basePath,
		Mux:        http.NewServeMux(),
		Pool:       pool,
		Middleware: []func(http.Handler) http.Handler{},
	}
}

// Use는 미들웨어를 서브 라우터에 추가합니다
func (sr *SubRouter) Use(middleware func(http.Handler) http.Handler) {
	sr.Middleware = append(sr.Middleware, middleware)
}

// HandleFunc는 경로에 대한 핸들러 함수를 등록합니다
func (sr *SubRouter) HandleFunc(pattern string, handler http.HandlerFunc) {
	// 패턴이 슬래시로 시작하지 않으면 슬래시를 추가합니다.
	if pattern != "" && !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	// 디버그 로그 추가
	loghandle.Info("라우터 '%s': 핸들러 함수 경로 '%s' 등록됨 (전체 경로: %s%s)",
		sr.Name, pattern, sr.BasePath, pattern)

	sr.Mux.HandleFunc(pattern, handler)
}

// Handle은 경로에 대한 핸들러를 등록합니다
func (sr *SubRouter) Handle(pattern string, handler http.Handler) {
	// 패턴이 슬래시로 시작하지 않으면 슬래시를 추가합니다.
	if pattern != "" && !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	// 디버그 로그 추가
	loghandle.Info("라우터 '%s': 핸들러 경로 '%s' 등록됨 (전체 경로: %s%s)",
		sr.Name, pattern, sr.BasePath, pattern)

	sr.Mux.Handle(pattern, handler)
}

// Method는 HTTP 메서드별 핸들러를 등록합니다
func (sr *SubRouter) Method(pattern string, handlers middleware.MethodHandler) {
	// 경로 패턴 처리
	// 패턴이 슬래시로 시작하지 않으면 슬래시를 추가합니다.
	if pattern != "" && !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	// 디버그 로그 추가
	loghandle.Info("라우터 '%s': 메서드 핸들러 경로 '%s' 등록됨 (전체 경로: %s%s)",
		sr.Name, pattern, sr.BasePath, pattern)

	sr.Mux.Handle(pattern, middleware.WorkerPoolMiddleware(sr.Pool, handlers))
}

// Group은 새로운 서브 그룹을 생성합니다
func (sr *SubRouter) Group(pattern string) *SubRouter {
	// 패턴이 슬래시로 시작하지 않으면 슬래시를 추가합니다.
	if pattern != "" && !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	subPath := path.Join(sr.BasePath, pattern)
	subRouter := NewSubRouter(sr.Name+pattern, subPath, sr.Pool)

	// 부모 라우터의 미들웨어를 상속 (로깅 미들웨어 제외)
	for _, mw := range sr.Middleware {
		// 로깅 미들웨어는 중복 적용하지 않음
		if !isLoggingMiddleware(mw) {
			subRouter.Middleware = append(subRouter.Middleware, mw)
		}
	}

	// 로깅 미들웨어가 없으면 추가
	if !hasLoggingMiddleware(subRouter.Middleware) && hasLoggingMiddleware(sr.Middleware) {
		// 부모에서 로깅 미들웨어 찾아서 추가
		for _, mw := range sr.Middleware {
			if isLoggingMiddleware(mw) {
				subRouter.Middleware = append(subRouter.Middleware, mw)
				break
			}
		}
	}

	// 서브 라우터를 현재 라우터에 등록
	// 패턴 끝에 슬래시를 추가하여 모든 하위 경로를 매칭
	sr.Mux.Handle(pattern+"/", http.StripPrefix(pattern, subRouter))

	// 패턴 자체도 등록
	sr.Mux.Handle(pattern, http.StripPrefix(pattern, subRouter))

	loghandle.Info("라우터 '%s': 서브그룹 '%s' 생성됨 (전체 경로: %s)", sr.Name, pattern, subPath)
	return subRouter
}

// isLoggingMiddleware는 주어진 미들웨어가 로깅 미들웨어인지 확인합니다
func isLoggingMiddleware(mw func(http.Handler) http.Handler) bool {
	// 함수 포인터 비교를 위해 미들웨어 함수의 주소를 문자열로 변환하여 비교
	// 이 방법은 완벽하지 않지만, 현재 상황에서는 충분히 작동합니다
	mwStr := fmt.Sprintf("%p", mw)
	loggerStr := fmt.Sprintf("%p", middleware.RequestLoggerMiddleware)
	return strings.Contains(mwStr, loggerStr) || strings.Contains(mwStr, "RequestLoggerMiddleware")
}

// hasLoggingMiddleware는 미들웨어 목록에 로깅 미들웨어가 있는지 확인합니다
func hasLoggingMiddleware(middlewares []func(http.Handler) http.Handler) bool {
	for _, mw := range middlewares {
		if isLoggingMiddleware(mw) {
			return true
		}
	}
	return false
}

// applyMiddleware는 핸들러에 미들웨어를 적용합니다
func applyMiddleware(handler http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}

// Handler는 서브 라우터의 최종 핸들러를 반환합니다
func (sr *SubRouter) Handler() http.Handler {
	return applyMiddleware(sr.Mux, sr.Middleware...)
}

// ServeHTTP는 HTTP 요청을 처리합니다
func (sr *SubRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 미들웨어가 적용된 핸들러로 요청 전달
	sr.Handler().ServeHTTP(w, r)
}
