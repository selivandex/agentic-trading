package risk

import (
	"github.com/shopspring/decimal"
)

// PositionSizer calculates appropriate position sizes based on risk parameters
type PositionSizer struct {
	riskPerTrade         decimal.Decimal // % of capital to risk per trade (e.g., 0.01 = 1%)
	maxPositionPct       decimal.Decimal // max % of portfolio per position (e.g., 0.10 = 10%)
	minPositionUSD       decimal.Decimal // minimum position size in USD
	maxPositionUSD       decimal.Decimal // maximum position size in USD
	useConfidenceScaling bool            // scale size by signal confidence?
}

// SizingInput contains parameters for position sizing calculation
type SizingInput struct {
	AvailableBalance decimal.Decimal // available capital
	Entry            decimal.Decimal // entry price
	StopLoss         decimal.Decimal // stop-loss price
	Confidence       decimal.Decimal // 0-1, from opportunity signal
	Leverage         int             // leverage to apply
}

// SizingResult contains calculated position size
type SizingResult struct {
	Size         decimal.Decimal // position size in base currency
	SizeUSD      decimal.Decimal // position size in USD
	RiskAmount   decimal.Decimal // dollar amount at risk
	RiskPercent  decimal.Decimal // % of portfolio at risk
	WasReduced   bool            // was size reduced due to limits?
	ReductionPct decimal.Decimal // by how much (%)
	Method       string          // sizing method used
}

// NewPositionSizer creates a new position sizer
func NewPositionSizer(
	riskPerTrade float64,
	maxPositionPct float64,
	minPositionUSD float64,
	maxPositionUSD float64,
	useConfidenceScaling bool,
) *PositionSizer {
	return &PositionSizer{
		riskPerTrade:         decimal.NewFromFloat(riskPerTrade),
		maxPositionPct:       decimal.NewFromFloat(maxPositionPct),
		minPositionUSD:       decimal.NewFromFloat(minPositionUSD),
		maxPositionUSD:       decimal.NewFromFloat(maxPositionUSD),
		useConfidenceScaling: useConfidenceScaling,
	}
}

// CalculateSize calculates appropriate position size using risk-based method
func (s *PositionSizer) CalculateSize(input *SizingInput) *SizingResult {
	// Calculate base size using risk-based sizing
	baseSize := s.riskBasedSizing(input)

	// Apply confidence scaling if enabled
	if s.useConfidenceScaling && !input.Confidence.IsZero() {
		// Scale from 50% to 100% based on confidence
		// confidence=0.5 → 50% size
		// confidence=0.7 → 70% size
		// confidence=1.0 → 100% size
		scaleFactor := input.Confidence
		if scaleFactor.LessThan(decimal.NewFromFloat(0.5)) {
			scaleFactor = decimal.NewFromFloat(0.5) // minimum 50%
		}
		baseSize = baseSize.Mul(scaleFactor)
	}

	// Apply maximum position limit (% of portfolio)
	maxPosition := input.AvailableBalance.Mul(s.maxPositionPct)
	finalSize := baseSize
	if baseSize.GreaterThan(maxPosition) {
		finalSize = maxPosition
	}

	// Calculate size in USD
	sizeUSD := finalSize.Mul(input.Entry)

	// Apply absolute USD limits
	if sizeUSD.GreaterThan(s.maxPositionUSD) {
		finalSize = s.maxPositionUSD.Div(input.Entry)
		sizeUSD = s.maxPositionUSD
	}

	// Calculate risk amount
	stopDistancePct := input.Entry.Sub(input.StopLoss).Abs().Div(input.Entry)
	riskAmount := sizeUSD.Mul(stopDistancePct)

	// Check if size was reduced
	wasReduced := finalSize.LessThan(baseSize)
	reductionPct := decimal.Zero
	if wasReduced && !baseSize.IsZero() {
		reductionPct = decimal.NewFromInt(1).Sub(finalSize.Div(baseSize)).Mul(decimal.NewFromInt(100))
	}

	// Calculate risk percent
	riskPercent := decimal.Zero
	if !input.AvailableBalance.IsZero() {
		riskPercent = riskAmount.Div(input.AvailableBalance).Mul(decimal.NewFromInt(100))
	}

	return &SizingResult{
		Size:         finalSize,
		SizeUSD:      sizeUSD,
		RiskAmount:   riskAmount,
		RiskPercent:  riskPercent,
		WasReduced:   wasReduced,
		ReductionPct: reductionPct,
		Method:       "risk_based",
	}
}

