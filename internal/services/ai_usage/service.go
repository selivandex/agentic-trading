package ai_usage

import (
	"context"
	"time"

	"prometheus/internal/domain/ai_usage"
	chrepo "prometheus/internal/repository/clickhouse"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service handles AI usage tracking business logic
// Provides abstraction over ClickHouse batch writer
type Service struct {
	repository *chrepo.AIUsageRepository
	log        *logger.Logger
}

// NewService creates a new AI usage service
func NewService(
	repository *chrepo.AIUsageRepository,
	log *logger.Logger,
) *Service {
	return &Service{
		repository: repository,
		log:        log,
	}
}

// Start starts the background batch writer
func (s *Service) Start(ctx context.Context) {
	s.log.Info("Starting AI usage service batch writer...")
	s.repository.Start(ctx)
}

// Stop stops the batch writer gracefully
func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("Stopping AI usage service batch writer...")
	if err := s.repository.Stop(ctx); err != nil {
		return errors.Wrap(err, "failed to stop batch writer")
	}
	s.log.Info("✓ AI usage service stopped")
	return nil
}

// Store adds an AI usage log to the batch
// Buffered operation - will flush when batch is full or timeout
func (s *Service) Store(ctx context.Context, log *ai_usage.UsageLog) error {
	if err := s.repository.Store(ctx, log); err != nil {
		return errors.Wrap(err, "failed to store AI usage log")
	}

	s.log.Debugw("AI usage log buffered",
		"agent", log.AgentName,
		"provider", log.Provider,
		"model", log.ModelID,
		"tokens", log.TotalTokens,
		"cost_usd", log.TotalCostUSD,
	)

	return nil
}

// GetUsageByUser retrieves usage statistics for a specific user
// This can be used for billing or analytics
func (s *Service) GetUsageByUser(ctx context.Context, userID string, from, to time.Time) ([]*ai_usage.UsageLog, error) {
	// TODO: Implement query method in repository first
	return nil, errors.New("not implemented")
}
