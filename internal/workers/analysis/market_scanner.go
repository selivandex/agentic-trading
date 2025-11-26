package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/agents"
	"prometheus/internal/domain/trading_pair"
	"prometheus/internal/domain/user"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// MarketScanner runs agent analysis for all active users
// This is the core worker that drives the agentic trading system
// It operates in two modes:
// 1. Scheduled: periodic full scan of all users (every 2 minutes)
// 2. Event-driven: immediate analysis on opportunity events (< 30s response time)
type MarketScanner struct {
	*workers.BaseWorker
	userRepo        user.Repository
	tradingPairRepo trading_pair.Repository
	agentFactory    *agents.Factory
	kafka           *kafka.Producer
	kafkaConsumer   *kafka.Consumer
	maxConcurrency  int                   // Maximum number of users to process concurrently
	opportunityChan chan OpportunityEvent // Channel for urgent opportunity events
	eventDriven     bool                  // Enable event-driven mode
}

// NewMarketScanner creates a new market scanner worker
func NewMarketScanner(
	userRepo user.Repository,
	tradingPairRepo trading_pair.Repository,
	agentFactory *agents.Factory,
	kafka *kafka.Producer,
	kafkaConsumer *kafka.Consumer,
	interval time.Duration,
	maxConcurrency int,
	eventDriven bool,
	enabled bool,
) *MarketScanner {
	if maxConcurrency <= 0 {
		maxConcurrency = 5 // Default: process 5 users concurrently
	}

	return &MarketScanner{
		BaseWorker:      workers.NewBaseWorker("market_scanner", interval, enabled),
		userRepo:        userRepo,
		tradingPairRepo: tradingPairRepo,
		agentFactory:    agentFactory,
		kafka:           kafka,
		kafkaConsumer:   kafkaConsumer,
		maxConcurrency:  maxConcurrency,
		opportunityChan: make(chan OpportunityEvent, 100), // Buffer for 100 events
		eventDriven:     eventDriven,
	}
}

// Start initializes the worker and starts event listener if enabled
func (ms *MarketScanner) Start(ctx context.Context) error {
	// Start event listener for opportunity events if event-driven mode is enabled
	if ms.eventDriven && ms.kafkaConsumer != nil {
		go ms.listenForOpportunities(ctx)
		ms.Log().Info("Market scanner: event-driven mode enabled, listening for opportunities")
	}

	return nil
}

// Run executes one iteration of market scanning for ALL active users
// This is the SCHEDULED mode - full scan of all users
// 1. Get all active users
// 2. For each user, get their active trading pairs
// 3. Run agent analysis pipeline for each user
// 4. Publish trading signals/decisions to Kafka
func (ms *MarketScanner) Run(ctx context.Context) error {
	start := time.Now()
	ms.Log().Info("Market scanner: starting scheduled scan")

	// Get all active users with trading enabled
	// Using a large limit to get all users (in production, implement proper pagination)
	users, err := ms.userRepo.List(ctx, 1000, 0)
	if err != nil {
		return errors.Wrap(err, "failed to list users")
	}

	// Filter only active users
	var activeUsers []*user.User
	for _, usr := range users {
		if usr.IsActive {
			activeUsers = append(activeUsers, usr)
		}
	}
	users = activeUsers

	if len(users) == 0 {
		ms.Log().Debug("No active users to scan")
		return nil
	}

	ms.Log().Info("Market scanner: processing users",
		"total_users", len(users),
		"max_concurrency", ms.maxConcurrency,
	)

	// Process users concurrently with a limit
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, ms.maxConcurrency)
	errorsCh := make(chan error, len(users))

	for _, usr := range users {
		wg.Add(1)
		go func(u *user.User) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := ms.scanUser(ctx, u); err != nil {
				errorsCh <- errors.Wrapf(err, "failed to scan user %s", u.ID)
			}
		}(usr)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errorsCh)

	// Collect errors
	var scanErrors []error
	for err := range errorsCh {
		scanErrors = append(scanErrors, err)
		ms.Log().Error("User scan error", "error", err)
	}

	duration := time.Since(start)
	ms.Log().Info("Market scanner: scheduled scan complete",
		"total_users", len(users),
		"errors", len(scanErrors),
		"duration", duration,
	)

	// Publish scan complete event
	ms.publishScanCompleteEvent(ctx, len(users), len(scanErrors), duration)

	return nil
}

