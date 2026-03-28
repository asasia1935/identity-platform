package mw

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func UserFromGatewayHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetHeader(UserIDHeader)
		if uid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
			return
		}

		c.Set(ContextUserKey, uid)
		c.Next()
	}
}
