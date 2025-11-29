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

// PublishRegimeChange publishes a regime change event (legacy, without ML interpretation)
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

// PublishRegimeChangeML publishes enhanced regime change event with ML interpretation
func (wp *WorkerPublisher) PublishRegimeChangeML(
	ctx context.Context,
	symbol, oldRegime, newRegime, trend string,
	confidence, volatility float64,
	explanation, strategicGuidance string,
	positionSizeMultiplier, cashReserveTarget float64,
	favoredStrategies, avoidedStrategies []string,
) error {
	event := &eventspb.RegimeChangedEvent{
		Base:                   NewBaseEvent(TopicRegimeChanged, "regime_detector_ml", ""),
		Symbol:                 symbol,
		OldRegime:              oldRegime,
		NewRegime:              newRegime,
		Confidence:             confidence,
		Volatility:             volatility,
		Trend:                  trend,
		Explanation:            explanation,
		StrategicGuidance:      strategicGuidance,
		PositionSizeMultiplier: positionSizeMultiplier,
		CashReserveTarget:      cashReserveTarget,
		FavoredStrategies:      favoredStrategies,
		AvoidedStrategies:      avoidedStrategies,
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
		Base:            NewBaseEvent(TopicPnLUpdated, "pnl_calculator", userID),
		DailyPnl:        dailyPnL,
		DailyPnlPercent: dailyPnLPercent,
		TotalPnl:        totalPnL,
		TradesCount:     int32(tradesCount),
		WinningTrades:   int32(winningTrades),
		LosingTrades:    int32(losingTrades),
		WinRate:         winRate,
	}

	return wp.publishProto(ctx, TopicPnLUpdated, userID, event)
}

// PublishJournalEntryCreated publishes journal entry created event
func (wp *WorkerPublisher) PublishJournalEntryCreated(
	ctx context.Context,
	userID, entryID, symbol, side, lessonLearned string,
	pnl, pnlPercent float64,
) error {
	event := &eventspb.JournalEntryCreatedEvent{
		Base:          NewBaseEvent(TopicJournalEntryCreated, "journal_compiler", userID),
		EntryId:       entryID,
		Symbol:        symbol,
		Side:          side,
		Pnl:           pnl,
		PnlPercent:    pnlPercent,
		LessonLearned: lessonLearned,
	}

	return wp.publishProto(ctx, TopicJournalEntryCreated, userID, event)
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
		Base:            NewBaseEvent(TopicDailyReport, "daily_report", userID),
		DailyPnl:        dailyPnL,
		DailyPnlPercent: dailyPnLPercent,
		TotalTrades:     int32(totalTrades),
		WinningTrades:   int32(winningTrades),
		WinRate:         winRate,
		SharpeRatio:     sharpeRatio,
		MaxDrawdown:     maxDrawdown,
	}

	return wp.publishProto(ctx, TopicDailyReport, userID, event)
}

// PublishStrategyDisabled publishes strategy disabled event
func (wp *WorkerPublisher) PublishStrategyDisabled(
	ctx context.Context,
	userID, strategyID, strategyName, reason string,
	winRate, profitFactor float64,
	totalTrades int,
) error {
	event := &eventspb.StrategyDisabledEvent{
		Base:         NewBaseEvent(TopicStrategyDisabled, "strategy_evaluator", userID),
		StrategyId:   strategyID,
		StrategyName: strategyName,
		Reason:       reason,
		WinRate:      winRate,
		ProfitFactor: profitFactor,
		TotalTrades:  int32(totalTrades),
	}

	return wp.publishProto(ctx, TopicStrategyDisabled, userID, event)
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
		Base:                 NewBaseEvent(TopicPositionPnLUpdated, "position_monitor", userID),
		PositionId:           positionID,
		Symbol:               symbol,
		UnrealizedPnl:        unrealizedPnL,
		UnrealizedPnlPercent: unrealizedPnLPercent,
		CurrentPrice:         currentPrice,
		EntryPrice:           entryPrice,
	}

	return wp.publishProto(ctx, TopicPositionPnLUpdated, positionID, event)
}

// PublishFVGDetected publishes Fair Value Gap detected event
func (wp *WorkerPublisher) PublishFVGDetected(
	ctx context.Context,
	symbol, fvgType string,
	topPrice, bottomPrice, gapPercent float64,
	filled bool,
) error {
	event := &eventspb.FVGDetectedEvent{
		Base:        NewBaseEvent(TopicFVGDetected, "smc_scanner", ""),
		Symbol:      symbol,
		Type:        fvgType,
		TopPrice:    topPrice,
		BottomPrice: bottomPrice,
		GapPercent:  gapPercent,
		Filled:      filled,
	}

	return wp.publishProto(ctx, TopicFVGDetected, symbol, event)
}

