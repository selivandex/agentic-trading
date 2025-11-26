package risk

import (
	"context"

	"github.com/google/uuid"

	riskpkg "prometheus/internal/risk"
	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"prometheus/pkg/errors"
)

// NewEmergencyCloseAllTool closes all open positions for the user using the kill switch.
func NewEmergencyCloseAllTool(deps shared.Deps) tool.Tool {
	return functiontool.New("emergency_close_all", "Kill switch to close all positions and cancel all orders immediately", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if deps.PositionRepo == nil || deps.OrderRepo == nil || deps.RiskRepo == nil {
			return nil, errors.Wrapf(errors.ErrInternal, "emergency_close_all: required repositories not configured")
		}

		// Extract user ID from context or args
		var userID uuid.UUID
		meta, ok := shared.MetadataFromContext(ctx)
		if ok && meta.UserID != uuid.Nil {
			userID = meta.UserID
		} else {
			// Try to extract from args
			if userIDStr, ok := args["user_id"].(string); ok {
				var err error
				userID, err = uuid.Parse(userIDStr)
				if err != nil {
					return nil, errors.Wrap(err, "emergency_close_all: invalid user_id")
				}
			} else {
				return nil, errors.ErrInvalidInput
			}
		}

		// Get reason from args
		reason, _ := args["reason"].(string)
		if reason == "" {
			reason = "Emergency close all triggered by user"
		}

		// Create kill switch instance
		killSwitch := riskpkg.NewKillSwitch(
			deps.PositionRepo,
			deps.OrderRepo,
			deps.RiskRepo,
			deps.Redis,
			deps.Log,
		)

		// Activate kill switch
		result, err := killSwitch.Activate(ctx, userID, reason)
		if err != nil {
			return nil, errors.Wrap(err, "emergency_close_all")
		}

		// Build response
		response := map[string]interface{}{
			"success":          true,
			"positions_closed": result.PositionsClosed,
			"orders_cancelled": result.OrdersCancelled,
			"circuit_breaker":  result.CircuitBreakerSet,
			"activated_at":     result.ActivatedAt.Format("2006-01-02 15:04:05"),
		}

		if len(result.Errors) > 0 {
			response["errors"] = result.Errors
			response["partial_success"] = true
		}

		return response, nil
	})
}
