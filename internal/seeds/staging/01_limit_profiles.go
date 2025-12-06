package staging

import (
	"context"
	"strings"

	"prometheus/internal/testsupport/seeds"
)

// SeedLimitProfiles creates limit profiles for staging (idempotent)
func SeedLimitProfiles(ctx context.Context, s *seeds.Seeder) error {
	log := s.Log()

	profiles := []struct {
		name            string
		desc            string
		activePositions int
		tier            string
	}{
		{"Starter", "Starter tier for new users", 2, "free"},
		{"Pro", "Pro tier for experienced traders", 5, "basic"},
		{"Expert", "Expert tier for professional traders", 10, "premium"},
	}

	for _, p := range profiles {
		builder := s.LimitProfile().
			WithName(p.name).
			WithDescription(p.desc).
			WithActivePositions(p.activePositions)

		switch p.tier {
		case "free":
			builder.WithFreeTier()
		case "basic":
			builder.WithBasicTier()
		case "premium":
			builder.WithPremiumTier()
		}

		_, err := builder.Insert()
		if err != nil {
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
