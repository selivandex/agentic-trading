package risk

import (
	"context"

	"github.com/shopspring/decimal"
)

// CorrelationService provides correlation data between crypto assets
// Implementations:
// - MockCorrelationService (testsupport/correlation_mock.go)
// - DBCorrelationService (repository/postgres/correlation_repository.go)
type CorrelationService interface {
	// GetCorrelations returns correlation coefficients for a symbol with other assets
	// Returns map[asset_symbol]correlation where correlation is between -1 and +1
	GetCorrelations(ctx context.Context, symbol string) (map[string]decimal.Decimal, error)

	// GetBTCCorrelation returns correlation coefficient with BTC specifically
	// Useful as BTC is the primary market driver for crypto
	GetBTCCorrelation(ctx context.Context, symbol string) (decimal.Decimal, error)
}

