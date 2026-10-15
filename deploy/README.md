# deploy/

File untuk satu server (D-19, D-26) dan stack lokal. Desainnya ada di [06](../docs/06-ci-cd.md) §2, §5, §7–§9; prosedur operasi ada di [09](../docs/09-runbook-operasional.md).

| File | Isi |
|---|---|
| `bootstrap.sh` | Bootstrap server. Aman dijalankan ulang, dan menjadi langkah pertama RB-11 |
| `ops/` | Stack `/opt/ops`: `compose.yaml`, `Caddyfile`, Prometheus + `alerts.yml`, Alertmanager, Grafana, `offsite.sh` + `msp-offsite.cron`, `.env.example` |
| `docker-compose.prod.yml`, `docker-compose.staging.yml` | Stack aplikasi staging & production. Disalin oleh `deploy.sh` (06 §4) |
| `deploy.sh` | Deploy satu environment: `deploy.sh <staging\|production> <image-tag>`. Dijalankan runner `deploy` dari job `deploy-staging`/`deploy-production` di `.gitlab-ci.yml`; rollback = job yang sama di pipeline rilis sebelumnya (09 RB-01, RB-02) |
| `.env.example` | Satu-satunya daftar key `.env` stack aplikasi (06 §7). Cara tiap key diisi (wajib, opsional, default) ada di kepala file |
| `postgres/` | Image Postgres + pgBackRest dan init pertama DB `msp` dan `kratos`. `roles.sql` adalah symlink ke `apps/api/db/roles.sql`, jadi salin dengan `cp -rL` |
| `kratos/` | Konfigurasi Ory Kratos untuk semua stack (04 §9): `kratos.yml`, skema identitas (email), mapper Google. Nilai per environment dan secret datang dari `.env` (daftar di kepala `kratos.yml`) |
| `docker-compose.local.yml`, `Caddyfile.local`, `.env.local.example` | Stack `local` di laptop (06 §2): service production dibangun dari source + `caddy` dan `mailpit`. Jalankan dari root repo: `docker compose -f deploy/docker-compose.local.yml up --build`, lalu buka `http://localhost:8000` (Mailpit `http://localhost:8025`). Nilai dummy dari `.env.local.example`; timpa di `.env.local` (tidak di-commit) |
| `admin-keys/<user>.pub` | Kunci SSH publik tiap admin. Nama file = nama user Linux |

## Bootstrap (founder)

1. Arahkan DNS `app.`, `staging.`, `gitlab.`, `registry.`, dan `grafana.<domain>` ke IP server.
2. Tambahkan `admin-keys/<user>.pub` untuk setiap admin.
3. Di server, sebagai root: `mkdir -p /opt/ops`. Isi `/opt/ops/.env` (format di `ops/.env.example`) dan `/opt/ops/rclone.conf` (remote `offsite`, 06 §8) dari password manager.
4. `scp -r deploy root@<ip>:`, lalu `ssh root@<ip> bash deploy/bootstrap.sh`. Login root dimatikan setelah run ini; run berikutnya memakai `sudo bash deploy/bootstrap.sh` sebagai admin. Bila ada input yang kurang, skrip berhenti dan menyebutkannya.
5. Langkah manual sekali (06 §9):
   - GitLab: password root awal ada di `/opt/ops/gitlab/config/initial_root_password` dan hilang setelah 24 jam. Simpan `gitlab-secrets.json` dari folder yang sama ke password manager (06 §7).
   - Runner `ci`: buat runner instance di GitLab dengan tag `ci`, lalu jalankan `cd /opt/ops && docker compose exec gitlab-runner gitlab-runner register --url https://gitlab.<domain> --token <token> --executor docker --template-config /etc/gitlab-runner/ci.template.toml`.
   - Runner `deploy`: buat runner dengan tag `deploy` dan centang *Protected*, lalu jalankan `sudo gitlab-runner register --url https://gitlab.<domain> --token <token> --executor shell`.
   - Pipeline schedule `backup`, uptime monitor eksternal (09 AL-01), lalu role, tabel, dan kiriman pertama `gmaps-db` (09 RB-15).
6. Cek hasilnya: `cd /opt/ops && docker compose ps` menunjukkan semua service jalan, dan `https://gitlab.<domain>` serta `https://grafana.<domain>` (dari IP admin) bisa dibuka.

Perubahan stack `/opt/ops` masuk lewat MR ke `deploy/ops/` (hanya di-merge founder, 06 §3). Setelah itu jalankan ulang `bootstrap.sh`; pipeline tidak me-restart stack ini (06 §2).

## CI/CD (founder, sekali)

Di GitLab, project ini:
1. **Settings → Repository**: protected branch `main` dan protected tag `v*` sesuai 06 §3.
2. **Settings → Repository → Deploy tokens**: buat token bernama persis `gitlab-deploy-token` dengan scope `read_registry`. GitLab memberikannya ke job sebagai `CI_DEPLOY_USER`/`CI_DEPLOY_PASSWORD`, dan `deploy.sh` memakainya untuk `docker login`.
3. **Settings → CI/CD → Variables**: untuk setiap key di `.env.example` yang nilainya berisi `CHANGE_ME`, buat variabel bernama sama dua kali, dengan environment scope `staging` dan `production` (jangan "All"), Protected, dan Masked bila secret. Key opsional (nilai kosong) dibuat bila dipakai, mis. `PGBACKREST_REPO1_CIPHER_PASS` hanya di `production`. Bila ada yang kurang, job deploy berhenti dan menyebut namanya.
4. **Settings → CI/CD → General pipelines**: biarkan "Prevent outdated deployment jobs" dan "Allow job retries for rollback deployments" aktif (rollback RB-02 = Run ulang job deploy lama).
