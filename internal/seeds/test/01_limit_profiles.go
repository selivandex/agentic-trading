package test

import (
	"context"
	"strings"

	"prometheus/internal/testsupport/seeds"
)

// SeedLimitProfiles creates minimal limit profile for tests (idempotent)
func SeedLimitProfiles(ctx context.Context, s *seeds.Seeder) error {
	log := s.Log()

	_, err := s.LimitProfile().
		WithName("Test Default").
		WithDescription("Default limit profile for tests").
		WithActivePositions(3).
		WithFreeTier().
		Insert()

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			log.Infow("Limit profile already exists, skipping", "name", "Test Default")
			return nil
		}
		return err
	}

	log.Infow("Created limit profile", "name", "Test Default")
	return nil
}
