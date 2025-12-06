package dev

import (
	"context"
	"strings"

	"prometheus/internal/domain/limit_profile"
	"prometheus/internal/testsupport/seeds"
)

// SeedLimitProfiles creates limit profile seeds for development (idempotent)
func SeedLimitProfiles(ctx context.Context, s *seeds.Seeder) error {
	log := s.Log()

	profiles := []struct {
		name            string
		desc            string
		tier            func() limit_profile.Limits
		activePositions int
	}{
		{
			name:            "Conservative",
			desc:            "Conservative trading limits for beginners",
			tier:            limit_profile.FreeTierLimits,
			activePositions: 3,
		},
		{
			name:            "Moderate",
			desc:            "Moderate trading limits for experienced traders",
			tier:            limit_profile.BasicTierLimits,
			activePositions: 5,
		},
		{
			name:            "Aggressive",
			desc:            "Aggressive trading limits for professional traders",
			tier:            limit_profile.PremiumTierLimits,
			activePositions: 10,
		},
	}

	for _, p := range profiles {
		// Idempotent: skip if exists
		// Note: GetByName requires repository, but seeder only has DB
		// For now, use MustInsert and handle duplicate key gracefully

		// TODO: Add FindOrCreate method to builders
		// For now, we'll catch the error and skip
		builder := s.LimitProfile().
			WithName(p.name).
			WithDescription(p.desc).
			WithActivePositions(p.activePositions)

		// Apply tier limits
		switch p.name {
		case "Conservative":
			builder.WithFreeTier()
		case "Moderate":
			builder.WithBasicTier()
		case "Aggressive":
			builder.WithPremiumTier()
		}

		_, err := builder.Insert()
		if err != nil {
			// Skip if duplicate (idempotent)
			if strings.Contains(err.Error(), "duplicate key") {
				log.Infow("Limit profile already exists, skipping", "name", p.name)
				continue
			}
			return err
		}

		log.Infow("Created limit profile", "name", p.name)
	}

	return nil
}
