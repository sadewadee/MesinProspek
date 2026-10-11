# 06 — CI/CD (lean)

| Field | Value |
|---|---|
| Versi | 0.3 (lean) · 2026-10-11. Sumber data = DB gmaps read-only (D-31), kiriman mingguan ramping founder dimuat ke `gmaps-db` (D-35, D-37) |
| Basis | [README](README.md) (stack F1, kosakata), [PRD](01-prd.md) NFR-03/04/05/09 + §12 + §13, D-18, D-19, D-22, D-26, D-29, D-31, D-35, D-37 ([00 §7](00-analisis-dokumen-awal.md)). Terkait: [04](04-technical-design.md) §2.1 (kapasitas), [05](05-test-strategy.md) (isi test), [08](08-engineering-standards.md) (`make`, migrasi), [09](09-runbook-operasional.md) (operasi) |

> KISS: satu server, GitLab CE + dua runner (CI dan deploy) + Docker Compose + skrip shell. Contoh YAML dan skrip bersifat ilustratif. Semua image dan tool di-pin ke versi eksak; angka versi di contoh hanya contoh.

## 1. Prinsip

- **Trunk-based.** Satu branch `main`. Branch kerja berumur pendek, MR kecil (08 STD-GIT-01).
- **Build once, promote.** Image dibangun sekali per commit `main` dan di-tag git SHA. Staging dan production menjalankan image yang sama; yang berbeda hanya `.env` dan batas resource.
- **Migrasi dulu.** `msp migrate` jalan sebelum container baru start. Setiap migrasi kompatibel dengan versi aplikasi sebelumnya.
- **Rollback = deploy tag sebelumnya.** Skema DB tidak di-rollback.
- **Founder menyetujui production** (D-22). Hanya founder yang membuat tag `v*` dan menjalankan deploy production.
- **Production didahulukan.** CI dan staging berbagi server dengan production, jadi batas resource-nya lebih kecil, build dibuat ringan (cache), dan keduanya yang dihentikan lebih dulu saat server sesak (09 AL-09).

## 2. Server & environment

Satu dedicated server di Indonesia untuk semuanya (D-19, D-26). Direktori Compose mengikuti README.

| Environment | Tempat | Deploy | Data |
|---|---|---|---|
| `local` | laptop | `docker compose` (Postgres + PostGIS, Kratos, Mailpit) | Fixture sintetis (05), termasuk DB sumber fixture sebagai database kedua di Postgres yang sama |
| `staging` | server, `/opt/msp-staging` | Otomatis tiap merge ke `main`, dan pada tag sebelum production | Data bisnis nyata dari `gmaps-db` (kiriman DB gmaps terbaru, read-only) + GeoAPI, akun test. Tanpa data customer |
| `production` | server, `/opt/msp` | Manual, tag `vX.Y.Z`, founder | Data nyata |

