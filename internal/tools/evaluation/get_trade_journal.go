package evaluation

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/position"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"prometheus/pkg/templates"

	"google.golang.org/adk/tool"
)

// NewGetTradeJournalTool returns trade journal entries with reasoning and outcomes
// MVP version: returns list of closed positions with basic info
func NewGetTradeJournalTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"get_trade_journal",
		"Get trade journal entries for a user. Returns trade history with entry/exit prices, P&L, reasoning, and outcomes. Useful for reviewing past trades and learning from mistakes.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Parse user_id
			userIDStr, ok := args["user_id"].(string)
			if !ok || userIDStr == "" {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "user_id is required")
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "invalid user_id format: %v", err)
			}

			// Parse limit (optional, default: 20)
			limit := 20
			if limitArg, ok := args["limit"].(float64); ok {
				limit = int(limitArg)
			} else if limitArg, ok := args["limit"].(int); ok {
				limit = limitArg
			}

			if limit < 1 {
				limit = 20
			} else if limit > 100 {
				limit = 100 // Cap at 100
			}

			// Parse status filter (optional, default: closed)
			status, _ := args["status"].(string)
			if status == "" {
				status = "closed"
			}

			// Parse min_date (optional)
			var startTime time.Time
			if minDateStr, ok := args["min_date"].(string); ok && minDateStr != "" {
				startTime, err = time.Parse(time.RFC3339, minDateStr)
				if err != nil {
					return nil, errors.Wrapf(errors.ErrInvalidInput, "invalid min_date format (use RFC3339): %v", err)
				}
			} else {
				// Default: last 30 days
				startTime = time.Now().AddDate(0, 0, -30)
			}

			// Get positions based on status
			var positions []*position.Position

			switch status {
			case "closed":
				positions, err = deps.PositionRepo.GetClosedInRange(ctx, userID, startTime, time.Now())
			case "open":
				positions, err = deps.PositionRepo.GetOpenByUser(ctx, userID)
			case "all":
				// Get both open and closed
				openPos, err1 := deps.PositionRepo.GetOpenByUser(ctx, userID)
				if err1 != nil {
					return nil, errors.Wrap(err1, "failed to fetch open positions")
				}
				closedPos, err2 := deps.PositionRepo.GetClosedInRange(ctx, userID, startTime, time.Now())
				if err2 != nil {
					return nil, errors.Wrap(err2, "failed to fetch closed positions")
				}
				positions = append(openPos, closedPos...)
			default:
				return nil, errors.Wrapf(errors.ErrInvalidInput, "invalid status: %s (valid: open, closed, all)", status)
			}

			if err != nil {
				return nil, errors.Wrap(err, "failed to fetch positions")
			}

			// Apply limit
			if len(positions) > limit {
				positions = positions[:limit]
			}

			// Format entries
			entries := make([]map[string]interface{}, 0, len(positions))
			for _, pos := range positions {
				entry := formatTradeEntry(pos)
				entries = append(entries, entry)
			}

			// Prepare template data
			templateData := map[string]interface{}{
				"Entries":    entries,
				"TotalCount": len(entries),
				"Limit":      limit,
				"Status":     status,
			}

			// Render using template
			tmpl := templates.Get()
			journal, err := tmpl.Render("tools/trade_journal", templateData)
			if err != nil {
				return nil, errors.Wrap(err, "failed to render trade journal template")
			}

			return map[string]interface{}{
				"journal": journal,
			}, nil
		},
		deps,
	).
		WithTimeout(10*time.Second).
		WithRetry(2, 1*time.Second).
		Build()
}

// formatTradeEntry formats a position into a journal entry
func formatTradeEntry(pos *position.Position) map[string]interface{} {
	entry := map[string]interface{}{
		"trade_id":    pos.ID.String(),
		"symbol":      pos.Symbol,
		"side":        string(pos.Side),
		"status":      string(pos.Status),
		"market_type": pos.MarketType,
		"opened_at":   pos.OpenedAt.Format(time.RFC3339),
	}

	// Entry price
	entryPrice, _ := pos.EntryPrice.Float64()
	entry["entry_price"] = entryPrice

	// Size
	size, _ := pos.Size.Float64()
	entry["size"] = size

	// Current/exit price
	if pos.Status == position.PositionClosed || pos.Status == position.PositionLiquidated {
		// For closed positions, current_price is the exit price
		exitPrice, _ := pos.CurrentPrice.Float64()
		entry["exit_price"] = exitPrice

		// Realized P&L
		pnl, _ := pos.RealizedPnL.Float64()
		entry["pnl"] = pnl

		// Calculate P&L percentage
		if pos.EntryPrice.GreaterThan(decimal.Zero) {
			pnlPct := pos.RealizedPnL.Div(pos.EntryPrice).Mul(decimal.NewFromInt(100))
			pnlPctFloat, _ := pnlPct.Float64()
			entry["pnl_percent"] = pnlPctFloat
		} else {
			entry["pnl_percent"] = 0.0
		}

		// Outcome
		if pnl > 0 {
			entry["outcome"] = "win"
		} else if pnl < 0 {
			entry["outcome"] = "loss"
		} else {
			entry["outcome"] = "breakeven"
		}

		// Closed at
		if pos.ClosedAt != nil {
			entry["closed_at"] = pos.ClosedAt.Format(time.RFC3339)

			// Hold time
			holdTime := pos.ClosedAt.Sub(pos.OpenedAt)
			entry["hold_time"] = formatDuration(holdTime)
		}
	} else {
		// For open positions, show current price and unrealized P&L
		currentPrice, _ := pos.CurrentPrice.Float64()
		entry["current_price"] = currentPrice

		unrealizedPnL, _ := pos.UnrealizedPnL.Float64()
		entry["unrealized_pnl"] = unrealizedPnL

		unrealizedPnLPct, _ := pos.UnrealizedPnLPct.Float64()
		entry["unrealized_pnl_percent"] = unrealizedPnLPct

		entry["outcome"] = "open"
	}

	// Reasoning (if available)
	if pos.OpenReasoning != "" {
		entry["reasoning"] = pos.OpenReasoning
	}

	// Risk management levels
	if !pos.StopLossPrice.IsZero() {
		stopLoss, _ := pos.StopLossPrice.Float64()
		entry["stop_loss"] = stopLoss
	}

	if !pos.TakeProfitPrice.IsZero() {
		takeProfit, _ := pos.TakeProfitPrice.Float64()
		entry["take_profit"] = takeProfit
	}

	// Leverage (for futures)
	if pos.Leverage > 1 {
		entry["leverage"] = pos.Leverage
	}

	return entry
}
