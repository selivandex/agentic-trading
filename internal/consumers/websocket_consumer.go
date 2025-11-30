package consumers

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	kafkaadapter "prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	eventspb "prometheus/internal/events/proto"
	positionsvc "prometheus/internal/services/position"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

const (
	// Batch configuration for WebSocket consumer
	wsBatchSize     = 100 // Higher batch since we handle multiple types
	wsFlushInterval = 10 * time.Second
	wsStatsInterval = 1 * time.Minute
)

// WebSocketConsumer is a unified consumer for all WebSocket events
// Handles market data from TopicWebSocketEvents and user data from TopicUserDataEvents
type WebSocketConsumer struct {
	consumer        *kafkaadapter.Consumer
	repository      market_data.Repository
	positionService positionService
	log             *logger.Logger

	// Batching (single batch for all event types)
	mu    sync.Mutex
	batch []interface{} // Mixed batch of market data types

	// Statistics
	statsMu               sync.Mutex
	totalReceived         int64
	totalProcessed        int64
	totalSkipped          int64 // Events skipped due to type filtering
	totalDeduplicated     int64 // Events removed as duplicates
	totalErrors           int64
	lastFlushTime         time.Time
	lastStatsLogTime      time.Time
	receivedSinceLastLog  int64
	processedSinceLastLog int64

	// Per-type statistics
	typeStats map[string]*eventTypeStats
}

// positionService interface for position management
type positionService interface {
	UpdateFromWebSocket(ctx context.Context, update *positionsvc.PositionWebSocketUpdate) error
}

type eventTypeStats struct {
	received  int64
	processed int64
	errors    int64
}

// NewWebSocketConsumer creates a new unified WebSocket consumer
func NewWebSocketConsumer(
	consumer *kafkaadapter.Consumer,
	repository market_data.Repository,
	posService positionService,
	log *logger.Logger,
) *WebSocketConsumer {
	now := time.Now()
	return &WebSocketConsumer{
		consumer:         consumer,
		repository:       repository,
		positionService:  posService,
		log:              log,
		batch:            make([]interface{}, 0, wsBatchSize),
		lastFlushTime:    now,
		lastStatsLogTime: now,
		typeStats:        make(map[string]*eventTypeStats),
	}
}

// Start begins consuming WebSocket events (market data + user data)
func (c *WebSocketConsumer) Start(ctx context.Context) error {
	c.log.Infow("Starting unified WebSocket consumer...",
		"topics", []string{"websocket.events", "user-data.events"},
		"batch_size", wsBatchSize,
		"flush_interval", wsFlushInterval,
	)

	// Use BatchConsumerLifecycle for DRY shutdown/flush management
	lifecycle := NewBatchConsumerLifecycle(
		BatchConsumerConfig{
			ConsumerName:  "WebSocket Consumer",
			FlushInterval: wsFlushInterval,
			StatsInterval: wsStatsInterval,
			Logger:        c.log,
		},
		c.consumer,
		c, // implements BatchConsumer interface
	)

	// Setup cleanup
	defer lifecycle.Start(ctx)()

	// Start background workers
	lifecycle.StartBackgroundWorkers(ctx)

	c.log.Info("🔄 Starting to read WebSocket events from Kafka...")

	// Use Consume() for message loop
	return c.consumer.Consume(ctx, c.handleMessage)
}

// handleMessage processes a single Kafka message
func (c *WebSocketConsumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	c.incrementStat(&c.totalReceived)
	c.incrementStat(&c.receivedSinceLastLog)

	processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.handleEvent(processCtx, msg.Value); err != nil {
		c.log.Errorw("Failed to handle WebSocket event",
			"error", err,
		)
		c.incrementStat(&c.totalErrors)
		return err
	}

	return nil
}

