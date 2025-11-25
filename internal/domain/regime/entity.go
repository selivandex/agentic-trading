package regime

import "time"

// MarketRegime represents detected market regime
type MarketRegime struct {
	Symbol     string     `db:"symbol"`
	Timestamp  time.Time  `db:"timestamp"`
	Regime     RegimeType `db:"regime"`
	Confidence float64    `db:"confidence"`
	Volatility VolLevel   `db:"volatility"`
	Trend      TrendType  `db:"trend"`

	// Supporting metrics
	ATR14        float64 `db:"atr_14"`
	ADX          float64 `db:"adx"`
	BBWidth      float64 `db:"bb_width"`
	Volume24h    float64 `db:"volume_24h"`
	VolumeChange float64 `db:"volume_change"`
}

// RegimeType defines market regime types
type RegimeType string

const (
	RegimeTrendUp      RegimeType = "trend_up"
	RegimeTrendDown    RegimeType = "trend_down"
	RegimeRange        RegimeType = "range"
	RegimeBreakout     RegimeType = "breakout"
	RegimeVolatile     RegimeType = "volatile"
	RegimeAccumulation RegimeType = "accumulation"
	RegimeDistribution RegimeType = "distribution"
)

// Valid checks if regime type is valid
func (r RegimeType) Valid() bool {
	switch r {
	case RegimeTrendUp, RegimeTrendDown, RegimeRange, RegimeBreakout,
		RegimeVolatile, RegimeAccumulation, RegimeDistribution:
		return true
	}
	return false
}

// String returns string representation
func (r RegimeType) String() string {
	return string(r)
}

// VolLevel defines volatility levels
type VolLevel string

const (
	VolLow     VolLevel = "low"
	VolMedium  VolLevel = "medium"
	VolHigh    VolLevel = "high"
	VolExtreme VolLevel = "extreme"
)

// Valid checks if volatility level is valid
func (v VolLevel) Valid() bool {
	switch v {
	case VolLow, VolMedium, VolHigh, VolExtreme:
		return true
	}
	return false
}

// String returns string representation
func (v VolLevel) String() string {
	return string(v)
}

// TrendType defines trend direction
type TrendType string

const (
	TrendBullish TrendType = "bullish"
	TrendBearish TrendType = "bearish"
	TrendNeutral TrendType = "neutral"
)

// Valid checks if trend type is valid
func (t TrendType) Valid() bool {
	switch t {
	case TrendBullish, TrendBearish, TrendNeutral:
		return true
	}
	return false
}

// String returns string representation
func (t TrendType) String() string {
	return string(t)
}
