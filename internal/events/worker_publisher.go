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
		ResumeAt:    nil,
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

// PublishDrawdownAlert publishes drawdown warning event
func (wp *WorkerPublisher) PublishDrawdownAlert(
	ctx context.Context,
	userID, reason string,
	currentDrawdown, maxDrawdown, percentage, dailyPnL float64,
) error {
	event := &eventspb.DrawdownAlert{
		Base:            NewBaseEvent(TopicDrawdownAlert, "risk_monitor", userID),
		Reason:          reason,
		CurrentDrawdown: currentDrawdown,
		MaxDrawdown:     maxDrawdown,
		Percentage:      percentage,
		DailyPnl:        dailyPnL,
	}

	return wp.publishProto(ctx, TopicDrawdownAlert, userID, event)
}

// PublishConsecutiveLossesAlert publishes consecutive losses warning
func (wp *WorkerPublisher) PublishConsecutiveLossesAlert(
	ctx context.Context,
	userID string,
	consecutiveLosses, maxAllowed int,
) error {
	event := &eventspb.ConsecutiveLossesAlert{
		Base:              NewBaseEvent(TopicRiskLimitExceeded, "risk_monitor", userID),
		ConsecutiveLosses: int32(consecutiveLosses),
		MaxAllowed:        int32(maxAllowed),
	}

	return wp.publishProto(ctx, TopicRiskLimitExceeded, userID, event)
}

// PublishPnLUpdated publishes PnL update event
func (wp *WorkerPublisher) PublishPnLUpdated(
	ctx context.Context,
	userID string,
	dailyPnL, dailyPnLPercent, totalPnL float64,
	tradesCount, winningTrades, losingTrades int,
	winRate float64,
) error {
	event := &eventspb.PnLUpdatedEvent{
		Base:            NewBaseEvent("user.pnl_updated", "pnl_calculator", userID),
		DailyPnl:        dailyPnL,
		DailyPnlPercent: dailyPnLPercent,
		TotalPnl:        totalPnL,
		TradesCount:     int32(tradesCount),
		WinningTrades:   int32(winningTrades),
		LosingTrades:    int32(losingTrades),
		WinRate:         winRate,
	}

	return wp.publishProto(ctx, "user.pnl_updated", userID, event)
}

// PublishJournalEntryCreated publishes journal entry created event
func (wp *WorkerPublisher) PublishJournalEntryCreated(
	ctx context.Context,
	userID, entryID, symbol, side, lessonLearned string,
	pnl, pnlPercent float64,
) error {
	event := &eventspb.JournalEntryCreatedEvent{
		Base:          NewBaseEvent("journal.entry_created", "journal_compiler", userID),
		EntryId:       entryID,
		Symbol:        symbol,
		Side:          side,
		Pnl:           pnl,
		PnlPercent:    pnlPercent,
		LessonLearned: lessonLearned,
	}

	return wp.publishProto(ctx, "journal.entry_created", userID, event)
}

// PublishDailyReport publishes daily performance report
func (wp *WorkerPublisher) PublishDailyReport(
	ctx context.Context,
	userID string,
	dailyPnL, dailyPnLPercent float64,
	totalTrades, winningTrades int,
	winRate, sharpeRatio, maxDrawdown float64,
) error {
	event := &eventspb.DailyReportEvent{
		Base:            NewBaseEvent("report.daily", "daily_report", userID),
		DailyPnl:        dailyPnL,
		DailyPnlPercent: dailyPnLPercent,
		TotalTrades:     int32(totalTrades),
		WinningTrades:   int32(winningTrades),
		WinRate:         winRate,
		SharpeRatio:     sharpeRatio,
		MaxDrawdown:     maxDrawdown,
	}

	return wp.publishProto(ctx, "report.daily", userID, event)
}

// PublishStrategyDisabled publishes strategy disabled event
func (wp *WorkerPublisher) PublishStrategyDisabled(
	ctx context.Context,
	userID, strategyID, strategyName, reason string,
	winRate, profitFactor float64,
	totalTrades int,
) error {
	event := &eventspb.StrategyDisabledEvent{
		Base:         NewBaseEvent("strategy.disabled", "strategy_evaluator", userID),
		StrategyId:   strategyID,
		StrategyName: strategyName,
		Reason:       reason,
		WinRate:      winRate,
		ProfitFactor: profitFactor,
		TotalTrades:  int32(totalTrades),
	}

	return wp.publishProto(ctx, "strategy.disabled", userID, event)
}

// PublishOrderCancelled publishes order cancelled event
func (wp *WorkerPublisher) PublishOrderCancelled(
	ctx context.Context,
	userID, orderID, symbol, exchange, reason string,
) error {
	event := &eventspb.OrderCancelledEvent{
		Base:     NewBaseEvent(TopicOrderCancelled, "order_sync", userID),
		OrderId:  orderID,
		Symbol:   symbol,
		Exchange: exchange,
		Reason:   reason,
	}

	return wp.publishProto(ctx, TopicOrderCancelled, orderID, event)
}

// PublishPositionPnLUpdated publishes position PnL update event
func (wp *WorkerPublisher) PublishPositionPnLUpdated(
	ctx context.Context,
	userID, positionID, symbol string,
	unrealizedPnL, unrealizedPnLPercent, currentPrice, entryPrice float64,
) error {
	event := &eventspb.PositionPnLUpdatedEvent{
		Base:                 NewBaseEvent("position.pnl_updated", "position_monitor", userID),
		PositionId:           positionID,
		Symbol:               symbol,
		UnrealizedPnl:        unrealizedPnL,
		UnrealizedPnlPercent: unrealizedPnLPercent,
		CurrentPrice:         currentPrice,
		EntryPrice:           entryPrice,
	}

	return wp.publishProto(ctx, "position.pnl_updated", positionID, event)
}

// PublishFVGDetected publishes Fair Value Gap detected event
func (wp *WorkerPublisher) PublishFVGDetected(
	ctx context.Context,
	symbol, fvgType string,
	topPrice, bottomPrice, gapPercent float64,
	filled bool,
) error {
	event := &eventspb.FVGDetectedEvent{
		Base:        NewBaseEvent("smc.fvg_detected", "smc_scanner", ""),
		Symbol:      symbol,
		Type:        fvgType,
		TopPrice:    topPrice,
		BottomPrice: bottomPrice,
		GapPercent:  gapPercent,
		Filled:      filled,
	}

	return wp.publishProto(ctx, "smc.fvg_detected", symbol, event)
}

// PublishOrderBlockDetected publishes Order Block detected event
func (wp *WorkerPublisher) PublishOrderBlockDetected(
	ctx context.Context,
	symbol, obType string,
	topPrice, bottomPrice, strength float64,
) error {
	event := &eventspb.OrderBlockDetectedEvent{
		Base:        NewBaseEvent("smc.order_block_detected", "smc_scanner", ""),
		Symbol:      symbol,
		Type:        obType,
		TopPrice:    topPrice,
		BottomPrice: bottomPrice,
		Strength:    strength,
	}

	return wp.publishProto(ctx, "smc.order_block_detected", symbol, event)
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
