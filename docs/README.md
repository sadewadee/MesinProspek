# Dokumentasi MesinProspek (MSP)

Dokumentasi produk & engineering untuk **MesinProspek — Sales Intelligence Engine**. Versi **0.2 (lean)**.

## Prinsip

1. **KISS.** Pilih solusi paling sederhana yang bekerja. Komponen baru hanya ditambahkan bila ada masalah nyata yang memerlukannya.
2. **DRY.** Satu sumber kebenaran per hal. Dokumen lain merujuk, tidak menyalin ulang.
3. **Make it work → make it right → make it fast.** Fase **F1** (work), **F2** (right), **F3** (fast). Hanya F1 yang dirinci.

Prinsip ini juga tertulis sebagai aturan wajib untuk setiap sesi Claude dan agent di [`CLAUDE.md`](../CLAUDE.md).

Kondisi saat ini:
- **Crawler PAUSED.** Input = dataset JSON-LD yang sudah ada + data wilayah GeoAPI.
- **Pembangun** = agent AI (maks. 8 paralel) diorkestrasi Claude. Founder menyetujui gerbang fase.
- **Isu hukum** ditangani tim legal, di luar dokumen ini.

## Peta dokumen

| No | Dokumen | Isi |
|---|---|---|
| 00 | [Analisis dokumen awal](00-analisis-dokumen-awal.md) | Temuan atas dokumen awal + register keputusan |
| 01 | [PRD](01-prd.md) | **Sumber kebenaran requirement**: scope F1, aturan skor/opportunity/wilayah, credit |
| 02 | [Design system & tokens](02-design-system/README.md) | Brand Forest & Mint, tokens (DTCG + CSS), komponen |
| 03 | [UX spec](03-ux-spec.md) | Layar F1, alur, state, copy |
| 04 | [Technical design](04-technical-design.md) | Arsitektur, data model, pipeline, API |
| 05 | [Test strategy](05-test-strategy.md) | Apa yang diuji, di mana, dan kapan |
| 06 | [CI/CD](06-ci-cd.md) | Pipeline GitLab, deploy ke dedicated server, backup |
| 07 | [Delivery plan](07-delivery-plan.md) | Paket kerja untuk agent AI dan gerbang fase |
| 08 | [Engineering standards](08-engineering-standards.md) | Aturan coding Go & Next.js yang esensial |
| 09 | [Runbook](09-runbook-operasional.md) | Prosedur operasi & alert |

## Stack F1

| Area | Pilihan |
|---|---|
| Repo & CI/CD | Monorepo di **GitLab CE self-hosted** + GitLab CI (D-18). Tanpa GitHub |
| Backend | **Go** (D-21). Satu binary `msp`. `net/http` + **oapi-codegen** (kontrak OpenAPI), **pgx + sqlc**, migrasi **goose**, antrean **River** (di atas Postgres) |
| Frontend | **Next.js** + TypeScript + Tailwind (membaca design tokens) + Radix. Client API di-generate dari OpenAPI |
| Auth | **Ory Kratos** self-hosted (D-20) |
| Database | **PostgreSQL 16 + PostGIS + pg_trgm** |
| File (export) | Disk lokal lewat interface `storage`. S3 di F3 |
| AI | **Claude**, fallback ke API **OpenAI-compatible** (D-15) |
| Wilayah | **GeoAPI** (`geoapi.mcpke.dev`) di-sync ke PostGIS. Peta: vector tiles GeoAPI + basemap OpenStreetMap |
| Hosting | **2 dedicated server** di Indonesia (D-19, D-26): `prod` (aplikasi + DB) dan `ops` (GitLab, runner, staging, monitoring). Docker Compose + Caddy (TLS otomatis) |
| Backup | **pgBackRest** (WAL kontinu) ke object storage S3-compatible off-site di Indonesia |
| Secret | Variabel CI terproteksi GitLab → file `.env` di server (izin 600) |
| Observability | Log JSON (`slog`), endpoint `/metrics` → Prometheus + Grafana + alert (email/Telegram) |

**Sengaja tidak dipakai di F1:** Redis, PgBouncer, object storage internal, server kedua/HA, blue/green deploy, Ansible, SOPS, image signing/SBOM, mutation testing, dan mesin search terpisah. Masing-masing baru ditambahkan bila ada kebutuhan nyata (F2/F3).

## Kosakata teknis bersama

Semua dokumen memakai nama-nama ini persis.

| Hal | Nama |
|---|---|
| Binary & subcommand | `msp api`, `msp worker`, `msp migrate`, `msp tool import <file.jsonl[.gz]>`, `msp tool geo-sync`, `msp tool rescore` |
| Target Makefile (lokal = CI) | `make generate` (oapi-codegen, client TS, sqlc, design tokens), `make lint`, `make test` |
| Format dataset | JSON Lines: satu objek JSON-LD schema.org per baris |
| Antrean River | `import`, `score`, `export`, `webhook`, `ai` |
| Service Compose (prod & staging) | `caddy`, `web`, `api`, `worker`, `kratos`, `postgres` |
| Service Compose (server ops) | `gitlab`, `gitlab-runner`, `prometheus`, `alertmanager`, `grafana` |
| Environment | `local` (laptop, Docker Compose), `staging` (server ops), `production` (server prod) |
| API | Prefix `/api/v1` · health `/healthz` (proses hidup) dan `/readyz` (DB siap) · metrik `/metrics` (internal) |
| Env var | Prefix `MSP_`, mis. `MSP_DATABASE_URL`, `MSP_ANTHROPIC_API_KEY`, `MSP_LLM_FALLBACK_BASE_URL`, `MSP_LLM_FALLBACK_API_KEY`, `MSP_LLM_FALLBACK_MODEL`, `MSP_GEOAPI_BASE_URL`, `MSP_STORAGE_DIR`, `MSP_ENCRYPTION_KEY` |
| Image | `<registry>/msp/api:<git-sha>`, `<registry>/msp/web:<git-sha>`. Rilis = tag Git `vX.Y.Z` |
| Versi model | Skor `priority-X.Y.Z`, wilayah `geo-X.Y.Z`, prompt AI `brief-X.Y.Z` |

## Struktur monorepo

```
MesinProspek/
├─ apps/
│  ├─ api/            Go: cmd/msp, internal/<modul>, db/migrations, db/queries
│  └─ web/            Next.js
├─ packages/
│  ├─ design-tokens/  build dari docs/02-design-system/tokens
│  └─ api-client/     client TS hasil generate OpenAPI
├─ contracts/openapi.yaml
├─ data/              taksonomi kategori, pola URL (sosial, link aggregator, subdomain gratis)
├─ deploy/            docker-compose.*.yml, Caddyfile, skrip bootstrap server
├─ docs/
└─ .gitlab-ci.yml
```

## Konvensi ID

`D-` keputusan · `Q-` pertanyaan terbuka · `FR-` / `BR-` / `NFR-` requirement (PRD) · `TC-` test · `WP-` paket kerja · `RB-` / `AL-` runbook & alert.
