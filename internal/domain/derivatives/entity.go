package derivatives

import "time"

// OptionsSnapshot represents options market data at a point in time
type OptionsSnapshot struct {
	Symbol    string    `ch:"symbol"`
	Timestamp time.Time `ch:"timestamp"`

	// Open Interest
	CallOI       float64 `ch:"call_oi"`
	PutOI        float64 `ch:"put_oi"`
	TotalOI      float64 `ch:"total_oi"`
	PutCallRatio float64 `ch:"put_call_ratio"`

	// Max Pain
	MaxPainPrice float64 `ch:"max_pain_price"`
	MaxPainDelta float64 `ch:"max_pain_delta"` // % from current price

	// Gamma
	GammaExposure float64 `ch:"gamma_exposure"`
	GammaFlip     float64 `ch:"gamma_flip"` // Price where gamma flips

	// IV
	IVIndex   float64 `ch:"iv_index"`
	IV25dCall float64 `ch:"iv_25d_call"`
	IV25dPut  float64 `ch:"iv_25d_put"`
	IVSkew    float64 `ch:"iv_skew"`

	// Large trades
	LargeCallVol float64 `ch:"large_call_vol"` // $1M+ trades
	LargePutVol  float64 `ch:"large_put_vol"`
}

// OptionsFlow represents a large options trade
type OptionsFlow struct {
	ID        string    `ch:"id"`
	Timestamp time.Time `ch:"timestamp"`
	Symbol    string    `ch:"symbol"`

	Side    string    `ch:"side"` // call, put
	Strike  float64   `ch:"strike"`
	Expiry  time.Time `ch:"expiry"`
	Premium float64   `ch:"premium"` // USD value
	Size    int       `ch:"size"`    // contracts
	Spot    float64   `ch:"spot"`    // Spot price at time

	TradeType string `ch:"trade_type"` // buy, sell
	Sentiment string `ch:"sentiment"`  // bullish, bearish
}
