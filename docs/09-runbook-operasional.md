# 09 — Runbook Operasional

Prosedur operasi F1 untuk `production` dan `staging`. Keduanya berjalan di **satu server** bersama GitLab dan monitoring (D-19, D-26). Versi 0.3 (lean) · 2026-10-11: data dibaca dari DB gmaps (D-31). Stack dan nama mengikuti [README](README.md). Pipeline, runner, batas resource, bootstrap server, dan konfigurasi backup ada di [06](06-ci-cd.md). Dokumen ini hanya berisi apa yang dikerjakan saat operasi.

**Konvensi.** Placeholder ditulis `<…>`. Waktu dalam WIB. Apa pun yang bisa diubah lewat GitLab atau halaman ops diubah lewat sana, bukan lewat SSH, termasuk pengaturan AI, plan, dan parameter estimasi (D-29, D-32). Perintah di server dijalankan dari direktori Compose (§1.1). `msp` dijalankan di image aplikasi: `docker compose run --rm worker msp <subcommand>`.

## 1. Inventaris
### 1.1 Server
Satu dedicated server di Indonesia. Semua domain lewat satu `caddy` di `/opt/ops`.

| Direktori Compose | Service | URL |
|---|---|---|
| `/opt/msp` (production) | `web`, `api`, `worker`, `kratos`, `postgres` (+ pgBackRest, stanza `msp`) | `https://app.<domain>` · health `/healthz`, `/readyz` · halaman ops `/ops` |
| `/opt/msp-staging` (staging) | sama dengan production | `https://staging.<domain>` |
| `/opt/ops` | `caddy`, `gitlab` (+ container registry), `gitlab-runner` (runner `ci`), `prometheus`, `alertmanager`, `grafana`, `node-exporter` | `https://gitlab.<domain>`, `https://registry.<domain>`, `https://grafana.<domain>` |
| host (systemd) | `gitlab-runner` (runner `deploy`, shell, user `deploy`) | — |

Direktori dibuat oleh `deploy/bootstrap.sh` (06 §9). Prometheus mengambil `/metrics` production dan staging lewat network Docker (06 §2), serta metrik host dari `node-exporter` (disk, CPU, RAM). Alertmanager mengirim alert ke email dan Telegram `<grup-ops>`.

Monitoring ini ada di server yang sama, jadi server yang mati hanya bisa terdeteksi dari luar. Untuk itu dipakai **uptime monitor eksternal gratis** (mis. UptimeRobot). Layanan ini memeriksa `https://app.<domain>/readyz` dan `https://gitlab.<domain>` dengan interval terpendek paketnya (umumnya 5 menit), lalu mengirim email ke founder dan `<engineer>` (AL-01).

### 1.2 Dependensi eksternal
| Dependensi | Dipakai untuk | Bila gangguan |
|---|---|---|
| **DB gmaps** (Postgres **milik founder**, di luar server ini; founder juga menjalankan scraper-nya) | Sumber data: job `import` harian dan `msp tool import --country ID` (RB-03), lewat koneksi keluar ber-TLS dengan akun read-only `MSP_SOURCE_DATABASE_URL`. Sisi gmaps mungkin meng-allowlist IP server (06 §2, Q-12) | Import gagal (AL-10) atau data berhenti bertambah (AL-11). App tetap jalan dengan data terakhir dan tanggalnya. Kabari founder |
| GeoAPI (`geoapi.mcpke.dev`) | `msp tool geo-sync --country ID`, vector tiles peta | Peta tanpa batas wilayah; tabel peringkat tetap jalan. Tunda geo-sync |
| Claude (Anthropic API) | AI brief (provider dan model diatur di Pengaturan AI, RB-14) | Pindah ke fallback bila fallback aktif (AL-08) |
| API OpenAI-compatible (fallback) | AI brief bila provider utama gagal | Keduanya gagal → kill switch AI (RB-07) |
| SMTP `<penyedia>` | Verifikasi email, undangan, email alert | User baru tidak bisa verifikasi; alert tetap lewat Telegram |
| Google OAuth | Login Google | Login email + password tetap jalan |
| Object storage S3-compatible off-site `<penyedia>` (Indonesia) | pgBackRest (backup + WAL) dan backup GitLab | WAL menumpuk di disk (AL-06); job `backup` merah |
| Uptime monitor eksternal `<penyedia>` | AL-01 (server/situs mati) | Server mati tidak terdeteksi otomatis; andalkan cek harian (§6) |

### 1.3 Log & perintah umum
Log aplikasi berformat JSON (`slog`) di stdout container. Docker merotasi log berdasarkan ukuran, jadi **simpan log ke file segera saat insiden**. Perintah yang sama berlaku di `/opt/msp-staging` (staging) dan `/opt/ops` (GitLab, monitoring, Caddy).

```bash
cd /opt/msp
docker compose ps                                              # status & health semua service
docker compose logs -f --since 30m api worker                  # ikuti log
# L1: error & warning satu service
docker compose logs --no-log-prefix --since 1h <service> | jq -cR 'fromjson? | select(.level=="ERROR" or .level=="WARN")'
# L2: pesan error API terbanyak
docker compose logs --no-log-prefix --since 30m api | jq -rR 'fromjson? | select(.level=="ERROR") | .msg' | sort | uniq -c | sort -rn | head
docker compose logs --timestamps --since 24h > ~/insiden-$(date +%F-%H%M).log   # simpan bukti
docker compose exec postgres psql -U postgres -d <db>          # psql production
docker stats --no-stream                                       # CPU/RAM per container, semua project
sudo journalctl -u gitlab-runner --since 1h                    # log runner deploy
```

