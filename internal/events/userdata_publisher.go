package events

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"prometheus/internal/adapters/kafka"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// UserDataPublisher publishes User Data WebSocket events to Kafka
type UserDataPublisher struct {
	producer *kafka.Producer
	logger   *logger.Logger
}

// NewUserDataPublisher creates a new User Data event publisher
func NewUserDataPublisher(producer *kafka.Producer, log *logger.Logger) *UserDataPublisher {
	return &UserDataPublisher{
		producer: producer,
		logger:   log,
	}
}

// PublishOrderUpdate publishes a UserDataOrderUpdateEvent to Kafka
func (p *UserDataPublisher) PublishOrderUpdate(
	ctx context.Context,
	userID, accountID uuid.UUID,
	exchange, orderID, clientOrderID, symbol, side, positionSide, orderType, status, executionType string,
	originalQty, filledQty, avgPrice, stopPrice, lastFilledQty, lastFilledPrice, commission, commissionAsset string,
	tradeTime, eventTime time.Time,
) error {
	event := &eventspb.UserDataOrderUpdateEvent{
		Base: &eventspb.BaseEvent{
			Id:        uuid.New().String(),
			Type:      "user_data.order_update",
			Timestamp: timestamppb.Now(),
			Source:    "user_data_websocket",
			UserId:    userID.String(),
			Version:   "1.0",
		},
		AccountId:       accountID.String(),
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

	data, err := proto.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "failed to marshal order update event")
	}

	if err := p.producer.PublishBinary(ctx, "user-data-order-updates", []byte(userID.String()), data); err != nil {
		return errors.Wrap(err, "failed to publish order update event")
	}

	p.logger.Debug("Published order update event",
		"user_id", userID,
		"account_id", accountID,
		"order_id", orderID,
		"symbol", symbol,
		"status", status,
	)

	return nil
}

// PublishPositionUpdate publishes a UserDataPositionUpdateEvent to Kafka
func (p *UserDataPublisher) PublishPositionUpdate(
	ctx context.Context,
	userID, accountID uuid.UUID,
	exchange, symbol, side, amount, entryPrice, markPrice, unrealizedPnL, maintenanceMargin, positionSide string,
	eventTime time.Time,
) error {
	event := &eventspb.UserDataPositionUpdateEvent{
		Base: &eventspb.BaseEvent{
			Id:        uuid.New().String(),
			Type:      "user_data.position_update",
			Timestamp: timestamppb.Now(),
			Source:    "user_data_websocket",
			UserId:    userID.String(),
			Version:   "1.0",
		},
		AccountId:         accountID.String(),
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

	data, err := proto.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "failed to marshal position update event")
	}

	if err := p.producer.PublishBinary(ctx, "user-data-position-updates", []byte(userID.String()), data); err != nil {
		return errors.Wrap(err, "failed to publish position update event")
	}

	p.logger.Debug("Published position update event",
		"user_id", userID,
		"account_id", accountID,
		"symbol", symbol,
		"side", side,
	)

	return nil
}

// PublishBalanceUpdate publishes a UserDataBalanceUpdateEvent to Kafka
func (p *UserDataPublisher) PublishBalanceUpdate(
	ctx context.Context,
	userID, accountID uuid.UUID,
	exchange, asset, walletBalance, crossWalletBalance, availableBalance, reasonType string,
	eventTime time.Time,
) error {
	event := &eventspb.UserDataBalanceUpdateEvent{
		Base: &eventspb.BaseEvent{
			Id:        uuid.New().String(),
			Type:      "user_data.balance_update",
			Timestamp: timestamppb.Now(),
			Source:    "user_data_websocket",
			UserId:    userID.String(),
			Version:   "1.0",
		},
		AccountId:          accountID.String(),
		Exchange:           exchange,
		Asset:              asset,
		WalletBalance:      walletBalance,
		CrossWalletBalance: crossWalletBalance,
		AvailableBalance:   availableBalance,
		ReasonType:         reasonType,
		EventTime:          timestamppb.New(eventTime),
	}

	data, err := proto.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "failed to marshal balance update event")
	}

	if err := p.producer.PublishBinary(ctx, "user-data-balance-updates", []byte(userID.String()), data); err != nil {
		return errors.Wrap(err, "failed to publish balance update event")
	}

	p.logger.Debug("Published balance update event",
		"user_id", userID,
		"account_id", accountID,
		"asset", asset,
		"reason", reasonType,
	)

	return nil
}

// PublishMarginCall publishes a UserDataMarginCallEvent to Kafka (CRITICAL!)
func (p *UserDataPublisher) PublishMarginCall(
	ctx context.Context,
	userID, accountID uuid.UUID,
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
		Base: &eventspb.BaseEvent{
			Id:        uuid.New().String(),
			Type:      "user_data.margin_call",
			Timestamp: timestamppb.Now(),
			Source:    "user_data_websocket",
			UserId:    userID.String(),
			Version:   "1.0",
		},
		AccountId:          accountID.String(),
		Exchange:           exchange,
		CrossWalletBalance: crossWalletBalance,
		PositionsAtRisk:    protoPositions,
		EventTime:          timestamppb.New(eventTime),
	}

	data, err := proto.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "failed to marshal margin call event")
	}

	if err := p.producer.PublishBinary(ctx, "user-data-margin-calls", []byte(userID.String()), data); err != nil {
		return errors.Wrap(err, "failed to publish margin call event")
	}

	p.logger.Error("⚠️ Published MARGIN CALL event",
		"user_id", userID,
		"account_id", accountID,
		"positions_at_risk", len(positionsAtRisk),
	)

	return nil
}

// PublishAccountConfigUpdate publishes a UserDataAccountConfigEvent to Kafka
func (p *UserDataPublisher) PublishAccountConfigUpdate(
	ctx context.Context,
	userID, accountID uuid.UUID,
	exchange, symbol string,
	leverage int,
	eventTime time.Time,
) error {
	event := &eventspb.UserDataAccountConfigEvent{
		Base: &eventspb.BaseEvent{
			Id:        uuid.New().String(),
			Type:      "user_data.account_config_update",
			Timestamp: timestamppb.Now(),
			Source:    "user_data_websocket",
			UserId:    userID.String(),
			Version:   "1.0",
		},
		AccountId: accountID.String(),
		Exchange:  exchange,
		Symbol:    symbol,
		Leverage:  int32(leverage),
		EventTime: timestamppb.New(eventTime),
	}

	data, err := proto.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "failed to marshal account config update event")
	}

	if err := p.producer.PublishBinary(ctx, "user-data-account-config", []byte(userID.String()), data); err != nil {
		return errors.Wrap(err, "failed to publish account config update event")
	}

	p.logger.Debug("Published account config update event",
		"user_id", userID,
		"account_id", accountID,
		"symbol", symbol,
		"leverage", leverage,
	)

	return nil
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
