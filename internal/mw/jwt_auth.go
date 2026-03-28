package mw

import (
	"github.com/asasia1935/identity-platform/internal/auth"
	"github.com/gin-gonic/gin"
)

// JWT 검증 미들웨어
func JWTRequired(tm *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// JWT 검증으로 uid 추출
		uid, ok := ExtractUserIDFromBearer(c, tm)
		if !ok {
			return
		}

		// 이후 핸들러에 쓰도록 저장
		c.Set(ContextUserKey, uid)

		c.Next()
	}
}
