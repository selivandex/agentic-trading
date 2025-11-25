package macro

import (
	"context"
	"time"
)

// Repository defines the interface for macro data access
type Repository interface {
	InsertEvent(ctx context.Context, event *MacroEvent) error
	GetUpcomingEvents(ctx context.Context, from time.Time, to time.Time) ([]MacroEvent, error)
	GetEventsByType(ctx context.Context, eventType EventType, limit int) ([]MacroEvent, error)
	GetHighImpactEvents(ctx context.Context, from time.Time, to time.Time) ([]MacroEvent, error)
}