// PublishOrderBlockDetected publishes Order Block detected event
func (wp *WorkerPublisher) PublishOrderBlockDetected(
	ctx context.Context,
	symbol, obType string,
	topPrice, bottomPrice, strength float64,
) error {
	event := &eventspb.OrderBlockDetectedEvent{
		Base:        NewBaseEvent(TopicOrderBlockDetected, "smc_scanner", ""),
		Symbol:      symbol,
		Type:        obType,
		TopPrice:    topPrice,
		BottomPrice: bottomPrice,
		Strength:    strength,
	}

	return wp.publishProto(ctx, TopicOrderBlockDetected, symbol, event)
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

// PublishMarketAnalysisRequest publishes market analysis request event
func (wp *WorkerPublisher) PublishMarketAnalysisRequest(
	ctx context.Context,
	userID, symbol, marketType, strategy string,
) error {
	event := &eventspb.MarketAnalysisRequestEvent{
		Base:       NewBaseEvent(TopicAnalysisRequested, "market_scanner", userID),
		Symbol:     symbol,
		MarketType: marketType,
		Strategy:   strategy,
	}

	return wp.publishProto(ctx, TopicAnalysisRequested, userID, event)
}

// PublishMarketScanComplete publishes market scan complete event
func (wp *WorkerPublisher) PublishMarketScanComplete(
	ctx context.Context,
	totalUsers, errorsCount int,
	durationMs int64,
) error {
	event := &eventspb.MarketScanCompleteEvent{
		Base:        NewBaseEvent(TopicScanComplete, "market_scanner", ""),
		TotalUsers:  int32(totalUsers),
		ErrorsCount: int32(errorsCount),
		DurationMs:  durationMs,
	}

	return wp.publishProto(ctx, TopicScanComplete, "global", event)
}

// PublishWhaleAlert publishes whale trade alert event
func (wp *WorkerPublisher) PublishWhaleAlert(
	ctx context.Context,
	exchange, symbol string, tradeID int64, side, sentiment string,
	price, quantity, valueUSD float64,
) error {
	event := &eventspb.WhaleAlertEvent{
		Base:      NewBaseEvent(TopicWhaleAlert, "whale_alert_collector", ""),
		Exchange:  exchange,
		Symbol:    symbol,
		TradeId:   tradeID,
		Price:     price,
		Quantity:  quantity,
		ValueUsd:  valueUSD,
		Side:      side,
		Sentiment: sentiment,
	}

	return wp.publishProto(ctx, TopicWhaleAlert, symbol, event)
}

// PublishLiquidationAlert publishes liquidation alert event
func (wp *WorkerPublisher) PublishLiquidationAlert(
	ctx context.Context,
	exchange, symbol, side string,
	price, quantity, valueUSD float64,
) error {
	event := &eventspb.LiquidationAlertEvent{
		Base:     NewBaseEvent(TopicLiquidationAlert, "liquidation_collector", ""),
		Exchange: exchange,
		Symbol:   symbol,
		Side:     side,
		Price:    price,
		Quantity: quantity,
		ValueUsd: valueUSD,
	}

	return wp.publishProto(ctx, TopicLiquidationAlert, symbol, event)
}

// PublishStrategyWarning publishes strategy performance warning event
func (wp *WorkerPublisher) PublishStrategyWarning(
	ctx context.Context,
	userID, strategyID, strategyName, reason string,
	winRate, profitFactor float64,
	totalTrades int,
) error {
	event := &eventspb.StrategyWarningEvent{
		Base:         NewBaseEvent(TopicStrategyWarning, "strategy_evaluator", userID),
		StrategyId:   strategyID,
		StrategyName: strategyName,
		Reason:       reason,
		WinRate:      winRate,
		ProfitFactor: profitFactor,
		TotalTrades:  int32(totalTrades),
	}

	return wp.publishProto(ctx, TopicStrategyWarning, userID, event)
}

// PublishAIUsage publishes AI model usage event
func (wp *WorkerPublisher) PublishAIUsage(
	ctx context.Context,
	userID, sessionID, agentName, agentType string,
	provider, modelID, modelFamily string,
	promptTokens, completionTokens, totalTokens uint32,
	inputCostUSD, outputCostUSD, totalCostUSD float64,
	toolCallsCount uint32,
	isCached, cacheHit bool,
	latencyMs uint32,
	reasoningStep uint32,
	workflowName string,
) error {
	event := &eventspb.AIUsageEvent{
		Base:             NewBaseEvent(TopicAIUsage, "agent_callback", userID),
		SessionId:        sessionID,
		AgentName:        agentName,
		AgentType:        agentType,
		Provider:         provider,
		ModelId:          modelID,
		ModelFamily:      modelFamily,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		InputCostUsd:     inputCostUSD,
		OutputCostUsd:    outputCostUSD,
		TotalCostUsd:     totalCostUSD,
		ToolCallsCount:   toolCallsCount,
		IsCached:         isCached,
		CacheHit:         cacheHit,
		LatencyMs:        latencyMs,
		ReasoningStep:    reasoningStep,
		WorkflowName:     workflowName,
	}

	return wp.publishProto(ctx, TopicAIUsage, sessionID, event)
}

// PublishOpportunityFound publishes trading opportunity event
func (wp *WorkerPublisher) PublishOpportunityFound(
	ctx context.Context,
	symbol, exchange, direction, timeframe, strategy, reasoning string,
	confidence, entry, stopLoss, takeProfit float64,
	indicators map[string]string,
) error {
	event := &eventspb.OpportunityFoundEvent{
		Base:       NewBaseEvent(TopicOpportunityFound, "market_research_workflow", ""),
		Symbol:     symbol,
		Exchange:   exchange,
		Direction:  direction,
		Confidence: confidence,
		Entry:      entry,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
		Timeframe:  timeframe,
		Strategy:   strategy,
		Reasoning:  reasoning,
		Indicators: indicators,
	}

	return wp.publishProto(ctx, TopicOpportunityFound, symbol, event)
}
