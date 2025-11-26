package trading
import (
	"prometheus/internal/tools/shared"
	"time"
	"prometheus/pkg/errors"
	"google.golang.org/adk/tool"
)
// NewGetPositionsTool returns open positions for the user
func NewGetPositionsTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"get_positions",
		"List open positions",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if deps.PositionRepo == nil {
				return nil, errors.Wrapf(errors.ErrInternal, "get_positions: position repository not configured")
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
				return nil, errors.Wrap(err, "get_positions: fetch positions")
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
		},
		deps,
	).
		WithTimeout(10*time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}
