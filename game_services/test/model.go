package test

import (
	"sync"
	"time"
)

// 테스트 게임 룸
type TestRoom struct {
	ID        uint64
	Name      string
	OwnerID   uint64
	MaxUsers  int
	IsPrivate bool
	Password  string
	CreatedAt time.Time

	// 참가자들
	players   map[uint64]*TestPlayer
	playersMu sync.RWMutex

	// 게임 상태
	gameState TestGameState
	isPlaying bool

	// 채팅 히스토리
	chatHistory []ChatMessage
	chatMu      sync.RWMutex
}

// 테스트 게임 플레이어
type TestPlayer struct {
	PlayerID   uint64
	Username   string
	JoinedAt   time.Time
	IsReady    bool
	Position   Position
	Score      int
	IsOnline   bool
	LastAction time.Time
}

// 위치 정보
type Position struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z,omitempty"`
}

// 게임 상태
type TestGameState struct {
	Status    GameStatus             `json:"status"`
	StartTime *time.Time             `json:"start_time,omitempty"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Round     int                    `json:"round"`
	Players   map[uint64]*TestPlayer `json:"players"`
	GameData  map[string]interface{} `json:"game_data"`
}

// 게임 상태 열거형
type GameStatus int

const (
	GAME_STATUS_WAITING GameStatus = iota
	GAME_STATUS_READY
	GAME_STATUS_PLAYING
	GAME_STATUS_PAUSED
	GAME_STATUS_FINISHED
)

// 채팅 메시지
type ChatMessage struct {
	ID        uint64    `json:"id"`
	PlayerID  uint64    `json:"player_id"`
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	TargetID  uint64    `json:"target_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// 룸 생성
func NewTestRoom(id uint64, name string, ownerID uint64, maxUsers int, isPrivate bool, password string) *TestRoom {
	return &TestRoom{
		ID:        id,
		Name:      name,
		OwnerID:   ownerID,
		MaxUsers:  maxUsers,
		IsPrivate: isPrivate,
		Password:  password,
		CreatedAt: time.Now(),
		players:   make(map[uint64]*TestPlayer),
		gameState: TestGameState{
			Status:   GAME_STATUS_WAITING,
			Players:  make(map[uint64]*TestPlayer),
			GameData: make(map[string]interface{}),
		},
		chatHistory: make([]ChatMessage, 0),
	}
}

// 플레이어 추가
func (tr *TestRoom) AddPlayer(playerID uint64, username string) error {
	tr.playersMu.Lock()
	defer tr.playersMu.Unlock()

	// 이미 참가한 플레이어인지 확인
	if _, exists := tr.players[playerID]; exists {
		return ErrPlayerAlreadyInRoom
	}

	// 룸이 가득 찬지 확인
	if len(tr.players) >= tr.MaxUsers {
		return ErrRoomFull
	}

	// 새 플레이어 생성
	player := &TestPlayer{
		PlayerID:   playerID,
		Username:   username,
		JoinedAt:   time.Now(),
		IsReady:    false,
		Position:   Position{X: 0, Y: 0},
		Score:      0,
		IsOnline:   true,
		LastAction: time.Now(),
	}

	tr.players[playerID] = player
	tr.gameState.Players[playerID] = player

	return nil
}

// 플레이어 제거
func (tr *TestRoom) RemovePlayer(playerID uint64) error {
	tr.playersMu.Lock()
	defer tr.playersMu.Unlock()

	if _, exists := tr.players[playerID]; !exists {
		return ErrPlayerNotInRoom
	}

	delete(tr.players, playerID)
	delete(tr.gameState.Players, playerID)

	// 방장이 나간 경우 새 방장 선정
	if tr.OwnerID == playerID && len(tr.players) > 0 {
		for newOwnerID := range tr.players {
			tr.OwnerID = newOwnerID
			break
		}
	}

	return nil
}

// 플레이어 조회
func (tr *TestRoom) GetPlayer(playerID uint64) (*TestPlayer, bool) {
	tr.playersMu.RLock()
	defer tr.playersMu.RUnlock()

	player, exists := tr.players[playerID]
	return player, exists
}

// 모든 플레이어 조회
func (tr *TestRoom) GetAllPlayers() map[uint64]*TestPlayer {
	tr.playersMu.RLock()
	defer tr.playersMu.RUnlock()

	result := make(map[uint64]*TestPlayer)
	for id, player := range tr.players {
		result[id] = player
	}
	return result
}

// 플레이어 준비 상태 변경
func (tr *TestRoom) SetPlayerReady(playerID uint64, ready bool) error {
	tr.playersMu.Lock()
	defer tr.playersMu.Unlock()

	player, exists := tr.players[playerID]
	if !exists {
		return ErrPlayerNotInRoom
	}

	player.IsReady = ready
	return nil
}

// 플레이어 위치 업데이트
func (tr *TestRoom) UpdatePlayerPosition(playerID uint64, x, y float32) error {
	tr.playersMu.Lock()
	defer tr.playersMu.Unlock()

	player, exists := tr.players[playerID]
	if !exists {
		return ErrPlayerNotInRoom
	}

	player.Position.X = x
	player.Position.Y = y
	player.LastAction = time.Now()

	return nil
}

// 채팅 메시지 추가
func (tr *TestRoom) AddChatMessage(playerID uint64, username, message, msgType string, targetID uint64) *ChatMessage {
	tr.chatMu.Lock()
	defer tr.chatMu.Unlock()

	chatMsg := ChatMessage{
		ID:        uint64(len(tr.chatHistory) + 1),
		PlayerID:  playerID,
		Username:  username,
		Message:   message,
		Type:      msgType,
		TargetID:  targetID,
		Timestamp: time.Now(),
	}

	tr.chatHistory = append(tr.chatHistory, chatMsg)

	// 최대 100개 메시지만 유지
	if len(tr.chatHistory) > 100 {
		tr.chatHistory = tr.chatHistory[1:]
	}

	return &chatMsg
}

// 게임 시작
func (tr *TestRoom) StartGame() error {
	tr.playersMu.Lock()
	defer tr.playersMu.Unlock()

	if tr.isPlaying {
		return ErrGameAlreadyStarted
	}

	// 모든 플레이어가 준비되었는지 확인
	for _, player := range tr.players {
		if !player.IsReady {
			return ErrPlayersNotReady
		}
	}

	// 최소 플레이어 수 확인
	if len(tr.players) < 1 {
		return ErrNotEnoughPlayers
	}

	// 게임 상태 변경
	tr.gameState.Status = GAME_STATUS_PLAYING
	now := time.Now()
	tr.gameState.StartTime = &now
	tr.gameState.Round = 1
	tr.isPlaying = true

	// 플레이어 점수 초기화
	for _, player := range tr.players {
		player.Score = 0
	}

	return nil
}

// 게임 종료
func (tr *TestRoom) EndGame() {
	tr.playersMu.Lock()
	defer tr.playersMu.Unlock()

	tr.gameState.Status = GAME_STATUS_FINISHED
	now := time.Now()
	tr.gameState.EndTime = &now
	tr.isPlaying = false

	// 모든 플레이어 준비 상태 해제
	for _, player := range tr.players {
		player.IsReady = false
	}
}

// 룸 정보 조회
func (tr *TestRoom) GetRoomInfo() map[string]interface{} {
	tr.playersMu.RLock()
	defer tr.playersMu.RUnlock()

	playerList := make([]map[string]interface{}, 0, len(tr.players))
	for _, player := range tr.players {
		playerList = append(playerList, map[string]interface{}{
			"player_id": player.PlayerID,
			"username":  player.Username,
			"is_ready":  player.IsReady,
			"score":     player.Score,
			"position":  player.Position,
		})
	}

	return map[string]interface{}{
		"id":            tr.ID,
		"name":          tr.Name,
		"owner_id":      tr.OwnerID,
		"max_users":     tr.MaxUsers,
		"current_users": len(tr.players),
		"is_private":    tr.IsPrivate,
		"is_playing":    tr.isPlaying,
		"game_status":   tr.gameState.Status,
		"players":       playerList,
		"created_at":    tr.CreatedAt,
	}
}
