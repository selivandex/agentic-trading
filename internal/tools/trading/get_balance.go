package trading

import (
	"prometheus/internal/tools/shared"

	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewGetBalanceTool lists active exchange accounts for the user.
func NewGetBalanceTool(deps shared.Deps) tool.Tool {
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        "get_balance",
			Description: "Retrieve account balances",
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if deps.ExchangeAccountRepo == nil {
				return nil, errors.Wrapf(errors.ErrInternal, "get_balance: exchange account repository not configured")
			}

			userID, err := parseUUIDArg(args["user_id"], "user_id")
			if err != nil {
				if meta, ok := shared.MetadataFromContext(ctx); ok {
					userID = meta.UserID
				} else {
					return nil, err
				}
			}

			accounts, err := deps.ExchangeAccountRepo.GetActiveByUser(ctx, userID)
			if err != nil {
				return nil, errors.Wrap(err, "get_balance: fetch accounts")
			}

			data := make([]map[string]interface{}, 0, len(accounts))
			for _, acc := range accounts {
				data = append(data, map[string]interface{}{
					"id":          acc.ID.String(),
					"exchange":    acc.Exchange.String(),
					"label":       acc.Label,
					"is_testnet":  acc.IsTestnet,
					"permissions": acc.Permissions,
				})
			}

			return map[string]interface{}{"accounts": data}, nil
		})
	return t
}
