package liquidation

import "time"

// Liquidation represents a liquidation event
type Liquidation struct {
	Exchange  string    `ch:"exchange"`
	Symbol    string    `ch:"symbol"`
	Timestamp time.Time `ch:"timestamp"`

	Side     string  `ch:"side"` // long, short
	Price    float64 `ch:"price"`
	Quantity float64 `ch:"quantity"`
	Value    float64 `ch:"value"` // USD
}

// LiquidationHeatmap represents liquidation levels
type LiquidationHeatmap struct {
	Symbol    string             `json:"symbol"`
	Timestamp time.Time          `json:"timestamp"`
	Levels    []LiquidationLevel `json:"levels"`
}

// LiquidationLevel represents a price level with liquidation data
type LiquidationLevel struct {
	Price         float64 `json:"price"`
	LongLiqValue  float64 `json:"long_liq_value"` // USD to be liquidated
	ShortLiqValue float64 `json:"short_liq_value"`
	Leverage      int     `json:"leverage"` // Estimated leverage
}
