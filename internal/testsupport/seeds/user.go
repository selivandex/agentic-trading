package seeds

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/domain/user"
	"prometheus/internal/testsupport"
)

// UserBuilder provides a fluent API for creating User entities
type UserBuilder struct {
	db     DBTX
	ctx    context.Context
	entity *user.User
}

// NewUserBuilder creates a new UserBuilder with sensible defaults
func NewUserBuilder(db DBTX, ctx context.Context) *UserBuilder {
	now := time.Now()
	return &UserBuilder{
		db:  db,
		ctx: ctx,
		entity: &user.User{
			ID:               uuid.New(),
			TelegramID:       testsupport.UniqueTelegramID(),
			TelegramUsername: testsupport.UniqueUsername(),
			FirstName:        "Test",
			LastName:         "User",
			LanguageCode:     "en",
			IsActive:         true,
			IsPremium:        false,
			LimitProfileID:   nil,
			Settings:         user.DefaultSettings(),
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}
}

// WithID sets a specific ID
func (b *UserBuilder) WithID(id uuid.UUID) *UserBuilder {
	b.entity.ID = id
	return b
}

// WithTelegramID sets the Telegram ID
func (b *UserBuilder) WithTelegramID(id int64) *UserBuilder {
	b.entity.TelegramID = id
	return b
}

// WithUsername sets the Telegram username
func (b *UserBuilder) WithUsername(username string) *UserBuilder {
	b.entity.TelegramUsername = username
	return b
}

// WithFirstName sets the first name
func (b *UserBuilder) WithFirstName(name string) *UserBuilder {
	b.entity.FirstName = name
	return b
}

// WithLastName sets the last name
func (b *UserBuilder) WithLastName(name string) *UserBuilder {
	b.entity.LastName = name
	return b
}

// WithLanguageCode sets the language code
func (b *UserBuilder) WithLanguageCode(code string) *UserBuilder {
	b.entity.LanguageCode = code
	return b
}

// WithActive sets the active status
func (b *UserBuilder) WithActive(active bool) *UserBuilder {
	b.entity.IsActive = active
	return b
}

// WithPremium sets the premium status
func (b *UserBuilder) WithPremium(premium bool) *UserBuilder {
	b.entity.IsPremium = premium
	return b
}

// WithLimitProfile sets the limit profile ID
func (b *UserBuilder) WithLimitProfile(profileID uuid.UUID) *UserBuilder {
	b.entity.LimitProfileID = &profileID
	return b
}

// WithSettings sets custom settings
func (b *UserBuilder) WithSettings(settings user.Settings) *UserBuilder {
	b.entity.Settings = settings
	return b
}

// WithRiskLevel sets the risk level in settings
func (b *UserBuilder) WithRiskLevel(level string) *UserBuilder {
	b.entity.Settings.RiskLevel = level
	return b
}

// WithMaxPositions sets max positions in settings
func (b *UserBuilder) WithMaxPositions(max int) *UserBuilder {
	b.entity.Settings.MaxPositions = max
	return b
}

// Build returns the built entity without inserting to DB
func (b *UserBuilder) Build() *user.User {
	return b.entity
}

// Insert inserts the user into the database and returns the entity
func (b *UserBuilder) Insert() (*user.User, error) {
	query := `
		INSERT INTO users (
			id, telegram_id, telegram_username, first_name, last_name,
			language_code, is_active, is_premium, limit_profile_id,
			settings, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := b.db.ExecContext(
		b.ctx,
		query,
		b.entity.ID,
		b.entity.TelegramID,
		b.entity.TelegramUsername,
		b.entity.FirstName,
		b.entity.LastName,
		b.entity.LanguageCode,
		b.entity.IsActive,
		b.entity.IsPremium,
		b.entity.LimitProfileID,
		b.entity.Settings,
		b.entity.CreatedAt,
		b.entity.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return b.entity, nil
}

// MustInsert inserts the user and panics on error (useful for tests)
func (b *UserBuilder) MustInsert() *user.User {
	entity, err := b.Insert()
	if err != nil {
		panic(err)
	}
	return entity
}

