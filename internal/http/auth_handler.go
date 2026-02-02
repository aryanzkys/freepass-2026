package http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"freepass-2026/internal/auth"
	"freepass-2026/internal/http/response"
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	Queries *db.Queries
	Auth    *auth.Service
}

type registerRequest struct {
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Phone    *string `json:"phone"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Email string  `json:"email"`
	Role  string  `json:"role"`
	Phone *string `json:"phone"`
}

type loginResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        userResponse `json:"user"`
}

func NewAuthHandler(queries *db.Queries, authService *auth.Service) *AuthHandler {
	return &AuthHandler{Queries: queries, Auth: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
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

	params := db.CreateUserParams{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         db.UserRoleUSER,
		Phone:        phoneText,
	}

	user, err := h.Queries.CreateUser(c.Request.Context(), params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error(c, http.StatusConflict, "email_already_exists", nil)
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
	c.JSON(http.StatusCreated, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := req.Password

	details := map[string]string{}
	if email == "" || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		details["email"] = "invalid"
	}
	if password == "" {
		details["password"] = "required"
	}
	if len(details) > 0 {
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	user, err := h.Queries.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	token, err := h.Auth.IssueToken(user.ID.String(), string(user.Role), time.Hour)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	resp := loginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		User: userResponse{
			ID:    user.ID.String(),
			Name:  user.Name,
			Email: user.Email,
			Role:  string(user.Role),
		},
	}
	if user.Phone.Valid {
		resp.User.Phone = &user.Phone.String
	}
	c.JSON(http.StatusOK, resp)
}
