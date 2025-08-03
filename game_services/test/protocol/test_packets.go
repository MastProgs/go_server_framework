package protocol

import (
	"encoding/json"
	"go_server_framework/protocol"
)

// 테스트 게임 로비 리스트 요청
type TestLobbyListPacket struct {
	Header protocol.CommonPacketHeader
}

func NewTestLobbyListPacket() *TestLobbyListPacket {
	return &TestLobbyListPacket{
		Header: protocol.NewPacketHeader(protocol.GAME_TEST, protocol.PACKET_TEST_LOBBY_LIST),
	}
}

func (tlp *TestLobbyListPacket) GetHeader() *protocol.CommonPacketHeader {
	return &tlp.Header
}

func (tlp *TestLobbyListPacket) Serialize() ([]byte, error) {
	// 빈 요청 패킷
	return []byte{}, nil
}

func (tlp *TestLobbyListPacket) Deserialize(data []byte) error {
	// 빈 요청 패킷
	return nil
}

func (tlp *TestLobbyListPacket) GetSize() uint32 {
	return uint32(protocol.HeaderSize)
}

// 테스트 게임 로비 생성 요청
type TestLobbyCreatePacket struct {
	Header    protocol.CommonPacketHeader
	RoomName  string `json:"room_name"`
	MaxUsers  int    `json:"max_users"`
	IsPrivate bool   `json:"is_private"`
	Password  string `json:"password,omitempty"`
}

func NewTestLobbyCreatePacket(roomName string, maxUsers int, isPrivate bool, password string) *TestLobbyCreatePacket {
	return &TestLobbyCreatePacket{
		Header:    protocol.NewPacketHeader(protocol.GAME_TEST, protocol.PACKET_TEST_LOBBY_CREATE),
		RoomName:  roomName,
		MaxUsers:  maxUsers,
		IsPrivate: isPrivate,
		Password:  password,
	}
}

func (tcp *TestLobbyCreatePacket) GetHeader() *protocol.CommonPacketHeader {
	return &tcp.Header
}

func (tcp *TestLobbyCreatePacket) Serialize() ([]byte, error) {
	data := struct {
		RoomName  string `json:"room_name"`
		MaxUsers  int    `json:"max_users"`
		IsPrivate bool   `json:"is_private"`
		Password  string `json:"password,omitempty"`
	}{
		RoomName:  tcp.RoomName,
		MaxUsers:  tcp.MaxUsers,
		IsPrivate: tcp.IsPrivate,
		Password:  tcp.Password,
	}
	
	return json.Marshal(data)
}

func (tcp *TestLobbyCreatePacket) Deserialize(data []byte) error {
	var decoded struct {
		RoomName  string `json:"room_name"`
		MaxUsers  int    `json:"max_users"`
		IsPrivate bool   `json:"is_private"`
		Password  string `json:"password,omitempty"`
	}
	
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	
	tcp.RoomName = decoded.RoomName
	tcp.MaxUsers = decoded.MaxUsers
	tcp.IsPrivate = decoded.IsPrivate
	tcp.Password = decoded.Password
	
	return nil
}

func (tcp *TestLobbyCreatePacket) GetSize() uint32 {
	data, _ := tcp.Serialize()
	return uint32(protocol.HeaderSize + len(data))
}

// 테스트 게임 로비 입장 요청
type TestLobbyJoinPacket struct {
	Header   protocol.CommonPacketHeader
	RoomID   uint64 `json:"room_id"`
	Password string `json:"password,omitempty"`
}

func NewTestLobbyJoinPacket(roomID uint64, password string) *TestLobbyJoinPacket {
	return &TestLobbyJoinPacket{
		Header:   protocol.NewPacketHeader(protocol.GAME_TEST, protocol.PACKET_TEST_LOBBY_JOIN),
		RoomID:   roomID,
		Password: password,
	}
}

func (tjp *TestLobbyJoinPacket) GetHeader() *protocol.CommonPacketHeader {
	return &tjp.Header
}

func (tjp *TestLobbyJoinPacket) Serialize() ([]byte, error) {
	data := struct {
		RoomID   uint64 `json:"room_id"`
		Password string `json:"password,omitempty"`
	}{
		RoomID:   tjp.RoomID,
		Password: tjp.Password,
	}
	
	return json.Marshal(data)
}

