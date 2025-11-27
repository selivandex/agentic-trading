package trading

import (
	"math"
	"time"

	"github.com/google/uuid"
	"google.golang.org/adk/tool"

	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"prometheus/pkg/templates"
)

// NewAccountStatusTool returns comprehensive account status in one call.
// Combines: get_balance (accounts), get_positions (open positions), get_portfolio_summary (aggregates), get_user_risk_profile.
// Returns human-readable text format optimized for LLM consumption.
func NewAccountStatusTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"get_account_status",
		"Get comprehensive account status. Returns connected exchanges, open positions, portfolio summary, PnL, and risk profile in readable format.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Get user ID
			userID, err := parseUUIDArg(args["user_id"], "user_id")
			if err != nil {
				if meta, ok := shared.MetadataFromContext(ctx); ok {
					userID = meta.UserID
				} else {
					return nil, err
				}
			}

			templateData := map[string]interface{}{
				"UserID":    userID.String(),
				"Timestamp": time.Now().UTC().Format(time.RFC3339),
			}

			// 1. Get connected exchange accounts
			accounts := make([]map[string]interface{}, 0)
			if deps.ExchangeAccountRepo != nil {
				accs, err := deps.ExchangeAccountRepo.GetActiveByUser(ctx, userID)
				if err == nil {
					for _, acc := range accs {
						accounts = append(accounts, map[string]interface{}{
							"Exchange":    acc.Exchange.String(),
							"Label":       acc.Label,
							"IsTestnet":   acc.IsTestnet,
							"Permissions": acc.Permissions,
						})
					}
				}
			}
			templateData["Accounts"] = accounts
			templateData["AccountsCount"] = len(accounts)

			// 2. Get open positions
			positions := make([]map[string]interface{}, 0)
			portfolio := map[string]interface{}{
				"TotalEquityUSD":   0.0,
				"TotalPnLUSD":      0.0,
				"TotalPnLPercent":  0.0,
				"LongExposure":     0.0,
				"ShortExposure":    0.0,
				"NetExposure":      0.0,
				"DailyPnL":         0.0,
				"ExposureBySymbol": make(map[string]float64),
			}
			health := map[string]interface{}{
				"Status":       "healthy",
				"Warnings":     []string{},
				"WarningCount": 0,
			}

			if deps.PositionRepo != nil {
				pos, err := deps.PositionRepo.GetOpenByUser(ctx, userID)
				if err == nil {
					var totalEquity, totalPnL, longExp, shortExp, dailyPnL float64
					exposureBySymbol := make(map[string]float64)

					for _, p := range pos {
						currentPrice, _ := p.CurrentPrice.Float64()
						size, _ := p.Size.Float64()
						entryPrice, _ := p.EntryPrice.Float64()
						pnl, _ := p.UnrealizedPnL.Float64()

						positionValue := currentPrice * size
						totalEquity += positionValue
						totalPnL += pnl
						exposureBySymbol[p.Symbol] += positionValue

						if p.Side.String() == "long" {
							longExp += positionValue
						} else {
							shortExp += positionValue
						}

						if p.OpenedAt.After(time.Now().Add(-24 * time.Hour)) {
							dailyPnL += pnl
						}

						positions = append(positions, map[string]interface{}{
							"Symbol":        p.Symbol,
							"Side":          p.Side.String(),
							"Size":          size,
							"EntryPrice":    entryPrice,
							"CurrentPrice":  currentPrice,
							"UnrealizedPnl": pnl,
							"PnlPercent":    calculatePnLPercent(entryPrice, currentPrice, p.Side.String()),
							"Status":        p.Status.String(),
						})
					}

					totalPnLPercent := 0.0
					if totalEquity > 0 {
						totalPnLPercent = (totalPnL / totalEquity) * 100
					}

					portfolio["TotalEquityUSD"] = math.Round(totalEquity*100) / 100
					portfolio["TotalPnLUSD"] = math.Round(totalPnL*100) / 100
					portfolio["TotalPnLPercent"] = math.Round(totalPnLPercent*100) / 100
					portfolio["LongExposure"] = math.Round(longExp*100) / 100
					portfolio["ShortExposure"] = math.Round(shortExp*100) / 100
					portfolio["NetExposure"] = math.Round((longExp-shortExp)*100) / 100
					portfolio["DailyPnL"] = math.Round(dailyPnL*100) / 100
					portfolio["ExposureBySymbol"] = exposureBySymbol

					// Health assessment
					healthStatus, warnings := assessHealth(totalPnLPercent, longExp, shortExp, len(pos))
					health["Status"] = healthStatus
					health["Warnings"] = warnings
					health["WarningCount"] = len(warnings)
				}
			}

			templateData["Positions"] = positions
			templateData["PositionsCount"] = len(positions)
			templateData["Portfolio"] = portfolio
			templateData["Health"] = health

			// 3. Add user risk profile
			riskProfile := map[string]interface{}{
				"RiskLevel":          "moderate",
				"MaxPositionSizeUSD": 0.0,
				"MaxTotalExposure":   0.0,
				"MaxPortfolioRisk":   0.0,
				"MaxDailyDrawdown":   0.0,
				"MaxPositions":       0,
				"CircuitBreakerOn":   false,
			}
			if deps.UserRepo != nil && userID != uuid.Nil {
				user, err := deps.UserRepo.GetByID(ctx, userID)
				if err == nil && user != nil {
					risk := user.GetRiskTolerance()
					riskProfile["RiskLevel"] = user.Settings.RiskLevel
					riskProfile["MaxPositionSizeUSD"] = risk.MaxPositionSizeUSD
					riskProfile["MaxTotalExposure"] = risk.MaxTotalExposureUSD
					riskProfile["MaxPortfolioRisk"] = risk.MaxPortfolioRiskPct
					riskProfile["MaxDailyDrawdown"] = risk.MaxDailyDrawdownPct
					riskProfile["MaxPositions"] = risk.MaxPositions
					riskProfile["CircuitBreakerOn"] = user.Settings.CircuitBreakerOn
				}
			}
			templateData["RiskProfile"] = riskProfile

			// Render using template
			tmpl := templates.Get()
			analysis, err := tmpl.Render("tools/account_status", templateData)
			if err != nil {
				return nil, errors.Wrap(err, "failed to render account status template")
			}

			return map[string]interface{}{
				"status": analysis,
			}, nil
		},
		deps,
	).
		WithTimeout(15*time.Second).
		WithRetry(2, 1*time.Second).
		Build()
}

// calculatePnLPercent calculates PnL percentage for a position
func calculatePnLPercent(entry, current float64, side string) float64 {
	if entry == 0 {
		return 0
	}
	if side == "long" {
		return ((current - entry) / entry) * 100
	}
	return ((entry - current) / entry) * 100
}

// assessHealth generates health assessment
func assessHealth(pnlPct, longExp, shortExp float64, posCount int) (string, []string) {
	status := "healthy"
	warnings := make([]string, 0)

	// Check PnL
	if pnlPct < -10 {
		status = "at_risk"
		warnings = append(warnings, "significant_drawdown")
	} else if pnlPct < -5 {
		status = "caution"
		warnings = append(warnings, "moderate_drawdown")
	}

	// Check concentration
	totalExp := longExp + shortExp
	if totalExp > 0 {
		netExpRatio := math.Abs(longExp-shortExp) / totalExp
		if netExpRatio > 0.8 {
			warnings = append(warnings, "high_directional_exposure")
		}
	}

	// Check position count
	if posCount > 10 {
		warnings = append(warnings, "many_open_positions")
	}

	return status, warnings
}