Antrean River (`import`, `score`, `export`, `webhook`, `ai`, `billing`) diperiksa di psql:
```sql
-- Q1: isi antrean
SELECT queue, state, count(*), min(scheduled_at) AS tertua FROM river_job
WHERE state IN ('available','running','retryable','discarded') GROUP BY 1, 2 ORDER BY 1, 2;
-- Q2: job gagal 24 jam terakhir + contoh error
SELECT queue, kind, count(*), (array_agg(errors->-1->>'error'))[1] AS contoh_error FROM river_job
WHERE state = 'discarded' AND finalized_at > now() - interval '24 hours' GROUP BY 1, 2;
```

## 2. Akses
- SSH hanya dengan kunci, dari IP di allowlist firewall. Tanpa login password dan tanpa login `root`. Tiap orang punya user sendiri.
- Agent AI tidak punya akses SSH. Mereka bekerja lewat GitLab (MR dan pipeline) dan tidak bisa menjalankan job production.
- Semua akun web operasional wajib MFA, kecuali login admin bawaan Grafana. Sebagai gantinya, Grafana hanya bisa diakses dari IP allowlist yang sama dengan SSH, dan password-nya panjang serta disimpan di password manager founder. Prometheus dan Alertmanager tidak dibuka ke internet: `ssh -L 9090:localhost:9090 -L 9093:localhost:9093 <user>@<server>`.
- Salinan semua secret production disimpan di password manager founder, sebagai cadangan bila GitLab hilang (RB-11).
- **Orang keluar:** hapus kuncinya dari `~/.ssh/authorized_keys` dan IP-nya dari allowlist. Nonaktifkan akun GitLab dan ops-nya, dan keluarkan ia dari penerima email uptime monitor. Ganti password admin Grafana, lalu rotasi secret yang pernah ia pegang (RB-12).

| Akses | Siapa | Catatan |
|---|---|---|
| SSH server | Founder, `<engineer>` | Darurat dan prosedur yang butuh shell |
| GitLab: jalankan deploy production | Founder | D-22. MFA |
| GitLab: Developer | `<engineer>`, akun bot agent AI | MFA untuk akun manusia |
| Halaman ops `/ops` | Founder, `<staf-ops>` | Akun ops, MFA TOTP (FR-AUTH-03). Termasuk Pengaturan AI, plan, dan parameter estimasi |
| Grafana | Founder, `<engineer>` | Login admin bawaan Grafana lewat HTTPS |
| Panel penyedia: hosting, DNS, storage backup, uptime monitor, Anthropic, LLM fallback, SMTP, Google Cloud | Founder | MFA |
| DB gmaps (akun read-only, allowlist IP) | Founder | Dikelola di sisi gmaps, bukan dari server ini |

## 3. Prosedur
### RB-01 — Deploy production
**Kapan:** tag `vX.Y.Z` lulus pipeline, sudah diverifikasi di staging, dan founder setuju.
1. Pastikan tidak ada alert aktif dan RB-09 hijau.
2. GitLab → **Build → Pipelines** → pipeline tag `vX.Y.Z` → job manual `deploy-production` → **Run**. Runner `deploy` menjalankannya langsung di server: menulis `.env`, lalu `docker compose pull` → `msp migrate` → `docker compose up -d --wait` di `/opt/msp` (06).
3. Gagal di `msp migrate`: versi lama tetap jalan. Baca log job, perbaiki lewat rilis baru. Gagal di `up --wait`: RB-02.

**Verifikasi:** `curl -fsS https://app.<domain>/readyz`; `docker compose ps` semua `healthy`. Smoke test di workspace uji `<workspace-uji>`: login email dan Google → search → profil (estimasi dan pola keramaian tampil dengan tanggal data) → analisis lokasi baru → unlock 1 prospek → export kecil → webhook ke `<endpoint-uji>` → AI brief → `/ops` terbuka dengan MFA. Pantau 5xx dan p95 di Grafana selama 30 menit.

### RB-02 — Rollback
**Kapan:** setelah deploy, error naik atau ada bug kritis yang tidak bisa diperbaiki dalam 30 menit.
1. GitLab → pipeline tag rilis sebelumnya → job deploy production → **Run** lagi. Skema DB tidak di-rollback; `msp migrate` versi lama tidak menjalankan apa pun.
2. Versi lama tidak cocok dengan skema baru (migrasi tidak kompatibel-mundur) → jangan rollback; buat tag hotfix. Data rusak → RB-10.

**Verifikasi:** `docker compose ps --format '{{.Service}} {{.Image}}'` menunjukkan sha tag lama (`git rev-list -n1 <tag-lama>`); `/readyz` OK; 5xx dan p95 kembali normal.

