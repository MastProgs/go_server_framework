package test

import (
	"go_server_framework/loghandle"
	"go_server_framework/network"
	"go_server_framework/packet_processor"
	"go_server_framework/protocol"
	"sync"
	"sync/atomic"
)

// 테스트 게임 서비스
type TestGameService struct {
	gameID    protocol.GameID
	name      string
	rooms     map[uint64]*TestRoom
	roomIDGen uint64 // atomic
	handlers  map[protocol.PacketType]packet_processor.PacketHandler

	// 플레이어 -> 룸 매핑
	playerRooms map[uint64]uint64

	// 제어
	mu        sync.RWMutex
	isRunning bool
}

func NewTestGameService() *TestGameService {
	return &TestGameService{
		gameID:      protocol.GAME_TEST,
		name:        "TestGameService",
		rooms:       make(map[uint64]*TestRoom),
		handlers:    make(map[protocol.PacketType]packet_processor.PacketHandler),
		playerRooms: make(map[uint64]uint64),
	}
}

// GameService 인터페이스 구현
func (tgs *TestGameService) GetGameID() protocol.GameID {
	return tgs.gameID
}

func (tgs *TestGameService) GetName() string {
	return tgs.name
}

func (tgs *TestGameService) HandlePacket(ctx *packet_processor.PacketContext) error {
	tgs.mu.RLock()
	handler, exists := tgs.handlers[ctx.Packet.GetHeader().PacketType]
	tgs.mu.RUnlock()

	if !exists {
		return packet_processor.ErrHandlerNotFound
	}

	return handler.Handle(ctx)
}

func (tgs *TestGameService) OnConnectionClosed(conn *network.Connection) error {
	if conn.PlayerSession == nil {
		return nil
	}

	playerID := conn.PlayerSession.PlayerID

	// 플레이어가 속한 룸에서 제거
	tgs.mu.Lock()
	roomID, exists := tgs.playerRooms[playerID]
	if exists {
		delete(tgs.playerRooms, playerID)

		if room, roomExists := tgs.rooms[roomID]; roomExists {
			room.RemovePlayer(playerID)

			// 룸이 비었으면 제거
			if len(room.GetAllPlayers()) == 0 {
				delete(tgs.rooms, roomID)
				loghandle.Info("빈 룸 제거: ID=%d", roomID)
			}
		}
	}
	tgs.mu.Unlock()

	loghandle.Info("플레이어 연결 종료 처리: ID=%d", playerID)
	return nil
}

func (tgs *TestGameService) RegisterHandlers(registry *packet_processor.PacketHandlerRegistry) error {
	// 핸들러들 생성
	lobbyListHandler := NewLobbyListHandler(tgs)
	lobbyCreateHandler := NewLobbyCreateHandler(tgs)
	lobbyJoinHandler := NewLobbyJoinHandler(tgs)
	lobbyLeaveHandler := NewLobbyLeaveHandler(tgs)
	gameActionHandler := NewGameActionHandler(tgs)
	chatHandler := NewChatHandler(tgs)

	// 로컬 핸들러 맵에 저장
	tgs.handlers[protocol.PACKET_TEST_LOBBY_LIST] = lobbyListHandler
	tgs.handlers[protocol.PACKET_TEST_LOBBY_CREATE] = lobbyCreateHandler
	tgs.handlers[protocol.PACKET_TEST_LOBBY_JOIN] = lobbyJoinHandler
	tgs.handlers[protocol.PACKET_TEST_LOBBY_LEAVE] = lobbyLeaveHandler
	tgs.handlers[protocol.PACKET_TEST_GAME_ACTION] = gameActionHandler
	tgs.handlers[protocol.PACKET_TEST_CHAT_SEND] = chatHandler

	// 글로벌 레지스트리에 등록
	registry.RegisterHandler(lobbyListHandler)
	registry.RegisterHandler(lobbyCreateHandler)
	registry.RegisterHandler(lobbyJoinHandler)
	registry.RegisterHandler(lobbyLeaveHandler)
	registry.RegisterHandler(gameActionHandler)
	registry.RegisterHandler(chatHandler)

	loghandle.Info("테스트 게임 서비스 핸들러 등록 완료")
	return nil
}

func (tgs *TestGameService) Start() error {
	tgs.mu.Lock()
	defer tgs.mu.Unlock()

	if tgs.isRunning {
		return nil
	}

	tgs.isRunning = true
	loghandle.Info("테스트 게임 서비스 시작")

	return nil
}

func (tgs *TestGameService) Stop() error {
	tgs.mu.Lock()
	defer tgs.mu.Unlock()

	if !tgs.isRunning {
		return nil
	}

	tgs.isRunning = false
	loghandle.Info("테스트 게임 서비스 종료")

	// 모든 룸 정리
	tgs.rooms = make(map[uint64]*TestRoom)
	tgs.playerRooms = make(map[uint64]uint64)

	return nil
}

