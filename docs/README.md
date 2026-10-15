# Dokumentasi MesinProspek (MSP)

Dokumentasi produk & engineering untuk **MesinProspek — Market Intelligence untuk Bisnis Lokal Indonesia**. Versi **0.3 (lean)**.

## Prinsip

1. **KISS.** Pilih solusi paling sederhana yang bekerja. Komponen baru hanya ditambahkan bila ada masalah nyata yang memerlukannya.
2. **DRY.** Satu sumber kebenaran per hal. Dokumen lain merujuk, tidak menyalin ulang.
3. **Make it work → make it right → make it fast.** Fase **F1** (work), **F2** (right), **F3** (fast). Hanya F1 yang dirinci.

Prinsip ini juga tertulis sebagai aturan wajib untuk setiap sesi Claude dan agent di [`CLAUDE.md`](../CLAUDE.md).

Kondisi saat ini:
- **Tanpa scraping; crawler website PAUSED.** Input = DB gmaps: kiriman mingguan ramping dari founder, dimuat ke `gmaps-db` di server kita, lalu dibaca read-only (snapshot berulang, D-31, D-35, D-37) + data wilayah GeoAPI.
- **Pembangun** = agent AI (maks. 8 paralel) diorkestrasi Claude. Founder menyetujui gerbang fase.
- **Cakupan F1** = Indonesia. Dataset global disimpan per negara dan diperbarui berkala.
- **Konfigurasi yang sering berubah** (model AI, plan & harga, parameter estimasi) diatur dari dashboard ops, bukan di kode. Secret tetap di `.env`.
- **Isu hukum** ditangani tim legal, di luar dokumen ini.

## Peta dokumen

| No | Dokumen | Isi |
|---|---|---|
| 00 | [Analisis dokumen awal](00-analisis-dokumen-awal.md) | Temuan atas dokumen awal + register keputusan |
| 01 | [PRD](01-prd.md) | **Sumber kebenaran requirement**: scope F1, rumus estimasi, analisis lokasi & wilayah, skor/opportunity (F2), credit |
| 02 | [Design system & tokens](02-design-system/README.md) | Brand Forest & Mint, tokens (DTCG + CSS), komponen |
| 03 | [UX spec](03-ux-spec.md) | Layar F1, alur, state, copy |
| 04 | [Technical design](04-technical-design.md) | Arsitektur, data model, pipeline, API |
| 05 | [Test strategy](05-test-strategy.md) | Apa yang diuji, di mana, dan kapan |
| 06 | [CI/CD](06-ci-cd.md) | Pipeline GitLab, deploy ke dedicated server, backup |
| 07 | [Delivery plan](07-delivery-plan.md) | Paket kerja untuk agent AI dan gerbang fase |
| 08 | [Engineering standards](08-engineering-standards.md) | Aturan coding Go & Next.js yang esensial |
| 09 | [Runbook](09-runbook-operasional.md) | Prosedur operasi & alert |
| 10 | [Ringkasan bisnis](10-bisnis.md) | Lean canvas, positioning, hipotesis harga, unit economics, rencana beta |

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
| Hosting | **1 dedicated server** di Indonesia untuk semuanya (D-19, D-26): production, staging, GitLab CE + runner, monitoring. Docker Compose + Caddy (TLS otomatis). Batas CPU/RAM per container; runner 1 job sekaligus |
| Backup | **pgBackRest** (WAL kontinu, terenkripsi) ke repo lokal, disalin rclone ke SFTP/FTPS di Indonesia; salinan B2 nonaktif sampai tim legal mengonfirmasi (D-39) |
| Secret | Variabel CI terproteksi GitLab → file `.env` di server (izin 600) |
| Observability | Log JSON (`slog`), endpoint `/metrics` → Prometheus + Grafana + alert (email/Telegram) |

**Sengaja tidak dipakai di F1:** lihat daftar di [`CLAUDE.md`](../CLAUDE.md) § KISS (satu sumber).

## Kosakata teknis bersama

Semua dokumen memakai nama-nama ini persis.

