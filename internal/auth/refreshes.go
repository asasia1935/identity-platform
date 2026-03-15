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

// Refresh Token Idempotency를 위한 인터페이스 정의 -> 동시 중복 작업을 방지(하려고 노력)하는 분산 잠금 기능
type Locker interface {
	TryLock(ctx context.Context, jti string) (bool, error)
}

// Redis 기반 Redis에 사용자별 refresh JTI를 저장하는 구현체
type RedisRefreshStore struct {
	rdb                   *redis.Client
	refreshTokenTTL       time.Duration
	refreshIdempotencyTTL time.Duration
}

// RedisRefreshStore 생성자 함수 (의존성 주입)
func NewRedisRefreshStore(rdb *redis.Client, refreshTokenTTL time.Duration, refreshIdempotencyTTL time.Duration) (*RedisRefreshStore, error) {
	if rdb == nil {
		return nil, errors.New("redis client is required")
	}
	if refreshTokenTTL <= 0 {
		return nil, errors.New("REFRESH_TOKEN_TTL must be positive")
	}
	if refreshIdempotencyTTL <= 0 {
		return nil, errors.New("REFRESH_IDEMPOTENCY_TTL must be positive")
	}
	return &RedisRefreshStore{
		rdb:                   rdb,
		refreshTokenTTL:       refreshTokenTTL,
		refreshIdempotencyTTL: refreshIdempotencyTTL,
	}, nil
}

// refresh JTI 키 함수 (네이밍 고정)
func (s *RedisRefreshStore) refreshKey(uid string) string {
	return "refresh:" + uid
}

// 사용자별 현재 유효한 Refresh Token의 JTI를 Redis에 TTL과 함께 저장 (값은 JTI로 저장 -> Get 시 JTI 반환)
func (s *RedisRefreshStore) Save(ctx context.Context, uid, jti string) error {
	return s.rdb.Set(ctx, s.refreshKey(uid), jti, s.refreshTokenTTL).Err()
}

// 사용자별 현재 유효한 Refresh Token의 JTI 조회
func (s *RedisRefreshStore) Get(ctx context.Context, uid string) (string, error) {
	return s.rdb.Get(ctx, s.refreshKey(uid)).Result()
}

// Refresh 상태 삭제 API (로그아웃/rotation 시 현재 JTI 제거)
func (s *RedisRefreshStore) Delete(ctx context.Context, uid string) error {
	return s.rdb.Del(ctx, s.refreshKey(uid)).Err()
}

// refresh idempotency lock 키 함수
func (s *RedisRefreshStore) refreshIdemKey(jti string) string {
	return "idem:refresh:" + jti
}

// 동일 Refresh 요청의 중복 처리를 방지하기 위해 JTI 기준으로 짧은 TTL의 락 시도 (ex 5초 이내 요청은 Idempotency)
func (s *RedisRefreshStore) TryLock(ctx context.Context, jti string) (bool, error) {
	return s.rdb.SetNX(ctx, s.refreshIdemKey(jti), "1", s.refreshIdempotencyTTL).Result()
}
