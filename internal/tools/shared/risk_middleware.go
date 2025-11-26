package shared
import (
	"github.com/google/uuid"
	"google.golang.org/adk/tool"
	"prometheus/pkg/errors"
)
// ToolExecutor is a function type that executes a tool with given arguments
type ToolExecutor func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error)
// WithRiskCheck wraps a trading tool execution with risk engine checks
// It ensures that the user is allowed to trade before executing the tool
func WithRiskCheck(deps Deps, executor ToolExecutor) ToolExecutor {
	return func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
		// Skip risk check if risk engine is not configured
		if !deps.HasRiskEngine() {
			return executor(ctx, args)
		}
		// Extract user ID from context metadata or args
		var userID uuid.UUID
		var err error
		if meta, ok := MetadataFromContext(ctx); ok {
			userID = meta.UserID
		} else {
			// Try to extract from args
			if userIDStr, ok := args["user_id"].(string); ok {
				userID, err = uuid.Parse(userIDStr)
				if err != nil {
					return nil, errors.Wrap(err, "risk_check: invalid user_id")
				}
			} else {
				// If no user_id in context or args, skip risk check
				// (some tools may not require user context)
				return executor(ctx, args)
			}
		}
		// Check if trading is allowed for this user
		canTrade, err := deps.RiskEngine.CanTrade(ctx, userID)
		if err != nil {
			// Check if it's a domain error (circuit breaker, drawdown, etc.)
			if errors.Is(err, errors.ErrCircuitBreakerTripped) ||
				errors.Is(err, errors.ErrDrawdownExceeded) ||
				errors.Is(err, errors.ErrConsecutiveLosses) {
				deps.Log.Warn("Trading blocked by risk engine",
					"user_id", userID,
					"reason", err.Error(),
				)
				return nil, err // Return domain error as-is
			}
			// For other errors, wrap with context
			return nil, errors.Wrap(err, "risk_check: failed to check trading permission")
		}
		if !canTrade {
			deps.Log.Warn("Trading blocked by risk engine",
				"user_id", userID,
				"tool", "trading",
			)
			return nil, errors.ErrTradingBlocked
		}
		// Risk check passed, execute the tool
		deps.Log.Debug("Risk check passed", "user_id", userID)
		return executor(ctx, args)
	}
}
// WithRiskCheckMultiple applies risk check middleware to multiple tools
func WithRiskCheckMultiple(deps Deps, executors map[string]ToolExecutor) map[string]ToolExecutor {
	wrapped := make(map[string]ToolExecutor)
	for name, executor := range executors {
		wrapped[name] = WithRiskCheck(deps, executor)
	}
	return wrapped
}
