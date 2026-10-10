# 09 — Runbook Operasional

## MesinProspek — Operasi Production, Backup & DR, Alert Playbook, Insiden & PDP

| Field | Value |
|---|---|
| Dokumen | Runbook operasional |
| Versi | 0.1 (Draft) |
| Tanggal | 2026-10-10 |
| Basis | `04-technical-design.md` (ADR-06/07/18/20/21/22, §2, §12–§18), `06-ci-cd.md` (§3, §6, §7, §9–§14), `01-prd.md` (NFR-AVAIL-01/02, NFR-PRIV-*, BR-PRIV-03, FR-AUD-003..005, FR-OPS-*, R-08), `05-test-strategy.md` (§3.25, TC-OPS-007/011/015/016) |
| Keputusan | D-18 (GitLab CE self-hosted), D-19 (dedicated server di DC Indonesia), D-21 (Go, binary `msp`) |
| Audience | On-call engineer, Maintainer, DevOps, operator ops console, DPO/PIC pelindungan data |
| Status | Draft. Semua perintah **ilustratif**; nama final mengikuti `infra/ansible/`, 04, dan 06 |

> Konvensi: `RB-nn` = prosedur (RB-0x akses, RB-1x–3x rutin, RB-4x backup/DR, RB-5x insiden). `AL-nn` = alert playbook. `Q-RB-nn` = pertanyaan terbuka (§10). Placeholder ditulis `<…>`. Jam dalam WIB (UTC+7); DB, log, dan timeline insiden memakai UTC + WIB. Perintah dijalankan dari laptop admin yang terhubung WireGuard kecuali disebut lain. Perubahan konfigurasi selalu lewat MR → pipeline (06 §11.3). Perintah manual di runbook ini hanya untuk diagnosis, mitigasi darurat, atau break-glass, dan wajib ditindaklanjuti MR adopsi atau dikembalikan ≤ 2 hari kerja (06 §11.4).

### Daftar isi

1. Ruang lingkup, peran & eskalasi · 2. Inventaris sistem · 3. Akses operasional · 4. Prosedur rutin · 5. Backup & disaster recovery · 6. Alert playbook · 7. Manajemen insiden & PDP · 8. Kalender pemeliharaan · 9. Lampiran · 10. Pertanyaan terbuka

**Indeks prosedur:** RB-01 Break-glass · RB-02 Akses support · RB-03 On/offboarding akses · RB-10 Deploy · RB-11 Rollback · RB-12 Hotfix · RB-13 Migrasi bermasalah · RB-14 Skala worker · RB-15 DLQ re-drive · RB-16 Webhook redrive · RB-17 Snapshot · RB-18 Model/geo/prompt · RB-19 Kill switch & flag · RB-20 Rotasi secret · RB-21 TLS · RB-22 Patch OS · RB-23 Rehydrate secret · RB-24 Disk penuh · RB-25 Suppression · RB-26 DSR · RB-27 Penutupan workspace · RB-28 Credit manual · RB-29 Koreksi data · RB-30 Menambah app-02 · RB-40 Verifikasi backup · RB-41 PITR · RB-42 Failover DB · RB-43 Failback DB · RB-44 SeaweedFS · RB-45 Kehilangan app-01 · RB-46 Kehilangan DC · RB-47 Drill bulanan · RB-50 Kegagalan PDP

---

## 1. Ruang lingkup, peran & eskalasi

### 1.1 Ruang lingkup & prinsip

Berlaku untuk production (`app-01`/`app-02`, `db-01`, `db-02`, `stor-01`, `ops-01`, `ci-01`), lokasi off-site, host drill/DR `dr-01`, dan dependensi eksternal (§2.6). Staging (`stg-01`) hanya dipakai untuk latihan prosedur dan alert drill. Standar kode (08), gate rilis (06 §8), dan desain (04) tidak diulang di sini.

1. **Containment dulu, diagnosis kemudian** (06 §14.5): kill switch → rollback → blokir akses → baru cari akar masalah.
2. **Production hanya berubah lewat pipeline protected tag** (06 P-9). Jalur manual = break-glass (RB-01) dan tercatat.
3. **Data production tidak meninggalkan zonanya** (06 P-7): tidak ada restore ke `stg-01` atau laptop. Data pribadi tidak ditempel ke chat/issue; gunakan ID berprefiks (`cmp_…`, `wsp_…`), `value_hash`, atau `correlation_id`.
4. **Forward fix untuk skema dan data.** Tidak ada down-migration di production (06 §9.1). PITR adalah upaya terakhir.
5. **Setiap aksi tercatat**: aksi aplikasi di `platform.audit_log` (otomatis), aksi infra di issue GitLab + auditd.

### 1.2 Model on-call (tim kecil, Proposed — Q-RB-01)

| Aspek | Ketentuan |
|---|---|
| Rotasi | 2–4 engineer + Tech lead. Rotasi mingguan **primary** dan **secondary**, serah terima Senin 10:00 WIB lewat issue `oncall-handover` (alert terbuka, insiden, maintenance terjadwal, silence aktif) |
| Jam layanan | Jam kerja (Sen–Jum 09:00–18:00 WIB): semua severity. Di luar jam kerja dan libur nasional: hanya page **SEV1/SEV2** |
| Target ack | SEV1 ≤ 15 menit (24/7) · SEV2 ≤ 30 menit jam kerja, ≤ 1 jam di luar jam kerja · SEV3 hari kerja yang sama · SEV4 backlog (06 §14.5) |
| Eskalasi otomatis | Page tidak di-ack dalam target → secondary. +15 menit → Tech lead (IC default) |
| Syarat masuk rotasi | Akses WireGuard + akun SSH bernama, MFA ops console, sudah melatih RB-11, RB-19, dan RB-42 di staging/drill, shadowing 1 minggu, nomor terdaftar di §1.4 |

### 1.3 Peran

| Peran | Tanggung jawab | Default |
|---|---|---|
| On-call primary | Ack alert, triage, menjalankan AL/RB, deklarasi insiden, containment awal | Rotasi |
| On-call secondary | Backup ack, diagnosis paralel, scribe bila primary menjadi IC | Rotasi |
| Incident Commander (IC) | Memimpin SEV1/SEV2: prioritas, keputusan, delegasi. IC tidak mengetik perintah bila ada orang lain | Tech lead; sementara primary sampai diserahkan |
| Comms lead | Update internal, customer (§7.4), dan founder | Founder/Product (SEV1), IC (SEV2) |
| Scribe | Timeline di issue insiden (UTC + WIB, aksi, pelaku, bukti) | Secondary |
| Maintainer | Satu-satunya yang menjalankan job production (deploy, rollback, publish, activate, infra apply) dan memegang kunci age/break-glass | Tech lead + 1 engineer senior (Q-CI-07) |
| DPO / PIC pelindungan data | Penilaian PDP, keputusan notifikasi bersama hukum, korespondensi subjek data/lembaga/tenant, register DSR | `<nama>` |
| Konsultan hukum | Validasi kewajiban dan isi notifikasi PDP (BR-PRIV-03), tenggat DSR | `<firma>` (eksternal) |
| Founder | Keputusan bisnis: PITR (06 §14.3), aktivasi lokasi DR, komunikasi publik, kompensasi | `<nama>` |
| Data owner | Rilis snapshot dan model (tag `ds-*`, `priority-*`, `geo-*`, `brief-*`) | Data-intel lead |
| Operator ops console | Suppression, DSR, merge review, grant credit, koreksi data, akses support | Ops internal (MFA) |

### 1.4 Matriks eskalasi (isi sebelum closed beta)

| Level | Siapa | Kapan | Kanal | Kontak |
|---|---|---|---|---|
| L1 | On-call primary | Semua page | Pager (TBD) + kanal chat | `<rotasi>` |
| L2 | On-call secondary | L1 tidak ack, atau butuh tangan kedua | Pager + telepon | `<rotasi>` |
| L3 | Tech lead / Maintainer (IC) | SEV1 apa pun; SEV2 > 30 menit; butuh deploy, rollback, failover, break-glass | Telepon | `<nama>` · `<telp>` |
| L4 | Founder | SEV1 ≤ 15 menit; keputusan PITR/DR; komunikasi publik | Telepon | `<nama>` · `<telp>` |
| PDP | DPO/PIC + konsultan hukum | Label `pdp-suspected` (segera, 24/7) | Telepon + email | `<nama>` · `<telp>` · `<email>` |
| Vendor DC | NOC provider (remote hands, IPMI/KVM, jaringan, listrik) | Hardware, jaringan, DC | Portal tiket + telepon NOC | `<provider>` · `<no. kontrak>` |
| Vendor lain | Provider LLM, SMTP, DNS, off-site storage, probe eksternal | Gangguan dependensi (§2.6) | Portal support | `<…>` |

### 1.5 Kanal komunikasi

| Kanal | Isi | Catatan |
|---|---|---|
| Kanal chat tim (TBD, self-hosted) — `#ops-alert` | Notifikasi Alertmanager (page & ticket), status pipeline | Q-CI-06. Jangan menempel data pribadi |
| Kanal insiden `#inc-<yyyymmdd>-<slug>` | Koordinasi per insiden SEV1/SEV2 | Dibuat IC; diarsipkan setelah PIR |
| Issue GitLab (label `incident`, `pdp-suspected`, `pir`, `oncall-handover`) | Timeline resmi, bukti, action item | Sumber kebenaran tertulis |
| Email on-call | Salinan semua alert | Jalur cadangan bila chat mati |
| Telepon/SMS (paging TBD), bridge suara/video | Page di luar jam kerja, eskalasi L3/L4/PDP, SEV1 | Provider paging & bridge = Q-RB-01 |
| Probe eksternal (di luar DC) | `app.<domain>` down, watchdog Alertmanager hilang | Wajib mengirim ke email/SMS **tanpa** melewati `ops-01` |
| Email ke customer | Notifikasi insiden ke Owner/Admin workspace terdampak | Template §7.4, dikirim lewat provider SMTP |

---

## 2. Inventaris sistem

Kritikalitas: **K1** = gangguan langsung ke customer atau risiko kehilangan data · **K2** = degradasi fitur atau hilangnya kemampuan operasional · **K3** = internal.

### 2.1 Server (04 §2.2, 06 §3.5)

| Host | Zona | Peran | Layanan utama | Inbound (nftables) | K |
|---|---|---|---|---|---|
| `app-01` | edge/prod | Aplikasi (SPOF di MVP, R-08) | Traefik, web, api, worker River (blue/green), kratos, redis, unbound, smokescreen, otel-collector | 80/443 publik; SSH dari subnet WireGuard dan IP privat `ops-01` | K1 |
| `app-02` | edge/prod | HA aplikasi (RB-30) | Identik `app-01` + VIP | Sama + VRRP | K1 setelah aktif |
| `db-01` | data/prod | DB primary (satu-satunya target write, juga antrean River) | PostgreSQL 16 + PostGIS, PgBouncer (transaction mode), pgBackRest, postgres_exporter | 5432/6432 dari app host; exporter dari `ops-01`; SSH | K1 |
| `db-02` | data/prod | Hot standby async + read replica (search/profil) | PostgreSQL 16, `hot_standby_feedback=on` | 5432 dari app host; SSH | K1 (read), target failover |
| `stor-01` | data/prod | Object storage | SeaweedFS (master, volume, filer, S3 gateway), repo1 pgBackRest, `weed filer.sync` off-site | 8333 (S3) dari app/db host; SSH | K1 |
| `ops-01` | ops | Ops, CI/CD, observability, akses admin | GitLab CE + registry, runner `deploy`, Prometheus, Alertmanager, Grafana, Loki, Tempo, GlitchTip, **Tang**, **WireGuard** | UDP 51820 publik; lainnya hanya dari WireGuard/zona privat | K2 (runtime tidak bergantung); K1 untuk deploy dan unlock LUKS saat reboot |
| `ci-01` | ci | CI (MR & build) | Runner `ci-docker`, `ci-build` | Tanpa inbound; tanpa rute ke prod/stg | K3 |
| `stg-01` | stg | Staging | All-in-one (stack sama, data sintetis) | 443 dan SSH dari WireGuard | K3 |
| `dr-01` | prod (terisolasi) | Target drill restore/DR | PostgreSQL + pgBackRest (boleh sewa per jam, disk LUKS) | Hanya dari WireGuard/`ops-01`, saat drill/DR | K3 (K1 saat DR) |
| Off-site | Lokasi kedua di Indonesia | Backup off-site (Q-09/Q-CI-04) | pgBackRest repo2, SeaweedFS off-site, salinan backup GitLab | Hanya dari zona prod | K1 untuk DR |

Spesifikasi hardware mengikuti 04 §17.2 (06 §3.5 menyebut sizing awal yang lebih kecil; Q-RB-19).

### 2.2 Container di `app-01` (04 §17.3, 06 §17.3)

Port internal di bawah adalah default upstream/contoh; nilai final ada di inventory Ansible.

| Service | Project Compose | Perintah | Port internal | Catatan |
|---|---|---|---|---|
| traefik | `msp-core` (Q-RB-03) | Traefik v3 | 80, 443 | File provider `/opt/traefik/dynamic/msp.yml`; ACME Let's Encrypt |
| web (2 replika) | `msp-blue` / `msp-green` | `msp-web@<digest>` | 3000 | UI saja |
| api (2 replika) | `msp-<warna>` | `msp-backend@<digest> api` | 8080 (`/healthz`, `/readyz`) | Service Traefik `api-<warna>@docker` |
| worker-pipeline / -scoring / -export / -webhook / -ai / -maintenance | `msp-<warna>` | `worker --queues=<q>` | — | `MaxWorkers` 32/8/4/32/16/8. `worker-webhook` di network terisolasi (hanya smokescreen, PgBouncer, Redis) |
| `msp-migrate-<versi>` | one-off | `migrate up` | — | Dijalankan `deploy.yml`, langsung ke `db-01` (`msp_owner`), lalu dihapus |
| kratos | `msp-core` | Ory Kratos (D-20 Open) | 4433 public / 4434 admin | Admin API tidak pernah lewat Traefik |
| redis | `msp-core` | Redis (TLS + ACL, `maxmemory` 8 GB) | 6379 | Cache/rate limit saja; tidak di-backup |
| unbound | `msp-core` | Resolver DNS | 53 | Satu-satunya egress DNS |
| smokescreen | `msp-core` | Egress proxy | 4750 | ACL role `webhook`, `ai`, `mail` |
| otel-collector | `msp-core` | OTel Collector | 4317/4318 | Kirim ke `ops-01` |
| node_exporter, cAdvisor | host | — | 9100, `<port>` | Di-scrape `ops-01` |

File penting di host: `/opt/msp/active_color`, `/opt/msp/<warna>/compose.yml`, secret tmpfs `/run/msp/<warna>/` (0400). Host menyimpan image 5 rilis terakhir (06 §14.1).

### 2.3 Data store

| Store | Lokasi | Isi | Proteksi | Target | K |
|---|---|---|---|---|---|
| PostgreSQL `msp` | `db-01` (+ `db-02`) | Schema modul + schema `river` (antrean, outbox) | pgBackRest repo1 + repo2, WAL kontinu, replika streaming | RPO ≤ 15 menit (praktik ≤ 1–2 menit) · RTO ≤ 4 jam | K1 |
| Bucket `msp-uploads` | `stor-01` | Upload customer (`{workspace_id}/…`) | `weed filer.sync` off-site | Lag sync ≤ 1 jam (usulan) | K1 |
| `msp-exports` | `stor-01` | File export, presigned 24 jam | Lifecycle 24 jam; dapat dibuat ulang | — | K2 |
| `msp-snapshots` | `stor-01` | Snapshot mentah + manifest sha256 | Object lock bila didukung; fallback salinan append-only off-site (TQ-17) | Immutable | K2 |
| `msp-tiles` | `stor-01` | PMTiles batas wilayah | Off-site; dapat dibangun ulang | — | K2 |
| `pgbackrest` (repo1) | `stor-01` | Backup DB terenkripsi | repo2 independen di off-site | — | K1 |
| `msp-evidence` | `stor-01` | SBOM, laporan rilis data/model, laporan drill | Off-site | ≥ 1 tahun | K3 |
| Bucket audit (`S3_BUCKET_AUDIT`, nama usulan `msp-audit`) | `stor-01` | Ekspor harian audit log + hash | Object lock bila didukung | Retensi TQ-14 | K2 |
| Redis | `app-01` | Cache sesi (TTL 60 s), cache, rate limit, counter | Tidak di-backup | — | K2 (degraded mode, AL-16) |
| GitLab (repo, registry, CI) | `ops-01` | Kode, image, riwayat deploy | `gitlab-backup` harian 02:00 → `stor-01` + off-site, retensi 14 hari | RPO 24 jam | K2 |
| Prometheus / Loki / Tempo / GlitchTip | `ops-01` | Metrik 30 hari, log 30 hari, trace 7 hari, error 90 hari | Tidak di-backup (usulan) | — | K3 |

### 2.4 Dashboard

| Dashboard | Lokasi | Dipakai di |
|---|---|---|
| API & SLO (latency, 5xx, error budget, anotasi deploy, label `msp.color`) | Grafana | RB-10/11, AL-01..03 |
| Webhook (attempt, success ratio, DLQ per endpoint, endpoint disabled) | Grafana | RB-16, AL-10, AL-18 |
| Billing/privasi (rekonsiliasi ledger, `suppression_leak_count`, lag propagasi) | Grafana | AL-20, AL-21 |
| Infra (host, disk, DB, replikasi, backup/WAL, SeaweedFS, Redis, TLS) | Grafana | AL-11..17 |
| Data Health, Pipeline Health + DLQ, Intelligence Health, Cost | Ops console + Grafana | RB-15/17/18, AL-04..09 |
| CI health, DORA | Grafana | §8 |
| GlitchTip (api, worker, web) | `ops-01` | Semua AL |
| Probe eksternal | `<layanan probe>` | AL-03 |

URL internal (`grafana.<ops-domain>`, `gitlab.<ops-domain>`, `glitchtip.<ops-domain>`) hanya dapat diakses lewat WireGuard.

### 2.5 Lokasi log

