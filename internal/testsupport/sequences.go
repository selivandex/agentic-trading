package testsupport

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

var (
	// Global counter for generating unique sequential IDs in tests
	testSequence uint64

	// Base timestamp to make names shorter
	baseTimestamp = time.Now().UnixNano()
)

func init() {
	// Initialize with current timestamp to ensure uniqueness across test runs
	testSequence = uint64(baseTimestamp % 1000000)
}

// NextSequence returns next unique sequence number
func NextSequence() uint64 {
	return atomic.AddUint64(&testSequence, 1)
}

// UniqueName generates a unique name with given prefix
// Example: UniqueName("test_profile") -> "test_profile_123456"
func UniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, NextSequence())
}

// UniqueEmail generates a unique email address
// Example: UniqueEmail("user") -> "user_123456@test.local"
func UniqueEmail(prefix string) string {
	return fmt.Sprintf("%s_%d@test.local", prefix, NextSequence())
}

// UniqueTelegramID generates a unique telegram ID for testing
// Returns ID in range [100000000, 999999999]
func UniqueTelegramID() int64 {
	seq := NextSequence()
	// Keep in valid telegram ID range
	return 100000000 + int64(seq%900000000)
}

// UniqueString generates a unique string identifier
// Useful when you need guaranteed uniqueness (uses UUID)
func UniqueString() string {
	return uuid.New().String()
}

// UniqueUsername generates a unique username
// Example: UniqueUsername() -> "user_123456"
func UniqueUsername() string {
	return fmt.Sprintf("user_%d", NextSequence())
}

// UniqueSymbol generates a unique trading symbol for tests
// Example: UniqueSymbol("BTC") -> "BTC_123456"
func UniqueSymbol(base string) string {
	return fmt.Sprintf("%s_%d", base, NextSequence())
}

// UniqueExchangeLabel generates a unique exchange label
// Example: UniqueExchangeLabel("binance") -> "binance_123456"
func UniqueExchangeLabel(exchange string) string {
	return fmt.Sprintf("%s_%d", exchange, NextSequence())
}

// UniqueSessionID generates a unique session ID string
func UniqueSessionID() string {
	return fmt.Sprintf("session_%d", NextSequence())
}

// UniqueEventID generates a unique event ID
func UniqueEventID() string {
	return fmt.Sprintf("event_%d_%s", NextSequence(), uuid.New().String()[:8])
}
