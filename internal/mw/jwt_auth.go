package mw

import (
	"net/http"
	"strings"

	"github.com/asasia1935/identity-platform/internal/auth"
	"github.com/gin-gonic/gin"
)

// JWT 검증 미들웨어
func JWTRequired(tm *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Authorization: Bearer <token>
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			c.Abort()
			return
		}

		rawToken := parts[1]

		claims, err := tm.VerifyAccessToken(rawToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			c.Abort()
			return
		}

		// 이후 핸들러에 쓰도록 저장
		c.Set("user", claims.Subject)

		c.Next()
	}
}
