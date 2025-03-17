package services

import (
	"go_server_framework/loghandle"
	"go_server_framework/router"
	"go_server_framework/services/test"
)

// RegisterAllServices는 모든 서비스를 등록합니다
func RegisterAllServices(manager *router.RouterManager) {
	// 테스트 서비스 등록
	test.RegisterRoutes(manager)
	loghandle.Info("테스트 서비스 등록 완료")

	// 여기에 추가 서비스 등록
	// 예: user.RegisterRoutes(manager)
	// 예: blog.RegisterRoutes(manager)
	// 예: shop.RegisterRoutes(manager)
}
