package dev

import (
	"context"
	"strings"

	"prometheus/internal/testsupport/seeds"
)

// SeedUsers creates user seeds with nested entities for development (idempotent)
func SeedUsers(ctx context.Context, s *seeds.Seeder) error {
	log := s.Log()

	// Developer user (John)
	johnUser, err := s.User().
		WithTelegramID(123456789).
		WithUsername("dev_user").
		WithFirstName("John").
		WithLastName("Developer").
		WithLanguageCode("en").
		WithActive(true).
		WithPremium(true).
		WithRiskLevel("moderate").
		WithEmail("admin@prometheus.local").
		WithPassword("admin123").
		Insert()

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			log.Infow("User already exists, skipping", "telegram_id", 123456789)
			return nil // Idempotent: skip rest if user exists
		}
		return err
	}

	log.Infow("Created user", "telegram_id", 123456789, "username", "dev_user")

	// John's exchange accounts
	_, err = s.ExchangeAccount().
		WithUserID(johnUser.ID).
		WithBinance().
		WithLabel("Dev Binance Testnet").
		WithTestnet(true).
		WithActive(true).
		WithAPIKey([]byte("test_api_key_binance")).
		WithSecret([]byte("test_secret_binance")).
		Insert()
	if err != nil && !strings.Contains(err.Error(), "duplicate key") {
		return err
	}

	_, err = s.ExchangeAccount().
		WithUserID(johnUser.ID).
		WithBybit().
		WithLabel("Dev Bybit Testnet").
		WithTestnet(true).
		WithActive(true).
		WithAPIKey([]byte("test_api_key_bybit")).
		WithSecret([]byte("test_secret_bybit")).
		Insert()
	if err != nil && !strings.Contains(err.Error(), "duplicate key") {
		return err
	}

	log.Infow("Created exchange accounts for user", "user_id", johnUser.ID)

	// John's strategies
	_, err = s.Strategy().
		WithUserID(johnUser.ID).
		WithName("BTC Momentum Strategy").
		WithDescription("Momentum-based strategy for Bitcoin trading").
		WithActive().
		WithSpot().
		WithModerateRisk().
		Insert()
	if err != nil && !strings.Contains(err.Error(), "duplicate key") {
		return err
	}

	_, err = s.Strategy().
		WithUserID(johnUser.ID).
		WithName("Multi-Asset Portfolio").
		WithDescription("Diversified portfolio across multiple assets").
		WithActive().
		WithSpot().
		WithConservativeRisk().
		Insert()
	if err != nil && !strings.Contains(err.Error(), "duplicate key") {
		return err
	}

	log.Infow("Created strategies for user", "user_id", johnUser.ID)

	// Test user (Jane)
	janeUser, err := s.User().
		WithTelegramID(987654321).
		WithUsername("test_user").
		WithFirstName("Jane").
		WithLastName("Tester").
		WithLanguageCode("en").
		WithActive(true).
		WithPremium(false).
		WithRiskLevel("conservative").
		Insert()

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			log.Infow("User already exists, skipping", "telegram_id", 987654321)
			return nil
		}
		return err
	}

	log.Infow("Created user", "telegram_id", 987654321, "username", "test_user")

	// Jane's exchange account
	_, err = s.ExchangeAccount().
		WithUserID(janeUser.ID).
		WithBinance().
		WithLabel("Test Binance Account").
		WithTestnet(true).
		WithActive(true).
		WithAPIKey([]byte("test_api_key_jane")).
		WithSecret([]byte("test_secret_jane")).
		Insert()
	if err != nil && !strings.Contains(err.Error(), "duplicate key") {
		return err
	}

	// Jane's strategy
	_, err = s.Strategy().
		WithUserID(janeUser.ID).
		WithName("Conservative BTC Strategy").
		WithDescription("Low-risk Bitcoin strategy").
		WithActive().
		WithSpot().
		WithConservativeRisk().
		Insert()
	if err != nil && !strings.Contains(err.Error(), "duplicate key") {
		return err
	}

	log.Infow("Created strategy for user", "user_id", janeUser.ID)

	return nil
}
