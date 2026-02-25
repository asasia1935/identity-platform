package mw

import (
	"net/http"

	"github.com/asasia1935/identity-platform/internal/auth"
	appmw "github.com/asasia1935/identity-platform/internal/mw"
	"github.com/gin-gonic/gin"
)

func AuthRequiredAndInjectUser(tm *auth.Manager) gin.HandlerFunc {
	// 기존 AuthRequired 미들웨어로 인증 먼저 수행
	authMw := appmw.AuthRequired(tm)

	return func(c *gin.Context) {
		authMw(c)
		if c.IsAborted() {
			return
		}

		// 인증이 성공한 상태에서 user 정보 추출
		v, ok := c.Get("user")
		userID, ok2 := v.(string)
		if !ok || !ok2 || userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// 업스트림으로 사용자 정보를 헤더에 주입
		c.Request.Header.Set("X-User-Id", userID)
		c.Next()
	}
}
