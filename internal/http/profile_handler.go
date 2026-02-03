package http

import (
	"errors"
	"net/http"
	"strings"

	httpcontext "freepass-2026/internal/http/context"
	"freepass-2026/internal/http/response"
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProfileHandler struct {
	Queries *db.Queries
}

type updateProfileRequest struct {
	Name  *string `json:"name"`
	Phone *string `json:"phone"`
}

func NewProfileHandler(queries *db.Queries) *ProfileHandler {
	return &ProfileHandler{Queries: queries}
}

func (h *ProfileHandler) GetMe(c *gin.Context) {
	userID, ok := httpcontext.UserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil || !userUUID.Valid {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	user, err := h.Queries.GetUserByID(c.Request.Context(), userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	resp := userResponse{
		ID:    user.ID.String(),
		Name:  user.Name,
		Email: user.Email,
		Role:  string(user.Role),
	}
	if user.Phone.Valid {
		resp.Phone = &user.Phone.String
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ProfileHandler) UpdateMe(c *gin.Context) {
	userID, ok := httpcontext.UserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil || !userUUID.Valid {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}
	if req.Name == nil && req.Phone == nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	user, err := h.Queries.GetUserByID(c.Request.Context(), userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	name := user.Name
	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		if v == "" {
			details := map[string]string{"name": "required"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
		name = v
	}

	phone := user.Phone
	if req.Phone != nil {
		v := strings.TrimSpace(*req.Phone)
		if v == "" {
			phone = pgtype.Text{}
		} else {
			phone = pgtype.Text{String: v, Valid: true}
		}
	}

	updated, err := h.Queries.UpdateUserProfile(c.Request.Context(), db.UpdateUserProfileParams{ID: user.ID, Name: name, Phone: phone})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	resp := userResponse{
		ID:    updated.ID.String(),
		Name:  updated.Name,
		Email: updated.Email,
		Role:  string(updated.Role),
	}
	if updated.Phone.Valid {
		resp.Phone = &updated.Phone.String
	}
	c.JSON(http.StatusOK, resp)
}
