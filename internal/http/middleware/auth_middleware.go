package middleware

import (
	"errors"
	"net/http"
	"strings"

	"freepass-2026/internal/auth"
	httpcontext "freepass-2026/internal/http/context"
	"freepass-2026/internal/http/response"
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		role, ok := httpcontext.UserRole(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
			return
		}
		if _, exists := allowed[role]; !exists {
			response.Error(c, http.StatusForbidden, "forbidden", nil)
			return
		}
		c.Next()
	}
}

func RequireCanteenOwner(queries *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := httpcontext.UserRole(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
			return
		}
		if role != "OWNER" && role != "ADMIN" {
			response.Error(c, http.StatusForbidden, "forbidden", nil)
			return
		}
		canteenID := c.Param("canteenId")
		if canteenID == "" {
			canteenID = c.Param("canteen_id")
		}
		if canteenID == "" {
			details := map[string]string{"canteen_id": "invalid_uuid"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
		var canteenUUID pgtype.UUID
		if err := canteenUUID.Scan(canteenID); err != nil {
			details := map[string]string{"canteen_id": "invalid_uuid"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
		if !canteenUUID.Valid {
			details := map[string]string{"canteen_id": "invalid_uuid"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
		canteen, err := queries.GetCanteenByID(c.Request.Context(), canteenUUID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				response.Error(c, http.StatusNotFound, "not_found", nil)
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		if role == "ADMIN" {
			c.Next()
			return
		}
		userID, ok := httpcontext.UserID(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
			return
		}
		var userUUID pgtype.UUID
		if err := userUUID.Scan(userID); err != nil {
			response.Error(c, http.StatusForbidden, "forbidden", nil)
			return
		}
		if !userUUID.Valid || userUUID != canteen.OwnerID {
			response.Error(c, http.StatusForbidden, "forbidden", nil)
			return
		}
		c.Next()
	}
}
