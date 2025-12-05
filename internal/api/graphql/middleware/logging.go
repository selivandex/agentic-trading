package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"prometheus/pkg/logger"

	"github.com/99designs/gqlgen/graphql"
)

// LoggingMiddleware logs GraphQL operations
type LoggingMiddleware struct {
	log *logger.Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(log *logger.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		log: log.With("component", "graphql"),
	}
}

// HTTPLoggingMiddleware logs HTTP requests to GraphQL endpoint
func (m *LoggingMiddleware) HTTPLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request
		m.log.Debugw("GraphQL HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"content_type", r.Header.Get("Content-Type"),
		)

		// Wrap response writer to capture status code and body
		wrapped := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Execute handler
		next.ServeHTTP(wrapped, r)

		// Log response with body (truncated for large responses)
		duration := time.Since(start)
		bodyPreview := string(wrapped.body)
		if len(bodyPreview) > 500 {
			bodyPreview = bodyPreview[:500] + "... (truncated)"
		}

		m.log.Infow("GraphQL HTTP response",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
			"body_length", len(wrapped.body),
			"body", bodyPreview,
		)
	})
}

// loggingResponseWriter wraps http.ResponseWriter to capture status code and body
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (rw *loggingResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *loggingResponseWriter) Write(b []byte) (int, error) {
	// Capture response body
	rw.body = append(rw.body, b...)
	return rw.ResponseWriter.Write(b)
}

// OperationMiddleware logs GraphQL operation execution
// This middleware is called for each GraphQL operation
func (m *LoggingMiddleware) OperationMiddleware() graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		oc := graphql.GetOperationContext(ctx)

		// Get authenticated user if present
		var userID string
		if usr := UserFromContext(ctx); usr != nil {
			userID = usr.ID.String()
		}

		// Start timer
		start := time.Now()

		// Log operation start
		fields := []interface{}{
			"operation_name", oc.OperationName,
			"operation_type", oc.Operation.Operation,
		}

		if userID != "" {
			fields = append(fields, "user_id", userID)
		}

		// Log variables (without sensitive data)
		if len(oc.Variables) > 0 {
			safeVars := sanitizeVariables(oc.Variables)
			if varsJSON, err := json.Marshal(safeVars); err == nil {
				fields = append(fields, "variables", string(varsJSON))
			}
		}

		m.log.Infow("GraphQL operation started", fields...)

		// Execute operation
		responseHandler := next(ctx)

		return func(ctx context.Context) *graphql.Response {
			// Get response
			resp := responseHandler(ctx)

			// Calculate duration
			duration := time.Since(start)

			// Log operation completion
			logFields := []interface{}{
				"operation_name", oc.OperationName,
				"operation_type", oc.Operation.Operation,
				"duration_ms", duration.Milliseconds(),
			}

			if userID != "" {
				logFields = append(logFields, "user_id", userID)
			}

			// Check for errors
			if resp != nil && len(resp.Errors) > 0 {
				errMsgs := make([]string, len(resp.Errors))
				errPaths := make([]interface{}, len(resp.Errors))
				errExtensions := make([]interface{}, len(resp.Errors))
				for i, err := range resp.Errors {
					errMsgs[i] = err.Message
					errPaths[i] = err.Path
					errExtensions[i] = err.Extensions
				}
				logFields = append(logFields, "errors", errMsgs, "error_paths", errPaths, "error_extensions", errExtensions)
				m.log.Errorw("GraphQL operation failed", logFields...)
			} else if resp != nil && resp.Data == nil {
				// Data is null but no errors - this is suspicious
				m.log.Warnw("GraphQL operation returned null data without errors", logFields...)
			} else {
				m.log.Infow("GraphQL operation completed", logFields...)
			}

			return resp
		}
	}
}

// sanitizeVariables removes sensitive data from variables before logging
func sanitizeVariables(vars map[string]interface{}) map[string]interface{} {
	// List of sensitive field names to redact
	sensitiveFields := map[string]bool{
		"password":       true,
		"token":          true,
		"secret":         true,
		"apiKey":         true,
		"apiSecret":      true,
		"accessToken":    true,
		"refreshToken":   true,
		"privateKey":     true,
		"encryptedKey":   true,
		"encryptedValue": true,
	}

	sanitized := make(map[string]interface{})
	for key, value := range vars {
		// Check if field is sensitive
		if sensitiveFields[key] {
			sanitized[key] = "***REDACTED***"
			continue
		}

		// Recursively sanitize nested objects
		switch v := value.(type) {
		case map[string]interface{}:
			sanitized[key] = sanitizeVariables(v)
		case []interface{}:
			sanitized[key] = sanitizeArray(v, sensitiveFields)
		default:
			sanitized[key] = value
		}
	}

	return sanitized
}

// sanitizeArray sanitizes array elements
func sanitizeArray(arr []interface{}, sensitiveFields map[string]bool) []interface{} {
	sanitized := make([]interface{}, len(arr))
	for i, item := range arr {
		switch v := item.(type) {
		case map[string]interface{}:
			sanitized[i] = sanitizeVariables(v)
		case []interface{}:
			sanitized[i] = sanitizeArray(v, sensitiveFields)
		default:
			sanitized[i] = item
		}
	}
	return sanitized
}
