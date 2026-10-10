# 04 — Technical Design (Software Design Document)

## MesinProspek — Sales Intelligence Engine (Sellable MVP)

| Field | Value |
|---|---|
| Dokumen | Technical Design / Software Design Document (SDD) |
| Versi | 0.1 (Draft) |
| Tanggal | 2026-10-10 |
| Basis | `01-prd.md` v0.1 (sumber kebenaran requirement), `00-analisis-dokumen-awal.md`, MVP Pack v1.2 (§04, §05, §11–§13, Rev 1.2), Crawler Spec v1.0 (PAUSED; pola §11, §12, §13, §14.2, §19.2 diadopsi) |
| Status crawler | **PAUSED** (D-01). Input = dataset snapshot + upload customer + data referensi wilayah (D-02) |
| Bahasa backend | **Go** (D-21, Accepted): satu binary `msp` (`api`, `worker`, `migrate`). Frontend tetap Next.js + TypeScript |
| Hosting | **Dedicated server di data center Indonesia** (D-19; dedicated = Accepted, stack = Proposed). CI/registry GitLab CE (D-18). Identity provider = D-20 (Open) |
| Audience | Engineering (backend, frontend, data), DevOps/SRE, QA, Security |
| Menutup gap | G-01, G-02, G-04, G-06 (rumus teknis), G-08 (teknis), G-13, G-14 (teknis), G-18, G-19 |

> Konvensi: setiap bagian mencantumkan FR/NFR/BR yang dipenuhi. Keputusan arsitektur baru dicatat sebagai `ADR-nn` (§1.3). Label rilis: **MVP0**, **MVP1**, **V1**, **V1.5**, **V2** mengikuti PRD §6. Snippet kode/SQL bersifat ilustratif (desain), bukan implementasi final.

### Daftar isi

1. Ringkasan & ADR register · 2. Arsitektur · 3. Modul backend · 4. Input Contract · 5. Data model · 6. Processing pipeline · 7. Scoring engine · 8. Opportunity engine · 9. Geo engine · 10. Query & search · 11. API design · 12. Events & webhooks · 13. AI subsystem · 14. Usage, credits, entitlements · 15. Security & privacy · 16. Observability · 17. Performa & kapasitas · 18. Konfigurasi, environment & feature flags · 19. Jalur reaktivasi crawler · 20. Pertanyaan teknis terbuka

---

## 1. Ringkasan & ADR register

### 1.1 Ringkasan

1. **Satu binary backend Go (`apps/api/cmd/msp`) berbentuk modular monolith** dengan 10 modul berbatas tegas (ADR-01, D-21). Subcommand `api` (HTTP, `net/http` + oapi-codegen strict-server), `worker` (River), dan `migrate` (goose) berasal dari binary yang sama.
2. **Engine source-agnostic.** Semua input masuk lewat **Input Contract**: Source Registry, dataset snapshot immutable `ds-YYYY.MM.n`, dan skema `ObservationRecord` (ADR-02). Crawler kelak hanya menjadi satu *source adapter* lagi (§19).
3. **Tenancy dua lapis** (D-03). *Global layer* berisi entity kanonik, evidence, metrik area, dan gap fact; dapat dibaca semua tenant secara read-only. *Tenant layer* berisi ICP, Offering, skor, list, upload, unlock, dan lain-lain, diisolasi dengan **PostgreSQL RLS** ditambah pemeriksaan aplikasi (ADR-03).
4. **Company 1—N Establishment** (D-05). Establishment adalah unit geo dan unit skor; Company adalah unit unlock dan unit baris hasil pencarian (ADR-04).
5. **PostgreSQL 16 + PostGIS + pg_trgm + unaccent** menangani canonical store, full-text search, dan geo. OpenSearch tidak dipakai di MVP (ADR-05).
6. **Pipeline idempotent per stage.** Kunci idempotensi = `hash(input + versi stage)`, checkpoint per batch disimpan di Postgres, dan stage yang gagal menghasilkan `Unknown`, bukan nilai palsu (§6).
7. **Skor deterministik dan berversi** (`priority-1.2.0`, Lampiran A PRD). Skor disimpan ringkas per epoch dan dipartisi per workspace; penjelasannya dibangun ulang dari faktor yang tersimpan (ADR-16).
8. **Opportunity = gap fact global × relevansi Offering tenant.** Relevansi dihitung saat dibaca, tidak dimaterialisasi per tenant (ADR-15).
9. **Event via River** (queue berbasis Postgres) yang di-*enqueue* dalam transaksi yang sama dengan perubahan domain, sehingga River sekaligus berfungsi sebagai transactional outbox (ADR-07). Webhook ditandatangani HMAC (`MSP-Signature`), di-retry hingga ≤ 24 jam, dan dikirim melalui **egress proxy + nftables** dengan kontrol SSRF (ADR-20).
10. **AI di balik abstraksi provider.** Output terstruktur dengan sitasi per kalimat, divalidasi oleh grounding validator. Nilai kontak pribadi tidak pernah dikirim ke LLM (ADR-09).
11. **Credit memakai ledger double-entry append-only** dengan mekanisme *hold → capture/release*, sehingga operasi yang gagal tidak pernah menagih (ADR-10).
12. **Infrastruktur self-hosted di Indonesia** (D-19): Docker Compose + Traefik di `app-01`, PostgreSQL primary/replica di `db-01`/`db-02`, MinIO untuk object storage, monitoring di `ops-01` (ADR-18).

### 1.2 Prinsip desain (diturunkan dari MVP Pack, tidak berubah)

| Prinsip | Konsekuensi teknis |
|---|---|
| Evidence over magic | Setiap claim tampil memiliki `claim_type` + `evidence_ids`. Evidence coverage diukur sebagai metrik (FR-DATA-006). |
| Unknown over invented | `UNKNOWN` adalah nilai first-class di enum, skor, API, dan export. Kegagalan sementara → UNKNOWN (BR-VER-02). |
| One canonical record, many projections | Satu store kanonik → projection `prospect_search`, profil, export, webhook, dan input AI dibangun dari sumber yang sama. |
| Tenant-aware & auditable | Setiap write tenant dijalankan dalam `tenancy.WithTx(ctx, workspaceID, fn)`. Aksi penting menghasilkan `audit_log`. |
| Reproducible | Skor dan metrik area dapat dihitung ulang dari `snapshot` + `model_version`/`geo_version` (NFR-DATA-01). |
| Modular acquisition | Engine tidak mengenal asal record. Asal record hanya tercatat di evidence. |

### 1.3 ADR register

Semua ADR berstatus **Proposed** kecuali ditandai lain. Kolom "Ref" menautkan ke keputusan D-xx (00 §7) dan requirement.

| ID | Keputusan | Konteks | Alternatif ditolak | Konsekuensi | Status | Ref |
|---|---|---|---|---|---|---|
| ADR-01 | **Backend Go, satu binary modular monolith** (`cmd/msp`: `api`, `worker`, `migrate`; 10 modul di `internal/`). Batas modul ditegakkan oleh visibility Go `internal/` + **golangci-lint `depguard`** | Tim kecil, domain belum stabil, butuh transaksi lintas modul (unlock + ledger + audit). Go: binary statis, konkurensi murah untuk pipeline/webhook, memori kecil di dedicated server, satu bahasa dengan Kratos (D-20) | Python/FastAPI (draf awal); microservice per §11 MVP Pack | Satu deploy, satu riwayat migrasi. Python hanya dipakai offline (notebook kalibrasi ML), di luar service. Pemecahan service nanti mengikuti batas modul | **Accepted** (Go, D-21) / Proposed (struktur) | D-21, D-11 |
| ADR-02 | Crawler di-pause → **Source Adapter interface + Input Contract** (Source Registry, snapshot, ObservationRecord JSON Schema) | Dataset sudah tersedia. Kontrak input belum pernah didefinisikan (G-01) | Import ad-hoc langsung ke tabel kanonik | Sumber baru (partner, crawler) cukup menulis adapter. Validasi dan provenance seragam | Proposed | D-01, D-02, FR-DATA-001 |
| ADR-03 | **Tenancy dua lapis + PostgreSQL RLS** (`current_setting('app.workspace_id')`) + filter aplikasi + role DB terpisah | Skor/ICP privat per tenant, data bisnis dipakai bersama (G-02) | DB/schema per tenant; data global disalin per tenant | Isolasi ganda (NFR-SEC-03). Semua query tenant wajib lewat `tenancy.WithTx` | Proposed | D-03, FR-AUD-006 |
| ADR-04 | **Company 1—N Establishment**. Skor per establishment, rollup company = maks (BR-SCORE-01) | Domain yang sama untuk cabang berbeda tidak boleh di-merge (G-04) | Satu tabel Company | ER dua level (§6.4). Baris hasil = company + establishment terbaik | Proposed | D-05 |
| ADR-05 | **Postgres FTS (`simple` + `unaccent`) + `pg_trgm` + PostGIS**; tanpa OpenSearch di MVP | Volume 5 juta baris muat di RAM DB. Mengurangi beban operasional | OpenSearch sejak awal | Satu sistem untuk konsistensi data. Projection dirancang agar bisa dipindah ke OpenSearch (V2) tanpa ubah API | Proposed | D-12, NFR-PERF-01 |
| ADR-06 | **River** (queue berbasis Postgres, Go) untuk semua job: retry + backoff, unique job (idempotensi), periodic job (leader election), dan **insert transaksional**. Antrean: `pipeline`, `scoring`, `export`, `webhook`, `ai`, `maintenance`; prioritas 1–4 dalam antrean. Redis hanya untuk cache, rate limit, dan counter (opsional) | Butuh retry/backoff/DLQ (Crawler §13) dan job yang tidak hilang saat restart | Dramatiq/Celery + Redis broker; Temporal | State job ada di Postgres (`river_job`) sehingga ikut backup/PITR. Job habis percobaan → state `discarded` (= DLQ) yang dapat di-*retry* dari ops console. Beban tambahan di Postgres dipantau (§17.4) | Proposed | FR-OPS-002 |
| ADR-07 | **Outbox = River insert transaksional**: perubahan domain + baris `platform.event` + job `event.dispatch` di-commit dalam **satu transaksi** (`InsertTx`). Worker dispatch melakukan fan-out ke handler internal dan webhook delivery | Event tidak boleh hilang atau terkirim tanpa commit (dual-write) | Tabel outbox + dispatcher polling; publish ke broker setelah commit | Tanpa komponen dispatcher terpisah. At-least-once; konsumen men-dedupe dengan `event_id` (MVP Pack §12) | Proposed | FR-EXP-003 |
| ADR-08 | **OpenAPI-first**: `contracts/openapi/openapi.yaml` direview lebih dulu. Server Go di-generate dengan **oapi-codegen (strict-server)** di atas `net/http` (routing 1.22+). Client TS untuk Next.js di-generate dari spec yang sama (`openapi-typescript` + `openapi-fetch`) ke `packages/api-client`. Next.js hanya UI; API Go disajikan same-origin di `/api` via Traefik (tanpa CORS) | Web dan integrasi butuh kontrak stabil (MVP Pack §12) | Code-first (spec dihasilkan dari kode) | Handler wajib memenuhi interface hasil generate. CI: regenerasi tanpa diff + `oasdiff` untuk breaking change. Schemathesis menguji kontrak | Proposed | PRD §12.2 |
| ADR-09 | **Abstraksi provider AI** (port `LLMProvider`, alias model dapat dikonfigurasi) + **grounding validator** deterministik + redaksi PII | Biaya, eval, dan risiko halusinasi (G-14) | Panggil SDK provider langsung dari endpoint | Provider bisa diganti tanpa ubah domain. Adapter memakai SDK Go resmi provider. Alias default: kelas Sonnet (brief), kelas Haiku (klasifikasi) | Proposed (provider: Open) | D-15, FR-AI-001..006 |
| ADR-10 | **Credit ledger double-entry append-only** dengan akun `available`/`held`/`consumed` dan alur *hold → capture/release* | Idempotensi, refund, dan rekonsiliasi 100% (RG-4) | Kolom saldo yang di-update langsung | Saldo = Σ ledger. Operasi gagal → release (tanpa tagihan) | Proposed | D-08, FR-BILL-001..005 |
| ADR-11 | Error **RFC 9457 `application/problem+json`** + katalog `code` stabil | Error berkode wajib untuk entitlement (FR-BILL-003) | Format error ad-hoc | Client menangani error berdasarkan `code`, bukan teks | Proposed | FR-BILL-003 |
| ADR-12 | **Cursor (keyset) pagination** dengan cursor ber-HMAC yang mengikat `epoch` skor | Offset lambat dan tidak stabil pada 5 juta baris | Offset/limit | Hitungan total terpisah (eksak ≤ 10.000, selebihnya estimasi; FR-SRCH-006) | Proposed | FR-SRCH-006 |
| ADR-13 | **Claim disimpan sebagai kolom bertipe (nilai pemenang) + `claim_meta` JSONB per field** (claim_type, confidence, observed_at, evidence_ids, alternates). Tidak memakai tabel claim generik (EAV) | Profil p95 ≤ 500 ms dan filter search butuh kolom bertipe. EAV ≈ 125 juta baris | Tabel `claim` generik | Satu baris dibaca per entity. Field baru = migrasi. Integritas `evidence_ids` dijaga oleh pipeline + checker | Proposed | BR-TRUST, NFR-PERF-02 |
| ADR-14 | **ID = UUIDv7** (dibangkitkan di Go) di DB, diekspos sebagai **ID berprefiks** (`cmp_`, `est_`, …, gaya TypeID). Merge membuat `entity_redirect` | ID stabil lintas re-run (FR-DATA-007), mudah dibaca di log | ID integer, ID deterministik dari hash sumber | Pemetaan sumber→entity lewat `entity_source_link`. ID lama tetap bisa di-resolve | Proposed | FR-DATA-007, MVP-01 |
| ADR-15 | **Relevansi opportunity tenant dihitung saat dibaca** (gap fact global × tag Offering), tidak dimaterialisasi per workspace | Materialisasi = 200 ws × 5 juta × ±3 gap | Tabel opportunity per tenant | Murah dan selalu konsisten dengan Offering. Snapshot opportunity ikut tersimpan di faktor skor | Proposed | D-04, FR-INTL-007 |
| ADR-16 | **Skor disimpan per epoch** (`intel.score`, partisi HASH per workspace, faktor ringkas `smallint[]`). Epoch baru diaktifkan secara atomik | Recompute ≤ 10 menit tanpa mengganggu pembaca (NFR-PERF-05) | UPDATE in-place; JSON penjelasan per baris | ±75 juta baris tetap ±17 GB. Penjelasan dibangun ulang dari faktor + config | Proposed | FR-INTL-008/009/011 |
| ADR-17 | **Peta: vector tile batas wilayah statis (PMTiles per `area_set_version`) + metrik via JSON**, di-join di client (MapLibre `feature-state`) | Geometri publik dan tidak berubah. Metrik bersifat tenant-specific | `ST_AsMVT` dinamis per tenant | Tile bisa di-cache selamanya. Endpoint `ST_AsMVT` hanya untuk dev/fallback | Proposed | FR-GEO-002 |
| ADR-18 | **Dedicated server di DC Indonesia**: Docker Compose + Traefik (`app-01`, kelak `app-02`), PostgreSQL 16 + PostGIS self-managed (`db-01` primary, `db-02` streaming replica, pgBackRest PITR), Redis self-managed (AOF), **MinIO** (S3 API), provisioning Ansible | Residensi data dan biaya. Tidak ada layanan cloud terkelola | AWS ap-southeast-3 (ECS/RDS/ElastiCache/S3) | Tim mengoperasikan sendiri backup, failover, patching. Deploy blue/green via Traefik (`06-ci-cd.md`). Abstraksi S3 dipertahankan | Accepted (dedicated) / Proposed (stack) | **D-19**, D-18, D-14, NFR-PRIV-03 |
| ADR-19 | **Port identitas provider-agnostic** (OIDC atau Kratos session → *server session milik aplikasi*). Rekomendasi: **Ory Kratos self-hosted** | Auth SaaS akan memindahkan data identitas user ke luar negeri | Auth SaaS; Keycloak | Aplikasi hanya bergantung pada klaim `sub`, `email`, `email_verified`, level MFA | **Open** | **D-20**, FR-AUTH-001/006 |
| ADR-20 | **Seluruh request keluar ke URL milik customer** lewat worker khusus → validasi aplikasi + `net.Dialer.Control` yang memeriksa ulang IP hasil resolve di setiap koneksi → **egress proxy (Smokescreen)** → aturan **nftables** (blok private/metadata/LAN) | Tidak ada security group cloud di dedicated server (NFR-SEC-05) | Validasi aplikasi saja | Tiga lapis kontrol SSRF. Proxy juga dipakai untuk egress AI/email (allowlist) | Proposed | FR-EXP-003, Crawler §12 |
| ADR-21 | **Observability self-hosted**: OpenTelemetry Go SDK + `log/slog` JSON → OpenTelemetry Collector → Prometheus + Grafana + Loki + Tempo + Alertmanager; **GlitchTip** untuk error tracking | Residensi data dan biaya; selaras D-19 | SaaS observability | Retensi dan PII di log dikontrol sendiri | Proposed | NFR-OBS-01 |
| ADR-22 | **Secret via SOPS + age** (didekripsi saat deploy ke tmpfs). **Enkripsi at-rest**: LUKS full-disk + repo pgBackRest terenkripsi + MinIO SSE. Enkripsi aplikasi (AES-256-GCM) untuk secret webhook | Tidak ada cloud KMS/secret manager | Vault (HashiCorp/OpenBao) | Operasi sederhana di MVP. Vault/OpenBao dipertimbangkan di V1 | Proposed | NFR-SEC-02/04 |
| ADR-23 | **Record upload customer = overlay privat**: MATCHED → link ke entity global; NEW → *private prospect* (tenant projection), muncul di search hanya bila opt-in | BR-ER-07: upload tidak boleh mengubah global | Merge upload ke global | Upload bernilai tanpa mencemari global. Search melakukan UNION terkontrol (§10.9) | Proposed | FR-DATA-012 |
| ADR-24 | **Observation disimpan per record** (envelope berisi field-level observation), **tabel dipartisi LIST per snapshot**, snapshot lama di-detach dan diarsip ke object storage | 5 juta record × ±15 field = 75 juta baris per snapshot bila disimpan per field | Satu baris per field | Partisi dapat di-drop/arsip. Evidence menunjuk ke record + field | Proposed | FR-DATA-001, G-19 |
| ADR-25 | **Scoping workspace via header `MSP-Workspace`** + web memakai **cookie same-origin** (`/api` di-route Traefik) | Satu user bisa anggota >1 workspace (FR-AUTH-002) | Path `/v1/workspaces/{id}/...` | Setiap tab membawa workspace eksplisit. API key (V1) terikat satu workspace | Proposed | FR-AUTH-002 |
| ADR-26 | **Akses data: pgx v5 + sqlc** (SQL type-safe, tanpa ORM) + **goose** (migrasi SQL). Bulk load via `pgx.CopyFrom`. RLS via `SET LOCAL app.workspace_id` per transaksi | Query berat FTS/pg_trgm/PostGIS/window function lebih jelas sebagai SQL asli | ORM (GORM/ent); query builder | SQL direview sebagai kode. Kode hasil sqlc per modul hanya boleh diimpor repository modul pemilik (depguard) | Proposed | D-21, D-12 |

---

## 2. Arsitektur

### 2.1 Context diagram

```
                       +-------------------------------------------+
  Sales user --------->|                                           |<--- Ops admin MSP (MFA, FR-AUTH-006)
  (browser, id-ID/en)  |              MesinProspek (MSP)           |
  Integrasi customer ->|  web (Next.js) + api (Go) + workers (Go) |---> Webhook / CRM customer (HTTPS, via egress proxy)
  (API key, V1)        |                                           |---> LLM provider (data level company saja, D-14/D-15)
                       +---+-----------------+-----------------+---+---> Email provider (SMTP: verifikasi, undangan)
                           ^                 ^                 ^     ---> DNS publik (MX/A/SPF via resolver Unbound)
                           |                 |                 |
              Dataset bisnis         Referensi wilayah     Upload CSV/XLSX
              (snapshot, ops)        (kode Kemendagri,     (customer, privat)
                                     batas, populasi)
```

### 2.2 Container & deployment diagram (D-19)

```
 Internet --443--> +=========================== app-01 (Docker Compose) ===========================+
                   | traefik  (TLS; app.<dom> -> web; app.<dom>/api -> api; files.<dom> -> MinIO)   |
                   |   web (Next.js SSR, x2)            api (`msp api`, x2) <-> kratos (D-20)        |
                   |   worker-pipeline  worker-scoring  worker-export  worker-ai  worker-maintenance |
                   |   (`msp worker --queues=…`; River: retry, periodic job, leader election)        |
                   |   worker-webhook --(hanya)--> smokescreen (egress proxy) --nftables--> Internet |
                   |   redis (cache/rate limit, opsional)  unbound (resolver DNS)  otel-collector    |
                   +============================+======================+===========================+
                                LAN privat (VLAN) |                      |
              +-------------------+--------------+                      |
              v                   v                                     v
   +----- db-01 -----+  stream  +----- db-02 -----+          +-------- stor-01 --------+
   | PostgreSQL 16   |--------->| hot standby     |          | MinIO (S3 API, SSE)     |
   | PostGIS, pgtrgm |   WAL    | (read replica:  |          | msp-snapshots (WORM)    |
   | PgBouncer       |          |  search/profil) |          | msp-uploads, msp-exports|
   | pgBackRest -----+----------+-----------------+--------->| msp-tiles, pgbackrest   |
   +-----------------+                                       +-------------------------+
                     +-------------- ops-01 --------------+   +---- ci-01 ----+
                     | GitLab CE + registry | Prometheus  |   | GitLab runners|
                     | Grafana, Loki, Tempo, Alertmanager |   +---------------+
                     | GlitchTip | Tang (unlock LUKS)     |
                     +------------------------------------+
```

| Host | Peran | Layanan | Catatan |
|---|---|---|---|
| `app-01` | Aplikasi | traefik, web, api, kratos, worker pools (River), redis (opsional), unbound, smokescreen, otel-collector | SPOF di MVP (99,5% masih tercapai dengan restore cepat). `app-02` ditambahkan untuk HA (Traefik di dua node + DNS failover/VRRP, Redis replica + Sentinel) |
| `db-01` | DB primary | PostgreSQL 16 + PostGIS, PgBouncer (transaction mode), pgBackRest, postgres_exporter | Satu-satunya target write. Juga menyimpan antrean River |
| `db-02` | DB replica | Hot standby (streaming, async), `hot_standby_feedback=on` | Melayani read search/profil. Failover manual via runbook (Patroni dipertimbangkan saat ada node ketiga) |
| `stor-01` | Object storage | MinIO single-node multi-drive (erasure coding) | Bucket privat, presigned URL via `files.<domain>` |
| `ops-01` | Ops | GitLab CE + registry, Prometheus, Alertmanager, Grafana, Loki, Tempo, GlitchTip, Tang | Detail CI di `06-ci-cd.md` |
| `ci-01` | CI | GitLab runners | — |

### 2.3 Contoh alur request

**(a) Search (FR-SRCH-001..006, NFR-PERF-01)**
1. Browser → Traefik → `web` (React Server Component, client TS hasil generate) → `GET /api/v1/prospects?icp_id=…&q=…` (Traefik meneruskan `/api` ke `msp api`) dengan cookie `__Host-msp_sess` dan header `MSP-Workspace`.
2. `platform.auth` me-resolve sesi (cache Redis, TTL 60 s) → `Principal(user, workspace, role)` → cek permission `PROSPECT_READ`.
3. `platform.tenancy` membuka transaksi read-only di **db-02**: `SELECT set_config('app.workspace_id', :ws, true)`.
4. `query.search` menjalankan keyset query pada `query.prospect_search ⋈ intel.score` (epoch aktif). Masking kontak memakai `usage.unlock`.
5. Respons berisi `items[]`, `next_cursor`, `count` (eksak/estimasi), dan header `MSP-Correlation-Id`.

**(b) Unlock (BR-CRED-02, FR-BILL-001)**
`POST /v1/unlocks` + `Idempotency-Key` → `usage.Service.UnlockCompanies` dalam satu transaksi di db-01: kunci baris `credit_balance` → cek saldo → tulis `usage_event` + baris ledger (`available → consumed`) + baris `unlock` + `audit_log` + `event(unlock.created)` + job River → commit → 200 berisi kontak yang sudah tidak ter-mask.

**(c) Simpan ICP → skor (FR-ICP-002, NFR-PERF-05)**
`PUT /v1/icps/{id}` → `icp_version` baru + `score_job(QUEUED)` + job River antrean `scoring` prioritas 1 (insert transaksional) → seleksi kandidat lewat SQL (filter wajib) → scoring per chunk 50.000 baris → `COPY` ke `intel.score` (epoch n+1) → `icp_area_metric` → aktifkan epoch n+1 secara atomik → `icp.scoring_completed`. Selama proses, UI melakukan polling `GET /v1/icps/{id}/scoring-status` dan menampilkan "Menghitung skor".

**(d) Export (FR-EXP-001/002, NFR-PERF-04)**
`POST /v1/exports/preview` (biaya) → `POST /v1/exports` + `Idempotency-Key` → **hold** credit untuk record yang belum di-unlock → job `export` mengalirkan query (server-side cursor) → file ke MinIO `msp-exports/{ws}/…` → **capture** + buat unlock → `export.completed` → `GET /v1/exports/{id}/download` mengembalikan 302 ke presigned URL (24 jam). Bila gagal → **release** + status `FAILED` yang dapat di-retry.

**(e) Kirim ke webhook (FR-EXP-004)**
`POST /v1/webhook-sends` → hold → satu `event(prospect.sent)` per prospek + job `event.dispatch` → worker membuat `webhook_delivery` + job `webhook.deliver` → `worker-webhook` mengirim lewat Smokescreen → 2xx → capture. DLQ (setelah ±20,6 jam) → release.

---

## 3. Modul backend (`apps/api/`, Go — D-21)

### 3.1 Struktur, build, dan pustaka

```
apps/api/
  cmd/msp/                 main: subcommand `api`, `worker`, `migrate`, dan `tool …` (ops: score-reproduce, taxonomy-suggest, snapshot-load)
  internal/
    app/                   composition root: wiring server HTTP (oapi-codegen strict), client + worker River, DAG pipeline, config
    platform/              auth, tenancy, rbac, audit, flags, events, jobs (helper River, DAG runner), idempotency, netguard,
                           ids, problem (RFC 9457), pagination, i18n, telemetry
    identity/ ingest/ intelligence/ geo/ query/ action/ ai/ usage/ ops/
       <modul>/service.go  facade publik: interface Service, tipe & DTO — satu-satunya API modul untuk modul lain
       <modul>/http.go     handler strict-server untuk operasi OpenAPI milik modul
       <modul>/jobs.go     worker River / stage pipeline milik modul
       <modul>/internal/   logika domain & repository (visibility Go `internal/`: tidak dapat diimpor dari luar modul)
    db/<modul>db/          kode hasil sqlc per schema modul (hanya diimpor repository modul pemilik; depguard)
    api/                   kode hasil oapi-codegen (interface server + tipe) dari contracts/openapi/openapi.yaml
  db/migrations/           goose SQL (satu riwayat; objek dikelompokkan per schema modul)
  db/queries/<modul>/      SQL sumber sqlc (FTS, pg_trgm, PostGIS, window function ditulis apa adanya)
```

- **Build:** `CGO_ENABLED=0` → binary statis → image `distroless/static` (non-root). Satu image untuk `api`, `worker`, dan `migrate`.
- **Tooling:** golangci-lint (`depguard`, `gosec`, `staticcheck`, `errcheck`, `govet`, `revive`), `govulncheck`, `go test -race`, `sqlc generate` + `sqlc vet`, dan `oapi-codegen`. CI memastikan regenerasi kode (sqlc, oapi-codegen) tidak menghasilkan diff (`06-ci-cd.md`).
- **Python** tidak dipakai di service. Python hanya untuk analisis offline (notebook kalibrasi bobot ER, prior revenue, model ML kelak). Hasilnya di-commit sebagai file konfigurasi berversi yang dibaca service Go.

