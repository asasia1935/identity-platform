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
	Env             string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// auth
	HTTPPort string

	// gateway
	GatewayHTTPPort string
	AuthUpstream    string

	// redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	SessionTTL    time.Duration
}

// TODO:
// 현재 config.Load()에서 모든 서비스를 동일한 Config 구조체로 로드하고 있음.
// 하지만 서비스별로 필요한 설정이 다르므로 (예: auth / gateway),
// 추후 서비스별 Config 로더로 분리하는 것을 고려할 수 있음.
//
// 예:
//   LoadAuthConfig()
//   LoadGatewayConfig()
//
// 현재는 MVP 단계이므로 단순화를 위해 하나의 Config 구조체로 관리한다.

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

	accessTokenTTLStr := getEnv("ACCESS_TOKEN_TTL", "15m")
	accessTokenTTL, err := time.ParseDuration(accessTokenTTLStr) // Duration으로 파싱
	if err != nil {
		return Config{}, errors.New("invalid ACCESS_TOKEN_TTL (e.g. 15m, 1h)")
	}
	cfg.AccessTokenTTL = accessTokenTTL

	refreshTokenTTLStr := getEnv("REFRESH_TOKEN_TTL", "168h")
	refreshTokenTTL, err := time.ParseDuration(refreshTokenTTLStr) // Duration으로 파싱
	if err != nil {
		return Config{}, errors.New("invalid REFRESH_TOKEN_TTL (e.g. 168h)")
	}
	cfg.RefreshTokenTTL = refreshTokenTTL

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

	sessionTTLStr := getEnv("SESSION_TTL", "15m")
	sessionTTL, err := time.ParseDuration(sessionTTLStr) // Duration으로 파싱
	if err != nil {
		return Config{}, errors.New("invalid SESSION_TTL (e.g. 15m, 1h)")
	}
	cfg.SessionTTL = sessionTTL

	return cfg, nil
}

func getEnv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}
