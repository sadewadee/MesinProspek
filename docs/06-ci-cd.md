# 06 — CI/CD

## MesinProspek — Branching, Pipeline, Quality Gate, Environment, Rilis

| Field | Value |
|---|---|
| Dokumen | CI/CD |
| Versi | 0.1 (Draft) |
| Tanggal | 2026-10-10 |
| Basis | `README.md` (konvensi & struktur monorepo), `00-analisis-dokumen-awal.md` (D-11, D-12, D-14), `01-prd.md` (§10 NFR, §12 release gate & DoD), `04-technical-design.md` (ADR-18, ADR-21, ADR-22), MVP Pack v1.2 §11, §14, §15 |
| Terkait | `04-technical-design.md` (arsitektur, data model, deployment diagram), `05-test-strategy.md` (isi dan level test suite) |
| Keputusan baru | **D-18** platform CI/CD = GitLab CE self-hosted · **D-19** hosting & runtime = dedicated server di DC Indonesia. Keduanya **Proposed** (§1.2). |
| Menggantikan | Baris README *Repo*, *CI/CD*, dan *Hosting* (GitHub, GitHub Actions, AWS). README perlu diperbarui. |
| Audience | Engineering, DevOps/Platform, QA, Tech lead |
| Menutup | G-17 (strategi CI/CD, environment, migrasi DB) |

> Konvensi: `QG-nn` = quality gate (§8). Semua pilihan berstatus **Proposed**. Contoh YAML, Ansible, dan konfigurasi bersifat **ilustratif**, bukan file final. Jam ditulis dalam WIB (UTC+7); schedule GitLab diset ke zona `Asia/Jakarta`. Path modul backend mengikuti 04 §3 (`apps/api/msp/<modul>/`); sub-path seperti `identity/internal/normalize/` adalah asumsi kerja.

---

## 1. Prinsip & keputusan

| # | Prinsip | Konsekuensi konkret |
|---|---|---|
| P-1 | **Trunk-based development** | Satu branch jangka panjang: `main`. Branch fitur berumur ≤ 2 hari (maks 5). Pekerjaan yang belum selesai masuk `main` di balik feature flag (§13). |
| P-2 | **Build once, deploy many** | Image dibangun **sekali** per commit `main` dan dipromosikan ke staging lalu production dengan **digest yang sama** (`sha256:…`). Versi dan konfigurasi environment disuntikkan saat runtime (§12). |
| P-3 | **Everything as code** | `.gitlab-ci.yml`, `infra/ansible/` (server, OS, layanan), secret terenkripsi SOPS, kebijakan repo (setelan proyek GitLab via script API), skema DB (Alembic), kontrak (`contracts/`), design tokens, dan konfigurasi versi skor/geo/prompt. Perubahan manual di server dianggap drift (§11.4). |
| P-4 | **Fast feedback** | Pipeline MR **≤ 10 menit p50** (p90 ≤ 15) lewat `rules:changes`, DAG `needs:`, cache uv/pnpm/Turborepo di cache S3 runner, dan BuildKit registry cache. |
| P-5 | **Shift-left security** | Secret scanning (CI + server hook), SAST, audit dependency, license policy, dan scan IaC berjalan di MR. Image yang gagal scan tidak ditandatangani, sehingga tidak bisa di-deploy (§5, §15). |
| P-6 | **Rilis data & model reproducible** | `ds-YYYY.MM.n`, `priority-X.Y.Z`, `geo-X.Y.Z`, `brief-X.Y.Z` dirilis lewat pipeline dengan gate dan approval, terpisah dari rilis kode. Versi terbit immutable (NFR-DATA-01, FR-INTL-009). |
| P-7 | **Data tetap di zonanya** | Seluruh infrastruktur (kode, CI, registry, runtime, backup) berada di Indonesia (D-14). Runner MR (`ci-01`) hanya memproses kode dan **data sintetis**, dan tidak punya rute ke zona production/staging. Dataset berlisensi dan data production hanya diproses di host environment-nya. |
| P-8 | **Rollback murah, forward-fix untuk data** | Rollback aplikasi = kembalikan traffic ke warna sebelumnya atau deploy ulang digest sebelumnya (≤ 5–15 menit). Skema DB tidak di-rollback: migrasi selalu kompatibel dengan versi N-1 (§9). |
| P-9 | **Gate production = protected tag** | Setiap operasi yang mengubah production (rilis aplikasi, apply infra, publish dataset, aktivasi model) berjalan di pipeline **protected tag** yang hanya bisa dibuat Maintainer. Job manual di pipeline tersebut hanya bisa dijalankan Maintainer (§2.5). |

### 1.1 Alur besar

```
 feat/* fix/* chore/* docs/*
        │  merge request
        ▼
 ┌──────────────────────────────────┐   review + approval   ┌───────────────────────────────┐
 │ MR pipeline (rules:changes, DAG) │ ────────────────────► │ mr:gate (manual, blocking):   │
 │ lint·type·test·migrasi·kontrak·  │                       │ approval owner, label, thread │
 │ tokens·a11y·E2E smoke·security   │                       └───────────────┬───────────────┘
 └──────────────────────────────────┘     rebase + squash + fast-forward    │
                                                                            ▼
                                                                          main
 ┌──────────────────────────────────────────────────────────────────────────────────────────┐
 │ main pipeline: build msp-api & msp-web (BuildKit) → Trivy → SBOM → cosign sign           │
 │   → deploy staging (Ansible, blue/green Traefik) → verifikasi → status `staging-verified`│
 │   → release:cut (manual, semantic-release) → tag vX.Y.Z + GitLab Release                 │
 └──────────────────────────────────────────────────────────────────────────────────────────┘
                                                                            │ protected tag
 ┌──────────────────────────────────────────────────────────────────────────▼───────────────┐
 │ tag pipeline vX.Y.Z: resolve digest → readiness gate → deploy:production (manual,        │
 │   Maintainer) → verify signature → migrate → canary 10% → 100% → bake → auto-rollback    │
 └──────────────────────────────────────────────────────────────────────────────────────────┘
 Paralel: scheduled pipelines (nightly, drift, Renovate, drill), tag pipeline `infra-*`, `ds-*`,
          `priority-*`/`geo-*`/`brief-*` (§10, §11), job rollback (§14)
```

### 1.2 Keputusan D-18 dan D-19 (Proposed)