- **`/opt/ops`** (file dari `deploy/ops/`): `caddy`, `gitlab` (+ container registry; git hanya lewat HTTPS), `gitlab-runner` (runner `ci`), `prometheus`, `alertmanager`, `grafana` (login admin bawaan lewat HTTPS, tanpa OAuth), `node-exporter`, `gmaps-db`. Stack ini di-update manual lewat SSH, karena pipeline tidak me-restart GitLab yang sedang menjalankannya. Caddy sengaja ada di sini, bukan di project aplikasi, agar GitLab tetap bisa diakses sebelum production ter-deploy (bootstrap, RB-11). Uptime dipantau oleh layanan eksternal (09 AL-01).
- **DB gmaps (D-31, D-35, D-37).** Scraper dan DB aslinya dijalankan founder terpisah; MesinProspek tidak melakukan scraping dan tidak punya koneksi jaringan ke server founder. Founder menyerahkan **kiriman mingguan ramping** (file CSV gzip dari pernyataan `export` di `source.sql`: hanya Indonesia, hanya kolom yang dipakai import, hanya baris baru; 04 §5.1), dan ops memuatnya ke service **`gmaps-db`** di `/opt/ops` (09 RB-15). `gmaps-db` hanya berisi kiriman terbaru (riwayat ada di DB MSP) dan dibaca bersama oleh staging dan production, read-only: `worker` keduanya bergabung ke network `gmaps` dan login sebagai role `msp_source_ro` (hanya `SELECT`; `MSP_SOURCE_DATABASE_URL`, §7). `msp tool import` berjalan lewat `docker compose run --rm worker`, jadi ikut network yang sama. `gmaps-db` tidak mempublikasikan port dan tidak di-backup (§8); file kiriman terbaru disimpan di `/opt/ops/gmaps-dump/`. Konfigurasinya di §5.
- **Edge.** Hanya `caddy` yang mempublikasikan port (80/443), dan semua domain lewat Caddy ini (§5). Docker melewati aturan `ufw` untuk port yang dipublish, jadi service lain tidak memakai `ports:`.
- **Network.** Stack `/opt/ops` membuat network Docker `edge-production` dan `edge-staging`. `web`, `api`, dan `kratos` tiap environment bergabung ke `edge-${MSP_ENV}`, sehingga `caddy` dan `prometheus` menjangkaunya lewat nama container (`msp-api-1:9090`, `msp-staging-api-1:9090`). Network dipisah per environment karena kedua project memakai nama service yang sama. Stack `/opt/ops` juga membuat network `gmaps`, yang hanya diikuti `gmaps-db` dan `worker` kedua environment; `worker` menjangkaunya sebagai `gmaps-db:5432`. `postgres` hanya ada di network project masing-masing.
- **Runner `ci`**: container `gitlab-runner` di `/opt/ops`, Docker executor, tag `ci`, `concurrent = 1`, dan `privileged = true` (volume `/certs/client`, `/cache`) agar job build dan E2E bisa memakai `docker:dind`. Batas job diatur lewat `cpus`/`memory` dan `service_cpus`/`service_memory` di `config.toml`. Privileged setara root di server, termasuk production. Risiko ini diterima di F1 karena runner hanya menjalankan MR tim, dan GitLab di server yang sama memang sudah memegang secret production. Pemisahan ada di §11.
- **Runner `deploy`**: paket `gitlab-runner` di host (bukan container), shell executor, berjalan sebagai user `deploy`, tag `deploy`, `concurrent = 1`. Runner ini ditandai *Protected*, sehingga hanya menjalankan job dari branch/tag terproteksi, yaitu `deploy/deploy.sh` dan `deploy/backup.sh`. Kedua runner hanya terdaftar untuk project ini. Deploy tidak memakai SSH.
- **Batas resource** wajib untuk setiap container (`deploy.resources.limits`). Total per kelompok ada di 04 §2.1; pembagian awal per service di bawah ini disesuaikan dengan data Grafana. Batas staging ada di `deploy/docker-compose.staging.yml`. File ini dipakai lewat `COMPOSE_FILE=compose.yaml:compose.staging.yaml` di `.env` staging.

| Service | Production | Staging |
|---|---|---|
| `postgres` | 12 CPU · 32 GB | 1,5 · 6 GB |
| `worker` | 4 · 6 GB | 1 · 3 GB |
| `api` | 2 · 3 GB | 0,5 · 1 GB |
| `web` | 1 · 2 GB | 0,5 · 1,5 GB |
| `kratos` | 0,5 · 0,5 GB | 0,5 · 0,5 GB |

Di `/opt/ops`: `caddy` 0,5 · 0,5 GB, `gitlab` 4 · 8 GB, job CI 2 · 4 GB + service `dind` 2 · 4 GB, monitoring 1 · 2 GB, dan `gmaps-db` 2 · 4 GB.

## 3. Alur Git

1. Agent membuat branch `wp/<nn>-<slug>` dari `main` (07), lalu membuka MR. Judul MR mengikuti Conventional Commits dengan WP sebagai scope (`feat(WP-14): …`, `fix(WP-21): …`, `!` untuk breaking). Deskripsi MR menyebut ID PRD (08 STD-GIT-02/04).
2. `check` hijau → orchestrator me-review → **squash merge** ke `main`. Judul MR menjadi pesan commit dan bahan changelog.
3. `main` → `build` → deploy `staging` otomatis. Founder membuat tag `vX.Y.Z` → E2E J-2..J-7 di staging → founder menjalankan deploy production.

| Aktor | Role GitLab | Boleh |
|---|---|---|
| Agent AI (akun bot) | Developer | Push branch, buka MR. Tanpa SSH (09 §2) |
| Orchestrator (Claude) | Developer | Review dan merge MR ke `main` |
| Founder | Maintainer (satu-satunya) | Membuat tag `v*`, menjalankan `deploy-production`, mengelola variabel CI |

