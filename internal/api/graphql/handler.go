package graphql

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"prometheus/internal/api/graphql/generated"
	"prometheus/internal/api/graphql/middleware"
	"prometheus/internal/api/graphql/resolvers"
	agentService "prometheus/internal/services/agent"
	authService "prometheus/internal/services/auth"
	fundWatchlistService "prometheus/internal/services/fundwatchlist"
	strategyService "prometheus/internal/services/strategy"
	userService "prometheus/internal/services/user"
	"prometheus/pkg/logger"
)

// Handler creates a new GraphQL HTTP handler with auth middleware
func Handler(
	authSvc *authService.Service,
	userSvc *userService.Service,
	strategySvc *strategyService.Service,
	fundWatchlistSvc *fundWatchlistService.Service,
	agentSvc *agentService.Service,
	log *logger.Logger,
) http.Handler {
	// Create resolver with injected services
	resolver := &resolvers.Resolver{
		AuthService:          authSvc,
		UserService:          userSvc,
		StrategyService:      strategySvc,
		FundWatchlistService: fundWatchlistSvc,
		AgentService:         agentSvc,
		Log:                  log.With("component", "graphql_resolvers"),
	}

	// Create GraphQL schema
	config := generated.Config{Resolvers: resolver}
	schema := generated.NewExecutableSchema(config)

	// Create GraphQL server with options
	srv := handler.NewDefaultServer(schema)

	// Set error presenter to handle HTTP status codes for auth errors
	srv.SetErrorPresenter(middleware.ErrorPresenter())

	// Add logging middleware for operation tracking
	loggingMiddleware := middleware.NewLoggingMiddleware(log)
	srv.AroundOperations(loggingMiddleware.OperationMiddleware())

	// Wrap with auth middleware (extracts JWT from Cookie header set by Next.js)
	authMiddleware := middleware.NewAuthMiddleware(authSvc, log)

	// Batch middleware for handling array of GraphQL requests
	batchMiddleware := middleware.NewBatchMiddleware(log)

	// Apply middlewares (order matters: batch -> status -> logging -> auth -> handler)
	// 1. Batch support - handles array of GraphQL requests
	// 2. HTTP status - sets proper HTTP status codes for errors
	// 3. HTTP logging - logs all HTTP requests
	// 4. Auth - validates JWT and adds user to context
	// 5. GraphQL handler with operation logging
	return batchMiddleware.Handler(
		middleware.HTTPStatusMiddleware(
			loggingMiddleware.HTTPLoggingMiddleware(
				authMiddleware.Handler(srv),
			),
		),
	)
}

// PlaygroundHandler creates GraphQL playground handler for development
func PlaygroundHandler() http.Handler {
	return playground.Handler("GraphQL Playground", "/graphql")
}
