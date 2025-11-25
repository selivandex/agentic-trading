package exchanges

import (
	"context"

	"github.com/shopspring/decimal"
)

// ExecuteBracketOrder places entry, stop-loss and take-profit legs atomically (best effort).
func ExecuteBracketOrder(ctx context.Context, ex Exchange, req BracketOrderRequest) (*BracketOrderResponse, error) {
	if req.Entry.Symbol == "" || req.Entry.Quantity.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidRequest
	}

	entryOrder, err := ex.PlaceOrder(ctx, &req.Entry)
	if err != nil {
		return nil, err
	}

	resp := &BracketOrderResponse{
		Entry: entryOrder,
	}

	exitSide := oppositeSide(req.Entry.Side)

	if req.StopLoss != nil {
		slReq := deriveExitRequest(req.Entry, *req.StopLoss, exitSide)
		slReq.StopPrice = req.StopLoss.Price
		stopOrder, err := ex.PlaceOrder(ctx, &slReq)
		if err != nil {
			return nil, err
		}
		resp.StopLoss = stopOrder
	}

	for _, tp := range req.TakeProfit {
		tpReq := deriveExitRequest(req.Entry, tp, exitSide)
		if tpReq.Quantity.IsZero() {
			tpReq.Quantity = req.Entry.Quantity
		}
		order, err := ex.PlaceOrder(ctx, &tpReq)
		if err != nil {
			return nil, err
		}
		resp.TakeProfit = append(resp.TakeProfit, order)
	}

	return resp, nil
}

// ExecuteLadderOrder breaks a large order into multiple limit orders.
func ExecuteLadderOrder(ctx context.Context, ex Exchange, req LadderOrderRequest) (*LadderOrderResponse, error) {
	if req.Template.Symbol == "" || len(req.Steps) == 0 {
		return nil, ErrInvalidRequest
	}

	result := &LadderOrderResponse{}

	for _, step := range req.Steps {
		if step.Amount.LessThanOrEqual(decimal.Zero) || step.Price.LessThanOrEqual(decimal.Zero) {
			return nil, ErrInvalidRequest
		}

		stepReq := req.Template
		stepReq.Price = step.Price
		stepReq.Quantity = step.Amount
		stepReq.Tag = step.Tag

		order, err := ex.PlaceOrder(ctx, &stepReq)
		if err != nil {
			return nil, err
		}

		result.Orders = append(result.Orders, order)
	}

	return result, nil
}

// ClosePosition submits a reduce-only market order to close an active position.
func ClosePosition(ctx context.Context, ex Exchange, symbol string, market MarketType, side PositionSide, amount decimal.Decimal) (*Order, error) {
	if symbol == "" || amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidRequest
	}

	orderSide := oppositeSide(positionSideToOrderSide(side))
	req := OrderRequest{
		Symbol:       symbol,
		Market:       market,
		Side:         orderSide,
		Type:         OrderTypeMarket,
		Quantity:     amount,
		TimeInForce:  TimeInForceIOC,
		ReduceOnly:   true,
		PositionSide: side,
	}

	return ex.PlaceOrder(ctx, &req)
}

// ProtectionRequest defines stop-loss and take-profit placement request.
type ProtectionRequest struct {
	Symbol            string
	Market            MarketType
	PositionSide      PositionSide
	Quantity          decimal.Decimal
	StopLossPrice     decimal.Decimal
	TakeProfitPrice   decimal.Decimal
	StopOrderType     OrderType
	TakeProfitType    OrderType
	TimeInForce       TimeInForce
	Tag               string
}

// ProtectionResponse returns any created protective orders.
type ProtectionResponse struct {
	StopLoss   *Order
	TakeProfit *Order
}

// UpdateProtectionOrders creates or updates stop-loss/take-profit orders for an open position.
func UpdateProtectionOrders(ctx context.Context, ex Exchange, req ProtectionRequest) (*ProtectionResponse, error) {
	if req.Symbol == "" || req.Quantity.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidRequest
	}

	exitSide := oppositeSide(positionSideToOrderSide(req.PositionSide))
	template := OrderRequest{
		Symbol:       req.Symbol,
		Market:       req.Market,
		Side:         exitSide,
		Quantity:     req.Quantity,
		TimeInForce:  req.TimeInForce,
		ReduceOnly:   true,
		PositionSide: req.PositionSide,
		Tag:          req.Tag,
	}

	resp := &ProtectionResponse{}

	if !req.StopLossPrice.IsZero() {
		slReq := template
		slReq.Type = fallbackOrderType(req.StopOrderType, OrderTypeStopMarket)
		slReq.StopPrice = req.StopLossPrice
		if slReq.Type == OrderTypeStopLimit {
			slReq.Price = req.StopLossPrice
		}
		order, err := ex.PlaceOrder(ctx, &slReq)
		if err != nil {
			return nil, err
		}
		resp.StopLoss = order
	}

	if !req.TakeProfitPrice.IsZero() {
		tpReq := template
		tpReq.Type = fallbackOrderType(req.TakeProfitType, OrderTypeLimit)
		tpReq.Price = req.TakeProfitPrice
		if tpReq.Type == OrderTypeStopLimit {
			tpReq.StopPrice = req.TakeProfitPrice
		}
		order, err := ex.PlaceOrder(ctx, &tpReq)
		if err != nil {
			return nil, err
		}
		resp.TakeProfit = order
	}

	return resp, nil
}

func deriveExitRequest(entry OrderRequest, leg BracketLeg, side OrderSide) OrderRequest {
	req := entry
	req.Side = side
	req.Type = fallbackOrderType(leg.Type, OrderTypeLimit)
	if leg.Amount.GreaterThan(decimal.Zero) {
		req.Quantity = leg.Amount
	}
	if leg.Price.GreaterThan(decimal.Zero) {
		req.Price = leg.Price
	}
	req.TimeInForce = fallbackTIF(leg.TimeInForce, entry.TimeInForce)
	req.Tag = leg.Tag
	req.ClientOrderID = ""
	req.ReduceOnly = true
	return req
}

func oppositeSide(side OrderSide) OrderSide {
	if side == OrderSideBuy {
		return OrderSideSell
	}
	return OrderSideBuy
}

func positionSideToOrderSide(side PositionSide) OrderSide {
	if side == PositionSideShort {
		return OrderSideSell
	}
	return OrderSideBuy
}

func fallbackOrderType(value OrderType, def OrderType) OrderType {
	if value == "" {
		return def
	}
	return value
}

func fallbackTIF(value, def TimeInForce) TimeInForce {
	if value == "" {
		return def
	}
	return value
}

