package mw

import (
	"net/http"
	"strings"

	"github.com/asasia1935/identity-platform/internal/auth"
	"github.com/gin-gonic/gin"
)

func ExtractUserIDFromBearer(c *gin.Context, tm *auth.TokenManager) (string, bool) {
	authHeader := c.GetHeader(auth.AuthorizationHeader)
	if authHeader == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}

	if !strings.HasPrefix(authHeader, auth.BearerPrefix) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}

	rawToken := strings.TrimPrefix(authHeader, auth.BearerPrefix)

	claims, err := tm.VerifyAccessToken(rawToken)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}

	return claims.Subject, true
}
