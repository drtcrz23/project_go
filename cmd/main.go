package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/drtcrz23/project_go/internal/config"
	"github.com/drtcrz23/project_go/internal/logger"
	"github.com/drtcrz23/project_go/internal/server"
	"github.com/drtcrz23/project_go/internal/subpub"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	// Initialize
	log, err := logger.NewLogger(cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	sp := subpub.NewSubPub()
	srv := server.NewServer(cfg, sp, log)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Received shutdown signal")
		cancel()
	}()

	// Run server
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
