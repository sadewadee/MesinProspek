# 06 — CI/CD (lean)

| Field | Value |
|---|---|
| Versi | 0.2 (lean). Menggantikan v0.1 · 2026-10-10 |
| Basis | [README](README.md) (stack F1, kosakata), [PRD](01-prd.md) NFR-03/04/05/09 + §12, D-18, D-19, D-22, D-26 ([00 §7](00-analisis-dokumen-awal.md)). Terkait: [05](05-test-strategy.md) (isi test), [08](08-engineering-standards.md) (`make`, migrasi), [09](09-runbook-operasional.md) (operasi) |

> KISS: GitLab CE + satu runner + Docker Compose + skrip shell. Contoh YAML dan skrip bersifat ilustratif. Semua image dan tool di-pin ke versi eksak; angka versi di contoh hanya contoh.

## 1. Prinsip

- **Trunk-based.** Satu branch `main`. Branch kerja berumur pendek, MR kecil (08 STD-GIT-01).
- **Build once, promote.** Image dibangun sekali per commit `main` dan di-tag git SHA. Staging dan production menjalankan image yang sama; yang berbeda hanya `.env`.
- **Migrasi dulu.** `msp migrate` jalan sebelum container baru start. Setiap migrasi kompatibel dengan versi aplikasi sebelumnya.
- **Rollback = deploy tag sebelumnya.** Skema DB tidak di-rollback.
- **Founder menyetujui production** (D-22). Hanya founder yang membuat tag `v*` dan menjalankan deploy production.

## 2. Server & environment

Ada dua dedicated server di Indonesia (D-19, D-26). Direktori Compose mengikuti README.
- **`ops`**: `/opt/ops` berisi `gitlab` (GitLab CE + container registry), `gitlab-runner` (satu runner, Docker executor), `prometheus`, `alertmanager`, `grafana` (login admin bawaan Grafana lewat HTTPS, tanpa OAuth), `blackbox-exporter` (probe ke production), dan `node-exporter`. `/opt/msp-staging` berisi stack staging. Stack `/opt/ops` di-update manual lewat SSH, karena pipeline tidak me-restart GitLab yang sedang menjalankannya.
- **`prod`**: `/opt/msp` berisi `caddy`, `web`, `api`, `worker`, `kratos`, `postgres`, `node-exporter`.

| Environment | Server | Deploy | Data |
|---|---|---|---|
| `local` | laptop | `docker compose` (Postgres + PostGIS, Kratos, Mailpit) | Fixture sintetis (05) |
| `staging` | `ops` | Otomatis tiap merge ke `main`, dan pada tag sebelum production | Dataset bisnis nyata + GeoAPI, akun test. Tanpa data customer |
| `production` | `prod` | Manual, tag `vX.Y.Z`, founder | Data nyata |

- **Runner**: `concurrent = 4`, `privileged = true` (volume `/certs/client`, `/cache`) agar job build dan E2E bisa memakai `docker:dind`. Runner hanya terdaftar untuk project ini. Privileged setara root di `ops`; risiko ini diterima di F1 karena runner hanya menjalankan MR tim.
- **Edge `ops`**: `caddy` staging adalah satu-satunya proses yang memakai port 80/443 di `ops`. Caddy ini juga meneruskan `gitlab.<domain>` (+ `registry.<domain>`) dan `grafana.<domain>` lewat network Docker bersama (`sites/ops.caddy`, §5).

## 3. Alur Git

1. Agent membuat branch `wp/<nn>-<slug>` dari `main` (07), lalu membuka MR. Judul MR mengikuti Conventional Commits dengan WP sebagai scope (`feat(WP-14): …`, `fix(WP-21): …`, `!` untuk breaking). Deskripsi MR menyebut ID PRD (08 STD-GIT-02/04).
2. `check` hijau → orchestrator me-review → **squash merge** ke `main`. Judul MR menjadi pesan commit dan bahan changelog.
3. `main` → `build` → deploy `staging` otomatis. Founder membuat tag `vX.Y.Z` → E2E J-2..J-6 di staging → founder menjalankan deploy production.

