package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"

	"msp/db/migrations"
)

// NewReadyCheck returns the readiness check of `msp api` (docs/04 §14): the
// database answers on pool (role msp_app) and holds every migration embedded
// in this binary. A database ahead of the binary is ready, because
// migrations are backward-compatible and a rollback replaces only the app
// (docs/06 §6). The check only reads goose_db_version; it never creates it.
//
// The returned errors may carry internal details (hosts, roles), so callers
// log them instead of showing them to clients.
func NewReadyCheck(pool *pgxpool.Pool) (func(context.Context) error, error) {
	sqlDB := stdlib.OpenDBFromPool(pool) // closing it would not close pool, so it lives as long as pool
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		return nil, fmt.Errorf("db: ready check: %w", err)
	}
	var want int64
	if sources := provider.ListSources(); len(sources) > 0 {
		want = sources[len(sources)-1].Version
	}
	store, err := database.NewStore(database.DialectPostgres, goose.DefaultTablename)
	if err != nil {
		return nil, fmt.Errorf("db: ready check: %w", err)
	}

	return func(ctx context.Context) error {
		got, err := store.GetLatestVersion(ctx, sqlDB)
		switch {
		case errors.Is(err, database.ErrVersionNotFound):
			return fmt.Errorf("db: not ready: no migration applied, binary needs version %d", want)
		case err != nil:
			return fmt.Errorf("db: not ready: %w", err)
		case got < want:
			return fmt.Errorf("db: not ready: database at migration %d, binary needs %d", got, want)
		}
		return nil
	}, nil
}
