package protocol

import "fmt"

type PacketType uint16
type GameID uint16

const (
	GAME_AUTH GameID = 0 // 인증
	GAME_TEST GameID = 1 // 테스트 게임
)

const (
	// 인증 패킷 (iota로 자동 증가)
	PACKET_AUTH_LOGIN          PacketType = iota + 1 // 1
	PACKET_AUTH_LOGOUT                               // 2
	PACKET_AUTH_HEARTBEAT                            // 3
	PACKET_AUTH_SESSION_CHECK                        // 4
	PACKET_AUTH_LOGIN_RESPONSE                       // 5

	// 마지막 인증 패킷 번호 기록
	_AUTH_PACKET_END
)

const (
	// Test 게임 패킷 (1000번대 시작)
	PACKET_TEST_BASE         PacketType = 1000
	PACKET_TEST_LOBBY_LIST   PacketType = PACKET_TEST_BASE + iota // 1000
	PACKET_TEST_LOBBY_CREATE                                      // 1001
	PACKET_TEST_LOBBY_JOIN                                        // 1002
	PACKET_TEST_LOBBY_LEAVE                                       // 1003
	PACKET_TEST_LOBBY_INFO                                        // 1004

	PACKET_TEST_GAME_START  // 1005
	PACKET_TEST_GAME_MOVE   // 1006
	PACKET_TEST_GAME_ACTION // 1007
	PACKET_TEST_GAME_STATE  // 1008
	PACKET_TEST_GAME_END    // 1009

	PACKET_TEST_CHAT_SEND    // 1010
	PACKET_TEST_CHAT_RECEIVE // 1011

	// 마지막 테스트 패킷 번호 기록
	_TEST_PACKET_END
)

// 패킷 타입으로 게임 ID 자동 판단
func GetGameIDFromPacketType(packetType PacketType) GameID {
	switch {
	case packetType >= 1 && packetType < _AUTH_PACKET_END:
		return GAME_AUTH
	case packetType >= PACKET_TEST_BASE && packetType < _TEST_PACKET_END:
		return GAME_TEST
	default:
		return GAME_AUTH // 기본값
	}
}

// 패킷 타입 검증
func IsValidPacketType(packetType PacketType) bool {
	gameID := GetGameIDFromPacketType(packetType)
	switch gameID {
	case GAME_AUTH:
		return packetType >= 1 && packetType < _AUTH_PACKET_END
	case GAME_TEST:
		return packetType >= PACKET_TEST_BASE && packetType < _TEST_PACKET_END
	default:
		return false
	}
}

// 컴파일 타임에 패킷 번호 충돌 체크
func init() {
	validatePacketRanges()
}

func validatePacketRanges() {
	// 인증 패킷 범위 체크
	if _AUTH_PACKET_END > 100 {
		panic(fmt.Sprintf("Auth packets exceed range: %d > 100", _AUTH_PACKET_END))
	}

	// 테스트 게임 패킷 범위 체크
	if _TEST_PACKET_END > PACKET_TEST_BASE+999 {
		panic(fmt.Sprintf("Test game packets exceed range: %d", _TEST_PACKET_END))
	}

	fmt.Printf("✅ Packet validation passed:\n")
	fmt.Printf("   Auth packets: 1~%d\n", _AUTH_PACKET_END-1)
	fmt.Printf("   Test game packets: %d~%d\n", PACKET_TEST_BASE, _TEST_PACKET_END-1)
}

// 개발 시 패킷 목록 출력 (디버깅용)
func PrintAllPackets() {
	fmt.Println("=== 인증 패킷 ===")
	fmt.Printf("PACKET_AUTH_LOGIN: %d\n", PACKET_AUTH_LOGIN)
	fmt.Printf("PACKET_AUTH_LOGOUT: %d\n", PACKET_AUTH_LOGOUT)
	fmt.Printf("PACKET_AUTH_HEARTBEAT: %d\n", PACKET_AUTH_HEARTBEAT)
	fmt.Printf("PACKET_AUTH_LOGIN_RESPONSE: %d\n", PACKET_AUTH_LOGIN_RESPONSE)

	fmt.Println("=== 테스트 게임 패킷 ===")
	fmt.Printf("PACKET_TEST_LOBBY_LIST: %d\n", PACKET_TEST_LOBBY_LIST)
	fmt.Printf("PACKET_TEST_LOBBY_CREATE: %d\n", PACKET_TEST_LOBBY_CREATE)
	fmt.Printf("PACKET_TEST_GAME_START: %d\n", PACKET_TEST_GAME_START)
	fmt.Printf("PACKET_TEST_GAME_MOVE: %d\n", PACKET_TEST_GAME_MOVE)
	fmt.Printf("PACKET_TEST_CHAT_SEND: %d\n", PACKET_TEST_CHAT_SEND)
}
