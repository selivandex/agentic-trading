package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"prometheus/internal/domain/session"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// SessionRepository implements session.Repository using PostgreSQL
type SessionRepository struct {
	db  *sqlx.DB
	log *logger.Logger
}

// NewSessionRepository creates a new PostgreSQL session repository
func NewSessionRepository(db *sqlx.DB) *SessionRepository {
	return &SessionRepository{
		db:  db,
		log: logger.Get().With("component", "session_repository"),
	}
}

// Create creates a new session
func (r *SessionRepository) Create(ctx context.Context, sess *session.Session) error {
	stateJSON, err := json.Marshal(sess.State)
	if err != nil {
		return errors.Wrap(err, "failed to marshal state")
	}

	query := `
		INSERT INTO adk_sessions (id, app_name, user_id, session_id, state, updated_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.db.ExecContext(ctx, query,
		sess.ID,
		sess.AppName,
		sess.UserID,
		sess.SessionID,
		stateJSON,
		sess.UpdatedAt,
		sess.CreatedAt,
	)

	if err != nil {
		return errors.Wrap(err, "failed to create session")
	}

	return nil
}

// Get retrieves a session with optional event filtering
func (r *SessionRepository) Get(ctx context.Context, appName, userID, sessionID string, opts *session.GetOptions) (*session.Session, error) {
	if opts == nil {
		opts = &session.GetOptions{}
	}

	// Get session
	query := `
		SELECT id, app_name, user_id, session_id, state, updated_at, created_at
		FROM adk_sessions
		WHERE app_name = $1 AND user_id = $2 AND session_id = $3
	`

	var sess session.Session
	var stateJSON []byte

	err := r.db.QueryRowContext(ctx, query, appName, userID, sessionID).Scan(
		&sess.ID,
		&sess.AppName,
		&sess.UserID,
		&sess.SessionID,
		&stateJSON,
		&sess.UpdatedAt,
		&sess.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.Wrap(errors.ErrNotFound, "session not found")
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get session")
	}

	if err := json.Unmarshal(stateJSON, &sess.State); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal state")
	}

	// Get events with filtering
	events, err := r.GetEvents(ctx, sess.ID, &session.GetEventsOptions{
		Limit: opts.NumRecentEvents,
		After: opts.After,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get events")
	}

	sess.Events = make([]session.Event, len(events))
	for i, e := range events {
		sess.Events[i] = *e
	}

	return &sess, nil
}

// List lists all sessions for an app/user
func (r *SessionRepository) List(ctx context.Context, appName, userID string) ([]*session.Session, error) {
	query := `
		SELECT id, app_name, user_id, session_id, state, updated_at, created_at
		FROM adk_sessions
		WHERE app_name = $1
	`
	args := []interface{}{appName}

	if userID != "" {
		query += ` AND user_id = $2`
		args = append(args, userID)
	}

	query += ` ORDER BY updated_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list sessions")
	}
	defer rows.Close()

	var sessions []*session.Session
	for rows.Next() {
		var sess session.Session
		var stateJSON []byte

		err := rows.Scan(
			&sess.ID,
			&sess.AppName,
			&sess.UserID,
			&sess.SessionID,
			&stateJSON,
			&sess.UpdatedAt,
			&sess.CreatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan session")
		}

		if err := json.Unmarshal(stateJSON, &sess.State); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal state")
		}

		sess.Events = []session.Event{} // Don't load events for list
		sessions = append(sessions, &sess)
	}

	return sessions, nil
}

// Delete deletes a session
func (r *SessionRepository) Delete(ctx context.Context, appName, userID, sessionID string) error {
	query := `
		DELETE FROM adk_sessions
		WHERE app_name = $1 AND user_id = $2 AND session_id = $3
	`

	result, err := r.db.ExecContext(ctx, query, appName, userID, sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to delete session")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return errors.Wrap(errors.ErrNotFound, "session not found")
	}

	return nil
}

// UpdateState updates session state
func (r *SessionRepository) UpdateState(ctx context.Context, appName, userID, sessionID string, state map[string]interface{}) error {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return errors.Wrap(err, "failed to marshal state")
	}

	query := `
		UPDATE adk_sessions
		SET state = $1, updated_at = $2
		WHERE app_name = $3 AND user_id = $4 AND session_id = $5
	`

	_, err = r.db.ExecContext(ctx, query, stateJSON, time.Now(), appName, userID, sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to update state")
	}

	return nil
}

