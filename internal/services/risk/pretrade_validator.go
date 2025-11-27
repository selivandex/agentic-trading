package riskservice

import (
	"fmt"
	"math"
)

// PreTradeValidator performs basic validation checks before sizing
type PreTradeValidator struct {
	minRR             float64
	maxLeverage       float64
	requireStopLoss   bool
	requireTakeProfit bool
	maxSlippagePct    float64
}

// Validate performs pre-trade validation checks
func (v *PreTradeValidator) Validate(plan *TradePlan, currentPrice float64) *ValidationResult {
	var errors []string
	var warnings []string

	// 1. Stop-loss validation
	if v.requireStopLoss && plan.StopLoss == 0 {
		errors = append(errors, "stop_loss_required")
	}

	// 2. Stop-loss must be on correct side
	if plan.StopLoss != 0 {
		if plan.Direction == "long" && plan.StopLoss >= plan.Entry {
			errors = append(errors, "invalid_stop_loss_long: SL must be below entry")
		}
		if plan.Direction == "short" && plan.StopLoss <= plan.Entry {
			errors = append(errors, "invalid_stop_loss_short: SL must be above entry")
		}

		// Check if stop-loss is too tight (<0.5%)
		slDistance := math.Abs(plan.Entry-plan.StopLoss) / plan.Entry
		if slDistance < 0.005 {
			warnings = append(warnings,
				fmt.Sprintf("stop_loss_very_tight: %.2f%%", slDistance*100))
		}
	}

	// 3. Take-profit validation
	if v.requireTakeProfit && plan.TakeProfit == 0 {
		errors = append(errors, "take_profit_required")
	}

	if plan.TakeProfit != 0 {
		if plan.Direction == "long" && plan.TakeProfit <= plan.Entry {
			warnings = append(warnings, "take_profit_below_entry")
		}
		if plan.Direction == "short" && plan.TakeProfit >= plan.Entry {
			warnings = append(warnings, "take_profit_above_entry")
		}
	}

	// 4. Risk/Reward ratio check
	rr := v.calculateRR(plan)
	if rr < v.minRR {
		errors = append(errors,
			fmt.Sprintf("rr_too_low: %.2f < %.2f minimum", rr, v.minRR))
	} else if rr < v.minRR*1.2 {
		// Close to minimum, warn
		warnings = append(warnings,
			fmt.Sprintf("rr_marginal: %.2f (min: %.2f)", rr, v.minRR))
	}

	// 5. Leverage check
	if plan.Leverage > v.maxLeverage {
		errors = append(errors,
			fmt.Sprintf("leverage_exceeded: %.1fx > %.1fx max", plan.Leverage, v.maxLeverage))
	} else if plan.Leverage > v.maxLeverage*0.8 {
		warnings = append(warnings,
			fmt.Sprintf("leverage_high: %.1fx (max: %.1fx)", plan.Leverage, v.maxLeverage))
	}

	// 6. Entry price vs current price (slippage check)
	if currentPrice > 0 {
		priceDiff := math.Abs(plan.Entry-currentPrice) / currentPrice
		if priceDiff > v.maxSlippagePct {
			warnings = append(warnings,
				fmt.Sprintf("entry_price_divergence: %.2f%%", priceDiff*100))
		}
	}

	// 7. Price sanity checks
	if plan.Entry <= 0 {
		errors = append(errors, "invalid_entry_price: must be > 0")
	}
	if plan.StopLoss < 0 {
		errors = append(errors, "invalid_stop_loss: must be >= 0")
	}
	if plan.TakeProfit < 0 {
		errors = append(errors, "invalid_take_profit: must be >= 0")
	}

	// 8. Direction validation
	if plan.Direction != "long" && plan.Direction != "short" {
		errors = append(errors,
			fmt.Sprintf("invalid_direction: %s (must be 'long' or 'short')", plan.Direction))
	}

	// 9. Leverage >= 1.0
	if plan.Leverage < 1.0 {
		errors = append(errors,
			fmt.Sprintf("invalid_leverage: %.2f (must be >= 1.0)", plan.Leverage))
	}

	return &ValidationResult{
		Valid:    len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
		RR:       rr,
	}
}

// calculateRR calculates Risk/Reward ratio
func (v *PreTradeValidator) calculateRR(plan *TradePlan) float64 {
	if plan.StopLoss == 0 || plan.TakeProfit == 0 {
		return 0
	}

	risk := math.Abs(plan.Entry - plan.StopLoss)
	reward := math.Abs(plan.TakeProfit - plan.Entry)

	if risk == 0 {
		return 0
	}

	return reward / risk
}
