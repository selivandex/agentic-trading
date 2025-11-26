package errors

import (
	"errors"
	"fmt"
)

// Domain error types for business logic

var (
	// ErrNotFound indicates a resource was not found
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists indicates a resource already exists
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput indicates invalid input parameters
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized indicates insufficient permissions
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates action is forbidden
	ErrForbidden = errors.New("forbidden")

	// ErrInternal indicates an internal server error
	ErrInternal = errors.New("internal error")

	// ErrTimeout indicates an operation timeout
	ErrTimeout = errors.New("operation timeout")

	// ErrUnavailable indicates a service is unavailable
	ErrUnavailable = errors.New("service unavailable")
)

// Risk-specific errors

var (
	// ErrTradingBlocked indicates trading is blocked by risk engine
	ErrTradingBlocked = errors.New("trading blocked by risk engine")

	// ErrCircuitBreakerTripped indicates circuit breaker is active
	ErrCircuitBreakerTripped = errors.New("circuit breaker tripped")

	// ErrDrawdownExceeded indicates daily drawdown limit exceeded
	ErrDrawdownExceeded = errors.New("daily drawdown limit exceeded")

	// ErrConsecutiveLosses indicates too many consecutive losses
	ErrConsecutiveLosses = errors.New("consecutive losses limit exceeded")

	// ErrMaxExposure indicates maximum position exposure reached
	ErrMaxExposure = errors.New("maximum exposure limit reached")

	// ErrKillSwitchActive indicates kill switch is active
	ErrKillSwitchActive = errors.New("kill switch is active")
)

// Exchange-specific errors

var (
	// ErrExchangeUnavailable indicates exchange API is unavailable
	ErrExchangeUnavailable = errors.New("exchange unavailable")

	// ErrInsufficientBalance indicates insufficient account balance
	ErrInsufficientBalance = errors.New("insufficient balance")

	// ErrInvalidSymbol indicates invalid trading symbol
	ErrInvalidSymbol = errors.New("invalid trading symbol")

	// ErrOrderRejected indicates order was rejected by exchange
	ErrOrderRejected = errors.New("order rejected by exchange")

	// ErrPositionNotFound indicates position not found
	ErrPositionNotFound = errors.New("position not found")

	// ErrRateLimitExceeded indicates API rate limit exceeded
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// AI cost-related errors

var (
	// ErrQuotaExceeded indicates cost quota limit exceeded
	ErrQuotaExceeded = errors.New("cost quota exceeded")

	// ErrDailyLimitExceeded indicates daily AI spending limit exceeded
	ErrDailyLimitExceeded = errors.New("daily AI cost limit exceeded")

	// ErrExecutionLimitExceeded indicates single execution cost limit exceeded
	ErrExecutionLimitExceeded = errors.New("execution cost limit exceeded")
)

// WebSocket-specific errors

var (
	// ErrWSNotConnected indicates WebSocket is not connected
	ErrWSNotConnected = errors.New("websocket not connected")

	// ErrWSSubscriptionFailed indicates WebSocket subscription failed
	ErrWSSubscriptionFailed = errors.New("websocket subscription failed")

	// ErrWSReconnectFailed indicates WebSocket reconnection failed
	ErrWSReconnectFailed = errors.New("websocket reconnection failed")

	// ErrWSMaxReconnectAttempts indicates max reconnection attempts reached
	ErrWSMaxReconnectAttempts = errors.New("max websocket reconnection attempts reached")
)

// DomainError wraps an error with additional context
type DomainError struct {
	Code    string
	Message string
	Err     error
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error
func (e *DomainError) Unwrap() error {
	return e.Err
}

// NewDomainError creates a new domain error
func NewDomainError(code, message string, err error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// ValidationError represents a validation error with field-specific details
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string, value interface{}) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	}
}

// MultiError wraps multiple errors
type MultiError struct {
	Errors []error
}

// Error implements the error interface
func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return "no errors"
	}
	if len(m.Errors) == 1 {
		return m.Errors[0].Error()
	}
	return fmt.Sprintf("multiple errors (%d): %v", len(m.Errors), m.Errors[0])
}

// Add adds an error to the list
func (m *MultiError) Add(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

// HasErrors returns true if there are any errors
func (m *MultiError) HasErrors() bool {
	return len(m.Errors) > 0
}

// ToError returns the MultiError as an error, or nil if no errors
func (m *MultiError) ToError() error {
	if !m.HasErrors() {
		return nil
	}
	return m
}

// Helper functions

// Is checks if err is or wraps target
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target type
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Wrap wraps an error with context
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with formatted context
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

func New(message string) error {
	return errors.New(message)
}

func Newf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}
