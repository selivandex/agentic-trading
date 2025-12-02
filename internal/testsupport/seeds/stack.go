package seeds

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/strategy"
	"prometheus/internal/domain/user"
)

// StackBuilder provides a convenient way to create a full test stack
// (User -> ExchangeAccount -> Strategy) in one go
type StackBuilder struct {
	seeder *Seeder

	// Created entities
	user            *user.User
	exchangeAccount *exchange_account.ExchangeAccount
	strategy        *strategy.Strategy

	// Customization functions
	userCustomizer     func(*UserBuilder) *UserBuilder
	exchangeCustomizer func(*ExchangeAccountBuilder) *ExchangeAccountBuilder
	strategyCustomizer func(*StrategyBuilder) *StrategyBuilder
}

// NewStackBuilder creates a new StackBuilder
func NewStackBuilder(seeder *Seeder) *StackBuilder {
	return &StackBuilder{
		seeder: seeder,
	}
}

// CustomizeUser allows customizing the user before creation
func (sb *StackBuilder) CustomizeUser(fn func(*UserBuilder) *UserBuilder) *StackBuilder {
	sb.userCustomizer = fn
	return sb
}

// CustomizeExchangeAccount allows customizing the exchange account before creation
func (sb *StackBuilder) CustomizeExchangeAccount(fn func(*ExchangeAccountBuilder) *ExchangeAccountBuilder) *StackBuilder {
	sb.exchangeCustomizer = fn
	return sb
}

// CustomizeStrategy allows customizing the strategy before creation
func (sb *StackBuilder) CustomizeStrategy(fn func(*StrategyBuilder) *StrategyBuilder) *StackBuilder {
	sb.strategyCustomizer = fn
	return sb
}

// Build creates all entities and returns them
func (sb *StackBuilder) Build() (*user.User, *exchange_account.ExchangeAccount, *strategy.Strategy, error) {
	// 1. Create User
	userBuilder := sb.seeder.User()
	if sb.userCustomizer != nil {
		userBuilder = sb.userCustomizer(userBuilder)
	}
	u, err := userBuilder.Insert()
	if err != nil {
		return nil, nil, nil, err
	}
	sb.user = u

	// 2. Create ExchangeAccount
	exchangeBuilder := sb.seeder.ExchangeAccount().WithUserID(u.ID)
	if sb.exchangeCustomizer != nil {
		exchangeBuilder = sb.exchangeCustomizer(exchangeBuilder)
	}
	acc, err := exchangeBuilder.Insert()
	if err != nil {
		return nil, nil, nil, err
	}
	sb.exchangeAccount = acc

	// 3. Create Strategy
	strategyBuilder := sb.seeder.Strategy().WithUserID(u.ID)
	if sb.strategyCustomizer != nil {
		strategyBuilder = sb.strategyCustomizer(strategyBuilder)
	}
	strat, err := strategyBuilder.Insert()
	if err != nil {
		return nil, nil, nil, err
	}
	sb.strategy = strat

	return u, acc, strat, nil
}

// MustBuild creates all entities and panics on error
func (sb *StackBuilder) MustBuild() (*user.User, *exchange_account.ExchangeAccount, *strategy.Strategy) {
	u, acc, strat, err := sb.Build()
	if err != nil {
		panic(err)
	}
	return u, acc, strat
}

// User returns the created user
func (sb *StackBuilder) User() *user.User {
	return sb.user
}

// ExchangeAccount returns the created exchange account
func (sb *StackBuilder) ExchangeAccount() *exchange_account.ExchangeAccount {
	return sb.exchangeAccount
}

// Strategy returns the created strategy
func (sb *StackBuilder) Strategy() *strategy.Strategy {
	return sb.strategy
}

// Stack creates a new StackBuilder from a Seeder
func (s *Seeder) Stack() *StackBuilder {
	return NewStackBuilder(s)
}

// QuickStack creates a basic user + exchange + strategy stack with defaults
// This is a convenience function for most common test setup
func (s *Seeder) QuickStack() (userID, exchangeAccountID, strategyID uuid.UUID, err error) {
	u, acc, strat, err := s.Stack().Build()
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	return u.ID, acc.ID, strat.ID, nil
}

// MustQuickStack creates a basic stack and panics on error
func (s *Seeder) MustQuickStack() (userID, exchangeAccountID, strategyID uuid.UUID) {
	userID, exchangeAccountID, strategyID, err := s.QuickStack()
	if err != nil {
		panic(err)
	}
	return
}

// QuickStackWithCapital creates a stack with custom capital allocation
func (s *Seeder) QuickStackWithCapital(capital decimal.Decimal) (userID, exchangeAccountID, strategyID uuid.UUID, err error) {
	u, acc, strat, err := s.Stack().
		CustomizeStrategy(func(sb *StrategyBuilder) *StrategyBuilder {
			return sb.WithCapital(capital)
		}).
		Build()
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	return u.ID, acc.ID, strat.ID, nil
}
