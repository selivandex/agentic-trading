package trading

import (
	"prometheus/internal/domain/order"
	"prometheus/internal/tools/shared"

	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewCancelOrderTool cancels an order by ID.
func NewCancelOrderTool(deps shared.Deps) tool.Tool {
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        "cancel_order",
			Description: "Cancel a specific order",
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if deps.OrderRepo == nil {
				return nil, errors.Wrapf(errors.ErrInternal, "cancel_order: order repository not configured")
			}
			orderID, err := parseUUIDArg(args["order_id"], "order_id")
			if err != nil {
				return nil, err
			}

			service := order.NewService(deps.OrderRepo)
			if err := service.Cancel(ctx, orderID); err != nil {
				return nil, err
			}

			return map[string]interface{}{"order_id": orderID.String(), "status": order.OrderStatusCanceled.String()}, nil
		})
	return t
}
