# 06 — CI/CD (lean)

| Field | Value |
|---|---|
| Versi | 0.4 (lean) · 2026-10-15. Pipeline, deploy, dan Compose dirujuk ke file repo (tidak disalin). Sumber data = DB gmaps read-only (D-31), kiriman mingguan ramping founder dimuat ke `gmaps-db` (D-35, D-37) |
| Basis | [README](README.md) (stack F1, kosakata), [PRD](01-prd.md) NFR-03/04/05/09 + §12 + §13, D-14, D-18, D-19, D-22, D-26, D-29, D-31, D-35, D-37, D-39 ([00 §7](00-analisis-dokumen-awal.md)). Terkait: [04](04-technical-design.md) §2.1 (kapasitas), [05](05-test-strategy.md) (isi test), [08](08-engineering-standards.md) (`make`, migrasi), [09](09-runbook-operasional.md) (operasi) |

> KISS: satu server, GitLab CE + dua runner (CI dan deploy) + Docker Compose + skrip shell. Dokumen ini menjelaskan desain dan alasannya; isi persisnya ada di file repo (`.gitlab-ci.yml`, `deploy/`, `scripts/ci/`) yang dirujuk tiap bagian. Semua image dan tool di-pin ke versi eksak di file-file itu.

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
- **Network.** Stack `/opt/ops` membuat network Docker `edge-production` dan `edge-staging`. `web`, `api`, `worker`, dan `kratos` tiap environment bergabung ke `edge-${MSP_ENV}`, sehingga `caddy` menjangkau layanan web dan `prometheus` mengambil metrik `api` dan `worker` lewat nama container (`msp-api-1:9090`, `msp-worker-1:9090`, dan versi `msp-staging-…`). Prometheus dan Alertmanager hanya dipublikasikan di `127.0.0.1` (untuk tunnel SSH, 09 §2); port publik hanya milik Caddy. Network dipisah per environment karena kedua project memakai nama service yang sama. Stack `/opt/ops` juga membuat network `gmaps`, yang hanya diikuti `gmaps-db` dan `worker` kedua environment; `worker` menjangkaunya sebagai `gmaps-db:5432`. `postgres` hanya ada di network project masing-masing.
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
| `kratos-migrate` (one-shot) | 0,5 · 0,25 GB | 0,5 · 0,25 GB |

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

Isi persis job ada di `.gitlab-ci.yml`; kepala file merangkum job per jenis pipeline (MR, `main`, tag `v*`). Job memanggil target `make` yang sama dengan lokal (08 §0).

| Stage | Job | Kapan | Isi |
|---|---|---|---|
| `check` | `check` | MR, `main` | `make generate` lalu tidak boleh ada file berubah atau file baru; `make lint` (termasuk gitleaks); `make test` (unit + integrasi pada service `postgres` dengan image Postgres + PostGIS yang sama dengan production, `CI=true`); dependency rentan: `govulncheck` dan `pnpm audit --prod` (04 §15.8). Test import membaca DB sumber fixture sebagai database kedua di service yang sama (05), tanpa service tambahan |
| | `mr-check` | MR | `scripts/ci/mr-check.sh`: judul Conventional Commits, deskripsi MR `feat`/`fix` menyebut WP + ID PRD (08 STD-GIT-02/04), migrasi yang sudah ada di `main` tidak diubah dan nomor migrasi baru di atas yang terakhir di `main` (STD-DB-01). Job terpisah karena butuh clone penuh |
| | `e2e` | MR, `main` | Di `docker:dind`: stack lokal (`deploy/docker-compose.local.yml` + overlay `deploy/docker-compose.e2e.yml`, tanpa port host) dibangun dari source, lalu service `playwright` menjalankan Playwright Chromium (05) ke `http://caddy`, termasuk J-1 dengan Mailpit. Exit code = hasil Playwright; report, trace, dan log stack disimpan bila gagal |
| `build` | `build` | `main` | Image `api` (context `apps/api`) dan `web` (context root repo, workspace pnpm) → registry GitLab dengan tag git SHA, plus `:main` sebagai cache inline build berikutnya |
| | `trivy` | `main` | Laporan HIGH/CRITICAL kedua image. Tidak memblokir; dibaca founder sebelum tag (§10) |
| `deploy-staging` | `deploy-staging` | `main` otomatis; tag `v*` | Runner `deploy`: `deploy/deploy.sh staging <sha>` di `/opt/msp-staging` |
| | `smoke-staging` | setelah `deploy-staging` | Runner `ci`, Playwright ke staging dengan akun test yang sudah terverifikasi: test bertanda `@smoke` (J-2) di `main`; semua test kecuali `@ci-only` (J-2..J-7) di semua browser pada tag. J-1 butuh Mailpit, jadi hanya di `e2e`. Sampai J-2 ada (WP-20) job ini lolos tanpa test (`--pass-with-no-tests`) |
| `deploy-production` | `deploy-production` | tag `vX.Y.Z`, manual, founder | Runner `deploy`: `deploy/deploy.sh production <sha>` di `/opt/msp`. Smoke manual per 09 RB-01 |
| — | rollback | manual, founder | **Retry** job `deploy-production` di pipeline tag sebelumnya (09 RB-02). Langkahnya sama dengan deploy, jadi tidak perlu job terpisah (DRY) |
| — | `backup` | *schedule* harian 01:00 WIB (WP-25) | pgBackRest + GitLab ke disk lokal (§8). Salinan off-site dibuat cron host `msp-offsite`, bukan pipeline |

