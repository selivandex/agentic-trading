package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"prometheus/internal/domain/menu_session"
	"prometheus/pkg/errors"
)

// MenuSessionRepository implements menu_session.Repository using Redis
type MenuSessionRepository struct {
	client *redis.Client
}

// NewMenuSessionRepository creates a new menu session repository
func NewMenuSessionRepository(client *redis.Client) *MenuSessionRepository {
	return &MenuSessionRepository{
		client: client,
	}
}

// Get retrieves a session by telegram ID
func (r *MenuSessionRepository) Get(ctx context.Context, telegramID int64) (*menu_session.Session, error) {
	key := r.getKey(telegramID)

	data, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "menu session not found for telegram_id=%d", telegramID)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get menu session from redis: telegram_id=%d", telegramID)
	}

	var session menu_session.Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal menu session: telegram_id=%d", telegramID)
	}

	return &session, nil
}

// Save stores a session with TTL
func (r *MenuSessionRepository) Save(ctx context.Context, session *menu_session.Session, ttl time.Duration) error {
	key := r.getKey(session.TelegramID)

	data, err := json.Marshal(session)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal menu session: telegram_id=%d", session.TelegramID)
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return errors.Wrapf(err, "failed to save menu session to redis: telegram_id=%d", session.TelegramID)
	}

	return nil
}

// Delete removes a session
func (r *MenuSessionRepository) Delete(ctx context.Context, telegramID int64) error {
	key := r.getKey(telegramID)

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return errors.Wrapf(err, "failed to delete menu session from redis: telegram_id=%d", telegramID)
	}

	return nil
}

// Exists checks if session exists
func (r *MenuSessionRepository) Exists(ctx context.Context, telegramID int64) (bool, error) {
	key := r.getKey(telegramID)

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, errors.Wrapf(err, "failed to check menu session existence: telegram_id=%d", telegramID)
	}

	return exists > 0, nil
}

func (r *MenuSessionRepository) getKey(telegramID int64) string {
	return fmt.Sprintf("menu_nav:%d", telegramID)
}
