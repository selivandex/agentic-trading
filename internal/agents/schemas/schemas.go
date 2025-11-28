package schemas

import "google.golang.org/genai"

// Helper functions for creating float64 pointers
func float64Ptr(v float64) *float64 {
	return &v
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
