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
	Queries *db.Queries
	Pool    *pgxpool.Pool
	Auth    *auth.Service
	Canteen *handlers.CanteenHandler
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
	engine.Group("/").Use(middleware.RequireAuth(deps.Auth))
	engine.Group("/").Use(middleware.RequireAuth(deps.Auth), middleware.RequireRole("ADMIN"))
	engine.Group("/").Use(middleware.RequireAuth(deps.Auth), middleware.RequireRole("OWNER", "ADMIN"), middleware.RequireCanteenOwner(deps.Queries))
	return engine
}