- **Protected branch `main`**: push = *No one*, merge = *Developers + Maintainers*, "Pipelines must succeed", "All threads must be resolved", squash wajib. **Protected tag `v*`**: create = *Maintainers*. Job manual di pipeline tag hanya bisa dijalankan oleh yang boleh membuat tag itu, yaitu founder.
- **Batas GitLab CE.** Approval wajib, CODEOWNERS, dan protected environment adalah fitur berbayar. Karena itu ada dua aturan proses: (1) agent tidak me-merge MR-nya sendiri; (2) MR yang mengubah `.gitlab-ci.yml`, `deploy/`, atau `scripts/ci/` hanya di-merge founder. Alasannya, job apa pun di `main` bisa membaca variabel terproteksi dan berjalan sebagai shell di server lewat runner `deploy`.

## 4. Pipeline (`.gitlab-ci.yml`)

| Stage | Kapan | Isi |
|---|---|---|
| `check` | MR, `main` | `make generate` + `git diff --exit-code`, `make lint`, `make test` (08 §0), `govulncheck`, cek MR. Job `e2e`: stack Compose + Playwright Chromium (05). Test import membaca DB sumber fixture (05): database kedua di service `postgres` yang sama, tanpa service tambahan |
| `build` | `main` | Image `api` & `web` → registry GitLab dengan tag SHA (cache inline dari image `:main`). Laporan `trivy image` HIGH/CRITICAL (tidak memblokir) |
| `deploy-staging` | `main` otomatis; tag `v*` | Runner `deploy`: `deploy/deploy.sh` di `/opt/msp-staging`. Lalu `smoke-staging` (runner `ci`): J-2 dengan akun test yang sudah terverifikasi (`main`), atau J-2..J-6 di semua browser dengan akun yang sama (tag). J-1 hanya di job `e2e` (Mailpit) |
| `deploy-production` | tag `vX.Y.Z`, manual, Maintainer | Runner `deploy`: `deploy/deploy.sh` di `/opt/msp`. Smoke manual per 09 RB-01 |
| `rollback` | manual, Maintainer | Retry job `deploy-production` di pipeline tag sebelumnya (09 RB-02). Langkahnya sama dengan deploy, jadi tidak perlu job terpisah (DRY) |
| *schedule* | harian 01:00 WIB | Job `backup` (§8): pgBackRest + GitLab |

