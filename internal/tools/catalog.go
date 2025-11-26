package tools

import (
	"context"
	"fmt"

	"google.golang.org/adk/tool/functiontool"
)

// Definition describes a tool's metadata for registration and documentation.
type Definition struct {
	Name        string
	Description string
	Category    string
}

// toolDefinitions enumerates all tools outlined for phases 6 and 7.
var toolDefinitions = []Definition{
	{Name: "get_price", Description: "Fetch current price with bid/ask spread", Category: "market_data"},
	{Name: "get_ohlcv", Description: "Retrieve historical OHLCV candles", Category: "market_data"},
	{Name: "get_orderbook", Description: "Get depth snapshot for a trading pair", Category: "market_data"},
	{Name: "get_trades", Description: "Stream or fetch recent trades", Category: "market_data"},
	{Name: "get_funding_rate", Description: "Return latest perpetual funding rate", Category: "market_data"},
	{Name: "get_open_interest", Description: "Query futures open interest", Category: "market_data"},
	{Name: "get_long_short_ratio", Description: "Provide long/short positioning ratio", Category: "market_data"},
	{Name: "get_liquidations", Description: "List recent liquidation events", Category: "market_data"},

	{Name: "get_trade_imbalance", Description: "Calculate buy vs sell pressure", Category: "order_flow"},
	{Name: "get_cvd", Description: "Compute cumulative volume delta", Category: "order_flow"},
	{Name: "get_whale_trades", Description: "Detect unusually large trades", Category: "order_flow"},
	{Name: "get_tick_speed", Description: "Measure trade velocity", Category: "order_flow"},
	{Name: "get_orderbook_imbalance", Description: "Bid/ask volume delta across levels", Category: "order_flow"},
	{Name: "detect_iceberg", Description: "Identify iceberg order patterns", Category: "order_flow"},
	{Name: "detect_spoofing", Description: "Spot spoofing or layered orders", Category: "order_flow"},
	{Name: "get_absorption_zones", Description: "Highlight liquidity absorption zones", Category: "order_flow"},

	{Name: "rsi", Description: "Relative Strength Index", Category: "momentum"},
	{Name: "stochastic", Description: "Stochastic oscillator", Category: "momentum"},
	{Name: "cci", Description: "Commodity Channel Index", Category: "momentum"},
	{Name: "roc", Description: "Rate of Change momentum", Category: "momentum"},

	{Name: "atr", Description: "Average True Range for volatility", Category: "volatility"},
	{Name: "bollinger", Description: "Bollinger Bands", Category: "volatility"},
	{Name: "keltner", Description: "Keltner Channels", Category: "volatility"},

	{Name: "sma", Description: "Simple Moving Average", Category: "trend"},
	{Name: "ema", Description: "Exponential Moving Average", Category: "trend"},
	{Name: "ema_ribbon", Description: "EMA ribbon across multiple periods", Category: "trend"},
	{Name: "macd", Description: "Moving Average Convergence Divergence", Category: "trend"},
	{Name: "supertrend", Description: "Supertrend indicator", Category: "trend"},
	{Name: "ichimoku", Description: "Ichimoku Cloud signals", Category: "trend"},
	{Name: "pivot_points", Description: "Classical and Fibonacci pivot points", Category: "trend"},
	{Name: "fibonacci", Description: "Fibonacci retracement and extension levels", Category: "trend"},

	{Name: "vwap", Description: "Volume Weighted Average Price", Category: "volume"},
	{Name: "obv", Description: "On-Balance Volume", Category: "volume"},
	{Name: "volume_profile", Description: "Volume profile by price", Category: "volume"},
	{Name: "delta_volume", Description: "Buy versus sell volume delta", Category: "volume"},

	{Name: "detect_fvg", Description: "Detect fair value gaps", Category: "smc"},
	{Name: "detect_order_blocks", Description: "Identify order blocks", Category: "smc"},
	{Name: "detect_liquidity_zones", Description: "Locate liquidity pools", Category: "smc"},
	{Name: "detect_stop_hunt", Description: "Spot stop-hunting moves", Category: "smc"},
	{Name: "get_market_structure", Description: "Detect BOS/CHoCH market structure", Category: "smc"},
	{Name: "detect_imbalances", Description: "Find price inefficiencies", Category: "smc"},
	{Name: "get_swing_points", Description: "Return swing highs and lows", Category: "smc"},

	{Name: "get_fear_greed", Description: "Crypto Fear & Greed index", Category: "sentiment"},
	{Name: "get_social_sentiment", Description: "Aggregate Twitter/Reddit sentiment", Category: "sentiment"},
	{Name: "get_news", Description: "Fetch latest crypto news", Category: "sentiment"},
	{Name: "get_trending", Description: "Trending coins and topics", Category: "sentiment"},
	{Name: "get_funding_sentiment", Description: "Sentiment inferred from funding", Category: "sentiment"},

	{Name: "get_whale_movements", Description: "Large wallet transfer tracker", Category: "onchain"},
	{Name: "get_exchange_flows", Description: "Exchange inflow/outflow", Category: "onchain"},
	{Name: "get_miner_reserves", Description: "Miner reserve balances", Category: "onchain"},
	{Name: "get_active_addresses", Description: "Active addresses count", Category: "onchain"},
	{Name: "get_nvt_ratio", Description: "Network Value to Transactions", Category: "onchain"},
	{Name: "get_sopr", Description: "Spent Output Profit Ratio", Category: "onchain"},
	{Name: "get_mvrv", Description: "Market Value to Realized Value", Category: "onchain"},
	{Name: "get_realized_pnl", Description: "Realized profit and loss", Category: "onchain"},
	{Name: "get_stablecoin_flows", Description: "Stablecoin supply flows", Category: "onchain"},

	{Name: "get_economic_calendar", Description: "Upcoming macro events", Category: "macro"},
	{Name: "get_fed_rate", Description: "Current Federal Reserve rate", Category: "macro"},
	{Name: "get_fed_watch", Description: "FedWatch rate probabilities", Category: "macro"},
	{Name: "get_cpi", Description: "Consumer Price Index data", Category: "macro"},
	{Name: "get_pmi", Description: "Purchasing Managers' Index", Category: "macro"},
	{Name: "get_macro_impact", Description: "Historical macro event impact", Category: "macro"},

	{Name: "get_options_oi", Description: "Options open interest", Category: "derivatives"},
	{Name: "get_max_pain", Description: "Max pain price for options", Category: "derivatives"},
	{Name: "get_put_call_ratio", Description: "Put/Call ratio", Category: "derivatives"},
	{Name: "get_gamma_exposure", Description: "Dealer gamma exposure", Category: "derivatives"},
	{Name: "get_options_flow", Description: "Large options trades", Category: "derivatives"},
	{Name: "get_iv_surface", Description: "Implied volatility surface", Category: "derivatives"},

	{Name: "btc_dominance", Description: "Bitcoin market dominance", Category: "correlation"},
	{Name: "usdt_dominance", Description: "Stablecoin dominance", Category: "correlation"},
	{Name: "altcoin_correlation", Description: "Altcoin correlation to BTC", Category: "correlation"},
	{Name: "stock_correlation", Description: "Correlation to equities indices", Category: "correlation"},
	{Name: "dxy_correlation", Description: "Correlation to Dollar Index", Category: "correlation"},
	{Name: "gold_correlation", Description: "Correlation to gold", Category: "correlation"},
	{Name: "get_session_volume", Description: "Volume by trading session", Category: "correlation"},

	{Name: "get_balance", Description: "Retrieve account balances", Category: "execution"},
	{Name: "get_positions", Description: "List open positions", Category: "execution"},
	{Name: "place_order", Description: "Place a market or limit order", Category: "execution"},
	{Name: "place_bracket_order", Description: "Place bracket order with SL/TP", Category: "execution"},
	{Name: "place_ladder_order", Description: "Place laddered profit targets", Category: "execution"},
	{Name: "place_iceberg_order", Description: "Submit iceberg order", Category: "execution"},
	{Name: "cancel_order", Description: "Cancel a specific order", Category: "execution"},
	{Name: "cancel_all_orders", Description: "Cancel all open orders", Category: "execution"},
	{Name: "close_position", Description: "Close a single position", Category: "execution"},
	{Name: "close_all_positions", Description: "Close all positions", Category: "execution"},
	{Name: "set_leverage", Description: "Configure leverage", Category: "execution"},
	{Name: "set_margin_mode", Description: "Switch margin mode", Category: "execution"},
	{Name: "move_sl_to_breakeven", Description: "Adjust stop to breakeven", Category: "execution"},
	{Name: "set_trailing_stop", Description: "Enable trailing stop", Category: "execution"},
	{Name: "add_to_position", Description: "DCA or increase position size", Category: "execution"},

	{Name: "check_circuit_breaker", Description: "Check if trading is allowed", Category: "risk"},
	{Name: "get_daily_pnl", Description: "Return today's PnL", Category: "risk"},
	{Name: "get_max_drawdown", Description: "Current drawdown stats", Category: "risk"},
	{Name: "get_exposure", Description: "Total portfolio exposure", Category: "risk"},
	{Name: "calculate_position_size", Description: "Compute position size", Category: "risk"},
	{Name: "validate_trade", Description: "Pre-trade validation checks", Category: "risk"},
	{Name: "emergency_close_all", Description: "Kill switch to close everything", Category: "risk"},

	{Name: "store_memory", Description: "Persist a memory entry", Category: "memory"},
	{Name: "search_memory", Description: "Semantic memory search", Category: "memory"},
	{Name: "get_trade_history", Description: "Retrieve past trades", Category: "memory"},
	{Name: "get_market_regime", Description: "Get detected market regime", Category: "memory"},
	{Name: "store_market_regime", Description: "Persist market regime detection", Category: "memory"},

	{Name: "get_strategy_stats", Description: "Strategy performance stats", Category: "evaluation"},
	{Name: "log_trade_decision", Description: "Write trade decision to journal", Category: "evaluation"},
	{Name: "get_trade_journal", Description: "Retrieve trade journal entries", Category: "evaluation"},
	{Name: "evaluate_last_trades", Description: "Evaluate recent trades", Category: "evaluation"},
	{Name: "get_best_strategies", Description: "Top performing strategies", Category: "evaluation"},
	{Name: "get_worst_strategies", Description: "Underperforming strategies", Category: "evaluation"},
}

// Definitions exposes a copy of all tool definitions.
func Definitions() []Definition {
	defs := make([]Definition, len(toolDefinitions))
	copy(defs, toolDefinitions)
	return defs
}

// RegisterAllTools registers placeholder implementations for every defined tool.
func RegisterAllTools(registry *Registry) {
	for _, def := range toolDefinitions {
		definition := def
		registry.Register(definition.Name, functiontool.New(definition.Name, definition.Description, func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("tool %s not implemented", definition.Name)
		}))
	}
}
