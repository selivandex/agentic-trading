package derivatives

import (
	"context"
	"math"
	"time"

	"prometheus/internal/domain/derivatives"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// GammaExposureCollector calculates gamma exposure (GEX) from options chains
// Monitors dealer positioning and gamma flip levels
type GammaExposureCollector struct {
	*workers.BaseWorker
	derivRepo derivatives.Repository
	apiKey    string
	symbols   []string
}

// NewGammaExposureCollector creates a new gamma exposure collector
func NewGammaExposureCollector(
	derivRepo derivatives.Repository,
	deribitAPIKey string,
	symbols []string,
	interval time.Duration,
	enabled bool,
) *GammaExposureCollector {
	return &GammaExposureCollector{
		BaseWorker: workers.NewBaseWorker("gamma_exposure_collector", interval, enabled),
		derivRepo:  derivRepo,
		apiKey:     deribitAPIKey,
		symbols:    symbols,
	}
}

// Run executes one iteration of gamma exposure collection
func (gc *GammaExposureCollector) Run(ctx context.Context) error {
	gc.Log().Debug("Gamma exposure collector: starting iteration")

	if gc.apiKey == "" {
		gc.Log().Debug("Deribit API key not configured, skipping iteration")
		return nil
	}

	// Calculate gamma exposure for each symbol
	processedCount := 0
	for _, symbol := range gc.symbols {
		// Check for context cancellation (graceful shutdown)
		select {
		case <-ctx.Done():
			gc.Log().Info("Gamma exposure collector interrupted by shutdown",
				"symbols_processed", processedCount,
				"symbols_remaining", len(gc.symbols)-processedCount,
			)
			return ctx.Err()
		default:
		}

		snapshot, err := gc.calculateGammaExposure(ctx, symbol)
		if err != nil {
			gc.Log().Error("Failed to calculate gamma exposure",
				"symbol", symbol,
				"error", err,
			)
			processedCount++
			continue
		}

		if snapshot == nil {
			processedCount++
			continue
		}

		// Store snapshot
		if err := gc.derivRepo.InsertOptionsSnapshot(ctx, snapshot); err != nil {
			gc.Log().Error("Failed to save gamma snapshot",
				"symbol", symbol,
				"error", err,
			)
			processedCount++
			continue
		}

		gc.Log().Debug("Gamma exposure calculated",
			"symbol", symbol,
			"gex", snapshot.GammaExposure,
			"gamma_flip", snapshot.GammaFlip,
		)
		processedCount++
	}

	gc.Log().Info("Gamma exposure collection complete", "symbols", len(gc.symbols))

	return nil
}

// Deribit options book response (placeholder types - not currently used)
// These types would be used if fetching from Deribit API directly
type deribitBookResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  deribitBookData `json:"result"`
}

type deribitBookData struct {
	Asks            [][]float64  `json:"asks"`
	Bids            [][]float64  `json:"bids"`
	UnderlyingPrice float64      `json:"underlying_price"`
	Stats           deribitStats `json:"stats"`
}

type deribitStats struct {
	Volume       float64 `json:"volume"`
	PriceChange  float64 `json:"price_change"`
	OpenInterest float64 `json:"open_interest"`
	High         float64 `json:"high"`
	Low          float64 `json:"low"`
}

