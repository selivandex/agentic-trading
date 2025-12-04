package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"

	"prometheus/internal/domain/user"
	"prometheus/pkg/errors"
)

// Compile-time check that we implement the interface
var _ user.Repository = (*UserRepository)(nil)

// UserRepository implements user.Repository using sqlx
type UserRepository struct {
	db DBTX
}

// NewUserRepository creates a new user repository
func NewUserRepository(db DBTX) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	// Marshal settings to JSON
	settingsJSON, err := json.Marshal(u.Settings)
	if err != nil {
		return errors.Wrap(err, "failed to marshal settings")
	}

	query := `
		INSERT INTO users (
			id, telegram_id, telegram_username, email, password_hash, first_name, last_name,
			language_code, is_active, is_premium, limit_profile_id, settings, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)`

	_, err = r.db.ExecContext(ctx, query,
		u.ID, u.TelegramID, u.TelegramUsername, u.Email, u.PasswordHash, u.FirstName, u.LastName,
		u.LanguageCode, u.IsActive, u.IsPremium, u.LimitProfileID, settingsJSON, u.CreatedAt, u.UpdatedAt,
	)

	return err
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	var u user.User
	var settingsJSON []byte

	query := `
		SELECT id, telegram_id, telegram_username, email, password_hash, first_name, last_name,
			   language_code, is_active, is_premium, limit_profile_id, settings, created_at, updated_at
		FROM users
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&u.ID, &u.TelegramID, &u.TelegramUsername, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.LanguageCode, &u.IsActive, &u.IsPremium, &u.LimitProfileID, &settingsJSON, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(errors.ErrNotFound, "user not found")
	}
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

	return &u, nil
}

// GetByTelegramID retrieves a user by Telegram ID
func (r *UserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	var u user.User
	var settingsJSON []byte

	query := `
		SELECT id, telegram_id, telegram_username, email, password_hash, first_name, last_name,
			   language_code, is_active, is_premium, limit_profile_id, settings, created_at, updated_at
		FROM users
		WHERE telegram_id = $1`

	row := r.db.QueryRowContext(ctx, query, telegramID)
	err := row.Scan(
		&u.ID, &u.TelegramID, &u.TelegramUsername, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.LanguageCode, &u.IsActive, &u.IsPremium, &u.LimitProfileID, &settingsJSON, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(errors.ErrNotFound, "user not found")
	}
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

	return &u, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	var settingsJSON []byte

	query := `
		SELECT id, telegram_id, telegram_username, email, password_hash, first_name, last_name,
			   language_code, is_active, is_premium, limit_profile_id, settings, created_at, updated_at
		FROM users
		WHERE email = $1`

	row := r.db.QueryRowContext(ctx, query, email)
	err := row.Scan(
		&u.ID, &u.TelegramID, &u.TelegramUsername, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.LanguageCode, &u.IsActive, &u.IsPremium, &u.LimitProfileID, &settingsJSON, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(errors.ErrNotFound, "user not found")
	}
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

	return &u, nil
}

// Update updates user data
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	// Marshal settings to JSON
	settingsJSON, err := json.Marshal(u.Settings)
	if err != nil {
		return errors.Wrap(err, "failed to marshal settings")
	}

	query := `
		UPDATE users SET
			telegram_username = $2,
			email = $3,
			password_hash = $4,
			first_name = $5,
			last_name = $6,
			language_code = $7,
			is_active = $8,
			is_premium = $9,
			limit_profile_id = $10,
			settings = $11,
			updated_at = NOW()
		WHERE id = $1`

	_, err = r.db.ExecContext(ctx, query,
		u.ID, u.TelegramUsername, u.Email, u.PasswordHash, u.FirstName, u.LastName,
		u.LanguageCode, u.IsActive, u.IsPremium, u.LimitProfileID, settingsJSON,
	)

	return err
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
		SELECT id, telegram_id, telegram_username, email, password_hash, first_name, last_name,
			   language_code, is_active, is_premium, limit_profile_id, settings, created_at, updated_at
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
			&u.ID, &u.TelegramID, &u.TelegramUsername, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
			&u.LanguageCode, &u.IsActive, &u.IsPremium, &u.LimitProfileID, &settingsJSON, &u.CreatedAt, &u.UpdatedAt,
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
