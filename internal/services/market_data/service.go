package market_data

import (
	"context"

	"prometheus/internal/domain/market_data"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service handles market data business logic
// Provides abstraction over ClickHouse repository for consumers
type Service struct {
	repository market_data.Repository
	log        *logger.Logger
}

// NewService creates a new market data service
func NewService(
	repository market_data.Repository,
	log *logger.Logger,
) *Service {
	return &Service{
		repository: repository,
		log:        log,
	}
}

// StoreMixedBatch stores a mixed batch of market data items
// Groups items by type and inserts them efficiently
func (s *Service) StoreMixedBatch(ctx context.Context, batch []interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	var (
		klines       []market_data.OHLCV
		tickers      []market_data.Ticker
		depths       []market_data.OrderBookSnapshot
		trades       []market_data.Trade
		markPrices   []market_data.MarkPrice
		liquidations []market_data.Liquidation
	)

	// Group by type
	for _, item := range batch {
		switch v := item.(type) {
		case market_data.OHLCV:
			klines = append(klines, v)
		case market_data.Ticker:
			tickers = append(tickers, v)
		case market_data.OrderBookSnapshot:
			depths = append(depths, v)
		case market_data.Trade:
			trades = append(trades, v)
		case market_data.MarkPrice:
			markPrices = append(markPrices, v)
		case market_data.Liquidation:
			liquidations = append(liquidations, v)
		}
	}

	// Insert each type
	if len(klines) > 0 {
		if err := s.repository.InsertOHLCV(ctx, klines); err != nil {
			return errors.Wrap(err, "insert klines")
		}
		s.log.Infow("  → Inserted klines", "count", len(klines))
	}

	if len(tickers) > 0 {
		if err := s.repository.InsertTicker(ctx, tickers); err != nil {
			return errors.Wrap(err, "insert tickers")
		}
		s.log.Infow("  → Inserted tickers", "count", len(tickers))
	}

	if len(depths) > 0 {
		if err := s.repository.InsertOrderBook(ctx, depths); err != nil {
			return errors.Wrap(err, "insert depths")
		}
		s.log.Infow("  → Inserted order book snapshots", "count", len(depths))
	}

	if len(trades) > 0 {
		if err := s.repository.InsertTrades(ctx, trades); err != nil {
			return errors.Wrap(err, "insert trades")
		}
		s.log.Infow("  → Inserted trades", "count", len(trades))
	}

	if len(markPrices) > 0 {
		if err := s.repository.InsertMarkPrice(ctx, markPrices); err != nil {
			return errors.Wrap(err, "insert mark prices")
		}
		s.log.Infow("  → Inserted mark prices", "count", len(markPrices))
	}

	if len(liquidations) > 0 {
		if err := s.repository.InsertLiquidations(ctx, liquidations); err != nil {
			return errors.Wrap(err, "insert liquidations")
		}
		s.log.Infow("  → Inserted liquidations", "count", len(liquidations))
	}

	return nil
}
