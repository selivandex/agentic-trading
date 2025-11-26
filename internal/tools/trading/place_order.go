package trading

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"prometheus/internal/domain/order"
	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewPlaceOrderTool creates a pending order record.
func NewPlaceOrderTool(deps shared.Deps) tool.Tool {
	return functiontool.New("place_order", "Place a market or limit order", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if deps.OrderRepo == nil {
			deps.Log.Error("Tool: place_order called without order repository")
			return nil, fmt.Errorf("place_order: order repository not configured")
		}

		userID, err := parseUUIDArg(args["user_id"], "user_id")
		if err != nil {
			if meta, ok := shared.MetadataFromContext(ctx); ok {
				userID = meta.UserID
			} else {
				deps.Log.Warn("Tool: place_order missing user_id", "error", err)
				return nil, err
			}
		}
		tradingPairID, err := parseUUIDArg(args["trading_pair_id"], "trading_pair_id")
		if err != nil {
			return nil, err
		}
		accountID, err := parseUUIDArg(args["exchange_account_id"], "exchange_account_id")
		if err != nil {
			return nil, err
		}

		symbol, _ := args["symbol"].(string)
		marketType, _ := args["market_type"].(string)
		sideStr, _ := args["side"].(string)
		typeStr, _ := args["type"].(string)
		amountStr, _ := args["amount"].(string)
		priceStr, _ := args["price"].(string)
		stopPriceStr, _ := args["stop_price"].(string)
		agentID, _ := args["agent_id"].(string)
		reasoning, _ := args["reasoning"].(string)
		reduceOnly, _ := args["reduce_only"].(bool)

		if symbol == "" || sideStr == "" || typeStr == "" || amountStr == "" {
			deps.Log.Warn("Tool: place_order missing required parameters", "symbol", symbol, "side", sideStr, "type", typeStr, "amount", amountStr)
			return nil, fmt.Errorf("place_order: symbol, side, type, and amount are required")
		}

		deps.Log.Info("Tool: place_order called", "user_id", userID, "symbol", symbol, "side", sideStr, "type", typeStr, "amount", amountStr)

		amount, err := decimal.NewFromString(amountStr)
		if err != nil {
			deps.Log.Error("Tool: place_order failed to parse amount", "amount", amountStr, "error", err)
			return nil, fmt.Errorf("place_order: parse amount: %w", err)
		}
		price := decimal.Zero
		if priceStr != "" {
			price, err = decimal.NewFromString(priceStr)
			if err != nil {
				return nil, fmt.Errorf("place_order: parse price: %w", err)
			}
		}
		stopPrice := decimal.Zero
		if stopPriceStr != "" {
			stopPrice, err = decimal.NewFromString(stopPriceStr)
			if err != nil {
				return nil, fmt.Errorf("place_order: parse stop price: %w", err)
			}
		}

		service := order.NewService(deps.OrderRepo)
		created, err := service.Place(ctx, order.PlaceParams{
			UserID:            userID,
			TradingPairID:     tradingPairID,
			ExchangeAccountID: accountID,
			Symbol:            symbol,
			MarketType:        marketType,
			Side:              order.OrderSide(sideStr),
			Type:              order.OrderType(typeStr),
			Price:             price,
			Amount:            amount,
			StopPrice:         stopPrice,
			ReduceOnly:        reduceOnly,
			AgentID:           agentID,
			Reasoning:         reasoning,
		})
		if err != nil {
			deps.Log.Error("Tool: place_order failed", "user_id", userID, "symbol", symbol, "error", err)
			return nil, err
		}

		deps.Log.Info("Tool: place_order success", "order_id", created.ID, "symbol", symbol, "side", created.Side, "status", created.Status)

		return map[string]interface{}{
			"order_id": created.ID.String(),
			"status":   created.Status.String(),
			"symbol":   created.Symbol,
			"side":     created.Side.String(),
			"type":     created.Type.String(),
		}, nil
	})
}
