# 09 — Runbook Operasional

Prosedur operasi F1 untuk `production` dan `staging`. Keduanya berjalan di **satu server** bersama GitLab dan monitoring (D-19, D-26). Versi 0.4 (lean) · 2026-10-15: data dibaca dari DB gmaps (D-31), lewat `gmaps-db` yang berisi kiriman mingguan ramping terbaru dari founder (D-35, D-37). Fitur prospek (unlock, webhook, skor) ada di F2 (D-40), jadi tidak punya prosedur di sini. Stack dan nama mengikuti [README](README.md). Pipeline, runner, batas resource, bootstrap server, dan konfigurasi backup ada di [06](06-ci-cd.md). Dokumen ini hanya berisi apa yang dikerjakan saat operasi.

**Konvensi.** Placeholder ditulis `<…>`. Waktu dalam WIB. Apa pun yang bisa diubah lewat GitLab atau halaman ops diubah lewat sana, bukan lewat SSH, termasuk pengaturan AI, plan, dan parameter estimasi (D-29, D-32). Perintah di server dijalankan dari direktori Compose (§1.1). `msp` dijalankan di image aplikasi: `docker compose run --rm worker msp <subcommand>`.

## 1. Inventaris
### 1.1 Server
Satu dedicated server di Indonesia. Semua domain lewat satu `caddy` di `/opt/ops`.

| Direktori Compose | Service | URL |
|---|---|---|
| `/opt/msp` (production) | `web`, `api`, `worker`, `kratos`, `postgres` (+ pgBackRest, stanza `msp`, repo lokal `/opt/msp/pgbackrest`); one-shot `kratos-migrate` saat deploy | `https://app.<domain>` · health `/healthz`, `/readyz` · halaman ops `/ops` |
| `/opt/msp-staging` (staging) | sama dengan production | `https://staging.<domain>` |
| `/opt/ops` | `caddy`, `gitlab` (+ container registry), `gitlab-runner` (runner `ci`), `prometheus`, `alertmanager`, `grafana`, `node-exporter`, `gmaps-db` (kiriman DB gmaps terbaru, dibaca staging dan production; file kiriman terbaru di `/opt/ops/gmaps-dump/`) | `https://gitlab.<domain>`, `https://registry.<domain>`, `https://grafana.<domain>`. `gmaps-db` tanpa URL dan tanpa port |
| host (systemd) | `gitlab-runner` (runner `deploy`, shell, user `deploy`) | — |
| host (cron, root) | `msp-offsite`: `/opt/ops/offsite.sh` tiap 5 menit, rclone menyalin `/opt/msp/pgbackrest` dan `/opt/ops/gitlab-backups/` ke remote `offsite` (06 §8). Log: `sudo journalctl -t msp-offsite` | — |

Direktori dibuat oleh `deploy/bootstrap.sh` (06 §9). Prometheus mengambil `/metrics` production dan staging lewat network Docker (06 §2), serta metrik host dari `node-exporter` (disk, CPU, RAM). Alertmanager mengirim alert ke email dan Telegram `<grup-ops>`.

Monitoring ini ada di server yang sama, jadi server yang mati hanya bisa terdeteksi dari luar. Untuk itu dipakai **uptime monitor eksternal gratis** (mis. UptimeRobot). Layanan ini memeriksa `https://app.<domain>/readyz` dan `https://gitlab.<domain>` dengan interval terpendek paketnya (umumnya 5 menit), lalu mengirim email ke founder dan `<engineer>` (AL-01).

### 1.2 Dependensi eksternal
| Dependensi | Dipakai untuk | Bila gangguan |
|---|---|---|
| **Kiriman mingguan DB gmaps** dari founder (founder menjalankan scraper dan menyimpan DB aslinya) | Sumber data. Kiriman ramping (hanya Indonesia, kolom yang dipakai import, baris baru) diserahkan tiap minggu dan dimuat ke `gmaps-db` (RB-15), lalu dibaca job `import.run` harian dan `msp tool import --country ID` (RB-03) dengan role read-only `msp_source_ro`. Tidak ada koneksi jaringan ke server founder (D-35, D-37) | Kiriman tidak datang → data berhenti bertambah (AL-11). App tetap jalan dengan data terakhir dan tanggalnya. Kabari founder |
| GeoAPI (`geoapi.mcpke.dev`) | `msp tool geo-sync --country ID`, vector tiles peta | Peta tanpa batas wilayah; tabel peringkat tetap jalan. Tunda geo-sync |
| Claude (Anthropic API) | Narasi AI (provider dan model diatur di Pengaturan AI, RB-14) | Pindah ke fallback bila fallback aktif (AL-08) |
| API OpenAI-compatible (fallback) | Narasi AI bila provider utama gagal | Keduanya gagal → kill switch AI (RB-07) |
| SMTP `<penyedia>` | Verifikasi email, undangan, email alert | User baru tidak bisa verifikasi; alert tetap lewat Telegram |
| Google OAuth | Login Google | Login email + password tetap jalan |
| Server SFTP/FTPS off-site milik founder (Indonesia, D-39), remote rclone `offsite` | Salinan repo pgBackRest (backup + WAL, terenkripsi) dan backup GitLab | Backup dan arsip WAL lokal tetap jalan; salinan off-site tertinggal (AL-12). Bila server hilang selama itu, data sejak sinkron terakhir ikut hilang. Kabari founder |
| Backblaze B2, remote rclone `b2` | Salinan kedua repo pgBackRest (terenkripsi). **Nonaktif** sampai tim legal menjawab Q-19 (B2 tidak punya region Indonesia). Menyalakan: tambah remote `b2` di `/opt/ops/rclone.conf`, lalu buka komentar baris B2 di `offsite.sh` (MR, lalu salin ke `/opt/ops`; 06 §8) | — (nonaktif) |
| Uptime monitor eksternal `<penyedia>` | AL-01 (server/situs mati) | Server mati tidak terdeteksi otomatis; andalkan cek harian (§6) |

### 1.3 Log & perintah umum
File `.env` berizin 600 milik user `deploy`, jadi semua perintah `docker compose` di server dijalankan dengan `sudo` (admin punya sudo; contoh di bawah menghilangkan `sudo` agar ringkas). Jangan menjalankan `docker compose down` di `/opt/ops`, karena stack itu memiliki network bersama.

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

Antrean River (README: `import`, `score`, `export`, `ai`, `billing`) diperiksa di psql:
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
- **Orang keluar:** hapus kuncinya dari `~/.ssh/authorized_keys` dan IP-nya dari allowlist. Nonaktifkan akun GitLab dan ops-nya, dan keluarkan ia dari penerima email uptime monitor. Ganti password admin Grafana dengan `cd /opt/ops && sudo docker compose exec grafana grafana cli admin reset-admin-password <baru>` lalu perbarui `MSP_GRAFANA_ADMIN_PASSWORD` di `/opt/ops/.env` (Grafana hanya membaca variabel itu saat start pertama), lalu rotasi secret yang pernah ia pegang (RB-12).