// handleEvent processes a single event based on type
// Uses WebSocketEventWrapper or UserDataEventWrapper with oneof for efficient type detection
func (c *WebSocketConsumer) handleEvent(ctx context.Context, data []byte) error {
	// Try WebSocket market data wrapper first
	var wsWrapper eventspb.WebSocketEventWrapper
	if err := proto.Unmarshal(data, &wsWrapper); err == nil && wsWrapper.Event != nil {
		return c.handleWebSocketEvent(ctx, &wsWrapper)
	}

	// Try User Data wrapper
	var udWrapper eventspb.UserDataEventWrapper
	if err := proto.Unmarshal(data, &udWrapper); err == nil && udWrapper.Event != nil {
		return c.handleUserDataEvent(ctx, &udWrapper)
	}

	c.log.Debugw("⚠️ Failed to unmarshal any wrapper type", "data_len", len(data))
	c.incrementStat(&c.totalSkipped)
	return nil
}

// handleWebSocketEvent processes WebSocket market data events
func (c *WebSocketConsumer) handleWebSocketEvent(ctx context.Context, wrapper *eventspb.WebSocketEventWrapper) error {
	// Check which event type is set in the oneof
	switch event := wrapper.Event.(type) {
	case *eventspb.WebSocketEventWrapper_Kline:
		c.trackTypeReceived("websocket.kline")
		return c.handleKlineTyped(ctx, event.Kline)

	case *eventspb.WebSocketEventWrapper_Ticker:
		c.trackTypeReceived("websocket.ticker")
		return c.handleTickerTyped(ctx, event.Ticker)

	case *eventspb.WebSocketEventWrapper_Depth:
		c.trackTypeReceived("websocket.depth")
		return c.handleDepthTyped(ctx, event.Depth)

	case *eventspb.WebSocketEventWrapper_Trade:
		c.trackTypeReceived("websocket.trade")
		return c.handleTradeTyped(ctx, event.Trade)

	case *eventspb.WebSocketEventWrapper_MarkPrice:
		c.trackTypeReceived("websocket.mark_price")
		return c.handleMarkPriceTyped(ctx, event.MarkPrice)

	case *eventspb.WebSocketEventWrapper_Liquidation:
		c.trackTypeReceived("websocket.liquidation")
		return c.handleLiquidationTyped(ctx, event.Liquidation)

	case *eventspb.WebSocketEventWrapper_FundingRate:
		c.trackTypeReceived("websocket.funding_rate")
		c.log.Debug("Funding rate events not implemented yet")
		c.incrementStat(&c.totalSkipped)
		return nil

	default:
		c.log.Debugw("Unknown WebSocket event in wrapper")
		c.incrementStat(&c.totalSkipped)
		return nil
	}
}

// handleUserDataEvent processes User Data events (orders, positions, balances, margin calls)
func (c *WebSocketConsumer) handleUserDataEvent(ctx context.Context, wrapper *eventspb.UserDataEventWrapper) error {
	// Check which event type is set in the oneof
	switch event := wrapper.Event.(type) {
	case *eventspb.UserDataEventWrapper_PositionUpdate:
		c.trackTypeReceived("userdata.position_update")
		if err := c.handlePositionUpdateTyped(ctx, event.PositionUpdate); err != nil {
			c.trackTypeError("userdata.position_update")
			return err
		}
		c.trackTypeProcessed("userdata.position_update")

	case *eventspb.UserDataEventWrapper_OrderUpdate:
		c.trackTypeReceived("userdata.order_update")
		c.log.Debugw("Order update received (not implemented yet)",
			"order_id", event.OrderUpdate.OrderId,
			"symbol", event.OrderUpdate.Symbol,
			"status", event.OrderUpdate.Status,
		)
		c.trackTypeProcessed("userdata.order_update")
		// TODO: Implement order update handling

	case *eventspb.UserDataEventWrapper_BalanceUpdate:
		c.trackTypeReceived("userdata.balance_update")
		c.log.Debugw("Balance update received",
			"user_id", event.BalanceUpdate.Base.UserId,
			"asset", event.BalanceUpdate.Asset,
			"wallet_balance", event.BalanceUpdate.WalletBalance,
		)
		c.trackTypeProcessed("userdata.balance_update")
		// TODO: Store balance updates if needed

	case *eventspb.UserDataEventWrapper_MarginCall:
		c.trackTypeReceived("userdata.margin_call")
		c.log.Errorw("⚠️ MARGIN CALL RECEIVED",
			"user_id", event.MarginCall.Base.UserId,
			"account_id", event.MarginCall.AccountId,
			"positions_at_risk", len(event.MarginCall.PositionsAtRisk),
		)
		c.trackTypeProcessed("userdata.margin_call")
		// TODO: Implement emergency actions (circuit breaker, telegram alert)

	case *eventspb.UserDataEventWrapper_AccountConfig:
		c.trackTypeReceived("userdata.account_config")
		c.log.Debugw("Account config updated",
			"user_id", event.AccountConfig.Base.UserId,
			"symbol", event.AccountConfig.Symbol,
			"leverage", event.AccountConfig.Leverage,
		)
		c.trackTypeProcessed("userdata.account_config")
		// TODO: Store leverage changes

	default:
		c.log.Debugw("Unknown User Data event in wrapper")
		c.incrementStat(&c.totalSkipped)
		return nil
	}

	return nil
}

