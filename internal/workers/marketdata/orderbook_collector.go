package marketdata

import (
	"context"
	"time"

	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// OrderBookCollector collects order book snapshots from exchanges
// This worker runs every 10 seconds to capture order book depth
type OrderBookCollector struct {
	*workers.BaseWorker
	mdRepo      market_data.Repository
	exchFactory exchanges.CentralFactory
	symbols     []string
	exchanges   []string // List of exchange names to collect from
	depth       int      // Order book depth (number of price levels)
}

// NewOrderBookCollector creates a new order book collector worker
func NewOrderBookCollector(
	mdRepo market_data.Repository,
	exchFactory exchanges.CentralFactory,
	symbols []string,
	exchanges []string,
	depth int,
	interval time.Duration,
	enabled bool,
) *OrderBookCollector {
	if depth <= 0 {
		depth = 20 // Default depth
	}

	return &OrderBookCollector{
		BaseWorker:  workers.NewBaseWorker("orderbook_collector", interval, enabled),
		mdRepo:      mdRepo,
		exchFactory: exchFactory,
		symbols:     symbols,
		exchanges:   exchanges,
		depth:       depth,
	}
}

// Run executes one iteration of order book collection
func (obc *OrderBookCollector) Run(ctx context.Context) error {
	obc.Log().Debug("OrderBook collector: starting iteration")

	totalBooks := 0
	errorCount := 0

	// Collect order books from each exchange
	for _, exchangeName := range obc.exchanges {
		exchangeClient, err := obc.exchFactory.GetClient(exchangeName)
		if err != nil {
			obc.Log().Error("Failed to get exchange client",
				"exchange", exchangeName,
				"error", err,
			)
			errorCount++
			continue
		}

		// Collect order books for all symbols on this exchange
		bookCount, err := obc.collectExchangeOrderBooks(ctx, exchangeClient, exchangeName)
		if err != nil {
			obc.Log().Error("Failed to collect exchange order books",
				"exchange", exchangeName,
				"error", err,
			)
			errorCount++
			continue
		}

		totalBooks += bookCount
	}

	obc.Log().Info("OrderBook collection complete",
		"total_books", totalBooks,
		"errors", errorCount,
	)

	return nil
}

// collectExchangeOrderBooks collects order books for all symbols from a specific exchange
func (obc *OrderBookCollector) collectExchangeOrderBooks(
	ctx context.Context,
	exchange exchanges.Exchange,
	exchangeName string,
) (int, error) {
	successCount := 0

	// Collect order book for each symbol
	for _, symbol := range obc.symbols {
		orderBook, err := obc.collectOrderBook(ctx, exchange, exchangeName, symbol)
		if err != nil {
			obc.Log().Error("Failed to collect order book",
				"exchange", exchangeName,
				"symbol", symbol,
				"error", err,
			)
			// Continue with other symbols
			continue
		}

		// Insert order book snapshot to ClickHouse
		if err := obc.mdRepo.InsertOrderBook(ctx, []market_data.OrderBookSnapshot{*orderBook}); err != nil {
			obc.Log().Error("Failed to insert order book snapshot",
				"exchange", exchangeName,
				"symbol", symbol,
				"error", err,
			)
			continue
		}

		successCount++
	}

	obc.Log().Debug("Exchange order books collected",
		"exchange", exchangeName,
		"book_count", successCount,
	)

	return successCount, nil
}

// collectOrderBook collects a single order book snapshot from an exchange
func (obc *OrderBookCollector) collectOrderBook(
	ctx context.Context,
	exchange exchanges.Exchange,
	exchangeName, symbol string,
) (*market_data.OrderBookSnapshot, error) {
	// Get order book from exchange
	exchangeOrderBook, err := exchange.GetOrderBook(ctx, symbol, obc.depth)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get order book for %s", symbol)
	}

	// Calculate total bid and ask depth
	bidDepth := 0.0
	askDepth := 0.0

	for _, bid := range exchangeOrderBook.Bids {
		bidDepth += bid.Amount.InexactFloat64()
	}

	for _, ask := range exchangeOrderBook.Asks {
		askDepth += ask.Amount.InexactFloat64()
	}

	// Convert to JSON strings (for ClickHouse storage)
	// In production, we'd use proper JSON marshaling
	bidsJSON := obc.formatOrderBookSide(exchangeOrderBook.Bids)
	asksJSON := obc.formatOrderBookSide(exchangeOrderBook.Asks)

	// Create order book snapshot
	snapshot := &market_data.OrderBookSnapshot{
		Exchange:  exchangeName,
		Symbol:    symbol,
		Timestamp: time.Now(),
		Bids:      bidsJSON,
		Asks:      asksJSON,
		BidDepth:  bidDepth,
		AskDepth:  askDepth,
	}

	return snapshot, nil
}

// formatOrderBookSide converts order book side to JSON string
func (obc *OrderBookCollector) formatOrderBookSide(levels []exchanges.OrderBookEntry) string {
	if len(levels) == 0 {
		return "[]"
	}

	// Simple JSON formatting (in production, use json.Marshal)
	result := "["
	for i, level := range levels {
		if i > 0 {
			result += ","
		}
		result += `{"price":` + level.Price.String() + `,"amount":` + level.Amount.String() + `}`
	}
	result += "]"

	return result
}
