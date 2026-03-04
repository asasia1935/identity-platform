package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/asasia1935/identity-platform/internal/auth"
	"github.com/asasia1935/identity-platform/internal/config"
	"github.com/asasia1935/identity-platform/internal/mw"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func NewRouter(tm *auth.Manager) *gin.Engine {
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
			level, "gateway", rid, p.Method, path, p.StatusCode, p.Latency.Milliseconds(), p.ClientIP,
		)
	}))

	// 504 타임아웃 테스트용 느린 서버
	r.GET("/auth/slow", func(c *gin.Context) {
		time.Sleep(10 * time.Second)
		c.String(200, "ok")
	})

	// Auth Health Check는 Gateway 체크 예외
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// 아래 API는 게이트웨이를 필수로 거쳐야함
	r.Use(mw.GatewayRequired())

	r.GET("/me", mw.AuthRequired(tm), func(c *gin.Context) {
		user, _ := c.Get("user")
		c.JSON(http.StatusOK, gin.H{
			"user": user,
		})
	})

	r.POST("/auth/login", func(c *gin.Context) {
		// 로그인 요청 바디(JSON) 구조 정의
		type LoginRequest struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		var req LoginRequest

		// JSON 바인딩
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid json",
			})
			return
		}

		// 임시로 인증 (하드코딩)
		if req.Username != "test" || req.Password != "1234" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid credentials",
			})
			return
		}

		// JWT의 token 반환
		token, err := tm.GenerateAccessToken(req.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "token issue failed",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"accessToken": token,
		})
	})

	return r
}

func main() {
	_ = godotenv.Load() // .env 없으면 무시

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	tm, err := auth.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		log.Fatal(err)
	}

	r := NewRouter(tm)
	log.Fatal(r.Run(":" + cfg.HTTPPort))
}
