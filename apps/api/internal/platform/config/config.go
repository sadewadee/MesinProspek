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
	// DatabaseURL connects as the runtime role msp_app (api, worker, tool).
	DatabaseURL string
	// MigrateDatabaseURL connects as the schema owner msp_owner; only
	// `msp migrate` uses it.
	MigrateDatabaseURL string
}

// Load reads the configuration from the process environment and validates
// it for the msp command about to run, so a command fails at start when a
// setting it needs is missing.
func Load(command string) (Config, error) {
	return parse(command, os.Getenv)
}

func parse(command string, getenv func(string) string) (Config, error) {
	cfg := Config{
		Env:                valueOr(getenv("MSP_ENV"), "local"),
		HTTPAddr:           valueOr(getenv("MSP_HTTP_ADDR"), ":8080"),
		DatabaseURL:        strings.TrimSpace(getenv("MSP_DATABASE_URL")),
		MigrateDatabaseURL: strings.TrimSpace(getenv("MSP_MIGRATE_DATABASE_URL")),
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

	if command == "migrate" && cfg.MigrateDatabaseURL == "" {
		return Config{}, fmt.Errorf("config: MSP_MIGRATE_DATABASE_URL is required for msp migrate")
	}
	if command == "api" && cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: MSP_DATABASE_URL is required for msp api")
	}

	return cfg, nil
}

func valueOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// TestEnv holds the settings that only integration tests read (STD-TEST-03,
// STD-TEST-04). It lives here because this is the only package allowed to
// read the environment.
type TestEnv struct {
	// DatabaseURL (MSP_TEST_DATABASE_URL) connects as a superuser to a
	// disposable Postgres 16 + PostGIS server. Tests create and drop their
	// own databases there.
	DatabaseURL string
	// CI (CI=true, set by GitLab CI) makes a missing test database or a
	// missing PostGIS extension fail the test instead of skipping it.
	CI bool
}

// LoadTestEnv reads TestEnv from the process environment.
func LoadTestEnv() TestEnv {
	return parseTestEnv(os.Getenv)
}

func parseTestEnv(getenv func(string) string) TestEnv {
	return TestEnv{
		DatabaseURL: strings.TrimSpace(getenv("MSP_TEST_DATABASE_URL")),
		CI:          getenv("CI") == "true",
	}
}
