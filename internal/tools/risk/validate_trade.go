package risk
import (
	"time"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"google.golang.org/adk/tool"
)
// NewValidateTradeTool performs lightweight pre-trade checks
func NewValidateTradeTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"validate_trade",
		"Pre-trade validation checks",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			amountStr, _ := args["amount"].(string)
			if amountStr == "" {
				return nil, errors.ErrInvalidInput
			}
			amount, err := decimal.NewFromString(amountStr)
			if err != nil || amount.LessThanOrEqual(decimal.Zero) {
				return nil, errors.ErrInvalidInput
			}
			// Optional circuit breaker check when risk repository is present
			userID := uuid.Nil
			if meta, ok := shared.MetadataFromContext(ctx); ok {
				userID = meta.UserID
			}
			allowed := true
			if userID != uuid.Nil && deps.RiskRepo != nil {
				state, err := deps.RiskRepo.GetState(ctx, userID)
				if err != nil {
					return nil, errors.Wrap(err, "validate_trade")
				}
				allowed = state == nil || !state.IsTriggered
			}
			return map[string]interface{}{"valid": allowed, "amount": amount.String()}, nil
		},
		deps,
	).
		WithTimeout(10*time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}
