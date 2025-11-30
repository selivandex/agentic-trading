package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"prometheus/internal/domain/limit_profile"
	pkgerrors "prometheus/pkg/errors"
)

// Compile-time check
var _ limit_profile.Repository = (*LimitProfileRepository)(nil)

// LimitProfileRepository implements limit_profile.Repository using sqlx
type LimitProfileRepository struct {
	db *sqlx.DB
}

// NewLimitProfileRepository creates a new limit profile repository
func NewLimitProfileRepository(db *sqlx.DB) *LimitProfileRepository {
	return &LimitProfileRepository{db: db}
}

// Create inserts a new limit profile
func (r *LimitProfileRepository) Create(ctx context.Context, profile *limit_profile.LimitProfile) error {
	query := `
		INSERT INTO limit_profiles (
			id, name, description, limits, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`

	_, err := r.db.ExecContext(ctx, query,
		profile.ID, profile.Name, profile.Description,
		profile.Limits, profile.IsActive,
		profile.CreatedAt, profile.UpdatedAt,
	)

	if err != nil {
		return pkgerrors.Wrap(err, "failed to create limit profile")
	}

	return nil
}

// GetByID retrieves a limit profile by ID
func (r *LimitProfileRepository) GetByID(ctx context.Context, id uuid.UUID) (*limit_profile.LimitProfile, error) {
	query := `
		SELECT id, name, description, limits, is_active, created_at, updated_at
		FROM limit_profiles
		WHERE id = $1`

	var profile limit_profile.LimitProfile
	err := r.db.GetContext(ctx, &profile, query, id)
	if err == sql.ErrNoRows {
		return nil, pkgerrors.Wrap(err, "limit profile not found")
	}
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get limit profile")
	}

	return &profile, nil
}

// GetByName retrieves a limit profile by name
func (r *LimitProfileRepository) GetByName(ctx context.Context, name string) (*limit_profile.LimitProfile, error) {
	query := `
		SELECT id, name, description, limits, is_active, created_at, updated_at
		FROM limit_profiles
		WHERE name = $1 AND is_active = true`

	var profile limit_profile.LimitProfile
	err := r.db.GetContext(ctx, &profile, query, name)
	if err == sql.ErrNoRows {
		return nil, pkgerrors.Wrap(err, "limit profile not found")
	}
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get limit profile by name")
	}

	return &profile, nil
}

// GetAll retrieves all limit profiles
func (r *LimitProfileRepository) GetAll(ctx context.Context) ([]*limit_profile.LimitProfile, error) {
	query := `
		SELECT id, name, description, limits, is_active, created_at, updated_at
		FROM limit_profiles
		ORDER BY created_at DESC`

	var profiles []*limit_profile.LimitProfile
	err := r.db.SelectContext(ctx, &profiles, query)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get all limit profiles")
	}

	return profiles, nil
}

// GetAllActive retrieves all active limit profiles
func (r *LimitProfileRepository) GetAllActive(ctx context.Context) ([]*limit_profile.LimitProfile, error) {
	query := `
		SELECT id, name, description, limits, is_active, created_at, updated_at
		FROM limit_profiles
		WHERE is_active = true
		ORDER BY created_at DESC`

	var profiles []*limit_profile.LimitProfile
	err := r.db.SelectContext(ctx, &profiles, query)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get active limit profiles")
	}

	return profiles, nil
}

// Update updates an existing limit profile
func (r *LimitProfileRepository) Update(ctx context.Context, profile *limit_profile.LimitProfile) error {
	query := `
		UPDATE limit_profiles
		SET name = $2,
			description = $3,
			limits = $4,
			is_active = $5,
			updated_at = NOW()
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		profile.ID, profile.Name, profile.Description,
		profile.Limits, profile.IsActive,
	)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to update limit profile")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.Wrap(err, "failed to get rows affected")
	}
	if rows == 0 {
		return pkgerrors.Wrap(sql.ErrNoRows, "limit profile not found")
	}

	return nil
}

// Delete soft-deletes a limit profile by setting is_active to false
func (r *LimitProfileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE limit_profiles
		SET is_active = false, updated_at = NOW()
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to delete limit profile")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.Wrap(err, "failed to get rows affected")
	}
	if rows == 0 {
		return pkgerrors.Wrap(sql.ErrNoRows, "limit profile not found")
	}

	return nil
}
