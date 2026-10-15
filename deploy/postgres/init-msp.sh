#!/usr/bin/env bash
# First init of the app database, run once by the postgres image on an empty data volume, right after
# 10-roles.sql (= apps/api/db/roles.sql, which creates msp_owner and msp_app). Finishes the one-time
# cluster setup of docs/04 §4.4: database msp owned by msp_owner, PostGIS and pg_trgm created by the
# superuser, and both role passwords. The passwords are read from the URLs in .env, so each secret has
# one source: MSP_MIGRATE_DATABASE_URL (msp_owner) and MSP_DATABASE_URL (msp_app).
# Later password changes: docs/09 RB-12. `msp migrate` then runs as msp_owner.
set -euo pipefail

# sql_password URL ROLE -> the URL's password, percent-decoded, as a quoted SQL literal.
# Fails if the URL has no password or its user is not ROLE.
sql_password() {
  perl -e '
    my ($url, $role) = @ARGV;
    $url =~ m{^postgres(?:ql)?://([^:/@]+):([^@]+)@} or die "init-msp: no user:password in the URL for $role\n";
    $1 eq $role or die "init-msp: URL user is $1, expected $role\n";
    (my $pw = $2) =~ s/%([0-9A-Fa-f]{2})/chr hex $1/ge;
    $pw =~ s/\x27/\x27\x27/g;
    print "\x27$pw\x27";
  ' "$1" "$2"
}
owner_pw=$(sql_password "${MSP_MIGRATE_DATABASE_URL:?set MSP_MIGRATE_DATABASE_URL in .env}" msp_owner)
app_pw=$(sql_password "${MSP_DATABASE_URL:?set MSP_DATABASE_URL in .env}" msp_app)

sql() { psql -v ON_ERROR_STOP=1 --username "${POSTGRES_USER:-postgres}" --no-password --no-psqlrc "$@"; }

# Passwords go through stdin, never the command line.
sql --dbname postgres <<SQL
ALTER ROLE msp_owner PASSWORD $owner_pw;
ALTER ROLE msp_app PASSWORD $app_pw;
CREATE DATABASE msp OWNER msp_owner;
SQL

# PostGIS is not a trusted extension: only a superuser can create it (apps/api/db/migrations/00001_baseline.sql).
sql --dbname msp <<'SQL'
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
SQL

# TODO(WP-08): role and database `kratos` for Kratos (docs/04 §9), password from its DSN in .env.
