package db_test

import (
	"context"
	"crypto/rand"
	"errors"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"msp/internal/platform/config"
	"msp/internal/platform/db"
	"msp/internal/platform/db/q"
)

// rlsProbe is a test-only workspace table with the RLS pattern of docs/04
// §4.4, owned by msp_owner like every migrated table. The setting is wrapped
// in NULLIF because after a transaction-local set_config a pooled connection
// reads it as an empty string instead of NULL, and casting an empty string
// to uuid is an error, not zero rows.
const rlsProbe = `
CREATE TABLE rls_probe (
	workspace_id uuid NOT NULL,
	note text NOT NULL
);
ALTER TABLE rls_probe ENABLE ROW LEVEL SECURITY;
ALTER TABLE rls_probe FORCE ROW LEVEL SECURITY;
CREATE POLICY ws ON rls_probe
	USING      (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid)
	WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::uuid);
`

func TestMigrateAndWorkspaceRLS(t *testing.T) {
	ctx := t.Context()
	dbURL := newDatabase(t)

	migrated := t.Run("migrations run from an empty database", func(t *testing.T) {
		applied, err := db.Migrate(ctx, withRole(t, dbURL, "msp_owner"))
		if err != nil {
			t.Fatalf("Migrate() error = %v", err)
		}
		if want := []int64{1}; !slices.Equal(applied, want) {
			t.Fatalf("Migrate() applied %v, want %v", applied, want)
		}
		if again, err := db.Migrate(ctx, withRole(t, dbURL, "msp_owner")); err != nil || len(again) != 0 {
			t.Fatalf("second Migrate() = %v, %v; want nothing applied", again, err)
		}
		super := connect(t, dbURL)
		rows, _ := super.Query(ctx, `SELECT extname FROM pg_extension WHERE extname IN ('postgis', 'pg_trgm') ORDER BY 1`)
		exts, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil || !slices.Equal(exts, []string{"pg_trgm", "postgis"}) {
			t.Fatalf("extensions = %v, %v; want pg_trgm and postgis", exts, err)
		}
		var owner string
		if err := super.QueryRow(ctx, `SELECT tableowner FROM pg_tables WHERE tablename = 'goose_db_version'`).Scan(&owner); err != nil || owner != "msp_owner" {
			t.Fatalf("goose_db_version owner = %q, %v; want msp_owner", owner, err)
		}
	})
	if !migrated {
		return
	}

	if _, err := connect(t, withRole(t, dbURL, "msp_owner")).Exec(ctx, rlsProbe); err != nil {
		t.Fatalf("create rls_probe: %v", err)
	}

	// One connection, so every query below reuses the connection that ran
	// the previous InWorkspace transaction.
	pool := newPool(t, withRole(t, dbURL, "msp_app")+"&pool_max_conns=1")
	d := db.New(pool)

	t.Run("runtime role is msp_app without superuser or BYPASSRLS", func(t *testing.T) {
		var role, owner string
		var super, bypass bool
		err := pool.QueryRow(ctx, `
			SELECT current_user, r.rolsuper, r.rolbypassrls, t.tableowner
			FROM pg_roles r, pg_tables t
			WHERE r.rolname = current_user AND t.tablename = 'rls_probe'`).Scan(&role, &super, &bypass, &owner)
		if err != nil {
			t.Fatal(err)
		}
		if role != "msp_app" || super || bypass || owner != "msp_owner" {
			t.Fatalf("role %s (superuser %t, bypassrls %t), rls_probe owner %s; want msp_app, false, false, msp_owner", role, super, bypass, owner)
		}
		var version int64
		if err := pool.QueryRow(ctx, `SELECT max(version_id) FROM goose_db_version`).Scan(&version); err != nil || version != 1 {
			t.Fatalf("msp_app reads goose version %d, %v; want 1", version, err)
		}
	})

	wsA, wsB, user := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	insert(t, d, wsA, user, "a1", "a2")
	insert(t, d, wsB, user, "b1")

	t.Run("query without workspace context returns 0 rows", func(t *testing.T) {
		if got := count(t, pool); got != 0 {
			t.Fatalf("count without context = %d, want 0", got)
		}
		if got := count(t, connect(t, dbURL)); got != 3 {
			t.Fatalf("count as superuser = %d, want 3 rows to exist", got)
		}
	})

	t.Run("workspace sees only its own rows", func(t *testing.T) {
		for ws, want := range map[uuid.UUID][]string{wsA: {"a1", "a2"}, wsB: {"b1"}} {
			if got := notes(t, d, ws, user); !slices.Equal(got, want) {
				t.Fatalf("notes in %s = %v, want %v", ws, got, want)
			}
		}
	})

	t.Run("context is set for the transaction", func(t *testing.T) {
		var gotWS, gotUser string
		err := d.InWorkspace(ctx, wsA, user, func(_ *q.Queries, tx pgx.Tx) error {
			return tx.QueryRow(ctx, `SELECT current_setting('app.workspace_id'), current_setting('app.user_id')`).Scan(&gotWS, &gotUser)
		})
		if err != nil || gotWS != wsA.String() || gotUser != user.String() {
			t.Fatalf("settings = %q, %q, %v; want %s, %s", gotWS, gotUser, err, wsA, user)
		}
	})

	t.Run("row of another workspace is rejected", func(t *testing.T) {
		err := d.InWorkspace(ctx, wsA, user, func(_ *q.Queries, tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO rls_probe (workspace_id, note) VALUES ($1, 'x')`, wsB)
			return err
		})
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Fatalf("insert into other workspace error = %v, want RLS violation 42501", err)
		}
	})

	t.Run("error from fn rolls back", func(t *testing.T) {
		errStop := errors.New("stop")
		err := d.InWorkspace(ctx, wsA, user, func(_ *q.Queries, tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `INSERT INTO rls_probe (workspace_id, note) VALUES ($1, 'a3')`, wsA); err != nil {
				return err
			}
			return errStop
		})
		if !errors.Is(err, errStop) {
			t.Fatalf("InWorkspace() error = %v, want %v", err, errStop)
		}
		if got := notes(t, d, wsA, user); !slices.Equal(got, []string{"a1", "a2"}) {
			t.Fatalf("notes after rollback = %v, want [a1 a2]", got)
		}
	})

	t.Run("nil workspace is rejected", func(t *testing.T) {
		called := false
		err := d.InWorkspace(ctx, uuid.Nil, user, func(*q.Queries, pgx.Tx) error {
			called = true
			return nil
		})
		if err == nil || called {
			t.Fatalf("InWorkspace(nil) error = %v, fn called %t; want error and no call", err, called)
		}
	})
}

func TestReadyCheck(t *testing.T) {
	ctx := t.Context()
	dbURL := newDatabase(t)
	ready, err := db.NewReadyCheck(newPool(t, withRole(t, dbURL, "msp_app")))
	if err != nil {
		t.Fatal(err)
	}
	owner := connect(t, withRole(t, dbURL, "msp_owner"))

	t.Run("not ready before migrate, without creating the version table", func(t *testing.T) {
		if err := ready(ctx); err == nil {
			t.Fatal("ready() = nil, want an error before msp migrate")
		}
		var exists bool
		if err := owner.QueryRow(ctx, `SELECT EXISTS (SELECT FROM pg_tables WHERE tablename = 'goose_db_version')`).Scan(&exists); err != nil || exists {
			t.Fatalf("goose_db_version exists = %t, %v; want the check to leave it to msp migrate", exists, err)
		}
	})

	if _, err := db.Migrate(ctx, withRole(t, dbURL, "msp_owner")); err != nil {
		t.Fatal(err)
	}

	t.Run("ready after migrate", func(t *testing.T) {
		if err := ready(ctx); err != nil {
			t.Fatalf("ready() = %v, want nil", err)
		}
	})

	t.Run("ready when the database is ahead of the binary", func(t *testing.T) {
		if _, err := owner.Exec(ctx, `INSERT INTO goose_db_version (version_id, is_applied) VALUES (99999999999999, true)`); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = owner.Exec(context.Background(), `DELETE FROM goose_db_version WHERE version_id = 99999999999999`)
		})
		if err := ready(ctx); err != nil {
			t.Fatalf("ready() = %v, want nil after a rollback of the app", err)
		}
	})

	t.Run("not ready when a migration is missing", func(t *testing.T) {
		if _, err := owner.Exec(ctx, `DELETE FROM goose_db_version WHERE version_id > 0`); err != nil {
			t.Fatal(err)
		}
		if err := ready(ctx); err == nil || !strings.Contains(err.Error(), "database at migration 0, binary needs 1") {
			t.Fatalf("ready() = %v, want the missing migration", err)
		}
	})

	t.Run("not ready when the database is unreachable", func(t *testing.T) {
		down, err := db.NewReadyCheck(newPool(t, "postgres://msp_app@127.0.0.1:1/msp?connect_timeout=2"))
		if err != nil {
			t.Fatal(err)
		}
		if err := down(ctx); err == nil {
			t.Fatal("ready() = nil, want an error for an unreachable database")
		}
	})
}

// newDatabase creates an empty database the way ops prepares one (roles from
// db/roles.sql, database owned by msp_owner, PostGIS created by a superuser)
// and returns its superuser URL. The database is dropped when the test ends.
// Without MSP_TEST_DATABASE_URL or PostGIS the test skips locally and fails
// in CI (STD-TEST-03).
func newDatabase(t *testing.T) string {
	t.Helper()
	env := config.LoadTestEnv()
	if env.DatabaseURL == "" {
		if env.CI {
			t.Fatal("MSP_TEST_DATABASE_URL is required in CI: integration tests run on a real Postgres 16 + PostGIS (STD-TEST-03)")
		}
		t.Skip("MSP_TEST_DATABASE_URL is not set: skipping integration test (set it to a superuser URL of a disposable Postgres 16 + PostGIS)")
	}
	ctx := t.Context()
	admin := connect(t, env.DatabaseURL)

	var hasPostGIS bool
	if err := admin.QueryRow(ctx, `SELECT EXISTS (SELECT FROM pg_available_extensions WHERE name = 'postgis')`).Scan(&hasPostGIS); err != nil {
		t.Fatal(err)
	}
	if !hasPostGIS {
		if env.CI {
			t.Fatal("the postgis extension is not available on the test server; CI must run Postgres with PostGIS (STD-TEST-03)")
		}
		t.Skip("the postgis extension is not available on the test server: skipping (the baseline migration needs PostGIS)")
	}

	roles, err := os.ReadFile("../../../db/roles.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, string(roles)); err != nil {
		t.Fatalf("db/roles.sql: %v", err)
	}

	name := "msp_test_" + strings.ToLower(rand.Text())
	ident := pgx.Identifier{name}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+ident+" OWNER msp_owner"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP DATABASE "+ident+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	})

	u, err := url.Parse(env.DatabaseURL)
	if err != nil {
		t.Fatalf("MSP_TEST_DATABASE_URL must be a postgres:// URL: %v", err)
	}
	u.Path = "/" + name
	dbURL := u.String()
	if _, err := connect(t, dbURL).Exec(ctx, `CREATE EXTENSION IF NOT EXISTS postgis`); err != nil {
		t.Fatal(err)
	}
	return dbURL
}

// withRole returns rawURL with the session running as role, which the
// superuser of rawURL may assume; no role password is needed.
func withRole(t *testing.T, rawURL, role string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	query := u.Query()
	query.Set("role", role)
	u.RawQuery = query.Encode()
	return u.String()
}

func newPool(t *testing.T, rawURL string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), rawURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func connect(t *testing.T, rawURL string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(t.Context(), rawURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

func insert(t *testing.T, d *db.DB, ws, user uuid.UUID, notes ...string) {
	t.Helper()
	err := d.InWorkspace(t.Context(), ws, user, func(_ *q.Queries, tx pgx.Tx) error {
		for _, n := range notes {
			if _, err := tx.Exec(t.Context(), `INSERT INTO rls_probe (workspace_id, note) VALUES ($1, $2)`, ws, n); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("insert into %s: %v", ws, err)
	}
}

func notes(t *testing.T, d *db.DB, ws, user uuid.UUID) []string {
	t.Helper()
	var got []string
	err := d.InWorkspace(t.Context(), ws, user, func(_ *q.Queries, tx pgx.Tx) error {
		rows, _ := tx.Query(t.Context(), `SELECT note FROM rls_probe ORDER BY note`)
		var err error
		got, err = pgx.CollectRows(rows, pgx.RowTo[string])
		return err
	})
	if err != nil {
		t.Fatalf("notes in %s: %v", ws, err)
	}
	return got
}

func count(t *testing.T, conn interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) int {
	t.Helper()
	var n int
	if err := conn.QueryRow(t.Context(), `SELECT count(*) FROM rls_probe`).Scan(&n); err != nil {
		t.Fatalf("count rls_probe: %v", err)
	}
	return n
}
