package dev

import (
	"context"

	"prometheus/internal/testsupport/seeds"
)

// SeedLimitProfiles creates limit profile seeds for development
func SeedLimitProfiles(ctx context.Context, s *seeds.Seeder) error {
	// Conservative profile
	conservative := s.LimitProfile().
		WithName("Conservative").
		WithDescription("Conservative trading limits for beginners").
		WithActivePositions(3).
		WithFreeTier()

	if _, err := conservative.Insert(); err != nil {
		return err
	}

	// Moderate profile
	moderate := s.LimitProfile().
		WithName("Moderate").
		WithDescription("Moderate trading limits for experienced traders").
		WithActivePositions(5).
		WithBasicTier()

	if _, err := moderate.Insert(); err != nil {
		return err
	}

	// Aggressive profile
	aggressive := s.LimitProfile().
		WithName("Aggressive").
		WithDescription("Aggressive trading limits for professional traders").
		WithActivePositions(10).
		WithPremiumTier()

	if _, err := aggressive.Insert(); err != nil {
		return err
	}

	return nil
}
