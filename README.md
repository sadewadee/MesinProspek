# MesinProspek

Market intelligence untuk bisnis lokal Indonesia. Dokumentasi produk dan engineering ada di [`docs/`](docs/README.md); aturan kerja wajib ada di [`CLAUDE.md`](CLAUDE.md).

## Menjalankan

Butuh Go 1.24+, Node 22+, dan pnpm 10.

```sh
pnpm install
make lint test build
MSP_ENV=local apps/api/bin/msp api   # http://localhost:8080/healthz
pnpm --filter @msp/web dev          # http://localhost:3000
```
