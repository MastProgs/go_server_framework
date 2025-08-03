package packet_processor

import (
	"fmt"
	"go_server_framework/loghandle"
	"go_server_framework/network"
	"go_server_framework/protocol"
	"sync"
)

// 게임 서비스 인터페이스
type GameService interface {
	GetGameID() protocol.GameID
	GetName() string
	HandlePacket(ctx *PacketContext) error
	OnConnectionClosed(conn *network.Connection) error
	RegisterHandlers(registry *PacketHandlerRegistry) error
	Start() error
	Stop() error
}

// 게임 라우터 (멀티 게임 패킷 분배)
type GameRouter struct {
	services        map[protocol.GameID]GameService
	handlerRegistry *PacketHandlerRegistry
	mu              sync.RWMutex
}

func NewGameRouter() *GameRouter {
	return &GameRouter{
		services:        make(map[protocol.GameID]GameService),
		handlerRegistry: NewPacketHandlerRegistry(),
	}
}

// 게임 서비스 등록
func (gr *GameRouter) RegisterGameService(service GameService) error {
	gr.mu.Lock()
	defer gr.mu.Unlock()

	gameID := service.GetGameID()

	// 중복 등록 체크
	if _, exists := gr.services[gameID]; exists {
		return fmt.Errorf("game service already registered: %s (ID: %d)", service.GetName(), gameID)
	}

	// 서비스 등록
	gr.services[gameID] = service

	// 핸들러 등록
	if err := service.RegisterHandlers(gr.handlerRegistry); err != nil {
		delete(gr.services, gameID)
		return fmt.Errorf("failed to register handlers for %s: %w", service.GetName(), err)
	}

	// 서비스 시작
	if err := service.Start(); err != nil {
		delete(gr.services, gameID)
		return fmt.Errorf("failed to start service %s: %w", service.GetName(), err)
	}

	loghandle.Info("게임 서비스 등록: %s (ID: %d)", service.GetName(), gameID)
	return nil
}

// 게임 서비스 제거
func (gr *GameRouter) UnregisterGameService(gameID protocol.GameID) error {
	gr.mu.Lock()
	defer gr.mu.Unlock()

	service, exists := gr.services[gameID]
	if !exists {
		return fmt.Errorf("game service not found: %d", gameID)
	}

	// 서비스 중지
	if err := service.Stop(); err != nil {
		loghandle.Error("게임 서비스 중지 실패: %s, %v", service.GetName(), err)
	}

	// 해당 게임의 핸들러들 제거
	handlers := gr.handlerRegistry.GetHandlersByGame(gameID)
	for _, handler := range handlers {
		gr.handlerRegistry.UnregisterHandler(handler.GetPacketType())
	}

	delete(gr.services, gameID)

	loghandle.Info("게임 서비스 제거: %s (ID: %d)", service.GetName(), gameID)
	return nil
}

// 패킷 라우팅 (핵심 기능)
func (gr *GameRouter) RoutePacket(conn *network.Connection, packet protocol.Packet) error {
	// 패킷 타입으로 게임 ID 추출
	gameID := protocol.GetGameIDFromPacketType(packet.GetHeader().PacketType)

	// 패킷 컨텍스트 생성
	ctx := NewPacketContext(conn, packet)

	// 인증이 필요한 패킷인지 확인
	if gameID != protocol.GAME_AUTH && !ctx.IsAuthenticated() {
		return gr.sendAuthRequiredError(conn)
	}

	// 게임 접근 권한 확인
	if gameID != protocol.GAME_AUTH && !ctx.HasGameAccess(gameID) {
		return gr.sendAccessDeniedError(conn, gameID)
	}

	// 해당 게임 서비스로 라우팅
	gr.mu.RLock()
	service, exists := gr.services[gameID]
	gr.mu.RUnlock()

	if !exists {
		return gr.sendServiceNotAvailableError(conn, gameID)
	}

	// 게임 서비스에서 패킷 처리
	return service.HandlePacket(ctx)
}

