package bootstrap

import (
	"context"
	"sync"
	"time"

	chclient "prometheus/internal/adapters/clickhouse"
	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/adapters/kafka"
	pgclient "prometheus/internal/adapters/postgres"
	redisclient "prometheus/internal/adapters/redis"
	"prometheus/internal/api"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Lifecycle manages graceful startup and shutdown of components
type Lifecycle struct {
	shutdownTimeout time.Duration
}

// NewLifecycle creates a new lifecycle manager
func NewLifecycle() *Lifecycle {
	return &Lifecycle{
		shutdownTimeout: 150 * time.Second, // 2.5 minutes for complete cleanup
	}
}

// Shutdown performs coordinated cleanup of all components in the correct order
// This is critical for a trading system - we must ensure:
// 1. No new requests accepted
// 2. Workers finish cleanly
// 3. User Data WebSocket connections closed
// 4. Kafka consumers unblock before waiting for goroutines
// 5. Producer closes after consumers
// 6. Logs and errors flushed
// 7. Database connections last (other components may need them)
func (l *Lifecycle) Shutdown(
	wg *sync.WaitGroup,
	httpServer *api.Server,
	workerScheduler *workers.Scheduler,
	marketDataFactory exchanges.CentralFactory,
	userDataManager *UserDataManager,
	kafkaProducer *kafka.Producer,
	notificationConsumer *kafka.Consumer,
	riskConsumer *kafka.Consumer,
	analyticsConsumer *kafka.Consumer,
	opportunityKafkaConsumer *kafka.Consumer,
	aiUsageKafkaConsumer *kafka.Consumer,
	positionGuardianConsumer *kafka.Consumer,
	telegramNotificationConsumer *kafka.Consumer,
	websocketKlineConsumer *kafka.Consumer,
	websocketMarkPriceConsumer *kafka.Consumer,
	websocketTickerConsumer *kafka.Consumer,
	websocketTradeConsumer *kafka.Consumer,
	pgClient *pgclient.Client,
	chClient *chclient.Client,
	redisClient *redisclient.Client,
	errorTracker errors.Tracker,
	log *logger.Logger,
) {
	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), l.shutdownTimeout)
	defer shutdownCancel()

	// ========================================
	// Step 1: Stop HTTP Server (5s timeout)
	// ========================================
	log.Info("[1/9] Stopping HTTP server...")
	httpCtx, httpCancel := context.WithTimeout(shutdownCtx, 5*time.Second)
	defer httpCancel()

	if err := httpServer.Shutdown(httpCtx); err != nil {
		log.Error("HTTP server shutdown failed", "error", err)
	} else {
		log.Info("✓ HTTP server stopped")
	}

	// ========================================
	// Step 2: Stop Background Workers
	// ========================================
	log.Info("[2/10] Stopping background workers...")
	if err := workerScheduler.Stop(); err != nil {
		log.Error("Workers shutdown failed", "error", err)
	} else {
		log.Info("✓ Workers stopped")
	}

	// ========================================
	// Step 3: Stop User Data WebSocket Manager
	// ========================================
	log.Info("[3/10] Stopping User Data WebSocket Manager...")
	if userDataManager != nil && userDataManager.Manager != nil {
		wsCtx, wsCancel := context.WithTimeout(shutdownCtx, 30*time.Second)
		if err := userDataManager.Manager.Stop(wsCtx); err != nil {
			log.Error("User Data Manager shutdown failed", "error", err)
		} else {
			log.Info("✓ User Data Manager stopped")
		}
		wsCancel()
	}

	// ========================================
	// Step 4: Close Kafka Consumers
	// Critical: Close consumers BEFORE waiting for goroutines
	// This unblocks ReadMessage() calls
	// ========================================
	log.Info("[4/10] Closing Kafka consumers...")
	l.closeKafkaConsumers(map[string]*kafka.Consumer{
		"notification":           notificationConsumer,
		"risk":                   riskConsumer,
		"analytics":              analyticsConsumer,
		"opportunity":            opportunityKafkaConsumer,
		"ai_usage":               aiUsageKafkaConsumer,
		"position_guardian":      positionGuardianConsumer,
		"telegram_notifications": telegramNotificationConsumer,
		"websocket_kline":        websocketKlineConsumer,
		"websocket_markprice":    websocketMarkPriceConsumer,
		"websocket_ticker":       websocketTickerConsumer,
		"websocket_trade":        websocketTradeConsumer,
	}, log)
	log.Info("✓ Kafka consumers closed")

	// ========================================
	// Step 5: Wait for Consumer Goroutines
	// ========================================
	log.Info("[5/10] Waiting for consumer goroutines...")
	l.waitForGoroutines(wg, 5*time.Second, log)

	// ========================================
	// Step 6: Close Kafka Producer
	// ========================================
	log.Info("[6/10] Closing Kafka producer...")
	if kafkaProducer != nil {
		if err := kafkaProducer.Close(); err != nil {
			log.Error("Kafka producer close failed", "error", err)
		} else {
			log.Info("✓ Kafka producer closed")
		}
	}

	// ========================================
	// Step 7: Flush Error Tracker
	// ========================================
	log.Info("[7/10] Flushing error tracker...")
	l.flushErrorTracker(errorTracker, shutdownCtx, log)

	// ========================================
	// Step 8: Sync Logs
	// ========================================
	log.Info("[8/10] Syncing logs...")
	if err := logger.Sync(); err != nil {
		log.Warn("Log sync completed with warnings")
	} else {
		log.Info("✓ Logs synced")
	}

	// ========================================
	// Step 9: Close Database Connections
	// LAST - other components may need them during shutdown
	// ========================================
	log.Info("[9/10] Closing database connections...")
	l.closeDatabases(pgClient, chClient, redisClient, log)

	log.Info("✅ Graceful shutdown complete")
}