**`deploy/deploy.sh <staging|production> <image-tag>`** dijalankan runner `deploy` langsung di server, untuk deploy dan rollback. Direktorinya tetap: `/opt/msp-staging` atau `/opt/msp`. Langkahnya (detail di kepala skrip):
1. Tulis `.env` (izin 600) dari variabel CI sesuai `deploy/.env.example` (§7), ditambah `MSP_ENV`, `MSP_IMAGE_REPO=$CI_REGISTRY_IMAGE`, `MSP_IMAGE_TAG=<git-sha>`, dan `COMPOSE_FILE`. Key wajib yang belum ada menghentikan deploy dan disebut namanya. Salin `docker-compose.prod.yml` sebagai `compose.yaml` (agar perintah 09 jalan tanpa `-f`), `docker-compose.staging.yml` sebagai `compose.staging.yaml`, `kratos/`, dan `postgres/` (`cp -rL`).
2. `docker login` dengan deploy token `gitlab-deploy-token` (`read_registry`), lalu `docker compose pull --ignore-buildable`.
3. `docker compose run --rm worker msp migrate`, lalu `docker compose run --rm kratos-migrate` (`kratos migrate sql`). Bila gagal, deploy berhenti dan container lama tetap jalan (09 RB-01 langkah 3).
4. `docker compose up -d --build --wait --wait-timeout 300 --remove-orphans`. `--build` hanya membangun ulang `postgres` bila Dockerfile-nya berubah.
5. `curl $MSP_PUBLIC_URL/readyz` lewat Caddy. Bila gagal, job merah dan founder memutuskan rollback (09 RB-02).

- Mengganti container menimbulkan jeda beberapa detik. Ini diterima untuk NFR-03 (99%); zero-downtime ada di F3.
- **Build once untuk web.** Jangan memakai `NEXT_PUBLIC_*` yang nilainya berbeda per environment, karena nilainya ter-inline saat build. API dipanggil lewat path relatif `/api/v1`.
- Push ke registry hanya dari `main`. Pipeline tag tidak membangun ulang image: tag dibuat di commit `main` yang hijau (§10), jadi image SHA-nya sudah ada; `pull` gagal bila tidak ada.

## 5. Docker Compose & Caddy

Isi persisnya ada di file berikut; komentar di dalamnya menjelaskan tiap pilihan.

| File | Isi |
|---|---|
| `deploy/docker-compose.prod.yml` | Stack aplikasi staging & production: `web`, `api` dan `worker` (image `api` yang sama), `kratos`, `kratos-migrate` (one-shot `kratos migrate sql`), `postgres` (build dari `deploy/postgres/`: PostGIS + pgBackRest). Semua `restart: unless-stopped` dengan `deploy.resources.limits` = kolom production §2 |
| `deploy/docker-compose.staging.yml` | Batas resource staging (§2) |
| `deploy/docker-compose.local.yml` | Stack lokal dari source: service production ditambah `caddy`, `mailpit`, dan one-shot `migrate`. Cara pakai di `deploy/README.md` |
| `deploy/ops/compose.yaml` | Stack `/opt/ops`, termasuk `gmaps-db` |
| `deploy/ops/Caddyfile` | Satu Caddy untuk semua domain |