```yaml
# .gitlab-ci.yml (ilustratif)
stages: [check, build, deploy-staging, deploy-production]
workflow:
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event" || $CI_PIPELINE_SOURCE == "schedule"
    - if: $CI_COMMIT_BRANCH == "main" || $CI_COMMIT_TAG =~ /^v\d+\.\d+\.\d+$/
default: { tags: [ci] }                       # runner Docker; job deploy memakai tag `deploy`
variables:
  API_IMAGE: $CI_REGISTRY_IMAGE/api:$CI_COMMIT_SHA     # = <registry>/msp/api:<git-sha>
  WEB_IMAGE: $CI_REGISTRY_IMAGE/web:$CI_COMMIT_SHA
.mr-main:
  stage: check
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
    - if: $CI_COMMIT_BRANCH == "main" && $CI_PIPELINE_SOURCE == "push"
.main: { rules: [{ if: $CI_COMMIT_BRANCH == "main" && $CI_PIPELINE_SOURCE == "push" }] }
check:
  extends: .mr-main
  image: golang:1.25-bookworm                 # tool Go via `go tool` (go.mod), tool JS via pnpm-lock
  services: [{ name: "postgis/postgis:16-3.5", alias: postgres }]
  variables:
    POSTGRES_PASSWORD: test
    MSP_DATABASE_URL: "postgres://postgres:test@postgres:5432/postgres?sslmode=disable"
    # DB sumber fixture (05): database kedua di service yang sama, dibuat & di-seed oleh setup test
    MSP_SOURCE_DATABASE_URL: "postgres://postgres:test@postgres:5432/gmaps_fixture?sslmode=disable"
    GOMODCACHE: $CI_PROJECT_DIR/.cache/go-mod
    GOCACHE: $CI_PROJECT_DIR/.cache/go-build
  cache: { key: { files: [apps/api/go.sum, pnpm-lock.yaml] }, paths: [.cache/, .pnpm-store/] }
  before_script:                              # .nvmrc = versi eksak Node, mis. 24.11.0
    - curl -fsSL "https://nodejs.org/dist/v$(cat .nvmrc)/node-v$(cat .nvmrc)-linux-x64.tar.gz" | tar -xz -C /usr/local --strip-components=1
    - corepack enable && pnpm install --frozen-lockfile --store-dir .pnpm-store && git fetch -q origin main
  script:
    - make generate && git diff --exit-code   # oapi-codegen, client TS, sqlc, tokens.css tidak basi
    - make lint                               # termasuk gitleaks (08 §0)
    - make test                               # go test -race + PostGIS, gate coverage, Vitest
    - (cd apps/api && go tool govulncheck ./...)
    - if [ -n "$CI_MERGE_REQUEST_IID" ]; then scripts/ci/mr-check.sh; fi   # judul/deskripsi MR, migrasi lama
e2e:
  extends: .mr-main
  image: docker:28
  services: ["docker:28-dind"]
  # e2e.yml: env test (termasuk DB sumber fixture di `postgres`), network `edge` dan `gmaps` tidak external, cache_from image :main, service `playwright`
  script: [docker compose -f deploy/docker-compose.prod.yml -f deploy/docker-compose.e2e.yml up --build --exit-code-from playwright]
build:
  extends: .main
  stage: build
  image: docker:28
  services: ["docker:28-dind"]
  script:
    - echo "$CI_REGISTRY_PASSWORD" | docker login -u "$CI_REGISTRY_USER" --password-stdin "$CI_REGISTRY"
    - |
      for app in api web; do                  # cache inline dari :main → build kecil dan cepat
        docker build -f apps/$app/Dockerfile --cache-from "$CI_REGISTRY_IMAGE/$app:main" --build-arg BUILDKIT_INLINE_CACHE=1 \
          -t "$CI_REGISTRY_IMAGE/$app:$CI_COMMIT_SHA" -t "$CI_REGISTRY_IMAGE/$app:main" .
        docker push --all-tags "$CI_REGISTRY_IMAGE/$app"
      done
trivy:                                        # laporan saja; dibaca founder sebelum tag rilis
  extends: .main
  stage: build
  needs: [build]
  image: { name: "aquasec/trivy:0.67.0", entrypoint: [""] }
  variables: { TRIVY_USERNAME: $CI_REGISTRY_USER, TRIVY_PASSWORD: $CI_REGISTRY_PASSWORD }
  script:
    - for i in "$API_IMAGE" "$WEB_IMAGE"; do trivy image --severity HIGH,CRITICAL --ignore-unfixed "$i"; done
.deploy:
  tags: [deploy]                              # shell di server, hanya ref terproteksi
  script: [deploy/deploy.sh "$CI_COMMIT_SHA"]
deploy-staging:
  extends: .deploy
  stage: deploy-staging
  rules:
    - if: $CI_COMMIT_BRANCH == "main" && $CI_PIPELINE_SOURCE == "push"
    - if: $CI_COMMIT_TAG =~ /^v\d+\.\d+\.\d+$/
  environment: staging
smoke-staging:
  stage: deploy-staging
  needs: [deploy-staging]
  image: mcr.microsoft.com/playwright:v1.56.0-noble   # versi = @playwright/test di lockfile
  rules:
    - if: $CI_COMMIT_BRANCH == "main" && $CI_PIPELINE_SOURCE == "push"
      variables: { E2E_ARGS: "--project=chromium --grep @smoke" }   # J-2; J-1 butuh Mailpit (hanya di stack test)
    - if: $CI_COMMIT_TAG =~ /^v\d+\.\d+\.\d+$/
      variables: { E2E_ARGS: "--grep-invert @ci-only" }              # J-2..J-7, semua browser
  environment: { name: staging, action: verify }                     # variabel staging: URL + akun test
  script:
    - corepack enable && pnpm install --frozen-lockfile
    - pnpm --filter ./apps/web exec playwright test $E2E_ARGS
deploy-production:
  extends: .deploy
  stage: deploy-production
  rules: [{ if: $CI_COMMIT_TAG =~ /^v\d+\.\d+\.\d+$/, when: manual }]
  environment: production
backup:                                       # pipeline schedule harian
  tags: [deploy]
  stage: check
  rules: [{ if: $CI_PIPELINE_SOURCE == "schedule" }]
  environment: { name: production, action: access }
  script: [deploy/backup.sh]
```

