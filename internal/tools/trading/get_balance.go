package trading

import (
	"context"
	"fmt"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewGetBalanceTool lists active exchange accounts for the user.
func NewGetBalanceTool(deps shared.Deps) tool.Tool {
	return functiontool.New("get_balance", "Retrieve account balances", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if deps.ExchangeAccountRepo == nil {
			deps.Log.Error("Tool: get_balance called without exchange account repository")
			return nil, fmt.Errorf("get_balance: exchange account repository not configured")
		}

		userID, err := parseUUIDArg(args["user_id"], "user_id")
		if err != nil {
			if meta, ok := shared.MetadataFromContext(ctx); ok {
				userID = meta.UserID
			} else {
				deps.Log.Warn("Tool: get_balance missing user_id", "error", err)
				return nil, err
			}
		}

		deps.Log.Debug("Tool: get_balance called", "user_id", userID)

		accounts, err := deps.ExchangeAccountRepo.GetActiveByUser(ctx, userID)
		if err != nil {
			deps.Log.Error("Tool: get_balance failed to fetch accounts", "user_id", userID, "error", err)
			return nil, fmt.Errorf("get_balance: fetch accounts: %w", err)
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

		deps.Log.Info("Tool: get_balance success", "user_id", userID, "accounts_count", len(accounts))
		return map[string]interface{}{"accounts": data}, nil
	})
}
