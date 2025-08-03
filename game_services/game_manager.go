package game_services

import (
	"go_server_framework/database"
	"go_server_framework/game_services/auth"
	"go_server_framework/game_services/test"
	"go_server_framework/loghandle"
	"go_server_framework/packet_processor"
)

// 게임 서비스 매니저 (기존 services.go 역할)
type GameServiceManager struct {
	processor *packet_processor.PacketProcessor
	db        *database.Repository

	// 등록된 서비스들
	authService *auth.AuthService
	testService *test.TestGameService
}

func NewGameServiceManager(processor *packet_processor.PacketProcessor, db *database.Repository) *GameServiceManager {
	return &GameServiceManager{
		processor: processor,
		db:        db,
	}
}

// 모든 게임 서비스 등록 (기존 RegisterAllServices와 유사)
func (gsm *GameServiceManager) RegisterAllGameServices() error {
	loghandle.Info("게임 서비스 등록 시작")

	// 인증 서비스 등록
	gsm.authService = auth.NewAuthService(gsm.db)
	if err := gsm.processor.RegisterGameService(gsm.authService); err != nil {
		return err
	}
	loghandle.Info("인증 서비스 등록 완료")

	// 테스트 게임 서비스 등록
	gsm.testService = test.NewTestGameService()
	if err := gsm.processor.RegisterGameService(gsm.testService); err != nil {
		return err
	}
	loghandle.Info("테스트 게임 서비스 등록 완료")

	loghandle.Info("모든 게임 서비스 등록 완료")
	return nil
}

// 서비스 매니저 종료
func (gsm *GameServiceManager) Shutdown() {
	loghandle.Info("게임 서비스 매니저 종료")

	// 각 서비스들은 패킷 프로세서에서 자동으로 정리됨
}

// 인증 서비스 조회
func (gsm *GameServiceManager) GetAuthService() *auth.AuthService {
	return gsm.authService
}

// 테스트 게임 서비스 조회
func (gsm *GameServiceManager) GetTestService() *test.TestGameService {
	return gsm.testService
}
