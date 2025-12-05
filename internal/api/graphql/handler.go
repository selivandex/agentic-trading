package graphql

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"prometheus/internal/api/graphql/generated"
	"prometheus/internal/api/graphql/middleware"
	"prometheus/internal/api/graphql/resolvers"
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
	log *logger.Logger,
) http.Handler {
	// Create resolver with injected services
	resolver := &resolvers.Resolver{
		AuthService:          authSvc,
		UserService:          userSvc,
		StrategyService:      strategySvc,
		FundWatchlistService: fundWatchlistSvc,
		Log:                  log.With("component", "graphql_resolvers"),
	}

	// Create GraphQL schema
	config := generated.Config{Resolvers: resolver}
	schema := generated.NewExecutableSchema(config)

	// Create GraphQL server with options
	srv := handler.NewDefaultServer(schema)

	// Add logging middleware for operation tracking
	loggingMiddleware := middleware.NewLoggingMiddleware(log)
	srv.AroundOperations(loggingMiddleware.OperationMiddleware())

	// Wrap with auth middleware (extracts JWT from Cookie header set by Next.js)
	authMiddleware := middleware.NewAuthMiddleware(authSvc, log)

	// Apply middlewares (order matters: logging -> auth -> handler)
	// 1. HTTP logging - logs all HTTP requests
	// 2. Auth - validates JWT and adds user to context
	// 3. GraphQL handler with operation logging
	return loggingMiddleware.HTTPLoggingMiddleware(
		authMiddleware.Handler(srv),
	)
}

// PlaygroundHandler creates GraphQL playground handler for development
func PlaygroundHandler() http.Handler {
	return playground.Handler("GraphQL Playground", "/graphql")
}
