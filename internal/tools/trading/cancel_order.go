package trading
import (
	"prometheus/internal/domain/order"
	"prometheus/internal/tools/shared"
	"time"
	"prometheus/pkg/errors"
	"google.golang.org/adk/tool"
)
// NewCancelOrderTool cancels an order by ID
func NewCancelOrderTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"cancel_order",
		"Cancel a specific order",
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
		},
		deps,
	).
		WithTimeout(10*time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}
