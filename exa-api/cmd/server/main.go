package main

import (
	"context"
	"log"
	"time"

	"github.com/ArminDashti/exa-api/internal/auth"
	"github.com/ArminDashti/exa-api/internal/config"
	httpserver "github.com/ArminDashti/exa-api/internal/http"
	"github.com/ArminDashti/exa-api/internal/store"
)

func main() {
	config.LoadDotEnv(".env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := store.Migrate(db, cfg.MigrationsDir); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	hash, err := auth.HashPassword(store.DefaultPassword)
	if err != nil {
		log.Fatalf("hash default password: %v", err)
	}
	if err := store.SeedDefaultUser(ctx, db, hash); err != nil {
		log.Fatalf("seed default user: %v", err)
	}

	srv := httpserver.New(cfg, db)
	log.Printf("exa-api listening on %s", cfg.Addr)
	if err := srv.Router().Run(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}
