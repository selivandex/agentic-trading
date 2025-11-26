package consumers

import (
	"context"
	"sync"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/agents"
	"prometheus/internal/domain/trading_pair"
	"prometheus/internal/domain/user"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"

	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// OpportunityConsumer handles market opportunity events with immediate agent analysis
// This enables event-driven trading: opportunity detected → instant analysis → quick execution
type OpportunityConsumer struct {
	consumer        *kafka.Consumer
	userRepo        user.Repository
	tradingPairRepo trading_pair.Repository
	agentFactory    *agents.Factory
	maxConcurrency  int
	log             *logger.Logger
}

// NewOpportunityConsumer creates a new opportunity event consumer
func NewOpportunityConsumer(
	consumer *kafka.Consumer,
	userRepo user.Repository,
	tradingPairRepo trading_pair.Repository,
	agentFactory *agents.Factory,
	maxConcurrency int,
	log *logger.Logger,
) *OpportunityConsumer {
	if maxConcurrency <= 0 {
		maxConcurrency = 5 // Default: 5 concurrent analyses
	}

	return &OpportunityConsumer{
		consumer:        consumer,
		userRepo:        userRepo,
		tradingPairRepo: tradingPairRepo,
		agentFactory:    agentFactory,
		maxConcurrency:  maxConcurrency,
		log:             log,
	}
}

// Start begins consuming opportunity events
func (oc *OpportunityConsumer) Start(ctx context.Context) error {
	oc.log.Info("Starting opportunity consumer (event-driven trading)...")

	// Consume messages (ReadMessage blocks until message or ctx cancelled)
	for {
		msg, err := oc.consumer.ReadMessage(ctx)
		if err != nil {
			// Check if error is due to context cancellation
			if ctx.Err() != nil {
				oc.log.Info("Opportunity consumer stopping (context cancelled)")
				return nil
			}
			oc.log.Error("Failed to read opportunity event", "error", err)
			continue
		}

		// Process opportunity event
		if err := oc.handleOpportunity(ctx, msg); err != nil {
			oc.log.Error("Failed to handle opportunity",
				"topic", msg.Topic,
				"error", err,
			)
		}
	}
}

// handleOpportunity processes a single opportunity event
func (oc *OpportunityConsumer) handleOpportunity(ctx context.Context, msg kafkago.Message) error {
	oc.log.Debug("Processing opportunity event",
		"topic", msg.Topic,
		"size", len(msg.Value),
	)

	// Deserialize opportunity event
	var event eventspb.OpportunityFoundEvent
	if err := proto.Unmarshal(msg.Value, &event); err != nil {
		return errors.Wrap(err, "unmarshal opportunity_found event")
	}

	oc.log.Info("Opportunity detected",
		"symbol", event.Symbol,
		"direction", event.Direction,
		"confidence", event.Confidence,
		"strategy", event.Strategy,
		"entry", event.Entry,
	)

	// Find all users monitoring this symbol
	pairs, err := oc.tradingPairRepo.GetActiveBySymbol(ctx, event.Symbol)
	if err != nil {
		return errors.Wrap(err, "get trading pairs for symbol")
	}

	if len(pairs) == 0 {
		oc.log.Debug("No users monitoring this symbol", "symbol", event.Symbol)
		return nil
	}

	oc.log.Info("Found users interested in opportunity",
		"symbol", event.Symbol,
		"users_count", len(pairs),
	)

	// Run agent analysis for each interested user (concurrent with limit)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, oc.maxConcurrency)

	for _, pair := range pairs {
		// Get user
		usr, err := oc.userRepo.GetByID(ctx, pair.UserID)
		if err != nil {
			oc.log.Error("Failed to get user", "user_id", pair.UserID, "error", err)
			continue
		}

		// Skip inactive users
		if !usr.IsActive || !usr.Settings.CircuitBreakerOn {
			continue
		}

		wg.Add(1)
		go func(u *user.User, p *trading_pair.TradingPair) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := oc.analyzeOpportunity(ctx, u, p, &event); err != nil {
				oc.log.Error("Failed to analyze opportunity",
					"user_id", u.ID,
					"symbol", p.Symbol,
					"error", err,
				)
			}
		}(usr, pair)
	}

	wg.Wait()

	oc.log.Info("Opportunity processing complete",
		"symbol", event.Symbol,
		"users_processed", len(pairs),
	)

	return nil
}

// analyzeOpportunity runs agent analysis for a specific opportunity
func (oc *OpportunityConsumer) analyzeOpportunity(
	ctx context.Context,
	usr *user.User,
	pair *trading_pair.TradingPair,
	opportunity *eventspb.OpportunityFoundEvent,
) error {
	oc.log.Info("Analyzing opportunity for user",
		"user_id", usr.ID,
		"symbol", pair.Symbol,
		"opportunity_direction", opportunity.Direction,
		"opportunity_confidence", opportunity.Confidence,
	)

	// TODO: Implement agent pipeline
	// 1. Run quick validation (is user still interested in this symbol?)
	// 2. Run RiskManager agent (can we take this trade?)
	// 3. If approved → run Executor agent (place the trade)
	// 4. Log decision to journal

	// For now, just log
	oc.log.Debug("Would run event-driven analysis",
		"user_id", usr.ID,
		"symbol", pair.Symbol,
		"opportunity_strategy", opportunity.Strategy,
		"opportunity_entry", opportunity.Entry,
	)

	return nil
}