| Akses | Siapa | Catatan |
|---|---|---|
| SSH server | Founder, `<engineer>` | Darurat dan prosedur yang butuh shell |
| GitLab: jalankan deploy production | Founder | D-22. MFA |
| GitLab: Developer | `<engineer>`, akun bot agent AI | MFA untuk akun manusia |
| Halaman ops `/ops` | Founder, `<staf-ops>` | Akun ops, MFA TOTP (FR-AUTH-03). Termasuk Pengaturan AI, plan, dan parameter estimasi |
| Grafana | Founder, `<engineer>` | Login admin bawaan Grafana lewat HTTPS |
| Panel penyedia: hosting, DNS, server backup SFTP/FTPS, uptime monitor, Anthropic, LLM fallback, SMTP, Google Cloud | Founder | MFA |
| `gmaps-db` (muat kiriman, role `msp_source_ro`) | Founder, `<engineer>` | Lewat SSH di `/opt/ops` (RB-15). DB gmaps asli ada di founder |

## 3. Prosedur
### RB-01 — Deploy production
**Kapan:** tag `vX.Y.Z` lulus pipeline, sudah diverifikasi di staging, dan founder setuju.
1. Pastikan tidak ada alert aktif dan RB-09 hijau.
2. GitLab → **Build → Pipelines** → pipeline tag `vX.Y.Z` → job manual `deploy-production` → **Run**. Runner `deploy` menjalankan `deploy/deploy.sh production <sha>` langsung di server, di `/opt/msp` (langkahnya di 06 §4).
3. Gagal sebelum `up` (pull, `msp migrate`, atau migrasi Kratos): container lama tetap jalan, tetapi `.env` dan `compose.yaml` di `/opt/msp` sudah menunjuk tag baru. Jalankan ulang job deploy rilis sebelumnya (RB-02 langkah 1) agar keduanya cocok lagi dengan yang jalan, baca log job, lalu perbaiki lewat rilis baru. Gagal di `up --wait`: RB-02.

**Verifikasi:** `curl -fsS https://app.<domain>/readyz`; `docker compose ps` semua `healthy`. Smoke test di workspace uji `<workspace-uji>`: login email dan Google → search → profil (estimasi dan pola keramaian tampil dengan tanggal data) → analisis lokasi baru → tambah ke list + catatan → export CSV kecil (kolom estimasi ada, kolom telepon tidak ada) → narasi AI → `/ops` terbuka dengan MFA. Pantau 5xx dan p95 di Grafana selama 30 menit.

### RB-02 — Rollback
**Kapan:** setelah deploy, error naik atau ada bug kritis yang tidak bisa diperbaiki dalam 30 menit.
1. GitLab → pipeline tag rilis sebelumnya → job deploy production → **Run** lagi. Skema DB tidak di-rollback; `msp migrate` versi lama tidak menjalankan apa pun.
2. Versi lama tidak cocok dengan skema baru (migrasi tidak kompatibel-mundur) → jangan rollback; buat tag hotfix. Data rusak → RB-10.

**Verifikasi:** `docker compose ps --format '{{.Service}} {{.Image}}'` menunjukkan sha tag lama (`git rev-list -n1 <tag-lama>`); `/readyz` OK; 5xx dan p95 kembali normal.

### RB-03 — Import dari DB gmaps
**Kapan:** otomatis tiap hari pukul 03.00 WIB lewat job `import.run` (antrean `import`) di production dan staging (masing-masing ke DB-nya sendiri; selesai sebelum jam kerja, NFR-02). Manual untuk import pertama (instalasi baru, setelah RB-04), setelah kiriman baru dimuat (RB-15) bila data perlu segera, setelah AL-10 atau AL-11 beres, setelah perbaikan pemetaan kolom, atau bila founder minta data terbaru segera.

Import membaca snapshot di `gmaps-db` (kiriman DB gmaps terbaru, RB-15) secara inkremental berdasarkan waktu scrape, hanya untuk negara aktif (`--country ID`; negara lain dilewati dan dihitung), lalu upsert per ID tempat. Membaca ulang rentang yang sama tidak membuat duplikat (FR-DATA-01). MesinProspek tidak melakukan scraping; snapshot baru hanya datang lewat kiriman mingguan dari founder, dan riwayatnya disimpan di DB MSP. Detail di 04.

1. **Harian:** tidak ada yang dijalankan. Di cek harian (§6), buka `/ops/imports`: import terakhir selesai, jeda snapshot terakhir di bawah ambang AL-11.
2. **Manual.** Pastikan tidak ada job di antrean `import` yang `running` (Q1). Jalankan di `tmux` agar tidak putus saat SSH putus, mulai dari staging:
   ```bash
   ssh -t <user>@<server> tmux new -s import
   cd /opt/msp-staging && docker compose run --rm worker msp tool import --country ID
   ```
   Bila laporan staging masuk akal (langkah 4), jalankan perintah yang sama di `/opt/msp`.
3. **Import pertama** membaca kiriman pertama (seluruh riwayat Indonesia), jadi bisa berjam-jam.
   - Import ini membebani `gmaps-db` dan DB MSP di server yang sama. Jalankan di jam sepi dan cek `df -h` (`gmaps-db`, DB staging, dan DB production di disk yang sama, RB-13). Setelah staging selesai, catat ukuran DB staging (04 §2.1); ruang untuk DB production kurang → RB-13 sebelum lanjut.
   - Import ini **bisa dilanjutkan**: bila terputus (SSH, restart container, deploy), jalankan ulang perintah yang sama. Import melanjutkan dari titik terakhir tanpa duplikat.
   - Selesaikan staging dulu, baru production; jangan bersamaan. Selama berjalan, AL-03 dan AL-04 (`import`, `score`) wajar. Kiriman mingguan berikutnya baru dimuat setelah keduanya selesai (RB-15 langkah 4).
4. **Baca laporan** (output tool dan `/ops/imports`): **snapshot baru**, **tempat baru**, **berubah**, **dilewati (negara lain)**, **ditolak per alasan**, serta snapshot terakhir dan jedanya.
   - Dilewati (negara lain) normalnya 0, karena kiriman hanya berisi baris Indonesia.
   - Ditolak > 5% dari snapshot Indonesia, atau ada alasan yang tidak dikenal → kirim laporan dan contohnya ke engineering sebelum import berikutnya.
   - Snapshot baru = 0 berhari-hari padahal import sukses → AL-11.
