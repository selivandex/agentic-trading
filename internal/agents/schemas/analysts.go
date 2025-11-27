package schemas

import "google.golang.org/genai"

// Common reasoning trace schema used by all analysts
func reasoningTraceSchema() *genai.Schema {
	return &genai.Schema{
		Type:        "ARRAY",
		Description: "Step-by-step reasoning process documenting observations and conclusions",
		Items: &genai.Schema{
			Type: "OBJECT",
			Properties: map[string]*genai.Schema{
				"step_number": {
					Type:        "INTEGER",
					Description: "Sequential step number (1, 2, 3, ...)",
					Minimum:     float64Ptr(1),
				},
				"step_name": {
					Type:        "STRING",
					Description: "Name/type of this reasoning step (e.g., 'assess_context', 'gather_data', 'analyze', 'conclude')",
				},
				"observation": {
					Type:        "STRING",
					Description: "What was observed or analyzed in this step",
				},
				"tool_used": {
					Type:        "STRING",
					Description: "Name of tool called in this step (null if no tool used)",
				},
				"tool_result_summary": {
					Type:        "STRING",
					Description: "Brief summary of tool output (null if no tool used)",
				},
				"conclusion": {
					Type:        "STRING",
					Description: "What was concluded from this step, what's next",
				},
			},
			Required: []string{"step_number", "step_name", "observation", "conclusion"},
		},
	}
}

// Common tool calls summary schema
func toolCallsSummarySchema() *genai.Schema {
	return &genai.Schema{
		Type:        "ARRAY",
		Description: "Summary of all tools called during analysis",
		Items: &genai.Schema{
			Type: "OBJECT",
			Properties: map[string]*genai.Schema{
				"tool": {
					Type:        "STRING",
					Description: "Tool name",
				},
				"result": {
					Type:        "STRING",
					Description: "Result status or brief outcome",
				},
			},
			Required: []string{"tool", "result"},
		},
	}
}

// MarketAnalystOutputSchema defines structured CoT output for Market Analyst
var MarketAnalystOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"reasoning_trace":    reasoningTraceSchema(),
		"tool_calls_summary": toolCallsSummarySchema(),
		"final_analysis": {
			Type:        "OBJECT",
			Description: "Final market analysis with structured findings",
			Properties: map[string]*genai.Schema{
				"direction": {
					Type:        "STRING",
					Description: "Market direction assessment",
					Enum:        []string{"bullish", "bearish", "neutral"},
				},
				"confidence": {
					Type:        "NUMBER",
					Description: "Confidence in this analysis (0-1)",
					Minimum:     float64Ptr(0),
					Maximum:     float64Ptr(1),
				},
				"key_findings": {
					Type:        "ARRAY",
					Description: "3-5 key findings from the analysis",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"supporting_factors": {
					Type:        "ARRAY",
					Description: "Factors supporting the direction",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"risk_factors": {
					Type:        "ARRAY",
					Description: "Risk factors or caveats",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"key_levels": {
					Type:        "ARRAY",
					Description: "Important support/resistance levels",
					Items: &genai.Schema{
						Type: "OBJECT",
						Properties: map[string]*genai.Schema{
							"type": {
								Type: "STRING",
								Enum: []string{"support", "resistance"},
							},
							"price": {
								Type:    "NUMBER",
								Minimum: float64Ptr(0),
							},
							"strength": {
								Type:        "STRING",
								Description: "Level strength",
								Enum:        []string{"strong", "moderate", "weak"},
							},
						},
					},
				},
				"rationale": {
					Type:        "STRING",
					Description: "2-3 sentence summary of the analysis rationale",
				},
			},
			Required: []string{"direction", "confidence", "key_findings", "rationale"},
		},
	},
	Required: []string{"reasoning_trace", "final_analysis", "tool_calls_summary"},
}

// SMCAnalystOutputSchema defines structured CoT output for Smart Money Concepts Analyst
var SMCAnalystOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"reasoning_trace":    reasoningTraceSchema(),
		"tool_calls_summary": toolCallsSummarySchema(),
		"final_analysis": {
			Type:        "OBJECT",
			Description: "Smart Money Concepts analysis",
			Properties: map[string]*genai.Schema{
				"direction": {
					Type: "STRING",
					Enum: []string{"bullish", "bearish", "neutral"},
				},
				"confidence": {
					Type:    "NUMBER",
					Minimum: float64Ptr(0),
					Maximum: float64Ptr(1),
				},
				"key_findings": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"supporting_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"risk_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"market_structure": {
					Type:        "STRING",
					Description: "Current market structure",
					Enum:        []string{"HH", "HL", "LH", "LL", "ranging"},
				},
				"fvg_zones": {
					Type:        "ARRAY",
					Description: "Fair Value Gap zones identified",
					Items: &genai.Schema{
						Type: "OBJECT",
						Properties: map[string]*genai.Schema{
							"type": {
								Type: "STRING",
								Enum: []string{"bullish_fvg", "bearish_fvg"},
							},
							"low": {
								Type:    "NUMBER",
								Minimum: float64Ptr(0),
							},
							"high": {
								Type:    "NUMBER",
								Minimum: float64Ptr(0),
							},
							"filled": {
								Type: "BOOLEAN",
							},
						},
					},
				},
				"rationale": {
					Type: "STRING",
				},
			},
			Required: []string{"direction", "confidence", "key_findings", "market_structure", "rationale"},
		},
	},
	Required: []string{"reasoning_trace", "final_analysis", "tool_calls_summary"},
}

