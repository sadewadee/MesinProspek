# 06 — CI/CD

## MesinProspek — Branching, Pipeline, Quality Gate, Environment, Rilis

| Field | Value |
|---|---|
| Dokumen | CI/CD |
| Versi | 0.1 (Draft) |
| Tanggal | 2026-10-10 |
| Basis | `README.md` (struktur monorepo), `00-analisis-dokumen-awal.md` (D-11, D-12, D-14), `01-prd.md` (§10 NFR, §12 release gate & DoD), `04-technical-design.md` (ADR-18, ADR-21, ADR-22, §18), `05-test-strategy.md` (§3), MVP Pack v1.2 §11, §14, §15 |
| Keputusan | **D-18** platform CI/CD = GitLab CE self-hosted (Proposed) · **D-19** hosting & runtime = dedicated server di DC Indonesia (dedicated: Accepted; stack: Proposed) · **D-21** backend = **Go** (Accepted) |
| Menggantikan | Baris README *Repo*, *Backend*, *Worker*, *Python tooling*, *CI/CD*, *Hosting* (GitHub, GitHub Actions, AWS, Python/FastAPI). README sudah diperbarui. |
| Audience | Engineering, DevOps/Platform, QA, Tech lead |
| Menutup | G-17 (strategi CI/CD, environment, migrasi DB) |

> Konvensi: `QG-nn` = quality gate (§8). Contoh YAML, Ansible, dan konfigurasi bersifat **ilustratif**, bukan file final. Jam ditulis dalam WIB (UTC+7); schedule GitLab diset ke zona `Asia/Jakarta`. Path package Go mengikuti layout 04 §3.1: logika domain berada di `apps/api/internal/<modul>/internal/<paket>` (mis. `internal/identity/internal/normalize`, `internal/identity/internal/er`, `internal/intelligence/internal/scoring`); bila berbeda, yang berlaku 04.

---

## 1. Prinsip & keputusan

| # | Prinsip | Konsekuensi konkret |
|---|---|---|
| P-1 | **Trunk-based development** | Satu branch jangka panjang: `main`. Branch fitur berumur ≤ 2 hari (maks 5). Pekerjaan yang belum selesai masuk `main` di balik feature flag (§13). |
| P-2 | **Build once, deploy many** | Image dibangun **sekali** per commit `main` dan dipromosikan ke staging lalu production dengan **digest yang sama**. Satu image backend dipakai untuk container `api`, `worker`, dan `migrate`. Versi dan konfigurasi environment disuntikkan saat runtime (§12). |
| P-3 | **Everything as code** | `.gitlab-ci.yml`, `infra/ansible/`, secret terenkripsi SOPS, setelan proyek GitLab (script API), migrasi SQL (goose), query (sqlc), kontrak (`contracts/`), design tokens, konfigurasi versi skor/geo/prompt. Perubahan manual di server dianggap drift (§11.4). |
| P-4 | **Fast feedback** | Pipeline MR **≤ 10 menit p50** (p90 ≤ 15) lewat `rules:changes`, DAG `needs:`, cache `GOCACHE`/`GOMODCACHE`/pnpm/Turborepo, dan BuildKit registry cache. |
| P-5 | **Shift-left security** | Secret scanning (CI + server hook), SAST, govulncheck, audit dependency, license policy, dan scan IaC berjalan di MR. Image yang gagal scan tidak ditandatangani, sehingga tidak bisa di-deploy (§5, §15). |
| P-6 | **Rilis data & model reproducible** | `ds-YYYY.MM.n`, `priority-X.Y.Z`, `geo-X.Y.Z`, `brief-X.Y.Z` dirilis lewat pipeline dengan gate dan approval, terpisah dari rilis kode. Versi terbit immutable (NFR-DATA-01, FR-INTL-009). |
| P-7 | **Data tetap di zonanya** | Seluruh infrastruktur (kode, CI, registry, runtime, backup) berada di Indonesia (D-14). Runner MR (`ci-01`) hanya memproses kode dan **data sintetis**, dan tidak punya rute ke zona staging/production. Dataset berlisensi dan data production hanya diproses di host environment-nya. |
| P-8 | **Rollback murah, forward-fix untuk data** | Rollback aplikasi = kembalikan traffic ke warna sebelumnya atau deploy ulang digest sebelumnya (≤ 5–15 menit). Skema DB tidak di-rollback: migrasi selalu kompatibel dengan versi N-1 (§9). |
| P-9 | **Gate production = protected tag** | Setiap operasi yang mengubah production (rilis aplikasi, apply infra, publish dataset, aktivasi model) berjalan di pipeline **protected tag** yang hanya bisa dibuat Maintainer. Job manual di pipeline tersebut hanya bisa dijalankan Maintainer (§2.5). |

### 1.1 Alur besar

```
 feat/* fix/* chore/* docs/*
        │  merge request
        ▼
 ┌──────────────────────────────────┐   review + approval   ┌───────────────────────────────┐
 │ MR pipeline (rules:changes, DAG) │ ────────────────────► │ mr:gate (manual, blocking):   │
 │ lint·test·fuzz·migrasi·sqlc·     │                       │ approval owner, label, thread │
 │ kontrak·tokens·a11y·E2E·security │                       └───────────────┬───────────────┘
 └──────────────────────────────────┘     rebase + squash + fast-forward    │
                                                                            ▼
                                                                          main
 ┌──────────────────────────────────────────────────────────────────────────────────────────┐
 │ main: build msp-backend & msp-web → Trivy → SBOM → cosign sign → deploy staging (Ansible,│
 │   blue/green Traefik) → verifikasi → `staging-verified` → [rc:start → `rc-verified`]     │
 │   → release:cut (manual, semantic-release) → tag vX.Y.Z + GitLab Release                 │
 └──────────────────────────────────────────────────────────────────────────────────────────┘
                                                                            │ protected tag
 ┌──────────────────────────────────────────────────────────────────────────▼───────────────┐
 │ tag vX.Y.Z: resolve digest → readiness → deploy:production (manual, Maintainer)          │
 │   → verify signature → msp migrate up → canary 10% → 100% → bake → auto-rollback         │
 └──────────────────────────────────────────────────────────────────────────────────────────┘
 Paralel: scheduled pipelines (nightly, drift, Renovate, drill), tag `infra-*`, `ds-*`,
          `priority-*`/`geo-*`/`brief-*` (§10, §11), job rollback (§14)
```

### 1.2 Register keputusan