5. **Skema DB gmaps berubah.** Terlihat di salah satu dari tiga tempat: `export` founder gagal, pemuatan RB-15 gagal (`HEADER MATCH` atau tipe kolom), atau import **gagal cepat** saat mulai dengan pesan yang menyebut kolom yang hilang atau berubah. Tidak ada data yang ditulis, dan app tetap jalan dengan data terakhir (FR-DATA-01, PRD §13).
   1. Simpan pesan error (`/ops/imports`, L1 untuk `worker`, atau output pemuatan), lalu tanyakan founder apa yang berubah di skema DB gmaps.
   2. Engineering memperbaiki **satu file pemetaan kolom** `source.sql` (ketiga pernyataannya, 04 §5.1) lewat MR biasa → `main` (staging otomatis) → tag patch → RB-01.
   3. Buat ulang tabel `gmaps-db` dari `source.sql` baru (RB-15, sekali per volume: `DROP TABLE business_listings`, lalu `create_table` + `GRANT`). Minta founder mengirim ulang kiriman dengan `export` baru dan batas yang sama dengan kiriman yang gagal, muat (RB-15), lalu jalankan import manual (langkah 2) di staging lalu production. Karena kegagalan terjadi sebelum data dibaca, tidak ada snapshot yang terlewat.

Hitung ulang (rantai di README) berjalan otomatis setelah import; tidak perlu RB-05.

**Verifikasi:** `/ops/imports` berstatus selesai, termasuk hitung ulang; `/ops/health`: jumlah bisnis bertambah tepat sebesar **tempat baru** (tidak ada duplikat), % terpetakan kelurahan ≥ 98% (FR-DATA-04), dan setelah import pertama % bisnis dengan ≥ 2 snapshot berjarak ≥ 30 hari terisi (FR-OPS-01); antrean `import` dan `score` kosong (Q1).

### RB-04 — Sync wilayah GeoAPI
**Kapan:** instalasi baru (sebelum import pertama), GeoAPI merilis data baru, atau % terpetakan wilayah turun.
1. Di `tmux`, jalankan untuk staging lalu production: `cd /opt/msp-staging && docker compose run --rm worker msp tool geo-sync --country ID`, lalu perintah yang sama di `/opt/msp`. Tool menahan diri di batas GeoAPI 120 request/menit, jadi prosesnya bisa lama. Bila terputus, jalankan ulang.
2. Setelah selesai, penetapan wilayah dan rantai hitung ulang (README) berjalan otomatis; tidak perlu RB-05. Pantau antrean `import` dan `score` (Q1) sampai kosong.

**Verifikasi:** laporan yang dicetak tool menampilkan jumlah area per level (provinsi, kab/kota, kecamatan, desa/kelurahan) tanpa error; `/ops/health` % terpetakan ≥ 98%; peta menampilkan atribusi GeoAPI.

### RB-05 — Rescore
**Kapan:** hanya setelah rilis membawa versi model baru (`est-X.Y.Z`, `geo-X.Y.Z`; perbaikan bug hitungan juga menaikkan versi). Setelah import (RB-03), geo-sync (RB-04), dan perubahan parameter estimasi (RB-14), hitung ulang sudah otomatis.
1. Jalankan di jam sepi karena beban DB naik: `cd /opt/msp && docker compose run --rm worker msp tool rescore`. Tool menjalankan rantai yang sama (README).
2. Pantau antrean `import` dan `score` (Q1) sampai kosong. Ada job `discarded` → AL-05.

**Verifikasi:** `/ops/health` menampilkan versi model aktif yang baru; 3 profil yang dikenal (mis. Signa Cafe) menampilkan estimasi dengan versi baru dan nilai yang masuk akal; p95 kembali normal.

### RB-06 — Plan workspace, grant credit, & refund manual
**Kapan:** workspace membayar plan atau top-up (billing F1 manual), kompensasi, atau credit terpotong karena kesalahan. Credit bulanan sesuai plan diberikan otomatis oleh job `billing.monthly_grant` (FR-BILL-01, 04 §12); jangan di-grant manual.
1. Plan: `/ops/workspaces` → `<workspace>` → pasang plan. Credit bulan berjalan masuk otomatis paling lambat 1 jam kemudian. Ganti plan di tengah bulan tidak memberi grant kedua; bila founder memutuskan selisihnya diberikan, pakai langkah 2.
2. Grant: `/ops/workspaces` → `<workspace>` → **Grant credit**: jenis (credit / AI credit), jumlah, dan alasan berisi nomor invoice atau tiket.
3. Refund: di ledger workspace pilih entri debit → **Refund** + alasan. Jangan pernah mengubah ledger lewat SQL.
4. Credit bulanan tidak masuk > 1 jam: cek antrean `billing` (Q1/Q2, AL-04/05) dan teruskan ke engineering. **Jangan** menggantinya dengan grant manual, karena job yang pulih akan tetap memberi grant sehingga credit menjadi dobel.

**Verifikasi:** saldo workspace berubah sesuai jumlah; entri ledger dan audit (`/ops/audit`) tercatat atas nama Anda (grant bulanan: aktor = sistem).

### RB-07 — Feature flag & kill switch AI
**Kapan:** provider AI gagal dan perubahan di Pengaturan AI (RB-14) tidak menolong (AL-08), biaya AI melonjak, output AI bermasalah, atau export bermasalah (bug, banjir request, dugaan kebocoran).
1. `/ops/flags` → `AI` atau `export` → **Off** + alasan. Berlaku langsung tanpa deploy; app customer menampilkan banner FITUR DIMATIKAN.
2. Mati > 30 menit → kirim pesan status ke workspace terdampak (§5). Penyebab sudah beres → **On** lagi.

**Verifikasi:** di workspace uji kontrol fitur nonaktif; tidak ada job baru di antrean terkait (Q1); perubahan flag ada di `/ops/audit`. Setelah On: satu narasi AI atau export uji berhasil.

### RB-08 — Permintaan suppression
**Kapan:** seseorang meminta bisnisnya atau URL-nya (website, akun sosial) tidak ditampilkan. Telepon tidak pernah ditampilkan maupun diekspor di F1 (D-40); suppression telepon baru ada di F2, jadi permintaan atas telepon cukup dicatat di langkah 1.
1. Catat di `<tracker>`: tanggal, pemohon, nilai, tautan bisnis.
2. `/ops/suppression` → tambah bisnis atau URL + alasan + nomor tiket. Lalu balas pemohon.
3. Permintaan lain (akses, penghapusan, keberatan, ancaman hukum) → teruskan ke tim legal dan ikuti instruksinya. Jangan menjanjikan apa pun.

**Verifikasi:** di workspace uji, nilai tidak tampil di search, profil, maupun file export. Entri tercatat di `/ops/audit`.

