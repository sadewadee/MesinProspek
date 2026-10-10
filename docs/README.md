# Dokumentasi MesinProspek (MSP)

Dokumentasi produk & engineering untuk **MesinProspek — Sales Intelligence Engine (Sellable MVP)**.

> **Status crawler: PAUSED.** Website Acquisition & Crawler Engine tidak dibangun di MVP. Engine intelligence bersifat source-agnostic dan menerima input dari **dataset yang sudah ada** dan **upload customer** melalui Input Contract. Lihat `00-analisis-dokumen-awal.md` §6 dan D-01.

## Peta dokumen

| No | Dokumen | Untuk siapa | Isi |
|---|---|---|---|
| 00 | [Analisis dokumen awal](00-analisis-dokumen-awal.md) | Founder, Product, Tech lead | Inkonsistensi, gap, dampak pause crawler, register keputusan, pertanyaan terbuka |
| 01 | [PRD](01-prd.md) | Semua | Persona, scope per release, FR/NFR/BR ber-ID, metrik, release gate |
| 02 | [Design system & tokens](02-design-system/README.md) | Design, Frontend | Design tokens (DTCG JSON + CSS variables), pedoman visual, komponen inti |
| 03 | [UX spec](03-ux-spec.md) | Design, Frontend, Product | Information architecture, user flow, spesifikasi layar dan state, copy |
| 04 | [Technical design](04-technical-design.md) | Engineering | Arsitektur, Input Contract, data model, pipeline, scoring, API & event, keamanan |
| 05 | [Test strategy & plan](05-test-strategy.md) | QA, Engineering | Strategi, level test, test case, edge case, traceability, entry/exit criteria |
| 06 | [CI/CD](06-ci-cd.md) | Engineering, DevOps | Branching, pipeline GitLab CI, quality gate, environment, deployment, migrasi, rollback |
| 07 | [Delivery plan](07-delivery-plan.md) | Founder, Product, Tech lead | Roadmap, epic & backlog ber-ID, rencana sprint MVP 0/MVP 1, milestone, DoR/DoD, RACI |
| 08 | [Engineering standards](08-engineering-standards.md) | Engineering | Standar kode Go & Next.js, struktur repo, kontrak API, DB & migrasi, error/logging, keamanan, code review |
| 09 | [Runbook operasional](09-runbook-operasional.md) | DevOps, Ops, Engineering | Deploy & rollback, backup/restore, alert playbook, insiden & notifikasi PDP, operasi data |

**Urutan baca:** 00 → 01 → (02, 03) untuk design/frontend, (04, 05, 06, 08) untuk engineering, 07 untuk perencanaan, 09 untuk operasi.

**Dokumen sumber (di luar repo):** *Sellable MVP Documentation Pack v1.2 — Intelligence Domains* (baseline produk) dan *MSP Website Acquisition & Crawler Engine Spec v1.0* (PAUSED).

## Konvensi ID

| Prefix | Arti | Didefinisikan di |
|---|---|---|
| `D-nn` | Keputusan (Proposed/Accepted/Open) | 00 §7, 04 §ADR |
| `I-nn` / `G-nn` / `Q-nn` | Inkonsistensi / gap / pertanyaan terbuka | 00 |
| `PER-nn` | Persona | 01 §3 |
| `FR-<AREA>-nnn` | Functional requirement. AREA ∈ AUTH, DATA, ICP, SRCH, PROF, INTL, GEO, SEG, EXP, AI, MON, BILL, AUD, OPS, RPT | 01 §8 |
| `NFR-<CAT>-nn` | Non-functional requirement | 01 §10 |
| `BR-<AREA>-nn` | Business rule | 01 §9 |
| `OBJ-n` / `RG-n` | Tujuan produk / release gate | 01 §4, §12 |
| `SCR-nn` / `FLOW-nn` | Layar / user flow | 03 |
| `ADR-nn` | Architecture decision record | 04 |
| `TC-<AREA>-nnn` / `EC-nn` | Test case / edge case data | 05 |
| `QG-nn` | Quality gate CI/CD | 06 |

## Konvensi teknis (Proposed, berlaku untuk semua dokumen)

Ringkasan default dari D-11, D-12, D-14, D-18, D-19, D-21 beserta pilihan tooling. Detail dan alasannya ada di `04-technical-design.md` (ADR) dan `06-ci-cd.md`.