// riskBasedSizing calculates size based on fixed risk per trade
// Formula: Position Size = (Risk Amount / Stop Distance %)
func (s *PositionSizer) riskBasedSizing(input *SizingInput) decimal.Decimal {
	// Risk amount we're willing to lose
	riskAmount := input.AvailableBalance.Mul(s.riskPerTrade)

	// Calculate stop distance (%)
	if input.StopLoss.IsZero() {
		// No stop-loss provided, use default 2% risk distance
		return riskAmount.Div(decimal.NewFromFloat(0.02))
	}

	stopDistancePct := input.Entry.Sub(input.StopLoss).Abs().Div(input.Entry)

	if stopDistancePct.IsZero() {
		// Stop is at entry (edge case), use default
		return riskAmount.Div(decimal.NewFromFloat(0.02))
	}

	// Position size = risk_amount / stop_distance_pct
	// Example: Risk $100, stop 2% away → position value = $5000
	positionValue := riskAmount.Div(stopDistancePct)

	// Convert to size in base currency
	return positionValue.Div(input.Entry)
}

// CalculateKellySizeWithStats calculates size using Kelly Criterion with trading stats
// Kelly formula: f = (bp - q) / b
// where b = avg_win/avg_loss, p = win_rate, q = 1-win_rate
// Only use Kelly if:
// - Sufficient sample size (20+ trades)
// - Positive expectancy
// - Otherwise falls back to risk-based sizing
func (s *PositionSizer) CalculateKellySizeWithStats(
	input *SizingInput,
	stats *TradingStats,
) *SizingResult {
	// Check if we have sufficient data
	if stats == nil || !stats.HasSufficientData() {
		// Not enough data, use risk-based sizing
		result := s.CalculateSize(input)
		result.Method = "risk_based_insufficient_kelly_data"
		return result
	}

	// Check if system has positive expectancy
	if !stats.IsPositiveExpectancy() {
		// Negative expectancy, don't trade!
		result := &SizingResult{
			Size:         decimal.Zero,
			SizeUSD:      decimal.Zero,
			RiskAmount:   decimal.Zero,
			RiskPercent:  decimal.Zero,
			WasReduced:   true,
			ReductionPct: decimal.NewFromInt(100),
			Method:       "kelly_negative_expectancy",
		}
		return result
	}

	// Avoid division by zero
	if stats.AvgLoss.IsZero() {
		return s.CalculateSize(input)
	}

	// Calculate Kelly %
	// Kelly = (WinRate * AvgWin - (1-WinRate) * AvgLoss) / AvgWin
	// Simplified: kelly = (bp - q) / b where b = AvgWin/AvgLoss
	b := stats.AvgWin.Div(stats.AvgLoss)
	p := stats.WinRate
	q := decimal.NewFromInt(1).Sub(p)

	kelly := b.Mul(p).Sub(q).Div(b)

	// Kelly can be negative (should not happen if expectancy is positive, but safety check)
	if kelly.LessThanOrEqual(decimal.Zero) {
		// Fall back to risk-based
		result := s.CalculateSize(input)
		result.Method = "risk_based_kelly_negative"
		return result
	}

	// Use fractional Kelly for safety (25% Kelly - conservative)
	// Full Kelly can lead to large drawdowns
	fractionalKelly := kelly.Mul(decimal.NewFromFloat(0.25))

	// Cap at maximum risk per trade setting
	if fractionalKelly.GreaterThan(s.riskPerTrade) {
		fractionalKelly = s.riskPerTrade
	}

	// Calculate position value based on Kelly %
	positionValue := input.AvailableBalance.Mul(fractionalKelly)
	size := positionValue.Div(input.Entry)

	// Apply max position limit
	maxPosition := input.AvailableBalance.Mul(s.maxPositionPct)
	if positionValue.GreaterThan(maxPosition) {
		size = maxPosition.Div(input.Entry)
		positionValue = maxPosition
	}

	sizeUSD := size.Mul(input.Entry)

	// Apply absolute USD limits
	if sizeUSD.GreaterThan(s.maxPositionUSD) {
		size = s.maxPositionUSD.Div(input.Entry)
		sizeUSD = s.maxPositionUSD
	}

	// Calculate risk metrics
	stopDistancePct := decimal.Zero
	if !input.StopLoss.IsZero() {
		stopDistancePct = input.Entry.Sub(input.StopLoss).Abs().Div(input.Entry)
	}
	riskAmount := sizeUSD.Mul(stopDistancePct)

	riskPercent := decimal.Zero
	if !input.AvailableBalance.IsZero() {
		riskPercent = riskAmount.Div(input.AvailableBalance).Mul(decimal.NewFromInt(100))
	}

	return &SizingResult{
		Size:         size,
		SizeUSD:      sizeUSD,
		RiskAmount:   riskAmount,
		RiskPercent:  riskPercent,
		WasReduced:   false,
		ReductionPct: decimal.Zero,
		Method:       "kelly_criterion_25pct",
	}
}