| Aktor | Role GitLab | Boleh |
|---|---|---|
| Agent AI (akun bot) | Developer | Push branch, buka MR. Tanpa SSH (09 §2) |
| Orchestrator (Claude) | Developer | Review dan merge MR ke `main` |
| Founder | Maintainer (satu-satunya) | Membuat tag `v*`, menjalankan `deploy-production`, mengelola variabel CI |

- **Protected branch `main`**: push = *No one*, merge = *Developers + Maintainers*, "Pipelines must succeed", "All threads must be resolved", squash wajib. **Protected tag `v*`**: create = *Maintainers*. Job manual di pipeline tag hanya bisa dijalankan oleh yang boleh membuat tag itu, yaitu founder.
- **Batas GitLab CE.** Approval wajib, CODEOWNERS, dan protected environment adalah fitur berbayar. Karena itu ada dua aturan proses: (1) agent tidak me-merge MR-nya sendiri; (2) MR yang mengubah `.gitlab-ci.yml`, `deploy/`, atau `scripts/ci/` hanya di-merge founder, karena job apa pun di `main` bisa membaca variabel terproteksi.

## 4. Pipeline (`.gitlab-ci.yml`)

| Stage | Kapan | Isi |
|---|---|---|
| `check` | MR, `main` | `make generate` + `git diff --exit-code`, `make lint`, `make test` (08 §0), `govulncheck`, cek MR. Job `e2e`: stack Compose + Playwright Chromium (05) |
| `build` | `main` | Image `api` & `web` → registry GitLab dengan tag SHA. Laporan `trivy image` HIGH/CRITICAL (tidak memblokir) |
| `deploy-staging` | `main` otomatis; tag `v*` | `deploy/deploy.sh` ke `ops`, lalu smoke J-2 dengan akun test yang sudah terverifikasi (`main`) atau J-2..J-6 semua browser dengan akun yang sama (tag). J-1 hanya di job `e2e` (Mailpit) |
| `deploy-production` | tag `vX.Y.Z`, manual, Maintainer | `deploy/deploy.sh` ke `prod`. Smoke manual per 09 RB-01 |
| `rollback` | manual, Maintainer | Retry job `deploy-production` di pipeline tag sebelumnya (09 RB-02). Langkahnya sama dengan deploy, jadi tidak perlu job terpisah (DRY) |
| *schedule* | harian 01:00 WIB | Job `backup` (§8) |