// listenForOpportunities listens for opportunity events from Kafka
// This is the EVENT-DRIVEN mode - immediate response to market opportunities
func (ms *MarketScanner) listenForOpportunities(ctx context.Context) {
	ms.Log().Info("Starting opportunity event listener")

	// Consume messages from opportunity topic
	go func() {
		err := ms.kafkaConsumer.Consume(ctx, func(ctx context.Context, msg kafkago.Message) error {
			var event OpportunityEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				ms.Log().Error("Failed to unmarshal opportunity event", "error", err)
				return err
			}

			ms.Log().Info("Received opportunity event",
				"symbol", event.Symbol,
				"type", event.Type,
				"direction", event.Direction,
				"price_change", event.PriceChange,
			)

			// Send to processing channel (non-blocking)
			select {
			case ms.opportunityChan <- event:
				// Event queued successfully
			default:
				ms.Log().Warn("Opportunity channel full, dropping event", "symbol", event.Symbol)
			}

			return nil
		})

		if err != nil {
			ms.Log().Error("Kafka consumer stopped", "error", err)
		}
	}()

	// Process opportunities from channel
	for {
		select {
		case <-ctx.Done():
			ms.Log().Info("Opportunity listener shutting down")
			return
		case event := <-ms.opportunityChan:
			// Process opportunity immediately (event-driven analysis)
			ms.handleOpportunity(ctx, event)
		}
	}
}

// handleOpportunity processes a single opportunity event
// Triggers immediate agent analysis for users monitoring this symbol
func (ms *MarketScanner) handleOpportunity(ctx context.Context, event OpportunityEvent) {
	start := time.Now()
	ms.Log().Info("Processing opportunity",
		"symbol", event.Symbol,
		"type", event.Type,
		"direction", event.Direction,
	)

	// Get all users with active trading pairs for this symbol
	pairs, err := ms.tradingPairRepo.GetActiveBySymbol(ctx, event.Symbol)
	if err != nil {
		ms.Log().Error("Failed to get trading pairs for symbol",
			"symbol", event.Symbol,
			"error", err,
		)
		return
	}

	if len(pairs) == 0 {
		ms.Log().Debug("No users monitoring this symbol", "symbol", event.Symbol)
		return
	}

	ms.Log().Info("Found users monitoring symbol",
		"symbol", event.Symbol,
		"users_count", len(pairs),
	)

	// Run agent analysis for each user interested in this symbol
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, ms.maxConcurrency)

	for _, pair := range pairs {
		// Get user for this trading pair
		usr, err := ms.userRepo.GetByID(ctx, pair.UserID)
		if err != nil {
			ms.Log().Error("Failed to get user", "user_id", pair.UserID, "error", err)
			continue
		}

		if !usr.IsActive || !usr.Settings.CircuitBreakerOn {
			continue
		}

		wg.Add(1)
		go func(u *user.User, p *trading_pair.TradingPair) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := ms.analyzeUserPair(ctx, u, p); err != nil {
				ms.Log().Error("Failed to analyze pair for opportunity",
					"user_id", u.ID,
					"symbol", p.Symbol,
					"error", err,
				)
			}
		}(usr, pair)
	}

	wg.Wait()

	duration := time.Since(start)
	ms.Log().Info("Opportunity processing complete",
		"symbol", event.Symbol,
		"users_processed", len(pairs),
		"duration", duration,
	)
}

