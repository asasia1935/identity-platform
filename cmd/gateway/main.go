package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/asasia1935/identity-platform/internal/auth"
	"github.com/asasia1935/identity-platform/internal/config"
	"github.com/asasia1935/identity-platform/internal/gateway"
	gwmw "github.com/asasia1935/identity-platform/internal/gateway/mw"
	"github.com/asasia1935/identity-platform/internal/mw"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func NewRouter(tm *auth.TokenManager, authProxy http.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(mw.RequestID())
	r.Use(gin.LoggerWithFormatter(func(p gin.LogFormatterParams) string {
		rid := p.Request.Header.Get(mw.RequestIDHeader) // 헤더에서 꺼내기
		// gin.Context Keys에서 꺼내려면 Custom middleware가 필요함(Formatter는 gin.Context를 직접 못 받음)

		// 상태코드가 400 이상일때 플래그 WARN으로 변경
		level := "INFO"
		if p.StatusCode >= 400 {
			level = "WARN"
		}

		// 경로 로그 추가
		path := p.Path
		if path == "" {
			path = p.Request.URL.Path
		}

		return fmt.Sprintf(
			"level=%s svc=%s rid=%s method=%s path=%s status=%d latency_ms=%d ip=%s\n",
			level, "auth", rid, p.Method, path, p.StatusCode, p.Latency.Milliseconds(), p.ClientIP,
		)
	}))

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// API 그룹 생성 (공개/보호 API 구분)
	api := r.Group("/api")
	auth := api.Group("/auth")

	// 공개 (로그인/리프레시 같은 Auth 전용 서비스 API는 Auth로 그냥 프록시)
	auth.POST("/login", func(c *gin.Context) {
		authProxy(c.Writer, c.Request)
	})
	auth.POST("/refresh", func(c *gin.Context) {
		authProxy(c.Writer, c.Request)
	})

	// 보호: 여기로 들어오는 순간 검증 + 사용자 주입 완료 (/me, 로그아웃 같은)
	protected := auth.Group("/")
	protected.Use(gwmw.AuthRequiredAndInjectUser(tm))

	// 이제 핸들러는 "프록시만" 하면 됨 (테스트도 매우 쉬움)
	protected.GET("/me", func(c *gin.Context) {
		authProxy(c.Writer, c.Request)
	})
	protected.POST("/logout", func(c *gin.Context) {
		authProxy(c.Writer, c.Request)
	})

	return r
}

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// 게이트웨이가 해당 정책으로 검증할 수 있도록 매니저 생성
	tm, err := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		log.Fatal(err)
	}

	// 업스트림 프록시 생성 (Auth로 보냄)
	authProxy, err := gateway.NewReverseProxyHandler(cfg.AuthUpstream, "/api")
	if err != nil {
		log.Fatal(err)
	}

	r := NewRouter(tm, authProxy)
	log.Fatal(r.Run(":" + cfg.GatewayHTTPPort))
}
