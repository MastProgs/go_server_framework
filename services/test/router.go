package test

import (
	"go_server_framework/middleware"
	"go_server_framework/router"
)

// RegisterRoutes는 테스트 서비스의 라우트를 등록합니다
func RegisterRoutes(manager interface{}) {
	// 타입 어설션
	routerManager, ok := manager.(*router.RouterManager)
	if !ok {
		return
	}

	// 테스트 서비스 라우터 등록
	testRouter := routerManager.RegisterSubRouter("test", "/test")

	// 기본 핸들러 등록
	testRouter.Method("/ping", middleware.MethodHandler{
		Get: PingHandler,
	})

	testRouter.Method("/echo", middleware.MethodHandler{
		Post: EchoHandler,
	})

	testRouter.Method("/info", middleware.MethodHandler{
		Get: InfoHandler,
	})

	// 로그인 핸들러 등록
	testRouter.Method("/login", middleware.MethodHandler{
		Post: LoginHandler,
	})

	// 보호된 API 그룹 생성
	protectedGroup := testRouter.Group("/protected")

	// JWT 인증 미들웨어 적용
	protectedGroup.Use(middleware.JWTAuthMiddleware)

	// 보호된 엔드포인트 등록
	protectedGroup.Method("/data", middleware.MethodHandler{
		Get: ProtectedHandler,
	})

	// 서브 그룹 생성 예제
	apiGroup := testRouter.Group("/api")

	// API 그룹에 핸들러 등록
	apiGroup.Method("/ping", middleware.MethodHandler{
		Get: PingHandler,
	})

	apiGroup.Method("/echo", middleware.MethodHandler{
		Post: EchoHandler,
	})
}