**`deploy/deploy.sh <git-sha>`** dijalankan runner `deploy` langsung di server, untuk staging, production, dan rollback. Variabel per environment: `DEPLOY_DIR` (`/opt/msp-staging` atau `/opt/msp`) dan `MSP_APP_HOST`.
1. Tulis `.env`: setiap key di `deploy/.env.example` diisi dari variabel CI, ditambah `MSP_IMAGE_REPO=$CI_REGISTRY_IMAGE` dan `MSP_IMAGE_TAG=<git-sha>`. Salin ke `$DEPLOY_DIR`: `.env` (chmod 600), `docker-compose.prod.yml` sebagai `compose.yaml` (agar perintah 09 jalan tanpa `-f`), `docker-compose.staging.yml` sebagai `compose.staging.yaml`, `kratos/`, dan `postgres/`.
2. `cd $DEPLOY_DIR`, `docker login` dengan deploy token `read_registry`, lalu `docker compose pull --ignore-buildable`.
3. `docker compose run --rm worker msp migrate`, lalu `docker compose run --rm kratos migrate sql -e --yes`. Bila gagal, deploy berhenti dan versi lama tetap jalan.
4. `docker compose up -d --build --wait --wait-timeout 300`. `--build` hanya membangun ulang `postgres` bila Dockerfile-nya berubah.
5. `curl -fsS https://$MSP_APP_HOST/readyz`. Bila gagal, job merah dan founder memutuskan rollback (09 RB-02).

- Mengganti container menimbulkan jeda beberapa detik. Ini diterima untuk NFR-03 (99%); zero-downtime ada di F3.
- **Build once untuk web.** Jangan memakai `NEXT_PUBLIC_*` yang nilainya berbeda per environment, karena nilainya ter-inline saat build. API dipanggil lewat path relatif `/api/v1`.
- Di MR, image dibangun oleh job `e2e` tanpa push; push ke registry hanya dari `main`. Pipeline tag tidak membangun ulang image: `pull` gagal bila image SHA tidak ada.

## 5. Docker Compose & Caddy

```yaml
# deploy/docker-compose.prod.yml (ringkas). Dipakai staging & production. Semua service: restart: unless-stopped
# Setiap service: deploy.resources.limits = kolom production §2; staging menimpanya lewat docker-compose.staging.yml
x-app: &app
  image: ${MSP_IMAGE_REPO}/api:${MSP_IMAGE_TAG}       # api & worker = image yang sama
  env_file: .env
  depends_on: { postgres: { condition: service_healthy } }
services:
  web:                                                # tanpa env_file: web tidak memegang secret
    image: ${MSP_IMAGE_REPO}/web:${MSP_IMAGE_TAG}
    networks: [default, edge]
    healthcheck: { test: ["CMD", "wget", "-qO-", "http://localhost:3000/"], interval: 5s }
  api: { <<: *app, command: ["msp", "api"], networks: [default, edge], healthcheck: { test: ["CMD", "wget", "-qO-", "http://localhost:8080/readyz"], interval: 5s } }   # :9090 metrik → Prometheus
  worker: { <<: *app, command: ["msp", "worker"], networks: [default, gmaps], stop_grace_period: 60s }   # job River sempat selesai; gmaps → gmaps-db
  kratos: { image: "oryd/kratos:v1.3.1", command: ["serve", "-c", "/etc/kratos/kratos.yml"], env_file: .env, volumes: ["./kratos:/etc/kratos:ro"], networks: [default, edge] }
  postgres:
    build: ./postgres                                 # FROM postgis/postgis:16-3.5 + apt pgbackrest
    image: msp-postgres:16
    env_file: .env                                    # POSTGRES_PASSWORD, PGBACKREST_* (prod)
    volumes: ["pgdata:/var/lib/postgresql/data"]
    healthcheck: { test: ["CMD", "pg_isready", "-U", "postgres"], interval: 5s }
networks:
  edge: { name: "edge-${MSP_ENV}", external: true }   # dibuat stack /opt/ops (§2)
  gmaps: { name: gmaps, external: true }              # dibuat stack /opt/ops, berisi gmaps-db (§2)
volumes: { pgdata: {} }
```

