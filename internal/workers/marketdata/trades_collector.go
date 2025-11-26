package marketdata

import (
	"context"
	"time"

	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// TradesCollector collects recent trades (tape) from exchanges
// This worker runs every 1 second to capture real-time trade flow
type TradesCollector struct {
	*workers.BaseWorker
	mdRepo      market_data.Repository
	exchFactory exchanges.CentralFactory
	symbols     []string
	exchanges   []string // List of exchange names to collect from
	limit       int      // Number of recent trades to fetch
}

// NewTradesCollector creates a new trades collector worker
func NewTradesCollector(
	mdRepo market_data.Repository,
	exchFactory exchanges.CentralFactory,
	symbols []string,
	exchanges []string,
	limit int,
	interval time.Duration,
	enabled bool,
) *TradesCollector {
	if limit <= 0 {
		limit = 100 // Default: fetch last 100 trades
	}

	return &TradesCollector{
		BaseWorker:  workers.NewBaseWorker("trades_collector", interval, enabled),
		mdRepo:      mdRepo,
		exchFactory: exchFactory,
		symbols:     symbols,
		exchanges:   exchanges,
		limit:       limit,
	}
}

// Run executes one iteration of trades collection
func (tc *TradesCollector) Run(ctx context.Context) error {
	tc.Log().Debug("Trades collector: starting iteration")

	totalTrades := 0
	errorCount := 0

	// Collect trades from each exchange
	for _, exchangeName := range tc.exchanges {
		exchangeClient, err := tc.exchFactory.GetClient(exchangeName)
		if err != nil {
			tc.Log().Error("Failed to get exchange client",
				"exchange", exchangeName,
				"error", err,
			)
			errorCount++
			continue
		}

		// Collect trades for all symbols on this exchange
		tradesCount, err := tc.collectExchangeTrades(ctx, exchangeClient, exchangeName)
		if err != nil {
			tc.Log().Error("Failed to collect exchange trades",
				"exchange", exchangeName,
				"error", err,
			)
			errorCount++
			continue
		}

		totalTrades += tradesCount
	}

	tc.Log().Info("Trades collection complete",
		"total_trades", totalTrades,
		"errors", errorCount,
	)

	return nil
}

// collectExchangeTrades collects trades for all symbols from a specific exchange
func (tc *TradesCollector) collectExchangeTrades(
	ctx context.Context,
	exchange exchanges.Exchange,
	exchangeName string,
) (int, error) {
	successCount := 0

	// Collect trades for each symbol
	for _, symbol := range tc.symbols {
		tradesCount, err := tc.collectTrades(ctx, exchange, exchangeName, symbol)
		if err != nil {
			tc.Log().Error("Failed to collect trades",
				"exchange", exchangeName,
				"symbol", symbol,
				"error", err,
			)
			// Continue with other symbols
			continue
		}

		successCount += tradesCount
	}

	tc.Log().Debug("Exchange trades collected",
		"exchange", exchangeName,
		"trade_count", successCount,
	)

	return successCount, nil
}

// collectTrades collects recent trades for a specific symbol
func (tc *TradesCollector) collectTrades(
	ctx context.Context,
	exchange exchanges.Exchange,
	exchangeName, symbol string,
) (int, error) {
	// Get recent trades from exchange
	exchangeTrades, err := exchange.GetTrades(ctx, symbol, tc.limit)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get trades for %s", symbol)
	}

	if len(exchangeTrades) == 0 {
		return 0, nil
	}

	// Convert and insert trades in batch
	trades := make([]market_data.Trade, 0, len(exchangeTrades))
	for _, et := range exchangeTrades {
		// Determine side and buyer flag
		side := "buy"
		isBuyer := true
		if et.Side == exchanges.OrderSideSell {
			side = "sell"
			isBuyer = false
		}

		trade := market_data.Trade{
			Exchange:  exchangeName,
			Symbol:    symbol,
			TradeID:   et.ID,
			Timestamp: et.Timestamp,
			Price:     et.Price.InexactFloat64(),
			Quantity:  et.Amount.InexactFloat64(),
			Side:      side,
			IsBuyer:   isBuyer,
		}
		trades = append(trades, trade)
	}

	// Insert trades batch to ClickHouse
	if err := tc.mdRepo.InsertTrades(ctx, trades); err != nil {
		return 0, errors.Wrap(err, "failed to insert trades batch")
	}

	tc.Log().Debug("Trades collected",
		"exchange", exchangeName,
		"symbol", symbol,
		"count", len(trades),
	)

	return len(trades), nil
}

// CalculateTradeImbalance calculates buy vs sell pressure from recent trades
// This is used by order flow analysis tools
func (tc *TradesCollector) CalculateTradeImbalance(trades []market_data.Trade) (buyVolume, sellVolume float64) {
	for _, trade := range trades {
		volume := trade.Quantity
		if trade.Side == "buy" {
			buyVolume += volume
		} else {
			sellVolume += volume
		}
	}

	return buyVolume, sellVolume
}

// DetectWhaleTrades filters trades above a certain USD value threshold
// Returns trades that are likely from whales (> $100k)
func (tc *TradesCollector) DetectWhaleTrades(trades []market_data.Trade, minUSDValue float64) []market_data.Trade {
	whaleTrades := make([]market_data.Trade, 0)

	for _, trade := range trades {
		tradeValue := trade.Price * trade.Quantity
		if tradeValue >= minUSDValue {
			whaleTrades = append(whaleTrades, trade)
		}
	}

	return whaleTrades
}