```yaml
# .gitlab-ci.yml (ilustratif)
stages: [check, build, deploy-staging, deploy-production]
workflow:
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event" || $CI_PIPELINE_SOURCE == "schedule"
    - if: $CI_COMMIT_BRANCH == "main" || $CI_COMMIT_TAG =~ /^v\d+\.\d+\.\d+$/
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
  variables: { POSTGRES_PASSWORD: test, MSP_DATABASE_URL: "postgres://postgres:test@postgres:5432/postgres?sslmode=disable" }
  before_script:                              # .nvmrc = versi eksak Node, mis. 24.11.0
    - curl -fsSL "https://nodejs.org/dist/v$(cat .nvmrc)/node-v$(cat .nvmrc)-linux-x64.tar.gz" | tar -xz -C /usr/local --strip-components=1
    - corepack enable && pnpm install --frozen-lockfile && git fetch -q origin main
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
  resource_group: e2e                         # satu stack E2E sekaligus di runner
  script: [docker compose -f deploy/docker-compose.prod.yml -f deploy/docker-compose.e2e.yml up --build --exit-code-from playwright]
build:
  extends: .main
  stage: build
  image: docker:28
  services: ["docker:28-dind"]
  script:
    - echo "$CI_REGISTRY_PASSWORD" | docker login -u "$CI_REGISTRY_USER" --password-stdin "$CI_REGISTRY"
    - docker build -f apps/api/Dockerfile -t "$API_IMAGE" . && docker push "$API_IMAGE"
    - docker build -f apps/web/Dockerfile -t "$WEB_IMAGE" . && docker push "$WEB_IMAGE"
trivy:                                        # laporan saja; dibaca founder sebelum tag rilis
  extends: .main
  stage: build
  needs: [build]
  image: { name: "aquasec/trivy:0.67.0", entrypoint: [""] }
  variables: { TRIVY_USERNAME: $CI_REGISTRY_USER, TRIVY_PASSWORD: $CI_REGISTRY_PASSWORD }
  script:
    - for i in "$API_IMAGE" "$WEB_IMAGE"; do trivy image --severity HIGH,CRITICAL --ignore-unfixed "$i"; done
deploy-staging:
  stage: deploy-staging
  image: mcr.microsoft.com/playwright:v1.56.0-noble   # versi = @playwright/test di lockfile
  rules:
    - if: $CI_COMMIT_BRANCH == "main" && $CI_PIPELINE_SOURCE == "push"
      variables: { E2E_ARGS: "--project=chromium --grep @smoke" }   # J-2; J-1 butuh Mailpit (hanya di stack test)
    - if: $CI_COMMIT_TAG =~ /^v\d+\.\d+\.\d+$/
      variables: { E2E_ARGS: "--grep-invert @ci-only" }              # J-2..J-6, semua browser
  environment: staging
  resource_group: staging
  before_script: [apt-get update -qq && apt-get install -yqq openssh-client curl && corepack enable && pnpm install --frozen-lockfile]
  script:
    - deploy/deploy.sh "$CI_COMMIT_SHA"
    - pnpm --filter ./apps/web exec playwright test $E2E_ARGS
deploy-production:
  stage: deploy-production
  image: alpine:3.22
  rules: [{ if: $CI_COMMIT_TAG =~ /^v\d+\.\d+\.\d+$/, when: manual }]
  environment: production
  resource_group: production
  before_script: [apk add --no-cache bash curl openssh-client]
  script: [deploy/deploy.sh "$CI_COMMIT_SHA"]
backup:                                       # pipeline schedule harian
  extends: deploy-production
  stage: check
  rules: [{ if: $CI_PIPELINE_SOURCE == "schedule" }]
  environment: { name: production, action: access }
  script: [deploy/backup.sh]
```

**`deploy/deploy.sh <git-sha>`** dipakai staging, production, dan rollback. Variabel per environment: `DEPLOY_HOST`, `DEPLOY_DIR` (`/opt/msp-staging` atau `/opt/msp`), `MSP_APP_HOST`.
1. Tulis `.env`: setiap key di `deploy/.env.example` diisi dari variabel CI, ditambah `MSP_IMAGE_REPO=$CI_REGISTRY_IMAGE` dan `MSP_IMAGE_TAG=<git-sha>`. Salin `.env` (chmod 600), `docker-compose.prod.yml` (sebagai `compose.yaml`, agar perintah 09 jalan tanpa `-f`), `Caddyfile`, `kratos/`, dan `postgres/` ke `$DEPLOY_DIR`.
2. `docker login` dengan deploy token `read_registry`, lalu `docker compose pull --ignore-buildable`.
3. `docker compose run --rm worker msp migrate`, lalu `docker compose run --rm kratos migrate sql -e --yes`. Bila gagal, deploy berhenti dan versi lama tetap jalan.
4. `docker compose up -d --build --wait --wait-timeout 300`. `--build` hanya membangun ulang `postgres` bila Dockerfile-nya berubah. Lalu `docker compose exec caddy caddy reload --config /etc/caddy/Caddyfile`, karena perubahan file bind-mount tidak memicu recreate.
5. `curl -fsS https://$MSP_APP_HOST/readyz`. Bila gagal, job merah dan founder memutuskan rollback (09 RB-02).

- Mengganti container menimbulkan jeda beberapa detik. Ini diterima untuk NFR-03 (99%); zero-downtime ada di F3.
- **Build once untuk web.** Jangan memakai `NEXT_PUBLIC_*` yang nilainya berbeda per environment, karena nilainya ter-inline saat build. API dipanggil lewat path relatif `/api/v1`.
- Di MR, image dibangun oleh job `e2e` tanpa push; push ke registry hanya dari `main`. Pipeline tag tidak membangun ulang image: `pull` gagal bila image SHA tidak ada.

