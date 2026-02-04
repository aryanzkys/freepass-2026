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

type PaymentHandler struct {
	Pool    *pgxpool.Pool
	Queries *db.Queries
}

type createPaymentRequest struct {
	Method string `json:"method"`
	Amount int32  `json:"amount"`
}

type paymentResponse struct {
	ID        string  `json:"id"`
	OrderID   string  `json:"order_id"`
	Method    *string `json:"method"`
	Amount    int32   `json:"amount"`
	Status    string  `json:"status"`
	PaidAt    string  `json:"paid_at"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type paymentOrderResponse struct {
	Payment paymentResponse `json:"payment"`
	Order   orderResponse   `json:"order"`
}

func NewPaymentHandler(pool *pgxpool.Pool, queries *db.Queries) *PaymentHandler {
	return &PaymentHandler{Pool: pool, Queries: queries}
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
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

	orderID := c.Param("orderId")
	if orderID == "" {
		details := map[string]string{"order_id": "invalid_uuid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}
	var orderUUID pgtype.UUID
	if err := orderUUID.Scan(orderID); err != nil || !orderUUID.Valid {
		details := map[string]string{"order_id": "invalid_uuid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	var req createPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}
	if req.Method == "" {
		details := map[string]string{"method": "required"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}
	if req.Amount <= 0 {
		details := map[string]string{"amount": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
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
	if order.PaymentMethod != db.PaymentMethodCASH {
		response.Error(c, http.StatusConflict, "invalid_payment_method", nil)
		return
	}

	if order.PaymentStatus == "PAID" {
		response.Error(c, http.StatusConflict, "payment_already_processed", nil)
		return
	}

	_, err = qtx.GetPaymentByOrderID(ctx, orderUUID)
	if err == nil {
		response.Error(c, http.StatusConflict, "payment_already_processed", nil)
		return
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	if req.Amount != order.TotalAmount {
		details := map[string]interface{}{"expected": order.TotalAmount, "received": req.Amount}
		response.Error(c, http.StatusConflict, "invalid_payment_amount", details)
		return
	}

	payment, err := qtx.CreatePayment(ctx, db.CreatePaymentParams{
		OrderID: orderUUID,
		Method:  pgtype.Text{String: req.Method, Valid: true},
		Amount:  req.Amount,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error(c, http.StatusConflict, "payment_already_processed", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	paidOrder, err := qtx.MarkOrderPaid(ctx, orderUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusConflict, "payment_already_processed", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	var method *string
	if payment.Method.Valid {
		method = &payment.Method.String
	}

	resp := paymentOrderResponse{
		Payment: paymentResponse{
			ID:        payment.ID.String(),
			OrderID:   payment.OrderID.String(),
			Method:    method,
			Amount:    payment.Amount,
			Status:    payment.Status,
			PaidAt:    timeToString(payment.PaidAt),
			CreatedAt: timeToString(payment.CreatedAt),
			UpdatedAt: timeToString(payment.UpdatedAt),
		},
		Order: orderResponse{
			ID:            paidOrder.ID.String(),
			UserID:        paidOrder.UserID.String(),
			CanteenID:     paidOrder.CanteenID.String(),
			PaymentMethod: string(paidOrder.PaymentMethod),
			PaymentStatus: string(paidOrder.PaymentStatus),
			OrderStatus:   string(paidOrder.OrderStatus),
			TotalAmount:   paidOrder.TotalAmount,
			CreatedAt:     timeToString(paidOrder.CreatedAt),
			UpdatedAt:     timeToString(paidOrder.UpdatedAt),
		},
	}
	c.JSON(http.StatusCreated, resp)
}
