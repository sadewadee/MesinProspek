// Package db is the way into the MesinProspek database: the pgx pool, the
// workspace transaction for row-level security, the sqlc queries (package q)
// and the goose migrations (docs/04 §3, §4.4).
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"msp/internal/platform/db/q"
)

// DB wraps the pool of the runtime role msp_app.
type DB struct {
	pool *pgxpool.Pool
}

// New returns a DB on pool. The caller owns the pool and closes it.
func New(pool *pgxpool.Pool) *DB {
	return &DB{pool: pool}
}

// InWorkspace runs fn in one transaction whose row-level security context is
// workspace ws and user user, and commits when fn returns nil or rolls back
// otherwise. The context is transaction-local, so it never leaks to the next
// user of the pooled connection. It is the only path to workspace tables
// (STD-DB-03, docs/04 §4.4).
func (d *DB) InWorkspace(ctx context.Context, ws, user uuid.UUID, fn func(*q.Queries, pgx.Tx) error) error {
	if ws == uuid.Nil {
		return errors.New("db: in workspace: workspace id is nil")
	}
	err := pgx.BeginFunc(ctx, d.pool, func(tx pgx.Tx) error {
		queries := q.New(tx)
		err := queries.SetWorkspaceContext(ctx, q.SetWorkspaceContextParams{
			WorkspaceID: ws.String(),
			UserID:      user.String(),
		})
		if err != nil {
			return fmt.Errorf("set workspace context: %w", err)
		}
		return fn(queries, tx)
	})
	if err != nil {
		return fmt.Errorf("db: in workspace: %w", err)
	}
	return nil
}
