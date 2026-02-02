package app

import (
	"context"

	"freepass-2026/internal/config"
	db "freepass-2026/internal/db"
	"freepass-2026/internal/http"
	sqlc "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	Cfg     config.Config
	Pool    *pgxpool.Pool
	Queries *sqlc.Queries
	Engine  *gin.Engine
}

func Build(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	queries := sqlc.New(pool)
	engine := http.NewRouter(http.Dependencies{Queries: queries, Pool: pool})
	return &App{Cfg: cfg, Pool: pool, Queries: queries, Engine: engine}, nil
}
