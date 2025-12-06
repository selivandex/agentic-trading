package dev

import (
	"context"

	"prometheus/internal/testsupport/seeds"
	"prometheus/pkg/logger"
)

// SeedUsers creates user seeds with nested entities for development
func SeedUsers(ctx context.Context, s *seeds.Seeder) error {
	log := logger.Get()

	// Developer user (John)
	johnUser := s.User().
		WithTelegramID(123456789).
		WithUsername("dev_user").
		WithFirstName("John").
		WithLastName("Developer").
		WithLanguageCode("en").
		WithActive(true).
		WithPremium(true).
		WithRiskLevel("moderate").
		MustInsert()

	log.Infow("Created user", "telegram_id", 123456789, "username", "dev_user")

	// John's exchange accounts
	johnBinance := s.ExchangeAccount().
		WithUserID(johnUser.ID).
		WithBinance().
		WithLabel("Dev Binance Testnet").
		WithTestnet(true).
		WithActive(true).
		WithAPIKey([]byte("test_api_key_binance")).
		WithSecret([]byte("test_secret_binance")).
		MustInsert()

	johnBybit := s.ExchangeAccount().
		WithUserID(johnUser.ID).
		WithBybit().
		WithLabel("Dev Bybit Testnet").
		WithTestnet(true).
		WithActive(true).
		WithAPIKey([]byte("test_api_key_bybit")).
		WithSecret([]byte("test_secret_bybit")).
		MustInsert()

	log.Infow("Created exchange accounts for user", "user_id", johnUser.ID, "count", 2)

	// John's strategies
	btcStrategy := s.Strategy().
		WithUserID(johnUser.ID).
		WithName("BTC Momentum Strategy").
		WithDescription("Momentum-based strategy for Bitcoin trading").
		WithActive().
		WithSpot().
		WithModerateRisk().
		MustInsert()

	portfolioStrategy := s.Strategy().
		WithUserID(johnUser.ID).
		WithName("Multi-Asset Portfolio").
		WithDescription("Diversified portfolio across multiple assets").
		WithActive().
		WithSpot().
		WithConservativeRisk().
		MustInsert()

	log.Infow("Created strategies for user", "user_id", johnUser.ID, "count", 2)

	// Test user (Jane)
	janeUser := s.User().
		WithTelegramID(987654321).
		WithUsername("test_user").
		WithFirstName("Jane").
		WithLastName("Tester").
		WithLanguageCode("en").
		WithActive(true).
		WithPremium(false).
		WithRiskLevel("conservative").
		MustInsert()

	log.Infow("Created user", "telegram_id", 987654321, "username", "test_user")

	// Jane's exchange account
	s.ExchangeAccount().
		WithUserID(janeUser.ID).
		WithBinance().
		WithLabel("Test Binance Account").
		WithTestnet(true).
		WithActive(true).
		WithAPIKey([]byte("test_api_key_jane")).
		WithSecret([]byte("test_secret_jane")).
		MustInsert()

	// Jane's strategy
	s.Strategy().
		WithUserID(janeUser.ID).
		WithName("Conservative BTC Strategy").
		WithDescription("Low-risk Bitcoin strategy").
		WithActive().
		WithSpot().
		WithConservativeRisk().
		MustInsert()

	log.Infow("Created strategy for user", "user_id", janeUser.ID)

	// Suppress unused vars
	_ = johnBinance
	_ = johnBybit
	_ = btcStrategy
	_ = portfolioStrategy

	return nil
}
