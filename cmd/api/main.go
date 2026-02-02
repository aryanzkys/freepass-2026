package main

import (
	"log"

	"freepass-2026/internal/app"
	"freepass-2026/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	engine, cleanup, err := app.Build(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()
	if err := engine.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}