### RB-03 — Import dari DB gmaps
**Kapan:** otomatis tiap hari lewat job `import` di production dan staging (masing-masing ke DB-nya sendiri; selesai sebelum jam kerja, NFR-02). Manual untuk import pertama (instalasi baru, setelah RB-04), setelah AL-10 atau AL-11 beres, setelah perbaikan pemetaan kolom, atau bila founder minta data terbaru segera.

Import membaca snapshot di DB gmaps secara inkremental berdasarkan waktu scrape, hanya untuk negara aktif (`--country ID`; negara lain dilewati dan dihitung), lalu upsert per ID tempat. Membaca ulang rentang yang sama tidak membuat duplikat (FR-DATA-01). Tidak ada file dataset, dan MesinProspek tidak melakukan scraping. Detail di 04.

1. **Harian:** tidak ada yang dijalankan. Di cek harian (§6), buka `/ops/imports`: import terakhir selesai, jeda snapshot terakhir ≤ 2 hari.
2. **Manual.** Pastikan tidak ada job di antrean `import` yang `running` (Q1). Jalankan di `tmux` agar tidak putus saat SSH putus, mulai dari staging:
   ```bash
   ssh -t <user>@<server> tmux new -s import
   cd /opt/msp-staging && docker compose run --rm worker msp tool import --country ID
   ```
   Bila laporan staging masuk akal (langkah 4), jalankan perintah yang sama di `/opt/msp`.
3. **Import pertama** membaca seluruh riwayat DB gmaps, jadi bisa berjam-jam.
   - Kabari founder dulu karena import ini membebani DB gmaps. Jalankan di jam sepi dan cek `df -h` (DB staging dan production di disk yang sama, RB-13).
   - Import ini **bisa dilanjutkan**: bila terputus (SSH, restart container, deploy), jalankan ulang perintah yang sama. Import melanjutkan dari titik terakhir tanpa duplikat.
   - Selesaikan staging dulu, baru production; jangan bersamaan. Selama berjalan, AL-03 dan AL-04 (`import`, `score`) wajar.
4. **Baca laporan** (output tool dan `/ops/imports`): **snapshot baru**, **tempat baru**, **berubah**, **dilewati (negara lain)**, **ditolak per alasan**, serta snapshot terakhir dan jedanya.
   - Dilewati (negara lain) bukan error; DB gmaps berisi seluruh dunia.
   - Ditolak > 5% dari snapshot Indonesia, atau ada alasan yang tidak dikenal → kirim laporan dan contohnya ke engineering sebelum import berikutnya.
   - Snapshot baru = 0 berhari-hari padahal import sukses → AL-11.
5. **Skema DB gmaps berubah.** Import memeriksa kolom sumber saat mulai dan **gagal cepat** dengan pesan yang menyebut kolom yang hilang atau berubah. Tidak ada data yang ditulis, dan app tetap jalan dengan data terakhir (FR-DATA-01, PRD §13).
   1. Simpan pesan error (`/ops/imports` atau L1 untuk `worker`), lalu tanyakan founder apa yang berubah di sisi gmaps.
   2. Engineering memperbaiki **satu file pemetaan kolom** (04) lewat MR biasa → `main` (staging otomatis) → tag patch → RB-01.
   3. Jalankan import manual (langkah 2) di staging lalu production. Karena import gagal sebelum membaca data, tidak ada snapshot yang terlewat.

Hitung ulang (rantai di README: estimasi → metrik wilayah → opportunity → skor) berjalan otomatis setelah import; tidak perlu RB-05.

**Verifikasi:** `/ops/imports` berstatus selesai, termasuk hitung ulang; `/ops/health`: jumlah bisnis bertambah tepat sebesar **tempat baru** (tidak ada duplikat), % terpetakan kelurahan ≥ 98% (FR-DATA-04), dan setelah import pertama % bisnis dengan ≥ 2 snapshot berjarak ≥ 30 hari terisi (FR-OPS-01); antrean `import` dan `score` kosong (Q1).

### RB-04 — Sync wilayah GeoAPI
**Kapan:** instalasi baru (sebelum import pertama), GeoAPI merilis data baru, atau % terpetakan wilayah turun.
1. Di `tmux`, jalankan untuk staging lalu production: `cd /opt/msp-staging && docker compose run --rm worker msp tool geo-sync --country ID`, lalu perintah yang sama di `/opt/msp`. Tool menahan diri di batas GeoAPI 120 request/menit, jadi prosesnya bisa lama. Bila terputus, jalankan ulang.
2. Setelah selesai, metrik wilayah dan skor dihitung ulang otomatis; tidak perlu RB-05. Pantau antrean `import` dan `score` (Q1) sampai kosong.

**Verifikasi:** laporan yang dicetak tool menampilkan jumlah area per level (provinsi, kab/kota, kecamatan, desa/kelurahan) tanpa error; `/ops/health` % terpetakan ≥ 98%; peta menampilkan atribusi GeoAPI.

### RB-05 — Rescore
**Kapan:** hanya setelah rilis membawa versi model/bobot baru (`est-X.Y.Z`, `priority-X.Y.Z`, `geo-X.Y.Z`; perbaikan bug hitungan juga menaikkan versi). Setelah import (RB-03), geo-sync (RB-04), dan perubahan parameter estimasi (RB-14), hitung ulang sudah otomatis.
1. Jalankan di jam sepi karena beban DB naik: `cd /opt/msp && docker compose run --rm worker msp tool rescore`. Tool menjalankan rantai yang sama (README): estimasi → metrik wilayah → opportunity → skor.
2. Pantau antrean `import` dan `score` (Q1) sampai kosong. Ada job `discarded` → AL-05.

