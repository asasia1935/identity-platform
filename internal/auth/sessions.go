package auth

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// 세션 스토어 인터페이스 정의 (추후 Redis 외 다른 구현체 추가 가능하도록 인터페이스로 추상화)
// 라우터/핸들러는 구체적인 Redis 구현체가 아닌 이 인터페이스에 의존하도록 설계
type SessionStore interface {
	Create(ctx context.Context, uid string) error
	Exists(ctx context.Context, uid string) (bool, error)
	Delete(ctx context.Context, uid string) error
}

// Redis 기반 세션 스토어 구현체
type RedisSessionStore struct {
	rdb        *redis.Client
	sessionTTL time.Duration
}

// RedisSessionStore 생성자 함수 (의존성 주입)
func NewRedisSessionStore(rdb *redis.Client, sessionTTL time.Duration) (*RedisSessionStore, error) {
	if rdb == nil {
		return nil, errors.New("redis client is required")
	}
	if sessionTTL <= 0 {
		return nil, errors.New("SESSION_TTL must be positive")
	}
	return &RedisSessionStore{
		rdb:        rdb,
		sessionTTL: sessionTTL,
	}, nil
}

// 세션키 함수 (네이밍 고정)
func (s *RedisSessionStore) sessionKey(uid string) string {
	return "sess:" + uid
}

// 세션 생성 API (값은 사용 X -> 존재하면 있는 것으로 확인)
func (s *RedisSessionStore) Create(ctx context.Context, uid string) error {
	return s.rdb.Set(ctx, s.sessionKey(uid), "1", s.sessionTTL).Err()
}

// 세션 체크 API
func (s *RedisSessionStore) Exists(ctx context.Context, uid string) (bool, error) {
	n, err := s.rdb.Exists(ctx, s.sessionKey(uid)).Result()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// 세션 삭제 API
func (s *RedisSessionStore) Delete(ctx context.Context, uid string) error {
	return s.rdb.Del(ctx, s.sessionKey(uid)).Err()
}
