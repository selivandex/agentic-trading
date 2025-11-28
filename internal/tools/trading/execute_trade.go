package trading

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/adk/tool"

	"prometheus/internal/domain/order"
	"prometheus/internal/services/execution"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
)

// NewExecuteTradeTool creates a tool that executes trades via ExecutionService.
// This replaces the Executor LLM agent with algorithmic execution logic.
func NewExecuteTradeTool(executionService *execution.ExecutionService) tool.Tool {
	return shared.NewToolBuilder(
		"execute_trade",
		"Execute a validated trading plan. Handles venue selection, order type, and placement. Call after strategy planning and risk validation.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return executeTradeImpl(ctx, args, executionService)
		},
		shared.Deps{}, // ExecutionService injected directly
	).
		WithTimeout(30*time.Second).
		WithRetry(2, 1*time.Second).
		Build()
}

func executeTradeImpl(
	ctx context.Context,
	args map[string]interface{},
	executionService *execution.ExecutionService,
) (map[string]interface{}, error) {
	// Extract user ID from context
	userID := uuid.Nil
	if meta, ok := shared.MetadataFromContext(ctx); ok {
		userID = meta.UserID
	}
	if userID == uuid.Nil {
		return nil, errors.New("user_id required")
	}

	// Parse parameters
	symbol, _ := args["symbol"].(string)
	sideStr, _ := args["side"].(string)
	amountStr, _ := args["amount"].(string)
	orderTypeStr, _ := args["order_type"].(string)
	exchange, _ := args["exchange"].(string)
	priceStr, _ := args["price"].(string)
	stopPriceStr, _ := args["stop_price"].(string)

	// Validate required fields
	if symbol == "" {
		return nil, errors.New("symbol is required")
	}
	if sideStr == "" {
		return nil, errors.New("side is required (buy/sell)")
	}
	if amountStr == "" {
		return nil, errors.New("amount is required")
	}

	// Parse side
	var side order.OrderSide
	switch sideStr {
	case "buy":
		side = order.OrderSideBuy
	case "sell":
		side = order.OrderSideSell
	default:
		return nil, errors.New("side must be 'buy' or 'sell'")
	}

	// Parse amount
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return nil, errors.Wrap(err, "invalid amount")
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("amount must be positive")
	}

	// Parse order type
	var orderType order.OrderType
	switch orderTypeStr {
	case "market", "":
		orderType = order.OrderTypeMarket
	case "limit":
		orderType = order.OrderTypeLimit
	case "stop_limit":
		orderType = order.OrderTypeStopLimit
	case "stop_market":
		orderType = order.OrderTypeStopMarket
	default:
		return nil, errors.New("invalid order_type")
	}

	// Parse optional price fields
	var price, stopPrice decimal.Decimal
	if priceStr != "" {
		price, err = decimal.NewFromString(priceStr)
		if err != nil {
			return nil, errors.Wrap(err, "invalid price")
		}
	}
	if stopPriceStr != "" {
		stopPrice, err = decimal.NewFromString(stopPriceStr)
		if err != nil {
			return nil, errors.Wrap(err, "invalid stop_price")
		}
	}

	// Default exchange if not specified
	if exchange == "" {
		exchange = "binance" // TODO: Get from user preferences
	}

	// Build trading plan
	plan := execution.TradingPlan{
		UserID:     userID,
		Symbol:     symbol,
		Side:       side,
		Amount:     amount,
		OrderType:  orderType,
		Price:      price,
		StopPrice:  stopPrice,
		Exchange:   exchange,
		Urgency:    execution.UrgencyMedium,
		TimeWindow: 60 * time.Second,
	}

	// Execute via service
	result, err := executionService.Execute(ctx, plan)
	if err != nil {
		return nil, errors.Wrap(err, "execution failed")
	}

	// Return result
	return map[string]interface{}{
		"order_id":    result.OrderID,
		"status":      result.Status,
		"filled_qty":  result.FilledQty.String(),
		"avg_price":   result.AvgPrice.String(),
		"fee":         result.Fee.String(),
		"slippage":    result.Slippage.String(),
		"duration_ms": result.Duration.Milliseconds(),
		"error":       result.ErrorMessage,
	}, nil
}