| Sumber | Lokasi | Cara akses / catatan |
|---|---|---|
| api, worker-*, web | stdout JSON → OTel Collector → Loki | `{service="api", env="production"} \| json \| correlation_id="<id>"` (field log 04 §16.2, PII teredaksi) |
| Traefik (access log) | Loki `{service="traefik"}` | Status, route, durasi, warna |
| Smokescreen (allow/deny) | Loki `{service="smokescreen"}` | AL-18 |
| Kratos | Loki `{service="kratos"}` | AL-19 |
| PostgreSQL | `/var/log/postgresql/` di `db-0x` (+ Loki) | Lock, error RLS, slow query |
| pgBackRest | `/var/log/pgbackrest/` di `db-01` | Backup, archive-push |
| SeaweedFS | journald/log container di `stor-01` (+ Loki) | filer.sync, volume |
| Host (auth, sudo, auditd) & WireGuard | `/var/log/auth.log`, `/var/log/audit/audit.log` (+ Loki); `wg show` + journald di `ops-01` | Bukti break-glass dan akses admin |
| GitLab & runner deploy | `/var/log/gitlab/` + log job GitLab | Siapa menjalankan job production |
| Audit aplikasi | `platform.audit_log` + ekspor harian bucket audit | Ops console / replika read-only |

### 2.6 Dependensi eksternal

| Dependensi | Dipakai untuk | Jalur | Bila gagal | K | Status |
|---|---|---|---|---|---|
| DNS publik `<domain>` | `app.`, `files.` | Provider DNS | Outage total untuk resolusi baru | K1 | Q-CI-08 |
| Let's Encrypt (ACME) | Sertifikat publik | Traefik | Renewal gagal → AL-17 | K2 | — |
| Provider LLM | AI brief, klasifikasi kategori | Smokescreen role `ai` (allowlist host) | Brief gagal → AL-08, kill switch | K2 | D-15 Open (usulan kelas Claude Sonnet/Haiku); DPA/region TQ-02 |
| Provider email (SMTP relay) | Verifikasi email, undangan, notifikasi | Smokescreen role `mail` | Sign-up/undangan tertunda | K2 | TQ-12 |
| Payment gateway Indonesia | Pembayaran self-serve | — | — | — | V1 (FR-BILL-008). MVP 1: invoice manual + RB-28 |
| Off-site S3-compatible | repo2 pgBackRest, SeaweedFS off-site | Uplink DC | DR terganggu → AL-12/13/15 | K1 (DR) | Q-09/Q-CI-04 |
| Provider DC | Listrik, jaringan, remote hands, konsol IPMI/KVM | — | AL-03 | K1 | Q-CI-04 |
| Probe eksternal | Deteksi DC/`ops-01` down | Internet | Kehilangan deteksi | K2 | Q-CI-06 |
| Endpoint webhook customer | Pengiriman event | Smokescreen role `webhook` | Retry ≤ 24 jam → DLQ | K3 per tenant | — |
| DNS publik (MX/A/TXT) | Verifikasi email/domain | Unbound | `dns.lookup` off → UNKNOWN | K3 | — |

---

## 3. Akses operasional

### 3.1 Least privilege

| Akses | On-call engineer | Maintainer | Operator ops console | Engineer lain |
|---|:-:|:-:|:-:|:-:|
| WireGuard (peer personal) | ✓ | ✓ | ✓ | ✓ (staging) |
| SSH host prod (akun bernama, sudo tercatat auditd) | ✓ diagnosis | ✓ | — | — |
| Docker di app host (setara root) | ✓ restart/diagnosis | ✓ | — | — |
| DB `msp_readonly_replica` di `db-02` (alasan dicatat di issue) | ✓ | ✓ | — | — |
| `sudo -u postgres` di host DB (replikasi, pgBackRest, `pg_stat_*`), **tanpa** membaca data tenant | ✓ | ✓ | — | — |
| Write DB manual (`msp_owner`/superuser) | — | Break-glass (RB-01) | — | — |
| Job production (deploy, rollback, publish, activate, apply infra) | — | ✓ | — | — |
| Ops console: flag, DLQ | ✓ | ✓ | ✓ | — |
| Ops console: suppression, DSR, grant credit, koreksi data, akses support | — | ✓ | ✓ | — |
| Dekripsi SOPS production | — | ✓ | — | — (encrypt saja) |
| Grafana, Loki, GlitchTip production | ✓ | ✓ | Dashboard ops | Baca |

Akses DB manusia di production hanya `msp_readonly_replica` dengan alasan tercatat (06 §3.1); grant untuk tabel diagnosis non-tenant (`river_job`, `goose_db_version`) = Q-RB-22. Mutasi data tenant hanya lewat fitur aplikasi/ops console yang teraudit, bukan SQL. Ops console memakai akun internal + MFA dengan audience terpisah; pembatasan ke WireGuard = Q-RB-24.

### 3.2 WireGuard & SSH

- WireGuard berjalan di `ops-01` (UDP 51820). Setiap orang punya peer sendiri; kunci privat dibuat di laptop, kunci publik masuk lewat MR ke role `wireguard`, diterapkan via tag `infra-*`.
- SSH hanya dari subnet WireGuard dan IP privat `ops-01`, login dengan kunci personal, tanpa root login. Perubahan konfigurasi lewat SSH dilarang (06 §11.3).

```bash
sudo wg-quick up msp-prod && sudo wg show msp-prod latest-handshakes   # di laptop; interface server di ops-01 = wg0
# ~/.ssh/config
Host ops-01
  HostName <ip-wg-ops-01>
  User <nama-anda>
Host app-01 app-02 db-01 db-02 stor-01 dr-01
  User <nama-anda>
  ProxyJump ops-01
```

### 3.3 Unlock LUKS setelah reboot

Semua server memakai LUKS2. Unlock otomatis lewat Clevis + **Tang di `ops-01`**. Konsekuensi: `ops-01` harus hidup saat host lain reboot, dan `ops-01` sendiri tidak bisa auto-unlock dari dirinya. Fallback: SSH ke dropbear-initramfs (`ssh -p <port-dropbear> root@<ip-host>` lalu `cryptroot-unlock`) atau konsol IPMI/KVM provider dengan passphrase LUKS dari SOPS/salinan offline. Setelah unlock: secret tmpfs hilang → RB-23.

### 3.4 Custody kunci SOPS/age dan kredensial kritis

| Kunci | Lokasi | Pemegang | Pemulihan |
|---|---|---|---|
| age production (runner) | GitLab variable `SOPS_AGE_KEY` (protected, scope `production`) | Runner `deploy` | Maintainer membuat kunci baru → `sops updatekeys` (RB-20) |
| age Maintainer personal (≥ 2) | Perangkat Maintainer (disarankan token hardware atau passphrase) | Maintainer A, B | Maintainer lain mengenkripsi ulang |
| Salinan offline: age privat, passphrase repo1/repo2 pgBackRest, passphrase LUKS | Amplop tersegel/brankas di 2 lokasi fisik berbeda | 2 Maintainer (06 §11.6) | **Tanpa ini backup tidak dapat dibuka.** Uji dekripsi tiap kuartal (§8) |
| cosign private key | `infra/secrets/signing.sops.yaml` (scope `signing`) | Job `sign:images` | RB-20 |
| SSH deploy per environment | GitLab variable `DEPLOY_SSH_KEY` | Runner `deploy` | Rotasi 90 hari |
| Akun & kunci SSH `breakglass` | Offline | 2 Maintainer | RB-01; rotasi setelah dipakai |
| `WEBHOOK_SECRET_MASTER_KEYS`, `CURSOR_HMAC_KEY`, `SESSION_PEPPER`, `VALUE_HASH_PEPPER` | SOPS (`app.sops.yaml`) → tmpfs | Deploy | RB-20. `VALUE_HASH_PEPPER` **tidak** dirotasi tanpa re-hash suppression (Q-RB-16) |

### RB-01 — Akses break-glass

- **Tujuan:** akses darurat saat jalur normal (pipeline, GitLab, WireGuard, akun personal) tidak tersedia. **Kapan:** `ops-01`/GitLab mati dan perlu rollback/failover; WireGuard mati; perlu write DB yang tidak tersedia di ops console saat SEV1/SEV2.
- **Prasyarat:** insiden SEV1/SEV2 terdeklarasi; persetujuan IC (lisan, dicatat). Bila sendirian: catat alasan dan beri tahu IC ≤ 1 jam.

1. Catat di issue insiden: alasan, jalur yang dipakai, waktu mulai (UTC).
2. Pilih jalur: (a) GitLab mati, WireGuard hidup → jalankan playbook dari laptop Maintainer; (b) WireGuard mati → konsol IPMI/KVM provider atau endpoint WireGuard cadangan (Q-RB-02); (c) write DB → `sudo -u postgres psql` di `db-01`, satu transaksi eksplisit, dengan reviewer kedua (screen share).
3. Ambil kredensial offline (kunci `breakglass`, kunci age Maintainer). Rekam sesi terminal.
4. Jalankan hanya perintah yang sudah diumumkan di kanal insiden.

```bash
script -a ~/breakglass-$(date -u +%Y%m%dT%H%MZ).log
export SOPS_AGE_KEY_FILE=~/.config/sops/age/<maintainer>.txt
ansible-playbook -i infra/ansible/inventories/production infra/ansible/playbooks/rollback.yml -e to=previous
sudo -u postgres psql msp -v ON_ERROR_STOP=1     # BEGIN; <statement>; <verifikasi>; COMMIT;
```

- **Verifikasi:** tujuan tercapai; tidak ada sesi break-glass tersisa (`who`, `wg show`).
- **Abort:** hentikan bila hasil tidak sesuai rencana; kembalikan keputusan ke IC.
- **Audit:** log terminal (tanpa secret) dilampirkan ke issue; auditd; kredensial break-glass yang dipakai **dirotasi ≤ 24 jam**; MR adopsi untuk setiap perubahan ≤ 2 hari kerja.

### RB-02 — Akses support ke data tenant (FR-OPS-006, 04 §15.9)

- **Tujuan:** membaca data workspace untuk menangani tiket, tanpa impersonation. **Kapan:** tiket customer yang membutuhkan konteks data tenant (export gagal, hasil ICP aneh, list hilang).
- **Prasyarat:** akun ops MFA; nomor tiket; persetujuan customer tercatat di tiket (usulan).

1. Ops console → Workspaces → `<wsp_…>` → **Akses support** → isi alasan (wajib, sebut nomor tiket) dan durasi ≤ 4 jam (`POST /v1/ops/support-access`).
2. Selama grant aktif, konsol membuka `tenancy.WithTx` **READ ONLY**. Setiap request tercatat `support.read`.
3. Jangan menyalin data pribadi ke tiket/chat. Gunakan ID dan tangkapan layar yang disensor.
4. Butuh perubahan data? Tidak bisa lewat grant ini (mutasi → 403). Minta Owner/Admin, atau pakai prosedur khusus (RB-19, RB-28, RB-29).
5. Cabut grant segera setelah selesai.

- **Verifikasi:** entri grant + `support.read` muncul di audit workspace (terlihat Owner/Admin). **Abort:** cabut grant kapan saja. **Audit:** otomatis; sampel direview bulanan (§8).

### RB-03 — Onboarding & offboarding akses

- **Tujuan:** memberi/mencabut akses sesuai §3.1. **Kapan:** anggota bergabung, berganti peran, atau keluar. **Prasyarat:** persetujuan Tech lead; offboarding Maintainer dikerjakan Maintainer lain.

1. **Onboarding:** MR ke role `users` (akun + SSH key personal) dan `wireguard` (peer) → tag `infra-*`; akun GitLab dengan 2FA; akun ops console dengan MFA; masuk rotasi setelah shadowing.
2. **Offboarding (hari yang sama):** blokir user GitLab; cabut sesi ops console ("logout semua sesi"); hapus peer WireGuard dan akun host via MR + tag `infra-*` (darurat: `sudo wg set wg0 peer <pubkey> remove` di `ops-01`, lalu MR adopsi).
3. Bila Maintainer: rotasi kunci age (`sops updatekeys`), SSH deploy, token bot, kredensial break-glass (RB-20); perbarui §1.4 dan `.sops.yaml`.

- **Verifikasi:** akses ditolak di semua jalur; job `drift` bersih. **Rollback:** pencabutan keliru dipulihkan lewat MR yang sama. **Audit:** issue `access-change`.

---

## 4. Prosedur rutin

Aksi `msp tool <aksi>` di production selalu berjalan sebagai container one-off `msp-backend@<digest> tool …` yang dipicu pipeline (runner `deploy`, playbook `tool-job.yml`), bukan dari shell interaktif (06 §10). Aksi ops console memakai endpoint `/v1/ops/*` (04 §11.5): akun internal + MFA, audience sesi terpisah, selalu diaudit. Contoh `curl` di bawah memakai cookie sesi ops `<sesi-ops>`; UI ops console adalah jalur utama.

### RB-10 — Deploy production normal (06 §6)

- **Tujuan:** merilis `vX.Y.Z` lewat blue/green + canary Traefik, deploy by digest. **Kapan:** Senin–Kamis 10:00–15:00 WIB; bukan masa freeze; error budget bulan berjalan terpakai ≤ 75% (06 §6.5).
- **Prasyarat:** Maintainer; commit berstatus `staging-verified` (+ `rc-verified` untuk minor/major); drill restore terakhir lulus ≤ 35 hari (QG-35); on-call diberi tahu.

1. Umumkan di kanal chat: versi, Maintainer, on-call, perkiraan selesai.
2. Pipeline `main` commit kandidat → job manual `release:cut` → tag `vX.Y.Z` + GitLab Release.
3. Pipeline tag: tunggu `release:readiness` hijau. Gagal → baca laporan; bypass hanya lewat kebijakan waiver (06 §8.3).
4. Jalankan job manual `deploy:production`. Bila ada ≥ 2 Maintainer, pemicu harus berbeda dari pemicu `release:cut` (four-eyes).
5. Playbook: verifikasi cosign → secret ke tmpfs → `msp migrate up` (one-off) → start warna baru → stop worker warna lama (River soft stop) → canary 10% (10 menit) → 100% (bake 20 menit) → warna lama dihentikan. `watch_canary` me-rollback otomatis bila threshold 06 §6.3 dilanggar.
6. Pantau Grafana *API & SLO* (filter `msp.color=<warna-baru>`) dan Pipeline Health (antrean, discarded).

```bash
ssh app-01 'C=$(cat /opt/msp/active_color); echo "aktif: $C"; docker compose -p msp-$C ps --format "{{.Service}} {{.Status}}"'
ssh app-01 'C=$(cat /opt/msp/active_color); docker inspect --format "{{.Config.Image}}" msp-$C-api-1; docker exec msp-$C-api-1 /msp tool healthcheck'
```

- **Verifikasi:** checklist pasca-deploy (§9.2); commit status `production-deployed`; release GlitchTip `msp@vX.Y.Z`; anotasi Grafana.
- **Rollback/abort:** selama canary/bake → rescue otomatis (< 1 menit). Setelah bake → RB-11. **Audit:** deployment GitLab environment `production` (aktor = pemicu job).

### RB-11 — Rollback aplikasi ke digest sebelumnya (06 §14.1)

- **Tujuan:** mengembalikan aplikasi ke digest sebelumnya ≤ 5 menit. **Kapan:** regresi setelah bake yang tidak bisa diatasi kill switch (RB-19) dalam ≤ 15 menit.
- **Prasyarat:** Maintainer; tidak ada migrasi `db-contract` di antara versi aktif dan versi target.

1. Bila masalah terlokalisasi pada satu kapabilitas, matikan flag dulu (06 §14.2).
2. Pipeline tag versi aktif → job manual **`rollback:production`** (`rollback.yml -e to=previous`): start warna lama dengan digest sebelumnya (image masih di cache host), worker dulu, alihkan traffic 100%, smoke.
3. Lebih dari satu rilis ke belakang: `-e to=vX.Y.Z` (≤ 15 menit; ditolak bila ada migrasi contract di antaranya).
4. GitLab/`ops-01` tidak tersedia: jalankan dari laptop Maintainer via WireGuard (RB-01 jalur a, ≤ 30 menit).
5. Rollback tidak menjalankan migrasi. Skema N tetap dipakai dan aman karena expand-only terhadap N-1.

```bash
ansible-playbook -i infra/ansible/inventories/production infra/ansible/playbooks/rollback.yml -e to=previous
ansible-playbook -i infra/ansible/inventories/production infra/ansible/playbooks/rollback.yml -e to=v1.4.2
```

- **Verifikasi:** digest container aktif = digest target; smoke lulus; 5xx dan p95 normal selama 30 menit; antrean River normal.
- **Abort:** bila versi lama juga gagal (masalah data/skema), jangan bolak-balik. IC memutuskan forward fix (RB-12/RB-13).
- **Audit:** deployment GitLab + issue `incident`; dihitung sebagai change failure (DORA).

### RB-12 — Hotfix (06 §6.6)

- **Tujuan:** merilis perbaikan cepat tanpa membawa perubahan lain yang belum siap. **Kapan:** bug produksi yang tidak cukup ditangani flag/rollback. Hotfix boleh di luar jendela deploy.
- **Prasyarat:** Maintainer (`release:cut`, deploy); on-call + Maintainer sebagai reviewer.

1. Containment dulu (RB-19/RB-11).
2. **Default — fix-forward dari `main`:** MR `fix/…` → review dipercepat (on-call + Maintainer) → merge → `staging-verified` → `release:cut` (patch) → RB-10. `rc-verified` tidak wajib untuk patch.
3. **Bila `main` memuat perubahan yang tidak boleh rilis:** Maintainer membuat `release/vX.Y` dari tag terakhir → cherry-pick → pipeline branch membangun dan memverifikasi di staging (staging sementara menjalankan versi hotfix) → `release:cut` di branch tersebut → `vX.Y.(Z+1)` → RB-10.
4. Fix yang sama wajib masuk `main` lewat MR terpisah.

- **Verifikasi:** gejala hilang di production; MR ke `main` ter-merge. **Abort:** gagal di canary → rescue otomatis; setelah bake → RB-11. **Audit:** link MR + tag di issue insiden.

### RB-13 — Migrasi DB bermasalah (forward fix, 06 §9 & §14.3)

- **Tujuan:** menangani migrasi yang gagal atau berdampak tanpa down-migration. **Kapan:** `msp migrate up` gagal saat deploy, lock contention, index INVALID, atau regresi performa/data setelah migrasi.
- **Prasyarat:** Maintainer + on-call; `sudo -u postgres` di `db-01` untuk diagnosis.

1. **Gagal saat deploy:** deploy berhenti dan warna lama tetap melayani (06 §5.2). Baca output container `msp-migrate-<versi>` di log job GitLab. Gagal karena `lock_timeout` 5 s (M-4) → cari blocker, ulangi deploy saat sepi.
2. **File `NO TRANSACTION` setengah jalan** (mis. `CREATE INDEX CONCURRENTLY` gagal) → index INVALID; perbaiki dengan migrasi forward (`DROP INDEX CONCURRENTLY IF EXISTS …` lalu buat ulang).
3. **Migrasi sukses tetapi aplikasi error/lambat** → rollback aplikasi (RB-11; skema tetap) atau flag, lalu migrasi perbaikan lewat pipeline normal yang dipercepat (review on-call + Maintainer).
4. **Migrasi merusak data** → hentikan writer terkait (flag, stop worker), nilai forward-fix dari data yang ada vs PITR (RB-41; keputusan IC + founder).
5. **Dilarang:** `msp migrate down` (ditolak binary), mengedit `goose_db_version`, `DROP`/`ALTER` manual di luar migrasi.

