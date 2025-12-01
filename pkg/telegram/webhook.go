package telegram

import (
	"encoding/json"
	"net/http"

	"prometheus/pkg/logger"
)

// WebhookHandler handles incoming Telegram webhook requests (framework-level)
// This is a reusable HTTP handler that works through Bot interface
type WebhookHandler struct {
	updateHandler func(Update) // User-provided update handler
	log           *logger.Logger
}

// NewWebhookHandler creates a new webhook handler
// The updateHandler will be called for each incoming update
func NewWebhookHandler(updateHandler func(Update), log *logger.Logger) *WebhookHandler {
	return &WebhookHandler{
		updateHandler: updateHandler,
		log:           log.With("component", "telegram_webhook"),
	}
}

// ServeHTTP implements http.Handler interface
func (wh *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		wh.log.Warnw("Invalid webhook request method", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse update from request body
	var update Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		wh.log.Errorw("Failed to decode webhook update", "error", err)
		wh.sendErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Post-process: parse commands from message text
	if update.Message != nil {
		update.Message.ParseCommand()
	}

	wh.log.Debugw("Received webhook update",
		"update_id", update.UpdateID,
		"has_message", update.HasMessage(),
		"has_callback", update.HasCallback(),
	)

	// Call user's update handler (non-blocking)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				wh.log.Errorw("Panic in update handler",
					"panic", r,
					"update_id", update.UpdateID,
				)
			}
		}()

		wh.updateHandler(update)
	}()

	// Always return 200 OK to acknowledge receipt
	// This prevents Telegram from retrying
	wh.sendSuccessResponse(w)
}

// HealthCheck returns webhook health status
func (wh *WebhookHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "telegram_webhook",
	})
}

// sendSuccessResponse sends successful webhook response
func (wh *WebhookHandler) sendSuccessResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true,
	})
}

// sendErrorResponse sends error webhook response
func (wh *WebhookHandler) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":          false,
		"error":       message,
		"description": message,
	})
}