// calculateGammaExposure calculates gamma exposure from the options chain
func (gc *GammaExposureCollector) calculateGammaExposure(ctx context.Context, symbol string) (*derivatives.OptionsSnapshot, error) {
	baseSymbol := symbol
	if symbol == "BTC/USDT" {
		baseSymbol = "BTC"
	} else if symbol == "ETH/USDT" {
		baseSymbol = "ETH"
	}

	// Fetch options instruments
	instruments, err := gc.fetchOptionsInstruments(ctx, baseSymbol)
	if err != nil {
		return nil, errors.Wrap(err, "fetch options instruments")
	}

	if len(instruments) == 0 {
		return nil, nil
	}

	// Get current spot price (from first instrument)
	spotPrice := gc.getSpotPrice(instruments)

	// Calculate gamma exposure across all strikes
	callOI, putOI, gammaExposure, gammaFlip := gc.calculateGEXFromInstruments(instruments, spotPrice)

	// Calculate max pain
	maxPain := gc.calculateMaxPain(instruments, spotPrice)

	// Calculate IV metrics
	ivIndex, iv25Call, iv25Put, ivSkew := gc.calculateIVMetrics(instruments, spotPrice)

	snapshot := &derivatives.OptionsSnapshot{
		Symbol:        symbol,
		Timestamp:     time.Now(),
		CallOI:        callOI,
		PutOI:         putOI,
		TotalOI:       callOI + putOI,
		PutCallRatio:  putOI / callOI,
		MaxPainPrice:  maxPain,
		MaxPainDelta:  ((maxPain - spotPrice) / spotPrice) * 100,
		GammaExposure: gammaExposure,
		GammaFlip:     gammaFlip,
		IVIndex:       ivIndex,
		IV25dCall:     iv25Call,
		IV25dPut:      iv25Put,
		IVSkew:        ivSkew,
		LargeCallVol:  0, // Would need trade data
		LargePutVol:   0,
	}

	return snapshot, nil
}

// optionsInstrument represents a single options contract
type optionsInstrument struct {
	name   string
	side   string // call/put
	strike float64
	expiry time.Time
	oi     float64 // open interest
	volume float64
	price  float64
	iv     float64 // implied volatility
	spot   float64
}

// fetchOptionsInstruments fetches all options instruments for a symbol
func (gc *GammaExposureCollector) fetchOptionsInstruments(ctx context.Context, symbol string) ([]optionsInstrument, error) {
	// TODO: Implement actual Deribit API integration to fetch instruments and greeks
	// For now, return empty slice until API integration is complete
	gc.Log().Debug("Deribit instruments API integration not yet complete", "symbol", symbol)
	return []optionsInstrument{}, nil
}

// getSpotPrice extracts spot price from instruments
func (gc *GammaExposureCollector) getSpotPrice(instruments []optionsInstrument) float64 {
	if len(instruments) == 0 {
		return 0
	}
	return instruments[0].spot
}

// calculateGEXFromInstruments calculates gamma exposure across the chain
func (gc *GammaExposureCollector) calculateGEXFromInstruments(instruments []optionsInstrument, spotPrice float64) (float64, float64, float64, float64) {
	var callOI, putOI, totalGamma float64
	gammaByStrike := make(map[float64]float64)

	for _, inst := range instruments {
		if inst.side == "call" {
			callOI += inst.oi
		} else {
			putOI += inst.oi
		}

		// Calculate gamma for this contract
		// Simplified Black-Scholes gamma calculation
		gamma := gc.calculateGamma(inst.strike, spotPrice, inst.expiry, inst.iv)

		// Gamma exposure = OI * gamma * spot^2 / 100
		// Dealers are short gamma, so we flip the sign
		gex := -inst.oi * gamma * spotPrice * spotPrice / 100

		totalGamma += gex
		gammaByStrike[inst.strike] += gex
	}

	// Find gamma flip level (where gamma changes sign)
	gammaFlip := gc.findGammaFlip(gammaByStrike, spotPrice)

	return callOI, putOI, totalGamma, gammaFlip
}

// calculateGamma calculates gamma using simplified Black-Scholes
func (gc *GammaExposureCollector) calculateGamma(strike, spot float64, expiry time.Time, iv float64) float64 {
	// Time to expiry in years
	t := time.Until(expiry).Hours() / (24 * 365)
	if t <= 0 {
		return 0
	}

	// Log moneyness
	d1 := (math.Log(spot/strike) + (0.5*iv*iv)*t) / (iv * math.Sqrt(t))

	// Gamma = N'(d1) / (spot * iv * sqrt(t))
	// where N'(d1) is the standard normal PDF
	nd1 := gc.normalPDF(d1)
	gamma := nd1 / (spot * iv * math.Sqrt(t))

	return gamma
}

