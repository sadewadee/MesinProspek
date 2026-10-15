-- One-time cluster setup for the MesinProspek database (docs/04 §4.4).
--
-- This is an ops step, not a goose migration: roles are cluster-wide, need a
-- role with CREATEROLE, and their passwords are secrets; PostGIS needs a
-- superuser as well. `msp migrate` runs as msp_owner and needs neither.
-- The integration tests run this file too. Safe to run again.
--
-- As a superuser, before the first `msp migrate`:
--   psql -U postgres -v ON_ERROR_STOP=1 -f roles.sql
--   psql -U postgres -c 'CREATE DATABASE msp OWNER msp_owner'
--   psql -U postgres -d msp -c 'CREATE EXTENSION IF NOT EXISTS postgis'
--   psql -U postgres -c '\password msp_owner' -c '\password msp_app'
-- msp_owner goes into MSP_MIGRATE_DATABASE_URL, msp_app into MSP_DATABASE_URL.

DO $$
BEGIN
  -- Owns every schema object; used only by `msp migrate`.
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'msp_owner') THEN
    CREATE ROLE msp_owner LOGIN;
  END IF;
  -- Runtime role of api, worker and tool: not an owner and never bypasses
  -- row-level security.
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'msp_app') THEN
    CREATE ROLE msp_app LOGIN NOBYPASSRLS;
  END IF;
END
$$;
