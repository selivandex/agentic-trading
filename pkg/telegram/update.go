package telegram

// Update represents an incoming Telegram update (abstraction from tgbotapi)
type Update struct {
	UpdateID int `json:"update_id"`

	// Message is present if this is a regular message
	Message *Message `json:"message,omitempty"`

	// CallbackQuery is present if this is a callback from inline keyboard
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// Message represents a Telegram message
type Message struct {
	MessageID int      `json:"message_id"`
	From      *User    `json:"from,omitempty"`
	Chat      *Chat    `json:"chat"`
	Text      string   `json:"text,omitempty"`
	IsCommand bool     `json:"-"` // Computed field, not from JSON
	Command   string   `json:"-"` // Parsed command (without /)
	Arguments string   `json:"-"` // Command arguments
	ReplyTo   *Message `json:"reply_to_message,omitempty"`
}

// CallbackQuery represents a callback query from inline keyboard button
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data"` // Callback data
}

// User represents a Telegram user
type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	IsBot     bool   `json:"is_bot,omitempty"`
}

// Chat represents a Telegram chat
type Chat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"` // "private", "group", "supergroup", "channel"
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
}

// HasMessage checks if update contains a message
func (u *Update) HasMessage() bool {
	return u.Message != nil
}

// HasCallback checks if update contains a callback query
func (u *Update) HasCallback() bool {
	return u.CallbackQuery != nil
}

// ParseCommand parses command from message text
// Call this after JSON unmarshaling to populate IsCommand, Command, Arguments
func (m *Message) ParseCommand() {
	if m == nil || m.Text == "" {
		return
	}

	// Check if message starts with /
	if m.Text[0] != '/' {
		m.IsCommand = false
		return
	}

	m.IsCommand = true

	// Parse command and arguments
	// Format: /command args or /command@botname args
	text := m.Text[1:] // Remove leading /

	// Split by whitespace
	parts := splitWhitespace(text)
	if len(parts) == 0 {
		return
	}

	// First part is command (may contain @botname)
	commandPart := parts[0]
	
	// Remove @botname if present
	if atIndex := indexOf(commandPart, '@'); atIndex != -1 {
		commandPart = commandPart[:atIndex]
	}

	m.Command = commandPart

	// Rest is arguments
	if len(parts) > 1 {
		m.Arguments = joinFrom(parts, 1, " ")
	}
}

// Helper functions for string manipulation

func splitWhitespace(s string) []string {
	var result []string
	var current string
	
	for _, ch := range s {
		if ch == ' ' || ch == '\t' || ch == '\n' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	
	if current != "" {
		result = append(result, current)
	}
	
	return result
}

func indexOf(s string, ch rune) int {
	for i, c := range s {
		if c == ch {
			return i
		}
	}
	return -1
}

func joinFrom(parts []string, start int, sep string) string {
	if start >= len(parts) {
		return ""
	}
	
	result := parts[start]
	for i := start + 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	
	return result
}
