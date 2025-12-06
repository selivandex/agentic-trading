package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"prometheus/internal/domain/user"
	"prometheus/pkg/auth"
	pkgerrors "prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// MockUserRepository is a mock for user.Repository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, usr *user.User) error {
	args := m.Called(ctx, usr)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	args := m.Called(ctx, telegramID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, usr *user.User) error {
	args := m.Called(ctx, usr)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, limit, offset int) ([]*user.User, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.User), args.Error(1)
}

func (m *MockUserRepository) GetUsersWithScope(ctx context.Context, scope *string, search *string, filters map[string]any) ([]*user.User, error) {
	args := m.Called(ctx, scope, search, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.User), args.Error(1)
}

func (m *MockUserRepository) GetUsersScopes(ctx context.Context) (map[string]int, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int), args.Error(1)
}

func testLogger() *logger.Logger {
	zapLogger, _ := zap.NewDevelopment()
	return &logger.Logger{SugaredLogger: zapLogger.Sugar()}
}

func TestService_Register(t *testing.T) {
	tests := []struct {
		name    string
		input   RegisterInput
		wantErr error
	}{
		{
			name: "successful registration",
			input: RegisterInput{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: nil,
		},
		{
			name: "invalid input - empty email",
			input: RegisterInput{
				Email:    "",
				Password: "password123",
			},
			wantErr: pkgerrors.ErrInvalidInput,
		},
		{
			name: "invalid input - empty password",
			input: RegisterInput{
				Email:    "test@example.com",
				Password: "",
			},
			wantErr: pkgerrors.ErrInvalidInput,
		},
		{
			name: "email already exists",
			input: RegisterInput{
				Email:     "existing@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: ErrEmailAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			jwtSvc := auth.NewJWTService("test-secret-key-min-32-characters-long", "test", time.Hour*24*365)
			log := testLogger()
			svc := NewService(mockRepo, jwtSvc, log)

			switch tt.wantErr {
			case ErrEmailAlreadyExists:
				// Mock existing user
				existingUser := &user.User{ID: uuid.New()}
				mockRepo.On("GetByEmail", mock.Anything, tt.input.Email).Return(existingUser, nil)
			case nil:
				// Mock email check - not found
				mockRepo.On("GetByEmail", mock.Anything, tt.input.Email).Return(nil, pkgerrors.ErrNotFound)
				// Mock create
				mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *user.User) bool {
					return u.Email != nil &&
						*u.Email == tt.input.Email &&
						u.PasswordHash != nil &&
						!u.CreatedAt.IsZero() &&
						!u.UpdatedAt.IsZero()
				})).Return(nil)
			}

			result, err := svc.Register(context.Background(), tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.Token)
				assert.NotNil(t, result.User)
				assert.Equal(t, tt.input.Email, *result.User.Email)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_Login(t *testing.T) {
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	hashString := string(hashedPassword)

	tests := []struct {
		name     string
		input    LoginInput
		mockUser *user.User
		mockErr  error
		wantErr  error
	}{
		{
			name: "successful login",
			input: LoginInput{
				Email:    "test@example.com",
				Password: password,
			},
			mockUser: &user.User{
				ID:           uuid.New(),
				Email:        stringPtr("test@example.com"),
				PasswordHash: &hashString,
				IsActive:     true,
				CreatedAt:    time.Now().UTC(),
				UpdatedAt:    time.Now().UTC(),
			},
			mockErr: nil,
			wantErr: nil,
		},
		{
			name: "user not found",
			input: LoginInput{
				Email:    "nonexistent@example.com",
				Password: password,
			},
			mockUser: nil,
			mockErr:  pkgerrors.ErrNotFound,
			wantErr:  ErrInvalidCredentials,
		},
		{
			name: "wrong password",
			input: LoginInput{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			mockUser: &user.User{
				ID:           uuid.New(),
				Email:        stringPtr("test@example.com"),
				PasswordHash: &hashString,
				IsActive:     true,
			},
			mockErr: nil,
			wantErr: ErrInvalidCredentials,
		},
		{
			name: "user not active",
			input: LoginInput{
				Email:    "test@example.com",
				Password: password,
			},
			mockUser: &user.User{
				ID:           uuid.New(),
				Email:        stringPtr("test@example.com"),
				PasswordHash: &hashString,
				IsActive:     false,
			},
			mockErr: nil,
			wantErr: pkgerrors.ErrInvalidInput, // Will get "user account is deactivated" error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			jwtSvc := auth.NewJWTService("test-secret-key-min-32-characters-long", "test", time.Hour*24*365)
			log := testLogger()
			svc := NewService(mockRepo, jwtSvc, log)

			mockRepo.On("GetByEmail", mock.Anything, tt.input.Email).Return(tt.mockUser, tt.mockErr)

			result, err := svc.Login(context.Background(), tt.input)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.Token)
				assert.NotNil(t, result.User)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_ValidateToken(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtSvc := auth.NewJWTService("test-secret-key-min-32-characters-long", "test", time.Hour*24*365)
	log := testLogger()
	svc := NewService(mockRepo, jwtSvc, log)

	userID := uuid.New()
	email := "test@example.com"

	// Generate valid token
	token, err := jwtSvc.GenerateToken(userID, email)
	require.NoError(t, err)

	// Mock user lookup
	mockUser := &user.User{
		ID:       userID,
		Email:    &email,
		IsActive: true,
	}
	mockRepo.On("GetByID", mock.Anything, userID).Return(mockUser, nil)

	result, err := svc.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	assert.Equal(t, userID, result.ID)
	mockRepo.AssertExpectations(t)
}

func TestService_ValidateToken_InvalidToken(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtSvc := auth.NewJWTService("test-secret-key-min-32-characters-long", "test", time.Hour*24*365)
	log := testLogger()
	svc := NewService(mockRepo, jwtSvc, log)

	_, err := svc.ValidateToken(context.Background(), "invalid-token")

	assert.ErrorIs(t, err, auth.ErrInvalidToken)
	mockRepo.AssertNotCalled(t, "GetByID")
}

func TestService_ValidateToken_UserNotActive(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtSvc := auth.NewJWTService("test-secret-key-min-32-characters-long", "test", time.Hour*24*365)
	log := testLogger()
	svc := NewService(mockRepo, jwtSvc, log)

	userID := uuid.New()
	email := "test@example.com"

	// Generate valid token
	token, err := jwtSvc.GenerateToken(userID, email)
	require.NoError(t, err)

	// Mock inactive user
	mockUser := &user.User{
		ID:       userID,
		Email:    &email,
		IsActive: false,
	}
	mockRepo.On("GetByID", mock.Anything, userID).Return(mockUser, nil)

	_, err = svc.ValidateToken(context.Background(), token)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deactivated")
	mockRepo.AssertExpectations(t)
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
