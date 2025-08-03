package network

import "errors"

// 네트워크 관련 에러들
var (
	ErrServerNotRunning     = errors.New("server is not running")
	ErrConnectionNotFound   = errors.New("connection not found")
	ErrConnectionClosed     = errors.New("connection is closed")
	ErrMaxConnectionsReached = errors.New("maximum connections reached")
	ErrInvalidAddress       = errors.New("invalid address")
	ErrListenerFailed       = errors.New("failed to start listener")
	ErrConnectionFailed     = errors.New("failed to establish connection")
	ErrSendBufferFull       = errors.New("send buffer is full")
	ErrReceiveTimeout       = errors.New("receive timeout")
	ErrInvalidPacket        = errors.New("invalid packet received")
)