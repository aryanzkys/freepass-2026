package handlers

import (
	"errors"
	"net/http"

	"freepass-2026/internal/http/response"
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OwnerOrderHandler struct {
	Pool    *pgxpool.Pool
	Queries *db.Queries
}

type ownerOrderStatusRequest struct {
	Status string `json:"status"`
}

type ownerPaymentVerifyRequest struct {
	Action string  `json:"action"`
	Reason *string `json:"reason"`
}

type verificationResponse struct {
	ID              string  `json:"id"`
	OrderID         string  `json:"order_id"`
	CanteenID       string  `json:"canteen_id"`
	UserID          string  `json:"user_id"`
	Status          string  `json:"status"`
	RejectionReason *string `json:"rejection_reason"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type refundResponse struct {
	ID        string `json:"id"`
	OrderID   string `json:"order_id"`
	CanteenID string `json:"canteen_id"`
	UserID    string `json:"user_id"`
	Amount    int64  `json:"amount"`
	Reason    string `json:"reason"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ownerVerifyPaymentResponse struct {
	Order        orderResponse        `json:"order"`
	Verification verificationResponse `json:"verification"`
	Refund       *refundResponse      `json:"refund"`
}

func NewOwnerOrderHandler(pool *pgxpool.Pool, queries *db.Queries) *OwnerOrderHandler {
	return &OwnerOrderHandler{Pool: pool, Queries: queries}
}

func (h *OwnerOrderHandler) ListIncomingOrders(c *gin.Context) {
	canteenUUID, ok := parseUUIDParam(c, "canteenId", "canteen_id")
	if !ok {
		return
	}
	items, err := h.Queries.ListOrdersByCanteenID(c.Request.Context(), canteenUUID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	resp := make([]orderResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, orderResponse{
			ID:            item.ID.String(),
			UserID:        item.UserID.String(),
			CanteenID:     item.CanteenID.String(),
			PaymentMethod: string(item.PaymentMethod),
			PaymentStatus: string(item.PaymentStatus),
			OrderStatus:   string(item.OrderStatus),
			TotalAmount:   item.TotalAmount,
			CreatedAt:     timeToString(item.CreatedAt),
			UpdatedAt:     timeToString(item.UpdatedAt),
		})
	}
	c.JSON(http.StatusOK, listResponse[orderResponse]{Data: resp})
}

func (h *OwnerOrderHandler) UpdateOrderStatus(c *gin.Context) {
	canteenUUID, ok := parseUUIDParam(c, "canteenId", "canteen_id")
	if !ok {
		return
	}
	orderUUID, ok := parseUUIDParam(c, "orderId", "order_id")
	if !ok {
		return
	}

	var req ownerOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}
	if !isValidOrderStatus(req.Status) {
		details := map[string]string{"status": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	order, err := h.Queries.GetOrderByID(c.Request.Context(), orderUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	if order.CanteenID != canteenUUID {
		response.Error(c, http.StatusNotFound, "not_found", nil)
		return
	}
	if order.PaymentMethod == db.PaymentMethodCASHLESSQRIS && order.PaymentStatus != db.PaymentStatusPAID {
		response.Error(c, http.StatusConflict, "order_not_paid", nil)
		return
	}

	from := string(order.OrderStatus)
	to := req.Status
	if !isValidTransition(from, to) {
		details := map[string]string{"from": from, "to": to}
		response.Error(c, http.StatusConflict, "invalid_order_status_transition", details)
		return
	}

	var updated db.Order
	if order.PaymentMethod == db.PaymentMethodCASH {
		updated, err = h.Queries.UpdateOrderStatus(c.Request.Context(), db.UpdateOrderStatusParams{ID: orderUUID, OrderStatus: db.OrderStatus(to)})
	} else {
		updated, err = h.Queries.UpdateOrderStatusIfPaid(c.Request.Context(), db.UpdateOrderStatusIfPaidParams{ID: orderUUID, OrderStatus: db.OrderStatus(to)})
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusConflict, "order_not_paid", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	resp := orderResponse{
		ID:            updated.ID.String(),
		UserID:        updated.UserID.String(),
		CanteenID:     updated.CanteenID.String(),
		PaymentMethod: string(updated.PaymentMethod),
		PaymentStatus: string(updated.PaymentStatus),
		OrderStatus:   string(updated.OrderStatus),
		TotalAmount:   updated.TotalAmount,
		CreatedAt:     timeToString(updated.CreatedAt),
		UpdatedAt:     timeToString(updated.UpdatedAt),
	}
	c.JSON(http.StatusOK, resp)
}

func (h *OwnerOrderHandler) VerifyOrderPayment(c *gin.Context) {
	canteenUUID, ok := parseUUIDParam(c, "canteenId", "canteen_id")
	if !ok {
		return
	}
	orderUUID, ok := parseUUIDParam(c, "orderId", "order_id")
	if !ok {
		return
	}

	var req ownerPaymentVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}
	if req.Action != "APPROVE" && req.Action != "REJECT" {
		details := map[string]string{"action": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}
	if req.Action == "REJECT" {
		if req.Reason == nil || *req.Reason == "" {
			details := map[string]string{"reason": "required"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
	}

	ctx := c.Request.Context()
	order, err := h.Queries.GetOrderByID(ctx, orderUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	if order.CanteenID != canteenUUID {
		response.Error(c, http.StatusNotFound, "not_found", nil)
		return
	}
	if order.PaymentMethod != db.PaymentMethodCASHLESSQRIS {
		response.Error(c, http.StatusConflict, "invalid_payment_method", nil)
		return
	}
	if order.PaymentStatus != db.PaymentStatusAWAITINGVERIFICATION || order.OrderStatus != db.OrderStatusWAITING {
		details := map[string]string{"order_status": string(order.OrderStatus), "payment_status": string(order.PaymentStatus)}
		response.Error(c, http.StatusConflict, "invalid_order_state", details)
		return
	}

	tx, err := h.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := db.New(tx)
	var verification db.PaymentVerification
	var updated db.Order
	var refund *db.Refund
	if req.Action == "APPROVE" {
		verification, err = qtx.UpdatePaymentVerificationApproved(ctx, order.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				response.Error(c, http.StatusConflict, "invalid_order_state", nil)
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		updated, err = qtx.SetOrderPaymentStatus(ctx, db.SetOrderPaymentStatusParams{ID: order.ID, PaymentStatus: db.PaymentStatusPAID})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
	} else {
		verification, err = qtx.UpdatePaymentVerificationRejected(ctx, db.UpdatePaymentVerificationRejectedParams{OrderID: order.ID, RejectionReason: pgtype.Text{String: *req.Reason, Valid: true}})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				response.Error(c, http.StatusConflict, "invalid_order_state", nil)
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		updated, err = qtx.SetOrderPaymentStatusAndOrderStatus(ctx, db.SetOrderPaymentStatusAndOrderStatusParams{ID: order.ID, PaymentStatus: db.PaymentStatusREJECTED, OrderStatus: db.OrderStatusPAYMENT})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		createdRefund, err := qtx.CreateRefund(ctx, db.CreateRefundParams{OrderID: order.ID, CanteenID: order.CanteenID, UserID: order.UserID, Amount: int64(order.TotalAmount), Reason: *req.Reason, Status: "PENDING"})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		refund = &createdRefund
	}

	if err := tx.Commit(ctx); err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	resp := ownerVerifyPaymentResponse{
		Order: orderResponse{
			ID:            updated.ID.String(),
			UserID:        updated.UserID.String(),
			CanteenID:     updated.CanteenID.String(),
			PaymentMethod: string(updated.PaymentMethod),
			PaymentStatus: string(updated.PaymentStatus),
			OrderStatus:   string(updated.OrderStatus),
			TotalAmount:   updated.TotalAmount,
			CreatedAt:     timeToString(updated.CreatedAt),
			UpdatedAt:     timeToString(updated.UpdatedAt),
		},
		Verification: verificationResponse{
			ID:        verification.ID.String(),
			OrderID:   verification.OrderID.String(),
			CanteenID: verification.CanteenID.String(),
			UserID:    verification.UserID.String(),
			Status:    verification.Status,
			CreatedAt: timeToString(verification.CreatedAt),
			UpdatedAt: timeToString(verification.UpdatedAt),
		},
	}
	if verification.RejectionReason.Valid {
		reason := verification.RejectionReason.String
		resp.Verification.RejectionReason = &reason
	}
	if refund != nil {
		resp.Refund = &refundResponse{
			ID:        refund.ID.String(),
			OrderID:   refund.OrderID.String(),
			CanteenID: refund.CanteenID.String(),
			UserID:    refund.UserID.String(),
			Amount:    refund.Amount,
			Reason:    refund.Reason,
			Status:    refund.Status,
			CreatedAt: timeToString(refund.CreatedAt),
			UpdatedAt: timeToString(refund.UpdatedAt),
		}
	}
	c.JSON(http.StatusOK, resp)
}

func isValidOrderStatus(value string) bool {
	switch value {
	case "WAITING", "COOKING", "READY", "COMPLETED":
		return true
	default:
		return false
	}
}

func isValidTransition(from string, to string) bool {
	switch from {
	case "WAITING":
		return to == "COOKING"
	case "COOKING":
		return to == "READY"
	case "READY":
		return to == "COMPLETED"
	default:
		return false
	}
}