**Verifikasi:** `/ops/health` menampilkan versi model aktif yang baru; 3 profil yang dikenal (mis. Signa Cafe) menampilkan estimasi atau breakdown skor dengan versi baru dan nilai yang masuk akal; p95 kembali normal.

### RB-06 — Plan workspace, grant credit, & refund manual
**Kapan:** workspace membayar plan atau top-up (billing F1 manual), kompensasi, atau credit terpotong karena kesalahan. Credit bulanan sesuai plan diberikan otomatis oleh job `billing.monthly_grant` (FR-BILL-01, 04 §12); jangan di-grant manual.
1. Plan: `/ops/workspaces` → `<workspace>` → pasang plan. Credit bulan berjalan masuk otomatis paling lambat 1 jam kemudian. Ganti plan di tengah bulan tidak memberi grant kedua; bila founder memutuskan selisihnya diberikan, pakai langkah 2.
2. Grant: `/ops/workspaces` → `<workspace>` → **Grant credit**: jenis (credit / AI credit), jumlah, dan alasan berisi nomor invoice atau tiket.
3. Refund: di ledger workspace pilih entri debit → **Refund** + alasan. Jangan pernah mengubah ledger lewat SQL.
4. Credit bulanan tidak masuk > 1 jam: cek antrean `billing` (Q1/Q2, AL-04/05) dan teruskan ke engineering. **Jangan** menggantinya dengan grant manual, karena job yang pulih akan tetap memberi grant sehingga credit menjadi dobel.

**Verifikasi:** saldo workspace berubah sesuai jumlah; entri ledger dan audit (`/ops/audit`) tercatat atas nama Anda (grant bulanan: aktor = sistem).

### RB-07 — Feature flag & kill switch AI
**Kapan:** provider AI gagal dan perubahan di Pengaturan AI (RB-14) tidak menolong (AL-08), biaya AI melonjak, output AI bermasalah, atau export/webhook bermasalah (bug, banjir request, dugaan kebocoran).
1. `/ops/flags` → `AI`, `export`, atau `webhook` → **Off** + alasan. Berlaku langsung tanpa deploy; app customer menampilkan banner FITUR DIMATIKAN.
2. Mati > 30 menit → kirim pesan status ke workspace terdampak (§5). Penyebab sudah beres → **On** lagi.

**Verifikasi:** di workspace uji kontrol fitur nonaktif; tidak ada job baru di antrean terkait (Q1); perubahan flag ada di `/ops/audit`. Setelah On: satu AI brief, export, atau webhook uji berhasil.

### RB-08 — Permintaan suppression
**Kapan:** seseorang meminta nilai kontaknya (telepon, URL) tidak ditampilkan.
1. Catat di `<tracker>`: tanggal, pemohon, nilai, tautan prospek.
2. `/ops/suppression` → tambah nilai + alasan + nomor tiket. Lalu balas pemohon.
3. Permintaan lain (akses, penghapusan, keberatan, ancaman hukum) → teruskan ke tim legal dan ikuti instruksinya. Jangan menjanjikan apa pun.

**Verifikasi:** di workspace uji yang sudah unlock prospek itu, nilai tidak tampil di profil ("Kontak tidak tersedia"), file export, maupun payload webhook. Entri tercatat di `/ops/audit`.

### RB-09 — Cek backup
**Kapan:** harian, saat AL-06, dan sebelum deploy.
```bash
cd /opt/msp
docker compose exec -u postgres postgres pgbackrest --stanza=msp info
docker compose exec -u postgres postgres pgbackrest --stanza=msp check
docker compose exec postgres psql -U postgres -c "SELECT last_archived_time, last_failed_time, failed_count FROM pg_stat_archiver;"
```
**Verifikasi:** `info` berstatus `ok` dengan backup terakhir sesuai jadwal 06 (≤ 26 jam); `check` selesai tanpa error; `last_archived_time` ≤ 15 menit lalu (RPO, NFR-04) dan tidak ada `last_failed_time` yang lebih baru; job `backup` terakhir (pipeline *schedule*) hijau, artinya pgBackRest dan backup GitLab ter-upload off-site.

### RB-10 — Restore PITR ke DB sementara atau server baru
**Kapan:** data rusak atau terhapus, server hilang (dari RB-11), atau uji restore bulanan.

**Aturan:** jangan menimpa DB yang masih jalan. Di project sementara start **hanya `postgres`**: `api`, `web`, dan `kratos` akan bergabung ke network production (06 §2), dan `worker` akan mengirim webhook, email, dan panggilan AI sungguhan.

