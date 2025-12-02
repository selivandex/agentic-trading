package testsupport

import (
	"log"
	"os"
	"sync"

	"prometheus/internal/adapters/config"
)

var (
	cfg     *config.Config
	cfgOnce sync.Once
	cfgErr  error
)

// init sets ENV=test to ensure .env.test is loaded
func init() {
	if os.Getenv("ENV") == "" {
		_ = os.Setenv("ENV", "test")
	}
}

// loadConfig loads configuration from environment (called once via sync.Once)
// This is lazy - only loads when GetConfig() is first called
func loadConfig() {
	cfgOnce.Do(func() {
		cfg, cfgErr = config.Load()
		if cfgErr != nil {
			// Don't panic here - let GetConfig() decide
			log.Printf("Warning: failed to load config in testsupport: %v", cfgErr)
		}
	})
}

// GetConfig returns the loaded configuration, loading it if needed
func GetConfig() *config.Config {
	loadConfig()
	if cfgErr != nil {
		panic(cfgErr)
	}
	return cfg
}

// MustGetConfig is an alias for GetConfig
func MustGetConfig() *config.Config {
	return GetConfig()
}

type Client struct {
	Cfg *config.Config
}

func NewClient() *Client {
	return &Client{Cfg: GetConfig()}
}
