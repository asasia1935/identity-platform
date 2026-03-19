package mw

import (
	"net/http"
	"strings"

	"github.com/asasia1935/identity-platform/internal/auth"
	"github.com/gin-gonic/gin"
)

func ExtractUserIDFromBearer(c *gin.Context, tm *auth.TokenManager) (string, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}

	claims, err := tm.VerifyAccessToken(parts[1])
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}

	return claims.Subject, true
}
