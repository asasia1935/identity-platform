package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	// 네 프로젝트 경로에 맞게 수정

	"github.com/asasia1935/identity-platform/internal/auth"
	"github.com/asasia1935/identity-platform/internal/mw"
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

// TestGateway_ToAuth_Me_Returns200AndUserKey -> 게이트웨이를 통해 Auth의 /me 엔드포인트에 접근했을 때 200 OK와 user 키가 있는 JSON이 반환되는지 검증하는 테스트
// Gateway -> Auth 흐름이 정상적으로 작동하는지 확인하는 통합 테스트
func TestGateway_ToAuth_Me_Returns200AndUserKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tm := newTestTokenManager(t)

	// Auth 라우터(/me + GatewayRequired + JWTRequired 만으로 최소 설정)
	authRouter := gin.New()
	authRouter.Use(mw.GatewayRequired())
	authRouter.GET("/me", mw.JWTRequired(tm), func(c *gin.Context) {
		user, _ := c.Get("user")
		c.JSON(http.StatusOK, gin.H{"user": user})
	})

	// 게이트웨이가 호출할 때 authRouter로 요청이 전달되도록 하는 핸들러 함수
	authProxy := func(w http.ResponseWriter, r *http.Request) {
		r2 := r.Clone(context.Background())

		// URL 객체 복사(안전)
		u := *r.URL
		r2.URL = &u

		// gateway 요청: /api/me  -> auth는 /me 기대
		r2.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")
		if r2.URL.Path == "" {
			r2.URL.Path = "/"
		}

		// GatewayRequired 통과용
		r2.Header.Set("X-Gateway-Verified", "true")

		authRouter.ServeHTTP(w, r2)
	}

	// 게이트웨이 라우터 생성
	gw := NewRouter(tm, authProxy)

	// 토큰 생성 후 /api/me 엔드포인트로 요청
	token, err := tm.GenerateAccessToken("test")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	// 200 OK 예상
	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	// JSON에 user 키가 있는지만(그리고 값이 test인지) 최소 검증
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v, body=%s", err, w.Body.String())
	}

	v, ok := body["user"]
	if !ok {
		t.Fatalf("response json missing 'user' key: %v", body)
	}

	us, ok := v.(string)
	if !ok || us == "" {
		t.Fatalf("user should be non-empty string, got %#v", v)
	}

	if us != "test" {
		t.Fatalf("expected user 'test', got %q", us)
	}
}