| ID | Sub-keputusan | Alasan singkat |
|---|---|---|
| **D-18** | **Platform CI/CD = GitLab CE self-hosted** (Proposed) | Tidak memakai GitHub (arahan founder). Satu paket: code host, MR, CI, registry, schedule. |
| D-18.1 | GitLab CE (Omnibus) di `ops-01` adalah **repo utama**. Mirror ke GitHub opsional (Q-CI-01). UI/git hanya lewat WireGuard. | Residensi kode, kontrol penuh |
| D-18.2 | GitLab CI: MR pipelines, `rules:changes`, `needs:` (DAG), `resource_group`, scheduled pipelines, `environment` untuk pencatatan deploy | Fitur inti tersedia di CE |
| D-18.3 | Runner **Docker executor** di `ci-01`: `ci-docker` (non-privileged, job MR) dan `ci-build` (BuildKit rootless). Runner **`deploy`** berstatus *protected* di `ops-01`, hanya mengambil job dari ref terlindungi. | Kode MR yang belum direview tidak pernah berjalan di host yang punya rute ke production |
| D-18.4 | **GitLab Container Registry**. Deploy selalu by digest. Cleanup: image MR 7 hari, `sha-*` 90 hari, `v*` permanen. | Satu sistem dengan CI |
| D-18.5 | Merge **squash + fast-forward** (rebase wajib), *Pipelines must succeed*, *All threads resolved*. Approval ditegakkan job manual blocking **`mr:gate`**. | CE tidak punya required approvals, Code Owners, atau merge train |
| D-18.6 | Gate production = **protected tags** (`v*`, `infra-*`, `ds-*`, `priority-*`, `geo-*`, `brief-*`; dibuat Maintainer saja) + **job manual** di pipeline tag | Pengganti protected environments (Premium) |
| D-18.7 | Rilis: **semantic-release** + `@semantic-release/gitlab`, dipicu manual (`release:cut`), tanpa commit balik ke repo | release-please berorientasi GitHub. Tag harus menunjuk commit yang sudah dibangun dan diverifikasi. |
| D-18.8 | Security: **Semgrep CE** + **gosec** (via golangci-lint) untuk SAST, **govulncheck**, **gitleaks** (CI + pre-receive hook), **Trivy** (fs, image, config), **OSV-Scanner** + **pnpm audit** + license allowlist, **OWASP ZAP** di staging | Pengganti CodeQL, Dependabot, dependency review |
| D-18.9 | Supply chain: **syft** (SBOM SPDX), **cosign key pair** (private key di SOPS, `--tlog-upload=false`), provenance BuildKit, image **`ci-tools`** buatan sendiri dengan versi tool di-pin | Tidak ada keyless OIDC. Tidak ada metadata ke transparency log publik. |
| D-18.10 | **Renovate self-hosted** sebagai scheduled pipeline | Tidak ada Dependabot |
| **D-19** | **Hosting & runtime = dedicated server di DC Indonesia** (dedicated: Accepted; stack: Proposed; = ADR-18) | Residensi (D-14), biaya |
| D-19.1 | Topologi: `app-01` (kelak `app-02`), `db-01` (primary), `db-02` (hot standby + read replica), `stor-01` (SeaweedFS), `ops-01` (GitLab, registry, monitoring, Tang), `ci-01` (runner), `stg-01` (staging), `dr-01` (target drill/DR, boleh sewa per jam) | Selaras 04 §2.2 |
| D-19.2 | **Docker Engine + Docker Compose** + **Traefik v3** (Let's Encrypt). Deploy **blue/green** dengan *weighted service* Traefik (canary 10% → 100%). | Sederhana untuk 1–2 app server |
| D-19.3 | **PostgreSQL 16 + PostGIS** bare metal (PGDG) + PgBouncer (transaction mode, hanya untuk trafik request `api`; client River di worker dan `msp migrate` terhubung langsung ke `db-01:5432` dengan pool kecil khusus, fallback River `PollOnly`; 04 §17.4), streaming replication ke `db-02`, failover manual via playbook | Patroni saat ada node ketiga |
| D-19.4 | **Redis** di `app-01` (opsional: cache, rate limit, counter; bukan sumber kebenaran). Object storage **SeaweedFS** (master + volume + filer + S3 gateway, Apache-2.0) di `stor-01`. Alternatif: **Garage**. | Antrean job ada di Postgres (River, ADR-06). MinIO community edition sudah diarsipkan (tanpa binary/image sejak Okt 2025). |
| D-19.5 | **Ansible**: hardening OS, user, nftables, **WireGuard** (akses admin/SSH), fail2ban, unattended security upgrades, Docker, layanan. Disk **LUKS** dengan unlock Clevis + Tang (fallback dropbear-initramfs). | Satu alat untuk provisioning, konfigurasi, dan deploy |
| D-19.6 | Secret: **SOPS + age** di repo, didekripsi Ansible saat deploy ke **tmpfs** (ADR-22). Kredensial deploy di **GitLab CI/CD variables masked + protected**, di-scope per environment. | Tanpa cloud secret manager |
| D-19.7 | Backup/DR: **pgBackRest** (full mingguan, diff harian, WAL kontinu `archive_timeout = 60s`) ke repo lokal (`stor-01`) **dan** off-site S3-compatible di lokasi lain di Indonesia. Replikasi object storage off-site (`weed filer.sync`/`weed filer.backup`). Backup GitLab harian. Drill restore bulanan. | NFR-AVAIL-02 |
| D-19.8 | Observability di `ops-01`: Prometheus, Alertmanager, Grafana, Loki, Tempo (via OpenTelemetry Collector), **GlitchTip**, blackbox exporter + **probe eksternal** di luar DC | ADR-21. Probe eksternal mendeteksi DC/jaringan down. |
| D-19.9 | Zona jaringan `edge/prod`, `data/prod`, `ops`, `ci`, `stg` (VLAN + nftables). Port publik hanya 80/443 (`app-01`) dan UDP WireGuard (`ops-01`). | Blast radius kecil |
| D-19.10 | Docker Swarm/k3s **bukan** bagian MVP. Ditinjau bila > 2 app server atau dibutuhkan autoscaling. | Hindari kompleksitas dini |
| **D-21** | **Backend = Go** (Accepted). Frontend tetap Next.js + TypeScript (Node 24, pnpm, Turborepo). | Keputusan founder |
| D-21.1 | `apps/api/` = satu Go module. Binary statis **`cmd/msp`** dengan subcommand `api`, `worker`, `migrate`, dan `tool …` (04 §3.1). Aksi `tool` yang dipanggil pipeline (`healthcheck` untuk healthcheck container distroless, `snapshot-publish`, `score-model-activate`, `score-reproduce`; §10) sudah tercantum di 04 §3.1. | Satu image, beda command per container |
| D-21.2 | Akses DB: **sqlc** (query di `apps/api/db/queries/`, kode hasil generate di `apps/api/internal/db/<modul>db/`), driver pgx v5 (ADR-26) | Query SQL eksplisit, type-safe |
| D-21.3 | Migrasi: **goose** (SQL di `apps/api/db/migrations/`, di-embed ke binary, dijalankan `msp migrate up`; ADR-26) | Migrasi ikut image yang sama (build once) |
| D-21.4 | Job queue + outbox: **River** (Postgres-backed, ADR-06/07). Tabel River dimigrasi oleh `msp migrate up` (rivermigrate) sebelum migrasi aplikasi. | Transaksional dengan data, tanpa broker terpisah |
| D-21.5 | Server API dari kontrak: **oapi-codegen** strict-server (ADR-08) ke `apps/api/internal/api/`. Client web: openapi-typescript ke `packages/api-client`. | Contract-first (ADR-08) |
| D-21.6 | Build `CGO_ENABLED=0` → binary statis → image **distroless/static** (nonroot). Konsekuensi: library berbasis C (mis. libpostal, GEOS) tidak dipakai; komputasi geo berat dilakukan di PostGIS. | Image kecil, permukaan serangan kecil |

### 1.3 Batasan GitLab CE & mitigasi

| Fitur yang tidak ada di CE (tier berbayar) | Mitigasi |
|---|---|
| Required MR approvals, approval rules, Code Owners approval | Job manual blocking `mr:gate` yang membaca approval via API + `.gitlab/CODEOWNERS` (§2.3) |
| Merge trains, merged results pipelines | Fast-forward merge (rebase wajib): pipeline terakhir selalu berjalan di atas `main` terbaru (§2.6) |
| Protected environments, deployment approvals | Protected tags + job manual (hanya Maintainer) + `resource_group` + cek *four-eyes* opsional (§6.2) |
| Push rules (blok secret) | **Server hook** `pre-receive` gitleaks (tersedia di self-managed) |
| Security dashboard, MR security widget | Job gagal + laporan sebagai artifact + komentar MR oleh script |
| Audit events lengkap, DORA dashboard | Log GitLab ke Loki. Deployment via `environment`. DORA dihitung script → Grafana (§16) |

> Sebelum bootstrap, verifikasi tier setiap fitur CE yang diandalkan (*protected runners*, *environment-scoped variables*, *server hooks*, *cleanup policies*, *project access tokens*) pada versi GitLab yang dipasang (§18 langkah 6).

### 1.4 Alternatif platform CI (pemetaan singkat)

| Kebutuhan | **GitLab CE** (dipilih) | Forgejo/Gitea + Actions (act_runner) | Woodpecker CI (+ Forgejo/Gitea) | Jenkins |
|---|---|---|---|---|
| Definisi pipeline | `.gitlab-ci.yml` | `.forgejo/workflows/*.yml` (mirip GitHub Actions) | `.woodpecker/*.yml` | `Jenkinsfile` (Groovy) |
| Path filter / DAG | `rules:changes` / `needs` | `on.paths` (per workflow) / `needs` | `when.path` / `depends_on` | `changeset` / `parallel` |
| Registry | Built-in | Package registry OCI built-in | Dari forge | Eksternal (Harbor/Distribution) |
| Gate production | Protected tag + job manual | Tag protection + `workflow_dispatch` (tanpa reviewer environment) | Event `deployment`/promote | Step `input` + matrix authorization |
| Secret | Variable masked/protected (+ scope env) | Secret repo/org | Secret + filter event | Credentials plugin |
| Catatan | Paling lengkap dalam satu paket. Berat (RAM ≥ 8 GB) | Ringan. Kompatibilitas Actions tidak 100% | Ringan, sederhana. Fitur gate minim | Fleksibel. Beban maintenance plugin tinggi |

Migrasi ke alternatif cukup murah: logika pipeline berada di `scripts/ci/`, `scripts/release/`, dan playbook Ansible, sedangkan YAML CI hanya mengorkestrasi.

### 1.5 Tooling tambahan (Proposed)

| Area | Pilihan | Alasan |
|---|---|---|
| Lint Go | golangci-lint (staticcheck, gosec, errcheck, revive, **depguard** untuk batas modul), `gofmt`, `goimports`, `go vet` | Satu runner lint. depguard menggantikan import-linter (ADR-01) |
| Test Go | `go test -race`, **gotestsum** (JUnit), **gocover-cobertura** (laporan coverage GitLab), fuzz native Go | Laporan test & coverage tampil di MR |
| Tool Go | Di-pin lewat directive `tool` di `apps/api/go.mod` (`go tool oapi-codegen`, `govulncheck`, `gotestsum`, `gocover-cobertura`). sqlc dan golangci-lint (binary resmi) di image `ci-tools`. | Versi tunggal, diperbarui Renovate |
| Rilis | semantic-release (`commit-analyzer`, `release-notes-generator`, `@semantic-release/gitlab`) + commitlint | D-18.7 |
| Kontrak | Spectral (+ `spectral-owasp-ruleset`), oasdiff, ajv-cli (JSON Schema 2020-12) | Lint, breaking change, validasi contoh payload |
| Lint SQL migrasi | squawk | Operasi berbahaya PostgreSQL (lock, index non-concurrent, rename/drop) |
| Docs, Ansible | markdownlint-cli2, lychee; ansible-lint, yamllint, Molecule (role kritis) | QG-21, QG-22 |
| Build, metrik CI, web perf | BuildKit rootless (tanpa runner privileged); gitlab-ci-pipelines-exporter → Prometheus; Lighthouse CI nightly (proxy lab NFR-PERF-03) | §5.1, §16.3 |
| Feature flag | `platform.feature_flag` (04 §18.3) + registry di repo | Audit wajib (FR-OPS-007). Fitur feature flag GitLab tidak dipakai. |

Python hanya muncul sebagai runtime alat pihak ketiga yang dijalankan dalam container (Ansible, Semgrep, Schemathesis), bukan di kode aplikasi.

---

## 2. Branching & merge request workflow

### 2.1 Model branch

| Branch | Fungsi | Umur | Dibuat oleh |
|---|---|---|---|
| `main` | Trunk, selalu dapat dirilis | Permanen | — |
| `feat/`, `fix/`, `chore/`, `docs/` + `<scope>-<deskripsi>` | Fitur, perbaikan, maintenance/CI, dokumentasi. Contoh `feat/srch-zero-result-guidance` | ≤ 2 hari (maks 5) | Engineer/Design/Product |
| `renovate/*` | Update dependency | Sampai merge | Bot Renovate |
| `release/vX.Y` | **Hanya** hotfix saat `main` tidak bisa dirilis (§6.6) | Sampai hotfix rilis | Maintainer |

Tidak ada branch `develop`. Fitur besar dipecah menjadi beberapa MR kecil di balik flag `rel.*`. Source branch dihapus otomatis setelah merge.

### 2.2 Conventional Commits

Merge memakai **squash** dengan template squash `%{title}` + baris `MR: %{reference}`. Akibatnya **judul MR = pesan commit** di `main`, dan divalidasi commitlint (QG-01).

```
<type>(<scope>)!: <ringkasan imperatif, huruf kecil>
type  : feat | fix | perf | refactor | test | docs | chore | ci | build | revert
scope : web | api | worker | platform | ingest | identity | intelligence | geo | query | action |
        ai | usage | ops | db | contracts | tokens | ui | infra | data | docs | deps
!     : breaking change (wajib label `api-breaking` bila menyangkut kontrak)
```

Contoh: `feat(intelligence): tambah faktor geo pada priority score` → minor; `fix(action): retry webhook memakai backoff eksponensial` → patch; `feat(contracts)!: ganti format cursor pagination /v1/prospects` → major; `chore(deps): update golangci-lint` → tidak memicu rilis.

Changelog di GitLab Release menautkan setiap baris ke MR, dan deskripsi MR memuat ID requirement (`Refs: FR-…`).

### 2.3 Review, CODEOWNERS, dan `mr:gate`

`.gitlab/CODEOWNERS` memakai format GitLab (langsung berlaku bila kelak pindah ke tier berbayar). Di CE, file ini dibaca `mr:gate` dan dipakai untuk menyarankan reviewer.

```
# .gitlab/CODEOWNERS (ilustratif). Sintaks GitLab: SATU pola path per baris, diikuti owner (@user, @grup, email).
# Tanpa header [Section], pola terakhir yang cocok menang → pola umum ditulis lebih dulu, pola spesifik sesudahnya.
*                                     @msp/backend @msp/web
/docs/                                @msp/product
/docs/02-design-system/               @msp/design
/apps/web/                            @msp/web
/packages/ui/                         @msp/web @msp/design
/packages/design-tokens/              @msp/design @msp/web
/apps/api/                            @msp/backend
/apps/api/internal/identity/          @msp/data-intel
/apps/api/internal/intelligence/      @msp/data-intel
/apps/api/internal/ai/                @msp/data-intel
/apps/api/internal/platform/          @msp/backend @msp/security
/apps/api/db/migrations/              @msp/backend @msp/platform
/contracts/                           @msp/api-owners
/data/                                @msp/data-intel
/infra/                               @msp/platform
/.gitlab-ci.yml                       @msp/platform
/ci/                                  @msp/platform
/scripts/ci/                          @msp/platform
/scripts/release/                     @msp/platform
```

Bila kelak dibutuhkan owner berlapis per area (mis. security wajib ikut untuk `platform`), file dapat dipecah dengan header section GitLab (`[Security]`, `[Data]`); setiap section dievaluasi terpisah dan `mr:gate` mewajibkan approval dari setiap section yang cocok. Versi lengkap (path paket kritis per modul) ada di `08-engineering-standards.md` §13.1.

Job **`mr:gate`** (manual, `allow_failure: false`) membuat pipeline berstatus *blocked*. Dengan *Pipelines must succeed*, MR **tidak bisa di-merge** sebelum job ini dijalankan dan lulus. Script memeriksa:
1. ≥ 1 approval dari owner setiap area yang tersentuh. **2 approval** untuk path berisiko (tabel di bawah) saat tim ≥ 4 engineer.
2. Approver ≠ author, dan approval diberikan **setelah commit terakhir** (dicek dari system note, karena CE tidak me-reset approval saat ada push).
3. Semua thread resolved, MR bukan draft.
4. Label wajib ada bila relevan (`api-breaking` saat oasdiff ERR, `db-contract` saat migrasi contract).
5. Pengecualian: MR Renovate berlabel `automerge` (§15.4).

Reviewer atau author menjalankan `mr:gate` setelah approval. Token yang dipakai bersifat read-only (`read_api`, role Reporter).

| Path berisiko (2 approval saat tim ≥ 4) | Alasan |
|---|---|
| `apps/api/db/migrations/**` dengan label `db-contract` | Operasi destruktif (§9) |
| `apps/api/internal/platform/**` (auth, tenancy, RLS) | Isolasi tenant (NFR-SEC-03) |
| `infra/**` | Server & secret |
| `.gitlab-ci.yml`, `ci/**`, `scripts/ci/**`, `scripts/release/**` | Supply chain (§15) |
| `contracts/**` dengan label `api-breaking` | Kontrak publik |

### 2.4 MR template (`.gitlab/merge_request_templates/Default.md`)

```markdown
## Ringkasan            <!-- Apa dan kenapa. Judul MR mengikuti Conventional Commits. -->
## Requirement          Refs: FR-___ / NFR-___ / BR-___      Test: TC-___ (05)
## Checklist
- [ ] Test ditambah/diubah sesuai level di 05; tidak ada data pribadi nyata di test/log/screenshot
- [ ] Migrasi: fase `expand`/`contract` dicatat, kompatibel dengan versi sebelumnya (§9); tabel tenant baru ber-RLS
- [ ] Kontrak: OpenAPI/JSON Schema diperbarui; stub Go & client TS di-regenerate; breaking → label `api-breaking`
- [ ] Feature flag terdaftar di registry (nama, tipe, owner, kedaluwarsa) (§13)
- [ ] Skor/geo/prompt: versi dinaikkan, laporan regresi/eval terlampir (§10)
- [ ] UI: screenshot sebelum-sesudah (1280 px + 360 px), copy id-ID & en-US
- [ ] Event analytics & audit dipasang; runbook diperbarui (bila perlu)
## Catatan migrasi / rollout
## Rencana versi API (wajib bila `api-breaking`)
```

### 2.5 Protected branch, protected tag, merge check

Setelan dikelola sebagai kode oleh `scripts/bootstrap/gitlab_settings` (GitLab API) dan diperiksa ulang oleh job `drift`.

| Setelan | Nilai |
|---|---|
| `main`: allowed to push | **No one** (termasuk force push) |
| `main`: allowed to merge | Developers + Maintainers (approval ditegakkan `mr:gate`) |
| `release/v*` | Push/merge: Maintainers |
| Merge method | **Fast-forward** + squash wajib. Template squash `%{title}` |
| Merge checks | Pipelines must succeed. *Skipped pipelines are considered successful* = **off**. All threads resolved. |
| Protected tags (create: Maintainers saja) | `v*` dan `design-tokens-v*` (juga bot rilis), `infra-*`, `ds-*`, `priority-*`, `geo-*`, `brief-*` |
| Runner `deploy` | *Protected*: hanya menjalankan job dari ref terlindungi |
| CI/CD variables | Kredensial: masked + protected + scope environment (§12.2) |
| Lain-lain | Sign-up nonaktif, 2FA wajib, proyek *private*, *public pipelines* off, CI job token scope dibatasi |

**Siapa bisa men-deploy production:** job manual di pipeline protected tag hanya bisa dijalankan oleh user yang berhak membuat tag tersebut, yaitu **Maintainer**. Jumlah Maintainer dibatasi (tech lead + 1–2 engineer senior, Q-CI-07).

### 2.6 Pengganti merge queue

Dengan **fast-forward merge**, MR wajib di-rebase ke `main` terbaru sebelum merge, sehingga pipeline terakhir MR (termasuk `mr:gate`) berjalan di atas commit yang persis akan menjadi `main`. Biayanya: rebase berulang saat `main` sibuk. Mitigasinya adalah MR kecil dan pipeline cepat. Bila antrean rebase menjadi bottleneck (> 5 MR/hari saling menunggu), evaluasi GitLab Premium (merge trains, Q-CI-02).

`rules:changes` pada MR dibandingkan dengan `refs/heads/main` (`compare_to`). Job `mr:title`, `mr:gate`, dan keamanan **selalu** berjalan, sehingga setiap MR punya pipeline.

### 2.7 Label yang dibaca pipeline

| Label | Efek |
|---|---|
| `api-breaking` | Mengizinkan breaking change OpenAPI (QG-13) bila bagian "Rencana versi API" terisi |
| `db-contract` | Mengizinkan migrasi fase contract (QG-11). 2 approval saat tim ≥ 4 |
| `ai-eval` | Memaksa AI eval live, mis. saat mengganti model provider (D-15) |
| `e2e-full` | Menjalankan E2E penuh di MR |
| `automerge` | Hanya untuk MR Renovate (§15.4) |

Label dibaca dari `$CI_MERGE_REQUEST_LABELS`. Setelah label ditambahkan, job terkait cukup di-retry.

### 2.8 Tag & versi

| Tag (protected) | Dibuat oleh | Memicu |
|---|---|---|
| `vX.Y.Z` | semantic-release (`release:cut`, token bot Maintainer) | Pipeline rilis production (§6) |
| `design-tokens-vX.Y.Z` | `release:cut` bila versi paket tokens berubah | Penanda SemVer paket (tanpa deploy) |
| `infra-YYYY.MM.DD.n` | Maintainer | Apply Ansible production (§11.2) |
| `ds-YYYY.MM.n` | Maintainer (data owner) | Pipeline publish snapshot (§10.2) |
| `priority-X.Y.Z`, `geo-X.Y.Z`, `brief-X.Y.Z` | Maintainer | Pipeline aktivasi versi di production (§10.3–§10.4) |

Seri `0.x` dipakai selama MVP 0 dan closed beta. `v1.0.0` = rilis MVP 1 yang lolos RG-1 s.d. RG-7 (QG-32).

---

## 3. Environments & topologi

### 3.1 Ringkasan environment

| Env | Tujuan | Data | Trigger deploy | Akses | URL | Sumber config |
|---|---|---|---|---|---|---|
| **local** | Pengembangan | Seed **sintetis** + fixture EC-xx dari 05 | `docker compose up` | Engineer | `http://localhost:3000` (web, `/api` → api), `:8025` Mailpit, `:8888` filer UI SeaweedFS | `.env` dari `.env.example` |
| **ci** | Verifikasi MR & main | Sintetis per job, dibuang | Otomatis | Pipeline | — (`services:` ephemeral di `ci-01`) | Variabel job |
| **preview** | — | — | **Ditunda** (§3.2) | — | — | — |
| **staging** | Prod-like: E2E penuh, DAST, k6, rehearsal rilis data/model, drill | Dataset sintetis 5 juta establishment (k6, 04 §18.2) + subset dataset **tanpa person/kontak pribadi** bila lisensi mengizinkan (Q-05). Tanpa PII customer, tanpa salinan DB production. | Otomatis per merge ke `main` | Engineer & tester lewat WireGuard. Deploy hanya via pipeline | `https://app.staging.<domain>` | `infra/ansible/inventories/staging` + SOPS |
| **production** | Customer | Penuh | Tag `vX.Y.Z` + job manual Maintainer | Customer via internet. Admin hanya via WireGuard dengan akun bernama + sudo tercatat (auditd). Akses DB manusia hanya `msp_readonly_replica` dengan alasan tercatat. | `https://app.<domain>` (`/api` → api, ADR-25), `https://files.<domain>` (presigned URL ke S3 gateway SeaweedFS) | `infra/ansible/inventories/production` + SOPS |

**Parity:** versi PostgreSQL/PostGIS, Redis, SeaweedFS, Go, dan Node di local/CI di-pin agar sama dengan staging/production. `scripts/ci/check_version_parity` membandingkan tag image CI, directive `toolchain` di `go.mod`, dan variabel Ansible (`postgres_version`, `postgis_version`) saat `infra/**`, `go.mod`, atau `.gitlab-ci.yml` berubah.

### 3.2 Keputusan: preview environment ditunda

MVP **tidak** memakai preview environment per MR. Penggantinya: (1) Storybook statis sebagai artifact MR (`expose_as`) + screenshot wajib; (2) stack lengkap di CI untuk E2E smoke; (3) staging diperbarui ≤ 30 menit setelah merge.

**Alasan:** kapasitas dedicated server tetap (tidak elastis), sehingga stack per MR memakan sumber daya staging. Preview web-only yang memanggil API staging menjadikan staging dependensi bersama yang rapuh. Tim MVP juga kecil. **Ditinjau ulang** bila tim ≥ 6 engineer atau review UI menunggu staging > 1 hari secara rutin. Opsi saat itu: web + API mock (MSW) per MR di `stg-01`.

### 3.3 Aturan data per environment

| Aturan | local | ci | staging | production |
|---|:-:|:-:|:-:|:-:|
| Data sintetis | ✓ | ✓ | ✓ | — |
| Subset dataset berlisensi (tanpa person/kontak pribadi) | — | — | ✓ | ✓ |
| Data/upload customer | — | — | — | ✓ |
| Restore backup production | — | — | — | Hanya drill di `dr-01` (zona prod), lalu dihapus (§7.1) |
| Auth stub / LLM `fake` | ✓ | ✓ | — | — |
| Flag fault-injection `chaos.*` | ✓ | ✓ | ✓ | Ditolak saat startup |
| Email keluar | Mailpit | Mailpit | Mailpit | Provider SMTP |

Binary **menolak start** bila `MSP_ENV ∈ {staging, production}` dan auth stub, `--dev-bootstrap`, atau flag `chaos.*` aktif (diuji unit test).

### 3.4 Local dev (`infra/docker/compose.yml`)

Service: `postgres` (`postgis/postgis:16-3.5`, pin digest), `redis`, `seaweedfs` (`weed server -s3`: master + volume + filer + S3 dalam satu container; init bucket `msp-snapshots`, `msp-uploads`, `msp-exports`, `msp-tiles`), `mailpit`, `kratos` (bila D-20 = Ory Kratos; default lokal auth stub), `smokescreen`, `migrate` (`msp migrate up`, lalu exit; service lain `depends_on: service_completed_successfully`), `api` dan `worker` (live reload via `air` atau `go run`), `web` (`next dev`), `webhook-sink`.

### 3.5 Topologi server & zona jaringan

```
                      Internet
          443/80 │                         │ UDP 51820 (WireGuard admin)
   ┌─────────────▼─────────────┐   ┌──────▼──────────────────────────────┐   probe eksternal
   │ app-01 (→ app-02)  [edge] │   │ ops-01                       [ops]  │◄── (di luar DC)
   │ Traefik, web, api, kratos │   │ GitLab CE + registry, runner deploy │
   │ worker (River) blue/green │   │ Prometheus, Alertmanager, Grafana,  │
   │ redis (cache), smokescreen│   │ Loki, Tempo, GlitchTip, Tang        │
   │ otel-collector            │   └──────┬───────────────┬──────────────┘
   └─────────────┬─────────────┘          │ SSH deploy    │ SSH deploy
      VLAN privat│ (prod)                 │               │
   ┌─────────────┼──────────────┬─────────┘        ┌──────▼───────┐   ┌──────────────┐
┌──▼──────────┐ ┌▼────────────┐ ┌▼─────────────┐   │ stg-01 [stg] │   │ ci-01  [ci]  │
│ db-01       │→│ db-02       │ │ stor-01      │   │ staging all- │   │ runner MR &  │
│ PG16+PostGIS│ │ hot standby │ │ SeaweedFS S3 │   │ in-one       │   │ build, cache │
│ PgBouncer   │ │ read replica│ │ repo lokal   │   └──────────────┘   └──────────────┘
│ pgBackRest  │ │             │ │ pgBackRest   │   ci-01: hanya ke ops-01 (GitLab/registry)
└──────┬──────┘ └─────────────┘ └──────┬───────┘
       └── WAL/backup ──────────────────┴──► off-site S3-compatible (lokasi lain, Indonesia)
```

| Zona | Host | Inbound yang diizinkan (nftables) |
|---|---|---|
| edge/prod | `app-01` (`app-02`) | 80/443 publik → Traefik. SSH hanya dari subnet WireGuard dan IP privat `ops-01` |
| data/prod | `db-01`, `db-02`, `stor-01` | 6432 (PgBouncer, `api`) dan 5432 (koneksi langsung: client River di worker, `migrate`; replica `db-02`) dari app host; 8333 (S3 gateway SeaweedFS) dari app host; scrape exporter dari `ops-01`; SSH dari WireGuard/`ops-01` |
| ops | `ops-01` | UDP 51820 publik (WireGuard). GitLab, registry, Grafana hanya dari WireGuard dan zona privat |
| ci | `ci-01` | Tanpa inbound publik. Outbound: internet (proxy modul/paket) + `ops-01`. **Tidak** punya rute ke zona prod/stg |
| stg | `stg-01` | 443 dari WireGuard. SSH dari WireGuard/`ops-01` |

**Sizing awal (indikatif, direvisi setelah Q-02):** `app-01` 16 core/64 GB/2× NVMe 1 TB RAID1 · `db-01`, `db-02` 16–24 core/128 GB/4× NVMe 2 TB RAID10 · `stor-01` 8 core/32 GB/≥ 4 disk data · `ops-01` 16 core/64 GB/2× NVMe 2 TB · `ci-01` 16 core/64 GB/2× NVMe 1 TB · `stg-01` 16 core/64–128 GB/2× NVMe 2 TB (menampung dataset sintetis 5 juta).

**Risiko yang diterima di MVP:** `app-01` adalah SPOF (04 §2.2). Failover DB manual (±30 menit). `ops-01` menampung GitLab sekaligus monitoring. Bila `ops-01` mati, deploy berhenti, tetapi rollback tetap bisa: image lama ada di cache host, dan playbook dapat dijalankan Maintainer dari laptop via WireGuard (break-glass). MVP 0–M-2: `app-01` tunggal (restore cepat via Ansible); `app-02` aktif sebelum M-3 (R-08, Q-CI-05).

---

## 4. Pipeline merge request

Pipeline berjalan pada event `merge_request_event`. Pipeline branch ganda dicegah `workflow:rules`. `interruptible: true` membatalkan pipeline lama saat ada push baru. MR draft hanya menjalankan lint, unit test, dan security.

### 4.1 Path filter (`rules:changes`, `compare_to: refs/heads/main`)

| Grup | Path | Job |
|---|---|---|
| backend | `apps/api/**`, `data/**` | `backend:lint`, `backend:test`, `backend:fuzz`, `backend:vuln` |
| db | `apps/api/db/**`, `apps/api/sqlc.yaml`, `apps/api/internal/db/**` | `db:migrations` |
| web | `apps/web/**`, `packages/**`, `pnpm-lock.yaml`, `turbo.json` | `web:lint`, `web:test`, `web:build` |
| contracts | `contracts/**`, `apps/api/internal/api/**`, `packages/api-client/**` | `contracts` |
| tokens | `docs/02-design-system/tokens/**`, `packages/design-tokens/**` | `tokens` |
| e2e | `apps/**`, `packages/**`, `contracts/**`, `infra/docker/**` | `build:mr-images`, `e2e:smoke` |
| scoring | konfigurasi versi skor & geo (`apps/api/internal/intelligence/internal/scoring/models/**`, konfigurasi geo) | `model:regression` |
| prompts | prompt & eval set modul `ai` | `ai:eval` |
| er | `apps/api/internal/identity/**` | `er:golden-synthetic` |
| infra | `infra/ansible/**`, `infra/docker/**` | `infra:lint`, `infra:molecule` |
| ci | `.gitlab-ci.yml`, `ci/**`, `scripts/ci/**`, `renovate.json` | `ci:lint` |
| docs | `docs/**`, `**/*.md` | `docs` |

Selalu berjalan: `mr:title`, `mr:gate`, `security:secrets`, `security:sast`, `security:deps`.

### 4.2 Job

| Job | Isi | Tool | Gate |
|---|---|---|---|
| `mr:title` | Validasi Conventional Commits pada `$CI_MERGE_REQUEST_TITLE` | commitlint | QG-01 |
| `mr:gate` | Approval, label, thread (§2.3). Manual, blocking | script + GitLab API | QG-02 |
| `backend:lint` | `gofmt -l` dan `goimports -l` kosong; `go mod tidy -diff`; `go mod verify`; `go vet ./...`; `golangci-lint run` (staticcheck, errcheck, revive, gosec, **depguard**: batas modul ADR-01, mis. `internal/query` tidak boleh mengimpor `internal/intelligence/...` kecuali paket publiknya) | Go toolchain, golangci-lint | QG-03, QG-04, QG-19 (gosec) |
| `web:lint` | ESLint, `prettier --check`, `tsc --noEmit` (`turbo run … --affected`) | pnpm, Turborepo | QG-03, QG-04 |
| `backend:test` | `services:` PostGIS + Redis + SeaweedFS → `msp migrate up` → `go test -race` unit + integration (`-tags=integration`, `-coverpkg=./...`) via gotestsum (JUnit) → **suite isolasi tenant** (`-run TenantIsolation`, koneksi role `msp_app` `NOBYPASSRLS`) → test redaksi PII (log & payload prompt) → gate coverage per package → Cobertura ke MR | go test, gotestsum, gocover-cobertura | QG-05, QG-06, QG-07, QG-09, QG-10 |
| `backend:fuzz` | Fuzz smoke native Go: setiap target `Fuzz*` di package normalizer (telepon, email, domain, nama, alamat) 20 detik; corpus `testdata/fuzz` + cache fuzz di `GOCACHE`. Lebih lama di nightly | `go test -fuzz` | QG-05 |
| `backend:vuln` | `go tool govulncheck ./...`: 0 kerentanan **yang terjangkau** (reachable) | govulncheck | QG-19 |
| `web:test` | Vitest + Testing Library dengan coverage threshold | Vitest | QG-05, QG-08 |
| `db:migrations` | (1) versi goose unik dan lebih besar dari versi tertinggi di `main` (tanpa *out-of-order*); (2) `msp migrate up` pada DB kosong (River + aplikasi); (3) `pg_dump --schema-only` = `apps/api/db/schema.sql` ter-commit (skema dapat direview, tidak ada drift); (4) `sqlc diff` (kode `internal/db/` up to date) + `sqlc vet`; (5) **squawk** pada file migrasi baru; (6) cek RLS: setiap tabel tenant (`workspace_id`) punya `ENABLE` + `FORCE ROW LEVEL SECURITY` + policy `USING`/`WITH CHECK` (allowlist ber-owner security); (7) fase contract hanya dengan label `db-contract` | goose (via `msp`), sqlc, squawk, psql | QG-11 |
| `contracts` | (1) `go tool oapi-codegen` → `git diff --exit-code` stub server; (2) Spectral; (3) **oasdiff** breaking vs `main` → gagal kecuali label `api-breaking` + rencana versi; (4) openapi-typescript → `git diff --exit-code packages/api-client`; (5) ajv: metaschema `contracts/events/**` & `contracts/input/**` + validasi contoh payload; (6) skema event versi terbit immutable; (7) Schemathesis smoke terhadap API ephemeral (tanpa 5xx, response sesuai skema) | oapi-codegen, Spectral, oasdiff, openapi-typescript, ajv-cli, Schemathesis | QG-12, QG-13, QG-14 |
| `tokens` | `node docs/02-design-system/tokens/build.mjs --check` (gagal bila `tokens.css` basi atau ada pasangan kontras < 4,5:1 teks / 3:1 UI, light & dark). Setelah `packages/design-tokens` ada: build Style Dictionary + `git diff --exit-code` + versi paket naik bila token berubah | Node, Style Dictionary | QG-15 |
| `web:build` | `next build` (standalone) + Storybook → axe pada setiap story (0 serious/critical) → Storybook artifact (`expose_as`, 7 hari) | Next.js, Storybook, axe-core | QG-16 |
| `build:mr-images` | Build `msp-backend` (dengan build tag `dev`: subcommand seed + `--dev-bootstrap`) & `msp-web` (BuildKit rootless) → `…/mr/<image>:<sha>` (cleanup 7 hari) | BuildKit | — |
| `e2e:smoke` | Image MR sebagai `services:` (api dengan `--dev-bootstrap`: migrasi + seed e2e, worker, web, PostGIS, Redis, SeaweedFS, webhook-sink) → Playwright `@smoke` (chromium, `parallel: 2`) + axe. *Critical path*: login (stub) → Offering → ICP → hasil pertama; search + filter + zero-result; profil + evidence drawer; unlock (debit credit); export kecil; webhook test; kill switch AI menampilkan pesan nonaktif. Detail di 05. | Playwright, @axe-core/playwright | QG-16, QG-17 |
| `er:golden-synthetic` | Golden set **sintetis** di repo → precision/recall dilaporkan di MR | go test | QG-26 |
| `model:regression` | Versi naik + versi terbit tidak berubah (`versions.lock`); recompute fixture "golden scores" → diff sebagai komentar MR | go test, script | QG-23, QG-24 |
| `ai:eval` | Respons LLM **rekaman** (provider `fake`, deterministik) untuk validator grounding & redaksi. Eval **live** (manual, label `ai-eval`/prompt berubah) memakai key ber-scope environment `ai-eval` (§10.4) | harness eval Go | QG-25 |
| `security:secrets` | gitleaks pada commit MR (`$CI_MERGE_REQUEST_DIFF_BASE_SHA..HEAD`) | gitleaks | QG-18 |
| `security:sast` | Semgrep CE (ruleset tersimpan di repo: snapshot `p/golang`, `p/typescript`, `p/owasp-top-ten` + aturan kustom; `--metrics=off`) | Semgrep | QG-19 |
| `security:deps` | OSV-Scanner (`go.mod`, `pnpm-lock.yaml`) + license allowlist, `pnpm audit --audit-level high`, Trivy fs (`vuln,secret,misconfig`) | OSV-Scanner, pnpm, Trivy | QG-19, QG-20 |
| `infra:lint`, `infra:molecule` | ansible-lint, yamllint, `--syntax-check`, Trivy config (Dockerfile, compose); Molecule untuk role kritis (`postgres`, `msp_app`, `traefik`, `firewall`) | ansible-lint, Molecule | QG-21 |
| `ci:lint` | CI Lint API, image job di-pin digest, tanpa `curl \| sh`, validasi `renovate.json` | script, ajv-cli | QG-21 |
| `docs` | markdownlint-cli2, lychee `--offline`, cek ID FR-/NFR-/QG- yang dirujuk ada di dokumen pendefinisi (advisory) | markdownlint-cli2, lychee | QG-22 |

### 4.3 Anggaran waktu (p50, paralel via `needs`)

| Job | Target | Teknik |
|---|---|---|
| `mr:title`, `security:secrets` | < 0,5 menit | — |
| `backend:lint`, `backend:vuln` | 1,5–2 menit | Cache `GOCACHE`/`GOMODCACHE` (key `go.sum`), cache golangci-lint |
| `web:lint`, `web:test` | 2–3 menit | pnpm store + Turborepo cache, `--affected` |
| `backend:test` | 4–6 menit | `-race` paralel per package, DB template per test package |
| `backend:fuzz`, `db:migrations`, `contracts`, `tokens` | 1–3 menit | Hanya bila path berubah |
| `web:build` | 5–6 menit | Cache `.next/cache` |
| `build:mr-images` → `e2e:smoke` | 2 + 6 menit (**jalur kritis**) | BuildKit registry cache, Go build cache mount, 2 shard, ≤ 10 skenario |

Semua cache disimpan di cache S3 runner (SeaweedFS lokal di `ci-01`). Bila p50 > 10 menit dua minggu berturut-turut: tambah concurrency runner, tambah shard, atau pindahkan test ke nightly (§16.3).

### 4.4 Syarat merge

*Pipelines must succeed* mencakup **semua** job yang terpicu, termasuk `mr:gate`. Tidak ada daftar required check terpisah. Job hanya boleh bersifat advisory (`allow_failure: true`) bila tercantum "Advisory" di §8.

---

## 5. Pipeline main

Trigger: push ke `main` (hasil fast-forward merge) dan ke `release/v*` (hotfix, §6.6). Seluruh job MR yang relevan berjalan ulang, ditambah job di bawah.

### 5.1 Build image (sekali per commit)

| Image | Isi | Container yang memakai |
|---|---|---|
| `msp-backend` | Binary statis `/msp` (`CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X …buildinfo.Commit=$CI_COMMIT_SHA"`) di **`gcr.io/distroless/static-debian12:nonroot`** (pin digest). Migrasi goose dan konfigurasi versi skor di-embed ke binary. | `msp api`, `msp worker --queues …`, `msp migrate up` (one-off), `msp tool …` (job rilis data/model) |
| `msp-web` | Next.js `output: standalone` di `node:24-slim` (pin digest, non-root) | web |

```dockerfile
# apps/api/Dockerfile (ilustratif)
FROM golang:<versi-toolchain>@sha256:<digest> AS build
WORKDIR /src/apps/api
COPY apps/api/go.mod apps/api/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY apps/api/ ./
ARG GIT_SHA
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X <module-path>/internal/platform/buildinfo.Commit=${GIT_SHA}" -o /out/msp ./cmd/msp
FROM gcr.io/distroless/static-debian12:nonroot@sha256:<digest>
COPY --from=build /out/msp /msp
ENTRYPOINT ["/msp"]
```

Langkah per image (runner `ci-build`):
1. BuildKit rootless → **push** `…/msp-backend:sha-<commit>` dengan provenance `mode=max`. Digest dicatat sebagai variabel `dotenv` (`BACKEND_IMAGE=…@sha256:…`). **Versi rilis tidak di-bake**, tetapi disuntikkan saat deploy (`MSP_VERSION`), karena tag dibuat setelah build (§6.1). Commit SHA ada di binary.
2. **Trivy image**: 0 Critical/High yang fixable (QG-27). Trivy membaca info modul Go di binary. Pengecualian hanya via VEX/`.trivyignore` bertanggal kedaluwarsa.
3. **SBOM** syft (SPDX JSON, termasuk modul Go dari binary) → artifact (90 hari) + bucket evidence `msp-evidence` di `stor-01` (retensi ≥ 1 tahun).
4. **cosign sign** + `cosign attest --type spdxjson` dengan private key dari SOPS (`--tlog-upload=false`). **Penandatanganan hanya terjadi setelah langkah 2 lolos**, sehingga tanda tangan berfungsi sebagai segel "lolos gate".

**Web & build-once:** nilai yang bergantung environment (URL, DSN GlitchTip, nama env) tidak boleh memakai `NEXT_PUBLIC_*` karena di-inline saat build. Nilai tersebut dibaca server saat runtime dan dikirim lewat runtime config. Lint rule kustom memblokir `NEXT_PUBLIC_*` di luar allowlist.

### 5.2 Deploy ke staging

Job `deploy:staging` (runner `deploy`, `resource_group: staging`, `environment: staging`) menjalankan `infra/ansible/playbooks/deploy.yml` dengan mekanisme yang **sama** dengan production (§6.3), tetapi tanpa canary bertahap:

1. Verifikasi tanda tangan cosign + attestation SBOM dengan `infra/keys/cosign.pub` (QG-28).
2. Materialisasi secret SOPS → **tmpfs** `/run/msp/<warna>/` (ADR-22).
3. Pull image by digest → **`msp migrate up`** sebagai container one-off (role `msp_owner`, advisory lock goose, expand-only). Gagal → deploy berhenti, warna lama tetap melayani.
4. Start warna baru (worker, api, web) → tunggu healthcheck (`msp tool healthcheck` → `/readyz`: DB, Redis, versi skema ≥ minimum yang dibutuhkan binary).
5. Hentikan worker warna lama (SIGTERM → River soft stop) → alihkan 100% traffic Traefik → smoke → hentikan api/web warna lama.
6. Catat deployment (GitLab environment), anotasi Grafana, release GlitchTip `msp@sha-<short>`.

### 5.3 Verifikasi staging (post-deploy)

| Suite | Isi | Gate |
|---|---|---|
| Smoke | Health, login user sintetis, search, profil, export kecil, webhook test, egress/nftables (05) | QG-29 |
| E2E penuh | Playwright seluruh suite (chromium, 4 shard) + axe | QG-29, QG-16 |
| Kontrak | Schemathesis terhadap API staging (`--checks all`, workspace sintetis khusus) | QG-29 |
| Isolasi tenant (deployed) | Dua workspace sintetis, akses silang via HTTP → 404/403; tanpa kebocoran di list/search/export/webhook | QG-29, QG-09 |
| Traceability | Penanda TC di laporan JUnit vs katalog 05 §6 | QG-37 |
| DAST | ZAP baseline nightly dan pada kandidat rilis (§5.4, §7), tidak per commit | QG-30 |

Semua lulus → job `mark:staging-verified` menulis **commit status `staging-verified`** (GitLab commit status API). Status ini prasyarat `release:cut` dan promosi. E2E memakai workspace sintetis per run (`e2e-<pipeline_id>`) yang dibersihkan job harian.

### 5.4 Kandidat rilis (pre-release, 05 §3.3)

Untuk rilis **minor/major**, Maintainer menjalankan job manual **`rc:start`** pada pipeline `main` commit kandidat (setelah `staging-verified`). Job ini memicu child pipeline `rc` di staging untuk SHA tersebut:

| Suite RC | Bukti |
|---|---|
| k6 penuh (dataset sintetis 5 juta, 50 VU) | QG-31 |
| ZAP full + review manual temuan | QG-30 |
| Re-run pipeline data 5 juta establishment (NFR-PERF-07) | Laporan durasi |
| Drill rollback blue/green (deploy N → rollback ke N-1 → verifikasi) | Laporan drill |
| AI eval live penuh | QG-25 |
| `soak-webhook` 24 jam (wajib untuk `v1.0.0` dan major) | RG-3 |

Semua lulus → commit status **`rc-verified`**. Item manual 05 (a11y manual, usability, UAT) dicatat di issue Release Readiness.

### 5.5 Kebijakan `main` merah

Build atau verifikasi gagal → notifikasi kanal tim + issue `main-red` yang di-assign ke author commit. Aturan **stop-the-line**: revert atau fix-forward ≤ 1 jam. Bila tidak ada pemilik, Maintainer me-revert. Commit yang gagal tidak pernah mendapat `staging-verified`.

---

## 6. Pipeline release

### 6.1 Versioning (semantic-release)

- Job **`release:cut`** (manual, pipeline `main`, `needs: mark:staging-verified`) menjalankan semantic-release: analisis commit sejak tag terakhir → versi (patch/minor/major) → tag `vX.Y.Z` + **GitLab Release** berisi catatan rilis. Plugin: `commit-analyzer` (preset conventionalcommits), `release-notes-generator`, `@semantic-release/gitlab`. **Tanpa** `@semantic-release/git`: tidak ada commit balik, sehingga tag menunjuk commit yang sudah dibangun dan `staging-verified`.
- Token: project access token bot (role Maintainer, scope `api` + `write_repository`), masked + protected, scope environment `release`, kedaluwarsa ≤ 365 hari.
- Changelog tinggal di GitLab Releases. Ringkasan untuk customer (id-ID) ditulis product dari Release tersebut.
- `design-tokens-vX.Y.Z`: versi di `packages/design-tokens/package.json` dinaikkan di MR (QG-15). `release:cut` membuat tag bila versi itu belum ber-tag.

### 6.2 Pipeline tag `vX.Y.Z`: readiness

| Cek (`release:resolve`, `release:readiness`) | Berlaku | Gate |
|---|---|---|
| Commit tag berstatus `staging-verified` (menunggu ≤ 45 menit) | Semua | QG-29 |
| Digest `msp-backend` & `msp-web` untuk commit tag (`sha-<commit>`) ada, tertandatangani, punya SBOM | Semua | QG-28 |
| Tidak ada temuan Critical/High terbuka: rescan Trivy kedua digest + govulncheck/SAST/deps terakhir `main` | Semua | QG-19, QG-27, NFR-SEC-06 |
| Commit tag berstatus `rc-verified` (k6, ZAP full, drill rollback, AI eval live) | Minor & major | QG-30, QG-31, QG-25 |
| ER golden set penuh (nightly terakhir) lulus | Minor & major | QG-26 |
| Drill restore terakhir lulus dan berumur ≤ 35 hari | Minor & major | QG-35 |
| Issue **Release Readiness** versi ini tertutup dengan bukti RG-1 s.d. RG-7 | Major dan `v1.0.0` | QG-32 |
| Bukan masa freeze (§6.5) dan error budget bulan berjalan ≤ 75% | Semua kecuali hotfix | NFR-AVAIL-01 |

Readiness lolos → job **`deploy:production`** (manual, `allow_failure: false`, `resource_group: production`, runner `deploy`) siap dijalankan oleh **Maintainer**. Saat ada ≥ 2 Maintainer, script *four-eyes* menolak bila pemicu deploy sama dengan pemicu `release:cut`.

### 6.3 Deploy production: blue/green + canary di Traefik

Dua project Compose per host: `msp-blue` dan `msp-green`. Masing-masing berisi web, api, dan worker pool. Service api/web setiap warna diberi label Docker (`api-blue@docker`, `api-green@docker`, dst.). Router dan **weighted service** didefinisikan di file provider Traefik yang dirender Ansible (§17.3).

```
 langkah                        traffic api/web           worker (River)
 0. awal                        blue 100%                 blue
 1. msp migrate up (expand)     blue 100%                 blue
 2. start green + healthcheck   blue 100%                 blue + green (keduanya mengambil job)
 3. SIGTERM worker blue → River soft stop (tidak ambil job baru, selesaikan yang berjalan) → green
 4. canary                      blue 90% / green 10%  (10 menit, dipantau Prometheus)
 5. promote                     green 100%            (bake 20 menit; blue api/web tetap hidup)
 6. selesai                     green 100%; file Traefik dirender ulang tanpa blue; blue dihentikan
 gagal di langkah 2–5 → rescue: traffic 100% ke blue, start ulang worker blue, hentikan green
```

- **Worker lebih dulu.** Worker baru menerima job args versi N-1 dan N. Field args hanya ditambah sebagai opsional (JSON Go mengabaikan field asing). *Job kind* baru hanya di-insert setelah worker baru aktif, atau di balik flag. Tanpa aturan ini, worker lama akan gagal memproses kind yang tidak dikenalnya.
- **Singleton aman.** Periodic job dan tugas tunggal memakai leader election River, sehingga aman berjalan di dua warna. Outbox = River `InsertTx` + job `event.dispatch` (ADR-07), sehingga tidak ada dispatcher terpisah.
- **Drain.** `stop_grace_period` per pool (mis. 15 menit untuk `pipeline`). `msp worker` memanggil soft stop River lalu `StopAndCancel` sebelum batas waktu. Job yang terputus di-retry, sehingga job wajib idempoten/resumable (04 §6.2).
- **Referensi warna.** File Traefik tidak boleh merujuk service warna yang sudah dihentikan, jadi file dirender ulang sebelum warna lama dimatikan.

**Rollback otomatis (QG-33).** Playbook memanggil `scripts/deploy/watch_canary` yang mengevaluasi query Prometheus tiap 30 detik selama canary dan bake. Pelanggaran → blok `rescue` Ansible.

| Sinyal | Threshold | Dasar |
|---|---|---|
| Rasio 5xx service warna baru (metrik Traefik) | > 2% selama 3 menit (normal < 0,5%) | NFR-AVAIL-01 |
| p95 `GET /v1/prospects` (histogram OTel, label `msp.color`) | > 800 ms selama 5 menit | NFR-PERF-01 |
| p95 `GET /v1/prospects/{id}` | > 500 ms selama 5 menit | NFR-PERF-02 |
| Server Traefik tidak sehat / container restart loop | > 0 selama 3 menit | Healthcheck |
| Worker: rasio job gagal (metrik River) | > 5% selama 10 menit | RG-5 |
| Job *discarded* (setara DLQ) | > 20 / 5 menit (awal, dikalibrasi) | RG-5 |

**Traffic beta rendah.** Selama canary dan bake, job `synthetic-load` menjalankan loop smoke read-only dari workspace sintetis `msp-synthetic-monitor` (tanpa credit, dikecualikan dari metrik produk dan billing) dengan header yang dipaksa ke warna baru, agar alarm punya sinyal.

**Multi-host (`app-02`):** playbook `serial: 1` per host di belakang VIP (keepalived/VRRP atau LB provider). Detail menunggu Q-CI-05.

### 6.4 Verifikasi pasca-deploy

Otomatis: smoke production (read-only, workspace sintetis) → commit status `production-deployed` → tag registry `vX.Y.Z` ditambahkan ke digest yang sama (kemudahan, bukan untuk deploy) → release GlitchTip `msp@vX.Y.Z` → anotasi Grafana → notifikasi. Checklist manual ada di §14.6.

### 6.5 Jendela deploy & freeze

Production: Senin–Kamis 10:00–15:00 WIB. Jumat ≥ 15:00, akhir pekan, dan libur nasional (termasuk periode Lebaran) hanya untuk hotfix. Rilis non-fix dibekukan bila error budget bulanan (99,5% ≈ 3,6 jam) terpakai > 75%.

### 6.6 Hotfix

1. **Default: fix-forward dari `main`** (fitur belum siap ada di balik flag): MR `fix/…` → merge → staging → `release:cut` (patch) → deploy.
2. Bila `main` memuat perubahan yang tidak boleh dirilis: Maintainer membuat `release/vX.Y` dari tag terakhir → cherry-pick → pipeline branch membangun dan memverifikasi di staging (staging sementara menjalankan versi hotfix) → `release:cut` pada branch tersebut (maintenance branch semantic-release) → `vX.Y.(Z+1)`. Fix yang sama wajib masuk `main` lewat MR terpisah.

---

## 7. Scheduled pipelines

Setiap schedule GitLab membawa variabel `SCHEDULE` yang menentukan job (`rules: if: $SCHEDULE == "nightly"`). Schedule berjalan di `main` (terlindungi), sehingga job deploy-side bisa memakai runner `deploy`.

| `SCHEDULE` · job | Jadwal (WIB) | Isi | Eksekusi | Gate / output |
|---|---|---|---|---|
| `nightly` · regresi penuh | Harian 02:00 | `go test -race` penuh (termasuk `-tags=slow`), property test profil nightly, Vitest penuh, Playwright chromium + firefox + webkit terhadap staging, viewport 360 px (NFR-COMP-01), Schemathesis 500 contoh/operasi | `ci-01` + staging | Issue `nightly-red` ke owner. Memblokir rilis bila belum ditutup |
| `nightly` · fuzz panjang | Harian 02:00 | Setiap target `Fuzz*` 10 menit. Crash → corpus baru di-commit lewat MR otomatis + issue | `ci-01` | Issue |
| `nightly` · k6 ringan | Harian 03:00 | Skenario NFR-PERF-01/02, lewat Traefik staging | Container k6 di `stg-01` (runner `deploy`) | Advisory (QG-31 penuh ada di RC) |
| `nightly` · ZAP baseline | Harian 03:30 | `zap-baseline` web + API staging. Mingguan: ZAP API scan berbasis OpenAPI | `ci-01` → staging | QG-30 |
| `nightly` · data quality | Harian 04:00 | ER golden set penuh, regresi skor vs baseline versi aktif, reproducibility (recompute sampel → hash identik), suite edge case EC-xx | `msp tool` one-off di `stg-01`. CI hanya menerima ringkasan agregat | QG-26, NFR-DATA-01 |
| `nightly` · rekonsiliasi | Harian 04:30 | Saldo credit = Σ ledger, usage event vs ledger (staging). Di production, job yang sama dijalankan River periodic job dan dipantau alert | `stg-01` | RG-4 |
| `nightly` · rescan image | Harian 05:00 | Trivy pada digest yang **sedang berjalan** di staging & production | `ci-01` | Issue `security` |
| `nightly` · Lighthouse | Harian 05:30 | Halaman kunci staging, budget LCP/INP | `ci-01` | Advisory (NFR-PERF-03) |
| `drift` | Harian 06:00 | `ansible-playbook site.yml --check --diff` (staging & production) + setelan GitLab | Runner `deploy` | Perubahan > 0 → issue `infra-drift` |
| `backup-check` | Harian 07:00 | `pgbackrest info`/`check` kedua repo, umur backup terakhir, lag `weed filer.sync` off-site, umur backup GitLab | Runner `deploy` | Issue + alert (juga dipantau Prometheus) |
| `renovate` | Tiap jam kerja, Sen–Jum | Renovate self-hosted (§15.4) | `ci-01` | MR dependency |
| `weekly` | Senin 02:00 | Semgrep penuh dengan ruleset terbaru, lychee eksternal, laporan flag kedaluwarsa (§13.3), AI eval live sandbox (budget dibatasi) | `ci-01` | Issue |
| `monthly` · restore drill | Selasa pertama, 09:00 | §7.1 | `dr-01` via runner `deploy` | QG-35 |
| `soak-webhook` (manual/RC) | Kandidat MVP 1 & major | 24 jam pengiriman webhook (termasuk endpoint flaky buatan) | `stg-01` | RG-3 (≥ 99%) |
| `alert-drill` (manual) | Sebelum MVP 1, lalu tiap kuartal | Injeksi kegagalan via `chaos.*` di staging: pipeline failure > 5%, backlog > 1 jam, AI failure > 5%, webhook failure > 5% → alert terpicu dan diterima on-call | `stg-01` | RG-5 |

### 7.1 Drill restore (bulanan)

1. Siapkan `dr-01` (server cadangan atau sewa per jam di zona prod, disk LUKS). Catat `T_target` = sekarang − 5 menit.
2. `pgbackrest restore --type=time --target=<T_target>` dari **repo off-site**, sekaligus menguji jalur DR lokasi lain.
3. **RTO** = mulai restore → PostgreSQL menerima koneksi dan query verifikasi lulus. Target ≤ 4 jam.
4. **RPO** = selisih `T_target` dengan transaksi terakhir yang ada (`max(created_at)` di `platform.audit_log` dan `usage.usage_event`). Target ≤ 15 menit.
5. Verifikasi integritas: jumlah baris tabel kunci vs production (toleransi jeda), versi goose = versi production, RLS policy ada, sampel objek di replika off-site terbaca. Restore backup GitLab diuji tiap kuartal.
6. Hapus data dengan aman (hapus volume LUKS). Laporan berisi metrik saja disimpan di `msp-evidence`. Gagal → incident SEV3, dan QG-35 memblokir rilis minor/major berikutnya.

Restore tidak dilakukan ke `stg-01` karena staging tidak boleh berisi data production (§3.3).

---

## 8. Quality gates

### 8.1 Daftar QG

Tahap: **MR** · **main** · **rc** = kandidat rilis (§5.4) · **rel** = pipeline tag `v*` · **sch** = scheduled · **data** = pipeline tag rilis data/model · **infra** = pipeline tag `infra-*`. *Blocking* = gagal → merge/deploy/aktivasi berhenti.

| ID | Gate | Tahap | Threshold | Blocking | Sumber |
|---|---|---|---|---|---|
| QG-01 | Judul MR Conventional Commits | MR | Lolos commitlint | Ya | §2.2 |
| QG-02 | MR gate | MR | ≥ 1 approval owner (2 untuk path berisiko saat tim ≥ 4), approver ≠ author, approval setelah commit terakhir, thread resolved, label wajib | Ya | §2.3 |
| QG-03 | Lint & format | MR | gofmt/goimports bersih, `go mod tidy -diff` kosong, `go vet` + golangci-lint (termasuk depguard batas modul) 0 temuan; ESLint, Prettier 0 error | Ya | DoD, NFR-MAINT-01, ADR-01 |
| QG-04 | Kompilasi & type check | MR | `go build ./...` (semua build tag), `tsc --noEmit`: 0 error | Ya | NFR-MAINT-01 |
| QG-05 | Unit, integration & fuzz smoke | MR | `go test -race` dan Vitest 100% lulus (karantina §16.3: hanya ber-issue + owner, maks 7 hari, ≤ 1% suite); fuzz smoke tanpa crash | Ya | DoD §12.2 |
| QG-06 | Coverage package kritis | MR | ≥ 85% statement (setara line, NFR-MAINT-01) untuk package normalisasi, ER, scoring | Ya | NFR-MAINT-01 |
| QG-07 | Coverage backend | MR | ≥ 75% statement untuk seluruh `apps/api` (tanpa kode generate `internal/db/`, `internal/api/`) | Ya | NFR-MAINT-01 |
| QG-08 | Coverage frontend | MR | ≥ 70% line `packages/ui` + logika `apps/web` (hooks, lib, formatter). Halaman dicakup E2E | Advisory 4 minggu, lalu Ya | Proposed |
| QG-09 | Isolasi tenant | MR, main, sch, rel | 100% lulus, role `msp_app` `NOBYPASSRLS` | Ya, **tidak bisa di-waive** | NFR-SEC-03, FR-AUD-006, RG-7 |
| QG-10 | Test privasi | MR | Redaksi PII di log dan payload prompt LLM: 100% lulus | Ya | NFR-PRIV-01, NFR-PRIV-04, FR-AI-005 |
| QG-11 | Keamanan migrasi | MR | Versi goose unik & berurutan; up dari kosong; `schema.sql` sesuai; `sqlc diff`/`vet` bersih; squawk 0 error; RLS lengkap; contract hanya dengan `db-contract` | Ya | §9, NFR-SEC-03 |
| QG-12 | Lint OpenAPI | MR | Spectral: 0 error, termasuk rule kustom subset OpenAPI 3.1 yang lolos spike toolchain Sprint 0 (04 TQ-18, §18 langkah 10) | Ya | DoD §12.2 |
| QG-13 | Breaking change OpenAPI | MR | oasdiff ERR = 0, **kecuali** label `api-breaking` + rencana versi | Ya | DoD §12.2 |
| QG-14 | Sinkronisasi kontrak | MR | Stub oapi-codegen & client TS up to date; contoh payload valid; skema event terbit immutable; Schemathesis smoke tanpa 5xx | Ya | ADR-08, DoD |
| QG-15 | Design tokens | MR | `build.mjs --check` lolos (CSS up to date, kontras ≥ WCAG AA light & dark); versi paket naik bila token berubah | Ya | NFR-A11Y-01 |
| QG-16 | Aksesibilitas | MR, main | axe: 0 pelanggaran serious/critical (story & halaman E2E) | Ya | DoD §12.2, NFR-A11Y-01 |
| QG-17 | E2E smoke | MR, main | 100% `@smoke` lulus (retry 1×; lulus setelah retry tercatat flaky) | Ya | DoD, RG-3 |
| QG-18 | Tidak ada secret | MR (+ pre-receive hook) | gitleaks: 0 temuan | Ya, **tidak bisa di-waive** | NFR-SEC-04 |
| QG-19 | Tidak ada kerentanan Critical/High | MR, rel | Semgrep ERROR = 0; gosec high = 0; govulncheck reachable = 0; OSV/pnpm audit/Trivy fs fixable Critical/High = 0 | Ya | NFR-SEC-06 |
| QG-20 | License policy | MR | Allowlist: MIT, Apache-2.0, BSD-2/3-Clause, ISC, MPL-2.0, 0BSD, Unlicense, CC0-1.0. AGPL/SSPL/GPL ditolak, LGPL perlu review | Ya | Kebijakan legal (Proposed) |
| QG-21 | Lint CI & IaC | MR | CI Lint valid, image di-pin digest; ansible-lint/yamllint bersih; Molecule lulus; Trivy config 0 High | Ya | §11, §15 |
| QG-22 | Dokumentasi | MR | markdownlint 0 error, link internal valid | Ya (path `docs/**`) | — |
| QG-23 | Versi model/geo/prompt immutable | MR | Konfigurasi versi terbit tidak berubah (`versions.lock`, sha256 di `intel.score_model`); perubahan wajib menaikkan versi | Ya | NFR-DATA-01, FR-INTL-009 |
| QG-24 | Regresi skor/geo ditinjau | MR (fixture), data (sampel staging) | Laporan diff terlampir; disetujui data-intel + product sebelum aktivasi; recompute 2× → hash identik | Ya (aktivasi) | FR-INTL-009, NFR-DATA-01 |
| QG-25 | AI eval | MR (rekaman), rc & data (live) | Grounding ≥ 98%; kebocoran PII = 0; format valid 100%; error/refusal ≤ 2% | Ya | FR-AI-001, -004, -005 |
| QG-26 | ER golden set | MR (sintetis), sch (penuh), rel | Precision ≥ 98%, recall ≥ 90% | Ya | FR-DATA-003, RG-6 |
| QG-27 | Scan image | main, rel | Trivy: 0 Critical/High fixable | Ya (image tidak di-sign) | NFR-SEC-06 |
| QG-28 | Integritas artefak | deploy (staging, prod) | `cosign verify` + `verify-attestation` dengan `cosign.pub` | Ya | §15 |
| QG-29 | Verifikasi staging | main | Smoke, E2E penuh, Schemathesis, isolasi tenant (deployed) 100% → `staging-verified` | Ya (untuk rilis) | DoD, RG-3 |
| QG-30 | DAST | sch (baseline), rc (full) | ZAP: 0 High; Medium ditriase ≤ 14 hari | Ya (rilis minor/major) | NFR-SEC-06, RG-7 |
| QG-31 | Performa kandidat rilis | rc | Search p95 ≤ 800 ms & p99 ≤ 2 s; profil p95 ≤ 500 ms; 50 VU; error < 1%; export 10.000 baris ≤ 2 menit | Ya (minor/major) | NFR-PERF-01, -02, -04 |
| QG-32 | Bukti release gate | rel (major, `v1.0.0`) | Issue Release Readiness memuat bukti RG-1 s.d. RG-7, ditutup product + tech lead | Ya | PRD §12.1 |
| QG-33 | Kesehatan canary/bake | rel | Tidak ada pelanggaran sinyal §6.3 | Ya (rollback otomatis) | NFR-AVAIL-01, NFR-PERF-01/02 |
| QG-34 | Validasi snapshot dataset | data | Threshold §10.2 | Ya (publish) | FR-DATA-001, NFR-DATA-01 |
| QG-35 | Drill restore | sch (bulanan) | RPO ≤ 15 menit, RTO ≤ 4 jam, drill terakhir ≤ 35 hari | Ya (rilis minor/major berikutnya) | NFR-AVAIL-02 |
| QG-36 | Perubahan infrastruktur | infra | `--check --diff` staging & production terlampir; apply production hanya dari tag `infra-*` | Ya | §11 |
| QG-37 | Traceability TC | main | Tidak ada TC P0 berotomasi tanpa test; tidak ada test yang merujuk TC tak dikenal (05 §3.2) | Ya | DoD §12.2 |

**Catatan performa staging:** `stg-01` adalah all-in-one dan lebih kecil dari production, sehingga QG-31 yang lulus di staging dianggap konservatif. Sebelum paid GA, k6 dijalankan sekali di hardware production (pre-launch, data sintetis) untuk memvalidasi angka absolut NFR-PERF.

### 8.2 Release gate MVP 1 → bukti dari pipeline

| RG | Bukti | Diproduksi oleh |
|---|---|---|
| RG-1 Relevansi | Laporan usability + median time-to-first-result | Product/Design → issue Release Readiness |
| RG-2 Kepercayaan | Laporan usability + evidence coverage ≥ 95% (dashboard Data Health) | Product + job data quality |
| RG-3 Aksi | E2E export & webhook (QG-29) + laporan `soak-webhook` ≥ 99% | Pipeline `main`, RC |
| RG-4 Usage | Suite billing/idempotensi (QG-05) + rekonsiliasi nightly 100% | MR, `nightly` |
| RG-5 Operasional | Laporan `alert-drill`: empat alert terpicu dan diterima | `alert-drill` |
| RG-6 Kualitas data | QG-26 penuh + ≥ 10 edge case lulus (05) | `nightly` |
| RG-7 Keamanan | QG-09 + 0 Critical/High (QG-19/27/30) + suite SSRF webhook (ADR-20) | MR, `main`, RC |

### 8.3 Kebijakan waiver

- Gate blocking hanya dapat di-waive tech lead (ditambah owner security untuk gate keamanan) lewat issue `qg-waiver` yang memuat alasan, mitigasi, dan **kedaluwarsa ≤ 14 hari**.
- **QG-09 dan QG-18 tidak pernah di-waive.** Temuan secret ditangani dengan rotasi.
- Pengecualian kerentanan memakai VEX/`.trivyignore`/`osv-scanner.toml`/komentar `//nolint:gosec // alasan + issue` bertanggal kedaluwarsa. Pengecualian kedaluwarsa otomatis membuat gate gagal lagi.

---

## 9. Strategi migrasi database

### 9.1 Prinsip

1. **Setiap migrasi kompatibel dengan versi aplikasi yang sedang berjalan (N-1).** `msp migrate up` berjalan sebelum warna baru di-start, dan rollback mengembalikan traffic ke warna lama di atas skema baru.
2. **Expand → migrate → contract** untuk perubahan yang tidak kompatibel, tersebar di beberapa rilis.
3. **Rollback skema = forward fix.** Blok `-- +goose Down` boleh ditulis untuk kenyamanan lokal, tetapi `msp migrate down` ditolak di staging/production. PITR adalah upaya terakhir (§14.3).
4. Migrasi pendek (target < 5 menit di production). Backfill dan rebuild berjalan sebagai **job River**.
5. Skema adalah SQL eksplisit: file goose = sumber kebenaran, `db/schema.sql` = hasil dump yang dapat direview, sqlc membaca skema dari migrasi.

### 9.2 Contoh: rename `identity.contact_point.verified_at` → `last_verified_at`

| Rilis | Fase | Migrasi (goose) | Kode Go + query sqlc |
|---|---|---|---|
| N | **Expand** | `ALTER TABLE identity.contact_point ADD COLUMN last_verified_at timestamptz` | Query sqlc **dual-write** ke kedua kolom, baca `verified_at`. Insert job River `BackfillLastVerifiedAt` (batch 5.000, idempotent, resumable, di-throttle, progres di ops console) |
| N+1 | **Migrate** | (opsional) `-- +goose NO TRANSACTION` + `CREATE INDEX CONCURRENTLY`. Bila perlu NOT NULL: `CHECK (… IS NOT NULL) NOT VALID` → `VALIDATE CONSTRAINT` | Baca `last_verified_at`, tetap dual-write (rollback ke N aman). Promosi setelah `count(*) WHERE last_verified_at IS NULL AND verified_at IS NOT NULL = 0` |
| N+2 | **Contract** (label `db-contract`) | `ALTER TABLE … DROP COLUMN verified_at`, setelah N+1 stabil di production dan N bukan lagi target rollback | Query sqlc tidak lagi menyebut `verified_at`. `sqlc vet` memastikan tidak ada query yang tersisa. |

Rename tabel, perubahan tipe, dan split kolom mengikuti pola yang sama. View kompatibilitas boleh dipakai selama transisi.

### 9.3 Aturan

| # | Aturan | Ditegakkan oleh |
|---|---|---|
| M-1 | Tidak ada perubahan destruktif (DROP, RENAME, penyempitan tipe, NOT NULL tanpa jalur aman) di rilis yang sama dengan kode yang berhenti bergantung padanya | `check_migration_phase` + `db-contract` (QG-11) |
| M-2 | Index pada tabel berisi: `CREATE INDEX CONCURRENTLY` di file berannotasi `-- +goose NO TRANSACTION` | squawk |
| M-3 | FK/CHECK baru: `NOT VALID`, lalu `VALIDATE CONSTRAINT` di migrasi terpisah | squawk |
| M-4 | `msp migrate up` menyetel `lock_timeout = '5s'` dan `statement_timeout` default per sesi. Override per file wajib dijustifikasi. Gagal karena lock → ulangi deploy | Kode `migrate`, review |
| M-5 | Backfill = job River (batched, idempotent, resumable, terpantau). Pengecualian: tabel referensi < 10.000 baris | Review |
| M-6 | Tabel tenant baru: `ENABLE` + `FORCE ROW LEVEL SECURITY` + policy berbasis `workspace_id` **di migrasi yang sama** dengan `CREATE TABLE`; index diawali `workspace_id` (04 §5) | `check_rls.sql` (QG-11) |
| M-7 | Role per 04 §5.8: `msp_owner` (DDL, hanya `migrate`), `msp_app` (`NOBYPASSRLS`), `msp_pipeline`, `msp_ops`, `msp_readonly_replica`. Grant eksplisit di migrasi. Runtime tidak pernah memakai `msp_owner`. | Ansible + test isolasi |
| M-8 | Satu runner migrasi pada satu waktu: goose session locker (advisory lock Postgres) | Kode `migrate` |
| M-9 | Versi goose berurutan (`goose -s`), tanpa *out-of-order*. Konflik nomor antar MR diselesaikan dengan rebase dan renumber | QG-11 |
| M-10 | Upgrade library River yang membawa migrasi skema River diperlakukan sebagai migrasi biasa (MR sendiri, expand-only, dicek di staging) | Review, Renovate tanpa automerge |
| M-11 | Operasi partisi (attach/detach, mis. `intel.score`, `ingest.observation`) mengikuti prosedur 04 §5 | Review |
| M-12 | Tidak ada data pribadi nyata di migrasi atau seed | Review + gitleaks |

### 9.4 Checklist review migrasi

- [ ] Fase (`expand`/`contract`) dicatat di komentar header file
- [ ] Kompatibel dengan kode rilis sebelumnya (baca & tulis); job args River kompatibel
- [ ] Estimasi baris terdampak dan durasi lock dicantumkan di MR
- [ ] Index `CONCURRENTLY` (`NO TRANSACTION`), constraint `NOT VALID` + `VALIDATE`
- [ ] Tidak ada backfill besar di dalam migrasi
- [ ] RLS + policy + grant untuk tabel tenant; `db/schema.sql` dan `internal/db/` di-regenerate
- [ ] Contract: rilis yang berhenti memakai objek sudah berjalan di production ≥ 1 rilis

### 9.5 Eksekusi per environment

| Env | Cara | Kredensial |
|---|---|---|
| local | Service `migrate` di compose (`msp migrate up`) | `.env` |
| ci | `go run ./cmd/msp migrate up` sebelum integration test; job `db:migrations` dari DB kosong; API service E2E dengan `--dev-bootstrap` | Service container |
| staging / production | Container one-off `msp-backend@<digest> migrate up` di app host (playbook `deploy.yml`), langsung ke `db-01` (bukan lewat PgBouncer transaction mode), sebelum warna baru di-start | `msp_owner` dari SOPS → tmpfs |
| production + `db-contract` | Sama, didahului `pgbackrest backup --type=incr` sebagai titik pemulihan tambahan | — |

### 9.6 PostGIS & ekstensi

- PostGIS **bukan** trusted extension, sehingga `CREATE EXTENSION postgis` butuh superuser. Di staging/production, Ansible (role `postgres`) membuat `postgis`, `pg_trgm`, `unaccent` saat provisioning DB. Migrasi pertama tetap memuat `CREATE EXTENSION IF NOT EXISTS` agar local/CI (superuser) setara. Di server, perintah itu menjadi no-op.
- Versi paket PGDG (`postgresql-16`, `postgresql-16-postgis-3`) di-pin di Ansible dan disamakan dengan image CI (parity, §3.1).
- Upgrade minor PostgreSQL/PostGIS (`ALTER EXTENSION postgis UPDATE`): playbook `db-upgrade.yml`, diuji di staging, di jendela maintenance (replica dulu, lalu switchover terencana). Upgrade major memakai `pg_upgrade --link` pada replica baru atau logical replication, lewat MR `infra/**` + rencana tertulis.

---

## 10. Rilis data & model

Kode yang mendukung versi baru di-deploy dulu (versi terdaftar, belum aktif). **Aktivasi** dilakukan lewat pipeline tag terlindungi yang namanya sama dengan versinya. Langkah berat berjalan sebagai container one-off `msp-backend@<digest> tool <aksi>` di host environment terkait, dipicu runner `deploy`. Aktor audit = user GitLab yang menjalankan job manual (FR-AUD-001).

### 10.1 Registry versi & reproducibility (NFR-DATA-01)

| Artefak | Versi | Disimpan di | Immutable |
|---|---|---|---|
| Snapshot dataset | `ds-YYYY.MM.n` | File mentah + manifest di bucket `msp-snapshots` (object lock bila didukung; fallback salinan append-only off-site), metadata di `ingest.dataset_snapshot` | Ya (FR-DATA-001) |
| Model skor | `priority-X.Y.Z` | File YAML di `apps/api/internal/intelligence/internal/scoring/models/` (di-embed ke binary) + sha256 di `intel.score_model` + `versions.lock` | Ya setelah aktif |
| Metrik geo | `geo-X.Y.Z` | Konfigurasi di repo + `versions.lock` | Ya |
| Prompt AI | `brief-X.Y.Z` | Prompt registry modul `ai` + `versions.lock` | Ya |

Setiap skor dan metrik area menyimpan `snapshot_version` + `model_version`/`geo_version` (04 §7.8, §9.1). Nightly memverifikasi recompute sampel menghasilkan hash identik. Laporan rilis disimpan di `msp-evidence` selama versi masih direferensikan data aktif + 1 tahun.

Status versi: `draft` → `staged` (tervalidasi di staging) → `active` (production) → `retired`.

### 10.2 Snapshot dataset (`ds-YYYY.MM.n`)

```
ops unggah file mentah + manifest ke bucket landing production → MR manifest `data/snapshots/ds-….yaml` → merge
 → Maintainer membuat tag ds-YYYY.MM.n → pipeline tag:
   [1] register + checksum  [2] validasi (prod, VALIDATING)  [3] rehearsal staging (sampel)
   [4] gate QG-34  [5] job manual `publish` (Maintainer)  [6] recompute skor stale  [7] verifikasi
```

| Langkah | Detail |
|---|---|
| 1. Register | Manifest: sumber, `captured_at`, periode referensi, tag lisensi (BR-PRIV-04), checksum per file. Checksum tidak cocok → berhenti. |
| 2. Validasi | Lifecycle 04 §4.2 (`VALIDATING`): parsing Input Contract, laporan validasi (jumlah baris, reject per alasan, fill-rate per field, delta vs snapshot aktif). Data tetap di zona production. |
| 3. Rehearsal staging | Sampel terstratifikasi (wilayah × kategori) tanpa person/kontak pribadi, bila lisensi mengizinkan (Q-05). Pipeline normalize → resolve → verify → skor → geo. Diukur: ER golden metrics, distribusi skor vs snapshot sebelumnya, durasi (ekstrapolasi ke NFR-PERF-07). |
| 4. Gate QG-34 | Threshold di bawah. Laporan dilampirkan ke pipeline dan `msp-evidence`. |
| 5. Publish | Job manual (Maintainer) → `msp tool snapshot-publish` (setara `/v1/ops/snapshots/{id}/publish`, 04 §11) → pointer snapshot aktif diganti atomik. Kill switch `pipeline.publish` (04 §18.3) dapat menahan langkah ini. |
| 6. Recompute | Skor yang lebih tua dari input ditandai stale dan di-recompute (FR-INTL-011, 04 §7.9). |
| 7. Verifikasi | Dashboard Data Health per snapshot (FR-OPS-001): evidence coverage ≥ 95%, rasio verifikasi, stale, kandidat duplikat. |

**Threshold QG-34 (default awal, dikalibrasi setelah Q-01/Q-02):**

| Cek | Threshold |
|---|---|
| Baris reject tanpa alasan | 0 |
| Reject rate total | ≤ 5% (2–5% → wajib catatan di deskripsi tag) |
| Fill-rate `source_record_id` / nama / kategori sumber | 100% / ≥ 99% / ≥ 95% |
| Alamat atau koordinat | ≥ 95% |
| `source_record_id` duplikat | 0 |
| Penurunan fill-rate field apa pun vs snapshot aktif | ≤ 10 poin persentase (lebih → wajib catatan) |
| Perubahan jumlah record vs snapshot aktif | −10% s.d. +30% (di luar → wajib catatan) |
| Establishment ber-koordinat yang terpetakan ke kelurahan | ≥ 90% (FR-DATA-009) |
| Tag lisensi | Ada |
| ER golden set pada rehearsal | Precision ≥ 98%, recall ≥ 90% |

**Rollback:** snapshot immutable dan ID entity stabil (FR-DATA-007). Rollback = publish ulang snapshot sebelumnya + recompute (04 §4.2).

### 10.3 Model skor (`priority-X.Y.Z`) dan geo (`geo-X.Y.Z`)

| Langkah | Detail | Gate |
|---|---|---|
| 1. MR konfigurasi | Versi baru (versi lama tidak boleh berubah). Unit test + fixture "golden scores" → diff di komentar MR | QG-23, QG-05 |
| 2. Merge & deploy | Versi terdaftar `draft` di staging & production | Pipeline biasa |
| 3. Tag versi | Maintainer membuat tag `priority-X.Y.Z` → pipeline aktivasi | — |
| 4. Recompute offline (staging) | `msp tool score-reproduce` menghitung versi lama & baru pada sampel staging (ICP/preset Offering representatif) | — |
| 5. Laporan regresi | Distribusi skor per band, matriks transisi band (HOT/WARM/COLD/UNSCORED), pergeseran per faktor, overlap top-50 per ICP sampel, perubahan coverage. Geo: distribusi metrik area, jumlah area coverage LOW, perubahan peringkat territory. Determinisme: hitung 2× → hash identik | QG-24 |
| 6. Approval | Job manual `activate` (Maintainer) setelah approval data-intel + product tercatat di issue rilis model. Bila transisi band > 15% sampel atau overlap top-50 < 70%, product wajib menulis justifikasi. | QG-24 |
| 7. Aktivasi | `msp tool score-model-activate` → versi `active` → recompute bertahap (epoch baru diaktifkan atomik, ADR-16; NFR-PERF-05). Epoch versi lama disimpan selama bake 7 hari. | — |
| 8. Pantau | Intelligence Health (FR-OPS-003): distribusi per versi, feedback skor (FR-INTL-010) | — |

**Rollback:** aktifkan kembali epoch versi sebelumnya (tersimpan selama bake). Setelah bake, versi lama tetap dapat direproduksi dari snapshot + versi.

### 10.4 Prompt AI (`brief-X.Y.Z`)

| Langkah | Detail | Gate |
|---|---|---|
| 1. MR prompt | Versi baru. Ganti provider/model (D-15) = bump minor + label `ai-eval` | QG-23 |
| 2. Eval MR | Rekaman (deterministik) di setiap MR. Live (manual) bila prompt berubah: eval set **sintetis** (≥ 200 profil company-level: data jarang/Unknown, multi-cabang, evidence bertentangan, person hanya sebagai role). Metrik: grounding (validator sitasi FR-AI-004) ≥ 98%; nilai contact point pribadi di prompt = 0; format valid 100%; error/refusal ≤ 2%; latency p95 dan biaya per brief dilaporkan (NFR-PERF-06, NFR-COST-01) | QG-25 |
| 3. Tag `brief-X.Y.Z` | Eval diulang di staging dengan set diperluas (company-level, D-14) | QG-25 |
| 4. Aktivasi bertahap | Flag `rel.ai.brief-<versi>`: workspace internal → cohort beta → 100%. Cache brief berbasis hash input **termasuk** versi prompt (FR-AI-003) | — |
| 5. Pantau | Kegagalan AI, feedback 👍/👎 (FR-AI-007), biaya per brief. Alert AI failure > 5% (RG-5) | — |

**Rollback:** aktifkan kembali versi prompt sebelumnya. Kill switch `ai.generation` (FR-AI-006) sebagai upaya terakhir.

---

## 11. Infrastructure as Code (Ansible)

### 11.1 Layout

```
infra/
├─ ansible/
│  ├─ ansible.cfg           host key checking aktif; known_hosts di repo
│  ├─ requirements.yml      collection di-pin: community.docker, community.postgresql, community.sops, ansible.posix
│  ├─ inventories/
│  │  ├─ staging/     hosts.yml, group_vars/{all,app,db,stor}/vars.yml, secrets/*.sops.yaml
│  │  └─ production/  (struktur sama)
│  ├─ roles/     base, users, firewall (nftables), wireguard, fail2ban, luks_unlock (Clevis+Tang), docker,
│  │             traefik, msp_app (compose blue/green, tmpfs secret), postgres (PG16+PostGIS, role, ekstensi,
│  │             replikasi, PgBouncer), pgbackrest, redis, seaweedfs (master/volume/filer/S3, filer.sync),
│  │             monitoring_agent, monitoring_stack, gitlab (Omnibus, server hooks, backup), gitlab_runner
│  ├─ playbooks/ site.yml (juga untuk drift --check), deploy.yml, rollback.yml, tool-job.yml, rehydrate.yml,
│  │             db-failover.yml, db-upgrade.yml, restore-drill.yml, rotate-secrets.yml
│  └─ molecule/
├─ docker/                  compose lokal & CI
└─ keys/                    cosign.pub, kunci publik age per environment
```

### 11.2 Pipeline infra

| Event | Langkah | Runner |
|---|---|---|
| MR menyentuh `infra/**` | ansible-lint, yamllint, `--syntax-check`, Molecule, Trivy config (QG-21). Tanpa akses ke server (MR tidak menerima variabel protected) | `ci-docker` |
| Merge ke `main` | `site.yml --check --diff` (staging & production) → diff sebagai artifact + komentar MR via API → **apply staging** otomatis | `deploy` |
| Tag `infra-YYYY.MM.DD.n` (Maintainer) | `--check --diff` production diulang → job manual **apply production** (`serial: 1`) | `deploy` |
| Schedule `drift` | `--check --diff` harian | `deploy` |

`--check` tidak sempurna: modul `command`/`shell` dilewati. Karena itu task wajib idempoten (`changed_when`/`creates` yang benar), dan task non-idempoten diberi tag `never_check`. Diff yang menyentuh secret disembunyikan (`no_log`, `diff: false`).

### 11.3 Penerapan perubahan

- **Tidak ada SSH manual untuk perubahan konfigurasi.** Semua lewat MR → tag. SSH manual hanya untuk diagnosis atau break-glass, tercatat (auditd + log WireGuard), dan ditindaklanjuti MR adopsi.
- Patch keamanan OS: unattended-upgrades untuk paket security (tanpa auto-reboot di host DB). Reboot terjadwal lewat playbook `serial: 1` di jendela maintenance. Upgrade GitLab mengikuti rilis keamanan (severity kritis ≤ 7 hari), dengan backup sebelum upgrade.

### 11.4 Drift

Perubahan yang terdeteksi job `drift` diselesaikan ≤ 2 hari kerja dengan MR adopsi (kode mengikuti server) atau re-apply (server mengikuti kode). Setelan proyek GitLab (protected branch/tag, merge method, variables) juga dibandingkan dengan `gitlab_settings`.

### 11.5 Least privilege di server

- User deploy `msp-deploy` di app host adalah anggota grup `docker` (setara root). Karena itu kunci SSH-nya hanya ada di variabel protected ber-scope environment, dan login dibatasi dari IP `ops-01`.
- Kunci SSH deploy berbeda per environment. Kunci admin manusia bersifat personal dan dicabut saat offboarding (role `users`).
- Port DB/Redis/SeaweedFS tidak pernah publik. S3 gateway SeaweedFS hanya terekspos lewat Traefik `files.<domain>` untuk presigned URL; master/volume/filer tidak pernah terekspos.

### 11.6 Backup & disaster recovery

| Data | Mekanisme | Tujuan | Retensi |
|---|---|---|---|
| PostgreSQL | **pgBackRest**: full mingguan (Minggu 01:00), diff harian (01:00), WAL archive kontinu (`archive_timeout = 60s`, async). Cipher `aes-256-cbc`, passphrase per repo di SOPS | repo1: S3 gateway SeaweedFS di `stor-01` (pemulihan cepat). repo2: **off-site S3-compatible** di lokasi lain, tetap di Indonesia | Full 4 minggu. PITR window 14 hari (Proposed) |
| Replika DB | Streaming replication `db-02` (async) | Kegagalan host (RPO ≈ detik) | — |
| Object storage (SeaweedFS) | Replikasi volume di `stor-01` + `weed filer.sync` (async, kontinu) ke filer SeaweedFS off-site, atau `weed filer.backup` ke S3-compatible off-site. Versioning/object lock/SSE **bila didukung**; fallback: salinan append-only off-site yang tidak bisa dihapus dari zona prod | Off-site | Lifecycle per bucket (export 24 jam per FR-EXP-001; snapshot permanen) |
| Redis | Tidak di-backup (hanya cache/rate limit; antrean job ada di Postgres) | — | — |
| GitLab | `gitlab-backup create` + `gitlab-secrets.json` + `gitlab.rb` (terenkripsi) + data registry, harian 02:00 | `stor-01` + off-site | 14 hari |
| Kunci | Kunci age privat & passphrase pgBackRest: salinan offline di 2 Maintainer | — | — |

**Target:** RPO ≤ 15 menit (praktiknya ≤ 1–2 menit), RTO ≤ 4 jam (NFR-AVAIL-02). Prometheus memantau umur backup, lag WAL archive, dan lag replikasi (alert per 04 §16).

| Skenario | Respons | RTO perkiraan |
|---|---|---|
| `db-01` mati | `db-failover.yml`: promote `db-02`, arahkan PgBouncer/app, bangun ulang `db-01` sebagai standby | ±30 menit |
| Korupsi data logis | PITR pgBackRest ke instance baru (§14.3) | 1–3 jam |
| `app-01` mati | Provision ulang via Ansible di server cadangan + deploy digest terakhir (`app-02` menurunkannya ke menit) | 1–3 jam |
| Kehilangan DC | Provision server di lokasi lain + restore dari repo off-site | Tergantung SLA provider (Q-CI-04). **Bisa melebihi 4 jam.** |

---

## 12. Secrets & konfigurasi

### 12.1 SOPS + age

```yaml
# .sops.yaml (ilustratif)
creation_rules:
  - path_regex: infra/ansible/inventories/production/secrets/.*\.sops\.yaml$
    age: age1prod...deployrunner,age1maint...alice,age1maint...bob
  - path_regex: infra/ansible/inventories/staging/secrets/.*\.sops\.yaml$
    age: age1stg...deployrunner,age1eng...team
  - path_regex: infra/secrets/signing\.sops\.yaml$
    age: age1ci...signing,age1maint...alice,age1maint...bob
```

- Engineer dapat **mengenkripsi** secret production (kunci publik di repo) tetapi tidak dapat mendekripsinya. Dekripsi production hanya oleh runner `deploy` (job tag) dan Maintainer.
- Perubahan secret = MR biasa dengan diff terenkripsi per nilai: reviewer melihat *key* yang berubah, bukan nilainya. Pre-commit hook dan `ci:lint` menolak file secret tanpa metadata `sops`.

### 12.2 GitLab CI/CD variables

| Variabel | Tipe | Protected | Scope environment | Isi |
|---|---|---|---|---|
| `SOPS_AGE_KEY` | File, masked | Ya | `staging` / `production` | Kunci age privat runner deploy |
| `DEPLOY_SSH_KEY` | File | Ya | `staging` / `production` | Kunci SSH user `msp-deploy` |
| `SOPS_AGE_KEY_SIGNING` | File | Ya | `signing` | Membuka `cosign.key` + password |
| `RELEASE_TOKEN` | Masked | Ya | `release` | Token bot rilis (Maintainer) |
| `RENOVATE_TOKEN` | Masked | Ya | `renovate` | Token bot Renovate |
| `LLM_EVAL_API_KEY` | Masked | Tidak (MR perlu) | `ai-eval` | Key LLM khusus eval, kuota & budget kecil |
| `MR_GATE_TOKEN` | Masked | Tidak | — | `read_api`, role Reporter |

Job hanya menerima variabel ber-scope bila menyatakan `environment:` yang sesuai. Job non-deploy memakai `environment: { name: <x>, action: prepare }`. Runner tidak menyimpan secret di disk, dan workspace job dibersihkan setelahnya.

### 12.3 Secret & konfigurasi di server

- Ansible mendekripsi SOPS di runner `deploy` lalu menulis setiap secret sebagai **file** di **tmpfs** `/run/msp/<warna>/` (mode 0400). File di-mount ke container sebagai Compose `secrets:` (`/run/secrets/<nama>`). Binary membaca konvensi **`MSP_<NAMA>_FILE`** (mis. `MSP_DATABASE_URL_APP_FILE=/run/secrets/database_url_app`), sehingga secret tidak tampil di `docker inspect`. Nilai secret tidak pernah dikirim sebagai env biasa; binary menolak start di staging/production bila secret diberikan tanpa `_FILE` (04 §18.1).
- Konsekuensi tmpfs (ADR-22): setelah reboot, secret hilang. Unit systemd `msp-rehydrate` memicu pipeline `rehydrate` lewat *pipeline trigger token* (token hanya bisa memicu, tidak membaca secret). Job berjalan di runner `deploy` dan menjalankan `rehydrate.yml`. Bila gagal, on-call menjalankannya manual (runbook).
- Konfigurasi non-secret = env var berprefiks `MSP_` sesuai 04 §18.1 (`MSP_ENV`, `MSP_VERSION`, `MSP_BASE_URL`, `MSP_S3_ENDPOINT_URL`, dll.; pengecualian: variabel standar `OTEL_*` seperti `OTEL_EXPORTER_OTLP_ENDPOINT`), dari variabel inventory Ansible. Binary gagal start bila variabel wajib hilang. `.env.example` di root berisi semua variabel dengan nilai dummy aman.

### 12.4 Jangan pernah mencetak secret & rotasi

Tidak ada `set -x` di job yang menyentuh secret. Task Ansible yang menyentuh secret memakai `no_log: true`. Log aplikasi memakai redaksi terstruktur (BR-PRIV-01, QG-10). Artifact CI hanya berisi data sintetis (trace Playwright 7 hari, diunggah hanya saat gagal).

| Secret | Rotasi terjadwal | Rotasi segera bila |
|---|---|---|
| Password role DB, kunci S3 SeaweedFS, Redis | 90 hari (`rotate-secrets.yml`, dual credential selama transisi) | Offboarding, dugaan bocor |
| Kunci age per environment | 12 bulan (`sops updatekeys`) | Offboarding Maintainer, runner dikompromikan |
| Kunci SSH deploy | 90 hari | Sama |
| Kunci cosign | 12 bulan; verifikasi menerima kunci lama + baru selama 30 hari | Dugaan bocor (semua image ditandatangani ulang) |
| Token GitLab (bot, Renovate, trigger) | Kedaluwarsa ≤ 365 hari, diganti 30 hari sebelumnya | Sama |
| API key LLM/email | 90 hari | Sama |
| Kunci master webhook (`MSP_WEBHOOK_SECRET_MASTER_KEYS_FILE`, berversi) | 12 bulan (lihat bawah) | Dugaan bocor |

**Kunci master webhook.** Secret per endpoint (FR-EXP-003) dienkripsi aplikasi (AES-256-GCM) dengan key ring berversi (`secret_kid`, ADR-22). Rotasi: (1) tambah versi kunci baru ke SOPS dan deploy (dekripsi menerima semua versi aktif, enkripsi memakai yang terbaru); (2) job River mengenkripsi ulang semua `secret_ciphertext`; (3) setelah 100% + 7 hari, versi lama dihapus.

---

## 13. Feature flags & progressive delivery

### 13.1 Tipe flag

| Tipe | Penamaan | Contoh | Umur | Saat flag store gagal |
|---|---|---|---|---|
| Release | `rel.<area>.<nama>` | `rel.geo.territory-ranking`, `rel.ai.brief-1.1.0` | Sementara: dihapus ≤ 30 hari setelah 100% | **Off** |
| Ops kill switch (FR-OPS-007, FR-AI-006) | Nama kapabilitas (04 §18.3) | `ai.generation`, `export.create`, `webhook.delivery`, `upload.create`, `pipeline.publish`, `scoring.bulk_rescore` | Permanen | Nilai cache terakhir. Tanpa cache → kapabilitas **on** |
| Fault injection | `chaos.<nama>` | `chaos.webhook-5xx` | Permanen | Off. Ditolak di production |
| Eksperimen | `exp.<nama>` | (V1+) | Sementara | Off |

**Permission dan plan bukan flag.** Entitlement (FR-BILL-003) dan RBAC (FR-AUTH-004) ditegakkan di server. Flag tidak pernah menjadi satu-satunya penghalang akses.

### 13.2 Mekanisme

- **Registry di repo** (`apps/api/internal/platform/flags/registry.yaml`): nama, tipe, owner, deskripsi, tanggal dibuat, kedaluwarsa (wajib untuk `rel.`/`exp.`), default. CI memeriksa bahwa setiap flag yang dipakai kode (Go & web) terdaftar, dan setiap flag terdaftar dipakai.
- **Nilai runtime** di `platform.feature_flag` (scope GLOBAL → PLAN → WORKSPACE), diubah lewat ops console (`/v1/ops/feature-flags/{key}`) dan **tercatat di audit** (FR-OPS-007). Cache in-process 30 detik + invalidasi pub/sub, sehingga kill switch berlaku ≤ 1 menit (04 §18.3).
- Antarmuka evaluasi tipis yang kompatibel OpenFeature, agar bisa berpindah ke vendor/Unleash self-hosted kelak tanpa mengubah call site.

### 13.3 Lifecycle & cleanup

```
dibuat (MR + registry) → dark launch (off di prod) → workspace internal MSP → cohort beta (allowlist)
→ persentase per workspace (hash workspace_id: 10% → 50% → 100%) → MR hapus flag + kode lama (≤ 30 hari)
```

Job `weekly` membuat issue untuk flag `rel.` yang melewati kedaluwarsa atau sudah 100% > 30 hari. CI gagal bila flag `rel.` lewat kedaluwarsa > 30 hari tanpa perpanjangan (MR yang mengubah tanggal + alasan). Rollout selalu per **workspace**, agar pengalaman konsisten dalam satu tim.

### 13.4 Dark launch

Deploy (perubahan kode) dipisahkan dari release (perubahan perilaku). Untuk pipeline berat (mis. faktor skor baru), dark launch menghitung hasil bayangan tanpa ditampilkan, lalu hasilnya dibandingkan di laporan regresi (§10.3).

---

## 14. Rollback & incident

### 14.1 Rollback aplikasi

| Situasi | Tindakan | Target waktu |
|---|---|---|
| Selama canary/bake | Otomatis oleh blok `rescue` deploy (§6.3) | < 1 menit |
| Setelah bake | Job manual **`rollback:production`** (Maintainer) → `rollback.yml -e to=previous`: start warna lama dengan digest sebelumnya (image masih di cache host), worker dulu, alihkan traffic 100%, smoke | ≤ 5 menit |
| Lebih dari satu rilis ke belakang | `rollback.yml -e to=vX.Y.Z`. Ditolak bila ada migrasi contract di antara keduanya | ≤ 15 menit |
| `ops-01` (GitLab) tidak tersedia | Maintainer menjalankan `rollback.yml` dari laptop via WireGuard (kunci break-glass), lalu dicatat di issue | ≤ 30 menit |

Rollback tidak menjalankan migrasi dan aman karena skema bersifat expand-only terhadap N-1 (§9.1). Host menyimpan image 5 rilis terakhir.

### 14.2 Matikan lewat flag

Langkah tercepat untuk kapabilitas bermasalah: kill switch (`ai.generation`, `export.create`, `webhook.delivery`, `upload.create`, `scoring.bulk_rescore`) atau flag `rel.*` ke off (≤ 1 menit, tercatat di audit). Langkah ini didahulukan sebelum rollback bila masalah terlokalisasi.

### 14.3 Database

- **Forward fix:** migrasi perbaikan lewat pipeline normal (dipercepat: review on-call + Maintainer).
- **PITR** (pgBackRest) hanya untuk korupsi/kehilangan data yang tidak bisa diperbaiki forward. Keputusan ada di incident commander + founder, karena data setelah titik restore hilang (termasuk ledger dan state job River, yang harus direkonsiliasi). Restore ke instance baru, lalu PgBouncer diarahkan ulang. Prosedur dilatih di drill bulanan (§7.1).
- **Kegagalan host primary:** `db-failover.yml` (§11.6).

### 14.4 Rollback data & model

| Artefak | Rollback |
|---|---|
| Snapshot dataset | Publish ulang snapshot sebelumnya + recompute (§10.2) |
| Versi skor/geo | Aktifkan kembali epoch versi sebelumnya (tersimpan selama bake 7 hari) |
| Versi prompt | Aktifkan kembali versi sebelumnya; kill switch `ai.generation` bila perlu |

### 14.5 Incident

| Severity | Contoh | Respons | Komunikasi |
|---|---|---|---|
| **SEV1** | Production down; **kebocoran data lintas tenant**; dugaan kegagalan pelindungan data pribadi; tagihan ganda massal; DC/jaringan down | Segera, 24/7. Incident commander ditunjuk | Kanal insiden + founder ≤ 15 menit; customer terdampak ≤ 4 jam (email); update tiap 1 jam |
| **SEV2** | Fitur inti terganggu (search, export, webhook, unlock), SLO burn tinggi, replikasi/backup gagal > 1 jam | ≤ 30 menit jam kerja, ≤ 1 jam di luar | Kanal insiden; customer bila > 1 jam |
| **SEV3** | Degradasi dengan workaround; drill gagal | Hari kerja yang sama | Issue |
| **SEV4** | Kosmetik, internal | Backlog | — |

Peran: incident commander, komunikasi, scribe. Mulai dengan containment (kill switch, rollback, blokir akses), baru diagnosis.

**Hook UU PDP (BR-PRIV-03) di runbook.** Insiden yang menyangkut data pribadi (contact point person, upload customer, akses lintas tenant) langsung diberi label `pdp-suspected`, yang memicu:
1. Pencatatan waktu insiden diketahui. **Hitung mundur 3×24 jam** untuk notifikasi kegagalan pelindungan data dimulai.
2. Eskalasi ke founder, penanggung jawab pelindungan data, dan konsultan hukum. Lingkup dan pihak yang wajib diberi tahu divalidasi konsultan hukum (BR-PRIV-03).
3. Preservasi bukti: `platform.audit_log` (immutable), log Traefik/aplikasi di Loki, auditd, log WireGuard, log akses S3 gateway. Retensi diperpanjang, tidak ada penghapusan.
4. Draf notifikasi (data terdampak, waktu dan cara kejadian, upaya penanganan dan pemulihan) disiapkan paralel dengan penanganan teknis.

**Post-incident review** (wajib untuk SEV1/2): blameless, ≤ 5 hari kerja, berisi timeline, akar masalah, dan action item ber-owner dan bertenggat (issue `pir`). Setiap insiden menghasilkan minimal satu test regresi atau alert baru. Metrik DORA diperbarui (§16.2).

### 14.6 Checklist verifikasi pasca-deploy

- [ ] Semua container warna aktif *healthy*; Traefik hanya merujuk warna aktif
- [ ] Smoke production lulus (workspace sintetis)
- [ ] 5xx dan p95 search/profil dalam threshold (§6.3) selama 30 menit
- [ ] Worker: antrean River normal, tidak ada lonjakan job discarded, rasio gagal < 5%; worker warna lama sudah berhenti
- [ ] Log container migrasi bersih; versi goose = versi yang diharapkan
- [ ] GlitchTip: tidak ada issue baru severity tinggi pada release ini
- [ ] Flag sesuai rencana rollout
- [ ] Deployment tercatat di GitLab environment; anotasi Grafana ada
- [ ] Catatan rilis dipublikasikan; perubahan yang terlihat customer dikomunikasikan (id-ID)

---

## 15. Supply chain security

### 15.1 Platform & pipeline

| Kontrol | Penerapan |
|---|---|
| Image job di-pin | Setiap `image:`/`services:` memakai `@sha256:<digest>`. Tool non-Go (sqlc, golangci-lint, squawk, oasdiff, Spectral, ajv, gitleaks, Trivy, syft, cosign, OSV-Scanner, Semgrep, Schemathesis, ansible, sops, age, commitlint, k6) dibundel dalam image **`ci-tools`** dari `ci/images/tools/Dockerfile` (versi di-pin + checksum, diperbarui Renovate). Tool Go di-pin lewat directive `tool` di `go.mod`. |
| Toolchain Go | `GOTOOLCHAIN=local` di CI (tidak mengunduh toolchain otomatis); versi image Go = directive `toolchain` di `go.mod` (cek parity). `GOFLAGS=-mod=readonly`, `go mod verify`, verifikasi checksum via `GOSUMDB`. |
| `include:` CI | Hanya `include: local`. Template eksternal tidak dipakai. |
| Tanpa unduhan liar | Larangan `curl \| sh`/`wget \| bash` di job (dicek `ci:lint`) |
| Runner terpisah | `ci-docker` (non-privileged, `allowed_images` dibatasi), `ci-build` (BuildKit rootless, tanpa `privileged`), `deploy` (protected, di `ops-01`). Runner MR tidak punya rute ke prod/stg. |
| Variabel | Masked + protected + scope environment (§12.2). Variabel protected tidak pernah tersedia di pipeline MR. |
| Push protection | Server hook `pre-receive` menjalankan gitleaks pada setiap push. Push yang mengandung secret ditolak. |
| Akun & GitLab | 2FA wajib, sign-up nonaktif, Maintainer minimal, token berkedaluwarsa, review akses per kuartal. Patch keamanan GitLab kritis ≤ 7 hari. UI hanya via WireGuard; sertifikat via ACME DNS-01 atau CA internal |

### 15.2 Dependency

- Instalasi beku: `go mod download` + `go mod verify` (dengan `go.sum`), `pnpm install --frozen-lockfile`.
- `security:deps` + `backend:vuln` di setiap MR + rescan nightly (QG-19, QG-20). Pengecualian bertanggal kedaluwarsa.
- Base image (`golang`, `distroless/static`, `node`) di-pin digest dan diperbarui Renovate.

### 15.3 Artefak: SBOM, provenance, signing

| Artefak | Dibuat | Disimpan | Retensi |
|---|---|---|---|
| SBOM SPDX (syft) | Setiap image `main` | Attestation cosign di registry + artifact + `msp-evidence` | ≥ 1 tahun, atau selama digest masih di-deploy |
| Provenance | BuildKit `mode=max` (image index) | Registry | Sama |
| Tanda tangan | cosign key pair setelah QG-27 lolos (`--tlog-upload=false`) | Registry (tag `.sig`/`.att`) | Sama |

Verifikasi saat deploy (QG-28): `cosign verify --key infra/keys/cosign.pub --insecure-ignore-tlog=true` + `cosign verify-attestation --type spdxjson`. Kunci privat hanya bisa dibuka job `sign:images` di pipeline `main`/`release/*` (scope `signing`). Cleanup policy registry tidak boleh menghapus tag `.sig`/`.att` milik digest yang masih disimpan.

### 15.4 Renovate (self-hosted, scheduled pipeline)

| Kebijakan | Nilai |
|---|---|
| Eksekusi | Image `renovate/renovate` (pin digest), `platform: gitlab`, token bot di scope `renovate`. Manager: `gomod` (termasuk directive `tool`), `npm`/pnpm, Dockerfile, `gitlab-ci`, Ansible Galaxy |
| `minimumReleaseAge` | 3 hari untuk semua update. Security update tanpa penundaan |
| Grouping | `go-tools`, `golangci-lint`, `js-lint`, `storybook`, `playwright`, `ci-images` (digest), `docker-base`, `ansible-collections`. `postUpdateOptions: ["gomodTidy"]` |
| Automerge | **Patch** Go/JS dan **minor devDependency** JS dengan pipeline hijau (label `automerge`, `mr:gate` mengizinkan tanpa approval). Tidak pernah automerge untuk River, pgx, sqlc, goose, oapi-codegen, `next`, `react`, collection Ansible, base image. Major selalu manual. |
| Batas | Maks 5 MR terbuka, 2 MR/jam. Dependency Dashboard issue aktif |

---

## 16. Observability pipeline & DORA

### 16.1 Deployment marker & anotasi rilis

Setiap deploy (staging dan production) menghasilkan:
- **Deployment GitLab** (`environment:`) beserta SHA dan job. Ini sumber data DORA.
- **Release GlitchTip**: `msp@vX.Y.Z` (production) atau `msp@sha-<short>` (staging). Source map web diunggah saat deploy dan **tidak** disajikan publik.
- **Anotasi Grafana** di dashboard SLO (latency, error rate, antrean River, replikasi DB).
- Atribut resource OpenTelemetry di setiap service: `service.name`, `service.version` (= `MSP_VERSION`), `deployment.environment.name`, revisi git (dari buildinfo binary), dan `msp.color` (dipakai query canary §6.3).

Stack (ADR-21, di `ops-01`): OTel Collector di setiap host → Prometheus (metrik, 30 hari), Loki (log JSON terstruktur, 30 hari, PII teredaksi), Tempo (trace, 7 hari), GlitchTip (error, 90 hari). Alertmanager → kanal chat + email on-call. Blackbox exporter memeriksa endpoint dari dalam DC. **Probe eksternal** di luar DC (VPS kecil di lokasi lain atau layanan uptime) memeriksa `app.<domain>` dan menerima heartbeat *watchdog* Alertmanager (dead man's switch), sehingga DC down atau `ops-01` mati tetap terdeteksi.

### 16.2 DORA (target tahap MVP)

| Metrik | Definisi operasional | Target MVP 1 | Sumber |
|---|---|---|---|
| Deployment frequency | Deploy production sukses per minggu | ≥ 2/minggu (staging: setiap merge) | GitLab Deployments API |
| Lead time for changes | Merge ke `main` → berjalan di production, p50 | ≤ 2 hari kerja | Commit + Deployments |
| Change failure rate | Deploy production yang memicu rollback, hotfix, atau SEV1/2 | ≤ 15% | Deployments + label insiden |
| Failed deployment recovery time | Deteksi → pulih | ≤ 1 jam (rollback ≤ 5–15 menit) | Issue insiden |

Dihitung script mingguan (GitLab API → Prometheus/Grafana, karena dashboard DORA GitLab berbayar) dan ditinjau di retro engineering bulanan.

### 16.3 Kesehatan CI

| Metrik | Target | Tindakan bila meleset |
|---|---|---|
| Durasi pipeline MR | p50 ≤ 10 menit, p90 ≤ 15 menit | Profil job; tambah concurrency/shard; pindahkan ke nightly |
| Merge → `staging-verified` | p50 ≤ 30 menit | Paralelkan E2E staging |
| Antre job (pending) | p90 ≤ 1 menit | Tambah `concurrent` runner atau kapasitas `ci-01` |
| Flaky rate & karantina | ≤ 1% job lulus setelah retry (05 §9.2); test ter-karantina ≤ **1%** suite | **Karantina** hanya dengan issue berlabel `flaky` + owner, maks **7 hari** (05 §5.4). Penanda di kode: test Go memanggil helper `testutil.Quarantine(t, "#<issue>")` (skip kecuali `MSP_QUARANTINE=run`), Playwright memakai tag `@quarantine` (dikecualikan di MR via `--grep-invert`). Test ter-karantina tetap jalan di nightly dan tidak memblokir MR. Job `quarantine-check` gagal bila ada penanda tanpa issue/owner, lewat 7 hari, atau total > 1% suite |
| `main` merah | ≤ 1 jam per kejadian | Stop-the-line (§5.5) |
| Disk `ci-01`/registry | < 80% | Cleanup policy, prune cache |

Data dikumpulkan **gitlab-ci-pipelines-exporter** dan laporan JUnit, lalu ditampilkan di dashboard Grafana "CI health".

---

## 17. Contoh konfigurasi (ilustratif)

### 17.1 `.gitlab-ci.yml` (dipersingkat)

Hanya job representatif yang ditampilkan. Job lain di §4.2 dan §5 mengikuti pola yang sama.

```yaml
workflow:
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
    - if: $CI_COMMIT_BRANCH && $CI_OPEN_MERGE_REQUESTS
      when: never                                        # cegah pipeline ganda branch + MR
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH || $CI_COMMIT_BRANCH =~ /^release\/v\d+\.\d+$/
    - if: $CI_COMMIT_TAG =~ /^v\d+\.\d+\.\d+$/ || $CI_COMMIT_TAG =~ /^(infra|ds|priority|geo|brief)-/
    - if: $CI_PIPELINE_SOURCE == "schedule" || $CI_PIPELINE_SOURCE == "trigger"

include:                             # hanya local (§15.1)
  - local: ci/mr.yml                 # lint, contracts, tokens, web, security, e2e
  - local: ci/data-release.yml       # tag ds-*, priority-*, geo-*, brief-*
  - local: ci/infra.yml              # MR infra, apply staging, tag infra-*
  - local: ci/scheduled.yml          # nightly, drift, renovate, drill, rehydrate, rc

stages: [validate, test, build, scan, deploy-staging, verify, release, deploy-production]

variables:
  TOOLS_IMAGE: $CI_REGISTRY/msp/ci-tools@sha256:<digest>      # Go toolchain + tool non-Go
  POSTGIS_IMAGE: postgis/postgis:16-3.5@sha256:<digest>       # = versi PGDG di Ansible
  GOCACHE: $CI_PROJECT_DIR/.cache/go-build
  GOMODCACHE: $CI_PROJECT_DIR/.cache/go-mod
  GOTOOLCHAIN: local
  GOFLAGS: -mod=readonly
  FF_NETWORK_PER_BUILD: "true"

default: { image: $TOOLS_IMAGE, tags: [ci-docker], interruptible: true }

.if-mr:   &if-mr   { if: '$CI_PIPELINE_SOURCE == "merge_request_event"' }
.if-main: &if-main { if: '$CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH' }
.if-rel:  &if-rel  { if: '$CI_COMMIT_TAG =~ /^v\d+\.\d+\.\d+$/' }
.go:
  cache: { key: { files: [apps/api/go.sum] }, paths: [.cache/go-build, .cache/go-mod] }
  rules:
    - <<: *if-mr
      changes: { paths: [apps/api/**/*, data/**/*], compare_to: refs/heads/main }
    - *if-main

mr:gate:
  stage: .post
  needs: []
  rules: [{ <<: *if-mr, when: manual, allow_failure: false }]   # pipeline "blocked" → MR tak bisa di-merge
  script: [scripts/ci/mr_gate --codeowners .gitlab/CODEOWNERS]   # QG-02

backend:test:
  stage: test
  extends: .go
  needs: []
  services:
    - { name: $POSTGIS_IMAGE, alias: postgres, variables: { POSTGRES_USER: msp_owner, POSTGRES_PASSWORD: ci, POSTGRES_DB: msp } }
  variables:
    MSP_ENV: ci
    MSP_DATABASE_URL_DIRECT: postgres://msp_owner:ci@postgres:5432/msp?sslmode=disable   # migrate (nilai dummy CI)
    MSP_DATABASE_URL_APP: postgres://msp_app:ci@postgres:5432/msp?sslmode=disable        # NOBYPASSRLS
  script:
    - cd apps/api && go run ./cmd/msp migrate up
    - go tool gotestsum --junitfile report.xml -- -race -tags=integration -coverpkg=./... -coverprofile=cover.out ./...  # QG-05
    - go tool gotestsum --junitfile tenant.xml -- -tags=integration -count=1 -run 'TenantIsolation' ./...               # QG-09
    - ../../scripts/ci/go_coverage_gate cover.out --total 75 --exclude internal/db,internal/api
        --pkg ./internal/identity/internal/normalize/...=85 --pkg ./internal/identity/internal/er/...=85
        --pkg ./internal/intelligence/internal/scoring/...=85                                                         # QG-06/07
    - go tool gocover-cobertura < cover.out > coverage.xml
  coverage: '/total:\s+\(statements\)\s+(\d+\.\d+)%/'
  artifacts:
    when: always
    reports:
      junit: [apps/api/report.xml, apps/api/tenant.xml]
      coverage_report: { coverage_format: cobertura, path: apps/api/coverage.xml }

db:migrations:
  stage: test
  extends: .go
  needs: []
  services: [{ name: $POSTGIS_IMAGE, alias: postgres, variables: { POSTGRES_USER: msp_owner, POSTGRES_PASSWORD: ci, POSTGRES_DB: msp } }]
  variables: { MSP_DATABASE_URL_DIRECT: "postgres://msp_owner:ci@postgres:5432/msp?sslmode=disable" }
  rules:
    - <<: *if-mr
      changes: { paths: [apps/api/db/**/*, apps/api/sqlc.yaml, apps/api/internal/db/**/*], compare_to: refs/heads/main }
  script:                                                                                    # QG-11
    - cd apps/api && ../../scripts/ci/check_goose_versions db/migrations origin/main
    - go run ./cmd/msp migrate up
    - pg_dump --schema-only --no-owner "$MSP_DATABASE_URL_DIRECT" | ../../scripts/ci/normalize_dump | diff -u db/schema.sql -
    - sqlc diff && sqlc vet
    - psql "$MSP_DATABASE_URL_DIRECT" -v ON_ERROR_STOP=1 -f ../../scripts/ci/check_rls.sql
    - git diff --name-only --diff-filter=A origin/main -- db/migrations | xargs -r squawk
    - ../../scripts/ci/check_migration_phase --labels "$CI_MERGE_REQUEST_LABELS"

e2e:smoke:                           # services = image MR (build tag dev)
  stage: verify
  image: mcr.microsoft.com/playwright:<versi>-noble@sha256:<digest>
  needs: [build:mr-images]
  rules: [{ <<: *if-mr, changes: { paths: [apps/**/*, packages/**/*, contracts/**/*], compare_to: refs/heads/main } }]
  parallel: 2
  services:
    - { name: $POSTGIS_IMAGE, alias: postgres, variables: { POSTGRES_USER: msp_owner, POSTGRES_PASSWORD: ci, POSTGRES_DB: msp } }
    - { name: "chrislusf/seaweedfs@sha256:<digest>", alias: s3, command: ["server", "-s3"] }
    - { name: "$CI_REGISTRY_IMAGE/mr/msp-backend:$CI_COMMIT_SHA", alias: api, command: ["api", "--dev-bootstrap=e2e"] }
    - { name: "$CI_REGISTRY_IMAGE/mr/msp-backend:$CI_COMMIT_SHA", alias: worker, command: ["worker"] }
    - { name: "$CI_REGISTRY_IMAGE/mr/msp-web:$CI_COMMIT_SHA", alias: web }
    - { name: "$CI_REGISTRY/msp/webhook-sink@sha256:<digest>", alias: webhook-sink }
  variables: { MSP_ENV: ci, MSP_S3_ENDPOINT_URL: "http://s3:8333" }   # + MSP_DATABASE_URL_* seperti backend:test
  script:
    - corepack enable && pnpm install --frozen-lockfile
    - scripts/ci/wait_for http://api:8080/readyz http://web:3000/healthz
    - pnpm --filter web exec playwright test --grep @smoke --shard=$CI_NODE_INDEX/$CI_NODE_TOTAL   # QG-16, QG-17

# ---------------- main ----------------
build:backend:                       # build:web identik → WEB_IMAGE
  stage: build
  image: moby/buildkit:rootless@sha256:<digest>
  tags: [ci-build]
  needs: []
  rules: [*if-main]
  variables: { BUILDKITD_FLAGS: --oci-worker-no-process-sandbox }
  script:
    - scripts/ci/buildkit_build apps/api/Dockerfile "$CI_REGISTRY_IMAGE/msp-backend:sha-$CI_COMMIT_SHA" --build-arg GIT_SHA=$CI_COMMIT_SHA --provenance --metadata meta.json
    - echo "BACKEND_IMAGE=$CI_REGISTRY_IMAGE/msp-backend@$(scripts/ci/digest_from_meta meta.json)" > build.env
  artifacts: { reports: { dotenv: build.env } }

scan-and-sign:
  stage: scan
  needs: [build:backend, build:web]
  rules: [*if-main]
  environment: { name: signing, action: prepare }               # akses SOPS_AGE_KEY_SIGNING
  variables: { TRIVY_USERNAME: $CI_REGISTRY_USER, TRIVY_PASSWORD: $CI_REGISTRY_PASSWORD }
  script:
    - trivy image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 "$BACKEND_IMAGE"   # QG-27
    - trivy image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 "$WEB_IMAGE"
    - scripts/ci/sbom_and_sign "$BACKEND_IMAGE" "$WEB_IMAGE"    # syft → cosign sign + attest (--tlog-upload=false)

deploy:staging:
  stage: deploy-staging
  tags: [deploy]
  needs: [build:backend, build:web, scan-and-sign]
  rules: [*if-main]
  resource_group: staging
  interruptible: false
  environment: { name: staging, url: "https://app.staging.<domain>" }
  script:
    - eval "$(ssh-agent -s)" && ssh-add "$DEPLOY_SSH_KEY" && export SOPS_AGE_KEY_FILE="$SOPS_AGE_KEY"
    - ansible-playbook -i infra/ansible/inventories/staging infra/ansible/playbooks/deploy.yml
      -e backend_image="$BACKEND_IMAGE" -e web_image="$WEB_IMAGE" -e msp_version="sha-$CI_COMMIT_SHORT_SHA"
      -e '{"traffic_steps": [{"weight": 100, "watch_minutes": 5}]}'

# … verify:staging (matrix smoke/e2e/schemathesis/tenant-isolation/traceability) → mark:staging-verified

release:cut:
  stage: release
  needs: [mark:staging-verified]
  rules: [{ <<: *if-main, when: manual }]
  environment: { name: release, action: prepare }               # akses RELEASE_TOKEN
  script:
    - cd tools/release && pnpm install --frozen-lockfile && GITLAB_TOKEN="$RELEASE_TOKEN" pnpm exec semantic-release

# ---------------- tag vX.Y.Z ----------------
release:readiness:
  stage: release
  needs: []
  rules: [*if-rel]
  script:
    - scripts/release/await_status "$CI_COMMIT_SHA" staging-verified 45m                    # QG-29
    - scripts/release/resolve_digests "$CI_COMMIT_SHA" > release.env                        # BACKEND_IMAGE, WEB_IMAGE, BUMP
    - scripts/release/check_readiness release.env "$CI_COMMIT_TAG"     # QG-19/25/26/27/30/31/32/35 sesuai BUMP
  artifacts: { reports: { dotenv: release.env } }

deploy:production:
  stage: deploy-production
  tags: [deploy]
  needs: [release:readiness]
  rules: [{ <<: *if-rel, when: manual, allow_failure: false }]  # tag terlindungi → hanya Maintainer
  resource_group: production
  interruptible: false
  timeout: 2h
  environment: { name: production, url: "https://app.<domain>" }
  script:
    - scripts/release/four_eyes
    - eval "$(ssh-agent -s)" && ssh-add "$DEPLOY_SSH_KEY" && export SOPS_AGE_KEY_FILE="$SOPS_AGE_KEY"
    - ansible-playbook -i infra/ansible/inventories/production infra/ansible/playbooks/deploy.yml
      -e backend_image="$BACKEND_IMAGE" -e web_image="$WEB_IMAGE" -e msp_version="$CI_COMMIT_TAG"
      -e '{"traffic_steps": [{"weight": 10, "watch_minutes": 10}, {"weight": 100, "watch_minutes": 20}]}'   # QG-33
```

### 17.2 Playbook deploy (cuplikan)

```yaml
# infra/ansible/playbooks/deploy.yml (ILUSTRATIF)
- name: Deploy MSP blue/green
  hosts: app
  serial: 1
  vars: { msp_root: /opt/msp, secrets_root: /run/msp }        # secrets_root = tmpfs (ADR-22)
  pre_tasks:
    - name: Verifikasi tanda tangan & SBOM (QG-28)
      ansible.builtin.command: "{{ repo_root }}/scripts/deploy/verify_image {{ item }}"
      loop: ["{{ backend_image }}", "{{ web_image }}"]
      delegate_to: localhost
      run_once: true
      changed_when: false
    - ansible.builtin.slurp: { src: "{{ msp_root }}/active_color" }
      register: active
    - ansible.builtin.set_fact:
        color_old: "{{ active.content | b64decode | trim }}"
        color_new: "{{ 'green' if (active.content | b64decode | trim) == 'blue' else 'blue' }}"
  tasks:
    - block:
        - name: Secret SOPS → tmpfs (file 0400)
          ansible.builtin.copy: { content: "{{ item.value }}", dest: "{{ secrets_root }}/{{ color_new }}/{{ item.key }}", mode: "0400" }
          loop: "{{ lookup('community.sops.sops', inventory_dir ~ '/secrets/app.sops.yaml') | from_yaml | dict2items }}"
          no_log: true
        - name: Render compose warna baru (image by digest)
          ansible.builtin.template: { src: compose.app.yml.j2, dest: "{{ msp_root }}/{{ color_new }}/compose.yml" }
        - community.docker.docker_compose_v2_pull: { project_src: "{{ msp_root }}/{{ color_new }}" }
        - name: msp migrate up (one-off, expand-only, advisory lock)
          run_once: true
          community.docker.docker_container:
            name: "msp-migrate-{{ msp_version }}"
            image: "{{ backend_image }}"
            command: ["migrate", "up"]
            env: { MSP_DATABASE_URL_DIRECT_FILE: /run/secrets/database_url_migrate }
            mounts: [{ type: bind, read_only: true, source: "{{ secrets_root }}/{{ color_new }}/database_url_migrate", target: /run/secrets/database_url_migrate }]
            networks: [{ name: msp-backend }]
            detach: false                          # tunggu selesai; exit code ≠ 0 → rescue
            cleanup: true
        - name: Start warna baru (worker, api, web) dan tunggu healthy
          community.docker.docker_compose_v2: { project_src: "{{ msp_root }}/{{ color_new }}", project_name: "msp-{{ color_new }}", state: present, wait: true, wait_timeout: 300 }
        - name: SIGTERM worker warna lama (River soft stop, stop_grace_period per pool)
          ansible.builtin.command: "docker compose -p msp-{{ color_old }} stop {{ worker_services | join(' ') }}"
        - name: Canary → promote, dipantau Prometheus (gagal = rescue)
          ansible.builtin.include_tasks: tasks/shift_traffic.yml       # render dynamic/msp.yml + watch_canary
          loop: "{{ traffic_steps }}"
          loop_control: { loop_var: step }
        - ansible.builtin.include_tasks: tasks/finalize_color.yml      # render tanpa warna lama, stop api/web lama
        - ansible.builtin.copy: { content: "{{ color_new }}\n", dest: "{{ msp_root }}/active_color" }
      rescue:
        - ansible.builtin.template: { src: traefik-msp.yml.j2, dest: /opt/traefik/dynamic/msp.yml }
          vars: { traffic: [{ color: "{{ color_old }}", weight: 100 }] }
        - ansible.builtin.command: "docker compose -p msp-{{ color_old }} start {{ worker_services | join(' ') }}"
        - community.docker.docker_compose_v2: { project_src: "{{ msp_root }}/{{ color_new }}", state: absent }
        - ansible.builtin.fail: { msg: "Deploy {{ msp_version }} gagal → rollback otomatis ke {{ color_old }}" }
```

### 17.3 Traefik blue/green & label Compose

```yaml
# /opt/traefik/dynamic/msp.yml — dirender Ansible (file provider, watch: true)
http:
  routers:
    msp-api: { rule: "Host(`app.<domain>`) && PathPrefix(`/api`)", entryPoints: [websecure], service: msp-api,
               middlewares: [api-strip-prefix, security-headers], tls: { certResolver: letsencrypt } }   # ADR-25
    msp-web: { rule: "Host(`app.<domain>`)", entryPoints: [websecure], service: msp-web,
               middlewares: [security-headers], tls: { certResolver: letsencrypt } }
  services:
    msp-api:
      weighted:
        healthCheck: {}
        services:                                  # hanya warna yang sedang hidup boleh dirujuk
          - { name: api-blue@docker,  weight: 90 }
          - { name: api-green@docker, weight: 10 }
    msp-web: { weighted: { healthCheck: {}, services: [{ name: web-blue@docker, weight: 90 }, { name: web-green@docker, weight: 10 }] } }
```

```yaml
# compose.app.yml.j2 (cuplikan) — satu project per warna: msp-blue / msp-green
x-backend: &backend
  image: "{{ backend_image }}"                     # digest yang sama untuk api, worker, migrate
  environment: &backend-env
    MSP_ENV: "{{ msp_env }}"
    MSP_VERSION: "{{ msp_version }}"
    OTEL_RESOURCE_ATTRIBUTES: "msp.color={{ color_new }}"
    MSP_DATABASE_URL_APP_FILE: /run/secrets/database_url_app        # via PgBouncer db-01:6432
  secrets: [database_url_app, redis_url, webhook_secret_master_keys]
  healthcheck: { test: ["CMD", "/msp", "tool", "healthcheck"], interval: 10s, retries: 3 }
x-worker: &worker                                  # client River: koneksi langsung, pool kecil khusus (04 §17.4)
  <<: *backend
  environment:
    <<: *backend-env
    MSP_DATABASE_URL_DIRECT_FILE: /run/secrets/database_url_direct  # db-01:5432 (LISTEN/NOTIFY, leader election)
  secrets: [database_url_app, database_url_direct, redis_url, webhook_secret_master_keys]
services:
  api:
    <<: *backend
    command: ["api"]
    deploy: { replicas: 2 }
    networks: [edge, backend]
    labels:
      - traefik.enable=true
      - traefik.http.services.api-{{ color_new }}.loadbalancer.server.port=8080
      - traefik.http.services.api-{{ color_new }}.loadbalancer.healthcheck.path=/readyz
      - traefik.http.services.api-{{ color_new }}.loadbalancer.healthcheck.interval=5s
  worker-pipeline:
    <<: *worker
    command: ["worker", "--queues=pipeline"]
    stop_grace_period: 15m                         # River soft stop; job wajib resumable
    networks: [backend]
secrets:
  database_url_app: { file: "/run/msp/{{ color_new }}/database_url_app" }
  database_url_direct: { file: "/run/msp/{{ color_new }}/database_url_direct" }
  # … redis_url, webhook_secret_master_keys
```

---

## 18. Checklist bootstrap

Urutan menyiapkan CI/CD dari nol. Setiap langkah menghasilkan artefak di repo, kecuali yang ditandai *(manual)*.

1. *(manual)* Kontrak dedicated server di DC Indonesia (Q-CI-04): `ops-01`, `ci-01`, `stg-01`, `app-01`, `db-01`, `db-02`, `stor-01`; VLAN privat; IP publik untuk `app-01` dan `ops-01`; penyimpanan off-site S3-compatible di lokasi lain di Indonesia. Pasang OS dengan LUKS.
2. *(manual)* Buat kunci age per environment + kunci Maintainer (salinan offline), kunci SSH deploy, dan key pair cosign (`cosign generate-key-pair`). Commit kunci publik ke `infra/keys/`.
3. Commit `infra/ansible` (inventory, role `base`, `users`, `firewall`, `wireguard`, `luks_unlock`) + `.sops.yaml`. Jalankan `site.yml` pertama dari laptop Maintainer via IP publik **sekali saja**, lalu tutup SSH publik. Selanjutnya semua akses lewat WireGuard.
4. Role `gitlab` di `ops-01` (Omnibus, ACME DNS-01 atau CA internal, registry, sign-up nonaktif, 2FA wajib, backup harian) + role `monitoring_stack`.
5. Buat grup `msp`, proyek `MesinProspek` (private), subgroup/tim untuk CODEOWNERS. Pindahkan repo (Q-CI-01).
6. **Verifikasi tier** fitur CE yang dipakai (§1.3).
7. Role `gitlab_runner`: `ci-01` (`ci-docker`, `ci-build`, cache S3 ke SeaweedFS lokal, `allowed_images`) dan `ops-01` (`deploy`, *protected*).
8. `scripts/bootstrap/gitlab_settings`: protected branch/tag (§2.5), fast-forward + squash, template squash, merge checks, cleanup policy registry, server hook gitleaks.
9. CI/CD variables (§12.2) dengan scope environment. Bot user: rilis (Maintainer), Renovate, `mr:gate` (Reporter). Token berkedaluwarsa.
10. Fondasi repo: `.gitlab/CODEOWNERS`, MR template, `.editorconfig`, `.env.example`, `commitlint.config.js`, `tools/release/` (semantic-release), `renovate.json`, `.golangci.yml` (termasuk aturan depguard batas modul), `apps/api/sqlc.yaml`, `.spectral.yaml`, `.squawk.toml`, `.semgrep/`, `.markdownlint-cli2.yaml`, `lychee.toml`, `ci/images/tools/Dockerfile`, directive `tool` di `apps/api/go.mod`. **Spike toolchain OpenAPI (Sprint 0, 04 TQ-18):** spec contoh OpenAPI 3.1 (problem + `errors[]`, cursor page, `oneOf`, nullable) dijalankan lewat oapi-codegen (≥ v2.8.0, Go ≥ 1.25) + openapi-typescript + Spectral + oasdiff + Schemathesis; konstruksi yang lolos menjadi rule Spectral kustom (QG-12). Bila gagal: spec ditulis OpenAPI 3.0.3 (atau di-*down-convert* lewat overlay) dan job `contracts` disesuaikan.
11. Pipeline pertama: build image `ci-tools`, lalu `.gitlab-ci.yml` MR (lint, test, security, `mr:gate`). Uji di MR percobaan, termasuk pemblokiran merge oleh `mr:gate`.
12. Provision `stg-01` (role `docker`, `traefik`, `msp_app`, Postgres/Redis/SeaweedFS staging) → pipeline `main`: build, scan, sign, `deploy:staging`, verifikasi, `staging-verified`.
13. Provision production: `db-01`/`db-02` (role `postgres`, replikasi, PgBouncer, `pgbackrest` ke `stor-01` + off-site), `stor-01` (SeaweedFS + `filer.sync` off-site), `app-01` (Traefik, `msp_app`), `monitoring_agent` di semua host.
14. `release:cut` + pipeline tag `v*`. Rilis uji `v0.1.0` ke production (health saja), termasuk uji rollback dan uji rescue canary (paksa gagal).
15. `ci/infra.yml` (check/diff, apply staging, tag `infra-*`) + schedule `drift`, `backup-check`.
16. Schedule `nightly` bertahap: regresi penuh → fuzz panjang → ZAP → k6 → data quality → rescan → Lighthouse. Lalu `renovate`, `weekly`, dan child pipeline `rc`.
17. Drill restore pertama secara manual di `dr-01`, baru kemudian dijadwalkan bulanan.
18. Flag registry + kill switch + laporan flag hygiene. Unit `msp-rehydrate` + trigger token.
19. Observability: anotasi deploy, release GlitchTip, dashboard SLO, DORA, CI health, probe eksternal + watchdog.
20. Runbook (deploy, rollback, failover DB, rehydrate, incident + hook PDP, drill) dan latihan on-call sebelum closed beta.

---

## 19. Pertanyaan terbuka

ID `Q-CI-nn` bersifat lokal di dokumen ini. Setelah diputuskan, keputusan dipindahkan ke register 00 §7.

| ID | Pertanyaan | Default sementara | Dampak |
|---|---|---|---|
| Q-CI-01 | Apakah source code **pindah penuh** dari GitHub ke GitLab, atau GitHub tetap ada sebagai mirror? | **GitLab primary.** Mirror push ke GitHub opsional (read-only, tanpa CI) | §2, §15, README |
| Q-CI-02 | Tetap di GitLab CE, atau beli Premium (approval rules, Code Owners, merge trains, protected environments)? | CE + mitigasi §1.3. Ditinjau saat tim ≥ 6 engineer | §2.3, §2.6, §6.2 |
| Q-CI-03 | Alternatif CI yang lebih ringan (Forgejo + Actions, Woodpecker) bila beban operasional GitLab terlalu besar? | GitLab CE | §1.4 |
| Q-CI-04 | Provider DC dan lokasi off-site (keduanya di Indonesia, kota/provider berbeda). SLA penggantian hardware dan ketersediaan `dr-01` | Jakarta + lokasi kedua di Indonesia | RTO kehilangan DC (§11.6), D-14 |
| Q-CI-05 | Kapan `app-02` + VIP (keepalived/VRRP atau LB provider) ditambahkan? Kapan failover DB otomatis (Patroni)? | `app-02` aktif sebelum M-3 (early paid, PRD R-08); Patroni saat ada node DB ketiga | NFR-AVAIL-01, §6.3 |
| Q-CI-06 | Kanal alert on-call dan lokasi probe eksternal | Chat + email; probe di VPS lokasi lain | §16.1 |
| Q-CI-07 | Siapa Maintainer (berhak deploy production) dan rotasi on-call di luar jam kerja? | Tech lead + 1 engineer senior | §2.5, §6.2, §14.5 |
| Q-CI-08 | Nama domain dan penyedia DNS (DNS-01 untuk GitLab internal, Let's Encrypt untuk `app.<domain>`) | `<domain>` placeholder | §3.1, §17.3 |
| Q-CI-09 | Lisensi dataset: apakah subset boleh dipakai di staging dan eval AI (Q-05)? | Hanya sintetis sampai dikonfirmasi | §10.2, QG-26 penuh |
| Q-CI-10 | Apakah SOPS + GitLab variables cukup untuk NFR-SEC-04 ("secret hanya di secret manager"), atau perlu OpenBao/Vault? | Cukup untuk MVP; OpenBao dievaluasi di V1 (ADR-22) | §12 |
| Q-CI-11 | ~~Aksi `msp tool` tambahan (`healthcheck`, `snapshot-publish`, `score-model-activate`) di luar daftar 04 §3.1 disetujui?~~ | **Ditutup**: sudah tercantum di 04 §3.1 | §5.1, §10, D-21.1 |
| Q-CI-13 | Object storage final: SeaweedFS atau Garage? Fitur object lock, versioning, dan SSE mana yang benar-benar didukung versi yang dipasang? | SeaweedFS; fitur yang tidak didukung diganti salinan append-only off-site + LUKS | §10.1, §11.6, ADR-22 |
| Q-CI-12 | Threshold QG-34 dan sinyal canary (job discarded, latency) awal | Nilai default §6.3 & §10.2 | Kalibrasi setelah Q-01/Q-02 + 4 minggu data beta |