// scanUser runs agent analysis for a single user
func (ms *MarketScanner) scanUser(ctx context.Context, usr *user.User) error {
	userStart := time.Now()
	ms.Log().Debug("Scanning user", "user_id", usr.ID)

	// Check if user has trading enabled (user is active and circuit breaker is on)
	if !usr.IsActive || !usr.Settings.CircuitBreakerOn {
		ms.Log().Debug("User has trading disabled, skipping", "user_id", usr.ID)
		return nil
	}

	// Get user's active trading pairs
	pairs, err := ms.tradingPairRepo.GetActiveByUser(ctx, usr.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get active trading pairs")
	}

	if len(pairs) == 0 {
		ms.Log().Debug("User has no active trading pairs", "user_id", usr.ID)
		return nil
	}

	ms.Log().Debug("User has active trading pairs",
		"user_id", usr.ID,
		"pairs_count", len(pairs),
	)

	// For each trading pair, run agent analysis
	var analysisErrors []error
	for _, pair := range pairs {
		if err := ms.analyzeUserPair(ctx, usr, pair); err != nil {
			analysisErrors = append(analysisErrors, err)
			ms.Log().Error("Failed to analyze pair",
				"user_id", usr.ID,
				"symbol", pair.Symbol,
				"error", err,
			)
			// Continue with other pairs even if one fails
		}
	}

	duration := time.Since(userStart)
	ms.Log().Info("User scan complete",
		"user_id", usr.ID,
		"pairs_analyzed", len(pairs),
		"errors", len(analysisErrors),
		"duration", duration,
	)

	return nil
}

// analyzeUserPair runs the full agent pipeline for a user's trading pair
// Pipeline: Analysis Agents → Strategy Planner → Risk Manager → (Executor if approved)
func (ms *MarketScanner) analyzeUserPair(ctx context.Context, usr *user.User, pair *trading_pair.TradingPair) error {
	ms.Log().Debug("Analyzing pair",
		"user_id", usr.ID,
		"symbol", pair.Symbol,
		"market_type", pair.MarketType,
	)

	// TODO: Implement full agent pipeline
	// This is where the magic happens!
	//
	// 1. Run analysis agents in parallel:
	//    - MarketAnalyst: Technical analysis
	//    - SMCAnalyst: Smart Money Concepts
	//    - SentimentAnalyst: News & social sentiment
	//    - OnChainAnalyst: Blockchain metrics
	//    - CorrelationAnalyst: Cross-market analysis
	//    - MacroAnalyst: Economic events
	//    - OrderFlowAnalyst: Tape reading
	//    - DerivativesAnalyst: Options flow
	//
	// 2. Collect all analysis results
	//
	// 3. Run StrategyPlanner to synthesize analysis into trade plan
	//
	// 4. Run RiskManager to validate and size positions
	//
	// 5. If approved by risk manager:
	//    - Publish signal to Kafka
	//    - Optionally: Auto-execute via Executor agent
	//
	// 6. Log all reasoning steps to database (Chain-of-Thought)

	// For now, just log that we would analyze
	ms.Log().Debug("Would run agent analysis pipeline",
		"user_id", usr.ID,
		"symbol", pair.Symbol,
		"strategy", pair.StrategyMode,
	)

	// Publish analysis request event (for future implementation)
	event := MarketAnalysisRequestEvent{
		UserID:     usr.ID.String(),
		Symbol:     pair.Symbol,
		MarketType: pair.MarketType.String(),
		Strategy:   pair.StrategyMode.String(),
		Timestamp:  time.Now(),
	}

	if err := ms.kafka.Publish(ctx, "market.analysis_requested", usr.ID.String(), event); err != nil {
		ms.Log().Error("Failed to publish analysis request event", "error", err)
	}

	return nil
}

