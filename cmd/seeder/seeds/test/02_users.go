package test

import (
	"context"

	"prometheus/internal/testsupport/seeds"
)

// SeedUsers creates minimal test user
func SeedUsers(ctx context.Context, s *seeds.Seeder) error {
	s.User().
		WithTelegramID(111111111).
		WithUsername("test_user").
		WithFirstName("Test").
		WithLastName("User").
		WithLanguageCode("en").
		WithActive(true).
		WithPremium(false).
		WithRiskLevel("conservative").
		MustInsert()

	return nil
}