// handlePositionUpdateTyped processes position update events using position service
func (c *WebSocketConsumer) handlePositionUpdateTyped(ctx context.Context, event *eventspb.UserDataPositionUpdateEvent) error {
	// Parse event into service update struct
	update, err := positionsvc.ParsePositionUpdateFromEvent(event)
	if err != nil {
		return errors.Wrap(err, "failed to parse position update")
	}

	c.log.Debugw("Processing position update",
		"user_id", update.UserID,
		"account_id", update.AccountID,
		"symbol", update.Symbol,
		"side", update.Side,
		"amount", update.Amount,
	)

	// Call service to handle update
	if err := c.positionService.UpdateFromWebSocket(ctx, update); err != nil {
		return errors.Wrap(err, "failed to update position")
	}

	c.incrementStat(&c.totalProcessed)
	c.incrementStat(&c.processedSinceLastLog)

	return nil
}

// handleKlineTyped processes kline/candlestick events from typed protobuf message
func (c *WebSocketConsumer) handleKlineTyped(ctx context.Context, event *eventspb.WebSocketKlineEvent) error {
	// Parse helper
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	marketType := event.MarketType
	if marketType == "" {
		marketType = "futures"
	}

	ohlcv := market_data.OHLCV{
		Exchange:    event.Exchange,
		Symbol:      event.Symbol,
		Timeframe:   event.Interval,
		MarketType:  marketType,
		OpenTime:    event.OpenTime.AsTime(),
		CloseTime:   event.CloseTime.AsTime(),
		Open:        parseFloat(event.Open),
		High:        parseFloat(event.High),
		Low:         parseFloat(event.Low),
		Close:       parseFloat(event.Close),
		Volume:      parseFloat(event.Volume),
		QuoteVolume: parseFloat(event.QuoteVolume),
		Trades:      uint64(event.TradeCount),
		IsClosed:    event.IsFinal,
		EventTime:   event.EventTime.AsTime(),
	}

	c.addToBatch(ohlcv)
	c.trackTypeProcessed("websocket.kline")
	return nil
}

// handleTickerTyped processes 24hr ticker events from typed protobuf message
func (c *WebSocketConsumer) handleTickerTyped(ctx context.Context, event *eventspb.WebSocketTickerEvent) error {
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	marketType := event.MarketType
	if marketType == "" {
		marketType = "spot"
	}

	ticker := market_data.Ticker{
		Exchange:           event.Exchange,
		Symbol:             event.Symbol,
		MarketType:         marketType,
		Timestamp:          time.Now(),
		LastPrice:          parseFloat(event.LastPrice),
		OpenPrice:          parseFloat(event.OpenPrice),
		HighPrice:          parseFloat(event.HighPrice),
		LowPrice:           parseFloat(event.LowPrice),
		Volume:             parseFloat(event.Volume),
		QuoteVolume:        parseFloat(event.QuoteVolume),
		PriceChange:        parseFloat(event.PriceChange),
		PriceChangePercent: parseFloat(event.PriceChangePercent),
		WeightedAvgPrice:   parseFloat(event.WeightedAvgPrice),
		TradeCount:         uint64(event.TradeCount),
		EventTime:          event.EventTime.AsTime(),
	}

	c.addToBatch(ticker)
	c.trackTypeProcessed("websocket.ticker")
	return nil
}