### RB-09 — Cek backup
**Kapan:** harian, saat AL-06 atau AL-12, dan sebelum deploy.
```bash
cd /opt/msp
docker compose exec -u postgres postgres pgbackrest --stanza=msp info
docker compose exec -u postgres postgres pgbackrest --stanza=msp check
docker compose exec postgres psql -U postgres -c "SELECT last_archived_time, last_failed_time, failed_count FROM pg_stat_archiver;"
sudo rclone --config /opt/ops/rclone.conf lsl --max-age 15m offsite:msp/pgbackrest/archive | tail -3   # WAL terbaru di off-site
sudo journalctl -t msp-offsite --since 1h | tail                                                    # error rclone
```
**Verifikasi:** `info` berstatus `ok` dengan backup terakhir sesuai jadwal 06 (≤ 26 jam); `check` selesai tanpa error; `last_archived_time` ≤ 15 menit lalu (RPO, NFR-04) dan tidak ada `last_failed_time` yang lebih baru; perintah `rclone` menampilkan WAL off-site ≤ 15 menit (salinan off-site berjalan, AL-12 tidak aktif); job `backup` terakhir (pipeline *schedule*) hijau, artinya backup pgBackRest dan GitLab dibuat di disk lokal.

### RB-10 — Restore PITR ke DB sementara atau server baru
**Kapan:** data rusak atau terhapus, server hilang (dari RB-11), atau uji restore bulanan.

**Aturan:** jangan menimpa DB yang masih jalan. Di project sementara start **hanya `postgres`**: `api`, `web`, dan `kratos` akan bergabung ke network production (06 §2), dan `worker` akan mengirim email dan panggilan AI sungguhan.

1. Tentukan waktu target sebelum kejadian (dari `/ops/audit` atau log). Untuk RB-11 tanpa target waktu, hapus opsi `--type` dan `--target*` agar DB pulih sampai WAL terakhir.
2. Siapkan direktori:
   - **Uji atau data rusak:** project sementara di server yang sama, dengan volume sendiri dan tanpa menyentuh production maupun staging. Cek dulu `df -h`, karena butuh ruang ± ukuran DB production + ukuran repo pgBackRest, dan jalankan di jam sepi. Siapkan dengan `sudo mkdir /opt/msp-restore && sudo cp -a /opt/msp/{.env,compose.yaml,compose.staging.yaml,kratos,postgres} /opt/msp-restore/` (repo lokal sengaja tidak ikut), lalu tambahkan `COMPOSE_FILE=compose.yaml:compose.staging.yaml` ke `.env`-nya (batas resource staging).
   - **Server baru (RB-11):** `/opt/msp` berisi file `deploy/` dari tag yang berlaku (`docker-compose.prod.yml` sebagai `compose.yaml`, `kratos/`, `postgres/` disalin dengan `cp -rL`) dan `.env` (izin 600) yang disusun dari `deploy/.env.example` tag itu: nilai `CHANGE_ME` dan `PGBACKREST_REPO1_CIPHER_PASS` dari password manager founder, `MSP_ENV=production`, `COMPOSE_FILE=compose.yaml`, key lain sesuai contohnya. `MSP_IMAGE_REPO`/`MSP_IMAGE_TAG` boleh bernilai apa saja, karena deploy di RB-11 langkah 6 menulis ulang `.env`.
3. Ambil repo pgBackRest dari **salinan off-site** (06 §8). Uji bulanan dan gerbang F1 selalu lewat jalur ini, bukan repo lokal, agar jalur saat server hilang terbukti. Passphrase repo ada di `.env` (`PGBACKREST_REPO1_CIPHER_PASS`).
   ```bash
   cd /opt/msp-restore                         # RB-11: /opt/msp
   sudo rclone --config /opt/ops/rclone.conf copy offsite:msp/pgbackrest ./pgbackrest
   sudo chown -R 999:999 ./pgbackrest          # uid postgres di image
   ```
   Data rusak dan server masih hidup: boleh memakai repo lokal yang lebih baru beberapa menit, `sudo cp -a /opt/msp/pgbackrest .`, sebagai ganti `rclone copy`.
4. Restore:
   ```bash
   cd /opt/msp-restore                         # RB-11: /opt/msp, tanpa --archive-mode=off
   docker compose run --rm --no-deps -u postgres postgres pgbackrest --stanza=msp --archive-mode=off \
     --type=time --target="<YYYY-MM-DD HH:MM:SS+07>" --target-action=promote restore
   docker compose up -d --wait postgres        # RB-11: lewati; deploy yang menyalakan semua service
   ```
   Gagal karena file backup set terbaru tidak lengkap (server hilang saat upload backup full masih berjalan) → ulangi dengan `--set=<label sebelumnya>` dari `pgbackrest --stanza=msp info`; WAL sesudahnya tetap dipakai.
5. Butuh sebagian data saja (mis. baris terhapus)? Ambil dari DB hasil restore bersama engineering (`pg_dump -t`), jangan restore penuh di atas production.

**Verifikasi:** `SELECT pg_is_in_recovery(), pg_last_xact_replay_timestamp();` → `f` dan waktu mendekati target; `SELECT max(version_id) FROM goose_db_version;` dan `SELECT count(*)` beberapa tabel utama cocok dengan production. Uji bulanan: catat sumber repo (off-site), durasi (RTO ≤ 8 jam), dan titik pulih (RPO ≤ 15 menit) di `<log-ops>`, lalu `docker compose down -v` dan `sudo rm -rf /opt/msp-restore`.

### RB-11 — Bangun ulang server
**Kapan:** server rusak, hilang, atau terkompromi. Production, staging, GitLab, dan monitoring semuanya ikut mati: SEV-1, RTO ≤ 8 jam (NFR-04). Bila server terkompromi: jangan hapus server lama (bukti), jalankan langkah kebocoran data (§5), dan rotasi semua secret (RB-12) sebelum deploy.
1. Pesan server baru di Indonesia dengan spesifikasi yang sama. Turunkan TTL DNS `<domain>` ke 300 detik bila masih sempat.
2. Bootstrap dari clone repo (06 §9), dengan `/opt/ops/.env` dan `/opt/ops/rclone.conf` dari password manager founder. Server lama masih hidup (terkompromi)? Pakai kredensial SFTP/FTPS baru di `rclone.conf` server baru dan cabut yang lama (RB-12), agar cron `msp-offsite` di server lama tidak ikut menyinkron. Cron di server baru belum menyalin apa pun sampai langkah 5 (guard, 06 §8).
3. Arahkan DNS semua subdomain `<domain>` ke IP baru. Ini dilakukan sebelum langkah berikutnya karena Caddy butuh DNS untuk sertifikat TLS, dan runner serta registry diakses lewat domain. `app.<domain>` belum melayani sampai langkah 6.
4. Pulihkan GitLab dari salinan off-site: `sudo rclone --config /opt/ops/rclone.conf copy offsite:msp/gitlab-backups /opt/ops/gitlab-backups`, lalu di container `gitlab` `chown git:git` file terbaru dan `gitlab-backup restore BACKUP=<id>`, dengan `gitlab-secrets.json` dan `gitlab.rb` dari password manager (06 §8). Lalu daftarkan ulang runner `ci` dan `deploy`. Tanpa backup: buat GitLab baru, push repo dari clone terbaru, lalu isi ulang variabel CI dari password manager.
5. Restore DB dengan RB-10 di `/opt/msp`: langkah 3 (repo dari off-site) dan `restore`, tanpa target waktu dan tanpa `--archive-mode=off`.
6. Retry `deploy-production` di pipeline tag yang sedang berlaku (RB-01). DB production kembali dari pgBackRest beserta riwayatnya, jadi `gmaps-db` cukup diisi kiriman terbaru. File kiriman di server lama ikut hilang: minta founder menjalankan `export` lagi dengan batas = snapshot terakhir production di `/ops/imports` − 1 hari, lalu muat dengan RB-15, termasuk langkah sekali per volume. Lalu jalankan pipeline `main` untuk staging. DB staging kosong; bila staging dibutuhkan, minta kiriman riwayat penuh (batas `-infinity`, satu kiriman melayani staging dan production), lalu RB-04 dan import pertama RB-03. Riwayat penuh untuk production hanya dibutuhkan bila DB MSP hilang melebihi backup.
7. Backup penuh: `cd /opt/msp && docker compose exec -u postgres postgres pgbackrest --stanza=msp --type=full backup`.

