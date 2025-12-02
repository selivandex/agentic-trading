package trading_pair

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for trading pair persistence
type Repository interface {
	// Create inserts a new trading pair
	Create(ctx context.Context, pair *TradingPair) error

	// GetByID retrieves a trading pair by ID
	GetByID(ctx context.Context, id uuid.UUID) (*TradingPair, error)

	// GetByUser retrieves all trading pairs for a user
	GetByUser(ctx context.Context, userID uuid.UUID) ([]*TradingPair, error)

	// GetActiveByUser retrieves active trading pairs for a user (including paused ones)
	GetActiveByUser(ctx context.Context, userID uuid.UUID) ([]*TradingPair, error)

	// GetActiveBySymbol retrieves active trading pairs for a specific symbol across all users
	GetActiveBySymbol(ctx context.Context, symbol string) ([]*TradingPair, error)

	// FindAllActive retrieves ALL active trading pairs from ALL users
	FindAllActive(ctx context.Context) ([]*TradingPair, error)

	// Update updates a trading pair
	Update(ctx context.Context, pair *TradingPair) error

	// Pause pauses a trading pair
	Pause(ctx context.Context, id uuid.UUID, reason string) error

	// Resume resumes a trading pair
	Resume(ctx context.Context, id uuid.UUID) error

	// Disable disables a trading pair (sets is_active to false)
	Disable(ctx context.Context, id uuid.UUID) error

	// Delete deletes a trading pair
	Delete(ctx context.Context, id uuid.UUID) error
}
