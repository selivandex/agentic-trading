package analysis

import (
	"context"

	"google.golang.org/adk/tool"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// MarketSnapshot contains raw analysis data from all tools.
type MarketSnapshot struct {
	Symbol            string
	Exchange          string
	Timeframe         string
	TechnicalAnalysis map[string]interface{}
	SMCAnalysis       map[string]interface{}
	MarketAnalysis    map[string]interface{}
}

// MarketSnapshotService fetches comprehensive market analysis without decision-making.
// Used for portfolio onboarding where we need data but not opportunity decisions.
type MarketSnapshotService struct {
	technicalTool tool.Tool
	smcTool       tool.Tool
	marketTool    tool.Tool
	log           *logger.Logger
}

// NewMarketSnapshotService creates a new market snapshot service.
func NewMarketSnapshotService(
	technicalTool tool.Tool,
	smcTool tool.Tool,
	marketTool tool.Tool,
) *MarketSnapshotService {
	return &MarketSnapshotService{
		technicalTool: technicalTool,
		smcTool:       smcTool,
		marketTool:    marketTool,
		log:           logger.Get().With("component", "market_snapshot"),
	}
}

// GetSnapshot fetches comprehensive market analysis for a symbol.
// Returns raw tool outputs without synthesis or decision-making.
func (s *MarketSnapshotService) GetSnapshot(
	ctx context.Context,
	symbol string,
	exchange string,
	timeframe string,
) (*MarketSnapshot, error) {
	s.log.Debug("Fetching market snapshot",
		"symbol", symbol,
		"exchange", exchange,
		"timeframe", timeframe,
	)

	snapshot := &MarketSnapshot{
		Symbol:    symbol,
		Exchange:  exchange,
		Timeframe: timeframe,
	}

	// TODO: Call tools in parallel for better performance
	// For now, call sequentially

	// Technical analysis
	technicalResult, err := s.callTechnicalTool(ctx, symbol, exchange, timeframe)
	if err != nil {
		s.log.Warn("Technical analysis failed", "error", err)
		// Continue with other tools even if one fails
	} else {
		snapshot.TechnicalAnalysis = technicalResult
	}

	// SMC analysis
	smcResult, err := s.callSMCTool(ctx, symbol, exchange, timeframe)
	if err != nil {
		s.log.Warn("SMC analysis failed", "error", err)
	} else {
		snapshot.SMCAnalysis = smcResult
	}

	// Market analysis
	marketResult, err := s.callMarketTool(ctx, symbol, exchange)
	if err != nil {
		s.log.Warn("Market analysis failed", "error", err)
	} else {
		snapshot.MarketAnalysis = marketResult
	}

	s.log.Info("Market snapshot fetched",
		"symbol", symbol,
		"technical_ok", snapshot.TechnicalAnalysis != nil,
		"smc_ok", snapshot.SMCAnalysis != nil,
		"market_ok", snapshot.MarketAnalysis != nil,
	)

	return snapshot, nil
}

// callTechnicalTool calls get_technical_analysis tool.
func (s *MarketSnapshotService) callTechnicalTool(
	ctx context.Context,
	symbol, exchange, timeframe string,
) (map[string]interface{}, error) {
	// TODO: Implement tool.Context wrapper for direct tool calls
	// For now, return placeholder
	return map[string]interface{}{
		"status": "tool_call_pending",
	}, errors.New("direct tool call not yet implemented - needs tool.Context")
}

// callSMCTool calls get_smc_analysis tool.
func (s *MarketSnapshotService) callSMCTool(
	ctx context.Context,
	symbol, exchange, timeframe string,
) (map[string]interface{}, error) {
	return map[string]interface{}{
		"status": "tool_call_pending",
	}, errors.New("direct tool call not yet implemented - needs tool.Context")
}

// callMarketTool calls get_market_analysis tool.
func (s *MarketSnapshotService) callMarketTool(
	ctx context.Context,
	symbol, exchange string,
) (map[string]interface{}, error) {
	return map[string]interface{}{
		"status": "tool_call_pending",
	}, errors.New("direct tool call not yet implemented - needs tool.Context")
}