```yaml
# deploy/ops/compose.yaml (cuplikan gmaps-db; service ops lain tidak ditampilkan)
# deploy.resources.limits = nilai /opt/ops di §2
services:
  gmaps-db:                                           # kiriman DB gmaps terbaru (D-37); tanpa ports:
    image: postgres:16                                # kiriman CSV, tidak bergantung versi Postgres founder
    environment: { POSTGRES_DB: gmaps, POSTGRES_PASSWORD: "${MSP_GMAPS_DB_PASSWORD}" }   # superuser, /opt/ops/.env
    volumes: ["gmaps-data:/var/lib/postgresql/data", "./gmaps-dump:/dump:ro"]           # file kiriman (09 RB-15)
    networks: [gmaps]
    healthcheck: { test: ["CMD", "pg_isready", "-U", "postgres"], interval: 5s }
networks:
  gmaps: { name: gmaps }                              # worker production & staging bergabung (external)
volumes: { gmaps-data: {} }
```

```caddyfile
# deploy/ops/Caddyfile. Satu Caddy untuk semua domain; target = nama container (project = nama direktori README)
(app) {                                  # {args[0]} = msp (production) atau msp-staging
	@api path /api/* /healthz /readyz
	reverse_proxy @api {args[0]}-api-1:8080
	handle_path /.ory/* {
		reverse_proxy {args[0]}-kratos-1:4433   # public base_url = https://<host>/.ory/; admin 4434 tidak dirutekan
	}
	reverse_proxy {args[0]}-web-1:3000      # metrik :9090 tidak dirutekan
}
app.{$MSP_DOMAIN} {
	import app msp
}
staging.{$MSP_DOMAIN} {
	import app msp-staging
}
gitlab.{$MSP_DOMAIN} {
	reverse_proxy gitlab:80                 # GitLab: listen_https = false, TLS di Caddy
}
registry.{$MSP_DOMAIN} {
	reverse_proxy gitlab:5050
}
grafana.{$MSP_DOMAIN} {
	@luar not remote_ip {$MSP_ADMIN_IPS}    # allowlist = IP admin SSH (09 §2)
	respond @luar 403
	reverse_proxy grafana:3000
}
```

Image `api` berbasis Alpine (ada `wget` untuk healthcheck). Image `web` memakai Next.js `output: "standalone"`.

## 6. Migrasi DB

Migrasi memakai goose di `apps/api/db/migrations`, di-embed di binary, dan dijalankan dengan `docker compose run --rm worker msp migrate`. Aturan lengkapnya di 08 STD-DB-01.
- **Backward-compatible saja**: tambah → backfill → pindah pemakaian → hapus di rilis berikutnya. Contoh rename kolom: tambah kolom baru, tulis ke keduanya, baca kolom baru, lalu drop kolom lama di rilis berikutnya.
- **Migrasi yang sudah ada di `main` tidak pernah diedit.** Perbaikan = migrasi baru. `scripts/ci/mr-check.sh` menolak perubahan file migrasi lama.
- **Migrasi jalan sebelum container baru start** (§4 langkah 3), satu transaksi per migrasi. Bila gagal, aplikasi lama tetap jalan. **Tanpa down migration** di staging/production: rollback hanya mengganti aplikasi, dan `msp migrate` versi lama tidak menjalankan apa pun.
- **Perubahan berat dipecah**: `CREATE INDEX CONCURRENTLY` (`-- +goose NO TRANSACTION`) untuk tabel besar. Backfill jutaan baris lewat job atau `msp tool`, bukan di dalam migrasi.

## 7. Secret & konfigurasi

