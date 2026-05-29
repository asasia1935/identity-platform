package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/asasia1935/identity-platform/internal/auth"
	"github.com/asasia1935/identity-platform/internal/mw"
)

// fakeSessionStoreFailOnCall -> 세션 스토어가 호출되면 테스트 실패하도록 하는 테스트용 세션 스토어 Fake 구현체 (미들웨어 검증용이기 때문에 세션 스토어가 호출되어서 실패하면 안됨)
type fakeSessionStoreFailOnCall struct {
	t *testing.T
}

func (s *fakeSessionStoreFailOnCall) Create(ctx context.Context, uid string) error {
	s.t.Fatalf("session store should not be called in this test")
	return nil
}

func (s *fakeSessionStoreFailOnCall) Exists(ctx context.Context, uid string) (bool, error) {
	s.t.Fatalf("session store should not be called in this test")
	return false, nil
}

func (s *fakeSessionStoreFailOnCall) Delete(ctx context.Context, uid string) error {
	s.t.Fatalf("session store should not be called in this test")
	return nil
}

// 테스트용 토큰 매니저 생성 헬퍼 함수 (고정 시크릿, 짧은 TTL)
func newTestTokenManager(t *testing.T) *auth.TokenManager {
	t.Helper()

	tm, err := auth.NewTokenManager("test-secret", 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}
	return tm
}

// fakeRefreshStoreFailOnCall -> 리프레시 스토어가 호출되면 테스트 실패하도록 하는 테스트용 리프레시 스토어 Fake 구현체 (미들웨어 검증용이기 때문에 리프레시 스토어가 호출되어서 실패하면 안됨)
type fakeRefreshStoreFailOnCall struct {
	t *testing.T
}

func (s *fakeRefreshStoreFailOnCall) Save(ctx context.Context, uid, jti string) error {
	s.t.Fatalf("refresh store should not be called in this test")
	return nil
}

func (s *fakeRefreshStoreFailOnCall) Get(ctx context.Context, uid string) (string, error) {
	s.t.Fatalf("refresh store should not be called in this test")
	return "", nil
}

func (s *fakeRefreshStoreFailOnCall) Delete(ctx context.Context, uid string) error {
	s.t.Fatalf("refresh store should not be called in this test")
	return nil
}

type fakeLockerFailOnCall struct {
	t *testing.T
}

func (s *fakeLockerFailOnCall) TryLock(ctx context.Context, jti string) (bool, error) {
	s.t.Fatalf("locker should not be called in this test")
	return false, nil
}

type fakeRateLimitFailOnCall struct {
	t *testing.T
}

func (s *fakeRateLimitFailOnCall) AllowLogin(ctx context.Context, ip string) (bool, error) {
	s.t.Fatalf("rate limiter should not be called in this test")
	return false, nil
}

func (s *fakeRateLimitFailOnCall) AllowRefresh(ctx context.Context, uid string) (bool, error) {
	s.t.Fatalf("rate limiter should not be called in this test")
	return false, nil
}

type fakeSessionStore struct {
	exists bool
}

func (s *fakeSessionStore) Create(ctx context.Context, uid string) error {
	s.exists = true
	return nil
}

func (s *fakeSessionStore) Exists(ctx context.Context, uid string) (bool, error) {
	return s.exists, nil
}

func (s *fakeSessionStore) Delete(ctx context.Context, uid string) error {
	s.exists = false
	return nil
}

type fakeRefreshStore struct {
	jti string
}

func (s *fakeRefreshStore) Save(ctx context.Context, uid, jti string) error {
	s.jti = jti
	return nil
}

func (s *fakeRefreshStore) Get(ctx context.Context, uid string) (string, error) {
	return s.jti, nil
}

func (s *fakeRefreshStore) Delete(ctx context.Context, uid string) error {
	s.jti = ""
	return nil
}

type fakeLocker struct{}

func (s *fakeLocker) TryLock(ctx context.Context, jti string) (bool, error) {
	return true, nil
}

type fakeRateLimiter struct{}

func (s *fakeRateLimiter) AllowLogin(ctx context.Context, ip string) (bool, error) {
	return true, nil
}

