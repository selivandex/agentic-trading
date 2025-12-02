package seeds

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/domain/limit_profile"
	"prometheus/internal/testsupport"
)

// LimitProfileBuilder provides a fluent API for creating LimitProfile entities
type LimitProfileBuilder struct {
	db     DBTX
	ctx    context.Context
	entity *limit_profile.LimitProfile
	limits *limit_profile.Limits
}

// NewLimitProfileBuilder creates a new LimitProfileBuilder with free tier defaults
func NewLimitProfileBuilder(db DBTX, ctx context.Context) *LimitProfileBuilder {
	now := time.Now()
	freeLimits := limit_profile.FreeTierLimits()

	limitsJSON, _ := json.Marshal(freeLimits)

	return &LimitProfileBuilder{
		db:  db,
		ctx: ctx,
		entity: &limit_profile.LimitProfile{
			ID:          uuid.New(),
			Name:        testsupport.UniqueName("test_profile"),
			Description: "Test limit profile",
			Limits:      limitsJSON,
			IsActive:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		limits: &freeLimits,
	}
}

// WithID sets a specific ID
func (b *LimitProfileBuilder) WithID(id uuid.UUID) *LimitProfileBuilder {
	b.entity.ID = id
	return b
}

// WithName sets the profile name
func (b *LimitProfileBuilder) WithName(name string) *LimitProfileBuilder {
	b.entity.Name = name
	return b
}

// WithDescription sets the description
func (b *LimitProfileBuilder) WithDescription(desc string) *LimitProfileBuilder {
	b.entity.Description = desc
	return b
}

// WithActive sets the active status
func (b *LimitProfileBuilder) WithActive(active bool) *LimitProfileBuilder {
	b.entity.IsActive = active
	return b
}

// WithFreeTier sets free tier limits
func (b *LimitProfileBuilder) WithFreeTier() *LimitProfileBuilder {
	limits := limit_profile.FreeTierLimits()
	b.limits = &limits
	limitsJSON, _ := json.Marshal(limits)
	b.entity.Limits = limitsJSON
	return b
}

// WithBasicTier sets basic tier limits
func (b *LimitProfileBuilder) WithBasicTier() *LimitProfileBuilder {
	limits := limit_profile.BasicTierLimits()
	b.limits = &limits
	limitsJSON, _ := json.Marshal(limits)
	b.entity.Limits = limitsJSON
	return b
}

// WithPremiumTier sets premium tier limits
func (b *LimitProfileBuilder) WithPremiumTier() *LimitProfileBuilder {
	limits := limit_profile.PremiumTierLimits()
	b.limits = &limits
	limitsJSON, _ := json.Marshal(limits)
	b.entity.Limits = limitsJSON
	return b
}

// WithLimits sets custom limits
func (b *LimitProfileBuilder) WithLimits(limits limit_profile.Limits) *LimitProfileBuilder {
	b.limits = &limits
	limitsJSON, _ := json.Marshal(limits)
	b.entity.Limits = limitsJSON
	return b
}

// WithExchangesCount sets the exchanges count limit
func (b *LimitProfileBuilder) WithExchangesCount(count int) *LimitProfileBuilder {
	b.limits.ExchangesCount = count
	limitsJSON, _ := json.Marshal(b.limits)
	b.entity.Limits = limitsJSON
	return b
}

// WithActivePositions sets the active positions limit
func (b *LimitProfileBuilder) WithActivePositions(count int) *LimitProfileBuilder {
	b.limits.ActivePositions = count
	limitsJSON, _ := json.Marshal(b.limits)
	b.entity.Limits = limitsJSON
	return b
}

// WithMonthlyAIRequests sets the monthly AI requests limit
func (b *LimitProfileBuilder) WithMonthlyAIRequests(count int) *LimitProfileBuilder {
	b.limits.MonthlyAIRequests = count
	limitsJSON, _ := json.Marshal(b.limits)
	b.entity.Limits = limitsJSON
	return b
}

// Build returns the built entity without inserting to DB
func (b *LimitProfileBuilder) Build() *limit_profile.LimitProfile {
	return b.entity
}

// Insert inserts the limit profile into the database and returns the entity
func (b *LimitProfileBuilder) Insert() (*limit_profile.LimitProfile, error) {
	query := `
		INSERT INTO limit_profiles (
			id, name, description, limits, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := b.db.ExecContext(
		b.ctx,
		query,
		b.entity.ID,
		b.entity.Name,
		b.entity.Description,
		b.entity.Limits,
		b.entity.IsActive,
		b.entity.CreatedAt,
		b.entity.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert limit profile: %w", err)
	}

	return b.entity, nil
}

// MustInsert inserts the limit profile and panics on error (useful for tests)
func (b *LimitProfileBuilder) MustInsert() *limit_profile.LimitProfile {
	entity, err := b.Insert()
	if err != nil {
		panic(err)
	}
	return entity
}