1. Tentukan waktu target sebelum kejadian (dari `/ops/audit` atau log). Untuk RB-11 tanpa target waktu, hapus opsi `--type` dan `--target*` agar DB pulih sampai WAL terakhir.
2. Siapkan direktori:
   - **Uji atau data rusak:** project sementara di server yang sama, dengan volume sendiri dan tanpa menyentuh production maupun staging. Cek dulu `df -h`, karena butuh ruang ± ukuran DB production, dan jalankan di jam sepi. Siapkan dengan `sudo cp -a /opt/msp /opt/msp-restore`, lalu tambahkan `COMPOSE_FILE=compose.yaml:compose.staging.yaml` ke `.env`-nya (batas resource staging).
   - **Server baru (RB-11):** `/opt/msp` berisi file `deploy/` dari tag yang berlaku (`docker-compose.prod.yml` sebagai `compose.yaml`, `kratos/`, `postgres/`) dan `.env` production dari password manager founder.

   Lalu jalankan:
```bash
cd /opt/msp-restore                           # RB-11: /opt/msp, tanpa --archive-mode=off
docker compose run --rm --no-deps -u postgres postgres pgbackrest --stanza=msp --archive-mode=off \
  --type=time --target="<YYYY-MM-DD HH:MM:SS+07>" --target-action=promote restore
docker compose up -d --wait postgres          # RB-11: lewati; deploy yang menyalakan semua service
```
3. Butuh sebagian data saja (mis. baris terhapus)? Ambil dari DB hasil restore bersama engineering (`pg_dump -t`), jangan restore penuh di atas production.

**Verifikasi:** `SELECT pg_is_in_recovery(), pg_last_xact_replay_timestamp();` → `f` dan waktu mendekati target; `SELECT max(version_id) FROM goose_db_version;` dan `SELECT count(*)` beberapa tabel utama cocok dengan production. Uji bulanan: catat durasi (RTO ≤ 8 jam) dan titik pulih (RPO ≤ 15 menit) di `<log-ops>`, lalu `docker compose down -v` dan `sudo rm -rf /opt/msp-restore`.

### RB-11 — Bangun ulang server
**Kapan:** server rusak, hilang, atau terkompromi. Production, staging, GitLab, dan monitoring semuanya ikut mati: SEV-1, RTO ≤ 8 jam (NFR-04). Bila server terkompromi: jangan hapus server lama (bukti), jalankan langkah kebocoran data (§5), dan rotasi semua secret (RB-12) sebelum deploy.
1. Pesan server baru di Indonesia dengan spesifikasi yang sama. Turunkan TTL DNS `<domain>` ke 300 detik bila masih sempat.
2. Bootstrap dari clone repo (06 §9), dengan `/opt/ops/.env` dari password manager founder.
3. Arahkan DNS semua subdomain `<domain>` ke IP baru. Ini dilakukan sebelum langkah berikutnya karena Caddy butuh DNS untuk sertifikat TLS, dan runner serta registry diakses lewat domain. `app.<domain>` belum melayani sampai langkah 6. Minta founder memasukkan IP baru ke allowlist DB gmaps bila sisi gmaps memakainya (Q-12).
4. Pulihkan GitLab dari backup off-site terbaru (`gitlab-backup restore`), dengan `gitlab-secrets.json` dan `gitlab.rb` dari password manager (06 §8). Lalu daftarkan ulang runner `ci` dan `deploy`. Tanpa backup: buat GitLab baru, push repo dari clone terbaru, lalu isi ulang variabel CI dari password manager.
5. Restore DB dengan RB-10 di `/opt/msp`, hanya langkah `restore`, tanpa target waktu dan tanpa `--archive-mode=off`.
6. Retry `deploy-production` di pipeline tag yang sedang berlaku (RB-01). Lalu jalankan pipeline `main` untuk staging; DB staging kosong, jadi jalankan RB-04 lalu import pertama RB-03 bila perlu.
7. Backup penuh: `cd /opt/msp && docker compose exec -u postgres postgres pgbackrest --stanza=msp --type=full backup`.

**Verifikasi:** smoke test RB-01 lulus, RB-09 hijau, uptime monitor eksternal hijau, semua target Prometheus hijau, dan import manual RB-03 di production berhasil terhubung ke DB gmaps.

### RB-12 — Rotasi secret
**Kapan:** pemegang secret keluar, secret bocor atau diduga bocor (segera), atau diminta penyedia. Provider, model, dan base URL AI bukan secret; ubah di RB-14. Yang dirotasi di sini adalah API key (`MSP_ANTHROPIC_API_KEY`, `MSP_LLM_FALLBACK_API_KEY`), kredensial DB gmaps (`MSP_SOURCE_DATABASE_URL`), dan secret lain di `.env`.
1. Buat nilai baru di penyedia; biarkan nilai lama aktif dulu.
2. GitLab → **Settings → CI/CD → Variables** → ubah variabel `MSP_…` terkait (Protected, Masked). Perbarui salinan di password manager founder.
3. Jalankan ulang job deploy production untuk tag yang sedang berlaku. `.env` ditulis ulang (izin 600) dan container yang berubah dibuat ulang.
4. Setelah verifikasi, cabut nilai lama di penyedia.