// runAnalysisAgents runs all analysis agents in parallel and collects results
func (ms *MarketScanner) runAnalysisAgents(
	ctx context.Context,
	usr *user.User,
	pair *trading_pair.TradingPair,
) (map[agents.AgentType]interface{}, error) {
	ms.Log().Debug("Running analysis agents",
		"user_id", usr.ID,
		"symbol", pair.Symbol,
	)

	// Create a context with timeout for agent execution
	agentCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// List of analysis agents to run
	analysisAgents := []agents.AgentType{
		agents.AgentMarketAnalyst,
		agents.AgentSMCAnalyst,
		agents.AgentSentimentAnalyst,
		agents.AgentOnChainAnalyst,
		agents.AgentCorrelationAnalyst,
		agents.AgentMacroAnalyst,
		agents.AgentOrderFlowAnalyst,
		agents.AgentDerivativesAnalyst,
	}

	// Run agents in parallel
	results := make(map[agents.AgentType]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup
	errorsCh := make(chan error, len(analysisAgents))

	for _, agentType := range analysisAgents {
		wg.Add(1)
		go func(at agents.AgentType) {
			defer wg.Done()

			result, err := ms.runAgent(agentCtx, at, usr, pair)
			if err != nil {
				errorsCh <- errors.Wrapf(err, "agent %s failed", at)
				return
			}

			mu.Lock()
			results[at] = result
			mu.Unlock()
		}(agentType)
	}

	wg.Wait()
	close(errorsCh)

	// Collect errors
	var agentErrors []error
	for err := range errorsCh {
		agentErrors = append(agentErrors, err)
	}

	if len(agentErrors) > 0 {
		return results, fmt.Errorf("some agents failed: %v", agentErrors)
	}

	return results, nil
}

// runAgent runs a single agent and returns its result
func (ms *MarketScanner) runAgent(
	ctx context.Context,
	agentType agents.AgentType,
	usr *user.User,
	pair *trading_pair.TradingPair,
) (interface{}, error) {
	ms.Log().Debug("Running agent",
		"agent", agentType,
		"user_id", usr.ID,
		"symbol", pair.Symbol,
	)

	// TODO: Create agent instance and run it
	// agent, err := ms.agentFactory.CreateAgent(agentType, usr.ID, pair.Symbol)
	// if err != nil {
	//     return nil, errors.Wrap(err, "failed to create agent")
	// }
	//
	// result, err := agent.Run(ctx)
	// if err != nil {
	//     return nil, errors.Wrap(err, "agent execution failed")
	// }
	//
	// return result, nil

	// For now, return placeholder
	return map[string]string{
		"agent":  string(agentType),
		"status": "not_implemented",
	}, nil
}

// Event structures

type MarketAnalysisRequestEvent struct {
	UserID     string    `json:"user_id"`
	Symbol     string    `json:"symbol"`
	MarketType string    `json:"market_type"` // spot, futures
	Strategy   string    `json:"strategy"`
	Timestamp  time.Time `json:"timestamp"`
}

type MarketScanCompleteEvent struct {
	TotalUsers     int           `json:"total_users"`
	ErrorsCount    int           `json:"errors_count"`
	Duration       time.Duration `json:"duration"`
	DurationMs     int64         `json:"duration_ms"`
	ScanStartTime  time.Time     `json:"scan_start_time"`
	ScanFinishTime time.Time     `json:"scan_finish_time"`
}

func (ms *MarketScanner) publishScanCompleteEvent(ctx context.Context, totalUsers, errorsCount int, duration time.Duration) {
	event := MarketScanCompleteEvent{
		TotalUsers:     totalUsers,
		ErrorsCount:    errorsCount,
		Duration:       duration,
		DurationMs:     duration.Milliseconds(),
		ScanStartTime:  time.Now().Add(-duration),
		ScanFinishTime: time.Now(),
	}

	if err := ms.kafka.Publish(ctx, "market.scan_complete", "global", event); err != nil {
		ms.Log().Error("Failed to publish scan complete event", "error", err)
	}
}