// AppendEvent appends an event to a session
func (r *SessionRepository) AppendEvent(ctx context.Context, sessionUUID uuid.UUID, event *session.Event) error {
	contentJSON, err := json.Marshal(event.Content)
	if err != nil {
		return errors.Wrap(err, "failed to marshal content")
	}

	actionsJSON, err := json.Marshal(event.Actions)
	if err != nil {
		return errors.Wrap(err, "failed to marshal actions")
	}

	var usageMetadataJSON []byte
	if event.UsageMetadata != nil {
		usageMetadataJSON, err = json.Marshal(event.UsageMetadata)
		if err != nil {
			return errors.Wrap(err, "failed to marshal usage metadata")
		}
	}

	query := `
		INSERT INTO adk_events (
			id, session_uuid, event_id, author, content, timestamp, branch,
			partial, turn_complete, actions, usage_metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}

	_, err = r.db.ExecContext(ctx, query,
		event.ID,
		sessionUUID,
		event.EventID,
		event.Author,
		contentJSON,
		event.Timestamp,
		event.Branch,
		event.Partial,
		event.TurnComplete,
		actionsJSON,
		usageMetadataJSON,
	)

	if err != nil {
		return errors.Wrap(err, "failed to append event")
	}

	return nil
}

// GetEvents retrieves events for a session
func (r *SessionRepository) GetEvents(ctx context.Context, sessionUUID uuid.UUID, opts *session.GetEventsOptions) ([]*session.Event, error) {
	if opts == nil {
		opts = &session.GetEventsOptions{}
	}

	query := `
		SELECT id, session_uuid, event_id, author, content, timestamp, branch,
		       partial, turn_complete, actions, usage_metadata
		FROM adk_events
		WHERE session_uuid = $1
	`
	args := []interface{}{sessionUUID}

	if !opts.After.IsZero() {
		query += ` AND timestamp >= $2`
		args = append(args, opts.After)
	}

	query += ` ORDER BY timestamp DESC`

	if opts.Limit > 0 {
		query += ` LIMIT $` + string(rune(len(args)+1))
		args = append(args, opts.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get events")
	}
	defer rows.Close()

	var events []*session.Event
	for rows.Next() {
		var event session.Event
		var contentJSON, actionsJSON, usageMetadataJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.SessionID,
			&event.EventID,
			&event.Author,
			&contentJSON,
			&event.Timestamp,
			&event.Branch,
			&event.Partial,
			&event.TurnComplete,
			&actionsJSON,
			&usageMetadataJSON,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan event")
		}

		if err := json.Unmarshal(contentJSON, &event.Content); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal content")
		}

		if err := json.Unmarshal(actionsJSON, &event.Actions); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal actions")
		}

		if len(usageMetadataJSON) > 0 {
			var metadata session.UsageMetadata
			if err := json.Unmarshal(usageMetadataJSON, &metadata); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal usage metadata")
			}
			event.UsageMetadata = &metadata
		}

		events = append(events, &event)
	}

	// Reverse to get chronological order (we fetched DESC for LIMIT)
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}

	return events, nil
}

// GetAppState retrieves application-level state
func (r *SessionRepository) GetAppState(ctx context.Context, appName string) (*session.AppState, error) {
	query := `SELECT app_name, state FROM adk_app_state WHERE app_name = $1`

	var appState session.AppState
	var stateJSON []byte

	err := r.db.QueryRowContext(ctx, query, appName).Scan(&appState.AppName, &stateJSON)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(errors.ErrNotFound, "app state not found")
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app state")
	}

	if err := json.Unmarshal(stateJSON, &appState.State); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal state")
	}

	return &appState, nil
}

// SetAppState sets application-level state
func (r *SessionRepository) SetAppState(ctx context.Context, appName string, state map[string]interface{}) error {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return errors.Wrap(err, "failed to marshal state")
	}

	query := `
		INSERT INTO adk_app_state (app_name, state)
		VALUES ($1, $2)
		ON CONFLICT (app_name) DO UPDATE SET state = $2
	`

	_, err = r.db.ExecContext(ctx, query, appName, stateJSON)
	if err != nil {
		return errors.Wrap(err, "failed to set app state")
	}

	return nil
}

// GetUserState retrieves user-level state
func (r *SessionRepository) GetUserState(ctx context.Context, appName, userID string) (*session.UserState, error) {
	query := `SELECT app_name, user_id, state FROM adk_user_state WHERE app_name = $1 AND user_id = $2`

	var userState session.UserState
	var stateJSON []byte

	err := r.db.QueryRowContext(ctx, query, appName, userID).Scan(&userState.AppName, &userState.UserID, &stateJSON)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(errors.ErrNotFound, "user state not found")
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user state")
	}

	if err := json.Unmarshal(stateJSON, &userState.State); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal state")
	}

	return &userState, nil
}

// SetUserState sets user-level state
func (r *SessionRepository) SetUserState(ctx context.Context, appName, userID string, state map[string]interface{}) error {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return errors.Wrap(err, "failed to marshal state")
	}

	query := `
		INSERT INTO adk_user_state (app_name, user_id, state)
		VALUES ($1, $2, $3)
		ON CONFLICT (app_name, user_id) DO UPDATE SET state = $3
	`

	_, err = r.db.ExecContext(ctx, query, appName, userID, stateJSON)
	if err != nil {
		return errors.Wrap(err, "failed to set user state")
	}

	return nil
}
