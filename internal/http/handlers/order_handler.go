package handlers

import (
	"errors"
	"net/http"

	httpcontext "freepass-2026/internal/http/context"
	"freepass-2026/internal/http/response"
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderHandler struct {
	Pool    *pgxpool.Pool
	Queries *db.Queries
}

type createOrderItemRequest struct {
	MenuItemID string `json:"menu_item_id"`
	Qty        int32  `json:"qty"`
}

type createOrderRequest struct {
	CanteenID     string                   `json:"canteen_id"`
	PaymentMethod *string                  `json:"payment_method"`
	Items         []createOrderItemRequest `json:"items"`
}

type orderResponse struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	CanteenID     string `json:"canteen_id"`
	PaymentMethod string `json:"payment_method"`
	PaymentStatus string `json:"payment_status"`
	OrderStatus   string `json:"order_status"`
	TotalAmount   int32  `json:"total_amount"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type orderItemResponse struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	MenuItemID    string `json:"menu_item_id"`
	Qty           int32  `json:"qty"`
	PriceSnapshot int32  `json:"price_snapshot"`
	Subtotal      int32  `json:"subtotal"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type createOrderResponse struct {
	Order orderResponse       `json:"order"`
	Items []orderItemResponse `json:"items"`
}

type qrisInfoResponse struct {
	OrderID       string `json:"order_id"`
	CanteenID     string `json:"canteen_id"`
	QRISStaticURL string `json:"qris_static_url"`
	OrderStatus   string `json:"order_status"`
	PaymentStatus string `json:"payment_status"`
	TotalAmount   int32  `json:"total_amount"`
}

type qrisVerificationResponse struct {
	ID              string  `json:"id"`
	OrderID         string  `json:"order_id"`
	CanteenID       string  `json:"canteen_id"`
	UserID          string  `json:"user_id"`
	Status          string  `json:"status"`
	RejectionReason *string `json:"rejection_reason"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type qrisConfirmResponse struct {
	Order        orderResponse            `json:"order"`
	Verification qrisVerificationResponse `json:"verification"`
}

type orderItemInput struct {
	MenuItemID pgtype.UUID
	Qty        int32
	Price      int32
	Subtotal   int32
}

func NewOrderHandler(pool *pgxpool.Pool, queries *db.Queries) *OrderHandler {
	return &OrderHandler{Pool: pool, Queries: queries}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
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

	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := map[string]string{"body": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	if req.CanteenID == "" {
		details := map[string]string{"canteen_id": "invalid_uuid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}
	var canteenUUID pgtype.UUID
	if err := canteenUUID.Scan(req.CanteenID); err != nil || !canteenUUID.Valid {
		details := map[string]string{"canteen_id": "invalid_uuid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}
	if len(req.Items) == 0 {
		details := map[string]string{"items": "required"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	paymentMethod := "CASH"
	if req.PaymentMethod != nil && *req.PaymentMethod != "" {
		paymentMethod = *req.PaymentMethod
	}
	if paymentMethod != "CASH" && paymentMethod != "CASHLESS_QRIS" {
		details := map[string]string{"payment_method": "invalid"}
		response.Error(c, http.StatusBadRequest, "validation_error", details)
		return
	}

	itemInputs := make([]orderItemInput, 0, len(req.Items))
	for _, item := range req.Items {
		if item.MenuItemID == "" {
			details := map[string]string{"menu_item_id": "invalid_uuid"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
		if item.Qty <= 0 {
			details := map[string]string{"qty": "invalid"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
		var menuUUID pgtype.UUID
		if err := menuUUID.Scan(item.MenuItemID); err != nil || !menuUUID.Valid {
			details := map[string]string{"menu_item_id": "invalid_uuid"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
		itemInputs = append(itemInputs, orderItemInput{MenuItemID: menuUUID, Qty: item.Qty})
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
	_, err = qtx.GetCanteenByID(ctx, canteenUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	total := int64(0)
	for i := range itemInputs {
		menu, err := qtx.LockMenuItemForUpdate(ctx, itemInputs[i].MenuItemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				details := map[string]interface{}{"menu_item_id": itemInputs[i].MenuItemID.String()}
				response.Error(c, http.StatusNotFound, "not_found", details)
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		if menu.CanteenID != canteenUUID {
			response.Error(c, http.StatusConflict, "invalid_menu_canteen", nil)
			return
		}
		if !menu.IsAvailable {
			details := map[string]interface{}{"menu_item_id": itemInputs[i].MenuItemID.String()}
			response.Error(c, http.StatusConflict, "menu_unavailable", details)
			return
		}
		if menu.Stock < itemInputs[i].Qty {
			details := map[string]interface{}{"menu_item_id": itemInputs[i].MenuItemID.String(), "available_stock": menu.Stock}
			response.Error(c, http.StatusConflict, "stock_not_enough", details)
			return
		}
		updated, err := qtx.DecreaseMenuItemStock(ctx, db.DecreaseMenuItemStockParams{ID: itemInputs[i].MenuItemID, Stock: itemInputs[i].Qty})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				details := map[string]interface{}{"menu_item_id": itemInputs[i].MenuItemID.String(), "available_stock": menu.Stock}
				response.Error(c, http.StatusConflict, "stock_not_enough", details)
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		price := updated.Price
		subtotal := price * itemInputs[i].Qty
		itemInputs[i].Price = price
		itemInputs[i].Subtotal = subtotal
		total += int64(subtotal)
	}

	orderStatus := db.OrderStatusWAITING
	if paymentMethod == "CASHLESS_QRIS" {
		orderStatus = db.OrderStatusPAYMENT
	}

	order, err := qtx.CreateOrder(ctx, db.CreateOrderParams{UserID: userUUID, CanteenID: canteenUUID, PaymentMethod: db.PaymentMethod(paymentMethod), PaymentStatus: db.PaymentStatusUNPAID, OrderStatus: orderStatus, TotalAmount: int32(total)})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	items := make([]orderItemResponse, 0, len(itemInputs))
	for _, item := range itemInputs {
		created, err := qtx.CreateOrderItem(ctx, db.CreateOrderItemParams{OrderID: order.ID, MenuItemID: item.MenuItemID, Qty: item.Qty, PriceSnapshot: item.Price, Subtotal: item.Subtotal})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		items = append(items, orderItemResponse{
			ID:            created.ID.String(),
			OrderID:       created.OrderID.String(),
			MenuItemID:    created.MenuItemID.String(),
			Qty:           created.Qty,
			PriceSnapshot: created.PriceSnapshot,
			Subtotal:      created.Subtotal,
			CreatedAt:     timeToString(created.CreatedAt),
			UpdatedAt:     timeToString(created.UpdatedAt),
		})
	}

	if err := tx.Commit(ctx); err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	resp := createOrderResponse{
		Order: orderResponse{
			ID:            order.ID.String(),
			UserID:        order.UserID.String(),
			CanteenID:     order.CanteenID.String(),
			PaymentMethod: string(order.PaymentMethod),
			PaymentStatus: string(order.PaymentStatus),
			OrderStatus:   string(order.OrderStatus),
			TotalAmount:   order.TotalAmount,
			CreatedAt:     timeToString(order.CreatedAt),
			UpdatedAt:     timeToString(order.UpdatedAt),
		},
		Items: items,
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *OrderHandler) ListMyOrders(c *gin.Context) {
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

	items, err := h.Queries.ListOrdersByUserID(c.Request.Context(), userUUID)
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

func (h *OrderHandler) GetOrderQRIS(c *gin.Context) {
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

	order, err := h.Queries.GetOrderByID(c.Request.Context(), orderUUID)
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
	if order.PaymentMethod != db.PaymentMethodCASHLESSQRIS {
		response.Error(c, http.StatusConflict, "invalid_payment_method", nil)
		return
	}

	canteen, err := h.Queries.GetCanteenQRISStaticByID(c.Request.Context(), order.CanteenID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	if !canteen.QrisStaticUrl.Valid || canteen.QrisStaticUrl.String == "" {
		response.Error(c, http.StatusConflict, "qris_not_configured", nil)
		return
	}

	resp := qrisInfoResponse{
		OrderID:       order.ID.String(),
		CanteenID:     order.CanteenID.String(),
		QRISStaticURL: canteen.QrisStaticUrl.String,
		OrderStatus:   string(order.OrderStatus),
		PaymentStatus: string(order.PaymentStatus),
		TotalAmount:   order.TotalAmount,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *OrderHandler) ConfirmOrderQRIS(c *gin.Context) {
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
	if role == "USER" && order.UserID != userUUID {
		response.Error(c, http.StatusForbidden, "forbidden", nil)
		return
	}
	if order.PaymentMethod != db.PaymentMethodCASHLESSQRIS {
		response.Error(c, http.StatusConflict, "invalid_payment_method", nil)
		return
	}
	if order.PaymentStatus == db.PaymentStatusAWAITINGVERIFICATION || order.PaymentStatus == db.PaymentStatusPAID {
		response.Error(c, http.StatusConflict, "payment_already_processed", nil)
		return
	}
	if order.OrderStatus != db.OrderStatusPAYMENT || order.PaymentStatus != db.PaymentStatusUNPAID {
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
	verification, err := qtx.UpsertPaymentVerificationPending(ctx, db.UpsertPaymentVerificationPendingParams{OrderID: order.ID, CanteenID: order.CanteenID, UserID: order.UserID})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	updated, err := qtx.SetOrderCashlessConfirmed(ctx, order.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			details := map[string]string{"order_status": string(order.OrderStatus), "payment_status": string(order.PaymentStatus)}
			response.Error(c, http.StatusConflict, "invalid_order_state", details)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	resp := qrisConfirmResponse{
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
		Verification: qrisVerificationResponse{
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
	c.JSON(http.StatusOK, resp)
}
