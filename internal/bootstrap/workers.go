package bootstrap

import (
	"time"

	"google.golang.org/adk/session"

	"prometheus/internal/adapters/config"
	"prometheus/internal/adapters/exchangefactory"
	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/adapters/kafka"
	"prometheus/internal/agents"
	"prometheus/internal/agents/workflows"
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/fundwatchlist"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/domain/order"
	"prometheus/internal/domain/position"
	strategyDomain "prometheus/internal/domain/strategy"
	"prometheus/internal/domain/user"
	"prometheus/internal/events"
	riskservice "prometheus/internal/services/risk"
	"prometheus/internal/workers"
	"prometheus/internal/workers/analysis"
	"prometheus/internal/workers/trading"
	"prometheus/pkg/crypto"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// provideWorkers initializes all background workers for MVP
func provideWorkers(
	userRepo user.Repository,
	fundWatchlistRepo fundwatchlist.Repository,
	strategyRepo strategyDomain.Repository,
	orderRepo order.Repository,
	positionRepo position.Repository,
	marketDataRepo market_data.Repository,
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
	userExchangeFactory *exchangefactory.UserExchangeFactory,
) *workers.Scheduler {
	log.Info("Initializing MVP workers...")

	scheduler := workers.NewScheduler()

	// Create workflow factory for workflows
	workflowFactory := workflows.NewFactory(agentFactory, defaultProvider, defaultModel)

	// ========================================
	// Analysis Workers - Market Research
	// ========================================

	// OpportunityFinder - runs market research workflow for fund watchlist symbols
	opportunityFinder, err := analysis.NewOpportunityFinder(
		workflowFactory,
		adkSessionService,
		templates.Get(),
		nil, // preScreener disabled for MVP
		[]string{"BTC/USDT", "ETH/USDT", "SOL/USDT"}, // MVP symbols
		"binance", // Primary exchange
		15*time.Minute,
		true,
	)
	if err != nil {
		log.Warnw("Failed to create OpportunityFinder", "error", err)
	} else {
		scheduler.RegisterWorker(opportunityFinder)
	}

	// ========================================
	// Trading Workers - Position Management
	// ========================================

	// Event publisher for position events
	positionEventPublisher := events.NewPublisher(kafkaProducer, log)

	// Position Monitor - updates PnL and checks SL/TP
	positionEventGenerator := trading.NewPositionEventGenerator(positionEventPublisher, log)
	positionMonitor := trading.NewPositionMonitor(
		userRepo,
		positionRepo,
		exchangeAccountRepo,
		exchFactory,
		*encryptor, // Dereference pointer to get interface value
		kafkaProducer,
		positionEventGenerator,
		1*time.Minute,
		true,
	)
	scheduler.RegisterWorker(positionMonitor)

	// Risk Monitor - checks portfolio risk limits
	riskMonitor := trading.NewRiskMonitor(
		userRepo,
		riskEngine,
		positionRepo,
		kafkaProducer,
		2*time.Minute,
		true,
	)
	scheduler.RegisterWorker(riskMonitor)

	// PnL Calculator - calculates strategy equity
	pnlCalculator := trading.NewPnLCalculator(
		userRepo,
		positionRepo,
		riskEngine,
		kafkaProducer,
		5*time.Minute,
		true,
	)
	scheduler.RegisterWorker(pnlCalculator)

	// Order Executor - executes pending orders on exchanges
	orderExecutor := trading.NewOrderExecutor(
		orderRepo,
		exchangeAccountRepo,
		userExchangeFactory,
		5*time.Second, // Poll every 5 seconds for pending orders
		true,          // Enabled
	)
	scheduler.RegisterWorker(orderExecutor)

	log.Infow("✓ MVP workers initialized", "count", len(scheduler.GetWorkers()))

	// Suppress unused warning
	_ = agentRegistry
	_ = fundWatchlistRepo // Will be used when OpportunityFinder uses it

	return scheduler
}
