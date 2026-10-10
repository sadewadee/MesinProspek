# 09 — Runbook Operasional

Prosedur operasi F1 untuk `production` dan `staging`. Versi 0.2 (lean). Stack dan nama mengikuti [README](README.md); pipeline, bootstrap server, dan konfigurasi backup ada di [06](06-ci-cd.md). Dokumen ini hanya berisi apa yang dikerjakan saat operasi.

**Konvensi.** Placeholder ditulis `<…>`. Waktu dalam WIB. Apa pun yang bisa diubah lewat GitLab atau halaman ops diubah lewat sana, bukan lewat SSH. Perintah di server dijalankan dari direktori Compose (§1.1); `msp` dijalankan di image aplikasi: `docker compose run --rm worker msp <subcommand>`.

## 1. Inventaris

### 1.1 Server

| Server | Service Compose | Direktori | URL |
|---|---|---|---|
| `prod` | `caddy`, `web`, `api`, `worker`, `kratos`, `postgres` (+ pgBackRest, stanza `msp`) | `/opt/msp` | `https://<domain>` · health `/healthz`, `/readyz` · halaman ops `/ops` |
| `ops` | `gitlab` (+ container registry), `gitlab-runner`, `prometheus`, `alertmanager`, `grafana` | `/opt/ops` | `https://gitlab.<domain>`, `https://grafana.<domain>` |
| `ops` (staging) | `caddy`, `web`, `api`, `worker`, `kratos`, `postgres` | `/opt/msp-staging` | `https://staging.<domain>` |

Direktori dibuat oleh `deploy/bootstrap.sh` (06). Prometheus mengambil `/metrics` aplikasi dan metrik host (disk, CPU, RAM) kedua server; Alertmanager mengirim alert ke email dan Telegram `<grup-ops>`. Probe eksternal `<layanan-uptime>` (di luar kedua server) memeriksa `https://<domain>/readyz` dan `https://gitlab.<domain>` tiap menit, sehingga server `ops` yang mati tetap terdeteksi.

### 1.2 Dependensi eksternal

| Dependensi | Dipakai untuk | Bila gangguan |
|---|---|---|
| GeoAPI (`geoapi.mcpke.dev`) | `msp tool geo-sync`, vector tiles peta | Peta tanpa batas wilayah; tabel peringkat tetap jalan. Tunda geo-sync |
| Claude (Anthropic API) | AI brief | Otomatis pindah ke fallback (AL-08) |
| API OpenAI-compatible (fallback) | AI brief bila Claude gagal | Keduanya gagal → kill switch AI (RB-07) |
| SMTP `<penyedia>` | Verifikasi email, undangan, email alert | User baru tidak bisa verifikasi; alert tetap lewat Telegram |
| Google OAuth | Login Google | Login email + password tetap jalan |
| Object storage S3-compatible off-site `<penyedia>` (Indonesia) | pgBackRest: backup + WAL | WAL menumpuk di disk `prod` (AL-06) |

### 1.3 Log & perintah umum

Log aplikasi berformat JSON (`slog`) di stdout container. Docker merotasi log berdasarkan ukuran, jadi **simpan log ke file segera saat insiden**. GitLab dan monitoring: perintah sama di `/opt/ops`; staging di `/opt/msp-staging`.

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
```

Antrean River (`import`, `score`, `export`, `webhook`, `ai`), dijalankan di psql:
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
- Semua akun web operasional wajib MFA. Prometheus dan Alertmanager tidak dibuka ke internet: `ssh -L 9090:localhost:9090 -L 9093:localhost:9093 <user>@<ops>`.
- Salinan semua secret production disimpan di password manager founder, sebagai cadangan bila GitLab hilang (RB-11).

| Akses | Siapa | Catatan |
|---|---|---|
| SSH `prod` dan `ops` | Founder, `<engineer>` | Darurat dan prosedur yang butuh shell |
| GitLab: jalankan deploy production | Founder | D-22. MFA |
| GitLab: Developer | `<engineer>`, akun bot agent AI | MFA untuk akun manusia |
| Halaman ops `/ops` | Founder, `<staf-ops>` | Akun ops, MFA TOTP (FR-AUTH-03) |
| Grafana | Founder, `<engineer>` | Login lewat akun GitLab |
| Panel penyedia: hosting, DNS, storage backup, Anthropic, LLM fallback, SMTP, Google Cloud | Founder | MFA |

**Orang keluar:** hapus kuncinya dari `~/.ssh/authorized_keys` dan IP-nya dari allowlist, nonaktifkan akun GitLab, ops, dan Grafana-nya, lalu rotasi secret yang pernah ia pegang (RB-12).

## 3. Prosedur

### RB-01 — Deploy production
**Kapan:** tag `vX.Y.Z` lulus pipeline, sudah diverifikasi di staging, dan founder setuju.
1. Pastikan tidak ada alert aktif dan RB-09 hijau.
2. GitLab → **Build → Pipelines** → pipeline tag `vX.Y.Z` → job manual deploy production → **Run**. Job menulis `.env`, lalu `docker compose pull` → `msp migrate` → `docker compose up -d --wait` (06).
3. Gagal di `msp migrate`: versi lama tetap jalan. Baca log job, perbaiki lewat rilis baru. Gagal di `up --wait`: RB-02.

**Verifikasi:** `curl -fsS https://<domain>/readyz`; `docker compose ps` semua `healthy`. Smoke test di workspace uji `<workspace-uji>`: login email dan Google → search → profil → unlock 1 prospek → export kecil → webhook ke `<endpoint-uji>` → AI brief → `/ops` terbuka dengan MFA. Pantau 5xx dan p95 di Grafana selama 30 menit.