**Verifikasi:** smoke test RB-01 lulus, RB-09 hijau (termasuk WAL baru sudah tersalin off-site), uptime monitor eksternal hijau, semua target Prometheus hijau, dan import manual RB-03 di production berhasil membaca `gmaps-db`.

### RB-12 — Rotasi secret
**Kapan:** pemegang secret keluar, secret bocor atau diduga bocor (segera), atau diminta penyedia. Provider, model, dan base URL AI bukan secret; ubah di RB-14. Yang dirotasi di sini adalah API key (`MSP_ANTHROPIC_API_KEY`, `MSP_LLM_FALLBACK_API_KEY`), password role `msp_source_ro` di `gmaps-db` (`MSP_SOURCE_DATABASE_URL`), kredensial SFTP/FTPS off-site (`/opt/ops/rclone.conf`), dan secret lain di `.env`.
1. Buat nilai baru di penyedia; biarkan nilai lama aktif dulu.
2. GitLab → **Settings → CI/CD → Variables** → ubah variabel `MSP_…` terkait (Protected, Masked). Perbarui salinan di password manager founder.
3. Jalankan ulang job deploy production untuk tag yang sedang berlaku. `.env` ditulis ulang (izin 600) dan container yang berubah dibuat ulang.
4. Setelah verifikasi, cabut nilai lama di penyedia.

Kasus khusus: password DB → `ALTER ROLE … PASSWORD …` lalu langsung langkah 2–3 di jam sepi (error singkat di antaranya). `MSP_SOURCE_DATABASE_URL` → password role lokal `msp_source_ro`, satu nilai untuk staging dan production. Di luar jam import (RB-15 langkah 4): `cd /opt/ops && docker compose exec gmaps-db psql -U postgres`, lalu `\password msp_source_ro`; langsung langkah 2 untuk scope `staging` dan `production` dan langkah 3 untuk keduanya (staging: Retry job `deploy-staging` di pipeline `main` terakhir). Import yang mulai di antaranya gagal login; verifikasi dengan import manual RB-03 di keduanya. Password superuser `gmaps-db` → `\password postgres` dengan cara yang sama, lalu perbarui `MSP_GMAPS_DB_PASSWORD` di `/opt/ops/.env` dan password manager (env ini hanya dibaca saat volume `gmaps-data` pertama kali dibuat). `MSP_ENCRYPTION_KEY` → jangan dirotasi tanpa rencana re-enkripsi dari engineering. Kredensial SFTP/FTPS (remote `offsite`) → buat password atau kunci SSH baru di server off-site, ubah `/opt/ops/rclone.conf` lewat SSH (`sudo rclone --config /opt/ops/rclone.conf config`; izin tetap 600, pemilik root), cek RB-09 setelah run berikutnya (≤ 5 menit), perbarui password manager, lalu cabut yang lama. `PGBACKREST_REPO1_CIPHER_PASS` → tidak bisa diganti untuk repo yang sudah ada; jangan dirotasi tanpa rencana engineering (repo baru + backup full). Salinannya **wajib** ada di password manager founder, di luar server: tanpa passphrase ini semua backup, lokal maupun off-site, tidak bisa dibuka. Secret cookie Kratos → tambahkan nilai baru di depan daftar dan simpan yang lama agar user tidak logout massal. Secret `/opt/ops/.env` (Grafana, SMTP/Telegram Alertmanager) → ubah lewat SSH, `cd /opt/ops && docker compose up -d`, perbarui password manager.

**Verifikasi:** `stat -c '%a %y' /opt/msp/.env` → `600` dengan waktu baru (kredensial SFTP/FTPS: `/opt/ops/rclone.conf`); fitur pemakai secret berfungsi (narasi AI, email, login Google, `pgbackrest check`, salinan off-site RB-09, import RB-03); status key di Pengaturan AI "Terpasang".

### RB-13 — Disk hampir penuh
**Kapan:** AL-07, sebelum import pertama (RB-03), atau sebelum memuat kiriman pertama atau kiriman riwayat penuh (RB-15). Semua project memakai disk yang sama, termasuk `gmaps-db`.
```bash
df -h && docker system df
sudo du -xh --max-depth=3 /var/lib/docker /opt /home 2>/dev/null | sort -h | tail -15
docker image prune -af --filter "until=336h"     # image > 14 hari; bisa di-pull ulang untuk rollback
docker builder prune -af --filter "until=168h"   # cache build
docker volume prune -af --filter "label=com.gitlab.gitlab-runner.managed=true"   # hanya volume cache runner CI
sudo journalctl --vacuum-size=500M
```
Registry GitLab: `cd /opt/ops && docker compose exec gitlab gitlab-ctl registry-garbage-collect -m` (registry read-only sebentar, jangan saat pipeline jalan). File kiriman di `/opt/ops/gmaps-dump/` selain yang terbaru boleh dihapus (RB-15). Masih penuh → DB staging boleh dikosongkan (`cd /opt/msp-staging && docker compose down -v`), lalu diisi lagi saat ruang cukup lewat pipeline `main` + RB-04 + kiriman riwayat penuh (RB-15) + RB-03. Tidak pernah untuk production.

**Jangan pernah** menghapus isi data Postgres, `pg_wal`, atau repo `/opt/msp/pgbackrest` (retensinya diatur pgBackRest; file yang dihapus manual merusak repo dan, lewat `rclone sync`, salinan off-site juga), dan jangan `docker volume prune` tanpa filter label runner. `pg_wal` yang membesar berarti arsip WAL gagal → AL-06. Bila disk tetap > 80% setelah dibersihkan, tambah disk (keputusan founder).

**Verifikasi:** `df -h` < 70%.

