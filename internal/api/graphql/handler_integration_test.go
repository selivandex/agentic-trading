package graphql_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	graphql "prometheus/internal/api/graphql"
	"prometheus/internal/domain/user"
	pgrepo "prometheus/internal/repository/postgres"
	authservice "prometheus/internal/services/auth"
	userservice "prometheus/internal/services/user"
	"prometheus/internal/testsupport"
	"prometheus/internal/testsupport/seeds"
	"prometheus/pkg/auth"
	"prometheus/pkg/logger"
)

// GraphQL request structure
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQL response structure
type GraphQLResponse struct {
	Data   map[string]interface{}   `json:"data"`
	Errors []map[string]interface{} `json:"errors,omitempty"`
}

func testLogger() *logger.Logger {
	zapLogger, _ := zap.NewDevelopment()
	return &logger.Logger{SugaredLogger: zapLogger.Sugar()}
}

// limitProfileAdapter adapts postgres LimitProfileRepository to domain interface
type limitProfileAdapter struct {
	repo *pgrepo.LimitProfileRepository
}

func (a *limitProfileAdapter) GetByName(ctx context.Context, name string) (user.LimitProfileInfo, error) {
	profile, err := a.repo.GetByName(ctx, name)
	if err != nil {
		return user.LimitProfileInfo{}, err
	}
	return user.LimitProfileInfo{ID: profile.ID}, nil
}

// testSetup contains all dependencies for GraphQL integration tests
type testSetup struct {
	pg          *testsupport.PostgresTestHelper
	handler     http.Handler
	authService *authservice.Service
	userService *userservice.Service
	jwtService  *auth.JWTService
	userRepo    *pgrepo.UserRepository
	log         *logger.Logger
}

// setupGraphQLTest creates all necessary dependencies for GraphQL integration tests
// This follows DRY principle - all tests use same setup
func setupGraphQLTest(t *testing.T) *testSetup {
	t.Helper()

	// Setup test database with transaction (auto-rollback)
	pg := testsupport.NewTestPostgres(t)

	// Seed required data (use unique name to avoid conflicts)
	seeder := seeds.New(pg.DB())
	seeder.LimitProfile().
		WithFreeTier().
		MustInsert() // Uses auto-generated unique name

	// Create repositories
	userRepo := pgrepo.NewUserRepository(pg.DB())
	limitProfileRepo := pgrepo.NewLimitProfileRepository(pg.DB())

	// Create services
	log := testLogger()
	jwtService := auth.NewJWTService("test-secret-key-min-32-characters-long", "test-issuer", time.Hour*24)
	authSvc := authservice.NewService(userRepo, jwtService, log)

	// User service with limit profile adapter
	limitProfileAdapter := &limitProfileAdapter{repo: limitProfileRepo}
	domainUserSvc := user.NewService(userRepo, limitProfileAdapter)
	userSvc := userservice.NewService(domainUserSvc, log)

	// GraphQL handler (strategy and fundWatchlist services not needed for auth tests)
	handler := graphql.Handler(authSvc, userSvc, nil, nil, log)

	return &testSetup{
		pg:          pg,
		handler:     handler,
		authService: authSvc,
		userService: userSvc,
		jwtService:  jwtService,
		userRepo:    userRepo,
		log:         log,
	}
}

func TestGraphQLHandler_RegisterAndLogin_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupGraphQLTest(t)
	defer setup.pg.Close()

	// Test 1: Register new user
	t.Run("Register creates user and returns token with cookie", func(t *testing.T) {
		// Use unique email for this test run
		testEmail := fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])

		registerMutation := `
			mutation Register($input: RegisterInput!) {
				register(input: $input) {
					token
					user {
						id
						email
						firstName
						lastName
					}
				}
			}
		`

		reqBody := GraphQLRequest{
			Query: registerMutation,
			Variables: map[string]interface{}{
				"input": map[string]interface{}{
					"email":     testEmail,
					"password":  "password123",
					"firstName": "John",
					"lastName":  "Doe",
				},
			},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		setup.handler.ServeHTTP(recorder, req)

		// Check response
		assert.Equal(t, http.StatusOK, recorder.Code)

		var response GraphQLResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should have no errors
		assert.Empty(t, response.Errors)

		// Check data
		registerData := response.Data["register"].(map[string]interface{})
		token := registerData["token"].(string)
		assert.NotEmpty(t, token)

		userData := registerData["user"].(map[string]interface{})
		assert.Equal(t, testEmail, userData["email"])
		assert.Equal(t, "John", userData["firstName"])
		assert.Equal(t, "Doe", userData["lastName"])

		// Backend returns token in JSON (Next.js will save to session)
		// No Set-Cookie header from backend
	})

	// Test 2: Login with credentials from test 1
	t.Run("Login with valid credentials returns token", func(t *testing.T) {
		// First register a user
		testEmail := fmt.Sprintf("login-%s@example.com", uuid.New().String()[:8])
		registerAndGetEmail(t, setup.handler, testEmail, "password123")

		loginMutation := `
			mutation Login($input: LoginInput!) {
				login(input: $input) {
					token
					user {
						id
						email
						createdAt
						updatedAt
					}
				}
			}
		`

		reqBody := GraphQLRequest{
			Query: loginMutation,
			Variables: map[string]interface{}{
				"input": map[string]interface{}{
					"email":    testEmail,
					"password": "password123",
				},
			},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		setup.handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var response GraphQLResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Empty(t, response.Errors)

		loginData := response.Data["login"].(map[string]interface{})
		token := loginData["token"].(string)
		assert.NotEmpty(t, token)

		// Verify JWT token is valid
		claims, err := setup.jwtService.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, testEmail, claims.Email)
	})

	// Test 3: Login with wrong password fails
	t.Run("Login with wrong password fails", func(t *testing.T) {
		loginMutation := `
			mutation Login($input: LoginInput!) {
				login(input: $input) {
					token
					user {
						id
						createdAt
						updatedAt
					}
				}
			}
		`

		reqBody := GraphQLRequest{
			Query: loginMutation,
			Variables: map[string]interface{}{
				"input": map[string]interface{}{
					"email":    "test@example.com",
					"password": "wrongpassword",
				},
			},
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		setup.handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var response GraphQLResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should have errors
		assert.NotEmpty(t, response.Errors)
		assert.Nil(t, response.Data["login"])
	})
}

