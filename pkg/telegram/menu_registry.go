package telegram

import (
	"context"
	"fmt"

	"prometheus/pkg/logger"
)

// MenuHandler defines interface for menu-based handlers
type MenuHandler interface {
	// GetMenuType returns menu type identifier (invest, exchanges, etc.)
	GetMenuType() string

	// GetScreenIDs returns all screen IDs this handler owns (for routing)
	GetScreenIDs() []string

	// HandleCallback processes callback for this menu
	HandleCallback(ctx context.Context, userID interface{}, telegramID int64, messageID int, data string) error

	// HandleMessage processes text message for this menu (messageID for deleting sensitive messages)
	HandleMessage(ctx context.Context, userID interface{}, telegramID int64, messageID int, text string) error

	// EndMenu ends the menu session (for cancel button)
	EndMenu(ctx context.Context, telegramID int64) error
}

// MenuRegistry manages menu handlers and routes callbacks automatically
type MenuRegistry struct {
	handlers       map[string]MenuHandler // screenID -> handler mapping (for legacy lookup)
	menuHandlers   map[string]MenuHandler // menuType -> handler mapping (invest, exchanges, etc.)
	sessionService SessionService         // Session service for getting current menu
	log            *logger.Logger
}

// NewMenuRegistry creates a new menu registry
func NewMenuRegistry(log *logger.Logger, sessionService SessionService) *MenuRegistry {
	return &MenuRegistry{
		handlers:       make(map[string]MenuHandler),
		menuHandlers:   make(map[string]MenuHandler),
		sessionService: sessionService,
		log:            log.With("component", "menu_registry"),
	}
}

// Register registers a menu handler by menu type
func (mr *MenuRegistry) Register(handler MenuHandler) {
	menuType := handler.GetMenuType()
	screenIDs := handler.GetScreenIDs()

	mr.log.Infow("Registering menu handler",
		"menu_type", menuType,
		"handler_type", fmt.Sprintf("%T", handler),
		"screen_count", len(screenIDs),
	)

	// Register by menu type (primary)
	if existing, exists := mr.menuHandlers[menuType]; exists {
		mr.log.Warnw("Menu type already registered, overwriting",
			"menu_type", menuType,
			"old_handler", fmt.Sprintf("%T", existing),
			"new_handler", fmt.Sprintf("%T", handler),
		)
	}
	mr.menuHandlers[menuType] = handler

	// Also map screen IDs for debugging
	for _, screenID := range screenIDs {
		mr.handlers[screenID] = handler
	}
}

// RouteCallback routes callback to appropriate handler based on session menu_type
func (mr *MenuRegistry) RouteCallback(ctx context.Context, userID interface{}, telegramID int64, messageID int, callbackData string) error {
	// Get current session to determine which menu is active
	session, err := mr.sessionService.GetSession(ctx, telegramID)
	if err != nil {
		mr.log.Debugw("No active menu session for callback",
			"telegram_id", telegramID,
			"callback_data", callbackData,
		)
		return fmt.Errorf("no active menu session")
	}

	menuType := session.GetMenuType()

	// Special handling for "cancel" button - end current menu
	if callbackData == "cancel" {
		return mr.handleCancel(ctx, telegramID, menuType)
	}

	// Find handler by menu type
	handler, exists := mr.menuHandlers[menuType]
	if !exists {
		mr.log.Errorw("No handler registered for menu type",
			"menu_type", menuType,
			"telegram_id", telegramID,
		)
		return fmt.Errorf("no handler for menu type: %s", menuType)
	}

	mr.log.Debugw("Routing callback to handler",
		"telegram_id", telegramID,
		"menu_type", menuType,
		"handler_type", fmt.Sprintf("%T", handler),
		"callback_data", callbackData,
	)

	return handler.HandleCallback(ctx, userID, telegramID, messageID, callbackData)
}

// handleCancel handles cancel button - ends current menu session
func (mr *MenuRegistry) handleCancel(ctx context.Context, telegramID int64, menuType string) error {
	mr.log.Debugw("Handling cancel button",
		"telegram_id", telegramID,
		"menu_type", menuType,
	)

	// Find handler by menu type
	handler, exists := mr.menuHandlers[menuType]
	if !exists {
		mr.log.Debugw("Cancel button pressed but no handler for menu type",
			"menu_type", menuType,
			"telegram_id", telegramID,
		)
		return nil // Not an error - just nothing to cancel
	}

	mr.log.Infow("Ending menu session via cancel button",
		"telegram_id", telegramID,
		"menu_type", menuType,
		"handler_type", fmt.Sprintf("%T", handler),
	)

	return handler.EndMenu(ctx, telegramID)
}

// RouteTextMessage routes text message to active menu
func (mr *MenuRegistry) RouteTextMessage(ctx context.Context, userID interface{}, telegramID int64, messageID int, text string) (bool, error) {
	// Get current session to determine which menu is active
	session, err := mr.sessionService.GetSession(ctx, telegramID)
	if err != nil {
		// No active session = no menu to route to
		return false, nil
	}

	menuType := session.GetMenuType()

	// Find handler by menu type
	handler, exists := mr.menuHandlers[menuType]
	if !exists {
		mr.log.Warnw("No handler registered for menu type",
			"menu_type", menuType,
			"telegram_id", telegramID,
		)
		return false, nil
	}

	mr.log.Debugw("Routing text message to menu handler",
		"telegram_id", telegramID,
		"menu_type", menuType,
		"handler_type", fmt.Sprintf("%T", handler),
	)

	if err := handler.HandleMessage(ctx, userID, telegramID, messageID, text); err != nil {
		return true, err
	}

	return true, nil
}

// HasActiveMenu checks if user has any active menu session
func (mr *MenuRegistry) HasActiveMenu(ctx context.Context, telegramID int64) (bool, error) {
	_, err := mr.sessionService.GetSession(ctx, telegramID)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// GetHandlers returns all registered handlers (for debugging)
func (mr *MenuRegistry) GetHandlers() []MenuHandler {
	handlers := make([]MenuHandler, 0, len(mr.menuHandlers))
	for _, handler := range mr.menuHandlers {
		handlers = append(handlers, handler)
	}
	return handlers
}

// GetHandlerForScreen returns handler for specific screen ID
func (mr *MenuRegistry) GetHandlerForScreen(screenID string) (MenuHandler, bool) {
	handler, exists := mr.handlers[screenID]
	return handler, exists
}
