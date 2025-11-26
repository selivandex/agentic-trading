package analysis

import (
	"context"
	"math"
	"time"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/events"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// abs returns absolute value of a float64
func abs(x float64) float64 {
	return math.Abs(x)
}

// SMCScanner scans for Smart Money Concepts patterns:
// - Fair Value Gaps (FVG)
// - Order Blocks (OB)
// - Liquidity zones
// - Break of Structure (BOS)
// - Change of Character (CHoCH)
type SMCScanner struct {
	*workers.BaseWorker
	mdRepo         market_data.Repository
	kafka          *kafka.Producer
	eventPublisher *events.WorkerPublisher
	symbols        []string
}

// NewSMCScanner creates a new SMC scanner worker
func NewSMCScanner(
	mdRepo market_data.Repository,
	kafka *kafka.Producer,
	symbols []string,
	interval time.Duration,
	enabled bool,
) *SMCScanner {
	return &SMCScanner{
		BaseWorker:     workers.NewBaseWorker("smc_scanner", interval, enabled),
		mdRepo:         mdRepo,
		kafka:          kafka,
		eventPublisher: events.NewWorkerPublisher(kafka),
		symbols:        symbols,
	}
}

// Run executes one iteration of SMC pattern scanning
func (ss *SMCScanner) Run(ctx context.Context) error {
	ss.Log().Debug("SMC scanner: starting iteration")

	if len(ss.symbols) == 0 {
		ss.Log().Warn("No symbols configured for SMC scanning")
		return nil
	}

	totalPatterns := 0

	for _, symbol := range ss.symbols {
		patterns, err := ss.scanSymbol(ctx, symbol)
		if err != nil {
			ss.Log().Error("Failed to scan symbol",
				"symbol", symbol,
				"error", err,
			)
			continue
		}

		totalPatterns += patterns
	}

	ss.Log().Debug("SMC scanning complete", "total_patterns", totalPatterns)

	return nil
}

// scanSymbol scans a single symbol for SMC patterns
func (ss *SMCScanner) scanSymbol(ctx context.Context, symbol string) (int, error) {
	// Get recent OHLCV data (last 100 candles of 15m for pattern detection)
	candles, err := ss.mdRepo.GetOHLCV(ctx, market_data.OHLCVQuery{
		Symbol:    symbol,
		Timeframe: "15m",
		Limit:     100,
	})
	if err != nil {
		return 0, errors.Wrap(err, "failed to get OHLCV data")
	}

	if len(candles) < 10 {
		return 0, nil // Not enough data
	}

	patternsFound := 0

	// Detect Fair Value Gaps
	fvgs := ss.detectFVG(candles)
	if len(fvgs) > 0 {
		ss.Log().Info("Fair Value Gaps detected",
			"symbol", symbol,
			"count", len(fvgs),
		)
		for _, fvg := range fvgs {
			ss.publishFVGEvent(ctx, symbol, fvg)
		}
		patternsFound += len(fvgs)
	}

	// Detect Order Blocks
	orderBlocks := ss.detectOrderBlocks(candles)
	if len(orderBlocks) > 0 {
		ss.Log().Info("Order Blocks detected",
			"symbol", symbol,
			"count", len(orderBlocks),
		)
		for _, ob := range orderBlocks {
			ss.publishOrderBlockEvent(ctx, symbol, ob)
		}
		patternsFound += len(orderBlocks)
	}

	// TODO: Add more pattern detection:
	// - Liquidity zones
	// - Break of Structure (BOS)
	// - Change of Character (CHoCH)
	// - Imbalances
	// - Displacement moves

	return patternsFound, nil
}

// detectFVG detects Fair Value Gaps
// FVG occurs when there's a gap between candle highs/lows
// indicating an area that price may revisit
func (ss *SMCScanner) detectFVG(candles []market_data.OHLCV) []FairValueGap {
	if len(candles) < 3 {
		return nil
	}

	var fvgs []FairValueGap

	// Scan through candles looking for 3-candle FVG pattern
	for i := 2; i < len(candles); i++ {
		prev := candles[i-2]
		curr := candles[i-1]
		next := candles[i]

		// Bullish FVG: gap up
		// Current candle low > previous candle high
		if curr.Low > prev.High {
			gap := curr.Low - prev.High
			gapPct := (gap / prev.High) * 100

			// Only consider significant gaps (> 0.5%)
			if gapPct > 0.5 {
				fvg := FairValueGap{
					Type:        "bullish",
					StartTime:   prev.OpenTime,
					EndTime:     curr.OpenTime,
					TopPrice:    curr.Low,
					BottomPrice: prev.High,
					Gap:         gap,
					GapPercent:  gapPct,
					Filled:      false,
				}

				// Check if gap was filled by next candle
				if next.Low <= prev.High {
					fvg.Filled = true
				}

				fvgs = append(fvgs, fvg)
			}
		}

		// Bearish FVG: gap down
		// Current candle high < previous candle low
		if curr.High < prev.Low {
			gap := prev.Low - curr.High
			gapPct := (gap / prev.Low) * 100

			if gapPct > 0.5 {
				fvg := FairValueGap{
					Type:        "bearish",
					StartTime:   prev.OpenTime,
					EndTime:     curr.OpenTime,
					TopPrice:    prev.Low,
					BottomPrice: curr.High,
					Gap:         gap,
					GapPercent:  gapPct,
					Filled:      false,
				}

				// Check if gap was filled
				if next.High >= prev.Low {
					fvg.Filled = true
				}

				fvgs = append(fvgs, fvg)
			}
		}
	}

	return fvgs
}

// detectOrderBlocks detects Order Blocks
// OB is the last bullish/bearish candle before a strong move in opposite direction
// Indicates institutional order placement
func (ss *SMCScanner) detectOrderBlocks(candles []market_data.OHLCV) []OrderBlock {
	if len(candles) < 5 {
		return nil
	}

	var orderBlocks []OrderBlock

	// Scan through candles looking for strong reversals
	for i := 3; i < len(candles)-1; i++ {
		curr := candles[i]
		next := candles[i+1]

		// Calculate move strength
		moveSize := abs(next.Close - next.Open)
		movePct := (moveSize / curr.Close) * 100

		// Strong bullish move (> 1.5%)
		if next.Close > next.Open && movePct > 1.5 {
			// Find last bearish candle before the move
			for j := i; j >= 0; j-- {
				if candles[j].Close < candles[j].Open {
					ob := OrderBlock{
						Type:        "bullish",
						Timestamp:   candles[j].OpenTime,
						TopPrice:    candles[j].Open,
						BottomPrice: candles[j].Close,
						Strength:    movePct,
					}
					orderBlocks = append(orderBlocks, ob)
					break
				}
			}
		}

		// Strong bearish move
		if next.Close < next.Open && movePct > 1.5 {
			// Find last bullish candle before the move
			for j := i; j >= 0; j-- {
				if candles[j].Close > candles[j].Open {
					ob := OrderBlock{
						Type:        "bearish",
						Timestamp:   candles[j].OpenTime,
						TopPrice:    candles[j].Close,
						BottomPrice: candles[j].Open,
						Strength:    movePct,
					}
					orderBlocks = append(orderBlocks, ob)
					break
				}
			}
		}
	}

	return orderBlocks
}

// Pattern structures

type FairValueGap struct {
	Type        string // bullish, bearish
	StartTime   time.Time
	EndTime     time.Time
	TopPrice    float64
	BottomPrice float64
	Gap         float64
	GapPercent  float64
	Filled      bool
}

type OrderBlock struct {
	Type        string // bullish, bearish
	Timestamp   time.Time
	TopPrice    float64
	BottomPrice float64
	Strength    float64 // Strength of the move that followed
}

// Event structures

type FVGEvent struct {
	Symbol      string    `json:"symbol"`
	Type        string    `json:"type"` // bullish, bearish
	TopPrice    float64   `json:"top_price"`
	BottomPrice float64   `json:"bottom_price"`
	GapPercent  float64   `json:"gap_percent"`
	Filled      bool      `json:"filled"`
	DetectedAt  time.Time `json:"detected_at"`
}

type OrderBlockEvent struct {
	Symbol      string    `json:"symbol"`
	Type        string    `json:"type"` // bullish, bearish
	TopPrice    float64   `json:"top_price"`
	BottomPrice float64   `json:"bottom_price"`
	Strength    float64   `json:"strength"`
	DetectedAt  time.Time `json:"detected_at"`
}

func (ss *SMCScanner) publishFVGEvent(ctx context.Context, symbol string, fvg FairValueGap) {
	if err := ss.eventPublisher.PublishFVGDetected(
		ctx,
		symbol,
		fvg.Type,
		fvg.TopPrice,
		fvg.BottomPrice,
		fvg.GapPercent,
		fvg.Filled,
	); err != nil {
		ss.Log().Error("Failed to publish FVG event", "error", err)
	}
}

func (ss *SMCScanner) publishOrderBlockEvent(ctx context.Context, symbol string, ob OrderBlock) {
	if err := ss.eventPublisher.PublishOrderBlockDetected(
		ctx,
		symbol,
		ob.Type,
		ob.TopPrice,
		ob.BottomPrice,
		ob.Strength,
	); err != nil {
		ss.Log().Error("Failed to publish Order Block event", "error", err)
	}
}
