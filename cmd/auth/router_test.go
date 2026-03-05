//go:build legacytests

// 세션 추가 전의 테스트는 임시로 빌드 X -> 추후 세션 구조 완성되었을때 테스트 수정하여 붙이는 것으로 결정

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/asasia1935/identity-platform/internal/auth"
)

// 테스트용 토큰 매니저 생성 헬퍼 함수 (고정 시크릿, 짧은 TTL)
func newTestTokenManager(t *testing.T) *auth.TokenManager {
	t.Helper()

	tm, err := auth.NewTokenManager("test-secret", 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}
	return tm
}

// TestAuthRouter_BlocksWhenGatewayHeaderMissing -> 보호된 엔드포인트(/me)가 반드시 Gateway를 통해서만 접근되도록 보장하는 테스트
// 헤더 없이 요청이 들어오면 미들웨어에 의해 403 Forbidden으로 차단됨
// (해당 테스트는 Auth 서비스가 게이트웨이 경유 계약을 유지하는지 확인하는 회귀 테스트(기능이 퇴보했는지 확인하는 테스트))
func TestAuthRouter_BlocksWhenGatewayHeaderMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tm := newTestTokenManager(t)

	r := NewRouter(tm)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected %d when gateway header missing, got %d", http.StatusForbidden, w.Code)
	}
}

// TestAuthRouter_AllowsGatewayHeaderButRejectsWithoutToken -> 보호된 엔드포인트(/me)가 게이트웨이 헤더는 허용하지만 인증 토큰이 없으면 401 Unauthorized로 거부하는 테스트
// 헤더는 있지만 토큰이 없는 요청이 들어오면 인증 미들웨어에 의해 401 Unauthorized로 차단됨
// (해당 테스트는 게이트웨이 경유는 했지만 인증 토큰이 없는 경우가 여전히 차단되는지 확인하는 회귀 테스트)
func TestAuthRouter_AllowsGatewayHeaderButRejectsWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tm := newTestTokenManager(t)
	r := NewRouter(tm)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Gateway-Verified", "true")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d when gateway header ok but token missing, got %d", http.StatusUnauthorized, w.Code)
	}
}