| ID | Sub-keputusan | Alasan singkat |
|---|---|---|
| **D-18** | **Platform CI/CD = GitLab CE self-hosted** | Tidak memakai GitHub (arahan founder). Satu paket: code host, MR, CI, registry, scheduled pipeline. |
| D-18.1 | GitLab CE (Omnibus) di `ops-01` adalah **repo utama**. Mirror ke GitHub opsional (Q-CI-01). Akses UI/git hanya lewat WireGuard. | Residensi kode, kontrol penuh |
| D-18.2 | GitLab CI: `.gitlab-ci.yml`, merge request pipelines, `rules:changes` (path filter), `needs:` (DAG), `resource_group`, scheduled pipelines, `environment` untuk pencatatan deploy | Fitur inti tersedia di CE |
| D-18.3 | Runner: GitLab Runner **Docker executor** di `ci-01` (`ci-docker` non-privileged untuk MR, `ci-build` BuildKit rootless untuk image). Runner **`deploy`** berstatus *protected* di `ops-01`, hanya mengambil job dari ref terlindungi. | Kode MR yang belum direview tidak pernah berjalan di host yang punya rute ke production |
| D-18.4 | **GitLab Container Registry** (`ops-01`). Deploy selalu by digest. Cleanup policy: image MR dihapus 7 hari, `sha-*` disimpan 90 hari, `v*` permanen. | Satu sistem dengan CI, di Indonesia |
| D-18.5 | Merge: **squash + fast-forward** (rebase wajib), *Pipelines must succeed*, *All threads resolved*. Approval ditegakkan job manual blocking **`mr:gate`** (CE tidak punya required approvals/Code Owners). | Pengganti merge train (Premium) dan approval rules (Premium) |
| D-18.6 | Gate production = **protected tags** (`v*`, `infra-*`, `ds-*`, `priority-*`, `geo-*`, `brief-*`) yang hanya bisa dibuat Maintainer + **job manual** di pipeline tag. | Pengganti protected environments (Premium) |
| D-18.7 | Rilis: **semantic-release** + `@semantic-release/gitlab`, dipicu manual (`release:cut`). Tanpa commit balik ke repo, sehingga commit yang di-tag = commit yang sudah dibangun dan diverifikasi. | release-please berorientasi GitHub. changesets berorientasi paket npm. |
| D-18.8 | Security scanning: **Semgrep CE** + **Bandit** (SAST), **gitleaks** (CI + pre-receive server hook), **Trivy** (fs, image, config), **OSV-Scanner** + **pip-audit** + **pnpm audit** (dependency), license allowlist (OSV-Scanner), **OWASP ZAP** baseline di staging | Pengganti CodeQL, Dependabot, dan dependency review GitHub |
| D-18.9 | Supply chain: **syft** (SBOM SPDX), **cosign** dengan **key pair** (private key terenkripsi SOPS; tanpa keyless OIDC; tanpa upload ke transparency log publik), provenance BuildKit, image **`ci-tools`** buatan sendiri dengan versi tool di-pin | Tidak ada OIDC GitHub/Fulcio. Tidak membocorkan metadata ke log publik. |
| D-18.10 | **Renovate self-hosted** sebagai scheduled pipeline (bot user + project access token) | Tidak ada Dependabot |
| **D-19** | **Hosting & runtime = dedicated server di DC Indonesia** | Residensi (D-14), biaya, sejalan dengan ADR-18 di 04 |
| D-19.1 | Topologi: `app-01` (kelak `app-02`), `db-01` (primary), `db-02` (hot standby + read replica), `stor-01` (MinIO), `ops-01` (GitLab, registry, monitoring, Tang), `ci-01` (runner), `stg-01` (staging all-in-one), `dr-01` (target drill/DR; boleh sewa per jam) | Selaras 04 §2.2 |
| D-19.2 | Runtime app: **Docker Engine + Docker Compose**. **Traefik v3** (Let's Encrypt) sebagai reverse proxy. Deploy **blue/green** dengan *weighted service* Traefik (canary 10% → 100%). | Sederhana untuk 1–2 server. Swarm/k3s hanya opsi masa depan (D-19.10). |
| D-19.3 | **PostgreSQL 16 + PostGIS** bare metal (paket PGDG) + PgBouncer. Streaming replication ke `db-02` (hot standby). Failover manual via playbook. | Kontrol versi & tuning. Patroni saat ada node ketiga. |
| D-19.4 | **Redis** (AOF `everysec`) di `app-01` (04 §2.2). **MinIO** single-node multi-drive (erasure coding) di `stor-01`. | Selaras 04 |
| D-19.5 | Provisioning & konfigurasi: **Ansible** (hardening OS, user, nftables, **WireGuard** untuk akses admin/SSH, fail2ban, unattended security upgrades, Docker, layanan). Disk **LUKS** dengan unlock otomatis Clevis + Tang (fallback dropbear-initramfs). | Satu alat untuk provisioning, konfigurasi, dan deploy |
| D-19.6 | Secret: **SOPS + age** di repo (`infra/`), didekripsi Ansible saat deploy ke **tmpfs** (ADR-22). Kredensial deploy disimpan di **GitLab CI/CD variables masked + protected**, di-scope per environment. | Tanpa cloud secret manager |
| D-19.7 | Backup/DR: **pgBackRest** (full mingguan, diff harian, WAL archive kontinu `archive_timeout = 60s`) ke repo lokal (`stor-01`) **dan** off-site S3-compatible di lokasi berbeda (tetap di Indonesia). Replikasi bucket MinIO off-site. Backup GitLab harian. Drill restore bulanan sebagai scheduled pipeline. | NFR-AVAIL-02 (RPO ≤ 15 menit, RTO ≤ 4 jam) |
| D-19.8 | Observability self-hosted di `ops-01`: Prometheus + Alertmanager + Grafana + Loki + Tempo (via OpenTelemetry Collector), **GlitchTip** (kompatibel Sentry SDK), blackbox exporter + **probe eksternal** di luar DC | ADR-21. Probe eksternal mendeteksi DC/jaringan down. |
| D-19.9 | Zona jaringan terpisah: `edge/prod`, `data/prod`, `ops`, `ci`, `stg` (nftables, VLAN privat). Port publik hanya 80/443 (`app-01`) dan UDP WireGuard (`ops-01`). | Blast radius kecil |
| D-19.10 | Docker Swarm/k3s **bukan** bagian MVP. Ditinjau bila > 2 app server atau dibutuhkan autoscaling. | Hindari kompleksitas dini |

### 1.3 Batasan GitLab CE & mitigasi

| Fitur yang tidak ada di CE (tier berbayar) | Dampak | Mitigasi di dokumen ini |
|---|---|---|
| Required MR approvals, approval rules, Code Owners approval | Approval tidak ditegakkan platform | Job manual blocking `mr:gate` yang membaca approval via API + `.gitlab/CODEOWNERS` (§2.3) |
| Merge trains, merged results pipelines | Tidak ada merge queue | Fast-forward merge (rebase wajib), sehingga pipeline terakhir selalu berjalan di atas `main` terbaru (§2.6) |
| Protected environments, deployment approvals | Job deploy tidak bisa dibatasi per environment | Protected tags + job manual (hanya Maintainer) + `resource_group` + cek *four-eyes* opsional (§6.2) |
| Push rules (blok secret, regex pesan commit) | — | **Server hook** `pre-receive` gitleaks (tersedia di self-managed) + validasi judul MR (§15.1) |
| Security dashboard, MR security widget | Temuan tidak tampil terpusat | Job gagal + laporan sebagai artifact + komentar MR oleh script |
| Audit events lengkap, DORA dashboard | Jejak & metrik terbatas | Log GitLab dikirim ke Loki. Deployment tercatat via `environment`. DORA dihitung script → Grafana (§16) |

> Sebelum bootstrap, verifikasi ulang tier setiap fitur yang dipakai (mis. *environment-scoped variables*, *protected runners*, *server hooks*, *cleanup policies*) terhadap versi GitLab CE yang dipasang. Lihat §18 langkah 6.

### 1.4 Alternatif platform CI (pemetaan singkat)

| Kebutuhan | **GitLab CE** (dipilih) | Forgejo/Gitea + Actions (act_runner) | Woodpecker CI (+ Forgejo/Gitea) | Jenkins |
|---|---|---|---|---|
| Definisi pipeline | `.gitlab-ci.yml` | `.forgejo/workflows/*.yml` (mirip GitHub Actions) | `.woodpecker/*.yml` | `Jenkinsfile` (Groovy) |
| Path filter / DAG | `rules:changes` / `needs` | `on.paths` (per workflow) / `needs` | `when.path` / `depends_on` | `changeset` / `parallel` |
| Registry | Built-in | Package registry OCI built-in | Dari forge | Eksternal (Harbor/Distribution) |
| Gate production | Protected tag + job manual | Tag protection + `workflow_dispatch` (tanpa reviewer environment) | Event `deployment`/promote | Step `input` + matrix authorization |
| Secret | Variable masked/protected (+ scope env) | Secret repo/org | Secret + filter event | Credentials plugin |
| Catatan | Paling lengkap dalam satu paket. Berat (RAM ≥ 8 GB) | Ringan. Kompatibilitas Actions tidak 100% | Ringan, sederhana. Fitur gate minim | Fleksibel. Beban maintenance plugin tinggi |

Migrasi ke alternatif cukup murah karena logika pipeline berada di script repo (`scripts/ci/`, `scripts/release/`) dan playbook Ansible. YAML CI hanya mengorkestrasi.

### 1.5 Tooling tambahan (di luar README, Proposed)

| Area | Pilihan | Alasan |
|---|---|---|
| Rilis & changelog | semantic-release (`commit-analyzer`, `release-notes-generator`, `@semantic-release/gitlab`) | D-18.7 |
| Validasi judul MR | commitlint (`@commitlint/config-conventional`) | Squash commit = judul MR |
| Lint OpenAPI | Spectral (+ `spectral-owasp-ruleset`) | Ruleset kustom YAML (mis. wajib `Idempotency-Key` pada operasi berbayar) |
| Breaking change API | oasdiff | OpenAPI 3.1, level ERR/WARN |
| Lint SQL migrasi | squawk | Operasi berbahaya PostgreSQL (lock, index non-concurrent, rename/drop) |
| Validasi JSON Schema | check-jsonschema | Draft 2020-12, CLI Python (`uvx`) |
| Client TS | openapi-typescript (+ `openapi-fetch`) | Final di 04 (ADR-08) |
| Batas modul | import-linter | ADR-01 di 04 |
| Docs lint | markdownlint-cli2 + lychee | Cepat, link internal offline |
| Ansible | ansible-lint, yamllint, Molecule (role kritis) | QG-21 |
| Build image | BuildKit rootless (`buildctl-daemonless.sh`) | Tanpa runner privileged |
| Paralelisasi Python | pytest-xdist | Target ≤ 10 menit |
| Metrik CI | gitlab-ci-pipelines-exporter → Prometheus | Durasi & flaky rate (§16.3) |
| Web performance | Lighthouse CI (nightly, staging) | Proxy lab NFR-PERF-03 |
| Feature flag | Flag store Postgres (`platform.feature_flag`, 04 §3.2) + registry di repo | Audit wajib (FR-OPS-007). Fitur feature flag GitLab tidak dipakai karena perubahan flag harus tercatat di audit produk. |

---

## 2. Branching & merge request workflow

### 2.1 Model branch

| Branch | Fungsi | Umur | Dibuat oleh |
|---|---|---|---|
| `main` | Trunk, selalu dapat dirilis | Permanen | — |
| `feat/<scope>-<deskripsi>` | Fitur baru, mis. `feat/srch-zero-result-guidance` | ≤ 2 hari (maks 5) | Engineer |
| `fix/<scope>-<deskripsi>` | Perbaikan bug | ≤ 2 hari | Engineer |
| `chore/<deskripsi>` | Maintenance, refactor, build, CI | ≤ 2 hari | Engineer |
| `docs/<deskripsi>` | Dokumentasi | ≤ 2 hari | Engineer/Design/Product |
| `renovate/*` | Update dependency | Sampai merge | Bot Renovate |
| `release/vX.Y` | **Hanya** hotfix saat `main` tidak bisa dirilis (§6.6) | Sampai hotfix rilis | Maintainer |

Tidak ada branch `develop`. Fitur besar dipecah menjadi beberapa MR kecil di balik flag `rel.*`. Source branch dihapus otomatis setelah merge.

### 2.2 Conventional Commits

Merge memakai **squash**, dengan template squash commit `%{title}` + baris `MR: %{reference}`. Akibatnya **judul MR = pesan commit** di `main`, dan divalidasi commitlint (QG-01).

```
<type>(<scope>)!: <ringkasan imperatif, huruf kecil>
type  : feat | fix | perf | refactor | test | docs | chore | ci | build | revert
scope : web | api | worker | platform | ingest | identity | intelligence | geo | query | action |
        ai | usage | ops | contracts | tokens | ui | infra | data | docs | deps
!     : breaking change (wajib label `api-breaking` bila menyangkut kontrak)
```

| Contoh judul | Efek semantic-release |
|---|---|
| `feat(intelligence): tambah faktor geo pada priority score` | minor |
| `fix(action): retry webhook memakai backoff eksponensial` | patch |
| `feat(contracts)!: ganti format cursor pagination /v1/prospects` | major |
| `chore(deps): update ruff` | tidak memicu rilis |

Changelog di GitLab Release menautkan setiap baris ke MR, dan deskripsi MR memuat ID requirement (`Refs: FR-…`).

### 2.3 Review, CODEOWNERS, dan `mr:gate`

File `.gitlab/CODEOWNERS` memakai format GitLab, sehingga langsung berlaku bila kelak pindah ke tier berbayar. Di CE, file ini dibaca oleh script `mr:gate` dan dipakai untuk menyarankan reviewer (komentar otomatis di MR).

```
# .gitlab/CODEOWNERS (ilustratif). Pola terakhir yang cocok menang.
*                                   @msp/backend @msp/web
/apps/web/                          @msp/web
/packages/ui/                       @msp/web @msp/design
/packages/design-tokens/            @msp/design @msp/web
/docs/02-design-system/             @msp/design
/apps/api/                          @msp/backend
/apps/api/msp/identity/             @msp/data-intel
/apps/api/msp/intelligence/         @msp/data-intel
/apps/api/msp/ai/                   @msp/data-intel
/apps/api/msp/usage/                @msp/backend
/apps/api/msp/platform/             @msp/backend @msp/security
/apps/api/migrations/               @msp/backend @msp/platform
/contracts/                         @msp/api-owners
/data/                              @msp/data-intel
/infra/                             @msp/platform
/.gitlab-ci.yml  /ci/  /scripts/ci/ @msp/platform
/docs/                              @msp/product
```

Job **`mr:gate`** (manual, `allow_failure: false`) menjadikan pipeline berstatus *blocked*. Dengan *Pipelines must succeed*, MR **tidak bisa di-merge** sebelum job ini dijalankan dan lulus. Script memeriksa:
1. ≥ 1 approval dari owner setiap area yang tersentuh. **2 approval** untuk path berisiko (tabel di bawah) saat tim ≥ 4 engineer.
2. Approver ≠ author, dan approval diberikan **setelah commit terakhir** (dicek dari system note MR, karena CE tidak me-reset approval saat ada push).
3. Semua thread resolved, MR bukan draft.
4. Label wajib ada bila relevan (`api-breaking` saat oasdiff ERR, `db-contract` saat migrasi contract).
5. Pengecualian: MR Renovate berlabel `automerge` (patch/minor devDependency, §15.4) tidak butuh approval manusia.

Reviewer (atau author) menjalankan `mr:gate` setelah approval diberikan. Token API yang dipakai bersifat read-only (`read_api`, role Reporter).

| Path berisiko (2 approval saat tim ≥ 4) | Alasan |
|---|---|
| `apps/api/migrations/**` dengan label `db-contract` | Operasi destruktif (§9) |
| `apps/api/msp/platform/**` (auth, tenancy, RLS) | Isolasi tenant (NFR-SEC-03) |
| `infra/**` | Server & secret |
| `.gitlab-ci.yml`, `ci/**`, `scripts/ci/**`, `scripts/release/**` | Supply chain (§15) |
| `contracts/**` dengan label `api-breaking` | Kontrak publik |

### 2.4 MR template

```markdown
<!-- .gitlab/merge_request_templates/Default.md (ilustratif) -->
## Ringkasan
<!-- Apa dan kenapa. Judul MR mengikuti Conventional Commits. -->

## Requirement
Refs: FR-___ / NFR-___ / BR-___      Test: TC-___ (05-test-strategy.md)

## Checklist
- [ ] Test ditambah/diubah sesuai level di 05
- [ ] Tidak ada data pribadi nyata di test, fixture, log, atau screenshot
- [ ] Migrasi DB: fase `expand`/`contract` dicatat, kompatibel dengan versi sebelumnya (§9)
- [ ] Tabel tenant baru memiliki RLS policy
- [ ] Kontrak: OpenAPI/JSON Schema diperbarui, client TS di-regenerate
- [ ] Breaking API? Label `api-breaking` + "Rencana versi API" di bawah
- [ ] Feature flag terdaftar di registry (nama, tipe, owner, kedaluwarsa) (§13)
- [ ] Skor/geo/prompt: versi dinaikkan, laporan regresi/eval terlampir (§10)
- [ ] UI: screenshot sebelum-sesudah (1280 px + 360 px), copy id-ID & en-US
- [ ] Event analytics & audit dipasang (bila relevan); runbook diperbarui (bila perlu)

## Catatan migrasi / rollout
## Rencana versi API (wajib bila `api-breaking`)
```

### 2.5 Protected branch, protected tag, merge check

Setelan dikelola sebagai kode oleh `scripts/bootstrap/gitlab_settings.py` (GitLab API) dan diperiksa ulang oleh job drift mingguan.

| Setelan | Nilai |
|---|---|
| `main`: allowed to push | **No one** (termasuk force push) |
| `main`: allowed to merge | Developers + Maintainers (approval ditegakkan `mr:gate`) |
| `release/v*`: push/merge | Maintainers |
| Merge method | **Fast-forward merge** + squash wajib. Template squash `%{title}` |
| Merge checks | Pipelines must succeed. *Skipped pipelines are considered successful* = **off**. All threads must be resolved. |
| Protected tags (create: Maintainers saja) | `v*` dan `design-tokens-v*` (juga bot rilis), `infra-*`, `ds-*`, `priority-*`, `geo-*`, `brief-*` |
| Runner `deploy` | *Protected*: hanya menjalankan job dari branch/tag terlindungi |
| CI/CD variables | Kredensial deploy & kunci: masked + protected + scope environment (§12.2) |
| Lain-lain | Sign-up nonaktif, 2FA wajib untuk semua user, visibilitas proyek *private*, *public pipelines* off, CI job token scope dibatasi ke proyek ini |

**Siapa bisa men-deploy production:** pipeline tag terlindungi berjalan dengan izin pembuat tag. Job manual di pipeline tag hanya bisa dijalankan oleh user yang berhak membuat tag tersebut, yaitu **Maintainer**. Jumlah Maintainer dibatasi (tech lead + 1–2 engineer senior, Q-CI-07).

### 2.6 Pengganti merge queue

CE tidak memiliki merge train. Dengan **fast-forward merge**, MR wajib di-rebase ke `main` terbaru sebelum merge. Pipeline terakhir MR (termasuk `mr:gate`) dengan demikian berjalan di atas commit yang persis akan menjadi `main`. Biayanya: rebase berulang saat `main` sibuk. Mitigasinya adalah MR kecil dan pipeline cepat (P-4). Bila antrean rebase menjadi bottleneck (> 5 MR/hari saling menunggu), evaluasi GitLab Premium (merge trains, Q-CI-02).

Path filter `rules:changes` pada MR pipeline dibandingkan dengan `refs/heads/main` (`compare_to`). Job keamanan dan `mr:gate` **selalu** berjalan, sehingga setiap MR punya pipeline.

### 2.7 Label yang dibaca pipeline

| Label | Efek |
|---|---|
| `api-breaking` | Mengizinkan breaking change OpenAPI (QG-13) bila bagian "Rencana versi API" terisi |
| `db-contract` | Mengizinkan migrasi fase contract (QG-11). 2 approval saat tim ≥ 4 |
| `ai-eval` | Memaksa AI eval berjalan, mis. saat mengganti model provider (D-15) |
| `e2e-full` | Menjalankan E2E penuh di MR |
| `automerge` | Hanya untuk MR Renovate (§15.4) |

Label dibaca dari `$CI_MERGE_REQUEST_LABELS`. Setelah label ditambahkan, job terkait cukup di-retry.

### 2.8 Tag & versi

| Tag (protected) | Dibuat oleh | Memicu |
|---|---|---|
| `vX.Y.Z` | semantic-release (job `release:cut`, token bot Maintainer) | Pipeline rilis production (§6) |
| `design-tokens-vX.Y.Z` | Job `release:cut` bila versi paket tokens berubah | Tidak ada deploy (penanda SemVer paket) |
| `infra-YYYY.MM.DD.n` | Maintainer | Apply Ansible production (§11.3) |
| `ds-YYYY.MM.n` | Maintainer (data owner) | Pipeline publish snapshot (§10.2) |
| `priority-X.Y.Z`, `geo-X.Y.Z`, `brief-X.Y.Z` | Maintainer | Pipeline aktivasi versi di production (§10.3–§10.4) |

Seri `0.x` dipakai selama MVP 0 dan closed beta. `v1.0.0` = rilis MVP 1 yang lolos RG-1 s.d. RG-7 (QG-32).

---

## 3. Environments & topologi

### 3.1 Ringkasan environment

| Env | Tujuan | Data | Trigger deploy | Akses | URL | Sumber config |
|---|---|---|---|---|---|---|
| **local** | Pengembangan | Seed **sintetis** (`msp.devtools.seed`) + fixture EC-xx dari 05 | Manual `docker compose up` | Engineer | `http://localhost:3000` (web; `/api` → api), `:8025` Mailpit, `:9001` MinIO console | `.env` dari `.env.example` |
| **CI** | Verifikasi MR & main | Sintetis per job, dibuang | Otomatis | Pipeline | — (`services:` ephemeral di `ci-01`) | Variabel job |
| **preview** | — | — | **Ditunda** (§3.2) | — | — | — |
| **staging** | Prod-like: E2E penuh, DAST, k6, rehearsal rilis data/model, drill alert | Sintetis (termasuk dataset sintetis 5 juta establishment untuk k6, 04 §17) + **subset dataset berlisensi** bila lisensi mengizinkan (Q-05). **Tanpa PII customer**, tanpa salinan DB production | Otomatis per merge ke `main` | Engineer & tester lewat WireGuard. Deploy hanya via pipeline | `https://app.staging.<domain>` | `infra/ansible/inventories/staging` + SOPS |
| **production** | Customer | Penuh | Tag `vX.Y.Z` + job manual Maintainer | Customer via internet. Admin hanya via WireGuard dengan akun bernama + sudo tercatat (auditd). Tanpa akses DB langsung kecuali `msp_readonly_replica` dengan alasan tercatat. | `https://app.<domain>` (`/api` → api, ADR-25), `https://files.<domain>` (MinIO presigned) | `infra/ansible/inventories/production` + SOPS |

**Parity:** versi PostgreSQL/PostGIS, Redis, MinIO, Python, dan Node di local/CI di-pin agar sama dengan staging/production. Script `scripts/ci/check_version_parity.py` membandingkan tag image CI dengan variabel Ansible (`postgres_version`, `postgis_version`, dll.) saat `infra/**` atau `.gitlab-ci.yml` berubah.

### 3.2 Keputusan: preview environment ditunda

MVP **tidak** memakai preview environment per MR. Sebagai gantinya: (1) Storybook statis sebagai artifact MR (*expose_as*, dapat dibuka dari MR) + screenshot wajib; (2) stack lengkap di CI untuk E2E smoke; (3) staging diperbarui ≤ 30 menit setelah merge.

**Alasan:** kapasitas dedicated server tetap (tidak elastis seperti cloud), sehingga stack per MR memakan RAM/CPU staging. Preview web-only yang memanggil API staging menjadikan staging dependensi bersama yang rapuh. Tim MVP juga kecil. **Ditinjau ulang** bila tim ≥ 6 engineer atau review UI menunggu staging > 1 hari secara rutin. Opsi saat itu: web + API mock (MSW) per MR di `stg-01`.

### 3.3 Aturan data per environment

| Aturan | local | CI | staging | production |
|---|:-:|:-:|:-:|:-:|
| Data sintetis | ✓ | ✓ | ✓ | — |
| Subset dataset berlisensi | — | — | ✓ | ✓ |
| Data/upload customer | — | — | — | ✓ |
| Restore backup production | — | — | — | Hanya drill di `dr-01` (zona prod), lalu dihapus (§11.6) |
| Auth stub (`MSP_AUTH_MODE=stub`) | ✓ | ✓ | — | — |
| Flag fault-injection `chaos.*` | ✓ | ✓ | ✓ | Ditolak saat startup |
| Email keluar | Mailpit | Mailpit | Mailpit (tester membaca via UI) | Provider SMTP |

Aplikasi **menolak start** bila `MSP_ENV ∈ {staging, production}` dan auth stub atau flag `chaos.*` aktif (diuji unit test).

### 3.4 Local dev (`infra/docker/compose.yml`)

| Service | Catatan |
|---|---|
| `postgres` (`postgis/postgis:16-3.5`, pin digest) | Superuser lokal. Ekstensi dibuat oleh migrasi pertama (§9.6) |
| `redis` (`redis:7`) | AOF aktif agar perilaku sama dengan production |
| `minio` | Bucket `msp-snapshots`, `msp-uploads`, `msp-exports`, `msp-tiles` dibuat oleh init container |
| `mailpit` | Email undangan/verifikasi |
| `kratos` | Bila D-20 = Ory Kratos. Default lokal: auth stub |
| `migrate` | `alembic upgrade head`, lalu exit. Service lain `depends_on: service_completed_successfully` |
| `api`, `worker`, `web` | Mode dev (`--reload`, `next dev`) |
| `webhook-sink` | Penerima webhook uji (FR-EXP-003) |

### 3.5 Topologi server & zona jaringan

```
                      Internet
          443/80 │                         │ UDP 51820 (WireGuard admin)
   ┌─────────────▼─────────────┐   ┌──────▼──────────────────────────────┐   probe eksternal
   │ app-01 (→ app-02)  [edge] │   │ ops-01                       [ops]  │◄── (di luar DC)
   │ Traefik, web, api, kratos │   │ GitLab CE + registry, runner deploy │
   │ worker pools blue/green,  │   │ Prometheus, Alertmanager, Grafana,  │
   │ redis (AOF), smokescreen, │   │ Loki, Tempo, GlitchTip, Tang        │
   │ otel-collector/alloy      │   └──────┬───────────────┬──────────────┘
   └─────────────┬─────────────┘          │ SSH deploy    │ SSH deploy
      VLAN privat│ (prod)                 │               │
   ┌─────────────┼──────────────┬─────────┘        ┌──────▼───────┐   ┌──────────────┐
┌──▼──────────┐ ┌▼────────────┐ ┌▼─────────────┐   │ stg-01 [stg] │   │ ci-01  [ci]  │
│ db-01       │→│ db-02       │ │ stor-01      │   │ staging all- │   │ runner MR &  │
│ PG16+PostGIS│ │ hot standby │ │ MinIO (EC)   │   │ in-one       │   │ build, cache │
│ PgBouncer   │ │ read replica│ │ repo lokal   │   └──────────────┘   │ MinIO runner │
│ pgBackRest  │ │             │ │ pgBackRest   │                      └──────────────┘
└──────┬──────┘ └─────────────┘ └──────┬───────┘   ci-01: hanya ke ops-01 (GitLab/registry)
       └── WAL/backup ──────────────────┴──► off-site S3-compatible (lokasi lain, Indonesia)
```

| Zona | Host | Inbound yang diizinkan (nftables) |
|---|---|---|
| edge/prod | `app-01` (`app-02`) | 80/443 publik → Traefik. SSH hanya dari subnet WireGuard dan IP privat `ops-01` |
| data/prod | `db-01`, `db-02`, `stor-01` | 5432/6432 dari app host; 9000 MinIO dari app host & Traefik; scrape exporter dari `ops-01`; SSH dari WireGuard/`ops-01` |
| ops | `ops-01` | UDP 51820 publik (WireGuard). GitLab, registry, Grafana hanya dari WireGuard dan zona privat |
| ci | `ci-01` | Tanpa inbound publik. Outbound: internet (package registry) + `ops-01` (GitLab, registry). **Tidak** punya rute ke zona prod/stg |
| stg | `stg-01` | 443 dari WireGuard. SSH dari WireGuard/`ops-01` |

**Sizing awal (indikatif, direvisi setelah Q-02):**

| Host | Spesifikasi |
|---|---|
| `app-01` | 16 core, 64 GB RAM, 2× NVMe 1 TB (RAID1) |
| `db-01`, `db-02` | 16–24 core, 128 GB RAM, 4× NVMe 2 TB (RAID10) |
| `stor-01` | 8 core, 32 GB RAM, ≥ 4 disk data (erasure coding) |
| `ops-01` | 16 core, 64 GB RAM, 2× NVMe 2 TB (RAID1) |
| `ci-01` | 16 core, 64 GB RAM, 2× NVMe 1 TB |
| `stg-01` | 16 core, 64–128 GB RAM, 2× NVMe 2 TB (menampung dataset sintetis 5 juta untuk k6) |

**Risiko yang diterima di MVP:** `app-01` adalah SPOF (04 §2.2). Failover DB manual (RTO ±30 menit). `ops-01` menampung GitLab sekaligus monitoring. Bila `ops-01` mati, deploy berhenti, tetapi rollback tetap bisa karena image lama ada di cache host dan playbook dapat dijalankan Maintainer dari laptop via WireGuard (break-glass). Penambahan `app-02` dijadwalkan sebelum paid GA (Q-CI-05).

---

## 4. Pipeline merge request

Pipeline berjalan pada event `merge_request_event`. Pipeline branch ganda dicegah oleh `workflow:rules`. `interruptible: true` membatalkan pipeline lama saat ada push baru. MR draft hanya menjalankan lint, unit test, dan security.

### 4.1 Path filter (`rules:changes`, `compare_to: refs/heads/main`)

| Grup | Path | Job |
|---|---|---|
| backend | `apps/api/**`, `data/**`, `pyproject.toml`, `uv.lock` | `backend:lint`, `backend:test` |
| migrations | `apps/api/migrations/**`, model SQLAlchemy (`apps/api/msp/**/internal/models*.py`) | `migrations` |
| web | `apps/web/**`, `packages/**`, `pnpm-lock.yaml`, `turbo.json` | `web:lint`, `web:test`, `web:build` |
| contracts | `contracts/**`, `apps/api/msp/**/routes.py`, `apps/api/msp/**/public.py`, `packages/api-client/**` | `contracts` |
| tokens | `docs/02-design-system/tokens/**`, `packages/design-tokens/**` | `tokens` |
| e2e | `apps/**`, `packages/**`, `contracts/**`, `infra/docker/**` | `build:mr-images`, `e2e:smoke` |
| scoring | `apps/api/msp/intelligence/internal/scoring/models/**`, konfigurasi geo | `model:regression` |
| prompts | `apps/api/msp/ai/**/prompts/**`, `apps/api/msp/ai/**/eval/**` | `ai:eval` |
| er | `apps/api/msp/identity/**` | `er:golden-synthetic` |
| infra | `infra/ansible/**`, `infra/docker/**` | `infra:lint`, `infra:molecule` |
| ci | `.gitlab-ci.yml`, `ci/**`, `scripts/ci/**`, `renovate.json` | `ci:lint` |
| docs | `docs/**`, `**/*.md` | `docs` |

Selalu berjalan: `mr:title`, `mr:gate`, `security:secrets`, `security:sast`, `security:deps`.

### 4.2 Job

| Job | Isi | Tool | Gate |
|---|---|---|---|
| `mr:title` | Validasi Conventional Commits pada `$CI_MERGE_REQUEST_TITLE` | commitlint | QG-01 |
| `mr:gate` | Approval, label, thread (§2.3). Manual, blocking. | script + GitLab API | QG-02 |
| `backend:lint` | `ruff check`, `ruff format --check`, `mypy` (strict untuk `msp`), `lint-imports` (batas modul ADR-01) | uv, ruff, mypy, import-linter | QG-03, QG-04 |
| `web:lint` | ESLint, `prettier --check`, `tsc --noEmit` (`turbo run … --affected`) | pnpm, Turborepo | QG-03, QG-04 |
| `backend:test` | `services:` PostGIS + Redis → `alembic upgrade head` → pytest unit + integration (`-n auto`) → **tenant isolation suite** (koneksi role `msp_app`, `NOBYPASSRLS`) → test redaksi PII (log & prompt) → coverage gate → laporan JUnit + Cobertura ke MR | pytest, Hypothesis (profil `ci`), pytest-xdist, coverage | QG-05, QG-06, QG-07, QG-09, QG-10 |
| `web:test` | Vitest + Testing Library dengan coverage threshold | Vitest | QG-05, QG-08 |
| `migrations` | (1) satu head; (2) `upgrade head` dari DB kosong; (3) `alembic check` (tidak ada perubahan model tertunda); (4) render SQL migrasi baru (offline `--sql`) → **squawk**; (5) cek RLS: setiap tabel tenant (`workspace_id`) memiliki `ENABLE` + `FORCE ROW LEVEL SECURITY` + policy `USING`/`WITH CHECK` (allowlist ber-owner security); (6) fase contract hanya dengan label `db-contract` | Alembic, squawk, `check_rls.py`, `check_migration_phase.py` | QG-11 |
| `contracts` | (1) spec hasil generate FastAPI **identik** dengan `contracts/openapi/openapi.yaml` (ADR-08); (2) Spectral; (3) **oasdiff** breaking vs `main` → gagal kecuali label `api-breaking` + rencana versi; (4) regenerate client TS → `git diff --exit-code`; (5) `check-jsonschema --check-metaschema` untuk `contracts/events/**` & `contracts/input/**` + validasi contoh payload; (6) skema event versi terbit immutable; (7) Schemathesis smoke terhadap API ephemeral (tanpa 5xx, response sesuai skema) | Spectral, oasdiff, openapi-typescript, check-jsonschema, Schemathesis | QG-12, QG-13, QG-14 |
| `tokens` | Validasi DTCG `design-tokens.json` (resolusi alias) → build Style Dictionary → `git diff --exit-code` output ter-commit → kontras: pasangan di `contrast-pairs.json` ≥ 4,5:1 (teks) / 3:1 (UI, teks besar), light & dark → versi paket naik bila token berubah | Style Dictionary, `check-contrast.ts` | QG-15 |
| `web:build` | `next build` (standalone) + Storybook → axe pada setiap story (0 serious/critical) → Storybook sebagai artifact (`expose_as`, 7 hari) | Next.js, Storybook, axe-core | QG-16 |
| `build:mr-images` | Build `msp-api` & `msp-web` (BuildKit rootless, cache registry) → push ke `…/mr/<image>:<sha>` (cleanup 7 hari) | BuildKit | — |
| `e2e:smoke` | Image MR sebagai `services:` (api + migrasi + seed, worker, web, PostGIS, Redis, MinIO, webhook-sink) → Playwright `@smoke` (chromium, `parallel: 2`) + axe. *Critical path*: login (stub) → Offering → ICP → hasil pertama; search + filter + zero-result; profil + evidence drawer; unlock (debit credit); export kecil; webhook test; kill switch AI menampilkan pesan nonaktif. Detail di 05. | Playwright, @axe-core/playwright | QG-16, QG-17 |
| `er:golden-synthetic` | Golden set **sintetis** di repo → precision/recall dilaporkan di MR | pytest | QG-26 |
| `model:regression` | Versi naik + versi terbit tidak berubah (`versions.lock`), recompute fixture "golden scores" → diff sebagai komentar MR | pytest, script | QG-23, QG-24 |
| `ai:eval` | Bila prompt berubah/label `ai-eval`: eval set sintetis (§10.4). Butuh API key LLM → variabel *masked* di-scope environment `ai-eval`, job `when: manual` untuk MR | harness eval | QG-25 |
| `security:secrets` | gitleaks pada commit MR (`$CI_MERGE_REQUEST_DIFF_BASE_SHA..HEAD`) | gitleaks | QG-18 |
| `security:sast` | Semgrep CE (ruleset tersimpan di repo: snapshot `p/python`, `p/typescript`, `p/owasp-top-ten` + aturan kustom; `--metrics=off`) + Bandit (`-lll`) | Semgrep, Bandit | QG-19 |
| `security:deps` | OSV-Scanner (`uv.lock`, `pnpm-lock.yaml`) + license allowlist, pip-audit, `pnpm audit --audit-level high`, Trivy fs (`vuln,secret,misconfig`) | OSV-Scanner, pip-audit, pnpm, Trivy | QG-19, QG-20 |
| `infra:lint`, `infra:molecule` | ansible-lint, yamllint, `--syntax-check`, Trivy config (Dockerfile, compose); Molecule untuk role kritis (`postgres`, `msp_app`, `traefik`, `firewall`) | ansible-lint, Molecule | QG-21 |
| `ci:lint` | Validasi `.gitlab-ci.yml` via API CI Lint, image job di-pin digest, tidak ada `curl \| sh`, validasi `renovate.json` | script, check-jsonschema | QG-21 |
| `docs` | markdownlint-cli2, lychee `--offline`, `check_doc_ids.py` (ID FR-/NFR-/QG- yang dirujuk ada di dokumen pendefinisi; advisory) | markdownlint-cli2, lychee | QG-22 |

### 4.3 Anggaran waktu (p50, paralel via `needs`)

| Job | Target | Teknik |
|---|---|---|
| `mr:title`, `security:secrets` | < 0,5 menit | — |
| `backend:lint`, `web:lint`, `web:test` | 1,5–3 menit | Cache uv/pnpm/Turborepo di cache S3 runner (MinIO `ci-01`), `--affected` |
| `backend:test` | 6–7 menit | `pytest -n auto`, Hypothesis profil `ci`, DB template |
| `contracts`, `tokens`, `migrations` | 1–4 menit | Hanya bila path berubah |
| `web:build` | 5–6 menit | Cache `.next/cache` |
| `build:mr-images` → `e2e:smoke` | 3 + 6 menit (**jalur kritis**) | BuildKit registry cache, 2 shard, ≤ 10 skenario |
| `security:sast`, `security:deps` | 2–3 menit | DB Trivy/OSV di-cache |

Bila p50 > 10 menit dua minggu berturut-turut: tambah concurrency runner, tambah shard, atau pindahkan test ke nightly (§16.3).

### 4.4 Syarat merge

*Pipelines must succeed* mencakup **semua** job yang terpicu, termasuk `mr:gate`. Tidak ada daftar required check terpisah. Job hanya boleh bersifat advisory (`allow_failure: true`) bila tercantum "Advisory" di §8.

---

## 5. Pipeline main

Trigger: push ke `main` (hasil fast-forward merge) dan ke `release/v*` (hotfix, §6.6).

### 5.1 Build image (sekali per commit)

| Image | Isi | Dipakai oleh |
|---|---|---|
| `msp-api` | Paket Python `msp` (multi-stage uv, `python:3.13-slim` pin digest, user non-root) | api, semua worker pool, dispatcher, scheduler, reconciler, migrasi (one-off), job data/model (§10) |
| `msp-web` | Next.js `output: standalone` di `node:24-slim` (pin digest, non-root) | web |

Seluruh service backend berasal dari satu image, sehingga versi API dan worker dijamin identik.

Langkah per image (runner `ci-build`):
1. BuildKit rootless → **push** `registry/<proyek>/msp-api:sha-<commit>` dengan provenance `mode=max`. Digest dicatat sebagai variabel `dotenv` (`API_IMAGE=…@sha256:…`). Label OCI memuat revisi git. **Versi aplikasi tidak di-bake**, tetapi disuntikkan saat deploy (`MSP_APP_VERSION`), karena tag rilis dibuat setelah build (§6.1).
2. **Trivy image**: 0 Critical/High yang fixable (QG-27). Pengecualian hanya via VEX/`.trivyignore` bertanggal kedaluwarsa.
3. **SBOM** syft (SPDX JSON) → artifact (90 hari) + disalin ke bucket evidence `msp-evidence` di `stor-01` (retensi ≥ 1 tahun).
4. **cosign sign** + `cosign attest --type spdxjson` dengan private key dari SOPS, `--tlog-upload=false`. **Penandatanganan hanya terjadi setelah langkah 2 lolos**, sehingga tanda tangan berfungsi sebagai segel "lolos gate".

**Web & build-once:** nilai yang bergantung environment (URL, DSN GlitchTip, nama env) tidak boleh memakai `NEXT_PUBLIC_*` karena di-inline saat build. Nilai tersebut dibaca server saat runtime dan dikirim lewat runtime config. Lint rule kustom memblokir `NEXT_PUBLIC_*` di luar allowlist.

### 5.2 Deploy ke staging

Job `deploy:staging` (runner `deploy`, `resource_group: staging`, `environment: staging`) menjalankan `infra/ansible/playbooks/deploy.yml` terhadap inventory staging dengan mekanisme yang **sama** dengan production (§6.3), tetapi tanpa canary bertahap:

1. Verifikasi tanda tangan cosign + attestation SBOM dengan `infra/keys/cosign.pub` (QG-28).
2. Materialisasi secret SOPS → **tmpfs** `/run/msp/<warna>/` (ADR-22).
3. Pull image by digest. **Migrasi** sebagai container one-off (`alembic upgrade head`, role `msp_owner`, advisory lock). Gagal → deploy berhenti, warna lama tetap melayani.
4. Start warna baru (worker, api, web) → tunggu healthcheck (`/readyz`: DB, Redis, revisi skema ≥ minimum yang dibutuhkan image).
5. Drain worker warna lama (SIGTERM graceful) → alihkan 100% traffic Traefik → smoke → hentikan api/web warna lama.
6. Catat deployment (GitLab environment), anotasi Grafana, release GlitchTip `msp@sha-<short>`.

### 5.3 Verifikasi staging (post-deploy)

| Suite | Isi | Gate |
|---|---|---|
| Smoke | Health, login user sintetis, search, profil, export kecil, webhook test | QG-29 |
| E2E penuh | Playwright seluruh suite (chromium, 4 shard) + axe di setiap halaman | QG-29, QG-16 |
| Kontrak | Schemathesis terhadap API staging (`--checks all`, workspace sintetis khusus) | QG-29 |
| Isolasi tenant (deployed) | Dua workspace sintetis, akses silang via HTTP → 404/403. Tanpa kebocoran di list/search/export/webhook | QG-29, QG-09 |
| DAST | ZAP baseline **tidak** dijalankan per commit (durasi). Jalan nightly dan untuk kandidat rilis (§7) | QG-30 |

Semua lulus → job `mark:staging-verified` menulis **commit status `staging-verified`** (GitLab commit status API) pada SHA tersebut. Status ini prasyarat `release:cut` dan promosi (§6.2). E2E memakai workspace sintetis per run (`e2e-<pipeline_id>`) yang dibersihkan job harian.

### 5.4 Kebijakan `main` merah

Build atau verifikasi gagal → notifikasi kanal tim + issue `main-red` yang di-assign ke author commit. Aturan **stop-the-line**: revert atau fix-forward ≤ 1 jam. Bila tidak ada pemilik, Maintainer me-revert. Commit yang gagal tidak pernah mendapat `staging-verified`.

---

## 6. Pipeline release

### 6.1 Versioning (semantic-release)

- Job **`release:cut`** (manual, di pipeline `main`, `needs: mark:staging-verified`) menjalankan semantic-release: menganalisis commit sejak tag terakhir → menentukan versi (patch/minor/major) → membuat tag `vX.Y.Z` + **GitLab Release** dengan catatan rilis. Plugin: `commit-analyzer` (preset conventionalcommits), `release-notes-generator`, `@semantic-release/gitlab`. **Tanpa** `@semantic-release/git`: tidak ada commit balik, sehingga tag menunjuk commit yang sudah dibangun dan `staging-verified` (build once).
- Token: project access token bot (role Maintainer, scope `api` + `write_repository`), masked + protected, di-scope ke environment `release`. Token berkedaluwarsa ≤ 365 hari (§12.4).
- Changelog tinggal di GitLab Releases. Ringkasan untuk customer (id-ID) ditulis product dari Release tersebut.
- `design-tokens-vX.Y.Z`: versi di `packages/design-tokens/package.json` dinaikkan di MR (QG-15). `release:cut` membuat tag bila versi itu belum ber-tag.

### 6.2 Pipeline tag `vX.Y.Z`: readiness

| Cek (job `release:resolve`, `release:readiness`) | Berlaku | Gate |
|---|---|---|
| Commit tag memiliki status `staging-verified` (menunggu ≤ 45 menit) | Semua | QG-29 |
| Digest `msp-api` & `msp-web` untuk commit tag (tag `sha-<commit>`) ada, tertandatangani, punya SBOM | Semua | QG-28 |
| Tidak ada temuan Critical/High terbuka: rescan Trivy kedua digest + laporan SAST/deps terakhir `main` | Semua | QG-19, QG-27, NFR-SEC-06 |
| ZAP baseline (commit ini atau nightly ≤ 24 jam pada commit yang sama atau lebih baru) tanpa High | Minor & major | QG-30 |
| Hasil k6 kandidat rilis memenuhi threshold | Minor & major | QG-31 |
| ER golden set penuh (nightly terakhir) lulus | Minor & major | QG-26 |
| Drill restore terakhir lulus dan berumur ≤ 35 hari | Minor & major | QG-35 |
| Issue **Release Readiness** versi ini tertutup dengan bukti RG-1 s.d. RG-7 | Major dan `v1.0.0` | QG-32 |
| Bukan masa freeze (§6.5) dan error budget bulan berjalan ≤ 75% | Semua kecuali hotfix | NFR-AVAIL-01 |

Readiness lolos → job **`deploy:production`** (manual, `allow_failure: false`, `resource_group: production`, runner `deploy`) siap dijalankan oleh **Maintainer**. Saat tim ≥ 2 Maintainer, script *four-eyes* menolak bila pemicu deploy sama dengan pemicu `release:cut`.

### 6.3 Deploy production: blue/green + canary di Traefik

Dua project Compose per host: `msp-blue` dan `msp-green`. Masing-masing berisi web, api, dan semua worker pool. Service api/web setiap warna diberi label Docker (`api-blue@docker`, `api-green@docker`, dst.). Router dan **weighted service** didefinisikan di file provider Traefik yang dirender Ansible (§17.3).

```
 langkah                       traffic api/web            worker
 0. awal                       blue 100%                  blue
 1. migrasi (expand-only)      blue 100%                  blue
 2. start green + healthcheck  blue 100%                  blue + green (keduanya konsumsi)
 3. drain worker blue (SIGTERM, selesaikan pesan in-flight)          green
 4. canary                     blue 90% / green 10%  (10 menit, dipantau Prometheus)
 5. promote                    green 100%            (bake 20 menit, blue api/web tetap hidup)
 6. selesai                    green 100%; file Traefik dirender ulang tanpa blue; blue dihentikan
 gagal di langkah 2–5 → rescue: traffic 100% ke blue, start ulang worker blue, hentikan green
```

- **Urutan worker lebih dulu.** Worker baru menerima payload job versi N-1 dan N. Tipe job/aktor baru hanya di-enqueue setelah worker baru aktif, atau di balik flag. Singleton (scheduler, dispatcher, reconciler) aman berjalan di dua warna karena memakai leader lock/`SKIP LOCKED` (04 §2.2, ADR-06, ADR-07).
- **Drain:** `stop_grace_period` per pool (mis. 15 menit untuk `pipeline`). Job panjang wajib resumable (state di Postgres, ADR-06). Pesan yang belum di-ack dikirim ulang.
- **Referensi warna:** file Traefik tidak boleh merujuk service warna yang sudah dihentikan. Karena itu file dirender ulang sebelum warna lama dimatikan.

**Rollback otomatis (QG-33).** Playbook memanggil `scripts/deploy/watch_canary.py`, yang mengevaluasi query Prometheus setiap 30 detik selama canary dan bake. Pelanggaran → blok `rescue` Ansible.

| Sinyal | Threshold | Dasar |
|---|---|---|
| Rasio 5xx service warna baru (metrik Traefik) | > 2% selama 3 menit (normal < 0,5%) | NFR-AVAIL-01 |
| p95 `GET /v1/prospects` (histogram OTel, label `color`) | > 800 ms selama 5 menit | NFR-PERF-01 |
| p95 `GET /v1/prospects/{id}` | > 500 ms selama 5 menit | NFR-PERF-02 |
| Server Traefik tidak sehat / container restart loop | > 0 selama 3 menit | Healthcheck |
| Worker: rasio job gagal | > 5% selama 10 menit | RG-5 |
| Inflow DLQ | > 20 pesan / 5 menit (awal, dikalibrasi) | RG-5 |

**Traffic beta rendah:** 10% traffic beta bisa terlalu sedikit untuk sinyal statistik. Selama canary dan bake, job `synthetic-load` menjalankan loop smoke read-only dari workspace sintetis `msp-synthetic-monitor` (tanpa credit, dikecualikan dari metrik produk dan billing), dengan header yang dipaksa ke warna baru.

**Multi-host (`app-02`):** playbook berjalan `serial: 1` per host di belakang VIP (keepalived/VRRP atau load balancer provider). Detail menunggu Q-CI-05.

### 6.4 Verifikasi pasca-deploy

Otomatis: smoke production (read-only, workspace sintetis) → commit status `production-deployed` → tag registry `vX.Y.Z` ditambahkan ke digest yang sama (kemudahan, bukan untuk deploy) → release GlitchTip `msp@vX.Y.Z` → anotasi Grafana → notifikasi. Checklist manual ada di §14.6.

### 6.5 Jendela deploy & freeze

Production: Senin–Kamis 10:00–15:00 WIB. Jumat ≥ 15:00, akhir pekan, dan libur nasional (termasuk periode Lebaran) hanya untuk hotfix. Rilis non-fix dibekukan bila error budget bulanan (99,5% ≈ 3,6 jam) terpakai > 75%.

### 6.6 Hotfix

1. **Default: fix-forward dari `main`.** `main` selalu dapat dirilis karena fitur belum siap ada di balik flag. Hotfix = MR `fix/…` → merge → staging → `release:cut` (patch) → deploy.
2. Bila `main` memuat perubahan yang tidak boleh dirilis: Maintainer membuat `release/vX.Y` dari tag terakhir → cherry-pick → pipeline branch membangun dan memverifikasi di staging (staging sementara menjalankan versi hotfix) → `release:cut` pada branch tersebut (konfigurasi `branches` semantic-release untuk maintenance branch) → tag `vX.Y.(Z+1)`. Fix yang sama wajib masuk `main` lewat MR terpisah.

---

## 7. Scheduled pipelines

Setiap schedule GitLab membawa variabel `SCHEDULE` yang menentukan job yang berjalan (`rules: if: $SCHEDULE == "nightly"`). Schedule berjalan di `main` (terlindungi), sehingga job deploy-side bisa memakai runner `deploy`.

| `SCHEDULE` · job | Jadwal (WIB) | Isi | Lokasi eksekusi | Gate / output |
|---|---|---|---|---|
| `nightly` · regresi penuh | Harian 01:00 | Seluruh pytest (termasuk `slow`, Hypothesis profil `nightly`), Vitest penuh, Playwright chromium + firefox + webkit terhadap staging, viewport 360 px (NFR-COMP-01) | `ci-01` + staging | Issue `nightly-red` |
| `nightly` · k6 | Harian 02:00 | NFR-PERF-01/02/04 pada dataset sintetis 5 juta + 200 workspace × 3 ICP (04 §17), 50 VU, lewat Traefik staging | Container k6 di `stg-01` (dipicu via runner `deploy`) | Advisory nightly. Blocking untuk kandidat rilis (QG-31) |
| `nightly` · ZAP | Harian 03:00 | `zap-baseline` web + API staging. Mingguan: ZAP API scan berbasis OpenAPI | `ci-01` → staging via WireGuard egress | QG-30 |
| `nightly` · data quality | Harian 03:30 | ER golden set penuh, regresi skor vs baseline versi aktif, reproducibility (recompute sampel → hash identik), suite edge case EC-xx | Container one-off di `stg-01`. CI hanya menerima ringkasan agregat | QG-26, NFR-DATA-01 |
| `nightly` · rekonsiliasi | Harian 04:00 | Saldo credit = Σ ledger, usage event vs ledger (staging). Di production, job yang sama dijadwalkan aplikasi dan dipantau alert | `stg-01` | RG-4 |
| `nightly` · rescan image | Harian 04:30 | Trivy pada digest yang **sedang berjalan** di staging & production | `ci-01` | Issue `security` |
| `nightly` · Lighthouse | Harian 05:00 | Halaman kunci staging, budget LCP/INP | `ci-01` | Advisory (NFR-PERF-03) |
| `drift` | Harian 06:00 | `ansible-playbook site.yml --check --diff` untuk staging & production + cek setelan GitLab | Runner `deploy` | Perubahan > 0 → issue `infra-drift` |
| `backup-check` | Harian 07:00 | `pgbackrest info`/`check` kedua repo, umur backup terakhir, lag replikasi MinIO off-site, umur backup GitLab | Runner `deploy` | Issue + alert (juga dipantau Prometheus) |
| `renovate` | Tiap jam kerja, Sen–Jum | Renovate self-hosted (§15.4) | `ci-01` | MR dependency |
| `weekly` · SAST penuh + link eksternal + flag hygiene | Senin 02:00 | Semgrep seluruh repo dengan ruleset terbaru, lychee eksternal, laporan flag kedaluwarsa (§13.3) | `ci-01` | Issue |
| `monthly` · restore drill | Selasa pertama, 09:00 | §7.1 | `dr-01` via runner `deploy` | QG-35 |
| `soak-webhook` (manual) | Kandidat MVP 1 | 24 jam pengiriman webhook (termasuk endpoint flaky buatan) | `stg-01` | RG-3 (≥ 99%) |
| `alert-drill` (manual) | Sebelum MVP 1, lalu tiap kuartal | Injeksi kegagalan via flag `chaos.*` di staging: pipeline failure > 5%, backlog > 1 jam, AI failure > 5%, webhook failure > 5% → verifikasi alert terpicu dan diterima on-call | `stg-01` | RG-5 |

### 7.1 Drill restore (bulanan)

1. Siapkan `dr-01` (server cadangan atau server sewa per jam di zona prod, disk LUKS). Catat `T_target` = sekarang − 5 menit.
2. `pgbackrest restore --type=time --target=<T_target>` dari **repo off-site** (sekaligus menguji jalur DR lokasi lain).
3. **RTO** = mulai restore → PostgreSQL menerima koneksi dan query verifikasi lulus. Target ≤ 4 jam.
4. **RPO** = selisih `T_target` dengan transaksi terakhir yang ada (`max(created_at)` di `platform.audit_log` dan `usage.usage_event`). Target ≤ 15 menit.
5. Verifikasi integritas: jumlah baris tabel kunci vs production (toleransi jeda), `alembic current` = head production, RLS policy ada, sampel objek MinIO off-site bisa dibaca. Restore backup GitLab diuji tiap kuartal.
6. Hapus data dengan aman (`blkdiscard`/hapus volume LUKS). Laporan berisi metrik saja (tanpa data) disimpan di `msp-evidence`. Gagal → incident SEV3, dan QG-35 memblokir rilis minor/major berikutnya.

---

## 8. Quality gates

### 8.1 Daftar QG

Tahap: **MR** = pipeline merge request · **main** = pipeline `main` · **rel** = pipeline tag `v*` · **sch** = scheduled · **data** = pipeline tag rilis data/model · **infra** = pipeline tag `infra-*`. *Blocking* = gagal → merge/deploy/aktivasi berhenti.

| ID | Gate | Tahap | Threshold | Blocking | Sumber |
|---|---|---|---|---|---|
| QG-01 | Judul MR Conventional Commits | MR | Lolos commitlint | Ya | §2.2 |
| QG-02 | MR gate | MR | ≥ 1 approval owner (2 untuk path berisiko saat tim ≥ 4), approver ≠ author, approval setelah commit terakhir, thread resolved, label wajib | Ya | §2.3 (pengganti approval rules) |
| QG-03 | Lint & format | MR | ruff, ESLint, Prettier, import-linter: 0 error | Ya | DoD, NFR-MAINT-01, ADR-01 |
| QG-04 | Type check | MR | mypy (strict untuk `msp`), `tsc --noEmit`: 0 error | Ya | NFR-MAINT-01 |
| QG-05 | Unit & integration test | MR | 100% lulus. Test karantina (§16.3) tidak dihitung, maks 14 hari | Ya | DoD §12.2 |
| QG-06 | Coverage modul kritis | MR | ≥ 85% line untuk normalisasi, ER, scoring | Ya | NFR-MAINT-01 |
| QG-07 | Coverage backend | MR | ≥ 75% line | Ya | NFR-MAINT-01 |
| QG-08 | Coverage frontend | MR | ≥ 70% line `packages/ui` + logika `apps/web` (hooks, lib, formatter). Halaman dicakup E2E | Advisory 4 minggu, lalu Ya | Proposed |
| QG-09 | Isolasi tenant | MR, main, sch, rel | 100% lulus, role `msp_app` `NOBYPASSRLS` | Ya, **tidak bisa di-waive** | NFR-SEC-03, FR-AUD-006, RG-7 |
| QG-10 | Test privasi | MR | Redaksi PII di log dan payload prompt LLM: 100% lulus | Ya | NFR-PRIV-01, NFR-PRIV-04, FR-AI-005 |
| QG-11 | Keamanan migrasi | MR | Satu head; upgrade dari kosong; `alembic check` bersih; squawk 0 error; RLS lengkap; contract hanya dengan `db-contract` | Ya | §9, NFR-SEC-03 |
| QG-12 | Lint OpenAPI | MR | Spectral: 0 error | Ya | DoD §12.2 |
| QG-13 | Breaking change OpenAPI | MR | oasdiff ERR = 0, **kecuali** label `api-breaking` + rencana versi | Ya | DoD §12.2 |
| QG-14 | Sinkronisasi kontrak | MR | Spec generate = spec kontrak; client TS ter-regenerate; contoh payload valid; skema event terbit immutable; Schemathesis smoke tanpa 5xx | Ya | ADR-08, DoD |
| QG-15 | Design tokens | MR | DTCG valid, output up to date, kontras ≥ WCAG AA, versi paket naik bila token berubah | Ya | NFR-A11Y-01 |
| QG-16 | Aksesibilitas | MR, main | axe: 0 pelanggaran serious/critical (story & halaman E2E) | Ya | DoD §12.2, NFR-A11Y-01 |
| QG-17 | E2E smoke | MR, main | 100% `@smoke` lulus (retry 1×; lulus setelah retry = flaky tercatat) | Ya | DoD, RG-3 |
| QG-18 | Tidak ada secret | MR (+ pre-receive hook) | gitleaks: 0 temuan | Ya, **tidak bisa di-waive** | NFR-SEC-04 |
| QG-19 | Tidak ada kerentanan Critical/High | MR, rel | Semgrep severity ERROR = 0; Bandit high = 0; OSV/pip-audit/pnpm audit/Trivy fs fixable Critical/High = 0 | Ya | NFR-SEC-06 |
| QG-20 | License policy | MR | Allowlist: MIT, Apache-2.0, BSD-2/3-Clause, ISC, MPL-2.0, PSF-2.0, 0BSD, Unlicense, CC0-1.0. AGPL/SSPL/GPL ditolak, LGPL perlu review | Ya | Kebijakan legal (Proposed) |
| QG-21 | Lint CI & IaC | MR | CI Lint valid, image di-pin digest; ansible-lint/yamllint bersih; Molecule lulus; Trivy config 0 High | Ya | §11, §15 |
| QG-22 | Dokumentasi | MR | markdownlint 0 error, link internal valid | Ya (path `docs/**`) | — |
| QG-23 | Versi model/geo/prompt immutable | MR | Konfigurasi versi terbit tidak berubah (`versions.lock`, sha256 di `intel.score_model`); perubahan wajib menaikkan versi | Ya | NFR-DATA-01, FR-INTL-009 |
| QG-24 | Regresi skor/geo ditinjau | MR (fixture), data (sampel staging) | Laporan diff terlampir; disetujui data-intel + product sebelum aktivasi; recompute 2× → hash identik | Ya (aktivasi) | FR-INTL-009, NFR-DATA-01 |
| QG-25 | AI eval | MR (prompt berubah), data (aktivasi) | Grounding ≥ 98%; kebocoran PII = 0; format valid 100%; error/refusal ≤ 2% | Ya | FR-AI-001, -004, -005 |
| QG-26 | ER golden set | MR (sintetis), sch (penuh), rel | Precision ≥ 98%, recall ≥ 90% | Ya (MR: sintetis; rel: penuh) | FR-DATA-003, RG-6 |
| QG-27 | Scan image | main, rel | Trivy: 0 Critical/High fixable | Ya (image tidak di-sign) | NFR-SEC-06 |
| QG-28 | Integritas artefak | deploy (staging, prod) | `cosign verify` + `verify-attestation` dengan `cosign.pub` | Ya | §15 |
| QG-29 | Verifikasi staging | main | Smoke, E2E penuh, Schemathesis, isolasi tenant (deployed) 100% → `staging-verified` | Ya (untuk rilis) | DoD, RG-3 |
| QG-30 | DAST baseline | sch, rel | ZAP: 0 High; Medium ditriase ≤ 14 hari | Ya (rilis) | NFR-SEC-06, RG-7 |
| QG-31 | Performa kandidat rilis | rel (minor/major), sch (advisory) | Search p95 ≤ 800 ms & p99 ≤ 2 s; profil p95 ≤ 500 ms; 50 VU; error < 1%; export 10.000 baris ≤ 2 menit | Ya (minor/major) | NFR-PERF-01, -02, -04 |
| QG-32 | Bukti release gate | rel (major, `v1.0.0`) | Issue Release Readiness memuat bukti RG-1 s.d. RG-7, ditutup product + tech lead | Ya | PRD §12.1 |
| QG-33 | Kesehatan canary/bake | rel | Tidak ada pelanggaran sinyal §6.3 | Ya (rollback otomatis) | NFR-AVAIL-01, NFR-PERF-01/02 |
| QG-34 | Validasi snapshot dataset | data | Threshold §10.2 | Ya (publish) | FR-DATA-001, NFR-DATA-01 |
| QG-35 | Drill restore | sch (bulanan) | RPO ≤ 15 menit, RTO ≤ 4 jam, drill terakhir ≤ 35 hari | Ya (rilis minor/major berikutnya) | NFR-AVAIL-02 |
| QG-36 | Perubahan infrastruktur | infra | `--check --diff` staging & production terlampir di MR; apply production hanya dari tag `infra-*` | Ya | §11 |

**Catatan performa staging:** `stg-01` adalah all-in-one dan lebih kecil dari production. QG-31 yang lulus di staging dianggap konservatif untuk production. Sebelum paid GA, k6 dijalankan sekali di hardware production (pre-launch, data sintetis) untuk memvalidasi angka absolut NFR-PERF.

### 8.2 Release gate MVP 1 → bukti dari pipeline

| RG | Bukti | Diproduksi oleh |
|---|---|---|
| RG-1 Relevansi | Laporan usability + median time-to-first-result | Product/Design → issue Release Readiness |
| RG-2 Kepercayaan | Laporan usability + evidence coverage ≥ 95% (dashboard Data Health) | Product + job data quality |
| RG-3 Aksi | E2E export & webhook (QG-29) + laporan `soak-webhook` ≥ 99% | Pipeline `main`, `soak-webhook` |
| RG-4 Usage | Suite billing/idempotensi (QG-05) + rekonsiliasi nightly 100% | MR, `nightly` |
| RG-5 Operasional | Laporan `alert-drill`: empat alert terpicu dan diterima | `alert-drill` |
| RG-6 Kualitas data | QG-26 penuh + ≥ 10 edge case lulus (05) | `nightly` |
| RG-7 Keamanan | QG-09 + 0 Critical/High (QG-19/27/30) + suite SSRF webhook (ADR-20) | MR, `main`, `nightly` |

### 8.3 Kebijakan waiver

- Gate blocking hanya dapat di-waive tech lead (ditambah owner security untuk gate keamanan) lewat issue `qg-waiver` yang memuat alasan, mitigasi, dan **kedaluwarsa ≤ 14 hari**. Waiver diterapkan sebagai variabel/allowlist yang dirujuk dari issue.
- **QG-09 dan QG-18 tidak pernah di-waive.** Temuan secret ditangani dengan rotasi.
- Pengecualian kerentanan memakai VEX/`.trivyignore`/`osv-scanner.toml` bertanggal kedaluwarsa. Pengecualian kedaluwarsa otomatis membuat gate gagal lagi.

---

## 9. Strategi migrasi database

### 9.1 Prinsip

1. **Setiap migrasi kompatibel dengan versi aplikasi yang sedang berjalan (N-1).** Migrasi berjalan sebelum warna baru di-start, dan rollback mengembalikan traffic ke warna lama di atas skema baru.
2. **Expand → migrate → contract** untuk perubahan yang tidak kompatibel, tersebar di beberapa rilis.
3. **Rollback skema = forward fix.** `downgrade()` boleh ditulis untuk kenyamanan lokal, tetapi tidak dijalankan di staging/production. PITR adalah upaya terakhir (§14.3).
4. Migrasi pendek (target < 5 menit di production). Backfill dan rebuild berjalan sebagai **background job**.

### 9.2 Contoh: rename `identity.contact_point.verified_at` → `last_verified_at`

| Rilis | Fase | Migrasi | Kode aplikasi |
|---|---|---|---|
| N | **Expand** | `ADD COLUMN last_verified_at timestamptz NULL` | **Dual-write** ke kedua kolom, baca `verified_at`. Enqueue job `backfill_last_verified_at` (batch 5.000, idempotent, resumable, di-throttle, progres di ops console) |
| N+1 | **Migrate** | (opsional) `CREATE INDEX CONCURRENTLY`. Bila perlu NOT NULL: `CHECK (… IS NOT NULL) NOT VALID` → `VALIDATE CONSTRAINT` | Baca `last_verified_at`, tetap dual-write (rollback ke N aman). Promosi setelah `count(*) WHERE last_verified_at IS NULL AND verified_at IS NOT NULL = 0` |
| N+2 | **Contract** (label `db-contract`) | `DROP COLUMN verified_at`, setelah N+1 stabil di production dan N bukan lagi target rollback | Berhenti menulis `verified_at`. ORM tidak memetakan kolom lama. Kode N+1 sudah tidak membacanya. |

Rename tabel, perubahan tipe, dan split kolom mengikuti pola yang sama. View kompatibilitas boleh dipakai selama transisi.

### 9.3 Aturan

| # | Aturan | Ditegakkan oleh |
|---|---|---|
| M-1 | Tidak ada perubahan destruktif (DROP, RENAME, penyempitan tipe, NOT NULL tanpa jalur aman) di rilis yang sama dengan kode yang berhenti bergantung padanya | `check_migration_phase.py` + `db-contract` (QG-11) |
| M-2 | Index pada tabel berisi: `CREATE INDEX CONCURRENTLY` di `autocommit_block()` Alembic | squawk |
| M-3 | FK/CHECK baru: `NOT VALID`, lalu `VALIDATE CONSTRAINT` di migrasi terpisah | squawk |
| M-4 | `SET lock_timeout = '5s'` dan `statement_timeout` default di `env.py`. Override wajib dijustifikasi. Gagal karena lock → ulangi deploy | `env.py`, review |
| M-5 | Backfill = background job Dramatiq (batched, idempotent, resumable, terpantau). Pengecualian: tabel referensi < 10.000 baris | Review |
| M-6 | Tabel tenant baru: `ENABLE` + `FORCE ROW LEVEL SECURITY` + policy berbasis `workspace_id` **di migrasi yang sama** dengan `CREATE TABLE`; index diawali `workspace_id` (04 §5.1) | `check_rls.py` (QG-11) |
| M-7 | Role per 04 §5.8: `msp_owner` (DDL, hanya untuk migrasi), `msp_app` (`NOBYPASSRLS`), `msp_pipeline`, `msp_ops`, `msp_readonly_replica`. Grant eksplisit di migrasi. Runtime tidak pernah memakai `msp_owner`. | Ansible + test isolasi |
| M-8 | Migrasi memegang advisory lock, sehingga dua runner tidak bisa berjalan bersamaan | `env.py` |
| M-9 | Satu head Alembic. Konflik diselesaikan dengan rebase dan perbaikan `down_revision` | QG-11 |
| M-10 | Tabel terpartisi (mis. `intel.score`, `ingest.observation`): operasi partisi (attach/detach) mengikuti prosedur 04 §5 dan memakai `CONCURRENTLY` bila tersedia | Review |
| M-11 | Tidak ada data pribadi nyata di migrasi atau seed | Review + gitleaks |

### 9.4 Checklist review migrasi

- [ ] Fase (`expand`/`contract`) dicatat di docstring revision
- [ ] Kompatibel dengan kode rilis sebelumnya (baca & tulis)
- [ ] Estimasi baris terdampak dan durasi lock dicantumkan di MR
- [ ] Index `CONCURRENTLY`, constraint `NOT VALID` + `VALIDATE`
- [ ] Tidak ada backfill besar di dalam migrasi
- [ ] RLS + policy + grant untuk tabel tenant
- [ ] Contract: rilis yang berhenti memakai objek sudah berjalan di production ≥ 1 rilis

### 9.5 Eksekusi per environment

| Env | Cara | Kredensial |
|---|---|---|
| local | Service `migrate` di compose | `.env` |
| CI | `alembic upgrade head` sebelum integration test. Job `migrations` dari DB kosong. API service di E2E menjalankan migrasi saat start. | Service container |
| staging / production | Container one-off `msp-api@<digest> alembic upgrade head` di app host (playbook `deploy.yml`), langsung ke `db-01` (bukan lewat PgBouncer transaction mode), sebelum warna baru di-start | `msp_owner` dari SOPS → tmpfs |
| production + `db-contract` | Sama, didahului `pgbackrest backup --type=incr` sebagai titik pemulihan tambahan | — |

### 9.6 PostGIS & ekstensi

- PostGIS **bukan** trusted extension, sehingga `CREATE EXTENSION postgis` butuh superuser. Di staging/production, Ansible (role `postgres`, user `postgres`) membuat `postgis`, `pg_trgm`, `unaccent` saat provisioning DB. Migrasi pertama tetap memuat `CREATE EXTENSION IF NOT EXISTS` agar local/CI (superuser) setara. Di server, perintah ini menjadi no-op.
- Versi paket PGDG (`postgresql-16`, `postgresql-16-postgis-3`) di-pin di Ansible dan disamakan dengan image CI (parity, §3.1).
- Upgrade minor PostGIS (`ALTER EXTENSION postgis UPDATE`) dan minor PostgreSQL: playbook `db-upgrade.yml`, diuji di staging, dilakukan di jendela maintenance (replica dulu, lalu switchover terencana). Upgrade major memakai `pg_upgrade --link` pada replica baru atau logical replication, lewat MR `infra/**` + rencana tertulis.

---

## 10. Rilis data & model

Kode yang mendukung versi baru di-deploy dulu (versi terdaftar, belum aktif). **Aktivasi** dilakukan lewat pipeline tag terlindungi yang namanya sama dengan versinya. Langkah berat berjalan sebagai container one-off (`msp-api@<digest> python -m msp.ops.cli …`) di host environment terkait, dipicu runner `deploy`. Aktor audit = user GitLab yang menjalankan job manual (FR-AUD-001).

### 10.1 Registry versi & reproducibility (NFR-DATA-01)

| Artefak | Versi | Disimpan di | Immutable |
|---|---|---|---|
| Snapshot dataset | `ds-YYYY.MM.n` | File mentah + manifest di MinIO `msp-snapshots` (WORM, 04 §2.2), metadata di `ingest.dataset_snapshot` | Ya (FR-DATA-001) |
| Model skor | `priority-X.Y.Z` | `apps/api/msp/intelligence/internal/scoring/models/*.yaml` + sha256 di `intel.score_model` + `versions.lock` | Ya setelah aktif |
| Metrik geo | `geo-X.Y.Z` | Konfigurasi di repo + `versions.lock` | Ya |
| Prompt AI | `brief-X.Y.Z` | Prompt registry modul `ai` + `versions.lock` | Ya |

Setiap skor dan metrik area menyimpan `snapshot_version` + `model_version`/`geo_version` (04 §7.8, §9.1). Nightly memverifikasi recompute sampel menghasilkan hash identik. Laporan rilis disimpan di `msp-evidence` selama versi masih direferensikan data aktif + 1 tahun.

Status versi: `draft` → `staged` (tervalidasi di staging) → `active` (production) → `retired`.

### 10.2 Snapshot dataset (`ds-YYYY.MM.n`)

```
ops unggah file mentah + manifest ke MinIO prod (landing) → buka MR manifest (`data/snapshots/ds-….yaml`)
 → merge → Maintainer membuat tag ds-YYYY.MM.n → pipeline tag:
   [1] register + checksum  [2] validasi (prod, status VALIDATING)  [3] rehearsal staging (sampel)
   [4] gate QG-34  [5] job manual `publish` (Maintainer)  [6] recompute skor stale  [7] verifikasi
```

| Langkah | Detail |
|---|---|
| 1. Register | Manifest: sumber, `captured_at`, periode referensi, tag lisensi (BR-PRIV-04), checksum per file. Checksum tidak cocok → berhenti. |
| 2. Validasi | Lifecycle 04 §4.2 (`VALIDATING`): parsing Input Contract, laporan validasi (jumlah baris, reject per alasan, fill-rate per field, delta vs snapshot aktif). Data tetap di zona production. |
| 3. Rehearsal staging | Sampel terstratifikasi (wilayah × kategori) bila lisensi mengizinkan (Q-05). Pipeline normalize → resolve → verify → skor → geo. Diukur: ER golden metrics, distribusi skor vs snapshot sebelumnya, durasi (ekstrapolasi ke NFR-PERF-07). |
| 4. Gate QG-34 | Threshold di bawah. Laporan dilampirkan ke pipeline dan disimpan di `msp-evidence`. |
| 5. Publish | Job manual (Maintainer) → `msp.ops.cli snapshot publish` (setara `/v1/ops/snapshots/{id}/publish`, 04 §11) → pointer snapshot aktif diganti atomik. |
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
| Establishment ber-koordinat terpetakan ke kelurahan | ≥ 90% (FR-DATA-009) |
| Tag lisensi | Ada |
| ER golden set pada rehearsal | Precision ≥ 98%, recall ≥ 90% |

**Rollback:** snapshot immutable dan ID entity stabil (FR-DATA-007). Rollback = publish ulang snapshot sebelumnya + recompute. Mekanismenya ada di 04 §4.2.

### 10.3 Model skor (`priority-X.Y.Z`) dan geo (`geo-X.Y.Z`)

| Langkah | Detail | Gate |
|---|---|---|
| 1. MR konfigurasi | Versi baru (versi lama tidak boleh berubah). Unit test + fixture "golden scores" → diff di komentar MR | QG-23, QG-05 |
| 2. Merge & deploy | Versi terdaftar `draft` di staging & production | Pipeline biasa |
| 3. Tag versi | Maintainer membuat tag `priority-X.Y.Z` → pipeline aktivasi | — |
| 4. Recompute offline (staging) | Container one-off menghitung versi lama & baru pada sampel staging (ICP/preset Offering representatif) | — |
| 5. Laporan regresi | Distribusi skor per band, matriks transisi band (HOT/WARM/COLD/UNSCORED), pergeseran per faktor, overlap top-50 per ICP sampel, perubahan coverage. Geo: distribusi metrik area, jumlah area coverage LOW, perubahan peringkat territory. Determinisme: hitung 2× → hash identik | QG-24 |
| 6. Approval | Job manual `activate` (Maintainer), setelah approval data-intel + product tercatat di issue rilis model. Bila transisi band > 15% sampel atau overlap top-50 < 70%, product wajib menulis justifikasi. | QG-24 |
| 7. Aktivasi | `msp.ops.cli score-model activate` → versi `active` → recompute bertahap (epoch baru diaktifkan atomik, ADR-16; NFR-PERF-05). Epoch versi lama disimpan selama bake 7 hari. | — |
| 8. Pantau | Intelligence Health (FR-OPS-003): distribusi per versi, feedback skor (FR-INTL-010) | — |

**Rollback:** aktifkan kembali epoch versi sebelumnya (masih tersimpan selama bake). Setelah bake, versi lama tetap dapat direproduksi dari snapshot + versi.

### 10.4 Prompt AI (`brief-X.Y.Z`)

| Langkah | Detail | Gate |
|---|---|---|
| 1. MR prompt | Versi baru. Ganti provider/model (D-15) = bump minor + label `ai-eval` | QG-23 |
| 2. Eval MR | Eval set **sintetis** di repo (≥ 200 profil company-level: data jarang/Unknown, multi-cabang, evidence bertentangan, person hanya sebagai role). Metrik: grounding (validator sitasi FR-AI-004) ≥ 98%; nilai contact point pribadi di prompt = 0; format valid 100%; error/refusal ≤ 2%; latency p95 dan biaya per brief dilaporkan (NFR-PERF-06, NFR-COST-01) | QG-25 |
| 3. Tag `brief-X.Y.Z` | Eval diulang di staging dengan set diperluas (company-level, D-14) | QG-25 |
| 4. Aktivasi bertahap | Flag `rel.ai.brief-<versi>`: workspace internal → cohort beta → 100%. Cache brief berbasis hash input **termasuk** versi prompt (FR-AI-003) | — |
| 5. Pantau | Kegagalan AI, feedback 👍/👎 (FR-AI-007), biaya per brief. Alert AI failure > 5% (RG-5) | — |

**Rollback:** aktifkan kembali versi prompt sebelumnya. Kill switch AI (FR-AI-006) sebagai upaya terakhir.

---

## 11. Infrastructure as Code (Ansible)

### 11.1 Layout

```
infra/
├─ ansible/
│  ├─ ansible.cfg              host key checking aktif; known_hosts di repo
│  ├─ requirements.yml         collection di-pin: community.docker, community.postgresql,
│  │                           community.sops, community.general, ansible.posix
│  ├─ inventories/
│  │  ├─ staging/     hosts.yml, group_vars/{all,app,db,stor}/vars.yml, secrets/*.sops.yaml
│  │  └─ production/  hosts.yml, group_vars/..., secrets/*.sops.yaml
│  ├─ roles/
│  │  ├─ base            hardening (SSH key-only, sysctl, auditd, chrony), unattended-upgrades (security)
│  │  ├─ users           akun admin bernama, sudo tercatat, tanpa login root
│  │  ├─ firewall        nftables per zona (§3.5)
│  │  ├─ wireguard       peer admin + runner deploy
│  │  ├─ fail2ban
│  │  ├─ luks_unlock     Clevis + Tang (ops-01), fallback dropbear-initramfs
│  │  ├─ docker          Docker Engine + compose plugin, log driver, daemon.json
│  │  ├─ traefik         static config, ACME, file provider, middlewares
│  │  ├─ msp_app         template compose blue/green, tmpfs secret, scripts deploy
│  │  ├─ postgres        PG16 + PostGIS (PGDG), tuning, role DB, ekstensi, replication, PgBouncer
│  │  ├─ pgbackrest      repo lokal (stor-01) + off-site, cipher, jadwal systemd timer
│  │  ├─ redis           AOF everysec, maxmemory policy
│  │  ├─ minio           single-node multi-drive, SSE, bucket, lifecycle, replikasi off-site
│  │  ├─ monitoring_agent  node/postgres/redis exporter, Grafana Alloy, otel-collector
│  │  ├─ monitoring_stack  Prometheus, Alertmanager, Grafana, Loki, Tempo, GlitchTip, blackbox
│  │  ├─ gitlab          Omnibus, server hooks, backup
│  │  └─ gitlab_runner   runner ci-docker, ci-build, deploy
│  ├─ playbooks/
│  │  ├─ site.yml          konvergensi penuh (dipakai juga untuk drift --check)
│  │  ├─ deploy.yml        rilis aplikasi blue/green (§17.2)
│  │  ├─ rollback.yml      kembali ke warna/digest sebelumnya (§14.1)
│  │  ├─ data-job.yml      container one-off untuk rilis data/model (§10)
│  │  ├─ rehydrate.yml     isi ulang secret tmpfs setelah reboot (§12.3)
│  │  ├─ db-failover.yml   promote db-02, arahkan PgBouncer/app, bangun ulang standby
│  │  ├─ db-upgrade.yml
│  │  ├─ restore-drill.yml
│  │  └─ rotate-secrets.yml
│  └─ molecule/            skenario role kritis
├─ docker/                 compose lokal & CI
└─ keys/                   cosign.pub, kunci publik age per environment
```

### 11.2 Pipeline infra

| Event | Langkah | Runner |
|---|---|---|
| MR menyentuh `infra/**` | ansible-lint, yamllint, `--syntax-check`, Molecule, Trivy config (QG-21). Tanpa akses ke server (MR tidak menerima variabel protected) | `ci-docker` |
| Merge ke `main` | `site.yml --check --diff` terhadap staging & production → diff sebagai artifact dan komentar MR (via API) → `site.yml` **apply ke staging** (otomatis) | `deploy` |
| Tag `infra-YYYY.MM.DD.n` (Maintainer) | `--check --diff` production diulang → job manual **apply production** (`serial: 1`, host demi host) | `deploy` |
| Scheduled `drift` | `--check --diff` harian | `deploy` |

`--check` tidak sempurna: modul `command`/`shell` dilewati. Karena itu task wajib idempoten dengan `changed_when`/`creates` yang benar, dan task non-idempoten diberi tag `never_check`. Diff yang mengandung secret disembunyikan dengan `no_log`/`diff: false`.

### 11.3 Penerapan perubahan

- **Tidak ada SSH manual untuk perubahan konfigurasi.** Semua lewat MR → tag. SSH manual hanya untuk diagnosis atau break-glass, tercatat (auditd + log WireGuard), dan ditindaklanjuti dengan MR adopsi.
- Patch keamanan OS: unattended-upgrades untuk paket security (tanpa auto-reboot di host DB). Reboot terjadwal lewat playbook `serial: 1` di jendela maintenance. Upgrade GitLab: mengikuti rilis keamanan ≤ 7 hari untuk severity kritis, dengan backup sebelum upgrade.

### 11.4 Drift

Setiap perubahan yang terdeteksi job `drift` diselesaikan ≤ 2 hari kerja dengan MR adopsi (kode mengikuti server) atau re-apply (server mengikuti kode). Setelan proyek GitLab (protected branch/tag, merge method, variables) juga dibandingkan dengan `gitlab_settings.py`.

### 11.5 Least privilege di server

- User deploy `msp-deploy` di app host: anggota grup `docker`. Akses ini setara root, sehingga kunci SSH-nya hanya ada di variabel protected ber-scope environment, dan login dibatasi dari IP `ops-01`.
- Kunci SSH deploy per environment berbeda. Kunci admin manusia bersifat personal (dicabut saat offboarding lewat role `users`).
- Port DB/Redis/MinIO tidak pernah terbuka ke publik. MinIO publik hanya lewat Traefik `files.<domain>` untuk presigned URL.

### 11.6 Backup & disaster recovery

| Data | Mekanisme | Jadwal | Tujuan | Retensi |
|---|---|---|---|---|
| PostgreSQL | **pgBackRest**: full mingguan (Minggu 01:00), diff harian (01:00), WAL archive kontinu (`archive_mode=on`, `archive_timeout = 60s`, `archive-async`) | systemd timer di `db-01` | repo1: MinIO `stor-01` (pemulihan cepat). repo2: **off-site S3-compatible** di lokasi berbeda, tetap di Indonesia | Full: 4 minggu. PITR window: 14 hari (Proposed) |
| Enkripsi backup | `repo-cipher-type=aes-256-cbc`, passphrase berbeda per repo (SOPS) | — | — | — |
| Replika DB | Streaming replication `db-02` (async) | Kontinu | Pemulihan kegagalan host (RPO ≈ detik) | — |
| MinIO | Versioning + replikasi bucket ke off-site S3-compatible (replikasi MinIO bila target MinIO; selain itu `mc mirror --watch` terjadwal) | Kontinu | Off-site | Lifecycle per bucket (export 24 jam per FR-EXP-001; snapshot permanen) |
| GitLab | `gitlab-backup create` + `gitlab-secrets.json` + `gitlab.rb` (terenkripsi) + data registry | Harian 02:00 | MinIO + off-site | 14 hari |
| Konfigurasi | Repo (Ansible, SOPS). Kunci age privat: salinan offline di 2 Maintainer (hardware key/brankas) | — | — | — |

**Target:** RPO ≤ 15 menit (praktiknya ≤ 1–2 menit dengan `archive_timeout = 60s`), RTO ≤ 4 jam (NFR-AVAIL-02). Prometheus memantau umur backup terakhir, lag WAL archive, dan lag replikasi. Alert dikirim bila lag WAL archive > 5 menit atau backup harian terlewat.

**Skenario:**

| Skenario | Respons | RTO perkiraan |
|---|---|---|
| `db-01` mati | `db-failover.yml`: promote `db-02`, arahkan PgBouncer/app, `db-01` dibangun ulang sebagai standby | ±30 menit |
| Korupsi data logis | PITR pgBackRest ke instance baru (§14.3) | 1–3 jam (tergantung ukuran DB) |
| `app-01` mati | Provision ulang dari Ansible di server cadangan + deploy digest terakhir (`app-02` mengurangi ini menjadi menit) | 1–3 jam |
| Kehilangan DC | Provision server di lokasi lain + restore dari repo off-site | Tergantung SLA provider (Q-CI-04). **Bisa melebihi 4 jam.** |

---

## 12. Secrets & konfigurasi

### 12.1 SOPS + age

```yaml
# .sops.yaml (ilustratif)
creation_rules:
  - path_regex: infra/ansible/inventories/production/secrets/.*\.sops\.yaml$
    age: >-
      age1prod...deployrunner, age1maint...alice, age1maint...bob
  - path_regex: infra/ansible/inventories/staging/secrets/.*\.sops\.yaml$
    age: >-
      age1stg...deployrunner, age1eng...team
  - path_regex: infra/secrets/signing\.sops\.yaml$
    age: >-
      age1ci...signing, age1maint...alice, age1maint...bob
```

- Engineer dapat **mengenkripsi** secret production (kunci publik ada di repo) tetapi tidak dapat mendekripsinya. Dekripsi production hanya oleh runner `deploy` (saat job tag) dan Maintainer.
- Perubahan secret = MR biasa, dengan diff terenkripsi per nilai. Reviewer melihat *key* yang berubah, bukan nilainya.
- File di repo selalu `*.sops.yaml`. Pre-commit hook dan job CI menolak file secret yang tidak terenkripsi (cek metadata `sops`).

### 12.2 GitLab CI/CD variables

| Variabel | Tipe | Protected | Scope environment | Isi |
|---|---|---|---|---|
| `SOPS_AGE_KEY` | File, masked | Ya | `staging` / `production` (nilai berbeda) | Kunci age privat runner deploy per environment |
| `DEPLOY_SSH_KEY` | File | Ya | `staging` / `production` | Kunci SSH user `msp-deploy` |
| `SOPS_AGE_KEY_SIGNING` | File | Ya | `signing` | Membuka `cosign.key` + password |
| `RELEASE_TOKEN` | Masked | Ya | `release` | Project access token bot rilis (Maintainer) |
| `RENOVATE_TOKEN` | Masked | Ya | `renovate` | Token bot Renovate |
| `LLM_EVAL_API_KEY` | Masked | Tidak (MR perlu) | `ai-eval` | Key LLM khusus eval, kuota & budget kecil |
| `MR_GATE_TOKEN` | Masked | Tidak | — | `read_api`, role Reporter |

Job hanya menerima variabel ber-scope bila menyatakan `environment:` yang sesuai. Job yang tidak men-deploy memakai `environment: { name: <x>, action: prepare }`. Runner tidak menyimpan secret di disk: semua datang dari variabel saat job berjalan, dan workspace job dibersihkan setelahnya.

### 12.3 Secret di server

- Ansible mendekripsi SOPS di runner `deploy` lalu menulis setiap secret sebagai **file** di **tmpfs** `/run/msp/<warna>/` (mode 0400). File di-mount ke container sebagai Compose `secrets:` (`/run/secrets/<nama>`) dan dibaca Pydantic Settings (`secrets_dir`). Secret tidak dikirim lewat env var, sehingga tidak tampil di `docker inspect`.
- Konsekuensi tmpfs (ADR-22): setelah reboot, secret hilang. Unit systemd `msp-rehydrate` saat boot memicu pipeline `rehydrate` lewat *pipeline trigger token* (token hanya bisa memicu, tidak membaca secret). Job berjalan di runner `deploy` dan menjalankan `rehydrate.yml`. Bila gagal, on-call menjalankannya manual (runbook).
- Konfigurasi non-secret: variabel inventory Ansible → env var container dengan prefix `MSP_` (12-factor). Contoh: `MSP_ENV`, `MSP_APP_VERSION`, `MSP_COLOR`, `MSP_OTEL_EXPORTER_OTLP_ENDPOINT`, `MSP_S3_ENDPOINT`, `MSP_LOG_LEVEL`. Aplikasi gagal start bila variabel wajib hilang.
- `.env.example` di root berisi semua variabel dengan nilai dummy aman untuk local. `.env` ada di `.gitignore`.

### 12.4 Jangan pernah mencetak secret & rotasi

- Tidak ada `set -x` di job yang menyentuh secret. Task Ansible yang menyentuh secret memakai `no_log: true`. Log aplikasi memakai redaksi terstruktur (BR-PRIV-01, QG-10). Artifact CI hanya berisi data sintetis (trace Playwright 7 hari, diunggah hanya saat gagal).

| Secret | Rotasi terjadwal | Rotasi segera bila |
|---|---|---|
| Password role DB, Redis, MinIO | 90 hari (`rotate-secrets.yml`, dual credential selama transisi) | Offboarding, dugaan bocor |
| Kunci age per environment | 12 bulan (re-encrypt via `sops updatekeys`) | Offboarding Maintainer, runner dikompromikan |
| Kunci SSH deploy | 90 hari | Sama |
| Kunci cosign | 12 bulan. Verifikasi menerima kunci lama + baru selama 30 hari | Dugaan bocor (semua image ditandatangani ulang) |
| Token GitLab (bot, Renovate, trigger) | Kedaluwarsa ≤ 365 hari, diganti 30 hari sebelumnya | Sama |
| API key LLM/email | 90 hari | Sama |
| Passphrase repo pgBackRest | Tidak dirotasi tanpa rencana (rotasi = repo baru) | Dugaan bocor |
| Kunci master webhook | 12 bulan (lihat bawah) | Dugaan bocor |

**Kunci master webhook.** Secret per endpoint (FR-EXP-003) dienkripsi aplikasi dengan AES-256-GCM memakai *key ring* berversi (`secret_kid`, 04 §5.5, ADR-22). Rotasi: (1) tambah versi kunci baru ke SOPS dan deploy (dekripsi menerima semua versi aktif, enkripsi memakai versi terbaru); (2) job background mengenkripsi ulang semua `secret_ciphertext` ke versi baru; (3) setelah 100% + 7 hari, versi lama dihapus dari key ring.

---

## 13. Feature flags & progressive delivery

### 13.1 Tipe flag

| Tipe | Penamaan | Contoh | Umur | Saat flag store gagal |
|---|---|---|---|---|
| Release | `rel.<area>.<nama>` | `rel.geo.territory-ranking`, `rel.ai.brief-1.1.0` | Sementara: dihapus ≤ 30 hari setelah 100% | **Off** |
| Ops kill switch (FR-OPS-007, FR-AI-006) | Nama kapabilitas (04) | `upload.create`, `export.create`, `webhook.deliver`, `ai.generate` | Permanen | Nilai cache terakhir. Tanpa cache → kapabilitas **on** |
| Fault injection | `chaos.<nama>` | `chaos.webhook-5xx` | Permanen | Off. Ditolak di production |
| Eksperimen | `exp.<nama>` | (V1+) | Sementara | Off |

Nama kill switch final ditetapkan di 04. Respons API saat kill switch aktif = `FEATURE_DISABLED` (04 §11).

**Permission dan plan bukan flag.** Entitlement (FR-BILL-003) dan RBAC (FR-AUTH-004) ditegakkan service di server. Flag tidak pernah menjadi satu-satunya penghalang akses.

### 13.2 Mekanisme

- **Registry di repo** (`apps/api/msp/platform/flags/registry.yaml`): nama, tipe, owner, deskripsi, tanggal dibuat, kedaluwarsa (wajib untuk `rel.`/`exp.`), default. CI memeriksa bahwa setiap flag yang dipakai kode (backend & web) terdaftar, dan setiap flag terdaftar dipakai.
- **Nilai runtime** per environment di `platform.feature_flag` (+ override per workspace), diubah lewat ops console (`/v1/ops/feature-flags/{key}`) dan **tercatat di audit log** (FR-OPS-007). Cache in-process (TTL 30 detik) + invalidasi via Redis pub/sub, sehingga kill switch berlaku ≤ 1 menit.
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
| Setelah bake, warna lama masih ada di host | Job manual **`rollback:production`** (pipeline tag versi saat ini atau `main`, Maintainer) → `rollback.yml -e to=previous`: start warna lama dengan digest sebelumnya (image masih di cache host), worker dulu, alihkan traffic 100%, smoke | ≤ 5 menit |
| Lebih dari satu rilis ke belakang | `rollback.yml -e to=vX.Y.Z`. Ditolak bila ada migrasi contract di antara keduanya | ≤ 15 menit |
| `ops-01` (GitLab) tidak tersedia | Maintainer menjalankan `rollback.yml` dari laptop via WireGuard (kunci age & SSH break-glass), lalu dicatat di issue | ≤ 30 menit |

Rollback tidak menjalankan migrasi dan aman karena skema bersifat expand-only terhadap N-1 (§9.1). Host menyimpan image 5 rilis terakhir (prune terjadwal).

### 14.2 Matikan lewat flag

Langkah tercepat untuk kapabilitas bermasalah: kill switch (`ai.generate`, `export.create`, `webhook.deliver`, `upload.create`) atau flag `rel.*` ke off (≤ 1 menit, tercatat di audit). Langkah ini didahulukan sebelum rollback bila masalah terlokalisasi.

### 14.3 Database

- **Forward fix:** migrasi perbaikan lewat pipeline normal (dipercepat: review on-call + Maintainer).
- **PITR** (pgBackRest) hanya untuk korupsi/kehilangan data yang tidak bisa diperbaiki secara forward. Keputusan ada di incident commander + founder, karena data setelah titik restore hilang (termasuk usage/ledger yang harus direkonsiliasi). Restore dilakukan ke instance baru, lalu PgBouncer diarahkan ulang. Prosedurnya dilatih di drill bulanan (§7.1).
- **Kegagalan host primary:** `db-failover.yml` (§11.6).

### 14.4 Rollback data & model

| Artefak | Rollback |
|---|---|
| Snapshot dataset | Publish ulang snapshot sebelumnya + recompute (§10.2) |
| Versi skor/geo | Aktifkan kembali epoch versi sebelumnya (tersimpan selama bake 7 hari) |
| Versi prompt | Aktifkan kembali versi sebelumnya. Kill switch AI bila perlu |

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
3. Preservasi bukti: `platform.audit_log` (immutable), log Traefik/aplikasi di Loki, log auditd dan WireGuard, log akses MinIO. Retensi diperpanjang, tidak ada penghapusan.
4. Draf notifikasi (data terdampak, waktu dan cara kejadian, upaya penanganan dan pemulihan) disiapkan paralel dengan penanganan teknis.

**Post-incident review** (wajib untuk SEV1/2): blameless, ≤ 5 hari kerja, berisi timeline, akar masalah, dan action item ber-owner dan bertenggat (issue `pir`). Setiap insiden menghasilkan minimal satu test regresi atau alert baru. Metrik DORA diperbarui (§16.2).

### 14.6 Checklist verifikasi pasca-deploy

- [ ] Semua container warna aktif *healthy*. Traefik hanya merujuk warna aktif
- [ ] Smoke production lulus (workspace sintetis)
- [ ] 5xx dan p95 search/profil dalam threshold (§6.3) selama 30 menit
- [ ] Worker: backlog normal, tidak ada lonjakan DLQ, rasio gagal < 5%. Worker warna lama sudah berhenti
- [ ] Log container migrasi bersih, `alembic current` = head yang diharapkan
- [ ] GlitchTip: tidak ada issue baru severity tinggi pada release ini
- [ ] Flag sesuai rencana rollout
- [ ] Deployment tercatat di GitLab environment, anotasi Grafana ada
- [ ] Catatan rilis dipublikasikan. Perubahan yang terlihat customer dikomunikasikan (id-ID)

---

## 15. Supply chain security

### 15.1 Platform & pipeline

| Kontrol | Penerapan |
|---|---|
| Image job di-pin | Setiap `image:` dan `services:` memakai `@sha256:<digest>`. Tool CI (oasdiff, squawk, gitleaks, Trivy, syft, cosign, OSV-Scanner, Semgrep, ansible, sops, age, k6, commitlint) dibundel dalam image **`ci-tools`** yang dibangun dari `ci/images/tools/Dockerfile` dengan versi di-pin + checksum. Renovate memperbaruinya. |
| `include:` CI | Hanya `include: local`. Template eksternal tidak dipakai. |
| Tanpa unduhan liar | Larangan `curl \| sh`/`wget \| bash` di job (dicek `ci:lint`) |
| Runner terpisah | `ci-docker` (non-privileged, `allowed_images` dibatasi ke registry internal + daftar image publik yang di-pin), `ci-build` (BuildKit rootless, tanpa `privileged`), `deploy` (protected, di `ops-01`). Runner MR tidak punya rute ke prod/stg. |
| Variabel | Masked + protected + scope environment (§12.2). Variabel protected tidak pernah tersedia di pipeline MR. |
| Push protection | Server hook `pre-receive` menjalankan gitleaks pada setiap push (batas waktu & ukuran dikonfigurasi). Push yang mengandung secret ditolak. |
| Akun | 2FA wajib, sign-up nonaktif, Maintainer minimal, token berkedaluwarsa, review akses per kuartal |
| GitLab | Ikuti rilis keamanan (patch kritis ≤ 7 hari). UI hanya via WireGuard. Sertifikat via ACME DNS-01 atau CA internal |

### 15.2 Dependency

- Lockfile wajib dan instalasi beku: `uv sync --locked`, `pnpm install --frozen-lockfile`.
- `security:deps` di setiap MR + rescan nightly (QG-19, QG-20). Pengecualian bertanggal kedaluwarsa.
- Base image di-pin digest dan diperbarui Renovate.

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
| Eksekusi | Image `renovate/renovate` (pin digest), `platform: gitlab`, token bot di scope `renovate` |
| `minimumReleaseAge` | 3 hari untuk semua update. Security update tanpa penundaan |
| Grouping | `python-dev` (ruff, mypy, pytest-*), `js-lint`, `storybook`, `playwright`, `ci-images` (digest), `docker-base`, `ansible-collections` |
| Automerge | **Patch** dan **minor devDependency** dengan pipeline hijau (label `automerge`, `mr:gate` mengizinkan tanpa approval). Tidak pernah automerge untuk `fastapi`, `sqlalchemy`, `alembic`, `psycopg`, `pydantic`, `dramatiq`, `next`, `react`, collection Ansible, base image. Major selalu manual. |
| Batas | Maks 5 MR terbuka, 2 MR/jam. Dependency Dashboard issue aktif |

---

## 16. Observability pipeline & DORA

### 16.1 Deployment marker & anotasi rilis

Setiap deploy (staging dan production) menghasilkan:
- **Deployment GitLab** (`environment:`) beserta SHA dan job. Ini sumber data DORA.
- **Release GlitchTip**: `msp@vX.Y.Z` (production) atau `msp@sha-<short>` (staging). Source map web diunggah saat deploy dan **tidak** disajikan publik.
- **Anotasi Grafana** (API) di dashboard SLO: latency, error rate, backlog, replikasi DB.
- Atribut resource OpenTelemetry setiap service: `service.name`, `service.version` (= `MSP_APP_VERSION`), `deployment.environment.name`, revisi git, dan `msp.color` (blue/green, dipakai query canary §6.3).

Stack (ADR-21, di `ops-01`): OTel Collector/Grafana Alloy di setiap host → Prometheus (metrik, retensi 30 hari), Loki (log JSON terstruktur, retensi 30 hari, PII teredaksi), Tempo (trace, 7 hari), GlitchTip (error, 90 hari). Alertmanager → kanal chat + email on-call. Blackbox exporter memeriksa endpoint dari dalam DC. **Probe eksternal** di luar DC (VPS kecil di lokasi lain atau layanan uptime) memeriksa `app.<domain>` dan menerima heartbeat *watchdog* Alertmanager (dead man's switch), sehingga DC down atau `ops-01` mati tetap terdeteksi.

### 16.2 DORA (target tahap MVP)

| Metrik | Definisi operasional | Target MVP 1 | Sumber |
|---|---|---|---|
| Deployment frequency | Deploy production sukses per minggu | ≥ 2/minggu (staging: setiap merge) | GitLab Deployments API |
| Lead time for changes | Merge ke `main` → berjalan di production, p50 | ≤ 2 hari kerja | Commit + Deployments |
| Change failure rate | Deploy production yang memicu rollback, hotfix, atau SEV1/2 | ≤ 15% | Deployments + label insiden |
| Failed deployment recovery time | Deteksi → pulih | ≤ 1 jam (rollback ≤ 5–15 menit) | Issue insiden |

Dihitung oleh script mingguan (GitLab API → Prometheus/Grafana, karena dashboard DORA GitLab berbayar) dan ditinjau di retro engineering bulanan.

### 16.3 Kesehatan CI

| Metrik | Target | Tindakan bila meleset |
|---|---|---|
| Durasi pipeline MR | p50 ≤ 10 menit, p90 ≤ 15 menit | Profil job; tambah concurrency/shard; pindahkan ke nightly |
| Merge → `staging-verified` | p50 ≤ 30 menit | Paralelkan E2E staging |
| Antre job (pending) | p90 ≤ 1 menit | Tambah `concurrent` runner atau kapasitas `ci-01` |
| Flaky rate | ≤ 2% job (lulus setelah retry) | **Karantina** (`@pytest.mark.quarantine` / `test.fixme` + issue): jalan di nightly, tidak memblokir MR, maks 14 hari |
| `main` merah | ≤ 1 jam per kejadian | Stop-the-line (§5.4) |
| Disk `ci-01`/registry | < 80% | Cleanup policy, prune cache |

Data dikumpulkan **gitlab-ci-pipelines-exporter** dan laporan JUnit, lalu ditampilkan di dashboard Grafana "CI health".

---

## 17. Contoh konfigurasi (ilustratif)

### 17.1 `.gitlab-ci.yml` (dipersingkat)

```yaml
workflow:
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
    - if: $CI_COMMIT_BRANCH && $CI_OPEN_MERGE_REQUESTS
      when: never                                        # cegah pipeline ganda branch + MR
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
    - if: $CI_COMMIT_BRANCH =~ /^release\/v\d+\.\d+$/
    - if: $CI_COMMIT_TAG =~ /^v\d+\.\d+\.\d+$/
    - if: $CI_COMMIT_TAG =~ /^(infra|ds|priority|geo|brief)-/   # job di ci/*.yml
    - if: $CI_PIPELINE_SOURCE == "schedule" || $CI_PIPELINE_SOURCE == "trigger"

include:
  - local: ci/data-release.yml        # tag ds-*, priority-*, geo-*, brief-*
  - local: ci/infra.yml               # MR infra, apply staging, tag infra-*
  - local: ci/scheduled.yml           # nightly, drift, renovate, drill, rehydrate

stages: [validate, test, build, scan, deploy-staging, verify, release, deploy-production]

variables:
  TOOLS_IMAGE: $CI_REGISTRY/msp/ci-tools@sha256:<digest>
  UV_IMAGE: ghcr.io/astral-sh/uv:python3.13-bookworm-slim@sha256:<digest>
  BUILDKIT_IMAGE: moby/buildkit:rootless@sha256:<digest>
  POSTGIS_IMAGE: postgis/postgis:16-3.5@sha256:<digest>     # = versi PGDG di Ansible
  REDIS_IMAGE: redis:7-alpine@sha256:<digest>
  UV_CACHE_DIR: $CI_PROJECT_DIR/.cache/uv
  FF_NETWORK_PER_BUILD: "true"                              # service saling terhubung per job

default:
  image: $TOOLS_IMAGE
  tags: [ci-docker]
  interruptible: true

.if-mr:   &if-mr   { if: '$CI_PIPELINE_SOURCE == "merge_request_event"' }
.if-main: &if-main { if: '$CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH' }
.if-rel:  &if-rel  { if: '$CI_COMMIT_TAG =~ /^v\d+\.\d+\.\d+$/' }

.rules-backend:
  rules:
    - <<: *if-mr
      changes: { paths: [apps/api/**/*, data/**/*, pyproject.toml, uv.lock], compare_to: refs/heads/main }
    - *if-main

# ---------------- MR: validate & test ----------------
mr:title:
  stage: validate
  needs: []
  rules: [*if-mr]
  script: [echo "$CI_MERGE_REQUEST_TITLE" | commitlint]                      # QG-01

mr:gate:
  stage: .post
  needs: []
  rules:
    - <<: *if-mr
      when: manual
      allow_failure: false                         # pipeline "blocked" → MR tak bisa di-merge
  script: [python scripts/ci/mr_gate.py --codeowners .gitlab/CODEOWNERS]     # QG-02

backend:test:
  stage: test
  image: $UV_IMAGE
  extends: .rules-backend
  needs: []
  services:
    - name: $POSTGIS_IMAGE
      alias: postgres
      variables: { POSTGRES_USER: msp, POSTGRES_PASSWORD: msp, POSTGRES_DB: msp_test }
    - name: $REDIS_IMAGE
      alias: redis
  variables:
    MSP_ENV: test
    MSP_AUTH_MODE: stub
    MSP_DATABASE_URL: postgresql+psycopg://msp:msp@postgres:5432/msp_test
    MSP_REDIS_URL: redis://redis:6379/0
  cache: { key: { files: [uv.lock] }, paths: [.cache/uv] }
  script:
    - uv sync --locked
    - uv run alembic -c apps/api/alembic.ini upgrade head
    - uv run pytest apps/api/tests -n auto -m "not slow and not ai_eval and not tenant_isolation"
      --cov=msp --cov-report= --junitxml=reports/junit-main.xml                 # QG-05
    - uv run pytest apps/api/tests -m tenant_isolation
      --cov=msp --cov-append --cov-report= --junitxml=reports/junit-tenant.xml  # QG-09
    - uv run coverage xml -o reports/coverage.xml
    - uv run coverage report --fail-under=75                                    # QG-07
    - uv run coverage report --fail-under=85
      --include="apps/api/msp/identity/internal/normalize/*,apps/api/msp/identity/internal/resolve/*,apps/api/msp/intelligence/internal/scoring/*"   # QG-06
  coverage: '/^TOTAL.+?(\d+(?:\.\d+)?%)$/'
  artifacts:
    when: always
    expire_in: 14 days
    reports:
      junit: reports/junit-*.xml
      coverage_report: { coverage_format: cobertura, path: reports/coverage.xml }

contracts:
  stage: test
  needs: []
  rules:
    - <<: *if-mr
      changes: { paths: [contracts/**/*, apps/api/msp/**/routes.py, packages/api-client/**/*], compare_to: refs/heads/main }
  script:
    - scripts/ci/export_openapi.sh > /tmp/openapi.yaml && diff -u contracts/openapi/openapi.yaml /tmp/openapi.yaml   # QG-14
    - spectral lint contracts/openapi/openapi.yaml --fail-severity=error                                           # QG-12
    - git fetch origin main --depth=50 && git show origin/main:contracts/openapi/openapi.yaml > /tmp/base.yaml
    - oasdiff breaking /tmp/base.yaml contracts/openapi/openapi.yaml --fail-on ERR
      || scripts/ci/require_label.sh api-breaking --section "Rencana versi API"                                     # QG-13
    - pnpm --filter @msp/api-client generate && git diff --exit-code packages/api-client                           # QG-14
    - check-jsonschema --check-metaschema contracts/events/*.schema.json contracts/input/*.schema.json
    - scripts/ci/validate_examples.sh

build:mr-images:
  stage: build
  image: $BUILDKIT_IMAGE
  tags: [ci-build]
  needs: []
  rules:
    - <<: *if-mr
      changes: { paths: [apps/**/*, packages/**/*, contracts/**/*, infra/docker/**/*], compare_to: refs/heads/main }
  variables: { BUILDKITD_FLAGS: --oci-worker-no-process-sandbox }
  script:
    - scripts/ci/buildkit_build.sh msp-api apps/api "$CI_REGISTRY_IMAGE/mr/msp-api:$CI_COMMIT_SHA"
    - scripts/ci/buildkit_build.sh msp-web apps/web "$CI_REGISTRY_IMAGE/mr/msp-web:$CI_COMMIT_SHA"

e2e:smoke:
  stage: verify
  image: mcr.microsoft.com/playwright:<versi>-noble@sha256:<digest>
  needs: [build:mr-images]
  rules:
    - <<: *if-mr
      changes: { paths: [apps/**/*, packages/**/*, contracts/**/*, infra/docker/**/*], compare_to: refs/heads/main }
  parallel: 2
  services:
    - { name: $POSTGIS_IMAGE, alias: postgres, variables: { POSTGRES_USER: msp, POSTGRES_PASSWORD: msp, POSTGRES_DB: msp } }
    - { name: $REDIS_IMAGE, alias: redis }
    - { name: "minio/minio@sha256:<digest>", alias: minio, command: ["server", "/data"] }
    - name: $CI_REGISTRY_IMAGE/mr/msp-api:$CI_COMMIT_SHA
      alias: api
      command: ["sh", "-c", "alembic upgrade head && python -m msp.devtools.seed --profile e2e && uvicorn msp.app.api:create_app --factory --host 0.0.0.0 --port 8000"]
    - { name: $CI_REGISTRY_IMAGE/mr/msp-api:$CI_COMMIT_SHA, alias: worker, command: ["dramatiq", "msp.app.worker"] }
    - { name: $CI_REGISTRY_IMAGE/mr/msp-web:$CI_COMMIT_SHA, alias: web }
    - { name: "$CI_REGISTRY/msp/webhook-sink@sha256:<digest>", alias: webhook-sink }
  variables:
    MSP_ENV: test
    MSP_AUTH_MODE: stub
    MSP_DATABASE_URL: postgresql+psycopg://msp:msp@postgres:5432/msp
    MSP_REDIS_URL: redis://redis:6379/0
    MSP_S3_ENDPOINT: http://minio:9000
  script:
    - corepack enable && pnpm install --frozen-lockfile
    - scripts/ci/wait_for.sh http://api:8000/readyz http://web:3000/healthz
    - pnpm --filter web exec playwright test --grep @smoke --shard=$CI_NODE_INDEX/$CI_NODE_TOTAL   # QG-16, QG-17
  artifacts: { when: on_failure, expire_in: 7 days, paths: [apps/web/playwright-report/] }

security:secrets:
  stage: validate
  needs: []
  rules: [*if-mr, *if-main]
  script: [gitleaks git --log-opts="${CI_MERGE_REQUEST_DIFF_BASE_SHA:-HEAD~1}..HEAD" --redact --exit-code 1]   # QG-18

security:deps:
  stage: test
  needs: []
  rules: [*if-mr, *if-main]
  script:
    - osv-scanner scan source -r --licenses="MIT,Apache-2.0,BSD-2-Clause,BSD-3-Clause,ISC,MPL-2.0,PSF-2.0,0BSD,Unlicense,CC0-1.0" .  # QG-19, QG-20
    - uv export --locked --format requirements-txt > /tmp/req.txt && pip-audit -r /tmp/req.txt
    - pnpm audit --audit-level high
    - trivy fs --scanners vuln,secret,misconfig --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 .

# … backend:lint, web:lint, web:test, migrations, tokens, web:build, security:sast, model:regression,
#   er:golden-synthetic, ai:eval, ci:lint, docs (pola sama)

# ---------------- main: build, scan, sign, deploy staging ----------------
build:api:
  stage: build
  image: $BUILDKIT_IMAGE
  tags: [ci-build]
  needs: []
  rules: [*if-main]
  variables: { BUILDKITD_FLAGS: --oci-worker-no-process-sandbox }
  script:
    - scripts/ci/buildkit_build.sh msp-api apps/api "$CI_REGISTRY_IMAGE/msp-api:sha-$CI_COMMIT_SHA" --provenance --metadata meta.json
    - echo "API_IMAGE=$CI_REGISTRY_IMAGE/msp-api@$(scripts/ci/digest_from_meta.sh meta.json)" > build.env
  artifacts: { reports: { dotenv: build.env } }

build:web:
  extends: build:api                                     # sama; menghasilkan WEB_IMAGE
  script:
    - scripts/ci/buildkit_build.sh msp-web apps/web "$CI_REGISTRY_IMAGE/msp-web:sha-$CI_COMMIT_SHA" --provenance --metadata meta.json
    - echo "WEB_IMAGE=$CI_REGISTRY_IMAGE/msp-web@$(scripts/ci/digest_from_meta.sh meta.json)" > build.env

scan:images:
  stage: scan
  needs: [build:api, build:web]
  rules: [*if-main]
  variables: { TRIVY_USERNAME: $CI_REGISTRY_USER, TRIVY_PASSWORD: $CI_REGISTRY_PASSWORD }
  script:
    - trivy image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 "$API_IMAGE"          # QG-27
    - trivy image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 "$WEB_IMAGE"

sign:images:
  stage: scan
  needs: [build:api, build:web, scan:images]
  rules: [*if-main]
  environment: { name: signing, action: prepare }       # akses SOPS_AGE_KEY_SIGNING
  script:
    - scripts/ci/sbom_and_sign.sh "$API_IMAGE" "$WEB_IMAGE"   # syft → cosign sign + attest (--tlog-upload=false)
  artifacts: { paths: [sbom-*.spdx.json], expire_in: 90 days }

deploy:staging:
  stage: deploy-staging
  tags: [deploy]
  needs: [build:api, build:web, sign:images]
  rules: [*if-main]
  resource_group: staging
  interruptible: false
  environment: { name: staging, url: "https://app.staging.<domain>" }
  script:
    - eval "$(ssh-agent -s)" && ssh-add "$DEPLOY_SSH_KEY"
    - export SOPS_AGE_KEY_FILE="$SOPS_AGE_KEY"
    - ansible-playbook -i infra/ansible/inventories/staging infra/ansible/playbooks/deploy.yml
      -e api_image="$API_IMAGE" -e web_image="$WEB_IMAGE" -e app_version="sha-$CI_COMMIT_SHORT_SHA"
      -e '{"traffic_steps": [{"weight": 100, "watch_minutes": 5}]}'

verify:staging:
  stage: verify
  needs: [deploy:staging]
  rules: [*if-main]
  parallel:
    matrix:
      - SUITE: [smoke, e2e-1, e2e-2, e2e-3, e2e-4, schemathesis, tenant-isolation]
  script: [scripts/ci/verify_staging.sh "$SUITE"]                                            # QG-29

mark:staging-verified:
  stage: verify
  needs: [verify:staging]
  rules: [*if-main]
  script: [scripts/ci/commit_status.sh "$CI_COMMIT_SHA" staging-verified success]

release:cut:
  stage: release
  needs: [mark:staging-verified]
  rules:
    - <<: *if-main
      when: manual
  environment: { name: release, action: prepare }       # akses RELEASE_TOKEN
  script:
    - cd tools/release && pnpm install --frozen-lockfile && GITLAB_TOKEN="$RELEASE_TOKEN" pnpm exec semantic-release
    - scripts/release/tag_design_tokens.sh

# ---------------- tag vX.Y.Z: production ----------------
release:resolve:
  stage: release
  needs: []
  rules: [*if-rel]
  script:
    - scripts/release/await_status.sh "$CI_COMMIT_SHA" staging-verified 45m                     # QG-29
    - scripts/release/resolve_digests.sh "$CI_COMMIT_SHA" > release.env                         # API_IMAGE, WEB_IMAGE, BUMP
  artifacts: { reports: { dotenv: release.env } }

release:readiness:
  stage: release
  needs: [release:resolve]
  rules: [*if-rel]
  script:
    - scripts/release/check_open_vulns.sh "$API_IMAGE" "$WEB_IMAGE"                              # QG-19, QG-27
    - '[ "$BUMP" = patch ] || scripts/release/check_rc_evidence.sh "$CI_COMMIT_SHA"'             # QG-26, QG-30, QG-31, QG-35
    - '[ "$BUMP" != major ] || scripts/release/check_release_readiness_issue.sh "$CI_COMMIT_TAG"' # QG-32

deploy:production:
  stage: deploy-production
  tags: [deploy]
  needs: [release:resolve, release:readiness]
  rules:
    - <<: *if-rel
      when: manual                      # tag terlindungi → hanya Maintainer yang bisa menjalankan
      allow_failure: false
  resource_group: production
  interruptible: false
  timeout: 2h
  environment: { name: production, url: "https://app.<domain>" }
  script:
    - scripts/release/four_eyes.sh                                                               # opsional
    - eval "$(ssh-agent -s)" && ssh-add "$DEPLOY_SSH_KEY"
    - export SOPS_AGE_KEY_FILE="$SOPS_AGE_KEY"
    - ansible-playbook -i infra/ansible/inventories/production infra/ansible/playbooks/deploy.yml
      -e api_image="$API_IMAGE" -e web_image="$WEB_IMAGE" -e app_version="$CI_COMMIT_TAG"
      -e '{"traffic_steps": [{"weight": 10, "watch_minutes": 10}, {"weight": 100, "watch_minutes": 20}]}'   # QG-33

# … verify:production (smoke, status production-deployed, tag registry vX.Y.Z, GlitchTip release, notifikasi)
# … rollback:production (manual, Maintainer): ansible-playbook … rollback.yml -e to=previous
```

### 17.2 Playbook deploy (cuplikan)

```yaml
# infra/ansible/playbooks/deploy.yml (ILUSTRATIF)
- name: Deploy MSP blue/green
  hosts: app
  serial: 1
  vars:
    msp_root: /opt/msp
    secrets_root: /run/msp                     # tmpfs (ADR-22)
  pre_tasks:
    - name: Verifikasi tanda tangan & SBOM (QG-28)
      delegate_to: localhost
      run_once: true
      changed_when: false
      ansible.builtin.command: "{{ playbook_dir }}/../../../scripts/deploy/verify_image.sh {{ item }}"
      loop: ["{{ api_image }}", "{{ web_image }}"]
    - name: Baca warna aktif
      ansible.builtin.slurp: { src: "{{ msp_root }}/active_color" }
      register: active
    - ansible.builtin.set_fact:
        color_old: "{{ active.content | b64decode | trim }}"
        color_new: "{{ 'green' if (active.content | b64decode | trim) == 'blue' else 'blue' }}"

  tasks:
    - block:
        - name: Secret SOPS → tmpfs (file 0400, tidak lewat env var)
          ansible.builtin.copy:
            content: "{{ item.value }}"
            dest: "{{ secrets_root }}/{{ color_new }}/{{ item.key }}"
            mode: "0400"
          loop: "{{ lookup('community.sops.sops', inventory_dir ~ '/secrets/app.sops.yaml') | from_yaml | dict2items }}"
          no_log: true

        - name: Render compose warna baru (image by digest)
          ansible.builtin.template: { src: compose.app.yml.j2, dest: "{{ msp_root }}/{{ color_new }}/compose.yml", mode: "0644" }

        - name: Pull image
          community.docker.docker_compose_v2_pull: { project_src: "{{ msp_root }}/{{ color_new }}" }

        - name: Migrasi DB (one-off, expand-only, advisory lock)
          run_once: true
          community.docker.docker_container:
            name: "msp-migrate-{{ app_version }}"
            image: "{{ api_image }}"
            command: ["alembic", "upgrade", "head"]
            networks: [{ name: msp-backend }]
            mounts:
              - { type: bind, read_only: true, source: "{{ secrets_root }}/{{ color_new }}/database_url_owner", target: /run/secrets/database_url }
            detach: false
            cleanup: true

        - name: Start warna baru (worker, api, web) dan tunggu healthy
          community.docker.docker_compose_v2:
            project_src: "{{ msp_root }}/{{ color_new }}"
            project_name: "msp-{{ color_new }}"
            state: present
            wait: true
            wait_timeout: 300

        - name: Drain worker warna lama (SIGTERM, stop_grace_period per pool)
          ansible.builtin.command: "docker compose -p msp-{{ color_old }} stop {{ worker_services | join(' ') }}"
          when: color_old != 'none'

        - name: Canary → promote, dipantau Prometheus (gagal = rescue)
          ansible.builtin.include_tasks: tasks/shift_traffic.yml     # render dynamic/msp.yml + watch_canary.py
          loop: "{{ traffic_steps }}"
          loop_control: { loop_var: step }

        - name: Render ulang Traefik tanpa warna lama, lalu hentikan api/web warna lama
          ansible.builtin.include_tasks: tasks/finalize_color.yml

        - ansible.builtin.copy: { content: "{{ color_new }}\n", dest: "{{ msp_root }}/active_color" }

      rescue:
        - name: Kembalikan 100% traffic ke warna lama
          ansible.builtin.template: { src: traefik-msp.yml.j2, dest: /opt/traefik/dynamic/msp.yml }
          vars: { traffic: [{ color: "{{ color_old }}", weight: 100 }] }
        - name: Start ulang worker warna lama
          ansible.builtin.command: "docker compose -p msp-{{ color_old }} start {{ worker_services | join(' ') }}"
        - name: Hentikan warna baru
          community.docker.docker_compose_v2: { project_src: "{{ msp_root }}/{{ color_new }}", state: absent }
        - ansible.builtin.fail: { msg: "Deploy {{ app_version }} gagal → rollback otomatis ke {{ color_old }}" }
```

### 17.3 Traefik blue/green & label Compose

```yaml
# /opt/traefik/dynamic/msp.yml — dirender Ansible (file provider, watch: true)
http:
  routers:
    msp-api:
      rule: "Host(`app.<domain>`) && PathPrefix(`/api`)"     # ADR-25: same-origin
      entryPoints: [websecure]
      service: msp-api
      middlewares: [api-strip-prefix, security-headers]
      tls: { certResolver: letsencrypt }
    msp-web:
      rule: "Host(`app.<domain>`)"
      entryPoints: [websecure]
      service: msp-web
      middlewares: [security-headers]
      tls: { certResolver: letsencrypt }
  services:
    msp-api:
      weighted:
        healthCheck: {}
        services:
          - { name: api-blue@docker,  weight: 90 }      # langkah canary; hanya warna yang hidup boleh dirujuk
          - { name: api-green@docker, weight: 10 }
    msp-web:
      weighted:
        healthCheck: {}
        services:
          - { name: web-blue@docker,  weight: 90 }
          - { name: web-green@docker, weight: 10 }
```

```yaml
# compose.app.yml.j2 (cuplikan) — satu project per warna: msp-blue / msp-green
services:
  api:
    image: "{{ api_image }}"
    command: ["uvicorn", "msp.app.api:create_app", "--factory", "--host", "0.0.0.0", "--port", "8000"]
    deploy: { replicas: 2 }
    environment: { MSP_ENV: "{{ msp_env }}", MSP_APP_VERSION: "{{ app_version }}", MSP_COLOR: "{{ color_new }}" }
    secrets: [database_url, redis_url, webhook_keyring]
    networks: [edge, backend]
    labels:
      - traefik.enable=true
      - traefik.http.services.api-{{ color_new }}.loadbalancer.server.port=8000
      - traefik.http.services.api-{{ color_new }}.loadbalancer.healthcheck.path=/readyz
      - traefik.http.services.api-{{ color_new }}.loadbalancer.healthcheck.interval=5s
  worker-pipeline:
    image: "{{ api_image }}"
    command: ["dramatiq", "msp.app.worker", "--queues", "pipeline"]
    stop_grace_period: 15m          # selesaikan pesan in-flight; job panjang resumable (ADR-06)
    secrets: [database_url_pipeline, redis_url]
    networks: [backend]
secrets:
  database_url: { file: "/run/msp/{{ color_new }}/database_url" }
  # … redis_url, webhook_keyring, database_url_pipeline
```

---

## 18. Checklist bootstrap

Urutan menyiapkan CI/CD dari nol. Setiap langkah menghasilkan artefak di repo, kecuali yang ditandai *(manual)*.

1. *(manual)* Kontrak dedicated server di DC Indonesia (Q-CI-04): `ops-01`, `ci-01`, `stg-01`, `app-01`, `db-01`, `db-02`, `stor-01`; VLAN privat; IP publik untuk `app-01` dan `ops-01`; penyimpanan off-site S3-compatible di lokasi lain di Indonesia. Pasang OS dengan LUKS.
2. *(manual)* Buat kunci age per environment + kunci Maintainer (salinan offline), kunci SSH deploy, dan key pair cosign (`cosign generate-key-pair`). Commit kunci publik ke `infra/keys/`.
3. Commit `infra/ansible` (inventory, role `base`, `users`, `firewall`, `wireguard`, `luks_unlock`) + `.sops.yaml`. Jalankan `site.yml` pertama dari laptop Maintainer via IP publik **sekali saja**, lalu tutup SSH publik. Selanjutnya semua akses lewat WireGuard.
4. Role `gitlab` di `ops-01` (Omnibus, ACME DNS-01 atau CA internal, registry aktif, sign-up nonaktif, 2FA wajib, backup harian) + role `monitoring_stack`.
5. Buat grup `msp`, proyek `MesinProspek` (private), tim/subgroup untuk CODEOWNERS (`web`, `backend`, `data-intel`, `platform`, `design`, `product`, `qa`, `security`, `api-owners`). Pindahkan repo (Q-CI-01).
6. **Verifikasi tier** fitur CE yang dipakai (§1.3): protected runners, environment-scoped variables, server hooks, cleanup policies, project access tokens.
7. Role `gitlab_runner` di `ci-01` (runner `ci-docker` & `ci-build`, cache S3 ke MinIO lokal, `allowed_images`) dan di `ops-01` (runner `deploy`, *protected*, tag `deploy`).
8. `scripts/bootstrap/gitlab_settings.py`: protected branch/tag (§2.5), fast-forward + squash, template squash, merge checks, cleanup policy registry, server hook gitleaks.
9. CI/CD variables (§12.2) dengan scope environment. Bot user: rilis (Maintainer), Renovate, `mr:gate` (Reporter). Token berkedaluwarsa.
10. Commit fondasi repo: `.gitlab/CODEOWNERS`, `.gitlab/merge_request_templates/Default.md`, `.editorconfig`, `.env.example`, `.gitignore`, `commitlint.config.js`, `tools/release/` (semantic-release), `renovate.json`, `.squawk.toml`, `.spectral.yaml`, `.semgrep/`, `.markdownlint-cli2.yaml`, `lychee.toml`, `ci/images/tools/Dockerfile`.
11. Pipeline pertama: build image `ci-tools`, lalu `.gitlab-ci.yml` MR (lint, test, security, `mr:gate`). Uji pada MR percobaan, termasuk pemblokiran merge oleh `mr:gate`.
12. Provision `stg-01` (role `docker`, `traefik`, `msp_app`, Postgres/Redis/MinIO staging) → job `main`: build, scan, sign, `deploy:staging`, verifikasi, `staging-verified`.
13. Provision production: `db-01`/`db-02` (role `postgres`, replikasi, PgBouncer, `pgbackrest` ke `stor-01` + off-site), `stor-01` (MinIO + replikasi), `app-01` (Traefik, `msp_app`), `monitoring_agent` di semua host.
14. `release:cut` + pipeline tag `v*`. Rilis uji `v0.1.0` ke production (halaman health saja), termasuk uji rollback dan uji rescue canary (paksa gagal).
15. `ci/infra.yml` (check/diff + apply staging + tag `infra-*`) dan schedule `drift`, `backup-check`.
16. Schedule `nightly` bertahap: regresi penuh → ZAP → k6 → data quality → rescan → Lighthouse. Lalu `renovate` dan `weekly`.
17. Drill restore pertama secara manual di `dr-01`, baru kemudian dijadwalkan bulanan.
18. Flag registry + kill switch + laporan flag hygiene. Unit `msp-rehydrate` + trigger token.
19. Observability: anotasi deploy, release GlitchTip, dashboard SLO, DORA, CI health, probe eksternal + watchdog.
20. Runbook (deploy, rollback, failover DB, rehydrate, incident + hook PDP, drill) dan latihan on-call sebelum closed beta.

---

## 19. Pertanyaan terbuka

ID `Q-CI-nn` bersifat lokal di dokumen ini. Setelah diputuskan, keputusan dipindahkan ke register 00 §7.

| ID | Pertanyaan | Default sementara | Dampak |
|---|---|---|---|
| Q-CI-01 | Apakah source code **pindah penuh** dari GitHub ke GitLab, atau GitHub tetap ada sebagai mirror? | **GitLab primary.** Mirror push ke GitHub opsional (read-only, tanpa CI) bila dibutuhkan untuk akses pihak luar | §2, §15, README |
| Q-CI-02 | Tetap di GitLab CE, atau beli GitLab Premium (approval rules, Code Owners, merge trains, protected environments)? | CE + mitigasi §1.3. Ditinjau ulang saat tim ≥ 6 engineer | §2.3, §2.6, §6.2 |
| Q-CI-03 | Alternatif CI yang lebih ringan (Forgejo + Actions, Woodpecker) bila beban operasional GitLab terlalu besar | GitLab CE | §1.4 |
| Q-CI-04 | Provider DC dan lokasi off-site (keduanya di Indonesia, kota/provider berbeda). SLA penggantian hardware dan ketersediaan `dr-01` (server cadangan vs sewa per jam) | Jakarta (utama) + lokasi kedua di Indonesia | RTO skenario kehilangan DC (§11.6), D-14 |
| Q-CI-05 | Kapan `app-02` + VIP (keepalived/VRRP atau LB provider) ditambahkan? Kapan failover DB otomatis (Patroni)? | `app-02` sebelum paid GA. Patroni saat ada node DB ketiga | NFR-AVAIL-01 (99,5%), §6.3 |
| Q-CI-06 | Kanal alert on-call (chat/WhatsApp/telepon) dan lokasi probe eksternal | Chat + email. Probe di VPS lokasi lain | §16.1 |
| Q-CI-07 | Siapa Maintainer (berhak deploy production) dan rotasi on-call di luar jam kerja? | Tech lead + 1 engineer senior | §2.5, §6.2, §14.5 |
| Q-CI-08 | Nama domain dan penyedia DNS (DNS-01 untuk GitLab internal, Let's Encrypt untuk `app.<domain>`) | `<domain>` placeholder | §3.1, §17.3 |
| Q-CI-09 | Lisensi dataset: apakah subset boleh dipakai di staging dan eval AI (Q-05)? | Hanya sintetis sampai dikonfirmasi | §10.2, QG-26 penuh |
| Q-CI-10 | Apakah SOPS + GitLab variables cukup untuk NFR-SEC-04 ("secret hanya di secret manager"), atau perlu OpenBao/Vault? | Cukup untuk MVP. OpenBao dievaluasi di V1 (ADR-22) | §12 |
| Q-CI-11 | Threshold QG-34 dan alarm canary (DLQ, latency) awal | Nilai default §6.3 & §10.2 | Kalibrasi setelah Q-01/Q-02 + 4 minggu data beta |
