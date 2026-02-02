package middleware

import (
	"net/http"
	"strings"

	"freepass-2026/internal/auth"
	"freepass-2026/internal/http/response"

	"github.com/gin-gonic/gin"
)

func RequireAuth(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
			return
		}
		claims, err := authService.VerifyToken(parts[1])
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
			return
		}
		c.Set("user_id", claims.Subject)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}
