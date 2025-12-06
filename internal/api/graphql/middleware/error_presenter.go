package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"

	pkgErrors "prometheus/pkg/errors"
)

// statusCodeResponseWriter wraps http.ResponseWriter to capture response body
// and check for auth errors before writing
type statusCodeResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
	written    bool
}

func (w *statusCodeResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		// Don't write yet, wait for body
	}
}

func (w *statusCodeResponseWriter) Write(b []byte) (int, error) {
	if w.written {
		return w.ResponseWriter.Write(b)
	}

	// Capture body
	w.body = append(w.body, b...)

	// Check if this is the complete response (GraphQL typically writes once)
	// Parse as JSON to detect error codes
	if len(w.body) > 0 {
		statusCode := w.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		// Check body for authentication errors
		if code := detectErrorCodeInBody(w.body); code != 0 {
			statusCode = code
		}

		// Now write status and body
		w.ResponseWriter.WriteHeader(statusCode)
		w.written = true
		return w.ResponseWriter.Write(w.body)
	}

	return len(b), nil
}

// detectErrorCodeInBody checks GraphQL response body for error extensions
// and returns appropriate HTTP status code
func detectErrorCodeInBody(body []byte) int {
	// Quick check: does body contain "errors" field?
	if !bytes.Contains(body, []byte(`"errors"`)) {
		return 0
	}

	// Parse JSON response
	var response struct {
		Errors []struct {
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return 0
	}

	// Check first error for code
	if len(response.Errors) > 0 && response.Errors[0].Extensions != nil {
		if code, ok := response.Errors[0].Extensions["code"].(string); ok {
			switch code {
			case "UNAUTHENTICATED":
				return http.StatusUnauthorized
			case "FORBIDDEN":
				return http.StatusForbidden
			case "NOT_FOUND":
				return http.StatusNotFound
			case "BAD_REQUEST":
				return http.StatusBadRequest
			case "CONFLICT":
				return http.StatusConflict
			case "INTERNAL_ERROR":
				return http.StatusInternalServerError
			case "UNAVAILABLE":
				return http.StatusServiceUnavailable
			case "TIMEOUT":
				return http.StatusGatewayTimeout
			}
		}
	}

	return 0
}

// ErrorPresenter creates a GraphQL error presenter that sets HTTP status codes
// based on error types
func ErrorPresenter() graphql.ErrorPresenterFunc {
	return func(ctx context.Context, err error) *gqlerror.Error {
		// Convert error to GraphQL error
		var gqlErr *gqlerror.Error
		if errors.As(err, &gqlErr) {
			// Already a GraphQL error
		} else {
			gqlErr = &gqlerror.Error{
				Message: err.Error(),
			}
		}

		// Determine HTTP status code based on error type
		statusCode := getHTTPStatusFromError(err)

		// Set status code in response writer if available
		if w := getResponseWriterFromContext(ctx); w != nil {
			if sw, ok := w.(*statusCodeResponseWriter); ok {
				if statusCode != http.StatusOK && !sw.written {
					sw.statusCode = statusCode
				}
			}
		}

		// Add extension with status code for debugging
		if gqlErr.Extensions == nil {
			gqlErr.Extensions = make(map[string]interface{})
		}
		gqlErr.Extensions["code"] = httpStatusToCode(statusCode)

		return gqlErr
	}
}

// getHTTPStatusFromError determines HTTP status code from error
func getHTTPStatusFromError(err error) int {
	if err == nil {
		return http.StatusOK
	}

	// Check for specific error types
	switch {
	case errors.Is(err, pkgErrors.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, pkgErrors.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, pkgErrors.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, pkgErrors.ErrAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, pkgErrors.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, pkgErrors.ErrTimeout):
		return http.StatusGatewayTimeout
	case errors.Is(err, pkgErrors.ErrUnavailable):
		return http.StatusServiceUnavailable
	}

	// Check error message for authentication keywords
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "unauthorized") ||
		strings.Contains(errMsg, "unauthenticated") ||
		strings.Contains(errMsg, "not authenticated") ||
		strings.Contains(errMsg, "user not found in context") {
		return http.StatusUnauthorized
	}

	if strings.Contains(errMsg, "forbidden") ||
		strings.Contains(errMsg, "access denied") {
		return http.StatusForbidden
	}

	// Default to 200 OK for GraphQL (errors in response body)
	// unless explicitly set above
	return http.StatusOK
}

// httpStatusToCode converts HTTP status code to string code
func httpStatusToCode(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "UNAUTHENTICATED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusInternalServerError:
		return "INTERNAL_ERROR"
	case http.StatusServiceUnavailable:
		return "UNAVAILABLE"
	case http.StatusGatewayTimeout:
		return "TIMEOUT"
	default:
		return "UNKNOWN"
	}
}

// getResponseWriterFromContext extracts http.ResponseWriter from context
func getResponseWriterFromContext(ctx context.Context) http.ResponseWriter {
	w, _ := ctx.Value(responseWriterKey).(http.ResponseWriter)
	return w
}

// HTTPStatusMiddleware wraps handler to intercept GraphQL errors and set HTTP status
func HTTPStatusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap response writer to capture status code
		sw := &statusCodeResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Replace response writer in context
		ctx := context.WithValue(r.Context(), responseWriterKey, sw)

		// Call next handler
		next.ServeHTTP(sw, r.WithContext(ctx))

		// Status code is already written by statusCodeResponseWriter
	})
}