Aturan yang dijaga file-file itu:
- **Secret seperlunya.** `web` tanpa `env_file` (hanya `MSP_KRATOS_PUBLIC_URL`), karena web tidak memegang secret. `postgres` hanya menerima key yang dibacanya, bukan seluruh `.env`, agar deploy tag baru tidak membuat ulang container DB.
- **Rute Caddy** (`app.<domain>` → project `msp`, `staging.<domain>` → `msp-staging`): `/api/*`, `/healthz`, `/readyz` → `api:8080`; `/.ory/*` → `kratos:4433` (public base URL Kratos = `<MSP_PUBLIC_URL>/.ory/`; admin `:4434` tidak dirutekan); selebihnya → `web:3000`. Metrik `:9090` tidak dirutekan. Grafana hanya dari IP admin.
- **`gmaps-db`** tanpa `ports:`, hanya di network `gmaps` (§2).
- **Healthcheck** memakai `wget`: image `api` berbasis Alpine, image `web` memakai Next.js `output: "standalone"`. Kratos dijalankan dengan `--sqa-opt-out`, jadi tidak mengirim telemetri ke Ory.

## 6. Migrasi DB

Migrasi memakai goose di `apps/api/db/migrations`, di-embed di binary, dan dijalankan dengan `docker compose run --rm worker msp migrate`. Aturan lengkapnya di 08 STD-DB-01.
- **Backward-compatible saja**: tambah → backfill → pindah pemakaian → hapus di rilis berikutnya. Contoh rename kolom: tambah kolom baru, tulis ke keduanya, baca kolom baru, lalu drop kolom lama di rilis berikutnya.
- **Migrasi yang sudah ada di `main` tidak pernah diedit.** Perbaikan = migrasi baru. `scripts/ci/mr-check.sh` menolak perubahan file migrasi lama.
- **Migrasi jalan sebelum container baru start** (§4 langkah 3), satu transaksi per migrasi. Bila gagal, aplikasi lama tetap jalan. **Tanpa down migration** di staging/production: rollback hanya mengganti aplikasi, dan `msp migrate` versi lama tidak menjalankan apa pun.
- **Perubahan berat dipecah**: `CREATE INDEX CONCURRENTLY` (`-- +goose NO TRANSACTION`) untuk tabel besar. Backfill jutaan baris lewat job atau `msp tool`, bukan di dalam migrasi.

## 7. Secret & konfigurasi

- `deploy/.env.example` adalah satu-satunya daftar key (DRY), berisi nilai dummy dan komentar: `MSP_*`, secret Kratos, `POSTGRES_PASSWORD`, `PGBACKREST_*`. Isinya hanya secret dan pengaturan proses (04 §14).
- **Pengaturan produk tidak ada di `.env`.** Provider, model, dan base URL AI (termasuk fallback), plan, serta parameter estimasi diubah dari halaman ops, tercatat di audit, dan berlaku tanpa deploy (D-29, D-32, 04). Dari AI, yang ada di `.env` hanya API key-nya: `MSP_ANTHROPIC_API_KEY` dan `MSP_LLM_FALLBACK_API_KEY`.
- **Variabel CI GitLab**, di-scope per environment (`staging` / `production`, jangan "All"): key wajib di `deploy/.env.example` (nilai contoh berisi `CHANGE_ME`) dan key opsional yang dipakai, **Protected** + **Masked** untuk secret. Cara tiap key diisi ada di kepala `.env.example`. Ditambah deploy token `gitlab-deploy-token` (`read_registry`), langkahnya di `deploy/README.md`.
- **`MSP_SOURCE_DATABASE_URL`** (kiriman DB gmaps di `gmaps-db`, D-31, D-37) adalah secret seperti key lain: variabel CI Protected + Masked di scope `staging` dan `production`, lalu ditulis ke `.env` masing-masing. Nilainya sama di kedua environment: `postgres://msp_source_ro:<password>@gmaps-db:5432/gmaps`. Role `msp_source_ro` dibuat ops sekali di `gmaps-db` (09 RB-15); rotasi password di 09 RB-12.
- Deploy menulis `.env` ke direktori environment (`/opt/msp-staging` atau `/opt/msp`) dengan izin 600 (§4). Validasi nilai cukup di satu tempat, yaitu saat aplikasi start (08 STD-GO-05). Nilai yang salah membuat `msp migrate` gagal sebelum ada container yang diganti. **Rotasi**: ubah variabel, lalu jalankan ulang job deploy untuk tag yang sedang berlaku (09 RB-12).
- `/opt/ops/.env` (domain, IP admin, password admin Grafana, SMTP/Telegram Alertmanager, password superuser `gmaps-db` `MSP_GMAPS_DB_PASSWORD`) dan `/opt/ops/rclone.conf` (kredensial SFTP/FTPS off-site, §8; root, izin 600) diisi manual saat bootstrap. Keduanya tidak pernah masuk repo.
- Salinan berikut disimpan di password manager founder: secret production, `/opt/ops/.env`, `/opt/ops/rclone.conf`, `PGBACKREST_REPO1_CIPHER_PASS`, serta `gitlab-secrets.json` + `gitlab.rb` (keduanya tidak ikut backup GitLab). Tanpa salinan ini, backup tidak bisa dipakai bila server hilang. Khusus `PGBACKREST_REPO1_CIPHER_PASS`: tanpa passphrase ini **semua** backup pgBackRest, lokal maupun off-site, tidak bisa dibuka, jadi salinannya wajib ada di luar server.

