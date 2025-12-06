package test

import (
	"context"
	"strings"

	"prometheus/internal/testsupport/seeds"
)

// SeedUsers creates minimal test user (idempotent)
func SeedUsers(ctx context.Context, s *seeds.Seeder) error {
	log := s.Log()

	_, err := s.User().
		WithTelegramID(111111111).
		WithUsername("test_user").
		WithFirstName("Test").
		WithLastName("User").
		WithLanguageCode("en").
		WithActive(true).
		WithPremium(false).
		WithRiskLevel("conservative").
		Insert()

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			log.Infow("User already exists, skipping", "telegram_id", 111111111)
			return nil
		}
		return err
	}

	log.Infow("Created test user", "telegram_id", 111111111)
	return nil
}
