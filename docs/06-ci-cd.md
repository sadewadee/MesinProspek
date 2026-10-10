# 06 — CI/CD

## MesinProspek — Branching, Pipeline, Quality Gate, Environment, Rilis

| Field | Value |
|---|---|
| Dokumen | CI/CD |
| Versi | 0.1 (Draft) |
| Tanggal | 2026-10-10 |
| Basis | `README.md` (konvensi teknis & struktur monorepo), `00-analisis-dokumen-awal.md` (D-11, D-12, D-14), `01-prd.md` (§10 NFR, §12 release gate & DoD), MVP Pack v1.2 §11, §14, §15 |
| Terkait | `04-technical-design.md` (arsitektur, data model, ADR), `05-test-strategy.md` (isi dan level test suite) |
| Audience | Engineering, DevOps/Platform, QA, Tech lead |
| Menutup | G-17 (strategi CI/CD, environment, migrasi DB) |

> Konvensi: `QG-nn` = quality gate (didefinisikan di §8). Semua pilihan tooling berstatus **Proposed** sesuai README. Contoh YAML di dokumen ini **ilustratif**, bukan file final: file nyata di `.github/workflows/` dibuat saat bootstrap (§18). Jam ditulis dalam WIB (UTC+7). Ekspresi cron selalu UTC. Path modul backend (mis. `msp/identity/resolve`) adalah asumsi kerja; nama final ditetapkan di `04-technical-design.md`.

---

## 1. Prinsip

| # | Prinsip | Konsekuensi konkret |
|---|---|---|
| P-1 | **Trunk-based development** | Satu branch jangka panjang: `main`. Branch fitur berumur pendek (target ≤ 2 hari, maksimal 5). Pekerjaan yang belum selesai masuk `main` di balik feature flag (§13), bukan di branch panjang. |
| P-2 | **Build once, deploy many** | Image dibangun **sekali** per commit di `main`. Staging dan production menjalankan **image digest yang sama** (`sha256:…`). Tidak ada rebuild per environment. Konfigurasi environment masuk saat runtime (§12). |
| P-3 | **Everything as code** | Pipeline (`.github/workflows/`), infrastruktur (`infra/terraform/`), kebijakan repo (CODEOWNERS, ruleset, Renovate), skema DB (Alembic), kontrak (`contracts/`), design tokens, konfigurasi versi skor/geo/prompt. Perubahan manual di console AWS dianggap drift (§11.5). |
| P-4 | **Fast feedback** | Target pipeline PR **≤ 10 menit p50** (p90 ≤ 15 menit) lewat path filtering, Turborepo `--affected`, cache uv/pnpm/Docker layer, dan job paralel. Test lambat dipindah ke merge queue, post-deploy, atau nightly. |
| P-5 | **Shift-left security** | Secret scanning, SAST, dependency review, license policy, dan scan IaC berjalan di PR. Image yang tidak lolos scan tidak pernah ditandatangani, sehingga tidak bisa di-deploy (§5, §15). |
| P-6 | **Rilis data & model yang reproducible** | Snapshot dataset (`ds-YYYY.MM.n`), model skor (`priority-X.Y.Z`), geo (`geo-X.Y.Z`), dan prompt (`brief-X.Y.Z`) dirilis lewat pipeline dengan gate dan approval, terpisah dari rilis kode. Versi yang sudah dirilis immutable (NFR-DATA-01, FR-INTL-009). |
| P-7 | **Data tetap di region** | Runner GitHub-hosted berada di luar Indonesia. Runner hanya memproses kode dan **data sintetis**. Dataset berlisensi, golden set, dan data production tidak pernah diunduh ke runner. Job yang butuh data berjalan sebagai ECS task di ap-southeast-3, dan CI hanya menerima laporan agregat (D-14, NFR-PRIV-03, BR-PRIV-04). |
| P-8 | **Rollback murah, forward-fix untuk data** | Rollback aplikasi = deploy ulang digest sebelumnya (≤ 15 menit). Skema DB tidak di-rollback: migrasi selalu kompatibel dengan versi aplikasi sebelumnya (expand/contract, §9). |

### 1.1 Alur besar

```
 feat/* fix/* chore/* docs/*
        │  PR
        ▼
 ┌───────────────────────────┐    review CODEOWNERS     ┌─────────────────────────────┐
 │ pr.yml (path-filtered)    │ ───────────────────────► │ merge queue (pr.yml pada    │
 │ lint·type·test·migrasi·   │                          │ merge_group, commit gabung) │
 │ kontrak·tokens·a11y·E2E   │                          └──────────────┬──────────────┘
 │ smoke·security → ci-ok    │                                         │ squash merge
 └───────────────────────────┘                                         ▼
                                                                     main
                                                                       │ push
                                                                       ▼
 ┌──────────────────────────────────────────────────────────────────────────────────────┐
 │ main.yml: build msp-api & msp-web (push by digest) → Trivy gate → SBOM → cosign sign │
 │        → deploy staging (migrate → worker → api → web) → verifikasi staging          │
 │        → commit status `staging-verified`                                            │
 └──────────────────────────────────────────────────────────────────────────────────────┘
                                                                       │
   release-please PR ──merge──► tag vX.Y.Z + GitHub Release ──────────►│
                                                                       ▼
 ┌──────────────────────────────────────────────────────────────────────────────────────┐
 │ release.yml: resolve digest commit tag → readiness gate → approval (Env production)  │
 │        → verify signature → migrate → canary + bake (auto-rollback) → verifikasi     │
 └──────────────────────────────────────────────────────────────────────────────────────┘

 Paralel: nightly.yml (regresi penuh, k6, ZAP, data quality, drift), infra.yml (Terraform),
          data-release.yml / model-release.yml (§10), rollback.yml (§14)
```

### 1.2 Pilihan tooling tambahan (di luar README, Proposed)

| Area | Pilihan | Alasan singkat | Alternatif yang ditolak |
|---|---|---|---|
| Rilis & changelog | **release-please** (manifest mode) | Repo poliglot (Python + TS). Konvensi sudah Conventional Commits. Release PR memberi titik review yang eksplisit. Satu versi aplikasi untuk seluruh deployable. | changesets (berorientasi paket npm, butuh file changeset manual per PR) |
| Lint OpenAPI | **Spectral** (+ `spectral-owasp-ruleset`) | Ruleset kustom berbasis YAML mudah ditulis (mis. wajib `Idempotency-Key` pada operasi berbayar). Ada ruleset OWASP API. | Redocly CLI (bagus, tetapi aturan kustomnya lebih terbatas) |
| Breaking change API | **oasdiff** | Mendukung OpenAPI 3.1, output level ERR/WARN, mudah di-diff terhadap base. | openapi-diff |
| Lint SQL migrasi | **squawk** | Fokus ke operasi berbahaya di PostgreSQL (lock, index non-concurrent, rename/drop). | Manual review saja |
| Validasi JSON Schema | **check-jsonschema** | CLI Python (jalan via `uvx`), mendukung draft 2020-12, juga memvalidasi workflow GitHub dan config Renovate. | ajv-cli |
| Client TS | **openapi-typescript** (+ `openapi-fetch`) | Ringan, type-only, cocok dipadukan dengan TanStack Query. Final di 04. | orval |
| Workflow lint | **actionlint** + **zizmor** | Sintaks dan keamanan GitHub Actions (injection, permission berlebih, pin SHA). | — |
| Docs lint | **markdownlint-cli2** + **lychee** | Cepat, link check internal bisa offline. | — |
| Supply chain | **syft** (SBOM SPDX), **cosign** keyless, provenance BuildKit (`mode=max`) | Sesuai kebutuhan SBOM + signing. Provenance BuildKit tidak bergantung plan GitHub (lihat §15.3). | Docker Content Trust |
| Registry | **Amazon ECR** di akun `shared`, ap-southeast-3, tag immutable | Satu region dengan runtime, IAM via OIDC, pull lintas akun ke staging/production. | GHCR (di luar region, auth terpisah) |
| Deploy ECS | ECS deployment **canary/blue-green bawaan** + alarm CloudWatch | Tanpa CodeDeploy. Bake time dan auto-rollback berbasis alarm. | CodeDeploy, rolling saja |
| State Terraform | Backend S3 dengan **native lockfile** (`use_lockfile`) | Tanpa tabel DynamoDB. | S3 + DynamoDB lock |
| Dependency update | **Renovate self-hosted** (GitHub Action + GitHub App token) | Tidak memberi akses kode ke pihak ketiga. Grouping dan automerge lebih fleksibel daripada Dependabot. | Mend-hosted Renovate, Dependabot |
| Paralelisasi Python | **pytest-xdist** | Memenuhi target ≤ 10 menit. | — |
| Feature flag | Flag store di Postgres + registry di repo, antarmuka tipis kompatibel OpenFeature | Audit wajib (FR-OPS-007), tanpa vendor tambahan di MVP. | LaunchDarkly/Unleash (biaya, residensi) |
| Web performance | **Lighthouse CI** (nightly, staging) | Proxy lab untuk NFR-PERF-03. Nilai p75 sesungguhnya dari RUM. | — |

---

## 2. Branching & PR workflow

### 2.1 Model branch

| Branch | Fungsi | Umur | Dibuat oleh |
|---|---|---|---|
| `main` | Trunk. Selalu dapat dirilis (fitur belum siap disembunyikan flag). | Permanen | — |
| `feat/<scope>-<deskripsi>` | Fitur baru. Contoh `feat/srch-zero-result-guidance` | ≤ 2 hari (maks 5) | Engineer |
| `fix/<scope>-<deskripsi>` | Perbaikan bug | ≤ 2 hari | Engineer |
| `chore/<deskripsi>` | Maintenance, refactor, build, CI | ≤ 2 hari | Engineer |
| `docs/<deskripsi>` | Dokumentasi, termasuk design tokens yang hanya berupa dokumentasi | ≤ 2 hari | Engineer/Design/Product |
| `release-please--branches--main` | Release PR otomatis | Sampai rilis | Bot release-please |
| `renovate/*` | Update dependency | Sampai merge | Bot Renovate |
| `release/vX.Y` | **Hanya** untuk hotfix saat `main` tidak bisa dirilis (§6.6) | Sampai hotfix rilis | Tech lead |

Aturan: tidak ada branch `develop`. Fitur yang butuh lebih dari 5 hari dipecah menjadi beberapa PR kecil di balik flag `rel.*` (§13). Branch dihapus otomatis setelah merge.

### 2.2 Conventional Commits

Merge memakai **squash**, sehingga **judul PR menjadi pesan commit** di `main`. Judul PR divalidasi di CI (QG-01).

```
<type>(<scope>)!: <ringkasan imperatif, huruf kecil>

type  : feat | fix | perf | refactor | test | docs | chore | ci | build | revert
scope : web | api | worker | identity | intelligence | query | action | usage | ops |
        contracts | tokens | ui | infra | data | docs | deps
!     : breaking change (wajib disertai footer "BREAKING CHANGE: ..." di body PR)
```

| Contoh judul | Efek di release-please |
|---|---|
| `feat(intelligence): tambah faktor geo pada priority score` | minor |
| `fix(action): retry webhook memakai backoff eksponensial` | patch |
| `feat(contracts)!: ganti format cursor pagination /v1/prospects` | major (dan wajib label `api-breaking`, §2.7) |
| `chore(deps): update ruff ke 0.x` | tidak masuk changelog |

Body PR yang menyertakan ID requirement (mis. `Refs: FR-SRCH-008, TC-SRCH-012`) dibawa ke changelog.

### 2.3 Review & CODEOWNERS

- Minimal **1 approval** dari CODEOWNERS area yang tersentuh. Path berisiko tinggi (lihat tabel) butuh **2 approval** saat tim ≥ 4 engineer. Sebelum itu cukup 1 approval + checklist keamanan di PR template.
- Approval lama gugur saat ada push baru (*dismiss stale approvals*), dan push terakhir harus disetujui orang lain (*require approval of the most recent push*).
- Penulis PR tidak boleh menyetujui PR-nya sendiri, termasuk PR bot yang dipicu penulis.

```
# .github/CODEOWNERS (ilustratif). Pola terakhir yang cocok menang, jadi urutan penting.
*                                         @sadewadee/backend @sadewadee/web
/apps/web/                                @sadewadee/web
/packages/ui/                             @sadewadee/web @sadewadee/design
/packages/design-tokens/                  @sadewadee/design @sadewadee/web
/docs/02-design-system/                   @sadewadee/design
/apps/api/                                @sadewadee/backend
/apps/api/src/msp/identity/               @sadewadee/data-intel
/apps/api/src/msp/intelligence/           @sadewadee/data-intel
/apps/api/src/msp/ai/prompts/             @sadewadee/data-intel @sadewadee/product
/apps/api/src/msp/usage/                  @sadewadee/backend          # billing & ledger
/apps/api/src/msp/auth/                   @sadewadee/security
/apps/api/migrations/                     @sadewadee/backend @sadewadee/platform
/contracts/                               @sadewadee/api-owners
/data/                                    @sadewadee/data-intel
/infra/                                   @sadewadee/platform
/.github/                                 @sadewadee/platform
/docs/                                    @sadewadee/product
/docs/04-technical-design.md              @sadewadee/backend
/docs/05-test-strategy.md                 @sadewadee/qa
/docs/06-ci-cd.md                         @sadewadee/platform
```

| Path berisiko tinggi (2 approval saat tim ≥ 4) | Alasan |
|---|---|
| `apps/api/migrations/**` dengan label `db-contract` | Operasi destruktif (§9) |
| `apps/api/src/msp/auth/**`, kebijakan RLS | Isolasi tenant (NFR-SEC-03) |
| `infra/terraform/envs/production/**` | Infrastruktur production |
| `.github/workflows/**` | Supply chain (§15) |
| `contracts/**` dengan label `api-breaking` | Kontrak publik |

> Catatan: tim (`@org/team`) hanya tersedia bila repo dimiliki **GitHub Organization**. Bila `sadewadee/MesinProspek` masih berada di akun personal, tim diganti username, dan beberapa fitur (merge queue, required reviewers environment) tidak tersedia. Lihat §18 langkah 1 dan Q-CI-01.

### 2.4 PR template

```markdown
<!-- .github/pull_request_template.md (ilustratif) -->
## Ringkasan
<!-- Apa dan kenapa. Judul PR mengikuti Conventional Commits. -->

## Requirement
Refs: FR-___ / NFR-___ / BR-___      Test: TC-___ (05-test-strategy.md)

## Checklist
- [ ] Test ditambah/diubah (unit/integration/E2E sesuai level di 05)
- [ ] Tidak ada data pribadi nyata di test, fixture, log, atau screenshot
- [ ] Migrasi DB: fase `expand` / `contract` dicatat. Kompatibel dengan versi aplikasi sebelumnya (§9)
- [ ] Tabel tenant baru memiliki RLS policy
- [ ] Perubahan kontrak: OpenAPI/JSON Schema diperbarui dan client TS di-regenerate
- [ ] Breaking API? Label `api-breaking` + bagian "Rencana versi API" di bawah
- [ ] Feature flag: nama, tipe, owner, tanggal kedaluwarsa terdaftar di registry (§13)
- [ ] Perubahan skor/geo/prompt: versi dinaikkan, laporan regresi/eval terlampir (§10)
- [ ] UI: screenshot/video sebelum-sesudah (desktop 1280 px + mobile 360 px), copy id-ID & en-US
- [ ] Event analytics & audit dipasang (bila relevan)
- [ ] Runbook/dokumentasi diperbarui (bila perilaku operasional berubah)

## Catatan migrasi / rollout
<!-- Urutan deploy khusus, backfill, flag yang harus dinyalakan, dsb. -->

## Rencana versi API (wajib bila `api-breaking`)
<!-- Endpoint terdampak, periode deprecation, konsumen terdampak. -->
```

### 2.5 Branch protection (repository ruleset) & required checks

Ruleset untuk `main` (dikelola sebagai kode lewat Terraform provider `github` atau script bootstrap, §18):

| Aturan | Nilai |
|---|---|
| Wajib pull request | Ya. Tanpa bypass, kecuali break-glass oleh admin repo (tercatat) |
| Required approvals | 1 (2 untuk path berisiko saat tim ≥ 4), CODEOWNERS wajib, dismiss stale, approval push terakhir |
| Conversation resolution | Wajib |
| Required status checks | `ci-ok` (agregator `pr.yml`), `CodeQL` (code scanning: blokir alert severity ≥ High), `pr-title` |
| Wajib merge queue | Ya (§2.6) |
| Metode merge | Squash saja. Linear history |
| Force push, hapus branch | Diblokir |
| Ruleset tag `v*` dan `design-tokens-v*` | Hanya GitHub App release (release-please) yang boleh membuat. Update/hapus diblokir untuk semua |

