package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the runtime configuration loaded from the environment.
type Config struct {
	DiscordToken string
	FluxerToken  string
	FluxerOrigin string
	DBPath       string
}

// Load reads configuration from a local .env file (if present) and the
// process environment. It fails when a required secret is missing.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DiscordToken: os.Getenv("DISCORD_BOT_TOKEN"),
		FluxerToken:  os.Getenv("FLUXER_BOT_TOKEN"),
		FluxerOrigin: getenvDefault("FLUXER_ORIGIN", "https://fluxer.app"),
		DBPath:       getenvDefault("DB_PATH", "./data/fluxify.db"),
	}

	if cfg.DiscordToken == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN is not set")
	}
	if cfg.FluxerToken == "" {
		return nil, fmt.Errorf("FLUXER_BOT_TOKEN is not set")
	}
	// only https origins are allowed for security reasons
	if (cfg.FluxerOrigin == "" || cfg.FluxerOrigin[:8] != "https://") {
		return nil, fmt.Errorf("FLUXER_ORIGIN must be a valid https URL")
	}
	return cfg, nil
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
