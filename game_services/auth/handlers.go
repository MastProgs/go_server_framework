package auth

import (
	"go_server_framework/loghandle"
	"go_server_framework/packet_processor"
	"go_server_framework/protocol"
	"time"
)

// 로그인 핸들러
type LoginHandler struct {
	packet_processor.BasePacketHandler
	authService *AuthService
}

func NewLoginHandler(authService *AuthService) *LoginHandler {
	return &LoginHandler{
		BasePacketHandler: packet_processor.NewBasePacketHandler(protocol.PACKET_AUTH_LOGIN, protocol.GAME_AUTH),
		authService:       authService,
	}
}

func (lh *LoginHandler) Handle(ctx *packet_processor.PacketContext) error {
	// 패킷 타입 확인
	loginPacket, ok := ctx.Packet.(*protocol.LoginPacket)
	if !ok {
		loghandle.Error("잘못된 로그인 패킷 타입")
		return lh.sendLoginResponse(ctx, false, "잘못된 패킷 타입", 0, nil)
	}

	// 이미 인증된 연결인지 확인
	if ctx.IsAuthenticated() {
		loghandle.Warn("이미 인증된 연결에서 로그인 시도: 연결 ID=%d", ctx.Connection.ID)
		return lh.sendLoginResponse(ctx, false, "이미 로그인됨", 0, nil)
	}

	// 사용자 인증
	player, err := lh.authService.AuthenticatePlayer(loginPacket.Username, loginPacket.Password)
	if err != nil {
		loghandle.Info("로그인 실패: %s, %v", loginPacket.Username, err)
		return lh.sendLoginResponse(ctx, false, "로그인 실패: 사용자명 또는 비밀번호가 잘못되었습니다", 0, nil)
	}

	// 중복 로그인 체크
	// if existingSession, _, exists := lh.authService.GetSessionByPlayerID(player.ID); exists {
	if _, _, exists := lh.authService.GetSessionByPlayerID(player.ID); exists {
		loghandle.Warn("중복 로그인 시도: %s (기존 세션 존재)", player.Username)
		// 기존 세션을 끊거나 새 로그인을 거부할 수 있음
		// 여기서는 새 로그인을 허용하고 기존 세션을 유지
	}

	// 세션 생성
	session := lh.authService.CreateSession(ctx.Connection.ID, player)

	// 연결에 세션 설정
	ctx.Connection.SetPlayerSession(session)

	// 플레이어 로그인 시간 업데이트
	player.UpdateLastLogin()

	loghandle.Info("로그인 성공: %s (연결: %d)", player.Username, ctx.Connection.ID)

	// 성공 응답 전송
	return lh.sendLoginResponse(ctx, true, "로그인 성공", player.ID, player.GameAccess)
}

func (lh *LoginHandler) sendLoginResponse(ctx *packet_processor.PacketContext, success bool, message string, playerID uint64, gameAccess []protocol.GameID) error {
	response := protocol.NewLoginResponsePacket(success, message, playerID, gameAccess)
	return ctx.SendResponse(response)
}

// 로그아웃 핸들러
type LogoutHandler struct {
	packet_processor.BasePacketHandler
	authService *AuthService
}

func NewLogoutHandler(authService *AuthService) *LogoutHandler {
	return &LogoutHandler{
		BasePacketHandler: packet_processor.NewBasePacketHandler(protocol.PACKET_AUTH_LOGOUT, protocol.GAME_AUTH),
		authService:       authService,
	}
}

