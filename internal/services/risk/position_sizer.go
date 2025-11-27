package riskservice

import (
	"math"
)

// PositionSizer calculates appropriate position sizes based on risk parameters
type PositionSizer struct {
	riskPerTrade      float64 // % of capital to risk per trade
	maxPositionPct    float64 // max % of portfolio per position
	minPositionUSD    float64 // minimum position size in USD
	maxPositionUSD    float64 // maximum position size in USD
	confidenceScaling bool    // scale size by confidence?
	useKelly          bool    // use Kelly criterion?
}

// CalculateSize calculates position size using configured method
func (s *PositionSizer) CalculateSize(
	plan *TradePlan,
	balance float64,
	confidence float64,
	stats *UserTradingStats, // TODO: define this type
) *SizeResult {

	var baseSize float64
	var method string

	// Calculate base size using primary method
	if s.useKelly && stats != nil && stats.SampleSize >= 20 {
		// Kelly Criterion (only with sufficient data)
		baseSize = s.kellySizing(balance, stats)
		method = "kelly"
	} else {
		// Risk-based sizing (default)
		baseSize = s.riskBasedSizing(plan, balance)
		method = "fixed_risk"
	}

	// Apply confidence scaling if enabled
	if s.confidenceScaling {
		// Scale from 50% to 100% based on confidence
		// confidence=0.5 → 50% size
		// confidence=0.7 → 70% size
		// confidence=1.0 → 100% size
		scaleFactor := math.Max(0.5, confidence)
		baseSize *= scaleFactor
	}

	// Apply maximum position limit (% of portfolio)
	maxPosition := balance * s.maxPositionPct
	finalSize := math.Min(baseSize, maxPosition)

	// Apply absolute USD limits
	sizeUSD := finalSize * plan.Entry
	if sizeUSD > s.maxPositionUSD {
		finalSize = s.maxPositionUSD / plan.Entry
		sizeUSD = s.maxPositionUSD
	}

	// Calculate risk amount
	stopDistancePct := math.Abs(plan.Entry-plan.StopLoss) / plan.Entry
	riskAmount := sizeUSD * stopDistancePct

	// Check reduction
	wasReduced := finalSize < baseSize
	reductionPct := 0.0
	if wasReduced {
		reductionPct = (1 - finalSize/baseSize) * 100
	}

	return &SizeResult{
		Size:         finalSize,
		SizeUSD:      sizeUSD,
		RiskAmount:   riskAmount,
		RiskPercent:  (riskAmount / balance) * 100,
		WasReduced:   wasReduced,
		ReductionPct: reductionPct,
		Method:       method,
	}
}

// riskBasedSizing calculates size based on fixed risk per trade
func (s *PositionSizer) riskBasedSizing(plan *TradePlan, balance float64) float64 {
	// Risk amount we're willing to lose
	riskAmount := balance * s.riskPerTrade

	// Calculate stop distance (%)
	if plan.StopLoss == 0 {
		// No stop-loss, use default (e.g., 2%)
		return balance * 0.02 / plan.Entry
	}

	stopDistancePct := math.Abs(plan.Entry-plan.StopLoss) / plan.Entry

	// Position size = risk_amount / stop_distance
	// Example: Risk $100, stop 2% away → position = $5000
	baseSize := riskAmount / stopDistancePct

	// Divide by entry price to get size in base currency
	return baseSize / plan.Entry
}

// kellySizing calculates size using Kelly Criterion
// Kelly formula: f = (bp - q) / b
// where b = avg_win/avg_loss, p = win_rate, q = 1-win_rate
func (s *PositionSizer) kellySizing(balance float64, stats *UserTradingStats) float64 {
	if stats.AvgLoss == 0 {
		// Avoid division by zero, fall back to fixed risk
		return balance * s.riskPerTrade
	}

	b := stats.AvgWin / stats.AvgLoss
	p := stats.WinRate
	q := 1 - p

	kelly := (b*p - q) / b

	// Kelly can be negative (system has negative expectancy)
	if kelly <= 0 {
		return 0
	}

	// Use fractional Kelly for safety (25% Kelly)
	fractionalKelly := kelly * 0.25

	// Cap at risk per trade limit
	fractionalKelly = math.Min(fractionalKelly, s.riskPerTrade)

	return balance * fractionalKelly
}

// UserTradingStats contains statistics for Kelly Criterion
// TODO: Move to proper domain type
type UserTradingStats struct {
	WinRate    float64 // 0-1
	AvgWin     float64 // average winning trade %
	AvgLoss    float64 // average losing trade %
	SampleSize int     // number of trades
}
