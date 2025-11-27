package risk

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// MarginCalculator calculates margin requirements for leveraged positions
type MarginCalculator struct {
	maxMarginUsagePct decimal.Decimal // max % of margin to use (e.g., 0.8 = 80%)
	maintenanceBuffer decimal.Decimal // buffer above maintenance margin (e.g., 0.2 = 20%)
}

// MarginInput contains parameters for margin calculation
type MarginInput struct {
	Size             decimal.Decimal // position size
	Entry            decimal.Decimal // entry price
	Leverage         int             // leverage (e.g., 5x)
	AvailableBalance decimal.Decimal // available balance
	UsedMargin       decimal.Decimal // currently used margin
}

// MarginResult contains margin calculation result
type MarginResult struct {
	Required        decimal.Decimal // margin required for this trade
	Available       decimal.Decimal // margin available
	UsageAfterTrade decimal.Decimal // % of margin used after trade
	Sufficient      bool            // is margin sufficient?
	Message         string          // explanation
}

// NewMarginCalculator creates a new margin calculator
func NewMarginCalculator(maxMarginUsagePct float64, maintenanceBuffer float64) *MarginCalculator {
	return &MarginCalculator{
		maxMarginUsagePct: decimal.NewFromFloat(maxMarginUsagePct),
		maintenanceBuffer: decimal.NewFromFloat(maintenanceBuffer),
	}
}

// Check validates margin requirements for a leveraged trade
func (c *MarginCalculator) Check(input *MarginInput) *MarginResult {
	result := &MarginResult{
		Available: input.AvailableBalance.Sub(input.UsedMargin),
	}

	// Calculate required margin for this position
	// Formula: Required Margin = (Size * Entry Price) / Leverage
	positionValue := input.Size.Mul(input.Entry)

	if input.Leverage <= 1 {
		// Spot trading (no leverage)
		result.Required = positionValue
	} else {
		result.Required = positionValue.Div(decimal.NewFromInt(int64(input.Leverage)))

		// Add maintenance buffer to avoid liquidation
		result.Required = result.Required.Mul(
			decimal.NewFromInt(1).Add(c.maintenanceBuffer),
		)
	}

	// Check if sufficient margin available
	if result.Required.GreaterThan(result.Available) {
		result.Sufficient = false
		result.Message = fmt.Sprintf(
			"insufficient_margin: need %s, have %s",
			result.Required.StringFixed(2),
			result.Available.StringFixed(2),
		)
		return result
	}

	// Calculate margin usage after this trade
	totalUsedAfter := input.UsedMargin.Add(result.Required)
	if !input.AvailableBalance.IsZero() {
		result.UsageAfterTrade = totalUsedAfter.Div(input.AvailableBalance).Mul(decimal.NewFromInt(100))
	}

	// Check if margin usage exceeds limit
	if result.UsageAfterTrade.GreaterThan(c.maxMarginUsagePct.Mul(decimal.NewFromInt(100))) {
		result.Sufficient = false
		result.Message = fmt.Sprintf(
			"margin_usage_exceeded: %.1f%% > %.1f%% limit",
			result.UsageAfterTrade.InexactFloat64(),
			c.maxMarginUsagePct.Mul(decimal.NewFromInt(100)).InexactFloat64(),
		)
		return result
	}

	result.Sufficient = true
	result.Message = fmt.Sprintf(
		"margin_ok: %.1f%% usage",
		result.UsageAfterTrade.InexactFloat64(),
	)

	return result
}

// CalculateLiquidationPrice calculates liquidation price for a position
func (c *MarginCalculator) CalculateLiquidationPrice(
	entry decimal.Decimal,
	leverage int,
	direction string,
	maintenanceMarginRate decimal.Decimal, // e.g., 0.005 = 0.5%
) decimal.Decimal {
	// Liquidation price formula (simplified):
	// Long: Liq = Entry * (1 - 1/Leverage + MMR)
	// Short: Liq = Entry * (1 + 1/Leverage - MMR)

	leverageDec := decimal.NewFromInt(int64(leverage))
	one := decimal.NewFromInt(1)

	if direction == "long" {
		// Long liquidation
		factor := one.Sub(one.Div(leverageDec)).Add(maintenanceMarginRate)
		return entry.Mul(factor)
	}

	// Short liquidation
	factor := one.Add(one.Div(leverageDec)).Sub(maintenanceMarginRate)
	return entry.Mul(factor)
}
