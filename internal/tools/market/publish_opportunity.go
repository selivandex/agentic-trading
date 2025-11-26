package market

import (
	"context"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/protobuf/proto"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/events"
	eventspb "prometheus/internal/events/proto"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
)

// NewPublishOpportunityTool creates the publish_opportunity tool
// This tool is used by OpportunitySynthesizer to publish validated trading opportunities to Kafka
func NewPublishOpportunityTool(kafkaProducer *kafka.Producer) tool.Tool {
	return shared.NewToolBuilder(
		"publish_opportunity",
		"Publishes a validated trading opportunity signal to event stream for user consumption. "+
			"Only call this when you have a high-confidence (>65%) opportunity with clear entry/stop/target levels. "+
			"This triggers personal trading workflows for all interested users.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return executePublishOpportunity(ctx, args, kafkaProducer)
		},
		shared.Deps{}, // No repo dependencies needed
	).
		WithTimeout(10 * time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}

func executePublishOpportunity(
	ctx context.Context,
	args map[string]interface{},
	kafkaProducer *kafka.Producer,
) (map[string]interface{}, error) {
	// Extract and validate arguments
	symbol, _ := args["symbol"].(string)
	exchange, _ := args["exchange"].(string)
	direction, _ := args["direction"].(string)
	confidence, _ := args["confidence"].(float64)
	entry, _ := args["entry"].(float64)
	stopLoss, _ := args["stop_loss"].(float64)
	takeProfit, _ := args["take_profit"].(float64)
	timeframe, _ := args["timeframe"].(string)
	strategy, _ := args["strategy"].(string)
	reasoning, _ := args["reasoning"].(string)

	// Validate required fields
	if symbol == "" {
		return nil, errors.New("symbol is required")
	}
	if exchange == "" {
		return nil, errors.New("exchange is required")
	}
	if direction != "long" && direction != "short" {
		return nil, errors.New("direction must be 'long' or 'short'")
	}

	// Validate confidence range
	if confidence < 0 || confidence > 1 {
		return nil, errors.New("confidence must be between 0 and 1")
	}

	// Validate confidence threshold
	if confidence < 0.65 {
		return map[string]interface{}{
			"published": false,
			"message":   "Confidence too low (<65%). Skipping publication.",
		}, nil
	}

	// Validate price levels
	if entry <= 0 {
		return nil, errors.New("entry price must be positive")
	}
	if stopLoss <= 0 {
		return nil, errors.New("stop_loss must be positive")
	}
	if takeProfit <= 0 {
		return nil, errors.New("take_profit must be positive")
	}

	// Validate R:R ratio
	var riskReward float64
	if direction == "long" {
		risk := entry - stopLoss
		reward := takeProfit - entry
		if risk <= 0 {
			return nil, errors.New("stop_loss must be below entry for long positions")
		}
		if reward <= 0 {
			return nil, errors.New("take_profit must be above entry for long positions")
		}
		riskReward = reward / risk
	} else {
		risk := stopLoss - entry
		reward := entry - takeProfit
		if risk <= 0 {
			return nil, errors.New("stop_loss must be above entry for short positions")
		}
		if reward <= 0 {
			return nil, errors.New("take_profit must be below entry for short positions")
		}
		riskReward = reward / risk
	}

	// Reject if R:R < 2:1
	if riskReward < 2.0 {
		return map[string]interface{}{
			"published": false,
			"message":   "Risk/Reward ratio too low (<2:1). Skipping publication.",
		}, nil
	}

	// Create protobuf event
	event := &eventspb.OpportunityFoundEvent{
		Base:       events.NewBaseEvent(events.TopicOpportunityFound, "market_research_workflow", ""),
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
		Indicators: make(map[string]string),
	}

	// Serialize to protobuf
	data, err := proto.Marshal(event)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal opportunity event")
	}

	// Publish to Kafka
	if err := kafkaProducer.PublishBinary(ctx, events.TopicOpportunityFound, []byte(symbol), data); err != nil {
		return nil, errors.Wrap(err, "failed to publish opportunity event to Kafka")
	}

	return map[string]interface{}{
		"published": true,
		"event_id":  event.Base.Id,
		"message":   "Opportunity published successfully to Kafka",
	}, nil
}
