# Postgres 16 + PostGIS 3 + pgBackRest for the app stacks (docs/06-ci-cd.md §5, §8).
# Base: the official postgres image (supported Debian, current 16.x minor), pinned by digest; bump the
# digest for each Postgres minor release. postgis/postgis is not used: its 16 tag is frozen on Debian
# bullseye (EOL) with Postgres 16.9, and PGDG has dropped bullseye. PostGIS and pgBackRest come from the
# image's own PGDG apt repo (trixie-pgdg). Their versions are not pinned: PGDG keeps only recent ones,
# so a pin would break the build once they roll off.
FROM postgres:16.15-trixie@sha256:ca0bd484cb98bf4b24eb1010e73fb3fcbd6714d240fbc1a10eea5b7dbecb641d
RUN apt-get update \
 && apt-get install -y --no-install-recommends postgresql-16-postgis-3 postgresql-16-postgis-3-scripts pgbackrest \
 && rm -rf /var/lib/apt/lists/*
