package risk

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/adk/tool"

	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
)

// NewPreTradeCheckTool performs comprehensive pre-trade validation in one call.
// Combines: check_circuit_breaker + validate_trade.
// This replaces 2 individual tools with a single comprehensive check.
func NewPreTradeCheckTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"pre_trade_check",
		"Comprehensive pre-trade validation in one call. Checks circuit breaker status and validates trade parameters. USE THIS BEFORE PLACING ANY ORDER.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Get user ID
			userID := uuid.Nil
			if idVal, ok := args["user_id"]; ok {
				switch v := idVal.(type) {
				case string:
					if parsed, err := uuid.Parse(v); err == nil {
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

			result := map[string]interface{}{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"user_id":   userID.String(),
			}

			// 1. Circuit Breaker Check
			circuitBreaker := map[string]interface{}{
				"allowed": true,
				"reason":  "no risk state configured",
			}

			if userID != uuid.Nil && deps.RiskRepo != nil {
				state, err := deps.RiskRepo.GetState(ctx, userID)
				if err != nil {
					return nil, errors.Wrap(err, "pre_trade_check: circuit breaker")
				}

				if state != nil && state.IsTriggered {
					circuitBreaker["allowed"] = false
					circuitBreaker["reason"] = state.TriggerReason
					circuitBreaker["triggered_at"] = state.TriggeredAt.Format(time.RFC3339)
				} else {
					circuitBreaker["reason"] = "ok"
				}
			}
			result["circuit_breaker"] = circuitBreaker

			// 2. Trade Validation (if amount provided)
			tradeValidation := map[string]interface{}{
				"valid": true,
			}

			if amountStr, ok := args["amount"].(string); ok && amountStr != "" {
				amount, err := decimal.NewFromString(amountStr)
				if err != nil || amount.LessThanOrEqual(decimal.Zero) {
					tradeValidation["valid"] = false
					tradeValidation["error"] = "invalid_amount"
				} else {
					tradeValidation["amount"] = amount.String()

					// Check against user risk tolerance if available
					if deps.UserRepo != nil && userID != uuid.Nil {
						user, err := deps.UserRepo.GetByID(ctx, userID)
						if err == nil && user != nil {
							risk := user.GetRiskTolerance()
							amountFloat, _ := amount.Float64()

							if risk.MaxPositionSizeUSD > 0 && amountFloat > risk.MaxPositionSizeUSD {
								tradeValidation["warning"] = "exceeds_max_position_size"
								tradeValidation["max_allowed"] = risk.MaxPositionSizeUSD
							}
						}
					}
				}
			}
			result["trade_validation"] = tradeValidation

			// 3. Overall decision
			canTrade := circuitBreaker["allowed"].(bool) && tradeValidation["valid"].(bool)
			result["can_trade"] = canTrade

			if !canTrade {
				reasons := make([]string, 0)
				if !circuitBreaker["allowed"].(bool) {
					reasons = append(reasons, "circuit_breaker_triggered")
				}
				if !tradeValidation["valid"].(bool) {
					reasons = append(reasons, "trade_validation_failed")
				}
				result["block_reasons"] = reasons
			}

			return result, nil
		},
		deps,
	).
		WithTimeout(10*time.Second).
		WithRetry(2, 500*time.Millisecond).
		Build()
}