// handleDepthTyped processes order book depth events from typed protobuf message
func (c *WebSocketConsumer) handleDepthTyped(ctx context.Context, event *eventspb.WebSocketDepthEvent) error {
	// Convert bids and asks to JSON
	// Note: OrderBookSnapshot stores bids/asks as JSON strings in ClickHouse
	bidsJSON := "[]"
	asksJSON := "[]"

	// Simple JSON serialization for order book levels
	// TODO: Use proper JSON marshaling if needed

	marketType := event.MarketType
	if marketType == "" {
		marketType = "spot"
	}

	snapshot := market_data.OrderBookSnapshot{
		Exchange:   event.Exchange,
		Symbol:     event.Symbol,
		MarketType: marketType,
		Timestamp:  time.Now(),
		Bids:       bidsJSON,
		Asks:       asksJSON,
		BidDepth:   0.0, // Calculate if needed
		AskDepth:   0.0, // Calculate if needed
		EventTime:  event.EventTime.AsTime(),
	}

	c.addToBatch(snapshot)
	c.trackTypeProcessed("websocket.depth")
	return nil
}

// handleTradeTyped processes trade events from typed protobuf message
func (c *WebSocketConsumer) handleTradeTyped(ctx context.Context, event *eventspb.WebSocketTradeEvent) error {
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	marketType := event.MarketType
	if marketType == "" {
		marketType = "spot"
	}

	trade := market_data.Trade{
		Exchange:     event.Exchange,
		Symbol:       event.Symbol,
		MarketType:   marketType,
		Timestamp:    time.Now(),
		TradeID:      event.TradeId,
		AggTradeID:   event.TradeId, // Use same ID for now
		Price:        parseFloat(event.Price),
		Quantity:     parseFloat(event.Quantity),
		FirstTradeID: event.TradeId,
		LastTradeID:  event.TradeId,
		IsBuyerMaker: event.IsBuyerMaker,
		EventTime:    event.EventTime.AsTime(),
	}

	c.addToBatch(trade)
	c.trackTypeProcessed("websocket.trade")
	return nil
}

// handleMarkPriceTyped processes mark price events from typed protobuf message
func (c *WebSocketConsumer) handleMarkPriceTyped(ctx context.Context, event *eventspb.WebSocketMarkPriceEvent) error {
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	markPrice := market_data.MarkPrice{
		Exchange:             event.Exchange,
		Symbol:               event.Symbol,
		MarketType:           "futures",
		Timestamp:            time.Now(),
		MarkPrice:            parseFloat(event.MarkPrice),
		IndexPrice:           parseFloat(event.IndexPrice),
		EstimatedSettlePrice: parseFloat(event.EstimatedSettlePrice),
		FundingRate:          parseFloat(event.FundingRate),
		NextFundingTime:      event.NextFundingTime.AsTime(),
		EventTime:            event.EventTime.AsTime(),
	}

	c.addToBatch(markPrice)
	c.trackTypeProcessed("websocket.mark_price")
	return nil
}

