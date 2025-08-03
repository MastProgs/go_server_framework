package auth

import (
	"go_server_framework/protocol"
	"time"
)

// 플레이어 모델
type Player struct {
	ID         uint64             `json:"id" db:"id"`
	Username   string             `json:"username" db:"username"`
	Password   string             `json:"-" db:"password"` // JSON에서 제외
	Email      string             `json:"email" db:"email"`
	CreatedAt  time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at" db:"updated_at"`
	LastLogin  *time.Time         `json:"last_login" db:"last_login"`
	IsActive   bool               `json:"is_active" db:"is_active"`
	GameAccess []protocol.GameID  `json:"game_access"` // 접근 가능한 게임 목록
}

// 플레이어 생성
func NewPlayer(username, password, email string) *Player {
	now := time.Now()
	return &Player{
		Username:   username,
		Password:   password, // 실제로는 해시해야 함
		Email:      email,
		CreatedAt:  now,
		UpdatedAt:  now,
		IsActive:   true,
		GameAccess: []protocol.GameID{protocol.GAME_AUTH}, // 기본적으로 인증만 가능
	}
}

// 게임 접근 권한 확인
func (p *Player) HasGameAccess(gameID protocol.GameID) bool {
	for _, allowedGame := range p.GameAccess {
		if allowedGame == gameID {
			return true
		}
	}
	return false
}

// 게임 접근 권한 추가
func (p *Player) AddGameAccess(gameID protocol.GameID) {
	if !p.HasGameAccess(gameID) {
		p.GameAccess = append(p.GameAccess, gameID)
		p.UpdatedAt = time.Now()
	}
}

// 게임 접근 권한 제거
func (p *Player) RemoveGameAccess(gameID protocol.GameID) {
	for i, allowedGame := range p.GameAccess {
		if allowedGame == gameID {
			p.GameAccess = append(p.GameAccess[:i], p.GameAccess[i+1:]...)
			p.UpdatedAt = time.Now()
			break
		}
	}
}

// 로그인 시간 업데이트
func (p *Player) UpdateLastLogin() {
	now := time.Now()
	p.LastLogin = &now
	p.UpdatedAt = now
}

// 플레이어 정보 업데이트
func (p *Player) Update() {
	p.UpdatedAt = time.Now()
}

// 로그인 요청 데이터
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 로그인 응답 데이터
type LoginResponse struct {
	Success    bool              `json:"success"`
	Message    string            `json:"message"`
	PlayerID   uint64            `json:"player_id,omitempty"`
	Username   string            `json:"username,omitempty"`
	GameAccess []protocol.GameID `json:"game_access,omitempty"`
	Token      string            `json:"token,omitempty"` // 추후 토큰 기반 인증용
}

// 로그아웃 요청 데이터
type LogoutRequest struct {
	PlayerID uint64 `json:"player_id"`
	Reason   string `json:"reason,omitempty"`
}

// 로그아웃 응답 데이터
type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// 하트비트 요청 데이터
type HeartbeatRequest struct {
	Timestamp uint64 `json:"timestamp"`
}

// 하트비트 응답 데이터
type HeartbeatResponse struct {
	ServerTime uint64 `json:"server_time"`
	Latency    uint64 `json:"latency"`
}

// 플레이어 통계
type PlayerStats struct {
	PlayerID      uint64        `json:"player_id"`
	TotalPlayTime time.Duration `json:"total_play_time"`
	LastLogin     *time.Time    `json:"last_login"`
	LoginCount    uint64        `json:"login_count"`
	GamesPlayed   map[protocol.GameID]uint64 `json:"games_played"`
}

// 세션 정보
type SessionInfo struct {
	ConnectionID uint64    `json:"connection_id"`
	PlayerID     uint64    `json:"player_id"`
	Username     string    `json:"username"`
	LoginTime    time.Time `json:"login_time"`
	LastActivity time.Time `json:"last_activity"`
	CurrentGame  protocol.GameID `json:"current_game"`
	IPAddress    string    `json:"ip_address"`
}