// SentimentAnalystOutputSchema defines structured CoT output for Sentiment Analyst
var SentimentAnalystOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"reasoning_trace":    reasoningTraceSchema(),
		"tool_calls_summary": toolCallsSummarySchema(),
		"final_analysis": {
			Type:        "OBJECT",
			Description: "Market sentiment analysis",
			Properties: map[string]*genai.Schema{
				"direction": {
					Type: "STRING",
					Enum: []string{"bullish", "bearish", "neutral"},
				},
				"confidence": {
					Type:    "NUMBER",
					Minimum: float64Ptr(0),
					Maximum: float64Ptr(1),
				},
				"key_findings": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"supporting_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"risk_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"sentiment_score": {
					Type:        "NUMBER",
					Description: "Overall sentiment score from -1 (bearish) to +1 (bullish)",
					Minimum:     float64Ptr(-1),
					Maximum:     float64Ptr(1),
				},
				"fear_greed_index": {
					Type:        "NUMBER",
					Description: "Fear & Greed index value (0-100)",
					Minimum:     float64Ptr(0),
					Maximum:     float64Ptr(100),
				},
				"rationale": {
					Type: "STRING",
				},
			},
			Required: []string{"direction", "confidence", "key_findings", "sentiment_score", "rationale"},
		},
	},
	Required: []string{"reasoning_trace", "final_analysis", "tool_calls_summary"},
}

// OrderFlowAnalystOutputSchema defines structured CoT output for Order Flow Analyst
var OrderFlowAnalystOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"reasoning_trace":    reasoningTraceSchema(),
		"tool_calls_summary": toolCallsSummarySchema(),
		"final_analysis": {
			Type:        "OBJECT",
			Description: "Order flow analysis",
			Properties: map[string]*genai.Schema{
				"direction": {
					Type: "STRING",
					Enum: []string{"bullish", "bearish", "neutral"},
				},
				"confidence": {
					Type:    "NUMBER",
					Minimum: float64Ptr(0),
					Maximum: float64Ptr(1),
				},
				"key_findings": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"supporting_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"risk_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"flow_bias": {
					Type:        "STRING",
					Description: "Current orderflow bias",
					Enum:        []string{"aggressive_buying", "passive_buying", "neutral", "passive_selling", "aggressive_selling"},
				},
				"cvd_trend": {
					Type:        "STRING",
					Description: "Cumulative Volume Delta trend",
					Enum:        []string{"strong_positive", "positive", "neutral", "negative", "strong_negative"},
				},
				"rationale": {
					Type: "STRING",
				},
			},
			Required: []string{"direction", "confidence", "key_findings", "flow_bias", "cvd_trend", "rationale"},
		},
	},
	Required: []string{"reasoning_trace", "final_analysis", "tool_calls_summary"},
}

// DerivativesAnalystOutputSchema defines structured CoT output for Derivatives Analyst
var DerivativesAnalystOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"reasoning_trace":    reasoningTraceSchema(),
		"tool_calls_summary": toolCallsSummarySchema(),
		"final_analysis": {
			Type:        "OBJECT",
			Description: "Derivatives market analysis",
			Properties: map[string]*genai.Schema{
				"direction": {
					Type: "STRING",
					Enum: []string{"bullish", "bearish", "neutral"},
				},
				"confidence": {
					Type:    "NUMBER",
					Minimum: float64Ptr(0),
					Maximum: float64Ptr(1),
				},
				"key_findings": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"supporting_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"risk_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"funding_rate": {
					Type:        "NUMBER",
					Description: "Current funding rate (positive = longs pay shorts)",
				},
				"oi_change": {
					Type:        "NUMBER",
					Description: "Open Interest percentage change",
				},
				"options_flow": {
					Type:        "STRING",
					Description: "Options flow bias",
					Enum:        []string{"bullish_calls", "bearish_puts", "neutral", "mixed"},
				},
				"rationale": {
					Type: "STRING",
				},
			},
			Required: []string{"direction", "confidence", "key_findings", "funding_rate", "oi_change", "rationale"},
		},
	},
	Required: []string{"reasoning_trace", "final_analysis", "tool_calls_summary"},
}

