package telegram

import (
	"prometheus/pkg/logger"
	"prometheus/pkg/telegram"
)

// NewWebhookHandler creates a Telegram webhook handler using pkg/telegram framework
// This is now just a thin wrapper around the framework's WebhookHandler
func NewWebhookHandler(updateHandler func(telegram.Update), log *logger.Logger) *telegram.WebhookHandler {
	return telegram.NewWebhookHandler(updateHandler, log)
}
