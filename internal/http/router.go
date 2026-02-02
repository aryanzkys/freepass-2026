package http

import (
	"freepass-2026/internal/http/middleware"
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Queries *db.Queries
	Pool    *pgxpool.Pool
}

func NewRouter(deps Dependencies) *gin.Engine {
	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Logger())
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORS())
	h := NewHandler(deps.Queries, deps.Pool)
	engine.GET("/health", h.Health)
	return engine
}
