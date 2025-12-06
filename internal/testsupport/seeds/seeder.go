package seeds

import (
	"context"
	"database/sql"

	"prometheus/pkg/logger"
)

// DBTX is the interface that both *sql.DB and *sql.Tx satisfy
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// Seeder is the central orchestrator for creating seed data
// It provides a fluent API to build complex test scenarios
type Seeder struct {
	db  DBTX
	ctx context.Context
	log *logger.Logger
}

// New creates a new Seeder instance
func New(db DBTX) *Seeder {
	return &Seeder{
		db:  db,
		ctx: context.Background(),
		log: logger.Get().With("component", "seeds"),
	}
}

// WithContext sets the context for database operations
func (s *Seeder) WithContext(ctx context.Context) *Seeder {
	s.ctx = ctx
	return s
}

// Log returns the logger instance
func (s *Seeder) Log() *logger.Logger {
	return s.log
}

// User starts building a User entity
func (s *Seeder) User() *UserBuilder {
	return NewUserBuilder(s.db, s.ctx, s.log)
}

// LimitProfile starts building a LimitProfile entity
func (s *Seeder) LimitProfile() *LimitProfileBuilder {
	return NewLimitProfileBuilder(s.db, s.ctx)
}

// ExchangeAccount starts building an ExchangeAccount entity
func (s *Seeder) ExchangeAccount() *ExchangeAccountBuilder {
	return NewExchangeAccountBuilder(s.db, s.ctx)
}

// Strategy starts building a Strategy entity
func (s *Seeder) Strategy() *StrategyBuilder {
	return NewStrategyBuilder(s.db, s.ctx)
}

// Position starts building a Position entity
func (s *Seeder) Position() *PositionBuilder {
	return NewPositionBuilder(s.db, s.ctx)
}

// Order starts building an Order entity
func (s *Seeder) Order() *OrderBuilder {
	return NewOrderBuilder(s.db, s.ctx)
}

// Memory starts building a Memory entity
func (s *Seeder) Memory() *MemoryBuilder {
	return NewMemoryBuilder(s.db, s.ctx)
}

// Agent starts building an Agent entity
func (s *Seeder) Agent() *AgentBuilder {
	return NewAgentBuilder(s.db, s.ctx)
}