// handleLiquidationTyped processes liquidation events from typed protobuf message
func (c *WebSocketConsumer) handleLiquidationTyped(ctx context.Context, event *eventspb.WebSocketLiquidationEvent) error {
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	marketType := event.MarketType
	if marketType == "" {
		marketType = "futures"
	}

	liquidation := market_data.Liquidation{
		Exchange:   event.Exchange,
		Symbol:     event.Symbol,
		MarketType: marketType,
		Timestamp:  time.Now(),
		Side:       event.Side,
		OrderType:  event.OrderType,
		Price:      parseFloat(event.Price),
		Quantity:   parseFloat(event.Quantity),
		Value:      parseFloat(event.Value),
		EventTime:  event.EventTime.AsTime(),
	}

	c.addToBatch(liquidation)
	c.trackTypeProcessed("websocket.liquidation")
	return nil
}

// addToBatch adds an item to batch (thread-safe)
func (c *WebSocketConsumer) addToBatch(item interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.batch = append(c.batch, item)
	c.incrementStat(&c.totalProcessed)
	c.incrementStat(&c.processedSinceLastLog)

	// Auto-flush if batch is full
	if len(c.batch) >= wsBatchSize {
		c.log.Infow("📦 Batch full, triggering flush",
			"batch_size", len(c.batch),
		)
		go c.FlushBatch(context.Background()) // Non-blocking flush
	}
}

