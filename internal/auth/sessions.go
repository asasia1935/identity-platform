package auth

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionStore struct {
	rdb        *redis.Client
	sessionTTL time.Duration
}

func NewSessionStore(rdb *redis.Client, sessionTTL time.Duration) (*SessionStore, error) {
	if rdb == nil {
		return nil, errors.New("redis client is required")
	}
	if sessionTTL <= 0 {
		return nil, errors.New("SESSION_TTL must be positive")
	}
	return &SessionStore{
		rdb:        rdb,
		sessionTTL: sessionTTL,
	}, nil
}

// 세션키 함수 (네이밍 고정)
func (s *SessionStore) sessionKey(uid string) string {
	return "sess:" + uid
}

// 세션 생성 API (값은 사용 X -> 존재하면 있는 것으로 확인)
func (s *SessionStore) Create(ctx context.Context, uid string) error {
	return s.rdb.Set(ctx, s.sessionKey(uid), "1", s.sessionTTL).Err()
}

// 세션 체크 API
func (s *SessionStore) Exists(ctx context.Context, uid string) (bool, error) {
	n, err := s.rdb.Exists(ctx, s.sessionKey(uid)).Result()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// 세션 삭제 API
func (s *SessionStore) Delete(ctx context.Context, uid string) error {
	return s.rdb.Del(ctx, s.sessionKey(uid)).Err()
}