## 5. Docker Compose & Caddy

```yaml
# deploy/docker-compose.prod.yml (ringkas). Dipakai staging & production. Semua service: restart: unless-stopped
x-app: &app
  image: ${MSP_IMAGE_REPO}/api:${MSP_IMAGE_TAG}       # api & worker = image yang sama
  env_file: .env
  depends_on: { postgres: { condition: service_healthy } }
services:
  caddy:
    image: caddy:2.10-alpine
    ports: ["80:80", "443:443"]                       # satu-satunya port yang dipublish
    environment: { MSP_APP_HOST: "${MSP_APP_HOST}", MSP_OPS_IP: "${MSP_OPS_IP}", MSP_METRICS_PASSWORD_HASH: "${MSP_METRICS_PASSWORD_HASH}" }
    volumes: ["./Caddyfile:/etc/caddy/Caddyfile:ro", "./sites:/etc/caddy/sites:ro", "caddy_data:/data"]
  web:                                                # tanpa env_file: web tidak memegang secret
    image: ${MSP_IMAGE_REPO}/web:${MSP_IMAGE_TAG}
    healthcheck: { test: ["CMD", "wget", "-qO-", "http://localhost:3000/"], interval: 5s }
  api: { <<: *app, command: ["msp", "api"], healthcheck: { test: ["CMD", "wget", "-qO-", "http://localhost:8080/readyz"], interval: 5s } }
  worker: { <<: *app, command: ["msp", "worker"], stop_grace_period: 60s }   # job River sempat selesai
  kratos: { image: "oryd/kratos:v1.3.1", command: ["serve", "-c", "/etc/kratos/kratos.yml"], env_file: .env, volumes: ["./kratos:/etc/kratos:ro"] }
  node-exporter: { image: "prom/node-exporter:v1.9.1", command: ["--path.rootfs=/host", "--web.telemetry-path=/node/metrics"], pid: host, volumes: ["/:/host:ro,rslave"] }   # metrik host
  postgres:
    build: ./postgres                                 # FROM postgis/postgis:16-3.5 + apt pgbackrest
    image: msp-postgres:16
    env_file: .env                                    # POSTGRES_PASSWORD, PGBACKREST_* (prod)
    volumes: ["pgdata:/var/lib/postgresql/data"]
    healthcheck: { test: ["CMD", "pg_isready", "-U", "postgres"], interval: 5s }
volumes: { caddy_data: {}, pgdata: {} }
```

```caddyfile
# deploy/Caddyfile. MSP_APP_HOST = app.<domain> (production) atau staging.<domain> (staging)
{$MSP_APP_HOST} {
	@api path /api/* /healthz /readyz
	reverse_proxy @api api:8080
	handle_path /.ory/* {
		reverse_proxy kratos:4433        # public base_url = https://<host>/.ory/; admin 4434 tidak dirutekan
	}
	@scrape {                            # hanya Prometheus di ops: allowlist IP + basic auth
		path /metrics /node/metrics
		remote_ip {$MSP_OPS_IP}
	}
	handle @scrape {
		basic_auth {
			prometheus {$MSP_METRICS_PASSWORD_HASH}   # caddy hash-password
		}
		reverse_proxy /node/metrics node-exporter:9100
		reverse_proxy api:9090           # listener metrik internal (MSP_METRICS_ADDR)
	}
	reverse_proxy web:3000
}
import sites/*.caddy                    # kosong di prod; di ops: gitlab, registry, grafana
```

Docker melewati aturan `ufw` untuk port yang dipublish, jadi hanya `caddy` yang boleh memakai `ports:`. Karena itu Prometheus di `ops` membaca `/metrics` (`api:9090`) dan `/node/metrics` (`node-exporter`) lewat Caddy. Image `api` berbasis Alpine (ada `wget` untuk healthcheck). Image `web` memakai Next.js `output: "standalone"`.

## 6. Migrasi DB

