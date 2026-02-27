package mw

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GatewayRequired : 게이트웨이를 거치지 않은 요청을 막는 미들웨어 로직(헤더가 있어야만 요청 성공)
func GatewayRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Gateway-Verified") != "true" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "gateway required",
			})
			return
		}
		c.Next()
	}
}