### RB-14 — Ubah Pengaturan AI, plan, atau parameter estimasi
**Kapan:** ganti provider, model utama, atau fallback AI (biaya, kualitas, atau provider gagal, AL-08); ubah plan (harga tampilan, seat, credit dan AI credit bulanan); atau ubah parameter estimasi (review rate, faktor belanja, benchmark harga, benchmark hotel, ambang penanda; daftar di PRD Lampiran E §E.5). Semuanya diatur dari halaman ops (D-29, D-32), berlaku tanpa deploy, dan tercatat di audit.
1. **AI:** halaman ops → **Pengaturan AI** → ubah provider/model/fallback → **Uji koneksi** → Simpan. Narasi baru memakai nilai ini dalam ≤ 1 menit. API key tidak bisa diubah di sini; key baru atau provider baru butuh RB-12 dulu. Status key "Belum diset" berarti tombol Simpan nonaktif. Untuk ganti model yang tidak mendesak, uji dulu di staging (halaman ops staging + satu narasi AI di workspace uji).
2. **Plan:** `/ops/plans` → ubah atau tambah plan. Plan tidak dihapus; nonaktifkan agar tidak bisa dipilih lagi. Perubahan credit bulanan berlaku pada grant otomatis bulan berikutnya. Selisih bulan berjalan lewat RB-06 bila founder memutuskan. Memasang plan ke workspace dilakukan di RB-06.
3. **Parameter estimasi:** ubah → hitung ulang → verifikasi, selalu di staging dulu, karena angka yang dilihat customer ikut berubah.
   1. Halaman ops staging → **Parameter estimasi** → ubah nilai → Simpan. Catat sumber nilai baru di `<log-ops>`. Hitung ulang (rantai di README) berjalan otomatis (FR-EST-06); jangan jalankan RB-05.
   2. Tunggu antrean `import` dan `score` kosong (Q1).
   3. Buka profil **Signa Cafe** (kasus golden PRD §12) di staging, atau profil lain di kelompok estimasi yang diubah. Parameter yang tampil harus nilai baru, dan angka bergeser sesuai rumus Lampiran E §E.2, mis. review rate tengah 1,5% → 1% membuat kunjungan dan omzet tengah naik 1,5×; faktor belanja 0,8 → 0,9 hanya menaikkan omzet (×1,125). Untuk benchmark hotel, buka satu profil hotel di kelas akomodasi yang diubah.
   4. Ulangi butir 1–3 di production, di jam sepi karena hitung ulang membebani DB.

   Benchmark yang dikosongkan membuat estimasi yang bergantung padanya tampil "Belum dapat diperkirakan" (BR-EST-01). Itu perilaku yang benar, bukan error.

**Verifikasi:** perubahan ada di `/ops/audit` (nilai lama → baru). AI: satu narasi AI di workspace uji berhasil dan L1 `worker` tanpa error provider. Plan: nilai baru tampil di `/ops/plans`. Parameter estimasi: di staging dan production, profil yang dicek menampilkan parameter baru dengan angka yang bergeser sesuai rumus, dan antrean `import` dan `score` kosong.

### RB-15 — Muat kiriman mingguan DB gmaps
**Kapan:** tiap minggu saat founder menyerahkan kiriman baru (§6); kiriman pertama saat instalasi baru (06 §9); bangun ulang server (RB-11); kiriman riwayat penuh untuk mengisi ulang DB staging atau sebelum `--reset` (04 §5.1); atau `gmaps-db` rusak (muat ulang file terbaru di server). Staging dan production membaca `gmaps-db` yang sama (06 §2), jadi prosedur ini dijalankan sekali untuk keduanya.

Kiriman = file CSV gzip hasil pernyataan `export` di `apps/api/internal/ingest/gmaps/source.sql` (04 §5.1, D-37): hanya baris Indonesia, hanya kolom yang dipakai import, dan hanya baris dengan `created_at` ≥ **batas**. Batas = `max(created_at)` kiriman sebelumnya − 1 hari, dicetak di langkah 4. Kiriman pertama dan kiriman riwayat penuh memakai batas `-infinity`. `gmaps-db` hanya menyimpan kiriman terbaru; riwayat ada di DB MSP. Founder dan ops memakai `source.sql` dari tag yang sedang berlaku di production, supaya `export`, tabel, dan import memakai kolom yang sama. Perintah ops dijalankan di `/opt/ops`, di `tmux` karena kiriman pertama bisa lama.

**Sebelum kiriman pertama (founder):** ukur S, yaitu ukuran kiriman riwayat penuh, untuk anggaran disk (04 §2.1). Perintah ini menjalankan `export` di DB gmaps asli tanpa menulis file:
```bash
sed -n '/^-- name: export/,/^$/p' source.sql \
  | psql -X -q -v ON_ERROR_STOP=1 -v since=-infinity '<url-db-gmaps>' | wc -c     # byte CSV ≈ S
```

**Sekali per volume `gmaps-data`** (instalasi baru, RB-11): buat role read-only dengan password yang sama dengan `MSP_SOURCE_DATABASE_URL` (password manager founder), lalu tabel dari `create_table` dan grant-nya. Setelah kolom di `source.sql` berubah (RB-03 langkah 5): `DROP TABLE business_listings`, lalu dua perintah terakhir saja.
```bash
cd /opt/ops                                                # source.sql dari tag production disalin ke sini
docker compose exec gmaps-db psql -U postgres -d gmaps     # lalu: CREATE ROLE msp_source_ro LOGIN;  \password msp_source_ro
sed -n '/^-- name: create_table/,/^$/p' source.sql | docker compose exec -T gmaps-db psql -U postgres -d gmaps -v ON_ERROR_STOP=1
docker compose exec gmaps-db psql -U postgres -d gmaps -c 'GRANT SELECT ON business_listings TO msp_source_ro'
```

1. **Founder membuat kiriman** di DB gmaps asli, dengan batas dari ops:
   ```bash
   set -o pipefail        # export gagal di tengah → sha256 tidak dicetak; file itu jangan dikirim
   sed -n '/^-- name: export/,/^$/p' source.sql \
     | psql -X -q -v ON_ERROR_STOP=1 -v since='<batas>' '<url-db-gmaps>' | gzip > gmaps-ID-<YYYYMMDD>.csv.gz \
     && sha256sum gmaps-ID-<YYYYMMDD>.csv.gz
   ```
2. **Founder menyalin file** ke `/opt/ops/gmaps-dump/` (izin baca 644) dan memberikan sha256-nya ke ops.
3. **Ops mencocokkan checksum dan ruang disk.**
   ```bash
   cd /opt/ops/gmaps-dump && echo '<sha256>  gmaps-ID-<YYYYMMDD>.csv.gz' | sha256sum -c
   df -h          # kiriman pertama atau riwayat penuh butuh ±2 × S (04 §2.1); kurang → RB-13
   ```
