package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func NewRedisClient(cfg RedisConfig) (*redis.Client, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("redis addr is empty")
	}

	// Redis 클라이언트 생성 (실제 연결은 수행 X)
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	// 부팅 시점에 Redis 연결 검증(Fail Fast) -> 실제 연결 시도 및 인증 검증
	// 연결 테스트가 안될 경우 2초 후 타임아웃으로 실패 처리 (서비스가 Redis에 의존적이므로 연결 실패 시 빠르게 실패하는 것이 좋음)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 연결 테스트 (Ping) -> 내부적으로 Lazy Connection이므로 실제 연결 시도는 여기서 이루어짐
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return rdb, nil
}