// 연결 종료 처리
func (gr *GameRouter) HandleConnectionClosed(conn *network.Connection) {
	gr.mu.RLock()
	defer gr.mu.RUnlock()

	// 모든 게임 서비스에 연결 종료 알림
	for _, service := range gr.services {
		if err := service.OnConnectionClosed(conn); err != nil {
			loghandle.Error("연결 종료 처리 실패: %s, %v", service.GetName(), err)
		}
	}
}

// 라우터 시작
func (gr *GameRouter) Start() error {
	loghandle.Info("게임 라우터 시작")

	// 등록된 모든 서비스 시작
	gr.mu.RLock()
	defer gr.mu.RUnlock()

	for _, service := range gr.services {
		if err := service.Start(); err != nil {
			loghandle.Error("게임 서비스 시작 실패: %s, %v", service.GetName(), err)
			return err
		}
	}

	loghandle.Info("게임 라우터 시작 완료 (서비스: %d개)", len(gr.services))
	return nil
}

// 라우터 종료
func (gr *GameRouter) Stop() error {
	loghandle.Info("게임 라우터 종료 시작")

	gr.mu.Lock()
	defer gr.mu.Unlock()

	// 모든 서비스 중지
	for gameID, service := range gr.services {
		if err := service.Stop(); err != nil {
			loghandle.Error("게임 서비스 중지 실패: %s, %v", service.GetName(), err)
		}
		delete(gr.services, gameID)
	}

	loghandle.Info("게임 라우터 종료 완료")
	return nil
}

// 등록된 서비스 목록 조회
func (gr *GameRouter) GetRegisteredServices() []GameService {
	gr.mu.RLock()
	defer gr.mu.RUnlock()

	services := make([]GameService, 0, len(gr.services))
	for _, service := range gr.services {
		services = append(services, service)
	}
	return services
}

// 특정 게임 서비스 조회
func (gr *GameRouter) GetGameService(gameID protocol.GameID) (GameService, bool) {
	gr.mu.RLock()
	defer gr.mu.RUnlock()

	service, exists := gr.services[gameID]
	return service, exists
}

// 핸들러 레지스트리 조회
func (gr *GameRouter) GetHandlerRegistry() *PacketHandlerRegistry {
	return gr.handlerRegistry
}

// 라우터 통계
func (gr *GameRouter) GetStats() map[string]interface{} {
	gr.mu.RLock()
	defer gr.mu.RUnlock()

	stats := map[string]interface{}{
		"registered_services": len(gr.services),
		"registered_handlers": gr.handlerRegistry.Count(),
	}

	// 서비스별 정보 추가
	services := make(map[string]interface{})
	for gameID, service := range gr.services {
		services[fmt.Sprintf("game_%d", gameID)] = map[string]interface{}{
			"name":    service.GetName(),
			"game_id": gameID,
		}
	}
	stats["services"] = services

	return stats
}

// 에러 응답 헬퍼 함수들
func (gr *GameRouter) sendAuthRequiredError(conn *network.Connection) error {
	// TODO: 인증 필요 에러 패킷 구현
	loghandle.Debug("인증 필요 에러 전송: 연결 ID=%d", conn.ID)
	return nil
}

func (gr *GameRouter) sendAccessDeniedError(conn *network.Connection, gameID protocol.GameID) error {
	// TODO: 접근 거부 에러 패킷 구현
	loghandle.Debug("접근 거부 에러 전송: 연결 ID=%d, 게임 ID=%d", conn.ID, gameID)
	return nil
}

func (gr *GameRouter) sendServiceNotAvailableError(conn *network.Connection, gameID protocol.GameID) error {
	// TODO: 서비스 사용 불가 에러 패킷 구현
	loghandle.Debug("서비스 사용 불가 에러 전송: 연결 ID=%d, 게임 ID=%d", conn.ID, gameID)
	return nil
}