// 룸 생성
func (tgs *TestGameService) CreateRoom(ownerID uint64, username, roomName string, maxUsers int, isPrivate bool, password string) (*TestRoom, error) {
	tgs.mu.Lock()
	defer tgs.mu.Unlock()

	// 플레이어가 이미 다른 룸에 있는지 확인
	if _, exists := tgs.playerRooms[ownerID]; exists {
		return nil, ErrPlayerAlreadyInRoom
	}

	// 새 룸 ID 생성
	roomID := atomic.AddUint64(&tgs.roomIDGen, 1)

	// 룸 생성
	room := NewTestRoom(roomID, roomName, ownerID, maxUsers, isPrivate, password)

	// 방장을 룸에 추가
	if err := room.AddPlayer(ownerID, username); err != nil {
		return nil, err
	}

	// 룸 저장
	tgs.rooms[roomID] = room
	tgs.playerRooms[ownerID] = roomID

	loghandle.Info("룸 생성: ID=%d, 이름=%s, 방장=%s", roomID, roomName, username)
	return room, nil
}

// 룸 입장
func (tgs *TestGameService) JoinRoom(playerID uint64, username string, roomID uint64, password string) (*TestRoom, error) {
	tgs.mu.Lock()
	defer tgs.mu.Unlock()

	// 플레이어가 이미 다른 룸에 있는지 확인
	if _, exists := tgs.playerRooms[playerID]; exists {
		return nil, ErrPlayerAlreadyInRoom
	}

	// 룸 조회
	room, exists := tgs.rooms[roomID]
	if !exists {
		return nil, ErrRoomNotFound
	}

	// 비밀번호 확인
	if room.IsPrivate && room.Password != password {
		return nil, ErrInvalidRoomPassword
	}

	// 룸에 플레이어 추가
	if err := room.AddPlayer(playerID, username); err != nil {
		return nil, err
	}

	// 플레이어 -> 룸 매핑 추가
	tgs.playerRooms[playerID] = roomID

	loghandle.Info("룸 입장: 플레이어=%s, 룸ID=%d", username, roomID)
	return room, nil
}

// 룸 나가기
func (tgs *TestGameService) LeaveRoom(playerID uint64) error {
	tgs.mu.Lock()
	defer tgs.mu.Unlock()

	// 플레이어가 속한 룸 조회
	roomID, exists := tgs.playerRooms[playerID]
	if !exists {
		return ErrPlayerNotInRoom
	}

	// 룸에서 플레이어 제거
	room, roomExists := tgs.rooms[roomID]
	if roomExists {
		room.RemovePlayer(playerID)

		// 룸이 비었으면 제거
		if len(room.GetAllPlayers()) == 0 {
			delete(tgs.rooms, roomID)
			loghandle.Info("빈 룸 제거: ID=%d", roomID)
		}
	}

	// 플레이어 -> 룸 매핑 제거
	delete(tgs.playerRooms, playerID)

	loghandle.Info("룸 나가기: 플레이어ID=%d, 룸ID=%d", playerID, roomID)
	return nil
}

// 플레이어가 속한 룸 조회
func (tgs *TestGameService) GetPlayerRoom(playerID uint64) (*TestRoom, bool) {
	tgs.mu.RLock()
	defer tgs.mu.RUnlock()

	roomID, exists := tgs.playerRooms[playerID]
	if !exists {
		return nil, false
	}

	room, roomExists := tgs.rooms[roomID]
	return room, roomExists
}

// 모든 룸 목록 조회
func (tgs *TestGameService) GetRoomList() []map[string]interface{} {
	tgs.mu.RLock()
	defer tgs.mu.RUnlock()

	roomList := make([]map[string]interface{}, 0, len(tgs.rooms))
	for _, room := range tgs.rooms {
		// 비공개 룸은 목록에서 제외
		if !room.IsPrivate {
			roomList = append(roomList, room.GetRoomInfo())
		}
	}

	return roomList
}

// 특정 룸 조회
func (tgs *TestGameService) GetRoom(roomID uint64) (*TestRoom, bool) {
	tgs.mu.RLock()
	defer tgs.mu.RUnlock()

	room, exists := tgs.rooms[roomID]
	return room, exists
}

// 서비스 통계
func (tgs *TestGameService) GetStats() map[string]interface{} {
	tgs.mu.RLock()
	defer tgs.mu.RUnlock()

	return map[string]interface{}{
		"name":          tgs.name,
		"game_id":       tgs.gameID,
		"is_running":    tgs.isRunning,
		"total_rooms":   len(tgs.rooms),
		"total_players": len(tgs.playerRooms),
	}
}
