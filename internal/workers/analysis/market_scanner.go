package analysis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"prometheus/internal/adapters/config"
	"prometheus/internal/adapters/kafka"
	"prometheus/internal/agents"
	"prometheus/internal/domain/trading_pair"
	"prometheus/internal/domain/user"
	"prometheus/internal/events"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// MarketScanner runs periodic agent analysis for all active users
// This is the SCHEDULED mode of the agentic trading system
// For event-driven analysis, see internal/consumers/opportunity_consumer.go
type MarketScanner struct {
	*workers.BaseWorker
	userRepo        user.Repository
	tradingPairRepo trading_pair.Repository
	agentFactory    *agents.Factory
	orchestrator    *agents.Orchestrator
	kafka           *kafka.Producer
	eventPublisher  *events.WorkerPublisher
	maxConcurrency  int    // Maximum number of users to process concurrently
	defaultProvider string // Default AI provider from config
	defaultModel    string // Default AI model from config
}

// NewMarketScanner creates a new market scanner worker
func NewMarketScanner(
	userRepo user.Repository,
	tradingPairRepo trading_pair.Repository,
	agentFactory *agents.Factory,
	kafka *kafka.Producer,
	runtimeConfig config.AgentsConfig,
	defaultProvider string,
	defaultModel string,
	interval time.Duration,
	maxConcurrency int,
	enabled bool,
) *MarketScanner {
	if maxConcurrency <= 0 {
		maxConcurrency = 5 // Default: process 5 users concurrently
	}

	// Create orchestrator for agent pipeline coordination
	costTracker := agents.NewCostTracker()
	orchestrator := agents.NewOrchestrator(agentFactory, runtimeConfig, costTracker)

	return &MarketScanner{
		BaseWorker:      workers.NewBaseWorker("market_scanner", interval, enabled),
		userRepo:        userRepo,
		tradingPairRepo: tradingPairRepo,
		agentFactory:    agentFactory,
		orchestrator:    orchestrator,
		kafka:           kafka,
		eventPublisher:  events.NewWorkerPublisher(kafka),
		maxConcurrency:  maxConcurrency,
		defaultProvider: defaultProvider,
		defaultModel:    defaultModel,
	}
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

	// Publish scan complete event using protobuf
	if err := ms.eventPublisher.PublishMarketScanComplete(
		ctx,
		len(users),
		len(scanErrors),
		duration.Milliseconds(),
	); err != nil {
		ms.Log().Error("Failed to publish scan complete event", "error", err)
	}

	return nil
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
	ms.Log().Info("Analyzing pair",
		"user_id", usr.ID,
		"symbol", pair.Symbol,
		"market_type", pair.MarketType,
	)

	// Get user's AI provider/model preferences, fallback to defaults
	provider := ms.defaultProvider
	model := ms.defaultModel

	// Check user settings for custom AI preferences
	if usr.Settings.DefaultAIProvider != "" {
		provider = usr.Settings.DefaultAIProvider
	}
	if usr.Settings.DefaultAIModel != "" {
		model = usr.Settings.DefaultAIModel
	}

	// Run full agent analysis pipeline via orchestrator
	// This executes:
	// 1. Parallel analysis agents (market, SMC, order flow, etc.)
	// 2. Strategy planner to synthesize results
	// 3. Risk manager to validate (TODO)
	plan, err := ms.orchestrator.RunAnalysisPipeline(
		ctx,
		usr.ID,
		pair.Symbol,
		pair.MarketType.String(),
		provider,
		model,
	)

	if err != nil {
		ms.Log().Error("Agent pipeline failed",
			"user_id", usr.ID,
			"symbol", pair.Symbol,
			"error", err,
		)
		return errors.Wrap(err, "agent pipeline execution failed")
	}

	ms.Log().Info("Agent pipeline complete",
		"user_id", usr.ID,
		"symbol", pair.Symbol,
		"analysis_count", len(plan.AnalysisInputs),
		"strategy_tokens", plan.StrategyOutput.TokensUsed,
		"strategy_cost", plan.StrategyOutput.CostUSD,
	)

	// TODO: Extract trade signal from plan and publish to Kafka
	// TODO: If auto-execution enabled, pass to executor agent
	// TODO: Store plan in database for audit

	// Publish analysis complete event
	if err := ms.eventPublisher.PublishMarketAnalysisRequest(
		ctx,
		usr.ID.String(),
		pair.Symbol,
		pair.MarketType.String(),
		pair.StrategyMode.String(),
	); err != nil {
		ms.Log().Error("Failed to publish analysis event", "error", err)
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

// Event structures removed - now using protobuf events via WorkerPublisher
