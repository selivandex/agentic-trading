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
	}

	// Create GraphQL schema
	config := generated.Config{Resolvers: resolver}
	schema := generated.NewExecutableSchema(config)

	// Create GraphQL server with options
	srv := handler.NewDefaultServer(schema)

	// Wrap with auth middleware (extracts JWT from HTTP-only cookie)
	authMiddleware := middleware.NewAuthMiddleware(authSvc, log)

	// Chain middlewares: ResponseWriter → Auth → GraphQL
	return middleware.ResponseWriterMiddleware(
		authMiddleware.Handler(srv),
	)
}

// PlaygroundHandler creates GraphQL playground handler for development
func PlaygroundHandler() http.Handler {
	return playground.Handler("GraphQL Playground", "/graphql")
}