### RB-02 — Rollback
**Kapan:** setelah deploy, error naik atau ada bug kritis yang tidak bisa diperbaiki dalam 30 menit.
1. GitLab → pipeline tag rilis sebelumnya → job deploy production → **Run** lagi. Skema DB tidak di-rollback; `msp migrate` versi lama tidak menjalankan apa pun.
2. Versi lama tidak cocok dengan skema baru (migrasi tidak kompatibel-mundur) → jangan rollback; buat tag hotfix. Data rusak → RB-10.

**Verifikasi:** `docker compose ps --format '{{.Service}} {{.Image}}'` menunjukkan sha tag lama (`git rev-list -n1 <tag-lama>`); `/readyz` OK; 5xx dan p95 kembali normal.

### RB-03 — Import dataset
**Kapan:** ada file dataset baru (`.jsonl` atau `.jsonl.gz`, satu objek JSON-LD per baris).
1. Jalankan dulu di staging (langkah sama di server `ops`, `/opt/msp-staging`). Lanjut ke production hanya bila laporannya masuk akal.
2. Cek ruang disk (`df -h`). Catat waktu mulai sebagai titik restore bila import salah: `date -Iseconds`.
3. Salin file dan jalankan di `tmux` agar tidak putus saat SSH putus:
```bash
ssh <user>@<prod> mkdir -p import && scp <file>.jsonl.gz <user>@<prod>:import/
ssh <user>@<prod>
tmux new -s import
cd /opt/msp && docker compose run --rm -v "$HOME/import:/import:ro" worker msp tool import /import/<file>.jsonl.gz
```
4. Baca laporan (output tool dan `/ops/imports`, bisa diunduh CSV): **baru**, **berubah**, **tidak berubah**, **ditolak per alasan**. Import ulang file yang sama harus menghasilkan baru = 0 dan berubah = 0. Ditolak > 5% atau ada alasan yang tidak dikenal → kirim laporan dan contoh baris ke engineering sebelum import berikutnya.

**Verifikasi:** `/ops/imports` berstatus selesai dan "Skor diperbarui"; `/ops/health`: jumlah prospek sesuai laporan, % terpetakan kelurahan ≥ 98% (FR-DATA-04); antrean `import` dan `score` kosong (Q1).

### RB-04 — Sync wilayah GeoAPI
**Kapan:** instalasi baru (sebelum import pertama), GeoAPI merilis data baru, atau % terpetakan wilayah turun.
1. Di `tmux`: `cd /opt/msp && docker compose run --rm worker msp tool geo-sync`. Tool menahan diri di batas GeoAPI 120 request/menit, jadi bisa lama. Bila terputus, jalankan ulang.
2. Lanjutkan dengan RB-05, karena wilayah dan `market_score` berubah.

**Verifikasi:** output tool menampilkan jumlah area per level (provinsi, kab/kota, kecamatan, desa/kelurahan) tanpa error; riwayat `geo-sync` di `/ops/imports`; `/ops/health` % terpetakan ≥ 98%; peta menampilkan atribusi GeoAPI.

