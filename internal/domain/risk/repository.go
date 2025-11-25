package risk

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for risk state data access
type Repository interface {
	// Circuit breaker state
	GetState(ctx context.Context, userID uuid.UUID) (*CircuitBreakerState, error)
	SaveState(ctx context.Context, state *CircuitBreakerState) error
	ResetDaily(ctx context.Context) error

	// Risk events
	CreateEvent(ctx context.Context, event *RiskEvent) error
	GetEvents(ctx context.Context, userID uuid.UUID, limit int) ([]*RiskEvent, error)
	AcknowledgeEvent(ctx context.Context, eventID uuid.UUID) error
}