| Hal | Nama |
|---|---|
| Binary & subcommand | `msp api`, `msp worker`, `msp migrate`, `msp tool import --country ID` (baca inkremental dari DB gmaps; juga job harian), `msp tool geo-sync --country ID`, `msp tool rescore` |
| Hitung ulang | Rantai tunggal. F1: estimasi → metrik wilayah. F2 menyambung opportunity → skor (D-40). Otomatis setelah import, setelah geo-sync, dan saat parameter estimasi diubah; di F2 juga saat ICP disimpan (ICP itu saja). `msp tool rescore` manual hanya setelah versi model/bobot berubah |
| Target Makefile (lokal = CI) | `make generate` (oapi-codegen, client TS, sqlc, design tokens), `make lint`, `make test` |
| Sumber data | Kiriman mingguan ramping dari founder (tabel `business_listings` DB gmaps: hanya Indonesia, kolom yang dipakai, baris baru), dimuat ke `gmaps-db` dan dibaca read-only (D-35, D-37). Satu baris = satu snapshot satu tempat; riwayat disimpan di DB MSP. Pemetaan kolom di 04 |
| Antrean River | `import`, `score` (estimasi & metrik wilayah; F2: juga opportunity & skor), `export`, `ai`, `billing` (grant bulanan). `webhook`: **F2** (D-40), nama dicadangkan |
| Service Compose (production & staging) | `web`, `api`, `worker`, `kratos`, `postgres`, dan one-shot `kratos-migrate` (`local` ditambah `caddy`, `mailpit`, dan one-shot `migrate`) |
| Service Compose (ops, di server yang sama) | `caddy` (satu untuk semua domain), `gitlab`, `gitlab-runner`, `prometheus`, `alertmanager`, `grafana`, `node-exporter`, `gmaps-db` (kiriman mingguan DB gmaps terbaru, dibaca read-only oleh `worker` staging dan production). Uptime dipantau layanan eksternal |
| Environment | `local` (laptop, Docker Compose), `staging` dan `production` (keduanya di satu server, project Compose terpisah) |
| Domain | `app.<domain>` (production), `staging.<domain>`, `gitlab.<domain>`, `registry.<domain>` (container registry GitLab), `grafana.<domain>` |
| Direktori Compose di server | `/opt/msp` (production), `/opt/msp-staging` (staging), `/opt/ops` (GitLab, runner, monitoring) |
| Menjalankan tool di server | `docker compose run --rm worker msp <subcommand>` |
| Backup | pgBackRest stanza `msp`, repo lokal `/opt/msp/pgbackrest`. Salinan off-site: cron host `msp-offsite` (`/opt/ops/offsite.sh`, rclone tiap 5 menit) ke remote `offsite` (SFTP/FTPS Indonesia); remote `b2` nonaktif (Q-19) |
| Rute HTTP | API `/api/v1` · Kratos `/.ory/*` · health `/healthz` (proses hidup) dan `/readyz` (DB siap) · metrik `/metrics` di listener internal `:9090`, hanya lewat network Docker (tidak di-route Caddy) |
| Env var | Prefix `MSP_`, mis. `MSP_DATABASE_URL` (role `msp_app`), `MSP_MIGRATE_DATABASE_URL` (role `msp_owner`, hanya `msp migrate`), `MSP_TEST_DATABASE_URL` (superuser khusus test integrasi), `MSP_SOURCE_DATABASE_URL` (kiriman mingguan ramping dari founder, dimuat ke `gmaps-db`; role read-only `msp_source_ro`), `MSP_ANTHROPIC_API_KEY`, `MSP_LLM_FALLBACK_API_KEY`, `MSP_GEOAPI_BASE_URL`, `MSP_STORAGE_DIR`, `MSP_ENCRYPTION_KEY`, `MSP_PUBLIC_URL`, `MSP_KRATOS_PUBLIC_URL`, `MSP_E2E_BASE_URL` dan `MSP_E2E_MAILPIT_URL` (hanya Playwright, `apps/web/playwright.config.ts`). Env Kratos, Postgres, dan pgBackRest memakai nama bawaan masing-masing (04 §14). Daftar lengkap key `.env` server: `deploy/.env.example`. Env hanya untuk secret & pengaturan proses; model AI dan plan ada di DB (setting ops) |
| Image | `<registry>/msp/api:<git-sha>`, `<registry>/msp/web:<git-sha>`. Rilis = tag Git `vX.Y.Z` |
| Versi model | Estimasi `est-X.Y.Z`, analisis lokasi `loc-X.Y.Z`, wilayah `geo-X.Y.Z`, prompt AI `brief-X.Y.Z`; F2: skor `priority-X.Y.Z` |

## Struktur monorepo

```
MesinProspek/
├─ apps/
│  ├─ api/            Go: cmd/msp, internal/<modul>, db/migrations, db/queries
│  └─ web/            Next.js
├─ packages/
│  ├─ design-tokens/  CSS & preset Tailwind dari docs/02-design-system/tokens/build.mjs
│  ├─ ui/             komponen React (Radix + tokens)
│  └─ api-client/     client TS hasil generate OpenAPI
├─ contracts/openapi.yaml
├─ data/              negara (countries.yaml), taksonomi kategori; F2: pola URL (sosial, link aggregator, subdomain gratis), bobot skor
├─ deploy/            docker-compose.*.yml, Caddyfile, skrip bootstrap server
├─ docs/
└─ .gitlab-ci.yml
```

## Konvensi ID

`D-` keputusan · `Q-` pertanyaan terbuka · `FR-` / `BR-` / `NFR-` requirement (PRD) · `TC-` test · `WP-` paket kerja · `RB-` / `AL-` runbook & alert.