**Kenapa satu agregator `ci-ok`:** job yang di-skip karena path filter dihitung lulus, tetapi workflow yang tidak terpicu sama sekali membuat required check menggantung. Karena itu path filter diterapkan di level **job** (bukan `on.paths`), lalu job `ci-ok` (`if: always()`) gagal bila ada job dependensi yang `failure` atau `cancelled`. Hasilnya, daftar required check tetap stabil walaupun job bertambah.

### 2.6 Merge queue

- Dipakai **GitHub merge queue**. `pr.yml` juga terpicu oleh event `merge_group`, sehingga check dijalankan ulang pada commit gabungan (base terbaru + PR). Ini mencegah `main` merah akibat dua PR yang masing-masing hijau tetapi konflik secara semantik.
- Konfigurasi: build concurrency 3, batas grup 3 PR, timeout check 30 menit, merge method squash.
- Pada `merge_group`, path filter dihitung terhadap `merge_group.base_sha`. Label PR tidak tersedia di payload event ini, jadi job yang bergantung pada label (`api-breaking`, `db-contract`) mengambil label lewat API dari nomor PR yang ada di `merge_group.head_ref` (`gh-readonly-queue/main/pr-<n>-<sha>`).
- Bila plan GitHub tidak menyediakan merge queue untuk repo privat, fallback: *require branches to be up to date* + auto-merge. Konsekuensinya PR harus di-rebase berulang (Q-CI-01).

### 2.7 Label yang dibaca pipeline

| Label | Efek |
|---|---|
| `api-breaking` | Mengizinkan breaking change OpenAPI (QG-12), selama bagian "Rencana versi API" terisi. Changelog ditandai. |
| `db-contract` | Mengizinkan migrasi fase contract (drop/rename) (QG-10). Wajib 2 approval saat tim ≥ 4. |
| `ai-eval` | Memaksa AI eval berjalan walau path prompt tidak berubah. Contoh: ganti model provider (D-15). |
| `e2e-full` | Menjalankan E2E penuh di PR, bukan hanya smoke. |
| `skip-e2e-smoke` | **Tidak ada.** Smoke tidak bisa di-skip, karena tujuannya mencegah kebiasaan melewati gate. |

### 2.8 Release tagging

- Tag `vX.Y.Z` dan GitHub Release **hanya** dibuat oleh release-please saat Release PR di-merge (§6.1). Tag dilindungi ruleset.
- Komponen kedua di manifest: `packages/design-tokens` → tag `design-tokens-vX.Y.Z` (SemVer paket, README).
- Versi data/model (`ds-*`, `priority-*`, `geo-*`, `brief-*`) **bukan** git tag. Versi ini dicatat di registry versi (DB + manifest S3 + file di repo) dan dirilis lewat pipeline §10.
- Seri 0.x dipakai selama MVP 0 dan closed beta. `v1.0.0` = rilis MVP 1 yang lolos RG-1 s.d. RG-7 (QG-31).

---

## 3. Environments

### 3.1 Ringkasan

| Env | Tujuan | Data | Trigger deploy | Akses | URL | Sumber config |
|---|---|---|---|---|---|---|
| **local** | Pengembangan harian | Seed **sintetis** (`msp.devtools.seed`), fixture edge case EC-xx dari 05. Tanpa data production atau berlisensi. | Manual: `docker compose up` | Engineer | `http://localhost:3000` (web), `:8000` (api), `:8025` (Mailpit), `:9001` (MinIO console) | `.env` dari `.env.example` |
| **CI** | Verifikasi PR/merge queue | Sintetis, dibuat per run, dibuang setelahnya | Otomatis per PR/`merge_group` | Pipeline saja | — (service container ephemeral) | `env:` workflow + service containers |
| **preview** | — | — | **Ditunda** (§3.2) | — | — | — |
| **staging** | Prod-like: integrasi, E2E penuh, DAST, load test, rehearsal rilis data/model, drill | Sintetis + **subset dataset berlisensi** (bila lisensi mengizinkan, Q-05). **Tanpa PII customer**, tanpa upload customer, tanpa salinan DB production. | Otomatis setiap merge ke `main` (`main.yml`) | Engineer: baca (console SSO, log). Deploy hanya via pipeline. ECS Exec diizinkan untuk debugging. | `https://app.staging.<domain>`, `https://api.staging.<domain>` | Terraform + SSM Parameter Store + Secrets Manager (akun staging) |
| **production** | Customer | Data penuh: dataset berlisensi, data tenant, upload customer | Tag `vX.Y.Z` → `release.yml` + approval Environment `production` | Tanpa akses manusia permanen. Break-glass via IAM Identity Center (MFA, sesi ≤ 1 jam, alert saat dipakai). ECS Exec dinonaktifkan. | `https://app.<domain>`, `https://api.<domain>` | Terraform + SSM + Secrets Manager (akun production) |

Prinsip parity: versi PostgreSQL/PostGIS, Redis, Python, dan Node di local/CI **sama** dengan staging/production (image di-pin digest, versi engine RDS di Terraform). Satu script `scripts/ci/check_version_parity.py` membandingkan keduanya di PR yang menyentuh `infra/**`.

### 3.2 Keputusan: preview environment ditunda

**Keputusan (Proposed):** MVP **tidak** memakai preview environment per PR, baik full-stack maupun web-only. Sebagai gantinya:

1. Build Storybook diunggah sebagai artifact PR (review komponen dan state), ditambah screenshot wajib di PR template untuk perubahan UI.
2. Stack lengkap di CI (docker compose) untuk E2E smoke.
3. Staging diperbarui setiap merge (≤ 30 menit setelah merge) untuk review produk.

**Alasan:**
- *Full-stack per PR* (RDS + Redis + ECS per PR) mahal dan lambat dibuat. Seed data juga harus sintetis, sehingga nilainya tidak jauh berbeda dari stack CI.
- *Web-only preview yang memanggil API staging* menjadikan staging dependensi bersama yang rapuh (CORS, callback auth, versi API PR yang belum di-deploy). Pola ini juga membuka akses ke data berlisensi staging dari URL preview, padahal hosting Next.js di AWS untuk preview butuh infrastruktur tambahan (OpenNext/Amplify).
- Tim MVP kecil, sehingga bottleneck review desain belum ada.

**Ditinjau ulang bila:** tim ≥ 6 engineer, atau review desain/produk menjadi bottleneck (PR UI menunggu staging > 1 hari secara rutin). Opsi saat itu: web-only preview dengan API mock (MSW) atau namespace staging per PR.

### 3.3 Aturan data per environment

| Aturan | local | CI | staging | production |
|---|:-:|:-:|:-:|:-:|
| Data sintetis | ✓ | ✓ | ✓ | — |
| Subset dataset berlisensi | — | — | ✓ (in-region) | ✓ |
| Data/upload customer | — | — | — | ✓ |
| Restore DB production | — | — | — | Hanya drill di akun production (§7) |
| Auth stub (`MSP_AUTH_MODE=stub`, JWT uji) | ✓ | ✓ | — | — |
| Flag fault-injection `chaos.*` | ✓ | ✓ | ✓ | Ditolak saat startup |

Aplikasi **menolak start** bila `MSP_ENV ∈ {staging, production}` dan `MSP_AUTH_MODE=stub` atau flag `chaos.*` aktif. Guard ini diuji di unit test.

### 3.4 Local dev (docker compose)

`infra/docker/compose.yml`:

| Service | Image (pin digest) | Catatan |
|---|---|---|
| `postgres` | `postgis/postgis:16-3.5` | Ekstensi `postgis`, `pg_trgm`, `unaccent` dibuat oleh migrasi pertama (§9.6) |
| `redis` | `redis:7-alpine` | Broker Dramatiq |
| `minio` | `minio/minio` | S3-compatible: bucket `msp-exports`, `msp-uploads`, `msp-snapshots` dibuat oleh init container. Bila image MinIO tidak lagi tersedia, ganti dengan alternatif S3-compatible (LocalStack/SeaweedFS) tanpa mengubah kode. |
| `mailpit` | `axllent/mailpit` | Email undangan/verifikasi |
| `migrate` | build `apps/api` | `alembic upgrade head`, lalu exit. Service lain `depends_on: service_completed_successfully` |
| `api` | build `apps/api` | `uvicorn msp.api.main:app --reload` |
| `worker` | build `apps/api` | `dramatiq msp.worker` |
| `web` | build `apps/web` | `next dev` |
| `webhook-sink` | image kecil echo server | Penerima webhook uji (E2E FR-EXP-003) |

`compose.ci.yml` meng-override: tanpa volume bind, build memakai cache GHA, `api`/`web` dalam mode production build, healthcheck ketat untuk `--wait`.

### 3.5 Akun AWS & isolasi

```
AWS Organization
├─ msp-shared       ECR (msp-api, msp-web), OIDC role push image, bucket evidence rilis (SBOM, laporan)
├─ msp-staging      VPC, ECS cluster msp-staging, RDS, ElastiCache, S3, Secrets Manager
└─ msp-production   VPC, ECS cluster msp-production, RDS (Multi-AZ, PITR), ElastiCache, S3, Secrets Manager
```

Semua di **ap-southeast-3 (Jakarta)** (D-14). Pemisahan akun membatasi blast radius dan membuat IAM per environment lebih sederhana. Staging dan production menarik image dari ECR `shared` lewat izin lintas akun (read-only).

---

## 4. Pipeline PR (`pr.yml`)

Trigger: `pull_request` (opened, synchronize, reopened, ready_for_review, labeled, unlabeled) dan `merge_group`. Run sebelumnya pada PR yang sama dibatalkan (`concurrency` + `cancel-in-progress`). PR draft hanya menjalankan lint dan unit test.

### 4.1 Path filter (job `changes`)

| Output | Path | Job yang dipicu |
|---|---|---|
| `backend` | `apps/api/**`, `data/**`, `pyproject.toml`, `uv.lock` | backend-lint, backend-tests |
| `migrations` | `apps/api/migrations/**`, model SQLAlchemy (`apps/api/src/msp/**/models*.py`) | migrations |
| `web` | `apps/web/**`, `packages/**`, `pnpm-lock.yaml`, `turbo.json` | web-lint, web-tests, web-build |
| `contracts` | `contracts/**`, router/schema API (`apps/api/src/msp/**/api/**`), `packages/api-client/**` | contracts |
| `tokens` | `docs/02-design-system/tokens/**`, `packages/design-tokens/**` | tokens |
| `e2e` | `apps/**`, `packages/**`, `contracts/**`, `infra/docker/**` | e2e-smoke |
| `scoring` | konfigurasi versi skor & geo (`apps/api/src/msp/intelligence/**/versions/**`) | model-regression |
| `prompts` | `apps/api/src/msp/ai/prompts/**`, `apps/api/src/msp/ai/eval/**` | ai-eval |
| `er` | `apps/api/src/msp/identity/resolve/**`, `apps/api/src/msp/identity/normalize/**` | er-golden-synthetic |
| `infra` | `infra/terraform/**` | `infra.yml` (workflow terpisah, §11) |
| `docs` | `docs/**`, `**/*.md` | docs |
| `workflows` | `.github/**`, `renovate.json` | workflow-lint |

Job keamanan (gitleaks, Trivy fs, dependency review) **selalu** berjalan.

### 4.2 Job

| Job | Isi | Tool | Gate |
|---|---|---|---|
| `pr-title` | Validasi Conventional Commits pada judul PR | `amannn/action-semantic-pull-request` | QG-01 |
| `backend-lint` | `ruff check`, `ruff format --check`, `mypy` (strict untuk `msp`) | uv, ruff, mypy | QG-02, QG-03 |
| `web-lint` | ESLint, `prettier --check`, `tsc --noEmit` untuk paket yang terdampak | pnpm, Turborepo `--affected` | QG-02, QG-03 |
| `backend-tests` | Service container PostGIS + Redis → `alembic upgrade head` → pytest unit + integration (`-n auto`) → **tenant isolation suite** (`-m tenant_isolation`, dengan koneksi role aplikasi tanpa `BYPASSRLS`) → test redaksi PII (log & prompt) → coverage gate | pytest, Hypothesis (profil `ci`), pytest-xdist, coverage | QG-04, QG-05, QG-06, QG-08, QG-09 |
| `web-tests` | Vitest + Testing Library (paket terdampak) dengan coverage threshold | Vitest | QG-04, QG-07 |
| `migrations` | (1) satu head Alembic; (2) `upgrade head` dari DB kosong; (3) `alembic check`: tidak ada perubahan model yang belum dimigrasi; (4) render SQL migrasi baru (offline `--sql`) → **squawk**; (5) cek RLS: setiap tabel tenant-layer (punya `workspace_id`) memiliki `ENABLE` + `FORCE ROW LEVEL SECURITY` dan ≥ 1 policy (`USING` + `WITH CHECK`), pengecualian lewat allowlist ber-CODEOWNER security; (6) fase migrasi: operasi contract hanya dengan label `db-contract` | Alembic, squawk, script `check_rls.py`, `check_migration_phase.py` | QG-10 |
| `contracts` | (1) spec ter-commit `contracts/openapi/openapi.yaml` **identik** dengan spec yang dilayani aplikasi (export dari FastAPI); (2) Spectral lint; (3) **oasdiff** breaking vs base → gagal kecuali label `api-breaking` + rencana versi; (4) regenerate client TS → `git diff --exit-code packages/api-client`; (5) `check-jsonschema --check-metaschema` untuk `contracts/events/**` dan `contracts/input/**`, lalu validasi semua contoh payload (`examples/`); (6) file skema event/webhook versi terbit (`*.v<N>.schema.json`) immutable; (7) Schemathesis smoke terhadap API ephemeral (service container, `--max-examples` kecil, check `not_a_server_error` + kesesuaian response schema) | Spectral, oasdiff, openapi-typescript, check-jsonschema, Schemathesis | QG-11, QG-12, QG-13 |
| `tokens` | Validasi `docs/02-design-system/tokens/design-tokens.json` (struktur DTCG + resolusi alias, gagal pada referensi putus) → build Style Dictionary → `git diff --exit-code` pada output ter-commit (CSS variables, Tailwind preset, TS) → script kontras: setiap pasangan di `contrast-pairs.json` ≥ 4,5:1 (teks) / 3:1 (UI & teks besar), mode light & dark | Style Dictionary, script `check-contrast.ts` | QG-14 |
| `web-build` | `next build` (output standalone) + build Storybook → test a11y komponen (axe-core pada setiap story; 0 pelanggaran serious/critical) → upload Storybook statis sebagai artifact (retensi 7 hari) | Next.js, Storybook, axe-core | QG-15 |
| `e2e-smoke` | `docker compose -f compose.yml -f compose.ci.yml up --wait` → seed sintetis → Playwright `@smoke` (chromium, 2 shard) + axe di halaman yang dikunjungi. Cakupan *critical path*: login (stub) → onboarding Offering → ICP → hasil pertama; search + filter + zero-result; profil + evidence drawer; unlock (debit credit); export kecil; webhook test ke `webhook-sink`; kill switch AI menampilkan pesan nonaktif. Detail skenario di 05. | Playwright, @axe-core/playwright | QG-15, QG-16 |
| `er-golden-synthetic` | Bila kode normalisasi/ER berubah: evaluasi pada golden set **sintetis** di repo → precision/recall dilaporkan di PR | pytest | QG-25 |
| `model-regression` | Bila konfigurasi skor/geo berubah: cek versi naik + versi terbit tidak berubah (`versions.lock`), recompute fixture "golden scores" → diff dilaporkan sebagai komentar PR | pytest, script | QG-22, QG-23 |
| `ai-eval` | Bila prompt berubah atau label `ai-eval`: jalankan eval set sintetis (§10.4). Butuh API key LLM → `environment: ai-eval`, tidak jalan untuk PR dari fork. | pytest + harness eval | QG-24 |
| `security` | gitleaks (seluruh diff PR), Trivy fs (`vuln,secret,misconfig`; `CRITICAL,HIGH`; `--ignore-unfixed`), dependency review (fail on severity `high`, license allow-list), Trivy license untuk `uv.lock` | gitleaks CLI, Trivy, `actions/dependency-review-action` | QG-17, QG-18, QG-19 |
| `codeql` (workflow `codeql.yml`) | Analisis `python`, `javascript-typescript`, `actions`. Juga terjadwal mingguan. | CodeQL | QG-18 |
| `workflow-lint` | actionlint, zizmor, cek semua `uses:` di-pin ke SHA 40 karakter, validasi `renovate.json` | actionlint, zizmor, check-jsonschema | QG-20 |
| `docs` | markdownlint-cli2, lychee `--offline` (link internal + anchor), script `check_doc_ids.py` (ID yang dirujuk, mis. FR-/NFR-/QG-, ada di dokumen pendefinisi; advisory) | markdownlint-cli2, lychee | QG-21 |
| `ci-ok` | Agregator (`if: always()`). Gagal bila ada dependensi `failure`/`cancelled` | — | Required check |

