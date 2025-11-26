package risk

import (
	"context"
	"time"

	"google.golang.org/adk/tool"

	"prometheus/internal/domain/user"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
)

// GetUserRiskProfileOutput defines the output schema for user risk profile
type GetUserRiskProfileOutput struct {
	RiskTolerance       string  `json:"risk_tolerance" description:"Risk tolerance level: conservative, moderate, aggressive"`
	MaxPositionSizePct  float64 `json:"max_position_size_pct" description:"Maximum position size as % of portfolio"`
	MaxDailyLossPct     float64 `json:"max_daily_loss_pct" description:"Maximum daily loss % before circuit breaker"`
	MaxDrawdownPct      float64 `json:"max_drawdown_pct" description:"Maximum drawdown % before circuit breaker"`
	CircuitBreakerOn    bool    `json:"circuit_breaker_on" description:"Whether circuit breaker is enabled"`
	PreferredAssets     []string `json:"preferred_assets" description:"User's preferred trading assets"`
	PreferredExchanges  []string `json:"preferred_exchanges" description:"User's preferred exchanges"`
	TradingEnabled      bool    `json:"trading_enabled" description:"Whether trading is currently enabled for user"`
}

// NewGetUserRiskProfileTool creates a tool for retrieving user's risk profile and preferences
func NewGetUserRiskProfileTool(deps shared.Deps) tool.Tool {
	handler := &getUserRiskProfileHandler{
		userRepo: deps.UserRepo,
	}

	return shared.NewToolBuilder(
		"get_user_risk_profile",
		"Get user's risk tolerance, position size limits, and trading preferences. "+
			"Essential for making risk-appropriate trading decisions.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Get user ID from context
			userID, err := parseUUIDArg(args["user_id"], "user_id")
			if err != nil {
				if meta, ok := shared.MetadataFromContext(ctx); ok {
					userID = meta.UserID
				} else {
					return nil, err
				}
			}

			output, err := handler.execute(ctx, userID)
			if err != nil {
				return nil, err
			}

			// Convert to map for ADK tool response
			return map[string]interface{}{
				"risk_tolerance":        output.RiskTolerance,
				"max_position_size_pct": output.MaxPositionSizePct,
				"max_daily_loss_pct":    output.MaxDailyLossPct,
				"max_drawdown_pct":      output.MaxDrawdownPct,
				"circuit_breaker_on":    output.CircuitBreakerOn,
				"preferred_assets":      output.PreferredAssets,
				"preferred_exchanges":   output.PreferredExchanges,
				"trading_enabled":       output.TradingEnabled,
			}, nil
		},
		deps,
	).
		WithTimeout(5 * time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}

type getUserRiskProfileHandler struct {
	userRepo user.Repository
}

func (h *getUserRiskProfileHandler) execute(ctx context.Context, userID interface{}) (*GetUserRiskProfileOutput, error) {
	// Convert userID to UUID
	uid, err := toUUID(userID)
	if err != nil {
		return nil, errors.Wrap(err, "invalid user_id")
	}

	// Get user from database
	usr, err := h.userRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user")
	}

	// Map risk tolerance to position size limits
	maxPositionSize := 10.0 // Default: 10%
	maxDailyLoss := 5.0     // Default: 5%
	maxDrawdown := 20.0     // Default: 20%

	riskLevel := usr.Settings.RiskLevel
	switch riskLevel {
	case "conservative":
		maxPositionSize = 5.0
		maxDailyLoss = 2.0
		maxDrawdown = 15.0
	case "moderate":
		maxPositionSize = 10.0
		maxDailyLoss = 5.0
		maxDrawdown = 25.0
	case "aggressive":
		maxPositionSize = 15.0
		maxDailyLoss = 10.0
		maxDrawdown = 35.0
	}

	// Extract preferred assets from settings
	// TODO: Add PreferredAssets and PreferredExchanges to user.Settings
	preferredAssets := []string{"BTC", "ETH", "SOL"} // Default for now

	// Extract preferred exchanges
	preferredExchanges := []string{"binance"} // Default for now

	return &GetUserRiskProfileOutput{
		RiskTolerance:      riskLevel,
		MaxPositionSizePct: maxPositionSize,
		MaxDailyLossPct:    maxDailyLoss,
		MaxDrawdownPct:     maxDrawdown,
		CircuitBreakerOn:   usr.Settings.CircuitBreakerOn,
		PreferredAssets:    preferredAssets,
		PreferredExchanges: preferredExchanges,
		TradingEnabled:     usr.IsActive && usr.Settings.CircuitBreakerOn,
	}, nil
}

