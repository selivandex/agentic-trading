package ai_usage

import (
	"context"
	"time"
)

// Repository defines operations for AI usage tracking
type Repository interface {
	// Store saves a usage log entry
	Store(ctx context.Context, log *UsageLog) error

	// GetUserDailyCost returns total cost for a user on a specific day
	GetUserDailyCost(ctx context.Context, userID string, date time.Time) (float64, error)

	// GetUserMonthlyCost returns total cost for a user in a specific month
	GetUserMonthlyCost(ctx context.Context, userID string, year int, month int) (float64, error)

	// GetProviderCosts returns costs grouped by provider for a time range
	GetProviderCosts(ctx context.Context, from, to time.Time) (map[string]float64, error)

	// GetAgentCosts returns costs grouped by agent type for a time range
	GetAgentCosts(ctx context.Context, from, to time.Time) (map[string]float64, error)

	// GetModelCosts returns costs grouped by model for a time range
	GetModelCosts(ctx context.Context, provider string, from, to time.Time) (map[string]float64, error)
}
