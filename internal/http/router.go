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
	Admin      *handlers.AdminHandler
}

func NewRouter(deps Dependencies) *gin.Engine {
	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Logger())
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORS())
	h := NewHandler(deps.Queries, deps.Pool)
	authHandler := NewAuthHandler(deps.Queries, deps.Auth)
	profileHandler := NewProfileHandler(deps.Queries)
	engine.GET("/health", h.Health)
	engine.POST("/auth/register", authHandler.Register)
	engine.POST("/auth/login", authHandler.Login)
	me := engine.Group("/me")
	me.Use(middleware.RequireAuth(deps.Auth))
	me.GET("", profileHandler.GetMe)
	me.PUT("", profileHandler.UpdateMe)
	engine.GET("/canteens", deps.Canteen.ListCanteens)
	engine.GET("/canteens/:canteenId/menus", deps.Canteen.ListMenusByCanteen)
	orders := engine.Group("/orders")
	orders.Use(middleware.RequireAuth(deps.Auth), middleware.RequireRole("USER", "ADMIN"))
	orders.GET("", deps.Order.ListMyOrders)
	orders.POST("", deps.Order.CreateOrder)
	orders.POST("/:orderId/payments", deps.Payment.CreatePayment)
	orders.POST("/:orderId/feedback", deps.Feedback.CreateFeedbackForOrder)
	orders.GET("/:orderId/payment/qris", deps.Order.GetOrderQRIS)
	orders.POST("/:orderId/payment/qris/confirm", deps.Order.ConfirmOrderQRIS)
	owner := engine.Group("/owner/canteens/:canteenId")
	owner.Use(middleware.RequireAuth(deps.Auth), middleware.RequireRole("OWNER", "ADMIN"), middleware.RequireCanteenOwner(deps.Queries))
	owner.POST("/menus", deps.OwnerMenu.CreateMenuItem)
	owner.PUT("/menus/:menuId", deps.OwnerMenu.UpdateMenuItem)
	owner.DELETE("/menus/:menuId", deps.OwnerMenu.DeleteMenuItem)
	owner.GET("/orders", deps.OwnerOrder.ListIncomingOrders)
	owner.PATCH("/orders/:orderId/status", deps.OwnerOrder.UpdateOrderStatus)
	owner.PATCH("/orders/:orderId/payment/verify", deps.OwnerOrder.VerifyOrderPayment)
	owner.DELETE("/feedbacks/:feedbackId", deps.Feedback.RemoveFeedbackAsOwner)
	admin := engine.Group("/admin")
	admin.Use(middleware.RequireAuth(deps.Auth), middleware.RequireRole("ADMIN"))
	admin.POST("/owners", deps.Admin.CreateOwner)
	admin.PUT("/owners/:ownerId", deps.Admin.UpdateOwner)
	admin.DELETE("/accounts/:userId", deps.Admin.DeleteAccount)
	admin.PUT("/canteens/:canteenId/qris", deps.Admin.UpdateCanteenQRIS)
	return engine
}
