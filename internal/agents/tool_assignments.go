package agents

// AgentToolMap defines allowed tools for each agent type.
var AgentToolMap = map[AgentType][]string{
	AgentMarketAnalyst: {
		"get_price",
		"get_ohlcv",
		"get_orderbook",
		"get_trades",
		"rsi",
		"stochastic",
		"cci",
		"roc",
		"atr",
		"bollinger",
		"keltner",
		"sma",
		"ema",
		"ema_ribbon",
		"macd",
		"supertrend",
		"ichimoku",
		"vwap",
		"obv",
		"volume_profile",
		"delta_volume",
		"fibonacci",
		"pivot_points",
		"get_swing_points",
	},

	AgentSMCAnalyst: {
		"get_ohlcv",
		"get_orderbook",
		"detect_fvg",
		"detect_order_blocks",
		"detect_liquidity_zones",
		"detect_stop_hunt",
		"get_market_structure",
		"detect_imbalances",
		"get_swing_points",
	},

	AgentSentimentAnalyst: {
		"get_fear_greed",
		"get_social_sentiment",
		"get_news",
		"get_trending",
		"get_funding_sentiment",
	},

	AgentOnChainAnalyst: {
		"get_whale_movements",
		"get_exchange_flows",
		"get_miner_reserves",
		"get_active_addresses",
		"get_nvt_ratio",
		"get_sopr",
		"get_mvrv",
		"get_realized_pnl",
		"get_stablecoin_flows",
	},

	AgentMacroAnalyst: {
		"get_economic_calendar",
		"get_fed_rate",
		"get_fed_watch",
		"get_cpi",
		"get_pmi",
		"get_macro_impact",
	},

	AgentOrderFlowAnalyst: {
		"get_orderbook",
		"get_trades",
		"get_trade_imbalance",
		"get_cvd",
		"get_whale_trades",
		"get_tick_speed",
		"get_orderbook_imbalance",
		"detect_iceberg",
		"detect_spoofing",
		"get_absorption_zones",
	},

	AgentDerivativesAnalyst: {
		"get_options_oi",
		"get_max_pain",
		"get_put_call_ratio",
		"get_gamma_exposure",
		"get_options_flow",
		"get_iv_surface",
	},

	AgentCorrelationAnalyst: {
		"btc_dominance",
		"usdt_dominance",
		"altcoin_correlation",
		"stock_correlation",
		"dxy_correlation",
		"gold_correlation",
		"get_session_volume",
	},

	AgentStrategyPlanner: {
		"search_memory",
		"get_trade_history",
		"get_market_regime",
		"get_strategy_stats",
	},

	AgentRiskManager: {
		"get_positions",
		"get_balance",
		"get_daily_pnl",
		"get_max_drawdown",
		"get_exposure",
		"calculate_position_size",
		"validate_trade",
		"check_circuit_breaker",
	},

	AgentExecutor: {
		"get_balance",
		"get_positions",
		"place_order",
		"place_bracket_order",
		"place_ladder_order",
		"cancel_order",
		"set_leverage",
		"set_margin_mode",
	},

	AgentPositionManager: {
		"get_positions",
		"get_balance",
		"move_sl_to_breakeven",
		"set_trailing_stop",
		"close_position",
		"add_to_position",
	},

	AgentSelfEvaluator: {
		"get_strategy_stats",
		"get_trade_journal",
		"evaluate_last_trades",
		"get_best_strategies",
		"get_worst_strategies",
	},
}
