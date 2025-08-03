package protocol

import "errors"

// 프로토콜 관련 에러들
var (
	ErrInvalidPacketSize   = errors.New("invalid packet size")
	ErrInvalidPacketType   = errors.New("invalid packet type")
	ErrInvalidGameID       = errors.New("invalid game ID")
	ErrPacketTooLarge      = errors.New("packet too large")
	ErrSerializationFailed = errors.New("serialization failed")
	ErrDeserializationFailed = errors.New("deserialization failed")
	ErrInvalidPacketFormat = errors.New("invalid packet format")
	ErrUnknownPacketType   = errors.New("unknown packet type")
)

// 최대 패킷 크기 (16MB)
const MaxPacketSize = 16 * 1024 * 1024

// 패킷 크기 검증
func ValidatePacketSize(size uint32) error {
	if size < HeaderSize {
		return ErrInvalidPacketSize
	}
	if size > MaxPacketSize {
		return ErrPacketTooLarge
	}
	return nil
}