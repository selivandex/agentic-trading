package testsupport

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNextSequence_Increments(t *testing.T) {
	seq1 := NextSequence()
	seq2 := NextSequence()
	seq3 := NextSequence()

	assert.Greater(t, seq2, seq1, "Sequence should increment")
	assert.Greater(t, seq3, seq2, "Sequence should increment")
	assert.Equal(t, seq1+1, seq2, "Should increment by 1")
	assert.Equal(t, seq2+1, seq3, "Should increment by 1")
}

func TestUniqueName_GeneratesUnique(t *testing.T) {
	name1 := UniqueName("test_profile")
	name2 := UniqueName("test_profile")
	name3 := UniqueName("test_profile")

	assert.NotEqual(t, name1, name2, "Names should be unique")
	assert.NotEqual(t, name2, name3, "Names should be unique")
	assert.NotEqual(t, name1, name3, "Names should be unique")
	assert.Contains(t, name1, "test_profile_", "Should contain prefix")
}

func TestUniqueEmail_GeneratesValid(t *testing.T) {
	email1 := UniqueEmail("user")
	email2 := UniqueEmail("admin")

	assert.NotEqual(t, email1, email2, "Emails should be unique")
	assert.Contains(t, email1, "@test.local", "Should contain domain")
	assert.Contains(t, email1, "user_", "Should contain prefix")
	assert.Contains(t, email2, "admin_", "Should contain prefix")
}

func TestUniqueTelegramID_InValidRange(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := UniqueTelegramID()
		assert.GreaterOrEqual(t, id, int64(100000000), "Should be >= 100000000")
		assert.LessOrEqual(t, id, int64(999999999), "Should be <= 999999999")
	}
}

func TestUniqueTelegramID_GeneratesUnique(t *testing.T) {
	seen := make(map[int64]bool)

	for i := 0; i < 1000; i++ {
		id := UniqueTelegramID()
		assert.False(t, seen[id], "Telegram ID should be unique: %d", id)
		seen[id] = true
	}
}

func TestUniqueString_GeneratesUUID(t *testing.T) {
	str1 := UniqueString()
	str2 := UniqueString()

	assert.NotEqual(t, str1, str2, "Should generate unique strings")
	assert.Len(t, str1, 36, "Should be valid UUID length")
	assert.Len(t, str2, 36, "Should be valid UUID length")
}

func TestUniqueUsername_GeneratesUnique(t *testing.T) {
	user1 := UniqueUsername()
	user2 := UniqueUsername()

	assert.NotEqual(t, user1, user2, "Usernames should be unique")
	assert.Contains(t, user1, "user_", "Should contain prefix")
}

func TestUniqueSymbol_PreservesBase(t *testing.T) {
	btc1 := UniqueSymbol("BTC")
	btc2 := UniqueSymbol("BTC")
	eth1 := UniqueSymbol("ETH")

	assert.NotEqual(t, btc1, btc2, "Symbols should be unique")
	assert.Contains(t, btc1, "BTC_", "Should contain base")
	assert.Contains(t, eth1, "ETH_", "Should contain base")
}

func TestUniqueExchangeLabel_GeneratesUnique(t *testing.T) {
	label1 := UniqueExchangeLabel("binance")
	label2 := UniqueExchangeLabel("binance")
	label3 := UniqueExchangeLabel("bybit")

	assert.NotEqual(t, label1, label2, "Labels should be unique")
	assert.NotEqual(t, label2, label3, "Labels should be unique")
	assert.Contains(t, label1, "binance_", "Should contain exchange name")
	assert.Contains(t, label3, "bybit_", "Should contain exchange name")
}

func TestUniqueSessionID_GeneratesUnique(t *testing.T) {
	sid1 := UniqueSessionID()
	sid2 := UniqueSessionID()

	assert.NotEqual(t, sid1, sid2, "Session IDs should be unique")
	assert.Contains(t, sid1, "session_", "Should contain prefix")
}

func TestUniqueEventID_GeneratesUnique(t *testing.T) {
	eid1 := UniqueEventID()
	eid2 := UniqueEventID()

	assert.NotEqual(t, eid1, eid2, "Event IDs should be unique")
	assert.Contains(t, eid1, "event_", "Should contain prefix")
}

func TestConcurrentSequenceGeneration(t *testing.T) {
	const goroutines = 100
	const iterations = 100

	seen := sync.Map{}
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				seq := NextSequence()
				_, loaded := seen.LoadOrStore(seq, true)
				assert.False(t, loaded, "Sequence %d should be unique", seq)
			}
		}()
	}

	wg.Wait()
}

func TestConcurrentUniqueNames(t *testing.T) {
	const goroutines = 50
	const iterations = 50

	seen := sync.Map{}
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				name := UniqueName("test")
				_, loaded := seen.LoadOrStore(name, true)
				assert.False(t, loaded, "Name %s should be unique", name)
			}
		}()
	}

	wg.Wait()
}


