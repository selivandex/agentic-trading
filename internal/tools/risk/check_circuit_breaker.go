package risk

import (
	"github.com/google/uuid"

	"prometheus/internal/tools/shared"

	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewCheckCircuitBreakerTool reports whether trading is allowed.
func NewCheckCircuitBreakerTool(deps shared.Deps) tool.Tool {
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        "check_circuit_breaker",
			Description: "Check if trading is allowed",
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			userID := uuid.Nil
			if idVal, ok := args["user_id"]; ok {
				switch v := idVal.(type) {
				case string:
					parsed, err := uuid.Parse(v)
					if err == nil {
						userID = parsed
					}
				case uuid.UUID:
					userID = v
				}
			}
			if userID == uuid.Nil {
				if meta, ok := shared.MetadataFromContext(ctx); ok {
					userID = meta.UserID
				}
			}

			if userID == uuid.Nil || deps.RiskRepo == nil {
				return map[string]interface{}{"allowed": true, "reason": "no risk state configured"}, nil
			}

			state, err := deps.RiskRepo.GetState(ctx, userID)
			if err != nil {
				return nil, errors.Wrap(err, "check_circuit_breaker")
			}
			allowed := state == nil || !state.IsTriggered
			reason := "ok"
			if state != nil && state.IsTriggered {
				reason = state.TriggerReason
			}

			return map[string]interface{}{"allowed": allowed, "reason": reason}, nil
		})
	return t
}
