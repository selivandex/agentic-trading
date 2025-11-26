package trading

import (
	"time"

	"github.com/shopspring/decimal"

	"prometheus/internal/domain/order"
	"prometheus/internal/tools/shared"

	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
)

// NewPlaceOrderTool creates a pending order record.
func NewPlaceOrderTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"place_order",
		"Place a market or limit order",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if deps.OrderRepo == nil {
				return nil, errors.Wrapf(errors.ErrInternal, "place_order: order repository not configured")
			}

			userID, err := parseUUIDArg(args["user_id"], "user_id")
			if err != nil {
				if meta, ok := shared.MetadataFromContext(ctx); ok {
					userID = meta.UserID
				} else {
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
				return nil, errors.ErrInvalidInput
			}

			amount, err := decimal.NewFromString(amountStr)
			if err != nil {
				return nil, errors.Wrap(err, "place_order: parse amount")
			}
			price := decimal.Zero
			if priceStr != "" {
				price, err = decimal.NewFromString(priceStr)
				if err != nil {
					return nil, errors.Wrap(err, "place_order: parse price")
				}
			}
			stopPrice := decimal.Zero
			if stopPriceStr != "" {
				stopPrice, err = decimal.NewFromString(stopPriceStr)
				if err != nil {
					return nil, errors.Wrap(err, "place_order: parse stop price")
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
				return nil, err
			}

			return map[string]interface{}{
				"order_id": created.ID.String(),
				"status":   created.Status.String(),
				"symbol":   created.Symbol,
				"side":     created.Side.String(),
				"type":     created.Type.String(),
			}, nil
		},
		deps,
	).
		WithTimeout(10*time.Second).
		WithRetry(3, 500*time.Millisecond).
		WithStats().
		Build()
}