### RB-05 — Rescore
**Kapan:** rilis membawa versi model baru (`priority-X.Y.Z`, `geo-X.Y.Z`), setelah RB-04, atau setelah bug skor diperbaiki.
1. Jalankan di jam sepi karena beban DB naik: `cd /opt/msp && docker compose run --rm worker msp tool rescore`.
2. Pantau antrean `score` (Q1) sampai kosong. Ada job `discarded` → AL-05.

**Verifikasi:** `/ops/health` menampilkan versi model aktif yang baru; 3 profil prospek yang dikenal menampilkan breakdown dengan versi baru dan band yang masuk akal; p95 kembali normal.

### RB-06 — Grant credit & refund manual
**Kapan:** workspace membayar paket atau top-up (billing F1 manual), kompensasi, atau credit terpotong karena kesalahan.
1. Grant: `/ops/workspaces` → `<workspace>` → **Grant credit**: jenis (credit / AI credit), jumlah, alasan berisi nomor invoice atau tiket.
2. Refund: di ledger workspace pilih entri debit → **Refund** + alasan.
3. Jangan pernah mengubah ledger lewat SQL.

**Verifikasi:** saldo workspace berubah sesuai jumlah; entri ledger dan audit (`/ops/audit`) tercatat atas nama Anda.

### RB-07 — Feature flag & kill switch AI
**Kapan:** provider AI gagal (AL-08), biaya AI melonjak, output AI bermasalah, atau export/webhook bermasalah (bug, banjir request, dugaan kebocoran).
1. `/ops/flags` → `AI`, `export`, atau `webhook` → **Off** + alasan. Berlaku langsung tanpa deploy; app customer menampilkan banner FITUR DIMATIKAN.
2. Mati > 30 menit → kirim pesan status ke workspace terdampak (§5).
3. Penyebab sudah beres → **On** lagi.

**Verifikasi:** di workspace uji kontrol fitur nonaktif; tidak ada job baru di antrean terkait (Q1); perubahan flag ada di `/ops/audit`. Setelah On: satu AI brief, export, atau webhook uji berhasil.

### RB-08 — Permintaan suppression
**Kapan:** seseorang meminta nilai kontaknya (telepon, URL) tidak ditampilkan.
1. Catat di `<tracker>`: tanggal, pemohon, nilai, tautan prospek.
2. `/ops/suppression` → tambah nilai + alasan + nomor tiket. Lalu balas pemohon.
3. Permintaan lain (akses, penghapusan, keberatan, ancaman hukum) → teruskan ke tim legal dan ikuti instruksinya. Jangan menjanjikan apa pun.

**Verifikasi:** di workspace uji yang sudah unlock prospek itu, nilai tidak tampil di profil ("Kontak tidak tersedia"), file export, dan payload webhook; entri ada di `/ops/audit`.

### RB-09 — Cek backup
**Kapan:** harian, saat AL-06, dan sebelum deploy.
```bash
cd /opt/msp
docker compose exec -u postgres postgres pgbackrest --stanza=msp info
docker compose exec -u postgres postgres pgbackrest --stanza=msp check
docker compose exec postgres psql -U postgres -c "SELECT last_archived_time, last_failed_time, failed_count FROM pg_stat_archiver;"
```
**Verifikasi:** `info` berstatus `ok` dengan backup terakhir sesuai jadwal 06 (≤ 24 jam); `check` selesai tanpa error; `last_archived_time` ≤ 15 menit lalu (RPO, NFR-04) dan tidak ada `last_failed_time` yang lebih baru.

### RB-10 — Restore PITR ke server/DB baru
**Kapan:** data rusak atau terhapus, server `prod` hilang (dari RB-11), atau uji restore bulanan.
**Aturan:** restore ke direktori atau server **baru**, jangan menimpa DB yang masih jalan. Untuk uji, **jangan** start `worker` karena akan mengirim webhook, email, dan panggilan AI sungguhan.
1. Tentukan waktu target sebelum kejadian (dari `/ops/audit` atau log). Untuk RB-11 tanpa target waktu: hapus opsi `--type` dan `--target*` agar pulih sampai WAL terakhir.
2. Siapkan direktori baru berisi file `deploy/` dan `.env` production (dari variabel GitLab), lalu:
```bash
cd /opt/msp-restore
docker compose run --rm --no-deps -u postgres postgres pgbackrest --stanza=msp --archive-mode=off \
  --type=time --target="<YYYY-MM-DD HH:MM:SS+07>" --target-action=promote restore
docker compose up -d --wait postgres api      # tanpa worker; --archive-mode=off hanya untuk uji
```
3. Butuh sebagian data saja (mis. baris terhapus)? Ambil dari DB hasil restore bersama engineering (`pg_dump -t`), jangan restore penuh di atas production.

