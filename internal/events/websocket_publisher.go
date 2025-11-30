package events

import (
	"context"
	"sync/atomic"
	"time"

	"prometheus/internal/adapters/kafka"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// WebSocketPublisher provides methods for publishing WebSocket market data events
type WebSocketPublisher struct {
	kafka    *kafka.Producer
	stopping atomic.Bool // Flag to prevent publishing during shutdown
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
		Base:        NewBaseEvent("websocket.kline", "websocket_"+exchange, ""),
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

	// Wrap in envelope for efficient type detection by consumer
	wrapper := &eventspb.WebSocketEventWrapper{
		Event: &eventspb.WebSocketEventWrapper_Kline{
			Kline: event,
		},
	}

	return wp.publishProto(ctx, TopicWebSocketEvents, symbol, wrapper)
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
		Base:               NewBaseEvent("websocket.ticker", "websocket_"+exchange, ""),
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

	wrapper := &eventspb.WebSocketEventWrapper{
		Event: &eventspb.WebSocketEventWrapper_Ticker{
			Ticker: event,
		},
	}

	return wp.publishProto(ctx, TopicWebSocketEvents, symbol, wrapper)
}

// PublishDepth publishes an order book depth event from WebSocket
func (wp *WebSocketPublisher) PublishDepth(
	ctx context.Context,
	exchange, symbol, marketType string,
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
		Base:         NewBaseEvent("websocket.depth", "websocket_"+exchange, ""),
		Exchange:     exchange,
		Symbol:       symbol,
		MarketType:   marketType,
		Bids:         pbBids,
		Asks:         pbAsks,
		LastUpdateId: lastUpdateID,
		EventTime:    timestamppb.New(eventTime),
	}

	wrapper := &eventspb.WebSocketEventWrapper{
		Event: &eventspb.WebSocketEventWrapper_Depth{
			Depth: event,
		},
	}

	return wp.publishProto(ctx, TopicWebSocketEvents, symbol, wrapper)
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
		Base:          NewBaseEvent("websocket.trade", "websocket_"+exchange, ""),
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

	wrapper := &eventspb.WebSocketEventWrapper{
		Event: &eventspb.WebSocketEventWrapper_Trade{
			Trade: event,
		},
	}

	return wp.publishProto(ctx, TopicWebSocketEvents, symbol, wrapper)
}

// PublishFundingRate publishes a funding rate event from WebSocket
func (wp *WebSocketPublisher) PublishFundingRate(
	ctx context.Context,
	exchange, symbol string,
	fundingRate string,
	fundingTime, nextFundingTime, eventTime time.Time,
) error {
	event := &eventspb.WebSocketFundingRateEvent{
		Base:            NewBaseEvent("websocket.funding_rate", "websocket_"+exchange, ""),
		Exchange:        exchange,
		Symbol:          symbol,
		FundingRate:     fundingRate,
		FundingTime:     timestamppb.New(fundingTime),
		NextFundingTime: timestamppb.New(nextFundingTime),
		EventTime:       timestamppb.New(eventTime),
	}

	wrapper := &eventspb.WebSocketEventWrapper{
		Event: &eventspb.WebSocketEventWrapper_FundingRate{
			FundingRate: event,
		},
	}

	return wp.publishProto(ctx, TopicWebSocketEvents, symbol, wrapper)
}

// PublishMarkPrice publishes a mark price event from WebSocket
func (wp *WebSocketPublisher) PublishMarkPrice(
	ctx context.Context,
	exchange, symbol string,
	markPrice, indexPrice, estimatedSettlePrice, fundingRate string,
	nextFundingTime, eventTime time.Time,
) error {
	event := &eventspb.WebSocketMarkPriceEvent{
		Base:                 NewBaseEvent("websocket.mark_price", "websocket_"+exchange, ""),
		Exchange:             exchange,
		Symbol:               symbol,
		MarkPrice:            markPrice,
		IndexPrice:           indexPrice,
		EstimatedSettlePrice: estimatedSettlePrice,
		FundingRate:          fundingRate,
		NextFundingTime:      timestamppb.New(nextFundingTime),
		EventTime:            timestamppb.New(eventTime),
	}

	wrapper := &eventspb.WebSocketEventWrapper{
		Event: &eventspb.WebSocketEventWrapper_MarkPrice{
			MarkPrice: event,
		},
	}

	return wp.publishProto(ctx, TopicWebSocketEvents, symbol, wrapper)
}

