package test

import "errors"

// 테스트 게임 서비스 관련 에러들
var (
	ErrRoomNotFound         = errors.New("room not found")
	ErrRoomFull            = errors.New("room is full")
	ErrRoomPasswordRequired = errors.New("room password required")
	ErrInvalidRoomPassword = errors.New("invalid room password")
	ErrPlayerNotInRoom     = errors.New("player not in room")
	ErrPlayerAlreadyInRoom = errors.New("player already in room")
	ErrNotRoomOwner        = errors.New("not room owner")
	ErrGameAlreadyStarted  = errors.New("game already started")
	ErrGameNotStarted      = errors.New("game not started")
	ErrPlayersNotReady     = errors.New("not all players are ready")
	ErrNotEnoughPlayers    = errors.New("not enough players")
	ErrInvalidAction       = errors.New("invalid game action")
	ErrInvalidPosition     = errors.New("invalid position")
	ErrChatMessageTooLong  = errors.New("chat message too long")
	ErrInvalidChatType     = errors.New("invalid chat type")
	ErrTargetPlayerNotFound = errors.New("target player not found")
)