### 4.3 Anggaran waktu (p50, job paralel)

| Job | Target | Teknik |
|---|---|---|
| changes, pr-title | < 0,5 menit | — |
| backend-lint | 1,5 menit | cache uv (`setup-uv enable-cache`), mypy cache |
| web-lint, web-tests | 2–3 menit | `pnpm` store cache, Turborepo cache (`.turbo` via `actions/cache`), `--affected` |
| backend-tests | 6–7 menit | `pytest -n auto`, Hypothesis profil `ci` (contoh lebih sedikit dari nightly), DB template untuk reset cepat |
| migrations | 2 menit | Hanya bila path migrasi/model berubah |
| contracts | 3–4 menit | — |
| tokens | 1 menit | — |
| web-build (+ Storybook + a11y) | 5–6 menit | cache `.next/cache` |
| e2e-smoke | 8–9 menit (**jalur kritis**) | buildx cache `type=gha`, 2 shard, hanya chromium, subset `@smoke` ≤ 10 skenario |
| security | 3 menit | DB Trivy di-cache |

Jalur kritis ≈ e2e-smoke ≈ 9–10 menit. Bila p50 melewati 10 menit dua minggu berturut-turut: pakai larger runner 4-core untuk `e2e-smoke` dan `backend-tests`, atau tambah shard (§16.3).

### 4.4 Required checks

| Check | PR | Merge queue | Keterangan |
|---|:-:|:-:|---|
| `ci-ok` | ✓ | ✓ | Mencakup semua job pada §4.2, kecuali CodeQL |
| `CodeQL` / code scanning rule | ✓ | ✓ | Alert baru severity ≥ High memblokir merge |
| `pr-title` | ✓ | — | Tidak relevan di merge queue |
| `ai-eval`, `model-regression` | lewat `ci-ok` | lewat `ci-ok` | Hanya bila path terkait berubah |

---

## 5. Pipeline main (`main.yml`)

Trigger: `push` ke `main` (setiap squash merge dari merge queue). Juga `push` ke `release/v*` untuk hotfix (§6.6).

### 5.1 Build image (sekali per commit)

| Image | Isi | Dipakai oleh service |
|---|---|---|
| `msp-api` | Paket Python `msp` (multi-stage uv, base `python:3.13-slim` pin digest, user non-root) | ECS `api` (uvicorn), `worker` (dramatiq), task one-off `migrate` (alembic), job data (§10) |
| `msp-web` | Next.js `output: standalone` di `node:24-slim` (pin digest, non-root) | ECS `web` |

Tiga service deployable (api, worker, web) berasal dari **dua image**. API dan worker memakai paket yang sama, sehingga versi keduanya dijamin identik. Migrasi juga berjalan dari image yang sama.

Langkah per image:
1. `docker buildx build` → **push ke ECR** dengan tag immutable `sha-<commit>`. Provenance BuildKit `mode=max` ikut tersimpan di image index. Label OCI: `org.opencontainers.image.revision`, `.version`, `.source`.
2. **Trivy image scan** pada digest yang baru di-push: 0 temuan Critical/High yang punya perbaikan (QG-26). Pengecualian hanya lewat VEX/`.trivyignore` yang memuat alasan dan tanggal kedaluwarsa.
3. **SBOM** syft (SPDX JSON) → disimpan sebagai artifact workflow dan diunggah ke bucket evidence `msp-shared` (retensi ≥ 1 tahun).
4. **cosign keyless sign** + `cosign attest --type spdxjson` (SBOM) untuk digest tersebut. **Penandatanganan hanya terjadi setelah gate 2 lolos**, sehingga tanda tangan berfungsi sebagai segel "lolos gate". Image yang gagal scan tetap ada di ECR, tetapi tidak bisa di-deploy karena verifikasi tanda tangan di §5.3 gagal.

**Web dan build-once:** variabel `NEXT_PUBLIC_*` di-inline saat build, sehingga nilai yang bergantung environment (URL API, DSN error tracking, nama environment) **tidak boleh** memakai `NEXT_PUBLIC_*`. Nilai tersebut dibaca saat runtime di server dan dikirim ke browser lewat runtime config. Lint rule kustom melarang `NEXT_PUBLIC_*` selain allowlist.

### 5.2 Gate image

| Gate | Gagal bila | Efek |
|---|---|---|
| QG-26 Trivy image | Critical/High fixable > 0 | Tidak di-sign → tidak bisa deploy. Issue `security` otomatis dibuat. |
| Ukuran image | `msp-api` > 600 MB atau `msp-web` > 300 MB (advisory) | Peringatan di summary |

### 5.3 Deploy ke staging

Memakai reusable workflow `_deploy-ecs.yml` (§17.4) dengan `environment: staging` dan `strategy: rolling`:

1. **Verifikasi** tanda tangan cosign + attestation SBOM untuk kedua digest. Identitas yang diterima: workflow `main.yml` di `refs/heads/main` atau `refs/heads/release/v*` repo ini, issuer `https://token.actions.githubusercontent.com` (QG-27).
2. **Migrasi DB** sebagai **one-off ECS task** (`msp-migrate`, image `msp-api` digest yang sama, command `alembic upgrade head`, kredensial role `msp_migrator`). Pipeline menunggu task selesai dan memeriksa exit code. Gagal → deploy berhenti, service lama tetap berjalan. Migrasi dijamin *expand-only* oleh QG-10 sehingga aman bagi versi yang masih berjalan.
3. **Update service** dengan task definition revisi baru yang merujuk digest: urutan **worker → api → web**.
   - Worker dulu: worker baru menerima payload job versi N-1 dan N. Job tipe baru baru di-enqueue setelah API baru aktif.
   - API sebelum web: web baru boleh memakai endpoint baru, sementara API baru tetap kompatibel dengan web lama.
4. Rolling deploy ECS dengan *deployment circuit breaker* (rollback otomatis bila task gagal sehat) dan health check ALB: `/healthz` (liveness) dan `/readyz` (DB, Redis, dan revisi skema ≥ revisi minimum yang dibutuhkan image).
5. **Deployment marker** dikirim ke observability backend dan error tracking (§16.1). GitHub Deployment dicatat beserta digest.

Concurrency `deploy-staging` (tanpa cancel). Bila beberapa commit antre, yang di-deploy adalah commit terbaru.

### 5.4 Verifikasi staging (post-deploy)

| Suite | Isi | Gate |
|---|---|---|
| Smoke | Health, login user sintetis, search, profil, export kecil, webhook test | QG-28 |
| E2E penuh | Playwright seluruh suite (chromium, 4 shard) + axe pada setiap halaman | QG-28, QG-15 |
| Kontrak | Schemathesis terhadap API staging (`--checks all`, stateful links bila didefinisikan), memakai workspace sintetis khusus | QG-28 |
| Tenant isolation (deployed) | Subset suite isolasi via HTTP dengan dua workspace sintetis: akses silang → 403/404, tanpa kebocoran di list/search/export/webhook | QG-28, QG-08 |
| DAST | ZAP baseline **tidak** dijalankan per commit (durasi). Jalan nightly dan untuk kandidat rilis (§7) | QG-29 |

Semua lulus → job `mark-verified` membuat **commit status `staging-verified`** pada SHA tersebut. Status ini prasyarat promosi (§6.2). E2E memakai workspace sintetis yang dibuat per run (prefix `e2e-<run_id>`) dan dibersihkan job harian, sehingga staging tidak perlu di-reset per run.

### 5.5 Kebijakan `main` merah

- Build atau verifikasi staging gagal → notifikasi ke kanal tim + issue otomatis berlabel `main-red` yang di-assign ke penulis commit.
- Aturan **stop-the-line**: penulis memilih **revert** atau **fix-forward** dalam ≤ 1 jam. Bila tidak ada pemilik, tech lead me-revert.
- Commit yang gagal verifikasi tidak pernah mendapat `staging-verified`, sehingga tidak bisa dipromosikan.

---

## 6. Pipeline release (`release.yml`)

### 6.1 Versioning & release-please

- `release-please.yml` berjalan di setiap push ke `main` dan memelihara **Release PR** berisi bump versi + `CHANGELOG.md` dari Conventional Commits. Manifest memuat dua komponen: `.` (aplikasi, `vX.Y.Z`) dan `packages/design-tokens` (`design-tokens-vX.Y.Z`).
- release-please memakai **token GitHub App** (bukan `GITHUB_TOKEN`). Event yang dibuat oleh `GITHUB_TOKEN` tidak memicu workflow lain, padahal tag/Release harus memicu `release.yml`.
- Merge Release PR = commit biasa di `main` → `main.yml` membangun dan memverifikasi image untuk commit tersebut di staging → release-please membuat tag `vX.Y.Z` + GitHub Release → `release.yml` terpicu oleh `release: published`.
- Versi aplikasi dibaca dari file versi saat build (label image, `service.version` di OpenTelemetry, header `X-MSP-Version`).

### 6.2 Readiness (sebelum approval)

| Cek | Berlaku untuk | Sumber |
|---|---|---|
| Commit tag memiliki status `staging-verified` (menunggu ≤ 45 menit bila masih berjalan) | Semua | QG-28 |
| Digest `msp-api` & `msp-web` untuk commit tag ada, tertandatangani, dan punya SBOM | Semua | QG-27 |
| Tidak ada alert Critical/High terbuka (code scanning, dependency, image scan digest ini) | Semua | QG-18, QG-26, NFR-SEC-06 |
| ZAP baseline untuk commit ini (atau nightly ≤ 24 jam pada commit yang sama atau lebih baru di staging) tanpa High | Minor & major | QG-29 |
| Hasil k6 kandidat rilis memenuhi threshold | Minor & major | QG-30 |
| ER golden set penuh (nightly terakhir) lulus | Minor & major | QG-25 |
| Drill restore terakhir lulus dan berumur ≤ 35 hari | Minor & major | QG-34 |
| Issue **Release Readiness** untuk versi ini tertutup dengan bukti RG-1 s.d. RG-7 tertaut | Major dan `v1.0.0` (MVP 1) | QG-31 |
| Bukan masa freeze (§6.5) atau error budget bulan berjalan belum terpakai > 75% | Semua kecuali hotfix | NFR-AVAIL-01 |

Readiness lolos → job deploy menunggu **approval Environment `production`**: required reviewers (tech lead atau on-call senior, bukan pemicu rilis), branch/tag rule hanya `v*`.

### 6.3 Deploy production

1. Verifikasi tanda tangan & attestation (QG-27). Digest yang sama dengan staging. Tidak ada build.
2. Bila rilis memuat migrasi berlabel `db-contract`: buat snapshot RDS manual terlebih dahulu sebagai jaring pengaman tambahan selain PITR.
3. **Migrasi** sebagai one-off ECS task (sama seperti staging).
4. **Worker**: rolling + circuit breaker + alarm (tanpa traffic HTTP).
5. **API & web**: strategi deployment ECS bawaan **canary** (bila strategi canary tidak tersedia di akun, pakai blue/green dengan bake time):

```
  traffic ──► ALB ──┬── 90% ── blue  (versi lama)
                    └── 10% ── green (versi baru)   canary 10 menit
                                 │ alarm OK
                                 ▼
                    100% ── green                   bake 20 menit (blue tetap siaga)
                                 │ alarm OK
                                 ▼
                    blue dihentikan → deploy selesai
             alarm ALARM kapan pun ──► traffic kembali ke blue otomatis (rollback)
```

**Alarm rollback otomatis (QG-32)**, dihitung per target group green bila memungkinkan:

| Alarm | Threshold | Dasar |
|---|---|---|
| Rasio HTTP 5xx | > 2% selama 3 menit (baseline normal < 0,5%) | NFR-AVAIL-01 |
| p95 latency `GET /v1/prospects` | > 800 ms selama 5 menit | NFR-PERF-01 |
| p95 latency `GET /v1/prospects/{id}` | > 500 ms selama 5 menit | NFR-PERF-02 |
| Target tidak sehat | > 0 selama 3 menit, atau circuit breaker terpicu | Health check |
| Worker: rasio job gagal | > 5% selama 10 menit | RG-5 |
| Worker: inflow DLQ | > 20 pesan / 5 menit (awal, dikalibrasi) | RG-5 |

Alarm memakai **CloudWatch** (terintegrasi native dengan ECS), terlepas dari vendor observability yang dipilih (Q-CI-03). Metrik aplikasi diekspor via OpenTelemetry ke CloudWatch.

**Traffic beta rendah:** 10% traffic beta bisa terlalu sedikit untuk sinyal statistik. Selama canary dan bake, job `synthetic-load` menjalankan loop smoke read-only dari workspace sintetis `msp-synthetic-monitor` (tidak memakai credit, dikecualikan dari metrik produk dan billing) agar alarm punya data.

### 6.4 Verifikasi pasca-deploy

Otomatis: smoke production (read-only, workspace sintetis) → commit status `production-deployed` → tag ECR `vX.Y.Z` ditambahkan ke digest yang sama (kemudahan, bukan untuk deploy) → finalisasi release di error tracking → notifikasi. Checklist manual ada di §14.6.

### 6.5 Jendela deploy & freeze

- Production: Senin–Kamis 10:00–15:00 WIB. Jumat ≥ 15:00, akhir pekan, dan hari libur nasional (termasuk periode Lebaran) hanya untuk hotfix.
- Freeze rilis non-fix bila error budget bulanan (99,5% ≈ 3,6 jam/bulan) sudah terpakai > 75%.

### 6.6 Hotfix

1. **Default: fix-forward dari `main`.** Karena `main` selalu dapat dirilis (fitur belum siap ada di balik flag), hotfix = PR `fix/…` biasa → merge → rilis patch.
2. Bila `main` memuat perubahan yang tidak boleh dirilis: tech lead membuat `release/vX.Y` dari tag terakhir → cherry-pick fix → `main.yml` membangun dan memverifikasi di staging (staging sementara menjalankan versi hotfix) → release-please (konfigurasi release branch) membuat `vX.Y.(Z+1)` → `release.yml`. Fix yang sama wajib ada di `main` (PR terpisah) sebelum issue ditutup.

---

## 7. Nightly & scheduled

