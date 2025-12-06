package test

import (
	"context"

	"prometheus/internal/testsupport/seeds"
)

// SeedLimitProfiles creates minimal limit profile for tests
func SeedLimitProfiles(ctx context.Context, s *seeds.Seeder) error {
	s.LimitProfile().
		WithName("Test Default").
		WithDescription("Default limit profile for tests").
		WithActivePositions(3).
		WithFreeTier().
		MustInsert()

	return nil
}