Migrasi memakai goose di `apps/api/db/migrations`, di-embed di binary, dan dijalankan dengan `docker compose run --rm worker msp migrate`. Aturan lengkapnya di 08 STD-DB-01.
- **Backward-compatible saja**: tambah → backfill → pindah pemakaian → hapus di rilis berikutnya. Contoh rename kolom: tambah kolom baru, tulis ke keduanya, baca kolom baru, lalu drop kolom lama di rilis berikutnya.
- **Migrasi yang sudah ada di `main` tidak pernah diedit.** Perbaikan = migrasi baru. `scripts/ci/mr-check.sh` menolak perubahan file migrasi lama.
- **Migrasi jalan sebelum container baru start** (§4 langkah 3), satu transaksi per migrasi. Bila gagal, aplikasi lama tetap jalan. **Tanpa down migration** di staging/production: rollback hanya mengganti aplikasi, dan `msp migrate` versi lama tidak menjalankan apa pun.
- **Perubahan berat dipecah**: `CREATE INDEX CONCURRENTLY` (`-- +goose NO TRANSACTION`) untuk tabel besar. Backfill jutaan baris lewat job atau `msp tool`, bukan di dalam migrasi.

## 7. Secret & konfigurasi

- `deploy/.env.example` adalah satu-satunya daftar key (DRY), berisi nilai dummy dan komentar: `MSP_*`, secret Kratos, `POSTGRES_PASSWORD`, `PGBACKREST_*`.
- **Variabel CI GitLab**, di-scope per environment (`staging` / `production`): setiap key di atas, **Protected** + **Masked** untuk secret. Juga `DEPLOY_HOST`, `DEPLOY_DIR`, `SSH_PRIVATE_KEY` (tipe File, satu kunci per server), `SSH_KNOWN_HOSTS`, dan deploy token registry.
- Deploy menulis `.env` ke `$DEPLOY_DIR` dengan izin 600 (§4). Validasi nilai cukup di satu tempat, yaitu saat aplikasi start (08 STD-GO-05). Nilai yang salah membuat `msp migrate` gagal sebelum ada container yang diganti. **Rotasi**: ubah variabel, lalu jalankan ulang job deploy untuk tag yang sedang berlaku (09 RB-12).
- Salinan secret production, `PGBACKREST_REPO1_CIPHER_PASS`, dan `gitlab-secrets.json` disimpan di password manager founder. Tanpa salinan ini, backup tidak bisa dipakai bila kedua server hilang.

## 8. Backup & restore

- **pgBackRest**, stanza `msp`, berjalan di container `postgres` `prod`. Repo = object storage S3-compatible off-site di Indonesia (D-14, penyedia Q-09), terenkripsi `aes-256-cbc`, `repo1-retention-full=4`. Konfigurasi lewat env `PGBACKREST_*`.
- **WAL kontinu.** Diaktifkan sekali di `prod`: `ALTER SYSTEM SET archive_mode = on`, `archive_command = 'pgbackrest --stanza=msp archive-push %p'`, `archive_timeout = '5min'`, lalu restart dan `pgbackrest stanza-create`. Data paling banyak hilang ≈ 5 menit → **RPO ≤ 15 menit** (NFR-04). Karena pengaturan ini tersimpan di data dir, staging tidak mengarsip, dan restore uji dengan `--archive-mode=off` (09 RB-10) tidak menulis ke repo production.
- **Jadwal.** Pipeline schedule harian 01:00 WIB menjalankan job `backup` → `deploy/backup.sh` lewat SSH ke `prod`: `pgbackrest backup` (`--type=full` hari Minggu, `diff` hari lain) → `check` → `info`. Job gagal bila backup terakhir > 26 jam, dan GitLab mengirim email ke founder. Bila `ops` mati, WAL tetap terarsip; yang tertunda hanya backup full/diff.
- **Uji restore** sekali sebelum rilis F1 (PRD §12.4), lalu tiap bulan, ke **DB sementara yang terpisah** di server `ops` (`/opt/msp-restore`, 09 RB-10). Durasi (RTO ≤ 8 jam) dan titik pulih dicatat. Setelah selesai: `docker compose down -v` dan hapus direktorinya, sehingga tidak ada data production yang tertinggal di server staging.
- **GitLab**: cron harian di `ops` menjalankan `gitlab-backup create CRON=1`, upload ke object storage yang sama (bucket terpisah) lewat `backup_upload_connection`. DB staging tidak di-backup; isinya bisa di-import ulang.