**Verifikasi:** `SELECT pg_is_in_recovery(), pg_last_xact_replay_timestamp();` → `f` dan waktu mendekati target; `api` `healthy` (`/readyz`); `SELECT count(*)` beberapa tabel utama cocok dengan production. Uji bulanan: catat durasi (RTO ≤ 8 jam) dan titik pulih (RPO ≤ 15 menit) di `<log-ops>`, lalu `docker compose down -v` dan hapus direktorinya.

### RB-11 — Bangun ulang server
**Kapan:** server rusak, hilang, atau terkompromi. `prod` = SEV-1 dengan target RTO ≤ 8 jam (NFR-04). Bila terkompromi: jangan hapus server lama (bukti), jalankan langkah kebocoran data (§5), dan rotasi semua secret (RB-12) sebelum deploy.

`prod`:
1. Pesan server baru di Indonesia. Turunkan TTL DNS `<domain>` ke 300 detik bila masih sempat.
2. Bootstrap dari clone repo: `ssh root@<ip-baru> 'bash -s' < deploy/bootstrap.sh` (06).
3. Restore DB dengan RB-10 di `/opt/msp` server baru, tanpa target waktu dan tanpa `--archive-mode=off`.
4. Ganti variabel host production di GitLab ke IP baru → jalankan deploy production tag yang sedang berlaku (RB-01).
5. Arahkan DNS `<domain>` ke IP baru; Caddy mengambil sertifikat TLS otomatis. Ganti target `prod` di konfigurasi Prometheus (`deploy/`).
6. Backup penuh: `docker compose exec -u postgres postgres pgbackrest --stanza=msp --type=full backup`.

`ops` (`prod` tetap jalan, tetapi deploy dan monitoring mati sampai selesai):
1. Server baru + bootstrap.
2. Pulihkan GitLab dari backup-nya (`gitlab-backup restore`, 06). Tanpa backup: GitLab baru, push repo dari clone terbaru, isi ulang variabel CI dari password manager founder.
3. Daftarkan ulang `gitlab-runner`, jalankan pipeline `main` (build image + deploy staging), lalu start `prometheus`, `alertmanager`, `grafana` dari `deploy/`.

**Verifikasi:** smoke test RB-01, RB-09 hijau, probe eksternal dan semua target Prometheus hijau.

### RB-12 — Rotasi secret
**Kapan:** pemegang secret keluar, secret bocor atau diduga bocor (segera), atau diminta penyedia.
1. Buat nilai baru di penyedia; biarkan nilai lama aktif dulu.
2. GitLab → **Settings → CI/CD → Variables** → ubah variabel `MSP_…` terkait (Protected, Masked). Perbarui salinan di password manager founder.
3. Jalankan ulang job deploy production untuk tag yang sedang berlaku: `.env` ditulis ulang (izin 600) dan container yang berubah dibuat ulang.
4. Setelah verifikasi, cabut nilai lama di penyedia.

Kasus khusus: password DB → `ALTER ROLE … PASSWORD …` lalu langsung langkah 2–3 di jam sepi (error singkat di antaranya). `MSP_ENCRYPTION_KEY` → jangan dirotasi tanpa rencana re-enkripsi dari engineering. Secret cookie Kratos → tambahkan nilai baru di depan daftar dan simpan yang lama agar user tidak logout massal.

**Verifikasi:** `stat -c '%a %y' /opt/msp/.env` → `600` dengan waktu baru; fitur pemakai secret berfungsi (AI brief, email, login Google, `pgbackrest check`).

### RB-13 — Disk hampir penuh
**Kapan:** AL-07, atau sebelum import besar.
```bash
df -h && docker system df
sudo du -xh --max-depth=3 /var/lib/docker /opt /home 2>/dev/null | sort -h | tail -15
docker image prune -af --filter "until=336h"     # image > 14 hari; bisa di-pull ulang untuk rollback
docker builder prune -af --filter "until=168h"   # cache build runner (ops)
sudo journalctl --vacuum-size=500M
rm ~/import/<file-yang-sudah-diimport>
```
Di `ops`, registry GitLab: `cd /opt/ops && docker compose exec gitlab gitlab-ctl registry-garbage-collect -m` (registry read-only sebentar, jangan saat pipeline jalan).
**Jangan pernah** menghapus isi data Postgres atau `pg_wal`, dan jangan `docker volume prune`. `pg_wal` membesar berarti arsip WAL gagal → AL-06. Tetap > 80% setelah dibersihkan → tambah disk (keputusan founder).

