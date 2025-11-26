package events

import (
	"context"

	"prometheus/internal/adapters/kafka"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// WorkerPublisher provides convenience methods for workers to publish events
// Wraps the standard Publisher with worker-specific helpers
type WorkerPublisher struct {
	kafka *kafka.Producer
}

// NewWorkerPublisher creates a new worker event publisher
func NewWorkerPublisher(kafka *kafka.Producer) *WorkerPublisher {
	return &WorkerPublisher{kafka: kafka}
}

// PublishRegimeChange publishes a regime change event
func (wp *WorkerPublisher) PublishRegimeChange(
	ctx context.Context,
	symbol, oldRegime, newRegime, trend string,
	confidence, volatility float64,
) error {
	event := &eventspb.RegimeChangedEvent{
		Base:       NewBaseEvent(TopicRegimeChanged, "regime_detector", ""),
		Symbol:     symbol,
		OldRegime:  oldRegime,
		NewRegime:  newRegime,
		Confidence: confidence,
		Volatility: volatility,
		Trend:      trend,
	}

	return wp.publishProto(ctx, TopicRegimeChanged, symbol, event)
}

// PublishCircuitBreakerTripped publishes circuit breaker event
func (wp *WorkerPublisher) PublishCircuitBreakerTripped(
	ctx context.Context,
	userID, reason string,
	currentLoss, threshold, drawdown float64,
	autoResume bool,
) error {
	event := &eventspb.CircuitBreakerTrippedEvent{
		Base:        NewBaseEvent(TopicCircuitBreakerTripped, "risk_monitor", userID),
		Reason:      reason,
		CurrentLoss: currentLoss,
		Threshold:   threshold,
		Drawdown:    drawdown,
		AutoResume:  autoResume,
		ResumeAt:    nil, // TODO: Calculate resume time if auto_resume
	}

	return wp.publishProto(ctx, TopicCircuitBreakerTripped, userID, event)
}

// PublishOrderFilled publishes order filled event
func (wp *WorkerPublisher) PublishOrderFilled(
	ctx context.Context,
	userID, orderID, symbol, exchange, side string,
	filledPrice, filledAmount, fee float64,
	feeCurrency string,
) error {
	event := &eventspb.OrderFilledEvent{
		Base:         NewBaseEvent(TopicOrderFilled, "order_sync", userID),
		OrderId:      orderID,
		Symbol:       symbol,
		Exchange:     exchange,
		Side:         side,
		FilledPrice:  filledPrice,
		FilledAmount: filledAmount,
		Fee:          fee,
		FeeCurrency:  feeCurrency,
	}

	return wp.publishProto(ctx, TopicOrderFilled, orderID, event)
}

// PublishPositionClosed publishes position closed event
func (wp *WorkerPublisher) PublishPositionClosed(
	ctx context.Context,
	userID, positionID, symbol, exchange, side, closeReason string,
	entryPrice, exitPrice, amount, pnl, pnlPercent float64,
	durationSeconds int64,
) error {
	event := &eventspb.PositionClosedEvent{
		Base:            NewBaseEvent(TopicPositionClosed, "position_monitor", userID),
		PositionId:      positionID,
		Symbol:          symbol,
		Exchange:        exchange,
		Side:            side,
		EntryPrice:      entryPrice,
		ExitPrice:       exitPrice,
		Amount:          amount,
		Pnl:             pnl,
		PnlPercent:      pnlPercent,
		DurationSeconds: durationSeconds,
		CloseReason:     closeReason,
	}

	return wp.publishProto(ctx, TopicPositionClosed, positionID, event)
}

// PublishJSON is a backward-compatible method for non-protobuf events
// TODO: Migrate all events to protobuf and remove this
func (wp *WorkerPublisher) PublishJSON(ctx context.Context, topic, key string, event interface{}) error {
	return wp.kafka.Publish(ctx, topic, key, event)
}

// publishProto serializes and publishes a protobuf message
func (wp *WorkerPublisher) publishProto(ctx context.Context, topic, key string, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshal protobuf")
	}

	if err := wp.kafka.PublishBinary(ctx, topic, []byte(key), data); err != nil {
		return errors.Wrap(err, "publish to kafka")
	}

	return nil
}

// Helper to create common events

// CreateStopLossHitEvent creates a position closed event for stop loss
func CreateStopLossHitEvent(userID, positionID, symbol, exchange, side string, entryPrice, exitPrice, amount, pnl, pnlPercent float64, duration int64) *eventspb.PositionClosedEvent {
	return &eventspb.PositionClosedEvent{
		Base:            NewBaseEvent(TopicStopLossTriggered, "position_monitor", userID),
		PositionId:      positionID,
		Symbol:          symbol,
		Exchange:        exchange,
		Side:            side,
		EntryPrice:      entryPrice,
		ExitPrice:       exitPrice,
		Amount:          amount,
		Pnl:             pnl,
		PnlPercent:      pnlPercent,
		DurationSeconds: duration,
		CloseReason:     "stop_loss",
	}
}

// CreateTakeProfitHitEvent creates a position closed event for take profit
func CreateTakeProfitHitEvent(userID, positionID, symbol, exchange, side string, entryPrice, exitPrice, amount, pnl, pnlPercent float64, duration int64) *eventspb.PositionClosedEvent {
	return &eventspb.PositionClosedEvent{
		Base:            NewBaseEvent(TopicTakeProfitHit, "position_monitor", userID),
		PositionId:      positionID,
		Symbol:          symbol,
		Exchange:        exchange,
		Side:            side,
		EntryPrice:      entryPrice,
		ExitPrice:       exitPrice,
		Amount:          amount,
		Pnl:             pnl,
		PnlPercent:      pnlPercent,
		DurationSeconds: duration,
		CloseReason:     "take_profit",
	}
}

// CreateWorkerFailedEvent creates a worker failed event
func CreateWorkerFailedEvent(workerName, errorMsg string, failCount int, lastSuccess *timestamppb.Timestamp) *eventspb.WorkerFailedEvent {
	return &eventspb.WorkerFailedEvent{
		Base:        NewBaseEvent(TopicWorkerFailed, workerName, ""),
		WorkerName:  workerName,
		Error:       errorMsg,
		FailCount:   int32(failCount),
		LastSuccess: lastSuccess,
	}
}
