package auth

import (
	"go_server_framework/database"
	"go_server_framework/loghandle"
	"go_server_framework/network"
	"go_server_framework/packet_processor"
	"go_server_framework/protocol"
	"sync"
	"time"
)

// 인증 서비스
type AuthService struct {
	gameID   protocol.GameID
	name     string
	db       *database.Repository
	sessions map[uint64]*network.PlayerSession // 연결별 세션
	handlers map[protocol.PacketType]packet_processor.PacketHandler

	// 설정
	sessionTimeout time.Duration
	maxSessions    int

	// 제어
	mu        sync.RWMutex
	isRunning bool
}

func NewAuthService(db *database.Repository) *AuthService {
	return &AuthService{
		gameID:         protocol.GAME_AUTH,
		name:           "AuthService",
		db:             db,
		sessions:       make(map[uint64]*network.PlayerSession),
		handlers:       make(map[protocol.PacketType]packet_processor.PacketHandler),
		sessionTimeout: 30 * time.Minute,
		maxSessions:    10000,
	}
}

// GameService 인터페이스 구현
func (as *AuthService) GetGameID() protocol.GameID {
	return as.gameID
}

func (as *AuthService) GetName() string {
	return as.name
}

func (as *AuthService) HandlePacket(ctx *packet_processor.PacketContext) error {
	as.mu.RLock()
	handler, exists := as.handlers[ctx.Packet.GetHeader().PacketType]
	as.mu.RUnlock()

	if !exists {
		return packet_processor.ErrHandlerNotFound
	}

	return handler.Handle(ctx)
}

func (as *AuthService) OnConnectionClosed(conn *network.Connection) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	// 세션 제거
	if session, exists := as.sessions[conn.ID]; exists {
		loghandle.Info("사용자 로그아웃: %s (연결: %d)", session.Username, conn.ID)
		delete(as.sessions, conn.ID)
	}

	return nil
}

func (as *AuthService) RegisterHandlers(registry *packet_processor.PacketHandlerRegistry) error {
	// 핸들러들 생성
	loginHandler := NewLoginHandler(as)
	logoutHandler := NewLogoutHandler(as)
	heartbeatHandler := NewHeartbeatHandler(as)

	// 로컬 핸들러 맵에 저장
	as.handlers[protocol.PACKET_AUTH_LOGIN] = loginHandler
	as.handlers[protocol.PACKET_AUTH_LOGOUT] = logoutHandler
	as.handlers[protocol.PACKET_AUTH_HEARTBEAT] = heartbeatHandler

	// 글로벌 레지스트리에 등록
	registry.RegisterHandler(loginHandler)
	registry.RegisterHandler(logoutHandler)
	registry.RegisterHandler(heartbeatHandler)

	loghandle.Info("인증 서비스 핸들러 등록 완료")
	return nil
}

func (as *AuthService) Start() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.isRunning {
		return nil
	}

	as.isRunning = true
	loghandle.Info("인증 서비스 시작")

	// 세션 정리 고루틴 시작
	go as.sessionCleanupLoop()

	return nil
}

func (as *AuthService) Stop() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if !as.isRunning {
		return nil
	}

	as.isRunning = false
	loghandle.Info("인증 서비스 종료")

	// 모든 세션 정리
	as.sessions = make(map[uint64]*network.PlayerSession)

	return nil
}

// 플레이어 인증
func (as *AuthService) AuthenticatePlayer(username, password string) (*Player, error) {
	// 데이터베이스에서 사용자 인증
	// TODO: 실제 데이터베이스 구현에 맞춰 수정
	loghandle.Debug("사용자 인증 시도: %s", username)

	// 임시 구현 (개발용)
	if username == "test" && password == "test123" {
		return &Player{
			ID:         1,
			Username:   username,
			GameAccess: []protocol.GameID{protocol.GAME_AUTH, protocol.GAME_TEST},
		}, nil
	}

	return nil, ErrInvalidCredentials
}

// 세션 생성
func (as *AuthService) CreateSession(connectionID uint64, player *Player) *network.PlayerSession {
	as.mu.Lock()
	defer as.mu.Unlock()

	session := &network.PlayerSession{
		PlayerID:     player.ID,
		Username:     player.Username,
		LoginTime:    time.Now(),
		LastActivity: time.Now(),
		GameAccess:   player.GameAccess,
	}

	as.sessions[connectionID] = session
	loghandle.Info("세션 생성: %s (연결: %d)", player.Username, connectionID)

	return session
}

// 세션 조회
func (as *AuthService) GetSession(connectionID uint64) (*network.PlayerSession, bool) {
	as.mu.RLock()
	defer as.mu.RUnlock()

	session, exists := as.sessions[connectionID]
	return session, exists
}

// 세션 제거
func (as *AuthService) RemoveSession(connectionID uint64) {
	as.mu.Lock()
	defer as.mu.Unlock()

	if session, exists := as.sessions[connectionID]; exists {
		loghandle.Info("세션 제거: %s (연결: %d)", session.Username, connectionID)
		delete(as.sessions, connectionID)
	}
}

// 세션 활성화 시간 업데이트
func (as *AuthService) UpdateSessionActivity(connectionID uint64) {
	as.mu.Lock()
	defer as.mu.Unlock()

	if session, exists := as.sessions[connectionID]; exists {
		session.LastActivity = time.Now()
	}
}

// 인증 여부 확인
func (as *AuthService) IsAuthenticated(connectionID uint64) bool {
	_, exists := as.GetSession(connectionID)
	return exists
}

// 플레이어 ID로 세션 조회
func (as *AuthService) GetSessionByPlayerID(playerID uint64) (*network.PlayerSession, uint64, bool) {
	as.mu.RLock()
	defer as.mu.RUnlock()

	for connID, session := range as.sessions {
		if session.PlayerID == playerID {
			return session, connID, true
		}
	}

	return nil, 0, false
}

// 세션 정리 루프
func (as *AuthService) sessionCleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			as.cleanupExpiredSessions()
		}

		// 서비스가 중지되면 루프 종료
		as.mu.RLock()
		running := as.isRunning
		as.mu.RUnlock()

		if !running {
			break
		}
	}
}

// 만료된 세션 정리
func (as *AuthService) cleanupExpiredSessions() {
	as.mu.Lock()
	defer as.mu.Unlock()

	now := time.Now()
	expired := []uint64{}

	for connID, session := range as.sessions {
		if now.Sub(session.LastActivity) > as.sessionTimeout {
			expired = append(expired, connID)
		}
	}

	for _, connID := range expired {
		if session, exists := as.sessions[connID]; exists {
			loghandle.Info("만료된 세션 제거: %s (연결: %d)", session.Username, connID)
			delete(as.sessions, connID)
		}
	}

	if len(expired) > 0 {
		loghandle.Info("만료된 세션 %d개 정리 완료", len(expired))
	}
}

// 서비스 통계
func (as *AuthService) GetStats() map[string]interface{} {
	as.mu.RLock()
	defer as.mu.RUnlock()

	return map[string]interface{}{
		"name":            as.name,
		"game_id":         as.gameID,
		"is_running":      as.isRunning,
		"active_sessions": len(as.sessions),
		"max_sessions":    as.maxSessions,
		"session_timeout": as.sessionTimeout.String(),
	}
}