## 8. Backup & restore

Off-site = **server SFTP/FTPS milik founder di Indonesia** (D-39, memenuhi D-14). pgBackRest menulis ke repo di disk lokal, lalu rclone menyalinnya ke sana. Repo sengaja lokal: arsip WAL tidak bergantung pada jaringan ke server off-site, dan FTPS tetap bisa dipakai (pgBackRest tidak mendukung FTP/FTPS).

- **pgBackRest**, stanza `msp`, berjalan di container `postgres` production. `repo1` = disk lokal (`posix`) di `/var/lib/pgbackrest`, yaitu bind mount `/opt/msp/pgbackrest` (§5). Terenkripsi di server sebelum disalin ke mana pun: `repo1-cipher-type=aes-256-cbc` dengan `PGBACKREST_REPO1_CIPHER_PASS` (§7). Retensi `repo1-retention-full=4` (full mingguan + diff harian) juga berlaku untuk salinan off-site, karena `rclone sync` meniru repo lokal. Konfigurasi lewat env `PGBACKREST_*`. Ukuran repo masuk anggaran disk (04 §2.1).
- **WAL kontinu.** Diaktifkan sekali di production: `sudo install -d -o 999 -g 999 -m 750 /opt/msp/pgbackrest` (999 = uid `postgres` di image), `ALTER SYSTEM SET archive_mode = on`, `archive_command = 'pgbackrest --stanza=msp archive-push %p'`, `archive_timeout = '60s'`, lalu restart dan `pgbackrest stanza-create`. WAL masuk repo lokal paling lambat ±1 menit. Karena pengaturan ini tersimpan di data dir, staging tidak mengarsip, dan restore dengan `--archive-mode=off` (09 RB-10) juga tidak.
- **GitLab.** Tanpa backup ini, satu-satunya salinan repo, variabel CI, dan registry ada di server ini. `gitlab-backup create CRON=1` menulis ke `/var/opt/gitlab/backups`, yaitu bind mount `/opt/ops/gitlab-backups/`, lalu disalin ke off-site oleh skrip yang sama. Retensi = `backup_keep_time` (singkat, mis. 3 hari), sama untuk lokal dan off-site. File backup GitLab tidak dienkripsi; variabel CI di dalamnya terenkripsi dengan kunci di `gitlab-secrets.json`, yang tidak ikut backup.
- **Salinan off-site (rclone).** Cron root `/etc/cron.d/msp-offsite` menjalankan `/opt/ops/offsite.sh` (keduanya dari `deploy/ops/`) **tiap 5 menit di host**, bukan lewat GitLab, jadi tetap jalan walau GitLab mati. Remote rclone `offsite` (backend `sftp`, atau `ftp` dengan `explicit_tls = true` untuk FTPS; untuk SFTP isi `known_hosts_file` agar host key diverifikasi) ada di `/opt/ops/rclone.conf` (§7).
  Urutan di `offsite.sh`: (1) repo lokal yang kosong (server baru, repo terhapus) tidak pernah disinkron, karena `sync` ikut menghapus di tujuan; (2) WAL disinkron setiap run tanpa menunggu upload besar; (3) backup pgBackRest + GitLab dilewati selama run sebelumnya masih mengunggah (mis. full mingguan). Tiap bagian yang berhasil menulis metrik `msp_offsite_last_success_timestamp_seconds{part="wal"|"backup"}` ke textfile collector `node-exporter` (09 AL-12). Baris salinan kedua B2 ada di skrip dalam keadaan nonaktif.
  **RPO off-site.** WAL masuk repo lokal ≤ ±1 menit dan tersalin pada run berikutnya (≤ 5 menit). Bila seluruh server hilang, data yang hilang normalnya **±5–10 menit terakhir**, masih di dalam RPO ≤ 15 menit (NFR-04). Bila server off-site tidak terjangkau, backup dan WAL tetap aman di lokal dan tersalin begitu tersambung lagi; selama itu RPO bila server hilang tidak terjamin, dan AL-12 berbunyi (09). `rclone sync` ikut menghapus di tujuan, jadi salinan off-site bukan arsip yang kebal hapus: isi repo lokal tidak pernah dihapus manual (09 RB-13).