// normalPDF calculates standard normal probability density function
func (gc *GammaExposureCollector) normalPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2*math.Pi)
}

// findGammaFlip finds the strike where gamma changes sign
func (gc *GammaExposureCollector) findGammaFlip(gammaByStrike map[float64]float64, spotPrice float64) float64 {
	// Find strikes above and below spot
	var strikesAbove, strikesBelow []float64

	for strike := range gammaByStrike {
		if strike > spotPrice {
			strikesAbove = append(strikesAbove, strike)
		} else {
			strikesBelow = append(strikesBelow, strike)
		}
	}

	// Find where cumulative gamma changes sign
	// Simplified: just return a strike near zero gamma
	for strike, gamma := range gammaByStrike {
		if math.Abs(gamma) < 1000 { // Near-zero gamma
			return strike
		}
	}

	return spotPrice // Default to spot
}

// calculateMaxPain finds the strike with maximum pain for option sellers
func (gc *GammaExposureCollector) calculateMaxPain(instruments []optionsInstrument, spotPrice float64) float64 {
	// Max pain = strike where option sellers lose the least
	// = strike where sum of (call_oi * max(0, spot-strike) + put_oi * max(0, strike-spot)) is minimized

	painByStrike := make(map[float64]float64)

	// Get unique strikes
	strikes := make(map[float64]bool)
	for _, inst := range instruments {
		strikes[inst.strike] = true
	}

	// Calculate pain for each potential expiry price
	for testStrike := range strikes {
		pain := 0.0

		for _, inst := range instruments {
			if inst.side == "call" && testStrike > inst.strike {
				pain += inst.oi * (testStrike - inst.strike)
			} else if inst.side == "put" && testStrike < inst.strike {
				pain += inst.oi * (inst.strike - testStrike)
			}
		}

		painByStrike[testStrike] = pain
	}

	// Find strike with minimum pain
	minPain := math.MaxFloat64
	maxPainStrike := spotPrice

	for strike, pain := range painByStrike {
		if pain < minPain {
			minPain = pain
			maxPainStrike = strike
		}
	}

	return maxPainStrike
}

// calculateIVMetrics calculates IV-related metrics
func (gc *GammaExposureCollector) calculateIVMetrics(instruments []optionsInstrument, spotPrice float64) (float64, float64, float64, float64) {
	// IV Index: ATM IV average
	// IV 25d: 25-delta call/put IV
	// IV Skew: difference between put and call IV

	var atmIVSum, atmCount float64
	var call25IV, put25IV float64

	for _, inst := range instruments {
		// ATM = strikes within 5% of spot
		if math.Abs(inst.strike-spotPrice)/spotPrice < 0.05 {
			atmIVSum += inst.iv
			atmCount++
		}

		// Approximate 25-delta options (roughly 10% OTM)
		if inst.side == "call" && inst.strike/spotPrice > 1.08 && inst.strike/spotPrice < 1.12 {
			call25IV = inst.iv
		}
		if inst.side == "put" && inst.strike/spotPrice < 0.92 && inst.strike/spotPrice > 0.88 {
			put25IV = inst.iv
		}
	}

	ivIndex := 0.0
	if atmCount > 0 {
		ivIndex = atmIVSum / atmCount
	}

	ivSkew := put25IV - call25IV

	return ivIndex, call25IV, put25IV, ivSkew
}

// InterpretGammaExposure provides interpretation of GEX
func InterpretGammaExposure(gex float64) string {
	// Negative GEX = dealers short gamma = resistance to price movement
	// Positive GEX = dealers long gamma = acceleration of price movement

	if gex < -5_000_000_000 {
		return "very_negative" // Strong resistance to moves
	} else if gex < -1_000_000_000 {
		return "negative"
	} else if gex > 1_000_000_000 {
		return "positive" // Price acceleration
	} else if gex > 5_000_000_000 {
		return "very_positive"
	}
	return "neutral"
}
