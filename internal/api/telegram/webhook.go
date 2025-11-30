package telegram

import (
	"encoding/json"
	"net/http"

	telegram "prometheus/internal/adapters/telegram"
	"prometheus/pkg/logger"
)

// WebhookHandler handles Telegram webhook requests
type WebhookHandler struct {
	bot     *telegram.Bot
	handler *telegram.Handler
	log     *logger.Logger
}

// NewWebhookHandler creates a new Telegram webhook handler
func NewWebhookHandler(bot *telegram.Bot, handler *telegram.Handler, log *logger.Logger) *WebhookHandler {
	return &WebhookHandler{
		bot:     bot,
		handler: handler,
		log:     log.With("component", "telegram_webhook"),
	}
}

// ServeHTTP handles incoming webhook requests from Telegram
func (wh *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Use library's HandleUpdate to parse the request
	// This handles method validation and JSON parsing
	update, err := wh.bot.GetAPI().HandleUpdate(r)
	if err != nil {
		wh.log.Errorw("Failed to handle webhook update", "error", err)

		// Return error response as JSON (library expects this format)
		errResp := map[string]string{"error": err.Error()}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(errResp)
		return
	}

	wh.log.Debugw("Received webhook update",
		"update_id", update.UpdateID,
		"has_message", update.Message != nil,
		"has_callback", update.CallbackQuery != nil,
	)

	// Route update to handler
	if err := wh.handler.Route(r.Context(), *update); err != nil {
		wh.log.Errorw("Failed to route update", "error", err)
		// Don't return error to Telegram - we processed it even if there was an error
		// This prevents Telegram from retrying the same update
	}

	// Always return 200 OK to acknowledge receipt
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true,
	})
}

// HealthCheck returns webhook health status
func (wh *WebhookHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "telegram_webhook",
	})
}
