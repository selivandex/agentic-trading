package seeds

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"prometheus/internal/domain/user"
	"prometheus/internal/testsupport"
	"prometheus/pkg/logger"
)

// UserBuilder provides a fluent API for creating User entities
type UserBuilder struct {
	db     DBTX
	log    *logger.Logger
	ctx    context.Context
	entity *user.User
}

// NewUserBuilder creates a new UserBuilder with sensible defaults
func NewUserBuilder(db DBTX, ctx context.Context, log *logger.Logger) *UserBuilder {
	now := time.Now()
	telegramID := testsupport.UniqueTelegramID()

	return &UserBuilder{
		db:  db,
		ctx: ctx,
		log: log,
		entity: &user.User{
			ID:               uuid.New(),
			TelegramID:       &telegramID,
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
	b.entity.TelegramID = &id
	return b
}

// WithUsername sets the Telegram username
func (b *UserBuilder) WithUsername(username string) *UserBuilder {
	b.entity.TelegramUsername = username
	return b
}

// WithEmail sets the email address
func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.entity.Email = &email
	return b
}

func (b *UserBuilder) WithPassword(password string) *UserBuilder {
	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		b.log.Errorw("Failed to hash password", "error", err)
		return b // Don't break chain - continue without password
	}

	hashString := string(passwordHash)
	b.entity.PasswordHash = &hashString
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
	// Marshal settings to JSON
	settingsJSON, err := json.Marshal(b.entity.Settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	query := `
		INSERT INTO users (
			id, telegram_id, telegram_username, email, password_hash, first_name, last_name,
			language_code, is_active, is_premium, limit_profile_id,
			settings, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err = b.db.ExecContext(
		b.ctx,
		query,
		b.entity.ID,
		b.entity.TelegramID,
		b.entity.TelegramUsername,
		b.entity.Email,
		b.entity.PasswordHash,
		b.entity.FirstName,
		b.entity.LastName,
		b.entity.LanguageCode,
		b.entity.IsActive,
		b.entity.IsPremium,
		b.entity.LimitProfileID,
		settingsJSON,
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