func TestGraphQLHandler_MeQuery_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupGraphQLTest(t)
	defer setup.pg.Close()

	// Create test user directly in DB
	userID := uuid.New()
	testUser := &user.User{
		ID:               userID,
		Email:            stringPtr(fmt.Sprintf("test-me-%s@example.com", userID.String()[:8])),
		PasswordHash:     stringPtr("$2a$10$hash"), // Not important for this test
		FirstName:        "Existing",
		LastName:         "User",
		LanguageCode:     "en",
		IsActive:         true,
		IsPremium:        false,
		Settings:         user.DefaultSettings(),
		TelegramID:       nil, // Email user, no telegram
		TelegramUsername: "",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err := setup.userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)

	// Generate valid token
	token, err := setup.jwtService.GenerateToken(testUser.ID, *testUser.Email)
	require.NoError(t, err)

	// Test 1: Me query with valid token in cookie
	t.Run("Me query with valid cookie returns user", func(t *testing.T) {
		meQuery := `
			query {
				me {
					id
					email
					firstName
					lastName
					isActive
				}
			}
		`

		reqBody := GraphQLRequest{
			Query: meQuery,
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: token,
		})

		recorder := httptest.NewRecorder()
		setup.handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var response GraphQLResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Empty(t, response.Errors)

		meData := response.Data["me"].(map[string]interface{})
		assert.Equal(t, testUser.ID.String(), meData["id"])
		assert.Equal(t, *testUser.Email, meData["email"])
		assert.Equal(t, testUser.FirstName, meData["firstName"])
		assert.Equal(t, true, meData["isActive"])
	})

	// Test 2: Me query without token fails
	t.Run("Me query without cookie fails", func(t *testing.T) {
		meQuery := `
			query {
				me {
					id
					email
				}
			}
		`

		reqBody := GraphQLRequest{
			Query: meQuery,
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		// No cookie!

		recorder := httptest.NewRecorder()
		setup.handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var response GraphQLResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should have error
		assert.NotEmpty(t, response.Errors)
		assert.Nil(t, response.Data["me"])
	})

	// Test 3: Me query with invalid token fails
	t.Run("Me query with invalid token fails", func(t *testing.T) {
		meQuery := `
			query {
				me {
					id
					email
				}
			}
		`

		reqBody := GraphQLRequest{
			Query: meQuery,
		}

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: "invalid-token-string",
		})

		recorder := httptest.NewRecorder()
		setup.handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var response GraphQLResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should have error
		assert.NotEmpty(t, response.Errors)
	})
}

