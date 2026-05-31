package config

import (
	"testing"
	"time"
)

func setValidConfigEnv(t *testing.T) {
	t.Helper()

	t.Setenv("APP_ENV", "test")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("ACCESS_TOKEN_TTL", "15m")
	t.Setenv("REFRESH_TOKEN_TTL", "168h")
	t.Setenv("REFRESH_IDEMPOTENCY_TTL", "5s")
	t.Setenv("HTTP_PORT", "8080")
	t.Setenv("GATEWAY_PORT", "8090")
	t.Setenv("AUTH_UPSTREAM", "http://localhost:8080")
	t.Setenv("REDIS_ADDR", "localhost:6379")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "0")
	t.Setenv("SESSION_TTL", "168h")
}

func TestLoadUsesRefreshRateLimitEnv(t *testing.T) {
	setValidConfigEnv(t)
	t.Setenv("LOGIN_RATE_LIMIT", "5")
	t.Setenv("LOGIN_RATE_WINDOW", "1m")
	t.Setenv("REFRESH_RATE_LIMIT", "17")
	t.Setenv("REFRESH_RATE_WINDOW", "30s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.LoginRateLimit != 5 {
		t.Fatalf("LoginRateLimit = %d, want 5", cfg.LoginRateLimit)
	}
	if cfg.LoginRateWindow != time.Minute {
		t.Fatalf("LoginRateWindow = %v, want %v", cfg.LoginRateWindow, time.Minute)
	}
	if cfg.RefreshRateLimit != 17 {
		t.Fatalf("RefreshRateLimit = %d, want 17", cfg.RefreshRateLimit)
	}
	if cfg.RefreshRateWindow != 30*time.Second {
		t.Fatalf("RefreshRateWindow = %v, want %v", cfg.RefreshRateWindow, 30*time.Second)
	}
}
