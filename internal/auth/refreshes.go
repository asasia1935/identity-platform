package auth

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Refresh Token의 JTI 관리를 위한 인터페이스 정의
type RefreshStore interface {
	Save(ctx context.Context, uid, jti string) error
	Get(ctx context.Context, uid string) (string, error)
	Delete(ctx context.Context, uid string) error
}

// Redis 기반 Redis에 사용자별 refresh JTI를 저장하는 구현체
type RedisRefreshStore struct {
	rdb *redis.Client
	ttl time.Duration
}

// RedisRefreshStore 생성자 함수 (의존성 주입)
func NewRedisRefreshStore(rdb *redis.Client, ttl time.Duration) (*RedisRefreshStore, error) {
	if rdb == nil {
		return nil, errors.New("redis client is required")
	}
	if ttl <= 0 {
		return nil, errors.New("REFRESH_TOKEN_TTL must be positive")
	}
	return &RedisRefreshStore{
		rdb: rdb,
		ttl: ttl,
	}, nil
}

// refresh JTI 키 함수 (네이밍 고정)
func (s *RedisRefreshStore) refreshKey(uid string) string {
	return "refresh:" + uid
}

// 사용자별 현재 유효한 Refresh Token의 JTI를 Redis에 TTL과 함께 저장 (값은 JTI로 저장 -> Get 시 JTI 반환)
func (s *RedisRefreshStore) Save(ctx context.Context, uid, jti string) error {
	return s.rdb.Set(ctx, s.refreshKey(uid), jti, s.ttl).Err()
}

// 사용자별 현재 유효한 Refresh Token의 JTI 조회
func (s *RedisRefreshStore) Get(ctx context.Context, uid string) (string, error) {
	return s.rdb.Get(ctx, s.refreshKey(uid)).Result()
}

// Refresh 상태 삭제 API (로그아웃/rotation 시 현재 JTI 제거)
func (s *RedisRefreshStore) Delete(ctx context.Context, uid string) error {
	return s.rdb.Del(ctx, s.refreshKey(uid)).Err()
}
