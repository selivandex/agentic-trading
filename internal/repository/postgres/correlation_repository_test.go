package postgres

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/pkg/logger"
)

// MockMacroDataReader implements MacroDataReader for testing
type MockMacroDataReader struct {
	correlations map[string]map[string]float64
}

func (m *MockMacroDataReader) GetLatestCorrelation(ctx context.Context, cryptoSymbol, asset string) (float64, error) {
	if symbolData, ok := m.correlations[cryptoSymbol]; ok {
		if corr, ok := symbolData[asset]; ok {
			return corr, nil
		}
	}
	return 0.0, nil
}

func (m *MockMacroDataReader) GetAllCorrelations(ctx context.Context, cryptoSymbol string) (map[string]float64, error) {
	if symbolData, ok := m.correlations[cryptoSymbol]; ok {
		return symbolData, nil
	}
	return nil, nil
}

func TestCorrelationRepository_GetCorrelations(t *testing.T) {
	log := logger.Get()

	// Setup mock data
	mockRepo := &MockMacroDataReader{
		correlations: map[string]map[string]float64{
			"BTC/USDT": {
				"SP500":    0.65,
				"GOLD":     0.45,
				"DXY":      -0.35,
				"ETH/USDT": 0.85,
				"NASDAQ":   0.72,
			},
			"ETH/USDT": {
				"SP500":    0.60,
				"BTC/USDT": 0.85,
				"NASDAQ":   0.68,
			},
		},
	}

	repo := NewCorrelationRepository(mockRepo, log)
	ctx := context.Background()

	// Test GetCorrelations for BTC
	correlations, err := repo.GetCorrelations(ctx, "BTC/USDT")
	require.NoError(t, err)

	// Verify correlations
	assert.True(t, correlations["SP500"].Equal(decimal.NewFromFloat(0.65)))
	assert.True(t, correlations["GOLD"].Equal(decimal.NewFromFloat(0.45)))
	assert.True(t, correlations["DXY"].Equal(decimal.NewFromFloat(-0.35)))
	assert.True(t, correlations["ETH/USDT"].Equal(decimal.NewFromFloat(0.85)))

	// Test cache by calling again (should use cached value)
	cachedCorrelations, err := repo.GetCorrelations(ctx, "BTC/USDT")
	require.NoError(t, err)
	assert.Equal(t, correlations, cachedCorrelations, "Should return cached correlations")
}

func TestCorrelationRepository_GetBTCCorrelation(t *testing.T) {
	log := logger.Get()

	mockRepo := &MockMacroDataReader{
		correlations: map[string]map[string]float64{
			"ETH/USDT": {
				"BTC/USDT": 0.85,
				"SP500":    0.60,
			},
			"SOL/USDT": {
				"BTC/USDT": 0.72,
			},
			"BTC/USDT": {
				"SP500": 0.65,
			},
		},
	}

	repo := NewCorrelationRepository(mockRepo, log)
	ctx := context.Background()

	// Test GetBTCCorrelation for ETH
	correlation, err := repo.GetBTCCorrelation(ctx, "ETH/USDT")
	require.NoError(t, err)
	assert.True(t, correlation.Equal(decimal.NewFromFloat(0.85)))

	// Test GetBTCCorrelation for SOL
	correlation, err = repo.GetBTCCorrelation(ctx, "SOL/USDT")
	require.NoError(t, err)
	assert.True(t, correlation.Equal(decimal.NewFromFloat(0.72)))

	// Test GetBTCCorrelation for BTC itself
	correlation, err = repo.GetBTCCorrelation(ctx, "BTC/USDT")
	require.NoError(t, err)
	assert.True(t, correlation.Equal(decimal.NewFromFloat(1.0)), "BTC should have correlation 1.0 with itself")
}

func TestCorrelationRepository_DefaultCorrelations(t *testing.T) {
	log := logger.Get()

	// Mock with no data
	mockRepo := &MockMacroDataReader{
		correlations: map[string]map[string]float64{},
	}

	repo := NewCorrelationRepository(mockRepo, log)
	ctx := context.Background()

	// Test GetCorrelations for unknown symbol (should return defaults)
	correlations, err := repo.GetCorrelations(ctx, "UNKNOWN/USDT")
	require.NoError(t, err)

	// Should have some default correlations
	assert.NotEmpty(t, correlations, "Should return default correlations for unknown symbol")
}

func TestCorrelationRepository_HighCorrelation(t *testing.T) {
	log := logger.Get()

	mockRepo := &MockMacroDataReader{
		correlations: map[string]map[string]float64{
			"ETH/USDT": {
				"BTC/USDT":  0.95, // Very high correlation
				"LINK/USDT": 0.88,
			},
		},
	}

	repo := NewCorrelationRepository(mockRepo, log)
	ctx := context.Background()

	correlations, err := repo.GetCorrelations(ctx, "ETH/USDT")
	require.NoError(t, err)

	// Verify high correlations
	btcCorr := correlations["BTC/USDT"]
	assert.True(t, btcCorr.GreaterThan(decimal.NewFromFloat(0.9)),
		"ETH should have high correlation with BTC")
}

func TestCorrelationRepository_NegativeCorrelation(t *testing.T) {
	log := logger.Get()

	mockRepo := &MockMacroDataReader{
		correlations: map[string]map[string]float64{
			"BTC/USDT": {
				"DXY": -0.45, // Negative correlation with dollar index
				"VIX": -0.35, // Negative correlation with volatility
			},
		},
	}

	repo := NewCorrelationRepository(mockRepo, log)
	ctx := context.Background()

	correlations, err := repo.GetCorrelations(ctx, "BTC/USDT")
	require.NoError(t, err)

	// Verify negative correlations
	dxyCorr := correlations["DXY"]
	assert.True(t, dxyCorr.LessThan(decimal.Zero),
		"BTC should have negative correlation with DXY")

	vixCorr := correlations["VIX"]
	assert.True(t, vixCorr.LessThan(decimal.Zero),
		"BTC should have negative correlation with VIX")
}

