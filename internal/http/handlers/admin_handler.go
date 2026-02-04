package handlers

import (
	"errors"
	"net/http"
	"strings"

	"freepass-2026/internal/config"
	"freepass-2026/internal/http/response"
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	Queries *db.Queries
	Cfg     config.Config
}

type adminCreateOwnerRequest struct {
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Phone    *string `json:"phone"`
}

type adminUpdateOwnerRequest struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
	Phone    *string `json:"phone"`
}

type adminUserResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	Phone     *string `json:"phone"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type adminUserWrapperResponse struct {
	User adminUserResponse `json:"user"`
}

type adminDeleteResponse struct {
	Deleted bool `json:"deleted"`
}

type adminUpdateQrisRequest struct {
	QRISStaticURL string `json:"qris_static_url"`
}

type adminCanteenResponse struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Location            *string `json:"location"`
	OwnerID             string  `json:"owner_id"`
	QRISStaticURL       *string `json:"qris_static_url"`
	QRISStaticUpdatedAt *string `json:"qris_static_updated_at"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

type adminCanteenWrapperResponse struct {
	Canteen adminCanteenResponse `json:"canteen"`
}

func NewAdminHandler(queries *db.Queries, cfg config.Config) *AdminHandler {
	return &AdminHandler{Queries: queries, Cfg: cfg}
}

func (h *AdminHandler) CreateOwner(c *gin.Context) {
	var req adminCreateOwnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := req.Password
	var phoneText pgtype.Text
	if req.Phone != nil {
		p := strings.TrimSpace(*req.Phone)
		if p != "" {
			phoneText = pgtype.Text{String: p, Valid: true}
		}
	}

	details := map[string]string{}
	if name == "" {
		details["name"] = "required"
	}
	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		details["email"] = "invalid"
	}
	if len(password) < 8 {
		details["password"] = "min_length_8"
	}
	if len(details) > 0 {
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	user, err := h.Queries.CreateUser(c.Request.Context(), db.CreateUserParams{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         db.UserRoleOWNER,
		Phone:        phoneText,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error(c, http.StatusConflict, "email_already_exists", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	resp := adminUserWrapperResponse{User: adminUserResponse{
		ID:        user.ID.String(),
		Name:      user.Name,
		Email:     user.Email,
		Role:      string(user.Role),
		CreatedAt: timeToString(user.CreatedAt),
		UpdatedAt: timeToString(user.UpdatedAt),
	}}
	if user.Phone.Valid {
		resp.User.Phone = &user.Phone.String
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *AdminHandler) UpdateOwner(c *gin.Context) {
	ownerUUID, ok := parseUUIDParam(c, "ownerId", "owner_id")
	if !ok {
		return
	}

	var req adminUpdateOwnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	if req.Name == nil && req.Email == nil && req.Password == nil && req.Phone == nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	user, err := h.Queries.GetUserByID(c.Request.Context(), ownerUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	if user.Role != db.UserRoleOWNER {
		response.Error(c, http.StatusNotFound, "not_found", nil)
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

	email := user.Email
	if req.Email != nil {
		v := strings.ToLower(strings.TrimSpace(*req.Email))
		if v == "" || !strings.Contains(v, "@") || !strings.Contains(v, ".") {
			details := map[string]string{"email": "invalid"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
		email = v
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

	if req.Password != nil {
		if len(*req.Password) < 8 {
			details := map[string]string{"password": "min_length_8"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		updated, err := h.Queries.UpdateUserAdminWithPassword(c.Request.Context(), db.UpdateUserAdminWithPasswordParams{
			ID:           user.ID,
			Name:         name,
			Email:        email,
			Phone:        phone,
			PasswordHash: string(hash),
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				response.Error(c, http.StatusConflict, "email_already_exists", nil)
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		resp := adminUserWrapperResponse{User: adminUserResponse{
			ID:        updated.ID.String(),
			Name:      updated.Name,
			Email:     updated.Email,
			Role:      string(updated.Role),
			CreatedAt: timeToString(updated.CreatedAt),
			UpdatedAt: timeToString(updated.UpdatedAt),
		}}
		if updated.Phone.Valid {
			resp.User.Phone = &updated.Phone.String
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	updated, err := h.Queries.UpdateUserAdminNoPassword(c.Request.Context(), db.UpdateUserAdminNoPasswordParams{
		ID:    user.ID,
		Name:  name,
		Email: email,
		Phone: phone,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error(c, http.StatusConflict, "email_already_exists", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	resp := adminUserWrapperResponse{User: adminUserResponse{
		ID:        updated.ID.String(),
		Name:      updated.Name,
		Email:     updated.Email,
		Role:      string(updated.Role),
		CreatedAt: timeToString(updated.CreatedAt),
		UpdatedAt: timeToString(updated.UpdatedAt),
	}}
	if updated.Phone.Valid {
		resp.User.Phone = &updated.Phone.String
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AdminHandler) DeleteAccount(c *gin.Context) {
	userUUID, ok := parseUUIDParam(c, "userId", "user_id")
	if !ok {
		return
	}

	_, err := h.Queries.GetUserByID(c.Request.Context(), userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	if err := h.Queries.DeleteUserByID(c.Request.Context(), userUUID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			response.Error(c, http.StatusConflict, "account_has_dependencies", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	c.JSON(http.StatusOK, adminDeleteResponse{Deleted: true})
}

func (h *AdminHandler) UpdateCanteenQRIS(c *gin.Context) {
	canteenUUID, ok := parseUUIDParam(c, "canteenId", "canteen_id")
	if !ok {
		return
	}

	var req adminUpdateQrisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}
	qrisURL := strings.TrimSpace(req.QRISStaticURL)
	if qrisURL == "" || (!strings.HasPrefix(qrisURL, "http://") && !strings.HasPrefix(qrisURL, "https://")) {
		details := map[string]string{"qris_static_url": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	updated, err := h.Queries.UpdateCanteenQRISStatic(c.Request.Context(), db.UpdateCanteenQRISStaticParams{ID: canteenUUID, QrisStaticUrl: pgtype.Text{String: qrisURL, Valid: true}})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	resp := adminCanteenWrapperResponse{Canteen: adminCanteenResponse{
		ID:        updated.ID.String(),
		Name:      updated.Name,
		OwnerID:   updated.OwnerID.String(),
		CreatedAt: timeToString(updated.CreatedAt),
		UpdatedAt: timeToString(updated.UpdatedAt),
	}}
	if updated.Location.Valid {
		resp.Canteen.Location = &updated.Location.String
	}
	if updated.QrisStaticUrl.Valid {
		resp.Canteen.QRISStaticURL = &updated.QrisStaticUrl.String
	}
	if updated.QrisStaticUpdatedAt.Valid {
		value := timeToString(updated.QrisStaticUpdatedAt)
		resp.Canteen.QRISStaticUpdatedAt = &value
	}
	c.JSON(http.StatusOK, resp)
}