| Workflow / job | Jadwal (WIB) | Cron (UTC) | Isi | Lokasi eksekusi | Gate / output |
|---|---|---|---|---|---|
| `nightly` · regresi penuh | Setiap hari 01:00 | `0 18 * * *` | Seluruh pytest (termasuk `slow`, Hypothesis profil `nightly`), Vitest penuh, Playwright penuh di chromium + firefox + webkit terhadap staging, viewport 360 px untuk halaman responsif (NFR-COMP-01) | Runner GitHub + staging | Issue `nightly-red` |
| `nightly` · k6 | Setiap hari 02:00 | `0 19 * * *` | Skenario NFR-PERF-01/02/04 pada dataset ukuran NFR-SCAL-01 (atau proporsional, dicatat di laporan) dengan 50 VU | **ECS task di VPC staging** (bukan runner GitHub, agar latency trans-region tidak ikut terukur) | Advisory nightly. Blocking untuk kandidat rilis (QG-30) |
| `nightly` · ZAP baseline | Setiap hari 03:00 | `0 20 * * *` | `zap-baseline` terhadap web + API staging. Mingguan: ZAP API scan berbasis OpenAPI dengan user sintetis | Runner GitHub → staging | QG-29 |
| `nightly` · data quality | Setiap hari 03:30 | `30 20 * * *` | ER golden set penuh (precision/recall), regresi skor terhadap baseline versi aktif, reproducibility geo & skor (recompute sampel → hash identik), suite edge case EC-xx | ECS task di staging. CI hanya menerima ringkasan agregat | QG-25, NFR-DATA-01 |
| `nightly` · rekonsiliasi | Setiap hari 04:00 | `0 21 * * *` | Saldo credit = Σ ledger, usage event vs ledger (staging). Job yang sama dijadwalkan di production oleh aplikasi, dengan alert | ECS task | RG-4 |
| `nightly` · rescan image | Setiap hari 04:30 | `30 21 * * *` | Trivy pada digest yang **sedang berjalan** di staging & production (CVE baru pada image lama) | Runner GitHub | Issue `security` bila Critical/High |
| `nightly` · Lighthouse CI | Setiap hari 05:00 | `0 22 * * *` | Halaman kunci staging. Budget LCP/INP sebagai proxy NFR-PERF-03 | Runner GitHub | Advisory |
| `infra-drift` | Setiap hari 06:00 | `0 23 * * *` | `terraform plan -detailed-exitcode` per env (role read-only) | Runner GitHub | Exit 2 → issue `infra-drift` |
| `renovate` | Setiap jam kerja, Sen–Jum | `0 2-10 * * 1-5` | Renovate self-hosted (§15.4) | Runner GitHub | PR dependency |
| `codeql` (terjadwal) | Senin 02:00 | `0 19 * * 0` | Analisis penuh dengan query terbaru | Runner GitHub | Alert code scanning |
| `link-check-external` | Senin 05:00 | `0 22 * * 0` | lychee dengan link eksternal di `docs/` | Runner GitHub | Advisory |
| `flag-hygiene` | Senin 08:00 | `0 1 * * 1` | Laporan flag kedaluwarsa atau 100% aktif > 30 hari (§13.3) | Runner GitHub | Issue per flag |
| `restore-drill` | Bulanan, Selasa pertama 09:00 | `0 2 1-7 * 2` | Restore PITR RDS production ke instance terisolasi di akun production → verifikasi → hancurkan (§7.1) | Akun production | QG-34 |
| `soak-webhook` | Manual / kandidat MVP 1 | `workflow_dispatch` | 24 jam pengiriman webhook ke sink (termasuk endpoint flaky buatan) di staging | ECS task | RG-3 (≥ 99%) |
| `alert-drill` | Manual, sebelum MVP 1 dan setiap kuartal | `workflow_dispatch` | Injeksi kegagalan via flag `chaos.*` di staging: pipeline failure > 5%, backlog > 1 jam, AI failure > 5%, webhook failure > 5% → verifikasi alert terpicu | Staging | RG-5 |

Catatan cron: GitHub menjalankan `schedule` dengan best effort dan bisa terlambat saat beban tinggi. Job yang waktunya kritis (drill) juga bisa dipicu manual.

### 7.1 Drill restore (bulanan)

1. Catat `T_target` = waktu sekarang − 5 menit. Lakukan restore PITR RDS production ke `msp-restore-drill-<tanggal>` (subnet terisolasi, tanpa akses aplikasi).
2. Ukur **RTO**: waktu dari mulai restore sampai instance *available* dan query verifikasi lulus. Target ≤ 4 jam.
3. Ukur **RPO**: selisih `T_target` dengan timestamp transaksi terakhir yang ada (mis. `max(created_at)` di `audit_log` dan `usage_event`). Target ≤ 15 menit.
4. Verifikasi integritas: jumlah baris tabel kunci dibandingkan dengan production (toleransi sesuai jeda), `alembic current` = head production, dan RLS policy ada.
5. Hancurkan instance. Laporan (tanpa data, hanya metrik) disimpan di bucket evidence. Gagal → incident SEV3 dan QG-34 memblokir rilis minor/major berikutnya.

---

## 8. Quality gates

### 8.1 Daftar QG

Tahap: **PR** = `pr.yml` pada PR dan merge queue · **main** = `main.yml` · **rel** = `release.yml` · **night** = terjadwal · **data** = pipeline rilis data/model. *Blocking* = gagal → merge/deploy/aktivasi terhenti.

| ID | Gate | Tahap | Threshold | Blocking | Sumber |
|---|---|---|---|---|---|
| QG-01 | Judul PR mengikuti Conventional Commits | PR | Lolos validasi type/scope | Ya | §2.2 |
| QG-02 | Lint & format bersih | PR | ruff, ESLint, Prettier: 0 error | Ya | DoD, NFR-MAINT-01 |
| QG-03 | Type check bersih | PR | mypy (strict untuk `msp`), `tsc --noEmit`: 0 error | Ya | NFR-MAINT-01 |
| QG-04 | Unit & integration test lulus | PR | 100% lulus. Test yang dikarantina (§16.3) tidak dihitung, maksimal 14 hari | Ya | DoD §12.2 |
| QG-05 | Coverage modul kritis | PR | ≥ 85% line untuk normalisasi, ER, scoring | Ya | NFR-MAINT-01 |
| QG-06 | Coverage backend keseluruhan | PR | ≥ 75% line | Ya | NFR-MAINT-01 |
| QG-07 | Coverage frontend | PR | ≥ 70% line untuk `packages/ui` dan logika `apps/web` (hooks, lib, formatter). Halaman dicakup E2E. | Advisory 4 minggu pertama, lalu Ya | Proposed (dokumen ini) |
| QG-08 | Suite isolasi tenant | PR, main, night, rel | 100% lulus, termasuk koneksi role aplikasi tanpa `BYPASSRLS` | Ya, **tidak bisa di-waive** | NFR-SEC-03, FR-AUD-006, RG-7 |
| QG-09 | Test privasi | PR | Redaksi PII di log dan payload prompt LLM: 100% lulus | Ya | NFR-PRIV-01, NFR-PRIV-04, FR-AI-005 |
| QG-10 | Keamanan migrasi | PR | Satu head; upgrade dari kosong sukses; `alembic check` bersih; squawk 0 error; RLS lengkap untuk tabel tenant; operasi contract hanya dengan label `db-contract` | Ya | §9, NFR-SEC-03 |
| QG-11 | Lint OpenAPI | PR | Spectral: 0 error (warning dilaporkan) | Ya | DoD §12.2 |
| QG-12 | Tidak ada breaking change OpenAPI tanpa rencana | PR | oasdiff level ERR = 0, **kecuali** label `api-breaking` + bagian "Rencana versi API" terisi | Ya | DoD §12.2 |
| QG-13 | Sinkronisasi kontrak | PR | Spec ter-commit = spec aplikasi; client TS ter-regenerate; contoh payload valid; skema event versi terbit tidak berubah; Schemathesis smoke tanpa 5xx | Ya | DoD §12.2 |
| QG-14 | Design tokens | PR | DTCG valid, alias terselesaikan, output build ter-commit up to date, semua pasangan kontras ≥ WCAG AA | Ya | NFR-A11Y-01 |
| QG-15 | Aksesibilitas | PR, main | axe: 0 pelanggaran **serious/critical** pada story komponen dan halaman E2E | Ya | DoD §12.2, NFR-A11Y-01 |
| QG-16 | E2E smoke | PR, main | 100% skenario `@smoke` lulus (retry 1× diizinkan; lulus setelah retry dicatat sebagai flaky) | Ya | DoD, RG-3 |
| QG-17 | Tidak ada secret | PR (+ push protection) | gitleaks: 0 temuan | Ya, **tidak bisa di-waive** | NFR-SEC-04 |
| QG-18 | Tidak ada kerentanan Critical/High baru | PR, rel | CodeQL alert ≥ High = 0; dependency review severity ≥ high = 0; Trivy fs fixable Critical/High = 0 | Ya | NFR-SEC-06 |
| QG-19 | License policy | PR | Dependency baru hanya dari allow-list (MIT, Apache-2.0, BSD-2/3-Clause, ISC, MPL-2.0, PSF-2.0, 0BSD, Unlicense, CC0-1.0). AGPL/SSPL/GPL ditolak. LGPL perlu review. | Ya | Kebijakan legal (Proposed) |
| QG-20 | Lint workflow & IaC | PR | actionlint + zizmor bersih; semua action di-pin SHA; `terraform fmt/validate`, tflint, Trivy config: 0 High | Ya | §11, §15 |
| QG-21 | Dokumentasi | PR | markdownlint 0 error; link internal valid | Ya (path `docs/**`) | — |
| QG-22 | Versi model/geo/prompt immutable | PR | Konfigurasi versi terbit tidak berubah (`versions.lock`); perubahan konfigurasi wajib menaikkan versi | Ya | NFR-DATA-01, FR-INTL-009 |
| QG-23 | Regresi skor/geo ditinjau | PR (fixture), data (sampel staging) | Laporan diff terlampir; disetujui data-intel + product sebelum aktivasi; determinisme: recompute 2× → hash identik | Ya (aktivasi) | FR-INTL-009, NFR-DATA-01 |
| QG-24 | AI eval | PR (bila prompt berubah), data (aktivasi) | Grounding ≥ 98%; kebocoran PII = 0; validitas format 100%; error/refusal ≤ 2% | Ya | FR-AI-001, FR-AI-004, FR-AI-005 |
| QG-25 | ER golden set | PR (sintetis, bila ER berubah), night (penuh), rel | Precision ≥ 98%, recall ≥ 90% | Ya (PR: sintetis; rel: penuh) | FR-DATA-003, RG-6 |
| QG-26 | Scan image | main | Trivy: 0 Critical/High yang fixable | Ya (image tidak di-sign) | NFR-SEC-06 |
| QG-27 | Integritas artefak | deploy (staging, prod) | Tanda tangan cosign + attestation SBOM valid untuk identitas workflow yang diharapkan | Ya | §15 |
| QG-28 | Verifikasi staging | main | Smoke, E2E penuh, Schemathesis, isolasi tenant (deployed): 100% lulus → `staging-verified` | Ya (untuk promosi) | DoD, RG-3 |
| QG-29 | DAST baseline | night, rel | ZAP: 0 temuan High; Medium ditriase ≤ 14 hari | Ya (rilis) | NFR-SEC-06, RG-7 |
| QG-30 | Performa kandidat rilis | rel (minor/major), night (advisory) | Search p95 ≤ 800 ms & p99 ≤ 2 s; profil p95 ≤ 500 ms; 50 VU; error rate < 1%; export 10.000 baris ≤ 2 menit | Ya (minor/major) | NFR-PERF-01, -02, -04 |
| QG-31 | Bukti release gate | rel (major, `v1.0.0`) | Issue Release Readiness memuat bukti RG-1 s.d. RG-7 dan ditutup oleh product + tech lead | Ya | PRD §12.1 |
| QG-32 | Kesehatan canary/bake | rel | Tidak ada alarm §6.3 selama canary + bake | Ya (rollback otomatis) | NFR-AVAIL-01, NFR-PERF-01 |
| QG-33 | Validasi snapshot dataset | data | Threshold §10.2 | Ya (publish) | FR-DATA-001, NFR-DATA-01 |
| QG-34 | Drill restore | night (bulanan) | RPO ≤ 15 menit, RTO ≤ 4 jam, drill terakhir ≤ 35 hari | Ya (rilis minor/major berikutnya) | NFR-AVAIL-02 |
| QG-35 | Perubahan infrastruktur | infra | Plan sukses dan dikomentari di PR; apply production hanya dari plan yang disetujui | Ya | §11 |

### 8.2 Release gate MVP 1 → bukti dari pipeline

| RG | Bukti | Diproduksi oleh |
|---|---|---|
| RG-1 Relevansi | Laporan usability (manual) + median time-to-first-result dari event analytics | Product/Design, dilampirkan ke issue Release Readiness |
| RG-2 Kepercayaan | Laporan usability + evidence coverage ≥ 95% (ekspor dashboard Data Health) | Product + job data quality |
| RG-3 Aksi | E2E select → export dan select → webhook (QG-28) + laporan `soak-webhook` 24 jam ≥ 99% | `main.yml`, `soak-webhook` |
| RG-4 Usage | Suite billing/idempotensi (QG-04) + laporan rekonsiliasi nightly 100% | `pr.yml`, `nightly` |
| RG-5 Operasional | Laporan `alert-drill`: empat alert terpicu dan diterima on-call | `alert-drill` |
| RG-6 Kualitas data | QG-25 penuh + suite edge case (≥ 10 EC lulus, daftar di 05) | `nightly` data quality |
| RG-7 Keamanan | QG-08 + 0 Critical/High (QG-18/26/29) + suite SSRF webhook lulus | `pr.yml`, `main.yml`, `nightly` |

### 8.3 Kebijakan waiver

- Gate blocking hanya dapat di-waive oleh tech lead (ditambah pemilik security untuk gate keamanan), lewat issue berlabel `qg-waiver` yang mencantumkan alasan, mitigasi, dan **tanggal kedaluwarsa ≤ 14 hari**.
- **QG-08 dan QG-17 tidak pernah di-waive.** Temuan secret ditangani dengan rotasi, bukan pengecualian.
- Pengecualian kerentanan memakai VEX/`.trivyignore` dengan tanggal kedaluwarsa. Pengecualian kedaluwarsa otomatis membuat gate gagal lagi.

---

## 9. Strategi migrasi database

### 9.1 Prinsip

1. **Setiap migrasi kompatibel dengan versi aplikasi yang sedang berjalan (N-1).** Migrasi berjalan *sebelum* rolling deploy, dan rollback aplikasi kembali ke image N-1 di atas skema baru.
2. **Expand → migrate → contract** untuk perubahan yang tidak kompatibel, tersebar di beberapa rilis.
3. **Rollback skema = forward fix.** Fungsi `downgrade()` boleh ditulis untuk kenyamanan lokal, tetapi tidak pernah dijalankan di staging/production. PITR adalah upaya terakhir (§14.3).
4. Migrasi pendek: target < 5 menit di production. Pekerjaan berat (backfill, rebuild) berjalan sebagai **background job**, bukan di migrasi.

### 9.2 Contoh: rename kolom `contact_point.verified_at` → `last_verified_at`

| Rilis | Fase | Migrasi | Kode aplikasi |
|---|---|---|---|
| N | **Expand** | `ADD COLUMN last_verified_at timestamptz NULL` (tanpa default, tanpa lock panjang) | Tulis ke **kedua** kolom (dual-write). Baca dari `verified_at`. Enqueue job backfill `backfill_last_verified_at` (batch 5.000 baris, idempotent, dapat dilanjutkan, di-throttle). |
| N+1 | **Migrate** | (opsional) `CREATE INDEX CONCURRENTLY` pada kolom baru. Bila perlu NOT NULL: `ADD CONSTRAINT … CHECK (last_verified_at IS NOT NULL) NOT VALID` → `VALIDATE CONSTRAINT` | Baca dari `last_verified_at`. Tetap dual-write (rollback ke N tetap aman). Rilis baru boleh dipromosikan setelah verifikasi backfill: `count(*) WHERE last_verified_at IS NULL AND verified_at IS NOT NULL = 0`. |
| N+2 | **Contract** (label `db-contract`) | `DROP COLUMN verified_at`. Migrasi berada di rilis setelah N+1 berjalan stabil di production, dan N bukan lagi target rollback. | Berhenti menulis `verified_at`. ORM tidak lagi memetakan kolom lama. **Kode rilis N+1 sudah tidak membaca kolom ini.** |

Rename tabel, perubahan tipe, dan split kolom mengikuti pola yang sama. Pada fase transisi, view kompatibilitas boleh dipakai.

### 9.3 Aturan

