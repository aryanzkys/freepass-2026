package handlers

import (
	"errors"
	"net/http"

	httpcontext "freepass-2026/internal/http/context"
	"freepass-2026/internal/http/response"
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FeedbackHandler struct {
	Pool    *pgxpool.Pool
	Queries *db.Queries
}

type createFeedbackRequest struct {
	Comment *string                     `json:"comment"`
	Items   []createFeedbackItemRequest `json:"items"`
}

type createFeedbackItemRequest struct {
	MenuItemID string `json:"menu_item_id"`
	Rating     int32  `json:"rating"`
}

type feedbackResponse struct {
	ID        string  `json:"id"`
	OrderID   string  `json:"order_id"`
	UserID    string  `json:"user_id"`
	CanteenID string  `json:"canteen_id"`
	Rating    *int32  `json:"rating"`
	Comment   *string `json:"comment"`
	IsRemoved bool    `json:"is_removed"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type feedbackWrapperResponse struct {
	Feedback feedbackResponse `json:"feedback"`
}

func NewFeedbackHandler(pool *pgxpool.Pool, queries *db.Queries) *FeedbackHandler {
	return &FeedbackHandler{Pool: pool, Queries: queries}
}

func (h *FeedbackHandler) CreateFeedbackForOrder(c *gin.Context) {
	role, ok := httpcontext.UserRole(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	if role != "USER" && role != "ADMIN" {
		response.Error(c, http.StatusForbidden, "forbidden", nil)
		return
	}
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

	orderUUID, ok := parseUUIDParam(c, "orderId", "order_id")
	if !ok {
		return
	}

	var req createFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}
	if len(req.Items) == 0 {
		details := map[string]string{"items": "required"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	itemRatings := make(map[pgtype.UUID]int32, len(req.Items))
	for _, item := range req.Items {
		if item.MenuItemID == "" {
			details := map[string]string{"menu_item_id": "invalid_uuid"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
		var menuUUID pgtype.UUID
		if err := menuUUID.Scan(item.MenuItemID); err != nil || !menuUUID.Valid {
			details := map[string]string{"menu_item_id": "invalid_uuid"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
		if item.Rating < 1 || item.Rating > 5 {
			details := map[string]string{"rating": "must_be_between_1_and_5"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
		if _, exists := itemRatings[menuUUID]; exists {
			response.Error(c, http.StatusConflict, "invalid_rated_items", nil)
			return
		}
		itemRatings[menuUUID] = item.Rating
	}

	ctx := c.Request.Context()
	tx, err := h.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := db.New(tx)
	order, err := qtx.GetOrderByID(ctx, orderUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	if role == "USER" && order.UserID != userUUID {
		response.Error(c, http.StatusForbidden, "forbidden", nil)
		return
	}

	if order.PaymentStatus != db.PaymentStatusPAID {
		response.Error(c, http.StatusConflict, "order_not_paid", nil)
		return
	}
	if order.OrderStatus != db.OrderStatusCOMPLETED {
		response.Error(c, http.StatusConflict, "order_not_completed", nil)
		return
	}

	orderItems, err := qtx.GetOrderItemsMenuIDsByOrderID(ctx, orderUUID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	orderMenuItems := make(map[pgtype.UUID]struct{}, len(orderItems))
	for _, item := range orderItems {
		orderMenuItems[item.MenuItemID] = struct{}{}
	}
	if len(orderMenuItems) != len(itemRatings) {
		response.Error(c, http.StatusConflict, "invalid_rated_items", nil)
		return
	}
	for menuID := range itemRatings {
		if _, ok := orderMenuItems[menuID]; !ok {
			response.Error(c, http.StatusConflict, "invalid_rated_items", nil)
			return
		}
	}

	_, err = qtx.GetFeedbackByOrderID(ctx, orderUUID)
	if err == nil {
		response.Error(c, http.StatusConflict, "feedback_already_exists", nil)
		return
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	existingRatings, err := qtx.GetMenuRatingsByOrderID(ctx, orderUUID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	if len(existingRatings) > 0 {
		response.Error(c, http.StatusConflict, "rating_already_exists", nil)
		return
	}

	created, err := qtx.CreateFeedback(ctx, db.CreateFeedbackParams{
		OrderID:   order.ID,
		UserID:    order.UserID,
		CanteenID: order.CanteenID,
		Rating:    pgtype.Int4{},
		Comment:   textFromPtr(req.Comment),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error(c, http.StatusConflict, "feedback_already_exists", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	for menuID, rating := range itemRatings {
		_, err := qtx.CreateMenuRating(ctx, db.CreateMenuRatingParams{
			OrderID:    order.ID,
			MenuItemID: menuID,
			UserID:     order.UserID,
			CanteenID:  order.CanteenID,
			Rating:     rating,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				response.Error(c, http.StatusConflict, "rating_already_exists", nil)
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	resp := feedbackWrapperResponse{
		Feedback: feedbackResponse{
			ID:        created.ID.String(),
			OrderID:   created.OrderID.String(),
			UserID:    created.UserID.String(),
			CanteenID: created.CanteenID.String(),
			Rating:    int4ToPtr(created.Rating),
			Comment:   textToPtr(created.Comment),
			IsRemoved: created.IsRemoved,
			CreatedAt: timeToString(created.CreatedAt),
			UpdatedAt: timeToString(created.UpdatedAt),
		},
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *FeedbackHandler) RemoveFeedbackAsOwner(c *gin.Context) {
	canteenUUID, ok := parseUUIDParam(c, "canteenId", "canteen_id")
	if !ok {
		return
	}
	feedbackUUID, ok := parseUUIDParam(c, "feedbackId", "feedback_id")
	if !ok {
		return
	}

	ctx := c.Request.Context()
	tx, err := h.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := db.New(tx)
	removed, err := qtx.SoftRemoveFeedbackByID(ctx, db.SoftRemoveFeedbackByIDParams{ID: feedbackUUID, CanteenID: canteenUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	if err := qtx.SoftRemoveMenuRatingsByOrderID(ctx, removed.OrderID); err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	resp := feedbackWrapperResponse{
		Feedback: feedbackResponse{
			ID:        removed.ID.String(),
			OrderID:   removed.OrderID.String(),
			UserID:    removed.UserID.String(),
			CanteenID: removed.CanteenID.String(),
			Rating:    int4ToPtr(removed.Rating),
			Comment:   textToPtr(removed.Comment),
			IsRemoved: removed.IsRemoved,
			CreatedAt: timeToString(removed.CreatedAt),
			UpdatedAt: timeToString(removed.UpdatedAt),
		},
	}
	c.JSON(http.StatusOK, resp)
}

func int4ToPtr(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	val := value.Int32
	return &val
}
