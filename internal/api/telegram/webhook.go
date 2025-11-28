package telegram

import (
	"encoding/json"
	"io"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	telegram "prometheus/internal/adapters/telegram"
	"prometheus/pkg/logger"
)

// WebhookHandler handles Telegram webhook requests
type WebhookHandler struct {
	handler *telegram.Handler
	log     *logger.Logger
}

// NewWebhookHandler creates a new Telegram webhook handler
func NewWebhookHandler(handler *telegram.Handler, log *logger.Logger) *WebhookHandler {
	return &WebhookHandler{
		handler: handler,
		log:     log.With("component", "telegram_webhook"),
	}
}

// ServeHTTP handles incoming webhook requests from Telegram
func (wh *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		wh.log.Error("Failed to read webhook body", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse update
	var update tgbotapi.Update
	if err := json.Unmarshal(body, &update); err != nil {
		wh.log.Error("Failed to unmarshal update", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	wh.log.Debug("Received webhook update",
		"update_id", update.UpdateID,
		"has_message", update.Message != nil,
		"has_callback", update.CallbackQuery != nil,
	)

	// Route update to handler
	if err := wh.handler.Route(r.Context(), update); err != nil {
		wh.log.Error("Failed to handle update", "error", err)
		// Don't return error to Telegram - we processed it even if there was an error
		// This prevents Telegram from retrying the same update
	}

	// Always return 200 OK to acknowledge receipt
	w.WriteHeader(http.StatusOK)
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