**Verifikasi:** `df -h` < 70%.

## 4. Alert

Aturan alert ada di `deploy/` (Prometheus + probe eksternal); ambang di bawah adalah acuannya. Level SEV ada di §5.

**AL-01 — Situs down.** *Ambang:* probe `https://<domain>/readyz` gagal 2× berturut. *Dampak:* semua user, SEV-1.
1. `docker compose ps` di `prod`. Service mati atau unhealthy → `docker compose logs --since 30m <service>` → `docker compose up -d <service>`.
2. `/healthz` OK tetapi `/readyz` gagal → DB bermasalah: log `postgres`, disk (RB-13).
3. Server tidak bisa di-SSH → console/reboot di panel hosting; rusak permanen → RB-11. Baru saja deploy → RB-02.
4. Probe `https://gitlab.<domain>` gagal → server `ops` mati: SEV-2 (prod jalan, tetapi tanpa monitoring dan deploy).

**AL-02 — Rasio 5xx tinggi.** *Ambang:* > 5% respons API 5xx selama 10 menit. *Dampak:* sebagian aksi gagal, SEV-2 (SEV-1 bila > 50%).
1. Error terbanyak: `docker compose logs --no-log-prefix --since 30m api | jq -rR 'fromjson? | select(.level=="ERROR") | .msg' | sort | uniq -c | sort -rn | head`.
2. Mulai setelah deploy → RB-02. Error DB → langkah DB di AL-03. Error login → `docker compose logs --since 30m kratos`.

**AL-03 — Latensi p95 di atas NFR-01.** *Ambang:* search p95 > 1 s atau profil p95 > 500 ms selama 15 menit. *Dampak:* app lambat, SEV-3 (SEV-2 bila > 2× target).
1. Sedang ada import atau rescore? Wajar sementara; bila mengganggu, pindahkan ke jam sepi.
2. Query macet: `SELECT pid, now()-query_start AS durasi, left(query,100) FROM pg_stat_activity WHERE state <> 'idle' ORDER BY durasi DESC LIMIT 10;` → yang > 5 menit dan bukan backup: `SELECT pg_cancel_backend(<pid>);`.
3. CPU, RAM, atau IO host jenuh di Grafana → laporkan ke engineering (query/indeks) dan founder (kapasitas).

**AL-04 — Antrean River tertahan.** *Ambang:* job `available` tertua > 15 menit di `export`, `webhook`, `ai`, atau > 2 jam di `import`, `score`. *Dampak:* export, webhook, AI brief, atau skor tertunda, SEV-2.
1. Q1, lalu `docker compose ps worker` dan `docker compose logs --since 30m worker`.
2. Worker mati atau macet → `docker compose restart worker`. Job idempotent (STD-DB-06), aman diulang.
3. Hanya `import`/`score` yang panjang karena import besar → tunggu dan pantau.

**AL-05 — Banyak job gagal.** *Ambang:* > 10 job `discarded` dalam 1 jam di antrean `import`, `export`, `webhook`, atau `ai`. *Dampak:* aksi user gagal (export gagal tidak ditagih), SEV-2.
1. Q2 untuk melihat error.
2. `webhook` → biasanya endpoint customer menolak atau timeout: kabari customer. `ai` → AL-08. `export` → disk (RB-13) atau bug (engineering). `import` → laporan RB-03.
3. F1 tidak punya redrive massal: setelah penyebab beres, user mengulang aksinya dan ops mengulang import (aman, upsert).

**AL-06 — Arsip WAL atau backup gagal.** *Ambang:* tidak ada WAL terarsip > 15 menit, `failed_count` naik, atau backup terakhir > 26 jam. *Dampak:* RPO ≤ 15 menit (NFR-04) tidak terjamin; `pg_wal` menumpuk sampai disk penuh. SEV-2 (SEV-1 bila disk > 90%).
1. RB-09, lalu `docker compose logs --since 1h postgres | grep -iE 'archive|pgbackrest'`.
2. Penyebab umum: storage off-site tidak terjangkau, kredensial kedaluwarsa (RB-12), kuota bucket habis, disk lokal penuh.
3. Setelah beres: `pgbackrest check`, lalu `docker compose exec -u postgres postgres pgbackrest --stanza=msp backup`.

