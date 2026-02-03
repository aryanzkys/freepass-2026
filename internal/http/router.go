package http

import (
	"freepass-2026/internal/auth"
	"freepass-2026/internal/http/handlers"
	"freepass-2026/internal/http/middleware"
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Queries    *db.Queries
	Pool       *pgxpool.Pool
	Auth       *auth.Service
	Canteen    *handlers.CanteenHandler
	Order      *handlers.OrderHandler
	Payment    *handlers.PaymentHandler
	Feedback   *handlers.FeedbackHandler
	OwnerMenu  *handlers.OwnerMenuHandler
	OwnerOrder *handlers.OwnerOrderHandler
}

func NewRouter(deps Dependencies) *gin.Engine {
	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Logger())
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORS())
	h := NewHandler(deps.Queries, deps.Pool)
	authHandler := NewAuthHandler(deps.Queries, deps.Auth)
	engine.GET("/health", h.Health)
	engine.POST("/auth/register", authHandler.Register)
	engine.POST("/auth/login", authHandler.Login)
	engine.GET("/canteens", deps.Canteen.ListCanteens)
	engine.GET("/canteens/:canteenId/menus", deps.Canteen.ListMenusByCanteen)
	orders := engine.Group("/orders")
	orders.Use(middleware.RequireAuth(deps.Auth), middleware.RequireRole("USER", "ADMIN"))
	orders.POST("", deps.Order.CreateOrder)
	orders.POST("/:orderId/payments", deps.Payment.CreatePayment)
	orders.POST("/:orderId/feedback", deps.Feedback.CreateFeedbackForOrder)
	owner := engine.Group("/owner/canteens/:canteenId")
	owner.Use(middleware.RequireAuth(deps.Auth), middleware.RequireRole("OWNER", "ADMIN"), middleware.RequireCanteenOwner(deps.Queries))
	owner.POST("/menus", deps.OwnerMenu.CreateMenuItem)
	owner.PUT("/menus/:menuId", deps.OwnerMenu.UpdateMenuItem)
	owner.DELETE("/menus/:menuId", deps.OwnerMenu.DeleteMenuItem)
	owner.GET("/orders", deps.OwnerOrder.ListIncomingOrders)
	owner.PATCH("/orders/:orderId/status", deps.OwnerOrder.UpdateOrderStatus)
	owner.DELETE("/feedbacks/:feedbackId", deps.Feedback.RemoveFeedbackAsOwner)
	return engine
}