Kasus khusus: password DB → `ALTER ROLE … PASSWORD …` lalu langsung langkah 2–3 di jam sepi (error singkat di antaranya). `MSP_SOURCE_DATABASE_URL` → founder membuat password atau akun read-only baru di sisi gmaps (nilai lama tetap aktif); langkah 2 untuk scope `staging` dan `production`; langkah 3 untuk keduanya (staging: Retry job `deploy-staging` di pipeline `main` terakhir); verifikasi dengan import manual RB-03 di keduanya; baru founder mencabut nilai lama. `MSP_ENCRYPTION_KEY` → jangan dirotasi tanpa rencana re-enkripsi dari engineering. Secret cookie Kratos → tambahkan nilai baru di depan daftar dan simpan yang lama agar user tidak logout massal. Secret `/opt/ops/.env` (Grafana, SMTP/Telegram Alertmanager) → ubah lewat SSH, `cd /opt/ops && docker compose up -d`, perbarui password manager.

**Verifikasi:** `stat -c '%a %y' /opt/msp/.env` → `600` dengan waktu baru; fitur pemakai secret berfungsi (AI brief, email, login Google, `pgbackrest check`, import RB-03); status key di Pengaturan AI "Terpasang".

### RB-13 — Disk hampir penuh
**Kapan:** AL-07, atau sebelum import pertama (RB-03). Semua project memakai disk yang sama.
```bash
df -h && docker system df
sudo du -xh --max-depth=3 /var/lib/docker /opt /home 2>/dev/null | sort -h | tail -15
docker image prune -af --filter "until=336h"     # image > 14 hari; bisa di-pull ulang untuk rollback
docker builder prune -af --filter "until=168h"   # cache build
docker volume prune -af --filter "label=com.gitlab.gitlab-runner.managed=true"   # hanya volume cache runner CI
sudo journalctl --vacuum-size=500M
```
Registry GitLab: `cd /opt/ops && docker compose exec gitlab gitlab-ctl registry-garbage-collect -m` (registry read-only sebentar, jangan saat pipeline jalan). Masih penuh → DB staging boleh dikosongkan (`cd /opt/msp-staging && docker compose down -v`), lalu diisi lagi lewat pipeline `main` + RB-04 + RB-03 saat ruang cukup. Tidak pernah untuk production.

**Jangan pernah** menghapus isi data Postgres atau `pg_wal`, dan jangan `docker volume prune` tanpa filter label runner. `pg_wal` yang membesar berarti arsip WAL gagal → AL-06. Bila disk tetap > 80% setelah dibersihkan, tambah disk (keputusan founder).

**Verifikasi:** `df -h` < 70%.

### RB-14 — Ubah Pengaturan AI, plan, atau parameter estimasi
**Kapan:** ganti provider, model utama, atau fallback AI (biaya, kualitas, atau provider gagal, AL-08); ubah plan (harga tampilan, seat, credit dan AI credit bulanan); atau ubah parameter estimasi (review rate, faktor belanja, benchmark harga, benchmark hotel, ambang penanda; daftar di PRD Lampiran E §E.5). Semuanya diatur dari halaman ops (D-29, D-32), berlaku tanpa deploy, dan tercatat di audit.
1. **AI:** halaman ops → **Pengaturan AI** → ubah provider/model/fallback → **Uji koneksi** → Simpan. Brief baru memakai nilai ini dalam ≤ 1 menit. API key tidak bisa diubah di sini; key baru atau provider baru butuh RB-12 dulu. Status key "Belum diset" berarti tombol Simpan nonaktif. Untuk ganti model yang tidak mendesak, uji dulu di staging (halaman ops staging + satu AI brief di workspace uji).
2. **Plan:** `/ops/plans` → ubah atau tambah plan. Plan tidak dihapus; nonaktifkan agar tidak bisa dipilih lagi. Perubahan credit bulanan berlaku pada grant otomatis bulan berikutnya. Selisih bulan berjalan lewat RB-06 bila founder memutuskan. Memasang plan ke workspace dilakukan di RB-06.
3. **Parameter estimasi:** ubah → hitung ulang → verifikasi, selalu di staging dulu, karena angka yang dilihat customer ikut berubah.
   1. Halaman ops staging → **Parameter estimasi** → ubah nilai → Simpan. Catat sumber nilai baru di `<log-ops>`. Hitung ulang (rantai di README) berjalan otomatis (FR-EST-06); jangan jalankan RB-05.
   2. Tunggu antrean `import` dan `score` kosong (Q1).
   3. Buka profil **Signa Cafe** (kasus golden PRD §12) di staging, atau profil lain di kelompok estimasi yang diubah. Parameter yang tampil harus nilai baru, dan angka bergeser sesuai rumus Lampiran E §E.2, mis. review rate tengah 1,5% → 1% membuat kunjungan dan omzet tengah naik 1,5×; faktor belanja 0,8 → 0,9 hanya menaikkan omzet (×1,125). Untuk benchmark hotel, buka satu profil hotel di kelas akomodasi yang diubah.
   4. Ulangi langkah 1–3 di production, di jam sepi karena hitung ulang membebani DB.

   Benchmark yang dikosongkan membuat estimasi yang bergantung padanya tampil "Belum dapat diperkirakan" (BR-EST-01). Itu perilaku yang benar, bukan error.

**Verifikasi:** perubahan ada di `/ops/audit` (nilai lama → baru). AI: satu AI brief di workspace uji berhasil dan L1 `worker` tanpa error provider. Plan: nilai baru tampil di `/ops/plans`. Parameter estimasi: langkah 3.3 lulus di staging dan production, dan antrean `import` dan `score` kosong.

