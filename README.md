# MesinProspek

Market intelligence untuk bisnis lokal Indonesia. Dokumentasi produk dan engineering ada di [`docs/`](docs/README.md); aturan kerja wajib ada di [`CLAUDE.md`](CLAUDE.md).

## Menjalankan

Butuh Go sesuai `apps/api/go.mod` (tool di `Makefile` mengunduh toolchain yang lebih baru sendiri lewat `GOTOOLCHAIN=auto`), Node sesuai `.nvmrc`, dan pnpm sesuai `packageManager` di `package.json`.

```sh
pnpm install
make generate lint test build                               # langkah utama job check di CI
docker compose -f deploy/docker-compose.local.yml up --build  # app http://localhost:8000, Mailpit http://localhost:8025
```

Test integrasi butuh Postgres 16 + PostGIS lewat `MSP_TEST_DATABASE_URL` (tanpa variabel itu, test integrasi di-skip). Detail stack lokal ada di [`deploy/README.md`](deploy/README.md).
