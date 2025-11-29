package main

import (
	"os"
	"os/signal"
	"syscall"

	"prometheus/internal/bootstrap"
	"prometheus/pkg/logger"
)

// main is the application entrypoint
// All initialization logic has been moved to internal/bootstrap for better organization
func main() {
	// Create dependency injection container
	container := bootstrap.NewContainer()

	// Initialize all components in phases:
	// 1. Config & Logging
	// 2. Infrastructure (Postgres, ClickHouse, Redis)
	// 3. Repositories
	// 4. Adapters (Kafka, Exchanges, Embeddings)
	// 5. Services
	// 6. Business Logic (Risk, Tools, Agents)
	// 7. Application (HTTP, Telegram)
	// 8. Background (Workers, Consumers)
	container.MustInit()

	// Ensure logs are flushed on exit
	defer func() {
		_ = logger.Sync()
	}()

	log := container.Log
	log.Info("✓ System initialization complete", "metrics", container.GetMetrics())

	// Start all background components (consumers, HTTP server)
	if err := container.Start(); err != nil {
		log.Fatalf("Failed to start components: %v", err)
	}

	// ========================================
	// Signal Handling & Graceful Shutdown
	// ========================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until shutdown signal received
	sig := <-quit
	log.Infof("📡 Received signal: %v, initiating graceful shutdown...", sig)

	// Perform coordinated cleanup (8-step shutdown sequence)
	container.Shutdown()

	log.Info("👋 Shutdown complete. Goodbye!")
}