func TestGraphQLHandler_FullAuthFlow_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupGraphQLTest(t)
	defer setup.pg.Close()

	// Full flow test: Register → Login → Me query
	t.Run("Complete auth flow: register, login, me query", func(t *testing.T) {
		email := fmt.Sprintf("flow-%s@test.com", uuid.New().String()[:8])
		password := "testpass123"

		// Step 1: Register
		registerMutation := `
			mutation Register($input: RegisterInput!) {
				register(input: $input) {
					token
					user {
						id
						email
						firstName
					}
				}
			}
		`

		registerReq := GraphQLRequest{
			Query: registerMutation,
			Variables: map[string]interface{}{
				"input": map[string]interface{}{
					"email":     email,
					"password":  password,
					"firstName": "Integration",
					"lastName":  "Test",
				},
			},
		}

		bodyBytes, _ := json.Marshal(registerReq)
		req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		setup.handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var registerResponse GraphQLResponse
		err := json.Unmarshal(recorder.Body.Bytes(), &registerResponse)
		require.NoError(t, err)
		assert.Empty(t, registerResponse.Errors, "Register should succeed")

		registerData := registerResponse.Data["register"].(map[string]interface{})
		registerToken := registerData["token"].(string)
		userID := registerData["user"].(map[string]interface{})["id"].(string)

		// Backend returns token in JSON (Next.js manages session)
		assert.NotEmpty(t, registerToken)

		// Step 2: Login with same credentials
		loginMutation := `
			mutation Login($input: LoginInput!) {
				login(input: $input) {
					token
					user {
						id
						email
					}
				}
			}
		`

		loginReq := GraphQLRequest{
			Query: loginMutation,
			Variables: map[string]interface{}{
				"input": map[string]interface{}{
					"email":    email,
					"password": password,
				},
			},
		}

		bodyBytes, _ = json.Marshal(loginReq)
		req = httptest.NewRequest("POST", "/graphql", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		recorder = httptest.NewRecorder()
		setup.handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var loginResponse GraphQLResponse
		err = json.Unmarshal(recorder.Body.Bytes(), &loginResponse)
		require.NoError(t, err)
		assert.Empty(t, loginResponse.Errors, "Login should succeed")

		loginData := loginResponse.Data["login"].(map[string]interface{})
		loginToken := loginData["token"].(string)
		assert.NotEmpty(t, loginToken)

		// Step 3: Use cookie to query /me
		meQuery := `
			query {
				me {
					id
					email
					firstName
					lastName
				}
			}
		`

		meReq := GraphQLRequest{
			Query: meQuery,
		}

		bodyBytes, _ = json.Marshal(meReq)
		req = httptest.NewRequest("POST", "/graphql", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		// Simulate Next.js sending token in Cookie header
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: loginToken,
		})

		recorder = httptest.NewRecorder()
		setup.handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var meResponse GraphQLResponse
		err = json.Unmarshal(recorder.Body.Bytes(), &meResponse)
		require.NoError(t, err)
		assert.Empty(t, meResponse.Errors, "Me query should succeed with valid cookie")

		meData := meResponse.Data["me"].(map[string]interface{})
		assert.Equal(t, userID, meData["id"])
		assert.Equal(t, email, meData["email"])
		assert.Equal(t, "Integration", meData["firstName"])

		// Step 4: Logout
		logoutMutation := `
			mutation {
				logout
			}
		`

		logoutReq := GraphQLRequest{
			Query: logoutMutation,
		}

		bodyBytes, _ = json.Marshal(logoutReq)
		req = httptest.NewRequest("POST", "/graphql", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		recorder = httptest.NewRecorder()
		setup.handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var logoutResponse GraphQLResponse
		err = json.Unmarshal(recorder.Body.Bytes(), &logoutResponse)
		require.NoError(t, err)
		assert.Empty(t, logoutResponse.Errors)
		assert.Equal(t, true, logoutResponse.Data["logout"])

		// Backend just returns true - Next.js will clear session
		// No Set-Cookie header from backend

		// Step 5: Try to use /me without token (simulating cleared session)
		bodyBytes, _ = json.Marshal(meReq)
		req = httptest.NewRequest("POST", "/graphql", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		// No cookie - Next.js cleared session

		recorder = httptest.NewRecorder()
		setup.handler.ServeHTTP(recorder, req)

		var meAfterLogout GraphQLResponse
		err = json.Unmarshal(recorder.Body.Bytes(), &meAfterLogout)
		require.NoError(t, err)
		assert.NotEmpty(t, meAfterLogout.Errors, "Me query should fail after logout")
	})
}

func TestGraphQLHandler_ConcurrentRequests_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupGraphQLTest(t)
	defer setup.pg.Close()

	t.Run("Multiple concurrent registrations", func(t *testing.T) {
		numUsers := 5
		results := make(chan error, numUsers)

		for i := 0; i < numUsers; i++ {
			go func(index int) {
				registerMutation := `
					mutation Register($input: RegisterInput!) {
						register(input: $input) {
							token
							user { id email }
						}
					}
				`

				reqBody := GraphQLRequest{
					Query: registerMutation,
					Variables: map[string]interface{}{
						"input": map[string]interface{}{
							"email":     fmt.Sprintf("user%d-%s@test.com", index, uuid.New().String()[:8]),
							"password":  "password123",
							"firstName": "User",
							"lastName":  fmt.Sprintf("%d", index),
						},
					},
				}

				bodyBytes, _ := json.Marshal(reqBody)
				req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")

				recorder := httptest.NewRecorder()
				setup.handler.ServeHTTP(recorder, req)

				var response GraphQLResponse
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					results <- err
					return
				}

				if len(response.Errors) > 0 {
					results <- fmt.Errorf("GraphQL error: %v", response.Errors)
					return
				}

				results <- nil
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numUsers; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent registration %d should succeed", i)
		}
	})
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

// registerAndGetEmail is a helper to register a user for subsequent tests
func registerAndGetEmail(t *testing.T, handler http.Handler, email, password string) {
	t.Helper()

	registerMutation := `
		mutation Register($input: RegisterInput!) {
			register(input: $input) {
				token
				user { id }
			}
		}
	`

	reqBody := GraphQLRequest{
		Query: registerMutation,
		Variables: map[string]interface{}{
			"input": map[string]interface{}{
				"email":     email,
				"password":  password,
				"firstName": "Test",
				"lastName":  "User",
			},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
}