func (tjp *TestLobbyJoinPacket) Deserialize(data []byte) error {
	var decoded struct {
		RoomID   uint64 `json:"room_id"`
		Password string `json:"password,omitempty"`
	}
	
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	
	tjp.RoomID = decoded.RoomID
	tjp.Password = decoded.Password
	
	return nil
}

func (tjp *TestLobbyJoinPacket) GetSize() uint32 {
	data, _ := tjp.Serialize()
	return uint32(protocol.HeaderSize + len(data))
}

// 테스트 게임 액션 패킷
type TestGameActionPacket struct {
	Header   protocol.CommonPacketHeader
	PlayerID uint64  `json:"player_id"`
	Action   string  `json:"action"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Data     string  `json:"data,omitempty"`
}

func NewTestGameActionPacket(playerID uint64, action string, x, y float32, data string) *TestGameActionPacket {
	return &TestGameActionPacket{
		Header:   protocol.NewPacketHeader(protocol.GAME_TEST, protocol.PACKET_TEST_GAME_ACTION),
		PlayerID: playerID,
		Action:   action,
		X:        x,
		Y:        y,
		Data:     data,
	}
}

func (tga *TestGameActionPacket) GetHeader() *protocol.CommonPacketHeader {
	return &tga.Header
}

func (tga *TestGameActionPacket) Serialize() ([]byte, error) {
	data := struct {
		PlayerID uint64  `json:"player_id"`
		Action   string  `json:"action"`
		X        float32 `json:"x"`
		Y        float32 `json:"y"`
		Data     string  `json:"data,omitempty"`
	}{
		PlayerID: tga.PlayerID,
		Action:   tga.Action,
		X:        tga.X,
		Y:        tga.Y,
		Data:     tga.Data,
	}
	
	return json.Marshal(data)
}

func (tga *TestGameActionPacket) Deserialize(data []byte) error {
	var decoded struct {
		PlayerID uint64  `json:"player_id"`
		Action   string  `json:"action"`
		X        float32 `json:"x"`
		Y        float32 `json:"y"`
		Data     string  `json:"data,omitempty"`
	}
	
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	
	tga.PlayerID = decoded.PlayerID
	tga.Action = decoded.Action
	tga.X = decoded.X
	tga.Y = decoded.Y
	tga.Data = decoded.Data
	
	return nil
}

func (tga *TestGameActionPacket) GetSize() uint32 {
	data, _ := tga.Serialize()
	return uint32(protocol.HeaderSize + len(data))
}

// 테스트 게임 채팅 패킷
type TestChatPacket struct {
	Header   protocol.CommonPacketHeader
	PlayerID uint64 `json:"player_id"`
	Message  string `json:"message"`
	Type     string `json:"type"` // "public", "private", "system"
	TargetID uint64 `json:"target_id,omitempty"`
}

func NewTestChatPacket(playerID uint64, message, chatType string, targetID uint64) *TestChatPacket {
	return &TestChatPacket{
		Header:   protocol.NewPacketHeader(protocol.GAME_TEST, protocol.PACKET_TEST_CHAT_SEND),
		PlayerID: playerID,
		Message:  message,
		Type:     chatType,
		TargetID: targetID,
	}
}

func (tch *TestChatPacket) GetHeader() *protocol.CommonPacketHeader {
	return &tch.Header
}

func (tch *TestChatPacket) Serialize() ([]byte, error) {
	data := struct {
		PlayerID uint64 `json:"player_id"`
		Message  string `json:"message"`
		Type     string `json:"type"`
		TargetID uint64 `json:"target_id,omitempty"`
	}{
		PlayerID: tch.PlayerID,
		Message:  tch.Message,
		Type:     tch.Type,
		TargetID: tch.TargetID,
	}
	
	return json.Marshal(data)
}

func (tch *TestChatPacket) Deserialize(data []byte) error {
	var decoded struct {
		PlayerID uint64 `json:"player_id"`
		Message  string `json:"message"`
		Type     string `json:"type"`
		TargetID uint64 `json:"target_id,omitempty"`
	}
	
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	
	tch.PlayerID = decoded.PlayerID
	tch.Message = decoded.Message
	tch.Type = decoded.Type
	tch.TargetID = decoded.TargetID
	
	return nil
}

func (tch *TestChatPacket) GetSize() uint32 {
	data, _ := tch.Serialize()
	return uint32(protocol.HeaderSize + len(data))
}