| Kebutuhan | Pustaka/komponen |
|---|---|
| HTTP & kontrak | `net/http` (routing 1.22+), `oapi-codegen` strict-server, middleware stdlib |
| Database | `pgx` v5 (pool, `CopyFrom`), `sqlc`, `goose` |
| Job & event | `River` (driver pgx v5) |
| Normalisasi | `nyaruka/phonenumbers` (port libphonenumber, region `ID`), `golang.org/x/net/publicsuffix`, `golang.org/x/text` (normalisasi Unicode NFKC + folding diakritik), `net/mail` |
| Upload | `encoding/csv`, `excelize` (streaming rows) |
| Snapshot & arsip | `parquet-go/parquet-go` (baca/tulis Parquet), `pgx.CopyFrom` untuk bulk load |
| Matching | `adrg/strutil` (Jaro-Winkler) + `similarity()` pg_trgm di SQL |
| DNS | `miekg/dns` (MX/A/TXT dengan TTL, deteksi null MX) ke resolver Unbound |
| Observability | `log/slog` (JSON), OpenTelemetry Go SDK, SDK Sentry Go → GlitchTip |
| ID | `google/uuid` (UUIDv7) + encoder prefix base32 |
| AI | SDK Go resmi provider di balik interface `LLMProvider` (D-15) |
| Auth | Klien Go Ory Kratos bila D-20 memilih Kratos (Kratos ditulis dalam Go, sehingga operasional dan debugging satu ekosistem); adapter OIDC (`coreos/go-oidc`) untuk opsi lain |

### 3.2 Tanggung jawab modul

Nama schema Postgres sama dengan nama modul (`intelligence` → schema `intel`). Hanya modul pemilik yang boleh menulis ke tabelnya.

| Modul | Tanggung jawab | Tabel milik (schema) | Interface publik (contoh) | MVP Pack §11 |
|---|---|---|---|---|
| `platform` | AuthN/sesi, workspace & membership, RBAC, konteks tenant (RLS), audit, feature flag, event, helper job/DAG, idempotensi, HTTP keluar aman (netguard), ID, error, pagination | `platform`: workspace, app_user, membership, invitation, session, api_key (V1), idempotency_record, job (resource job untuk user), stage_run, event, feature_flag, audit_log, audit_log_pii, support_access_grant; schema `river` (tabel River) | `rbac.Require(perm)`, `tenancy.WithTx(ctx, ws, fn)`, `audit.Record(ctx, tx, …)`, `events.Emit(ctx, tx, evt)`, `flags.Enabled(ctx, key)`, `jobs.InsertTx(ctx, tx, args)`, `netguard.Client(role)` | lintas modul |
| `ingest` | Source Registry, snapshot lifecycle, mapping spec, load observation (`CopyFrom`), laporan validasi, parsing upload | `ingest`: source, dataset_snapshot, mapping_spec, observation, validation_report, pipeline_run, upload, upload_row | `RegisterSnapshot`, `IterObservations(ctx, snapshot, batch)`, `UploadRows` | (baru; upstream contract) |
| `identity` | Normalisasi, ER/dedupe, verifikasi, **klasifikasi industri** (kategori = atribut kanonik "industry", MVP Pack §04), claim builder, evidence, merge review, redirect, suppression, linking upload | `identity`: company, establishment, person, contact_point, evidence, establishment_class, entity_source_link, entity_redirect, merge_candidate, merge_decision, high_value_entity, dns_lookup, suppression, upload_row_link | `GetEstablishment`, `ResolveID`, `ClaimsFor`, `LinkUploadRows`, `MarkHighValue`, `IsSuppressed` | identity-service (+ klasifikasi dari intelligence-service) |
| `intelligence` | Digital presence lite, commercial (BR-REV), faktor dasar, gap fact, Offering, ICP, scoring, opportunity (on read), feedback | `intel`: digital_presence, technology, commercial_profile, establishment_factor, gap_fact, field_fill_rate, score_model, offering, icp, icp_version, score, score_context, score_job, icp_area_metric, feedback | `ScoreICP`, `ExplainScore`, `OpportunitiesFor`, `ICPCandidateFilter` | intelligence-service |
| `geo` | Referensi wilayah, crosswalk, penetapan area, metrik area global, fungsi formula, tile | `geo`: area_set, area, area_boundary, area_population, area_crosswalk, establishment_area, area_metric | `Assign`, `AreaMetrics`, `MarketOpportunity` (fungsi murni) | intelligence-service (geo) |
| `query` | Projection `prospect_search`, search/filter/sort/count, perakitan profil + masking, saved search, private prospect | `query`: prospect_search, private_prospect_search, saved_search | `Search`, `GetProfile`, `Count` | query-service |
| `action` | List, segment, tag, catatan, assignment, status kerja, export, webhook (endpoint + delivery), activity, orkestrasi refresh | `action`: list, list_item, segment, tag, prospect_tag, note, assignment, prospect_status, export_job, distribution_log, webhook_endpoint, webhook_delivery, activity | `Export`, `SendToWebhook`, `RecordActivity` | action-service |
| `ai` | Port provider, prompt registry, input builder, grounding validator, cache, biaya | `ai`: ai_artifact, ai_call | `GenerateBrief` | intelligence-service (AI) |
| `usage` | Plan, entitlement, fair-use, usage event, ledger, hold/capture, unlock, refund | `usage`: plan, plan_entitlement, subscription, credit_lot, credit_ledger, credit_balance, usage_event, unlock, profile_view | `Hold`, `Capture`, `Release`, `UnlockCompanies`, `CheckEntitlement` | usage-service |
| `ops` | API konsol internal: data/pipeline/intelligence/cost health, DLQ (job River `discarded`), merge review, admin tenant, support access, DSR, laporan data | `ops`: data_issue_report, data_subject_request, metric_snapshot | (konsumen saja) | ops-console |

### 3.3 Aturan dependensi

```
internal/app                     (composition root: boleh impor semua)
internal/ops
internal/action | internal/ai    (saudara independen: tidak saling impor)
internal/query
internal/intelligence
internal/geo
internal/identity
internal/ingest
internal/usage
internal/platform                (paling bawah)
```

1. Modul hanya boleh mengimpor modul yang berada **di bawahnya**, dan hanya paket akar modul (`service.go`). `internal/<modul>/internal/...` tidak dapat diimpor dari luar modul karena aturan compiler Go.
2. Kebutuhan dari bawah ke atas (mis. `identity` perlu memberi tahu `action` agar me-rekey list setelah merge) dipenuhi lewat **event** (`events.Emit` → job River), bukan import.
3. Stage pipeline lintas modul tidak saling memanggil. Setiap modul mendaftarkan worker stage-nya di `jobs.go`, lalu DAG disusun di `internal/app/dag.go`.
4. Query berat lintas modul (builder projection) membaca **view read-only yang dipublikasikan** modul pemilik (`identity.v_establishment_public`, …), bukan tabel internalnya. Paket sqlc `internal/db/<modul>db` hanya boleh diimpor oleh modul pemilik.

```yaml
# .golangci.yml (cuplikan) — aturan per modul dibangkitkan tools/gen-depguard dari tabel layer di atas
linters:
  enable: [depguard, gosec, staticcheck, errcheck, govet, revive]
linters-settings:
  depguard:
    rules:
      query-layer:
        files: ["**/internal/query/**"]
        deny:
          - { pkg: "<mod>/internal/action", desc: "query berada di bawah action (§3.3)" }
          - { pkg: "<mod>/internal/ai",     desc: "query berada di bawah ai (§3.3)" }
          - { pkg: "<mod>/internal/ops",    desc: "ops adalah lapisan teratas" }
          - { pkg: "<mod>/internal/db/inteldb", desc: "tabel intel hanya via intelligence.Service" }
      platform-bottom:
        files: ["**/internal/platform/**"]
        deny:
          - { pkg: "<mod>/internal/identity", desc: "platform tidak boleh bergantung pada modul domain" }
```

---

## 4. Input Contract (menutup G-01)

Memenuhi FR-DATA-001, FR-DATA-008, FR-DATA-012, BR-ER-07, BR-PRIV-04, dan PRD §11. Artefak kontrak disimpan di `contracts/input/` (JSON Schema 2020-12) dan berversi SemVer (`observation-record/1.0.0`).

```
  Source adapter (DATASET | UPLOAD | PARTNER | CRAWLER[paused])
          |  menghasilkan ObservationRecord (JSONL/Parquet) + manifest
          v
  [Source Registry] -- policy lisensi --> [Dataset snapshot ds-YYYY.MM.n] -- validasi --> [observation (partisi/snapshot)]
                                                                                              |
                                                                                              v
                                                                            pipeline (§6) -> entity kanonik + evidence
```

### 4.1 Source Registry

Setiap sumber terdaftar sebelum datanya boleh masuk. Tabel `ingest.source`: `id (src_)`, `code` unik, `name`, `type ∈ {DATASET, UPLOAD, PARTNER, CRAWLER}`, `status ∈ {ACTIVE, PAUSED, RETIRED}` (CRAWLER selalu `PAUSED` selama D-01), `license_tags text[]`, `policy jsonb`, `trust jsonb`, `contract_version`, `owner`.

```yaml
# Disimpan di ingest.source.policy/trust; diubah hanya oleh ops (tercatat di audit)
source: biz_dataset_a
type: DATASET
status: ACTIVE
contract_version: observation-record/1.0.0
license_tags: [internal_processing, display_allowed, derived_export_allowed, attribution_required]
policy:
  retention: { raw_snapshot_days: 730, observation_online_snapshots: 2 }
  display:   { fields_allowed: "*", attribution: "Sumber: Dataset A (snapshot {version})" }
  export:
    allowed: true
    fields_denied: [raw.*]            # default konservatif sampai Q-05 dijawab: hanya field kanonik/derived
    attribution_column: true
  pii_fields: [person.name, person.email, person.phone]
trust:   # confidence dasar per field (0–100), dikalibrasi QA per snapshot
  name: 90
  geo: 90
  phone: 80
  website: 80
  category: 75
  email: 70
  business_status: 70
```

| Tag lisensi | Efek di sistem |
|---|---|
| `internal_processing` | Boleh diproses pipeline. Tanpa tag ini snapshot tidak bisa di-publish |
| `display_allowed` / `display_restricted` | Projection/profil menampilkan field sesuai `display.fields_allowed` |
| `derived_export_allowed` / `raw_export_allowed` | Export builder (§10, FR-EXP-001) memfilter kolom berdasarkan `export.fields_denied` |
| `attribution_required` | Atribusi tampil di evidence drawer dan kolom export |
| `retention_limited` | Job retensi menghapus file mentah/partisi observation setelah `retention.*` |

Guardrail Rev 1.2 §8 (data turunan Google Places) dan BR-PRIV-04 diterapkan **hanya** melalui policy ini, sehingga perubahan ketentuan tidak menyentuh kode engine.

### 4.2 Siklus hidup dataset snapshot (G-19)

```
 register (ops)        job validate           ops approve + pipeline_run selesai          snapshot berikutnya publish
STAGED ---------> VALIDATING ---------> VALIDATED -------------------------------> PUBLISHED --------------------------> SUPERSEDED
                      |                     |
                      +----> REJECTED <-----+   (gate gagal tanpa override / ditolak ops)
```

| State | Arti | Aksi sistem |
|---|---|---|
| `STAGED` | Manifest terdaftar. File diunggah ke `msp-snapshots/{source}/{version}/` (object lock/WORM, sha256 per file) | Versi `ds-YYYY.MM.n` dialokasikan (unik per source; `n` = urutan dalam bulan) |
| `VALIDATING` | Parse + mapping + validasi berjalan | Mengisi partisi `ingest.observation` baru dan `validation_report` |
| `VALIDATED` | Laporan siap; gate dievaluasi | Ops meninjau. Override gate wajib alasan (audit). Pipeline (§6) dijalankan pada state ini |
| `PUBLISHED` | `pipeline_run` sukses sampai swap projection | `current_snapshot` untuk source diperbarui → event `snapshot.published` → re-score fan-out (§7.9) |
| `SUPERSEDED` | Snapshot lebih baru sudah publish | Partisi observation dipertahankan sesuai `observation_online_snapshots`, lalu di-detach dan diarsip ke MinIO (Parquet) |
| `REJECTED` | Ditolak | Partisi di-drop. File mentah disimpan untuk audit sesuai retensi |

Gate default (dapat dikonfigurasi per source):

| Gate | Ambang |
|---|---|
| Baris gagal parse | ≤ 1% |
| Baris reject (semua alasan) | ≤ 5% |
| Field wajib (`source_record_id`, `name`, koordinat **atau** alamat) | ≥ 98% terisi |
| Koordinat di dalam bounding box Indonesia | ≥ 99% dari koordinat yang ada |
| `source_record_id` duplikat dalam snapshot | duplikat ke-2 dst. di-reject (`DUP_SOURCE_RECORD_ID`) |
| Selisih jumlah baris vs snapshot sebelumnya | ±20% (di luar itu wajib konfirmasi ops) |

Snapshot bersifat **immutable**: koreksi data = snapshot baru (`ds-2026.10.2`), tidak pernah mengedit snapshot lama (FR-DATA-001).

### 4.3 Skema `ObservationRecord`

Satuan kontrak adalah **observation per field**. Transport dan penyimpanan dikelompokkan per **record sumber** (envelope) agar jumlah baris tetap terkendali (ADR-24). Field evidence pointer diadopsi dari Crawler Spec §11.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://contracts.mesinprospek.id/input/observation-record/1.0.0.json",
  "title": "ObservationRecord",
  "type": "object",
  "required": ["source_id", "snapshot_id", "source_record_id", "observed_at", "country", "observations"],
  "properties": {
    "source_id":        { "type": "string", "pattern": "^src_[0-9a-z]{26}$" },
    "snapshot_id":      { "type": "string", "pattern": "^ds-[0-9]{4}\\.[0-9]{2}\\.[0-9]+$" },
    "source_record_id": { "type": "string", "minLength": 1, "maxLength": 256 },
    "entity_hint": {
      "type": "object",
      "properties": {
        "kind": { "enum": ["establishment", "company", "person"] },
        "parent_source_record_id": { "type": "string" },
        "brand_key": { "type": "string" }
      }
    },
    "observed_at": { "type": "string", "format": "date-time" },
    "country":     { "const": "ID" },
    "locale":      { "type": "string", "default": "id-ID" },
    "record_hash": { "type": "string", "pattern": "^sha256:[0-9a-f]{64}$" },
    "observations": { "type": "array", "minItems": 1, "items": { "$ref": "#/$defs/Observation" } }
  },
  "$defs": {
    "Observation": {
      "type": "object",
      "required": ["field", "raw_value"],
      "properties": {
        "field":       { "type": "string", "pattern": "^[a-z_]+(\\.[a-z_]+)*(\\[[0-9]+\\])?$" },
        "raw_value":   { "type": ["string", "number", "boolean", "array", "object", "null"] },
        "observed_at": { "type": "string", "format": "date-time" },
        "extraction_confidence": { "type": "number", "minimum": 0, "maximum": 1, "default": 1 },
        "evidence":    { "$ref": "#/$defs/EvidencePointer" }
      }
    },
    "EvidencePointer": {
      "type": "object",
      "required": ["kind"],
      "properties": {
        "kind":              { "enum": ["DATASET_ROW", "UPLOAD_ROW", "PARTNER_RECORD", "CRAWL_PAGE"] },
        "uri":               { "type": "string" },
        "row":               { "type": "integer", "minimum": 0 },
        "source_url":        { "type": "string" },
        "final_url":         { "type": "string" },
        "selector_hint":     { "type": "string" },
        "snippet_hash":      { "type": "string" },
        "captured_at":       { "type": "string", "format": "date-time" },
        "extractor_version": { "type": "string" }
      }
    }
  }
}
```

- `extraction_confidence` = keyakinan ekstraktor (Crawler §11), **bukan** confidence sales. Untuk dataset default 1,0. Confidence claim = `trust[field] × extraction_confidence` (§5.6).
- `record_hash` = sha256 atas observation kanonik yang diurutkan. Dipakai untuk change detection antar snapshot (Crawler §14.2): record dengan hash sama dilewati di stage berat (FR-DATA-013, V1).
- Pointer `CRAWL_PAGE` (`source_url`, `final_url`, `selector_hint`, `snippet_hash`) sudah ada di kontrak agar crawler (V2) tidak memerlukan perubahan skema.

#### 4.3.1 Katalog field kanonik

| Field | Tipe mentah | Dipakai oleh | Bila tidak ada di dataset |
|---|---|---|---|
| `name` | string | Identitas, ER, search | Record di-reject (`MISSING_REQUIRED:name`) |
| `category.source` | string / array | Klasifikasi (FR-INTL-001) | Kategori "Lainnya", confidence rendah |
| `address.full`, `address.{province,city,district,village,postal_code}` | string | Tampilan, geo-assign (FR-DATA-009), ER | Geo hanya dari koordinat |
| `geo.lat`, `geo.lng` | number | ER (jarak), geo-assign (PIP), peta | Geo dari alamat; ER tanpa fitur jarak |
| `phone[]` | string | Contact point, ER, WA-likely | Contactability turun (tetap diketahui = 0) |
| `website` | string | Digital lite, ER (domain), OPP-WEB-* | Faktor Digital = Unknown bila fill-rate < 20% |
| `email[]` | string | Contact point, OPP-EMAIL-NON-DOMAIN | OPP-EMAIL inactive |
| `social.{instagram,facebook,tiktok,…}` | url | Digital lite, contactability ("hanya profil sosial") | — |
| `business_status` | enum string | BR-SCORE-07, metrik area | `UNKNOWN` (tidak dikecualikan) |
| `hours` | object / string | OPP-LIST-INCOMPLETE | Dihitung sebagai kosong |
| `rating`, `review_count` | number | Commercial, OPP-REP-*, aktivitas area | OPP-REP inactive; aktivitas area Unknown |
| `price_level` | number | BR-REV-03 | Pengali tidak dipakai |
| `employee_count` / `employee_range` | number / string | BR-REV-01 | Band karyawan INFERRED dari prior |
| `person[].{name,role,email,phone}` | object | Contact intelligence | Coverage person rendah (R-01) |
| `brand` | string | Pengelompokan company (ER) | Pengelompokan via domain/telepon |
| `has_online_booking` | boolean | OPP-BOOKING-NONE | Tipe inactive (Lampiran B) |

### 4.4 Mapping spec (dataset → observation)

Mapping spec berversi dan disimpan di `ingest.mapping_spec`. Versinya ikut menentukan versi stage `load` (§6.2).

```yaml
mapping: biz_dataset_a
version: 1.2.0
contract: observation-record/1.0.0
input:
  format: parquet                       # parquet | csv | jsonl
  encoding: utf-8
  record_id: place_id                   # -> source_record_id
  observed_at: { column: scraped_at, format: "%Y-%m-%d %H:%M:%S", tz: Asia/Jakarta }
fields:
  name:                 { column: title }
  category.source:      { column: categories, split: "," }
  address.full:         { column: address }
  address.village:      { column: kelurahan }
  address.district:     { column: kecamatan }
  address.city:         { column: kota }
  address.province:     { column: provinsi }
  address.postal_code:  { column: kode_pos }
  geo.lat:              { column: latitude,  type: float, range: [-11.5, 6.5] }
  geo.lng:              { column: longitude, type: float, range: [94.5, 141.5] }
  phone:                { column: phone, split: "[/;,]|\\batau\\b", max_items: 3 }
  website:              { column: website }
  email:                { columns: [email_1, email_2] }
  business_status:      { column: status, map: { OPERATIONAL: OPERATIONAL, CLOSED_TEMPORARILY: CLOSED_TEMPORARILY, CLOSED_PERMANENTLY: CLOSED_PERMANENTLY } }
  rating:               { column: rating, type: float, range: [1, 5] }
  review_count:         { column: reviews, type: int, min: 0 }
  hours:                { column: opening_hours, parse: json }
  brand:                { column: brand }
required: [source_record_id, name]
require_any: [[geo.lat, geo.lng], [address.full]]
null_tokens: ["", "-", "n/a", "null", "none", "tidak ada"]
```

### 4.5 Laporan validasi

Dihasilkan saat `VALIDATING` dan disimpan di `ingest.validation_report` (JSONB) + MinIO. Ditampilkan di ops console (FR-DATA-001, FR-OPS-001).

```json
{
  "snapshot": "ds-2026.10.1", "source": "biz_dataset_a", "mapping_version": "1.2.0",
  "rows": { "received": 5104332, "parsed": 5103990, "valid": 5021877, "rejected": 82113 },
  "rejects_by_reason": { "INVALID_COORD": 40211, "MISSING_REQUIRED:name": 1203, "OUT_OF_COUNTRY": 3555,
                         "DUP_SOURCE_RECORD_ID": 3420, "UNPARSEABLE_ROW": 342, "FIELD_TYPE:rating": 33382 },
  "fill_rate": {
    "overall": { "phone": 0.81, "website": 0.34, "email": 0.07, "hours": 0.62, "rating": 0.71 },
    "by_segment": [ { "category_source": "Dentist", "city": "Kota Denpasar", "n": 412,
                      "fill": { "website": 0.68, "email": 0.11, "phone": 0.93 } } ]
  },
  "delta_vs_previous": { "previous": "ds-2026.07.1", "new": 120334, "gone": 98110, "changed": 803221, "unchanged": 4098322 },
  "gates": [ { "gate": "reject_rate", "threshold": 0.05, "value": 0.016, "pass": true } ],
  "generated_at": "2026-10-09T03:12:44Z"
}
```

- Field-level error tidak memblokir record: field bermasalah ditandai, record tetap berjalan (MVP Pack §05). Record di-reject hanya bila field wajib gagal.
- **Fill-rate BR-OPP-02** dihitung ulang setelah klasifikasi dan geo-assign per **kategori MSP × kab/kota (kode Kemendagri)**, juga per kategori × provinsi dan kategori × nasional (fallback segmen < 20 establishment), lalu disimpan di `intel.field_fill_rate(snapshot_id, field, category_code, scope_level, scope_code, n, filled, rate)`. Gap engine membaca tabel ini (§8.2).

Katalog alasan reject: `MISSING_REQUIRED:<field>`, `INVALID_COORD`, `OUT_OF_COUNTRY`, `UNPARSEABLE_ROW`, `DUP_SOURCE_RECORD_ID`, `FIELD_TYPE:<field>`, `FIELD_RANGE:<field>`, `ENCODING_ERROR`, `SUPPRESSED_VALUE` (nilai cocok dengan suppression; field dibuang, record tetap jalan).

### 4.6 Upload customer (FR-DATA-012, BR-ER-07, ADR-23)

```
PENDING_UPLOAD -> UPLOADED -> PARSING -> MAPPING_REQUIRED -> PREVIEW_READY -> PROCESSING -> COMPLETED
       |              |           |                                                |
       +--------------+-----------+------------------> FAILED / CANCELLED <--------+
