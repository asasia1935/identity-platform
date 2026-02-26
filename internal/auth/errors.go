package auth

import "errors"

// 토큰 관련 에러 정의
var (
	ErrTokenMissing = errors.New("token missing")
	ErrTokenInvalid = errors.New("token invalid")
	ErrTokenExpired = errors.New("token expired")
)
