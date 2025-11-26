package trading

import (
	"context"
	"fmt"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool/functiontool"
)

// NewGetPositionsTool returns open positions for the user.
func NewGetPositionsTool(deps shared.Deps) *functiontool.Tool {
	return functiontool.New("get_positions", "List open positions", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if deps.PositionRepo == nil {
			return nil, fmt.Errorf("get_positions: position repository not configured")
		}

		userID, err := parseUUIDArg(args["user_id"], "user_id")
		if err != nil {
			if meta, ok := shared.MetadataFromContext(ctx); ok {
				userID = meta.UserID
			} else {
				return nil, err
			}
		}

		positions, err := deps.PositionRepo.GetOpenByUser(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("get_positions: fetch positions: %w", err)
		}

		data := make([]map[string]interface{}, 0, len(positions))
		for _, p := range positions {
			data = append(data, map[string]interface{}{
				"id":             p.ID.String(),
				"symbol":         p.Symbol,
				"side":           p.Side.String(),
				"size":           p.Size.String(),
				"entry_price":    p.EntryPrice.String(),
				"current_price":  p.CurrentPrice.String(),
				"unrealized_pnl": p.UnrealizedPnL.String(),
				"status":         p.Status.String(),
			})
		}

		return map[string]interface{}{"positions": data}, nil
	})
}
