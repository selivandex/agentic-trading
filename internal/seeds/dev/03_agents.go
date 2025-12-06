package dev

import (
	"context"
	"strings"

	"prometheus/internal/adapters/ai"
	"prometheus/internal/domain/agent"
	"prometheus/internal/testsupport/seeds"
)

// SeedAgents creates AI agent definitions with DeepSeek reasoning model (idempotent)
func SeedAgents(ctx context.Context, s *seeds.Seeder) error {
	log := s.Log()

	// DeepSeek Reasoner - Deep thinking agent for complex analysis
	_, err := s.Agent().
		WithIdentifier("deep_reasoner").
		WithName("Deep Reasoner").
		WithDescription("Advanced reasoning agent powered by DeepSeek-R1 for complex market analysis").
		WithExpert().
		WithSystemPrompt(`You are an expert market analyst with deep reasoning capabilities.
Use chain-of-thought reasoning to analyze complex market situations.
Think step-by-step before making recommendations.`).
		WithInstructions(`1. Analyze market data thoroughly
2. Use reasoning steps to break down complex problems
3. Consider multiple perspectives and scenarios
4. Provide well-reasoned recommendations with confidence levels`).
		WithDeepSeek(ai.ModelDeepSeekReasoner).
		WithTemperature(1.0). // DeepSeek Reasoner works best at higher temperature
		WithMaxTokens(8000).
		WithTools([]string{
			"get_market_data",
			"calculate_indicators",
			"get_order_book",
			"analyze_sentiment",
		}).
		WithMaxCostPerRun(2.0).
		WithTimeout(600). // 10 minutes for deep reasoning
		WithActive(true).
		Insert()

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			log.Infow("Agent already exists, skipping", "identifier", "deep_reasoner")
		} else {
			return err
		}
	} else {
		log.Infow("Created agent", "identifier", "deep_reasoner", "model", "deepseek-reasoner-r1")
	}

	// Portfolio Architect - Strategic planning with reasoning
	_, err = s.Agent().
		WithIdentifier(agent.IdentifierPortfolioArchitect).
		WithName("Portfolio Architect").
		WithDescription("Strategic portfolio designer using reasoning model").
		WithCoordinator().
		WithSystemPrompt(`You are a portfolio architect specialized in designing optimal trading strategies.
Use deep reasoning to analyze risk-return profiles and create balanced portfolios.`).
		WithInstructions(`1. Analyze user's risk tolerance and capital
2. Reason through asset allocation strategies
3. Design diversified portfolio with clear rationale
4. Provide detailed reasoning for allocation decisions`).
		WithDeepSeek(ai.ModelDeepSeekReasoner).
		WithTemperature(1.0).
		WithMaxTokens(16000).
		WithTools([]string{
			"get_user_profile",
			"get_market_data",
			"calculate_correlation",
			"analyze_risk",
		}).
		WithMaxCostPerRun(3.0).
		WithTimeout(900). // 15 minutes
		WithActive(true).
		Insert()

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			log.Infow("Agent already exists, skipping", "identifier", agent.IdentifierPortfolioArchitect)
		} else {
			return err
		}
	} else {
		log.Infow("Created agent", "identifier", agent.IdentifierPortfolioArchitect, "model", "deepseek-reasoner-r1")
	}

	// Technical Analyzer - Fast analysis with Claude
	_, err = s.Agent().
		WithIdentifier(agent.IdentifierTechnicalAnalyzer).
		WithName("Technical Analyzer").
		WithDescription("Fast technical analysis expert using Claude").
		WithExpert().
		WithSystemPrompt(`You are a technical analysis expert.
Analyze charts, patterns, and indicators to identify trading opportunities.`).
		WithInstructions(`1. Analyze price action and chart patterns
2. Calculate and interpret technical indicators
3. Identify support/resistance levels
4. Provide clear entry/exit signals`).
		WithClaude(ai.ModelClaude45Sonnet).
		WithTemperature(0.3). // Lower temperature for consistent TA
		WithMaxTokens(4000).
		WithTools([]string{
			"get_ohlcv",
			"calculate_indicators",
			"detect_patterns",
			"analyze_volume",
		}).
		WithMaxCostPerRun(0.5).
		WithTimeout(180). // 3 minutes
		WithActive(true).
		Insert()

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			log.Infow("Agent already exists, skipping", "identifier", agent.IdentifierTechnicalAnalyzer)
		} else {
			return err
		}
	} else {
		log.Infow("Created agent", "identifier", agent.IdentifierTechnicalAnalyzer, "model", "claude-3.5-sonnet")
	}

	return nil
}
