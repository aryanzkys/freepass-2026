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
	CanteenID string                   `json:"canteen_id"`
	Items     []createOrderItemRequest `json:"items"`
}

type orderResponse struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	CanteenID     string `json:"canteen_id"`
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

	order, err := qtx.CreateOrder(ctx, db.CreateOrderParams{UserID: userUUID, CanteenID: canteenUUID, TotalAmount: int32(total)})
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