| # | Aturan | Ditegakkan oleh |
|---|---|---|
| M-1 | Tidak ada perubahan destruktif (DROP, RENAME, penyempitan tipe, NOT NULL tanpa jalur aman) di rilis yang sama dengan kode yang berhenti bergantung padanya. | `check_migration_phase.py` + label `db-contract` (QG-10) |
| M-2 | Index pada tabel yang sudah berisi data: `CREATE INDEX CONCURRENTLY` di dalam `autocommit_block()` Alembic. | squawk (`require-concurrent-index-creation`) |
| M-3 | Foreign key dan CHECK baru: `NOT VALID`, lalu `VALIDATE CONSTRAINT` di migrasi terpisah. | squawk |
| M-4 | `SET lock_timeout = '5s'` dan `statement_timeout` default di `env.py`. Override per migrasi harus dijustifikasi. Gagal karena lock → retry pipeline, bukan menunggu tanpa batas. | `env.py`, review |
| M-5 | Backfill data = background job Dramatiq yang batched, idempotent, resumable, dan terpantau (progres di ops console). Pengecualian: tabel referensi kecil (< 10.000 baris). | Review |
| M-6 | Tabel tenant-layer baru: `ENABLE` + `FORCE ROW LEVEL SECURITY` + policy `USING`/`WITH CHECK` berbasis `workspace_id` **di migrasi yang sama** dengan `CREATE TABLE`. | `check_rls.py` (QG-10) |
| M-7 | Role terpisah: `msp_migrator` (pemilik DDL), `msp_app` (DML, `NOBYPASSRLS`, bukan owner tabel), `msp_readonly` (analitik ops). Aplikasi tidak pernah berjalan sebagai owner. | Terraform + test isolasi |
| M-8 | Migrasi memegang advisory lock Postgres, sehingga dua runner tidak bisa berjalan bersamaan. | `env.py` |
| M-9 | Satu head Alembic. Konflik dua PR diselesaikan dengan rebase dan perbaikan `down_revision`, bukan merge revision. | QG-10 |
| M-10 | Tidak ada data pribadi nyata di migrasi atau seed. | Review + gitleaks |

### 9.4 Checklist review migrasi

- [ ] Fase dicatat (`expand` / `contract`) di docstring revision
- [ ] Kompatibel dengan kode rilis sebelumnya (baca dan tulis)
- [ ] Estimasi baris terdampak di production dan durasi lock dicantumkan di deskripsi PR
- [ ] Index `CONCURRENTLY`, constraint `NOT VALID` + `VALIDATE`
- [ ] Tidak ada backfill besar di dalam migrasi
- [ ] RLS + policy untuk tabel tenant
- [ ] Grant untuk `msp_app` / `msp_readonly` eksplisit
- [ ] Migrasi contract: rilis yang berhenti memakai kolom sudah berjalan di production ≥ 1 rilis

### 9.5 Eksekusi per environment

| Env | Cara | Kredensial |
|---|---|---|
| local | Service `migrate` di compose, atau `uv run alembic upgrade head` | `.env` |
| CI | Step `alembic upgrade head` sebelum integration test. Job `migrations` dari DB kosong. | Service container |
| staging | One-off ECS task `msp-migrate` (digest yang sama) di `_deploy-ecs.yml`, sebelum update service | Secrets Manager → `msp_migrator` |
| production | Sama dengan staging, setelah approval Environment `production`. Snapshot manual RDS untuk rilis `db-contract`. | Secrets Manager → `msp_migrator` |

### 9.6 PostGIS & ekstensi

- `postgis`, `pg_trgm`, dan `unaccent` dibuat di **migrasi pertama** (`CREATE EXTENSION IF NOT EXISTS`). Di RDS, `msp_migrator` anggota `rds_superuser`, khusus untuk keperluan ini.
- Versi PostgreSQL/PostGIS di image local/CI di-pin agar sama dengan versi RDS (parity check, §3.1).
- Upgrade minor PostGIS (`ALTER EXTENSION postgis UPDATE`) dan upgrade engine RDS adalah **perubahan terencana**: diuji dulu di staging. Upgrade major memakai RDS Blue/Green Deployments, lewat change request di Terraform (§11) dan jendela maintenance yang diumumkan.

---

## 10. Rilis data & model

Rilis data/model terpisah dari rilis kode: kode yang mendukung versi baru di-deploy dulu (versi terdaftar, belum aktif), lalu **aktivasi** dilakukan lewat pipeline dengan gate dan approval. Semua langkah berat berjalan sebagai ECS task di akun yang sesuai (P-7). Workflow GitHub hanya mengorkestrasi dan menyimpan laporan agregat.

### 10.1 Registry versi & reproducibility (NFR-DATA-01)

| Artefak | Format versi | Disimpan di | Immutable |
|---|---|---|---|
| Snapshot dataset | `ds-YYYY.MM.n` | Manifest + file mentah di S3 (versioning, checksum), metadata di DB | Ya (FR-DATA-001) |
| Model skor | `priority-X.Y.Z` | File konfigurasi di repo + `versions.lock` (hash), baris registry di DB | Ya setelah dirilis |
| Metrik geo | `geo-X.Y.Z` | Sama | Ya |
| Prompt AI | `brief-X.Y.Z` | Template + parameter model di repo + `versions.lock` | Ya |

Setiap skor dan metrik area menyimpan `snapshot_version` + `model_version`/`geo_version`. Nightly memverifikasi bahwa recompute sampel dengan pasangan versi yang sama menghasilkan hash identik. Laporan rilis (validasi, regresi, eval) disimpan di bucket evidence minimal selama versi tersebut masih direferensikan oleh data aktif, ditambah 1 tahun.

Status versi: `draft` → `staged` (tervalidasi di staging) → `active` (production) → `retired`. Hanya satu versi `active` per jenis (per preset Offering untuk skor).

### 10.2 Snapshot dataset (`ds-YYYY.MM.n`) — `data-release.yml`

```
ops upload file mentah + manifest ──► [1] register (prod, landing bucket, checksum)
        ──► [2] validasi import mode staged (prod, belum dipublikasikan) ──► laporan validasi
        ──► [3] rehearsal di staging (subset/sampel sesuai lisensi): pipeline penuh
        ──► [4] gate QG-33 ──► [5] approval Environment data-production ──► [6] publish
        ──► [7] recompute skor stale (FR-INTL-011) ──► [8] verifikasi pasca-publish
```

| Langkah | Detail |
|---|---|
| 1. Register | Manifest: sumber, `captured_at`, periode referensi, tag lisensi (BR-PRIV-04), checksum per file. Checksum tidak cocok → berhenti. |
| 2. Validasi | Import ke status *staged*: parsing Input Contract, laporan jumlah baris, reject per alasan, fill-rate per field, delta terhadap snapshot aktif (FR-DATA-001). Data tetap di akun production. |
| 3. Rehearsal staging | Sampel terstratifikasi (wilayah × kategori) disalin ke staging bila lisensi mengizinkan (Q-05). Pipeline normalisasi → ER → verifikasi → skor → geo dijalankan. Diukur: ER golden metrics, distribusi skor vs snapshot sebelumnya, durasi (diekstrapolasi ke NFR-PERF-07). |
| 4. Gate QG-33 | Lihat tabel threshold di bawah. |
| 5. Approval | Environment `data-production`: data owner + engineer on-call. |
| 6. Publish | Pipeline production pada snapshot staged → setelah selesai, pointer snapshot aktif diganti secara atomik. |
| 7. Recompute | Skor yang lebih tua dari input ditandai stale dan di-recompute (FR-INTL-011). Progres dipantau. |
| 8. Verifikasi | Dashboard Data Health per snapshot (FR-OPS-001): rasio verifikasi, stale, kandidat duplikat, evidence coverage ≥ 95%. |

**Threshold QG-33 (default awal, dikalibrasi setelah Q-01/Q-02):**

| Cek | Threshold |
|---|---|
| Baris yang tidak bisa diproses tanpa alasan reject | 0 (setiap reject wajib punya alasan) |
| Reject rate total | ≤ 5% (2–5% → wajib catatan di approval) |
| Fill-rate `source_record_id` / nama | 100% / ≥ 99% |
| Fill-rate kategori sumber | ≥ 95% |
| Alamat atau koordinat | ≥ 95% |
| `source_record_id` duplikat dalam snapshot | 0 |
| Penurunan fill-rate field apa pun vs snapshot aktif | ≤ 10 poin persentase (lebih → wajib catatan) |
| Perubahan jumlah record vs snapshot aktif | −10% s.d. +30% (di luar rentang → wajib catatan) |
| Establishment ber-koordinat yang terpetakan ke kelurahan | ≥ 90% (FR-DATA-009) |
| Tag lisensi | Ada |
| ER golden set pada rehearsal | Precision ≥ 98%, recall ≥ 90% |

**Rollback:** snapshot immutable dan ID entity stabil (FR-DATA-007), sehingga rollback = **publish ulang snapshot sebelumnya** (pointer aktif dikembalikan) + recompute. Data tenant yang merujuk entity tetap valid. Mekanisme detail ada di 04 §Dataset Snapshot.

### 10.3 Model skor (`priority-X.Y.Z`) dan geo (`geo-X.Y.Z`) — `model-release.yml`

| Langkah | Detail | Gate |
|---|---|---|
| 1. PR konfigurasi | Bobot/aturan/preset baru → **versi baru** (versi lama tidak boleh diubah). Unit test + recompute fixture "golden scores" → diff di komentar PR. | QG-22, QG-04 |
| 2. Merge & deploy | Versi terdaftar sebagai `draft` di staging dan production. Belum aktif. | Pipeline biasa |
| 3. Recompute offline (staging) | ECS task menghitung versi lama dan baru pada sampel staging (beberapa ICP/Offering preset representatif). | — |
| 4. Laporan regresi | Artefak: distribusi skor per band, matriks transisi band (HOT/WARM/COLD/UNSCORED), pergeseran rata-rata per faktor, overlap top-50 per ICP sampel, perubahan coverage. Geo: distribusi metrik area, jumlah area coverage LOW, perubahan peringkat territory. Determinisme: hitung 2× → hash identik. | QG-23 |
| 5. Approval | Environment `model-production`: data-intel + product. Bila transisi band > 15% sampel atau overlap top-50 < 70%, product wajib menulis justifikasi. | QG-23 |
| 6. Aktivasi | Versi → `active` di production → job recompute bertahap (batched, progres, target NFR-PERF-05 untuk scope ICP). Skor versi lama tetap tersimpan sampai versi baru stabil (bake 7 hari). | — |
| 7. Pantau | Dashboard Intelligence Health (FR-OPS-003): distribusi skor per versi, feedback "terlalu tinggi/rendah" (FR-INTL-010). | — |

**Rollback:** aktifkan kembali versi sebelumnya. Skornya masih tersimpan selama bake, sehingga tidak perlu recompute. Setelah bake, versi sebelumnya tetap dapat direproduksi dari snapshot + versi (FR-INTL-009).

### 10.4 Prompt AI (`brief-X.Y.Z`)

| Langkah | Detail | Gate |
|---|---|---|
| 1. PR prompt | Template/parameter baru → versi baru. Penggantian model provider (D-15) diperlakukan sebagai bump minor `brief-*` + label `ai-eval`. | QG-22 |
| 2. Eval PR | Eval set **sintetis** di repo (≥ 200 profil company-level, termasuk data jarang/Unknown, multi-cabang, evidence bertentangan, dan person yang hanya diwakili role). Metrik: grounding (validator sitasi FR-AI-004) ≥ 98%, kebocoran nilai contact point pribadi ke prompt = 0, validitas format 100%, error/refusal ≤ 2%, plus latency p95 dan biaya per brief (indikatif, NFR-PERF-06, NFR-COST-01). | QG-24 |
| 3. Eval aktivasi | Set eval diperluas di staging (company-level, sesuai D-14) dijalankan ulang terhadap versi kandidat. | QG-24 |
| 4. Aktivasi bertahap | Flag `rel.ai.brief-<versi>`: workspace internal → cohort beta → 100%. Cache brief berbasis hash input **termasuk** versi prompt (FR-AI-003). | — |
| 5. Pantau | Tingkat kegagalan AI, feedback 👍/👎 (FR-AI-007), biaya per brief. Alert AI failure > 5% (RG-5). | — |

**Rollback:** aktifkan kembali versi prompt sebelumnya (flag/registry). Kill switch `ks.ai` (FR-AI-006) sebagai upaya terakhir.

---

## 11. Infrastructure as Code

### 11.1 Layout

```
infra/terraform/
├─ modules/
│  ├─ network/            VPC, subnet privat/publik, NAT, VPC endpoints (ECR, S3, Secrets Manager, Logs)
│  ├─ ecs-cluster/
│  ├─ ecs-service/        service + baseline task def + ALB target group + autoscaling + alarm rollback
│  ├─ rds-postgres/       Multi-AZ (prod), PITR, parameter group, enkripsi KMS
│  ├─ elasticache-redis/
│  ├─ s3-bucket/          enkripsi, versioning, block public access, lifecycle
│  ├─ ecr/                tag immutable, scan on push, kebijakan pull lintas akun
│  ├─ github-oidc/        OIDC provider + role per tujuan (§12.1)
│  ├─ secrets/            Secrets Manager + rotasi
│  └─ observability/      log group, dashboard SLO, alarm, SNS → on-call
├─ envs/
│  ├─ shared/             ECR, bucket evidence, role push image   (tambahan: akun msp-shared)
│  ├─ staging/
│  └─ production/
└─ .tflint.hcl
infra/deploy/taskdefs/     template task definition (api, worker, web, migrate) yang dirender CI
```

- Terraform memiliki cluster, service, ALB, IAM, dan baseline task definition. Revisi task definition per deploy dibuat oleh CI, sehingga `aws_ecs_service` memakai `lifecycle { ignore_changes = [task_definition] }` (juga `desired_count` bila autoscaling). Drift detection tidak menganggap deploy aplikasi sebagai drift.
- `required_version` dan versi provider di-pin. `.terraform.lock.hcl` di-commit dengan hash multi-platform.

### 11.2 Workflow `infra.yml`

| Event | Langkah | Role OIDC |
|---|---|---|
| PR menyentuh `infra/terraform/**` | `fmt -check`, `validate`, tflint, Trivy config (QG-20) → `plan` untuk env yang terdampak → **komentar sticky di PR** (ringkasan resource add/change/destroy + plan lengkap sebagai artifact; nilai sensitif tetap ter-mask) | `gha-tf-plan-<env>` (read-only + baca state) |
| Merge ke `main` | `plan -out` staging → apply otomatis (Environment `staging-infra`) → `plan -out` production → **approval** Environment `production-infra` → `apply` **file plan yang sama** | `gha-tf-apply-<env>` |
| Terjadwal | Drift detection (§7) | `gha-tf-plan-<env>` |

Plan yang memuat `destroy` pada resource stateful (RDS, S3 data, KMS key) gagal, kecuali PR berlabel `infra-destroy` dan resource tersebut tidak memiliki `prevent_destroy`.

### 11.3 Remote state

- Backend S3 per akun (`msp-tfstate-<env>`): versioning, enkripsi SSE-KMS, block public access, **`use_lockfile = true`** (native S3 locking).
- Akses state hanya untuk role plan/apply env tersebut dan break-glass admin.

### 11.4 Tagging

`default_tags` di provider: `Project=mesinprospek`, `Environment=<env>`, `Owner=<tim>`, `ManagedBy=terraform`, `Repo=sadewadee/MesinProspek`, `CostCenter=<…>`, `DataClassification=<public|internal|confidential|personal>`. Tag dipakai untuk cost allocation (FR-OPS-004) dan kebijakan akses.

### 11.5 Least privilege & drift

- Role apply production tidak bisa diasumsikan dari PR. Hanya job dengan Environment `production-infra` yang bisa (§12.1).
- Tidak ada `terraform apply` dari laptop, kecuali break-glass (tercatat di CloudTrail dan dilaporkan di issue).
- Perubahan manual di console → drift terdeteksi ≤ 24 jam → diselesaikan dengan PR (adopsi ke kode) atau revert dengan apply.

---

## 12. Secrets & konfigurasi

### 12.1 GitHub OIDC → AWS IAM (tanpa long-lived key)

| Role | Akun | Dipakai oleh | Trust (`sub` claim) | Izin |
|---|---|---|---|---|
| `gha-ecr-push` | shared | `main.yml` build | `repo:sadewadee/MesinProspek:ref:refs/heads/main` (+ `release/v*`) | Push/pull ECR `msp-*` |
| `gha-deploy-staging` | staging | `_deploy-ecs.yml` | `repo:sadewadee/MesinProspek:environment:staging` | Register task def, update service, run-task `msp-migrate`, `iam:PassRole` terbatas ke role task |
| `gha-deploy-production` | production | `_deploy-ecs.yml` | `…:environment:production` | Sama dengan staging, di akun production |
| `gha-data-jobs-<env>` | staging/prod | rilis data/model, nightly data quality | `…:environment:data-<env>` | Run-task job data, baca laporan S3 |
| `gha-tf-plan-<env>` | per env | `infra.yml` PR & drift | `…:pull_request` dan `…:ref:refs/heads/main` | ReadOnly + baca state |
| `gha-tf-apply-<env>` | per env | `infra.yml` apply | `…:environment:<env>-infra` | Admin terbatas pada resource bertag `Project=mesinprospek` |

