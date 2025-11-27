package testsupport

import (
	"context"

	"github.com/shopspring/decimal"
)

// MockCorrelationService provides predefined correlation data for testing
type MockCorrelationService struct {
	// predefined correlations for common crypto assets
	correlations map[string]map[string]float64
}

// NewMockCorrelationService creates a mock correlation service with realistic crypto correlations
func NewMockCorrelationService() *MockCorrelationService {
	return &MockCorrelationService{
		correlations: map[string]map[string]float64{
			"BTC/USDT": {
				"ETH/USDT":   0.85, // high correlation
				"BNB/USDT":   0.75,
				"SOL/USDT":   0.70,
				"MATIC/USDT": 0.65,
				"DOGE/USDT":  0.60,
			},
			"ETH/USDT": {
				"BTC/USDT":   0.85,
				"BNB/USDT":   0.80,
				"SOL/USDT":   0.75,
				"MATIC/USDT": 0.70,
			},
			"SOL/USDT": {
				"BTC/USDT": 0.70,
				"ETH/USDT": 0.75,
			},
		},
	}
}

// GetCorrelations returns mock correlations for a symbol
func (s *MockCorrelationService) GetCorrelations(ctx context.Context, symbol string) (map[string]decimal.Decimal, error) {
	result := make(map[string]decimal.Decimal)

	if corrs, ok := s.correlations[symbol]; ok {
		for asset, corr := range corrs {
			result[asset] = decimal.NewFromFloat(corr)
		}
	}

	// Add BTC correlation for any asset if not present
	if _, hasBTC := result["BTC/USDT"]; !hasBTC && symbol != "BTC/USDT" {
		// Default: most crypto is highly correlated with BTC
		result["BTC/USDT"] = decimal.NewFromFloat(0.75)
	}

	return result, nil
}

// GetBTCCorrelation returns mock BTC correlation
func (s *MockCorrelationService) GetBTCCorrelation(ctx context.Context, symbol string) (decimal.Decimal, error) {
	if symbol == "BTC/USDT" {
		return decimal.NewFromFloat(1.0), nil // perfect correlation with itself
	}

	if corrs, ok := s.correlations[symbol]; ok {
		if btcCorr, hasBTC := corrs["BTC/USDT"]; hasBTC {
			return decimal.NewFromFloat(btcCorr), nil
		}
	}

	// Default correlation for unknown assets
	return decimal.NewFromFloat(0.70), nil
}

// WithCorrelation allows setting custom correlation for testing
func (s *MockCorrelationService) WithCorrelation(symbol, asset string, correlation float64) *MockCorrelationService {
	if _, ok := s.correlations[symbol]; !ok {
		s.correlations[symbol] = make(map[string]float64)
	}
	s.correlations[symbol][asset] = correlation
	return s
}
