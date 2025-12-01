package telegram

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// InMemorySession implements Session interface with in-memory storage
type InMemorySession struct {
	telegramID      int64
	messageID       int
	currentScreen   string
	data            map[string]interface{}
	callbackData    map[string]map[string]interface{} // callback key -> params
	navigationStack []string
	createdAt       time.Time
	updatedAt       time.Time
	mu              sync.RWMutex
}

// NewInMemorySession creates a new in-memory session
func NewInMemorySession(telegramID int64, initialScreen string, initialData map[string]interface{}) *InMemorySession {
	now := time.Now()

	data := make(map[string]interface{})
	for k, v := range initialData {
		data[k] = v
	}

	return &InMemorySession{
		telegramID:      telegramID,
		currentScreen:   initialScreen,
		data:            data,
		callbackData:    make(map[string]map[string]interface{}),
		navigationStack: make([]string, 0),
		createdAt:       now,
		updatedAt:       now,
	}
}

// GetTelegramID returns telegram user ID
func (s *InMemorySession) GetTelegramID() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.telegramID
}

// GetMessageID returns current message ID
func (s *InMemorySession) GetMessageID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.messageID
}

// SetMessageID sets current message ID
func (s *InMemorySession) SetMessageID(messageID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messageID = messageID
	s.updatedAt = time.Now()
}

// GetCurrentScreen returns current screen ID
func (s *InMemorySession) GetCurrentScreen() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentScreen
}

// SetCurrentScreen sets current screen ID
func (s *InMemorySession) SetCurrentScreen(screenID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentScreen = screenID
	s.updatedAt = time.Now()
}

// GetData retrieves session data by key
func (s *InMemorySession) GetData(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// SetData stores session data
func (s *InMemorySession) SetData(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	s.updatedAt = time.Now()
}

// GetString retrieves string value from session
func (s *InMemorySession) GetString(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.data[key]
	if !ok {
		return "", false
	}

	str, ok := val.(string)
	return str, ok
}

// PushScreen adds screen to navigation stack
func (s *InMemorySession) PushScreen(screenID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.navigationStack = append(s.navigationStack, screenID)
	s.updatedAt = time.Now()
}

// PopScreen removes and returns last screen from stack
func (s *InMemorySession) PopScreen() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.navigationStack) == 0 {
		return "", false
	}

	lastIndex := len(s.navigationStack) - 1
	screenID := s.navigationStack[lastIndex]
	s.navigationStack = s.navigationStack[:lastIndex]
	s.updatedAt = time.Now()

	return screenID, true
}

// HasNavigationHistory checks if there are screens in navigation stack
func (s *InMemorySession) HasNavigationHistory() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.navigationStack) > 0
}

// GetCreatedAt returns session creation time
func (s *InMemorySession) GetCreatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.createdAt
}

// GetUpdatedAt returns last update time
func (s *InMemorySession) GetUpdatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.updatedAt
}

// SetCallbackData stores callback parameters with short key (for Telegram 64-byte limit)
func (s *InMemorySession) SetCallbackData(key string, data map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbackData[key] = data
	s.updatedAt = time.Now()
}

// GetCallbackData retrieves callback parameters by short key
func (s *InMemorySession) GetCallbackData(key string) (map[string]interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.callbackData[key]
	return data, ok
}

// ClearCallbackData removes all callback data (called when navigating to new screen)
func (s *InMemorySession) ClearCallbackData() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbackData = make(map[string]map[string]interface{})
	s.updatedAt = time.Now()
}

// InMemorySessionService implements SessionService with in-memory storage
type InMemorySessionService struct {
	sessions map[int64]*InMemorySession
	mu       sync.RWMutex
}

// NewInMemorySessionService creates a new in-memory session service
func NewInMemorySessionService() *InMemorySessionService {
	return &InMemorySessionService{
		sessions: make(map[int64]*InMemorySession),
	}
}

// GetSession retrieves session for telegram user
func (s *InMemorySessionService) GetSession(ctx context.Context, telegramID int64) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[telegramID]
	if !exists {
		return nil, fmt.Errorf("session not found for telegram_id: %d", telegramID)
	}

	return session, nil
}

// SaveSession saves session with TTL (TTL ignored in memory implementation)
func (s *InMemorySessionService) SaveSession(ctx context.Context, session Session, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cast to InMemorySession
	memSession, ok := session.(*InMemorySession)
	if !ok {
		return fmt.Errorf("invalid session type: expected *InMemorySession")
	}

	s.sessions[session.GetTelegramID()] = memSession
	return nil
}

// DeleteSession removes session
func (s *InMemorySessionService) DeleteSession(ctx context.Context, telegramID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, telegramID)
	return nil
}

// SessionExists checks if session exists
func (s *InMemorySessionService) SessionExists(ctx context.Context, telegramID int64) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.sessions[telegramID]
	return exists, nil
}

// CreateSession creates new session
func (s *InMemorySessionService) CreateSession(ctx context.Context, telegramID int64, initialScreen string, initialData map[string]interface{}, ttl time.Duration) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := NewInMemorySession(telegramID, initialScreen, initialData)
	s.sessions[telegramID] = session

	return session, nil
}

// Cleanup removes expired sessions (call periodically)
func (s *InMemorySessionService) Cleanup(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for telegramID, session := range s.sessions {
		if session.GetUpdatedAt().Before(cutoff) {
			delete(s.sessions, telegramID)
			removed++
		}
	}

	return removed
}

// Count returns number of active sessions
func (s *InMemorySessionService) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// Clear removes all sessions (for testing)
func (s *InMemorySessionService) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = make(map[int64]*InMemorySession)
}
