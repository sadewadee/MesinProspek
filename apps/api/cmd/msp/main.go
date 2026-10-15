// Command msp is the single MesinProspek binary: `msp api`, `msp worker`,
// `msp migrate` and `msp tool <subcommand>` (docs/README.md).
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"msp/internal/platform/config"
	"msp/internal/platform/db"
	"msp/internal/platform/httpx"
)

const usage = `usage: msp <command>

commands:
  api       serve the HTTP API
  worker    run background jobs
  migrate   apply database migrations
  tool      operational tools (import, geo-sync, rescore)
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}

	cfg, err := config.Load(args[0])
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	logger := slog.New(slog.NewJSONHandler(stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	switch args[0] {
	case "api":
		if err := serveAPI(ctx, cfg, logger); err != nil {
			logger.Error("api stopped", "error", err)
			return 1
		}
		return 0
	case "migrate":
		applied, err := db.Migrate(ctx, cfg.MigrateDatabaseURL)
		if err != nil {
			logger.Error("migrate failed", "error", err)
			return 1
		}
		logger.Info("migrations applied", "count", len(applied), "versions", applied)
		return 0
	case "worker":
		// TODO(WP-11): the first job WP registers the River queues of docs/04 §2.
		// Until then the worker idles, so the Compose stacks run it like any
		// long-lived service instead of restarting it.
		logger.Info("worker has no queues yet; waiting for shutdown")
		<-ctx.Done()
		return 0
	case "tool":
		logger.Error("command not implemented yet", "command", args[0])
		return 2
	default:
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}
}

func serveAPI(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	// The pool connects lazily: the api starts while the database is down and
	// /readyz answers 503 until it is up and migrated.
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()
	ready, err := db.NewReadyCheck(pool)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpx.NewRouter(ready, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		logger.Info("api listening", "addr", cfg.HTTPAddr, "env", cfg.Env)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errc; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	logger.Info("api stopped")
	return nil
}
