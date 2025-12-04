package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_GenerateToken(t *testing.T) {
	service := NewJWTService("test-secret-key-min-32-characters-long", "test-issuer", time.Hour)
	userID := uuid.New()
	email := "test@example.com"

	token, err := service.GenerateToken(userID, email)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTService_ValidateToken(t *testing.T) {
	tests := []struct {
		name      string
		secretKey string
		userID    uuid.UUID
		email     string
		duration  time.Duration
		wantErr   error
	}{
		{
			name:      "valid token",
			secretKey: "test-secret-key-min-32-characters-long",
			userID:    uuid.New(),
			email:     "test@example.com",
			duration:  time.Hour,
			wantErr:   nil,
		},
		{
			name:      "expired token",
			secretKey: "test-secret-key-min-32-characters-long",
			userID:    uuid.New(),
			email:     "test@example.com",
			duration:  -time.Hour, // Already expired
			wantErr:   ErrExpiredToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewJWTService(tt.secretKey, "test-issuer", tt.duration)

			// Generate token
			token, err := service.GenerateToken(tt.userID, tt.email)
			require.NoError(t, err)

			// Validate token
			claims, err := service.ValidateToken(token)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.userID, claims.UserID)
			assert.Equal(t, tt.email, claims.Email)
			assert.Equal(t, tt.userID.String(), claims.Subject)
			assert.Equal(t, "test-issuer", claims.Issuer)
		})
	}
}

func TestJWTService_ValidateToken_InvalidSecret(t *testing.T) {
	service1 := NewJWTService("secret-key-1-min-32-characters-long!!!", "issuer", time.Hour)
	service2 := NewJWTService("secret-key-2-min-32-characters-long!!!", "issuer", time.Hour)

	userID := uuid.New()
	token, err := service1.GenerateToken(userID, "test@example.com")
	require.NoError(t, err)

	// Try to validate with different secret
	_, err = service2.ValidateToken(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTService_ValidateToken_MalformedToken(t *testing.T) {
	service := NewJWTService("test-secret-key-min-32-characters-long", "issuer", time.Hour)

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "invalid format",
			token: "not.a.valid.token",
		},
		{
			name:  "random string",
			token: "random-string-that-is-not-jwt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ValidateToken(tt.token)
			assert.ErrorIs(t, err, ErrInvalidToken)
		})
	}
}

func TestJWTService_RefreshToken(t *testing.T) {
	service := NewJWTService("test-secret-key-min-32-characters-long", "issuer", time.Hour*24*365)
	userID := uuid.New()
	email := "test@example.com"

	// Generate original token
	originalToken, err := service.GenerateToken(userID, email)
	require.NoError(t, err)

	// Wait a bit so timestamps differ
	time.Sleep(time.Millisecond * 10)

	// Refresh token
	newToken, err := service.RefreshToken(originalToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newToken)
	// Tokens might be same if generated at exact same millisecond, that's ok

	// Validate new token
	claims, err := service.ValidateToken(newToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
}

func TestJWTService_TokenExpiration(t *testing.T) {
	// Create service with short duration
	service := NewJWTService("test-secret-key-min-32-characters-long", "issuer", time.Millisecond*500)
	userID := uuid.New()

	token, err := service.GenerateToken(userID, "test@example.com")
	require.NoError(t, err)

	// Token should be valid immediately
	_, err = service.ValidateToken(token)
	assert.NoError(t, err)

	// Wait for token to expire (add extra buffer for clock skew)
	time.Sleep(time.Millisecond * 600)

	// Token should be expired now
	_, err = service.ValidateToken(token)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestJWTService_LongLivedToken(t *testing.T) {
	// Test 1 year duration as per requirements
	service := NewJWTService("test-secret-key-min-32-characters-long", "issuer", time.Hour*24*365)
	userID := uuid.New()
	email := "test@example.com"

	token, err := service.GenerateToken(userID, email)
	require.NoError(t, err)

	claims, err := service.ValidateToken(token)
	require.NoError(t, err)

	// Check expiration is approximately 1 year from now
	expectedExpiry := time.Now().Add(time.Hour * 24 * 365)
	actualExpiry := claims.ExpiresAt.Time

	// Allow 1 minute tolerance
	assert.WithinDuration(t, expectedExpiry, actualExpiry, time.Minute)
}
