package bootstrap

import (
	"strings"
	"time"

	"google.golang.org/adk/session"

	"prometheus/internal/adapters/config"
	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/adapters/kafka"
	"prometheus/internal/agents"
	"prometheus/internal/agents/workflows"
	"prometheus/internal/domain/derivatives"
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/journal"
	"prometheus/internal/domain/macro"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/domain/onchain"
	"prometheus/internal/domain/order"
	"prometheus/internal/domain/position"
	"prometheus/internal/domain/regime"
	"prometheus/internal/domain/sentiment"
	"prometheus/internal/domain/trading_pair"
	"prometheus/internal/domain/user"
	"prometheus/internal/events"
	regimeml "prometheus/internal/ml/regime"
	analysisservice "prometheus/internal/services/analysis"
	riskservice "prometheus/internal/services/risk"
	"prometheus/internal/workers"
	"prometheus/internal/workers/analysis"
	derivworkers "prometheus/internal/workers/derivatives"
	"prometheus/internal/workers/evaluation"
	macroworkers "prometheus/internal/workers/macro"
	"prometheus/internal/workers/marketdata"
	onchainworkers "prometheus/internal/workers/onchain"
	sentimentworkers "prometheus/internal/workers/sentiment"
	"prometheus/internal/workers/trading"
	"prometheus/pkg/crypto"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// provideWorkers initializes all background workers
func provideWorkers(
	userRepo user.Repository,
	tradingPairRepo trading_pair.Repository,
	orderRepo order.Repository,
	positionRepo position.Repository,
	journalRepo journal.Repository,
	marketDataRepo market_data.Repository,
	regimeRepo regime.Repository,
	sentimentRepo sentiment.Repository,
	onchainRepo onchain.Repository,
	derivRepo derivatives.Repository,
	macroRepo macro.Repository,
	exchangeAccountRepo exchange_account.Repository,
	riskEngine *riskservice.RiskEngine,
	exchFactory exchanges.Factory,
	marketDataFactory exchanges.CentralFactory,
	encryptor *crypto.Encryptor,
	kafkaProducer *kafka.Producer,
	agentFactory *agents.Factory,
	agentRegistry *agents.Registry,
	adkSessionService session.Service,
	defaultProvider string,
	defaultModel string,
	cfg *config.Config,
	log *logger.Logger,
) *workers.Scheduler {
	log.Info("Initializing workers...")

	scheduler := workers.NewScheduler()

	// Create workflow factory for market research and personal trading workflows
	workflowFactory := workflows.NewFactory(agentFactory, defaultProvider, defaultModel)

	// Phase 3: Create PathSelector for intelligent routing between fast-path and committee
	pathSelector, err := workflowFactory.CreatePathSelector()
	if err != nil {
		log.Fatal("Failed to create PathSelector", "error", err)
	}
	log.Info("PathSelector created successfully",
		"high_stakes_threshold", pathSelector.GetMetrics()["high_stakes_threshold"],
		"volatility_threshold", pathSelector.GetMetrics()["volatility_threshold"],
		"target_committee_pct", pathSelector.GetMetrics()["target_committee_pct"],
	)

	// Get configuration values
	baseAssets := cfg.Workers.GetBaseAssets()
	quoteCurrency := cfg.Workers.QuoteCurrency
	exchanges := cfg.Workers.GetExchanges()
	timeframes := cfg.Workers.GetTimeframes()
	primaryExchange := cfg.Workers.PrimaryExch
	orderBookDepth := cfg.Workers.OrderBookDepth

	// Build default symbols from base assets + quote currency
	defaultSymbols := make([]string, len(baseAssets))
	for i, asset := range baseAssets {
		defaultSymbols[i] = asset + "/" + quoteCurrency
	}

	// ========================================
	// Trading Workers (high frequency)
	// ========================================

	// Event publisher for position events
	eventPublisher := events.NewPublisher(kafkaProducer, log)

	// Position event generator for trigger checking
	positionEventGenerator := trading.NewPositionEventGenerator(eventPublisher, log)

	// Position monitor: Updates PnL and checks SL/TP
	scheduler.RegisterWorker(trading.NewPositionMonitor(
		userRepo,
		positionRepo,
		exchangeAccountRepo,
		exchFactory,
		*encryptor,
		kafkaProducer,
		positionEventGenerator,
		cfg.Workers.PositionMonitorInterval,
		true, // enabled
	))

	// Order sync: Syncs order status with exchanges
	scheduler.RegisterWorker(trading.NewOrderSync(
		userRepo,
		orderRepo,
		positionRepo,
		exchangeAccountRepo,
		exchFactory,
		*encryptor,
		kafkaProducer,
		cfg.Workers.OrderSyncInterval,
		true, // enabled
	))

	// Risk monitor: Checks circuit breakers and risk limits
	scheduler.RegisterWorker(trading.NewRiskMonitor(
		userRepo,
		riskEngine,
		positionRepo,
		kafkaProducer,
		cfg.Workers.RiskMonitorInterval,
		true, // enabled
	))

	// PnL calculator: Calculates daily PnL and updates risk state
	scheduler.RegisterWorker(trading.NewPnLCalculator(
		userRepo,
		positionRepo,
		riskEngine,
		kafkaProducer,
		cfg.Workers.PnLCalculatorInterval,
		true, // enabled
	))

	// ========================================
	// Market Data Workers (medium frequency)
	// ========================================

	// OHLCV collector: Collects candles
	// Pass base assets and quote currency - each exchange will format symbols accordingly
	scheduler.RegisterWorker(marketdata.NewOHLCVCollector(
		marketDataRepo,
		exchFactory,
		baseAssets,
		quoteCurrency,
		timeframes,
		cfg.Workers.OHLCVCollectorInterval,
		true, // enabled
	))

	// Ticker collector: Collects real-time prices
	scheduler.RegisterWorker(marketdata.NewTickerCollector(
		marketDataRepo,
		marketDataFactory,
		defaultSymbols,
		exchanges,
		cfg.Workers.TickerCollectorInterval,
		true, // enabled
	))

	// OrderBook collector: Collects order book snapshots
	scheduler.RegisterWorker(marketdata.NewOrderBookCollector(
		marketDataRepo,
		marketDataFactory,
		defaultSymbols,
		exchanges,
		orderBookDepth,
		cfg.Workers.OrderBookCollectorInterval,
		true, // enabled
	))

	// Trades collector: Collects real-time trades
	scheduler.RegisterWorker(marketdata.NewTradesCollector(
		marketDataRepo,
		marketDataFactory,
		defaultSymbols,
		exchanges,
		100, // limit per request
		cfg.Workers.TradesCollectorInterval,
		true, // enabled
	))

	// Funding collector: Collects funding rates from futures
	scheduler.RegisterWorker(marketdata.NewFundingCollector(
		marketDataRepo,
		marketDataFactory,
		defaultSymbols,
		exchanges,
		cfg.Workers.FundingCollectorInterval,
		true, // enabled
	))

	// ========================================
	// Sentiment Workers (low frequency)
	// ========================================

	// Use base assets from config for sentiment tracking
	scheduler.RegisterWorker(sentimentworkers.NewNewsCollector(
		sentimentRepo,
		cfg.MarketData.NewsAPIKey,
		baseAssets,
		cfg.Workers.NewsCollectorInterval,
		true, // enabled
	))

	scheduler.RegisterWorker(sentimentworkers.NewTwitterCollector(
		sentimentRepo,
		cfg.MarketData.TwitterAPIKey,
		cfg.MarketData.TwitterAPISecret,
		cfg.Workers.GetTwitterAccounts(),
		defaultSymbols,
		cfg.Workers.TwitterCollectorInterval,
		cfg.MarketData.TwitterAPIKey != "",
	))

	scheduler.RegisterWorker(sentimentworkers.NewRedditCollector(
		sentimentRepo,
		cfg.MarketData.RedditClientID,
		cfg.MarketData.RedditClientSecret,
		cfg.Workers.GetRedditSubs(),
		defaultSymbols,
		cfg.Workers.RedditCollectorInterval,
		cfg.MarketData.RedditClientID != "",
	))

	scheduler.RegisterWorker(sentimentworkers.NewFearGreedCollector(
		sentimentRepo,
		cfg.Workers.FearGreedCollectorInterval,
		true, // Always enabled
	))

	// ========================================
	// On-Chain Workers (medium frequency)
	// ========================================

	scheduler.RegisterWorker(onchainworkers.NewWhaleMovementCollector(
		onchainRepo,
		cfg.MarketData.WhaleAlertAPIKey,
		cfg.Workers.GetBlockchains(),
		float64(cfg.Workers.MinWhaleAmount),
		cfg.Workers.WhaleMovementCollectorInterval,
		cfg.MarketData.WhaleAlertAPIKey != "",
	))

	scheduler.RegisterWorker(onchainworkers.NewExchangeFlowCollector(
		onchainRepo,
		cfg.MarketData.CryptoquantAPIKey,
		exchanges,  // Use same exchanges as market data
		baseAssets, // Use base assets for on-chain flow tracking
		cfg.Workers.ExchangeFlowCollectorInterval,
		cfg.MarketData.CryptoquantAPIKey != "",
	))

	scheduler.RegisterWorker(onchainworkers.NewNetworkMetricsCollector(
		onchainRepo,
		cfg.MarketData.BlockchainAPIKey,
		cfg.MarketData.EtherscanAPIKey,
		cfg.Workers.GetBlockchains(),
		cfg.Workers.NetworkMetricsCollectorInterval,
		true, // Always enabled
	))

	scheduler.RegisterWorker(onchainworkers.NewMinerMetricsCollector(
		onchainRepo,
		cfg.MarketData.GlassnodeAPIKey,
		cfg.Workers.GetMiningPools(),
		cfg.Workers.MinerMetricsCollectorInterval,
		cfg.MarketData.GlassnodeAPIKey != "",
	))

	// ========================================
	// Macro Workers (low frequency)
	// ========================================

	// Convert event type strings to EventType constants
	eventTypeStrs := cfg.Workers.GetMacroEventTypesAsStrings()
	eventTypes := make([]macro.EventType, 0, len(eventTypeStrs))
	for _, et := range eventTypeStrs {
		eventTypes = append(eventTypes, macro.EventType(strings.ToLower(et)))
	}

	scheduler.RegisterWorker(macroworkers.NewEconomicCalendarCollector(
		macroRepo,
		cfg.MarketData.TradingEconomicsKey,
		cfg.Workers.GetMacroCountries(),
		eventTypes,
		cfg.Workers.EconomicCalendarCollectorInterval,
		cfg.MarketData.TradingEconomicsKey != "",
	))

	scheduler.RegisterWorker(macroworkers.NewMarketCorrelationCollector(
		marketDataRepo,
		macroRepo,
		cfg.MarketData.AlphaVantageKey,
		cfg.Workers.GetCorrelationSymbols(),
		cfg.Workers.GetTraditionalAssets(),
		cfg.Workers.MarketCorrelationCollectorInterval,
		cfg.MarketData.AlphaVantageKey != "",
	))

	// ========================================
	// Derivatives Workers (medium frequency)
	// ========================================

	scheduler.RegisterWorker(derivworkers.NewOptionsFlowCollector(
		derivRepo,
		cfg.MarketData.DeribitAPIKey,
		cfg.MarketData.DeribitAPISecret,
		cfg.Workers.GetOptionsSymbols(),
		float64(cfg.Workers.MinOptionsPremium),
		cfg.Workers.OptionsFlowCollectorInterval,
		cfg.MarketData.DeribitAPIKey != "",
	))

	scheduler.RegisterWorker(derivworkers.NewGammaExposureCollector(
		derivRepo,
		cfg.MarketData.DeribitAPIKey,
		cfg.Workers.GetOptionsSymbols(),
		cfg.Workers.GammaExposureCollectorInterval,
		cfg.MarketData.DeribitAPIKey != "",
	))

	scheduler.RegisterWorker(derivworkers.NewFundingAggregator(
		marketDataRepo,
		exchanges,
		defaultSymbols,
		cfg.Workers.FundingAggregatorInterval,
		true, // Always enabled
	))

	log.Info("Worker intervals configured",
		"position_monitor", cfg.Workers.PositionMonitorInterval,
		"order_sync", cfg.Workers.OrderSyncInterval,
		"risk_monitor", cfg.Workers.RiskMonitorInterval,
		"pnl_calculator", cfg.Workers.PnLCalculatorInterval,
		"market_scanner", cfg.Workers.MarketScannerMaxConcurrency,
		"opportunity_finder", cfg.Workers.OpportunityFinderInterval,
	)

	// ========================================
	// Analysis Workers (core agentic system)
	// ========================================

	// Phase 5: Pre-screener for cost optimization
	preScreener := analysisservice.NewPreScreener(
		analysisservice.PreScreenConfig{
			Enabled:            true,
			MinPriceChangePct:  0.002, // 0.2%
			MinVolumePct:       0.50,  // 50%
			MinATRPct:          0.01,  // 1%
			CooldownDuration:   2 * time.Hour,
			CheckPriceMovement: true,
			CheckVolume:        true,
			CheckVolatility:    true,
			CheckOrderBook:     false,
			CheckCooldown:      true,
		},
		marketDataRepo,
	)

	// Opportunity finder: Runs market research workflow
	opportunityFinder, err := analysis.NewOpportunityFinder(
		workflowFactory,
		adkSessionService,
		templates.Get(),
		preScreener,
		defaultSymbols,
		primaryExchange,
		cfg.Workers.OpportunityFinderInterval,
		true, // enabled
	)
	if err != nil {
		log.Fatalf("Failed to create opportunity finder: %v", err)
	}
	scheduler.RegisterWorker(opportunityFinder)

	// Phase 4: ML-Based Regime Detection
	scheduler.RegisterWorker(analysis.NewFeatureExtractor(
		marketDataRepo,
		regimeRepo,
		defaultSymbols,
		1*time.Hour, // Run every hour
		true,        // enabled
	))

	// Load ONNX regime classifier (optional - graceful degradation)
	regimeClassifier, classifierErr := regimeml.NewClassifier("models/regime_detector.onnx")
	if classifierErr != nil {
		log.Warn("Regime ML classifier not available, will use algorithmic fallback", "error", classifierErr)
		scheduler.RegisterWorker(analysis.NewRegimeDetector(
			marketDataRepo,
			regimeRepo,
			kafkaProducer,
			defaultSymbols,
			cfg.Workers.RegimeDetectorInterval,
			true, // enabled
		))
	} else {
		log.Info("Regime ML classifier loaded successfully")
		defer regimeClassifier.Close()

		mlRegimeDetector, err := analysis.NewRegimeDetectorML(
			marketDataRepo,
			regimeRepo,
			regimeClassifier,
			nil, // interpreterAgent - will be used in future
			adkSessionService,
			kafkaProducer,
			defaultSymbols,
			cfg.Workers.RegimeDetectorInterval,
			true, // enabled
		)
		if err != nil {
			log.Fatal("Failed to create ML regime detector", "error", err)
		}
		scheduler.RegisterWorker(mlRegimeDetector)
	}

	// SMC scanner: Scans for Smart Money Concepts patterns
	scheduler.RegisterWorker(analysis.NewSMCScanner(
		marketDataRepo,
		kafkaProducer,
		defaultSymbols,
		cfg.Workers.SMCScannerInterval,
		true, // enabled
	))

	// ========================================
	// Evaluation Workers (low frequency)
	// ========================================

	scheduler.RegisterWorker(evaluation.NewStrategyEvaluator(
		userRepo,
		tradingPairRepo,
		journalRepo,
		kafkaProducer,
		cfg.Workers.StrategyEvaluatorInterval,
		true, // enabled
	))

	scheduler.RegisterWorker(evaluation.NewJournalCompiler(
		userRepo,
		positionRepo,
		journalRepo,
		kafkaProducer,
		cfg.Workers.JournalCompilerInterval,
		true, // enabled
	))

	scheduler.RegisterWorker(evaluation.NewDailyReport(
		userRepo,
		positionRepo,
		journalRepo,
		kafkaProducer,
		cfg.Workers.DailyReportInterval,
		true, // enabled
	))

	log.Infof("✓ Workers initialized: %d registered", len(scheduler.GetWorkers()))

	// Suppress unused warning
	_ = agentRegistry
	_ = pathSelector

	return scheduler
}
