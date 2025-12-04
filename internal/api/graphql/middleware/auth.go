package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"prometheus/internal/domain/user"
	"prometheus/internal/services/auth"
	"prometheus/pkg/logger"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// AuthCookieName is the name of the JWT cookie
	AuthCookieName = "auth_token"
	// userContextKey is the context key for authenticated user
	userContextKey contextKey = "authenticated_user"
	// responseWriterKey is the context key for http.ResponseWriter
	responseWriterKey contextKey = "http_response_writer"
)

// TokenValidator defines interface for validating JWT tokens
// This allows mocking in tests
type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (*user.User, error)
}

// Ensure auth.Service implements TokenValidator
var _ TokenValidator = (*auth.Service)(nil)

// AuthMiddleware validates JWT token from HTTP-only cookie
type AuthMiddleware struct {
	authService TokenValidator
	log         *logger.Logger
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authService TokenValidator, log *logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		log:         log.With("middleware", "auth"),
	}
}

// Handler wraps HTTP handler with JWT authentication
// Token is extracted from HTTP-only cookie
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get auth token from cookie
		cookie, err := r.Cookie(AuthCookieName)
		if err != nil {
			// No auth cookie - continue without user in context
			// This allows public queries to work
			next.ServeHTTP(w, r)
			return
		}

		// Validate token and get user
		usr, err := m.authService.ValidateToken(r.Context(), cookie.Value)
		if err != nil {
			// Invalid token - log and continue without user
			m.log.Debugw("Invalid auth token",
				"error", err,
				"remote_addr", r.RemoteAddr,
			)
			// Clear invalid cookie
			clearCookie(w)
			next.ServeHTTP(w, r)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), userContextKey, usr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFromContext extracts authenticated user from context
func UserFromContext(ctx context.Context) *user.User {
	usr, ok := ctx.Value(userContextKey).(*user.User)
	if !ok {
		return nil
	}
	return usr
}

// SetAuthCookie sets HTTP-only auth cookie with JWT token
// This should be called from GraphQL context
func SetAuthCookie(ctx context.Context, token string) error {
	// Get ResponseWriter from context (set by ResponseWriterMiddleware)
	w, ok := ctx.Value(responseWriterKey).(http.ResponseWriter)
	if !ok {
		return fmt.Errorf("http.ResponseWriter not found in context")
	}

	// Create HTTP-only cookie with 1 year expiration
	cookie := &http.Cookie{
		Name:     AuthCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Only over HTTPS in production
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int((time.Hour * 24 * 365).Seconds()), // 1 year
	}

	http.SetCookie(w, cookie)
	return nil
}

// ClearAuthCookie clears the auth cookie
func ClearAuthCookie(ctx context.Context) error {
	w, ok := ctx.Value(responseWriterKey).(http.ResponseWriter)
	if !ok {
		return fmt.Errorf("http.ResponseWriter not found in context")
	}

	clearCookie(w)
	return nil
}

// clearCookie helper to clear auth cookie
func clearCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1, // Delete cookie
	}
	http.SetCookie(w, cookie)
}

// ResponseWriterMiddleware adds http.ResponseWriter to context for cookie setting
// This must be applied before AuthMiddleware
func ResponseWriterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), responseWriterKey, w)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
