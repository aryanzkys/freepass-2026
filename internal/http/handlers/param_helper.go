package handlers

import (
	"net/http"

	"freepass-2026/internal/http/response"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

func parseUUIDParam(c *gin.Context, key string, fallback string) (pgtype.UUID, bool) {
	value := c.Param(key)
	if value == "" && fallback != "" {
		value = c.Param(fallback)
	}
	if value == "" {
		details := map[string]string{paramKey(key, fallback): "invalid_uuid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return pgtype.UUID{}, false
	}
	var parsed pgtype.UUID
	if err := parsed.Scan(value); err != nil || !parsed.Valid {
		details := map[string]string{paramKey(key, fallback): "invalid_uuid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return pgtype.UUID{}, false
	}
	return parsed, true
}

func paramKey(key string, fallback string) string {
	switch key {
	case "menuId", "menu_id":
		return "menu_id"
	case "orderId", "order_id":
		return "order_id"
	case "feedbackId", "feedback_id":
		return "feedback_id"
	default:
		return "canteen_id"
	}
}

func textFromPtr(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}