- `deploy/.env.example` adalah satu-satunya daftar key (DRY), berisi nilai dummy dan komentar: `MSP_*`, secret Kratos, `POSTGRES_PASSWORD`, `PGBACKREST_*`. Isinya hanya secret dan pengaturan proses (04 §14).
- **Pengaturan produk tidak ada di `.env`.** Provider, model, dan base URL AI (termasuk fallback), plan, serta parameter estimasi diubah dari halaman ops, tercatat di audit, dan berlaku tanpa deploy (D-29, D-32, 04). Dari AI, yang ada di `.env` hanya API key-nya: `MSP_ANTHROPIC_API_KEY` dan `MSP_LLM_FALLBACK_API_KEY`.
- **Variabel CI GitLab**, di-scope per environment (`staging` / `production`): setiap key di atas, **Protected** + **Masked** untuk secret. Juga `DEPLOY_DIR` dan deploy token registry.
- **`MSP_SOURCE_DATABASE_URL`** (kiriman DB gmaps di `gmaps-db`, D-31, D-37) adalah secret seperti key lain: variabel CI Protected + Masked di scope `staging` dan `production`, lalu ditulis ke `.env` masing-masing. Nilainya sama di kedua environment: `postgres://msp_source_ro:<password>@gmaps-db:5432/gmaps`. Role `msp_source_ro` dibuat ops sekali di `gmaps-db` (09 RB-15); rotasi password di 09 RB-12.
- Deploy menulis `.env` ke `$DEPLOY_DIR` dengan izin 600 (§4). Validasi nilai cukup di satu tempat, yaitu saat aplikasi start (08 STD-GO-05). Nilai yang salah membuat `msp migrate` gagal sebelum ada container yang diganti. **Rotasi**: ubah variabel, lalu jalankan ulang job deploy untuk tag yang sedang berlaku (09 RB-12).
- `/opt/ops/.env` (domain, IP admin, password admin Grafana, SMTP/Telegram Alertmanager, password superuser `gmaps-db` `MSP_GMAPS_DB_PASSWORD`) diisi manual saat bootstrap.
- Salinan berikut disimpan di password manager founder: secret production, `/opt/ops/.env`, `PGBACKREST_REPO1_CIPHER_PASS`, serta `gitlab-secrets.json` + `gitlab.rb` (keduanya tidak ikut backup GitLab). Tanpa salinan ini, backup tidak bisa dipakai bila server hilang.

## 8. Backup & restore

- **pgBackRest**, stanza `msp`, berjalan di container `postgres` production. Repo = object storage S3-compatible off-site di Indonesia (D-14, penyedia Q-09), terenkripsi `aes-256-cbc`, `repo1-retention-full=4`. Konfigurasi lewat env `PGBACKREST_*`.
- **WAL kontinu.** Diaktifkan sekali di production: `ALTER SYSTEM SET archive_mode = on`, `archive_command = 'pgbackrest --stanza=msp archive-push %p'`, `archive_timeout = '5min'`, lalu restart dan `pgbackrest stanza-create`. Data paling banyak hilang ≈ 5 menit → **RPO ≤ 15 menit** (NFR-04). Karena pengaturan ini tersimpan di data dir, staging tidak mengarsip, dan restore uji dengan `--archive-mode=off` (09 RB-10) tidak menulis ke repo production.
- **GitLab off-site.** Tanpa backup ini, satu-satunya salinan repo, variabel CI, dan registry ada di server ini. `gitlab-backup create CRON=1` di-upload ke object storage yang sama (bucket terpisah) lewat `backup_upload_connection`. Salinan lokal hanya disimpan sebentar (`backup_keep_time`), dan retensi off-site diatur dengan aturan lifecycle bucket.
- **Jadwal.** Pipeline schedule harian 01:00 WIB menjalankan job `backup` → `deploy/backup.sh` di runner `deploy`: di `/opt/msp` `pgbackrest backup` (`--type=full` hari Minggu, `diff` hari lain) → `check` → `info`, lalu di `/opt/ops` `gitlab-backup create`. Job gagal bila salah satu langkah gagal atau backup pgBackRest terakhir > 26 jam, dan GitLab mengirim email ke founder. WAL tetap terarsip walau GitLab mati.
- **Uji restore** sekali sebelum rilis F1 (PRD §12.4), lalu tiap bulan. Restore dilakukan ke **project sementara di server yang sama** (`/opt/msp-restore`, 09 RB-10), dengan volume sendiri, hanya `postgres`, dan tidak menyentuh data staging. Durasi (RTO ≤ 8 jam) dan titik pulih dicatat. Setelah selesai, jalankan `docker compose down -v` dan hapus direktorinya. DB staging tidak di-backup; bila hilang, diisi ulang dari kiriman riwayat penuh founder (09 RB-15, RB-03).
- **`gmaps-db` tidak di-backup** (pgBackRest hanya untuk `postgres` production; backup DB kita tidak berubah). Isinya hanya kiriman terbaru, yang bisa dimuat ulang dari file di `/opt/ops/gmaps-dump/` (09 RB-15); riwayat ada di DB MSP yang di-backup. Bila server hilang, file itu ikut hilang: founder menjalankan `export` lagi (09 RB-11). Kiriman riwayat penuh hanya dibutuhkan bila DB MSP hilang melebihi backup, atau untuk mengisi ulang DB staging.

## 9. Bootstrap server

