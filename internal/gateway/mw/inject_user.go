package mw

import (
	"github.com/asasia1935/identity-platform/internal/auth"
	appmw "github.com/asasia1935/identity-platform/internal/mw"
	"github.com/gin-gonic/gin"
)

func AuthRequiredAndInjectUser(tm *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// JWT 검증으로 uid 추출
		uid, ok := appmw.ExtractUserIDFromBearer(c, tm)
		if !ok {
			return
		}

		// 업스트림으로 사용자 정보를 헤더에 주입
		c.Request.Header.Set(appmw.UserIDHeader, uid)
		c.Next()
	}
}
