package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	Env      string
	HTTPPort string

	JWTSecret      string
	AccessTokenTTL time.Duration
}

// 로딩 코드
func Load() (Config, error) {
	cfg := Config{
		Env:      getEnv("APP_ENV", "dev"),
		HTTPPort: getEnv("HTTP_PORT", "8080"),

		JWTSecret: strings.TrimSpace(os.Getenv("JWT_SECRET")),
	}

	ttlStr := getEnv("ACCESS_TOKEN_TTL", "15m")
	ttl, err := time.ParseDuration(ttlStr) // Duration으로 파싱
	if err != nil {
		return Config{}, errors.New("invalid ACCESS_TOKEN_TTL (e.g. 15m, 1h)")
	}
	cfg.AccessTokenTTL = ttl

	// 없을 경우 바로 에러로 종료
	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}
