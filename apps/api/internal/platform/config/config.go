// Package config reads process configuration from MSP_* environment
// variables once at start-up and validates it (STD-GO-05). It is the only
// package allowed to read the environment.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Config holds process settings. Values that ops change at runtime live in
// the database, not here (docs/04 §14).
type Config struct {
	Env      string // local | staging | production
	LogLevel slog.Level
	HTTPAddr string
}

// Load reads and validates the configuration from the process environment.
func Load() (Config, error) {
	return parse(os.Getenv)
}

func parse(getenv func(string) string) (Config, error) {
	cfg := Config{
		Env:      valueOr(getenv("MSP_ENV"), "local"),
		HTTPAddr: valueOr(getenv("MSP_HTTP_ADDR"), ":8080"),
	}

	switch cfg.Env {
	case "local", "staging", "production":
	default:
		return Config{}, fmt.Errorf("config: MSP_ENV must be local, staging or production, got %q", cfg.Env)
	}

	level := valueOr(getenv("MSP_LOG_LEVEL"), "info")
	if err := cfg.LogLevel.UnmarshalText([]byte(level)); err != nil {
		return Config{}, fmt.Errorf("config: MSP_LOG_LEVEL: %w", err)
	}

	return cfg, nil
}

func valueOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