// PublishLiquidation publishes a liquidation event from WebSocket (futures only)
func (wp *WebSocketPublisher) PublishLiquidation(
	ctx context.Context,
	exchange, symbol, marketType, side, orderType string,
	price, quantity, value string,
	eventTime time.Time,
) error {
	event := &eventspb.WebSocketLiquidationEvent{
		Base:       NewBaseEvent("websocket.liquidation", "websocket_"+exchange, ""),
		Exchange:   exchange,
		Symbol:     symbol,
		MarketType: marketType,
		Side:       side,
		OrderType:  orderType,
		Price:      price,
		Quantity:   quantity,
		Value:      value,
		EventTime:  timestamppb.New(eventTime),
	}

	wrapper := &eventspb.WebSocketEventWrapper{
		Event: &eventspb.WebSocketEventWrapper_Liquidation{
			Liquidation: event,
		},
	}

	return wp.publishProto(ctx, TopicWebSocketEvents, symbol, wrapper)
}

// ========================================
// User Data Events (Orders, Positions, Balance, Margin Calls)
// ========================================

// PublishOrderUpdate publishes a UserDataOrderUpdateEvent to Kafka using wrapper
func (wp *WebSocketPublisher) PublishOrderUpdate(
	ctx context.Context,
	userID, accountID string,
	exchange, orderID, clientOrderID, symbol, side, positionSide, orderType, status, executionType string,
	originalQty, filledQty, avgPrice, stopPrice, lastFilledQty, lastFilledPrice, commission, commissionAsset string,
	tradeTime, eventTime time.Time,
) error {
	event := &eventspb.UserDataOrderUpdateEvent{
		Base:            NewBaseEvent("user_data.order_update", "user_data_websocket", userID),
		AccountId:       accountID,
		Exchange:        exchange,
		OrderId:         orderID,
		ClientOrderId:   clientOrderID,
		Symbol:          symbol,
		Side:            side,
		PositionSide:    positionSide,
		Type:            orderType,
		Status:          status,
		ExecutionType:   executionType,
		OriginalQty:     originalQty,
		FilledQty:       filledQty,
		AvgPrice:        avgPrice,
		StopPrice:       stopPrice,
		LastFilledQty:   lastFilledQty,
		LastFilledPrice: lastFilledPrice,
		Commission:      commission,
		CommissionAsset: commissionAsset,
		TradeTime:       timestamppb.New(tradeTime),
		EventTime:       timestamppb.New(eventTime),
	}

	wrapper := &eventspb.UserDataEventWrapper{
		Event: &eventspb.UserDataEventWrapper_OrderUpdate{
			OrderUpdate: event,
		},
	}

	return wp.publishProto(ctx, TopicWebSocketEvents, userID, wrapper)
}

// PublishPositionUpdate publishes a UserDataPositionUpdateEvent to Kafka using wrapper
func (wp *WebSocketPublisher) PublishPositionUpdate(
	ctx context.Context,
	userID, accountID string,
	exchange, symbol, side, amount, entryPrice, markPrice, unrealizedPnL, maintenanceMargin, positionSide string,
	eventTime time.Time,
) error {
	event := &eventspb.UserDataPositionUpdateEvent{
		Base:              NewBaseEvent("user_data.position_update", "user_data_websocket", userID),
		AccountId:         accountID,
		Exchange:          exchange,
		Symbol:            symbol,
		Side:              side,
		Amount:            amount,
		EntryPrice:        entryPrice,
		MarkPrice:         markPrice,
		UnrealizedPnl:     unrealizedPnL,
		MaintenanceMargin: maintenanceMargin,
		PositionSide:      positionSide,
		EventTime:         timestamppb.New(eventTime),
	}

	wrapper := &eventspb.UserDataEventWrapper{
		Event: &eventspb.UserDataEventWrapper_PositionUpdate{
			PositionUpdate: event,
		},
	}

	return wp.publishProto(ctx, TopicWebSocketEvents, userID, wrapper)
}

