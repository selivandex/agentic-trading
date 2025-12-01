package telegram

// InlineKeyboardMarkup represents an inline keyboard (abstraction from tgbotapi)
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton
}

// InlineKeyboardButton represents a button in inline keyboard
type InlineKeyboardButton struct {
	Text         string
	CallbackData string
	URL          string
}

// NewInlineKeyboardMarkup creates a new inline keyboard markup
func NewInlineKeyboardMarkup(rows ...[]InlineKeyboardButton) InlineKeyboardMarkup {
	return InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

// NewInlineKeyboardRow creates a row of inline keyboard buttons
func NewInlineKeyboardRow(buttons ...InlineKeyboardButton) []InlineKeyboardButton {
	return buttons
}

// NewInlineKeyboardButtonData creates a button with callback data
func NewInlineKeyboardButtonData(text, callbackData string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         text,
		CallbackData: callbackData,
	}
}

// NewInlineKeyboardButtonURL creates a button with URL
func NewInlineKeyboardButtonURL(text, url string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: text,
		URL:  url,
	}
}
