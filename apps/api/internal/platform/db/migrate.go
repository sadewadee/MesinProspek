package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"msp/db/migrations"
)

// Migrate applies the pending goose migrations embedded in the binary and
// returns the versions it applied, oldest first. databaseURL connects as the
// schema owner msp_owner (MSP_MIGRATE_DATABASE_URL, docs/04 §14).
func Migrate(ctx context.Context, databaseURL string) (applied []int64, err error) {
	connConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: migrate: %w", err)
	}
	sqlDB := stdlib.OpenDB(*connConfig)
	defer func() {
		if cerr := sqlDB.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("db: migrate: close: %w", cerr))
		}
	}()

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		return nil, fmt.Errorf("db: migrate: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("db: migrate: %w", err)
	}
	applied = make([]int64, 0, len(results))
	for _, r := range results {
		applied = append(applied, r.Source.Version)
	}
	return applied, nil
}