func (s *fakeRateLimiter) AllowRefresh(ctx context.Context, uid string) (bool, error) {
	return true, nil
}

// TestAuthRouter_BlocksWhenGatewayHeaderMissing -> 보호된 엔드포인트(/me)가 반드시 Gateway를 통해서만 접근되도록 보장하는 테스트
// 헤더 없이 요청이 들어오면 미들웨어에 의해 403 Forbidden으로 차단됨
// (해당 테스트는 Auth 서비스가 게이트웨이 경유 계약을 유지하는지 확인하는 회귀 테스트(기능이 퇴보했는지 확인하는 테스트))
func TestAuthRouter_BlocksWhenGatewayHeaderMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tm := newTestTokenManager(t)
	ss := &fakeSessionStoreFailOnCall{t: t}
	rs := &fakeRefreshStoreFailOnCall{t: t}
	lo := &fakeLockerFailOnCall{t: t}
	rl := &fakeRateLimitFailOnCall{t: t}

	r := NewRouter(tm, ss, rs, lo, rl)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
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
	ss := &fakeSessionStoreFailOnCall{t: t}
	rs := &fakeRefreshStoreFailOnCall{t: t}
	lo := &fakeLockerFailOnCall{t: t}
	rl := &fakeRateLimitFailOnCall{t: t}

	r := NewRouter(tm, ss, rs, lo, rl)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set(mw.GatewayVerifiedHeader, "true")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d when gateway header ok but token missing, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthRouter_LoginTokenResponseUsesSnakeCaseFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := NewRouter(
		newTestTokenManager(t),
		&fakeSessionStore{},
		&fakeRefreshStore{},
		&fakeLocker{},
		&fakeRateLimiter{},
	)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"username":"test","password":"1234"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(mw.GatewayVerifiedHeader, "true")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	assertTokenResponseUsesSnakeCaseFields(t, w.Body.Bytes())
}

func TestAuthRouter_RefreshTokenResponseUsesSnakeCaseFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := NewRouter(
		newTestTokenManager(t),
		&fakeSessionStore{},
		&fakeRefreshStore{},
		&fakeLocker{},
		&fakeRateLimiter{},
	)

	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"username":"test","password":"1234"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginReq.Header.Set(mw.GatewayVerifiedHeader, "true")
	loginW := httptest.NewRecorder()

	r.ServeHTTP(loginW, loginReq)

	if loginW.Code != http.StatusOK {
		t.Fatalf("login expected %d, got %d, body=%s", http.StatusOK, loginW.Code, loginW.Body.String())
	}

	var loginBody map[string]any
	if err := json.Unmarshal(loginW.Body.Bytes(), &loginBody); err != nil {
		t.Fatalf("login response should be valid json: %v", err)
	}
	refreshToken, ok := loginBody["refresh_token"].(string)
	if !ok || refreshToken == "" {
		t.Fatalf("login response missing non-empty refresh_token: %v", loginBody)
	}

	refreshBody, err := json.Marshal(map[string]string{"refresh_token": refreshToken})
	if err != nil {
		t.Fatalf("failed to marshal refresh request: %v", err)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(refreshBody))
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshReq.Header.Set(mw.GatewayVerifiedHeader, "true")
	refreshW := httptest.NewRecorder()

	r.ServeHTTP(refreshW, refreshReq)

	if refreshW.Code != http.StatusOK {
		t.Fatalf("refresh expected %d, got %d, body=%s", http.StatusOK, refreshW.Code, refreshW.Body.String())
	}

	assertTokenResponseUsesSnakeCaseFields(t, refreshW.Body.Bytes())
}

func assertTokenResponseUsesSnakeCaseFields(t *testing.T, raw []byte) {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("response should be valid json: %v, body=%s", err, string(raw))
	}
	if _, ok := body["access_token"]; !ok {
		t.Fatalf("response missing access_token: %v", body)
	}
	if _, ok := body["refresh_token"]; !ok {
		t.Fatalf("response missing refresh_token: %v", body)
	}
	if _, ok := body["accessToken"]; ok {
		t.Fatalf("response should not include accessToken: %v", body)
	}
	if _, ok := body["refreshToken"]; ok {
		t.Fatalf("response should not include refreshToken: %v", body)
	}
}
