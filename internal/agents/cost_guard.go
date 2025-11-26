package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// CostGuard enforces hard limits on AI spending to prevent cost explosions
type CostGuard struct {
	maxDailyCostPerUser decimal.Decimal
	maxCostPerExecution decimal.Decimal
	cache               CostCache
	tracker             *CostTracker
	log                 *logger.Logger
}

// CostCache provides fast access to spending data (typically Redis)
type CostCache interface {
	GetDailySpending(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)
	IncrementSpending(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, ttl time.Duration) error
	GetExecutionSpending(ctx context.Context, executionID string) (decimal.Decimal, error)
	SetExecutionSpending(ctx context.Context, executionID string, amount decimal.Decimal, ttl time.Duration) error
}

// NewCostGuard creates a new cost guard with specified limits
func NewCostGuard(
	maxDailyCostPerUser, maxCostPerExecution decimal.Decimal,
	cache CostCache,
	tracker *CostTracker,
) *CostGuard {
	return &CostGuard{
		maxDailyCostPerUser: maxDailyCostPerUser,
		maxCostPerExecution: maxCostPerExecution,
		cache:               cache,
		tracker:             tracker,
		log:                 logger.Get().With("component", "cost_guard"),
	}
}

// CheckDailyLimit checks if user has exceeded daily spending limit
// Returns error if limit would be exceeded
func (cg *CostGuard) CheckDailyLimit(ctx context.Context, userID uuid.UUID) error {
	spending, err := cg.cache.GetDailySpending(ctx, userID)
	if err != nil {
		// If cache fails, be conservative and allow (but log error)
		cg.log.Errorf("Failed to get daily spending from cache: %v", err)
		return nil
	}

	if spending.GreaterThanOrEqual(cg.maxDailyCostPerUser) {
		cg.log.Warnf("User %s exceeded daily cost limit: $%s / $%s",
			userID, spending.StringFixed(2), cg.maxDailyCostPerUser.StringFixed(2))
		return errors.Wrapf(errors.ErrQuotaExceeded,
			"daily AI cost limit exceeded: $%s / $%s",
			spending.StringFixed(2), cg.maxDailyCostPerUser.StringFixed(2))
	}

	// Warn if approaching limit (80%)
	threshold := cg.maxDailyCostPerUser.Mul(decimal.NewFromFloat(0.80))
	if spending.GreaterThanOrEqual(threshold) {
		cg.log.Warnf("User %s approaching daily cost limit: $%s / $%s (80%% threshold)",
			userID, spending.StringFixed(2), cg.maxDailyCostPerUser.StringFixed(2))
	}

	return nil
}

// CheckExecutionLimit checks if a single execution would exceed per-execution limit
func (cg *CostGuard) CheckExecutionLimit(ctx context.Context, executionID string, estimatedCost decimal.Decimal) error {
	if estimatedCost.GreaterThan(cg.maxCostPerExecution) {
		cg.log.Warnf("Execution %s estimated cost exceeds limit: $%s / $%s",
			executionID, estimatedCost.StringFixed(2), cg.maxCostPerExecution.StringFixed(2))
		return errors.Wrapf(errors.ErrQuotaExceeded,
			"execution cost limit exceeded: $%s / $%s",
			estimatedCost.StringFixed(2), cg.maxCostPerExecution.StringFixed(2))
	}

	return nil
}

// RecordCost records actual cost after execution
// This updates both daily and execution spending
func (cg *CostGuard) RecordCost(ctx context.Context, userID uuid.UUID, executionID string, actualCost decimal.Decimal) error {
	// Update daily spending with 24h TTL
	if err := cg.cache.IncrementSpending(ctx, userID, actualCost, 24*time.Hour); err != nil {
		cg.log.Errorf("Failed to increment daily spending: %v", err)
		// Continue anyway - we don't want to fail the operation
	}

	// Store execution spending with 1h TTL
	if err := cg.cache.SetExecutionSpending(ctx, executionID, actualCost, 1*time.Hour); err != nil {
		cg.log.Errorf("Failed to set execution spending: %v", err)
	}

	// Also update cost tracker for analytics
	if cg.tracker != nil {
		cg.tracker.RecordCost(userID.String(), actualCost.InexactFloat64())
	}

	cg.log.Debugf("Recorded cost for user %s execution %s: $%s",
		userID, executionID, actualCost.StringFixed(4))

	return nil
}

// GetDailySpending returns current daily spending for a user
func (cg *CostGuard) GetDailySpending(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	return cg.cache.GetDailySpending(ctx, userID)
}

// GetRemainingDailyBudget returns how much budget user has left today
func (cg *CostGuard) GetRemainingDailyBudget(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	spending, err := cg.cache.GetDailySpending(ctx, userID)
	if err != nil {
		return decimal.Zero, err
	}

	remaining := cg.maxDailyCostPerUser.Sub(spending)
	if remaining.LessThan(decimal.Zero) {
		remaining = decimal.Zero
	}

	return remaining, nil
}