func (loh *LogoutHandler) Handle(ctx *packet_processor.PacketContext) error {
	// 인증 확인
	if !ctx.IsAuthenticated() {
		return loh.sendLogoutResponse(ctx, false, "인증되지 않은 사용자")
	}

	// 패킷 타입 확인
	logoutPacket, ok := ctx.Packet.(*protocol.LogoutPacket)
	if !ok {
		loghandle.Error("잘못된 로그아웃 패킷 타입")
		return loh.sendLogoutResponse(ctx, false, "잘못된 패킷 타입")
	}

	// 플레이어 ID 확인
	if logoutPacket.PlayerID != ctx.GetPlayerID() {
		loghandle.Warn("잘못된 플레이어 ID로 로그아웃 시도: 요청=%d, 실제=%d",
			logoutPacket.PlayerID, ctx.GetPlayerID())
		return loh.sendLogoutResponse(ctx, false, "권한 없음")
	}

	username := ctx.PlayerSession.Username

	// 세션 제거
	loh.authService.RemoveSession(ctx.Connection.ID)

	// 연결에서 세션 제거
	ctx.Connection.ClearPlayerSession()

	loghandle.Info("로그아웃 완료: %s (연결: %d, 사유: %s)",
		username, ctx.Connection.ID, logoutPacket.Reason)

	// 성공 응답 전송
	return loh.sendLogoutResponse(ctx, true, "로그아웃 완료")
}

func (loh *LogoutHandler) sendLogoutResponse(ctx *packet_processor.PacketContext, success bool, message string) error {
	// TODO: 로그아웃 응답 패킷 구현
	loghandle.Debug("로그아웃 응답: %s", message)
	return nil
}

// 하트비트 핸들러
type HeartbeatHandler struct {
	packet_processor.BasePacketHandler
	authService *AuthService
}

func NewHeartbeatHandler(authService *AuthService) *HeartbeatHandler {
	return &HeartbeatHandler{
		BasePacketHandler: packet_processor.NewBasePacketHandler(protocol.PACKET_AUTH_HEARTBEAT, protocol.GAME_AUTH),
		authService:       authService,
	}
}

func (hh *HeartbeatHandler) Handle(ctx *packet_processor.PacketContext) error {
	// 패킷 타입 확인
	heartbeatPacket, ok := ctx.Packet.(*protocol.HeartbeatPacket)
	if !ok {
		loghandle.Error("잘못된 하트비트 패킷 타입")
		return nil // 하트비트는 에러 응답 없이 무시
	}

	// 연결 활성화 시간 업데이트
	ctx.Connection.UpdateActivity()

	// 인증된 사용자의 경우 세션 활성화 시간도 업데이트
	if ctx.IsAuthenticated() {
		hh.authService.UpdateSessionActivity(ctx.Connection.ID)
	}

	// 하트비트 응답 전송
	return hh.sendHeartbeatResponse(ctx, heartbeatPacket.Timestamp)
}

func (hh *HeartbeatHandler) sendHeartbeatResponse(ctx *packet_processor.PacketContext, clientTimestamp uint64) error {
	serverTime := uint64(time.Now().Unix())
	latency := serverTime - clientTimestamp

	// TODO: 하트비트 응답 패킷 구현
	loghandle.Debug("하트비트 응답: 지연시간=%dms", latency)
	return nil
}

// 세션 체크 핸들러 (클라이언트가 세션 상태를 확인할 때 사용)
type SessionCheckHandler struct {
	packet_processor.BasePacketHandler
	authService *AuthService
}

func NewSessionCheckHandler(authService *AuthService) *SessionCheckHandler {
	return &SessionCheckHandler{
		BasePacketHandler: packet_processor.NewBasePacketHandler(protocol.PACKET_AUTH_SESSION_CHECK, protocol.GAME_AUTH),
		authService:       authService,
	}
}

func (sch *SessionCheckHandler) Handle(ctx *packet_processor.PacketContext) error {
	isValid := ctx.IsAuthenticated()

	if isValid {
		// 세션 활성화 시간 업데이트
		sch.authService.UpdateSessionActivity(ctx.Connection.ID)
		loghandle.Debug("세션 유효 확인: 연결 ID=%d, 플레이어 ID=%d",
			ctx.Connection.ID, ctx.GetPlayerID())
	} else {
		loghandle.Debug("세션 무효: 연결 ID=%d", ctx.Connection.ID)
	}

	// TODO: 세션 체크 응답 패킷 구현
	return nil
}
