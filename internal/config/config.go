package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// shared
	Env            string
	JWTSecret      string
	AccessTokenTTL time.Duration

	// auth
	HTTPPort string

	// gateway
	GatewayHTTPPort string
	AuthUpstream    string

	// redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

// 로딩 코드
func Load() (Config, error) {
	cfg := Config{
		Env:             getEnv("APP_ENV", "dev"),
		JWTSecret:       strings.TrimSpace(os.Getenv("JWT_SECRET")),
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		GatewayHTTPPort: getEnv("GATEWAY_PORT", "8090"),
		AuthUpstream:    getEnv("AUTH_UPSTREAM", "http://localhost:8080"),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
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

	dbStr := getEnv("REDIS_DB", "0")
	db, err := strconv.Atoi(dbStr)
	if err != nil {
		return Config{}, errors.New("invalid REDIS_DB (must be integer)")
	}
	cfg.RedisDB = db

	return cfg, nil
}

func getEnv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}
