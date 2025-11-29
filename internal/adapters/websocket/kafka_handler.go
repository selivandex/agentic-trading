package websocket

import (
	"context"

	"prometheus/internal/events"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// KafkaEventHandler implements EventHandler and publishes events to Kafka via WebSocketPublisher
type KafkaEventHandler struct {
	publisher *events.WebSocketPublisher
	logger    *logger.Logger
}

// NewKafkaEventHandler creates a new Kafka event handler
func NewKafkaEventHandler(publisher *events.WebSocketPublisher, log *logger.Logger) *KafkaEventHandler {
	return &KafkaEventHandler{
		publisher: publisher,
		logger:    log,
	}
}

// OnKline handles kline events
func (h *KafkaEventHandler) OnKline(event *KlineEvent) error {
	err := h.publisher.PublishKline(
		context.Background(),
		event.Exchange,
		event.Symbol,
		event.MarketType,
		event.Interval,
		event.OpenTime,
		event.CloseTime,
		event.EventTime,
		event.Open,
		event.High,
		event.Low,
		event.Close,
		event.Volume,
		event.QuoteVolume,
		event.TradeCount,
		event.IsFinal,
	)
	if err != nil {
		return errors.Wrap(err, "failed to publish kline event")
	}

	h.logger.Debug("Published kline event to Kafka",
		"exchange", event.Exchange,
		"symbol", event.Symbol,
		"market_type", event.MarketType,
		"interval", event.Interval,
		"is_final", event.IsFinal,
	)

	return nil
}

// OnTicker handles ticker events
func (h *KafkaEventHandler) OnTicker(event *TickerEvent) error {
	err := h.publisher.PublishTicker(
		context.Background(),
		event.Exchange,
		event.Symbol,
		event.MarketType,
		event.PriceChange,
		event.PriceChangePercent,
		event.WeightedAvgPrice,
		event.LastPrice,
		event.LastQty,
		event.OpenPrice,
		event.HighPrice,
		event.LowPrice,
		event.Volume,
		event.QuoteVolume,
		event.OpenTime,
		event.CloseTime,
		event.EventTime,
		event.FirstTradeID,
		event.LastTradeID,
		event.TradeCount,
	)
	if err != nil {
		return errors.Wrap(err, "failed to publish ticker event")
	}

	h.logger.Debug("Published ticker event to Kafka",
		"exchange", event.Exchange,
		"symbol", event.Symbol,
		"market_type", event.MarketType,
	)

	return nil
}

// OnDepth handles depth events
func (h *KafkaEventHandler) OnDepth(event *DepthEvent) error {
	// Convert to events.PriceLevel format
	bids := make([]events.PriceLevel, len(event.Bids))
	for i, bid := range event.Bids {
		bids[i] = events.PriceLevel{
			Price:    bid.Price,
			Quantity: bid.Quantity,
		}
	}

	asks := make([]events.PriceLevel, len(event.Asks))
	for i, ask := range event.Asks {
		asks[i] = events.PriceLevel{
			Price:    ask.Price,
			Quantity: ask.Quantity,
		}
	}

	err := h.publisher.PublishDepth(
		context.Background(),
		event.Exchange,
		event.Symbol,
		event.MarketType,
		bids,
		asks,
		event.LastUpdateID,
		event.EventTime,
	)
	if err != nil {
		return errors.Wrap(err, "failed to publish depth event")
	}

	h.logger.Debug("Published depth event to Kafka",
		"exchange", event.Exchange,
		"symbol", event.Symbol,
	)

	return nil
}

// OnTrade handles trade events
func (h *KafkaEventHandler) OnTrade(event *TradeEvent) error {
	err := h.publisher.PublishTrade(
		context.Background(),
		event.Exchange,
		event.Symbol,
		event.MarketType,
		event.TradeID,
		event.Price,
		event.Quantity,
		event.BuyerOrderID,
		event.SellerOrderID,
		event.TradeTime,
		event.EventTime,
		event.IsBuyerMaker,
	)
	if err != nil {
		return errors.Wrap(err, "failed to publish trade event")
	}

	h.logger.Debug("Published trade event to Kafka",
		"exchange", event.Exchange,
		"symbol", event.Symbol,
		"market_type", event.MarketType,
	)

	return nil
}

// OnFundingRate handles funding rate events
func (h *KafkaEventHandler) OnFundingRate(event *FundingRateEvent) error {
	err := h.publisher.PublishFundingRate(
		context.Background(),
		event.Exchange,
		event.Symbol,
		event.FundingRate,
		event.FundingTime,
		event.NextFundingTime,
		event.EventTime,
	)
	if err != nil {
		return errors.Wrap(err, "failed to publish funding rate event")
	}

	h.logger.Debug("Published funding rate event to Kafka",
		"exchange", event.Exchange,
		"symbol", event.Symbol,
	)

	return nil
}

// OnMarkPrice handles mark price events
func (h *KafkaEventHandler) OnMarkPrice(event *MarkPriceEvent) error {
	err := h.publisher.PublishMarkPrice(
		context.Background(),
		event.Exchange,
		event.Symbol,
		event.MarkPrice,
		event.IndexPrice,
		event.EstimatedSettlePrice,
		event.FundingRate,
		event.NextFundingTime,
		event.EventTime,
	)
	if err != nil {
		return errors.Wrap(err, "failed to publish mark price event")
	}

	h.logger.Debug("Published mark price event to Kafka",
		"exchange", event.Exchange,
		"symbol", event.Symbol,
	)

	return nil
}

// OnError handles errors
func (h *KafkaEventHandler) OnError(err error) {
	h.logger.Error("WebSocket error",
		"error", err.Error(),
	)
}
