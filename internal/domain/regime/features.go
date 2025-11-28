package regime

import "time"

// Features represents extracted regime features for ML model input
type Features struct {
	Symbol    string    `ch:"symbol" json:"symbol"`
	Timestamp time.Time `ch:"timestamp" json:"timestamp"`

	// Volatility (4 features)
	ATR14         float64 `ch:"atr_14" json:"atr_14"`
	ATRPct        float64 `ch:"atr_pct" json:"atr_pct"`
	BBWidth       float64 `ch:"bb_width" json:"bb_width"`
	HistoricalVol float64 `ch:"historical_vol" json:"historical_vol"`

	// Trend (8 features)
	ADX              float64 `ch:"adx" json:"adx"`
	EMA9             float64 `ch:"ema_9" json:"ema_9"`
	EMA21            float64 `ch:"ema_21" json:"ema_21"`
	EMA55            float64 `ch:"ema_55" json:"ema_55"`
	EMA200           float64 `ch:"ema_200" json:"ema_200"`
	EMAAlignment     string  `ch:"ema_alignment" json:"ema_alignment"`
	HigherHighsCount uint8   `ch:"higher_highs_count" json:"higher_highs_count"`
	LowerLowsCount   uint8   `ch:"lower_lows_count" json:"lower_lows_count"`

	// Volume (3 features)
	Volume24h             float64 `ch:"volume_24h" json:"volume_24h"`
	VolumeChangePct       float64 `ch:"volume_change_pct" json:"volume_change_pct"`
	VolumePriceDivergence float64 `ch:"volume_price_divergence" json:"volume_price_divergence"`

	// Structure (3 features)
	SupportBreaks        uint8 `ch:"support_breaks" json:"support_breaks"`
	ResistanceBreaks     uint8 `ch:"resistance_breaks" json:"resistance_breaks"`
	ConsolidationPeriods uint8 `ch:"consolidation_periods" json:"consolidation_periods"`

	// Cross-asset (2 features)
	BTCDominance         float64 `ch:"btc_dominance" json:"btc_dominance"`
	CorrelationTightness float64 `ch:"correlation_tightness" json:"correlation_tightness"`

	// Derivatives (3 features)
	FundingRate     float64 `ch:"funding_rate" json:"funding_rate"`
	FundingRateMA   float64 `ch:"funding_rate_ma" json:"funding_rate_ma"`
	Liquidations24h float64 `ch:"liquidations_24h" json:"liquidations_24h"`

	// Training label (manual, for offline training only)
	RegimeLabel string `ch:"regime_label" json:"regime_label,omitempty"`
}

// ToFeatureVector converts Features to float64 slice for ML model input
// Order must match training script feature order (22 features total)
func (f *Features) ToFeatureVector() []float64 {
	return []float64{
		// Volatility (4)
		f.ATR14, f.ATRPct, f.BBWidth, f.HistoricalVol,
		// Trend (8)
		f.ADX, f.EMA9, f.EMA21, f.EMA55, f.EMA200,
		float64(f.HigherHighsCount), float64(f.LowerLowsCount),
		// Volume (3)
		f.Volume24h, f.VolumeChangePct, f.VolumePriceDivergence,
		// Structure (3)
		float64(f.SupportBreaks), float64(f.ResistanceBreaks), float64(f.ConsolidationPeriods),
		// Cross-asset (2)
		f.BTCDominance, f.CorrelationTightness,
		// Derivatives (3)
		f.FundingRate, f.FundingRateMA, f.Liquidations24h,
	}
}
