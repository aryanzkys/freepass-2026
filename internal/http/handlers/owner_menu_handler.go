package handlers

import (
	"errors"
	"net/http"

	"freepass-2026/internal/http/response"
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type OwnerMenuHandler struct {
	Queries *db.Queries
}

type ownerMenuCreateRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Price       int32   `json:"price"`
	Stock       int32   `json:"stock"`
	IsAvailable *bool   `json:"is_available"`
}

type ownerMenuUpdateRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Price       int32   `json:"price"`
	Stock       int32   `json:"stock"`
	IsAvailable *bool   `json:"is_available"`
}

func NewOwnerMenuHandler(queries *db.Queries) *OwnerMenuHandler {
	return &OwnerMenuHandler{Queries: queries}
}

func (h *OwnerMenuHandler) CreateMenuItem(c *gin.Context) {
	canteenUUID, ok := parseUUIDParam(c, "canteenId", "canteen_id")
	if !ok {
		return
	}

	var req ownerMenuCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	details := map[string]string{}
	if req.Name == "" {
		details["name"] = "required"
	}
	if req.Price < 0 {
		details["price"] = "invalid"
	}
	if req.Stock < 0 {
		details["stock"] = "invalid"
	}
	if len(details) > 0 {
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	isAvailable := true
	if req.IsAvailable != nil {
		isAvailable = *req.IsAvailable
	}

	params := db.CreateMenuItemParams{
		CanteenID:   canteenUUID,
		Name:        req.Name,
		Description: textFromPtr(req.Description),
		Price:       req.Price,
		Stock:       req.Stock,
		IsAvailable: isAvailable,
	}
	item, err := h.Queries.CreateMenuItem(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	resp := menuItemResponse{
		ID:          item.ID.String(),
		CanteenID:   item.CanteenID.String(),
		Name:        item.Name,
		Description: textToPtr(item.Description),
		Price:       item.Price,
		Stock:       item.Stock,
		IsAvailable: item.IsAvailable,
		CreatedAt:   timeToString(item.CreatedAt),
		UpdatedAt:   timeToString(item.UpdatedAt),
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *OwnerMenuHandler) UpdateMenuItem(c *gin.Context) {
	canteenUUID, ok := parseUUIDParam(c, "canteenId", "canteen_id")
	if !ok {
		return
	}
	menuUUID, ok := parseUUIDParam(c, "menuId", "menu_id")
	if !ok {
		return
	}

	var req ownerMenuUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	details := map[string]string{}
	if req.Name == "" {
		details["name"] = "required"
	}
	if req.Price < 0 {
		details["price"] = "invalid"
	}
	if req.Stock < 0 {
		details["stock"] = "invalid"
	}
	if req.IsAvailable == nil {
		details["is_available"] = "required"
	}
	if len(details) > 0 {
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	menu, err := h.Queries.GetMenuItemByID(c.Request.Context(), menuUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	if menu.CanteenID != canteenUUID {
		response.Error(c, http.StatusNotFound, "not_found", nil)
		return
	}

	params := db.UpdateMenuItemParams{
		ID:          menuUUID,
		Name:        req.Name,
		Description: textFromPtr(req.Description),
		Price:       req.Price,
		Stock:       req.Stock,
		IsAvailable: *req.IsAvailable,
	}
	updated, err := h.Queries.UpdateMenuItem(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	resp := menuItemResponse{
		ID:          updated.ID.String(),
		CanteenID:   updated.CanteenID.String(),
		Name:        updated.Name,
		Description: textToPtr(updated.Description),
		Price:       updated.Price,
		Stock:       updated.Stock,
		IsAvailable: updated.IsAvailable,
		CreatedAt:   timeToString(updated.CreatedAt),
		UpdatedAt:   timeToString(updated.UpdatedAt),
	}
	c.JSON(http.StatusOK, resp)
}

func (h *OwnerMenuHandler) DeleteMenuItem(c *gin.Context) {
	canteenUUID, ok := parseUUIDParam(c, "canteenId", "canteen_id")
	if !ok {
		return
	}
	menuUUID, ok := parseUUIDParam(c, "menuId", "menu_id")
	if !ok {
		return
	}

	menu, err := h.Queries.GetMenuItemByID(c.Request.Context(), menuUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	if menu.CanteenID != canteenUUID {
		response.Error(c, http.StatusNotFound, "not_found", nil)
		return
	}

	if err := h.Queries.DeleteMenuItem(c.Request.Context(), db.DeleteMenuItemParams{ID: menuUUID, CanteenID: canteenUUID}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			response.Error(c, http.StatusConflict, "menu_has_dependencies", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	c.Status(http.StatusNoContent)
}
