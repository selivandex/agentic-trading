package events

import (
	"context"
	"time"

	"prometheus/internal/adapters/kafka"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// WebSocketPublisher provides methods for publishing WebSocket market data events
type WebSocketPublisher struct {
	kafka *kafka.Producer
}

// NewWebSocketPublisher creates a new WebSocket event publisher
func NewWebSocketPublisher(kafka *kafka.Producer) *WebSocketPublisher {
	return &WebSocketPublisher{kafka: kafka}
}

// PublishKline publishes a kline/candlestick event from WebSocket
func (wp *WebSocketPublisher) PublishKline(
	ctx context.Context,
	exchange, symbol, marketType, interval string,
	openTime, closeTime, eventTime time.Time,
	open, high, low, close, volume, quoteVolume string,
	tradeCount int64,
	isFinal bool,
) error {
	event := &eventspb.WebSocketKlineEvent{
		Base:        NewBaseEvent(TopicWebSocketKline, "websocket_"+exchange, ""),
		Exchange:    exchange,
		Symbol:      symbol,
		MarketType:  marketType,
		Interval:    interval,
		OpenTime:    timestamppb.New(openTime),
		CloseTime:   timestamppb.New(closeTime),
		Open:        open,
		High:        high,
		Low:         low,
		Close:       close,
		Volume:      volume,
		QuoteVolume: quoteVolume,
		TradeCount:  tradeCount,
		IsFinal:     isFinal,
		EventTime:   timestamppb.New(eventTime),
	}

	return wp.publishProto(ctx, TopicWebSocketKline, symbol, event)
}

// PublishTicker publishes a 24hr ticker statistics event from WebSocket
func (wp *WebSocketPublisher) PublishTicker(
	ctx context.Context,
	exchange, symbol, marketType string,
	priceChange, priceChangePercent, weightedAvgPrice string,
	lastPrice, lastQty, openPrice, highPrice, lowPrice string,
	volume, quoteVolume string,
	openTime, closeTime, eventTime time.Time,
	firstTradeID, lastTradeID, tradeCount int64,
) error {
	event := &eventspb.WebSocketTickerEvent{
		Base:               NewBaseEvent(TopicWebSocketTicker, "websocket_"+exchange, ""),
		Exchange:           exchange,
		Symbol:             symbol,
		MarketType:         marketType,
		PriceChange:        priceChange,
		PriceChangePercent: priceChangePercent,
		WeightedAvgPrice:   weightedAvgPrice,
		LastPrice:          lastPrice,
		LastQty:            lastQty,
		OpenPrice:          openPrice,
		HighPrice:          highPrice,
		LowPrice:           lowPrice,
		Volume:             volume,
		QuoteVolume:        quoteVolume,
		OpenTime:           timestamppb.New(openTime),
		CloseTime:          timestamppb.New(closeTime),
		FirstTradeId:       firstTradeID,
		LastTradeId:        lastTradeID,
		TradeCount:         tradeCount,
		EventTime:          timestamppb.New(eventTime),
	}

	return wp.publishProto(ctx, TopicWebSocketTicker, symbol, event)
}

// PublishDepth publishes an order book depth event from WebSocket
func (wp *WebSocketPublisher) PublishDepth(
	ctx context.Context,
	exchange, symbol string,
	bids, asks []PriceLevel,
	lastUpdateID int64,
	eventTime time.Time,
) error {
	// Convert to protobuf PriceLevel
	pbBids := make([]*eventspb.PriceLevel, len(bids))
	for i, bid := range bids {
		pbBids[i] = &eventspb.PriceLevel{
			Price:    bid.Price,
			Quantity: bid.Quantity,
		}
	}

	pbAsks := make([]*eventspb.PriceLevel, len(asks))
	for i, ask := range asks {
		pbAsks[i] = &eventspb.PriceLevel{
			Price:    ask.Price,
			Quantity: ask.Quantity,
		}
	}

	event := &eventspb.WebSocketDepthEvent{
		Base:         NewBaseEvent(TopicWebSocketDepth, "websocket_"+exchange, ""),
		Exchange:     exchange,
		Symbol:       symbol,
		Bids:         pbBids,
		Asks:         pbAsks,
		LastUpdateId: lastUpdateID,
		EventTime:    timestamppb.New(eventTime),
	}

	return wp.publishProto(ctx, TopicWebSocketDepth, symbol, event)
}

// PriceLevel represents a price level for order book
type PriceLevel struct {
	Price    string
	Quantity string
}

// PublishTrade publishes a single trade event from WebSocket
func (wp *WebSocketPublisher) PublishTrade(
	ctx context.Context,
	exchange, symbol, marketType string,
	tradeID int64,
	price, quantity string,
	buyerOrderID, sellerOrderID int64,
	tradeTime, eventTime time.Time,
	isBuyerMaker bool,
) error {
	event := &eventspb.WebSocketTradeEvent{
		Base:          NewBaseEvent(TopicWebSocketTrade, "websocket_"+exchange, ""),
		Exchange:      exchange,
		Symbol:        symbol,
		MarketType:    marketType,
		TradeId:       tradeID,
		Price:         price,
		Quantity:      quantity,
		BuyerOrderId:  buyerOrderID,
		SellerOrderId: sellerOrderID,
		TradeTime:     timestamppb.New(tradeTime),
		IsBuyerMaker:  isBuyerMaker,
		EventTime:     timestamppb.New(eventTime),
	}

	return wp.publishProto(ctx, TopicWebSocketTrade, symbol, event)
}

// PublishFundingRate publishes a funding rate event from WebSocket
func (wp *WebSocketPublisher) PublishFundingRate(
	ctx context.Context,
	exchange, symbol string,
	fundingRate string,
	fundingTime, nextFundingTime, eventTime time.Time,
) error {
	event := &eventspb.WebSocketFundingRateEvent{
		Base:            NewBaseEvent(TopicWebSocketFundingRate, "websocket_"+exchange, ""),
		Exchange:        exchange,
		Symbol:          symbol,
		FundingRate:     fundingRate,
		FundingTime:     timestamppb.New(fundingTime),
		NextFundingTime: timestamppb.New(nextFundingTime),
		EventTime:       timestamppb.New(eventTime),
	}

	return wp.publishProto(ctx, TopicWebSocketFundingRate, symbol, event)
}

// PublishMarkPrice publishes a mark price event from WebSocket
func (wp *WebSocketPublisher) PublishMarkPrice(
	ctx context.Context,
	exchange, symbol string,
	markPrice, indexPrice, estimatedSettlePrice, fundingRate string,
	nextFundingTime, eventTime time.Time,
) error {
	event := &eventspb.WebSocketMarkPriceEvent{
		Base:                 NewBaseEvent(TopicWebSocketMarkPrice, "websocket_"+exchange, ""),
		Exchange:             exchange,
		Symbol:               symbol,
		MarkPrice:            markPrice,
		IndexPrice:           indexPrice,
		EstimatedSettlePrice: estimatedSettlePrice,
		FundingRate:          fundingRate,
		NextFundingTime:      timestamppb.New(nextFundingTime),
		EventTime:            timestamppb.New(eventTime),
	}

	return wp.publishProto(ctx, TopicWebSocketMarkPrice, symbol, event)
}

// publishProto serializes and publishes a protobuf message
func (wp *WebSocketPublisher) publishProto(ctx context.Context, topic, key string, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshal protobuf")
	}

	if err := wp.kafka.PublishBinary(ctx, topic, []byte(key), data); err != nil {
		return errors.Wrap(err, "publish to kafka")
	}

	return nil
}
