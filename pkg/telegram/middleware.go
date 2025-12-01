package telegram

import (
	"fmt"
	"time"

	"prometheus/pkg/logger"
)

// LoggingMiddleware logs command execution with timing
func LoggingMiddleware(log *logger.Logger) CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return func(ctx *CommandContext) error {
			start := time.Now()

			log.Infow("Executing command",
				"command", ctx.Command,
				"telegram_id", ctx.TelegramID,
				"has_args", ctx.Args != "",
			)

			err := next(ctx)
			duration := time.Since(start)

			if err != nil {
				log.Errorw("Command failed",
					"command", ctx.Command,
					"telegram_id", ctx.TelegramID,
					"duration_ms", duration.Milliseconds(),
					"error", err,
				)
			} else {
				log.Debugw("Command completed",
					"command", ctx.Command,
					"telegram_id", ctx.TelegramID,
					"duration_ms", duration.Milliseconds(),
				)
			}

			return err
		}
	}
}

// RecoveryMiddleware recovers from panics in command handlers
func RecoveryMiddleware(log *logger.Logger) CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return func(ctx *CommandContext) (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Errorw("Command handler panicked",
						"command", ctx.Command,
						"telegram_id", ctx.TelegramID,
						"panic", r,
					)
					err = fmt.Errorf("internal error")
					// Send user-friendly error
					ctx.Bot.SendMessage(ctx.ChatID, "❌ An unexpected error occurred. Our team has been notified.")
				}
			}()

			return next(ctx)
		}
	}
}

// RateLimitMiddleware prevents command spam (basic in-memory implementation)
func RateLimitMiddleware(maxPerMinute int, log *logger.Logger) CommandMiddleware {
	// Simple in-memory rate limiter (for production use Redis)
	// telegram_id -> []timestamp
	requests := make(map[int64][]time.Time)

	return func(next CommandHandler) CommandHandler {
		return func(ctx *CommandContext) error {
			now := time.Now()
			userID := ctx.TelegramID

			// Clean up old timestamps (older than 1 minute)
			if timestamps, exists := requests[userID]; exists {
				var valid []time.Time
				cutoff := now.Add(-1 * time.Minute)
				for _, ts := range timestamps {
					if ts.After(cutoff) {
						valid = append(valid, ts)
					}
				}
				requests[userID] = valid
			}

			// Check rate limit
			if len(requests[userID]) >= maxPerMinute {
				log.Warnw("Rate limit exceeded",
					"telegram_id", userID,
					"command", ctx.Command,
					"requests_count", len(requests[userID]),
				)
				return ctx.Bot.SendMessage(ctx.ChatID, "⏱️ Slow down! Please wait a moment before trying again.")
			}

			// Add current request
			requests[userID] = append(requests[userID], now)

			return next(ctx)
		}
	}
}

// AuthRequiredMiddleware ensures user is authenticated
func AuthRequiredMiddleware() CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return func(ctx *CommandContext) error {
			if ctx.User == nil {
				return ctx.Bot.SendMessage(ctx.ChatID, "❌ You must be registered to use this command. Use /start first.")
			}
			return next(ctx)
		}
	}
}

// AdminOnlyMiddleware checks if user has admin privileges
// Requires User interface with IsAdmin() bool method
func AdminOnlyMiddleware(checkAdmin func(user interface{}) bool) CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return func(ctx *CommandContext) error {
			if ctx.User == nil || !checkAdmin(ctx.User) {
				return ctx.Bot.SendMessage(ctx.ChatID, "❌ This command requires administrator privileges.")
			}
			return next(ctx)
		}
	}
}

// MaintenanceModeMiddleware blocks all commands during maintenance
func MaintenanceModeMiddleware(isInMaintenance func() bool, allowedUsers []int64) CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return func(ctx *CommandContext) error {
			if !isInMaintenance() {
				return next(ctx)
			}

			// Check if user is in allowed list
			for _, allowed := range allowedUsers {
				if ctx.TelegramID == allowed {
					return next(ctx)
				}
			}

			return ctx.Bot.SendMessage(ctx.ChatID, "🔧 System is currently under maintenance. Please try again later.")
		}
	}
}

// TypingIndicatorMiddleware shows "typing..." while command executes
func TypingIndicatorMiddleware() CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return func(ctx *CommandContext) error {
			// Send typing action via Bot API
			// Implementation should be done in Bot adapter
			// For now, just pass through
			return next(ctx)
		}
	}
}

// MetricsMiddleware tracks command usage metrics
func MetricsMiddleware(recordMetric func(command string, success bool, duration time.Duration)) CommandMiddleware {
	return func(next CommandHandler) CommandHandler {
		return func(ctx *CommandContext) error {
			start := time.Now()
			err := next(ctx)
			duration := time.Since(start)

			recordMetric(ctx.Command, err == nil, duration)

			return err
		}
	}
}
