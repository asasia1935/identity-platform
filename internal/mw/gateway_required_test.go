package mw_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/asasia1935/identity-platform/internal/mw"
)

// GatewayRequired는 이 서비스가 반드시 게이트웨이를 통해서만 접근되도록 보장하는 미들웨어
// (게이트웨이의 리버스 프록시가 업스트림 요청에 헤더를 주입하고 해당 헤더가 없는 요청은 직접 접근으로 간주하여 거부하는 방식)
// -> 보안 강화 장치 X, 아키텍처적 의도 표현 및 실수 방지 장치 O (테스트 실패시 게이트웨이 설정 문제 또는 직접 접근 시도 의심 가능)

func TestGatewayRequired_BlocksWhenHeaderMissing(t *testing.T) {
	// Gin 테스트 모드로 설정 (로그 출력 최소화)
	gin.SetMode(gin.TestMode)

	// 미들웨어 테스트용 라우터 설정
	r := gin.New()
	r.GET("/protected", mw.GatewayRequired(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 게이트웨이 헤더 없이 요청 생성
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	// 요청 처리
	r.ServeHTTP(w, req)

	// 403 Forbidden 예상
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected %d when %s missing, got %d", http.StatusForbidden, mw.GatewayVerifiedHeader, w.Code)
	}
}

func TestGatewayRequired_AllowsWhenHeaderPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/protected", mw.GatewayRequired(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(mw.GatewayVerifiedHeader, "true")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 200 OK 예상
	if w.Code != http.StatusOK {
		t.Fatalf("expected %d when %s=true, got %d", http.StatusOK, mw.GatewayVerifiedHeader, w.Code)
	}
}
