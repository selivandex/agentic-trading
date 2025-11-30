package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"prometheus/pkg/logger"
)

// MenuHandler defines interface for menu-based handlers
type MenuHandler interface {
	// GetScreenIDs returns all screen IDs this handler owns (for routing)
	GetScreenIDs() []string

	// HandleCallback processes callback for this menu
	HandleCallback(ctx context.Context, userID uuid.UUID, telegramID int64, messageID int, data string) error

	// HandleMessage processes text message for this menu (optional)
	HandleMessage(ctx context.Context, userID uuid.UUID, telegramID int64, text string) error

	// IsInMenu checks if user has active session in this menu
	IsInMenu(ctx context.Context, telegramID int64) (bool, error)
}

// MenuRegistry manages menu handlers and routes callbacks automatically
type MenuRegistry struct {
	handlers  map[string]MenuHandler // screenID -> handler mapping
	textMenus []MenuHandler          // Handlers that accept text input
	log       *logger.Logger
}

// NewMenuRegistry creates a new menu registry
func NewMenuRegistry(log *logger.Logger) *MenuRegistry {
	return &MenuRegistry{
		handlers:  make(map[string]MenuHandler),
		textMenus: make([]MenuHandler, 0),
		log:       log.With("component", "menu_registry"),
	}
}

// Register registers a menu handler
func (mr *MenuRegistry) Register(handler MenuHandler) {
	screenIDs := handler.GetScreenIDs()

	mr.log.Debugw("Registering menu handler",
		"screen_ids", screenIDs,
		"handler_type", fmt.Sprintf("%T", handler),
	)

	for _, screenID := range screenIDs {
		if existing, exists := mr.handlers[screenID]; exists {
			mr.log.Warnw("Overwriting existing handler for screen",
				"screen_id", screenID,
				"old_handler", fmt.Sprintf("%T", existing),
				"new_handler", fmt.Sprintf("%T", handler),
			)
		}
		mr.handlers[screenID] = handler
	}

	// Add to text menus list (for message routing)
	mr.textMenus = append(mr.textMenus, handler)
}

// RouteCallback routes callback to appropriate handler based on screen ID
func (mr *MenuRegistry) RouteCallback(ctx context.Context, userID uuid.UUID, telegramID int64, messageID int, callbackData string) error {
	// Special handling for "back" button - find active menu
	if callbackData == "back" {
		return mr.routeBackButton(ctx, userID, telegramID, messageID)
	}

	// Extract screen ID from callback (first part before :)
	screenID := callbackData
	if idx := strings.Index(callbackData, ":"); idx != -1 {
		screenID = callbackData[:idx]
	}

	mr.log.Debugw("Routing callback",
		"telegram_id", telegramID,
		"screen_id", screenID,
		"callback_data", callbackData,
	)

	handler, exists := mr.handlers[screenID]
	if !exists {
		mr.log.Warnw("No handler found for screen",
			"screen_id", screenID,
			"callback_data", callbackData,
		)
		return fmt.Errorf("no handler for screen: %s", screenID)
	}

	mr.log.Debugw("Routing to handler",
		"screen_id", screenID,
		"handler_type", fmt.Sprintf("%T", handler),
	)

	return handler.HandleCallback(ctx, userID, telegramID, messageID, callbackData)
}

// routeBackButton routes "back" button to active menu
func (mr *MenuRegistry) routeBackButton(ctx context.Context, userID uuid.UUID, telegramID int64, messageID int) error {
	mr.log.Debugw("Routing back button",
		"telegram_id", telegramID,
	)

	// Find which menu is active
	for _, handler := range mr.textMenus {
		inMenu, err := handler.IsInMenu(ctx, telegramID)
		if err != nil {
			mr.log.Errorw("Failed to check menu status for back routing",
				"handler_type", fmt.Sprintf("%T", handler),
				"error", err,
			)
			continue
		}

		if inMenu {
			mr.log.Debugw("Routing back button to active menu",
				"telegram_id", telegramID,
				"handler_type", fmt.Sprintf("%T", handler),
			)
			return handler.HandleCallback(ctx, userID, telegramID, messageID, "back")
		}
	}

	mr.log.Warnw("Back button pressed but no active menu found",
		"telegram_id", telegramID,
	)

	return fmt.Errorf("no active menu found for back button")
}

// RouteTextMessage routes text message to active menu
func (mr *MenuRegistry) RouteTextMessage(ctx context.Context, userID uuid.UUID, telegramID int64, text string) (bool, error) {
	// Check each menu to see if user has active session
	for _, handler := range mr.textMenus {
		inMenu, err := handler.IsInMenu(ctx, telegramID)
		if err != nil {
			mr.log.Errorw("Failed to check menu status",
				"handler_type", fmt.Sprintf("%T", handler),
				"error", err,
			)
			continue
		}

		if inMenu {
			mr.log.Debugw("Routing text message to menu handler",
				"telegram_id", telegramID,
				"handler_type", fmt.Sprintf("%T", handler),
			)

			if err := handler.HandleMessage(ctx, userID, telegramID, text); err != nil {
				return true, err
			}
			return true, nil
		}
	}

	return false, nil // No menu found
}

// HasActiveMenu checks if user has any active menu session
func (mr *MenuRegistry) HasActiveMenu(ctx context.Context, telegramID int64) (bool, error) {
	for _, handler := range mr.textMenus {
		inMenu, err := handler.IsInMenu(ctx, telegramID)
		if err != nil {
			mr.log.Errorw("Failed to check menu status",
				"handler_type", fmt.Sprintf("%T", handler),
				"error", err,
			)
			continue
		}

		if inMenu {
			return true, nil
		}
	}

	return false, nil
}
