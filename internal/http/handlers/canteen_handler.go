package handlers

import (
	"errors"
	"net/http"
	"time"

	"freepass-2026/internal/http/response"
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type CanteenHandler struct {
	Queries *db.Queries
}

type canteenResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Location  *string `json:"location"`
	OwnerID   string  `json:"owner_id"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type menuItemResponse struct {
	ID          string  `json:"id"`
	CanteenID   string  `json:"canteen_id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Price       int32   `json:"price"`
	Stock       int32   `json:"stock"`
	IsAvailable bool    `json:"is_available"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type listResponse[T any] struct {
	Data []T `json:"data"`
}

func NewCanteenHandler(queries *db.Queries) *CanteenHandler {
	return &CanteenHandler{Queries: queries}
}

func (h *CanteenHandler) ListCanteens(c *gin.Context) {
	items, err := h.Queries.ListCanteens(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	resp := make([]canteenResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, canteenResponse{
			ID:        item.ID.String(),
			Name:      item.Name,
			Location:  textToPtr(item.Location),
			OwnerID:   item.OwnerID.String(),
			CreatedAt: timeToString(item.CreatedAt),
			UpdatedAt: timeToString(item.UpdatedAt),
		})
	}
	c.JSON(http.StatusOK, listResponse[canteenResponse]{Data: resp})
}

func (h *CanteenHandler) ListMenusByCanteen(c *gin.Context) {
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

	_, err := h.Queries.GetCanteenByID(c.Request.Context(), canteenUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	items, err := h.Queries.ListMenuItemsByCanteenID(c.Request.Context(), canteenUUID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	resp := make([]menuItemResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, menuItemResponse{
			ID:          item.ID.String(),
			CanteenID:   item.CanteenID.String(),
			Name:        item.Name,
			Description: textToPtr(item.Description),
			Price:       item.Price,
			Stock:       item.Stock,
			IsAvailable: item.IsAvailable,
			CreatedAt:   timeToString(item.CreatedAt),
			UpdatedAt:   timeToString(item.UpdatedAt),
		})
	}
	c.JSON(http.StatusOK, listResponse[menuItemResponse]{Data: resp})
}

func textToPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func timeToString(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}