// closeKafkaConsumers closes all Kafka consumers
func (l *Lifecycle) closeKafkaConsumers(consumers map[string]*kafka.Consumer, log *logger.Logger) {
	for name, consumer := range consumers {
		if consumer != nil {
			if err := consumer.Close(); err != nil {
				log.Error("Kafka consumer close failed", "consumer", name, "error", err)
			}
		}
	}
}

// waitForGoroutines waits for all goroutines with a timeout
func (l *Lifecycle) waitForGoroutines(wg *sync.WaitGroup, timeout time.Duration, log *logger.Logger) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("✓ All goroutines finished")
	case <-time.After(timeout):
		log.Warn("⚠ Some goroutines did not finish within timeout", "timeout", timeout)
	}
}

// flushErrorTracker flushes the error tracker (Sentry, etc.)
func (l *Lifecycle) flushErrorTracker(tracker errors.Tracker, ctx context.Context, log *logger.Logger) {
	if tracker == nil {
		return
	}

	flushCtx, flushCancel := context.WithTimeout(ctx, 3*time.Second)
	defer flushCancel()

	if err := tracker.Flush(flushCtx); err != nil {
		log.Error("Error tracker flush failed", "error", err)
	} else {
		log.Info("✓ Error tracker flushed")
	}
}

// closeDatabases closes all database connections
func (l *Lifecycle) closeDatabases(
	pgClient *pgclient.Client,
	chClient *chclient.Client,
	redisClient *redisclient.Client,
	log *logger.Logger,
) {
	var dbErrors []error

	if pgClient != nil {
		if err := pgClient.Close(); err != nil {
			dbErrors = append(dbErrors, errors.Wrap(err, "postgres"))
		}
	}

	if chClient != nil {
		if err := chClient.Close(); err != nil {
			dbErrors = append(dbErrors, errors.Wrap(err, "clickhouse"))
		}
	}

	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			dbErrors = append(dbErrors, errors.Wrap(err, "redis"))
		}
	}

	if len(dbErrors) > 0 {
		log.Error("Database close errors", "errors", dbErrors)
	} else {
		log.Info("✓ Database connections closed")
	}
}
