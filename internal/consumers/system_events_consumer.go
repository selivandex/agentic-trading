package consumers

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"prometheus/internal/adapters/kafka"
	telegram "prometheus/internal/adapters/telegram"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/logger"
	tg "prometheus/pkg/telegram"
)

// SystemEventsConsumer processes all system events with type-based routing
type SystemEventsConsumer struct {
	consumer     *kafka.Consumer
	orchestrator OnboardingOrchestrator
	bot          tg.Bot // Use interface from pkg/telegram
	log          *logger.Logger
}

// OnboardingOrchestrator interface for running portfolio initialization
type OnboardingOrchestrator interface {
	StartOnboarding(ctx context.Context, session *telegram.OnboardingSession) error
}

// NewSystemEventsConsumer creates a new system events consumer
func NewSystemEventsConsumer(
	consumer *kafka.Consumer,
	orchestrator OnboardingOrchestrator,
	bot tg.Bot, // Uses telegram.Bot interface from pkg/telegram
	log *logger.Logger,
) *SystemEventsConsumer {
	return &SystemEventsConsumer{
		consumer:     consumer,
		orchestrator: orchestrator,
		bot:          bot,
		log:          log.With("component", "system_events_consumer"),
	}
}

// Start starts consuming system events
func (sec *SystemEventsConsumer) Start(ctx context.Context) error {
	sec.log.Info("Starting system events consumer...")

	// Ensure consumer is closed on exit
	defer func() {
		sec.log.Info("Closing system events consumer...")
		if err := sec.consumer.Close(); err != nil {
			sec.log.Errorw("Failed to close system events consumer", "error", err)
		} else {
			sec.log.Info("✓ System events consumer closed")
		}
	}()

	sec.log.Infow("Subscribed to system events", "topic", "system.events")

	// Consume messages
	for {
		msg, err := sec.consumer.ReadMessageWithShutdownCheck(ctx)
		if err != nil {
			// Check if error is due to context cancellation or reader closure
			if ctx.Err() != nil {
				sec.log.Info("System events consumer stopped (context cancelled)")
				return nil
			}

			sec.log.Errorw("Failed to read message", "error", err)
			continue
		}

		// Process message
		if err := sec.routeEvent(ctx, msg.Value); err != nil {
			sec.log.Errorw("Failed to process system event",
				"error", err,
				"offset", msg.Offset,
			)
		}
	}
}

// routeEvent routes events based on type
func (sec *SystemEventsConsumer) routeEvent(ctx context.Context, data []byte) error {
	// Try to unmarshal as PortfolioInitializationJobEvent
	var portfolioJob eventspb.PortfolioInitializationJobEvent
	if err := proto.Unmarshal(data, &portfolioJob); err == nil && portfolioJob.Base != nil {
		if portfolioJob.Base.Type == "portfolio.initialization_requested" {
			return sec.handlePortfolioInitialization(ctx, &portfolioJob)
		}
	}

	// Try WorkerFailedEvent
	var workerFailed eventspb.WorkerFailedEvent
	if err := proto.Unmarshal(data, &workerFailed); err == nil && workerFailed.Base != nil {
		if workerFailed.Base.Type == "system.worker_failed" {
			return sec.handleWorkerFailed(ctx, &workerFailed)
		}
	}

	// Unknown event type - log and skip
	sec.log.Debugw("Unknown system event type, skipping",
		"data_size", len(data),
	)

	return nil
}

// handlePortfolioInitialization processes portfolio initialization job
func (sec *SystemEventsConsumer) handlePortfolioInitialization(ctx context.Context, job *eventspb.PortfolioInitializationJobEvent) error {
	sec.log.Infow("Processing portfolio initialization job",
		"user_id", job.UserId,
		"strategy_id", job.StrategyId,
		"capital", job.Capital,
	)

	// Parse UUIDs
	userID, err := uuid.Parse(job.UserId)
	if err != nil {
		sec.log.Errorw("Invalid user_id", "error", err, "user_id", job.UserId)
		return nil
	}

	var strategyID *uuid.UUID
	if job.StrategyId != "" {
		sID, err := uuid.Parse(job.StrategyId)
		if err != nil {
			sec.log.Errorw("Invalid strategy_id", "error", err, "strategy_id", job.StrategyId)
			return nil
		}
		strategyID = &sID
	}

	var accountID *uuid.UUID
	if job.ExchangeAccountId != "" {
		accID, err := uuid.Parse(job.ExchangeAccountId)
		if err != nil {
			sec.log.Errorw("Invalid exchange_account_id", "error", err)
			return nil
		}
		accountID = &accID
	}

	// Create onboarding session
	onboardingSession := &telegram.OnboardingSession{
		TelegramID:        job.TelegramId,
		UserID:            userID,
		StrategyID:        strategyID, // Pass pre-created strategy ID
		Capital:           job.Capital,
		ExchangeAccountID: accountID,
		RiskProfile:       job.RiskProfile,
		MarketType:        job.MarketType,
	}

	// Run portfolio initialization workflow (service will publish notifications)
	if err := sec.orchestrator.StartOnboarding(ctx, onboardingSession); err != nil {
		sec.log.Errorw("Portfolio initialization failed",
			"error", err,
			"user_id", job.UserId,
			"strategy_id", job.StrategyId,
		)

		// Notify user about failure
		_ = sec.bot.SendMessage(job.TelegramId, "❌ Portfolio creation failed.\n\nPlease try again with /invest")
		return nil // Don't retry, user needs to trigger again
	}

	sec.log.Infow("Portfolio initialization completed successfully",
		"user_id", job.UserId,
		"strategy_id", job.StrategyId,
	)

	return nil
}

// handleWorkerFailed handles worker failed events
func (sec *SystemEventsConsumer) handleWorkerFailed(ctx context.Context, event *eventspb.WorkerFailedEvent) error {
	sec.log.Warnw("Worker failure detected",
		"worker", event.WorkerName,
		"error", event.Error,
		"fail_count", event.FailCount,
	)

	// TODO: Add alerting logic (Telegram admin notifications, PagerDuty, etc.)
	return nil
}

// Close closes the consumer
func (sec *SystemEventsConsumer) Close() error {
	sec.log.Info("✓ System events consumer closed")
	return sec.consumer.Close()
}
