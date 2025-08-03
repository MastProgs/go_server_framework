package packet_processor

import (
	"fmt"
	"go_server_framework/network"
	"go_server_framework/protocol"
	"sync"
	"time"
)

// 패킷 핸들러 인터페이스
type PacketHandler interface {
	Handle(ctx *PacketContext) error
	GetPacketType() protocol.PacketType
	GetGameID() protocol.GameID
}

// 패킷 처리 컨텍스트
type PacketContext struct {
	Connection    *network.Connection
	Packet        protocol.Packet
	PlayerSession *network.PlayerSession
	GameID        protocol.GameID
	Response      chan protocol.Packet // 응답 패킷 전송용
	
	// 메타데이터
	ProcessStartTime time.Time
	RequestID        string
}

// 새 패킷 컨텍스트 생성
func NewPacketContext(conn *network.Connection, packet protocol.Packet) *PacketContext {
	return &PacketContext{
		Connection:       conn,
		Packet:          packet,
		PlayerSession:   conn.PlayerSession,
		GameID:          protocol.GetGameIDFromPacketType(packet.GetHeader().PacketType),
		Response:        make(chan protocol.Packet, 1),
		ProcessStartTime: time.Now(),
		RequestID:       generateRequestID(),
	}
}

// 응답 패킷 전송
func (ctx *PacketContext) SendResponse(packet protocol.Packet) error {
	return ctx.Connection.SendPacket(packet)
}

// 에러 응답 전송
func (ctx *PacketContext) SendError(errorCode int, message string) error {
	// TODO: 공통 에러 응답 패킷 구현
	return nil
}

// 인증 여부 확인
func (ctx *PacketContext) IsAuthenticated() bool {
	return ctx.PlayerSession != nil
}

// 플레이어 ID 조회
func (ctx *PacketContext) GetPlayerID() uint64 {
	if ctx.PlayerSession != nil {
		return ctx.PlayerSession.PlayerID
	}
	return 0
}

// 게임 접근 권한 확인
func (ctx *PacketContext) HasGameAccess(gameID protocol.GameID) bool {
	if ctx.PlayerSession == nil {
		return false
	}
	
	for _, allowedGame := range ctx.PlayerSession.GameAccess {
		if allowedGame == gameID {
			return true
		}
	}
	return false
}

// 요청 ID 생성
func generateRequestID() string {
	// TODO: UUID 또는 다른 고유 ID 생성
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// 기본 패킷 핸들러 구조체
type BasePacketHandler struct {
	packetType protocol.PacketType
	gameID     protocol.GameID
}

func NewBasePacketHandler(packetType protocol.PacketType, gameID protocol.GameID) BasePacketHandler {
	return BasePacketHandler{
		packetType: packetType,
		gameID:     gameID,
	}
}

func (bph *BasePacketHandler) GetPacketType() protocol.PacketType {
	return bph.packetType
}

func (bph *BasePacketHandler) GetGameID() protocol.GameID {
	return bph.gameID
}

// 패킷 핸들러 레지스트리
type PacketHandlerRegistry struct {
	handlers map[protocol.PacketType]PacketHandler
	mu       sync.RWMutex
}

func NewPacketHandlerRegistry() *PacketHandlerRegistry {
	return &PacketHandlerRegistry{
		handlers: make(map[protocol.PacketType]PacketHandler),
	}
}

// 핸들러 등록
func (phr *PacketHandlerRegistry) RegisterHandler(handler PacketHandler) {
	phr.mu.Lock()
	defer phr.mu.Unlock()
	
	phr.handlers[handler.GetPacketType()] = handler
}

// 핸들러 조회
func (phr *PacketHandlerRegistry) GetHandler(packetType protocol.PacketType) (PacketHandler, bool) {
	phr.mu.RLock()
	defer phr.mu.RUnlock()
	
	handler, exists := phr.handlers[packetType]
	return handler, exists
}

// 모든 핸들러 조회
func (phr *PacketHandlerRegistry) GetAllHandlers() map[protocol.PacketType]PacketHandler {
	phr.mu.RLock()
	defer phr.mu.RUnlock()
	
	result := make(map[protocol.PacketType]PacketHandler)
	for k, v := range phr.handlers {
		result[k] = v
	}
	return result
}

// 게임별 핸들러 조회
func (phr *PacketHandlerRegistry) GetHandlersByGame(gameID protocol.GameID) []PacketHandler {
	phr.mu.RLock()
	defer phr.mu.RUnlock()
	
	var result []PacketHandler
	for _, handler := range phr.handlers {
		if handler.GetGameID() == gameID {
			result = append(result, handler)
		}
	}
	return result
}

// 핸들러 제거
func (phr *PacketHandlerRegistry) UnregisterHandler(packetType protocol.PacketType) {
	phr.mu.Lock()
	defer phr.mu.Unlock()
	
	delete(phr.handlers, packetType)
}

// 등록된 핸들러 수
func (phr *PacketHandlerRegistry) Count() int {
	phr.mu.RLock()
	defer phr.mu.RUnlock()
	
	return len(phr.handlers)
}