## 4. Alert
Aturan alert ada di `deploy/ops/` (Prometheus); AL-01 berasal dari uptime monitor eksternal. Ambang di bawah adalah acuannya. Level SEV di §5; L1, L2, Q1, Q2 di §1.3.

| ID | Alert & ambang | Dampak | Langkah pertama |
|---|---|---|---|
| AL-01 | **Situs/server down** (uptime monitor eksternal, email): `https://app.<domain>/readyz` atau `https://gitlab.<domain>` gagal 2× berturut | App: semua user, SEV-1. Hanya GitLab: deploy dan CI berhenti, SEV-3 | Keduanya gagal → server, jaringan, atau `caddy`: SSH lalu `cd /opt/ops && docker compose ps caddy`. Tidak bisa SSH → console/reboot di panel hosting; rusak permanen → RB-11. Hanya app → `cd /opt/msp && docker compose ps`; service mati/unhealthy → L1 → `docker compose up -d <service>`. `/healthz` OK tetapi `/readyz` gagal → DB: log `postgres`, disk (RB-13). Baru deploy → RB-02. Hanya GitLab → `cd /opt/ops && docker compose ps gitlab`, L1 |
| AL-02 | **5xx tinggi:** > 5% respons API 5xx selama 10 menit | Sebagian aksi gagal. SEV-2, SEV-1 bila > 50% | L2. Mulai setelah deploy → RB-02. Error DB → langkah DB di AL-03. Error login → L1 untuk `kratos` |
| AL-03 | **p95 di atas NFR-01:** search > 1 s atau profil > 500 ms selama 15 menit | App lambat. SEV-3, SEV-2 bila > 2× target | Ada import, rescore, atau hitung ulang parameter yang jalan → wajar sementara; pindahkan ke jam sepi bila mengganggu. CPU/RAM host jenuh → AL-09. Query macet: `SELECT pid, now()-query_start AS durasi, left(query,100) FROM pg_stat_activity WHERE state <> 'idle' ORDER BY durasi DESC LIMIT 10;`, lalu untuk yang > 5 menit dan bukan backup: `SELECT pg_cancel_backend(<pid>);`. Masih lambat → engineering (query/indeks) |
| AL-04 | **Antrean River tertahan:** job `available` tertua > 15 menit di `export`, `webhook`, `ai`, `billing`, atau > 2 jam di `import`, `score` | Export, webhook, AI brief, credit bulanan, atau skor tertunda. SEV-2 | Q1, `docker compose ps worker`, L1 untuk `worker`. Worker mati/macet → `docker compose restart worker` (job idempotent, STD-DB-06). Bila hanya `import`/`score` yang panjang karena import pertama (RB-03) atau hitung ulang parameter (RB-14) → tunggu |
| AL-05 | **Banyak job gagal:** > 10 job `discarded` dalam 1 jam di antrean mana pun | Aksi user gagal (export/kirim yang gagal bisa diulang tanpa biaya). SEV-2 | Q2. `webhook` → biasanya endpoint customer menolak/timeout: kabari customer. `ai` → AL-08. `export` → disk (RB-13) atau bug (engineering). `import` → AL-10. `billing` → engineering, jangan grant manual (RB-06). Tidak ada redrive massal di F1: setelah penyebab beres, user mengulang aksinya; ops menjalankan import manual RB-03 (aman, tanpa duplikat) |
| AL-06 | **Arsip WAL/backup gagal:** tidak ada WAL terarsip > 15 menit, `failed_count` naik, atau backup terakhir > 26 jam | RPO ≤ 15 menit (NFR-04) tidak terjamin; `pg_wal` menumpuk sampai disk penuh. SEV-2, SEV-1 bila disk > 90% | RB-09 dan L1 untuk `postgres`. Penyebab umum: storage off-site tidak terjangkau, kredensial kedaluwarsa (RB-12), kuota bucket habis, disk lokal penuh. Setelah beres: `docker compose exec -u postgres postgres pgbackrest --stanza=msp backup` |
| AL-07 | **Disk penuh:** > 80% peringatan, > 90% kritis | Postgres berhenti menulis bila disk penuh → situs down. SEV-2, SEV-1 bila > 90% | RB-13 |
| AL-08 | **Provider AI gagal:** > 20% AI brief memakai fallback selama 15 menit (peringatan); > 20% gagal di provider utama dan fallback selama 10 menit (kritis) | AI brief lambat/gagal, fitur lain normal. SEV-3 / SEV-2 | Halaman status penyedia + L1 untuk `worker`. 401/403 → key salah atau dicabut (RB-12). 429 → kuota/rate limit di konsol penyedia. 5xx/timeout → di Pengaturan AI (RB-14) aktifkan fallback atau ganti provider/model utama, cek dengan **Uji koneksi**. Kritis > 30 menit → kill switch AI (RB-07); nyalakan lagi setelah pulih |
| AL-09 | **Resource host jenuh:** CPU > 85% atau RAM > 90% selama 15 menit | Production bisa melambat (AL-03). SEV-3, SEV-2 bila p95 production ikut naik | `docker stats --no-stream`. Periksa CI dan staging dulu; keduanya boleh dihentikan untuk melindungi production: `cd /opt/msp-staging && docker compose stop` dan `cd /opt/ops && docker compose stop gitlab-runner` (job CI tertunda). Nyalakan lagi dengan `start` setelah reda. Penyebab di production (import, rescore, query) → AL-03. Sering terjadi → founder (kapasitas, atau pisahkan GitLab/CI; PRD §13) |
| AL-10 | **Import gagal:** tidak ada import sukses dari DB gmaps > 26 jam (job harian gagal, macet, atau tidak jalan) | Data tidak bertambah; app tetap jalan dan estimasi menampilkan tanggal datanya. SEV-3 | Pesan error di `/ops/imports`, Q2, L1 untuk `worker`. Kolom sumber hilang/berubah → RB-03 langkah 5. Koneksi timeout/ditolak/TLS → founder: DB gmaps hidup? IP server masih di allowlist (Q-12)? Login ditolak → kredensial (RB-12). Setelah beres → import manual RB-03 |
| AL-11 | **Data sumber berhenti bertambah:** import sukses, tetapi tidak ada snapshot baru (Indonesia) > 2 hari | Data makin menua. Penyebabnya di sisi scraper. SEV-3 | AL-10 juga aktif → tangani AL-10 dulu. Bila tidak: cek snapshot terakhir dan jeda di `/ops/imports`, lalu kabari founder (pemilik scraper dan DB gmaps). Tidak ada tindakan di server kita |

