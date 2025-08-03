package packet_processor

import "errors"

// 패킷 프로세서 관련 에러들
var (
	ErrQueueClosed           = errors.New("completion queue is closed")
	ErrQueueFull            = errors.New("completion queue is full")
	ErrInvalidPacketType    = errors.New("invalid packet type")
	ErrHandlerNotFound      = errors.New("packet handler not found")
	ErrServiceNotFound      = errors.New("game service not found")
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrAccessDenied         = errors.New("access denied")
	ErrServiceUnavailable   = errors.New("service unavailable")
	ErrInvalidContext       = errors.New("invalid packet context")
	ErrHandlerPanic         = errors.New("handler panic occurred")
	ErrProcessingTimeout    = errors.New("packet processing timeout")
)