## 9. Bootstrap server

`deploy/bootstrap.sh` idempotent dan sama untuk kedua server: `ssh root@<ip> 'bash -s' < deploy/bootstrap.sh`.
- **User**: satu user per admin (sudo, kunci SSH) dan user `deploy` untuk CI (grup `docker`, setara root). Direktori Compose (README) dimiliki `deploy`.
- **SSH** kunci saja (`PasswordAuthentication no`, `PermitRootLogin no`). **Firewall `ufw`**: tolak semua koneksi masuk kecuali 80/443, dan 22 dari allowlist (IP admin + IP `ops` untuk runner).
- **Update**: `unattended-upgrades` untuk security update; reboot manual di jam sepi (09 §6). **Docker**: Engine + plugin compose dari repo apt resmi; rotasi log `json-file` (`max-size: 50m`, `max-file: 3`).
- Langkah tambahan sekali di `ops`: start stack `/opt/ops`, daftarkan runner, pasang cron backup GitLab.
- Langkah tambahan di `prod`: cron penjaga `ops` (tiap 5 menit `curl -fsS https://grafana.<domain>/api/health`; gagal 3× berturut-turut → email ke founder), agar matinya server monitoring tetap ketahuan (09 §1.1).

**Bangun ulang server = bootstrap + restore + deploy** (09 RB-11). `prod`: bootstrap → restore pgBackRest → Retry `deploy-production` tag yang berlaku → DNS. `ops`: bootstrap → `gitlab-backup restore` → daftarkan runner → pipeline `main`.

## 10. Rilis & versi

- **SemVer** lewat tag `vX.Y.Z` di commit `main` yang hijau. Sebelum gerbang F1 (PRD §12) lulus: `v0.y.z`; rilis F1 = `v1.0.0`. `feat` → minor, `fix` → patch, `!` → major.
- **Changelog**: `scripts/changelog.sh <tag-sebelumnya>` mengelompokkan judul commit squash (`feat`, `fix`, lainnya). Orchestrator menyiapkan teksnya; founder membuat tag lewat **Code → Tags → New tag** dan menempelkannya sebagai *Release notes* (menjadi GitLab Release).
- Sebelum tag, founder mengecek: pipeline `main` hijau, laporan `trivy` tanpa CRITICAL yang punya fix, catatan migrasi di MR, dan untuk rilis F1 juga item 05 §7.
- **Hotfix** lewat jalur yang sama (MR → `main` → tag patch); F1 tidak memakai release branch. Versi model (`priority-X.Y.Z`, `geo-X.Y.Z`, `brief-X.Y.Z`) ada di kode dan ikut rilis aplikasi. Aktivasinya lewat `msp tool rescore` (09 RB-05).

## 11. Upgrade F2/F3 (hanya bila dibutuhkan)

| Upgrade | Pemicu |
|---|---|
| Deploy tanpa downtime (2 replika `api` di belakang Caddy, atau blue/green); server aplikasi kedua + replika DB | Target 99,5% dan HA di F3 (NFR-03) |
| Image CI siap pakai (Go + Node), cache Go/pnpm/BuildKit, runner kedua | Pipeline MR > 15 menit atau antrean E2E sering menunggu |
| Renovate, image signing + SBOM, Semgrep, ZAP | Update dependency manual mulai berat, diminta customer/audit, atau ada temuan keamanan |
| SOPS/Vault, Ansible/Terraform | Lebih dari 2 server atau tim ops bertambah |
| Preview environment per MR | Review UI di staging menjadi bottleneck |
| GitLab berbayar (protected environment, approval wajib) | Tim manusia bertambah dan aturan proses §3 tidak cukup |
