package agents

import (
	"sync"
	"time"

	"prometheus/internal/adapters/ai"
)

// CostTracker tracks AI model usage costs
type CostTracker struct {
	mu    sync.RWMutex
	costs map[string]*ModelCost // model ID -> cost data
}

// ModelCost tracks cost for a specific model
type ModelCost struct {
	ModelID      string
	InputTokens  int64
	OutputTokens int64
	TotalCostUSD float64
	CallCount    int64
}

// NewCostTracker creates a new cost tracker
func NewCostTracker() *CostTracker {
	return &CostTracker{
		costs: make(map[string]*ModelCost),
	}
}

// RecordUsage records token usage for a model
func (ct *CostTracker) RecordUsage(modelInfo *ai.ModelInfo, inputTokens, outputTokens int) float64 {
	cost := CalculateCost(modelInfo, inputTokens, outputTokens)

	ct.mu.Lock()
	defer ct.mu.Unlock()

	if _, exists := ct.costs[modelInfo.Name]; !exists {
		ct.costs[modelInfo.Name] = &ModelCost{
			ModelID: modelInfo.Name,
		}
	}

	mc := ct.costs[modelInfo.Name]
	mc.InputTokens += int64(inputTokens)
	mc.OutputTokens += int64(outputTokens)
	mc.TotalCostUSD += cost
	mc.CallCount++

	return cost
}

// GetCost returns cost data for a specific model
func (ct *CostTracker) GetCost(modelID string) (*ModelCost, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	cost, ok := ct.costs[modelID]
	return cost, ok
}

// GetAllCosts returns all cost data
func (ct *CostTracker) GetAllCosts() map[string]ModelCost {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	costs := make(map[string]ModelCost, len(ct.costs))
	for id, cost := range ct.costs {
		costs[id] = *cost
	}

	return costs
}

// TotalCost returns the total cost across all models
func (ct *CostTracker) TotalCost() float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var total float64
	for _, cost := range ct.costs {
		total += cost.TotalCostUSD
	}

	return total
}

// RecordCost records a direct cost without model/token information
// Used by cost guard for simple tracking
func (ct *CostTracker) RecordCost(userID string, cost float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Store under special "direct" model key
	modelKey := "direct_cost"
	if mc, exists := ct.costs[modelKey]; exists {
		mc.TotalCostUSD += cost
		mc.CallCount++
	} else {
		ct.costs[modelKey] = &ModelCost{
			ModelID:      modelKey,
			TotalCostUSD: cost,
			CallCount:    1,
		}
	}
}

// Reset clears all cost data
func (ct *CostTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.costs = make(map[string]*ModelCost)
}

// CalculateCost calculates the cost for a given token usage
func CalculateCost(modelInfo *ai.ModelInfo, inputTokens, outputTokens int) float64 {
	inputCost := float64(inputTokens) / 1_000.0 * modelInfo.InputCostPer1K
	outputCost := float64(outputTokens) / 1_000.0 * modelInfo.OutputCostPer1K
	return inputCost + outputCost
}

// AgentExecutionResult contains the result of an agent execution
type AgentExecutionResult struct {
	Output        interface{}
	TokensUsed    int
	InputTokens   int
	OutputTokens  int
	CostUSD       float64
	Duration      time.Duration
	ToolCallCount int
}

// UserCostTracker tracks costs per user
type UserCostTracker struct {
	mu        sync.RWMutex
	userCosts map[string]*UserCost // user ID -> cost data
}

// UserCost tracks cost for a specific user
type UserCost struct {
	UserID         string
	TotalCostUSD   float64
	DailyCostUSD   float64
	MonthlyCostUSD float64
	LastReset      time.Time
}

// NewUserCostTracker creates a new user cost tracker
func NewUserCostTracker() *UserCostTracker {
	return &UserCostTracker{
		userCosts: make(map[string]*UserCost),
	}
}

// RecordUserUsage records cost for a specific user
func (uct *UserCostTracker) RecordUserUsage(userID string, cost float64) {
	uct.mu.Lock()
	defer uct.mu.Unlock()

	if _, exists := uct.userCosts[userID]; !exists {
		uct.userCosts[userID] = &UserCost{
			UserID:    userID,
			LastReset: time.Now(),
		}
	}

	uc := uct.userCosts[userID]
	uc.TotalCostUSD += cost
	uc.DailyCostUSD += cost
	uc.MonthlyCostUSD += cost
}

// GetUserCost returns cost data for a specific user
func (uct *UserCostTracker) GetUserCost(userID string) (*UserCost, bool) {
	uct.mu.RLock()
	defer uct.mu.RUnlock()

	cost, ok := uct.userCosts[userID]
	return cost, ok
}

// ResetDailyCosts resets daily costs for all users (called at midnight)
func (uct *UserCostTracker) ResetDailyCosts() {
	uct.mu.Lock()
	defer uct.mu.Unlock()

	for _, uc := range uct.userCosts {
		uc.DailyCostUSD = 0
		uc.LastReset = time.Now()
	}
}

// ResetMonthlyCosts resets monthly costs for all users (called at month start)
func (uct *UserCostTracker) ResetMonthlyCosts() {
	uct.mu.Lock()
	defer uct.mu.Unlock()

	for _, uc := range uct.userCosts {
		uc.MonthlyCostUSD = 0
	}
}

// ExceededDailyLimit checks if user exceeded daily cost limit
func (ct *CostTracker) ExceededDailyLimit(userID string) bool {
	// Simple implementation - check if total cost exceeds threshold
	// In production, use UserCostTracker for per-user limits
	const dailyLimit = 10.0 // $10 daily limit per user

	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var totalCost float64
	for _, mc := range ct.costs {
		totalCost += mc.TotalCostUSD
	}

	return totalCost > dailyLimit
}