4. **Muat, di luar jendela import.** Hindari sekitar 03.00 WIB (job harian `import.run`). `/ops/imports` di staging dan production tidak boleh menunjukkan run yang sedang berjalan (import memegang advisory lock, 04 §5.1). Import terakhir di keduanya juga harus berstatus selesai **setelah** kiriman sekarang dimuat (tanggal muat di `<log-ops>`); belum → RB-03 dulu. Baris kiriman sekarang yang belum terbaca dan lebih tua dari batas kiriman baru akan hilang dari riwayat. DB staging yang sengaja dikosongkan (RB-13) atau belum diisi (RB-11) tidak ditunggu. Lalu muat dalam satu transaksi:
   ```bash
   cd /opt/ops
   docker compose exec gmaps-db psql -U postgres -d gmaps -v ON_ERROR_STOP=1 -1 \
     -c 'TRUNCATE business_listings' \
     -c "COPY business_listings FROM PROGRAM 'gzip -dc /dump/gmaps-ID-<YYYYMMDD>.csv.gz' WITH (FORMAT csv, HEADER MATCH)" \
     -c 'ANALYZE business_listings' \
     -c "SELECT count(*) AS baris, max(created_at) - interval '1 day' AS batas_berikutnya FROM business_listings"
   ```
   Gagal (file rusak, kolom atau tipe tidak cocok) → seluruh transaksi batal dan kiriman lama tetap utuh; kolom tidak cocok → RB-03 langkah 5. Berhasil → catat tanggal muat, `baris`, dan `batas_berikutnya` di `<log-ops>`, lalu kirim `batas_berikutnya` ke founder untuk kiriman minggu depan.
5. **Import.** Jalankan import manual RB-03 langkah 2 (staging dulu, lalu production), atau tunggu job harian. Kiriman pertama dan kiriman riwayat penuh → RB-03 langkah 3 (lama, bisa dilanjutkan).
6. **Cek `/ops/imports`** di staging dan production: import setelah pemuatan berstatus selesai, snapshot terakhir bergeser ke sekitar tanggal kiriman, dan **snapshot baru** > 0 (bila 0 padahal kiriman baru → kabari founder, AL-11). `permission denied` → ulangi `GRANT` (sekali per volume). `source schema mismatch` → RB-03 langkah 5.
7. **Hapus file kiriman yang lebih lama** di `/opt/ops/gmaps-dump/`; simpan hanya yang terbaru (06 §8).

**Verifikasi:** langkah 6 terpenuhi di staging dan production; `<log-ops>` berisi tanggal muat dan `batas_berikutnya`; `/opt/ops/gmaps-dump/` hanya berisi file terbaru.

## 4. Alert
Aturan alert ada di `deploy/ops/` (Prometheus); AL-01 berasal dari uptime monitor eksternal. Ambang di bawah adalah acuannya. Level SEV di §5; L1, L2, Q1, Q2 di §1.3.

| ID | Alert & ambang | Dampak | Langkah pertama |
|---|---|---|---|
| AL-01 | **Situs/server down** (uptime monitor eksternal, email): `https://app.<domain>/readyz` atau `https://gitlab.<domain>` gagal 2× berturut | App: semua user, SEV-1. Hanya GitLab: deploy dan CI berhenti, SEV-3 | Keduanya gagal → server, jaringan, atau `caddy`: SSH lalu `cd /opt/ops && docker compose ps caddy`. Tidak bisa SSH → console/reboot di panel hosting; rusak permanen → RB-11. Hanya app → `cd /opt/msp && docker compose ps`; service mati/unhealthy → L1 → `docker compose up -d <service>`. `/healthz` OK tetapi `/readyz` gagal → DB: log `postgres`, disk (RB-13). Baru deploy → RB-02. Hanya GitLab → `cd /opt/ops && docker compose ps gitlab`, L1 |
| AL-02 | **5xx tinggi:** > 5% respons API 5xx selama 10 menit | Sebagian aksi gagal. SEV-2, SEV-1 bila > 50% | L2. Mulai setelah deploy → RB-02. Error DB → langkah DB di AL-03. Error login → L1 untuk `kratos` |
| AL-03 | **p95 di atas NFR-01:** search > 1 s atau profil > 500 ms selama 15 menit | App lambat. SEV-3, SEV-2 bila > 2× target | Ada import, rescore, atau hitung ulang parameter yang jalan → wajar sementara; pindahkan ke jam sepi bila mengganggu. CPU/RAM host jenuh → AL-09. Query macet: `SELECT pid, now()-query_start AS durasi, left(query,100) FROM pg_stat_activity WHERE state <> 'idle' ORDER BY durasi DESC LIMIT 10;`, lalu untuk yang > 5 menit dan bukan backup: `SELECT pg_cancel_backend(<pid>);`. Masih lambat → engineering (query/indeks) |
| AL-04 | **Antrean River tertahan:** job `available` tertua > 15 menit di `export`, `ai`, `billing`, atau > 2 jam di `import`, `score` | Export, narasi AI, credit bulanan, atau estimasi dan metrik wilayah tertunda. SEV-2 | Q1, `docker compose ps worker`, L1 untuk `worker`. Worker mati/macet → `docker compose restart worker` (job idempotent, STD-DB-06). Bila hanya `import`/`score` yang panjang karena import pertama (RB-03) atau hitung ulang parameter (RB-14) → tunggu |
| AL-05 | **Banyak job gagal:** > 10 job `discarded` dalam 1 jam di antrean mana pun | Aksi user gagal (export yang gagal bisa diulang tanpa biaya). SEV-2 | Q2. `ai` → AL-08. `export` → disk (RB-13) atau bug (engineering). `import` → AL-10. `billing` → engineering, jangan grant manual (RB-06). Tidak ada redrive massal di F1: setelah penyebab beres, user mengulang aksinya; ops menjalankan import manual RB-03 (aman, tanpa duplikat) |
| AL-06 | **Arsip WAL/backup gagal:** tidak ada WAL terarsip > 15 menit, `failed_count` naik, atau backup terakhir > 26 jam | RPO ≤ 15 menit (NFR-04) tidak terjamin; `pg_wal` menumpuk sampai disk penuh. SEV-2, SEV-1 bila disk > 90% | RB-09 dan L1 untuk `postgres`. Repo ada di disk lokal, jadi penyebab umum: disk penuh (RB-13), pemilik/izin `/opt/msp/pgbackrest` bukan uid 999, `PGBACKREST_*` salah. Gangguan server off-site tidak memicu alert ini (→ AL-12). Setelah beres: `docker compose exec -u postgres postgres pgbackrest --stanza=msp backup` |
| AL-07 | **Disk penuh:** > 80% peringatan, > 90% kritis | Postgres berhenti menulis bila disk penuh → situs down. SEV-2, SEV-1 bila > 90% | RB-13 |
| AL-08 | **Provider AI gagal:** > 20% narasi AI memakai fallback selama 15 menit (peringatan); > 20% gagal di provider utama dan fallback selama 10 menit (kritis) | narasi AI lambat/gagal, fitur lain normal. SEV-3 / SEV-2 | Halaman status penyedia + L1 untuk `worker`. 401/403 → key salah atau dicabut (RB-12). 429 → kuota/rate limit di konsol penyedia. 5xx/timeout → di Pengaturan AI (RB-14) aktifkan fallback atau ganti provider/model utama, cek dengan **Uji koneksi**. Kritis > 30 menit → kill switch AI (RB-07); nyalakan lagi setelah pulih |
| AL-09 | **Resource host jenuh:** CPU > 85% atau RAM > 90% selama 15 menit | Production bisa melambat (AL-03). SEV-3, SEV-2 bila p95 production ikut naik | `docker stats --no-stream`. Periksa CI dan staging dulu; keduanya boleh dihentikan untuk melindungi production: `cd /opt/msp-staging && docker compose stop` dan `cd /opt/ops && docker compose stop gitlab-runner` (job CI tertunda). Nyalakan lagi dengan `start` setelah reda. Penyebab di production (import, rescore, query) → AL-03. Sering terjadi → founder (kapasitas, atau pisahkan GitLab/CI; PRD §13) |
| AL-10 | **Import gagal:** tidak ada import sukses dari `gmaps-db` > 26 jam (job harian gagal, macet, atau tidak jalan) | Data tidak bertambah; app tetap jalan dan estimasi menampilkan tanggal datanya. SEV-3 | Pesan error di `/ops/imports`, Q2, L1 untuk `worker`. Kolom sumber hilang/berubah → RB-03 langkah 5. Koneksi gagal → `cd /opt/ops && docker compose ps gmaps-db`, L1 untuk `gmaps-db`, disk (RB-13). Tabel `business_listings` tidak ada → buat (RB-15, sekali per volume). `permission denied` → `GRANT` di RB-15 (sekali per volume). Login ditolak → password `msp_source_ro` (RB-12). Setelah beres → import manual RB-03 |
| AL-11 | **Data sumber berhenti bertambah:** import sukses, tetapi tidak ada snapshot baru (Indonesia) > 8 hari (kiriman mingguan + 1 hari) | Data makin menua: belum ada kiriman baru, kiriman baru belum dimuat, atau kiriman baru tidak berisi snapshot baru (scraper berhenti). SEV-3 | AL-10 juga aktif → tangani AL-10 dulu. Kiriman baru sudah diterima tetapi belum dimuat → RB-15. Belum ada kiriman baru, atau snapshot terakhir di `/ops/imports` tidak bergeser setelah kiriman dimuat → kabari founder (pemilik scraper dan DB gmaps) |
| AL-12 | **Salinan off-site tertinggal:** sinkron rclone sukses terakhir (`msp_offsite_last_success_timestamp_seconds`, textfile `node-exporter`, 06 §8) > 15 menit (= RPO, NFR-04) untuk `part="wal"` atau > 26 jam untuk `part="backup"`, atau metrik tidak ada | Backup lokal tetap jalan, tetapi bila server hilang, data yang belum tersalin ikut hilang (RPO NFR-04 tidak terjamin). SEV-2 | RB-09 dan `sudo journalctl -t msp-offsite --since 1h`. Penyebab umum: server SFTP/FTPS tidak terjangkau atau penuh (kabari founder), kredensial berubah (RB-12), guard berhenti karena repo lokal kosong (cari penyebabnya, jangan dipaksa sinkron), cron `msp-offsite` hilang. Hanya `backup` yang tertinggal saat upload full mingguan atau backup GitLab besar, atau di instalasi baru sebelum job `backup` pertama → tunggu. Setelah beres, run berikutnya (≤ 5 menit) menyalin semua yang tertinggal |

