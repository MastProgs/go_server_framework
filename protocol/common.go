package protocol

import (
	"encoding/binary"
)

// 공통 패킷 헤더
type CommonPacketHeader struct {
	Length     uint32     // 패킷 길이 (헤더 포함)
	GameID     GameID     // 어느 게임의 패킷인지
	PacketType PacketType // 패킷 타입
	Sequence   uint32     // 시퀀스 번호
}

// 패킷 인터페이스
type Packet interface {
	GetHeader() *CommonPacketHeader
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
	GetSize() uint32
}

// 기본 패킷 구조체
type BasePacket struct {
	Header CommonPacketHeader
	Data   []byte
}

func (bp *BasePacket) GetHeader() *CommonPacketHeader {
	return &bp.Header
}

func (bp *BasePacket) GetSize() uint32 {
	return bp.Header.Length
}

// 헤더 크기 상수
const HeaderSize = 16 // uint32(4) + GameID(2) + PacketType(2) + uint32(4) + padding

// 패킷 헤더 직렬화
func SerializeHeader(header *CommonPacketHeader) []byte {
	buf := make([]byte, HeaderSize)
	binary.LittleEndian.PutUint32(buf[0:4], header.Length)
	binary.LittleEndian.PutUint16(buf[4:6], uint16(header.GameID))
	binary.LittleEndian.PutUint16(buf[6:8], uint16(header.PacketType))
	binary.LittleEndian.PutUint32(buf[8:12], header.Sequence)
	// buf[12:16]은 패딩 (추후 확장용)
	return buf
}

// 패킷 헤더 역직렬화
func DeserializeHeader(data []byte) (*CommonPacketHeader, error) {
	if len(data) < HeaderSize {
		return nil, ErrInvalidPacketSize
	}

	header := &CommonPacketHeader{
		Length:     binary.LittleEndian.Uint32(data[0:4]),
		GameID:     GameID(binary.LittleEndian.Uint16(data[4:6])),
		PacketType: PacketType(binary.LittleEndian.Uint16(data[6:8])),
		Sequence:   binary.LittleEndian.Uint32(data[8:12]),
	}

	return header, nil
}

// 완전한 패킷 직렬화
func SerializePacket(packet Packet) ([]byte, error) {
	header := packet.GetHeader()

	// 패킷 데이터 직렬화
	data, err := packet.Serialize()
	if err != nil {
		return nil, err
	}

	// 총 길이 계산 및 헤더 업데이트
	totalLength := uint32(HeaderSize + len(data))
	header.Length = totalLength

	// 헤더 직렬화
	headerBytes := SerializeHeader(header)

	// 헤더 + 데이터 결합
	result := make([]byte, totalLength)
	copy(result[:HeaderSize], headerBytes)
	copy(result[HeaderSize:], data)

	return result, nil
}

// 시퀀스 번호 생성기
type SequenceGenerator struct {
	counter uint32
}

func NewSequenceGenerator() *SequenceGenerator {
	return &SequenceGenerator{counter: 0}
}

func (sg *SequenceGenerator) Next() uint32 {
	sg.counter++
	return sg.counter
}

// 글로벌 시퀀스 생성기
var GlobalSequence = NewSequenceGenerator()

// 새 패킷 헤더 생성 헬퍼
func NewPacketHeader(gameID GameID, packetType PacketType) CommonPacketHeader {
	return CommonPacketHeader{
		Length:     0, // Serialize 시에 계산됨
		GameID:     gameID,
		PacketType: packetType,
		Sequence:   GlobalSequence.Next(),
	}
}