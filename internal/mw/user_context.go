package mw

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func UserFromGatewayHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetHeader("X-User-ID")
		if uid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
			return
		}

		c.Set("user", uid)
		c.Next()
	}
}