```

1. `POST /v1/uploads` (nama file, ukuran, tipe) → cek entitlement dan flag `upload.create` → baris `ingest.upload` + **presigned PUT URL** MinIO (`msp-uploads/{workspace_id}/{upload_id}/source.<ext>`, berlaku 15 menit, `Content-Length` ≤ 20 MB).
2. Browser mengunggah langsung ke `files.<domain>` (Traefik → MinIO). `POST /v1/uploads/{id}/complete` memicu job `PARSING`.
3. Parsing di worker antrean `maintenance`:
   - CSV (`encoding/csv` + `x/text/encoding/charmap`): deteksi encoding (UTF-8/UTF-8-BOM/Windows-1252), sniff delimiter (`,` `;` tab), baris header wajib.
   - XLSX (`excelize`, iterator `Rows()` streaming): hanya `.xlsx` (bukan `.xlsm`/`.xls`), nilai formula dibaca dari cache, sheet pertama atau sheet yang dipilih.
   - Penjaga zip bomb: total uncompressed ≤ 200 MB, ≤ 1.000 entry. Batas ≤ 50.000 baris data dan ≤ 100 kolom.
   - Hasilnya header + 20 baris sampel + **saran mapping** (kamus sinonim header: "nama usaha"→`name`, "no hp/telp/wa"→`phone`, "alamat"→`address.full`, "web/situs"→`website`, "kota/kab"→`address.city`).
4. `PUT /v1/uploads/{id}/mapping` → preview validasi 1.000 baris pertama (valid/reject per alasan). Minimum: `name` + (`website` atau `phone` atau `address.full`) (PRD §11).
5. `POST /v1/uploads/{id}/process` (async): normalisasi (fungsi BR-NORM yang sama) → verifikasi format/DNS → **linking** (`identity.linking`, fitur ER §6.4). Rangkaian ini adalah DAG upload yang disusun di `internal/app/dag.go` (`ingest.parse_upload` → `identity.normalize_link` → `intel.score_private` → `query.project_private`), sehingga `ingest` tidak mengimpor modul di atasnya:

| Skor link | Status | Perilaku |
|---|---|---|
| ≥ 0,92 dan tidak ada identifier kuat yang bertentangan | `MATCHED` | Di-link otomatis ke establishment/company global |
| 0,80–0,92 | `POSSIBLE` | Ditampilkan sebagai "Kemungkinan cocok". User mengonfirmasi atau menolak (`POST …/rows/{row}/link`). Default: tidak di-link |
| < 0,80 | `NEW` | Menjadi **private prospect** (`query.private_prospect_search`, RLS) |
| — | `REJECTED` | Baris tidak valid, beserta alasan |

6. Laporan (`GET /v1/uploads/{id}/report`): matched/possible/new/rejected, match rate, dan CSV baris reject yang dapat diunduh.

Privasi dan aturan:
- `ingest.upload`, `ingest.upload_row`, dan `identity.upload_row_link` adalah tabel **tenant layer** (RLS). Data upload **tidak pernah** menulis ke global layer (BR-ER-07).
- Contact point dari upload milik workspace itu sendiri **tidak di-mask dan tidak dikenai credit**. Kontak ini tampil di profil sebagai claim OBSERVED dengan evidence "Upload: `<nama file>` baris N" (FR-PROF-004).
- Private prospect diproses engine yang sama (klasifikasi, geo, skor untuk ICP aktif) dalam konteks workspace. Gap fact untuk private prospect dihitung saat proses upload dan disimpan di `private_prospect_search.gap_types`.
- File mentah dihapus 30 hari setelah `COMPLETED` (dapat dikonfigurasi). Baris hasil parse bertahan sampai upload dihapus user atau workspace ditutup (FR-AUD-005). Suppression global tidak otomatis menghapus data upload tenant (tenant = pengendali). Permintaan subjek data atas data upload diteruskan ke tenant (BR-PRIV-02, §15.8).

### 4.7 Data referensi wilayah & crosswalk (FR-DATA-008, D-13, G-08)

| Dataset | Kunci | Isi | Versi |
|---|---|---|---|
| Kode & nama wilayah Kemendagri | `code` format `PP.KK.CC.DDDD` (contoh `51.71.01.1001`) | level (1 provinsi … 4 kelurahan/desa), nama, jenis (KOTA/KABUPATEN, KELURAHAN/DESA), parent | `area_set_version`, mis. `kmd-2025` |
| Batas wilayah (opsional) | `code` | `geometry(MultiPolygon, 4326)` + luas km² | Ikut `area_set_version` + sumber |
| Populasi per kelurahan/desa | `code` | populasi, luas, `reference_period` (mis. `2025-S1`), sumber | `population_version`, mis. `pop-2025S1` |
| Crosswalk BPS ↔ Kemendagri | (`bps_code`, `kmd_code`) | metode (`EXACT_NAME_PARENT`, `FUZZY_NAME`, `MANUAL`), confidence, kardinalitas | `crosswalk_version` |

Proses import (ops, `POST /v1/ops/reference/*`):
1. Validasi format kode, hierarki parent, dan keunikan. Kode lama yang hilang ditandai `valid_to` (pemekaran/penggabungan), bukan dihapus.
2. Geometri divalidasi (`ST_IsValid`, perbaikan `ST_MakeValid`), dipecah dengan `ST_Subdivide` untuk index PIP, lalu disederhanakan *topology-aware* secara offline untuk tile (§9.5).
3. Populasi dengan kode BPS dipetakan lewat crosswalk. Relasi 1→N (satu kode BPS ke beberapa kode Kemendagri) **tidak** dibagi proporsional: metrik disimpan di level parent terdekat yang konsisten, dan coverage area anak ditandai `LOW` (A-02, fallback kecamatan).
4. Setiap baris populasi wajib memiliki `reference_period` dan `source` (acceptance FR-DATA-008).
5. `area_set_version` baru memicu geo-assign ulang dan metrik area baru (`geo_run`). Versi lama dipertahankan untuk reproducibility.

### 4.8 Adapter sumber lain

Adapter mengimplementasikan port `SourceAdapter` (§19.1). Di MVP ada dua adapter: `DatasetFileAdapter` (MVP0) dan `UploadAdapter` (MVP1, tenant-scoped, keluarannya tidak masuk global). `PartnerApiAdapter` (V1+) dan `CrawlerAdapter` (V2, PAUSED) memakai kontrak yang sama.

---

## 5. Data model

Memenuhi D-03, D-05, BR-TRUST, MVP-01..03, FR-AUD-006, dan NFR-DATA-01.

### 5.1 Lapisan, schema, konvensi

| Schema | Layer | Pemilik | Isi |
|---|---|---|---|
| `ingest` | Global (+ tenant: upload, upload_row) | ingest | Source, snapshot, observation, laporan |
| `identity` | Global (+ tenant: upload_row_link) | identity | Entity kanonik, evidence, ER, suppression |
| `intel` | Global (class, factor, gap) + tenant (offering, icp, score, feedback) | intelligence | Intelligence |
| `geo` | Global | geo | Wilayah, metrik area |
| `query` | Global (`prospect_search`) + tenant (`private_prospect_search`, `saved_search`) | query | Read model |
| `action`, `usage`, `ai` | Tenant | modul masing-masing | Workflow, billing, AI |
| `platform` | Campuran | platform | Workspace, user, audit, event, flag, job (+ schema `river` untuk antrean) |
| `ops` | Internal (sebagian ber-`workspace_id`) | ops | DSR, laporan data, metrik |

Konvensi:
- **ID:** `uuid` (UUIDv7, dibangkitkan aplikasi). Di API diekspos sebagai `prefix_` + 26 karakter base32 Crockford (ADR-14). Prefix: `wsp usr mem inv cmp est per cpt evd src snp ofr icp lst seg tag nte asg upl exp whe whd evt job brf use led unl aud dsr rpt mrg`. Kode wilayah memakai natural key Kemendagri.
- **Waktu:** `timestamptz` dalam UTC (NFR-L10N-01). Tanggal referensi data disimpan sebagai teks periode (`2025-S1`).
- **Enum stabil** memakai PostgreSQL `ENUM`. Daftar yang sering berubah (alasan reject, kode opportunity) memakai `text` + validasi aplikasi/katalog.
- **Tenant table:** kolom pertama `workspace_id uuid NOT NULL`, RLS `ENABLE` + `FORCE`, dan setiap index diawali `workspace_id`.
- **Tidak ada hard delete pada global layer** kecuali lewat job retensi atau DSR. Entity yang di-merge → `status = MERGED` + redirect.

### 5.2 ERD — global layer

```
ingest.source 1--< ingest.dataset_snapshot 1--< ingest.observation (partisi LIST per snapshot)
                                                       |
                       identity.entity_source_link >---+---< identity.evidence (DATASET_RECORD, RULE, MODEL, DNS_LOOKUP, OPS_CORRECTION)
                                |                                   ^  (dirujuk lewat claim_meta.*.ev)
                                v                                   |
identity.company 1--< identity.establishment 1--< identity.contact_point >--0..1 identity.person >-- company
      |                      |  1--1 geo.establishment_area >--1 geo.area 1--< geo.area_metric (snapshot × geo_version × kategori)
      |                      |  1--1 identity.establishment_class / intel.digital_presence / intel.establishment_factor
      |                      |  1--< intel.gap_fact             1--< intel.technology
      |                      |  1--1 query.prospect_search (projection)
      1--1 intel.commercial_profile
identity.merge_candidate, identity.merge_decision, identity.entity_redirect, identity.suppression, identity.dns_lookup
geo.area_set 1--< geo.area 1--0..1 geo.area_boundary ; geo.area 1--< geo.area_population ; geo.area_crosswalk
```

### 5.3 ERD — tenant layer

```
platform.workspace 1--< platform.membership >--1 platform.app_user (global, data akun)
     |-- platform.invitation, platform.session, platform.api_key (V1), platform.audit_log, platform.event
     |-- intel.offering 1--< intel.icp 1--< intel.icp_version
     |                         |--< intel.score (epoch, partisi HASH workspace) >-- identity.establishment
     |                         |--1 intel.score_context ; --< intel.score_job ; --< intel.icp_area_metric >-- geo.area
     |-- action.list 1--< action.list_item >-- identity.company
     |-- action.segment, query.saved_search, action.tag 1--< action.prospect_tag >-- company
     |-- action.note, action.assignment, action.prospect_status, action.activity (partisi bulanan)
     |-- ingest.upload 1--< ingest.upload_row 1--0..1 identity.upload_row_link >-- establishment/company
     |                                     \--0..1 query.private_prospect_search
     |-- action.export_job ; action.webhook_endpoint 1--< action.webhook_delivery
     |-- usage.subscription, usage.credit_lot, usage.credit_ledger, usage.credit_balance, usage.usage_event,
     |   usage.unlock >-- company, usage.profile_view
     |-- ai.ai_artifact 1--< ai.ai_call ; intel.feedback ; ops.data_issue_report
```

### 5.4 Tabel global utama

```sql
CREATE TABLE identity.company (
  id                 uuid PRIMARY KEY,                       -- cmp_
  status             identity.entity_status NOT NULL DEFAULT 'ACTIVE',   -- ACTIVE | MERGED | REMOVED
  name               text NOT NULL,
  name_norm          text NOT NULL,
  brand              text,
  legal_form         text,                                   -- PT, CV, UD, Tbk, Persero, Koperasi, ...
  registrable_domain text,                                   -- domain milik sendiri (bukan platform)
  establishment_count integer NOT NULL DEFAULT 1,
  claim_meta         jsonb NOT NULL DEFAULT '{}',
  entity_seq         bigint NOT NULL,                        -- naik pada setiap perubahan kanonik
  created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL
);
CREATE INDEX ON identity.company (registrable_domain) WHERE registrable_domain IS NOT NULL;

CREATE TABLE identity.establishment (
  id               uuid PRIMARY KEY,                         -- est_
  company_id       uuid NOT NULL REFERENCES identity.company(id),
  status           identity.entity_status NOT NULL DEFAULT 'ACTIVE',
  name             text NOT NULL,
  name_norm        text NOT NULL,
  aliases          text[] NOT NULL DEFAULT '{}',
  address_display  text,
  address_norm     text,
  address_parts    jsonb,                                    -- province/city/district/village/postal mentah ternormalisasi
  location         geography(Point, 4326),
  geohash7         text,                                     -- kunci blocking ER (~150 m)
  business_status  identity.business_status NOT NULL DEFAULT 'UNKNOWN',
  rating           numeric(2,1),
  review_count     integer,
  price_level      smallint,
  opening_hours    jsonb,
  claim_meta       jsonb NOT NULL DEFAULT '{}',
  entity_seq       bigint NOT NULL,
  first_seen_snapshot_id uuid, last_seen_snapshot_id uuid,
  suppressed       boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL
);
CREATE INDEX ON identity.establishment USING gist (location);
CREATE INDEX ON identity.establishment (geohash7);
CREATE INDEX ON identity.establishment USING gin (name_norm gin_trgm_ops);
CREATE INDEX ON identity.establishment (company_id);

CREATE TABLE identity.contact_point (
  id                 uuid PRIMARY KEY,                       -- cpt_
  company_id         uuid NOT NULL,
  establishment_id   uuid,                                   -- NULL = level company
  person_id          uuid,                                   -- NULL = bukan milik person
  type               identity.contact_type NOT NULL,         -- PHONE | EMAIL | WHATSAPP | SOCIAL | OTHER
  value_display      text NOT NULL,                          -- untuk tampilan (setelah unlock)
  value_norm         text NOT NULL,                          -- E.164 / email (domain lowercase) / URL kanonik
  value_hash         bytea NOT NULL,                         -- HMAC-SHA256(pepper, value_norm): dedupe & suppression
  phone_type         identity.phone_type,                    -- MOBILE | FIXED_LINE | FIXED_OR_MOBILE | TOLL_FREE | UAN | SHORT_CODE | VOIP | UNKNOWN
  is_shared          boolean NOT NULL DEFAULT false,         -- nomor bersama/call center (BR-ER-01)
  wa_likely          boolean NOT NULL DEFAULT false,         -- INFERRED (BR-VER-01)
  email_kind         identity.email_kind,                    -- OWN_DOMAIN | WEBMAIL | DISPOSABLE | UNKNOWN
  is_role_based      boolean NOT NULL DEFAULT false,
  verification_status identity.verification_status NOT NULL DEFAULT 'UNKNOWN',
  verification_confidence smallint,
  verified_at        timestamptz,
  verification_method text,                                  -- 'syntax+mx@1', 'libphonenumber@8.x'
  claim_meta         jsonb NOT NULL DEFAULT '{}',
  suppressed         boolean NOT NULL DEFAULT false,
  UNIQUE (company_id, type, value_norm, establishment_id, person_id)
);
CREATE INDEX ON identity.contact_point (value_hash);

CREATE TABLE identity.evidence (
  id                uuid PRIMARY KEY,                        -- evd_
  kind              identity.evidence_kind NOT NULL,         -- DATASET_RECORD | RULE | MODEL | DNS_LOOKUP | OPS_CORRECTION | CRAWL_PAGE
  source_id         uuid,
  snapshot_id       uuid,
  observation_id    uuid,                                    -- record observasi (kind = DATASET_RECORD)
  source_record_ref text,
  fields            text[] NOT NULL,                         -- field yang didukung
  captured_at       timestamptz NOT NULL,                    -- observed_at sumber / waktu lookup
  snippet_hash      text,                                    -- sha256 nilai mentah (Crawler §11)
  extractor_version text NOT NULL,                           -- versi mapping/rule/model/prompt
  confidence        smallint NOT NULL CHECK (confidence BETWEEN 0 AND 100),
  pointer           jsonb,                                   -- EvidencePointer (§4.3)
  detail            jsonb,                                   -- {rule:'wa_likely@1', inputs:[...]} | {dns:{mx:[...]}}
  created_at        timestamptz NOT NULL
);
CREATE INDEX ON identity.evidence (snapshot_id, observation_id);
```

| Tabel | Kolom kunci | Constraint / index | Catatan |
|---|---|---|---|
| `ingest.dataset_snapshot` | id (`snp_`), source_id, version `ds-YYYY.MM.n`, status, observed_from/to, reference_period, license_tags (salinan), manifest jsonb, report_id | UNIQUE(source_id, version) | §4.2 |
| `ingest.observation` | (snapshot_id, id), source_record_id, observed_at, record_hash, raw jsonb, entity_hint, load_status, reject_reasons | PK(snapshot_id, id), UNIQUE(snapshot_id, source_record_id); `PARTITION BY LIST (snapshot_id)` | ADR-24 |
| `identity.person` | id (`per_`), company_id, establishment_id, name, role_title, role_norm (OWNER, DIRECTOR, MANAGER, PRACTITIONER, MARKETING, OTHER), is_decision_maker, claim_meta, suppressed | idx(company_id) | P1 PII (§15.6) |
| `identity.entity_source_link` | (source_id, source_record_id) → establishment_id, company_id, first/last_snapshot_id | PK(source_id, source_record_id) | Kunci stabilitas ID (FR-DATA-007) |
| `identity.entity_redirect` | from_id PK, to_id, entity_type, reason, merge_decision_id, created_at | — | ADR-14 |
| `identity.merge_candidate` | id (`mrg_`), entity_type, a_id < b_id, score, features jsonb, er_version, high_value, status (OPEN, AUTO_MERGED, AUTO_SEPARATE, DECIDED, EXPIRED) | UNIQUE(a_id, b_id, er_version) | Queue review (FR-OPS-005) |
| `identity.merge_decision` | id, a_id, b_id, anchors jsonb (source_record ref kedua sisi), decision (MERGE, KEEP_SEPARATE, SPLIT), reason, decided_by, decided_at, active | idx(a_id), idx(b_id) | Final terhadap re-run (BR-ER-06) |
| `identity.suppression` | id, kind (PERSON, CONTACT_POINT, ESTABLISHMENT, COMPANY, VALUE_HASH), target_id, value_hash, reason, dsr_id, created_by, created_at, lifted_at, propagated_at | idx(value_hash), idx(target_id) | §15.7 |
| `identity.dns_lookup` | domain PK, mx jsonb, has_a, null_mx, spf text, status, checked_at, ttl_s | — | Cache + evidence DNS (FR-DATA-011) |
| `identity.establishment_class` | establishment_id PK, sector_code, category_code, confidence, method, taxonomy_version | idx(category_code) | FR-INTL-001 |
| `identity.high_value_entity` | establishment_id PK, reasons text[] (ICP_CANDIDATE, TENANT_ACTIVITY), last_marked_at | — | Hanya ID, tanpa data tenant (BR-ER-04) |
| `intel.digital_presence` | establishment_id PK, website_url, website_type, registrable_domain, own_domain, best_email_kind, email_provider_hint, spf_includes text[], social jsonb, has_online_booking, claim_meta, dns_checked_at | — | FR-DATA-010/011 |
| `intel.technology` | (subject_id, technology) PK, category, confidence, observed_at, evidence_id | — | MVP: dari DNS (MX/SPF) + field dataset |
| `intel.commercial_profile` | company_id PK, employee_band, employee_claim jsonb, revenue_band, revenue_confidence, basis jsonb, as_of, model_version (`rev-0.1.0`) | — | BR-REV |
| `intel.establishment_factor` | establishment_id PK, snapshot_id, dq_components jsonb, completeness_by_tag jsonb, contactability, commercial, commercial_conf, base_version | — | Faktor global (§6.9) |
| `intel.gap_fact` | id, establishment_id, company_id, type (OPP-…), family, snapshot_id, confidence, severity, evidence_ids uuid[], peer_context jsonb, catalog_version, status | idx(establishment_id), idx(type) | §8 |
| `intel.score_model` | version PK (`priority-1.2.0`), config yaml, sha256, status, activated_at | — | FR-INTL-009; versi lama tidak dihapus |
| `geo.area` | code PK, area_set_version, level 1–4, name, kind, parent_code, valid_from/to | idx(parent_code), idx(level) | — |
| `geo.area_boundary` | (code, area_set_version), geom MultiPolygon 4326, geom_sub (hasil subdivide, tabel anak), area_km2, source | GiST(geom_sub) | — |
| `geo.area_population` | (code, population_version), population, area_km2, reference_period, source | — | — |
| `geo.establishment_area` | establishment_id PK, prov/kab/kec/kel_code, method (PIP, ADDRESS_NAME, POSTAL, NEAREST), geo_confidence, area_set_version | idx(kel_code), idx(kab_code) | FR-DATA-009 |
| `geo.area_metric` | (snapshot_id, geo_version, population_version, area_code, category_code) PK; `*` = semua kategori; metrik (§9.2), peer_group, peer_n, coverage, inputs_hash | — | FR-GEO-001 |
| `query.prospect_search` | §10.1 | — | Projection |

### 5.5 Tabel tenant utama

Semua tabel di bawah memiliki `workspace_id` + RLS.

| Tabel | Kolom kunci | Catatan |
|---|---|---|
| `platform.workspace` | id, name, slug, status (ACTIVE, PENDING_DELETION, PURGED), settings jsonb (`member_can_export`, locale, timezone), plan_code, deletion_requested_at | Tidak ber-RLS; akses via membership |
| `platform.membership` | (workspace_id, user_id) PK, role (OWNER, ADMIN, MANAGER, MEMBER, VIEWER), status, joined_at | Role = enum + peta permission di kode (§15.2); tidak ada tabel `role` di MVP |
| `platform.invitation` | id, workspace_id, email, role, token_hash, expires_at (7 hari), accepted_at | FR-AUTH-003 |
| `intel.offering` | id (`ofr_`), workspace_id, name, tags text[] (katalog FR-ICP-001), primary_tag | — |
| `intel.icp` | id (`icp_`), workspace_id, name, offering_id NOT NULL, preset_code, current_version, is_default, status | Wajib tepat satu Offering (FR-ICP-002) |
| `intel.icp_version` | (icp_id, version) PK, workspace_id, criteria jsonb (§7.4), created_by, created_at | Immutable |
| `intel.score` | §7.9 | Partisi HASH(workspace_id) × 32 |
| `intel.score_context` | (workspace_id, icp_id) PK, active_epoch, icp_version, model_version, snapshot_set jsonb, scored_at, status | Dasar flag stale |
| `intel.score_job` | id, workspace_id, icp_id, target_epoch, trigger, status, total, done, started_at, finished_at | Progres UI |
| `intel.icp_area_metric` | (workspace_id, icp_id, epoch, area_code) PK, target_count, target_density, market_opportunity_score, hot_warm_count, avg_dq, territory_score | §9.4 |
| `intel.feedback` | id, workspace_id, user_id, subject_kind (OPPORTUNITY, SCORE, AI_BRIEF), subject_ref, establishment_id, verdict, reason_code, comment, version_ref | FR-INTL-010, FR-AI-007 |
| `query.saved_search` | id, workspace_id, owner_id, name, query jsonb, icp_id | FR-SRCH-005 |
| `query.private_prospect_search` | upload_row_id PK, workspace_id, kolom serupa `prospect_search` | ADR-23 |
| `action.list` / `action.list_item` | list: id, workspace_id, name, owner_id, shared. item: (list_id, company_id) PK, establishment_id, upload_row_id, added_by, added_at | FR-SEG-001 |
| `action.segment` | id, workspace_id, name, owner_id, query jsonb, icp_id, stats_cache jsonb | FR-SEG-002 |
| `action.tag` / `action.prospect_tag` | tag: id, workspace_id, name, color_token. link: (tag_id, company_id) | FR-SEG-003 |
| `action.note` | id, workspace_id, company_id, author_id, body (≤ 10 KB), created_at, edited_at | FR-SEG-004 |
| `action.assignment` | id, workspace_id, company_id or list_id, assignee_id, assigned_by, status, created_at | FR-SEG-005 |
| `action.prospect_status` | (workspace_id, company_id) PK, status (NEW, RESEARCHED, READY, CONTACTED, DISQUALIFIED), updated_by | FR-SEG-006 |
| `action.activity` | id, workspace_id, actor_id, type, company_id, establishment_id, ref, metadata, occurred_at | Partisi RANGE bulanan; north star (PRD §5) |
| `ingest.upload` / `ingest.upload_row` | upload: id, workspace_id, filename, status, mapping jsonb, counts. row: id, upload_id, row_number, raw jsonb, normalized jsonb, status, reject_reasons | §4.6 |
| `identity.upload_row_link` | upload_row_id PK, workspace_id, company_id, establishment_id, link_score, status (MATCHED, POSSIBLE, NEW, REJECTED, CONFIRMED, DECLINED), decided_by | BR-ER-07 |
| `action.export_job` | id, workspace_id, selection jsonb, format, fields, status, rows_total/done, file_key, expires_at, hold_txn_id, error | FR-EXP-001 |
| `action.distribution_log` | (workspace_id, company_id, channel EXPORT/WEBHOOK, ref_id) PK, delivered_at, file_expires_at | Dasar regenerasi export aktif dan event `suppression.applied` (FR-AUD-003) |
| `action.webhook_endpoint` | id (`whe_`), workspace_id, url, secret_ciphertext, secret_kid, events text[], status (ACTIVE, DISABLED_MANUAL, DISABLED_FAILING), last_validated_at | FR-EXP-003 |
| `action.webhook_delivery` | id (`whd_`), workspace_id, endpoint_id, event_id, attempt, status, next_attempt_at, response_code, latency_ms, error_code, hold_txn_id | Retensi 30 hari |
| `usage.*` | §14.2 | — |
| `ai.ai_artifact` / `ai.ai_call` | §13 | — |
| `platform.audit_log` | §15.5 | Partisi bulanan |
| `platform.event` | id (`evt_`), workspace_id, type, version, subject, payload, occurred_at, retained_until (30 hari) | §12.3. Dikirim lewat job River |
| `ops.data_issue_report` | id (`rpt_`), workspace_id, reporter_id, company_id, claim_ref, reason_code, comment, status, resolution | FR-DATA-014 |
| `ops.data_subject_request` | id (`dsr_`), type, status, requester (terenkripsi), subject_hashes, due_at, handled_by | Tanpa workspace (FR-AUD-004) |
| `platform.feature_flag` | key, scope (GLOBAL, PLAN, WORKSPACE), scope_id, enabled, rollout jsonb, updated_by | FR-OPS-007 |

### 5.6 Model claim (ADR-13, BR-TRUST)

Nilai pemenang disimpan di kolom bertipe. Metadata per field disimpan di `claim_meta`:

```json
{
  "name":    { "t": "OBSERVED", "c": 92, "at": "2026-09-30T00:00:00Z", "ev": ["evd_01j9…"], "src": ["src_01j8…"] },
  "website": { "t": "OBSERVED", "c": 62, "at": "2026-09-30T00:00:00Z", "ev": ["evd_01j9…", "evd_01j7…"],
               "alt": [ { "v": "abcdental.business.site", "c": 55, "ev": ["evd_01j7…"] } ] },
  "category_code": { "t": "DERIVED", "c": 95, "at": "2026-10-09T02:00:00Z", "ev": ["evd_rule…"], "ver": "taxonomy-1.0.0" }
}
```

**Claim builder** (dijalankan untuk setiap entity yang tersentuh pipeline):
1. Kumpulkan kandidat nilai dari evidence terbaru per source (snapshot berbeda dari source yang sama **tidak** dihitung independen; yang dipakai hanya yang terbaru).
2. Kelompokkan berdasarkan nilai ternormalisasi. Confidence evidence = `trust[field] × extraction_confidence × 100`.
3. Satu kelompok: `c = 1 − Π(1 − cᵢ/100)` atas source independen, maksimum 99 (BR-TRUST-02).
4. Beberapa kelompok (konflik): pemenang = argmax(`c_group × recency`), dengan `recency` = 1,0 (≤ 180 hari), 0,8 (≤ 365), 0,6 (> 365). Confidence pemenang − 20 (minimum 1). Maksimal 3 alternate disimpan di `alt`.
5. `t` (claim_type): `OBSERVED` bila dari record sumber; `DERIVED` bila dari rule deterministik atas nilai observed (registrable domain, website_type, kategori MSP); `INFERRED` bila dari prior/heuristik/model (WA-likely, band revenue/karyawan inferred, opportunity); `UNKNOWN` bila tidak ada evidence (field tidak ditulis; API mengembalikan `{ "claim_type": "UNKNOWN" }`).
6. **Koreksi ops (FR-OPS-009):** evidence `OPS_CORRECTION` (merujuk `data_issue_report`) selalu menang atas evidence dataset dengan `captured_at` ≤ waktu koreksi, dan tetap menang saat snapshot yang sama di-re-run. Koreksi kalah hanya oleh observasi dari snapshot **lebih baru** (`observed_at` > waktu koreksi). Koreksi menaikkan `entity_seq` sehingga skor terkait di-recompute.
7. **Freshness tidak disimpan.** Freshness dihitung saat dibaca dari `at` / `verified_at` dengan ambang BR-TRUST-04 karena bergantung pada waktu sekarang. Projection menyimpan `freshness_at` agar filter berbasis tanggal tidak memerlukan update harian.

`claim_ref` (dipakai sitasi AI dan laporan data) = `<entity_id>#<field>`, contoh `est_01j9…#website`.

### 5.7 Enum

| Enum | Nilai |
|---|---|
| `claim_type` | OBSERVED, DERIVED, INFERRED, UNKNOWN |
| `verification_status` | VERIFIED, LIKELY_VALID, RISKY, INVALID, UNKNOWN |
| `contact_type` | PHONE, EMAIL, WHATSAPP, SOCIAL, OTHER |
| `phone_type` | MOBILE, FIXED_LINE, FIXED_OR_MOBILE, TOLL_FREE, UAN, SHORT_CODE, VOIP, UNKNOWN |
| `website_type` | OWN_DOMAIN, PLATFORM_SUBDOMAIN, SOCIAL_PROFILE, LINK_AGGREGATOR, MESSAGING_LINK, MARKETPLACE, NONE, INVALID, UNKNOWN |
| `email_kind` | OWN_DOMAIN, WEBMAIL, DISPOSABLE, UNKNOWN |
| `business_status` | OPERATIONAL, CLOSED_TEMPORARILY, CLOSED_PERMANENTLY, UNKNOWN |
| `entity_status` | ACTIVE, MERGED, REMOVED |
| `score_band` | HOT, WARM, COLD, UNSCORED |
| `freshness` (dihitung) | FRESH, AGING, STALE |
| `snapshot_status` | STAGED, VALIDATING, VALIDATED, PUBLISHED, SUPERSEDED, REJECTED |
| `job_status` | QUEUED, RUNNING, SUCCEEDED, PARTIAL, FAILED, CANCELLED |
| `delivery_status` | PENDING, IN_FLIGHT, DELIVERED, RETRYING, DLQ, CANCELLED |
| `ledger_entry_type` | GRANT, DEBIT, HOLD, CAPTURE, RELEASE, REFUND, EXPIRE, ADJUST |
| `evidence_kind` | DATASET_RECORD, RULE, MODEL, DNS_LOOKUP, OPS_CORRECTION (FR-OPS-009), CRAWL_PAGE (reserved). Evidence upload dirender dari `upload_row` (tenant), tidak disimpan di tabel global |
| `role` | OWNER, ADMIN, MANAGER, MEMBER, VIEWER |
| `work_status` | NEW, RESEARCHED, READY, CONTACTED, DISQUALIFIED |

### 5.8 RLS & role database (ADR-03, FR-AUD-006, NFR-SEC-03)

| Role DB | Dipakai oleh | Hak |
|---|---|---|
| `msp_owner` | Migrasi (goose, `msp migrate`) | Pemilik objek. Tidak dipakai runtime |
| `msp_app` | api, worker tenant (scoring, export, webhook, ai) | `NOBYPASSRLS`. Global: `SELECT` saja. Tenant: DML sesuai kebutuhan, di bawah RLS |
| `msp_pipeline` | worker pipeline | DML pada schema global. **Tidak punya hak** atas tabel tenant mana pun |
| `msp_ops` | endpoint `/v1/ops/*` | DML merge/suppression/DSR. Akses tenant hanya lewat `tenancy.WithTx` + `support_access_grant` dalam transaksi `READ ONLY` |
| `msp_readonly_replica` | koneksi ke db-02 | `SELECT` (RLS tetap berlaku) |
| (schema `river`) | api, worker | DML untuk `msp_app` dan `msp_pipeline` (insert/fetch job). Tabel River tidak berisi data tenant selain ID di argumen job |

```sql
-- Pola untuk setiap tabel tenant
ALTER TABLE action.list ENABLE ROW LEVEL SECURITY;
ALTER TABLE action.list FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON action.list
  USING      (workspace_id = current_setting('app.workspace_id', true)::uuid)
  WITH CHECK (workspace_id = current_setting('app.workspace_id', true)::uuid);

-- Setiap transaksi tenant (dibungkus tenancy.WithTx, pgx). Setara SET LOCAL app.workspace_id = …;
-- set_config dipakai karena menerima parameter bind. Aman untuk PgBouncer transaction mode
BEGIN;
SELECT set_config('app.workspace_id', '0192…', true);   -- is_local = true → hilang saat COMMIT
-- ... query ...
COMMIT;

-- Global layer: read-only untuk app (GRANT per tabel global, dibangkitkan migrasi), record ter-suppress tidak terlihat
GRANT SELECT ON identity.company, identity.establishment, identity.contact_point, identity.person, geo.area_metric /* … */ TO msp_app;
ALTER TABLE identity.contact_point ENABLE ROW LEVEL SECURITY;
CREATE POLICY hide_suppressed ON identity.contact_point FOR SELECT TO msp_app, msp_readonly_replica USING (NOT suppressed);
CREATE POLICY pipeline_all   ON identity.contact_point TO msp_pipeline, msp_ops USING (true) WITH CHECK (true);
```

- `current_setting(..., true)` mengembalikan NULL bila tidak di-set → **tidak ada baris** (fail closed).
- Tabel terpartisi (`intel.score`) hanya diakses lewat tabel induk. Hak akses langsung ke partisi dicabut.
- Pemeriksaan aplikasi: repository tenant selalu menambahkan `workspace_id = :ws` secara eksplisit (lapis kedua). Uji lintas tenant otomatis di CI menjalankan setiap endpoint dengan workspace A terhadap ID milik B dan mengharapkan 404 (lihat `05-test-strategy.md`).

### 5.9 Stabilitas ID, merge, dan split (FR-DATA-007, BR-ER-06)

- **Pembuatan:** record sumber baru tanpa link dan tanpa match → entity baru (UUIDv7) + `entity_source_link`.
- **Re-run / snapshot baru:** `(source_id, source_record_id)` yang sudah ada selalu kembali ke entity yang sama, sehingga ID stabil tanpa bergantung pada skor ER.
- **Merge:** pemenang = ID tertua (UUIDv7 terkecil) agar deterministik. Yang kalah diberi `status = MERGED` + `entity_redirect`, dan link sumbernya dipindah. Claim dibangun ulang. Event internal `entity.merged {from, to}` diterima `action` (list_item, tag, note, assignment, status), `usage` (unlock), `intel` (score → stale), dan `ai` (artifact → invalid). Masing-masing me-rekey tabelnya per workspace. Bila workspace sudah meng-unlock kedua company → 1 credit di-refund (TQ-06).
- **API:** `GET /v1/prospects/{id_lama}` → `308 Permanent Redirect` ke ID baru. Client TS mengikuti redirect secara otomatis.
- **Split (ops):** record sumber yang dipilih dipindah ke entity baru. Entity asal mempertahankan ID dan referensi tenant. Keputusan dicatat di `merge_decision(SPLIT)` dan berlaku pada re-run berikutnya.

---
## 6. Processing pipeline

Memenuhi FR-DATA-001..011, FR-INTL-001, BR-NORM, BR-ER, BR-VER, MVP Pack §05, dan NFR-PERF-07.

### 6.1 DAG stage (per snapshot)

```
ingest.load -> identity.normalize -> identity.resolve -> identity.verify ----------------------+
                                                     \-> identity.classify -> geo.assign ------+
                                                                                               v
          intel.derive (digital lite + commercial) -> geo.area_metrics -> intel.fill_rate -> intel.gap_facts
                                                                                               |
                                    intel.base_factors -> query.project (shadow + swap) -> [PUBLISHED] -> fan-out re-score (§7.9)
