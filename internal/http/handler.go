package http

import (
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	Queries *db.Queries
	Pool    *pgxpool.Pool
}

func NewHandler(queries *db.Queries, pool *pgxpool.Pool) *Handler {
	return &Handler{Queries: queries, Pool: pool}
}

func (h *Handler) Health(c *gin.Context) {
	if h.Queries == nil || h.Pool == nil {
		c.JSON(200, gin.H{"status": "ok"})
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}
