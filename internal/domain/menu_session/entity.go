package menu_session

import (
	"time"
)

// Session represents a menu navigation session
type Session struct {
	TelegramID      int64                             `json:"telegram_id"`
	MessageID       int                               `json:"message_id"`       // Current bot message ID (for editing)
	CurrentScreen   string                            `json:"current_screen"`   // Current screen ID
	NavigationStack []string                          `json:"navigation_stack"` // Stack of screen IDs for back button
	Data            map[string]interface{}            `json:"data"`             // Screen-specific data
	CallbackData    map[string]map[string]interface{} `json:"callback_data"`    // Callback key -> params (for Telegram 64-byte limit)
	CreatedAt       time.Time                         `json:"created_at"`
	UpdatedAt       time.Time                         `json:"updated_at"`
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
		CallbackData:    make(map[string]map[string]interface{}),
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

// GetTelegramID returns telegram user ID (implements telegram.Session interface)
func (s *Session) GetTelegramID() int64 {
	return s.TelegramID
}

// GetMessageID returns current message ID (implements telegram.Session interface)
func (s *Session) GetMessageID() int {
	return s.MessageID
}

// SetMessageID sets current message ID (implements telegram.Session interface)
func (s *Session) SetMessageID(messageID int) {
	s.MessageID = messageID
	s.UpdatedAt = time.Now()
}

// GetCurrentScreen returns current screen ID (implements telegram.Session interface)
func (s *Session) GetCurrentScreen() string {
	return s.CurrentScreen
}

// SetCurrentScreen sets current screen ID (implements telegram.Session interface)
func (s *Session) SetCurrentScreen(screenID string) {
	s.CurrentScreen = screenID
	s.UpdatedAt = time.Now()
}

// GetCreatedAt returns session creation time (implements telegram.Session interface)
func (s *Session) GetCreatedAt() time.Time {
	return s.CreatedAt
}

// GetUpdatedAt returns last update time (implements telegram.Session interface)
func (s *Session) GetUpdatedAt() time.Time {
	return s.UpdatedAt
}

// SetCallbackData stores callback parameters with short key (for Telegram 64-byte limit)
func (s *Session) SetCallbackData(key string, data map[string]interface{}) {
	if s.CallbackData == nil {
		s.CallbackData = make(map[string]map[string]interface{})
	}
	s.CallbackData[key] = data
	s.UpdatedAt = time.Now()
}

// GetCallbackData retrieves callback parameters by short key
func (s *Session) GetCallbackData(key string) (map[string]interface{}, bool) {
	if s.CallbackData == nil {
		return nil, false
	}
	data, ok := s.CallbackData[key]
	return data, ok
}

// ClearCallbackData removes all callback data (called when navigating to new screen)
func (s *Session) ClearCallbackData() {
	s.CallbackData = make(map[string]map[string]interface{})
	s.UpdatedAt = time.Now()
}
