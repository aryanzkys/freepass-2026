package app

import (
	"freepass-2026/internal/config"
	"freepass-2026/internal/db"
	"freepass-2026/internal/http"
	"github.com/gin-gonic/gin"
)

func Build(cfg config.Config) (*gin.Engine, func(), error) {
	conn, err := db.New(cfg.DatabaseURL)
	if err != nil {
		return nil, func() {}, err
	}
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(func(c *gin.Context) {
		c.Set("db", conn)
		c.Next()
	})
	http.RegisterRoutes(engine)
	cleanup := func() {
		_ = conn.Close()
	}
	return engine, cleanup, nil
}
