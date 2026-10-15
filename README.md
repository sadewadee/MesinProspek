# MesinProspek

Market intelligence untuk bisnis lokal Indonesia. Dokumentasi produk dan engineering ada di [`docs/`](docs/README.md); aturan kerja wajib ada di [`CLAUDE.md`](CLAUDE.md).

## Menjalankan

Butuh Go sesuai `apps/api/go.mod` (tool di `Makefile` mengunduh toolchain yang lebih baru sendiri lewat `GOTOOLCHAIN=auto`), Node sesuai `.nvmrc`, dan pnpm sesuai `packageManager` di `package.json`.

```sh
pnpm install
make lint test build
MSP_ENV=local apps/api/bin/msp api   # http://localhost:8080/healthz
pnpm --filter @msp/web dev          # http://localhost:3000
```