Audience `sts.amazonaws.com`. Kondisi trust memakai `StringEquals` (bukan wildcard luas) pada `sub` dan `aud`. Job tanpa kebutuhan AWS tidak mendapat `id-token: write`.

### 12.2 Penyimpanan secret & konfigurasi

| Jenis | Tempat | Contoh |
|---|---|---|
| Secret runtime | **AWS Secrets Manager** (per akun), disuntik via `secrets` task definition | Kredensial DB (rotasi otomatis RDS), API key LLM, provider email, managed auth, kunci master webhook |
| Konfigurasi non-secret | **SSM Parameter Store** | Endpoint, ukuran batch, URL publik, nama bucket |
| Secret CI | GitHub Environment secrets (minimal) | Private key GitHub App release, API key LLM untuk `ai-eval` |
| Variabel CI non-secret | GitHub Environment/Repo variables | ARN role, ID akun, URL environment |

### 12.3 Konvensi environment variable (12-factor)

- Prefix `MSP_`, huruf besar, dibaca lewat Pydantic Settings. Contoh: `MSP_ENV` (`local|test|staging|production`), `MSP_DATABASE_URL`, `MSP_REDIS_URL`, `MSP_S3_BUCKET_EXPORTS`, `MSP_OTEL_EXPORTER_OTLP_ENDPOINT`, `MSP_AUTH_MODE`, `MSP_LOG_LEVEL`.
- Web: runtime config di server (§5.1). `NEXT_PUBLIC_*` hanya untuk nilai yang identik di semua environment.
- Aplikasi gagal start (fail fast) bila variabel wajib hilang. Nilai secret tidak pernah dicetak, termasuk di pesan error validasi.
- `.env.example` di root berisi semua variabel dengan nilai dummy aman untuk local. `.env` ada di `.gitignore`. Pre-commit hook gitleaks dianjurkan untuk engineer.

### 12.4 Jangan pernah mencetak secret

- Workflow tidak memakai `set -x` pada step yang menyentuh secret. Nilai turunan secret di-mask (`::add-mask::`).
- Log aplikasi memakai redaksi terstruktur (BR-PRIV-01). Test redaksi termasuk QG-09.
- Artifact CI (trace Playwright, screenshot) hanya berasal dari data sintetis. Retensi 7 hari, diunggah hanya saat gagal.

### 12.5 Rotasi kunci master webhook

Secret per endpoint webhook (FR-EXP-003) disimpan terenkripsi dengan **kunci master** (envelope encryption via AWS KMS, detail di 04). Rotasi:
1. Buat versi kunci baru (`webhook-kek-v{n+1}`). Aplikasi bisa mendekripsi dengan v{n} dan v{n+1}, sedangkan enkripsi baru memakai v{n+1}.
2. Job background mengenkripsi ulang semua secret endpoint ke v{n+1} (idempotent, progres terpantau).
3. Setelah 100% dan bake 7 hari, v{n} dinonaktifkan, lalu dihapus sesuai kebijakan KMS.

Rotasi terjadwal tahunan, dan segera bila ada dugaan kebocoran. Rotasi secret per endpoint oleh customer (periode overlap dua secret) didefinisikan di 04.

---

## 13. Feature flags & progressive delivery

### 13.1 Tipe flag

| Tipe | Prefix | Contoh | Umur | Default saat flag store gagal |
|---|---|---|---|---|
| Release | `rel.` | `rel.geo.territory-ranking`, `rel.ai.brief-1.1.0` | Sementara: dihapus ≤ 30 hari setelah 100% | **Off** |
| Ops kill switch (FR-OPS-007, FR-AI-006) | `ks.` | `ks.ai`, `ks.export`, `ks.webhook`, `ks.upload` | Permanen | Nilai terakhir di cache. Tanpa cache → kapabilitas **on** (tidak memutus layanan karena flag store gagal) |
| Fault injection (staging) | `chaos.` | `chaos.webhook-5xx`, `chaos.pipeline-fail` | Permanen | Off. Ditolak di production (§3.3) |
| Eksperimen | `exp.` | (V1+) | Sementara | Off |

**Permission dan plan bukan flag.** Entitlement plan (FR-BILL-003) dan RBAC (FR-AUTH-004) ditegakkan oleh service entitlement/permission di server. Flag tidak pernah menjadi satu-satunya penghalang akses ke data atau fitur berbayar.

### 13.2 Mekanisme

- **Registry di repo** (`apps/api/src/msp/flags/registry.yaml`): nama, tipe, owner, deskripsi, tanggal dibuat, tanggal kedaluwarsa (wajib untuk `rel.`/`exp.`), default. CI memeriksa bahwa setiap flag yang dipakai di kode (backend & web) terdaftar, dan setiap flag terdaftar dipakai.
- **Nilai runtime** per environment disimpan di Postgres (tabel flag + override per workspace), diubah lewat ops console, dan **tercatat di audit log** (FR-OPS-007). Evaluasi di-cache in-process (TTL 30 detik) dan disebarkan lewat Redis pub/sub agar kill switch berlaku ≤ 1 menit.
- Antarmuka evaluasi tipis yang kompatibel dengan OpenFeature, agar bisa berpindah ke vendor kelak tanpa mengubah call site.

### 13.3 Lifecycle & cleanup

```
dibuat (PR + registry) → dark launch (off di prod) → workspace internal MSP → cohort beta (allowlist)
→ persentase per workspace (hash workspace_id: 10% → 50% → 100%) → PR hapus flag + kode lama (≤ 30 hari)
```

- Job `flag-hygiene` mingguan membuat issue untuk flag `rel.` yang melewati tanggal kedaluwarsa atau sudah 100% > 30 hari.
- CI gagal bila flag `rel.` melewati kedaluwarsa > 30 hari tanpa perpanjangan (perpanjangan = PR yang mengubah tanggal, dengan alasan).
- Rollout bertahap selalu per **workspace** (bukan per user), agar pengalaman konsisten di satu tim.

### 13.4 Dark launch

Kode baru di-deploy dalam keadaan off, sehingga deploy (perubahan kode) terpisah dari release (perubahan perilaku). Untuk pipeline berat (mis. faktor skor baru), dark launch dapat menghitung hasil secara bayangan (shadow) tanpa ditampilkan, lalu hasilnya dibandingkan di laporan regresi (§10.3).

---

## 14. Rollback & incident

### 14.1 Rollback aplikasi

- Workflow **`rollback.yml`** (`workflow_dispatch`, input: `environment`, `to` = tag versi atau `previous`). Satu perintah:
  `gh workflow run rollback.yml -f environment=production -f to=previous`
- Langkah: resolve digest dari riwayat GitHub Deployments → verifikasi tanda tangan (QG-27) → update service **tanpa migrasi** (urutan web → api → worker, kebalikan deploy) → smoke. Target ≤ 15 menit.
- Aman karena skema bersifat expand-only terhadap N-1 (§9.1). Rollback melewati lebih dari satu rilis hanya boleh bila tidak ada migrasi contract di antaranya. Workflow memeriksa ini dan menolak bila ada.
- Approval: Environment `production` (on-call cukup satu reviewer. Saat SEV1, admin repo dapat mem-bypass review dan pemakaian bypass tercatat).

### 14.2 Matikan lewat flag

Langkah tercepat untuk kapabilitas yang bermasalah: kill switch `ks.ai`, `ks.export`, `ks.webhook`, `ks.upload` (≤ 1 menit, tercatat di audit), atau set flag `rel.*` terkait ke off. Pilihan ini didahulukan sebelum rollback bila masalah terlokalisasi.

### 14.3 Database

- **Forward fix:** migrasi perbaikan baru melalui pipeline normal (bisa dipercepat: review on-call + tech lead).
- **PITR** (RPO ≤ 15 menit) hanya untuk korupsi atau kehilangan data yang tidak bisa diperbaiki secara forward. Keputusan ada di incident commander + founder, karena data yang ditulis setelah titik restore hilang (termasuk usage/ledger, yang harus direkonsiliasi). Restore dilakukan ke instance baru, lalu aplikasi diarahkan ulang. Prosedurnya dilatih di drill bulanan (§7.1).

### 14.4 Rollback data & model

| Artefak | Rollback |
|---|---|
| Snapshot dataset | Publish ulang snapshot aktif sebelumnya + recompute (§10.2) |
| Versi skor/geo | Aktifkan kembali versi sebelumnya. Skornya masih ada selama bake 7 hari (§10.3) |
| Versi prompt | Aktifkan kembali versi sebelumnya via flag/registry. Kill switch `ks.ai` bila perlu |

### 14.5 Incident

| Severity | Contoh | Respons | Komunikasi |
|---|---|---|---|
| **SEV1** | Production down; **kebocoran data lintas tenant**; dugaan kegagalan pelindungan data pribadi; tagihan ganda massal | Segera, 24/7. Incident commander ditunjuk | Kanal insiden + founder ≤ 15 menit; customer terdampak ≤ 4 jam (email); status berkala setiap 1 jam |
| **SEV2** | Fitur inti terganggu (search, export, webhook, unlock), SLO burn tinggi | ≤ 30 menit jam kerja, ≤ 1 jam di luar jam kerja | Kanal insiden; customer bila > 1 jam |
| **SEV3** | Degradasi sebagian dengan workaround; drill gagal | Hari kerja yang sama | Issue |
| **SEV4** | Kosmetik, internal | Backlog | — |

Peran: incident commander, komunikasi, scribe. Mulai dengan containment (kill switch, rollback), baru diagnosis.

**Hook UU PDP (BR-PRIV-03) di runbook:** setiap insiden yang menyangkut data pribadi (contact point person, data upload customer, akses lintas tenant) langsung ditandai `pdp-suspected`. Penandaan ini memicu:
1. Pencatatan waktu diketahuinya insiden. **Hitung mundur 3×24 jam** untuk notifikasi kegagalan pelindungan data dimulai.
2. Eskalasi ke founder dan penanggung jawab pelindungan data, serta konsultan hukum. Ruang lingkup dan pihak yang wajib diberi tahu divalidasi konsultan hukum (BR-PRIV-03).
3. Preservasi bukti: audit log (immutable), CloudTrail, log akses S3/ALB untuk jendela waktu terkait. Retensi diperpanjang, tidak ada penghapusan.
4. Draf notifikasi (isi minimal: data yang terdampak, waktu dan cara kejadian, upaya penanganan dan pemulihan) disiapkan paralel dengan penanganan teknis.

**Post-incident review** (SEV1/SEV2 wajib, SEV3 opsional): blameless, ≤ 5 hari kerja, timeline + akar masalah + action item ber-owner dan bertenggat (issue berlabel `pir`). Setiap insiden menghasilkan minimal satu test regresi atau alert baru. Change failure rate dan MTTR diperbarui (§16.2).

### 14.6 Checklist verifikasi pasca-deploy

- [ ] `/healthz` dan `/readyz` hijau untuk semua task. Jumlah task sesuai desired count
- [ ] Smoke production lulus (workspace sintetis)
- [ ] Rasio 5xx, p95 search/profil dalam threshold (§6.3) selama 30 menit
- [ ] Worker: backlog normal, tidak ada lonjakan DLQ, rasio job gagal < 5%
- [ ] Log migration task bersih, `alembic current` = head yang diharapkan
- [ ] Error tracking: tidak ada issue baru severity tinggi pada release ini
- [ ] Status flag sesuai rencana rollout (tidak ada flag yang ikut menyala tanpa sengaja)
- [ ] Deployment marker dan anotasi rilis tercatat (§16.1)
- [ ] Changelog dipublikasikan. Perubahan yang terlihat customer dikomunikasikan (id-ID)

---

## 15. Supply chain security

### 15.1 GitHub Actions

| Kontrol | Penerapan |
|---|---|
| Pin SHA | Semua `uses:` pihak ketiga dan resmi di-pin ke SHA commit 40 karakter, dengan komentar versi (`# vX.Y.Z`). Renovate memperbarui SHA dan komentar. Pelajaran insiden `tj-actions/changed-files` (Maret 2025): tag yang bisa dipindah dapat menyuntikkan kode yang membocorkan secret. |
| Permission minimal | `permissions: {}` di level workflow. Setiap job membuka hanya yang dibutuhkan (`contents: read`, `id-token: write` hanya untuk job AWS, `pull-requests: write` hanya untuk komentar). |
| Checkout | `persist-credentials: false` kecuali job yang perlu push (release-please memakai token App). |
| Trigger berbahaya | Tidak ada `pull_request_target` yang men-checkout kode PR. Tidak ada interpolasi `${{ github.event.* }}` langsung di `run:` (lewat `env:`). Ditegakkan oleh zizmor (QG-20). |
| Fork | Repo privat. Bila kelak ada kontributor eksternal, PR fork tidak mendapat secret dan job `ai-eval`/deploy tidak berjalan. |
| Runner | GitHub-hosted (ephemeral). Self-hosted runner tidak dipakai di MVP. |
| Pengubahan workflow | `.github/**` milik CODEOWNERS platform. Environment `production` membatasi deploy ke tag `v*`, jadi workflow di branch fitur tidak bisa deploy ke production. |

### 15.2 Dependency

- Lockfile wajib dan instalasi beku: `uv sync --locked`, `pnpm install --frozen-lockfile`.
- Dependency review di PR (QG-18, QG-19). Trivy license untuk `uv.lock` sebagai pemeriksa silang.
- Base image di-pin digest dan diperbarui Renovate.

### 15.3 Artefak: SBOM, provenance, signing

| Artefak | Dibuat | Disimpan | Retensi |
|---|---|---|---|
| SBOM SPDX (syft) | Setiap image `main` | Attestation cosign di ECR + artifact workflow + bucket evidence | ≥ 1 tahun, atau selama digest masih di-deploy |
| Provenance SLSA | BuildKit `mode=max` (image index). Bila plan GitHub mendukung artifact attestations untuk repo privat, tambahkan `actions/attest-build-provenance`. | ECR | Sama |
| Tanda tangan | cosign keyless (Fulcio/Rekor) setelah QG-26 lolos | ECR (+ transparency log) | Sama |

Verifikasi saat deploy (QG-27): `cosign verify` + `cosign verify-attestation --type spdxjson` dengan `--certificate-identity-regexp` yang mengunci repo, workflow `main.yml`, dan ref `main`/`release/v*`, serta `--certificate-oidc-issuer https://token.actions.githubusercontent.com`. Role deploy hanya boleh merujuk image dari ECR `msp-*` (kondisi IAM), sehingga image dari registry lain tidak bisa dipakai.

> Catatan privasi: entri transparency log publik Sigstore memuat identitas workflow (nama repo, path workflow, ref). Isinya bukan kode atau data, tetapi tetap terlihat publik. Bila tidak dapat diterima, ganti ke `cosign --key awskms://…` (kunci KMS di akun shared, tanpa upload tlog). Lihat Q-CI-04.

### 15.4 Renovate

| Kebijakan | Nilai |
|---|---|
| Jadwal | Jam kerja Senin–Jumat (WIB) |
| `minimumReleaseAge` | 3 hari untuk semua update (perlindungan dari rilis berbahaya yang ditarik kembali). Security update: tanpa penundaan. |
| Grouping | `python-dev` (ruff, mypy, pytest-*), `js-lint` (eslint, prettier, typescript-eslint), `storybook`, `playwright`, `github-actions` (SHA), `docker-base`, `terraform-providers` |
| Automerge | **Patch** dan **minor devDependency** dengan CI hijau, lewat merge queue. Pengecualian: tidak pernah automerge untuk `fastapi`, `sqlalchemy`, `alembic`, `psycopg`, `pydantic`, `next`, `react`, provider Terraform, base image. Major selalu manual. |
| Batas | Maksimal 5 PR terbuka, 2 PR/jam |
| Dashboard | Dependency Dashboard issue aktif |

### 15.5 Tag & rilis terlindungi

Ruleset tag (§2.5), dan Release hanya dibuat oleh GitHub App release. Perubahan pada `release-please-config.json` dan `.release-please-manifest.json` milik CODEOWNERS platform.

---

## 16. Observability pipeline & DORA

### 16.1 Deployment marker & anotasi rilis

