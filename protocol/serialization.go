package protocol

import (
	"fmt"
	"sync"
)

// 패킷 팩토리 - 패킷 타입별 생성자 관리
type PacketFactory struct {
	creators map[PacketType]func() Packet
	mu       sync.RWMutex
}

// 글로벌 패킷 팩토리
var GlobalPacketFactory = NewPacketFactory()

func NewPacketFactory() *PacketFactory {
	pf := &PacketFactory{
		creators: make(map[PacketType]func() Packet),
	}

	// 기본 패킷들 등록
	pf.registerDefaultPackets()

	return pf
}

// 기본 패킷들 등록
func (pf *PacketFactory) registerDefaultPackets() {
	// 인증 패킷들
	pf.RegisterPacketType(PACKET_AUTH_LOGIN, func() Packet { return &LoginPacket{} })
	pf.RegisterPacketType(PACKET_AUTH_LOGIN_RESPONSE, func() Packet { return &LoginResponsePacket{} })
	pf.RegisterPacketType(PACKET_AUTH_HEARTBEAT, func() Packet { return &HeartbeatPacket{} })
	pf.RegisterPacketType(PACKET_AUTH_LOGOUT, func() Packet { return &LogoutPacket{} })
	
	// 테스트 게임 패킷들은 별도로 등록 (의존성 순환 방지)
	// 실제로는 각 게임 서비스에서 등록해야 함
}

// 패킷 타입 등록
func (pf *PacketFactory) RegisterPacketType(packetType PacketType, creator func() Packet) {
	pf.mu.Lock()
	defer pf.mu.Unlock()
	pf.creators[packetType] = creator
}

// 패킷 생성
func (pf *PacketFactory) CreatePacket(packetType PacketType) (Packet, error) {
	pf.mu.RLock()
	creator, exists := pf.creators[packetType]
	pf.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown packet type: %d", packetType)
	}

	return creator(), nil
}

// 등록된 패킷 타입들 조회
func (pf *PacketFactory) GetRegisteredTypes() []PacketType {
	pf.mu.RLock()
	defer pf.mu.RUnlock()

	types := make([]PacketType, 0, len(pf.creators))
	for packetType := range pf.creators {
		types = append(types, packetType)
	}
	return types
}

// 패킷 역직렬화 (바이트 배열에서 패킷으로)
func DeserializePacket(data []byte) (Packet, error) {
	// 헤더 먼저 역직렬화
	if len(data) < HeaderSize {
		return nil, ErrInvalidPacketSize
	}

	header, err := DeserializeHeader(data[:HeaderSize])
	if err != nil {
		return nil, err
	}

	// 패킷 크기 검증
	if err := ValidatePacketSize(header.Length); err != nil {
		return nil, err
	}

	if len(data) < int(header.Length) {
		return nil, ErrInvalidPacketSize
	}

	// 패킷 타입 검증
	if !IsValidPacketType(header.PacketType) {
		return nil, ErrInvalidPacketType
	}

	// 패킷 생성
	packet, err := GlobalPacketFactory.CreatePacket(header.PacketType)
	if err != nil {
		return nil, err
	}

	// 헤더 설정
	*packet.GetHeader() = *header

	// 데이터 역직렬화
	if header.Length > HeaderSize {
		packetData := data[HeaderSize:header.Length]
		if err := packet.Deserialize(packetData); err != nil {
			return nil, err
		}
	}

	return packet, nil
}

// 편의 함수들
func RegisterPacketType(packetType PacketType, creator func() Packet) {
	GlobalPacketFactory.RegisterPacketType(packetType, creator)
}

func CreatePacket(packetType PacketType) (Packet, error) {
	return GlobalPacketFactory.CreatePacket(packetType)
}

// 패킷 크기 계산 (실제 직렬화 없이)
func CalculatePacketSize(packet Packet) uint32 {
	return packet.GetSize()
}

// 디버깅용 패킷 정보 출력
func PrintPacketInfo(packet Packet) {
	header := packet.GetHeader()
	fmt.Printf("=== Packet Info ===\n")
	fmt.Printf("Length: %d\n", header.Length)
	fmt.Printf("GameID: %d\n", header.GameID)
	fmt.Printf("PacketType: %d\n", header.PacketType)
	fmt.Printf("Sequence: %d\n", header.Sequence)
	fmt.Printf("Size: %d\n", packet.GetSize())
}