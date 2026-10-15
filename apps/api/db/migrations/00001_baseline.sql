-- +goose Up
-- Baseline (WP-03). Runs as the schema owner msp_owner after the one-time
-- cluster setup in db/roles.sql (docs/04 §4.4).

-- PostGIS is not a trusted extension, so only a superuser can create it.
-- db/roles.sql has ops create it first; then this line is a no-op.
CREATE EXTENSION IF NOT EXISTS postgis;
-- pg_trgm is trusted: the database owner creates it.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- The runtime role msp_app is neither owner nor BYPASSRLS (docs/04 §4.4).
-- It may read and write every table the migrations create; a migration
-- REVOKEs what msp_app must not do (e.g. UPDATE/DELETE on audit_log), and
-- row-level security limits workspace tables to the current workspace.
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO msp_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO msp_app;

-- /readyz runs as msp_app and checks the applied migration version (docs/04 §14).
GRANT SELECT ON goose_db_version TO msp_app;
