package auth

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// redis 키 prefix 상수화
const (
	rateLoginPrefix   = "rate:login:"
	rateRefreshPrefix = "rate:refresh:"
)

// Rate Limiter 인터페이스
type RateLimiter interface {
	AllowLogin(ctx context.Context, ip string) (bool, error)
	AllowRefresh(ctx context.Context, uid string) (bool, error)
}

// RateLimit 전용 정책 값들 (Config에서 받아옴)
type RateLimitPolicy struct {
	LoginLimit    int64
	LoginWindow   time.Duration
	RefreshLimit  int64
	RefreshWindow time.Duration
}

// 구현체
type RedisRateLimiter struct {
	rdb    *redis.Client
	policy RateLimitPolicy
}

// 생성자
func NewRedisRateLimiter(rdb *redis.Client, policy RateLimitPolicy) (*RedisRateLimiter, error) {
	if rdb == nil {
		return nil, errors.New("redis client is required")
	}

	return &RedisRateLimiter{
		rdb:    rdb,
		policy: policy,
	}, nil
}

func (r *RedisRateLimiter) AllowLogin(ctx context.Context, ip string) (bool, error) {
	key := r.LoginKey(ip)
	return r.allow(ctx, key, r.policy.LoginLimit, r.policy.LoginWindow)
}

func (r *RedisRateLimiter) AllowRefresh(ctx context.Context, uid string) (bool, error) {
	key := r.RefreshKey(uid)
	return r.allow(ctx, key, r.policy.RefreshLimit, r.policy.RefreshWindow)
}

func (r *RedisRateLimiter) allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	count, err := r.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		if err := r.rdb.Expire(ctx, key, window).Err(); err != nil {
			return false, err
		}
	}

	// TODO : 추후 리턴할 때 남은 TTL 값을 로그나 헤더로 줄 수 있음 (운영용) -> 클라 입장에서도 몇초 후 다시 시도해야 하는지 알 수 있음
	if count > limit {
		return false, nil
	}

	return true, nil
}

func (r *RedisRateLimiter) LoginKey(ip string) string {
	return rateLoginPrefix + ip
}

func (r *RedisRateLimiter) RefreshKey(uid string) string {
	return rateRefreshPrefix + uid
}