// FlushBatch implements BatchConsumer interface
func (c *WebSocketConsumer) FlushBatch(ctx context.Context) error {
	c.mu.Lock()
	if len(c.batch) == 0 {
		c.mu.Unlock()
		return nil
	}

	// Copy batch and reset
	batchCopy := make([]interface{}, len(c.batch))
	copy(batchCopy, c.batch)
	originalSize := len(c.batch)
	c.batch = c.batch[:0] // Reset batch
	c.lastFlushTime = time.Now()
	c.mu.Unlock()

	start := time.Now()

	// Deduplicate batch before insert
	dedupedBatch := c.deduplicateBatch(batchCopy)
	dedupCount := originalSize - len(dedupedBatch)

	if dedupCount > 0 {
		c.statsMu.Lock()
		c.totalDeduplicated += int64(dedupCount)
		c.statsMu.Unlock()

		c.log.Infow("🔄 Deduplicated batch",
			"original", originalSize,
			"removed", dedupCount,
			"final", len(dedupedBatch),
		)
	}

	// Group items by type and insert
	if err := c.insertMixedBatch(ctx, dedupedBatch); err != nil {
		c.log.Error("Failed to flush batch", "error", err)
		return err
	}

	duration := time.Since(start)
	c.log.Infow("✅ Flushed mixed batch to ClickHouse",
		"size", len(dedupedBatch),
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

// insertMixedBatch groups items by type and inserts them
func (c *WebSocketConsumer) insertMixedBatch(ctx context.Context, batch []interface{}) error {
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
		if err := c.repository.InsertOHLCV(ctx, klines); err != nil {
			return errors.Wrap(err, "insert klines")
		}
		c.log.Infow("  → Inserted klines", "count", len(klines))
	}

	if len(tickers) > 0 {
		if err := c.repository.InsertTicker(ctx, tickers); err != nil {
			return errors.Wrap(err, "insert tickers")
		}
		c.log.Infow("  → Inserted tickers", "count", len(tickers))
	}

	if len(depths) > 0 {
		if err := c.repository.InsertOrderBook(ctx, depths); err != nil {
			return errors.Wrap(err, "insert depths")
		}
		c.log.Infow("  → Inserted order book snapshots", "count", len(depths))
	}

	if len(trades) > 0 {
		if err := c.repository.InsertTrades(ctx, trades); err != nil {
			return errors.Wrap(err, "insert trades")
		}
		c.log.Infow("  → Inserted trades", "count", len(trades))
	}

	if len(markPrices) > 0 {
		if err := c.repository.InsertMarkPrice(ctx, markPrices); err != nil {
			return errors.Wrap(err, "insert mark prices")
		}
		c.log.Infow("  → Inserted mark prices", "count", len(markPrices))
	}

	if len(liquidations) > 0 {
		if err := c.repository.InsertLiquidations(ctx, liquidations); err != nil {
			return errors.Wrap(err, "insert liquidations")
		}
		c.log.Infow("  → Inserted liquidations", "count", len(liquidations))
	}

	return nil
}

// deduplicateBatch removes duplicate events based on unique keys
// For each event type, keeps only the latest version based on event_time
func (c *WebSocketConsumer) deduplicateBatch(batch []interface{}) []interface{} {
	if len(batch) == 0 {
		return batch
	}

	// Separate maps for each type to maintain type-specific deduplication
	klinesSeen := make(map[string]market_data.OHLCV)
	tickersSeen := make(map[string]market_data.Ticker)
	depthsSeen := make(map[string]market_data.OrderBookSnapshot)
	tradesSeen := make(map[string]market_data.Trade)
	markPricesSeen := make(map[string]market_data.MarkPrice)
	liquidationsSeen := make(map[string]market_data.Liquidation)

	// Process each item and deduplicate
	for _, item := range batch {
		switch v := item.(type) {
		case market_data.OHLCV:
			key := getKlineKey(v)
			if existing, exists := klinesSeen[key]; exists {
				// Keep the one with latest event_time
				if v.EventTime.After(existing.EventTime) {
					klinesSeen[key] = v
				}
			} else {
				klinesSeen[key] = v
			}

		case market_data.Ticker:
			key := getTickerKey(v)
			if existing, exists := tickersSeen[key]; exists {
				if v.EventTime.After(existing.EventTime) {
					tickersSeen[key] = v
				}
			} else {
				tickersSeen[key] = v
			}

		case market_data.OrderBookSnapshot:
			key := getDepthKey(v)
			if existing, exists := depthsSeen[key]; exists {
				if v.EventTime.After(existing.EventTime) {
					depthsSeen[key] = v
				}
			} else {
				depthsSeen[key] = v
			}

		case market_data.Trade:
			key := getTradeKey(v)
			if existing, exists := tradesSeen[key]; exists {
				if v.EventTime.After(existing.EventTime) {
					tradesSeen[key] = v
				}
			} else {
				tradesSeen[key] = v
			}

		case market_data.MarkPrice:
			key := getMarkPriceKey(v)
			if existing, exists := markPricesSeen[key]; exists {
				if v.EventTime.After(existing.EventTime) {
					markPricesSeen[key] = v
				}
			} else {
				markPricesSeen[key] = v
			}

		case market_data.Liquidation:
			key := getLiquidationKey(v)
			if existing, exists := liquidationsSeen[key]; exists {
				if v.EventTime.After(existing.EventTime) {
					liquidationsSeen[key] = v
				}
			} else {
				liquidationsSeen[key] = v
			}
		}
	}

	// Rebuild deduplicated batch
	result := make([]interface{}, 0, len(batch))
	for _, v := range klinesSeen {
		result = append(result, v)
	}
	for _, v := range tickersSeen {
		result = append(result, v)
	}
	for _, v := range depthsSeen {
		result = append(result, v)
	}
	for _, v := range tradesSeen {
		result = append(result, v)
	}
	for _, v := range markPricesSeen {
		result = append(result, v)
	}
	for _, v := range liquidationsSeen {
		result = append(result, v)
	}

	return result
}

// Deduplication key generators for each type

func getKlineKey(item market_data.OHLCV) string {
	// exchange_symbol_interval_opentime
	return item.Exchange + "_" + item.Symbol + "_" + item.Timeframe + "_" +
		strconv.FormatInt(item.OpenTime.Unix(), 10)
}

func getTickerKey(item market_data.Ticker) string {
	// exchange_symbol_markettype_timestamp (rounded to second)
	return item.Exchange + "_" + item.Symbol + "_" + item.MarketType + "_" +
		strconv.FormatInt(item.Timestamp.Unix(), 10)
}

func getDepthKey(item market_data.OrderBookSnapshot) string {
	// exchange_symbol_markettype_timestamp (rounded to second)
	// Order book snapshots are frequent, keep only latest per second
	return item.Exchange + "_" + item.Symbol + "_" + item.MarketType + "_" +
		strconv.FormatInt(item.Timestamp.Unix(), 10)
}

func getTradeKey(item market_data.Trade) string {
	// exchange_symbol_tradeid (trades are unique by ID)
	return item.Exchange + "_" + item.Symbol + "_" +
		strconv.FormatInt(item.TradeID, 10)
}

func getMarkPriceKey(item market_data.MarkPrice) string {
	// exchange_symbol_timestamp (rounded to second)
	return item.Exchange + "_" + item.Symbol + "_" +
		strconv.FormatInt(item.Timestamp.Unix(), 10)
}

func getLiquidationKey(item market_data.Liquidation) string {
	// exchange_symbol_side_timestamp_price (liquidations are relatively rare)
	// Use price as part of key since multiple liquidations can happen at same time
	return item.Exchange + "_" + item.Symbol + "_" + item.Side + "_" +
		strconv.FormatInt(item.Timestamp.UnixMilli(), 10) + "_" +
		strconv.FormatFloat(item.Price, 'f', 2, 64)
}

// LogStats implements BatchConsumer interface
func (c *WebSocketConsumer) LogStats(final bool) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	timeSinceLastFlush := time.Since(c.lastFlushTime)
	timeSinceLastLog := time.Since(c.lastStatsLogTime)

	receivedPerSec := int64(0)
	processedPerSec := int64(0)
	if timeSinceLastLog.Seconds() > 0 {
		receivedPerSec = int64(float64(c.receivedSinceLastLog) / timeSinceLastLog.Seconds())
		processedPerSec = int64(float64(c.processedSinceLastLog) / timeSinceLastLog.Seconds())
	}

	c.mu.Lock()
	pendingBatch := len(c.batch)
	c.mu.Unlock()

	stats := map[string]interface{}{
		"total_received":        c.totalReceived,
		"total_processed":       c.totalProcessed,
		"total_deduplicated":    c.totalDeduplicated,
		"total_skipped":         c.totalSkipped,
		"total_errors":          c.totalErrors,
		"pending_batch":         pendingBatch,
		"received_per_sec":      receivedPerSec,
		"processed_per_sec":     processedPerSec,
		"time_since_last_flush": timeSinceLastFlush.Round(time.Second).String(),
	}

	// Add per-type stats
	for eventType, typeStats := range c.typeStats {
		stats[eventType+"_received"] = typeStats.received
		stats[eventType+"_processed"] = typeStats.processed
		if typeStats.errors > 0 {
			stats[eventType+"_errors"] = typeStats.errors
		}
	}

	// Convert map to flat key-value list for structured logging
	args := make([]interface{}, 0, len(stats)*2)
	for k, v := range stats {
		args = append(args, k, v)
	}

	if final {
		c.log.Infow("📊 [FINAL] WebSocket Consumer stats", args...)
	} else {
		c.log.Infow("📊 WebSocket Consumer stats", args...)
	}

	// Reset interval counters
	c.receivedSinceLastLog = 0
	c.processedSinceLastLog = 0
	c.lastStatsLogTime = time.Now()
}

// Helper methods for statistics

func (c *WebSocketConsumer) incrementStat(stat *int64) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	*stat++
}

func (c *WebSocketConsumer) trackTypeReceived(eventType string) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	if c.typeStats[eventType] == nil {
		c.typeStats[eventType] = &eventTypeStats{}
	}
	c.typeStats[eventType].received++
}

func (c *WebSocketConsumer) trackTypeProcessed(eventType string) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	if c.typeStats[eventType] == nil {
		c.typeStats[eventType] = &eventTypeStats{}
	}
	c.typeStats[eventType].processed++
}

func (c *WebSocketConsumer) trackTypeError(eventType string) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	if c.typeStats[eventType] == nil {
		c.typeStats[eventType] = &eventTypeStats{}
	}
	c.typeStats[eventType].errors++
}
