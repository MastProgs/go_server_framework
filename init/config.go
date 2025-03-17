package init

import (
	"go_server_framework/config"
	"go_server_framework/loghandle"
)

var Config *config.Config

// InitConfig는 애플리케이션 설정을 초기화합니다
func InitConfig() {
	// 설정 로드
	Config = config.GetConfig()

	loghandle.Info("설정 초기화 완료: 서버 포트=%d", Config.Server.Port)
}