// EstimateCost estimates cost for an agent execution based on agent type and model
// This is a simple heuristic - actual costs may vary
func (cg *CostGuard) EstimateCost(agentType string, model string) decimal.Decimal {
	// Base estimates per agent type (rough averages)
	// These should be tuned based on actual usage patterns
	baseCosts := map[string]float64{
		"market_analyst":      0.15, // ~25 tool calls, 50k tokens
		"smc_analyst":         0.08, // ~15 tool calls, 25k tokens
		"sentiment_analyst":   0.05, // ~10 tool calls, 15k tokens
		"onchain_analyst":     0.05, // ~10 tool calls, 15k tokens
		"correlation_analyst": 0.04, // ~8 tool calls, 12k tokens
		"macro_analyst":       0.04, // ~8 tool calls, 12k tokens
		"orderflow_analyst":   0.06, // ~12 tool calls, 20k tokens
		"derivatives_analyst": 0.05, // ~10 tool calls, 15k tokens
		"strategy_planner":    0.08, // ~8 tool calls + thinking, 30k tokens
		"risk_manager":        0.03, // ~5 tool calls, 10k tokens
		"executor":            0.02, // ~3 tool calls, 5k tokens
		"position_manager":    0.03, // ~5 tool calls, 10k tokens
		"self_evaluator":      0.05, // ~10 tool calls, 15k tokens
	}

	baseCost, ok := baseCosts[agentType]
	if !ok {
		baseCost = 0.05 // Default estimate
	}

	// Model multipliers (based on actual pricing)
	modelMultipliers := map[string]float64{
		"claude-3-opus":   2.0,  // Most expensive
		"claude-3-sonnet": 1.0,  // Baseline
		"claude-3-haiku":  0.25, // Cheapest Claude
		"gpt-4":           1.5,  // Expensive
		"gpt-4-turbo":     1.2,
		"gpt-3.5-turbo":   0.1,  // Very cheap
		"deepseek-chat":   0.05, // Extremely cheap
		"gemini-pro":      0.5,  // Mid-range
	}

	multiplier, ok := modelMultipliers[model]
	if !ok {
		multiplier = 1.0 // Default
	}

	estimated := decimal.NewFromFloat(baseCost * multiplier)
	return estimated
}

// RedisCostCache implements CostCache using Redis
type RedisCostCache struct {
	redis RedisClient
}

// RedisClient interface for Redis operations (minimal interface for cost tracking)
type RedisClient interface {
	GetString(ctx context.Context, key string) (string, error)
	SetString(ctx context.Context, key string, value string, ttl time.Duration) error
	IncrByFloat(ctx context.Context, key string, value float64) (float64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

// NewRedisCostCache creates a Redis-backed cost cache
func NewRedisCostCache(redis RedisClient) *RedisCostCache {
	return &RedisCostCache{redis: redis}
}

// GetDailySpending retrieves daily spending from Redis
func (rc *RedisCostCache) GetDailySpending(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	key := fmt.Sprintf("cost:daily:%s", userID.String())
	val, err := rc.redis.GetString(ctx, key)
	if err != nil {
		// Key doesn't exist = no spending yet (Redis returns error on missing key)
		return decimal.Zero, nil
	}

	spending, err := decimal.NewFromString(val)
	if err != nil {
		return decimal.Zero, errors.Wrapf(err, "invalid spending value")
	}

	return spending, nil
}

// IncrementSpending adds to daily spending using atomic increment
func (rc *RedisCostCache) IncrementSpending(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, ttl time.Duration) error {
	key := fmt.Sprintf("cost:daily:%s", userID.String())

	// Use atomic float increment
	_, err := rc.redis.IncrByFloat(ctx, key, amount.InexactFloat64())
	if err != nil {
		return errors.Wrapf(err, "failed to increment spending")
	}

	// Set TTL (only if key was just created)
	if err := rc.redis.Expire(ctx, key, ttl); err != nil {
		return errors.Wrapf(err, "failed to set TTL")
	}

	return nil
}

// GetExecutionSpending retrieves execution spending
func (rc *RedisCostCache) GetExecutionSpending(ctx context.Context, executionID string) (decimal.Decimal, error) {
	key := fmt.Sprintf("cost:exec:%s", executionID)
	val, err := rc.redis.GetString(ctx, key)
	if err != nil {
		return decimal.Zero, nil
	}

	spending, err := decimal.NewFromString(val)
	if err != nil {
		return decimal.Zero, errors.Wrapf(err, "invalid execution spending")
	}

	return spending, nil
}

// SetExecutionSpending sets execution spending
func (rc *RedisCostCache) SetExecutionSpending(ctx context.Context, executionID string, amount decimal.Decimal, ttl time.Duration) error {
	key := fmt.Sprintf("cost:exec:%s", executionID)
	return rc.redis.SetString(ctx, key, amount.String(), ttl)
}