```sql
-- db-01: sudo -u postgres psql msp
SELECT version_id, is_applied, tstamp FROM goose_db_version ORDER BY id DESC LIMIT 5;
SELECT pid, now()-query_start AS umur, wait_event_type, state, left(query,100) FROM pg_stat_activity
 WHERE datname='msp' AND (wait_event_type='Lock' OR now()-query_start > interval '5 min');
SELECT pg_blocking_pids(<pid>);
SELECT indexrelid::regclass FROM pg_index WHERE NOT indisvalid;
SELECT pg_cancel_backend(<pid>);   -- hanya dengan persetujuan IC; jangan terminate tanpa alasan kuat
```

- **Verifikasi:** versi goose sesuai; tidak ada index INVALID; smoke + p95 normal.
- **Rollback:** tidak ada (forward only); PITR sebagai upaya terakhir. **Audit:** issue insiden + MR forward fix (label `db-hotfix`, usulan).

### RB-14 — Skala worker & backlog antrean River

- **Tujuan:** menurunkan umur antrean (AL-05) tanpa membebani DB berlebihan. **Kapan:** `msp_queue_oldest_age_seconds > 3600`, fan-out re-skor besar, pipeline run panjang.
- **Prasyarat:** SSH + Docker di `app-01` (on-call); ops console untuk flag.

1. Identifikasi antrean dan penyebab: worker mati/restart loop, job lambat karena downstream (LLM, webhook), DB jenuh, atau banjir job.
2. Worker mati → `docker compose -p msp-<warna> up -d worker-<q>`. Job `running` yang tertinggal di-rescue River setelah 15 menit.
3. Prioritaskan interaktif: matikan `scoring.bulk_rescore` (fan-out ditunda, prioritas 1 tetap jalan).
4. Tambah kapasitas sementara dengan menambah replika container worker (aman untuk River). Permanen: MR `MSP_WORKER_MAX_<QUEUE>` + deploy ulang digest yang sama.
5. Jangan scale bila koneksi `db-01` > 80% `max_connections` (300) atau CPU `db-01` > 85%. River memakai koneksi langsung, bukan PgBouncer.
6. Downstream lambat (LLM/webhook) → jangan scale; lihat AL-08/AL-10.

```bash
ssh app-01 'docker compose -p msp-green up -d --no-recreate --scale worker-export=2 worker-export'
# ringkasan antrean per state: query River di §9.1
```

- **Verifikasi:** umur antrean < 1 jam; tidak ada lonjakan `discarded`.
- **Rollback:** kembalikan replika (`--scale worker-export=1`) dan nyalakan `scoring.bulk_rescore` setelah backlog habis. **Audit:** flag teraudit otomatis; scale manual dicatat di issue + MR adopsi bila permanen.

### RB-15 — Re-drive job River `discarded` (DLQ, FR-OPS-002)

- **Tujuan:** memproses ulang job yang kehabisan percobaan setelah penyebabnya diperbaiki. **Kapan:** AL-06, laporan user (export gagal), pemulihan pasca-insiden.
- **Prasyarat:** akun ops console; penyebab sudah diketahui.

1. Ops console → Pipeline Health → DLQ (`GET /v1/ops/dlq`). Kelompokkan per `queue`, `kind`, error terakhir.
2. Klasifikasi: (a) transient (timeout DB/jaringan) → redrive; (b) bug → tunggu fix ter-deploy, lalu redrive; (c) data buruk/non-retryable → jangan redrive, catat (perbaikan di mapping/snapshot berikutnya).
3. Redrive: `POST /v1/ops/dlq/{id}/redrive` (River `JobRetry`). Untuk banyak job: batch ≤ 100, pantau antrean di antara batch (bulk redrive = Q-RB-06).
4. Stage pipeline: alternatif re-run stage lewat `/v1/ops/pipeline-runs/{id}` (batch `SUCCEEDED` dilewati, idempoten).
5. Job `webhook.deliver` → RB-16. Job `export` gagal → hold sudah di-release; user dapat me-retry export (status `FAILED`).

```sql
-- db-02 (msp_readonly_replica): ringkasan DLQ 24 jam
SELECT queue, kind, count(*), max(finalized_at) FROM river_job
 WHERE state='discarded' AND finalized_at > now()-interval '24 hours' GROUP BY 1,2 ORDER BY 3 DESC;
SELECT id, kind, attempt, errors[array_length(errors,1)]->>'error' AS error_terakhir
  FROM river_job WHERE state='discarded' ORDER BY finalized_at DESC LIMIT 20;
```

- **Verifikasi:** job redrive `completed`; `msp_dlq_size` turun; tidak ada efek ganda (job idempoten per `idem_key`).
- **Abort:** redrive memicu error massal → hentikan, kembali ke diagnosis.
- **Audit:** setiap redrive tercatat di audit (aktor ops).

### RB-16 — Redrive webhook & endpoint gagal (FR-EXP-003, 04 §12.4)

- **Tujuan:** memulihkan pengiriman webhook setelah kegagalan. **Kapan:** AL-10; endpoint `DISABLED_FAILING`; delivery masuk DLQ karena masalah di sisi MSP.
- **Prasyarat:** ops console (flag), Grafana/Loki; Maintainer untuk perubahan Smokescreen/nftables (MR).

1. Tentukan sisi kegagalan dari dashboard Webhook: satu endpoint dominan (sisi customer) vs semua endpoint (sisi MSP: Smokescreen, nftables, Unbound, CA bundle, network `worker-webhook`).
2. **Sisi customer:** MSP tidak bertindak selain memastikan Admin mendapat email otomatis (≥ 20 DLQ berturut-turut → `DISABLED_FAILING`; `410` → langsung disable). Admin memperbaiki endpoint, mengaktifkan ulang, dan memakai **Kirim ulang** (`POST /v1/webhook-deliveries/{id}/redeliver`, tanpa tagihan baru).
3. **Sisi MSP:** matikan `webhook.delivery` (delivery di-snooze, tetap `PENDING`), perbaiki, lalu nyalakan lagi **≤ 24 jam**. Delivery yang berumur > 24 jam saat dilanjutkan masuk DLQ.
4. Delivery yang telanjur DLQ karena kesalahan MSP: kirim daftar delivery ID ke Admin workspace terdampak (template §7.4) untuk redeliver, atau ops me-redrive job `webhook.deliver` lewat RB-15 setelah semantik credit disepakati (Q-RB-15). Payload berasal dari `platform.event` (retensi 30 hari), sehingga redrive setelah 30 hari tidak mungkin.
5. Delivery yang memuat subjek ter-suppress berstatus `CANCELLED` dan **tidak boleh** dikirim ulang.

```logql
sum by (http_status, error_code) (count_over_time({service="worker-webhook", env="production"} | json | error_code!="" [15m]))
{service="smokescreen", env="production"} | json | decision="deny"
```

- **Verifikasi:** success ratio agregat > 95%; tidak ada endpoint disabled akibat kesalahan MSP. **Abort:** menyalakan kembali `webhook.delivery` memicu kegagalan massal → matikan lagi (selama masih ≤ 24 jam). **Audit:** flag dan redrive teraudit; daftar workspace terdampak di issue.

### RB-17 — Publish & rollback snapshot dataset `ds-YYYY.MM.n` (06 §10.2, 04 §4.2)

- **Tujuan:** menerbitkan snapshot baru dengan gate QG-34, atau kembali ke snapshot sebelumnya. **Kapan:** rilis data terjadwal (kadens Q-03, default kuartalan) atau koreksi massal; snapshot lama tidak pernah diedit.
- **Prasyarat:** data owner (Maintainer); file + manifest dengan sha256 per file dan tag lisensi (BR-PRIV-04).

1. Ops mengunggah file mentah + manifest ke bucket landing production (`msp-snapshots/<source>/<versi>/`, kredensial tanpa hak overwrite/delete).
2. MR manifest `data/snapshots/ds-….yaml` → merge.
3. Maintainer membuat tag `ds-YYYY.MM.n` → pipeline: [1] register + checksum → [2] validasi (production, `VALIDATING`) → [3] rehearsal staging (sampel tanpa kontak pribadi) → [4] gate QG-34 → [5] job manual **`publish`** → [6] recompute skor stale → [7] verifikasi.
4. Selama pipeline run (≤ 12 jam, NFR-PERF-07), pantau Pipeline Health (AL-04/05). `pipeline.publish` off menahan langkah 5.
5. Setelah publish: pantau fan-out re-skor (antrean `scoring` prioritas 4, maks 4 berjalan) dan Data Health (evidence coverage ≥ 95%).

```bash
aws --endpoint-url https://<s3-internal>:8333 s3 cp ./ds-2026.11.1/ s3://msp-snapshots/<source>/ds-2026.11.1/ --recursive
git tag -a ds-2026.11.1 -m "Snapshot <source> 2026-11 #1 (issue #<n>)" && git push origin ds-2026.11.1
```

- **Verifikasi:** event `snapshot.published`; Data Health per snapshot sesuai gate; tidak ada lonjakan stale yang tidak di-recompute.
- **Rollback:** snapshot immutable dan ID entity stabil. Rollback = publish ulang snapshot sebelumnya + recompute (06 §14.4). Transisi `SUPERSEDED → PUBLISHED` belum ada di state machine 04 §4.2 (Q-RB-07). Partisi observation snapshot sebelumnya masih online (`observation_online_snapshots: 2`).
- **Audit:** aktor = user GitLab yang menjalankan `publish`; override gate wajib alasan (audit).

### RB-18 — Aktivasi & rollback versi skor (`priority-X.Y.Z`), geo (`geo-X.Y.Z`), prompt (`brief-X.Y.Z`)

- **Tujuan:** mengaktifkan versi model/konfigurasi yang sudah ter-deploy (status `draft`) secara terkontrol (06 §10.3–§10.4). **Kapan:** versi baru lulus MR (QG-23); rollback bila Intelligence Health, feedback, atau alert memburuk.
- **Prasyarat:** kode versi sudah di production; issue rilis model dengan approval data-intel + product.

| Langkah | Skor & geo | Prompt AI |
|---|---|---|
| Tag | `priority-X.Y.Z` / `geo-X.Y.Z` (Maintainer) | `brief-X.Y.Z` |
| Uji | `msp tool score-reproduce` di staging → laporan regresi QG-24 (transisi band, overlap top-50, coverage, determinisme) | Eval staging set diperluas (QG-25: grounding ≥ 98%, kontak pribadi di prompt = 0) |
| Approval | Justifikasi product bila transisi band > 15% sampel atau overlap top-50 < 70% | Data-intel + product |
| Aktivasi | Job manual `activate` → `msp tool score-model-activate` → recompute bertahap, epoch atomik | Flag `rel.ai.brief-<versi>`: workspace internal → cohort beta → 100% |
| Pantau | Intelligence Health; AL-05, AL-07 | AL-08, AL-09, feedback 👍/👎, biaya per brief. Mekanisme aktivasi (config vs flag) dan tag `angle-*` = Q-RB-23 |
| Rollback | Aktifkan kembali epoch versi sebelumnya (disimpan selama bake 7 hari); setelah bake, versi lama direproduksi dari snapshot + versi | Flag `rel.ai.brief-<versi>` off (kembali ke versi aktif sebelumnya); kill switch `ai.generation` sebagai upaya terakhir |

```bash
git tag -a priority-1.3.0 -m "Model skor 1.3.0 (issue #<n>)" && git push origin priority-1.3.0
# rollback dalam masa bake (ilustratif; aksi rollback = Q-RB-06)
ansible-playbook -i infra/ansible/inventories/production infra/ansible/playbooks/tool-job.yml -e tool_args="score-model-activate --model priority-1.2.0"
```

- **Verifikasi:** `model_version`/`geo_version` di skor baru sesuai; job skor ≤ 10 menit per ICP (NFR-PERF-05); determinisme (hash identik) di laporan. **Rollback:** baris *Rollback* di tabel. **Audit:** aktor job GitLab; laporan di `msp-evidence`. Aktivasi geo belum punya aksi `tool` bernama (Q-RB-06).

### RB-19 — AI kill switch & feature flag (FR-OPS-007, 04 §18.3, 06 §13)

- **Tujuan:** mematikan kapabilitas bermasalah ≤ 1 menit tanpa deploy. **Kapan:** langkah containment pertama untuk masalah yang terlokalisasi. **Prasyarat:** akun ops console + MFA (on-call atau ops).

| Flag | Efek bila off | Pakai saat |
|---|---|---|
| `ai.generation` | Tombol AI nonaktif, `503 FEATURE_DISABLED` | AL-08, AL-09 (otomatis saat budget harian habis), AL-24 |
| `export.create` | Export baru ditolak; job berjalan tetap selesai | Bug export, stor-01 bermasalah, AL-20 |
| `webhook.delivery` | Delivery di-snooze (tetap `PENDING`, nyalakan ≤ 24 jam) | Masalah egress sisi MSP, AL-18 (scope WORKSPACE) |
| `upload.create` | Upload baru ditolak | Bug parser, stor-01 bermasalah |
| `pipeline.publish` | Snapshot tidak dapat di-publish | Gate meragukan, insiden DB |
| `dns.lookup` | Verifikasi email → UNKNOWN (bukan INVALID) | Unbound/DNS publik bermasalah |
| `search.include_private` | Toggle private prospect disembunyikan | Bug union private prospect |
| `scoring.bulk_rescore` | Fan-out re-skor ditunda | Backlog, beban DB, p95 search |

1. Ops console → Feature flags → `<key>` → pilih scope (GLOBAL → PLAN → WORKSPACE; yang paling spesifik menang) → nilai → alasan (nomor insiden).
2. Efek ≤ 1 menit (cache 30 s + pub/sub). Verifikasi di UI/API.
3. Off > 30 menit dan terlihat customer → komunikasikan (§7.4).
4. Saat menyalakan kembali: cek efek samping (delivery snooze > 24 jam → DLQ; backlog fan-out re-skor).

```bash
curl -fsS -X PUT "https://app.<domain>/api/v1/ops/feature-flags/ai.generation" -H "Cookie: <sesi-ops>" \
  -H "Content-Type: application/json" -d '{"scope":"GLOBAL","enabled":false,"reason":"INC-<n>: AI failure > 5%"}'
```

- **Verifikasi:** perilaku sesuai kolom *Efek bila off* dalam ≤ 1 menit. **Rollback:** nyalakan kembali (langkah 4). **Audit:** setiap perubahan flag teraudit otomatis.
- **Catatan:** bila flag store gagal, kill switch memakai nilai cache terakhir; tanpa cache → kapabilitas **on** (06 §13.1). Flag `rel.*` default off.

### RB-20 — Rotasi secret (06 §12.4, 04 §15.4)

- **Tujuan:** mengganti secret secara aman. **Kapan:** sesuai jadwal tabel; segera bila dugaan bocor, offboarding Maintainer, atau runner dikompromikan.
- **Prasyarat:** Maintainer (dekripsi SOPS production).

| Secret | Jadwal | Cara |
|---|---|---|
| Password role DB, kunci S3 SeaweedFS, Redis ACL | 90 hari | `rotate-secrets.yml` (dual credential selama transisi) → deploy/rehydrate → cabut kredensial lama |
| Kunci master webhook (`WEBHOOK_SECRET_MASTER_KEYS`) | 12 bulan | (1) tambah versi kunci baru ke SOPS + deploy (dekripsi menerima semua versi aktif, enkripsi pakai terbaru); (2) job River mengenkripsi ulang semua `secret_ciphertext`; (3) setelah 100% + 7 hari, hapus versi lama |
| Kunci cosign | 12 bulan | `cosign generate-key-pair` → private ke `signing.sops.yaml`, publik ke `infra/keys/`; verifikasi menerima kunci lama + baru 30 hari. Bocor → tanda tangan ulang semua digest yang disimpan |
| Kunci age per environment | 12 bulan | `age-keygen` → update `.sops.yaml` → `sops updatekeys` semua file → ganti variable `SOPS_AGE_KEY` → hapus kunci lama |
| Kunci SSH deploy | 90 hari | Kunci baru → `authorized_keys` via Ansible → ganti `DEPLOY_SSH_KEY` |
| Token GitLab (bot rilis, Renovate, trigger) | ≤ 365 hari | Ganti 30 hari sebelum kedaluwarsa |
| API key LLM / email | 90 hari | Konsol provider → SOPS → deploy |
| `CURSOR_HMAC_KEY`, `SESSION_PEPPER` | Tidak terjadwal (Q-RB-16) | Rotasi membatalkan cursor terbuka / mengeluarkan semua sesi; umumkan bila sengaja |
| `VALUE_HASH_PEPPER` | **Jangan dirotasi** tanpa migrasi re-hash | Rotasi memutus pencocokan suppression (`value_hash`) → risiko insiden privasi |

```bash
sops infra/ansible/inventories/production/secrets/app.sops.yaml
sops updatekeys -y infra/ansible/inventories/production/secrets/*.sops.yaml
ansible-playbook -i infra/ansible/inventories/production infra/ansible/playbooks/rotate-secrets.yml -e secret=db_app
```

- **Verifikasi:** semua service healthy; kredensial lama ditolak (uji eksplisit); tidak ada `authentication failed` di log setelah transisi.
- **Rollback:** selama dual credential, kembali ke kredensial lama di SOPS + deploy. **Audit:** issue `secret-rotation` (tanggal, secret, pelaku; tanpa nilai). Rotasi karena dugaan bocor = insiden (§7).

### RB-21 — Pembaruan sertifikat TLS