```

| Stage | Modul | Input → output | Unit batch | Perilaku gagal | Ref |
|---|---|---|---|---|---|
| `load` | ingest | File snapshot → partisi `observation` + laporan | file part / 50k baris | Baris reject tercatat, batch lain lanjut | FR-DATA-001 |
| `normalize` | identity | observation → kandidat ternormalisasi (+ nilai mentah) | 10k record | Error per field; record lanjut | FR-DATA-002 |
| `resolve` | identity | kandidat → entity, link, merge_candidate | sel geohash5 | Tidak terselesaikan → entity terpisah + kandidat | FR-DATA-003/004 |
| `verify` | identity | contact point/domain → status | 1k domain | Timeout → UNKNOWN, bukan INVALID | FR-DATA-005, BR-VER-02 |
| `classify` | identity | kategori sumber → taksonomi MSP | 10k | Tidak terpetakan → "Lainnya" | FR-INTL-001 |
| `assign` | geo | koordinat/alamat → kode wilayah | kab/kota | Gagal → area NULL, `geo_confidence = 0` | FR-DATA-009 |
| `derive` | intelligence | digital lite, commercial | 10k | Field hilang → UNKNOWN | FR-DATA-010, FR-INTL-004 |
| `area_metrics` | geo | establishment_area + populasi → `area_metric` | provinsi | Coverage LOW → metrik Unknown | FR-GEO-001 |
| `fill_rate` | intelligence | → `field_fill_rate` | SQL tunggal | — | BR-OPP-02 |
| `gap_facts` | intelligence | atribut + metrik + fill-rate → `gap_fact` | 10k | Tipe tidak dapat dievaluasi → tidak dideteksi | FR-INTL-007 |
| `base_factors` | intelligence | → `establishment_factor` | 10k | Faktor Unknown | Lampiran A |
| `project` | query | semua → `prospect_search` (shadow table + swap) | 50k | Rebuild ulang dari awal (idempotent) | MVP Pack §05 Project |

### 6.2 Orkestrasi, idempotensi, re-run

- **Kunci idempotensi batch** = `sha256(stage_name ‖ stage_version ‖ input_fingerprint ‖ batch_key)`. `stage_version` mencakup versi kode stage dan versi konfigurasi yang dipakainya (mapping spec, `data/reference/*` lists, taksonomi, `er-1.0.0`, `geo-1.0.0`, katalog opportunity). `input_fingerprint` = snapshot_id + kunci idempotensi stage hulu.
- `platform.stage_run(pipeline_run_id, stage, batch_key, idem_key UNIQUE, status, attempts, records_in, records_out, error_counts jsonb, started_at, finished_at)` berfungsi sebagai checkpoint. Batch dengan `idem_key` yang sudah `SUCCEEDED` dilewati, sehingga pipeline bisa dilanjutkan setelah restart. Mode `force` menghasilkan pipeline_run baru.
- Setiap write memakai **upsert pada natural key** (`entity_source_link`, `(establishment_id)` pada tabel 1:1, `(snapshot_id, …)` pada tabel per snapshot). Menjalankan batch dua kali menghasilkan state yang sama (FR-DATA-007, MVP-11).
- **Re-run pada snapshot yang sama** tidak membuat entity baru karena `entity_source_link` dan keputusan manual (`merge_decision`) diterapkan **sebelum** ER otomatis. **Snapshot lebih baru:** record dengan `record_hash` sama dan versi stage sama dilewati di stage `normalize`..`derive`. Evidence baru tetap dibuat agar freshness ikut maju.
- **River:** setiap batch = satu job (`MaxAttempts` 5, `NextRetry` eksponensial 2 s → 5 menit + jitter, pola Crawler §8.3), unik per `idem_key` (`UniqueOpts`) sehingga enqueue ganda tidak menjalankan batch dua kali. Error non-retryable dikembalikan sebagai `river.JobCancel(err)` agar tidak di-retry. Job yang kehabisan percobaan berstatus `discarded` (= DLQ) dengan error terakhir, attempt, dan argumen (stage, versi, batch_key). Job ini dapat ditinjau dan di-*retry* dari ops console (FR-OPS-002, Crawler §13.1).
- Job macet (worker mati) dikembalikan ke antrean oleh *rescuer* River (`RescueStuckJobsAfter` 15 menit). Antrean berada di Postgres, sehingga crash worker atau Redis tidak menghilangkan job.
- **Refresh prospek** (FR-MON-001) memakai mini-DAG untuk satu company: `normalize` (data snapshot terbaru) → `resolve` (lokal) → `verify` (paksa, abaikan cache > 24 jam) → `derive` → `gap_facts` → `base_factors` → upsert projection → re-score establishment tersebut untuk semua ICP aktif workspace peminta.

### 6.3 Normalisasi (BR-NORM)

Nilai mentah selalu disimpan di `observation.raw` dan dapat ditampilkan di evidence (FR-DATA-002). Daftar pola berversi di `data/reference/` dan versinya masuk ke `stage_version`.

**Telepon (BR-NORM-01, BR-VER):** library `nyaruka/phonenumbers` (port Go libphonenumber), region default `ID`.

| Input | Aturan | Hasil |
|---|---|---|
| `0812-3456-7890` | Hapus pemisah (spasi, `-`, `.`, `()`); awalan `0` → `+62` | `+6281234567890`, MOBILE |
| `62 812 3456 7890`, `+62 (0)812…` | Awalan `62`/`+62`; hapus trunk `(0)` | `+6281234567890` |
| `(0361) 234567` | Kode area 2–4 digit (021 Jakarta, 022 Bandung, 031 Surabaya, 0361 Denpasar, …) | `+62361234567`, FIXED_LINE |
| `021-5551234 ext 12` | `ext`, `ext.`, `#`, `psw` → extension | `+62215551234;ext=12` |
| `81234567890` | 9–12 digit diawali `8` tanpa awalan → asumsikan seluler, confidence −10 | `+6281234567890` |
| `0812… / 0813…` | Pecah pada `/ , ;` dan kata "atau", maksimal 3 | Beberapa contact point |
| `0800-1-234567` | Toll-free | TOLL_FREE, `is_shared = true` |
| `1500-123`, `14045` | UAN / short code (bukan E.164 identitas) | UAN/SHORT_CODE, shared, status RISKY |
| `0812` | Tidak valid / terlalu pendek | INVALID |

- **Mobile → WA-likely:** `phone_type = MOBILE` (awalan +628) → `wa_likely = true`, claim INFERRED "Kemungkinan WhatsApp" (BR-VER-01). Nilai observed `wa.me/…` → contact point `WHATSAPP` OBSERVED.
- **Nomor bersama (BR-ER-08):** `is_shared = true` bila nomor muncul di ≥ 3 establishment milik company berbeda (nama dan domain berbeda) dalam satu snapshot (dihitung di awal `resolve` dari kandidat ternormalisasi, sebelum dipakai sebagai blok), tercantum di `data/reference/phone_shared.yaml`, atau bertipe TOLL_FREE/UAN/SHORT_CODE. Nomor bersama bukan identifier kuat (BR-ER-01).

**Email (BR-NORM-02):** trim, buang `mailto:`, domain di-lowercase + IDNA, local-part dipertahankan untuk tampilan. Ditolak bila domain contoh/placeholder (`example.com`, `test.com`) atau gagal syntax (`net/mail` + aturan tambahan). Domain di `webmail.txt` (gmail.com, yahoo.com, yahoo.co.id, ymail.com, hotmail.com, outlook.com, live.com, icloud.com, proton.me, …) → `WEBMAIL`. Domain di `disposable_email.txt` → `DISPOSABLE`. Local-part di `role_localparts.txt` (info, admin, sales, cs, marketing, office, contact, support, …) → `is_role_based = true`.

**Domain/website (BR-NORM-03, FR-DATA-010):** parse URL (tambahkan skema bila tidak ada), host di-lowercase + IDNA, buang `www.`, port default, fragment, dan query tracking (`utm_*`, `fbclid`, `gclid`; daftar izin/tolak, bukan buang semua query, sesuai Crawler §7.2). URL lengkap disimpan untuk tampilan. Registrable domain dihitung dengan **Public Suffix List** via `golang.org/x/net/publicsuffix` (versi di-pin di `go.mod`; tambahan di `data/reference/public_suffix_extra.dat`), mencakup `co.id`, `my.id`, `web.id`, `or.id`, `ac.id`, `sch.id`, `go.id`, `biz.id`, `net.id`, `desa.id`, `ponpes.id`. Bagian *private* PSL (mis. `blogspot.com`) menjadikan `abc.blogspot.com` registrable domain tersendiri, sehingga `website_type` selalu dicek terhadap daftar platform **sebelum** domain dianggap OWN_DOMAIN. Klasifikasi `website_type`:

| Tipe | Contoh pola (file di `data/reference/domains/`) | Identitas company? |
|---|---|---|
| OWN_DOMAIN | registrable domain yang tidak ada di daftar mana pun | Ya (identifier kuat) |
| PLATFORM_SUBDOMAIN | `*.blogspot.com`, `*.wordpress.com`, `*.wixsite.com`, `*.weebly.com`, `sites.google.com`, `*.business.site`, `*.carrd.co`, `*.notion.site`, `*.github.io`, `*.vercel.app`, `*.netlify.app`, `*.myshopify.com` | Tidak |
| SOCIAL_PROFILE | instagram.com, facebook.com, fb.com, tiktok.com, x.com, twitter.com, youtube.com, linkedin.com | Tidak |
| LINK_AGGREGATOR | linktr.ee, lynk.id, s.id, bio.link, beacons.ai, taplink.cc, msha.ke | Tidak |
| MESSAGING_LINK | wa.me, api.whatsapp.com, wa.link, t.me, line.me | Tidak (WA → contact point) |
| MARKETPLACE | tokopedia.com, shopee.co.id, bukalapak.com, lazada.co.id, gofood.co.id | Tidak |
| NONE / INVALID | kosong / tidak dapat di-parse | — |

**Nama (BR-NORM-04):** NFKC (`x/text/unicode/norm`), folding diakritik (`x/text` transform), collapse spasi, normalisasi tanda baca, lowercase untuk `name_norm`. Token bentuk badan usaha (`legal_forms.yaml`: PT, P.T., Perseroan Terbatas, CV, UD, Usaha Dagang, Tbk, (Persero), Koperasi, KSP, Yayasan, PD, Perum, Firma) dan gelar (`titles.yaml`: dr., drg., Dr., Prof., H., Hj., Ir., Sp.\*) dipisahkan. Alias yang dibangkitkan: (1) asli; (2) tanpa badan usaha; (3) tanpa gelar; (4) tanpa kata generik kategori (`klinik`, `apotek`, `toko`, `warung`, `rumah makan`, …), khusus untuk matching; (5) varian sinonim ID↔EN dari `data/taxonomy/synonyms_id_en.yaml` (contoh: "klinik gigi" ↔ "dental clinic") untuk search (FR-SRCH-001). Nilai asli tidak pernah dihapus.

**Alamat (BR-NORM-05):** `address_display` dipertahankan apa adanya. `address_norm` = lowercase, singkatan baku (`jl.`→`jalan`, `kel.`→`kelurahan`, `kec.`→`kecamatan`, `kab.`→`kabupaten`, `no.`→`nomor`), dan kode pos diekstrak. Komponen administratif (dari kolom dataset atau parsing akhir alamat) disimpan di `address_parts`. Pemetaan ke kode Kemendagri dilakukan di stage `geo.assign` (§6.7) agar `identity` tidak bergantung pada `geo`.

### 6.4 Entity resolution (BR-ER, FR-DATA-003/004)

Konfigurasi `er-1.0.0` (bobot dan ambang) dikalibrasi pada golden set (RG-6).

**Langkah per batch (sel geohash5 ± tetangga):**
1. Terapkan `entity_source_link` (record yang sudah dikenal) dan `merge_decision` aktif (must-link / cannot-link).
2. **Blocking** untuk record baru/berubah terhadap entity yang ada dan terhadap sesamanya:

| Blok | Kunci | Catatan |
|---|---|---|
| B1 spasial | `ST_DWithin(location, 150 m)` + `similarity(name_norm) ≥ 0,3` | Index GiST + trigram |
| B2 telepon | E.164 sama dan `is_shared = false` | — |
| B3 domain | registrable domain sama dengan `website_type = OWN_DOMAIN` | Lintas kota (company-level) |
| B4 tanpa koordinat | kode pos/kelurahan teks sama + overlap token alamat ≥ 0,5, atau `name_norm` sama | — |

3. **Fitur pasangan**:
   - `name_sim` = maks(Jaro-Winkler atas token terurut (`adrg/strutil`), trigram (`similarity()` pg_trgm, dihitung saat blocking), token-set berbobot IDF), dihitung setelah alias (2)–(4). Token generik kategori mendapat bobot rendah via IDF korpus.
   - `dist_score` = 1,0 (≤ 25 m), 0,8 (≤ 50 m), 0,4 (≤ 100 m), 0,1 (≤ 200 m), 0 (> 200 m). Bila koordinat tidak ada: 1,0 untuk `address_norm` yang sama, selain itu fitur absen.
   - `phone_agree` ∈ {1, 0, absen}; `domain_agree` ∈ {1, 0, absen}; `sector_agree` (kategori MSP).
4. **Aturan & skor:**
   - R0 (BR-ER-05): dua domain milik sendiri yang berbeda → **cannot-link** (tidak pernah auto-merge).
   - R1 (BR-ER-02): (identifier kuat sama **atau** `name_sim ≥ 0,90`) **dan** (jarak ≤ 50 m **atau** alamat ternormalisasi sama) → establishment sama.
   - Skor fuzzy: `s = Σ wᵢ·fᵢ / Σ wᵢ` atas fitur yang ada, dengan w = {name 0,50; dist 0,25; phone 0,15; domain 0,10}. `name_sim` wajib ada. Sektor berbeda → `s × 0,85`.
   - BR-ER-04: `s ≥ 0,92` dan tanpa konflik → auto-merge. 0,80 ≤ s < 0,92 → `merge_candidate`. Bila **high value** → queue review; selain itu tetap terpisah (`AUTO_SEPARATE`). `s < 0,80` → terpisah.
   - *High value* = establishment yang ada di kandidat ICP aktif atau memiliki aktivitas tenant (list, unlock, assignment). `intelligence` (setelah scoring) dan `action`/`usage` (saat aktivitas) memanggil `identity.Service.MarkHighValue(ctx, ids)`, yang menulis `identity.high_value_entity` (hanya ID + jenis alasan). Pipeline global membaca tabel ini tanpa menyentuh tabel tenant.
5. **Clustering:** union-find atas edge merge. Bila sebuah union akan menyatukan pasangan cannot-link, edge terlemah dibuang dan pasangan dikirim ke review. Ini mencegah *chaining* transitif.
6. **Company grouping:** establishment dengan OWN_DOMAIN yang sama dan berjarak > 200 m → company sama, establishment berbeda (BR-ER-03). Nomor non-shared yang sama + `name_sim ≥ 0,80` → company sama. Field `brand` yang sama di ≥ 2 kab/kota → kandidat grouping (review). **BR-ER-09:** domain sendiri sama pada jarak 50–200 m → tetap satu company; establishment hanya di-merge bila `name_sim ≥ 0,90` **dan** `address_norm` sama, selain itu establishment terpisah dengan flag `geo_ambiguous` (masuk review bila high value).
7. Bangun ulang claim (§5.6) dan naikkan `entity_seq` untuk entity yang tersentuh.

**Review queue (FR-OPS-005):** diurutkan high-value dulu, lalu |s − 0,92| terkecil. Ops memilih MERGE / KEEP_SEPARATE (+ alasan). Keputusan berlaku final terhadap re-run sampai diubah ops (BR-ER-06) dan tercatat di audit (FR-AUD-001). Metrik: ukuran queue, umur tertua, precision sampel ≥ 98%, rasio review/total kandidat ≤ 5% (BR-ER-10), serta recall dilaporkan dua kali: termasuk pasangan di queue (terdeteksi) dan auto-recall saja.

### 6.5 Verifikasi (BR-VER, FR-DATA-005, FR-DATA-011)

| Objek | Pemeriksaan | Pemetaan status |
|---|---|---|
| Email | Syntax (validator RFC 5322 subset, tanpa jaringan) → MX domain | INVALID: syntax gagal, NXDOMAIN, *null MX* (RFC 7505), atau tidak ada MX (mengikuti PRD; lihat TQ-03). RISKY: role-based, disposable. LIKELY_VALID: syntax valid + MX ada. VERIFIED: hanya via provider (V1). UNKNOWN: timeout/SERVFAIL |
| Telepon | `is_possible` + `is_valid` + `number_type` (region ID) | LIKELY_VALID: valid dan tipe MOBILE/FIXED_LINE/FIXED_OR_MOBILE. RISKY: valid tetapi tipe UNKNOWN/UAN/SHORT_CODE. INVALID: tidak valid |
| Domain | A/AAAA, MX, TXT (SPF) | Mengisi `dns_lookup` dan `digital_presence` (`email_provider_hint` dari MX: Google Workspace, Microsoft 365, Zoho, hosting lokal; `spf_includes` → `intel.technology`) |
| WhatsApp | — | Selalu INFERRED/UNKNOWN di MVP (BR-VER-01) |

- **DNS:** lookup lewat resolver rekursif **Unbound** milik sendiri di app-01 (bukan resolver publik, untuk menghindari rate limit). Library `miekg/dns` (goroutine pool), timeout 2 s per query, lifetime 4 s, 2 retry, konkurensi global 200 query/s. Hasil disimpan di `identity.dns_lookup` dan sebagai evidence `DNS_LOOKUP`. Cache: positif = min(TTL record, 24 jam) di Redis + tabel. Re-verify bila > 30 hari atau saat refresh. NXDOMAIN di-cache 1 jam. Timeout/SERVFAIL **tidak** di-cache → status UNKNOWN (BR-VER-02).
- `verified_at` = waktu pemeriksaan. `verification_confidence`: LIKELY_VALID email 75, telepon 70; RISKY 40; INVALID 90 (keyakinan atas status invalid).
- Adapter provider verifikasi email (V1, `VerifierProvider` port) menambahkan status VERIFIED dan deteksi catch-all.

### 6.6 Klasifikasi industri (FR-INTL-001)

- Taksonomi dua level di `data/taxonomy/msp-taxonomy-1.0.0.yaml` (sektor → kategori; kode seperti `HLT.DENTAL`, `FNB.CAFE`), dengan label id-ID/en-US dan sinonim.
- Peta kategori sumber per source di `data/taxonomy/source-maps/<source>.yaml` (contoh `Dentist` → `HLT.DENTAL`, confidence 95).
- Urutan: (1) peta eksak kategori primer (confidence 90–95); (2) kategori sekunder; (3) aturan kata kunci nama (`klinik gigi`, `drg.`, `dental` → `HLT.DENTAL`, confidence 60–80); (4) tidak terpetakan → `OTH.OTHER` ("Lainnya", confidence 30).
- **AI hanya untuk string kategori sumber yang belum terpetakan** (ratusan string, bukan per bisnis). Tool ops `msp tool taxonomy-suggest` (modul `ops`, memakai `ai` dengan alias kelas Haiku) mengusulkan mapping secara batch, ops meninjau, lalu usulan di-commit ke source-map. Biaya kecil dan tetap ter-govern. Stage `classify` (modul `identity`) tetap deterministik dan tidak memanggil LLM.
- Metrik: % establishment aktif non-"Lainnya" (target ≥ 95%, FR-INTL-001) per source, dan akurasi pada sampel berlabel n ≥ 300 (≥ 90%, RG-6).

### 6.7 Penetapan area (FR-DATA-009)

| Urutan | Metode | `geo_confidence` |
|---|---|---|
| 1 | Point-in-polygon: `ST_Contains(geom_sub, point)` pada batas `area_set_version` aktif (index GiST atas polygon hasil `ST_Subdivide`) | 0,95. Bila jarak ke batas < 30 m: 0,80 |
| 2 | Kecocokan nama administratif hierarkis (provinsi → kab/kota → kecamatan → kelurahan) dengan alias (`Jaksel`, `Kota Adm.`, `Kab.`) | Eksak 0,90; fuzzy ≥ 0,85: 0,70 |
| 3 | Kode pos → kelurahan/kecamatan (tabel referensi kode pos, bila ada) | 0,60 |
| 4 | Ada koordinat tetapi tanpa batas: kelurahan dengan centroid terdekat di kecamatan hasil langkah 2 | 0,50 |
| — | Gagal semua | area NULL, 0 |

Konflik antara PIP dan nama alamat (beda kecamatan): PIP menang, confidence −0,2, dan kasus ditandai untuk QA sampling. Target: ≥ 90% establishment ber-koordinat terpetakan (FR-DATA-009) dan akurasi kelurahan ≥ 95% pada sampel berlabel (RG-6).

### 6.8 Derive: digital lite, commercial, faktor dasar

- **Digital presence lite** (FR-DATA-010): `website_type`, `own_domain`, `best_email_kind`, social profiles, penyedia email dari MX, tool dari SPF (FR-DATA-011), dan `has_online_booking` bila ada di dataset.
- **Commercial (BR-REV):** band karyawan observed bila ada. Selain itu INFERRED dari prior kategori × tier kota (`data/reference/employee_priors-0.1.0.yaml`, confidence ≤ 60). Band revenue v0 = prior kategori × tier kota (`revenue_priors-0.1.0.yaml`) × pengali (jumlah cabang, persentil volume ulasan, price level), confidence ≤ 70 (BR-REV-03), dengan `basis[]` yang dapat dibaca (BR-REV-05). Point estimate tidak disimpan (BR-REV-04). Tier kota berasal dari `data/reference/city_tier.yaml` (TQ-08).
- **Faktor dasar global** (`intel.establishment_factor`): Contactability, Commercial Potential (+ confidence), komponen Data Quality (verifikasi, freshness, evidence coverage, dan completeness per tag Offering). Faktor yang bergantung pada tenant (ICP Fit, Digital, Geographic, Opportunity) dihitung saat scoring (§7).

### 6.9 Estimasi throughput (NFR-PERF-07: 5 juta establishment ≤ 12 jam)

Asumsi app-01 (32 core): worker `pipeline` dengan 32 goroutine kerja; db-01 (32 core, 256 GB).

| Stage | Perkiraan | Catatan |
|---|---|---|
| load | 15 mnt | COPY paralel dari Parquet |
| normalize | 20 mnt | ±5k rec/s per core (Go) × ±16 core efektif; bottleneck pada write DB |
| resolve | 90–120 mnt | 5 juta × ±5 kandidat = 25 juta pasangan; Jaro-Winkler paralel (goroutine) per sel |
| verify | 80 mnt (paralel dengan classify/assign) | ±1 juta domain unik @ 200 qps; telepon CPU-only |
| classify + assign | 30 + 40 mnt | PIP dengan `ST_Subdivide` |
| derive + area_metrics + fill_rate + gap_facts + base_factors | 60 mnt | Sebagian besar SQL set-based |
| project | 45–60 mnt | Shadow table + `CREATE INDEX` paralel + swap |
| **Total** | **±6–7 jam** | Margin ±40% terhadap 12 jam. Re-run snapshot berikutnya jauh lebih cepat karena record tak berubah dilewati |

---
## 7. Scoring engine

Memenuhi FR-ICP-003/004, FR-INTL-002..006/008/009/011, BR-SCORE-01..08, Lampiran A PRD, NFR-PERF-05, dan NFR-DATA-01.

### 7.1 Konfigurasi model berversi

File `apps/api/internal/intelligence/internal/scoring/models/priority-1.2.0.yaml` (di-*embed* ke binary via `embed`) didaftarkan ke `intel.score_model` (sha256). Versi yang sudah aktif bersifat immutable. Perubahan bobot, aturan, atau preset menghasilkan versi baru + job recompute (FR-INTL-009).

```yaml
model: priority
version: 1.2.0
factors:                       # urutan = indeks array intel.score.factors
  - { key: icp_fit,        weight: 25, plugin: icp_fit@1 }
  - { key: data_quality,   weight: 10, plugin: data_quality@1 }
  - { key: contactability, weight: 10, plugin: contactability@1 }
  - { key: commercial,     weight: 15, plugin: commercial_potential@1 }
  - { key: digital_opp,    weight: 10, plugin: digital_opportunity@1 }
  - { key: geo_opp,        weight: 15, plugin: geographic_opportunity@1 }
  - { key: signal,         weight: 5,  plugin: signal_strength@0, enabled: false }   # Unknown sampai V1 (I-07)
  - { key: opportunity,    weight: 10, plugin: opportunity_strength@1 }
bands: { hot: 80, warm: 60 }            # D-07, BR-SCORE-05
coverage_min: 0.60                      # BR-SCORE-04
precision: 1                            # simpan 1 desimal; tampil bilangan bulat half-up; band dari nilai tampil (BR-SCORE-10)
percentile: midrank                     # BR-SCORE-09
presets:                                # FR-ICP-004: faktor yang disebut ditetapkan, sisanya diskalakan proporsional agar Σ = 100
  default: {}
  reputation_mgmt: { opportunity: 20, digital_opp: 5, geo_opp: 10 }
  website_dev:     { digital_opp: 15, opportunity: 5 }
  local_seo:       { digital_opp: 12, geo_opp: 18, opportunity: 5 }
  booking_system:  { digital_opp: 15, commercial: 15, opportunity: 5 }
icp_fit:
  criterion_weights: { category: 3, area: 2, employee_band: 1, revenue_band: 1, digital: 1, contact_required: 1 }
```

`model_version` yang tercatat di skor = `priority-1.2.0+<preset>` (misalnya `priority-1.2.0+reputation_mgmt`). Preset selain `default` dan `reputation_mgmt` adalah usulan awal yang divalidasi dengan customer beta (Q-07).

### 7.2 Antarmuka plugin faktor

```go
type FactorPlugin interface {
    Key() string
    Version() string
    Requires() []string                                 // kolom projection/faktor dasar yang dibaca
    Compute(b *FactorBatch, sc *ScoringContext) FactorResult
}
// FactorResult: Values []float64 (0..100; math.NaN() = UNKNOWN), ClaimType []ClaimType, Reasons map[int]string (alasan Unknown)
```

`ScoringContext` berisi workspace, ICP version, Offering tags, preset, `as_of` (tanggal skor), dan metrik area per ICP. Plugin bersifat murni: tidak melakukan I/O dan tidak memakai jam sistem (selalu `sc.AsOf`).

### 7.3 Definisi operasional faktor

| Faktor | Perhitungan (Lampiran A + detail SDD) | UNKNOWN bila |
|---|---|---|
| ICP Fit | Rata-rata tertimbang hasil kriteria (§7.4): match 1, partial 0,5, no 0. Kriteria Unknown dikeluarkan | Semua kriteria Unknown → band UNSCORED (BR-SCORE-04) |
| Data Quality | `0,4·completeness + 0,3·verifikasi + 0,2·freshness + 0,1·evidence_coverage`. Completeness = proporsi field relevan Offering yang diketahui (field set per tag di `scoring/field_sets.yaml`). Verifikasi dari status kontak terbaik: VERIFIED 100, LIKELY_VALID 80, RISKY 40, UNKNOWN 20, INVALID 0. Freshness: FRESH 100, AGING 60, STALE 20 | Tidak pernah |
| Contactability | Kanal terbaik: email person VERIFIED 100; email person LIKELY_VALID 90 (TQ-04); email company LIKELY_VALID 80; seluler LIKELY_VALID (WA-likely) 80; telepon tetap LIKELY_VALID 60; hanya profil sosial 30; tidak ada 0. Pengali status: RISKY/UNKNOWN 0,5, INVALID 0. +10 bila ada kontak decision-maker (maksimal 100) | Tidak pernah ("tidak ada" = 0) |
| Commercial | Persentil mid-rank (BR-SCORE-09) skala proxy (ordinal band revenue, band karyawan, jumlah cabang, persentil volume ulasan) dalam kategori × tier kota, × (0,5 + 0,5·confidence/100) | Tidak ada proxy skala satu pun |
| Digital Opportunity | Maks `opportunity_score` keluarga DIGITAL yang relevan (§8.3). 0 bila data digital diketahui tetapi tidak ada gap relevan | `website` dan `email` tidak dapat dievaluasi (fill-rate segmen < 20%) |
| Geographic Opportunity | `market_opportunity_score` kelurahan untuk ICP (§9.4). Fallback kecamatan bila kelurahan LOW. Level yang dipakai dicatat di breakdown (FR-INTL-006) | Coverage LOW di kelurahan dan kecamatan, atau establishment tanpa area |
| Signal | — | Selalu (V1) |
| Opportunity Strength | Maks `opportunity_score` non-DIGITAL yang relevan (BR-SCORE-08) | Tidak ada tipe non-DIGITAL relevan yang dapat dievaluasi |

### 7.4 ICP: kriteria dan evaluasi (FR-ICP-002/003)

```json
{
  "criteria": [
    { "key": "category",         "mode": "required",  "values": ["HLT.DENTAL", "HLT.CLINIC"] },
    { "key": "area",             "mode": "required",  "values": ["51.71", "51.03"] },
    { "key": "employee_band",    "mode": "preferred", "values": ["5-19", "20-99"] },
    { "key": "revenue_band",     "mode": "preferred", "values": ["R2", "R3"] },
    { "key": "digital",          "mode": "preferred", "values": ["NO_OWN_WEBSITE"] },
    { "key": "contact_required", "mode": "required",  "values": ["PHONE"] }
  ],
  "exclusions": { "categories": [], "areas": [], "company_ids": [], "in_list_ids": [], "include_closed": false }
}
```

| Kriteria | Match | Partial | No | Unknown |
|---|---|---|---|---|
| category | Kategori sama | Sektor sama | Lainnya | Kategori "Lainnya" dengan confidence < 50 |
| area | Di dalam salah satu area | Kab/kota sama tetapi kecamatan/kelurahan berbeda | Di luar | Tanpa area |
| employee_band / revenue_band | Band sama | Band bersebelahan | Lainnya | Band tidak diketahui |
| digital | Kondisi terpenuhi | — | Tidak terpenuhi | Field digital tidak dapat dievaluasi |
| contact_required | Ada kanal ≥ LIKELY_VALID | Ada kanal RISKY/UNKNOWN | Tidak ada | — |

Kriteria `required` diterjemahkan menjadi **filter SQL** pada projection: `(employee_band IS NULL OR employee_band = ANY(:v))`. Nilai `no` → tidak masuk hasil (Lampiran A). Nilai Unknown lolos filter, kriterianya dikeluarkan dari rata-rata, baris diberi flag `unconfirmed_required` ("Kriteria wajib belum terkonfirmasi"), dan band maksimal WARM (BR-SCORE-12). Exclusion dan `include_closed = false` (BR-SCORE-07) juga diterapkan di SQL.

### 7.5 Renormalisasi, coverage, band (BR-SCORE-03..05)

```
K        = faktor dengan nilai diketahui (bukan NaN, enabled)
priority = round_1dp( Σ_{k∈K} w_k·s_k / Σ_{k∈K} w_k )                 # disimpan 1 desimal
display  = round_half_up(priority)                                      # bilangan bulat untuk UI (BR-SCORE-10)
coverage = Σ_{k∈K} w_k / Σ_{semua enabled+disabled} w_k        # signal (5) dihitung di penyebut → coverage MVP maks 0,95
band     = UNSCORED                           bila icp_fit UNKNOWN
         = HOT (display ≥ 80) | WARM (≥ 60) | COLD   selain itu
         = min(band, WARM)                    bila coverage < 0,60 (+ limited_data, BR-SCORE-11: nilai tidak dipotong)
         = min(band, WARM)                    bila unconfirmed_required (BR-SCORE-12)
```

### 7.6 Rollup company (BR-SCORE-01)

Dalam job scoring, untuk setiap `company_id` di antara establishment yang lolos filter ICP, `is_company_best = true` diberikan pada establishment dengan `value` tertinggi (tie-break: `entity_seq` terbaru, lalu `establishment_id`). Search hanya membaca baris `is_company_best`. "+N cabang" diambil dari `company.establishment_count` (FR-SRCH-004).

### 7.7 Penjelasan skor (FR-PROF-002, BR-SCORE-06)

`GET /v1/prospects/{id}/score?icp_id=` membangun penjelasan dari baris skor (nilai faktor tersimpan) + config model + data claim saat ini (untuk rincian kriteria ICP, diberi label bila `stale`).

```json
{
  "prospect_id": "cmp_01j9…", "establishment_id": "est_01j9…", "icp_id": "icp_01j9…", "icp_version": 3,
  "model_version": "priority-1.2.0+default", "calculated_at": "2026-10-10T03:14:07Z", "as_of": "2026-10-10",
  "inputs": { "snapshots": { "biz_dataset_a": "ds-2026.10.1" }, "geo_version": "geo-1.0.0", "population_version": "pop-2025S1" },
  "value": 84.3, "display_value": 84, "band": "HOT", "coverage": 0.95, "limited_data": false, "unconfirmed_required": false, "stale": false,
  "factors": [
    { "key": "icp_fit", "status": "KNOWN", "value": 92, "weight": 25, "effective_weight": 0.263, "contribution": 24.2, "claim_type": "DERIVED",
      "criteria": [ { "key": "category", "result": "MATCH", "claim_ref": "est_01j9…#category_code" },
                    { "key": "employee_band", "result": "UNKNOWN" } ] },
    { "key": "digital_opp", "status": "KNOWN", "value": 64, "weight": 10, "effective_weight": 0.105, "contribution": 6.7, "claim_type": "INFERRED",
      "opportunities": [ { "type": "OPP-WEB-NONE", "score": 64, "relevance": 1.0, "gap_fact_id": "gap_01j9…" } ] },
    { "key": "signal", "status": "UNKNOWN", "weight": 5, "reason": "NOT_AVAILABLE_UNTIL_V1" }
  ]
}
```

`effective_weight = w_k / Σ_{K} w`, dan `contribution = effective_weight × value`. Σ contribution = `value` (sebelum pembulatan).

### 7.8 Determinisme & reproducibility (FR-INTL-008/009, NFR-DATA-01)

- Fungsi skor murni atas: projection pada snapshot tertentu + `icp_version` + `model_version` + `as_of`. Tidak ada randomness. Aritmetika float64, disimpan 1 desimal, ditampilkan bilangan bulat half-up (BR-SCORE-10). Komponen waktu (freshness) dihitung terhadap `as_of` = `calculated_at` (BR-SCORE-13).
- Skor lama **tidak disimpan selamanya**. Yang dipertahankan adalah semua versi config model, ICP version, metadata snapshot, dan versi stage. Perintah ops `msp tool score-reproduce --workspace … --icp … --version … --snapshot … --as-of …` menghitung ulang dan membandingkan hasilnya (diuji di CI dengan fixture).

### 7.9 Recompute, epoch, dan progres (NFR-PERF-05, FR-INTL-011)

```sql
CREATE TABLE intel.score (
  workspace_id uuid NOT NULL, icp_id uuid NOT NULL, epoch integer NOT NULL,
  establishment_id uuid NOT NULL, company_id uuid NOT NULL,
  value numeric(4,1) NOT NULL,    -- 0.0..100.0 (BR-SCORE-10); -1 = UNSCORED (urut terakhir; API → null)
  band intel.score_band NOT NULL, coverage numeric(3,2) NOT NULL,
  factors smallint[] NOT NULL,    -- urutan config; -1 = UNKNOWN
  is_company_best boolean NOT NULL, limited_data boolean NOT NULL, unconfirmed_required boolean NOT NULL,
  input_seq bigint NOT NULL,      -- entity_seq establishment saat dihitung
  bucket smallint NOT NULL,       -- hash(establishment_id) % 100 (estimasi count, §10.6)
  PRIMARY KEY (workspace_id, icp_id, epoch, establishment_id)
) PARTITION BY HASH (workspace_id);                                          -- 32 partisi
CREATE INDEX score_rank ON intel.score (workspace_id, icp_id, epoch, value DESC, establishment_id DESC)
  WHERE is_company_best;
```

| Pemicu | Antrean | Lingkup |
|---|---|---|
| Simpan/ubah ICP atau Offering-nya | `scoring` prioritas 1 | ICP tersebut |
| Snapshot published / area_set atau populasi baru | `scoring` prioritas 4 (staggered) | Semua ICP aktif semua workspace |
| Versi model baru diaktifkan | `scoring` prioritas 4 | Semua ICP yang memakai model |
| Refresh prospek, merge/split | `scoring` prioritas 1 | Establishment terkait × ICP workspace terkait |
| Harian: kelas freshness berpindah (BR-SCORE-13) | `scoring` prioritas 4 | Hanya establishment yang kelas freshness-nya berubah sejak `as_of`. Job menaikkan `entity_seq` (otomatis stale), lalu re-skor inkremental |

Alur job (per ICP):
1. Seleksi kandidat dengan SQL (filter wajib + exclusion) pada projection → ≤ 200.000 baris (asumsi NFR). Kandidat diurutkan berdasarkan skor dasar global (DQ + contactability + commercial) agar **halaman pertama (≥ 50 kandidat teratas pre-ranking) ber-skor ≤ 30 detik** (NFR-PERF-05). Chunk pertama 10.000 baris ditargetkan ±10 detik (FR-ICP-005).
2. Hitung metrik area per ICP (§9.4) sebelum scoring, karena dipakai faktor Geographic.
3. Scoring per chunk 50.000 baris di goroutine pool (±100k baris/s) → `pgx.CopyFrom` ke epoch `n+1`. Progres `score_job.done/total` diperbarui per chunk.
4. Tandai `is_company_best`, hitung `icp_area_metric` (territory score butuh hitungan HOT/WARM).
5. Dalam satu transaksi: `score_context.active_epoch = n+1`. Epoch lama dihapus setelah ≥ 1 jam agar cursor pagination yang masih terbuka tetap konsisten (§10.5).
6. Target: ≤ 2 menit untuk 200.000 kandidat (batas NFR 10 menit). Skoring pertama ICP baru membaca epoch yang sedang ditulis (progresif). Re-skor menampilkan epoch lama berlabel stale sampai swap.

**Stale (FR-INTL-011)** dihitung saat dibaca, tanpa UPDATE massal: `stale = score.input_seq < projection.entity_seq OR score_context.icp_version < icp.current_version OR score_context.snapshot_set ≠ current_snapshot_set OR score_context.model_version ≠ model aktif`. Badge stale tampil sampai recompute selesai; pemicu di atas otomatis mengantrekan recompute.

---

## 8. Opportunity engine

Memenuhi FR-INTL-007, FR-INTL-010, BR-OPP-01..05, Lampiran B PRD, dan D-04.

### 8.1 Katalog sebagai data

`data/taxonomy/opportunity-catalog-0.1.0.yaml` memuat setiap tipe Lampiran B: `code`, `family`, `fields`, `detector` (nama fungsi), `severity`, `base_confidence`, `absence_claim` (perlu guard fill-rate?), `offering_tags.primary/secondary`, dan `status` (`active`, `inactive`, `v1`). Status *inactive* ditetapkan otomatis per source bila field sumbernya tidak ada di mapping spec (BR-OPP-01), misalnya OPP-BOOKING-NONE tanpa `has_online_booking`.

### 8.2 Gap fact global (per snapshot)

Stage `gap_facts` mengevaluasi detector untuk setiap establishment aktif dan menulis `intel.gap_fact` (global, tanpa konteks tenant).

```
confidence = base_confidence(type)                      # mis. 85 untuk field observed, 70 untuk derived
           × trust_adj (claim confidence field sumber / 100, minimal 0,6)
           - 30   bila absence_claim dan 0,20 ≤ fill_rate(field, kategori, kab/kota) < 0,50    (BR-OPP-02)
           tidak dideteksi bila absence_claim dan fill_rate < 0,20
           - 10   bila claim sumber STALE
fill_rate    = segmen kategori × kab/kota bila n ≥ 20, selain itu kategori × provinsi, lalu kategori × nasional (BR-OPP-02)
peer_context = { scope: "HLT.DENTAL × 51.71", peer_n, peer_rate }  # mis. 68% punya website; hanya bila peer_n ≥ 10
evidence_ids = evidence field sumber + evidence RULE (detector@versi, input)
```

- Gap fact yang tidak terdeteksi lagi pada snapshot baru → `status = RESOLVED` (riwayat dipertahankan; input untuk signal V1).
- OPP-MKT-HIGH-COMPETITION memakai `competition_index` (area, kategori establishment) dari `geo.area_metric`. OPP-OPS-MULTI-BRANCH melekat pada semua establishment company dengan ≥ 3 establishment.

### 8.3 Relevansi tenant dihitung saat dibaca (ADR-15)

```
relevance(type, offering) = 1,0  bila tag primer tipe ∩ tags Offering ≠ ∅
                          = 0,6  bila hanya tag sekunder yang cocok
                          = 0    selain itu → gap tetap tampil di kartu digital/reputasi, tetapi bukan "opportunity"
opportunity_score = severity × confidence/100 × relevance                     (BR-OPP-03)
```

- Tabel relevansi `type × offering_tag` kecil (12 × 9) dan di-cache di memori. Untuk profil: `gap_fact` establishment ⋈ tag Offering milik ICP aktif → daftar opportunity terurut dalam < 5 ms.
- Search filter "tipe opportunity" (FR-SRCH-002): projection menyimpan `gap_types text[]` (GIN). Aplikasi menerjemahkan pilihan user menjadi tipe yang relevan untuk Offering aktif → `gap_types && :types`.
- Opportunity yang dipakai saat scoring disalin ke penjelasan faktor (§7.7), sehingga skor tetap dapat dijelaskan walau Offering berubah kemudian.
- Private prospect (upload) memakai detector yang sama; hasilnya disimpan di `private_prospect_search`.

### 8.4 Rationale (BR-OPP-04, NFR-L10N-01)

Template ICU MessageFormat di `data/taxonomy/opportunity-rationale.<locale>.yaml`:

```yaml
OPP-WEB-NONE:
  id-ID: "Tidak ada website pada data sumber (snapshot {snapshot}).{peer, select, none {} other { {peer_rate, number, percent} {category_plural} di {area_name} memiliki website.}}"
  en-US: "No website in the source data (snapshot {snapshot}).{peer, select, none {} other { {peer_rate, number, percent} of {category_plural} in {area_name} have a website.}}"
OPP-REP-LOW-RATING:
  id-ID: "Rating {rating} dari {review_count} ulasan, di bawah 4,0."
```

Contoh hasil: *"Tidak ada website pada data sumber (snapshot 2026.10). 68% klinik gigi di Kota Denpasar memiliki website."* Label snapshot = `ds-2026.10.1` → "2026.10". Setiap rationale disertai disclaimer hipotesis (BR-OPP-05) di UI.

### 8.5 Feedback (FR-INTL-010)

`POST /v1/prospects/{id}/feedback` → `intel.feedback` (`OPPORTUNITY`: RELEVANT/NOT_RELEVANT + `reason_code` {SUDAH_PUNYA, TIDAK_SESUAI_OFFERING, DATA_SALAH, LAINNYA}; `SCORE`: TOO_HIGH/TOO_LOW). Agregat per tipe, per versi katalog, dan per preset ditampilkan di Intelligence Health (FR-OPS-003). Feedback tidak mengubah skor secara otomatis di MVP; kalibrasi severity dilakukan manual per rilis katalog.

---

## 9. Geo engine

Memenuhi FR-GEO-001..005/009, FR-PROF-006, Lampiran C PRD (`geo-1.0.0`), dan NFR-DATA-01.

### 9.1 Versi dan kunci reproducibility

`geo.area_metric` dikunci oleh `(snapshot_id, geo_version, population_version, area_set_version, area_code, category_code)` dan menyimpan `inputs_hash`. Kombinasi aktif dicatat di `geo.geo_run`. Perhitungan dilakukan dengan SQL set-based yang deterministik, sehingga menjalankan ulang dengan kunci yang sama menghasilkan nilai identik.

### 9.2 Metrik global (per area L1–L4 × kategori, `*` = semua)

```sql
-- Hitungan dense dalam kab/kota: setiap kelurahan × kategori yang muncul di kab/kota itu (nol ikut dihitung)
WITH cnt AS (
  SELECT ea.kel_code, ec.category_code, count(*) AS n
  FROM geo.establishment_area ea
  JOIN identity.establishment_class ec USING (establishment_id)
  JOIN identity.v_establishment_public e ON e.id = ea.establishment_id
  WHERE e.business_status <> 'CLOSED_PERMANENTLY'
  GROUP BY 1, 2
), dense AS (
  SELECT a.code, a.peer_group, c.category_code, coalesce(cnt.n, 0) AS n, p.population, p.area_km2
  FROM geo.area a
  JOIN geo.area_population p ON p.code = a.code AND p.population_version = :pop
  JOIN (SELECT DISTINCT left(kel_code, 5) AS kab, category_code FROM cnt) c ON c.kab = left(a.code, 5)
  LEFT JOIN cnt ON cnt.kel_code = a.code AND cnt.category_code = c.category_code
  WHERE a.level = 4
)
), dens AS (
  SELECT code, peer_group, category_code, n * 1000.0 / nullif(population, 0) AS d FROM dense
  WHERE population > 0
)
SELECT code, category_code, d AS category_density,
       -- persentil mid-rank (BR-SCORE-09): 100 × (#nilai < d + 0,5 × #nilai = d) / n
       100.0 * ((rank() OVER w - 1) + 0.5 * count(*) OVER (PARTITION BY peer_group, category_code, d))
             / count(*) OVER (PARTITION BY peer_group, category_code)     AS competition_index
FROM dens
WINDOW w AS (PARTITION BY peer_group, category_code ORDER BY d);
```

| Metrik | Implementasi |
|---|---|
| population, population_density | Dari `area_population` (wajib `reference_period`) |
| business_count, business_density | Establishment aktif (bukan CLOSED_PERMANENTLY) per area / populasi × 1.000 |
| category_count, category_density | Seperti di atas per kategori |
| peer_group | L4: kab/kota yang sama; L3: provinsi yang sama. Bila < 10 area → naik satu level. Disimpan `peer_group`, `peer_n` |
| competition_index | Persentil mid-rank category_density dalam peer group. Low < 33, Medium 33–66, High > 66 |
| coverage | LOW bila population < 500, atau business_count < 5, atau (business_density < 0,25 × median peer **dan** population_density ≥ median peer). Median via `percentile_cont(0.5) WITHIN GROUP` per peer group |
| white_space_score | `pct(population) × (100 − pct(category_density)) / 100`, hanya bila population ≥ 1.000 dan coverage OK; selain itu NULL (Unknown) |
| activity | Persentil median `review_count` per establishment di area dalam peer group. Unknown bila tidak ada data ulasan |

Level L3/L2/L1 dihitung dengan agregasi dari L4 (hitungan) dan populasi level tersebut, bukan rata-rata persentil. Semua `pct()` Lampiran C memakai **mid-rank** (BR-SCORE-09), diimplementasikan sebagai pola window di atas dan fungsi Go yang sama untuk faktor skor (diuji silang).

### 9.3 Guardrail tampilan (FR-GEO-009)

API selalu mengembalikan `reference_period`, `coverage`, `geo_version`, dan `guardrail_codes` (`POPULATION_IS_CONTEXT`, `DENSITY_IS_SUPPLY_PROXY`, `LIMITED_DATA`). Copy guardrail dirender UI dari kode tersebut (lihat 03 UX spec).

### 9.4 Metrik tenant per ICP (dihitung dalam job scoring)

```
target_business_count(area) = establishment kandidat ICP (lolos filter wajib) di area
target_density              = target_business_count / population × 1.000
competition_ICP             = persentil mid-rank kepadatan Σ kategori ICP dalam peer group      (ICP tanpa kategori → business_density)
market_opportunity_score    = 0,35·pct(population) + 0,25·pct(target_density) + 0,15·pct(activity) + 0,25·(100 − competition_ICP)
                              (komponen Unknown di-renormalisasi; coverage LOW → Unknown)
territory_score             = 0,5·market_opportunity_score + 0,3·pct(hot_warm_count) + 0,2·avg(data_quality target)
```

Hasil ditulis ke `intel.icp_area_metric` (≤ ±91 ribu baris per ICP untuk L4+L3+L2) pada epoch yang sama dengan skor. Territory ranking (FR-GEO-003) membaca tabel ini. Hasil dirangkum dalam cache Redis per `(icp, epoch, level, parent)` selama 10 menit. Penggabungan beberapa kategori ICP menjadi `competition_ICP` adalah interpretasi SDD (TQ-09).

### 9.5 Pengiriman peta (ADR-17, FR-GEO-002)

- Saat import `area_set_version`: geometri disederhanakan per rentang zoom (z4–6 provinsi/kab, z7–10 kecamatan, z11–14 kelurahan) dengan simplifikasi *topology-aware* (tanpa celah antar polygon), lalu dibangun menjadi **PMTiles** (`msp-tiles/areas/{area_set_version}.pmtiles`). Setiap fitur hanya memuat `code`, `level`, dan `name`.
- Disajikan dari MinIO via `files.<domain>` dengan `Cache-Control: public, max-age=31536000, immutable` (data publik non-personal; CDN opsional).
- Nilai metrik: `GET /v1/areas/metrics?level=4&parent=51.71&metric=market_opportunity&icp_id=…` → `{ "values": { "51.71.01.1001": 72.4, … }, "legend": {…}, "reference_period": "2025-S1" }`. Client MapLibre GL menerapkannya lewat `feature-state` berdasarkan `code`.
- Fallback dev/debug: `GET /v1/tiles/areas/{z}/{x}/{y}.mvt` dengan `ST_AsMVT(ST_AsMVTGeom(ST_Transform(geom, 3857), ST_TileEnvelope(z, x, y)))`.
- Basemap opsional (TQ-11). Choropleth tetap berfungsi tanpa basemap.
- **Tanpa poligon batas** (Q-10, A-03, FR-DATA-008): peta turun ke layer titik/cluster (centroid establishment teragregasi per area, `GET /v1/areas/metrics?…&geometry=centroid`) + tampilan tabel. Ranking dan drilldown tidak terdampak.

---
## 10. Query & search

Memenuhi FR-SRCH-001..008, FR-PROF-001..008, FR-GEO-004, BR-CRED-02, NFR-PERF-01/02, dan ADR-05/12.

### 10.1 Projection `query.prospect_search` (global, satu baris per establishment aktif tidak ter-suppress)

```sql
CREATE TABLE query.prospect_search (
  establishment_id uuid PRIMARY KEY, company_id uuid NOT NULL,
  name text NOT NULL, name_norm text NOT NULL, aliases text[] NOT NULL,
  sector_code text, category_code text, category_conf smallint,
  prov_code text, kab_code text, kec_code text, kel_code text, location geography(Point, 4326),
  address_display text, business_status identity.business_status,
  employee_band text, revenue_band text, revenue_conf smallint,
  rating numeric(2,1), review_count integer,
  has_phone boolean, has_mobile boolean, has_email boolean, has_website boolean, has_person_contact boolean,
  best_contact_status identity.verification_status, website_type text, email_kind text,
  gap_types text[] NOT NULL DEFAULT '{}',
  dq_base smallint, contactability smallint, commercial smallint, base_rank real,   -- sort "kualitas data" + urutan kandidat
  freshness_at timestamptz, establishment_count integer, entity_seq bigint NOT NULL, bucket smallint NOT NULL,
  search_tsv tsvector NOT NULL
);
CREATE INDEX ps_tsv   ON query.prospect_search USING gin (search_tsv);
CREATE INDEX ps_trgm  ON query.prospect_search USING gin (name_norm gin_trgm_ops);
CREATE INDEX ps_geo   ON query.prospect_search (kab_code, category_code);
CREATE INDEX ps_kel   ON query.prospect_search (kel_code);
CREATE INDEX ps_gaps  ON query.prospect_search USING gin (gap_types);
CREATE INDEX ps_dq    ON query.prospect_search (dq_base DESC, establishment_id DESC);
CREATE STATISTICS ps_kab_cat (dependencies, ndistinct) ON kab_code, category_code FROM query.prospect_search;
```

- `search_tsv = setweight(to_tsvector('simple', msp_unaccent(name_norm ‖ aliases)), 'A') ‖ setweight(…label kategori id+en…, 'B') ‖ setweight(…alamat…, 'C')`. Konfigurasi `simple` dipakai karena nama bisnis adalah proper noun campuran ID/EN; stemming justru merusak. Variasi ditangani alias dan sinonim (§6.3). `msp_unaccent` adalah wrapper `IMMUTABLE` atas `unaccent` agar dapat dipakai di kolom/index.
- Rebuild penuh saat snapshot publish: shadow table → index dibuat paralel → `ALTER TABLE … RENAME` dalam satu transaksi. Perubahan kecil (refresh, merge, suppression) memakai upsert/delete per baris dan menaikkan `entity_seq`.

### 10.2 Keyword (FR-SRCH-001)

Query user dinormalisasi dengan fungsi nama yang sama (badan usaha/gelar dibuang), sinonim diekspansi, lalu:
`search_tsv @@ websearch_to_tsquery('simple', :q_expanded) OR name_norm % :q_norm` (ambang trigram 0,35 via `SET LOCAL pg_trgm.similarity_threshold`). Contoh: "klinik gigi abc" → `(klinik & gigi | dental & clinic) & abc` → menemukan "ABC Dental Clinic".

### 10.3 Pemetaan filter (FR-SRCH-002)

| Filter | SQL |
|---|---|
| Kategori / wilayah (hierarki) | `category_code = ANY(:c)`; wilayah memakai kolom level yang dipilih (`kab_code = ANY`, `kel_code = ANY`, …) |
| Band karyawan/revenue | `employee_band = ANY(:b)` |
| Ketersediaan kontak | `has_phone`, `has_email`, `has_website`, `has_mobile` (WA-likely) |
| Status verifikasi | `best_contact_status = ANY(:s)` |
| Band prioritas / rentang ICP fit | `s.band = ANY(:b)`, `s.factors[1] BETWEEN :lo AND :hi` |
| Tipe opportunity | `gap_types && :types_relevan_offering` (§8.3) |
| Freshness | `freshness_at >= :as_of - interval '<ambang BR-TRUST-04>'` |
| Status bisnis, rating | `business_status = ANY`, `rating BETWEEN` |
| "Belum ada di list saya" | `NOT EXISTS (SELECT 1 FROM action.list_item li JOIN action.list l … WHERE li.company_id = p.company_id)` (RLS) |
| Status unlock | `[NOT] EXISTS (SELECT 1 FROM usage.unlock u WHERE u.company_id = p.company_id)` (RLS) |
| Tag / assignment / status kerja (MVP1) | `EXISTS` pada tabel tenant |

### 10.4 Strategi join skor tenant × projection

```sql
SELECT p.*, s.value, s.band, s.coverage, s.limited_data, (s.input_seq < p.entity_seq) AS stale
FROM intel.score s
JOIN query.prospect_search p ON p.establishment_id = s.establishment_id
WHERE s.workspace_id = :ws AND s.icp_id = :icp AND s.epoch = :epoch AND s.is_company_best
  AND p.kab_code = ANY(:kab) AND p.category_code = ANY(:cat)
  AND (s.value, s.establishment_id) < (:cur_value, :cur_id)          -- keyset
ORDER BY s.value DESC, s.establishment_id DESC
LIMIT 51;                                                             -- 50 + 1 untuk deteksi halaman berikut
```

- Default (sort priority): plan dimulai dari index `score_rank` (sudah dibatasi kandidat ICP ≤ 200k) → nested loop ke PK projection. Filter yang sangat selektif (keyword, satu kelurahan) membuat planner memulai dari projection, lalu menggabungkan score melalui PK. Statistik + `CREATE STATISTICS` menjaga estimasi.
- Sort lain: ICP fit (`s.factors[1]`), kualitas data (`p.dq_base`), nama (`p.name_norm`). Masing-masing memakai keyset `(kunci, establishment_id)`.
- Tanpa ICP aktif: hanya projection, band `UNSCORED`, sort default `dq_base`.
- Read query diarahkan ke **db-02** (replica). Aksi yang langsung diikuti pembacaan (setelah unlock/list add) memakai primary selama 5 detik (*read-your-writes* via flag sesi).

### 10.5 Pagination (ADR-12)

Cursor = base64url(JSON `{v, k:[nilai sort, id], e: epoch, qh: hash(query+filter+icp)}`) + HMAC-SHA256 (kunci `CURSOR_HMAC_KEY`). Cursor tidak valid atau `qh` berbeda → `400 CURSOR_INVALID`. Epoch lama dipertahankan ≥ 1 jam setelah swap; setelah itu → `409 CURSOR_STALE`, dan client memulai ulang dari halaman pertama.

### 10.6 Hitungan (FR-SRCH-006)

1. `SELECT count(*) FROM (<query tanpa ORDER/LIMIT> LIMIT 10001)`. Hasil ≤ 10.000 → `{count, exact: true}`.
2. Bila 10.001 → estimasi dengan sampel deterministik: query yang sama + `AND s.bucket < 5` (5% dari hash ID) × 20, dibulatkan dua angka signifikan → `{count: 48000, exact: false}` dan UI menampilkan "±48.000".
3. Hitungan + distribusi band (facets) di-cache di Redis per `(ws, icp, epoch, filter_hash)` selama 5 menit.

### 10.7 Profil & masking (FR-PROF-*, BR-CRED-02, FR-AUD-003)

`query.Service.GetProfile(ctx, companyID, icpID)` merakit: company + establishment (≤ 50, terurut skor) + contact point (RLS global menyembunyikan yang ter-suppress) + digital/commercial + gap→opportunity (§8.3) + konteks area + skor/penjelasan ringkas + overlay tenant (upload, catatan, status, tag). Masking dilakukan **di server** (BR-CRED-02): tanpa unlock, **semua** nilai contact point (telepon, WhatsApp, email) dan **nama person** diganti format ter-mask (`+62 812-****-7890`, `h****@abcdental.co.id`, `A*** S***`) sehingga nilai asli tidak pernah dikirim. Nama company, kategori, alamat, website, dan role person tetap terlihat. Kontak dari upload workspace sendiri tidak di-mask. Bagian global di-cache per `(company_id, entity_seq)` selama 1 jam; overlay tenant tidak di-cache. Membuka profil yang belum pernah dibuka di siklus ini mencatat `usage.profile_view` dan dicek terhadap kuota (BR-CRED-01, §14.4). Target p95 ≤ 500 ms (NFR-PERF-02).

### 10.8 Rencana performa (NFR-PERF-01: p95 ≤ 800 ms, p99 ≤ 2 s, 5 juta baris, 50 user konkuren)

- Working set projection ± 3,5 GB + index ± 3,5 GB + partisi skor workspace aktif → muat di `shared_buffers`/page cache db-02 (256 GB).
- `statement_timeout = 3s` untuk endpoint search, `work_mem` 64 MB per sesi search, dan maksimal 20 koneksi search di PgBouncer.
- Uji beban k6 di staging dengan dataset sintetis 5 juta baris + 200 workspace × 3 ICP (lihat `05-test-strategy.md`). Bila p95 terlampaui: (1) partial index per band HOT/WARM, (2) materialisasi `company_best` per epoch, (3) pindah projection ke OpenSearch (V2, ADR-05).

### 10.9 Private prospect (ADR-23)

Bila `include_private=true` (toggle "Sertakan data upload"), search menjalankan `UNION ALL` antara hasil global dan `query.private_prospect_search` (RLS, ≤ 200k baris per workspace) dengan kunci sort yang sama. Keyset diterapkan pada kedua sisi, lalu digabung dengan `ORDER BY … LIMIT 51`. ID private prospect berprefiks `upr_` (upload row).

---

## 11. API design

Memenuhi MVP Pack §12 (diperluas), FR-AUTH-004, FR-BILL-001/003/004, FR-EXP-*, dan ADR-08/11/12/25.

### 11.1 Konvensi

| Aspek | Aturan |
|---|---|
| Basis | REST JSON, prefix `/v1`. Web: `https://app.<domain>/api/v1` (same-origin). Integrasi (V1): `https://api.<domain>/v1` |
| Auth | Web: cookie sesi `__Host-msp_sess` (HttpOnly, Secure, SameSite=Lax) + header CSRF `MSP-CSRF` (double-submit) untuk metode non-GET. API key (V1): `Authorization: Bearer msp_live_…` |
| Workspace | Header `MSP-Workspace: wsp_…` wajib untuk endpoint tenant, divalidasi terhadap membership. API key terikat satu workspace (header harus cocok bila dikirim) |
| ID & waktu | ID berprefiks (ADR-14). Waktu ISO-8601 UTC. Angka uang dalam integer rupiah |
| Error | RFC 9457 `application/problem+json` + `code` + `correlation_id` (§11.2) |
| Pagination | Cursor (`?cursor=&limit=` maks 100) → `{items, next_cursor, count?}` |
| Idempotensi | Header `Idempotency-Key` (UUID/ULID, ≤ 64 karakter) **wajib** di semua POST berbayar dan disarankan di POST lainnya (§11.3) |
| Async | `202 Accepted` + `Location: /v1/jobs/{id}` (atau resource spesifik) + body job `{id, type, status, progress{done,total}, result, error}` |
| Rate limit | Token bucket Redis per user + per workspace: search 60/mnt/user, profil 120/mnt/user, default 600/mnt/workspace, export 10/jam/workspace. Header `RateLimit-Policy`, `RateLimit`, `Retry-After` |
| ETag | Opsional: `ETag` pada profil (hash `entity_seq`, epoch, status unlock) → `If-None-Match` → 304. `If-Match` pada PUT ICP/Offering mencegah *lost update* |
| Lokal | `Accept-Language: id-ID` (default) / `en-US` untuk `title` error dan rationale |
| Korelasi | `MSP-Correlation-Id` diterima/dibangkitkan dan dikembalikan di setiap respons |
| Kontrak | Setiap operasi OpenAPI memiliki `x-msp-permission` dan `x-msp-billable`, dipakai test matriks RBAC (§15.2) |
| Implementasi | `net/http` + handler oapi-codegen strict-server. Middleware (sesi, workspace, RBAC dari `x-msp-permission`, entitlement, idempotensi, rate limit, OTel) membaca metadata operasi hasil generate |

### 11.2 Katalog error (`code` stabil)

| HTTP | `code` | Kapan |
|---|---|---|
| 400 | `VALIDATION_FAILED` (422 untuk body), `CURSOR_INVALID`, `WORKSPACE_REQUIRED` | Input tidak valid |
| 401 | `UNAUTHENTICATED`, `SESSION_EXPIRED`, `API_KEY_REVOKED` | — |
| 402 | `INSUFFICIENT_CREDITS`, `INSUFFICIENT_AI_CREDITS` | Saldo kurang (FR-BILL-004) |
| 403 | `PERMISSION_DENIED`, `EMAIL_NOT_VERIFIED`, `MFA_REQUIRED`, `ENTITLEMENT_EXCEEDED`, `EXPORT_NOT_ALLOWED_FOR_ROLE` | RBAC/plan (FR-AUTH-004, FR-BILL-003) |
| 404 | `NOT_FOUND` | Termasuk resource milik workspace lain (tidak membocorkan keberadaan) |
| 308 | — | ID ter-merge → `Location` ID baru |
| 409 | `CONFLICT`, `IDEMPOTENCY_IN_PROGRESS`, `CURSOR_STALE`, `VERSION_MISMATCH` | — |
| 413/422 | `UPLOAD_TOO_LARGE`, `UPLOAD_INVALID`, `WEBHOOK_URL_REJECTED` (SSRF), `IDEMPOTENCY_KEY_REUSED` | — |
| 429 | `RATE_LIMITED`, `FAIR_USE_EXCEEDED` | — |
| 502/503/504 | `AI_GROUNDING_FAILED`, `AI_UNAVAILABLE`, `FEATURE_DISABLED` (kill switch), `DEPENDENCY_TIMEOUT` | — |

```json
{ "type": "https://docs.mesinprospek.id/errors/insufficient-credits", "title": "Credit tidak cukup", "status": 402,
  "code": "INSUFFICIENT_CREDITS", "detail": "Butuh 120 credit, saldo tersedia 80.",
  "instance": "/v1/exports", "correlation_id": "c0f3…", "required": 120, "available": 80 }
```

### 11.3 Idempotensi (BR-CRED-04, FR-BILL-001)

`platform.idempotency_record(workspace_id, key, method_path, request_hash, status, response_code, response_body, created_at)` dengan PK `(workspace_id, key)`, kedaluwarsa 24 jam. Request yang sama + key yang sama → respons tersimpan diputar ulang dengan header `Idempotent-Replayed: true`. Key yang sama dengan body berbeda → 422 `IDEMPOTENCY_KEY_REUSED`. Request masih berjalan → 409 `IDEMPOTENCY_IN_PROGRESS`. Usage event per record memakai kunci turunan `sha256(key ‖ company_id)`, dan `UNIQUE(workspace_id, company_id)` pada `unlock` menjamin tidak ada tagihan ganda walau key berbeda.

### 11.4 Objek seleksi (FR-SRCH-007, FR-EXP-001/004)

```json
{ "selection": { "type": "ids", "ids": ["cmp_…"] } }
{ "selection": { "type": "query", "query": { "q": "klinik gigi", "filters": { … }, "icp_id": "icp_…" }, "max": 5000 } }
{ "selection": { "type": "list", "list_id": "lst_…" } }      // juga "segment"
```

Selection `query` dievaluasi ulang di server dan dibatasi `max` serta batas plan (FR-EXP-006). `POST /v1/selections/preview` mengembalikan jumlah, jumlah yang sudah di-unlock, biaya credit, dan batas (FR-BILL-004).

### 11.5 Katalog endpoint

Role = role minimum (urutan Viewer < Member < Manager < Admin < Owner, mengikuti PRD §8.1.1). B = billable.

| Method | Path | Tujuan | FR | B | Role |
|---|---|---|---|---|---|
| GET | `/v1/auth/login`, `/v1/auth/callback` | Mulai/selesaikan login (adapter D-20) → sesi | AUTH-001 | — | publik |
| POST | `/v1/auth/logout`, `/v1/auth/logout-all` | Akhiri sesi ini / semua sesi | AUTH-005 | — | user |
| GET | `/v1/me` · PATCH `/v1/me/preferences` | Profil user, membership, preferensi (ICP aktif, bahasa, zona waktu) | AUTH-002 | — | user |
| GET/POST | `/v1/workspaces` | Daftar workspace saya / buat workspace | AUTH-002 | — | user |
| GET/PATCH | `/v1/workspace` | Detail/pengaturan workspace (mis. `member_can_export`) | AUTH | — | Viewer/Admin |
| DELETE | `/v1/workspace` | Minta penutupan (tenggang 30 hari); POST `/v1/workspace/restore` membatalkan | AUD-005 | — | Owner |
| GET · PATCH · DELETE | `/v1/members`, `/v1/members/{id}` | Anggota, ubah role, keluarkan | AUTH-003/005 | — | Admin |
| GET · POST · DELETE | `/v1/invitations`, `/v1/invitations/{id}` | Undangan (kedaluwarsa 7 hari) | AUTH-003 | — | Admin |
| POST | `/v1/invitations/{token}/accept` | Terima undangan | AUTH-003 | — | user |
| GET · POST · DELETE | `/v1/api-keys` (V1) | Kelola API key (ditampilkan sekali) | AUTH-007 | — | Admin |
| GET | `/v1/catalog/{offering-tags,categories,opportunity-types,areas-levels}` | Katalog statis berversi | ICP-001 | — | Viewer |
| GET · POST · PATCH · DELETE | `/v1/offerings`, `/v1/offerings/{id}` | Kelola Offering | ICP-001 | — | Manager |
| GET · POST | `/v1/icps` | Daftar/buat ICP | ICP-002/006 | — | Viewer/Manager |
| GET · PUT · DELETE | `/v1/icps/{id}` | Detail / versi baru / arsip | ICP-002 | — | Viewer/Manager |
| GET | `/v1/icps/{id}/versions`, `/v1/icps/{id}/scoring-status` | Riwayat versi; progres skor | ICP-002, NFR-PERF-05 | — | Viewer |
| GET | `/v1/score-presets` | Preset bobot per Offering | ICP-004 | — | Viewer |
| GET | `/v1/prospects` | Search + filter + sort + cursor + count | SRCH-001..006 | — | Viewer |
| GET | `/v1/prospects/facets` | Distribusi band/kategori/wilayah untuk query | SRCH-002, SEG-002 | — | Viewer |
| GET | `/v1/prospects/{id}` | Profil lengkap (company `cmp_` / `est_` / `upr_`) | PROF-001..008 | fair-use | Viewer |
| GET | `/v1/prospects/{id}/establishments` | Daftar cabang + konteks area | PROF-005 | — | Viewer |
| GET | `/v1/prospects/{id}/score?icp_id=` | Penjelasan skor | PROF-002, INTL-008 | — | Viewer |
| GET | `/v1/prospects/{id}/opportunities?icp_id=` | Opportunity relevan + rationale | INTL-007 | — | Viewer |
| GET | `/v1/prospects/{id}/claims/{field}/evidence` · GET `/v1/evidence/{id}` | Evidence drawer | PROF-004, DATA-006 | — | Viewer |
| POST | `/v1/prospects/{id}/refresh` | Re-verify + re-score (job) | MON-001 | 1 credit | Member |
| POST | `/v1/prospects/{id}/feedback` | Feedback opportunity/skor | INTL-010 | — | Member |
| POST | `/v1/prospects/{id}/reports` | Laporkan data salah | DATA-014 | — | Member |
| PUT | `/v1/prospects/{id}/status` | Status kerja ringan | SEG-006 | — | Member |
| GET | `/v1/prospects/{id}/activity` | Timeline aktivitas | PROF-010 | — | Viewer |
| GET · POST · PATCH · DELETE | `/v1/prospects/{id}/notes`, `/v1/notes/{id}` | Catatan | SEG-004 | — | Member |
| POST | `/v1/selections/preview` | Jumlah + biaya + batas sebelum aksi massal | BILL-004, SRCH-007 | — | Member |
| POST · GET | `/v1/unlocks`, `/v1/unlocks?company_id=` | Unlock kontak (bulk ≤ 1.000) / status | BILL, BR-CRED-02 | 1/company | Member |
| GET · POST | `/v1/saved-searches` · DELETE `/{id}` | Saved search | SRCH-005 | — | Member |
| GET · POST · PATCH · DELETE | `/v1/lists`, `/v1/lists/{id}` | List statis | SEG-001 | — | Viewer (GET) / Member |
| GET · POST · DELETE | `/v1/lists/{id}/items` | Isi list (selection) | SEG-001 | — | Viewer (GET) / Member |
| GET · POST · PATCH · DELETE | `/v1/segments`, `/v1/segments/{id}`; GET `/{id}/stats` | Segment dinamis + statistik | SEG-002, GEO-005 | — | Viewer (GET) / Member |
| GET · POST · PATCH · DELETE | `/v1/tags`, `/v1/tags/{id}`; POST `/{id}/apply`, `/{id}/remove` | Tag + bulk tag | SEG-003 | — | Member |
| POST · GET · DELETE | `/v1/assignments`, `/v1/assignments?assignee=me`, `/{id}` | Assign prospek/list, "Tugas saya" | SEG-005 | — | Manager (GET: Member) |
| GET | `/v1/areas?parent=&level=` · `/v1/areas/{code}` | Hierarki wilayah; detail + metrik global (kategori) | GEO-001, PROF-006 | — | Viewer |
| GET | `/v1/areas/metrics` | Nilai choropleth per area (global/ICP) | GEO-002 | — | Viewer |
| GET | `/v1/areas/ranking` · `/v1/areas/{code}/breakdown` | Territory ranking + rincian | GEO-003 | — | Viewer |
| GET | `/v1/tiles/areas/{z}/{x}/{y}.mvt` | Fallback tile dinamis (dev) | GEO-002 | — | Viewer |
| POST · GET · DELETE | `/v1/uploads`, `/v1/uploads/{id}` | Buat upload (presigned) / status / hapus | DATA-012 | — | Member |
| POST · PUT | `/v1/uploads/{id}/complete`, `/{id}/mapping`, `/{id}/process` | Selesai unggah, mapping, proses | DATA-012 | — | Member |
| GET · POST | `/v1/uploads/{id}/rows?status=`, `/{id}/report`, `/{id}/rows/{row}/link` | Baris, laporan, konfirmasi link | DATA-012 | — | Member |
| POST | `/v1/exports/preview` · POST `/v1/exports` | Pratinjau biaya; buat job export | EXP-001/002/006 | auto-unlock | Member* |
| GET · POST · DELETE | `/v1/exports`, `/{id}`, `/{id}/download` (302), `/{id}/retry`, `/{id}` (cancel) | Status, unduh 24 jam, retry | EXP-001 | — | Member* |
| GET · POST · PATCH · DELETE | `/v1/webhooks`, `/v1/webhooks/{id}` | Endpoint webhook + langganan event | EXP-003 | — | Admin |
| POST | `/v1/webhooks/{id}/test`, `/{id}/rotate-secret` | Kirim `webhook.test` (MVP Pack: `/v1/webhooks/test`); rotasi secret | EXP-003 | — | Admin |
| GET · POST | `/v1/webhooks/{id}/deliveries`, `/v1/webhook-deliveries/{id}`, `/{id}/redeliver` | Log pengiriman, kirim ulang DLQ | EXP-003 | — | Admin |
| POST | `/v1/webhook-sends` | Kirim seleksi ke endpoint | EXP-004 | auto-unlock | Member* |
| POST · GET | `/v1/ai/briefs`, `/v1/ai/briefs/{id}`, `/v1/prospects/{id}/briefs` | Generate (202) / ambil brief & sales angle | AI-001..003 | 1 ai_credit | Member (POST) / Viewer (GET) |
| POST | `/v1/ai/briefs/{id}/feedback` | 👍/👎 + alasan | AI-007 | — | Member |
| GET | `/v1/usage` · `/v1/usage/events` · `/v1/credits/ledger` | Ringkasan pemakaian; event; ledger | BILL-002/007 | — | Manager / Admin |
| GET | `/v1/entitlements` | Batas plan, saldo credit, flag efektif (gating UI) | BILL-003, OPS-007 | — | Viewer |
| GET | `/v1/audit-log` | Penampil audit | AUD-002 | — | Admin |
| GET | `/v1/jobs`, `/v1/jobs/{id}` | Job export/upload/refresh: milik sendiri (Member) / semua job workspace (Manager+) | PRD §8.1.1 | — | Member |
| POST | `/v1/public/data-subject-requests` | Intake DSR (tanpa login; captcha + verifikasi email) | AUD-004 | — | publik |
| GET · POST · PATCH | `/v1/ops/sources`, `/v1/ops/snapshots`, `/{id}/validate`, `/{id}/report`, `/{id}/publish`, `/{id}/reject` | Source Registry & snapshot | DATA-001 | — | ops |
| POST | `/v1/ops/reference/{areas,boundaries,population,crosswalk}/imports` | Import referensi wilayah | DATA-008 | — | ops |
| GET · POST | `/v1/ops/pipeline-runs`, `/{id}`, `/v1/ops/dlq`, `/v1/ops/dlq/{id}/redrive` | Pipeline health, re-run stage, DLQ | OPS-002 | — | ops |
| GET | `/v1/ops/health/{data,pipeline,intelligence,cost}` | Dashboard internal | OPS-001..004 | — | ops |
| GET · POST | `/v1/ops/merge-candidates`, `/{id}/decision`, `/v1/ops/entities/{id}/split` | Review merge/split | OPS-005, DATA-004 | — | ops |
| GET · POST · DELETE | `/v1/ops/suppressions` | Suppression global | AUD-003 | — | ops |
| GET · PATCH | `/v1/ops/data-subject-requests`, `/v1/ops/data-reports` | Antrean DSR & laporan data | OPS-008, AUD-004 | — | ops |
| GET · POST · PUT | `/v1/ops/workspaces`, `/{id}/credit-grants`, `/{id}/plan`, `/v1/ops/support-access` | Admin tenant, grant credit, akses support | OPS-006, BILL-006 | — | ops |
| GET · PUT | `/v1/ops/feature-flags/{key}` · GET/POST `/v1/ops/score-models` | Flag & kill switch; versi model | OPS-007, INTL-009 | — | ops |

\* Member hanya bila setting workspace `member_can_export = true` (PRD §8.1.1). Endpoint `/v1/ops/*` hanya untuk akun internal dengan MFA (FR-AUTH-006), memakai audience sesi terpisah, dan selalu diaudit. `GET /healthz` dan `GET /readyz` berada di luar `/v1`.

Pemetaan MVP Pack §12: `POST /v1/prospects/export` → `POST /v1/exports`; `POST /v1/webhooks/test` → `POST /v1/webhooks/{id}/test` (URL baru divalidasi saat create); `GET /v1/usage` tetap.

---
## 12. Events & webhooks

Memenuhi FR-EXP-003/004/005, MVP Pack §12 (event rules), NFR-SEC-05, RG-3, dan ADR-07/20.

### 12.1 Envelope (`contracts/events/envelope-1.0.json`)

```json
{
  "id": "evt_01j9x…", "type": "prospect.sent", "version": "1.0",
  "occurred_at": "2026-10-10T04:05:06Z", "workspace_id": "wsp_01j8…",
  "subject": { "type": "prospect", "id": "cmp_01j9…" },
  "model_version": "priority-1.2.0+default", "correlation_id": "c0f3…",
  "data": { "prospect": { "id": "cmp_01j9…", "name": { "value": "ABC Dental", "claim_type": "OBSERVED", "confidence": 92 },
            "contacts": [ { "type": "PHONE", "value": "+6281234567890", "status": "LIKELY_VALID", "verified_at": "2026-10-09T02:00:00Z", "claim_type": "OBSERVED" } ],
            "score": { "value": 84, "band": "HOT", "coverage": 0.95 }, "opportunities": [ { "type": "OPP-WEB-NONE", "score": 64 } ] },
            "batch_id": "job_01j9…" }
}
```

ID immutable, ada `workspace_id`, konsumen men-dedupe berdasarkan `id`, dan `model_version` disertakan bila menyangkut intelligence (MVP Pack §12). Nilai sensitif tidak pernah ditulis ke log (§15.6). Payload webhook hanya berisi kontak yang sudah di-unlock (atau di-unlock otomatis oleh aksi kirim) dan tidak pernah berisi record ter-suppress (FR-EXP-005).

### 12.2 Katalog event

| Event | Pemicu | Webhook? | Rilis |
|---|---|---|---|
| `prospect.sent` | Aksi "Kirim ke webhook" (per prospek) | Ya | MVP1 |
| `prospect.refreshed` | Refresh prospek selesai | Ya | MVP1 |
| `prospect.scored` | Skor prospek di list/assignment workspace berubah band (ringkas, batch per job) | Ya | MVP1 |
| `icp.scoring_completed` | Epoch baru aktif | Ya | MVP1 |
| `export.completed` / `export.failed` | Job export selesai/gagal | Ya | MVP1 |
| `upload.completed` | Upload selesai diproses | Ya | MVP1 |
| `list.items_added`, `assignment.created` | Aksi organize | Ya | MVP1 |
| `credit.balance_low` | Saldo < 10% grant bulanan | Ya | MVP1 |
| `suppression.applied` | Record yang pernah diterima workspace (export/webhook) di-suppress. Payload hanya ID + jenis, tanpa nilai kontak (FR-AUD-003) | Ya | MVP1 |
| `usage.recorded` | Usage event tercatat | Internal (Ya di V1) | MVP1 |
| `webhook.test` | Tombol uji | Ya (ke endpoint yang diuji saja) | MVP1 |
| `webhook.delivered` | Delivery sukses | Internal (metrik/log) | MVP1 |
| `opportunity.detected`, `signal.detected` | Diff snapshot untuk prospek yang di-watch | Ya | V1 |
| `prospect.created`, `prospect.updated` | Perubahan entity global | Internal saja (volume global tidak dikirim ke tenant) | MVP0 |
| `entity.merged`, `snapshot.published`, `suppression.created`, `unlock.created` | Koordinasi antar modul | Internal | MVP0/1 |

### 12.3 Event via River (outbox transaksional, ADR-07)

```go
err := tenancy.WithTx(ctx, ws, func(tx pgx.Tx) error {
    if err := usageRepo.InsertUnlock(ctx, tx, u); err != nil { return err }             // perubahan domain
    evt := events.New("unlock.created", ws, subject, payload)
    return events.Emit(ctx, tx, evt) // INSERT platform.event + riverClient.InsertTx(ctx, tx, DispatchArgs{EventID: evt.ID}, nil)
})
```

- Event dan job `event.dispatch` di-commit **atomik** dengan perubahan domain. Rollback → tidak ada event. Tidak ada dispatcher polling terpisah: River membangunkan worker via `LISTEN/NOTIFY` (latensi sub-detik).
- Worker `event.dispatch`: (1) meng-*enqueue* job per handler internal yang terdaftar (`UniqueOpts` per `(event_id, consumer)` → dedupe); (2) untuk event yang dapat di-webhook, membuat `webhook_delivery(PENDING)` untuk setiap endpoint aktif yang berlangganan + job `webhook.deliver`. Semuanya dalam satu transaksi.
- Handler internal harus idempoten (upsert pada natural key). `platform.event` disimpan 30 hari (sumber payload untuk *redeliver* dan audit pengiriman). Job `completed` dibersihkan River setelah 24 jam.

### 12.4 Pengiriman webhook

| Aspek | Desain |
|---|---|
| Request | `POST` JSON, `User-Agent: MesinProspek-Webhooks/1.0`, header `MSP-Event-Id`, `MSP-Event-Type`, `MSP-Delivery-Id`, `MSP-Signature` |
| Signature | `MSP-Signature: t=<unix_ts>,v1=<hex(HMAC-SHA256(secret, "<t>.<raw_body>"))>`. Selama rotasi (24 jam) dikirim dua `v1` (secret lama + baru) |
| Replay | Penerima menolak bila selisih `now − t` > 300 s dan men-dedupe `MSP-Event-Id` (didokumentasikan + contoh kode verifikasi) |
| Sukses | 2xx dalam 10 detik. Body respons dibaca maksimal 64 KB lalu dibuang |
| Retry | Semua non-2xx/timeout/error jaringan kecuali `410 Gone`. Jadwal: **1 m, 5 m, 30 m, 2 j, 6 j, 12 j** (±10% jitter) → 7 percobaan, ±20 j 36 m (≤ 24 jam, FR-EXP-003), diimplementasikan dengan `NextRetry` kustom + `MaxAttempts` 7 pada job River `webhook.deliver`. `429` menghormati `Retry-After` (≤ 1 jam) dalam jadwal yang sama |
| DLQ | Percobaan habis (job `discarded`) → delivery `DLQ` + `release` hold credit. Kirim ulang manual (`/redeliver`) membuat percobaan baru tanpa tagihan baru |
| Kesehatan endpoint | ≥ 20 delivery berturut-turut masuk DLQ → `DISABLED_FAILING` + email ke Admin. `410` → langsung disable |
| Log | `webhook_delivery` menyimpan attempt, kode respons, latensi, error (tanpa body request), retensi 30 hari (FR-EXP-003) |
| Kill switch | Flag `webhook.delivery` off → job di-*snooze* (`river.JobSnooze`), delivery tetap `PENDING`. Bila umur > 24 jam saat dilanjutkan → DLQ |

### 12.5 Kontrol SSRF & egress (Crawler §12 diadopsi, NFR-SEC-05, RG-7)

Tiga lapis, karena dedicated server tidak memiliki security group cloud (D-19):

1. **Validasi aplikasi** (`internal/platform/netguard`) saat endpoint dibuat **dan** sebelum setiap delivery:
   - Hanya `https://`. Tanpa userinfo (`user:pass@`). Host wajib berupa nama (tanpa IP literal). Tolak `localhost`, `*.local`, `*.internal`, `*.localhost`, dan domain milik MSP sendiri.
   - Port 443 atau port ≥ 1024 yang tidak ada di daftar blok (22, 25, 2375/2376, 3306, 5432, 6379, 9000, 9200, 11211, …).
   - Resolve A/AAAA lewat resolver internal. **Semua** IP hasil resolve harus global: blok 0.0.0.0/8, 10/8, 100.64/10, 127/8, 169.254/16 (termasuk metadata 169.254.169.254), 172.16/12, 192.0.0/24, 192.0.2/24, 192.168/16, 198.18/15, 198.51.100/24, 203.0.113/24, 224/4, 240/4, ::1, ::/128, fc00::/7, fe80::/10, ff00::/8, 2001:db8::/32. IPv4-mapped/NAT64 (`::ffff:0:0/96`, `64:ff9b::/96`) diperiksa alamat tertanamnya. Subnet LAN server dan IP publik server MSP sendiri juga diblok.
   - **Pemeriksaan saat connect:** `http.Transport` memakai `net.Dialer{Control: netguard.Check}` yang memvalidasi IP tujuan pada **setiap koneksi** (termasuk koneksi baru akibat redirect pada klien yang mengizinkan redirect). Dengan begitu DNS rebinding antara validasi dan koneksi tidak lolos. Di produksi, satu-satunya tujuan dial adalah Smokescreen (alamat proxy di-allowlist di `Control`), dan Smokescreen yang memeriksa IP akhir. Mode direct (dev/fallback) menjalankan pemeriksaan penuh di `Control`.
   - Redirect **dimatikan** untuk webhook (`CheckRedirect` → `http.ErrUseLastResponse`; 3xx = gagal, dicatat). TLS diverifikasi (minimal TLS 1.2). Timeout connect 3 s, total 10 s. Respons maksimal 64 KB. Tanpa dekompresi.
2. **Egress proxy Smokescreen**: `worker-webhook` hanya dapat keluar lewat `smokescreen:4750` (HTTP CONNECT). Smokescreen me-resolve sendiri dan menolak IP privat/terlarang (mencegah DNS rebinding/TOCTOU). ACL per role: `webhook` = internet publik minus daftar blok; `ai` = allowlist host provider LLM; `mail` = host SMTP provider.
3. **nftables di host** (chain `DOCKER-USER`/forward): subnet network `webhook-net` hanya boleh ke `smokescreen`, PgBouncer `db-01:6432`, dan Redis. Container Smokescreen hanya boleh keluar ke internet publik, dengan blok eksplisit untuk RFC1918, CGNAT, link-local/metadata, loopback, multicast, dan subnet LAN DC. Container lain default-deny egress internet, kecuali Unbound (53/udp+tcp) dan Traefik (ACME).

Fixture uji SSRF (05): URL ke IP privat, nama yang resolve ke 127.0.0.1, rebinding, redirect ke metadata, IPv6-mapped, port terlarang. Semua harus ditolak.

---

## 13. AI subsystem

Memenuhi FR-AI-001..007, NFR-PERF-06, NFR-PRIV-04, NFR-COST-01, D-14, D-15, dan ADR-09.

### 13.1 Abstraksi provider

```go
type LLMProvider interface {
    Name() string
    Generate(ctx context.Context, req LLMRequest) (LLMResponse, error)
}
// LLMRequest: ModelAlias ("brief-default" | "classify-default"), System, Messages,
//             OutputSchema (JSON Schema → output terstruktur bila provider mendukung), MaxTokens, Timeout, Metadata
// LLMResponse: Content (JSON), StopReason, Usage{InputTokens, OutputTokens, CachedTokens}, ProviderModel, Latency
```

- Alias → ID model diatur lewat konfigurasi (`LLM_MODEL_BRIEF`, `LLM_MODEL_CLASSIFY`). Usulan default (D-15, Open): model Claude **kelas Sonnet** untuk brief/sales angle dan **kelas Haiku** untuk klasifikasi kategori. ID model tidak di-hardcode.
- Adapter: `anthropic` (usulan default, SDK Go resmi provider; memakai output terstruktur berbasis JSON Schema dan prompt caching untuk prefix stabil), `fake` (test/CI, deterministik), dan provider lain kelak. Retry dilakukan adapter untuk 429/5xx (maksimal 2) dengan budget waktu total.
- Egress ke provider melalui Smokescreen (role `ai`, allowlist host). Data yang keluar dari Indonesia hanya data level company (D-14). Status DPA/region provider = TQ-02.
- Prefix prompt (system + instruksi + skema) dijaga byte-stabil agar cache prompt provider efektif. Data per prospek selalu diletakkan sesudah prefix.

### 13.2 Prompt registry

`apps/api/internal/ai/prompts/{brief,sales_angle}/<semver>/` (di-*embed* via `embed.FS`) berisi `system.md`, `user.tmpl` (`text/template`), `output.schema.json`, dan `meta.yaml` (alias model, max_tokens, bahasa yang didukung, versi validator). Versi = `brief-1.0.0`, `angle-1.0.0`. Versi aktif dipilih lewat config. Perubahan prompt = versi baru + eval offline lulus (§13.7) sebelum diaktifkan. `ai_artifact` menyimpan `prompt_version` dan `provider_model` (label "Dibuat AI" + versi di UI, FR-AI-001).

### 13.3 Input builder & redaksi (FR-AI-005, NFR-PRIV-04)

Fakta dibangun dari profil kanonik dan diberi alias pendek `f1..fn` yang dipetakan ke `claim_ref`/`evidence_ids`:

```json
{ "language": "id-ID", "offering": { "name": "Jasa website klinik", "tags": ["website_dev"] },
  "facts": [
    { "ref": "f1", "text": "Nama: ABC Dental", "claim_type": "OBSERVED", "confidence": 92 },
    { "ref": "f2", "text": "Kategori: Klinik gigi; 3 cabang di Kota Denpasar", "claim_type": "DERIVED" },
    { "ref": "f3", "text": "Website: tidak ada pada data sumber (snapshot 2026.10)", "claim_type": "OBSERVED" },
    { "ref": "f4", "text": "68% klinik gigi di Kota Denpasar memiliki website", "claim_type": "DERIVED" },
    { "ref": "f5", "text": "Band revenue (perkiraan): R2, confidence 55", "claim_type": "INFERRED" },
    { "ref": "f6", "text": "Kontak decision-maker: ada (peran: Pemilik)", "claim_type": "OBSERVED" } ] }
```

- **Tidak pernah dikirim:** nilai contact point apa pun (telepon/email/WA, company maupun person), nama person (person = peran saja), catatan tenant, dan isi upload. Unlock tidak mengubah aturan ini.
- **Test redaksi otomatis:** setiap prompt yang dirakit di test/CI dipindai regex email/telepon (format ID & E.164) dan token nama person dari fixture. Bila ada temuan, test gagal (FR-AI-005). Di produksi, pemindai yang sama berjalan sebelum panggilan; bila ditemukan → abort + alert.

### 13.4 Skema output

```json
{ "type": "object", "required": ["language", "sections"], "additionalProperties": false,
  "properties": {
    "language": { "enum": ["id-ID", "en-US"] },
    "sections": { "type": "array", "items": { "type": "object", "required": ["heading", "sentences"],
      "properties": { "heading": { "type": "string" },
        "sentences": { "type": "array", "items": { "type": "object", "required": ["text", "citations"],
          "properties": { "text": { "type": "string", "maxLength": 400 },
                          "citations": { "type": "array", "minItems": 1, "items": { "type": "string", "pattern": "^f[0-9]+$" } },
                          "kind": { "enum": ["FACT", "HYPOTHESIS"] } } } } } } } } }
```

### 13.5 Grounding validator (FR-AI-004)

Deterministik, berversi (`grounding-1.0.0`):
1. Lolos JSON Schema. Bahasa sesuai permintaan (rasio stopword).
2. Setiap kalimat punya ≥ 1 sitasi, dan semua sitasi ada di tabel fakta.
3. **Angka** (bilangan, persen, rupiah, tahun) di kalimat harus muncul (ternormalisasi) di fakta yang dikutip.
4. **Nama diri/wilayah** (token kapital di luar daftar stopword/allowlist) harus muncul di fakta yang dikutip.
5. Kalimat yang hanya mengutip fakta INFERRED atau berjenis HYPOTHESIS wajib memakai kata hedging ("kemungkinan", "diperkirakan", "hipotesis") (BR-TRUST-05). Kata jaminan ("pasti", "dijamin") dilarang (BR-OPP-05).
6. Tidak ada pola email/telepon/URL di output.

Gagal → regenerate **sekali** dengan daftar pelanggaran ditambahkan ke prompt. Gagal lagi → `AI_GROUNDING_FAILED`, hold ai_credit di-*release* (refund), artifact tersimpan berstatus `FAILED` untuk eval. Metrik target: lolos ≥ 98% (PRD §5).

### 13.6 Alur, cache, biaya

```
POST /v1/ai/briefs {prospect_id, icp_id, kind, language, regenerate}  (+Idempotency-Key)
  -> flag ai.generation + cap workspace -> key = (workspace_id, icp_id/offering_id, input_hash, prompt_version, language)
     input_hash = sha256(alias→model ‖ kind ‖ facts kanonik ‖ offering)
  -> cache hit (ai_artifact SUCCEEDED, workspace sama, < 7 hari, regenerate=false) -> 200, tanpa biaya   (FR-AI-003)
  -> else: hold 1 ai_credit -> 202 {brief_id} -> worker ai: generate -> validate (≤ 1 regenerate) -> capture / release
```

- `ai.ai_artifact(id brf_, workspace_id, company_id, establishment_id, icp_id, kind, language, input_hash, prompt_version, provider_model, output jsonb, citations_map jsonb, status, created_at, expires_at)`. Cache **tidak pernah** dibagi lintas workspace (FR-AI-003). Ini diuji dengan test isolasi cache.
- `ai.ai_call(id, workspace_id, artifact_id, attempt, provider, model, input_tokens, output_tokens, cached_tokens, cost_micro_usd, latency_ms, status, error_code)`. Tabel harga per model ada di config (FR-OPS-004).
- Timeout 30 s per percobaan, total ≤ 60 s. Target p95 ≤ 20 s (NFR-PERF-06) dicapai dengan `max_tokens` ±1.200 dan prompt ringkas. Client melakukan polling `GET /v1/ai/briefs/{id}`.
- Artifact ditandai `INVALIDATED` bila prospek di-merge/di-suppress atau claim yang dikutip berubah (`entity_seq`).

### 13.7 Kontrol biaya & eval (FR-AI-006, NFR-COST-01, R-05)

- Lapisan batas: (1) saldo `ai_credit` (entitlement plan); (2) **cap biaya bulanan per workspace** (USD, config per plan) → `FEATURE_DISABLED` untuk workspace itu; (3) **budget harian global** → kill switch otomatis `ai.generation` + alert; (4) kill switch manual oleh ops (audit).
- Eval offline (`apps/api/internal/ai/eval/`, `go test -tags=eval`): ±100 profil golden (berbagai kelengkapan data, termasuk kasus Unknown). Metrik: lolos validator, presisi sitasi (label manual pada sampel), kebocoran redaksi = 0, kepatuhan hedging. Dijalankan pada setiap perubahan prompt/validator (PR memakai respons rekaman; nightly memakai provider asli dengan cap biaya).
- Online: feedback 👍/👎 (FR-AI-007) dan sampel 2% brief untuk review manusia di ops console, termasuk AI failure rate untuk alert RG-5.

---

## 14. Usage, credits, entitlements

Memenuhi FR-BILL-001..007, BR-CRED-01..06, Lampiran E PRD, D-08, RG-4, dan ADR-10.

### 14.1 Tabel

| Tabel | Kolom kunci |
|---|---|
| `usage.plan` / `usage.plan_entitlement` | plan_code (STARTER, GROWTH, AGENCY, ENTERPRISE); entitlement key → nilai (seats, saved_icps, profile_views_month, credits_month, ai_credits_month, webhook_endpoints, export_rows_max) |
| `usage.subscription` | workspace_id, plan_code, period_start/end, overrides jsonb (Enterprise/ops), status |
| `usage.credit_lot` | id, workspace_id, meter (`credit`, `ai_credit`), kind (MONTHLY_GRANT, TOPUP, ADJUSTMENT), amount, expires_at (akhir siklus / +12 bulan, BR-CRED-06) |
| `usage.credit_ledger` | id (`led_`), txn_id, workspace_id, meter, account (`available`, `held`, `consumed`, `external`), amount (signed), lot_id, entry_type, usage_event_id, ref_type/ref_id, actor, created_at. Append-only: INSERT saja, trigger menolak UPDATE/DELETE |
| `usage.credit_balance` | (workspace_id, meter) PK, available, held, version. Cache saldo yang diperbarui dalam transaksi yang sama dengan ledger |
| `usage.usage_event` | id (`use_`), workspace_id, user_id, operation (UNLOCK, EXPORT_RECORD, WEBHOOK_RECORD, REFRESH, AI_BRIEF, AI_ANGLE), meter, units, idempotency_key, ref, status (HELD, CAPTURED, RELEASED, REFUNDED), occurred_at; UNIQUE(workspace_id, idempotency_key) |
| `usage.unlock` | (workspace_id, company_id) PK, unlocked_at, unlocked_by, method (DIRECT, EXPORT, WEBHOOK), usage_event_id, contact_point_ids uuid[] (snapshot saat unlock), status (ACTIVE, REFUNDED_INVALID) |
| `usage.profile_view` | (workspace_id, period, company_id) PK, first_viewed_at, viewer_id. Kuota fair-use = jumlah profil **berbeda** yang dibuka per siklus (membuka ulang tidak dihitung) |

### 14.2 Semantik ledger double-entry

Setiap transaksi bisnis (`txn_id`) terdiri dari ≥ 2 baris yang berjumlah nol:

| Transaksi | Baris |
|---|---|
| GRANT (bulanan/top-up/ops) | `external −n`, `available +n` (lot baru) |
| DEBIT (unlock langsung) | `available −1`, `consumed +1` |
| HOLD / reserve (export, webhook, refresh, AI; BR-CRED-07) | `available −n`, `held +n` |
| CAPTURE (sukses) | `held −k`, `consumed +k` (k ≤ n, hanya yang benar-benar terkirim/di-unlock) |
| RELEASE (gagal/sisa) | `held −(n−k)`, `available +(n−k)` |
| REFUND (BR-CRED-05b, merge ganda) | `consumed −1`, `available +1` |
| EXPIRE (lot kedaluwarsa) | `available −sisa_lot`, `external +sisa_lot` |

Saldo tersedia = Σ baris (`workspace`, `meter`, `available`). Konsumsi mengalokasikan lot berdasarkan `expires_at` terdekat lebih dulu (BR-CRED-08). Rekonsiliasi harian: (a) `credit_balance` = Σ ledger; (b) Σ `consumed` = Σ `usage_event` CAPTURED − REFUNDED; (c) setiap txn berjumlah 0. Selisih ≠ 0 → alert kritis (RG-4).

### 14.3 Alur

- **Unlock** (BR-CRED-02): company yang sudah di-unlock dilewati (0 credit). Sisanya: lock saldo → DEBIT per company → `unlock` → audit. Unlock berlaku untuk semua contact point company (termasuk cabang dan person), sekarang dan nanti.
- **Export/webhook** (BR-CRED-03): pratinjau menghitung record yang belum di-unlock → HOLD sebesar itu → setelah file/delivery sukses: CAPTURE + unlock untuk record yang benar-benar terkirim → RELEASE sisanya. Gagal atau DLQ → RELEASE (BR-CRED-05a). Record yang sudah di-unlock gratis. Private prospect dari upload sendiri gratis.
- **Refresh** (FR-MON-001): HOLD 1 → CAPTURE saat job sukses. **AI**: HOLD 1 ai_credit → CAPTURE bila lolos validator. Cache hit = 0.
- **Refund INVALID ≤ 30 hari** (BR-CRED-05b): di MVP1 pemicunya hanya hasil re-verifikasi saat refresh atau laporan data user yang dikonfirmasi ops (BR-CRED-05). Perubahan status karena re-run pipeline biasa tidak memicu refund. Handler memeriksa `unlock` ≤ 30 hari untuk company tersebut. Bila semua `contact_point_ids` yang di-snapshot sekarang INVALID → REFUND + `status = REFUNDED_INVALID` (unlock tetap terlihat).

### 14.4 Entitlement & fair-use (FR-BILL-003, BR-CRED-01)

Middleware `usage.RequireEntitlement(key, increment)` (dipasang per operasi lewat metadata OpenAPI `x-msp-entitlement`) membaca plan + override + pemakaian berjalan (seat = anggota + undangan aktif; ICP tersimpan; endpoint webhook; baris export per job). Pelanggaran → `403 ENTITLEMENT_EXCEEDED {limit, current}`. Fair-use profil (BR-CRED-01): pada 100% kuota, search tetap berjalan. Membuka profil yang **belum** dibuka di siklus ini → `429 FAIR_USE_EXCEEDED` (dengan pesan upgrade). Profil yang sudah dibuka, sudah di-unlock, atau di-assign tetap dapat diakses. Ops dapat menaikkan batas (override, audit). `/v1/entitlements` mengembalikan batas + pemakaian + flag efektif untuk gating UI.

### 14.5 Konkurensi (mencegah overspend)

Setiap operasi berbayar berjalan dalam satu transaksi: `SELECT … FROM usage.credit_balance WHERE workspace_id = :ws AND meter = :m FOR UPDATE` → cek `available ≥ n` → INSERT ledger + UPDATE balance → commit. Bulk unlock diproses per chunk 200 company per transaksi (lock singkat). `UNIQUE` pada `usage_event(workspace_id, idempotency_key)` dan `unlock(workspace_id, company_id)` menjadi pengaman terakhir. Pola ini diuji dengan uji konkurensi (N request paralel pada saldo terbatas, tidak boleh negatif).

---
## 15. Security & privacy

Memenuhi NFR-SEC-01..06, NFR-PRIV-01..04, FR-AUTH-001..006, FR-AUD-001..006, FR-OPS-006, BR-PRIV-01..04, dan RG-7. Baseline OWASP ASVS L2.

### 15.1 Autentikasi & sesi (D-20 Open, ADR-19)

| Opsi D-20 | Kelebihan | Kekurangan |
|---|---|---|
| **Ory Kratos self-hosted (rekomendasi)** | Data identitas tetap di DC Indonesia. Verifikasi email, login Google (OIDC), TOTP MFA, recovery. Ringan (Go, Postgres), satu ekosistem dengan backend (D-21) | UI self-service flow dibangun di Next.js. Operasi dan upgrade ditangani tim |
| Keycloak self-hosted | Fitur lengkap (SAML untuk SSO V2), admin UI | Berat (JVM), kustomisasi tema lebih sulit |
| OIDC SaaS terkelola | Paling cepat, operasi minimal | Data identitas user di luar negeri (bertentangan dengan semangat D-14/NFR-PRIV-03), biaya per MAU |

Desain aplikasi agnostik terhadap pilihan ini: port `IdentityProvider` dengan adapter `kratos` (klien Go Kratos, validasi sesi via `/sessions/whoami`) dan `oidc` (Authorization Code + PKCE). Keduanya menghasilkan `Identity{subject, email, email_verified, name, aal|amr}`, lalu **aplikasi menerbitkan sesinya sendiri**:
- `platform.session(id_hash, user_id, created_at, last_seen_at, idle_expires_at, absolute_expires_at, revoked_at, ip, ua)`. Cookie `__Host-msp_sess` berisi token acak 256-bit (yang disimpan hanya hash-nya). Cache Redis TTL 60 s.
- Idle timeout 12 jam, absolut 7 hari. *Logout semua sesi* mencabut semua baris. Mengeluarkan anggota → cache membership dibersihkan via pub/sub dan TTL ≤ 60 s menjamin akses hilang ≤ 1 menit (FR-AUTH-005).
- Tanpa `email_verified` → `403 EMAIL_NOT_VERIFIED` (FR-AUTH-001). Akun ops wajib MFA (`aal2` / `amr` berisi `otp`/`mfa`); bila tidak → `MFA_REQUIRED` (FR-AUTH-006). Konsol ops memakai audience/cookie terpisah.
- API key (V1): `msp_live_<prefix>_<secret>`, disimpan `sha256(secret)` + prefix, memiliki scope, dan hanya ditampilkan sekali (FR-AUTH-007).

### 15.2 Otorisasi (FR-AUTH-004)

`Permission` enum: `PROSPECT_READ, CONTACT_UNLOCK, ORGANIZE, ASSIGN, EXPORT, ICP_MANAGE, AI_GENERATE, INTEGRATION_MANAGE, MEMBER_MANAGE, USAGE_READ, AUDIT_READ, BILLING_MANAGE, WORKSPACE_DELETE, UPLOAD`. Peta role → permission di `internal/platform/rbac` mengikuti PRD §8.1.1 (UPLOAD = Member+, TQ-05; EXPORT untuk Member mengikuti `member_can_export`). Setiap operasi OpenAPI mendeklarasikan `x-msp-permission`. Middleware membaca metadata operasi hasil generate dan menolak sebelum handler dipanggil; operasi tanpa deklarasi membuat CI gagal. Penolakan → 403 + `audit_log(authz.denied)`. Test matriks dibangkitkan dari OpenAPI (`x-msp-permission`) × 5 role.

### 15.3 Isolasi tenant (FR-AUD-006, NFR-SEC-03)

Ada tiga lapis: (1) RLS + role DB (§5.8); (2) repository selalu memfilter `workspace_id`, dan ID di path di-resolve ke workspace pemilik (milik workspace lain → 404); (3) cache key selalu diawali `ws:{id}:`, dan objek MinIO tenant berada di prefix `{workspace_id}/` yang hanya diakses lewat presigned URL dari API. Uji lintas tenant otomatis di CI (RG-7).

### 15.4 Secret & enkripsi (NFR-SEC-02/04, ADR-22)

- **In transit:** TLS 1.2+ di Traefik (sertifikat ACME). LAN: PostgreSQL `sslmode=verify-full` (CA internal), Redis TLS + ACL, MinIO TLS.
- **At rest:** LUKS2 full-disk di semua server (unlock otomatis via Clevis/Tang di ops-01, fallback passphrase dari konsol remote). Repo pgBackRest `repo-cipher-type=aes-256-cbc`. MinIO SSE (kunci dari KMS bridge/static key yang dikelola SOPS).
- **Secret:** SOPS + age di repo infra. Kunci privat age hanya ada di host deploy. Deploy (Ansible) mendekripsi ke `/run/msp/secrets/*.env` (tmpfs, 0600) → `env_file`/Docker secrets. Rotasi dicatat. Secret scanning di CI (NFR-SEC-04).
- **Secret webhook:** AES-256-GCM di aplikasi dengan master key berversi (`secret_kid`). Dekripsi hanya di `worker-webhook` dan endpoint `rotate-secret`.

### 15.5 Audit log (FR-AUD-001/002, I-14)

`platform.audit_log(id, occurred_at, workspace_id NULL, actor_type (USER, API_KEY, OPS, SYSTEM), actor_id, actor_ip, user_agent, action, target_type, target_id, reason, metadata jsonb (sudah diredaksi), correlation_id, prev_hash, hash)`, dipartisi bulanan.
- Aksi wajib: login/logout, perubahan role/anggota/undangan, export, unlock, konfigurasi webhook/API key, suppression, keputusan merge/split, grant credit/plan, akses support, perubahan flag, override gate snapshot, authz.denied.
- Append-only: role app hanya `INSERT`. Trigger menolak `UPDATE`/`DELETE`. Tidak ada endpoint ubah/hapus.
- **Hash chain** (opsional, aktif default): `hash = sha256(prev_hash ‖ canonical_json(entry))` per partisi bulan. Verifikasi harian. Ekspor harian ke MinIO bucket object-lock (retensi = TQ-14). Kolom P1 (`actor_ip`, `user_agent`) disimpan di tabel samping `audit_log_pii` agar pseudonimisasi (FR-AUD-005) tidak memutus hash chain.
- Berbeda dari `action.activity`, yang merupakan timeline produk dan terlihat user.

### 15.6 Klasifikasi data & redaksi log (BR-PRIV-01, NFR-PRIV-01)

| Kelas | Contoh | Penyimpanan | Log/analytics | LLM | Export |
|---|---|---|---|---|---|
| B — Data bisnis publik | Nama usaha, alamat, kategori, telepon/email usaha, rating | Global layer | Boleh (ID/nama usaha) | Ya, kecuali nilai kontak | Sesuai lisensi + unlock |
| P1 — Data pribadi | Nama person, email/telepon person, email/nama user, IP | Global (person) / platform | **Hash/redaksi** | **Tidak** | Hanya setelah unlock; P1 user tidak di-export |
| T — Data tenant | ICP, catatan, upload, list | Tenant layer (RLS) | ID saja | Offering/ICP saja | Milik tenant |
| S — Secret | Token sesi, API key, secret webhook, kunci | Hash/terenkripsi | Tidak pernah | Tidak | Tidak |

`slog.Handler` dengan `ReplaceAttr` menghapus kunci terlarang (`email`, `phone`, `value`, `raw_value`, `authorization`, `cookie`, `secret`, `token`, `person_name`) dan memindai teks pesan dengan regex email/telepon → `h:<8 hex HMAC>` (masih dapat dikorelasikan, tidak dapat dibaca). Hal yang sama berlaku untuk breadcrumb GlitchTip dan atribut span.

### 15.7 Suppression (FR-AUD-003, BR-PRIV-02, NFR-PRIV-02 ≤ 24 jam)

Suppression berlaku untuk data **global layer** di semua workspace dan mengalahkan semua sumber lain (BR-PRIV-02).

```
ops/DSR -> identity.suppression (+ value_hash) + event suppression.created          [satu transaksi]
   -> identity: tandai contact_point/person/establishment suppressed=true (RLS global langsung menyembunyikan)
   -> query: upsert/delete baris prospect_search terkait; hapus cache profil & facet; ai: artifact terkait INVALIDATED
   -> action: delivery PENDING/RETRYING yang memuat subjek -> CANCELLED; export yang sedang berjalan memfilter saat generate
   -> action: file export AKTIF (belum kedaluwarsa) yang memuat subjek (dicari via action.distribution_log)
              -> link lama dicabut, file dibuat ulang tanpa record itu, pemilik diberi notifikasi
   -> action: setiap workspace yang pernah menerima record lewat export/webhook -> event suppression.applied (tanpa nilai kontak)
   -> identity/ingest: nilai cocok value_hash pada snapshot berikutnya dibuang saat normalize (SUPPRESSED_VALUE)
   -> sweep per jam: kebocoran (nilai ter-suppress di projection, file export aktif, delivery pending) = 0 -> set propagated_at
```

Data upload tenant **tidak** di-suppress otomatis karena tenant adalah pengendali data tersebut (MSP = prosesor). Permintaan subjek data atas data upload diteruskan ke tenant (§15.8), sedangkan nilai yang sama di global layer tetap di-suppress. Metrik `suppression_leak_count` > 0 atau `propagated_at − created_at` > 24 jam → alert.

### 15.8 DSR, laporan data, penutupan workspace

- **DSR** (FR-AUD-004): intake form publik (captcha + verifikasi email pemohon) atau email ops → `ops.data_subject_request` (RECEIVED → VERIFYING → IN_PROGRESS → [FORWARDED_TO_TENANT] → COMPLETED | REJECTED). Subjek dicari via `value_hash`. Tipe akses (laporan data yang ada), koreksi (laporan data → ops), hapus/keberatan (suppression + penghapusan nilai P1 dari global). Pengakuan penerimaan ≤ 1 hari kerja (email otomatis). Tenggat per tipe dikonfigurasi berdasarkan review hukum (beberapa hak bertenggat 3×24 jam, FR-AUD-004) dan dipantau di FR-OPS-008. Bila subjek ditemukan di data upload tenant, permintaan diteruskan ke Owner/Admin workspace tersebut (notifikasi + status `FORWARDED_TO_TENANT`) dan dieksekusi atas instruksi tenant (BR-PRIV-02).
- **Laporan data** (FR-DATA-014): `ops.data_issue_report` dengan `claim_ref`. Hasilnya bisa berupa koreksi ops (evidence `OPS_CORRECTION`, FR-OPS-009, §5.6), merge/split, atau tolak. Laporan yang mengonfirmasi kontak INVALID juga memicu cek refund BR-CRED-05.
- **Penutupan workspace** (FR-AUD-005): `PENDING_DELETION` (akses diblokir kecuali Owner dapat membatalkan) → hari ke-30 job `purge_workspace` memanggil handler purge yang didaftarkan setiap modul (semua tabel tenant), menghapus prefix MinIO `{ws}/`, key Redis, dan sesi → verifikasi hitungan = 0 → `audit(workspace.purged)`. **Pengecualian** (FR-AUD-005): `audit_log` serta `credit_ledger`/`usage_event` dipertahankan selama masa retensi hukum, lalu **dipseudonimkan** (ID aktor diganti HMAC ber-pepper, baris `audit_log_pii` dihapus, metadata P1 dibuang). Backup PITR menua dan keluar sesuai retensi pgBackRest (14 hari full + WAL), dan hal ini dinyatakan di kebijakan privasi.

### 15.9 Akses support (FR-OPS-006)

`platform.support_access_grant(workspace_id, ops_user_id, reason NOT NULL, expires_at ≤ 4 jam)`. Selama grant aktif, konsol ops membuka `tenancy.WithTx` **READ ONLY** untuk workspace tersebut. Setiap request dicatat (`support.read`, path, target). Tidak ada impersonation. Owner/Admin melihat entri ini di audit log workspace mereka.

### 15.10 File upload/export

Upload: hanya `.csv`/`.xlsx`, batas ukuran/baris/kolom, penjaga zip bomb, parser tidak mengeksekusi formula/macro, dan file mentah tidak pernah disajikan ulang ke browser. Export: mitigasi **CSV/formula injection** (sel yang diawali `=`, `+`, `-`, `@`, tab, atau CR diberi prefiks `'`), CSV UTF-8 dengan BOM, dan XLSX tanpa formula.

### 15.11 Threat model (STRIDE-lite)

| Ancaman | Contoh | Mitigasi |
|---|---|---|
| Spoofing | Pencurian sesi, login palsu | Cookie `__Host-` HttpOnly/SameSite, CSRF, MFA ops, idle timeout, rate limit login (Kratos) |
| Tampering | Cursor/ID dimanipulasi, payload webhook dipalsukan, audit diubah | Cursor HMAC, RLS + cek pemilik, signature webhook, audit append-only + hash chain |
| Repudiation | "Saya tidak export itu" | Audit dengan aktor/IP/waktu, ledger + usage event |
| Information disclosure | Kebocoran lintas tenant, kontak ter-mask bocor, PII di log/LLM, SSRF ke metadata/LAN | RLS + uji CI, masking di server, redaksi log, redaksi prompt + test, tiga lapis SSRF (§12.5) |
| Denial of service | Query search berat, upload bom, export massal, banjir webhook | `statement_timeout`, rate limit, batas plan, penjaga zip, antrean terpisah per jenis kerja |
| Elevation of privilege | Member mengubah role, akses `/v1/ops` | RBAC server-side + test matriks, audience ops terpisah + MFA, role DB minimal |

---

## 16. Observability

Memenuhi NFR-OBS-01, RG-5, FR-OPS-001..004, dan ADR-21.

### 16.1 Stack

OTel Go SDK (`otelhttp` untuk server/klien HTTP, tracer pgx, middleware River yang meneruskan `traceparent` + `correlation_id` di metadata job, Redis) → **OTel Collector** di setiap host → `ops-01`: **Prometheus** (metrik, retensi 30 hari) + **Loki** (log JSON, 30 hari) + **Tempo** (trace, 7 hari; sampling 10% + 100% error) + **Grafana** (dashboard) + **Alertmanager** (on-call: email + kanal chat tim) + **GlitchTip** (error, kompatibel SDK Sentry). Exporter: node, postgres, redis, MinIO, Traefik, cAdvisor, pgBackRest (umur backup).

### 16.2 Standar field log (Crawler §19.2 diadopsi)

`ts, level, service (web|api|worker-<queue>), env, version, logger, msg, correlation_id, request_id, trace_id, span_id, workspace_id` (bila diizinkan), `user_id` (ID pseudonim), `job_id, stage, snapshot_id, batch_key, entity_id, endpoint_host_hash` (webhook), `url_hash, worker_id, duration_ms, http.method, http.route, http.status, error.code, error.type`. Nilai PII mengikuti §15.6.

### 16.3 Katalog metrik (pilihan)

| Domain | Metrik |
|---|---|
| Pipeline | `msp_stage_duration_seconds{stage}`, `msp_stage_records_total{stage,outcome}`, `msp_stage_failure_ratio{stage}`, `msp_queue_depth{queue}`, `msp_queue_oldest_age_seconds{queue}`, `msp_job_retries_total{queue}`, `msp_dlq_size{queue}` (dihitung dari tabel `river_job`: state `available`, `running`, `retryable`, `discarded`) |
| Data health (per snapshot, ditulis ke `ops.metric_snapshot`) | total company/establishment, verified contact rate, stale rate, duplicate candidate rate, merge review backlog/umur, evidence coverage, % kategori "Lainnya", % establishment ber-area |
| Intelligence | Distribusi skor/band per `model_version`, distribusi coverage, % UNSCORED, opportunity acceptance per tipe, durasi job skor (p95), grounding pass rate |
| Produk | Latensi search/profil p95/p99, time-to-first-result, actioned prospects/minggu (dari `activity`) |
| Biaya | `msp_ai_cost_usd_total{prompt_version,model}`, biaya per brief, token, query DNS, biaya per enriched record |
| Webhook | Attempt, success ratio, latensi, DLQ per endpoint (label = ID endpoint), endpoint disabled |
| Billing/privasi | Selisih rekonsiliasi ledger, `suppression_leak_count`, lag propagasi suppression |
| Infra | Lag replikasi db-02, umur backup/WAL terakhir, disk/IO, memori Redis, sertifikat TLS kedaluwarsa |

### 16.4 SLO & alert (RG-5)

| SLO / alert | Ambang | Jendela |
|---|---|---|
| Ketersediaan API (NFR-AVAIL-01) | 99,5% bulanan; burn-rate alert 2%/jam | 1 j / 6 j |
| Search p95 ≤ 800 ms, profil p95 ≤ 500 ms (NFR-PERF-01/02) | Pelanggaran > 15 menit | 15 m |
| **Pipeline failure > 5%** (RG-5) | `msp_stage_failure_ratio > 0,05` per stage | 15 m |
| **Backlog > 1 jam** (RG-5) | `msp_queue_oldest_age_seconds > 3600` di antrean mana pun | 5 m |
| **AI failure > 5%** (RG-5) | Gagal/total brief > 0,05 (minimal 20 request) | 15 m |
| **Webhook failure > 5%** (RG-5) | Attempt gagal/total > 0,05 (minimal 50 attempt, agregat semua endpoint) | 1 j |
| Skor ICP ≤ 10 menit (NFR-PERF-05) | Job > 10 menit | per job |
| Suppression ≤ 24 jam | Leak > 0 atau lag > 24 jam | 1 j |
| Ledger | Selisih rekonsiliasi ≠ 0 | harian |
| Infra | Lag replika > 60 s; backup terakhir > 26 jam; WAL archive gagal; disk > 80% | 5 m |

Drill alert di staging adalah bukti RG-5 (`05-test-strategy.md`).

### 16.5 Dashboard

| Dashboard | Lokasi | FR |
|---|---|---|
| Data Health (per snapshot) | Ops console (dari `ops.metric_snapshot`) + Grafana | FR-OPS-001 |
| Pipeline Health + DLQ redrive | Ops console + Grafana | FR-OPS-002 |
| Intelligence Health | Ops console | FR-OPS-003 |
| Cost | Ops console + Grafana | FR-OPS-004 |
| API & SLO, Webhook, Billing/privasi, Infra | Grafana | NFR-OBS-01 |

---

## 17. Performa & kapasitas

Memenuhi NFR-SCAL-01 (5 juta establishment, ±84 ribu kelurahan/desa, 200 workspace, 2.000 user, 50 konkuren), NFR-PERF-*, dan NFR-AVAIL-02. Angka akan direvisi setelah Q-02.

### 17.1 Estimasi volume & storage (tahun pertama)

| Objek | Baris | Ukuran (data + index) |
|---|---|---|
| establishment / company / person | 5 jt / ±4,2 jt / ±0,5 jt | ±9 GB |
| contact_point | ±8 jt | ±4 GB |
| evidence (2 snapshot online + derived) | ±25 jt | ±8 GB |
| observation (2 snapshot online, record-level JSONB) | 10 jt | ±18 GB |
| gap_fact, establishment_factor, class, digital, area | ±20 jt | ±8 GB |
| geo boundary (detail + subdivide) + area_metric | ±90 rb + ±7 jt | ±6 GB |
| prospect_search | 5 jt | ±7 GB |
| intel.score (200 ws × ±2,5 ICP × ±150 rb) | ±75 jt | ±17 GB |
| tenant lain (activity, audit, delivery, ledger, upload) | ±30 jt | ±12 GB |
| **Total DB** | | **±90 GB**; ×2 headroom (shadow table, bloat, index rebuild) = ±200 GB |
| MinIO: snapshot mentah (Parquet), arsip observation, upload, export, tile, repo pgBackRest | | ±1,5–3 TB |

### 17.2 Spesifikasi server (usulan D-19)

| Host | CPU | RAM | Disk | Catatan |
|---|---|---|---|---|
| `app-01` (kelak `app-02` identik) | 32 core (64 thread) | 64 GB | 2 × 1,92 TB NVMe RAID1 | Worker pipeline memakai semua core saat run. Proses Go hemat memori (±1–4 GB per pool). Redis `maxmemory` 8 GB |
| `db-01` | 32 core | 256 GB ECC | 4 × 1,92 TB NVMe RAID10 (±3,8 TB) | `shared_buffers` 64 GB, `effective_cache_size` 192 GB |
| `db-02` | Sama dengan db-01 | 256 GB | Sama | Target failover + read search |
| `stor-01` | 16 core | 64 GB | 4 × 8 TB SSD/HDD enterprise, erasure coding (EC:2) → ±16 TB usable + 1 NVMe OS | Bucket backup memakai object lock |
| `ops-01` | 16 core | 64 GB | 2 × 3,84 TB NVMe RAID1 | GitLab + registry + stack observability |
| `ci-01` | 16 core | 64 GB | 2 × 1,92 TB NVMe | Runner Docker |

Jaringan: VLAN privat 10 GbE antar server, uplink publik hanya ke app-01 (dan ops-01 untuk akses GitLab via VPN/WireGuard). Backup off-site (repo2 pgBackRest + replikasi bucket MinIO) ke lokasi kedua di Indonesia (TQ-15).

### 17.3 Tata letak proses (app-01)

Semua container `msp` memakai image yang sama dengan subcommand berbeda.

| Container | Perintah | Konkurensi | Catatan |
|---|---|---|---|
| traefik | — | 1 | TLS, routing `/api` → api |
| web (Next.js) | — | 2 replika | UI saja |
| api | `msp api` | 2 replika (`GOMAXPROCS` 4) | Pool pgx 20/replika via PgBouncer. Blue/green via Traefik |
| worker-pipeline | `msp worker --queues=pipeline` | `MaxWorkers` 32 | Aktif penuh saat pipeline run |
| worker-scoring | `msp worker --queues=scoring` | `MaxWorkers` 8 | Prioritas 1 (ICP interaktif) selalu didahulukan atas prioritas 4 (fan-out) |
| worker-export | `msp worker --queues=export` | `MaxWorkers` 4 | Streaming ke MinIO multipart |
| worker-webhook | `msp worker --queues=webhook` | `MaxWorkers` 32 | Network terisolasi (§12.5) |
| worker-ai | `msp worker --queues=ai` | `MaxWorkers` 16 | I/O bound |
| worker-maintenance | `msp worker --queues=maintenance` | `MaxWorkers` 8 | Upload, email, refund, purge, periodic job (River leader election menggantikan scheduler) |
| kratos, redis (opsional), unbound, smokescreen, otel-collector | — | 1 | — |

### 17.4 Baseline PostgreSQL

`max_connections` 300 (PgBouncer `default_pool_size` 40), `work_mem` 32–64 MB, `maintenance_work_mem` 4 GB, `max_parallel_workers_per_gather` 4, `max_parallel_maintenance_workers` 8, `random_page_cost` 1.1 (NVMe), `wal_compression` on, `archive_mode` on (pgBackRest async archive-push), autovacuum agresif untuk `score`, `river_job`, `event`, `webhook_delivery`, `stage_run`. River memakai **koneksi langsung** ke db-01 (bukan lewat PgBouncer transaction mode) untuk `LISTEN/NOTIFY` dan leader election, sedangkan query aplikasi lewat PgBouncer. Data bulk (score, projection) memakai `COPY`.

### 17.5 Bottleneck & mitigasi

| Bottleneck | Mitigasi |
|---|---|
| ER pasangan kandidat | Blocking ketat, perbandingan paralel (goroutine pool), batch per sel geohash, skip record tak berubah (`record_hash`) |
| PIP 5 juta titik × 84 ribu polygon | `ST_Subdivide` + GiST; penetapan per kab/kota paralel |
| Rebuild projection | Shadow table + index paralel + swap. Upsert inkremental untuk perubahan kecil |
| Penulisan skor | `COPY` ke epoch baru, tanpa UPDATE; hapus epoch lama secara async |
| Join search | Kandidat ICP dibatasi, index `score_rank` parsial, replica db-02, `CREATE STATISTICS` |
| DNS lookup | Resolver Unbound lokal, cache, konkurensi terkendali |
| SPOF app-01 | MVP: restore via Ansible + image registry (RTO ≤ 4 jam). V1: app-02 aktif-aktif |
| Fan-out re-skor saat publish | Job `scoring` prioritas 4 di-stagger (maksimal 4 berjalan bersamaan); prioritas 1 (ICP interaktif) selalu didahulukan |

### 17.6 Backup & DR (NFR-AVAIL-02: RPO ≤ 15 menit, RTO ≤ 4 jam)

pgBackRest: full mingguan, diff harian, archive WAL kontinu (`archive_timeout = 60s`) → RPO ≤ 1–2 menit (repo1 MinIO, repo2 off-site). Restore ±200 GB pada NVMe ±15–30 menit + replay WAL. Skenario utama adalah failover ke db-02 (promote) dengan RTO < 30 menit. MinIO: versioning + object lock untuk bucket snapshot/backup/audit, plus replikasi ke off-site. Uji restore bulanan otomatis ke host staging (bukti NFR-AVAIL-02). Redis hanya cache/rate limit dan tidak di-backup. Antrean River ikut backup/PITR Postgres.

---

## 18. Konfigurasi, environment & feature flags

Memenuhi FR-OPS-007, NFR-SEC-04, dan D-19.

### 18.1 Konfigurasi (12-factor)

Semua konfigurasi berasal dari environment variable (di-parse ke struct config Go dan divalidasi saat start; nilai wajib yang hilang → gagal start). Konfigurasi domain berversi (model skor, katalog, taksonomi, daftar referensi) adalah file di repo, bukan env. Variabel utama:

| Kelompok | Variabel |
|---|---|
| Inti | `MSP_ENV` (local, ci, staging, production), `MSP_VERSION`, `MSP_BASE_URL`, `MSP_LOG_LEVEL` |
| Database | `DATABASE_URL_APP`, `DATABASE_URL_PIPELINE`, `DATABASE_URL_OPS`, `DATABASE_URL_REPLICA`, `DATABASE_URL_DIRECT` (River listener, migrate) |
| Worker | `MSP_WORKER_QUEUES` (mis. `pipeline,scoring`), `MSP_WORKER_MAX_<QUEUE>` |
| Redis (opsional) | `REDIS_URL` (cache, rate limit, counter fair-use). Kosong → cache in-process (dev) |
| Object storage | `S3_ENDPOINT_URL`, `S3_PUBLIC_ENDPOINT` (`files.<domain>`), `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY`, `S3_BUCKET_{SNAPSHOTS,UPLOADS,EXPORTS,TILES}` |
| Auth (D-20) | `IDP_KIND` (kratos, oidc), `KRATOS_PUBLIC_URL`, `KRATOS_ADMIN_URL`, `OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `SESSION_PEPPER` |
| Kriptografi | `CURSOR_HMAC_KEY`, `VALUE_HASH_PEPPER`, `WEBHOOK_SECRET_MASTER_KEYS` (berversi) |
| AI | `LLM_PROVIDER`, `LLM_API_KEY`, `LLM_MODEL_BRIEF`, `LLM_MODEL_CLASSIFY`, `AI_DAILY_BUDGET_USD`, `AI_PRICE_TABLE` |
| Jaringan | `EGRESS_PROXY_URL` (Smokescreen), `DNS_RESOLVER` (Unbound), `SMTP_URL` |
| Observability | `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_TRACES_SAMPLER_ARG`, `GLITCHTIP_DSN` |

### 18.2 Environment

| Env | Infrastruktur | Data | Integrasi eksternal |
|---|---|---|---|
| local | docker compose (`infra/docker/`): postgres+postgis, redis, minio, kratos, mailpit, smokescreen | Fixture + dataset sintetis kecil | LLM `fake`, webhook ke receiver lokal |
| ci | Container ephemeral (service GitLab CI) | Fixture, golden set ER, eval AI rekaman | `fake` |
| staging | Host staging di DC yang sama (VM/server terpisah, spesifikasi diperkecil) | Dataset sintetis 5 juta baris (uji beban) + subset dataset tanpa person/kontak pribadi | LLM asli dengan cap kecil, receiver webhook uji |
| production | Topologi §2.2 | Produksi | Asli |

Data produksi yang mengandung P1 tidak pernah disalin ke local/ci/staging.

### 18.3 Feature flag & kill switch (FR-OPS-007)

`platform.feature_flag` (scope GLOBAL → PLAN → WORKSPACE; yang paling spesifik menang) di-cache in-process 30 s dan diinvalidasi via pub/sub. Setiap perubahan tercatat di audit.

| Flag | Default | Efek bila off |
|---|---|---|
| `ai.generation` | on | Tombol AI nonaktif + `503 FEATURE_DISABLED` (FR-AI-006) |
| `export.create` | on | Export baru ditolak; job berjalan tetap selesai |
| `webhook.delivery` | on | Delivery ditahan (§12.4) |
| `upload.create` | on | Upload baru ditolak |
| `pipeline.publish` | on | Snapshot tidak dapat di-publish |
| `dns.lookup` | on | Verifikasi email → UNKNOWN (bukan INVALID) |
| `search.include_private` | on | Toggle private prospect disembunyikan |
| `scoring.bulk_rescore` | on | Fan-out re-skor ditunda (interaktif tetap jalan) |

---

## 19. Jalur reaktivasi crawler

Memenuhi D-01, PRD §6 (V2), dan analisis §2.1/§6.

### 19.1 Port `SourceAdapter`

```go
type SourceAdapter interface {
    SourceCode() string
    ContractVersion() string                              // "observation-record/1.0.0"
    Describe() SourceDescriptor                           // tipe, field yang dihasilkan, tag lisensi default
    Produce(ctx context.Context, run AdapterRun) iter.Seq2[ObservationRecord, error]
}
// Hasil ditulis sebagai part JSONL/Parquet + manifest ke msp-snapshots/{source}/{version}/,
// lalu masuk lifecycle snapshot yang sama (§4.2)
```

Crawler kelak memproduksi **snapshot bergulir** (mis. per batch mingguan `ds-2027.03.2`) atau snapshot per job untuk crawl on-demand. Engine hilir (§6) tidak berubah.

### 19.2 Pemetaan Crawler Spec → Input Contract

| Crawler Spec | Komponen MSP yang menerima | Status MVP |
|---|---|---|
| §5 CrawlJob lifecycle | `AdapterRun` + `dataset_snapshot` (STAGED…) | Kontrak siap |
| §11 Evidence & provenance | `EvidencePointer` `CRAWL_PAGE` + `identity.evidence` (`snippet_hash`, `extractor_version`, `pointer.source_url/final_url/selector_hint`) | **Sudah di skema** |
| §16 RawPage (immutable, content_hash) | Bucket `msp-raw-pages` (object lock) + `record_hash` | Bucket direncanakan |
| §16 PageObservation (field, value, selector_hint, confidence) | `Observation` (`field`, `raw_value`, `extraction_confidence`, `evidence`) | **Sudah di kontrak** |
| §12 SSRF | `internal/platform/netguard` + Smokescreen + nftables (§12.5), dengan role ACL `crawler` terpisah | **Dibangun di MVP** (untuk webhook) |
| §8.3/§13 Retry, backoff, DLQ | Retry River (`NextRetry`), job `discarded` + retry dari ops console | **Dibangun di MVP** |
| §14.2 Change detection | `record_hash` + diff snapshot (FR-DATA-013) | V1 |
| §15 Kontrol multi-tenant (kuota) | `usage` entitlement (`crawl_pages_month`) | Desain siap |
| §19 Observability | Standar log/metrik §16 | **Dibangun di MVP** |
| §6 Policy, §7 frontier, §8 fetch, robots | Modul baru `internal/acquisition` (schema `crawl`), di bawah `ingest` | V2 |
| §17 API `/v1/crawl/*` | `/v1/ops/crawl/*` (internal dahulu) | V2 |

Opportunity yang otomatis aktif bila crawler mengisi fieldnya: OPP-BOOKING-NONE (`has_online_booking`), perluasan OPP-EMAIL-* (email dari halaman), dan person/decision-maker (R-01).

### 19.3 Prasyarat untuk melanjutkan

1. Review hukum: UU 27/2022 (PDP), ketentuan situs/robots (RFC 9309), basis pemrosesan, retensi raw page.
2. Entri Source Registry `CRAWLER` dengan tag lisensi dan policy (§4.1). Model credit crawler diselaraskan dengan D-08 (tanpa mata uang ketiga, I-04).
3. Egress khusus crawler: role ACL Smokescreen + IP keluar terpisah dari webhook, rate limit per domain.
4. Bukti nilai: analisis fill-rate dataset (Q-01) menunjukkan gap (email, person, booking) yang membenarkan biaya crawl.
5. Kontrak `observation-record` stabil (≥ 1 rilis tanpa breaking change). Uji beban sesuai Crawler §29.3.

---

## 20. Pertanyaan teknis terbuka

### 20.1 Ketergantungan pada Q-01..Q-07 (00 §8)

| Q | Bagian SDD yang terdampak | Default sementara |
|---|---|---|
| Q-01 Skema/fill-rate dataset | Mapping spec (§4.4), tipe opportunity aktif (§8.1), field set DQ, trust per field | Katalog field §4.3.1; tipe tanpa field = inactive |
| Q-02 Volume & cakupan | Sizing §17, partisi, throughput §6.9 | 5 juta establishment |
| Q-03 Kadens update | Retensi snapshot online (2), jadwal re-skor, signal V1 | Kuartalan, publish manual |
| Q-04 Sumber populasi & sistem kode | Crosswalk §4.7, `population_version`, coverage | Kemendagri + crosswalk BPS |
| Q-05 Lisensi dataset | `source.policy` (display/export/retensi), atribusi, ketentuan data turunan Google Places | Konservatif: export hanya field kanonik/derived |
| Q-06 Ground truth revenue | Prior `revenue_priors`, confidence ≤ 70 | Prior manual per kategori × tier kota |
| Q-07 Customer beta | Preset bobot (§7.1), event webhook prioritas, CRM V1 | Preset usulan |
| Q-09 Spesifikasi server & lokasi off-site | §17.2, §17.6 (DR) | Spesifikasi usulan §17.2 |
| Q-10 Poligon batas wilayah + lisensi | §6.7 (PIP), §9.5 (peta) | Fallback alamat + titik/cluster + tabel |

### 20.2 Pertanyaan teknis baru

| ID | Pertanyaan | Usulan default |
|---|---|---|
| TQ-01 | **D-20**: Kratos vs Keycloak vs OIDC SaaS | Ory Kratos self-hosted (§15.1) |
| TQ-02 | Region/DPA provider LLM (data company keluar Indonesia, D-14/D-15) | Lanjut dengan data level company + redaksi; konfirmasi hukum |
| TQ-03 | Email dengan A record tetapi tanpa MX: PRD → INVALID, padahal RFC 5321 mengizinkan *implicit MX* | Ikuti PRD (INVALID) di MVP; usul ubah ke RISKY |
| TQ-04 | Nilai Contactability untuk email person LIKELY_VALID (PRD hanya mendefinisikan VERIFIED = 100, padahal VERIFIED baru ada di V1) | 90 |
| TQ-05 | Record upload NEW (private prospect) ikut search utama? (Izin upload sudah ditetapkan PRD §8.1.1: Member+) | Hanya via toggle opt-in (§10.9) |
| TQ-06 | Dua company yang sama-sama sudah di-unlock lalu di-merge: refund? | Refund 1 credit otomatis |
| TQ-07 | ~~ICP terikat Offering~~ | **Ditutup** oleh FR-ICP-002 (tepat satu Offering per ICP) |
| TQ-08 | Definisi "tier kota" untuk Commercial Potential dan prior revenue belum ada di PRD | `data/reference/city_tier.yaml` (3 tier) disusun product |
| TQ-09 | `competition_index`/`market_opportunity_score` untuk ICP multi-kategori atau tanpa kategori | Σ kategori ICP; tanpa kategori → business_density |
| TQ-10 | ~~Suppression vs data upload~~ | **Ditutup** oleh BR-PRIV-02/FR-AUD-004 (diteruskan ke tenant) |
| TQ-11 | Basemap peta (lisensi, residensi) | Tanpa basemap di MVP1 atau tile OSM self-hosted |
| TQ-12 | Provider email transaksional (verifikasi Kratos, undangan) dan residensinya | SMTP relay; vendor ditentukan |
| TQ-13 | Definisi kuota fair-use: jumlah profil **berbeda** per siklus (usulan SDD) atau setiap pembukaan? Perilaku 100% sudah ditetapkan BR-CRED-01 | Profil berbeda |
| TQ-14 | Retensi: audit log, snapshot mentah, evidence snapshot lama, log | Audit 2 tahun, snapshot 2 tahun, log 30 hari (review hukum) |
| TQ-15 | Status lisensi/distribusi MinIO community (AGPL) dan lokasi off-site (Q-09) | DC kedua di Indonesia; evaluasi alternatif S3-compatible (Garage/SeaweedFS/Ceph RGW) bila perlu |
| TQ-16 | ~~Re-skor karena drift freshness~~ | **Ditutup** oleh BR-SCORE-13 (cek harian kelas freshness, §7.9) |

### 20.3 Catatan untuk PRD/analisis

- Kontrak skor PRD tidak menjelaskan bahwa Data Quality bergantung pada Offering ("completeness field relevan Offering"). SDD memakai field set per tag Offering (§7.3).
- Signal (bobot 5) yang selalu Unknown membuat `coverage` MVP maksimal 0,95. Ini konsisten dengan BR-SCORE-03, tetapi perlu dijelaskan di UX.
- Brief SDD menyebut `percent_rank()` untuk persentil, sedangkan PRD BR-SCORE-09 menetapkan mid-rank. SDD mengikuti PRD (§9.2).
- Lampiran E memakai "ICP tersimpan" sebagai entitlement, sedangkan ICP aktif dipilih per user (FR-ICP-006). Entitlement `saved_icps` di §14.