`deploy/bootstrap.sh` bersifat idempotent dan dijalankan dari clone repo: `scp -r deploy root@<ip>:` lalu `ssh root@<ip> bash deploy/bootstrap.sh`. Karena login root dimatikan setelah run pertama, run berikutnya memakai user admin + `sudo`.
- **User**: satu user per admin (sudo, kunci SSH), dan user `deploy` (grup `docker`, setara root) untuk runner `deploy`. Direktori Compose (README) dimiliki `deploy`.
- **SSH** kunci saja (`PasswordAuthentication no`, `PermitRootLogin no`). **Firewall `ufw`**: tolak semua koneksi masuk kecuali 80/443, dan 22 dari allowlist IP admin.
- **Update**: `unattended-upgrades` untuk security update; reboot manual di jam sepi (09 §6). **Docker**: Engine + plugin compose dari repo apt resmi; rotasi log `json-file` (`max-size: 50m`, `max-file: 3`).
- **Ops**: pasang paket `gitlab-runner` (runner `deploy`, user `deploy`), salin `deploy/ops/` ke `/opt/ops`, lalu `docker compose up -d`. Bootstrap berhenti dengan pesan bila `/opt/ops/.env` belum ada; isi file itu dari password manager.
- **Manual sekali**: daftarkan runner `ci` dan `deploy` (§2), buat pipeline schedule `backup`, daftarkan `https://app.<domain>/readyz` dan `https://gitlab.<domain>` di uptime monitor eksternal (09 AL-01), dan buat tabel serta role `msp_source_ro` di `gmaps-db`, lalu muat kiriman pertama dari founder (09 RB-15).

**Bangun ulang server** (RTO ≤ 8 jam, NFR-04; langkah di 09 RB-11): server baru → bootstrap → DNS ke IP baru (Caddy butuh DNS untuk TLS; runner dan registry diakses lewat domain) → `gitlab-backup restore` + daftarkan ulang runner → restore pgBackRest ke `/opt/msp` → Retry `deploy-production` tag yang berlaku → muat kiriman DB gmaps dari founder ke `gmaps-db` (09 RB-15) → pipeline `main` untuk staging.

## 10. Rilis & versi

- **SemVer** lewat tag `vX.Y.Z` di commit `main` yang hijau. Sebelum gerbang F1 (PRD §12) lulus: `v0.y.z`; rilis F1 = `v1.0.0`. `feat` → minor, `fix` → patch, `!` → major.
- **Changelog**: `scripts/changelog.sh <tag-sebelumnya>` mengelompokkan judul commit squash (`feat`, `fix`, lainnya). Orchestrator menyiapkan teksnya; founder membuat tag lewat **Code → Tags → New tag** dan menempelkannya sebagai *Release notes* (menjadi GitLab Release).
- Sebelum tag, founder mengecek: pipeline `main` hijau, laporan `trivy` tanpa CRITICAL yang punya fix, catatan migrasi di MR, dan untuk rilis F1 juga item 05 §7.
- **Hotfix** lewat jalur yang sama (MR → `main` → tag patch); F1 tidak memakai release branch. Versi model (daftar di README) ada di kode dan ikut rilis aplikasi. Hasil tersimpan dengan versi baru dihitung lewat `msp tool rescore` (09 RB-05). Parameter estimasi bukan versi model: diubah dari halaman ops tanpa rilis (09 RB-14).

## 11. Upgrade F2/F3 (hanya bila dibutuhkan)

| Upgrade | Pemicu |
|---|---|
| Pindahkan GitLab + CI (dan staging) ke server kedua | Contention terukur: AL-09 sering, atau p95 production naik saat pipeline/import staging berjalan (PRD §13, 04 §2.1) |
| Deploy tanpa downtime (2 replika `api` di belakang Caddy, atau blue/green); server aplikasi kedua + replika DB | Target 99,5% dan HA di F3 (NFR-03) |
| Image CI siap pakai (Go + Node), runner CI tambahan | Pipeline MR > 15 menit atau antrean job sering menunggu |
| Renovate, image signing + SBOM, Semgrep, ZAP | Update dependency manual mulai berat, diminta customer/audit, atau ada temuan keamanan |
| SOPS/Vault, Ansible/Terraform | Server atau tim ops bertambah |
| Preview environment per MR | Review UI di staging menjadi bottleneck |
| GitLab berbayar (protected environment, approval wajib) | Tim manusia bertambah dan aturan proses §3 tidak cukup |
