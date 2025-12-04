package risk

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// PositionValidator performs pre-trade validation checks
type PositionValidator struct {
	minRiskReward   decimal.Decimal
	maxLeverage     int
	requireStopLoss bool
}

// TradeParams contains parameters for a proposed trade
type TradeParams struct {
	Symbol     string
	Direction  string // "long" or "short"
	Entry      decimal.Decimal
	StopLoss   decimal.Decimal
	TakeProfit decimal.Decimal
	Size       decimal.Decimal
	Leverage   int
}

// ValidationResult contains validation result
type ValidationResult struct {
	Valid        bool
	Errors       []string
	Warnings     []string
	RiskReward   decimal.Decimal
	StopDistance decimal.Decimal // % distance to stop
}

// NewPositionValidator creates a new position validator
func NewPositionValidator(minRR float64, maxLeverage int, requireStopLoss bool) *PositionValidator {
	return &PositionValidator{
		minRiskReward:   decimal.NewFromFloat(minRR),
		maxLeverage:     maxLeverage,
		requireStopLoss: requireStopLoss,
	}
}

// Validate performs comprehensive pre-trade validation
func (v *PositionValidator) Validate(params *TradeParams) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// 1. Stop-loss validation
	if v.requireStopLoss && params.StopLoss.IsZero() {
		result.Valid = false
		result.Errors = append(result.Errors, "stop_loss_required")
		return result // Can't continue without stop-loss
	}

	if !params.StopLoss.IsZero() {
		// Check stop-loss is on correct side
		if params.Direction == "long" && params.StopLoss.GreaterThanOrEqual(params.Entry) {
			result.Valid = false
			result.Errors = append(result.Errors, "invalid_stop_loss_long: must be below entry")
		}
		if params.Direction == "short" && params.StopLoss.LessThanOrEqual(params.Entry) {
			result.Valid = false
			result.Errors = append(result.Errors, "invalid_stop_loss_short: must be above entry")
		}

		// Calculate stop distance
		stopDiff := params.Entry.Sub(params.StopLoss).Abs()
		result.StopDistance = stopDiff.Div(params.Entry).Mul(decimal.NewFromInt(100)) // as %

		// Warn if stop is very tight (<0.5%)
		if result.StopDistance.LessThan(decimal.NewFromFloat(0.5)) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("stop_very_tight: %.2f%%", result.StopDistance.InexactFloat64()))
		}

		// Warn if stop is very wide (>10%)
		if result.StopDistance.GreaterThan(decimal.NewFromFloat(10)) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("stop_very_wide: %.2f%%", result.StopDistance.InexactFloat64()))
		}
	}

	// 2. Take-profit validation (if provided)
	if !params.TakeProfit.IsZero() {
		if params.Direction == "long" && params.TakeProfit.LessThanOrEqual(params.Entry) {
			result.Warnings = append(result.Warnings, "take_profit_below_entry")
		}
		if params.Direction == "short" && params.TakeProfit.GreaterThanOrEqual(params.Entry) {
			result.Warnings = append(result.Warnings, "take_profit_above_entry")
		}
	}

	// 3. Risk/Reward ratio
	if !params.StopLoss.IsZero() && !params.TakeProfit.IsZero() {
		risk := params.Entry.Sub(params.StopLoss).Abs()
		reward := params.TakeProfit.Sub(params.Entry).Abs()

		if !risk.IsZero() {
			result.RiskReward = reward.Div(risk)

			if result.RiskReward.LessThan(v.minRiskReward) {
				result.Valid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("rr_too_low: %.2f < %.2f minimum",
						result.RiskReward.InexactFloat64(),
						v.minRiskReward.InexactFloat64(),
					))
			} else if result.RiskReward.LessThan(v.minRiskReward.Mul(decimal.NewFromFloat(1.2))) {
				// Close to minimum, warn
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("rr_marginal: %.2f", result.RiskReward.InexactFloat64()))
			}
		}
	}

	// 4. Leverage check
	if params.Leverage > v.maxLeverage {
		result.Valid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("leverage_exceeded: %dx > %dx max", params.Leverage, v.maxLeverage))
	} else if params.Leverage > int(float64(v.maxLeverage)*0.8) {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("leverage_high: %dx", params.Leverage))
	}

	// 5. Price sanity checks
	if params.Entry.LessThanOrEqual(decimal.Zero) {
		result.Valid = false
		result.Errors = append(result.Errors, "invalid_entry_price")
	}
	if params.StopLoss.LessThan(decimal.Zero) {
		result.Valid = false
		result.Errors = append(result.Errors, "invalid_stop_loss")
	}
	if params.TakeProfit.LessThan(decimal.Zero) {
		result.Valid = false
		result.Errors = append(result.Errors, "invalid_take_profit")
	}

	// 6. Direction validation
	if params.Direction != "long" && params.Direction != "short" {
		result.Valid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("invalid_direction: %s", params.Direction))
	}

	// 7. Size validation
	if params.Size.LessThanOrEqual(decimal.Zero) {
		result.Valid = false
		result.Errors = append(result.Errors, "invalid_size")
	}

	return result
}
