package menu_session

import (
	"time"
)

// Session represents a menu navigation session
type Session struct {
	TelegramID      int64                  `json:"telegram_id"`
	MessageID       int                    `json:"message_id"`       // Current bot message ID (for editing)
	CurrentScreen   string                 `json:"current_screen"`   // Current screen ID
	NavigationStack []string               `json:"navigation_stack"` // Stack of screen IDs for back button
	Data            map[string]interface{} `json:"data"`             // Screen-specific data
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// NewSession creates a new menu session
func NewSession(telegramID int64, initialScreen string, initialData map[string]interface{}) *Session {
	if initialData == nil {
		initialData = make(map[string]interface{})
	}

	return &Session{
		TelegramID:      telegramID,
		CurrentScreen:   initialScreen,
		NavigationStack: []string{},
		Data:            initialData,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// PushScreen adds current screen to stack and sets new screen
func (s *Session) PushScreen(screenID string) {
	s.NavigationStack = append(s.NavigationStack, s.CurrentScreen)
	s.CurrentScreen = screenID
	s.UpdatedAt = time.Now()
}

// PopScreen returns to previous screen
func (s *Session) PopScreen() (previousScreen string, ok bool) {
	if len(s.NavigationStack) == 0 {
		return "", false
	}

	previousScreen = s.NavigationStack[len(s.NavigationStack)-1]
	s.NavigationStack = s.NavigationStack[:len(s.NavigationStack)-1]
	s.CurrentScreen = previousScreen
	s.UpdatedAt = time.Now()

	return previousScreen, true
}

// SetData sets a value in session data
func (s *Session) SetData(key string, value interface{}) {
	if s.Data == nil {
		s.Data = make(map[string]interface{})
	}
	s.Data[key] = value
	s.UpdatedAt = time.Now()
}

// GetData gets a value from session data
func (s *Session) GetData(key string) (interface{}, bool) {
	if s.Data == nil {
		return nil, false
	}
	val, ok := s.Data[key]
	return val, ok
}

// GetString gets a string value from session data
func (s *Session) GetString(key string) (string, bool) {
	val, ok := s.GetData(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// HasNavigationHistory checks if back button should be shown
func (s *Session) HasNavigationHistory() bool {
	return len(s.NavigationStack) > 0
}