- **Salinan kedua Backblaze B2: nonaktif.** Backblaze tidak punya region di Indonesia (US West/East, EU Central, CA East), jadi salinan ini menunggu jawaban tim legal (Q-19). Isinya hanya repo pgBackRest yang sudah terenkripsi; backup GitLab tidak ikut. Menyalakannya cukup dengan remote `b2` dan baris B2 di `offsite.sh` (09 §1.2), tanpa perubahan lain.
- **Jadwal.** Pipeline schedule harian 01:00 WIB menjalankan job `backup` → `deploy/backup.sh` di runner `deploy`: di `/opt/msp` `pgbackrest backup` (`--type=full` hari Minggu, `diff` hari lain) → `check` → `info`, lalu di `/opt/ops` `gitlab-backup create`. Keduanya menulis ke disk lokal; `msp-offsite` menyalinnya. Job gagal bila salah satu langkah gagal atau backup pgBackRest terakhir > 26 jam, dan GitLab mengirim email ke founder. WAL tetap terarsip dan tersalin walau GitLab mati.
- **Uji restore** sekali sebelum rilis F1 (PRD §12 butir 5, 05 §7), lalu tiap bulan, **dari salinan off-site**: repo disalin dengan `rclone copy` dari `offsite:msp/pgbackrest`, bukan diambil dari repo lokal, sehingga jalur saat server hilang ikut terbukti. Restore dilakukan ke **project sementara di server yang sama** (`/opt/msp-restore`, 09 RB-10), dengan volume sendiri, hanya `postgres`, dan tidak menyentuh data staging. Durasi (RTO ≤ 8 jam) dan titik pulih dicatat. Setelah selesai, jalankan `docker compose down -v` dan hapus direktorinya. DB staging tidak di-backup; bila hilang, diisi ulang dari kiriman riwayat penuh founder (09 RB-15, RB-03).
- **`gmaps-db` tidak di-backup** (pgBackRest hanya untuk `postgres` production; backup DB kita tidak berubah). Isinya hanya kiriman terbaru, yang bisa dimuat ulang dari file di `/opt/ops/gmaps-dump/` (09 RB-15); riwayat ada di DB MSP yang di-backup. Bila server hilang, file itu ikut hilang: founder menjalankan `export` lagi (09 RB-11). Kiriman riwayat penuh hanya dibutuhkan bila DB MSP hilang melebihi backup, atau untuk mengisi ulang DB staging.

## 9. Bootstrap server

