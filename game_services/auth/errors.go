package auth

import "errors"

// 인증 서비스 관련 에러들
var (
	ErrInvalidCredentials    = errors.New("invalid username or password")
	ErrUserNotFound         = errors.New("user not found")
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrInvalidUsername      = errors.New("invalid username format")
	ErrInvalidPassword      = errors.New("invalid password format")
	ErrInvalidEmail         = errors.New("invalid email format")
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionExpired       = errors.New("session expired")
	ErrAlreadyAuthenticated = errors.New("already authenticated")
	ErrNotAuthenticated     = errors.New("not authenticated")
	ErrAccessDenied         = errors.New("access denied")
	ErrMaxSessionsReached   = errors.New("maximum sessions reached")
	ErrDuplicateLogin       = errors.New("duplicate login detected")
	ErrAccountDisabled      = errors.New("account is disabled")
	ErrPasswordExpired      = errors.New("password expired")
	ErrAccountLocked        = errors.New("account is locked")
)