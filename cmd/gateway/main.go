package main

import (
	"log"
	"net/http"

	"github.com/asasia1935/identity-platform/internal/auth"
	"github.com/asasia1935/identity-platform/internal/config"
	"github.com/asasia1935/identity-platform/internal/gateway"
	gwmw "github.com/asasia1935/identity-platform/internal/gateway/mw"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func NewRouter(tm *auth.Manager, authProxy http.HandlerFunc) *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// API 그룹 생성 (공개/보호 API 구분)
	api := r.Group("/api")

	// 공개 (로그인/리프레시 같은 Auth 전용 서비스 API는 Auth로 그냥 프록시)
	api.Any("/auth/*any", func(c *gin.Context) {
		authProxy(c.Writer, c.Request)
	})

	// 보호: 여기로 들어오는 순간 검증 + 사용자 주입 완료 (/me 같은 )
	protected := api.Group("/")
	protected.Use(gwmw.AuthRequiredAndInjectUser(tm))

	// 이제 핸들러는 "프록시만" 하면 됨 (테스트도 매우 쉬움)
	protected.GET("/me", func(c *gin.Context) {
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
