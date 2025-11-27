package schemas

import "google.golang.org/genai"

// Helper functions for creating float64 pointers
func float64Ptr(v float64) *float64 {
	return &v
}

// ExecutorInputSchema defines the input schema for the Executor agent
var ExecutorInputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"symbol": {
			Type:        "STRING",
			Description: "Trading pair symbol (e.g. BTC/USDT, ETH/USDT)",
			Pattern:     "^[A-Z]+/[A-Z]+$",
		},
		"side": {
			Type:        "STRING",
			Description: "Order side",
			Enum:        []string{"buy", "sell"},
		},
		"quantity": {
			Type:        "NUMBER",
			Description: "Order quantity in base currency",
			Minimum:     float64Ptr(0.001),
		},
		"order_type": {
			Type:        "STRING",
			Description: "Type of order to place",
			Enum:        []string{"market", "limit", "stop_limit"},
		},
		"price": {
			Type:        "NUMBER",
			Description: "Limit price (required for limit and stop_limit orders)",
			Minimum:     float64Ptr(0.01),
		},
		"stop_price": {
			Type:        "NUMBER",
			Description: "Stop price (required for stop_limit orders)",
			Minimum:     float64Ptr(0.01),
		},
	},
	Required: []string{"symbol", "side", "quantity", "order_type"},
}

// ExecutorOutputSchema defines the output schema for the Executor agent
var ExecutorOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"order_id": {
			Type:        "STRING",
			Description: "Unique order identifier from exchange",
		},
		"status": {
			Type:        "STRING",
			Description: "Order execution status",
			Enum:        []string{"filled", "partially_filled", "pending", "rejected", "cancelled"},
		},
		"filled_qty": {
			Type:        "NUMBER",
			Description: "Quantity filled",
			Minimum:     float64Ptr(0),
		},
		"avg_price": {
			Type:        "NUMBER",
			Description: "Average fill price",
			Minimum:     float64Ptr(0),
		},
		"fee": {
			Type:        "NUMBER",
			Description: "Transaction fee in quote currency",
			Minimum:     float64Ptr(0),
		},
		"error": {
			Type:        "STRING",
			Description: "Error message if order failed",
		},
	},
	Required: []string{"order_id", "status"},
}

// RiskManagerOutputSchema defines the output schema for the Risk Manager agent
var RiskManagerOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"approved": {
			Type:        "BOOLEAN",
			Description: "Whether the trade passed risk validation",
		},
		"risk_score": {
			Type:        "NUMBER",
			Description: "Risk score from 0 (safest) to 100 (most risky)",
			Minimum:     float64Ptr(0),
			Maximum:     float64Ptr(100),
		},
		"reasons": {
			Type:        "ARRAY",
			Description: "List of reasons for approval/rejection",
			Items: &genai.Schema{
				Type: "STRING",
			},
		},
		"suggested_modifications": {
			Type:        "OBJECT",
			Description: "Suggested modifications to reduce risk (e.g. reduce position size)",
		},
		"max_position_size": {
			Type:        "NUMBER",
			Description: "Maximum recommended position size",
			Minimum:     float64Ptr(0),
		},
	},
	Required: []string{"approved", "risk_score", "reasons"},
}

// StrategyPlannerOutputSchema defines the output schema for the Strategy Planner agent
var StrategyPlannerOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"recommendation": {
			Type:        "STRING",
			Description: "Overall trade recommendation",
			Enum:        []string{"strong_buy", "buy", "hold", "sell", "strong_sell", "no_trade"},
		},
		"entry_price": {
			Type:        "NUMBER",
			Description: "Recommended entry price",
			Minimum:     float64Ptr(0),
		},
		"stop_loss": {
			Type:        "NUMBER",
			Description: "Stop loss price",
			Minimum:     float64Ptr(0),
		},
		"take_profit": {
			Type:        "ARRAY",
			Description: "Multiple take profit levels",
			Items: &genai.Schema{
				Type:    "NUMBER",
				Minimum: float64Ptr(0),
			},
		},
		"position_size": {
			Type:        "NUMBER",
			Description: "Recommended position size as percentage of portfolio",
			Minimum:     float64Ptr(0),
			Maximum:     float64Ptr(100),
		},
		"confidence": {
			Type:        "NUMBER",
			Description: "Confidence in this strategy (0-1)",
			Minimum:     float64Ptr(0),
			Maximum:     float64Ptr(1),
		},
		"time_horizon": {
			Type:        "STRING",
			Description: "Expected trade duration",
			Enum:        []string{"scalp", "intraday", "swing", "position"},
		},
		"key_factors": {
			Type:        "ARRAY",
			Description: "Key factors supporting this strategy",
			Items: &genai.Schema{
				Type: "STRING",
			},
		},
	},
	Required: []string{"recommendation", "confidence"},
}
