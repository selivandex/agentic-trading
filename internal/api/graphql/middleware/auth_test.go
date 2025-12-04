package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"prometheus/internal/domain/user"
	"prometheus/pkg/auth"
	"prometheus/pkg/logger"
)

// MockAuthService is a mock for auth.Service
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) ValidateToken(ctx context.Context, token string) (*user.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func testLogger() *logger.Logger {
	zapLogger, _ := zap.NewDevelopment()
	return &logger.Logger{SugaredLogger: zapLogger.Sugar()}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	mockAuthSvc := new(MockAuthService)
	log := testLogger()
	middleware := NewAuthMiddleware(mockAuthSvc, log)

	userID := uuid.New()
	expectedUser := &user.User{
		ID:    userID,
		Email: stringPtr("test@example.com"),
	}

	token := "valid-jwt-token"
	mockAuthSvc.On("ValidateToken", mock.Anything, token).Return(expectedUser, nil)

	// Create test request with auth cookie
	req := httptest.NewRequest("GET", "/graphql", nil)
	req.AddCookie(&http.Cookie{
		Name:  AuthCookieName,
		Value: token,
	})

	// Response recorder
	recorder := httptest.NewRecorder()

	// Test handler that checks if user is in context
	var capturedUser *user.User
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// Apply middleware
	middleware.Handler(handler).ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotNil(t, capturedUser)
	assert.Equal(t, userID, capturedUser.ID)
	mockAuthSvc.AssertExpectations(t)
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	mockAuthSvc := new(MockAuthService)
	log := testLogger()
	middleware := NewAuthMiddleware(mockAuthSvc, log)

	// Create test request WITHOUT auth cookie
	req := httptest.NewRequest("GET", "/graphql", nil)
	recorder := httptest.NewRecorder()

	// Test handler
	var capturedUser *user.User
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// Apply middleware
	middleware.Handler(handler).ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Nil(t, capturedUser) // No user in context
	mockAuthSvc.AssertNotCalled(t, "ValidateToken")
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	mockAuthSvc := new(MockAuthService)
	log := testLogger()
	middleware := NewAuthMiddleware(mockAuthSvc, log)

	token := "invalid-token"
	mockAuthSvc.On("ValidateToken", mock.Anything, token).Return(nil, auth.ErrInvalidToken)

	// Create test request with invalid auth cookie
	req := httptest.NewRequest("GET", "/graphql", nil)
	req.AddCookie(&http.Cookie{
		Name:  AuthCookieName,
		Value: token,
	})

	recorder := httptest.NewRecorder()

	// Test handler
	var capturedUser *user.User
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// Apply middleware
	middleware.Handler(handler).ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Nil(t, capturedUser) // No user in context due to invalid token

	// Check that cookie was cleared
	cookies := recorder.Result().Cookies()
	var authCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == AuthCookieName {
			authCookie = c
			break
		}
	}
	require.NotNil(t, authCookie)
	assert.Equal(t, -1, authCookie.MaxAge) // Cookie marked for deletion

	mockAuthSvc.AssertExpectations(t)
}

func TestSetAuthCookie(t *testing.T) {
	token := "test-jwt-token"

	// Create response recorder
	recorder := httptest.NewRecorder()

	// Create context with response writer
	ctx := context.WithValue(context.Background(), responseWriterKey, recorder)

	err := SetAuthCookie(ctx, token)

	require.NoError(t, err)

	// Check cookie was set
	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)

	cookie := cookies[0]
	assert.Equal(t, AuthCookieName, cookie.Name)
	assert.Equal(t, token, cookie.Value)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
	assert.Equal(t, "/", cookie.Path)

	// Check expiration is ~1 year
	expectedMaxAge := int((time.Hour * 24 * 365).Seconds())
	assert.Equal(t, expectedMaxAge, cookie.MaxAge)
}

func TestClearAuthCookie(t *testing.T) {
	// Create response recorder
	recorder := httptest.NewRecorder()

	// Create context with response writer
	ctx := context.WithValue(context.Background(), responseWriterKey, recorder)

	err := ClearAuthCookie(ctx)

	require.NoError(t, err)

	// Check cookie was cleared
	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)

	cookie := cookies[0]
	assert.Equal(t, AuthCookieName, cookie.Name)
	assert.Equal(t, "", cookie.Value)
	assert.Equal(t, -1, cookie.MaxAge) // Marked for deletion
}

func TestResponseWriterMiddleware(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	// Handler that checks if ResponseWriter is in context
	var hasWriter bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hasWriter = r.Context().Value(responseWriterKey).(http.ResponseWriter)
		w.WriteHeader(http.StatusOK)
	})

	// Apply middleware
	ResponseWriterMiddleware(handler).ServeHTTP(recorder, req)

	assert.True(t, hasWriter)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
