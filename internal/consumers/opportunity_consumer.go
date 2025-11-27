package consumers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
	"google.golang.org/protobuf/proto"

	kafkaadapter "prometheus/internal/adapters/kafka"
	"prometheus/internal/agents/workflows"
	"prometheus/internal/domain/trading_pair"
	"prometheus/internal/domain/user"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// OpportunityConsumer handles market opportunity events with personal trading workflows
// This is the event-driven trading engine: receives global opportunity signals and
// runs personalized trading decisions for each interested user
type OpportunityConsumer struct {
	consumer        *kafkaadapter.Consumer
	userRepo        user.Repository
	tradingPairRepo trading_pair.Repository
	workflowFactory *workflows.Factory
	sessionService  session.Service
	maxConcurrency  int
	log             *logger.Logger
}

// NewOpportunityConsumer creates a new opportunity event consumer
func NewOpportunityConsumer(
	consumer *kafkaadapter.Consumer,
	userRepo user.Repository,
	tradingPairRepo trading_pair.Repository,
	workflowFactory *workflows.Factory,
	sessionService session.Service,
	maxConcurrency int,
	log *logger.Logger,
) *OpportunityConsumer {
	if maxConcurrency <= 0 {
		maxConcurrency = 5 // Default: 5 concurrent user workflows
	}

	return &OpportunityConsumer{
		consumer:        consumer,
		userRepo:        userRepo,
		tradingPairRepo: tradingPairRepo,
		workflowFactory: workflowFactory,
		sessionService:  sessionService,
		maxConcurrency:  maxConcurrency,
		log:             log,
	}
}

// Start begins consuming opportunity events
func (oc *OpportunityConsumer) Start(ctx context.Context) error {
	oc.log.Info("Starting opportunity consumer (event-driven trading)...")

	// Ensure consumer is closed on exit
	defer func() {
		oc.log.Info("Closing opportunity consumer...")
		if err := oc.consumer.Close(); err != nil {
			oc.log.Error("Failed to close opportunity consumer", "error", err)
		} else {
			oc.log.Info("✓ Opportunity consumer closed")
		}
	}()

	// Consume messages (ReadMessage blocks until message or ctx cancelled)
	for {
		msg, err := oc.consumer.ReadMessage(ctx)
		if err != nil {
			// Check if error is due to context cancellation or reader closure
			if ctx.Err() != nil {
				oc.log.Info("Opportunity consumer stopping (context cancelled)")
				return nil
			}
			// Reader might be closed during shutdown, log at debug level
			oc.log.Debug("Failed to read opportunity event", "error", err)
			continue
		}

		// Pass ctx directly to handleOpportunity. It will create its own workflowCtx
		// for running workflows, but will use ctx for shutdown detection to:
		// 1. Skip starting new workflows during shutdown
		// 2. Cancel running workflows early
		// 3. Wait with timeout for workflow completion
		if err := oc.handleOpportunity(ctx, msg); err != nil {
			oc.log.Error("Failed to handle opportunity",
				"topic", msg.Topic,
				"error", err,
			)
		}

		// Check if we should stop AFTER processing current message
		if ctx.Err() != nil {
			oc.log.Info("Opportunity consumer stopping after processing current message")
			return nil
		}
	}
}

// handleOpportunity processes a single opportunity event
func (oc *OpportunityConsumer) handleOpportunity(ctx context.Context, msg kafka.Message) error {
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

	// Run personal trading workflow for each interested user (concurrent with limit)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, oc.maxConcurrency)

	// Create workflow context with timeout to prevent infinite workflows.
	// During normal operation: workflows have up to 45 seconds to complete.
	// During shutdown: workflows are cancelled immediately via ctx cancellation.
	workflowCtx, workflowCancel := context.WithTimeout(ctx, 45*time.Second)
	defer workflowCancel()

	// Channel to signal early termination during shutdown
	shutdownCh := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(shutdownCh)
		// Cancel all workflows immediately on shutdown
		workflowCancel()
	}()

	for _, pair := range pairs {
		// Check if shutdown was requested before starting new workflow
		select {
		case <-shutdownCh:
			oc.log.Warn("Shutdown requested, skipping remaining workflows",
				"symbol", event.Symbol,
				"remaining_users", len(pairs),
			)
			// Don't start new workflows during shutdown
			goto waitForCompletion
		default:
		}

		// Get user
		usr, err := oc.userRepo.GetByID(ctx, pair.UserID)
		if err != nil {
			oc.log.Error("Failed to get user", "user_id", pair.UserID, "error", err)
			continue
		}

		// Skip inactive users or users with circuit breaker off
		if !usr.IsActive || !usr.Settings.CircuitBreakerOn {
			continue
		}

		wg.Add(1)
		go func(u *user.User, p *trading_pair.TradingPair) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-workflowCtx.Done():
				oc.log.Warn("Workflow cancelled before starting (context done)",
					"user_id", u.ID,
					"symbol", p.Symbol,
				)
				return
			}

			if err := oc.runPersonalTradingWorkflow(workflowCtx, u, p, &event); err != nil {
				oc.log.Error("Personal trading workflow failed",
					"user_id", u.ID,
					"symbol", p.Symbol,
					"error", err,
				)
			}
		}(usr, pair)
	}