**AL-07 — Disk > 80%.** *Ambang:* > 80% peringatan, > 90% kritis, di server mana pun. *Dampak:* Postgres berhenti menulis bila disk penuh → situs down. SEV-2; SEV-1 bila > 90% di `prod`. Langkah: RB-13.

**AL-08 — Provider AI gagal.** *Ambang:* > 20% AI brief memakai fallback selama 15 menit (peringatan, Claude bermasalah); > 20% AI brief gagal di kedua provider selama 10 menit (kritis). *Dampak:* AI brief lambat atau gagal, fitur lain normal. SEV-3 / SEV-2.
1. Cek halaman status penyedia dan `docker compose logs --no-log-prefix --since 30m worker | jq -cR 'fromjson? | select(.level!="INFO")' | tail -50`.
2. 401/403 → kunci salah atau dicabut (RB-12). 429 → kuota atau rate limit di konsol penyedia. 5xx/timeout → tunggu penyedia.
3. Kritis > 30 menit → kill switch AI (RB-07); nyalakan lagi setelah pulih.

## 5. Insiden

| Level | Contoh | Respons |
|---|---|---|
| SEV-1 | Situs down, data hilang atau rusak, dugaan kebocoran data, credit terpotong salah secara massal | Segera, kapan pun; founder dilibatkan |
| SEV-2 | Fitur utama gagal (login, export, webhook, AI), backup/WAL gagal, server `ops` mati | ≤ 4 jam |
| SEV-3 | Lambat sebagian, job gagal sporadis, fallback AI aktif | Hari kerja berikutnya |

1. **Stabilkan** dengan jalan tercepat: rollback (RB-02), kill switch (RB-07), restart service, bersihkan disk. Akar masalah belakangan.
2. **Komunikasikan.** Tulis di `<grup-ops>`: apa, sejak kapan, siapa yang menangani. SEV-1/2 yang dirasakan customer → email status ke admin workspace terdampak, update tiap ≤ 2 jam sampai selesai.
3. **Perbaiki** permanen lewat MR dan rilis biasa (RB-01).
4. **Tulis singkat** (SEV-1/2, ≤ 1 halaman, ≤ 3 hari kerja): kronologi WIB, dampak (user, data, credit), penyebab, perubahan agar tidak terulang. Simpan di `<tempat-write-up>`.

```text
Subjek: [MesinProspek] Gangguan <fitur> — <tanggal>

Halo,
Sejak pukul <jam> WIB, <fitur> di MesinProspek <tidak dapat digunakan / lebih lambat dari biasanya>.
Tim kami sedang menanganinya. <Credit tidak ditagih untuk export yang gagal.>
Kabar berikutnya paling lambat pukul <jam> WIB.

Pembaruan <jam> WIB: <fitur> sudah normal kembali sejak pukul <jam> WIB. Mohon maaf atas ketidaknyamanannya.

Salam,
Tim MesinProspek
```

**Dugaan kebocoran data** (data customer, kontak, secret, atau akses tidak sah):
1. Beri tahu founder **dan** tim legal segera, lewat telepon, tidak hanya chat.
2. Hentikan kebocoran tanpa menghapus bukti: matikan fitur terkait (RB-07), cabut akses atau kunci (RB-12), blokir IP.
3. Simpan bukti: log container (§1.3) ke file dan salin keluar server. Jangan menghapus container, volume, atau server. Audit log tetap di DB.
4. Ikuti instruksi tim legal. Jangan berkomunikasi ke luar soal kebocoran tanpa persetujuan tim legal.

## 6. Jadwal rutin

| Kapan | Tugas |
|---|---|
| Harian | RB-09; tinjau alert 24 jam terakhir |
| Mingguan | `df -h` di kedua server (RB-13 bila > 70%); Q2 job gagal; tren 5xx dan p95 di Grafana |
| Bulanan | Uji restore RB-10 di server `ops`; update paket OS lalu reboot di jam sepi; update image dasar (`postgres`, `caddy`, `kratos`, `gitlab`, monitoring) lewat MR; tinjau daftar akses §2 |
| Per rilis | RB-01 lengkap dengan smoke test; catat versi dan waktu deploy di `<log-ops>` |
