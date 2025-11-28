package correlation

import (
	"context"
	"fmt"
	"math"
	"strings"

	"prometheus/internal/domain/risk"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// CorrelationAnalysisTool provides correlation analysis for OpportunitySynthesizer
// Data is pre-computed by correlation workers and retrieved from database
type CorrelationAnalysisTool struct {
	correlationService risk.CorrelationService
	log                *logger.Logger
}

// NewCorrelationAnalysisTool creates a new correlation analysis tool
func NewCorrelationAnalysisTool(correlationService risk.CorrelationService) *CorrelationAnalysisTool {
	return &CorrelationAnalysisTool{
		correlationService: correlationService,
		log:                logger.Get().With("component", "correlation_tool"),
	}
}

// Execute performs correlation analysis and returns human-readable text
func (t *CorrelationAnalysisTool) Execute(ctx context.Context, symbol string) (string, error) {
	t.log.Debug("Executing correlation analysis", "symbol", symbol)

	// Get all correlations for this symbol
	correlations, err := t.correlationService.GetCorrelations(ctx, symbol)
	if err != nil {
		return "", errors.Wrap(err, "failed to get correlations")
	}

	if len(correlations) == 0 {
		return t.formatNoDataResponse(symbol), nil
	}

	// Get BTC correlation specifically (most important for crypto)
	btcCorr, err := t.correlationService.GetBTCCorrelation(ctx, symbol)
	if err != nil {
		return "", errors.Wrap(err, "failed to get BTC correlation")
	}

	// Format analysis
	return t.formatAnalysis(symbol, correlations, btcCorr.InexactFloat64()), nil
}

// formatAnalysis creates human-readable correlation analysis
func (t *CorrelationAnalysisTool) formatAnalysis(symbol string, correlations map[string]interface{}, btcCorr float64) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CORRELATION ANALYSIS - %s\n\n", symbol))

	// BTC Correlation (most important)
	sb.WriteString(fmt.Sprintf("BTC CORRELATION: %.2f\n", btcCorr))
	sb.WriteString(fmt.Sprintf("Interpretation: %s\n\n", t.interpretCorrelation(btcCorr, "BTC")))

	// Correlation regime
	regime := t.determineRegime(btcCorr)
	sb.WriteString(fmt.Sprintf("REGIME: %s\n\n", regime))

	// Other correlations
	if len(correlations) > 0 {
		sb.WriteString("OTHER CORRELATIONS:\n")
		for asset, corrVal := range correlations {
			if asset == "BTC/USDT" && symbol != "BTC/USDT" {
				continue // Already covered above
			}

			corr := 0.0
			switch v := corrVal.(type) {
			case float64:
				corr = v
			case float32:
				corr = float64(v)
			default:
				// Try to extract from decimal or other types
				if dec, ok := corrVal.(interface{ InexactFloat64() float64 }); ok {
					corr = dec.InexactFloat64()
				}
			}

			sb.WriteString(fmt.Sprintf("- %s: %.2f (%s)\n", asset, corr, t.classifyStrength(corr)))
		}
		sb.WriteString("\n")
	}

	// Trading implications
	sb.WriteString("TRADING IMPLICATIONS:\n")
	implications := t.getTradingImplications(symbol, btcCorr, regime)
	for _, implication := range implications {
		sb.WriteString(fmt.Sprintf("- %s\n", implication))
	}

	// Divergence risk
	sb.WriteString(fmt.Sprintf("\nDIVERGENCE RISK: %s\n", t.getDivergenceRisk(btcCorr)))

	return sb.String()
}

// interpretCorrelation provides interpretation of correlation value
func (t *CorrelationAnalysisTool) interpretCorrelation(corr float64, asset string) string {
	absCorr := math.Abs(corr)

	if absCorr >= 0.8 {
		if corr > 0 {
			return fmt.Sprintf("Very strong positive correlation with %s (moves together)", asset)
		}
		return fmt.Sprintf("Very strong negative correlation with %s (moves opposite)", asset)
	} else if absCorr >= 0.6 {
		if corr > 0 {
			return fmt.Sprintf("Strong positive correlation with %s", asset)
		}
		return fmt.Sprintf("Strong negative correlation with %s", asset)
	} else if absCorr >= 0.4 {
		if corr > 0 {
			return fmt.Sprintf("Moderate positive correlation with %s", asset)
		}
		return fmt.Sprintf("Moderate negative correlation with %s", asset)
	} else if absCorr >= 0.2 {
		return fmt.Sprintf("Weak correlation with %s", asset)
	}
	return fmt.Sprintf("Very weak/no correlation with %s (moves independently)", asset)
}

