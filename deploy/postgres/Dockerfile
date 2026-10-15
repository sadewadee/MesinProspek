# Postgres 16 + PostGIS + pgBackRest for the app stacks (docs/06-ci-cd.md §5, §8).
# The base tag is rolling, so it is pinned by digest. pgBackRest comes from the image's PGDG apt repo.
FROM postgis/postgis:16-3.5@sha256:94146ac37bc61e2322f88016056c5920729cb8c64c8542ed590af8fc2abdac07
RUN apt-get update \
 && apt-get install -y --no-install-recommends pgbackrest \
 && rm -rf /var/lib/apt/lists/*
