package auth

import (
	"testing"
	"time"
)

func TestGenerateRefreshTokenUsesConfiguredTTL(t *testing.T) {
	refreshTTL := 2 * time.Hour
	tm, err := NewTokenManager("test-secret", 15*time.Minute, refreshTTL)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	token, _, err := tm.GenerateRefreshToken("test")
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	claims, err := tm.VerifyRefreshToken(token)
	if err != nil {
		t.Fatalf("VerifyRefreshToken() error = %v", err)
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		t.Fatalf("refresh token should include iat and exp claims")
	}

	got := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	if got != refreshTTL {
		t.Fatalf("refresh token ttl = %v, want %v", got, refreshTTL)
	}
}

func TestNewTokenManagerRejectsNonPositiveRefreshTTL(t *testing.T) {
	if _, err := NewTokenManager("test-secret", 15*time.Minute, 0); err == nil {
		t.Fatal("expected error for non-positive refresh ttl")
	}
}