| Area | Pilihan |
|---|---|
| Repo | Monorepo. Host kode + CI + registry: **GitLab CE self-hosted** (D-18). Repo GitHub saat ini dapat di-mirror (Q-08). |
| Frontend | Next.js (App Router) + TypeScript (strict), Tailwind CSS yang membaca design tokens, komponen di atas Radix primitives, TanStack Query. Next.js hanya untuk UI. API Go disajikan di origin yang sama (`/api`) lewat Traefik. Client TS di-generate dari OpenAPI (openapi-typescript + openapi-fetch) |
| Backend | **Go** (D-21; versi stabil terbaru, minimal 1.26). Satu binary `cmd/msp` dengan subcommand `api`, `worker`, `migrate`. `net/http` + **oapi-codegen** (contract-first), **pgx v5 + sqlc**, migrasi **goose**, log `slog` JSON. Modular monolith (D-11) dengan batas modul via `internal/` + `depguard` |
| Worker & antrean | **River** (antrean berbasis Postgres: retry, backoff, unique job, periodic job, enqueue transaksional, sekaligus outbox event/webhook). Redis hanya untuk cache, rate limit, dan counter fair-use |
| Database | PostgreSQL 16+ dengan PostGIS, `pg_trgm`, `unaccent`. Full-text search Postgres (MVP) |
| Storage | S3-compatible: **MinIO** self-hosted (export, upload, raw snapshot) |
| Kontrak | OpenAPI 3.1 (`contracts/openapi/`), JSON Schema untuk event, webhook, dan Input Contract (`contracts/events/`, `contracts/input/`) |
| JS tooling | Node 24 LTS, pnpm workspaces, Turborepo, ESLint, Prettier |
| Go tooling | gofmt/goimports, golangci-lint (termasuk gosec, staticcheck, depguard), govulncheck, sqlc, oapi-codegen. Build `CGO_ENABLED=0` → image distroless/static |
| Test | `go test` (+ `-race`, fuzzing native), rapid (property-based), golden file, Schemathesis, Vitest + Testing Library, Playwright + axe-core, k6 |
| CI/CD | **GitLab CI** (runner Docker self-hosted), GitLab Container Registry. Deploy staging → production via job manual pada tag rilis terproteksi. **Tanpa GitHub** (D-18). |
| Hosting | **Dedicated server** di data center Indonesia (D-19): Docker Compose di belakang Traefik, PostgreSQL + PostGIS self-managed (replika + pgBackRest PITR), Redis, MinIO. Provisioning dengan Ansible. Secret: SOPS + age. |
| Observability | OpenTelemetry → Prometheus/Grafana/Loki/Tempo (self-hosted), log JSON terstruktur, error tracking GlitchTip (kompatibel Sentry SDK) |
| Versioning | Aplikasi SemVer `vX.Y.Z` · API `/v1` · skor `priority-X.Y.Z` · geo `geo-X.Y.Z` · prompt AI `brief-X.Y.Z` · snapshot `ds-YYYY.MM.n` · design tokens SemVer paket |

### Struktur monorepo (target)

```
MesinProspek/
├─ apps/
│  ├─ web/                 Next.js (UI customer + ops console)
│  └─ api/                 Go module: cmd/msp (api|worker|migrate), internal/<modul>, db/migrations, db/queries
├─ packages/
│  ├─ design-tokens/       Sumber: docs/02-design-system/tokens → build CSS/Tailwind/TS
│  ├─ ui/                  Komponen React
│  └─ api-client/          Client TS hasil generate OpenAPI
├─ contracts/
│  ├─ openapi/             openapi.yaml
│  ├─ events/              JSON Schema event & webhook
│  └─ input/               JSON Schema Input Contract
├─ data/
│  ├─ taxonomy/            Taksonomi industri & mapping kategori
│  └─ reference/           Daftar platform domain, webmail, pola telepon
├─ infra/
│  ├─ docker/              docker-compose lokal
│  └─ ansible/             Inventory, role, playbook (staging & production)
├─ docs/
└─ .gitlab-ci.yml          Pipeline CI/CD (GitLab)
```

## Status dokumen

Semua dokumen di folder ini berstatus **Draft v0.1**. Keputusan berlabel *Proposed* menjadi default kerja sampai dikonfirmasi. Ubah statusnya di register keputusan (00 §7) saat keputusan diambil.
