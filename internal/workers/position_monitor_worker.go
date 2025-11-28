package workers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/adk/agent"

	"prometheus/internal/domain/position"
	"prometheus/pkg/logger"
)

// PositionMonitorWorker monitors open positions and triggers PositionManager agent when needed.
// Replaces infinite loop agent with scheduled worker for cost efficiency.
type PositionMonitorWorker struct {
	positionRepo position.Repository
	agent        agent.Agent // PositionManager agent
	interval     time.Duration
	log          *logger.Logger
}

// NewPositionMonitorWorker creates a new position monitoring worker.
func NewPositionMonitorWorker(
	positionRepo position.Repository,
	positionManagerAgent agent.Agent,
	interval time.Duration,
) *PositionMonitorWorker {
	return &PositionMonitorWorker{
		positionRepo: positionRepo,
		agent:        positionManagerAgent,
		interval:     interval,
		log:          logger.Get().With("component", "position_monitor_worker"),
	}
}

// Run starts the position monitoring loop.
// Only calls LLM agent when there are actual positions to monitor.
func (w *PositionMonitorWorker) Run(ctx context.Context) error {
	w.log.Info("Position monitor worker started", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := w.checkPositions(ctx); err != nil {
				w.log.Error("Position check failed", "error", err)
				// Continue running despite errors
			}

		case <-ctx.Done():
			w.log.Info("Position monitor worker stopped")
			return ctx.Err()
		}
	}
}

// checkPositions fetches open positions and processes them if any exist.
func (w *PositionMonitorWorker) checkPositions(ctx context.Context) error {
	// TODO: Need to implement GetAllOpen method in position.Repository
	// For now, this is a placeholder that will be implemented when wiring workers
	// Current position.Repository only has GetOpenByUser(userID) method
	// We need to either:
	// 1. Add GetAllOpen() method to repository interface, or
	// 2. Iterate through all active users and call GetOpenByUser for each

	w.log.Debug("Position check placeholder - needs GetAllOpen implementation")
	return nil

	/*
		// Future implementation once GetAllOpen is added to repository:
		openPositions, err := w.positionRepo.GetAllOpen(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to get open positions")
		}

		if len(openPositions) == 0 {
			w.log.Debug("No open positions to monitor, skipping")
			return nil
		}

		// Monitoring and processing code will be enabled after GetAllOpen is implemented
		positionsByUser := w.groupByUser(openPositions)
		for userID, userPositions := range positionsByUser {
			if err := w.processUserPositions(ctx, userID, userPositions); err != nil {
				w.log.Error("Failed to process user positions", "user_id", userID, "error", err)
				continue
			}
		}
		return nil
	*/
}

// groupByUser groups positions by user ID.
func (w *PositionMonitorWorker) groupByUser(positions []*position.Position) map[uuid.UUID][]*position.Position {
	grouped := make(map[uuid.UUID][]*position.Position)

	for _, pos := range positions {
		grouped[pos.UserID] = append(grouped[pos.UserID], pos)
	}

	return grouped
}

// processUserPositions calls PositionManager agent for a user's positions.
func (w *PositionMonitorWorker) processUserPositions(
	ctx context.Context,
	userID uuid.UUID,
	positions []*position.Position,
) error {
	w.log.Debug("Processing user positions",
		"user_id", userID,
		"count", len(positions),
	)

	// TODO: Call agent with user's positions
	// For now, this is a placeholder - actual agent invocation
	// will be wired in cmd/main.go when integrating with ADK session system

	w.log.Info("User positions processed",
		"user_id", userID,
		"positions", len(positions),
	)

	return nil
}