// classifyStrength classifies correlation strength
func (t *CorrelationAnalysisTool) classifyStrength(corr float64) string {
	absCorr := math.Abs(corr)

	if absCorr >= 0.8 {
		return "very strong"
	} else if absCorr >= 0.6 {
		return "strong"
	} else if absCorr >= 0.4 {
		return "moderate"
	} else if absCorr >= 0.2 {
		return "weak"
	}
	return "very weak"
}

// determineRegime determines the correlation regime
func (t *CorrelationAnalysisTool) determineRegime(btcCorr float64) string {
	absCorr := math.Abs(btcCorr)

	switch {
	case absCorr >= 0.80:
		return "HIGH CORRELATION (tight coupling with BTC)"
	case absCorr >= 0.60:
		return "MODERATE CORRELATION (generally follows BTC)"
	case absCorr >= 0.40:
		return "LOW CORRELATION (some independence from BTC)"
	default:
		return "DECOUPLING (moving independently of BTC)"
	}
}

// getTradingImplications provides trading implications based on correlation
func (t *CorrelationAnalysisTool) getTradingImplications(symbol string, btcCorr float64, regime string) []string {
	implications := []string{}

	absCorr := math.Abs(btcCorr)

	if absCorr >= 0.80 {
		implications = append(implications, "BTC direction is critical - check BTC analysis first")
		implications = append(implications, fmt.Sprintf("If BTC reverses, %s likely to follow", symbol))
		implications = append(implications, "Portfolio diversification limited (high beta to BTC)")
	} else if absCorr >= 0.60 {
		implications = append(implications, "BTC moves will influence but not dominate")
		implications = append(implications, fmt.Sprintf("%s has some independent price action", symbol))
		implications = append(implications, "Moderate portfolio diversification benefit")
	} else if absCorr >= 0.40 {
		implications = append(implications, fmt.Sprintf("%s showing significant independence from BTC", symbol))
		implications = append(implications, "Focus on asset-specific analysis over BTC trends")
		implications = append(implications, "Good diversification opportunity")
	} else {
		implications = append(implications, fmt.Sprintf("%s decoupled from BTC - analyze independently", symbol))
		implications = append(implications, "Asset-specific catalysts driving price")
		implications = append(implications, "Excellent diversification (low correlation risk)")
	}

	return implications
}

// getDivergenceRisk assesses risk of correlation breakdown
func (t *CorrelationAnalysisTool) getDivergenceRisk(btcCorr float64) string {
	absCorr := math.Abs(btcCorr)

	if absCorr >= 0.80 {
		return "LOW (historically very stable correlation)"
	} else if absCorr >= 0.60 {
		return "MODERATE (correlation can weaken during alt seasons)"
	} else if absCorr >= 0.40 {
		return "ELEVATED (correlation already weak, monitor for regime change)"
	}
	return "HIGH (very low correlation, unpredictable relative movement)"
}

// formatNoDataResponse formats response when no correlation data is available
func (t *CorrelationAnalysisTool) formatNoDataResponse(symbol string) string {
	return fmt.Sprintf(`CORRELATION ANALYSIS - %s

NO DATA AVAILABLE

Correlation data has not yet been computed for this symbol.
Using default assumption: moderate correlation with BTC (~0.70)

TRADING IMPLICATIONS:
- Assume typical altcoin behavior (follows BTC with some lag)
- Monitor BTC price action as primary driver
- Lower confidence in correlation-based analysis

NOTE: Correlation data will be available after workers compute it.`, symbol)
}

// GetDefinition returns the tool definition for ADK registration
func (t *CorrelationAnalysisTool) GetDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        "get_correlation_analysis",
		"description": "Analyzes correlation of a cryptocurrency with BTC and other major assets. Returns correlation coefficients, regime classification, and trading implications. Correlation data is pre-computed by workers.",
		"category":    "correlation",
		"parameters": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"symbol": map[string]interface{}{
					"type":        "string",
					"description": "Trading pair symbol (e.g., 'ETH/USDT', 'SOL/USDT')",
				},
			},
			"required": []string{"symbol"},
		},
	}
}