func TestCorrelationRepository_CacheExpiration(t *testing.T) {
	log := logger.Get()

	mockRepo := &MockMacroDataReader{
		correlations: map[string]map[string]float64{
			"BTC/USDT": {
				"SP500": 0.65,
			},
		},
	}

	repo := NewCorrelationRepository(mockRepo, log)
	ctx := context.Background()

	// First call - populate cache
	_, err := repo.GetCorrelations(ctx, "BTC/USDT")
	require.NoError(t, err)

	// Clear cache manually (simulating expiration)
	repo.ClearCache()

	// Second call - should fetch from mock again
	correlations, err := repo.GetCorrelations(ctx, "BTC/USDT")
	require.NoError(t, err)
	assert.NotNil(t, correlations)
}

func TestCorrelationRepository_MultipleSymbols(t *testing.T) {
	log := logger.Get()

	mockRepo := &MockMacroDataReader{
		correlations: map[string]map[string]float64{
			"BTC/USDT": {
				"SP500": 0.65,
				"GOLD":  0.45,
			},
			"ETH/USDT": {
				"SP500":    0.60,
				"BTC/USDT": 0.85,
			},
			"SOL/USDT": {
				"BTC/USDT": 0.70,
				"ETH/USDT": 0.75,
			},
		},
	}

	repo := NewCorrelationRepository(mockRepo, log)
	ctx := context.Background()

	// Test multiple symbols
	symbols := []string{"BTC/USDT", "ETH/USDT", "SOL/USDT"}

	for _, symbol := range symbols {
		correlations, err := repo.GetCorrelations(ctx, symbol)
		require.NoError(t, err, "Should get correlations for "+symbol)
		assert.NotEmpty(t, correlations, "Should have correlations for "+symbol)
	}
}

func TestCorrelationRepository_ZeroCorrelation(t *testing.T) {
	log := logger.Get()

	mockRepo := &MockMacroDataReader{
		correlations: map[string]map[string]float64{
			"STABLE/USDT": {
				"BTC/USDT": 0.05, // Near zero correlation
				"SP500":    0.02,
			},
		},
	}

	repo := NewCorrelationRepository(mockRepo, log)
	ctx := context.Background()

	correlations, err := repo.GetCorrelations(ctx, "STABLE/USDT")
	require.NoError(t, err)

	// Verify near-zero correlations
	btcCorr := correlations["BTC/USDT"]
	assert.True(t, btcCorr.Abs().LessThan(decimal.NewFromFloat(0.1)),
		"Stablecoin should have near-zero correlation with BTC")
}

func TestCorrelationRepository_TradFiCorrelations(t *testing.T) {
	log := logger.Get()

	mockRepo := &MockMacroDataReader{
		correlations: map[string]map[string]float64{
			"BTC/USDT": {
				"SP500":  0.65,
				"NASDAQ": 0.72,
				"GOLD":   0.45,
				"BONDS":  -0.25,
				"DXY":    -0.35,
			},
		},
	}

	repo := NewCorrelationRepository(mockRepo, log)
	ctx := context.Background()

	correlations, err := repo.GetCorrelations(ctx, "BTC/USDT")
	require.NoError(t, err)

	// Verify TradFi correlations
	assert.True(t, correlations["SP500"].GreaterThan(decimal.Zero))
	assert.True(t, correlations["NASDAQ"].GreaterThan(decimal.Zero))
	assert.True(t, correlations["GOLD"].GreaterThan(decimal.Zero))
	assert.True(t, correlations["BONDS"].LessThan(decimal.Zero))
	assert.True(t, correlations["DXY"].LessThan(decimal.Zero))
}

func TestCorrelationRepository_AltcoinBTCCorrelation(t *testing.T) {
	log := logger.Get()

	mockRepo := &MockMacroDataReader{
		correlations: map[string]map[string]float64{
			"ETH/USDT": {
				"BTC/USDT": 0.85,
			},
			"ADA/USDT": {
				"BTC/USDT": 0.78,
			},
			"DOT/USDT": {
				"BTC/USDT": 0.72,
			},
		},
	}

	repo := NewCorrelationRepository(mockRepo, log)
	ctx := context.Background()

	// Test BTC correlation for different altcoins
	altcoins := []string{"ETH/USDT", "ADA/USDT", "DOT/USDT"}

	for _, symbol := range altcoins {
		btcCorr, err := repo.GetBTCCorrelation(ctx, symbol)
		require.NoError(t, err)
		assert.True(t, btcCorr.GreaterThan(decimal.NewFromFloat(0.7)),
			symbol+" should have strong positive correlation with BTC")
	}
}

func TestCorrelationRepository_CacheClearing(t *testing.T) {
	log := logger.Get()

	mockRepo := &MockMacroDataReader{
		correlations: map[string]map[string]float64{
			"BTC/USDT": {
				"SP500": 0.65,
			},
		},
	}

	repo := NewCorrelationRepository(mockRepo, log)
	ctx := context.Background()

	// Populate cache
	_, err := repo.GetCorrelations(ctx, "BTC/USDT")
	require.NoError(t, err)

	// Verify cache has data
	assert.NotEmpty(t, repo.cache)

	// Test ClearCache
	repo.ClearCache()

	// Verify cache is cleared
	assert.Empty(t, repo.cache)
	assert.Equal(t, int64(0), repo.cacheTTL)
}