## 5. Insiden
| Level | Contoh | Respons |
|---|---|---|
| SEV-1 | Situs down, server hilang, data hilang atau rusak, dugaan kebocoran data, credit terpotong salah secara massal | Segera, kapan pun; founder dilibatkan |
| SEV-2 | Fitur utama gagal (login, analisis lokasi, export, AI), backup/WAL gagal, salinan off-site tertinggal, monitoring mati | ≤ 4 jam |
| SEV-3 | Lambat sebagian, job gagal sporadis, fallback AI aktif, import harian gagal atau data sumber berhenti bertambah, GitLab/CI mati sementara production normal | Hari kerja berikutnya |

1. **Stabilkan** dengan jalan tercepat: rollback (RB-02), kill switch (RB-07), restart service, hentikan CI/staging (AL-09), bersihkan disk. Akar masalah belakangan.
2. **Komunikasikan.** Tulis di `<grup-ops>`: apa, sejak kapan, siapa yang menangani. SEV-1/2 yang dirasakan customer → email status ke admin workspace terdampak, update tiap ≤ 2 jam sampai selesai.
3. **Perbaiki** permanen lewat MR dan rilis biasa (RB-01).
4. **Tulis singkat** (SEV-1/2, ≤ 1 halaman, ≤ 3 hari kerja): kronologi WIB, dampak (user, data, credit), penyebab, perubahan agar tidak terulang. Simpan di `<tempat-write-up>`.

```text
Subjek: [MesinProspek] Gangguan <fitur> — <tanggal>
Halo,
Sejak pukul <jam> WIB, <fitur> di MesinProspek <tidak dapat digunakan / lebih lambat dari biasanya>.
Tim kami sedang menanganinya. <Export yang gagal bisa diulang tanpa biaya.>
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
| Harian | RB-09, termasuk job `backup` dan salinan off-site. `/ops/imports`: import harian selesai dan jeda snapshot di bawah ambang AL-11 (RB-03). Tinjau alert 24 jam terakhir di Grafana. Bila Grafana tidak terbuka, monitoring mati dan tidak ada yang memperingatkan → SEV-2 |
| Mingguan | `df -h` (RB-13 bila > 70%); Q2 job gagal; tren 5xx, p95, CPU, dan RAM di Grafana |
| Mingguan (kiriman DB gmaps) | Founder membuat dan mengirim kiriman dengan batas dari `<log-ops>`; ops memuatnya dengan RB-15 di luar sekitar 03.00 WIB, cek `/ops/imports` di staging dan production, lalu catat tanggal muat dan `batas_berikutnya` di `<log-ops>` |
| Bulanan | Uji restore RB-10 dari salinan off-site (DB sementara di server yang sama); update paket OS lalu reboot di jam sepi (semua service ikut restart, termasuk GitLab); update image dasar (`postgres`, `caddy`, `kratos`, `gitlab`, monitoring) lewat MR, `/opt/ops` diterapkan manual; tinjau daftar akses §2 |
| Per rilis | RB-01 lengkap dengan smoke test; catat versi dan waktu deploy di `<log-ops>` |
