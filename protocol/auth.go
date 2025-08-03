package protocol

import (
	"encoding/binary"
	"encoding/json"
	"time"
)

// 로그인 패킷
type LoginPacket struct {
	Header   CommonPacketHeader
	Username string
	Password string
}

func NewLoginPacket(username, password string) *LoginPacket {
	return &LoginPacket{
		Header:   NewPacketHeader(GAME_AUTH, PACKET_AUTH_LOGIN),
		Username: username,
		Password: password,
	}
}

func (lp *LoginPacket) GetHeader() *CommonPacketHeader {
	return &lp.Header
}

func (lp *LoginPacket) Serialize() ([]byte, error) {
	data := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: lp.Username,
		Password: lp.Password,
	}

	return json.Marshal(data)
}

func (lp *LoginPacket) Deserialize(data []byte) error {
	var decoded struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	lp.Username = decoded.Username
	lp.Password = decoded.Password
	return nil
}

func (lp *LoginPacket) GetSize() uint32 {
	data, _ := lp.Serialize()
	return uint32(HeaderSize + len(data))
}

// 로그인 응답 패킷
type LoginResponsePacket struct {
	Header     CommonPacketHeader
	Success    bool
	Message    string
	PlayerID   uint64
	GameAccess []GameID
}

func NewLoginResponsePacket(success bool, message string, playerID uint64, gameAccess []GameID) *LoginResponsePacket {
	return &LoginResponsePacket{
		Header:     NewPacketHeader(GAME_AUTH, PACKET_AUTH_LOGIN_RESPONSE),
		Success:    success,
		Message:    message,
		PlayerID:   playerID,
		GameAccess: gameAccess,
	}
}

func (lrp *LoginResponsePacket) GetHeader() *CommonPacketHeader {
	return &lrp.Header
}

func (lrp *LoginResponsePacket) Serialize() ([]byte, error) {
	data := struct {
		Success    bool     `json:"success"`
		Message    string   `json:"message"`
		PlayerID   uint64   `json:"player_id"`
		GameAccess []GameID `json:"game_access"`
	}{
		Success:    lrp.Success,
		Message:    lrp.Message,
		PlayerID:   lrp.PlayerID,
		GameAccess: lrp.GameAccess,
	}

	return json.Marshal(data)
}

func (lrp *LoginResponsePacket) Deserialize(data []byte) error {
	var decoded struct {
		Success    bool     `json:"success"`
		Message    string   `json:"message"`
		PlayerID   uint64   `json:"player_id"`
		GameAccess []GameID `json:"game_access"`
	}

	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	lrp.Success = decoded.Success
	lrp.Message = decoded.Message
	lrp.PlayerID = decoded.PlayerID
	lrp.GameAccess = decoded.GameAccess
	return nil
}

func (lrp *LoginResponsePacket) GetSize() uint32 {
	data, _ := lrp.Serialize()
	return uint32(HeaderSize + len(data))
}

// 하트비트 패킷
type HeartbeatPacket struct {
	Header    CommonPacketHeader
	Timestamp uint64
}

func NewHeartbeatPacket() *HeartbeatPacket {
	return &HeartbeatPacket{
		Header:    NewPacketHeader(GAME_AUTH, PACKET_AUTH_HEARTBEAT),
		Timestamp: uint64(time.Now().Unix()),
	}
}

func (hp *HeartbeatPacket) GetHeader() *CommonPacketHeader {
	return &hp.Header
}

func (hp *HeartbeatPacket) Serialize() ([]byte, error) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, hp.Timestamp)
	return buf, nil
}

func (hp *HeartbeatPacket) Deserialize(data []byte) error {
	if len(data) < 8 {
		return ErrInvalidPacketSize
	}
	hp.Timestamp = binary.LittleEndian.Uint64(data[:8])
	return nil
}

func (hp *HeartbeatPacket) GetSize() uint32 {
	return uint32(HeaderSize + 8)
}

// 로그아웃 패킷
type LogoutPacket struct {
	Header   CommonPacketHeader
	PlayerID uint64
	Reason   string
}

func NewLogoutPacket(playerID uint64, reason string) *LogoutPacket {
	return &LogoutPacket{
		Header:   NewPacketHeader(GAME_AUTH, PACKET_AUTH_LOGOUT),
		PlayerID: playerID,
		Reason:   reason,
	}
}

func (lop *LogoutPacket) GetHeader() *CommonPacketHeader {
	return &lop.Header
}

func (lop *LogoutPacket) Serialize() ([]byte, error) {
	data := struct {
		PlayerID uint64 `json:"player_id"`
		Reason   string `json:"reason"`
	}{
		PlayerID: lop.PlayerID,
		Reason:   lop.Reason,
	}

	return json.Marshal(data)
}

func (lop *LogoutPacket) Deserialize(data []byte) error {
	var decoded struct {
		PlayerID uint64 `json:"player_id"`
		Reason   string `json:"reason"`
	}

	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	lop.PlayerID = decoded.PlayerID
	lop.Reason = decoded.Reason
	return nil
}

func (lop *LogoutPacket) GetSize() uint32 {
	data, _ := lop.Serialize()
	return uint32(HeaderSize + len(data))
}