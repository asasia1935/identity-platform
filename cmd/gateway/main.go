package main

import (
	"log"
	"net/http"

	"github.com/asasia1935/identity-platform/internal/auth"
	"github.com/asasia1935/identity-platform/internal/config"
	"github.com/asasia1935/identity-platform/internal/gateway"
	"github.com/asasia1935/identity-platform/internal/mw"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func NewRouter(tm *auth.Manager, authProxy http.HandlerFunc) *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// 로그인/리프레시 같은 Auth 전용 서비스 API는 Auth로 그냥 프록시
	// Gateway: /api/auth/login  -> Auth: /auth/login
	r.Any("/api/auth/*any", func(c *gin.Context) {
		authProxy(c.Writer, c.Request)
	})

	// 보호 API는 Gateway에서 검증하고(미들웨어), 업스트림으로 전달
	// 예시로 /api/me 를 Auth의 /me 로 프록시
	r.GET("/api/me", mw.AuthRequired(tm), func(c *gin.Context) {
		// 미들웨어가 user를 세팅해둔 상태라고 가정
		user, _ := c.Get("user")

		// downstream(업스트림)으로 사용자 주입 (나중에 Meetup에도 똑같이 씀)
		c.Request.Header.Set("X-User-Id", user.(string))

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
	tm, err := auth.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL)
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