// PublishBalanceUpdate publishes a UserDataBalanceUpdateEvent to Kafka using wrapper
func (wp *WebSocketPublisher) PublishBalanceUpdate(
	ctx context.Context,
	userID, accountID string,
	exchange, asset, walletBalance, crossWalletBalance, availableBalance, reasonType string,
	eventTime time.Time,
) error {
	event := &eventspb.UserDataBalanceUpdateEvent{
		Base:               NewBaseEvent("user_data.balance_update", "user_data_websocket", userID),
		AccountId:          accountID,
		Exchange:           exchange,
		Asset:              asset,
		WalletBalance:      walletBalance,
		CrossWalletBalance: crossWalletBalance,
		AvailableBalance:   availableBalance,
		ReasonType:         reasonType,
		EventTime:          timestamppb.New(eventTime),
	}

	wrapper := &eventspb.UserDataEventWrapper{
		Event: &eventspb.UserDataEventWrapper_BalanceUpdate{
			BalanceUpdate: event,
		},
	}

	return wp.publishProto(ctx, TopicWebSocketEvents, userID, wrapper)
}

// PositionAtRiskData represents a position that may be liquidated
type PositionAtRiskData struct {
	Symbol            string
	Side              string
	Amount            string
	MarginType        string
	UnrealizedPnL     string
	MaintenanceMargin string
	PositionSide      string
}

// PublishMarginCall publishes a UserDataMarginCallEvent to Kafka using wrapper (CRITICAL!)
func (wp *WebSocketPublisher) PublishMarginCall(
	ctx context.Context,
	userID, accountID string,
	exchange, crossWalletBalance string,
	positionsAtRisk []PositionAtRiskData,
	eventTime time.Time,
) error {
	// Convert positions at risk
	protoPositions := make([]*eventspb.PositionAtRisk, 0, len(positionsAtRisk))
	for _, pos := range positionsAtRisk {
		protoPositions = append(protoPositions, &eventspb.PositionAtRisk{
			Symbol:            pos.Symbol,
			Side:              pos.Side,
			Amount:            pos.Amount,
			MarginType:        pos.MarginType,
			UnrealizedPnl:     pos.UnrealizedPnL,
			MaintenanceMargin: pos.MaintenanceMargin,
			PositionSide:      pos.PositionSide,
		})
	}

	event := &eventspb.UserDataMarginCallEvent{
		Base:               NewBaseEvent("user_data.margin_call", "user_data_websocket", userID),
		AccountId:          accountID,
		Exchange:           exchange,
		CrossWalletBalance: crossWalletBalance,
		PositionsAtRisk:    protoPositions,
		EventTime:          timestamppb.New(eventTime),
	}

	wrapper := &eventspb.UserDataEventWrapper{
		Event: &eventspb.UserDataEventWrapper_MarginCall{
			MarginCall: event,
		},
	}

	return wp.publishProto(ctx, TopicWebSocketEvents, userID, wrapper)
}

// PublishAccountConfigUpdate publishes a UserDataAccountConfigEvent to Kafka using wrapper
func (wp *WebSocketPublisher) PublishAccountConfigUpdate(
	ctx context.Context,
	userID, accountID string,
	exchange, symbol string,
	leverage int,
	eventTime time.Time,
) error {
	event := &eventspb.UserDataAccountConfigEvent{
		Base:      NewBaseEvent("user_data.account_config_update", "user_data_websocket", userID),
		AccountId: accountID,
		Exchange:  exchange,
		Symbol:    symbol,
		Leverage:  int32(leverage),
		EventTime: timestamppb.New(eventTime),
	}

	wrapper := &eventspb.UserDataEventWrapper{
		Event: &eventspb.UserDataEventWrapper_AccountConfig{
			AccountConfig: event,
		},
	}

	return wp.publishProto(ctx, TopicWebSocketEvents, userID, wrapper)
}

// ========================================
// Internal helpers
// ========================================

// Shutdown sets the stopping flag to prevent new publications
func (wp *WebSocketPublisher) Shutdown() {
	wp.stopping.Store(true)
}

// publishProto serializes and publishes a protobuf message
func (wp *WebSocketPublisher) publishProto(ctx context.Context, topic, key string, msg proto.Message) error {
	// Check if we're shutting down - silently ignore new events
	if wp.stopping.Load() {
		return nil
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshal protobuf")
	}

	if err := wp.kafka.PublishBinary(ctx, topic, []byte(key), data); err != nil {
		return errors.Wrap(err, "publish to kafka")
	}

	return nil
}