`deploy/bootstrap.sh` bersifat idempotent dan dijalankan dari clone repo: `scp -r deploy root@<ip>:` lalu `ssh root@<ip> bash deploy/bootstrap.sh`. Karena login root dimatikan setelah run pertama, run berikutnya memakai user admin + `sudo`.
- **Postgres pertama kali** (volume data kosong): `deploy/postgres/init-msp.sh` menjalankan `apps/api/db/roles.sql` (lewat symlink `deploy/postgres/roles.sql`; `deploy.sh` menyalin dengan `cp -rL`), membuat database `msp` milik `msp_owner`, membuat `postgis` + `pg_trgm` sebagai superuser, mengisi password kedua role dari `MSP_MIGRATE_DATABASE_URL` dan `MSP_DATABASE_URL` di `.env`, serta membuat role dan database `kratos` dengan password dari `DSN`. Setelah itu `msp migrate` berjalan sebagai `msp_owner` dan `kratos-migrate` sebagai `kratos`.
- **User**: satu user per admin dari `deploy/admin-keys/<user>.pub` (sudo tanpa password, kunci SSH), dan user `deploy` (grup `docker`, setara root) untuk runner `deploy`. Direktori Compose (README) dimiliki `deploy`.
- **SSH** kunci saja (`PasswordAuthentication no`, `PermitRootLogin no`). **Firewall `ufw`**: tolak semua koneksi masuk kecuali 80/443, dan 22 dari allowlist IP admin.
- **Update**: `unattended-upgrades` untuk security update; reboot manual di jam sepi (09 §6). **Docker**: Engine + plugin compose dari repo apt resmi; rotasi log `json-file` (`max-size: 50m`, `max-file: 3`).
- **Ops**: pasang paket `gitlab-runner` (runner `deploy`, user `deploy`) dan `rclone`, salin `deploy/ops/` ke `/opt/ops` (termasuk `offsite.sh` dan direktori `textfile/` yang di-mount read-only ke `node-exporter`), pasang `msp-offsite` di `/etc/cron.d/` (§8), lalu `docker compose up -d`. Bootstrap berhenti dengan pesan bila `/opt/ops/.env` atau `/opt/ops/rclone.conf` belum ada; isi keduanya dari password manager. Di server baru cron `msp-offsite` belum menyalin apa pun sampai repo pgBackRest dipulihkan (guard di §8).
- **Manual sekali**: daftarkan runner `ci` dan `deploy` (§2), buat pipeline schedule `backup`, daftarkan `https://app.<domain>/readyz` dan `https://gitlab.<domain>` di uptime monitor eksternal (09 AL-01), dan buat tabel serta role `msp_source_ro` di `gmaps-db`, lalu muat kiriman pertama dari founder (09 RB-15).

**Bangun ulang server** (RTO ≤ 8 jam, NFR-04; langkah di 09 RB-11): server baru → bootstrap → DNS ke IP baru (Caddy butuh DNS untuk TLS; runner dan registry diakses lewat domain) → salin backup GitLab dari off-site (`rclone copy`) + `gitlab-backup restore` + daftarkan ulang runner → salin repo pgBackRest dari off-site ke `/opt/msp/pgbackrest` + restore → Retry `deploy-production` tag yang berlaku → muat kiriman DB gmaps dari founder ke `gmaps-db` (09 RB-15) → pipeline `main` untuk staging.

## 10. Rilis & versi

- **SemVer** lewat tag `vX.Y.Z` di commit `main` yang hijau. Sebelum gerbang F1 (PRD §12) lulus: `v0.y.z`; rilis F1 = `v1.0.0`. `feat` → minor, `fix` → patch, `!` → major.
- **Changelog**: `scripts/changelog.sh <tag-sebelumnya>` mengelompokkan judul commit squash (`feat`, `fix`, lainnya). Orchestrator menyiapkan teksnya; founder membuat tag lewat **Code → Tags → New tag** dan menempelkannya sebagai *Release notes* (menjadi GitLab Release).
- Sebelum tag, founder mengecek: pipeline `main` hijau, laporan `trivy` tanpa CRITICAL yang punya fix, catatan migrasi di MR, dan untuk rilis F1 juga item 05 §7.
- Buat tag saat tidak ada pipeline `main` yang sedang berjalan. Pipeline tag langsung men-deploy staging, sehingga job `deploy-staging` pipeline `main` yang lebih lama dibatalkan GitLab ("older than the latest deployment", setting *Prevent outdated deployment jobs*). Bila itu terjadi, **Retry** job tersebut setelah `smoke-staging` pipeline tag selesai.
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