// MacroAnalystOutputSchema defines structured CoT output for Macro Analyst
var MacroAnalystOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"reasoning_trace":    reasoningTraceSchema(),
		"tool_calls_summary": toolCallsSummarySchema(),
		"final_analysis": {
			Type:        "OBJECT",
			Description: "Macro economic and market analysis",
			Properties: map[string]*genai.Schema{
				"direction": {
					Type: "STRING",
					Enum: []string{"bullish", "bearish", "neutral"},
				},
				"confidence": {
					Type:    "NUMBER",
					Minimum: float64Ptr(0),
					Maximum: float64Ptr(1),
				},
				"key_findings": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"supporting_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"risk_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"regime": {
					Type:        "STRING",
					Description: "Current market risk regime",
					Enum:        []string{"risk_on", "risk_off", "transitioning", "uncertain"},
				},
				"catalysts": {
					Type:        "ARRAY",
					Description: "Key upcoming catalysts or events",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"rationale": {
					Type: "STRING",
				},
			},
			Required: []string{"direction", "confidence", "key_findings", "regime", "rationale"},
		},
	},
	Required: []string{"reasoning_trace", "final_analysis", "tool_calls_summary"},
}

// OnChainAnalystOutputSchema defines structured CoT output for On-Chain Analyst
var OnChainAnalystOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"reasoning_trace":    reasoningTraceSchema(),
		"tool_calls_summary": toolCallsSummarySchema(),
		"final_analysis": {
			Type:        "OBJECT",
			Description: "On-chain data analysis",
			Properties: map[string]*genai.Schema{
				"direction": {
					Type: "STRING",
					Enum: []string{"bullish", "bearish", "neutral"},
				},
				"confidence": {
					Type:    "NUMBER",
					Minimum: float64Ptr(0),
					Maximum: float64Ptr(1),
				},
				"key_findings": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"supporting_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"risk_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"whale_activity": {
					Type:        "STRING",
					Description: "Whale wallet activity assessment",
					Enum:        []string{"accumulation", "distribution", "neutral", "mixed"},
				},
				"exchange_flows": {
					Type:        "STRING",
					Description: "Exchange flow direction",
					Enum:        []string{"inflow", "outflow", "balanced"},
				},
				"rationale": {
					Type: "STRING",
				},
			},
			Required: []string{"direction", "confidence", "key_findings", "whale_activity", "exchange_flows", "rationale"},
		},
	},
	Required: []string{"reasoning_trace", "final_analysis", "tool_calls_summary"},
}

// CorrelationAnalystOutputSchema defines structured CoT output for Correlation Analyst
var CorrelationAnalystOutputSchema = &genai.Schema{
	Type: "OBJECT",
	Properties: map[string]*genai.Schema{
		"reasoning_trace":    reasoningTraceSchema(),
		"tool_calls_summary": toolCallsSummarySchema(),
		"final_analysis": {
			Type:        "OBJECT",
			Description: "Cross-asset correlation analysis",
			Properties: map[string]*genai.Schema{
				"direction": {
					Type: "STRING",
					Enum: []string{"bullish", "bearish", "neutral"},
				},
				"confidence": {
					Type:    "NUMBER",
					Minimum: float64Ptr(0),
					Maximum: float64Ptr(1),
				},
				"key_findings": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"supporting_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"risk_factors": {
					Type: "ARRAY",
					Items: &genai.Schema{
						Type: "STRING",
					},
				},
				"correlation_regime": {
					Type:        "STRING",
					Description: "Current correlation regime",
					Enum:        []string{"high_correlation", "moderate_correlation", "low_correlation", "decoupling"},
				},
				"btc_correlation": {
					Type:        "NUMBER",
					Description: "Correlation coefficient with BTC (-1 to +1)",
					Minimum:     float64Ptr(-1),
					Maximum:     float64Ptr(1),
				},
				"rationale": {
					Type: "STRING",
				},
			},
			Required: []string{"direction", "confidence", "key_findings", "correlation_regime", "btc_correlation", "rationale"},
		},
	},
	Required: []string{"reasoning_trace", "final_analysis", "tool_calls_summary"},
}
