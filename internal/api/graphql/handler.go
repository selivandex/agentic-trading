package graphql

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"prometheus/internal/api/graphql/generated"
	"prometheus/internal/api/graphql/resolvers"
	fundWatchlistService "prometheus/internal/services/fund_watchlist"
	strategyService "prometheus/internal/services/strategy"
	userService "prometheus/internal/services/user"
)

// Handler creates a new GraphQL HTTP handler
func Handler(
	userSvc *userService.Service,
	strategySvc *strategyService.Service,
	fundWatchlistSvc *fundWatchlistService.Service,
) http.Handler {
	// Create resolver with injected services
	resolver := &resolvers.Resolver{
		UserService:          userSvc,
		StrategyService:      strategySvc,
		FundWatchlistService: fundWatchlistSvc,
	}

	// Create GraphQL schema
	config := generated.Config{Resolvers: resolver}
	schema := generated.NewExecutableSchema(config)

	// Create GraphQL server with options
	srv := handler.NewDefaultServer(schema)

	return srv
}

// PlaygroundHandler creates GraphQL playground handler for development
func PlaygroundHandler() http.Handler {
	return playground.Handler("GraphQL Playground", "/graphql")
}
