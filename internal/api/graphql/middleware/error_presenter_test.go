package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	pkgErrors "prometheus/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHTTPStatusFromError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "nil error returns 200",
			err:            nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ErrUnauthorized returns 401",
			err:            pkgErrors.ErrUnauthorized,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "wrapped ErrUnauthorized returns 401",
			err:            pkgErrors.Wrap(pkgErrors.ErrUnauthorized, "user not found"),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "ErrForbidden returns 403",
			err:            pkgErrors.ErrForbidden,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "ErrNotFound returns 404",
			err:            pkgErrors.ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "ErrAlreadyExists returns 409",
			err:            pkgErrors.ErrAlreadyExists,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "ErrInvalidInput returns 400",
			err:            pkgErrors.ErrInvalidInput,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "ErrTimeout returns 504",
			err:            pkgErrors.ErrTimeout,
			expectedStatus: http.StatusGatewayTimeout,
		},
		{
			name:           "ErrUnavailable returns 503",
			err:            pkgErrors.ErrUnavailable,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "unauthorized in message returns 401",
			err:            errors.New("unauthorized: user not found in context"),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "not authenticated in message returns 401",
			err:            errors.New("not authenticated"),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "unauthenticated in message returns 401",
			err:            errors.New("unauthenticated request"),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "forbidden in message returns 403",
			err:            errors.New("forbidden action"),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "access denied in message returns 403",
			err:            errors.New("access denied"),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "unknown error returns 200",
			err:            errors.New("some random error"),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := getHTTPStatusFromError(tt.err)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestHTTPStatusToCode(t *testing.T) {
	tests := []struct {
		status       int
		expectedCode string
	}{
		{http.StatusUnauthorized, "UNAUTHENTICATED"},
		{http.StatusForbidden, "FORBIDDEN"},
		{http.StatusNotFound, "NOT_FOUND"},
		{http.StatusBadRequest, "BAD_REQUEST"},
		{http.StatusConflict, "CONFLICT"},
		{http.StatusInternalServerError, "INTERNAL_ERROR"},
		{http.StatusServiceUnavailable, "UNAVAILABLE"},
		{http.StatusGatewayTimeout, "TIMEOUT"},
		{http.StatusTeapot, "UNKNOWN"}, // Any unknown status
	}

	for _, tt := range tests {
		t.Run(tt.expectedCode, func(t *testing.T) {
			code := httpStatusToCode(tt.status)
			assert.Equal(t, tt.expectedCode, code)
		})
	}
}

func TestHTTPStatusMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.Handler
		expectedStatus int
	}{
		{
			name: "successful response",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("success"))
			}),
			expectedStatus: http.StatusOK,
		},
		{
			name: "unauthorized response",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate what error presenter would do
				if sw, ok := w.(*statusCodeResponseWriter); ok {
					sw.statusCode = http.StatusUnauthorized
					sw.WriteHeader(http.StatusUnauthorized)
				}
				_, _ = w.Write([]byte("unauthorized"))
			}),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("POST", "/graphql", nil)
			rec := httptest.NewRecorder()

			// Wrap handler with middleware
			handler := HTTPStatusMiddleware(tt.handler)

			// Execute request
			handler.ServeHTTP(rec, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestStatusCodeResponseWriter(t *testing.T) {
	t.Run("captures body and detects error code", func(t *testing.T) {
		rec := httptest.NewRecorder()
		sw := &statusCodeResponseWriter{
			ResponseWriter: rec,
			statusCode:     http.StatusOK,
		}

		// Write GraphQL response with UNAUTHENTICATED error
		body := []byte(`{"errors":[{"message":"unauthorized","extensions":{"code":"UNAUTHENTICATED"}}],"data":null}`)
		_, _ = sw.Write(body)

		// Should have written 401 status
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.True(t, sw.written)
	})

	t.Run("writes 200 for successful response", func(t *testing.T) {
		rec := httptest.NewRecorder()
		sw := &statusCodeResponseWriter{
			ResponseWriter: rec,
			statusCode:     http.StatusOK,
		}

		// Write successful GraphQL response
		body := []byte(`{"data":{"me":{"id":"123","email":"test@example.com"}}}`)
		_, _ = sw.Write(body)

		// Should have written 200 status
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, sw.written)
	})

	t.Run("detects FORBIDDEN code", func(t *testing.T) {
		rec := httptest.NewRecorder()
		sw := &statusCodeResponseWriter{
			ResponseWriter: rec,
			statusCode:     http.StatusOK,
		}

		// Write GraphQL response with FORBIDDEN error
		body := []byte(`{"errors":[{"message":"forbidden","extensions":{"code":"FORBIDDEN"}}],"data":null}`)
		_, _ = sw.Write(body)

		// Should have written 403 status
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestErrorPresenter(t *testing.T) {
	presenter := ErrorPresenter()

	tests := []struct {
		name         string
		err          error
		expectedCode string
	}{
		{
			name:         "unauthorized error",
			err:          pkgErrors.ErrUnauthorized,
			expectedCode: "UNAUTHENTICATED",
		},
		{
			name:         "forbidden error",
			err:          pkgErrors.ErrForbidden,
			expectedCode: "FORBIDDEN",
		},
		{
			name:         "not found error",
			err:          pkgErrors.ErrNotFound,
			expectedCode: "NOT_FOUND",
		},
		{
			name:         "wrapped unauthorized",
			err:          pkgErrors.Wrap(pkgErrors.ErrUnauthorized, "user not found"),
			expectedCode: "UNAUTHENTICATED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with status code writer
			rec := httptest.NewRecorder()
			sw := &statusCodeResponseWriter{
				ResponseWriter: rec,
				statusCode:     http.StatusOK,
			}
			ctx := context.WithValue(context.Background(), responseWriterKey, sw)

			// Present error
			gqlErr := presenter(ctx, tt.err)

			// Check GraphQL error
			require.NotNil(t, gqlErr)
			assert.NotEmpty(t, gqlErr.Message)

			// Check extension code
			require.NotNil(t, gqlErr.Extensions)
			code, ok := gqlErr.Extensions["code"].(string)
			require.True(t, ok)
			assert.Equal(t, tt.expectedCode, code)
		})
	}
}

func TestErrorPresenterWithoutContext(t *testing.T) {
	presenter := ErrorPresenter()

	// Create simple context without response writer
	ctx := context.Background()

	// Present error
	err := pkgErrors.ErrUnauthorized
	gqlErr := presenter(ctx, err)

	// Should still work, just won't set status code
	require.NotNil(t, gqlErr)
	assert.NotEmpty(t, gqlErr.Message)

	// Check extension code
	require.NotNil(t, gqlErr.Extensions)
	code, ok := gqlErr.Extensions["code"].(string)
	require.True(t, ok)
	assert.Equal(t, "UNAUTHENTICATED", code)
}
