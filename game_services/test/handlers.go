package test

import (
	testProtocol "go_server_framework/game_services/test/protocol"
	"go_server_framework/loghandle"
	"go_server_framework/packet_processor"
	"go_server_framework/protocol"
)

// 로비 리스트 핸들러
type LobbyListHandler struct {
	packet_processor.BasePacketHandler
	testService *TestGameService
}

func NewLobbyListHandler(testService *TestGameService) *LobbyListHandler {
	return &LobbyListHandler{
		BasePacketHandler: packet_processor.NewBasePacketHandler(protocol.PACKET_TEST_LOBBY_LIST, protocol.GAME_TEST),
		testService:       testService,
	}
}

func (llh *LobbyListHandler) Handle(ctx *packet_processor.PacketContext) error {
	// 룸 리스트 조회
	roomList := llh.testService.GetRoomList()

	loghandle.Debug("로비 리스트 요청: 플레이어=%d, 룸수=%d", ctx.GetPlayerID(), len(roomList))

	// TODO: 응답 패킷 구현 및 전송
	return nil
}

// 로비 생성 핸들러
type LobbyCreateHandler struct {
	packet_processor.BasePacketHandler
	testService *TestGameService
}

func NewLobbyCreateHandler(testService *TestGameService) *LobbyCreateHandler {
	return &LobbyCreateHandler{
		BasePacketHandler: packet_processor.NewBasePacketHandler(protocol.PACKET_TEST_LOBBY_CREATE, protocol.GAME_TEST),
		testService:       testService,
	}
}

func (lch *LobbyCreateHandler) Handle(ctx *packet_processor.PacketContext) error {
	createPacket, ok := ctx.Packet.(*testProtocol.TestLobbyCreatePacket)
	if !ok {
		return protocol.ErrInvalidPacketFormat
	}

	playerID := ctx.GetPlayerID()
	username := ctx.PlayerSession.Username

	room, err := lch.testService.CreateRoom(playerID, username, createPacket.RoomName,
		createPacket.MaxUsers, createPacket.IsPrivate, createPacket.Password)

	if err != nil {
		loghandle.Error("룸 생성 실패: %v", err)
		return err
	}

	loghandle.Info("룸 생성 성공: ID=%d, 이름=%s", room.ID, room.Name)

	// TODO: 성공 응답 패킷 전송
	return nil
}

// 로비 입장 핸들러
type LobbyJoinHandler struct {
	packet_processor.BasePacketHandler
	testService *TestGameService
}

func NewLobbyJoinHandler(testService *TestGameService) *LobbyJoinHandler {
	return &LobbyJoinHandler{
		BasePacketHandler: packet_processor.NewBasePacketHandler(protocol.PACKET_TEST_LOBBY_JOIN, protocol.GAME_TEST),
		testService:       testService,
	}
}

func (ljh *LobbyJoinHandler) Handle(ctx *packet_processor.PacketContext) error {
	joinPacket, ok := ctx.Packet.(*testProtocol.TestLobbyJoinPacket)
	if !ok {
		return protocol.ErrInvalidPacketFormat
	}

	playerID := ctx.GetPlayerID()
	username := ctx.PlayerSession.Username

	room, err := ljh.testService.JoinRoom(playerID, username, joinPacket.RoomID, joinPacket.Password)
	if err != nil {
		loghandle.Error("룸 입장 실패: %v", err)
		return err
	}

	loghandle.Info("룸 입장 성공: 플레이어=%s, 룸ID=%d", username, room.ID)

	// TODO: 성공 응답 패킷 전송 및 룸 내 다른 플레이어들에게 알림
	return nil
}

// 로비 나가기 핸들러
type LobbyLeaveHandler struct {
	packet_processor.BasePacketHandler
	testService *TestGameService
}

func NewLobbyLeaveHandler(testService *TestGameService) *LobbyLeaveHandler {
	return &LobbyLeaveHandler{
		BasePacketHandler: packet_processor.NewBasePacketHandler(protocol.PACKET_TEST_LOBBY_LEAVE, protocol.GAME_TEST),
		testService:       testService,
	}
}

func (llh *LobbyLeaveHandler) Handle(ctx *packet_processor.PacketContext) error {
	playerID := ctx.GetPlayerID()

	err := llh.testService.LeaveRoom(playerID)
	if err != nil {
		loghandle.Error("룸 나가기 실패: %v", err)
		return err
	}

	loghandle.Info("룸 나가기 성공: 플레이어ID=%d", playerID)

	// TODO: 성공 응답 패킷 전송
	return nil
}

// 게임 액션 핸들러
type GameActionHandler struct {
	packet_processor.BasePacketHandler
	testService *TestGameService
}

func NewGameActionHandler(testService *TestGameService) *GameActionHandler {
	return &GameActionHandler{
		BasePacketHandler: packet_processor.NewBasePacketHandler(protocol.PACKET_TEST_GAME_ACTION, protocol.GAME_TEST),
		testService:       testService,
	}
}

func (gah *GameActionHandler) Handle(ctx *packet_processor.PacketContext) error {
	actionPacket, ok := ctx.Packet.(*testProtocol.TestGameActionPacket)
	if !ok {
		return protocol.ErrInvalidPacketFormat
	}

	playerID := ctx.GetPlayerID()

	// 플레이어가 속한 룸 조회
	room, exists := gah.testService.GetPlayerRoom(playerID)
	if !exists {
		return ErrPlayerNotInRoom
	}

	// 위치 업데이트
	if err := room.UpdatePlayerPosition(playerID, actionPacket.X, actionPacket.Y); err != nil {
		return err
	}

	loghandle.Debug("게임 액션 처리: 플레이어=%d, 액션=%s, 위치=(%.2f,%.2f)",
		playerID, actionPacket.Action, actionPacket.X, actionPacket.Y)

	// TODO: 룸 내 다른 플레이어들에게 액션 브로드캐스트
	return nil
}

// 채팅 핸들러
type ChatHandler struct {
	packet_processor.BasePacketHandler
	testService *TestGameService
}

func NewChatHandler(testService *TestGameService) *ChatHandler {
	return &ChatHandler{
		BasePacketHandler: packet_processor.NewBasePacketHandler(protocol.PACKET_TEST_CHAT_SEND, protocol.GAME_TEST),
		testService:       testService,
	}
}

func (ch *ChatHandler) Handle(ctx *packet_processor.PacketContext) error {
	chatPacket, ok := ctx.Packet.(*testProtocol.TestChatPacket)
	if !ok {
		return protocol.ErrInvalidPacketFormat
	}

	playerID := ctx.GetPlayerID()
	username := ctx.PlayerSession.Username

	// 플레이어가 속한 룸 조회
	room, exists := ch.testService.GetPlayerRoom(playerID)
	if !exists {
		return ErrPlayerNotInRoom
	}

	// 채팅 메시지 추가
	chatMsg := room.AddChatMessage(playerID, username, chatPacket.Message, chatPacket.Type, chatPacket.TargetID)

	loghandle.Debug("채팅 메시지: 플레이어=%s, 타입=%s, 메시지=%s",
		username, chatPacket.Type, chatPacket.Message)

	// TODO: 룸 내 플레이어들에게 채팅 메시지 브로드캐스트
	_ = chatMsg
	return nil
}
