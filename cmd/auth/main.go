package main

import (
	"log"
	"net/http"

	"github.com/asasia1935/identity-platform/internal/auth"
	"github.com/asasia1935/identity-platform/internal/config"
	"github.com/asasia1935/identity-platform/internal/mw"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func NewRouter(tm *auth.Manager) *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

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