Setiap deploy (staging dan production) mengirim:
- **GitHub Deployment** + status (environment, digest, versi), yang menjadi sumber data DORA.
- **Release di error tracking**: `msp@<versi>` (production) atau `msp@sha-<short>` (staging), commit terkait, environment. Source map web diunggah saat build dan **tidak** disajikan publik (hidden source maps).
- **Anotasi di observability backend**: marker deploy di dashboard SLO (latency, error rate, backlog), sehingga regresi bisa langsung dikaitkan dengan rilis.
- Atribut resource OpenTelemetry di setiap service: `service.name`, `service.version` (= versi aplikasi), `deployment.environment.name`, dan revisi commit (`vcs.ref.head.revision` atau atribut kustom `msp.git_sha`).

### 16.2 DORA (target tahap MVP)

| Metrik | Definisi operasional | Target MVP 1 | Sumber data |
|---|---|---|---|
| Deployment frequency | Deploy production sukses per minggu | ≥ 2/minggu (staging: setiap merge) | GitHub Deployments |
| Lead time for changes | Merge ke `main` → berjalan di production, p50 | ≤ 2 hari kerja | Commit + Deployments |
| Change failure rate | Deploy production yang memicu rollback, hotfix, atau insiden SEV1/2 | ≤ 15% | Deployments + label insiden |
| Failed deployment recovery time (MTTR) | Deteksi → pulih (rollback/flag/fix) | ≤ 1 jam (rollback ≤ 15 menit) | Issue insiden |

Dilaporkan bulanan oleh script yang membaca GitHub API (tanpa vendor tambahan) dan ditinjau di retro engineering.

### 16.3 Kesehatan CI

| Metrik | Target | Tindakan bila meleset |
|---|---|---|
| Durasi pipeline PR | p50 ≤ 10 menit, p90 ≤ 15 menit | Profil job terlambat; shard; larger runner; pindahkan test ke nightly |
| Durasi merge queue | p50 ≤ 15 menit | Kurangi batas grup, tambah concurrency |
| Merge → `staging-verified` | p50 ≤ 30 menit | Paralelkan E2E staging |
| Flaky rate | ≤ 2% run (job yang lulus setelah retry) | Test flaky masuk **karantina** (`@pytest.mark.quarantine` / `test.fixme` dengan tautan issue): tetap jalan di nightly, tidak memblokir PR, maksimal 14 hari, lalu diperbaiki atau dihapus |
| Waktu `main` merah | ≤ 1 jam per kejadian | Stop-the-line (§5.5) |
| Biaya CI | Dipantau bulanan (menit runner, storage artifact/cache) | Retensi artifact default 14 hari, laporan rilis ke S3 |

Data diambil dari GitHub Actions API dan laporan JUnit (artifact), lalu dirangkum di dashboard sederhana (job mingguan).

---

## 17. Contoh workflow (ilustratif)

> Notasi pin: `uses: owner/action@<sha> # vX.Y.Z`. `<sha>` diganti SHA 40 karakter saat bootstrap dan dipelihara Renovate. Contoh dipersingkat: beberapa job dan step ditandai `# …`.

### 17.1 `pr.yml`

```yaml
name: pr
on:
  pull_request:
    types: [opened, synchronize, reopened, ready_for_review, labeled, unlabeled]
  merge_group:
    types: [checks_requested]

permissions: {}                       # default tanpa izin; dibuka per job

concurrency:
  group: pr-${{ github.event.pull_request.number || github.event.merge_group.head_ref }}
  cancel-in-progress: true

jobs:
  changes:
    runs-on: ubuntu-24.04
    permissions: { contents: read, pull-requests: read }
    outputs:
      backend:    ${{ steps.f.outputs.backend }}
      migrations: ${{ steps.f.outputs.migrations }}
      web:        ${{ steps.f.outputs.web }}
      contracts:  ${{ steps.f.outputs.contracts }}
      tokens:     ${{ steps.f.outputs.tokens }}
      e2e:        ${{ steps.f.outputs.e2e }}
      docs:       ${{ steps.f.outputs.docs }}
    steps:
      - uses: actions/checkout@<sha> # vX.Y.Z
        with: { persist-credentials: false, fetch-depth: 0 }
      - id: f
        uses: dorny/paths-filter@<sha> # vX.Y.Z
        with:
          base: ${{ github.event.merge_group.base_sha || github.event.pull_request.base.sha }}
          filters: |
            backend:    ['apps/api/**', 'data/**', 'pyproject.toml', 'uv.lock']
            migrations: ['apps/api/migrations/**', 'apps/api/src/msp/**/models*.py']
            web:        ['apps/web/**', 'packages/**', 'pnpm-lock.yaml', 'turbo.json']
            contracts:  ['contracts/**', 'apps/api/src/msp/**/api/**', 'packages/api-client/**']
            tokens:     ['docs/02-design-system/tokens/**', 'packages/design-tokens/**']
            e2e:        ['apps/**', 'packages/**', 'contracts/**', 'infra/docker/**']
            docs:       ['docs/**', '**/*.md']

  backend-tests:
    needs: changes
    if: needs.changes.outputs.backend == 'true'
    runs-on: ubuntu-24.04
    permissions: { contents: read }
    services:
      postgres:
        image: postgis/postgis:16-3.5@sha256:<digest>   # samakan dengan versi RDS
        env: { POSTGRES_USER: msp, POSTGRES_PASSWORD: msp, POSTGRES_DB: msp_test }
        ports: ["5432:5432"]
        options: >-
          --health-cmd "pg_isready -U msp" --health-interval 5s --health-retries 20
      redis:
        image: redis:7-alpine@sha256:<digest>
        ports: ["6379:6379"]
    env:
      MSP_ENV: test
      MSP_AUTH_MODE: stub
      MSP_DATABASE_URL: postgresql+psycopg://msp:msp@localhost:5432/msp_test
      MSP_REDIS_URL: redis://localhost:6379/0
    steps:
      - uses: actions/checkout@<sha> # vX.Y.Z
        with: { persist-credentials: false }
      - uses: astral-sh/setup-uv@<sha> # vX.Y.Z
        with: { python-version: "3.13", enable-cache: true }
      - run: uv sync --locked
      - name: Migrasi ke head (prasyarat integration test)
        run: uv run alembic -c apps/api/alembic.ini upgrade head
      - name: Unit + integration (QG-04)
        run: >-
          uv run pytest apps/api/tests -n auto
          -m "not slow and not ai_eval and not tenant_isolation"
          --cov=msp --cov-report= --junitxml=reports/junit-main.xml
      - name: Tenant isolation suite (QG-08)
        run: >-
          uv run pytest apps/api/tests -m tenant_isolation
          --cov=msp --cov-append --cov-report= --junitxml=reports/junit-tenant.xml
      - name: Coverage gate (QG-05, QG-06)
        run: |
          uv run coverage report --fail-under=75
          uv run coverage report --fail-under=85 --include="apps/api/src/msp/identity/normalize/*,apps/api/src/msp/identity/resolve/*,apps/api/src/msp/intelligence/scoring/*"
      - if: always()
        uses: actions/upload-artifact@<sha> # vX.Y.Z
        with: { name: backend-reports, path: reports/, retention-days: 14 }

  migrations:
    needs: changes
    if: needs.changes.outputs.migrations == 'true'
    runs-on: ubuntu-24.04
    permissions: { contents: read, pull-requests: read }
    services:
      postgres: { image: "postgis/postgis:16-3.5@sha256:<digest>", env: { POSTGRES_USER: msp, POSTGRES_PASSWORD: msp, POSTGRES_DB: msp_mig }, ports: ["5432:5432"] }
    env:
      MSP_DATABASE_URL: postgresql+psycopg://msp:msp@localhost:5432/msp_mig
    steps:
      # … checkout (fetch-depth: 0), setup-uv, uv sync --locked
      - run: test "$(uv run alembic heads | wc -l)" -eq 1                 # satu head
      - run: uv run alembic upgrade head                                  # dari DB kosong
      - run: uv run alembic check                                         # tak ada perubahan model tertunda
      - run: uv run python scripts/ci/check_rls.py                        # RLS tabel tenant
      - name: Squawk pada SQL migrasi baru
        run: |
          uv run python scripts/ci/render_new_migrations_sql.py --base "${{ github.event.merge_group.base_sha || github.event.pull_request.base.sha }}" > new.sql
          squawk new.sql
      - name: Fase contract hanya dengan label db-contract
        env: { GH_TOKEN: "${{ github.token }}" }
        run: uv run python scripts/ci/check_migration_phase.py --labels "$(scripts/ci/pr_labels.sh)"

  contracts:
    needs: changes
    if: needs.changes.outputs.contracts == 'true'
    runs-on: ubuntu-24.04
    permissions: { contents: read, pull-requests: read }
    env:
      BASE_SHA: ${{ github.event.merge_group.base_sha || github.event.pull_request.base.sha }}
    steps:
      # … checkout (fetch-depth: 0), setup-uv, setup-node 24 + pnpm (cache), install beku
      - name: Spec ter-commit == spec aplikasi
        run: uv run python -m msp.tools.export_openapi > /tmp/openapi.yaml && diff -u contracts/openapi/openapi.yaml /tmp/openapi.yaml
      - run: pnpm exec spectral lint contracts/openapi/openapi.yaml --fail-severity=error   # QG-11
      - name: Breaking change vs base (QG-12)
        env: { GH_TOKEN: "${{ github.token }}" }
        run: |
          git show "$BASE_SHA:contracts/openapi/openapi.yaml" > /tmp/base.yaml
          if ! oasdiff breaking /tmp/base.yaml contracts/openapi/openapi.yaml --fail-on ERR; then
            scripts/ci/require_label.sh api-breaking --with-section "Rencana versi API"
          fi
      - name: Client TS up to date (QG-13)
        run: pnpm --filter @msp/api-client generate && git diff --exit-code packages/api-client
      - name: JSON Schema & contoh payload (QG-13)
        run: |
          uvx check-jsonschema --check-metaschema contracts/events/*.schema.json contracts/input/*.schema.json
          scripts/ci/validate_examples.sh
      # … Schemathesis smoke terhadap API ephemeral (service postgres/redis seperti backend-tests)

  e2e-smoke:
    needs: changes
    if: needs.changes.outputs.e2e == 'true'
    runs-on: ubuntu-24.04
    permissions: { contents: read }
    strategy:
      fail-fast: false
      matrix: { shard: [1, 2] }
    steps:
      - uses: actions/checkout@<sha> # vX.Y.Z
        with: { persist-credentials: false }
      - uses: docker/setup-buildx-action@<sha> # vX.Y.Z
      - name: Build & start stack (cache GHA)
        run: |
          docker buildx bake -f infra/docker/compose.yml -f infra/docker/compose.ci.yml \
            --set '*.cache-from=type=gha' --set '*.cache-to=type=gha,mode=max' --load
          docker compose -f infra/docker/compose.yml -f infra/docker/compose.ci.yml up -d --wait
          docker compose -f infra/docker/compose.yml run --rm api python -m msp.devtools.seed --profile e2e
      # … setup-node 24 + pnpm, playwright install --with-deps chromium
      - run: pnpm --filter web exec playwright test --grep @smoke --shard=${{ matrix.shard }}/2
      - if: failure()
        uses: actions/upload-artifact@<sha> # vX.Y.Z
        with: { name: "playwright-${{ matrix.shard }}", path: apps/web/playwright-report/, retention-days: 7 }

  security:
    runs-on: ubuntu-24.04
    permissions: { contents: read, pull-requests: write }   # komentar dependency review
    steps:
      - uses: actions/checkout@<sha> # vX.Y.Z
        with: { persist-credentials: false, fetch-depth: 0 }
      - run: gitleaks git --log-opts="${{ github.event.pull_request.base.sha || github.event.merge_group.base_sha }}..HEAD" --redact --exit-code 1
      - uses: aquasecurity/trivy-action@<sha> # vX.Y.Z
        with: { scan-type: fs, scanners: "vuln,secret,misconfig", severity: "CRITICAL,HIGH", ignore-unfixed: true, exit-code: "1" }
      - if: github.event_name == 'pull_request'
        uses: actions/dependency-review-action@<sha> # vX.Y.Z
        with:
          fail-on-severity: high
          allow-licenses: MIT, Apache-2.0, BSD-2-Clause, BSD-3-Clause, ISC, MPL-2.0, PSF-2.0, 0BSD, Unlicense, CC0-1.0

  # … backend-lint, web-lint, web-tests, tokens, web-build, docs, workflow-lint (pola sama)

  ci-ok:
    if: always()
    needs: [changes, backend-tests, migrations, contracts, e2e-smoke, security]   # + semua job lain
    runs-on: ubuntu-24.04
    permissions: {}
    steps:
      - name: Gagal bila ada job failure/cancelled
        if: contains(needs.*.result, 'failure') || contains(needs.*.result, 'cancelled')
        run: exit 1
      - run: echo "Semua gate PR lulus atau tidak relevan (skipped)."
```

### 17.2 `main.yml`

```yaml
name: main
on:
  push:
    branches: [main, "release/v*"]

permissions: {}

concurrency:
  group: main-build-${{ github.sha }}
  cancel-in-progress: false

env:
  AWS_REGION: ap-southeast-3
  ECR: ${{ vars.ECR_REGISTRY }}          # <shared-account>.dkr.ecr.ap-southeast-3.amazonaws.com

jobs:
  build-api:
    runs-on: ubuntu-24.04
    permissions: { contents: read, id-token: write }
    outputs:
      image: ${{ steps.out.outputs.image }}
    steps:
      - uses: actions/checkout@<sha> # vX.Y.Z
        with: { persist-credentials: false }
      - uses: aws-actions/configure-aws-credentials@<sha> # vX.Y.Z
        with: { role-to-assume: "${{ vars.AWS_ECR_PUSH_ROLE_ARN }}", aws-region: "${{ env.AWS_REGION }}" }
      - uses: aws-actions/amazon-ecr-login@<sha> # vX.Y.Z
      - uses: docker/setup-buildx-action@<sha> # vX.Y.Z
      - id: build
        uses: docker/build-push-action@<sha> # vX.Y.Z
        with:
          context: .
          file: apps/api/Dockerfile
          push: true
          tags: ${{ env.ECR }}/msp-api:sha-${{ github.sha }}     # immutable
          provenance: mode=max
          cache-from: type=gha,scope=msp-api
          cache-to: type=gha,scope=msp-api,mode=max
          build-args: |
            GIT_SHA=${{ github.sha }}
      - id: out
        run: echo "image=${ECR}/msp-api@${{ steps.build.outputs.digest }}" >> "$GITHUB_OUTPUT"
      - name: Trivy image gate (QG-26)
        uses: aquasecurity/trivy-action@<sha> # vX.Y.Z
        with: { image-ref: "${{ steps.out.outputs.image }}", severity: "CRITICAL,HIGH", ignore-unfixed: true, exit-code: "1" }
      - uses: anchore/sbom-action@<sha> # vX.Y.Z
        with: { image: "${{ steps.out.outputs.image }}", format: spdx-json, output-file: sbom-msp-api.spdx.json }
      - uses: sigstore/cosign-installer@<sha> # vX.Y.Z
      - name: Sign + attest SBOM (keyless) — hanya setelah gate lolos
        env: { IMG: "${{ steps.out.outputs.image }}" }
        run: |
          cosign sign --yes "$IMG"
          cosign attest --yes --type spdxjson --predicate sbom-msp-api.spdx.json "$IMG"

  build-web:
    # … identik dengan build-api: apps/web/Dockerfile, image msp-web, output `image`
    runs-on: ubuntu-24.04
    permissions: { contents: read, id-token: write }
    outputs: { image: "${{ steps.out.outputs.image }}" }
    steps: [ { run: "echo dipersingkat" } ]

  deploy-staging:
    needs: [build-api, build-web]
    uses: ./.github/workflows/_deploy-ecs.yml
    permissions: { contents: read, id-token: write, deployments: write }
    with:
      environment: staging
      api_image: ${{ needs.build-api.outputs.image }}
      web_image: ${{ needs.build-web.outputs.image }}
      version: sha-${{ github.sha }}
      strategy: rolling

  verify-staging:
    needs: deploy-staging
    runs-on: ubuntu-24.04
    environment: staging                     # vars URL & kredensial user sintetis
    permissions: { contents: read }
    strategy:
      fail-fast: false
      matrix:
        suite: [smoke, e2e-1, e2e-2, e2e-3, e2e-4, schemathesis, tenant-isolation]
    steps:
      # … checkout, setup, lalu: scripts/ci/verify_staging.sh "${{ matrix.suite }}" "${{ vars.API_URL }}" "${{ vars.APP_URL }}"
      - run: echo "dipersingkat"

  mark-verified:
    needs: verify-staging
    runs-on: ubuntu-24.04
    permissions: { statuses: write }
    steps:
      - env: { GH_TOKEN: "${{ github.token }}" }
        run: >-
          gh api "repos/${{ github.repository }}/statuses/${{ github.sha }}"
          -f state=success -f context=staging-verified
          -f description="Smoke, E2E, Schemathesis, isolasi tenant lulus di staging"

  notify-failure:
    needs: [build-api, build-web, deploy-staging, verify-staging]
    if: failure()
    runs-on: ubuntu-24.04
    permissions: { issues: write }
    steps:
      - run: echo "buat/ perbarui issue main-red + notifikasi kanal"   # stop-the-line (§5.5)
```

