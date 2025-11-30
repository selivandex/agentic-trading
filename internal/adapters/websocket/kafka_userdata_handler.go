package websocket

import (
	"context"

	"prometheus/internal/events"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// KafkaUserDataHandler implements UserDataEventHandler and publishes events to Kafka
// This is the bridge between WebSocket events and Kafka event pipeline
type KafkaUserDataHandler struct {
	publisher *events.WebSocketPublisher
	logger    *logger.Logger
}

// NewKafkaUserDataHandler creates a new Kafka event handler for User Data WebSocket
func NewKafkaUserDataHandler(publisher *events.WebSocketPublisher, log *logger.Logger) *KafkaUserDataHandler {
	return &KafkaUserDataHandler{
		publisher: publisher,
		logger:    log,
	}
}

// OnOrderUpdate handles order update events and publishes them to Kafka
func (h *KafkaUserDataHandler) OnOrderUpdate(ctx context.Context, event *OrderUpdateEvent) error {
	if event == nil {
		return errors.New("received nil order update event")
	}

	err := h.publisher.PublishOrderUpdate(
		ctx,
		event.UserID.String(),
		event.AccountID.String(),
		event.Exchange,
		event.OrderID,
		event.ClientOrderID,
		event.Symbol,
		event.Side,
		event.PositionSide,
		event.Type,
		event.Status,
		event.ExecutionType,
		event.OriginalQty,
		event.FilledQty,
		event.AvgPrice,
		event.StopPrice,
		event.LastFilledQty,
		event.LastFilledPrice,
		event.Commission,
		event.CommissionAsset,
		event.TradeTime,
		event.EventTime,
	)

	if err != nil {
		h.logger.Error("Failed to publish order update to Kafka",
			"account_id", event.AccountID,
			"order_id", event.OrderID,
			"error", err,
		)
		return errors.Wrap(err, "failed to publish order update")
	}

	return nil
}

// OnPositionUpdate handles position update events and publishes them to Kafka
func (h *KafkaUserDataHandler) OnPositionUpdate(ctx context.Context, event *PositionUpdateEvent) error {
	if event == nil {
		return errors.New("received nil position update event")
	}

	err := h.publisher.PublishPositionUpdate(
		ctx,
		event.UserID.String(),
		event.AccountID.String(),
		event.Exchange,
		event.Symbol,
		event.Side,
		event.Amount,
		event.EntryPrice,
		event.MarkPrice,
		event.UnrealizedPnL,
		event.MaintenanceMargin,
		event.PositionSide,
		event.EventTime,
	)

	if err != nil {
		h.logger.Error("Failed to publish position update to Kafka",
			"account_id", event.AccountID,
			"symbol", event.Symbol,
			"error", err,
		)
		return errors.Wrap(err, "failed to publish position update")
	}

	return nil
}

// OnBalanceUpdate handles balance update events and publishes them to Kafka
func (h *KafkaUserDataHandler) OnBalanceUpdate(ctx context.Context, event *BalanceUpdateEvent) error {
	if event == nil {
		return errors.New("received nil balance update event")
	}

	err := h.publisher.PublishBalanceUpdate(
		ctx,
		event.UserID.String(),
		event.AccountID.String(),
		event.Exchange,
		event.Asset,
		event.WalletBalance,
		event.CrossWalletBalance,
		event.AvailableBalance,
		event.ReasonType,
		event.EventTime,
	)

	if err != nil {
		h.logger.Error("Failed to publish balance update to Kafka",
			"account_id", event.AccountID,
			"asset", event.Asset,
			"error", err,
		)
		return errors.Wrap(err, "failed to publish balance update")
	}

	return nil
}

// OnMarginCall handles margin call events and publishes them to Kafka (CRITICAL!)
func (h *KafkaUserDataHandler) OnMarginCall(ctx context.Context, event *MarginCallEvent) error {
	if event == nil {
		return errors.New("received nil margin call event")
	}

	// Convert positions at risk
	positionsAtRisk := make([]events.PositionAtRiskData, 0, len(event.PositionsAtRisk))
	for _, pos := range event.PositionsAtRisk {
		positionsAtRisk = append(positionsAtRisk, events.PositionAtRiskData{
			Symbol:            pos.Symbol,
			Side:              pos.Side,
			Amount:            pos.Amount,
			MarginType:        pos.MarginType,
			UnrealizedPnL:     pos.UnrealizedPnL,
			MaintenanceMargin: pos.MaintenanceMargin,
			PositionSide:      pos.PositionSide,
		})
	}

	err := h.publisher.PublishMarginCall(
		ctx,
		event.UserID.String(),
		event.AccountID.String(),
		event.Exchange,
		event.CrossWalletBalance,
		positionsAtRisk,
		event.EventTime,
	)

	if err != nil {
		h.logger.Error("Failed to publish MARGIN CALL to Kafka",
			"account_id", event.AccountID,
			"positions_at_risk", len(event.PositionsAtRisk),
			"error", err,
		)
		return errors.Wrap(err, "failed to publish margin call")
	}

	// Log critical alert
	h.logger.Error("⚠️ MARGIN CALL Published",
		"account_id", event.AccountID,
		"user_id", event.UserID,
		"positions_at_risk", len(event.PositionsAtRisk),
	)

	return nil
}

// OnAccountConfigUpdate handles account config update events and publishes them to Kafka
func (h *KafkaUserDataHandler) OnAccountConfigUpdate(ctx context.Context, event *AccountConfigEvent) error {
	if event == nil {
		return errors.New("received nil account config update event")
	}

	err := h.publisher.PublishAccountConfigUpdate(
		ctx,
		event.UserID.String(),
		event.AccountID.String(),
		event.Exchange,
		event.Symbol,
		event.Leverage,
		event.EventTime,
	)

	if err != nil {
		h.logger.Error("Failed to publish account config update to Kafka",
			"account_id", event.AccountID,
			"symbol", event.Symbol,
			"error", err,
		)
		return errors.Wrap(err, "failed to publish account config update")
	}

	return nil
}

// OnError handles WebSocket errors
func (h *KafkaUserDataHandler) OnError(err error) {
	h.logger.Error("User Data WebSocket error",
		"error", err,
	)
	// Don't publish errors to Kafka - just log them
	// Could add metrics/alerting here if needed
}