- **Tujuan:** mencegah sertifikat kedaluwarsa. **Kapan:** AL-17, atau inventaris sertifikat bulanan (§8). **Prasyarat:** on-call (Traefik), Maintainer (CA internal via Ansible).
- **Cakupan:** publik (`app.<domain>`, `files.<domain>`: Traefik ACME Let's Encrypt, renew otomatis ±30 hari sebelum habis) dan internal (CA internal Ansible: PostgreSQL `verify-full`, Redis TLS, mTLS gRPC SeaweedFS, GitLab bila memakai CA internal/DNS-01).

1. Cek sisa umur dan log ACME Traefik. Penyebab umum: port 80 terblokir (HTTP-01), record DNS berubah, token DNS-01 kedaluwarsa, rate limit Let's Encrypt.
2. Perbaiki penyebab lalu restart Traefik agar ACME mencoba ulang. `acme.json` rusak → backup file, hapus entri domain terkait, restart.
3. Sertifikat internal: terbitkan ulang lewat role Ansible terkait (`site.yml --tags certs`, nama tag usulan) via tag `infra-*`, restart service berurutan (replika dulu).

```bash
echo | openssl s_client -connect app.<domain>:443 -servername app.<domain> 2>/dev/null | openssl x509 -noout -enddate
ssh app-01 'docker logs --since 24h traefik 2>&1 | grep -i acme | tail -20'
ssh db-01 'sudo openssl x509 -noout -enddate -in /etc/postgresql/16/main/server.crt'
```

- **Verifikasi:** `probe_ssl_earliest_cert_expiry` > 30 hari; koneksi DB `verify-full` sukses. **Rollback:** pulihkan `acme.json` dari backup; sertifikat lama tetap dipakai sampai kedaluwarsa. **Audit:** issue bila manual.

### RB-22 — Patch OS rolling & reboot

- **Tujuan:** menerapkan patch keamanan dan reboot tanpa mengganggu SLO. **Kapan:** unattended-upgrades memasang patch security harian (tanpa auto-reboot di host DB). Reboot di jendela maintenance bulanan, atau ≤ 7 hari untuk kerentanan kritis.
- **Prasyarat:** silence Alertmanager untuk host terkait; `ops-01` (Tang) sehat; pemberitahuan customer ≥ 48 jam bila `app-01` di-reboot (usulan, selama `app-02` belum ada).

1. Daftar host yang butuh reboot. Urutan: `stg-01` → `ci-01` → `db-02` → `stor-01` → `db-01` → `app-01` → `ops-01` terakhir (dan sendirian).
2. `db-02`: reboot, tunggu replikasi kembali streaming dan lag < 60 s.
3. `stor-01`: matikan `export.create` dan `upload.create` selama reboot; WAL archive repo1 tertunda sebentar.
4. `db-01`: switchover terencana (RB-43 langkah 3) **atau** jeda write singkat ≤ 5 menit di jam sepi dengan worker dihentikan dan flag writer off.
5. `app-01`: downtime singkat (SPOF, R-08); setelah boot jalankan RB-23.
6. `ops-01`: unlock LUKS via dropbear/konsol (§3.3); pastikan GitLab, Prometheus, Tang kembali.

```bash
ansible -i infra/ansible/inventories/production all -b -m shell -a 'test -f /var/run/reboot-required && cat /var/run/reboot-required.pkgs || true'
amtool --alertmanager.url=http://localhost:9093 silence add instance=~"db-02.*" --duration=1h --comment="RB-22 reboot db-02"
ansible-playbook -i infra/ansible/inventories/production infra/ansible/playbooks/os-patch.yml --limit db-02   # serial: 1, nama usulan (Q-RB-05)
```

- **Verifikasi:** host up, LUKS terbuka, service healthy, replikasi & backup normal, silence dihapus. **Abort:** host gagal boot → konsol IPMI, boot kernel sebelumnya; hentikan rolling, jangan lanjut ke host berikutnya. **Audit:** issue maintenance; deployment infra tercatat.

### RB-23 — Rehydrate secret setelah reboot (06 §12.3)

- **Tujuan:** memulihkan secret tmpfs `/run/msp/<warna>/` yang hilang setelah reboot.
- **Kapan/gejala:** setelah reboot `app-01`, container `msp-*` restart loop dengan error konfigurasi wajib hilang. **Prasyarat:** on-call (cek), Maintainer (playbook dari laptop).

1. Normalnya unit systemd `msp-rehydrate` memicu pipeline `rehydrate` (trigger token) → runner `deploy` menjalankan `rehydrate.yml`.
2. Cek: `systemctl status msp-rehydrate` dan status pipeline di GitLab.
3. Gagal → jalankan ulang pipeline dari GitLab, atau Maintainer dari laptop (RB-01 bila GitLab mati).
4. Start ulang project warna aktif.

```bash
ansible-playbook -i infra/ansible/inventories/production infra/ansible/playbooks/rehydrate.yml --limit app-01
ssh app-01 'ls /run/msp/$(cat /opt/msp/active_color) | wc -l; docker compose -p msp-$(cat /opt/msp/active_color) up -d'
```

- **Verifikasi:** jumlah file secret sesuai (jangan `cat` isinya); healthcheck lulus. **Abort:** gagal berulang → cek variable `SOPS_AGE_KEY` dan trigger token, eskalasi Maintainer. **Audit:** log job GitLab.

### RB-24 — Disk penuh

- **Tujuan:** memulihkan ruang disk tanpa merusak data. **Kapan:** AL-14 (> 80%) atau tren habis < 7 hari. **Prasyarat:** SSH + sudo (on-call); menghapus slot/data hanya dengan persetujuan IC.
- **Aturan:** jangan pernah menghapus file di `pg_wal`, file repo pgBackRest (pakai `expire`), atau image 5 rilis terakhir di app host.

| Host | Penyebab umum | Tindakan |
|---|---|---|
| `db-01` | WAL menumpuk karena archive gagal (AL-12) atau replication slot tidak aktif (db-02 mati); bloat `river_job`, `platform.event`, `webhook_delivery`, epoch skor lama | Perbaiki archive; slot tidak aktif lama → drop hanya dengan persetujuan IC (db-02 harus dibangun ulang); cek autovacuum dan job retensi |
| `app-01` | Image lama, log container | `docker image prune -f` (dangling saja); hapus image lama secara eksplisit di luar 5 rilis terakhir |
| `stor-01` | Export tidak kedaluwarsa, repo pgBackRest, volume perlu vacuum | Cek lifecycle `msp-exports` (24 jam); `pgbackrest expire`; `volume.vacuum`; tambah disk/volume |
| `ops-01` | Prometheus/Loki melebihi retensi, registry, backup GitLab | Cek retensi 30 hari; cleanup policy registry + `gitlab-ctl registry-garbage-collect` |
| `ci-01` | Cache build/runner | Prune cache BuildKit/runner |

```bash
df -hT && sudo du -xh --max-depth=2 /var /opt 2>/dev/null | sort -h | tail -15
docker system df && docker image ls --digests | head -20
sudo -u postgres psql -c "SELECT slot_name, active, pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn)) FROM pg_replication_slots;"
sudo -u postgres pgbackrest --stanza=msp expire
```

- **Verifikasi:** penggunaan < 75% dan tren (`predict_linear`) aman 14 hari. **Abort:** ragu → jangan hapus; menambah kapasitas lebih aman daripada menghapus. **Audit:** issue + action item kapasitas.

### RB-25 — Pemrosesan suppression (FR-AUD-003, NFR-PRIV-02 ≤ 24 jam, 04 §15.7)

- **Tujuan:** menghapus person/contact point dari semua projection dan sistem hilir ≤ 24 jam (target operasional ≤ 1 jam). **Kapan:** DSR hapus/keberatan (RB-26), permintaan langsung, atau keputusan ops.
- **Prasyarat:** operator ops console; dasar permintaan tercatat (nomor DSR/tiket).

1. Ops console → Suppression → tambah (`POST /v1/ops/suppressions`): jenis (contact point email/telepon, person, establishment), nilai (disimpan sebagai `value_hash`), alasan, rujukan DSR. T0 = `created_at`. Nilai mentah tidak ditulis di tiket.
2. Sistem menjalankan rantai propagasi dalam satu transaksi + job: global layer `suppressed=true` (RLS langsung menyembunyikan) → projection `prospect_search` + cache profil/facet → AI artifact `INVALIDATED` → delivery `PENDING/RETRYING` yang memuat subjek `CANCELLED` → file export **aktif** yang memuat subjek (via `action.distribution_log`) dibuat ulang tanpa record itu, link lama dicabut, pemilik diberi notifikasi → event **`suppression.applied`** (tanpa nilai kontak) ke setiap workspace yang pernah menerima record lewat export/webhook.
3. Pantau detail suppression: sweep per jam mengisi `propagated_at` bila kebocoran = 0.
4. Verifikasi propagasi: (a) subjek tidak muncul di search/profil dari workspace sintetis `msp-synthetic-monitor` (cari dengan nama usaha, bukan nilai kontak); (b) file export aktif terkait sudah dibuat ulang; (c) event `suppression.applied` terkirim ke workspace penerima; (d) `suppression_leak_count` = 0.
5. Data upload tenant tidak di-suppress otomatis (tenant = pengendali). Teruskan ke tenant (RB-26 langkah 5); nilai yang sama di global layer tetap di-suppress.
6. Snapshot berikutnya: laporan validasi menunjukkan `SUPPRESSED_VALUE` untuk nilai yang cocok.
7. Catat `propagated_at − created_at` di DSR/tiket, lalu tutup.

```promql
max(suppression_leak_count)
max(msp_suppression_propagation_lag_seconds)   # nama metrik lag = usulan (Q-RB-09)
```

- **Verifikasi:** langkah 4 lulus dan `propagated_at` terisi ≤ 24 jam setelah T0.
- **Rollback:** hapus suppression (`DELETE /v1/ops/suppressions/{id}`) hanya bila salah input dan disetujui DPO. **Audit:** entri suppression (aktor, alasan) di audit; event `suppression.created` internal.

### RB-26 — Permintaan subjek data / DSR (FR-AUD-004, FR-OPS-008, 04 §15.8)

- **Tujuan:** memenuhi hak akses, koreksi, hapus, dan keberatan subjek data dalam tenggat. **Kapan:** DSR masuk lewat form/email. **Prasyarat:** operator ops console; DPO untuk penolakan dan penerusan ke tenant.
- **Status:** `RECEIVED → VERIFYING → IN_PROGRESS → [FORWARDED_TO_TENANT] → COMPLETED | REJECTED`.

1. **Intake:** form publik (captcha + verifikasi email) atau email ops → `ops.data_subject_request`. Pengakuan penerimaan otomatis ≤ 1 hari kerja; untuk intake email, ops membuat record di hari yang sama.
2. **Verifikasi identitas** pemohon (verifikasi email; kepemilikan nomor telepon lewat OTP/panggilan balik; prosedur dokumen ditetapkan bersama hukum). Jangan mengonfirmasi ada/tidaknya data sebelum terverifikasi.
3. **Cari subjek** via `value_hash` di ops console. Hasil: global layer (person/contact point) dan kecocokan di upload tenant (hanya workspace ID + jumlah, bukan isi).
4. **Eksekusi per tipe:** akses → laporan data (field, sumber/snapshot, tujuan pemrosesan) lewat kanal aman; koreksi → laporan data → RB-29; hapus/keberatan → RB-25 + penghapusan nilai P1 dari global layer.
5. **Data upload tenant:** status `FORWARDED_TO_TENANT`, beri tahu Owner/Admin workspace (notifikasi + email), dieksekusi atas instruksi tenant (BR-PRIV-02).
6. **Tenggat** per tipe dikonfigurasi berdasarkan review hukum (beberapa hak 3×24 jam; Q-RB-14). Eskalasi ke DPO bila sisa tenggat < 24 jam.
7. Tutup `COMPLETED`/`REJECTED` dengan alasan; balas pemohon.

- **Verifikasi:** timeline status lengkap, tenggat terpenuhi. **Abort:** identitas tidak terverifikasi → `REJECTED` beralasan; pembatalan suppression mengikuti RB-25. **Audit:** semua transisi teraudit; register DSR dipegang DPO.

### RB-27 — Penutupan workspace (FR-AUD-005, 04 §15.8)

- **Tujuan:** menghapus data tenant setelah masa tenggang 30 hari dan melakukan pseudonimisasi audit/ledger. **Kapan:** workspace `PENDING_DELETION`; eksekusi hari ke-30. **Prasyarat:** operator ops (pantau), DPO (pengecualian).

1. Owner menutup workspace → `PENDING_DELETION`: akses diblokir, delivery webhook berhenti; Owner masih dapat membatalkan. Permintaan lewat support → konfirmasi bahwa pemohon adalah Owner.
2. Hari ke-30: job `purge_workspace` (antrean `maintenance`) memanggil handler purge setiap modul (semua tabel tenant), menghapus prefix `{workspace_id}/` di object storage, key Redis `ws:{id}:*`, dan sesi → verifikasi hitungan = 0 → `audit(workspace.purged)`.
3. `audit_log` dan `credit_ledger`/`usage_event` dipertahankan dan **dipseudonimkan**: ID aktor diganti HMAC ber-pepper, baris `audit_log_pii` dihapus, metadata P1 dibuang (waktu pseudonimisasi = Q-RB-12).
4. Ops memeriksa laporan purge (jumlah per tabel sebelum/sesudah). Job gagal → RB-15 (job idempoten).
5. Backup PITR menua keluar sesuai retensi pgBackRest (14 hari); salinan off-site append-only mengikuti retensinya (Q-RB-12). Penghapusan lebih awal dari 30 hari tidak didukung MVP → eskalasi DPO.

```bash
echo "fs.ls /buckets/msp-uploads/<workspace_id>" | weed shell -master=stor-01:9333       # harapan: kosong
ssh app-01 'docker exec <redis> redis-cli --tls <opsi-tls> --user <acl> --scan --pattern "ws:<workspace_id>:*" | head'
```

- **Verifikasi:** laporan purge = 0 untuk semua tabel tenant; prefix object storage kosong. **Rollback:** hanya sebelum hari ke-30 (Owner membatalkan); setelah purge tidak dipulihkan (PITR bukan mekanisme restore per tenant, usulan). **Audit:** `workspace.purged` + laporan.

### RB-28 — Grant & refund credit manual (FR-BILL-006, 04 §14)

- **Tujuan:** menambah credit atau mengoreksi tagihan. **Kapan:** invoice manual lunas (MVP 1), kompensasi insiden, refund otomatis gagal.
- **Prasyarat:** operator ops; dasar tercatat (nomor invoice/issue). Grant di atas `<N>` credit butuh persetujuan ops kedua (usulan).

1. Ops console → Workspaces → `<wsp_…>` → Credit grants (`POST /v1/ops/workspaces/{id}/credit-grants`): meter (`credit`/`ai_credit`), jumlah, kind (`TOPUP`/`ADJUSTMENT`), `expires_at` (BR-CRED-06: akhir siklus atau +12 bulan), alasan, `Idempotency-Key`.
2. Refund manual untuk operasi tertentu (refund otomatis gagal): belum ada endpoint khusus → grant `ADJUSTMENT` dengan rujukan `usage_event` (`use_…`) di alasan (Q-RB-11).
3. Ubah plan: `PUT /v1/ops/workspaces/{id}/plan`.
4. **Ledger append-only:** jangan pernah UPDATE/DELETE baris ledger (trigger menolak). Koreksi selalu dengan transaksi kompensasi.

```bash
curl -fsS -X POST "https://app.<domain>/api/v1/ops/workspaces/<wsp_id>/credit-grants" -H "Cookie: <sesi-ops>" \
  -H "Idempotency-Key: grant-<tiket>" -H "Content-Type: application/json" \
  -d '{"meter":"credit","amount":500,"kind":"ADJUSTMENT","expires_at":"<iso8601>","reason":"INC-<n> kompensasi"}'
```

- **Verifikasi:** saldo bertambah; txn berjumlah 0; rekonsiliasi harian berikutnya lulus. **Rollback:** grant keliru → transaksi kompensasi pengurangan (`ADJUSTMENT`), bukan menghapus baris. **Audit:** grant credit teraudit.

### RB-29 — Koreksi data oleh ops (FR-OPS-009, 04 §5.6)

- **Tujuan:** memperbaiki claim yang salah. **Kapan:** laporan user (FR-DATA-014) atau DSR koreksi di antrean FR-OPS-008. **Prasyarat:** operator ops console; bukti pendukung.

1. Buka `ops.data_issue_report` di ops console (Laporan data). Validasi bukti (sumber resmi, konfirmasi bisnis); jangan mengoreksi berdasarkan klaim yang belum terverifikasi.
2. Keputusan (`PATCH /v1/ops/data-reports`): koreksi (field, nilai baru, catatan bukti), merge/split (FR-OPS-005), atau tolak.
3. Sistem membuat evidence `OPS_CORRECTION` yang menang atas evidence dataset dengan `captured_at` ≤ waktu koreksi dan tetap menang saat snapshot yang sama di-re-run. Koreksi kalah hanya oleh observasi snapshot lebih baru.
4. Koreksi menaikkan `entity_seq` → skor terkait stale lalu di-recompute.
5. Bila laporan mengonfirmasi kontak INVALID → sistem memeriksa refund BR-CRED-05 (unlock ≤ 30 hari).

- **Verifikasi:** profil menampilkan nilai baru dengan evidence koreksi; skor ter-recompute.
- **Rollback:** koreksi baru (append) yang mengembalikan nilai; evidence tidak pernah dihapus. **Audit:** keputusan dan koreksi teraudit.

### RB-30 — Menambah `app-02` (R-08)

- **Tujuan:** menghilangkan SPOF `app-01`. **Kapan:** PRD R-08: wajib aktif sebelum M-3 (early paid); 06: sebelum paid GA (Q-RB-17).
- **Prasyarat:** keputusan VIP (keepalived/VRRP atau LB provider, Q-CI-05); server identik `app-01`; strategi ACME dua node (DNS-01 atau penyimpanan sertifikat bersama).

1. MR inventory: host `app-02` di grup `app`, variabel VIP, Redis replica + Sentinel, Traefik di dua node, nftables, target monitoring, akses WireGuard/SSH.
2. Tag `infra-*` → apply `site.yml --limit app-02` (`serial: 1`).
3. Deploy digest production saat ini ke `app-02` (`deploy.yml` memang `serial: 1`; `active_color` per host).
4. Aktifkan VIP; arahkan DNS `app.<domain>` dan `files.<domain>` ke VIP.
5. Uji failover di jendela maintenance: hentikan Traefik di `app-01` → VIP pindah → smoke → pulihkan. Uji failover Redis Sentinel.
6. Perbarui §2, RB-45 (RTO turun ke menit), dan tutup R-08.

- **Verifikasi:** kedua node melayani; worker River di dua host aman (leader election); probe eksternal hijau saat failover.
- **Rollback:** keluarkan `app-02` dari VIP/inventory. **Audit:** tag `infra-*` + issue.

---

## 5. Backup & disaster recovery

### 5.1 Target & skenario (NFR-AVAIL-02, 06 §11.6)

| Skenario | Prosedur | RTO target | RPO |
|---|---|---|---|
| `db-01` mati | RB-42 failover ke `db-02` | ±30 menit (≤ 4 jam) | ≈ detik (replikasi async) |
| Korupsi data logis | RB-41 PITR ke instance baru | 1–3 jam (≤ 4 jam) | Titik pilihan; data setelah titik hilang |
| `stor-01` rusak / objek terhapus | RB-44 | ≤ 4 jam untuk bucket prioritas | Lag `filer.sync` |
| `app-01` mati | RB-45 provision ulang (atau `app-02`) | 1–3 jam | 0 (stateless) |
| Kehilangan seluruh DC | RB-46 lokasi cadangan | **≤ 24 jam** (PRD NFR-AVAIL-02) | ≤ 15 menit (repo2 off-site) |

### 5.2 Jadwal backup & RB-40 — Verifikasi backup harian

| Data | Jadwal (WIB) | Tujuan | Retensi |
|---|---|---|---|
| PostgreSQL full | Minggu 01:00 | repo1 (`stor-01`, S3 gateway) + repo2 (off-site) | Full 4 minggu; PITR window 14 hari (Proposed) |
| PostgreSQL diff | Harian 01:00 | repo1 + repo2 | Mengikuti full |
| WAL | Kontinu, `archive_timeout = 60s`, archive-push async | repo1 + repo2 | PITR window |
| PostgreSQL incr tambahan | Sebelum migrasi `db-contract` (06 §9.5) | repo1 | Mengikuti |
| SeaweedFS | `weed filer.sync` kontinu (async) | Filer off-site (atau `weed filer.backup` ke S3 off-site) | Lifecycle per bucket |
| GitLab | Harian 02:00 (`gitlab-backup create` + `gitlab-secrets.json` + `gitlab.rb` + registry) | `stor-01` + off-site | 14 hari |
| Redis | Tidak di-backup | — | — |

**RB-40 (harian, on-call):** job terjadwal `backup-check` 07:00 menjalankan `pgbackrest info`/`check` kedua repo, umur backup terakhir, lag `filer.sync`, dan umur backup GitLab (06 §7). On-call membaca hasilnya setiap pagi; gagal → issue + AL-12/AL-13/AL-15. Verifikasi manual:

```bash
ssh db-01 'sudo -u postgres pgbackrest --stanza=msp info'             # status ok, backup terakhir, rentang WAL per repo
ssh db-01 'sudo -u postgres pgbackrest --stanza=msp check'            # uji archive-push WAL ke semua repo
ssh db-01 'sudo -u postgres psql -Atc "SELECT last_archived_time, failed_count, last_failed_time FROM pg_stat_archiver"'
```

**Verifikasi** (05 §3.25, TC-OPS-011): tidak ada celah WAL > 15 menit, backup terakhir sesuai jadwal, repo terenkripsi (`repo-cipher-type=aes-256-cbc`), salinan off-site ada. **Prasyarat:** SSH + `sudo -u postgres` di `db-01` (read-only). **Audit:** hasil job `backup-check`; kegagalan → issue. Nama stanza `msp` adalah usulan.

### RB-41 — Restore PITR (pgBackRest)

- **Tujuan:** memulihkan DB ke titik waktu sebelum korupsi/kehilangan data logis. **Kapan:** hanya bila forward fix tidak mungkin (06 §14.3). **Keputusan IC + founder**, karena data setelah titik restore hilang (termasuk ledger dan state job River).
- **Prasyarat:** Maintainer; host target baru (spare atau `dr-01` di zona prod, disk LUKS); passphrase repo.

1. Tentukan titik target `T` (UTC) dari `platform.audit_log`, log aplikasi (Loki), atau waktu deploy/migrasi. Catat di issue.
2. Containment: matikan writer berat (`pipeline.publish`, `scoring.bulk_rescore`, `export.create`, `webhook.delivery`, `upload.create`), hentikan worker terkait. Freeze write penuh (hentikan api) = keputusan IC (mode maintenance belum ada, Q-RB-10).
3. Preservasi: catat LSN/waktu primary saat ini. Bila perlu rekonsiliasi data setelah `T`, ambil dump tabel terkait dari primary lama (tetap di zona prod).
4. Provision host target (`site.yml --limit <host>` role `postgres`, versi PGDG sama).
5. Restore dari repo1 (lebih cepat) atau repo2 (bila repo1 tidak tersedia). Restore ±200 GB ±15–30 menit + replay WAL.
6. Verifikasi di host baru sebelum dialihkan: versi goose = production, policy RLS ada, hitungan baris tabel kunci wajar, data rusak tidak ada.
7. Alihkan aplikasi ke host baru dengan mekanisme yang sama seperti failover (RB-42 langkah 5). Fence primary lama.
8. Bangun ulang `db-02` sebagai standby dari primary baru; jalankan full backup (timeline baru).
9. Rekonsiliasi pasca-`T`: jalankan rekonsiliasi ledger (AL-20), periksa job River yang hilang (export/webhook/AI in-flight → user me-retry; hold konsisten pada `T`), objek S3 yang dibuat setelah `T` menjadi orphan (bersihkan kemudian), beri tahu customer yang aksinya hilang (§7.4).

```bash
sudo -u postgres pgbackrest --stanza=msp --repo=1 --delta --type=time \
  --target="2026-10-10 03:12:00+00" --target-action=promote restore
sudo systemctl start postgresql@16-main
sudo -u postgres psql msp -Atc "SELECT pg_is_in_recovery(), (SELECT max(version_id) FROM goose_db_version), (SELECT count(*) FROM pg_policies)"
```

- **Verifikasi:** query verifikasi lulus; smoke production; rekonsiliasi ledger = 0 selisih.
- **Abort:** host target tetap terisolasi sampai verifikasi lulus; primary lama tidak disentuh sampai pengalihan.
- **Audit:** keputusan IC + founder di issue; RTO/RPO aktual dicatat (template §9.2).

### RB-42 — Failover `db-01` → `db-02` (promote)

- **Tujuan:** memulihkan write saat primary gagal. Target ±30 menit. **Kapan:** `db-01` mati atau tidak dapat dipulihkan ≤ 15 menit (host down, disk rusak, crash loop). Postgres crash biasa → coba `systemctl restart postgresql@16-main` dulu.
- **Prasyarat:** SEV1 terdeklarasi; Maintainer; `db-02` sehat (lag terakhir diketahui).

1. Matikan writer berat (`pipeline.publish`, `scoring.bulk_rescore`). Write API sudah gagal.
2. **Fence `db-01`** agar tidak kembali sebagai primary kedua (split brain): matikan via IPMI, atau stop Postgres + blok 5432/6432 dengan nftables.
3. Catat posisi `db-02` (estimasi RPO): `pg_last_wal_replay_lsn()`, `pg_last_xact_replay_timestamp()`.
4. Jalankan `db-failover.yml` (pipeline atau laptop Maintainer via RB-01): promote `db-02`, aktifkan PgBouncer untuk primary baru (letak PgBouncer = Q-RB-04), ubah variabel primary di inventory, render ulang `DATABASE_URL_*` (APP, PIPELINE, OPS, DIRECT; REPLICA sementara → primary), rehydrate dan restart container `msp-*`, ubah `pg1-host` pgBackRest.
5. `pgbackrest check` dari primary baru, lalu full backup.
6. Kapasitas: tanpa replika, search/profil membaca primary. Pantau AL-02 dan beban `db-02`; tunda fan-out re-skor sampai standby baru ada.

```bash
ansible-playbook -i infra/ansible/inventories/production infra/ansible/playbooks/db-failover.yml -e new_primary=db-02 -e fence_host=db-01
# setara manual di db-02 (hanya bila playbook tidak tersedia):
sudo -u postgres psql -c "SELECT pg_promote(wait => true, wait_seconds => 60);"
sudo -u postgres pgbackrest --stanza=msp check && sudo -u postgres pgbackrest --stanza=msp --type=full backup
```

- **Verifikasi:** `SELECT pg_is_in_recovery()` = `f` di `db-02`; smoke write (workspace sintetis); worker River memproses; WAL archive berjalan; rekonsiliasi ledger manual lulus (transaksi yang belum terreplikasi hilang, job idempoten bisa berjalan ulang).
- **Abort:** bila `db-01` ternyata pulih sebelum promote, batalkan dan cabut fence.
- **Audit:** issue insiden; perubahan inventory lewat MR adopsi.

### RB-43 — Failback / switchover terencana

- **Tujuan:** mengembalikan redundansi (standby baru) dan, bila diinginkan, mengembalikan peran primary ke `db-01`. **Kapan:** setelah RB-42, atau switchover terencana (patch OS, upgrade). **Prasyarat:** Maintainer; jendela maintenance untuk switchover.
- **Catatan:** spesifikasi `db-01` dan `db-02` identik, jadi failback **opsional**. Lebih aman mempertahankan peran tertukar dan memperbarui inventory/dokumen, karena switchover selalu butuh jeda write.

1. Setelah `db-01` diperbaiki: bangun ulang sebagai standby dari primary saat ini (`pgbackrest --type=standby restore`, atau `pg_rewind` bila data dir utuh dan divergensi kecil). Usulan playbook `db-rebuild-standby.yml` (Q-RB-05).
2. Tunggu lag ≈ 0 dan stabil ≥ 24 jam.
3. Switchover (jendela maintenance, umumkan bila jeda write > 1 menit): (a) flag writer off dan stop worker (River soft stop); (b) pastikan `replay_lsn` standby = `pg_current_wal_lsn()` primary; (c) stop Postgres primary dengan bersih; (d) promote target dan jalankan `db-failover.yml -e new_primary=<target>`; (e) `pg_rewind` primary lama menjadi standby; (f) start worker, flag on, full backup.

```bash
# opsi A — bangun ulang dari backup sebagai standby
ssh db-01 'sudo -u postgres pgbackrest --stanza=msp --delta --type=standby restore && sudo systemctl start postgresql@16-main'
# opsi B — data dir utuh, divergensi kecil (Postgres di db-01 dalam keadaan stop)
ssh db-01 'sudo -u postgres /usr/lib/postgresql/16/bin/pg_rewind --target-pgdata=/var/lib/postgresql/16/main --source-server="host=db-02 user=<repl> dbname=postgres" --progress'
```

- **Verifikasi:** `pg_stat_replication` menunjukkan standby `streaming`; lag < 60 s; AL-11 hijau. **Abort:** sebelum promote, lag tidak nol → batalkan, start worker lagi di primary lama. **Audit:** issue maintenance.

### RB-44 — SeaweedFS: cek replikasi & restore

- **Tujuan:** memastikan salinan off-site sehat dan memulihkan objek/bucket. **Kapan:** cek mingguan, AL-15, objek terhapus, `stor-01` rusak. **Prasyarat:** SSH `stor-01`; kredensial S3 lokal & off-site (SOPS).
- **Cek mingguan:** lag `weed filer.sync` (log/metrik) ≤ 1 jam (usulan); bandingkan jumlah objek + checksum sampel per bucket lokal vs off-site; verifikasi sha256 manifest snapshot.

1. **Objek/prefix terhapus:** salin balik dari off-site ke bucket lokal dengan klien S3, lalu verifikasi checksum.
2. **Volume rusak:** `volume.fsck` untuk menemukan objek hilang; single node tanpa replika lokal → pulihkan objek dari off-site.
3. **`stor-01` hilang:** matikan `export.create` dan `upload.create`; WAL tetap ke repo2 (repo1 gagal → AL-12). Provision `stor-01` baru (`site.yml --limit stor-01`), sinkron balik dari off-site dengan prioritas: `msp-uploads` → `msp-tiles` → bucket audit → `msp-snapshots` → `msp-evidence`. `msp-exports` tidak dipulihkan (lifecycle 24 jam; user export ulang). Repo1 pgBackRest dibuat ulang dengan full backup baru.

```bash
weed shell -master=stor-01:9333          # lalu: lock · cluster.ps · volume.list · volume.fsck · fs.du /buckets · s3.bucket.list · unlock
aws --endpoint-url https://<offsite-s3> s3 sync s3://msp-uploads/<prefix>/ ./restore/<prefix>/
aws --endpoint-url https://<s3-internal>:8333 s3 sync ./restore/<prefix>/ s3://msp-uploads/<prefix>/
```

- **Verifikasi:** checksum sampel cocok (05 §3.25); bila object lock/WORM didukung, objek backup tidak dapat dihapus sebelum masa retensi; bila tidak, fallback append-only off-site terverifikasi. **Abort:** jangan menimpa objek yang masih ada; restore ke direktori sementara dulu bila ragu. **Audit:** issue + laporan di `msp-evidence`.

### RB-45 — Kehilangan `app-01`

- **Tujuan:** memulihkan aplikasi saat host `app-01` rusak (R-08). Target 1–3 jam; `app-02` (RB-30) menurunkannya ke menit. **Kapan:** AL-03 dan `app-01` tidak pulih ≤ 30 menit. **Prasyarat:** Maintainer, server cadangan, IPMI, digest rilis terakhir.

1. Konfirmasi lewat IPMI/NOC bahwa host tidak bisa dipulihkan cepat (≤ 30 menit).
2. Siapkan server cadangan (spesifikasi sama). Gunakan hostname dan IP privat yang sama bila memungkinkan.
3. `site.yml --limit app-01` (base, firewall, docker, traefik, msp_app, monitoring_agent) dari pipeline tag `infra-*` atau laptop Maintainer.
4. Deploy digest production terakhir (lihat deployment GitLab environment `production` atau tag registry `vX.Y.Z`) dengan `deploy.yml`.
5. IP publik berubah → perbarui DNS `app.`/`files.` (TTL rendah, usulan 300 s); Traefik menerbitkan sertifikat baru (waspadai rate limit Let's Encrypt).
6. Verifikasi Smokescreen + nftables egress (05 smoke egress) sebelum `webhook.delivery` dinyalakan.

- **Verifikasi:** probe eksternal hijau; smoke; antrean River mulai turun. **Abort:** `app-01` asli pulih lebih dulu → pakai kembali, server cadangan jadi standby. **Audit:** issue insiden.

### RB-46 — Kehilangan seluruh DC (RTO ≤ 24 jam)

- **Tujuan:** menjalankan layanan di lokasi cadangan di Indonesia dari backup off-site (NFR-AVAIL-02, NFR-PRIV-03). **Kapan:** DC tidak tersedia dan estimasi pulih dari provider > 4 jam, atau infrastruktur hancur. Keputusan aktivasi: IC + founder.
- **Prasyarat (disiapkan sebelumnya):** kontrak lokasi cadangan (Q-09/Q-CI-04); inventory `dr` (Q-RB-05); salinan offline runbook + clone repo `infra/` di laptop Maintainer (GitLab ikut hilang); salinan offline kunci age dan passphrase; image rilis dapat diambil dari backup registry off-site (Q-RB-13); TTL DNS rendah.

| Fase | Langkah | Target kumulatif |
|---|---|---|
| 1. Deklarasi | SEV1, IC, founder; status ke customer (§7.4); minta estimasi provider | T+1 jam |
| 2. Provision | Server minimal di lokasi cadangan: app, db, stor (+ ops ringan untuk WireGuard/Tang/monitoring); LUKS; WireGuard baru | T+8 jam |
| 3. Database | `pgbackrest --repo=2 restore` (PITR ke titik terakhir di repo2), verifikasi seperti RB-41 langkah 6 | T+12 jam |
| 4. Object storage | Promosikan filer off-site atau sinkron bucket prioritas (urutan RB-44) | T+14 jam |
| 5. Aplikasi | `site.yml` + `deploy.yml` dari laptop Maintainer dengan image dari registry yang dipulihkan | T+16 jam |
| 6. Jaringan | DNS `app.`/`files.` → IP baru; sertifikat ACME; probe eksternal ke IP baru | T+18 jam |
| 7. Verifikasi & buka | Smoke, isolasi tenant (sampel), egress, rekonsiliasi ledger; nyalakan flag bertahap | T+20–24 jam |
| 8. Sesudahnya | Pulihkan GitLab dari backup off-site; observability; rencana kembali ke DC utama (switchover terencana) | — |

- **Verifikasi:** RPO aktual (transaksi terakhir di `usage.usage_event`/`platform.audit_log`) ≤ 15 menit; layanan inti berjalan. **Kembali/abort:** DC utama pulih sebelum fase 5 → batalkan aktivasi; setelah layanan pindah, kembali hanya lewat switchover terencana (fase 8). **Audit:** timeline lengkap di issue (dibuat di GitLab pulihan atau dokumen sementara, lalu dipindah).

### RB-47 — Drill restore bulanan di `dr-01` (06 §7.1, QG-35)

- **Tujuan:** membuktikan RPO/RTO (NFR-AVAIL-02) dan jalur restore off-site. **Prasyarat:** `dr-01` tersedia, runner `deploy`, passphrase repo2.
- **Jadwal (kapan):** Selasa pertama tiap bulan 09:00 WIB, schedule `monthly` via runner `deploy` (`restore-drill.yml`). Gagal → SEV3 dan QG-35 memblokir rilis minor/major berikutnya. Restore **tidak pernah** ke `stg-01`. 05 §3.25 menyebut kuartalan; runbook mengikuti 06 (Q-RB-18).

1. Siapkan `dr-01` (zona prod, disk LUKS). Catat `T_target` = sekarang − 5 menit.
2. `pgbackrest restore --type=time --target=<T_target>` dari **repo2 (off-site)**, sekaligus menguji jalur DR lokasi lain.
3. **RTO** = mulai restore → PostgreSQL menerima koneksi dan query verifikasi lulus (target ≤ 4 jam).
4. **RPO** = selisih `T_target` dengan transaksi terakhir yang ada (`max(created_at)` di `platform.audit_log` dan `usage.usage_event`), target ≤ 15 menit.
5. Integritas: jumlah baris tabel kunci vs production (toleransi jeda), versi goose = production, policy RLS ada, sampel objek off-site terbaca. Restore backup GitLab diuji tiap kuartal.
6. Hapus data dengan aman (hancurkan volume LUKS). Simpan laporan berisi **metrik saja** di `msp-evidence` (template §9.2).

- **Verifikasi:** RTO ≤ 4 jam dan RPO ≤ 15 menit tercatat. **Abort:** gagal di tengah → tetap hapus data `dr-01`, buka issue SEV3. **Audit:** laporan drill + log job `monthly`.

---

## 6. Alert playbook

### 6.1 Routing & aturan umum

- Setiap alert membawa label `severity` (`page` | `ticket`) dan `sev` (`SEV1`..`SEV4`). `page` → pager + kanal chat + email; `ticket` → kanal chat + issue GitLab (06 §16.1).
- Anotasi wajib: `summary`, `runbook_url` (anchor AL-nn di dokumen ini), `dashboard` (TC-OPS-007 mensyaratkan tautan dashboard + runbook).
- Inhibit: AL-03 (aplikasi down) menahan AL-01/02/05; `db-01` down menahan AL-11/12.
- Maintenance terencana: silence dengan komentar nomor RB/issue dan masa berlaku (`amtool silence add … --duration`). Silence tanpa masa berlaku dilarang.
- **Watchdog** (selalu firing) dikirim ke probe eksternal sebagai dead man's switch. Hilangnya watchdog = Alertmanager/`ops-01` mati (AL-03).
- Ambang bertanda *(usulan)* belum ada di 04 §16.4/06 §6.3 dan perlu disetujui (Q-RB-09). Nama metrik exporter diverifikasi saat bootstrap; metrik `msp_*` baru diusulkan di Q-RB-09.
- Alert RG-5 (AL-04, AL-05, AL-08, AL-10) dilatih lewat `alert-drill` di staging sebelum MVP 1 dan tiap kuartal (TC-OPS-007).

```yaml
# ops-01: /etc/prometheus/rules/msp.yml (ILUSTRATIF)
groups:
- name: msp
  rules:
  - alert: MSPPipelineStageFailureHigh          # AL-04 (RG-5)
    expr: max by (stage) (msp_stage_failure_ratio) > 0.05
    for: 15m
    labels: { severity: ticket, sev: SEV3 }
    annotations: { runbook_url: "<gitlab>/msp/MesinProspek/-/blob/main/docs/09-runbook-operasional.md#<al-04>" }
  - alert: MSPQueueBacklogOld                   # AL-05 (RG-5)
    expr: max by (queue) (msp_queue_oldest_age_seconds) > 3600
    for: 5m
    labels: { severity: page, sev: SEV2 }
  - alert: MSPApiErrorBudgetFastBurn            # AL-01
    expr: |
      (sum(rate(traefik_service_requests_total{service=~"msp-api.*",code=~"5.."}[1h])) / sum(rate(traefik_service_requests_total{service=~"msp-api.*"}[1h])) > 14.4*0.005)
      and (sum(rate(traefik_service_requests_total{service=~"msp-api.*",code=~"5.."}[5m])) / sum(rate(traefik_service_requests_total{service=~"msp-api.*"}[5m])) > 14.4*0.005)
    labels: { severity: page, sev: SEV2 }
```

### 6.2 Playbook per alert

#### AL-01 — Error budget API terbakar (5xx)

- **Alert & ambang:** `MSPApiErrorBudgetFastBurn` (1 j) / `MSPApiErrorBudgetSlowBurn` (6 j). Burn rate error budget 99,5% bulanan **2%/jam**, jendela 1 j / 6 j (04 §16.4). Contoh: rasio 5xx router `msp-api` > 7,2% (14,4 × 0,5%) pada 1 j **dan** 5 m → page SEV2; > 3% pada 6 j → ticket (interpretasi jendela 6 j = Q-RB-09).
- **Dampak:** request customer gagal; budget ±3,6 jam/bulan tergerus; rilis non-fix dibekukan bila terpakai > 75%.
- **Diagnosis cepat:** deploy atau perubahan flag baru-baru ini (anotasi Grafana, audit flag)? Route dan `error.code` dominan (dashboard API & SLO)? Issue baru di GlitchTip? Pool PgBouncer penuh, `statement_timeout`, kesehatan `db-01`/`db-02`? Redis (AL-16)?
- **Mitigasi:** terkait rilis → RB-11; kapabilitas tertentu → RB-19; DB → AL-11/RB-42; lonjakan traffic → identifikasi sumber, rate limit, tambah replika api sementara.
- **Eskalasi:** 5xx > 50% atau berlangsung > 30 menit → SEV1 (IC + founder). **RB:** RB-11, RB-19, RB-42.

#### AL-02 — Pelanggaran p95 latency (NFR-PERF-01/02)

- **Alert & ambang:** `MSPSearchLatencyP95` p95 `GET /v1/prospects` > 800 ms; `MSPProfileLatencyP95` p95 `GET /v1/prospects/{id}` > 500 ms; pelanggaran > 15 menit (04 §16.4). Ticket SEV3.
- **Dampak:** search/profil lambat; risiko time-to-first-result (RG-1).
- **Diagnosis cepat:** beban/lag `db-02` (search & profil membaca replika); fan-out re-skor atau pipeline `project` (swap projection) sedang berjalan; autovacuum pada `intel.score`; query lambat (`pg_stat_statements`, bila aktif); Redis down (cache miss); rilis baru (bandingkan per `msp.color`).
- **Mitigasi:** `scoring.bulk_rescore` off; tunda publish snapshot (`pipeline.publish`); rollback bila regresi rilis (RB-11); index/forward fix (RB-13).
- **Eskalasi:** > 1 jam atau p99 search > 2 s → SEV2. **RB:** RB-19, RB-11, RB-13.

#### AL-03 — Probe ketersediaan down / watchdog hilang

- **Alert & ambang:** probe eksternal (di luar DC) dan blackbox exporter: `probe_success == 0` untuk `https://app.<domain>` selama 3 menit *(usulan)*; watchdog Alertmanager tidak diterima probe eksternal > 10 menit *(usulan)*. Page.
- **Dampak:** outage total (SEV1) atau monitoring buta (SEV2).
- **Diagnosis cepat:** dari luar DC: `curl -sv https://app.<domain>/`. Semua host tidak terjangkau → masalah DC/jaringan → NOC provider (RB-46 bila berkepanjangan). Hanya `app-01` → IPMI/konsol (RB-45). Traefik mati/restart loop → `docker logs traefik`. Sertifikat kedaluwarsa → AL-17. DNS → provider DNS. Watchdog hilang tetapi aplikasi hidup → `ops-01`/Alertmanager mati: pulihkan `ops-01` (deploy berhenti, rollback tetap bisa via RB-01).
- **Mitigasi:** restart Traefik/container; failover host; komunikasi customer ≤ 4 jam (SEV1).
- **Eskalasi:** SEV1 langsung ke IC + founder ≤ 15 menit. **RB:** RB-45, RB-46, RB-21.

#### AL-04 — Pipeline failure > 5% (RG-5)

- **Alert & ambang:** `MSPPipelineStageFailureHigh` — `msp_stage_failure_ratio > 0,05` per stage, 15 m (04 §16.4). Ticket SEV3.
- **Dampak:** run snapshot lambat/gagal; publish tertunda. Customer tidak langsung terdampak karena publish hanya terjadi setelah run sukses.
- **Diagnosis cepat:** stage mana dan `error_counts` di `platform.stage_run`; log `{service="worker-pipeline"} | json | stage="<stage>"`; jenis error: data (mapping, parse) vs infra (DB timeout, DNS/Unbound untuk `verify`) vs bug (rilis baru).
- **Mitigasi:** tahan `pipeline.publish`; masalah DNS → `dns.lookup` off (hasil UNKNOWN, bukan INVALID); perbaiki lalu re-run stage (idempoten, RB-15); data buruk → tolak snapshot dan siapkan snapshot baru (RB-17).
- **Eskalasi:** menghalangi publish yang dijanjikan ke customer → SEV2. **RB:** RB-15, RB-17, RB-19.

#### AL-05 — Backlog antrean > 1 jam (RG-5)

- **Alert & ambang:** `MSPQueueBacklogOld` — `msp_queue_oldest_age_seconds > 3600` di antrean mana pun, 5 m (04 §16.4). Page SEV2 untuk `export`, `webhook`, `scoring`, `ai`; ticket SEV3 untuk `pipeline`, `maintenance`.
- **Dampak:** `scoring` → skor stale, ICP lambat (NFR-PERF-05); `export` → NFR-PERF-04 dilanggar; `webhook` → keterlambatan (jendela retry 24 jam berjalan); `ai` → brief timeout; `maintenance` → email, refund, purge tertunda.
- **Diagnosis cepat:** antrean mana; container worker hidup (`docker compose -p msp-<warna> ps`); durasi job; beban DB; downstream lambat.
- **Mitigasi:** RB-14.
- **Eskalasi:** backlog bertambah > 2 jam meski sudah mitigasi → IC. **RB:** RB-14, RB-15.

#### AL-06 — Lonjakan job `discarded` (DLQ)

- **Alert & ambang:** `MSPJobsDiscardedSpike` — `msp_dlq_size` naik > 20 dalam 15 menit per antrean *(usulan; sinyal canary 06 §6.3 memakai > 20 / 5 menit saat deploy)*. Ticket SEV3.
- **Dampak:** pekerjaan gagal permanen (export, webhook, pipeline, purge, refund).
- **Diagnosis cepat:** RB-15 langkah 1 (kelompokkan per `kind` + error terakhir); korelasikan dengan deploy atau gangguan dependensi.
- **Mitigasi:** perbaiki penyebab → redrive (RB-15); terkait rilis → RB-11.
- **Eskalasi:** `kind` purge/refund/suppression → beri tahu DPO/ops. **RB:** RB-15, RB-16.

#### AL-07 — Job skor ICP > 10 menit (NFR-PERF-05)

- **Alert & ambang:** `MSPScoringJobSlow` — durasi job skor > 10 menit (per job, 04 §16.4). Ticket SEV3.
- **Dampak:** user melihat "Menghitung skor" terlalu lama; skor stale.
- **Diagnosis cepat:** jumlah kandidat (> 200.000?); `worker-scoring` jenuh oleh prioritas 4 (fan-out, maks 4 bersamaan); kontensi DB saat `COPY` epoch; autovacuum.
- **Mitigasi:** `scoring.bulk_rescore` off agar prioritas 1 didahulukan; RB-14.
- **Eskalasi:** berulang di banyak workspace → SEV2. **RB:** RB-14, RB-19.

#### AL-08 — AI failure > 5% (RG-5)

- **Alert & ambang:** `MSPAIFailureHigh` — brief gagal/total > 0,05 dengan minimal 20 request, 15 m (04 §16.4). Ticket SEV3.
- **Dampak:** brief AI gagal (hold ai_credit di-release, tidak ada tagihan).
- **Diagnosis cepat:** `error_code` di `ai.ai_call`: 429/5xx provider (status page provider), timeout (30 s/percobaan), deny Smokescreen (allowlist host berubah), `AI_GROUNDING_FAILED` (regresi prompt/model setelah rollout), abort redaksi (AL-24).
- **Mitigasi:** provider down → `ai.generation` off (pesan ke user); regresi prompt → flag `rel.ai.brief-<versi>` off (RB-18); egress → perbaiki ACL Smokescreen lewat MR.
- **Eskalasi:** > 25% selama > 1 jam → SEV2; vendor support provider. **RB:** RB-19, RB-18.

#### AL-09 — Biaya AI: budget harian habis / cap workspace

- **Alert & ambang:** `MSPAIDailyBudgetExhausted` — budget harian global (`AI_DAILY_BUDGET_USD`) tercapai → kill switch otomatis `ai.generation` + alert (04 §13.7). Peringatan ≥ 80% budget *(usulan)*. Cap bulanan per workspace tercapai → `FEATURE_DISABLED` untuk workspace itu (info, bukan page).
- **Dampak:** AI nonaktif untuk semua workspace sampai dinyalakan ulang.
- **Diagnosis cepat:** `msp_ai_cost_usd_total` per `prompt_version`/`model`; workspace dengan biaya tertinggi (dashboard Cost); cache hit rate turun (input hash berubah); loop regenerate; tabel harga (`AI_PRICE_TABLE`) salah.
- **Mitigasi:** penyalahgunaan satu workspace → flag `ai.generation` scope WORKSPACE off; nyalakan global lagi setelah penyebab jelas; menaikkan budget = perubahan env + deploy (persetujuan founder). Waktu reset budget harian = Q-RB-09.
- **Eskalasi:** founder untuk keputusan budget. **RB:** RB-19.

#### AL-10 — Webhook failure > 5% (RG-5)

- **Alert & ambang:** `MSPWebhookFailureHigh` — attempt gagal/total > 0,05 dengan minimal 50 attempt, agregat semua endpoint, 1 j (04 §16.4). Ticket SEV3.
- **Dampak:** event terlambat; setelah ±20 jam 36 menit delivery masuk DLQ dan hold di-release.
- **Diagnosis cepat:** satu endpoint dominan (sisi customer) vs merata (sisi MSP); distribusi status HTTP; deny Smokescreen; Unbound; container `smokescreen` dan `worker-webhook`.
- **Mitigasi:** RB-16.
- **Eskalasi:** sisi MSP → SEV2. **RB:** RB-16, RB-19.

#### AL-11 — Lag replikasi DB > 60 s

- **Alert & ambang:** `PGReplicationLagHigh` — lag `db-02` > 60 s, 5 m (04 §16.4). Ticket SEV3; page SEV2 bila > 15 menit atau `db-02` down.
- **Dampak:** search/profil membaca data basi (mis. kontak tampak masih ter-mask sesaat setelah unlock); RPO failover membesar; WAL tertahan di `db-01` (disk).
- **Diagnosis cepat:** `pg_stat_replication` di `db-01` (§9.1); `db-02` hidup? IO tinggi karena `COPY` epoch skor/swap projection; jaringan VLAN.
- **Mitigasi:** tahan penulis berat (`scoring.bulk_rescore`, `pipeline.publish`); `db-02` down lama → bangun ulang standby (RB-43 langkah 1) dan pantau slot/disk `db-01` (RB-24); lag > 15 menit dan `db-01` longgar → arahkan `DATABASE_URL_REPLICA` ke primary (MR inventory + deploy digest sama).
- **Eskalasi:** Maintainer. **RB:** RB-43, RB-24.

#### AL-12 — WAL archive gagal

- **Alert & ambang:** `PGWALArchiveFailing` — `pg_stat_archiver.failed_count` naik atau `last_archived_time` > 5 menit lalu, 5 m (04 §16.4). Page SEV2.
- **Dampak:** RPO PITR membesar; `pg_wal` menumpuk dan bisa menghentikan `db-01` bila disk penuh.
- **Diagnosis cepat:** `/var/log/pgbackrest/*archive-push*`; `pgbackrest check`; repo1 (`stor-01:8333`) dan repo2 (off-site) terjangkau? kredensial S3 baru dirotasi? TLS? kapasitas repo?
- **Mitigasi:** pulihkan konektivitas/kredensial repo. **Jangan** menghapus file `pg_wal`. Bila satu repo mati lama dan `pg_wal` > 70% disk: Maintainer memutuskan perubahan darurat konfigurasi pgBackRest (archive ke repo sehat) via MR, lalu full backup ke repo yang pulih.
- **Eskalasi:** > 1 jam → libatkan IC (06 §14.5: replikasi/backup gagal > 1 jam = SEV2). **RB:** RB-40, RB-24.

#### AL-13 — Backup gagal / terlewat

- **Alert & ambang:** `PGBackupMissing` — backup terakhir > 26 jam, 5 m (04 §16.4), atau job `backup-check` gagal. Ticket SEV3; SEV2 bila > 48 jam atau kedua repo.
- **Dampak:** titik restore menua; QG-35 dan target RTO terancam.
- **Diagnosis cepat:** `pgbackrest info` (status per repo); log `/var/log/pgbackrest/`; timer/cron backup di `db-01`; lock backup lain; ruang repo.
- **Mitigasi:** jalankan backup manual di jam sepi (`sudo -u postgres pgbackrest --stanza=msp --type=diff backup`), lalu cek repo2.
- **Eskalasi:** Maintainer. **RB:** RB-40.

#### AL-14 — Disk > 80%

- **Alert & ambang:** `HostDiskUsageHigh` — filesystem > 80%, 5 m (04 §16.4). Ticket SEV3; page SEV2 bila > 90% di `db-01`/`stor-01` atau `predict_linear` habis < 24 jam *(usulan)*.
- **Dampak:** DB berhenti menerima write; upload/export/backup gagal.
- **Diagnosis & mitigasi:** RB-24.
- **Eskalasi:** Maintainer; NOC provider untuk tambah disk. **RB:** RB-24.

#### AL-15 — Kapasitas SeaweedFS / sync off-site

- **Alert & ambang:** `SeaweedFSCapacityLow` — disk data `stor-01` > 80% atau slot volume bebas < 10% *(usulan)*; `SeaweedFSOffsiteSyncLag` — lag `filer.sync` > 1 jam *(usulan)*. Ticket SEV3.
- **Dampak:** upload/export gagal tulis; repo1 pgBackRest gagal; salinan off-site tertinggal (RPO objek).
- **Diagnosis cepat:** `weed shell`: `volume.list`, `fs.du /buckets`; log `filer.sync`; konektivitas off-site.
- **Mitigasi:** `volume.vacuum`; cek lifecycle `msp-exports`; tambah volume/disk; restart `filer.sync` dari checkpoint.
- **Eskalasi:** > 90% → SEV2. **RB:** RB-44, RB-24.

#### AL-16 — Redis down (degraded mode)

- **Alert & ambang:** `RedisDown` — `redis_up == 0` selama 2 menit *(usulan)*. Ticket SEV3.
- **Dampak:** mode degradasi. Cache sesi (TTL 60 s), profil, dan facet miss → beban DB naik. Antrean River **tidak** terpengaruh (Postgres). Perilaku rate limit dan counter fair-use saat Redis down belum didefinisikan (Q-RB-08).
- **Diagnosis cepat:** `docker compose -p msp-core ps redis`, log OOM (`maxmemory` 8 GB, eviction), sertifikat TLS, password ACL setelah rotasi.
- **Mitigasi:** `docker compose -p msp-core restart redis` (tidak ada data yang perlu dipulihkan); pantau AL-01/AL-02.
- **Eskalasi:** menyebabkan 5xx → SEV2. **RB:** RB-20 (bila ACL).

#### AL-17 — Sertifikat TLS segera kedaluwarsa

- **Alert & ambang:** `TLSCertExpiringSoon` — `probe_ssl_earliest_cert_expiry - time()` < 21 hari (ticket SEV3) dan < 7 hari (page SEV2) *(usulan)*, untuk sertifikat publik dan internal (PostgreSQL, Redis, SeaweedFS, GitLab).
- **Dampak:** kedaluwarsa = outage (publik) atau kegagalan koneksi internal `verify-full`.
- **Diagnosis & mitigasi:** RB-21. ACME seharusnya renew ±30 hari sebelumnya; alert < 21 hari berarti renewal gagal.
- **Eskalasi:** Maintainer. **RB:** RB-21.

#### AL-18 — Lonjakan blok egress proxy (dugaan SSRF)

- **Alert & ambang:** `EgressProxyDenySpike` — deny Smokescreen role `webhook` > 10 dalam 15 menit atau > 5× baseline, dan deny **apa pun** untuk role `ai`/`mail` *(usulan)*. Ticket SEV3.
- **Dampak:** indikasi percobaan SSRF lewat URL webhook (IP privat, metadata, rebinding), atau salah konfigurasi allowlist.
- **Diagnosis cepat:** `{service="smokescreen"} | json | decision="deny"` dikelompokkan per role, kelas IP tujuan, `endpoint_host_hash`; cocokkan dengan endpoint webhook dan workspace pemiliknya; cek audit perubahan konfigurasi webhook (rejected oleh netguard saat pendaftaran?); counter drop nftables (§9.1).
- **Mitigasi:** pola dari satu workspace → `webhook.delivery` scope WORKSPACE off + tinjau akun; salah allowlist `ai`/`mail` → MR ACL.
- **Eskalasi:** berulang dari satu workspace → SEV2 (keamanan). **Ada bukti request mencapai jaringan privat/metadata → SEV1 + RB-50.** **RB:** RB-19, RB-50.

#### AL-19 — Lonjakan kegagalan autentikasi / brute force

- **Alert & ambang:** `AuthFailureSpike` — login gagal (Kratos) > 50 per 5 menit global atau > 10 per 5 menit per akun/IP ter-hash; kegagalan MFA akun ops > 5 per 15 menit *(usulan)*. Ticket SEV3.
- **Dampak:** credential stuffing / brute force; risiko pengambilalihan akun.
- **Diagnosis cepat:** log Kratos dan Traefik (IP dan email teredaksi sebagai HMAC, tetap dapat dikorelasikan); akun target; apakah ada login sukses setelah rentetan gagal; akun ops?
- **Mitigasi:** andalkan rate limit Kratos; blokir sumber via middleware Traefik (MR darurat); paksa logout semua sesi akun terdampak; minta reset password.
- **Eskalasi:** akun ops menjadi target → SEV2 (pertimbangkan membatasi ops console ke WireGuard). Dugaan pengambilalihan berhasil → SEV1 + RB-50. **RB:** RB-50.

#### AL-20 — Ketidakcocokan rekonsiliasi ledger (RG-4)

- **Alert & ambang:** `LedgerReconciliationMismatch` — selisih ≠ 0 pada rekonsiliasi harian (River periodic job): (a) `credit_balance` ≠ Σ ledger, (b) Σ `consumed` ≠ Σ `usage_event` CAPTURED − REFUNDED, (c) txn tidak berjumlah 0 (04 §14.2). Page SEV2 (kritis).
- **Dampak:** saldo/tagihan salah; potensi tagihan ganda.
- **Diagnosis cepat:** laporan rekonsiliasi per workspace dan cek mana yang gagal (alat bantu = Q-RB-06); korelasi dengan deploy modul `usage`; hold `HELD` tanpa capture/release (job export/webhook hilang); restore/failover terbaru (RB-41/42).
- **Mitigasi:** pola sistemik → matikan kapabilitas berbayar terkait (`export.create`, `webhook.delivery`, `ai.generation`; unlock tidak punya kill switch, Q-RB-08) atau rollback (RB-11). Koreksi hanya dengan transaksi kompensasi (RB-28), tidak pernah UPDATE ledger. Beri tahu workspace terdampak.
- **Eskalasi:** tagihan ganda massal → SEV1 (06 §14.5). **RB:** RB-28, RB-19, RB-11.

#### AL-21 — Pelanggaran SLA propagasi suppression

- **Alert & ambang:** `SuppressionPropagationBreach` — `suppression_leak_count > 0` atau `propagated_at − created_at` > 24 jam, 1 j (04 §15.7, §16.4). Page SEV2.
- **Dampak:** data pribadi yang sudah ditolak subjek masih tampil/terkirim (NFR-PRIV-02).
- **Diagnosis cepat:** laporan sweep per komponen (projection, file export aktif, delivery pending, AI artifact); job gagal di DLQ antrean `maintenance`/`export`; regenerasi file export gagal.
- **Mitigasi:** perbaiki dan redrive (RB-15); cabut link export lama secara manual bila regenerasi macet; ulangi verifikasi RB-25 langkah 4.
- **Eskalasi:** DPO dilibatkan. Kebocoran > 24 jam yang terkonfirmasi terlihat oleh tenant → label `pdp-suspected`, DPO menilai (RB-50). **RB:** RB-25, RB-15, RB-50.

#### AL-22 — Anomali isolasi tenant (SEV1)

- **Alert & ambang:** `MSPTenantIsolationAnomaly` — nilai > 0 dari salah satu sinyal *(usulan)*: (a) `msp_tenant_guard_violations_total` (cek aplikasi: baris hasil resolve milik workspace lain, prefix cache `ws:{id}:` atau prefix object storage tidak cocok); (b) error PostgreSQL SQLSTATE `42501` "new row violates row-level security policy" dari role `msp_app`; (c) laporan customer/peneliti. **Page SEV1** (kebocoran lintas tenant, 06 §14.5).
- **Dampak:** potensi kebocoran data lintas tenant (FR-AUD-006, NFR-SEC-03).
- **Diagnosis cepat:** ambil `correlation_id` → Loki/Tempo: route, workspace, user, versi/warna; apakah data benar-benar dikembalikan (status, ukuran respons) atau hanya ditolak; GlitchTip; deploy terbaru.
- **Mitigasi (containment dulu):** matikan kapabilitas/route terkait (RB-19) atau rollback (RB-11); cabut sesi user terkait bila ada indikasi sengaja; **preservasi bukti** (jangan hapus log, perpanjang retensi); label `pdp-suspected` → RB-50 (T0 = saat diketahui).
- **Eskalasi:** IC + founder ≤ 15 menit, DPO segera. **RB:** RB-50, RB-11, RB-19.

#### AL-23 — Hash chain audit rusak / ekspor audit gagal

- **Alert & ambang:** `AuditHashChainBroken` — verifikasi harian hash chain `platform.audit_log` gagal, atau ekspor harian ke bucket audit gagal (04 §15.5) *(usulan)*. Page SEV2.
- **Dampak:** bukti tamper-evidence hilang; kemungkinan manipulasi oleh insider.
- **Diagnosis cepat:** partisi/entri pertama yang gagal; bandingkan dengan salinan ekspor harian di bucket audit; aktivitas superuser DB (auditd, log PostgreSQL), akses break-glass.
- **Mitigasi:** preservasi (snapshot partisi, salinan ekspor); bekukan akses superuser non-esensial.
- **Eskalasi:** manipulasi terkonfirmasi → SEV1 (keamanan). **RB:** RB-01 (review log), RB-50 bila menyangkut data pribadi.

#### AL-24 — Pelanggaran redaksi prompt AI

- **Alert & ambang:** `AIPromptRedactionViolation` — pemindai sebelum panggilan LLM menemukan pola email/telepon/nama person → panggilan di-abort + alert (04 §13.3). Page SEV2.
- **Dampak:** bug input builder yang mengancam NFR-PRIV-04/FR-AI-005 (data pribadi ke luar negeri).
- **Diagnosis cepat:** `prompt_version`, `kind`, jalur input builder, rilis terbaru; pastikan **tidak ada** `ai.ai_call` untuk artifact yang di-abort.
- **Mitigasi:** berulang → `ai.generation` off; fix + test redaksi; rollback versi prompt (RB-18).
- **Eskalasi:** bukti ada data pribadi yang terkirim ke provider → SEV1 + RB-50. **RB:** RB-19, RB-18, RB-50.

---

## 7. Manajemen insiden

### 7.1 Severity (06 §14.5, dengan contoh produk)

| SEV | Contoh MSP | Respons | Komunikasi |
|---|---|---|---|
| **SEV1** | `app.<domain>` down > 5 menit (AL-03); `db-01` mati dan write gagal; kebocoran lintas tenant (AL-22); dugaan kegagalan PDP (`pdp-suspected`): kontak person/upload customer terekspos, data pribadi terkirim ke LLM (AL-24); tagihan ganda massal (AL-20); DC/jaringan down; kunci production (age, cosign, SSH deploy) bocor | Segera, 24/7, IC ditunjuk | Kanal insiden + founder ≤ 15 menit; customer terdampak ≤ 4 jam (email); update tiap 1 jam |
| **SEV2** | Search/export/webhook/unlock gagal untuk banyak workspace; fast burn error budget (AL-01); backlog export/scoring/webhook > 1 jam (AL-05); WAL archive atau backup gagal > 1 jam (AL-12/13); suppression > 24 jam (AL-21); rekonsiliasi ledger ≠ 0 (AL-20); hash chain audit rusak (AL-23) | ≤ 30 menit jam kerja, ≤ 1 jam di luar | Kanal insiden; customer bila dampak > 1 jam |
| **SEV3** | AI failure > 5% (AL-08); pipeline failure (AL-04); satu endpoint webhook gagal; latency breach singkat; sertifikat < 21 hari; drill restore gagal | Hari kerja yang sama | Issue |
| **SEV4** | Kosmetik, dashboard rusak, alert noise, CI flaky | Backlog | — |

### 7.2 Alur insiden

1. **Deteksi** (alert, probe, laporan customer) → **ack** oleh on-call.
2. **Deklarasi:** tetapkan SEV, tunjuk IC, buat kanal `#inc-<yyyymmdd>-<slug>` dan issue `incident` (checklist §9.2). IC mengumumkan serah terima peran secara eksplisit ("IC sekarang: X").
3. **Containment dulu:** kill switch (RB-19), rollback (RB-11), failover (RB-42), blokir akses/cabut sesi.
4. **Diagnosis → pemulihan → verifikasi** (metrik normal ≥ 30 menit).
5. **Resolved:** komunikasi selesai (§7.4); PIR wajib untuk SEV1/SEV2 ≤ 5 hari kerja (§7.6).

Insiden yang menyentuh data pribadi (contact point person, upload customer, akses lintas tenant, data akun user) **langsung** diberi label `pdp-suspected`; RB-50 berjalan paralel dengan penanganan teknis. Perubahan severity dicatat beserta alasannya.

### 7.3 Ritme respons & komunikasi

| Sejak deklarasi | SEV1 | SEV2 |
|---|---|---|
| ≤ 15 menit | IC, kanal, issue; founder diberi tahu; DPO bila `pdp-suspected` | Ack (≤ 30 menit / 1 jam), issue |
| ≤ 30 menit | Containment pertama dijalankan | Containment |
| Berkala | Update internal tiap 1 jam | Update internal tiap 2 jam (usulan) |
| ≤ 4 jam | Email ke Owner/Admin workspace terdampak | Email bila dampak customer > 1 jam |
| Selesai | Email "pulih"; PIR ≤ 5 hari kerja | Sama |

Status page publik di luar DC belum ada (Q-RB-21); kanal customer saat ini email + banner in-app bila aplikasi hidup.

### 7.4 Template komunikasi customer (id-ID)

Kirim ke Owner/Admin workspace terdampak. Jangan mencantumkan data pribadi, nama customer lain, atau detail keamanan yang dapat disalahgunakan.

```text
Subjek: [MesinProspek] Gangguan <fitur> — <sedang ditangani | update hh:mm WIB>

Halo <Nama>,

Sejak pukul <hh:mm> WIB, <tanggal>, terjadi gangguan pada <fitur, mis. pencarian prospek / export / pengiriman webhook>.
Dampak bagi Anda: <mis. export baru belum dapat dibuat>. Aksi yang gagal tidak memotong credit Anda.
Status saat ini: <sedang diselidiki / penyebab ditemukan / pemulihan berjalan / dipantau>.
Yang perlu Anda lakukan: <tidak ada / ulangi export setelah pukul …>.
Kabar berikutnya paling lambat pukul <hh:mm> WIB.

Tim MesinProspek · <email support>
```

```text
Subjek: [MesinProspek] Gangguan <fitur> telah pulih

Gangguan pada <fitur> berlangsung pukul <hh:mm>–<hh:mm> WIB, <tanggal>, dan kini telah pulih sepenuhnya.
Penyebab: <satu kalimat>. Dampak: <…>.
Credit: aksi yang gagal tidak ditagih / <n> credit telah dikembalikan ke saldo workspace Anda.
Pencegahan: <satu–dua kalimat>.
Mohon maaf atas ketidaknyamanannya. Hubungi <email support> bila Anda masih menemukan kendala.
```

```text
Subjek: [MesinProspek] Pemeliharaan terjadwal <tanggal>, pukul <hh:mm>–<hh:mm> WIB

Kami akan melakukan pemeliharaan infrastruktur. Selama ±<n> menit, <aplikasi tidak dapat diakses / export dan upload dijeda>.
Data Anda tidak terdampak. Mohon maaf atas ketidaknyamanannya.
```

### 7.5 RB-50 — Prosedur kegagalan pelindungan data pribadi (PDP)

- **Dasar:** UU 27/2022 tentang Pelindungan Data Pribadi, **Pasal 46**: bila terjadi kegagalan pelindungan data pribadi, pengendali wajib menyampaikan pemberitahuan **tertulis paling lambat 3×24 jam** kepada subjek data pribadi dan lembaga. Pemberitahuan minimal memuat: (a) data pribadi yang terungkap; (b) kapan dan bagaimana data terungkap; (c) upaya penanganan dan pemulihan. Dalam hal tertentu, masyarakat juga wajib diberi tahu. Aturan pelaksana (PP 33/2026) **dilaporkan** berlaku 16 Januari 2027: **verifikasi ke JDIH** dan sesuaikan isi, format, dan kanal. Seluruh keputusan divalidasi konsultan hukum (BR-PRIV-03). Bagian ini prosedur operasional, bukan nasihat hukum.
- **Tujuan:** memenuhi kewajiban notifikasi ≤ 3×24 jam secara terukur dan terdokumentasi. **Prasyarat:** DPO + IC di issue `pdp-suspected`; kontak konsultan hukum (§1.4).
- **Kapan:** setiap insiden berlabel `pdp-suspected` (AL-22, AL-24, AL-18/AL-19 dengan bukti akses, AL-21 terkonfirmasi, laporan customer/peneliti, kredensial atau perangkat hilang).

| Data | Pengendali | Peran MSP | Kewajiban notifikasi |
|---|---|---|---|
| Global layer: person, contact point person (dataset) | MSP | Pengendali | MSP → subjek + lembaga ≤ 3×24 jam |
| Data akun user (email, nama, IP, sesi) | MSP | Pengendali | Sama |
| Upload customer & data tenant (list, catatan, ICP) | Tenant | Prosesor | MSP → tenant **tanpa penundaan** (target internal ≤ 24 jam, Q-RB-20); tenant memberi tahu subjek/lembaga, MSP membantu |
| Data bisnis (kelas B) | — | — | Umumnya bukan data pribadi; usaha perorangan yang memuat nama/kontak pemilik dinilai kasus per kasus |

| Waktu | Tonggak |
|---|---|
| **T0** | Diketahui (indikasi kredibel pertama; definisi final = Q-RB-20). Catat UTC + WIB di issue; pasang hitung mundur T0+72 jam |
| ≤ T0+1 jam | DPO, founder, konsultan hukum dihubungi; containment berjalan; preservasi bukti dimulai |
| ≤ T0+24 jam | Asesmen awal selesai; tenant yang menjadi pengendali diberi tahu |
| ≤ T0+48 jam | Keputusan notifikasi (DPO + hukum + founder); draf notifikasi final |
| **≤ T0+72 jam** | Notifikasi tertulis terkirim ke subjek data dan lembaga (bila wajib) |

1. **Deteksi & label:** beri label `pdp-suspected` (SEV1 default), catat T0, tambahkan DPO ke kanal insiden.
2. **Containment** tanpa menghapus bukti: kill switch/rollback, cabut sesi dan kredensial, blokir endpoint/akun, rotasi secret terkait (RB-20).
3. **Preservasi bukti:** `platform.audit_log` (immutable + ekspor bucket audit), log Loki (perpanjang retensi/ekspor ke `msp-evidence`), auditd, log WireGuard, log S3 gateway, log Traefik, GlitchTip. Catat hash setiap artefak di log bukti.
4. **Asesmen:** data apa (kelas P1/T/S, field), berapa subjek (angka + metode hitung), tenant mana dan siapa pengendalinya, kapan dan bagaimana, apakah data benar-benar diakses/keluar atau hanya terekspos, risiko bagi subjek, status mitigasi.
5. **Keputusan notifikasi** (DPO + konsultan hukum + founder): wajib/tidak; penerima (subjek, lembaga, tenant, masyarakat); kanal. Keputusan "tidak memberi tahu" didokumentasikan beserta alasannya.
6. **Notifikasi ke tenant** (bila tenant pengendali): ringkasan fakta agar tenant dapat memenuhi kewajibannya sendiri, narahubung MSP, komitmen update.
7. **Notifikasi tertulis ke subjek dan lembaga** ≤ T0+72 jam. Kanal ke lembaga mengikuti aturan pelaksana; bila lembaga belum beroperasi, kanal ditetapkan konsultan hukum.
8. **Tindak lanjut:** laporan lanjutan bila ada fakta baru; PIR; perbarui register insiden PDP dan DSR terkait.

- **Verifikasi:** bukti kirim notifikasi ≤ T0+72 jam; log bukti lengkap. **Rollback:** tidak berlaku; koreksi lewat laporan lanjutan. **Audit:** issue `pdp-suspected`, log bukti, salinan notifikasi di `msp-evidence` (akses terbatas DPO).

```text
Subjek: Pemberitahuan kegagalan pelindungan data pribadi — MesinProspek

Yth. <Nama subjek data / Lembaga>,

Kami memberitahukan bahwa pada <tanggal, jam WIB> kami mengetahui terjadinya kegagalan pelindungan data pribadi
yang berkaitan dengan <Anda / subjek data sebanyak ± n orang>.
1. Data pribadi yang terungkap: <jenis data, mis. alamat email kerja dan nomor telepon>.
2. Kapan dan bagaimana: <rentang waktu>; <penyebab singkat, mis. kesalahan konfigurasi yang membuat data terlihat oleh pengguna lain>.
3. Upaya penanganan dan pemulihan: <containment, perbaikan, langkah pencegahan>.
4. Langkah yang disarankan bagi Anda: <mis. waspada terhadap pesan yang mengatasnamakan …>.
5. Narahubung: <nama/jabatan DPO>, <email>, <telepon>.   [cek apakah diwajibkan aturan pelaksana]

Hormat kami,
<Nama, jabatan> — <badan hukum MesinProspek>
```

| No | Waktu (UTC) | Bukti | Lokasi + sha256 | Diambil oleh | Catatan |
|---|---|---|---|---|---|
| 1 | `<…>` | Ekspor `audit_log` partisi `<yyyy-mm>` | `msp-evidence/inc-<n>/…` · `<hash>` | `<nama>` | `<…>` |

### 7.6 Post-incident review (blameless)

Wajib untuk SEV1/SEV2, ≤ 5 hari kerja, berbentuk issue `pir`. Fokus pada sistem dan proses, bukan individu. Setiap insiden menghasilkan minimal satu test regresi atau alert baru (06 §14.5).

```markdown
# PIR — INC-<n>: <judul>
Severity · durasi (deteksi → pulih) · IC · penulis · tanggal review · peserta
## Ringkasan (≤ 3 kalimat, non-teknis)
## Dampak: workspace, user, request gagal, data, credit, error budget terpakai
## Timeline: | UTC | WIB | kejadian | sumber bukti |
## Deteksi: bagaimana terdeteksi, MTTD, apakah alert & runbook membantu
## Faktor penyebab: teknis · proses · organisasi (tanpa menyalahkan individu)
## Yang berjalan baik / yang perlu diperbaiki
## Action item: | # | Aksi | Owner | Tenggat | Issue |   (≥ 1 test regresi atau alert baru)
## PDP: label pdp-suspected? keputusan notifikasi + rujukan RB-50
```

---

## 8. Kalender pemeliharaan

| Frekuensi | Tugas | Pemilik | Bukti |
|---|---|---|---|
| Harian | Hasil `backup-check` 07:00 (pgBackRest dua repo, umur backup, lag `filer.sync`, backup GitLab) — RB-40 | On-call | Issue bila gagal |
| Harian | Verifikasi hash chain audit + ekspor harian ke bucket audit (AL-23) | On-call | Hasil job |
| Harian | Rekonsiliasi ledger = 0 (AL-20); DLQ (Pipeline Health); antrean suppression/DSR terhadap tenggat (FR-OPS-008) | On-call / ops | Dashboard |
| Harian | Issue `infra-drift` (06:00, selesai ≤ 2 hari kerja), `nightly-red`, `security` rescan | Maintainer | Issue |
| Mingguan | Review MR Renovate (security tanpa penundaan; River, pgx, sqlc, goose, oapi-codegen, `next`, `react`, collection Ansible, base image tidak pernah automerge) | Engineer rotasi | MR |
| Mingguan | Triage GlitchTip; review noise alert; tren DLQ; endpoint `DISABLED_FAILING`; laporan flag `weekly`; cek replikasi SeaweedFS (RB-44); serah terima on-call | On-call | Issue `oncall-handover` |
| Mingguan | Review biaya AI (biaya per brief, budget harian, cap workspace) | Product + on-call | Dashboard Cost |
| Bulanan | **Drill restore** `dr-01` Selasa pertama 09:00 (RB-47, QG-35 ≤ 35 hari) | Maintainer | Laporan `msp-evidence` |
| Bulanan | Review SLO & error budget; retro DORA; review biaya (server, off-site, LLM, email) | Tech lead | Notulen |
| Bulanan | Jendela patch/reboot OS (RB-22); sampel audit akses support (RB-02) dan log break-glass | Maintainer / DPO | Issue |
| Bulanan | Inventaris sertifikat internal; metrik suppression/DSR (lag, jumlah, tenggat) | On-call / DPO | Laporan |
| Kuartalan | **Review akses**: anggota GitLab & Maintainer, peer WireGuard, akun host, akun ops console, role DB, token (06 §15.1) | Tech lead + DPO | Checklist |
| Kuartalan | Alert drill RG-5 di staging (`alert-drill`, TC-OPS-007); restore backup GitLab (06 §7.1) | On-call | Laporan drill |
| Kuartalan | Rotasi 90 hari: password role DB, kunci S3, Redis, SSH deploy, API key LLM/email (RB-20) | Maintainer | Issue `secret-rotation` |
| Kuartalan | Uji dekripsi salinan offline kunci age dan passphrase pgBackRest/LUKS (§3.4) | 2 Maintainer | Catatan (tanpa nilai) |
| Kuartalan | Review kapasitas vs 04 §17 (DB ±200 GB dengan headroom, object 1,5–3 TB, CPU/IO); tabletop DR (RB-46); review runbook ini | Tech lead | Notulen + MR |
| Pra-rilis mayor | Drill rollback blue/green (TC-OPS-016); switchover DB terencana di staging (RB-43) | Maintainer | Laporan |
| Tahunan | Rotasi kunci age, cosign, master key webhook; token GitLab ≤ 365 hari (RB-20); review retensi (TQ-14) dan RB-50 bersama konsultan hukum | Maintainer / DPO | Issue |

---

## 9. Lampiran

### 9.1 Cheat sheet perintah

```bash
# Docker Compose (app-01)
C=$(cat /opt/msp/active_color)
docker compose -p msp-$C ps                         # status warna aktif
docker compose -p msp-$C logs -f --since 15m api     # log (juga di Loki)
docker compose -p msp-$C restart worker-export
docker compose -p msp-core ps                       # traefik, kratos, redis, unbound, smokescreen, otel (Q-RB-03)
docker exec msp-$C-api-1 /msp tool healthcheck

# Ansible (dari runner deploy atau laptop Maintainer)
I=infra/ansible/inventories/production; P=infra/ansible/playbooks
ansible-playbook -i $I $P/site.yml --check --diff --limit db-02
ansible-playbook -i $I $P/rollback.yml -e to=previous
ansible-playbook -i $I $P/rehydrate.yml --limit app-01
ansible-playbook -i $I $P/db-failover.yml -e new_primary=db-02 -e fence_host=db-01
ansible-playbook -i $I $P/tool-job.yml -e tool_args="<aksi> <argumen>"

# pgBackRest (db-01, sebagai postgres)
pgbackrest --stanza=msp info [--repo=2]
pgbackrest --stanza=msp check
pgbackrest --stanza=msp --type=diff backup
pgbackrest --stanza=msp --delta --type=time --target="<YYYY-MM-DD HH:MM:SS+00>" --target-action=promote restore
```

```sql
-- Replikasi (db-01): status standby, lag, slot, archiver
SELECT client_addr, state, sync_state, write_lag, flush_lag, replay_lag,
       pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn)) AS lag_bytes FROM pg_stat_replication;
SELECT slot_name, active, pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn)) FROM pg_replication_slots;
SELECT archived_count, last_archived_time, failed_count, last_failed_time FROM pg_stat_archiver;
-- Replika (db-02)
SELECT pg_is_in_recovery(), now() - pg_last_xact_replay_timestamp() AS replay_delay;
-- River (db-02, msp_readonly_replica; grant = Q-RB-22)
SELECT queue, state, count(*) FROM river_job GROUP BY 1,2 ORDER BY 1,2;
SELECT queue, max(now() - scheduled_at) AS umur_tertua FROM river_job WHERE state='available' GROUP BY 1;
SELECT id, kind, attempt, max_attempts, errors[array_length(errors,1)]->>'error' FROM river_job WHERE id = <id>;
```

```bash
# SeaweedFS (stor-01)
weed shell -master=stor-01:9333     # lock | cluster.ps | volume.list | volume.fsck | volume.vacuum | fs.ls /buckets | fs.du /buckets/<b> | s3.bucket.list | unlock

# nftables & WireGuard
sudo nft list ruleset | less                       # aturan egress (DOCKER-USER/forward), counter drop
sudo wg show; sudo wg show wg0 latest-handshakes   # di ops-01
sudo wg set wg0 peer <pubkey> remove               # darurat offboarding (lalu MR)

# Observability
logcli query '{service="api",env="production"} | json | correlation_id="<id>"' --since=1h
amtool --alertmanager.url=http://localhost:9093 silence add alertname=<nama> --duration=1h --comment="<RB/issue>"
```

### 9.2 Checklist & template

**Deploy production (RB-10)**

- [ ] Jendela deploy, bukan freeze, error budget ≤ 75%
- [ ] `staging-verified` (+ `rc-verified` minor/major), drill restore ≤ 35 hari, readiness hijau
- [ ] Ada migrasi? Fase expand/contract dicek; `db-contract` didahului backup incr
- [ ] On-call diberi tahu; rencana rollback jelas (digest sebelumnya)
- [ ] Pasca-deploy (06 §14.6): container warna aktif healthy dan Traefik hanya merujuk warna aktif; smoke lulus; 5xx & p95 dalam threshold 30 menit; antrean River normal, worker warna lama berhenti; versi goose sesuai; tidak ada issue GlitchTip baru severity tinggi; flag sesuai rencana; deployment & anotasi tercatat; catatan rilis (id-ID) terbit

**Bukti drill restore (RB-47 / RB-41)**

| Field | Nilai |
|---|---|
| Tanggal, pelaksana, host target | `<…>` · `<…>` · `dr-01` |
| Repo sumber, ukuran backup, label backup | repo2 · `<GB>` · `<label>` |
| `T_target` (UTC) | `<…>` |
| Mulai restore / DB menerima koneksi / verifikasi lulus | `<t1>` / `<t2>` / `<t3>` |
| **RTO terukur** (`t3 − t1`) vs target ≤ 4 jam | `<…>` · lulus/gagal |
| Transaksi terakhir (`audit_log`, `usage_event`) | `<…>` |
| **RPO terukur** (`T_target − transaksi terakhir`) vs target ≤ 15 menit | `<…>` · lulus/gagal |
| Versi goose / jumlah policy RLS / selisih baris tabel kunci | `<…>` / `<…>` / `<…>` |
| Sampel objek off-site (n, checksum cocok) | `<…>` |
| Penghapusan data (volume LUKS dihancurkan) | ya/tidak, `<waktu>` |
| Temuan & action item | `<issue>` |

**Insiden (on-call/IC)**

- [ ] Ack; SEV ditetapkan; IC, comms, scribe ditunjuk
- [ ] Kanal `#inc-…` + issue `incident` dibuat; waktu deteksi (UTC + WIB) dicatat
- [ ] Menyentuh data pribadi? → label `pdp-suspected`, T0 dicatat, DPO dihubungi (RB-50)
- [ ] Containment dijalankan (flag/rollback/failover/blokir) dan dicatat
- [ ] Founder diberi tahu (SEV1 ≤ 15 menit); customer ≤ 4 jam (SEV1)
- [ ] Bukti dipreservasi (log, audit, auditd) — tidak ada penghapusan
- [ ] Pemulihan diverifikasi ≥ 30 menit; komunikasi "pulih" terkirim
- [ ] Credit terdampak dikoreksi (RB-28); PIR dijadwalkan (SEV1/SEV2)

### 9.3 Glosarium

| Istilah | Arti di runbook ini |
|---|---|
| Blue/green | Dua project Compose (`msp-blue`, `msp-green`); satu aktif, satu untuk versi baru |
| Canary / bake | 10% traffic ke warna baru selama 10 menit / 100% selama 20 menit sambil dipantau |
| Digest | Hash image (`@sha256:…`); deploy selalu by digest |
| Break-glass | Jalur akses darurat di luar prosedur normal, tercatat dan dirotasi setelahnya |
| DLQ / `discarded` | Job River yang kehabisan percobaan; dapat di-redrive |
| Epoch | Generasi skor ICP; epoch baru diaktifkan atomik |
| Fence | Mematikan/mengisolasi primary lama agar tidak terjadi split brain |
| Failover / switchover / failback | Promote standby karena gagal / pertukaran peran terencana / mengembalikan peran semula |
| PITR | Point-in-time recovery dari backup + WAL pgBackRest |
| RPO / RTO | Data maksimum yang boleh hilang / waktu maksimum sampai pulih |
| Stanza | Konfigurasi pgBackRest per cluster PostgreSQL (usulan nama `msp`) |
| Rehydrate | Menulis ulang secret ke tmpfs setelah reboot |
| Error budget / burn rate | Sisa ketidaktersediaan yang diizinkan SLO (99,5% ≈ 3,6 jam/bulan) / laju pemakaiannya |
| `value_hash` | Hash ber-pepper nilai kontak untuk mencari dan men-suppress tanpa menyimpan nilai mentah |
| Suppression | Penghapusan global person/contact point dari semua projection dan sistem hilir |
| DSR | Permintaan subjek data (akses, koreksi, hapus, keberatan) |
| Purge / pseudonimisasi | Penghapusan data tenant hari ke-30 / penggantian identitas aktor dengan HMAC |
| T0 | Saat kegagalan pelindungan data pribadi diketahui; awal hitung mundur 3×24 jam |

---

## 10. Pertanyaan terbuka

ID `Q-RB-nn` bersifat lokal di dokumen ini. Setelah diputuskan, pindahkan ke register 00 §7 atau perbarui 04/06.

| ID | Pertanyaan / gap | Default sementara | Ref |
|---|---|---|---|
| Q-RB-01 | Komposisi rotasi on-call, provider paging (telepon/SMS), kanal chat self-hosted, bridge insiden | Model §1.2; chat + email; paging ditentukan | Q-CI-06, Q-CI-07 |
| Q-RB-02 | WireGuard dan Tang sama-sama di `ops-01`: bila `ops-01` mati, akses admin dan auto-unlock LUKS hilang | Konsol IPMI/KVM via VPN provider + endpoint WireGuard kedua terbatas (mis. di `app-01`, hanya SSH ke host prod) | 04 §15.4, 06 §3.5 |
| Q-RB-03 | Nama project Compose untuk layanan bersama (traefik, kratos, redis, unbound, smokescreen, otel) | `msp-core` di `/opt/msp-core/` | 06 §17.3 |
| Q-RB-04 | PgBouncer hanya di `db-01` (04 §2.2): saat `db-01` mati, PgBouncer ikut mati. Endpoint primary saat failover belum didefinisikan | PgBouncer di kedua host DB + nama DNS internal/variabel `db_primary_host` yang diubah `db-failover.yml` | 04 §2.2, 06 §11.6 |
| Q-RB-05 | Playbook/inventory yang dirujuk runbook tetapi belum ada di 06 §11.1: `os-patch.yml`, `db-rebuild-standby.yml`, `pitr-restore.yml`, inventory `dr`, tag `certs` | Ditambahkan ke `infra/ansible/` | 06 §11.1 |
| Q-RB-06 | Aksi ops/`tool` yang belum ada: bulk DLQ redrive, rollback `score-model-activate`, aktivasi `geo-X.Y.Z`, laporan `ledger-reconcile` per workspace, `audit-verify` | Tambah aksi `msp tool` + endpoint ops | 04 §3.1, 06 §10.3, Q-CI-11 |
| Q-RB-07 | Rollback snapshot = publish ulang snapshot sebelumnya, tetapi state machine 04 §4.2 tidak punya transisi `SUPERSEDED → PUBLISHED` | Tambah transisi khusus ops (audit) | 04 §4.2, 06 §10.2 |
| Q-RB-08 | Perilaku degraded tidak terdefinisi: rate limit & counter fair-use saat Redis down; tidak ada kill switch untuk unlock | Rate limit fail-open terbatas (in-process), fair-use dari tabel `usage.profile_view`; flag `unlock.create` | 04 §14.4, §18.3, 05 §3.18 |
| Q-RB-09 | Threshold & metrik usulan belum ada di 04 §16.4: AL-03, AL-06, AL-09 (80% + waktu reset budget harian), AL-14 tren, AL-15..19, AL-22..24, interpretasi burn-rate 6 jam; metrik baru `msp_ai_briefs_total`, `msp_webhook_attempts_total`, `msp_suppression_propagation_lag_seconds`, `msp_tenant_guard_violations_total` | Nilai §6 | 04 §16.3–16.4 |
| Q-RB-10 | Tidak ada mode maintenance/read-only API untuk PITR, switchover, atau DR | Flag `platform.read_only` (write → 503 dengan pesan) | 04 §18.3 |
| Q-RB-11 | Refund manual untuk satu operasi tidak punya endpoint | Grant `ADJUSTMENT` dengan `ref_type=usage_event` | 04 §14, FR-BILL-006 |
| Q-RB-12 | Kapan pseudonimisasi audit/ledger saat penutupan workspace (hari ke-30 atau akhir retensi)? Bagaimana dengan salinan off-site append-only? | Pseudonimisasi saat purge hari ke-30; retensi TQ-14 | FR-AUD-005, 04 §15.8 |
| Q-RB-13 | Image rilis untuk DR seluruh DC hanya ada di registry `ops-01` (+ backup GitLab harian) | Mirror image `v*` ke registry/S3 off-site | 06 §11.6 |
| Q-RB-14 | Tenggat DSR per tipe hak | Ditetapkan konsultan hukum; sementara 3×24 jam untuk hapus/keberatan | FR-AUD-004 |
| Q-RB-15 | Ops me-redrive delivery webhook atas nama tenant: otorisasi (akses support read-only) dan semantik credit | Hanya untuk kesalahan sisi MSP, tanpa tagihan, teraudit | 04 §12.4, §15.9 |
| Q-RB-16 | Rotasi `CURSOR_HMAC_KEY`, `SESSION_PEPPER`, `VALUE_HASH_PEPPER` tidak ada di 06 §12.4 | Hanya saat dugaan bocor; `VALUE_HASH_PEPPER` butuh migrasi re-hash | 06 §12.4 |
| Q-RB-17 | Tenggat `app-02`: PRD R-08 "sebelum M-3 (early paid)" vs 06 "sebelum paid GA" | Ikuti PRD (lebih awal) | R-08, Q-CI-05 |
| Q-RB-18 | Frekuensi drill restore: 06 §7.1 bulanan vs 05 §3.25 "pre-release, kuartalan" | Bulanan (06, QG-35) | 05 §3.25 |
| Q-RB-19 | Sizing server berbeda: 04 §17.2 (`app-01` 32 core, DB 256 GB) vs 06 §3.5 (16 core, DB 128 GB) | Ikuti 04 sampai Q-02 | 04 §17.2, 06 §3.5 |
| Q-RB-20 | PDP: definisi "diketahui" (T0), lembaga dan kanal pelaporan, isi wajib (narahubung?) menurut aturan pelaksana (PP 33/2026, verifikasi JDIH), target notifikasi ke tenant saat MSP prosesor | T0 = indikasi kredibel pertama; tenant ≤ 24 jam | BR-PRIV-03 |
| Q-RB-21 | Status page publik di luar DC | Status page self-hosted di VPS probe eksternal | 06 §16.1 |
| Q-RB-22 | Grant `msp_readonly_replica` untuk diagnosis (`river`, `goose_db_version`, `ops`) dan prosedur write break-glass | Grant `SELECT` pada tabel non-tenant tersebut | 04 §5.8, 06 §3.1 |
| Q-RB-23 | Aktivasi prompt: 04 §13.2 "dipilih lewat config" vs 06 §10.4 flag `rel.ai.brief-<versi>`; versi `angle-X.Y.Z` tidak ada di tag terlindungi 06 §2.8 | Flag (06) + tambah tag `angle-*` | 04 §13.2, 06 §10.4 |
| Q-RB-24 | Ops console dapat diakses publik (MFA) atau dibatasi WireGuard/allowlist IP? | Batasi ke WireGuard | 04 §15.1, FR-AUTH-006 |
