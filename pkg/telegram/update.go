package telegram

// Update represents an incoming Telegram update (abstraction from tgbotapi)
type Update struct {
	UpdateID int

	// Message is present if this is a regular message
	Message *Message

	// CallbackQuery is present if this is a callback from inline keyboard
	CallbackQuery *CallbackQuery
}

// Message represents a Telegram message
type Message struct {
	MessageID int
	From      *User
	Chat      *Chat
	Text      string
	IsCommand bool
	Command   string   // Parsed command (without /)
	Arguments string   // Command arguments
	ReplyTo   *Message // Message this is replying to
}

// CallbackQuery represents a callback query from inline keyboard button
type CallbackQuery struct {
	ID      string
	From    *User
	Message *Message
	Data    string // Callback data
}

// User represents a Telegram user
type User struct {
	ID        int64
	FirstName string
	LastName  string
	Username  string
	IsBot     bool
}

// Chat represents a Telegram chat
type Chat struct {
	ID       int64
	Type     string // "private", "group", "supergroup", "channel"
	Title    string
	Username string
}

// HasMessage checks if update contains a message
func (u *Update) HasMessage() bool {
	return u.Message != nil
}

// HasCallback checks if update contains a callback query
func (u *Update) HasCallback() bool {
	return u.CallbackQuery != nil
}
