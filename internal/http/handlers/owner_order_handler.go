package handlers

import (
	"errors"
	"net/http"

	"freepass-2026/internal/http/response"
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type OwnerOrderHandler struct {
	Queries *db.Queries
}

type ownerOrderStatusRequest struct {
	Status string `json:"status"`
}

func NewOwnerOrderHandler(queries *db.Queries) *OwnerOrderHandler {
	return &OwnerOrderHandler{Queries: queries}
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
	if order.PaymentStatus != db.PaymentStatusPAID {
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

	updated, err := h.Queries.UpdateOrderStatusIfPaid(c.Request.Context(), db.UpdateOrderStatusIfPaidParams{ID: orderUUID, OrderStatus: db.OrderStatus(to)})
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
		PaymentStatus: string(updated.PaymentStatus),
		OrderStatus:   string(updated.OrderStatus),
		TotalAmount:   updated.TotalAmount,
		CreatedAt:     timeToString(updated.CreatedAt),
		UpdatedAt:     timeToString(updated.UpdatedAt),
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