## 5. Insiden
| Level | Contoh | Respons |
|---|---|---|
| SEV-1 | Situs down, server hilang, data hilang atau rusak, dugaan kebocoran data, credit terpotong salah secara massal | Segera, kapan pun; founder dilibatkan |
| SEV-2 | Fitur utama gagal (login, export, webhook, AI), backup/WAL gagal, monitoring mati | ≤ 4 jam |
| SEV-3 | Lambat sebagian, job gagal sporadis, fallback AI aktif, import harian gagal atau data sumber berhenti bertambah, GitLab/CI mati sementara production normal | Hari kerja berikutnya |

1. **Stabilkan** dengan jalan tercepat: rollback (RB-02), kill switch (RB-07), restart service, hentikan CI/staging (AL-09), bersihkan disk. Akar masalah belakangan.
2. **Komunikasikan.** Tulis di `<grup-ops>`: apa, sejak kapan, siapa yang menangani. SEV-1/2 yang dirasakan customer → email status ke admin workspace terdampak, update tiap ≤ 2 jam sampai selesai.
3. **Perbaiki** permanen lewat MR dan rilis biasa (RB-01).
4. **Tulis singkat** (SEV-1/2, ≤ 1 halaman, ≤ 3 hari kerja): kronologi WIB, dampak (user, data, credit), penyebab, perubahan agar tidak terulang. Simpan di `<tempat-write-up>`.

```text
Subjek: [MesinProspek] Gangguan <fitur> — <tanggal>
Halo,
Sejak pukul <jam> WIB, <fitur> di MesinProspek <tidak dapat digunakan / lebih lambat dari biasanya>.
Tim kami sedang menanganinya. <Export atau kirim webhook yang gagal bisa diulang tanpa biaya.>
Kabar berikutnya paling lambat pukul <jam> WIB.
Salam, Tim MesinProspek

Subjek: [MesinProspek] Selesai: gangguan <fitur> — <tanggal>
Halo,
<Fitur> sudah normal kembali sejak pukul <jam> WIB. Mohon maaf atas ketidaknyamanannya.
Salam, Tim MesinProspek
```

**Dugaan kebocoran data** (data customer, kontak, secret, atau akses tidak sah):
1. Beri tahu founder **dan** tim legal segera, lewat telepon, tidak hanya chat.
2. Hentikan kebocoran tanpa menghapus bukti: matikan fitur terkait (RB-07), cabut akses atau kunci (RB-12), blokir IP.
3. Simpan bukti: log container (§1.3) ke file dan salin keluar server. Jangan menghapus container, volume, atau server. Audit log tetap di DB.
4. Ikuti instruksi tim legal. Jangan berkomunikasi ke luar soal kebocoran tanpa persetujuan tim legal.

## 6. Jadwal rutin
| Kapan | Tugas |
|---|---|
| Harian | RB-09, termasuk job `backup`. `/ops/imports`: import harian selesai dan jeda snapshot ≤ 2 hari (RB-03). Tinjau alert 24 jam terakhir di Grafana. Bila Grafana tidak terbuka, monitoring mati dan tidak ada yang memperingatkan → SEV-2 |
| Mingguan | `df -h` (RB-13 bila > 70%); Q2 job gagal; tren 5xx, p95, CPU, dan RAM di Grafana |
| Bulanan | Uji restore RB-10 (DB sementara di server yang sama); update paket OS lalu reboot di jam sepi (semua service ikut restart, termasuk GitLab); update image dasar (`postgres`, `caddy`, `kratos`, `gitlab`, monitoring) lewat MR, `/opt/ops` diterapkan manual; tinjau daftar akses §2 |
| Per rilis | RB-01 lengkap dengan smoke test; catat versi dan waktu deploy di `<log-ops>` |
