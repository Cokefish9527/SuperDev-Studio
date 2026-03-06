package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"superdevstudio/internal/app"
)

func main() {
	cfg := app.LoadConfig()
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}
	defer func() {
		_ = application.Close()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("super-dev studio backend listening on %s", cfg.Addr)
	if err := application.Run(ctx); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
