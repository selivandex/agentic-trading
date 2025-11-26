package risk

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool/functiontool"
)

// NewEmergencyCloseAllTool closes all open positions for the user.
func NewEmergencyCloseAllTool(deps shared.Deps) *functiontool.Tool {
	return functiontool.New("emergency_close_all", "Kill switch to close everything", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if deps.PositionRepo == nil {
			return nil, fmt.Errorf("emergency_close_all: position repository not configured")
		}

		meta, ok := shared.MetadataFromContext(ctx)
		if !ok || meta.UserID == uuid.Nil {
			return nil, fmt.Errorf("emergency_close_all: user metadata required")
		}

		positions, err := deps.PositionRepo.GetOpenByUser(ctx, meta.UserID)
		if err != nil {
			return nil, fmt.Errorf("emergency_close_all: fetch positions: %w", err)
		}

		closed := 0
		for _, pos := range positions {
			price := pos.CurrentPrice
			if price.IsZero() {
				price = pos.EntryPrice
			}
			if err := deps.PositionRepo.Close(ctx, pos.ID, price, decimal.Zero); err == nil {
				closed++
			}
		}

		return map[string]interface{}{"closed_positions": closed}, nil
	})
}
