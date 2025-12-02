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
	strategyDomain "prometheus/internal/domain/strategy"
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
	strategyRepo    strategyDomain.Repository
	userService     *user.Service
	workflowFactory *workflows.Factory
	sessionService  session.Service
	maxConcurrency  int
	log             *logger.Logger
}

// NewOpportunityConsumer creates a new opportunity event consumer
func NewOpportunityConsumer(
	consumer *kafkaadapter.Consumer,
	strategyRepo strategyDomain.Repository,
	userService *user.Service,
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
		strategyRepo:    strategyRepo,
		userService:     userService,
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
		msg, err := oc.consumer.ReadMessageWithShutdownCheck(ctx)
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
			oc.log.Errorw("Failed to handle opportunity",
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
	oc.log.Debugw("Processing opportunity event",
		"topic", msg.Topic,
		"size", len(msg.Value),
	)

	// Deserialize opportunity event
	var event eventspb.OpportunityFoundEvent
	if err := proto.Unmarshal(msg.Value, &event); err != nil {
		return errors.Wrap(err, "unmarshal opportunity_found event")
	}

	oc.log.Infow("Opportunity detected",
		"symbol", event.Symbol,
		"direction", event.Direction,
		"confidence", event.Confidence,
		"strategy", event.Strategy,
		"entry", event.Entry,
	)

	// Get all active strategies and filter by target_allocations
	// New architecture: users don't have trading_pairs, they have target_allocations in Strategy
	strategies, err := oc.strategyRepo.GetAllActive(ctx)
	if err != nil {
		return errors.Wrap(err, "get active strategies")
	}

	// Filter strategies that have this symbol in target_allocations
	var interestedStrategies []*strategyDomain.Strategy
	for _, strat := range strategies {
		allocations, err := strat.ParseTargetAllocations()
		if err != nil {
			oc.log.Warnw("Failed to parse target allocations",
				"strategy_id", strat.ID,
				"error", err,
			)
			continue
		}

		// Check if symbol is in target allocations
		if _, hasSymbol := allocations[event.Symbol]; hasSymbol {
			interestedStrategies = append(interestedStrategies, strat)
		}
	}

	if len(interestedStrategies) == 0 {
		oc.log.Debugw("No strategies interested in this symbol", "symbol", event.Symbol)
		return nil
	}

	oc.log.Infow("Found strategies interested in opportunity",
		"symbol", event.Symbol,
		"strategies_count", len(interestedStrategies),
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

	for _, strategy := range interestedStrategies {
		// Check if shutdown was requested before starting new workflow
		select {
		case <-shutdownCh:
			oc.log.Warnw("Shutdown requested, skipping remaining workflows",
				"symbol", event.Symbol,
				"remaining_strategies", len(interestedStrategies),
			)
			// Don't start new workflows during shutdown
			goto waitForCompletion
		default:
		}

		// Get user using UserService (Clean Architecture)
		usr, err := oc.userService.GetByID(ctx, strategy.UserID)
		if err != nil {
			oc.log.Errorw("Failed to get user", "user_id", strategy.UserID, "error", err)
			continue
		}

		// Skip inactive users or users with circuit breaker off
		if !usr.IsActive || !usr.Settings.CircuitBreakerOn {
			oc.log.Debugw("Skipping user (inactive or circuit breaker off)",
				"user_id", usr.ID,
				"is_active", usr.IsActive,
				"circuit_breaker_on", usr.Settings.CircuitBreakerOn,
			)
			continue
		}

		wg.Add(1)
		go func(u *user.User, strat *strategyDomain.Strategy) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-workflowCtx.Done():
				oc.log.Warnw("Workflow cancelled before starting (context done)",
					"user_id", u.ID,
					"symbol", event.Symbol,
				)
				return
			}

			if err := oc.runPersonalTradingWorkflow(workflowCtx, u, strat, &event); err != nil {
				oc.log.Errorw("Personal trading workflow failed",
					"user_id", u.ID,
					"symbol", event.Symbol,
					"error", err,
				)
			}
		}(usr, strategy)
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
		oc.log.Infow("Opportunity processing complete",
			"symbol", event.Symbol,
			"users_processed", len(interestedStrategies),
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
	strategy *strategyDomain.Strategy,
	opportunity *eventspb.OpportunityFoundEvent,
) error {
	oc.log.Infow("Running personal trading workflow",
		"user_id", usr.ID,
		"strategy_id", strategy.ID,
		"symbol", opportunity.Symbol,
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
			{Text: oc.buildTradingPrompt(usr, strategy, opportunity)},
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
					oc.log.Infow("Order placed by portfolio manager",
						"user_id", usr.ID,
						"symbol", opportunity.Symbol,
					)
				}
			}
		}

		// Check if workflow is complete
		if event.TurnComplete && event.IsFinalResponse() {
			duration := time.Since(startTime)
			oc.log.Infow("Personal trading workflow complete",
				"user_id", usr.ID,
				"symbol", opportunity.Symbol,
				"session_id", sessionID,
				"order_placed", orderPlaced,
				"duration", duration,
			)
			break
		}
	}

	return nil
}

// buildTradingPrompt creates the input prompt for personal trading workflow using templates
func (oc *OpportunityConsumer) buildTradingPrompt(
	usr *user.User,
	strategy *strategyDomain.Strategy,
	opp *eventspb.OpportunityFoundEvent,
) string {
	// Get target allocation for this symbol
	allocations, _ := strategy.ParseTargetAllocations()
	targetAlloc := allocations[opp.Symbol] * 100 // Convert to percentage

	cashReserve, _ := strategy.CashReserve.Float64()

	// Prepare template data
	data := map[string]interface{}{
		// Opportunity data
		"Symbol":       opp.Symbol,
		"Exchange":     opp.Exchange,
		"Direction":    opp.Direction,
		"Confidence":   opp.Confidence * 100, // As percentage
		"StrategyType": opp.Strategy,
		"EntryPrice":   opp.Entry,
		"StopLoss":     opp.StopLoss,
		"TakeProfit":   opp.TakeProfit,
		"Timeframe":    opp.Timeframe,
		"Reasoning":    opp.Reasoning,
		// User/Strategy context
		"UserID":           usr.ID.String(),
		"StrategyID":       strategy.ID.String(),
		"StrategyName":     strategy.Name,
		"RiskTolerance":    string(strategy.RiskTolerance),
		"TargetAllocation": targetAlloc,
		"AvailableCash":    cashReserve,
	}

	// Render template with CoT framework
	// TODO: Add templates.Get() dependency to consumer
	// For now, use simple approach
	_ = data

	return fmt.Sprintf(`TODO: Use template workflows/personal_trading_input.tmpl
Symbol: %s, Strategy: %s, Target: %.1f%%`,
		opp.Symbol,
		strategy.Name,
		targetAlloc,
	)
}
