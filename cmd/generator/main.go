package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"prometheus/pkg/logger"
)

func main() {
	// Parse flags
	tableName := flag.String("table", "", "PostgreSQL table name")
	resourceName := flag.String("resource", "", "Resource name (default: PascalCase of table)")
	dryRun := flag.Bool("dry-run", false, "Show what would be generated without creating files")
	backendOnly := flag.Bool("backend-only", false, "Generate only backend code")
	frontendOnly := flag.Bool("frontend-only", false, "Generate only frontend code")
	flag.Parse()

	if *tableName == "" {
		fmt.Println("Error: --table flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Setup logger
	log := logger.MustNew("info")

	ctx := context.Background()

	// Initialize generator
	gen := NewGenerator(log)

	// Configure generator
	config := &Config{
		TableName:    *tableName,
		ResourceName: *resourceName,
		DryRun:       *dryRun,
		BackendOnly:  *backendOnly,
		FrontendOnly: *frontendOnly,
	}

	// Run generation
	if err := gen.Generate(ctx, config); err != nil {
		log.Errorw("Generation failed", "error", err)
		os.Exit(1)
	}

	log.Infow("✅ Generation completed successfully!")
}

