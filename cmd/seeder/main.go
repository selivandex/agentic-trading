package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"

	_ "github.com/lib/pq"

	"prometheus/internal/adapters/config"
	devseeds "prometheus/internal/seeds/dev"
	stagingseeds "prometheus/internal/seeds/staging"
	testseeds "prometheus/internal/seeds/test"
	"prometheus/internal/testsupport/seeds"
	"prometheus/pkg/logger"
)

func main() {
	// Parse flags
	env := flag.String("env", "dev", "Environment: dev, staging, test")
	dryRun := flag.Bool("dry-run", false, "List seed functions without executing")
	flag.Parse()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Initialize logger
	if err := logger.Init(cfg.App.LogLevel, cfg.App.Env); err != nil {
		panic("failed to init logger: " + err.Error())
	}

	log := logger.Get()

	log.Infow("Starting seeder",
		"environment", *env,
		"dry_run", *dryRun,
		"database", cfg.Postgres.Database,
	)

	// Get seed functions for environment
	seedFuncs := getSeedFunctions(*env)
	if len(seedFuncs) == 0 {
		log.Warnw("No seeds available for environment", "environment", *env)
		return
	}

	log.Infow("Found seed functions", "environment", *env, "count", len(seedFuncs))

	if *dryRun {
		log.Info("✅ Dry-run mode: seed functions validated")
		return
	}

	// Connect to database
	db, err := connectDB(cfg.Postgres)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Info("Successfully connected to database")

	// Create seeder instance
	ctx := context.Background()
	seeder := seeds.New(db)

	// Execute each seed function in order
	for i, seedFunc := range seedFuncs {
		log.Infow("Executing seed", "step", i+1, "total", len(seedFuncs))

		if err := seedFunc(ctx, seeder); err != nil {
			log.Errorw("Failed to execute seed",
				"step", i+1,
				"error", err,
			)
			return
		}

		log.Infow("✅ Seed completed", "step", i+1)
	}

	log.Info("✅ All seeds applied successfully")
}

// connectDB creates a database connection
func connectDB(cfg config.PostgresConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxConns / 2)

	return db, nil
}

// getSeedFunctions returns seed functions for the given environment
// Order matters - dependencies should be seeded first
func getSeedFunctions(env string) []func(context.Context, *seeds.Seeder) error {
	switch env {
	case "dev":
		return []func(context.Context, *seeds.Seeder) error{
			devseeds.SeedLimitProfiles,
			devseeds.SeedAgents,
			devseeds.SeedUsers,
		}
	case "test":
		return []func(context.Context, *seeds.Seeder) error{
			testseeds.SeedLimitProfiles,
			testseeds.SeedUsers,
		}
	case "staging":
		return []func(context.Context, *seeds.Seeder) error{
			stagingseeds.SeedLimitProfiles,
		}
	default:
		return nil
	}
}
