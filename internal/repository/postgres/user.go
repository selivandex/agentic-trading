package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"prometheus/internal/domain/user"
	"prometheus/pkg/logger"
)

// Compile-time check that we implement the interface
var _ user.Repository = (*UserRepository)(nil)

// UserRepository implements user.Repository using sqlx
type UserRepository struct {
	db  *sqlx.DB
	log *logger.Logger
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{
		db:  db,
		log: logger.Get().With("repository", "user"),
	}
}

// Create inserts a new user
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	// Marshal settings to JSON
	settingsJSON, err := json.Marshal(u.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	query := `
		INSERT INTO users (
			id, telegram_id, telegram_username, first_name, last_name,
			language_code, is_active, is_premium, settings, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	_, err = r.db.ExecContext(ctx, query,
		u.ID, u.TelegramID, u.TelegramUsername, u.FirstName, u.LastName,
		u.LanguageCode, u.IsActive, u.IsPremium, settingsJSON, u.CreatedAt, u.UpdatedAt,
	)

	return err
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			r.log.Warn("Slow query detected", "method", "GetByID", "duration", duration, "user_id", id)
		}
	}()

	var u user.User
	var settingsJSON []byte

	query := `SELECT * FROM users WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&u.ID, &u.TelegramID, &u.TelegramUsername, &u.FirstName, &u.LastName,
		&u.LanguageCode, &u.IsActive, &u.IsPremium, &settingsJSON, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Unmarshal settings
	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &u.Settings); err != nil {
			u.Settings = user.DefaultSettings()
		}
	} else {
		u.Settings = user.DefaultSettings()
	}

	r.log.Debug("User retrieved successfully", "user_id", id, "telegram_id", u.TelegramID)
	return &u, nil
}

// GetByTelegramID retrieves a user by Telegram ID
func (r *UserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			r.log.Warn("Slow query detected", "method", "GetByTelegramID", "duration", duration, "telegram_id", telegramID)
		}
	}()

	var u user.User
	var settingsJSON []byte

	query := `SELECT * FROM users WHERE telegram_id = $1`

	row := r.db.QueryRowContext(ctx, query, telegramID)
	err := row.Scan(
		&u.ID, &u.TelegramID, &u.TelegramUsername, &u.FirstName, &u.LastName,
		&u.LanguageCode, &u.IsActive, &u.IsPremium, &settingsJSON, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		r.log.Error("Failed to get user by telegram ID", "telegram_id", telegramID, "error", err)
		return nil, err
	}

	// Unmarshal settings
	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &u.Settings); err != nil {
			u.Settings = user.DefaultSettings()
		}
	} else {
		u.Settings = user.DefaultSettings()
	}

	r.log.Debug("User retrieved by telegram ID", "user_id", u.ID, "telegram_id", telegramID)
	return &u, nil
}

// Update updates user data
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			r.log.Warn("Slow query detected", "method", "Update", "duration", duration, "user_id", u.ID)
		}
	}()

	// Marshal settings to JSON
	settingsJSON, err := json.Marshal(u.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	query := `
		UPDATE users SET
			telegram_username = $2,
			first_name = $3,
			last_name = $4,
			language_code = $5,
			is_active = $6,
			is_premium = $7,
			settings = $8,
			updated_at = NOW()
		WHERE id = $1`

	_, err = r.db.ExecContext(ctx, query,
		u.ID, u.TelegramUsername, u.FirstName, u.LastName,
		u.LanguageCode, u.IsActive, u.IsPremium, settingsJSON,
	)

	if err != nil {
		r.log.Error("Failed to update user", "user_id", u.ID, "error", err)
		return err
	}

	r.log.Debug("User updated successfully", "user_id", u.ID)
	return nil
}

// Delete deletes a user by ID
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List retrieves paginated list of users
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*user.User, error) {
	var users []*user.User

	query := `
		SELECT id, telegram_id, telegram_username, first_name, last_name,
			   language_code, is_active, is_premium, settings, created_at, updated_at
		FROM users 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var u user.User
		var settingsJSON []byte

		err := rows.Scan(
			&u.ID, &u.TelegramID, &u.TelegramUsername, &u.FirstName, &u.LastName,
			&u.LanguageCode, &u.IsActive, &u.IsPremium, &settingsJSON, &u.CreatedAt, &u.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal settings
		if len(settingsJSON) > 0 {
			if err := json.Unmarshal(settingsJSON, &u.Settings); err != nil {
				u.Settings = user.DefaultSettings()
			}
		} else {
			u.Settings = user.DefaultSettings()
		}

		users = append(users, &u)
	}

	return users, rows.Err()
}