### 17.3 `release.yml` (+ `release-please.yml`)

```yaml
# release-please.yml
name: release-please
on:
  push:
    branches: [main]
permissions: {}
jobs:
  release-please:
    runs-on: ubuntu-24.04
    permissions: { contents: read }
    steps:
      - id: app
        uses: actions/create-github-app-token@<sha> # vX.Y.Z
        with: { app-id: "${{ vars.RELEASE_APP_ID }}", private-key: "${{ secrets.RELEASE_APP_PRIVATE_KEY }}" }
      - uses: googleapis/release-please-action@<sha> # vX.Y.Z
        with:
          token: ${{ steps.app.outputs.token }}       # token App: Release memicu release.yml
          config-file: release-please-config.json
          manifest-file: .release-please-manifest.json
```

```yaml
# release.yml
name: release
on:
  release:
    types: [published]
  workflow_dispatch:
    inputs:
      tag: { description: "Tag rilis, mis. v1.4.0", required: true, type: string }

permissions: {}

concurrency:
  group: release-production
  cancel-in-progress: false

jobs:
  resolve:
    if: startsWith(github.event.release.tag_name || inputs.tag, 'v')     # abaikan design-tokens-v*
    runs-on: ubuntu-24.04
    permissions: { contents: read, statuses: read, id-token: write }
    outputs:
      sha:       ${{ steps.r.outputs.sha }}
      bump:      ${{ steps.r.outputs.bump }}         # patch | minor | major
      api_image: ${{ steps.r.outputs.api_image }}
      web_image: ${{ steps.r.outputs.web_image }}
    env:
      TAG: ${{ github.event.release.tag_name || inputs.tag }}
    steps:
      - uses: actions/checkout@<sha> # vX.Y.Z
        with: { persist-credentials: false, fetch-depth: 0, ref: "${{ env.TAG }}" }
      - uses: aws-actions/configure-aws-credentials@<sha> # vX.Y.Z
        with: { role-to-assume: "${{ vars.AWS_ECR_READ_ROLE_ARN }}", aws-region: ap-southeast-3 }
      - id: r
        env: { GH_TOKEN: "${{ github.token }}" }
        run: |
          SHA="$(git rev-list -n1 "$TAG")"
          scripts/release/await_status.sh "$SHA" staging-verified 45m      # QG-28
          scripts/release/resolve_digests.sh "$SHA" >> "$GITHUB_OUTPUT"     # api_image, web_image dari tag sha-<SHA>
          echo "sha=$SHA" >> "$GITHUB_OUTPUT"
          echo "bump=$(scripts/release/bump_type.sh "$TAG")" >> "$GITHUB_OUTPUT"

  readiness:
    needs: resolve
    runs-on: ubuntu-24.04
    permissions: { contents: read, security-events: read, issues: read }
    steps:
      # … checkout
      - name: Tidak ada Critical/High terbuka (QG-18, QG-26)
        env: { GH_TOKEN: "${{ github.token }}" }
        run: scripts/release/check_open_vulns.sh "${{ needs.resolve.outputs.sha }}"
      - name: ZAP, k6, ER golden, drill restore (QG-29, QG-30, QG-25, QG-34)
        if: needs.resolve.outputs.bump != 'patch'
        run: scripts/release/check_rc_evidence.sh "${{ needs.resolve.outputs.sha }}"
      - name: Bukti RG-1..RG-7 (QG-31)
        if: needs.resolve.outputs.bump == 'major'
        env: { GH_TOKEN: "${{ github.token }}" }
        run: scripts/release/check_release_readiness_issue.sh "${{ github.event.release.tag_name || inputs.tag }}"

  deploy-production:
    needs: [resolve, readiness]
    uses: ./.github/workflows/_deploy-ecs.yml           # job di dalamnya memakai Environment production
    permissions: { contents: read, id-token: write, deployments: write }
    with:
      environment: production
      api_image: ${{ needs.resolve.outputs.api_image }}
      web_image: ${{ needs.resolve.outputs.web_image }}
      version: ${{ github.event.release.tag_name || inputs.tag }}
      strategy: canary                                  # canary 10% → 100% → bake; rollback via alarm

  post-release:
    needs: [resolve, deploy-production]
    runs-on: ubuntu-24.04
    environment: production
    permissions: { contents: read, statuses: write, id-token: write }
    steps:
      # … smoke production (workspace sintetis), status production-deployed,
      #   tambah tag ECR vX.Y.Z ke digest yang sama, finalisasi release error tracking, notifikasi
      - run: echo "dipersingkat"
```

### 17.4 Reusable deploy `_deploy-ecs.yml`

```yaml
name: _deploy-ecs
on:
  workflow_call:
    inputs:
      environment: { type: string, required: true }     # staging | production
      api_image:   { type: string, required: true }     # <ecr>/msp-api@sha256:...
      web_image:   { type: string, required: true }
      version:     { type: string, required: true }
      strategy:    { type: string, required: true }     # rolling | canary

permissions: {}

jobs:
  deploy:
    runs-on: ubuntu-24.04
    environment:
      name: ${{ inputs.environment }}                   # production: required reviewers + tag v*
      url: ${{ vars.APP_URL }}
    concurrency:
      group: deploy-${{ inputs.environment }}
      cancel-in-progress: false
    permissions: { contents: read, id-token: write, deployments: write }
    timeout-minutes: 90
    env:
      CLUSTER: msp-${{ inputs.environment }}
      CERT_ID: '^https://github\.com/sadewadee/MesinProspek/\.github/workflows/main\.yml@refs/heads/(main|release/v.+)$'
    steps:
      - uses: actions/checkout@<sha> # vX.Y.Z
        with: { persist-credentials: false }
      - uses: aws-actions/configure-aws-credentials@<sha> # vX.Y.Z
        with: { role-to-assume: "${{ vars.AWS_DEPLOY_ROLE_ARN }}", aws-region: ap-southeast-3 }
      - uses: sigstore/cosign-installer@<sha> # vX.Y.Z

      - name: Verifikasi tanda tangan & SBOM (QG-27)
        run: |
          for IMG in "${{ inputs.api_image }}" "${{ inputs.web_image }}"; do
            cosign verify "$IMG" --certificate-identity-regexp "$CERT_ID" \
              --certificate-oidc-issuer https://token.actions.githubusercontent.com > /dev/null
            cosign verify-attestation "$IMG" --type spdxjson --certificate-identity-regexp "$CERT_ID" \
              --certificate-oidc-issuer https://token.actions.githubusercontent.com > /dev/null
          done

      - name: Render task definition (api, worker, web, migrate)
        run: scripts/deploy/render_taskdefs.sh "${{ inputs.environment }}" "${{ inputs.api_image }}" "${{ inputs.web_image }}" "${{ inputs.version }}"
        # membaca template infra/deploy/taskdefs/*.json + nilai SSM; menulis out/*.json

      - name: Migrasi DB (one-off ECS task, expand-only)
        run: scripts/deploy/run_oneoff.sh --cluster "$CLUSTER" --taskdef out/migrate.json --command "alembic upgrade head" --timeout 15m

      - name: Deploy worker (rolling + circuit breaker)
        uses: aws-actions/amazon-ecs-deploy-task-definition@<sha> # vX.Y.Z
        with: { task-definition: out/worker.json, service: msp-worker, cluster: "${{ env.CLUSTER }}", wait-for-service-stability: true }

      - name: Deploy api
        uses: aws-actions/amazon-ecs-deploy-task-definition@<sha> # vX.Y.Z
        with: { task-definition: out/api.json, service: msp-api, cluster: "${{ env.CLUSTER }}", wait-for-service-stability: true }
        # strategi (rolling/canary + bake + alarm rollback) dikonfigurasi pada service oleh Terraform per env

      - name: Deploy web
        uses: aws-actions/amazon-ecs-deploy-task-definition@<sha> # vX.Y.Z
        with: { task-definition: out/web.json, service: msp-web, cluster: "${{ env.CLUSTER }}", wait-for-service-stability: true }

      - name: Deployment marker & catatan digest
        if: always()
        run: scripts/deploy/mark_deployment.sh "${{ inputs.environment }}" "${{ inputs.version }}" "${{ job.status }}"
```

---

## 18. Checklist bootstrap

Urutan menyiapkan CI/CD dari nol. Setiap langkah menghasilkan artefak di repo atau di Terraform, kecuali yang ditandai *(manual)*.

1. *(manual)* Pindahkan repo ke **GitHub Organization** dan pastikan plan mendukung fitur di tabel prasyarat di bawah. Buat tim: `web`, `backend`, `data-intel`, `platform`, `design`, `product`, `qa`, `security`, `api-owners`.
2. *(manual)* Setelan repo: squash merge saja, judul PR sebagai default pesan commit, auto-delete branch, nonaktifkan merge commit & rebase merge, aktifkan secret scanning + push protection, Dependabot alerts (sumber data alert), dan code scanning.
3. Commit fondasi: `.github/CODEOWNERS`, `.github/pull_request_template.md`, `.editorconfig`, `.env.example`, `.gitignore`, `release-please-config.json`, `.release-please-manifest.json` (mulai `0.1.0`), `renovate.json`, `.squawk.toml`, `.spectral.yaml`, `.markdownlint-cli2.yaml`, `lychee.toml`.
4. Buat **ruleset** `main` dan ruleset tag (§2.5). Status check wajib diisi setelah langkah 9, karena check harus pernah berjalan dulu.
5. *(manual)* Buat **AWS Organization** dengan akun `msp-shared`, `msp-staging`, `msp-production`. Buat bucket state Terraform (bootstrap kecil, sekali jalan) dan IAM Identity Center + MFA untuk akses manusia.
6. Terraform `envs/shared`: OIDC provider GitHub, ECR `msp-api`/`msp-web` (tag immutable, scan on push), bucket evidence, role `gha-ecr-push`.
7. Terraform `envs/staging` lalu `envs/production`: network, ECS, RDS (PostGIS), ElastiCache, S3, Secrets Manager/SSM, observability + alarm, role OIDC per tujuan (§12.1), role DB (`msp_migrator`, `msp_app`, `msp_readonly`).
8. **GitHub Environments**: `staging`, `production` (required reviewers, tag `v*`), `staging-infra`, `production-infra` (reviewers), `data-staging`, `data-production` (reviewers), `model-production` (data-intel + product), `ai-eval` (secret API key LLM). Isi variables: ARN role, `APP_URL`, `API_URL`, `ECR_REGISTRY`.
9. Workflow pertama: `pr.yml` (dimulai dengan `changes`, lint, test, security, `ci-ok`), `codeql.yml`, `workflow-lint` (actionlint + zizmor). Buka PR uji → aktifkan required checks `ci-ok`, `CodeQL`, `pr-title`.
10. Aktifkan **merge queue** pada ruleset `main` dan tambahkan trigger `merge_group` (sudah ada di `pr.yml`).
11. Buat GitHub App **release** (izin contents + pull requests) → simpan private key sebagai secret → `release-please.yml`.
12. `main.yml` + `_deploy-ecs.yml`: build, scan, SBOM, sign, deploy staging, verifikasi, `staging-verified`.
13. `release.yml` + `rollback.yml`. Lakukan **rilis uji** `v0.1.0` ke production (halaman kosong/health saja), termasuk uji rollback.
14. `infra.yml` (plan di PR, apply dengan approval) + drift detection.
15. Renovate self-hosted (`renovate.yml`, token GitHub App terpisah dengan izin minimal) + Dependency Dashboard.
16. `nightly.yml` bertahap: regresi penuh → ZAP → k6 (ECS task) → data quality → rescan image → Lighthouse.
17. `restore-drill`: jalankan sekali secara manual sebelum menjadwalkannya.
18. Flag registry + kill switch + job `flag-hygiene`.
19. Integrasi observability: marker deploy, release error tracking, dashboard DORA & kesehatan CI.
20. Dokumentasikan runbook (deploy, rollback, incident + hook PDP, drill) dan latih on-call sebelum closed beta.

**Prasyarat platform GitHub (perlu diverifikasi terhadap plan aktual):**

| Fitur | Dipakai untuk | Ketersediaan untuk repo privat (dokumentasi GitHub saat penulisan) | Fallback |
|---|---|---|---|
| Teams / CODEOWNERS tim | §2.3 | Butuh Organization | Username individual |
| Merge queue | §2.6 | Organization + plan Enterprise Cloud untuk repo privat | Require up-to-date + auto-merge |
| Environment required reviewers | §6.2, §11, §10 | Plan berbayar. Beberapa protection rule untuk repo privat hanya di Enterprise | Workflow dispatch terbatas + approval via Release PR. Two-person rule ditegakkan lewat review. |
| Code scanning (CodeQL), dependency review | QG-18, QG-19 | GitHub Code Security (add-on) | Semgrep CE + Trivy + osv-scanner |
| Secret scanning + push protection | QG-17 | GitHub Secret Protection (add-on) | gitleaks di CI + pre-commit |
| Artifact attestations | §15.3 | Repo privat butuh Enterprise Cloud | Provenance BuildKit + cosign (default dokumen ini) |

---

## 19. Pertanyaan terbuka

ID `Q-CI-nn` bersifat lokal di dokumen ini. Setelah diputuskan, keputusan dipindahkan ke register 00 §7.

| ID | Pertanyaan | Default sementara | Dampak |
|---|---|---|---|
| Q-CI-01 | Plan GitHub dan kepemilikan repo (akun personal vs Organization; Team vs Enterprise Cloud; add-on Code Security/Secret Protection)? | Organization + plan yang mendukung merge queue dan environment reviewers. Bila tidak, pakai fallback §18. | Merge queue, approval production, CodeQL, push protection |
| Q-CI-02 | Vendor cloud final. D-14 menetapkan region Jakarta, AWS ap-southeast-3 baru diusulkan. Ketersediaan fitur ECS (strategi canary/blue-green bawaan) dan layanan lain di region ini perlu dikonfirmasi. | AWS ap-southeast-3 | Seluruh §3, §5–§7, §11, §12 |
| Q-CI-03 | Vendor observability & error tracking (managed OTel backend, Sentry SaaS vs self-hosted). Residensi data telemetry: apakah log/trace (sudah diredaksi) boleh disimpan di luar Indonesia? | Alarm rollback di CloudWatch (vendor-agnostik). Error tracking dengan scrubbing PII. | §6.3, §16, NFR-PRIV-03 |
| Q-CI-04 | Signing keyless (identitas workflow tercatat di transparency log publik) vs kunci KMS? | Keyless | §15.3 |
| Q-CI-05 | Preview environment: kapan dibutuhkan, dan dalam bentuk apa (web + MSW vs namespace staging)? | Ditunda (§3.2) | Kecepatan review UI |
| Q-CI-06 | Lisensi dataset: apakah subset boleh dipakai di staging dan untuk eval AI (Q-05)? | Hanya sintetis sampai dikonfirmasi | Kualitas rehearsal §10.2, QG-25 penuh |
| Q-CI-07 | Ukuran tim dan rotasi on-call: siapa reviewer Environment production dan penerima alert di luar jam kerja? | Tech lead + 1 engineer senior | §6.2, §14.5 |
| Q-CI-08 | Nama domain production/staging dan penyedia DNS/sertifikat | `<domain>` placeholder. ACM + Route 53 | URL §3.1, CORS, callback auth |
| Q-CI-09 | Penyedia managed auth (FR-AUTH-001): dukungan tenant dev/staging terpisah dan mode stub untuk CI | Tenant terpisah per environment. Stub hanya di local/CI | §3.3, E2E |
| Q-CI-10 | Threshold QG-33 (validasi snapshot) dan alarm DLQ/latency awal | Nilai default di §6.3 dan §10.2 | Kalibrasi setelah Q-01/Q-02 dan 4 minggu data beta |