waitForCompletion:
	// Wait for all started workflows to complete with timeout.
	// During shutdown, workflows are cancelled via workflowCtx, so this should be fast.
	// Timeout: 60s (45s workflow timeout + 15s buffer)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	waitTimeout := 60 * time.Second
	// During shutdown, use shorter timeout since workflows should already be cancelled
	if ctx.Err() != nil {
		waitTimeout = 15 * time.Second
	}

	select {
	case <-done:
		oc.log.Info("Opportunity processing complete",
			"symbol", event.Symbol,
			"users_processed", len(pairs),
		)
	case <-time.After(waitTimeout):
		oc.log.Warn("Timeout waiting for workflows to complete",
			"symbol", event.Symbol,
			"timeout", waitTimeout,
			"during_shutdown", ctx.Err() != nil,
		)
		// Force cancel remaining workflows
		workflowCancel()

		// Give goroutines additional 5s to cleanup after force cancel
		select {
		case <-done:
			oc.log.Info("Workflows completed after force cancel")
		case <-time.After(5 * time.Second):
			oc.log.Error("Some workflows still running after force cancel - possible goroutine leak",
				"symbol", event.Symbol,
			)
		}
	}

	return nil
}

// runPersonalTradingWorkflow runs the trading workflow for a specific user
func (oc *OpportunityConsumer) runPersonalTradingWorkflow(
	ctx context.Context,
	usr *user.User,
	pair *trading_pair.TradingPair,
	opportunity *eventspb.OpportunityFoundEvent,
) error {
	oc.log.Info("Running personal trading workflow",
		"user_id", usr.ID,
		"symbol", pair.Symbol,
		"opportunity_confidence", opportunity.Confidence,
	)

	// Create personal trading workflow
	workflow, err := oc.workflowFactory.CreatePersonalTradingWorkflow()
	if err != nil {
		return errors.Wrap(err, "failed to create personal trading workflow")
	}

	// Create ADK runner
	runnerInstance, err := runner.New(runner.Config{
		AppName:        fmt.Sprintf("prometheus_trading_%s", usr.ID),
		Agent:          workflow,
		SessionService: oc.sessionService,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create runner")
	}

	// Build input prompt with PRE-ANALYZED signal + user context
	input := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: oc.buildTradingPrompt(usr, pair, opportunity)},
		},
	}

	// Run workflow
	sessionID := uuid.New().String()
	userID := usr.ID.String()

	runConfig := agent.RunConfig{
		StreamingMode: agent.StreamingModeNone, // No streaming for background jobs
	}

	// Track execution
	startTime := time.Now()
	orderPlaced := false

	// Execute workflow and track events
	for event, err := range runnerInstance.Run(ctx, userID, sessionID, input, runConfig) {
		if err != nil {
			return errors.Wrap(err, "workflow execution failed")
		}

		if event == nil || event.LLMResponse.Partial {
			continue
		}

		// Check for order placement
		if event.LLMResponse.Content != nil {
			for _, part := range event.LLMResponse.Content.Parts {
				if part.FunctionCall != nil && part.FunctionCall.Name == "place_order" {
					orderPlaced = true
					oc.log.Info("Order placed by executor",
						"user_id", usr.ID,
						"symbol", pair.Symbol,
					)
				}
			}
		}

		// Check if workflow is complete
		if event.TurnComplete && event.IsFinalResponse() {
			duration := time.Since(startTime)
			oc.log.Info("Personal trading workflow complete",
				"user_id", usr.ID,
				"symbol", pair.Symbol,
				"session_id", sessionID,
				"order_placed", orderPlaced,
				"duration", duration,
			)
			break
		}
	}

	return nil
}

// buildTradingPrompt creates the input prompt for personal trading workflow
func (oc *OpportunityConsumer) buildTradingPrompt(
	usr *user.User,
	pair *trading_pair.TradingPair,
	opp *eventspb.OpportunityFoundEvent,
) string {
	return fmt.Sprintf(`# Trading Opportunity Signal

## Pre-Analyzed Market Signal (Global Research)

**Symbol**: %s
**Exchange**: %s
**Direction**: %s
**Confidence**: %.2f
**Strategy**: %s

**Entry Price**: %.2f
**Stop Loss**: %.2f
**Take Profit**: %.2f
**Timeframe**: %s

**Analysis Reasoning**:
%s

---

## Your Personal Context

**User ID**: %s
**Risk Tolerance**: %s
**Trading Pair**: %s

**Your Task**:

You are executing a personal trading workflow. The market research has ALREADY been completed by 8 specialist analysts. Your job is to make a personalized trading decision based on:

1. **The PRE-ANALYZED market signal above** (objective market view)
2. **Your personal context** (portfolio, risk profile, existing positions)

### Step 1: StrategyPlanner
- Use tools to get YOUR context:
  - get_portfolio_summary() - understand current portfolio
  - get_positions() - check existing positions
  - get_user_risk_profile() - understand risk limits
- Decide: Should YOU take this trade? If yes, with what size?
- Output: Personal trading plan

### Step 2: RiskManager
- Validate the plan against YOUR risk limits
- Check: daily loss limits, position size limits, correlation
- Output: Approved/rejected + adjusted plan

### Step 3: Executor
- If approved → place order using place_order() tool
- If rejected → log reason and skip

**Think step-by-step. Be thorough but decisive.**

Your decision should be personalized to YOUR portfolio and risk profile, not just the global signal.`,
		opp.Symbol,
		opp.Exchange,
		opp.Direction,
		opp.Confidence,
		opp.Strategy,
		opp.Entry,
		opp.StopLoss,
		opp.TakeProfit,
		opp.Timeframe,
		opp.Reasoning,
		usr.ID,
		usr.Settings.RiskLevel,
		pair.Symbol,
	)
}
