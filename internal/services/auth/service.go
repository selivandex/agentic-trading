package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"prometheus/internal/domain/user"
	"prometheus/pkg/auth"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

var (
	// ErrInvalidCredentials is returned when email/password don't match
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrEmailAlreadyExists is returned when trying to register with existing email
	ErrEmailAlreadyExists = errors.New("email already exists")
	// ErrUserNotFound is returned when user doesn't exist
	ErrUserNotFound = errors.New("user not found")
)

// Service handles authentication logic (Application Layer)
type Service struct {
	userRepo   user.Repository
	jwtService *auth.JWTService
	log        *logger.Logger
}

// NewService creates a new auth service
func NewService(userRepo user.Repository, jwtService *auth.JWTService, log *logger.Logger) *Service {
	return &Service{
		userRepo:   userRepo,
		jwtService: jwtService,
		log:        log.With("service", "auth"),
	}
}

// RegisterInput contains data for user registration
type RegisterInput struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

// LoginInput contains data for user login
type LoginInput struct {
	Email    string
	Password string
}

// AuthResponse contains auth result with JWT token
type AuthResponse struct {
	Token string
	User  *user.User
}

// Register registers a new user with email/password
func (s *Service) Register(ctx context.Context, input RegisterInput) (*AuthResponse, error) {
	// Validate input
	if input.Email == "" || input.Password == "" {
		return nil, errors.ErrInvalidInput
	}

	// Check if email already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil && !errors.Is(err, errors.ErrNotFound) {
		return nil, errors.Wrap(err, "failed to check email")
	}
	if existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(err, "failed to hash password")
	}

	// Create user
	hashString := string(passwordHash)
	now := time.Now().UTC()
	usr := &user.User{
		ID:               uuid.New(),
		Email:            &input.Email,
		PasswordHash:     &hashString,
		FirstName:        input.FirstName,
		LastName:         input.LastName,
		LanguageCode:     "en",
		IsActive:         true,
		IsPremium:        false,
		Settings:         user.DefaultSettings(),
		TelegramID:       nil, // NULL for email-based users
		TelegramUsername: "",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.userRepo.Create(ctx, usr); err != nil {
		return nil, errors.Wrap(err, "failed to create user")
	}

	// Generate JWT token
	token, err := s.jwtService.GenerateToken(usr.ID, input.Email)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate token")
	}

	s.log.Infow("User registered",
		"user_id", usr.ID,
		"email", input.Email,
	)

	return &AuthResponse{
		Token: token,
		User:  usr,
	}, nil
}

// Login authenticates a user with email/password
func (s *Service) Login(ctx context.Context, input LoginInput) (*AuthResponse, error) {
	// Validate input
	if input.Email == "" || input.Password == "" {
		return nil, errors.ErrInvalidInput
	}

	// Get user by email
	usr, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, errors.Wrap(err, "failed to get user")
	}

	// Check if user has password hash
	if usr.PasswordHash == nil {
		return nil, fmt.Errorf("user %s registered via Telegram, use Telegram auth", input.Email)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(*usr.PasswordHash), []byte(input.Password)); err != nil {
		s.log.Debugw("Failed login attempt", "email", input.Email)
		return nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !usr.IsActive {
		return nil, errors.New("user account is deactivated")
	}

	// Generate JWT token
	token, err := s.jwtService.GenerateToken(usr.ID, input.Email)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate token")
	}

	s.log.Infow("User logged in",
		"user_id", usr.ID,
		"email", input.Email,
	)

	return &AuthResponse{
		Token: token,
		User:  usr,
	}, nil
}

// ValidateToken validates JWT token and returns user
func (s *Service) ValidateToken(ctx context.Context, token string) (*user.User, error) {
	// Validate JWT
	claims, err := s.jwtService.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	// Get user from database
	usr, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, errors.Wrap(err, "failed to get user")
	}

	// Check if user is active
	if !usr.IsActive {
		return nil, errors.New("user account is deactivated")
	}

	return usr, nil
}

// GetUserFromToken is a convenience method that extracts user from token
func (s *Service) GetUserFromToken(token string) (uuid.UUID, error) {
	claims, err := s.jwtService.ValidateToken(token)
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
}
