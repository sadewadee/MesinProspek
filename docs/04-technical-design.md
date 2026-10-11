# 04 — Technical Design (lean)

## MesinProspek — Market Intelligence untuk Bisnis Lokal Indonesia

| Field | Value |
|---|---|
| Versi | 0.3 (lean). Sumber data = DB gmaps (D-31), estimasi (D-32), analisis lokasi (D-33) |
| Tanggal | 2026-10-11 |
| Sumber kebenaran | Requirement: [PRD](01-prd.md) v0.3 · Keputusan: [00 §7](00-analisis-dokumen-awal.md#7-register-keputusan-status-v02) · Nama teknis: [README](README.md#kosakata-teknis-bersama) |
| Cakupan | **F1 dirinci.** F2/F3 hanya daftar evolusi (§16) |

> Dokumen ini **tidak menyalin** rumus PRD. Rumus estimasi (Lampiran E), analisis lokasi (F), skor (A), katalog opportunity (B), dan metrik wilayah (C) tetap di PRD. Di sini hanya dijelaskan cara menghitung dan menyimpannya.

## 1. Ringkasan & keputusan arsitektur

- Satu binary Go `msp` + satu PostgreSQL. Data, antrean, search, dan geo semuanya di Postgres. Tidak ada Redis, PgBouncer, object storage, atau search engine terpisah (D-26). Semuanya berjalan di **satu dedicated server** (D-19, §2). Data bisnis **global** (`places`, `place_snapshots`, `estimates`, `areas`) dibaca semua workspace dan ditandai per negara (`country_code`); F1 hanya mengaktifkan `ID` (D-28). Data **workspace** diisolasi dengan RLS (D-03).
- Alur data: `msp tool geo-sync --country ID` (sekali) → import inkremental dari **DB gmaps** (job harian `import.run` atau `msp tool import --country ID`) → `geo.assign` → rantai hitung ulang **estimasi → metrik wilayah → opportunity → skor** (§5). Logika inti (normalisasi, penggabungan snapshot, estimasi, pola keramaian, analisis lokasi, gap, skor) berupa **fungsi Go murni** yang mudah di-unit-test, termasuk golden test PRD §12. SQL dipakai untuk agregat, join, dan geo.
- Nilai yang sering berubah (model AI, plan & kuota, parameter estimasi) disimpan di DB dan diatur dari halaman ops, bukan di kode atau env (D-29). Secret tetap di `.env`.

| ADR | Keputusan | Rujukan |
|---|---|---|
| ADR-01 | Modular monolith: satu binary `msp` (api, worker, migrate, tool), modul sedikit dan datar | D-11 |
| ADR-02 | PostgreSQL 16 + PostGIS + pg_trgm + unaccent untuk data, antrean River, FTS, point-in-polygon, dan radius (`ST_DWithin`) | D-12, D-26 |
| ADR-03 | Data global + data workspace dalam satu DB; tabel workspace wajib RLS `app.workspace_id` | D-03, FR-PRIV-01 |
| ADR-04 | Claim type diturunkan **saat dibaca** dari asal field. Tiap field menyimpan waktu snapshot asalnya; baris sumber terbaru disimpan sebagai evidence | BR-TRUST-01, BR-NORM-05 |
| ADR-05 | Skor disimpan per (ICP, place), dihitung di Go, ditulis per batch (hasil pertama muncul cepat) | FR-INTL-04, FR-ICP-01 |
| ADR-06 | OpenAPI-first: `contracts/openapi.yaml` → server Go + client TS di-generate | README |
| ADR-07 | Identitas di Kratos; DB kita hanya menyimpan user, membership, dan role | D-20 |
| ADR-08 | Credit = ledger append-only, saldo = `SUM`, satu fungsi debit yang diserialkan dengan lock baris workspace | BR-CRED-01 |
| ADR-09 | Webhook keluar langsung dari worker; guard SSRF di dial time, tanpa egress proxy | FR-ACT-04, NFR-05 |
| ADR-10 | File export di disk lokal lewat interface `storage`; unduh lewat URL bertanda tangan berumur pendek | D-26 |
| ADR-11 | Satu server untuk production, staging, dan ops; isolasi lewat project Compose terpisah + batas CPU/RAM per container | D-19, D-26 |
| ADR-12 | Konfigurasi runtime milik ops (setting AI, parameter estimasi `est.*`, plan) di tabel `settings` dan `plans`, dibaca dengan cache singkat; env hanya untuk secret dan pengaturan proses | D-29 |
| ADR-13 | Sumber data = **DB gmaps dibaca langsung**, read-only, inkremental per keyset (`created_at`, `id`) dengan cursor di `imports`. Nama kolom sumber hanya ada di satu file `source.sql`. Snapshot digabung per field + riwayat harian | D-31, BR-NORM-05 |
| ADR-14 | Rumus estimasi dan analisis lokasi = fungsi Go murni berversi (`est-`, `loc-`) di satu paket `estimate`. Parameter dari `settings`, disimpan per hasil. Analisis lokasi dihitung sinkron di api lalu disimpan per workspace | D-32, D-33, FR-EST-06, FR-LOC-01 |

## 2. Arsitektur F1

```
                                 Browser (HTTPS)      uptime monitor eksternal (/readyz)
                                        │                     │
┌─ dedicated server (Indonesia) ────────▼─────────────────────▼───────────────────────┐
│ caddy  satu-satunya port publik 80/443 (+ SSH key-only), TLS otomatis, semua domain │
│   app.<domain> → production · staging.<domain> → staging                            │
│     ├─ /            → web     Next.js (SSR memanggil api di jaringan internal)      │
│     ├─ /api/v1/*    → api     msp api ── /sessions/whoami ──► kratos                │
│     └─ /.ory/*      → kratos  public API saja (admin API tidak di-route)            │
│   gitlab.<domain>, registry.<domain> → gitlab · grafana.<domain> → grafana          │
│                                                                                     │
│ /opt/msp  (production)                       /opt/msp-staging  (staging)            │
│   web · api · kratos                           stack sama; project Compose,         │
│   worker   msp worker (River)                  network, DB, volume, .env terpisah   │
│   postgres 16 + PostGIS + pg_trgm + unaccent                                        │
│     ├─ db msp     global · workspace (RLS) · River                                  │
│     └─ db kratos  identitas                                                         │
│   volume storage  (file export; api + worker)                                       │
│                                                                                     │
│ /opt/ops  gitlab · gitlab-runner (concurrent = 1) · prometheus · alertmanager ·     │
│           grafana · node-exporter      prometheus ── api:9090 (network Docker)      │
│                                                                                     │
│ Setiap container punya batas CPU/RAM (§2.1)                                         │
└──────┬──────────────────────────────────────────────────────────────────────────────┘
       │ pgBackRest (WAL kontinu) · backup GitLab
       ▼
 S3-compatible off-site (Indonesia)
Eksternal: worker → DB gmaps (read-only, §5.1; jalur jaringan: Q-12) · worker → GeoAPI (geo-sync)
           worker → LLM sesuai setting AI (§11) · worker → URL webhook customer (§10)
           kratos → Google OIDC, SMTP · api → SMTP (undangan)
```

| Proses | Perintah | Catatan |
|---|---|---|
| `api` | `msp api` | HTTP `:8080`, metrik `:9090` (listener internal). Analisis lokasi dihitung di sini (§7.3) |
| `worker` | `msp worker` | Semua queue River. Concurrency: `import` 1, `score` 2, `export` 2, `webhook` 10, `ai` 4, `billing` 1 |
| migrasi | `msp migrate` | goose + migrasi River, dijalankan pipeline deploy sebelum `api`/`worker` diganti (06) |
| tool | `msp tool import\|geo-sync\|rescore` | `docker compose run --rm worker msp tool …` di `/opt/msp` (production) |

Satu `caddy` memegang port 80/443 dan me-route semua domain (README). Production dan staging adalah project Compose terpisah dengan network default, database, volume, dan `.env` sendiri, jadi staging tidak bisa menjangkau database production. Satu-satunya yang dipakai bersama adalah **DB gmaps eksternal**: keduanya membacanya read-only (§15). Detail Compose, Caddyfile, dan bootstrap ada di 06.

### 2.1 Kapasitas server (perkiraan)

Saran awal untuk **dataset Indonesia saja**: **16 vCPU, 64 GB RAM, 2× NVMe ±2 TB (RAID1)**. Angka ini perkiraan dan diverifikasi setelah import pertama (Q-09). RAID1 melindungi dari disk rusak, bukan pengganti backup off-site.

| Kelompok container | Batas CPU | Batas RAM | Catatan |
|---|---|---|---|
| Production `postgres` | 4 | 16 GB | Sisa RAM host dipakai page cache Postgres |
| Production `api`, `worker`, `web`, `kratos` + `caddy` bersama | 3 | 6 GB | `worker` mendapat porsi terbesar (import, estimasi, skor) |
| Staging (semua service) | 2 | 6 GB | Data nyata untuk gerbang rilis (PRD §12) |
| `gitlab` (+ registry) | 4 | 8 GB | Ukuran GitLab CE untuk tim kecil |
| `gitlab-runner` + job CI | 2 | 4 GB | `concurrent = 1` di `config.toml`: satu job sekaligus |
| Monitoring (`prometheus`, `alertmanager`, `grafana`, `node-exporter`) | 1 | 2 GB | — |
| **Total batas** | 16 | 42 GB | ±22 GB sisa untuk OS dan page cache |

- **Batas per container wajib** di setiap file Compose (`deploy.resources.limits.cpus` / `.memory`, pembagian per service di 06), sehingga build CI atau import besar tidak membuat production kehabisan CPU/RAM.
- **Disk (perkiraan):** DB production dan DB staging masing-masing puluhan sampai ratusan GB: `places` (+ baris sumber terbaru sebagai evidence), `place_snapshots` (satu baris per tempat per hari, terus bertambah), `estimates`, index, polygon desa, dan skor. Ukuran riwayat bergantung jumlah snapshot di DB gmaps dan diukur setelah import pertama. GitLab + registry ±200 GB dengan cleanup policy. Sisanya untuk WAL lokal dan pertumbuhan.
- **Bila contention terukur** (p95 naik saat pipeline CI atau import berjalan), GitLab/CI dipindah ke server lain (§16, PRD §13).

## 3. Modul Go

Tata letak: `apps/api/cmd/msp` (subcommand), `internal/<modul>`, `internal/api` (hasil generate oapi-codegen, jangan diedit), `db/migrations` (goose SQL), `db/queries` (sqlc → `internal/platform/db/q`). Target Makefile yang sama dipakai lokal dan CI: `make generate` (oapi-codegen, client TS, sqlc, design tokens), `make lint`, `make test`.

| Modul | Isi (satu baris) |
|---|---|
| `platform` | `config` (env), `db` (pgxpool, `InWorkspace`, sqlc), `httpx` (router, problem+json, middleware), `auth` (Kratos whoami → `Principal`), `storage`, `settings` (tabel `flags` + `settings`, cache in-process 10 s), `audit`, `crypto` (AES-GCM, HMAC), `jobs` (River client + semua struct args job) |
| `estimate` | Rumus PRD Lampiran E (`est-X.Y.Z`) dan F (`loc-X.Y.Z`) sebagai fungsi murni: laju ulasan, kunjungan & omzet, hotel, pola keramaian, analisis lokasi, pembulatan, validasi parameter `est.*`. Plus job `est.compute` (§5.4) |
| `ingest` | Baca DB gmaps lewat `ingest/gmaps/source.sql`, cek kolom, validasi, filter negara, normalisasi (BR-NORM), gabung snapshot, upsert `places` + `place_snapshots`, laporan; job `import.run` (§5.1) |
| `geo` | Client GeoAPI + `geo-sync`, penetapan wilayah (PostGIS), metrik wilayah (Lampiran C) |
| `intel` | Tabel claim, deteksi gap (Lampiran B), skor (Lampiran A), search, profil, `ProspectView` (satu proyeksi claim + masking + estimasi untuk search/profil/export/webhook/fakta Narasi AI) |
| `workspace` | Workspace, anggota & undangan (+ batas seat plan), offering, ICP, list, tag, pemilik, catatan, saved search, feedback |
| `action` | Unlock, export, webhook (+ guard SSRF), analisis lokasi (query tetangga + `estimate.AnalyzeLocation` + debit), Narasi AI (interface `LLM`) |
| `billing` | Ledger credit, saldo, fungsi debit tunggal, grant/refund, plan, grant bulanan (`billing.monthly_grant`) |
| `ops` | Endpoint ops: workspace (plan, grant), plan, setting AI & parameter estimasi, status import, kesehatan data, flag, suppression, audit |

**Aturan dependensi.** `platform` tidak mengimpor modul domain. Urutan impor yang diizinkan: `estimate` → `ingest` → `geo` → `intel` → `workspace` → `billing` → `action` → `ops` (modul kanan boleh mengimpor modul kiri, tidak sebaliknya). `estimate` tidak mengimpor modul domain mana pun, dan file rumusnya tanpa I/O. Semua query MSP lewat paket sqlc bersama; satu-satunya pengecualian adalah query DB gmaps (§5.1). Struct args job ada di `platform/jobs`, jadi enqueue tidak perlu mengimpor modul pelaksana. Siklus impor ditolak compiler Go.

## 4. Data model

**Konvensi.** Satu schema `public`. ID entity kita **UUIDv7** (`uuid`, dibuat di Go `uuid.NewV7()`, terurut waktu). Pengecualian: `places.cid` (Google CID, kunci alami `UNIQUE`, D-05), `areas.id` = ID integer GeoAPI (D-13), `categories.id` = slug teks. Waktu `timestamptz` (UTC); jam buka dan pola keramaian disimpan dalam **waktu lokal tempat** (`places.timezone`, BR-NORM-06). Uang = `bigint` Rupiah. Tabel workspace punya `workspace_id uuid NOT NULL` + RLS.

### 4.1 Tabel inti `places` dan `estimates`

```sql
CREATE TABLE places (                    -- satu baris per tempat: nilai terkini hasil gabung snapshot (BR-NORM-05)
  id              uuid PRIMARY KEY,
  cid             text NOT NULL UNIQUE,                  -- Google CID, kunci alami (D-05)
  google_place_id text,                                  -- "ChIJ…", disimpan bila ada
  country_code char(2) NOT NULL, timezone text,          -- ISO 3166-1 alpha-2 (D-28); IANA, mis. Asia/Makassar
  name text NOT NULL,
  category_raw text, categories_raw text[],              -- kategori utama + daftar kategori sumber
  category_id text NOT NULL REFERENCES categories, est_group text NOT NULL, accom_class text,  -- BR-NORM-03, E.3
  phone_raw text, phone_e164 text, phone_kind text,      -- 'mobile' | 'fixed'
  url_raw text, url_kind text,  -- own_domain|social|aggregator|messaging|free_subdomain; NULL = kosong
  social_links jsonb, menu_url text, address_text text, maps_url text,  -- social_links: {"instagram": "https://…"}
  geom geometry(Point, 4326), opening_hours jsonb,       -- jadwal Senin–Minggu, waktu lokal (BR-NORM-06)
  price_raw text, price_min bigint, price_max bigint, price_mid bigint,  -- BR-NORM-04; max NULL = rentang terbuka
  popular_times smallint[] CHECK (cardinality(popular_times) = 168),     -- 7×24, indeks = hari×24 + jam (Senin = 0)
  rating numeric(2,1), review_count int, stars int[],    -- stars = jumlah ulasan bintang 1..5
  about jsonb, is_claimed boolean,                       -- atribut "about" sumber; listing diklaim pemilik
  review_pctl smallint,                                  -- percent_rank kategori × kab/kota
  desa_id bigint, kec_id bigint, kab_id bigint, prov_id bigint,  -- NULL = di luar semua polygon/tanpa koordinat
  raw jsonb NOT NULL,              -- SourceRow terbaru (evidence; nama alias source.sql, di-TOAST)
  field_observed jsonb NOT NULL,   -- field → created_at snapshot asal nilainya (BR-NORM-05, BR-TRUST-02)
  first_observed_at timestamptz NOT NULL, observed_at timestamptz NOT NULL,  -- snapshot tertua & terbaru
  norm_version text NOT NULL, import_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('msp', coalesce(name, '')), 'A') ||
    setweight(to_tsvector('msp', coalesce(category_raw, '')), 'B') ||
    setweight(to_tsvector('msp', coalesce(address_text, '')), 'C')) STORED
);

CREATE TABLE estimates (                 -- satu baris per tempat kelompok makan/kafe/hotel; hanya ditulis est.compute
  place_id      uuid PRIMARY KEY REFERENCES places ON DELETE CASCADE,
  model_version text NOT NULL,           -- est-X.Y.Z
  params        jsonb NOT NULL,          -- parameter est.* yang dipakai baris ini (FR-EST-06)
  inputs        jsonb NOT NULL,          -- titik snapshot laju, harga, hari buka yang dipakai (evidence)
  data_as_of    date,                    -- snapshot terbaru yang dipakai (BR-TRUST-02)
  velocity_year numeric(10,1), velocity_recent numeric(10,1),        -- ulasan/tahun (E.1)
  visitors_low int, visitors_mid int, visitors_high int,             -- kunjungan/tahun, belum dibulatkan (E.2, E.3)
  spend_per_visit bigint, price_source text,   -- own | area_median | benchmark | unknown (E.2)
  revenue_low bigint, revenue_mid bigint, revenue_high bigint,       -- Rupiah/tahun; hotel = omzet kamar
  room_nights int, occupancy numeric(5,4),     -- hotel, nilai tengah (E.3)
  flags  text[] NOT NULL DEFAULT '{}',   -- five_star_heavy, velocity_changed, price_unknown, hotel_params_missing, occupancy_inconsistent
  reason text,                           -- diisi bila tidak ada estimasi sama sekali, mis. history_lt_30d (BR-EST-01)
  computed_at timestamptz NOT NULL
);
```

Nilai tak dibulatkan disimpan untuk filter dan urutan. Pembulatan BR-EST-02 hanya terjadi di proyeksi (§4.5). Kode `flags`/`reason` stabil; teks Indonesianya ada di 03.

### 4.2 Tabel global (dibaca semua workspace)

| Tabel | Kolom kunci | Catatan |
|---|---|---|
| `users` | `id` (= ID identitas Kratos), `email`, `name`, `is_ops` | Dibuat JIT saat request pertama |
| `places` | §4.1 | Satu baris per `cid` |
| `place_snapshots` | PK (`place_id`, `day`), `observed_at` (snapshot terakhir hari itu), `review_count`, `rating`, `stars`, `price_min`, `price_max` | Riwayat BR-NORM-05; dasar laju ulasan (E.1) dan grafik riwayat ulasan. `day` = tanggal lokal tempat |
| `estimates` | §4.1 | PRD Lampiran E (§5.4) |
| `imports` | `id`, `country_code`, `trigger` (`job`\|`cli`), `reset`, `cursor_at` + `cursor_id` (dimajukan per batch), `newest_at` (snapshot sumber terbaru yang diterima untuk negara ini), `status` (`running`\|`done`\|`failed`), `counts` jsonb, `rejects` jsonb (per alasan + 100 sampel `cid` + `created_at`), `warnings` jsonb, `error`, `started_at`, `finished_at` | Riwayat & status import (FR-OPS-01). **Watermark** = cursor run terakhir negara itu |
| `areas` | `id` bigint (GeoAPI), `country_code`, `admin_level` 1–4, `name`, `external_code`, `parent_id`, `area_km2`, `population`, `density`, `centroid`, `geom` MultiPolygon (level 4 saja), `synced_at` | GIST(`geom`). Populasi = Kontur 2023 (perkiraan model) |
| `area_metrics` | PK (`area_id`, `est_group`; `'*'` = semua), `business_count`, `density_per_1k`, `competition_pct`, `competition_level`, `est_n`, `demand_visits`, `demand_avg`, `price_p25`, `price_median`, `price_p75`, `limited`, `market_score`, `model_version` (`geo-X.Y.Z`) | Lampiran C, level 3–4, tidak bergantung ICP. NULL = tidak diketahui (ambang minimal Lampiran C) atau "Data terbatas" |
| `categories` | `id` (mis. `fnb.restoran`, `lainnya`), `parent_id`, `name`, `visitor_source` (`kesehatan`\|`akomodasi`\|`kantor_pendidikan`\|`ritel`\|NULL) | Taksonomi 2 level, seed dari `data/categories.yaml`. `visitor_source` = jenis sumber pengunjung siang (Lampiran F) |
| `peer_stats` | PK (`category_id`, `kab_id`), `n`, `url_fill`, `own_domain_share`, `social_fill`, `review_p25` | Kelompok pembanding BR-OPP-01 dan teks alasan |
| `place_gaps` | PK (`place_id`, `code`), `severity`, `params` jsonb (mis. `{"peer_n":212,"own_domain_pct":61}`) | Gap global. Relevansi Offering dihitung saat skor |
| `suppressions` | `id`, `kind` (`phone`\|`url`\|`place`), `value` (ternormalisasi), `reason`, `created_by`; UQ(`kind`,`value`) | FR-PRIV-02 |
| `flags` | `key` (`ai`, `export`, `webhook`), `enabled`, `updated_by`, `updated_at` | Kill switch global |
| `settings` | `key` text PK, `value` jsonb, `updated_by`, `updated_at` | Setting ops (D-29): kunci AI (§11) dan parameter estimasi `est.*` (§5.4). Nilai awal di-seed migrasi. Setiap perubahan diaudit |
| `plans` | `id`, `name` (UQ), `price_idr` bigint (harga tampilan), `seats`, `monthly_credits`, `monthly_ai_credits`, `active`, `updated_by`, `updated_at` | Dikelola ops (§12), tidak di-seed: nilai awal diisi ops dari 10 §3. `active=false` = tidak bisa dipilih lagi, workspace yang sudah memakainya tetap jalan |
| `audit_log` | `id`, `at`, `actor_id`, `workspace_id` NULL, `action`, `target_type`, `target_id`, `meta` jsonb, `ip` | Append-only (FR-OPS-02) |

**Mapping kategori (BR-NORM-03)** ada di satu file data, `data/categories.yaml`: taksonomi MSP, kelompok estimasi per kategori, label sumber dwibahasa, kelas akomodasi, dan `visitor_source`. File ini di-embed ke binary (dipakai `ingest`) dan men-seed tabel `categories`.

```yaml
- id: fnb.kafe
  name: Kafe
  est_group: kafe
  sources: { cafe: ~, kafe: ~, coffee shop: ~, kedai kopi: ~, bar: ~ }
- id: akomodasi.hotel
  name: Hotel
  est_group: hotel
  sources: { hotel: hotel, hotel bintang 3: hotel_3, 3-star hotel: hotel_3, resor: resort }  # label → accom_class
```

### 4.3 Tabel workspace (RLS)

| Tabel | Kolom kunci | Catatan |
|---|---|---|
| `workspaces` | `id`, `name`, `plan_id` (NULL = belum ada plan), `created_at` | Policy: workspace aktif atau milik membership user |
| `members` | `id`, `user_id` (NULL sampai undangan diterima), `email`, `role` (`admin`\|`member`), `status` (`invited`\|`active`), `invite_token_hash`, `active_icp_id` | UQ(`workspace_id`,`email`). ICP aktif per user (FR-ICP-02) |
| `offerings` | `id`, `name`, `tags` text[] (PRD §8), `description` | Tag pertama = tag primer untuk override bobot |
| `icps` | `id`, `offering_id`, `name`, `criteria` jsonb (§6.1), `score_status` (`pending`\|`scoring`\|`ready`), `candidates`, `scored`, `model_version`, `scored_at` | Progres scoring untuk UI |
| `icp_area_scores` | PK (`icp_id`, `area_id`), `level`, `matching_count`, `hot_warm_count` | Bagian Lampiran C yang bergantung ICP |
| `scores` | PK (`icp_id`, `place_id`), `priority`, `band`, `fit`, `coverage`, `limited`, `factors` jsonb, `top_opp`, `model_version`, `computed_at` | Hanya bisnis yang lolos kriteria *wajib* |
| `location_analyses` | `id`, `input` jsonb (titik, `radius_m`, `group`, harga rencana, `comparator_place_id`), `result` jsonb, `model_version` (`loc-X.Y.Z`), `est_version` (`est-X.Y.Z`), `data_as_of`, `idem_key` (UQ per workspace), `ledger_id`, `relevant` bool (NULL = belum dinilai), `created_by`, `created_at` | FR-LOC-01, FR-PRIV-01. Dibuka ulang tanpa dihitung ulang (§7.3). `relevant` = feedback kesimpulan (FR-INTL-05) |
| `lists` / `list_items` | `id`, `name`, `created_by` / PK (`list_id`, `place_id`), `added_by` | FR-ACT-01. Kolom estimasi dibaca dari `estimates` |
| `tags` · `assignments` | PK (`workspace_id`, `place_id`, `tag`) · PK (`workspace_id`, `place_id`), `owner_id` | Tag dan pemilik (FR-ACT-01) |
| `notes` | `id`, `place_id`, `author_id`, `body`, `created_at` | — |
| `saved_searches` | `id`, `name`, `query` jsonb (`q`, `filters`, `sort`, `icp_id`), `created_by` | FR-SRCH-02 |
| `feedback` | PK (`workspace_id`, `place_id`, `opp_code`), `relevant` bool, `user_id`, `at` | FR-INTL-05 untuk opportunity |
| `unlocks` | PK (`workspace_id`, `place_id`), `via` (`unlock`\|`export`\|`webhook`), `ledger_id`, `user_id`, `at` | Permanen (FR-ACT-02). Biaya kontak hanya tercatat di sini (`ledger_id`) + `credit_ledger` |
| `credit_ledger` | `id`, `meter` (`credit`\|`ai_credit`), `kind` (`grant`\|`debit`\|`refund`), `delta` int, `idem_key`, `ref` (`unlock`\|`location_analysis`\|`ai_brief`\|`grant`…), `actor_id`, `note`, `at`; UQ(`workspace_id`,`idem_key`) | Append-only (§12) |
| `exports` | `id`, `idem_key` (UQ per workspace), `format` (`csv`\|`xlsx`), `selection` jsonb, `status`, `rows`, `file_key`, `error`, `expires_at` | FR-ACT-03 |
| `webhooks` | `id`, `url`, `secret_enc` bytea, `active`, `created_by` | Secret dienkripsi AES-GCM (`MSP_ENCRYPTION_KEY`) |
| `webhook_deliveries` | `id`, `webhook_id`, `idem_key`, `event`, `place_ids` uuid[], `status` (`pending`\|`ok`\|`failed`), `attempts`, `last_code`, `last_error`, `done_at` | Log pengiriman. Payload dibangun ulang tiap percobaan, tidak disimpan |
| `ai_briefs` | `id`, `place_id` atau `analysis_id` (tepat satu terisi), `icp_id`, `input_hash`, `prompt_version`, `provider`, `model`, `status`, `output` jsonb, `error`, `expires_at` | Narasi AI. Cache 7 hari per workspace (§11) |

### 4.4 Pola RLS

```sql
ALTER TABLE scores ENABLE ROW LEVEL SECURITY;  ALTER TABLE scores FORCE ROW LEVEL SECURITY;
CREATE POLICY ws ON scores
  USING      (workspace_id = current_setting('app.workspace_id', true)::uuid)
  WITH CHECK (workspace_id = current_setting('app.workspace_id', true)::uuid);
-- members: tambahan OR user_id = current_setting('app.user_id', true)::uuid  (agar /me bisa daftar workspace)
```

```go
// platform/db — satu-satunya jalan ke tabel workspace
func (d *DB) InWorkspace(ctx context.Context, ws, user uuid.UUID, fn func(*q.Queries, pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, d.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `SELECT set_config('app.workspace_id', $1, true), set_config('app.user_id', $2, true)`,
			ws.String(), user.String()) // set_config(…, true) = SET LOCAL yang menerima parameter bind
		if err != nil {
			return err
		}
		return fn(q.New(tx), tx)
	})
}
```

- Setting tidak ada → `NULL` → tidak ada baris (fail closed).
- Dua role DB: `msp_owner` (pemilik objek, hanya `msp migrate`) dan `msp_app` (api, worker, tool; `NOBYPASSRLS`, bukan owner). `UPDATE`/`DELETE` pada `audit_log` dan `credit_ledger` di-`REVOKE` dari `msp_app`.
- Akses lintas workspace hanya lewat sedikit fungsi `SECURITY DEFINER` yang eksplisit: `accept_invite(token_hash, user_id)`, `icps_to_rescore()`, `ops_workspaces()` (juga dipakai `billing.monthly_grant`).

### 4.5 Claim type diturunkan saat dibaca (KISS)

Claim type tidak disimpan per nilai. Satu tabel di Go memetakan **field → claim type + asal**. Nilai `NULL` selalu menjadi **Tidak diketahui** (BR-TRUST-03).

```go
// intel/claims.go — satu tabel untuk search, profil, export, webhook, dan fakta Narasi AI (DRY)
var Fields = map[string]Field{
	"name":          {Claim: Observed, Path: "name"},
	"telephone":     {Claim: Observed, Path: "phone"},
	"whatsapp":      {Claim: Inferred, From: "phone", Rule: "BR-NORM-01"},
	"category":      {Claim: Derived, From: "category,categories", Rule: "BR-NORM-03"},
	"price":         {Claim: Derived, From: "price", Rule: "BR-NORM-04"},
	"popular_times": {Claim: Observed, Path: "popular_times"},
	"velocity":      {Claim: Derived, Rule: "est E.1"},
	"core_hours":    {Claim: Derived, Rule: "est E.4"},
	"visitors":      {Claim: Inferred, Rule: "est E.2/E.3"}, // juga revenue, occupancy
	"desa":          {Claim: Derived, From: "lat,lng", Rule: "FR-DATA-04"},
	"population":    {Claim: Inferred, Source: "GeoAPI · Kontur 2023 (perkiraan model)"},
} // ... opportunity selalu Inferred (BR-TRUST-01)
```

- **Bentuk nilai di API:** `{"value": "Kafe", "claim": "derived", "source": {"cid": "5398716892142984346", "snapshot_at": "2026-09-24T03:10:00Z", "path": "category", "rule": "BR-NORM-03"}}`. Estimasi: `{"value": {"low": 13800, "mid": 18400, "high": 27600}, "claim": "inferred", "source": {"model": "est-1.0.0", "data_as_of": "2026-09-24", "params": {…}}}`. F1 tidak punya field `confidence` (claim type sudah cukup). Kategori **Lainnya** cukup claim `derived` dengan nilai "Lainnya".
- **Evidence** = referensi snapshot sumber (`cid` + `snapshot_at` dari `field_observed`) + nilai mentah (`*_raw`) + `raw` (baris sumber terbaru, nama alias `source.sql`).
- **Masking (FR-ACT-02, FR-PRIV-02):** yang di-mask hanya **nomor telepon** (`+62 812-****-**90`, `"locked": true`) sampai di-unlock. Website dan akun sosial selalu terlihat. Nilai ter-suppress disembunyikan di semua tampilan dan tidak bisa di-unlock. Proyeksi claim + masking + estimasi (dibulatkan BR-EST-02) ada di **satu** fungsi `intel.ProspectView`, dipakai search, profil, export, webhook, dan fakta Narasi AI. Evidence (`raw`, kunci `phone`) melewati mask yang sama.
- **Freshness (BR-TRUST-02)** dihitung per nilai dari `field_observed`; estimasi memakai `estimates.data_as_of`.

## 5. Pipeline F1

```
msp tool geo-sync --country ID                                   (sekali; ulang bila GeoAPI berubah)
import.run (harian 03.00 WIB) │ msp tool import --country ID [--reset]                     (§5.1)
   └─► geo.assign ─► est.compute ─► geo.metrics ─► intel.derive ─► score.all ─► score.icp × N
       queue import  └─────────────────────────── queue score ───────────────────────────────┘
```

Setiap tahap adalah job River yang meng-enqueue tahap berikutnya di transaksi yang sama (`InsertTx`). Setiap tahap **menghitung ulang penuh** dan idempoten, jadi aman diulang. Pola untuk tahap setelah import: `pg_try_advisory_lock(<kind>)`; bila tahap yang sama sedang berjalan → `JobSnooze(30 s)`. Import punya kuncinya sendiri (§5.1). Import dan tahap rantai memakai `Timeout` per kind yang cukup panjang (default River 1 menit). Versi incremental baru di F3.

### 5.1 Import dari DB gmaps

Satu baris sumber = satu snapshot satu tempat (PRD §7.1). **Semua nama tabel dan kolom DB gmaps hanya ada di satu file: `apps/api/internal/ingest/gmaps/source.sql`** (di-embed, PRD §13). Query ini memberi alias tetap yang dibaca Go ke struct `SourceRow` (`pgx.RowToStructByName`). Bila skema scraper berubah, cukup file ini yang diubah (09 RB-03).

```sql
-- apps/api/internal/ingest/gmaps/source.sql — SATU-SATUNYA tempat nama kolom DB gmaps.
-- Alias = field SourceRow. Kolom yang tidak dipilih tidak pernah di-import (mis. user_reviews,
-- owner, status, images, description).
SELECT id, cid, place_id AS google_place_id, title AS name,
       category, categories, address, address_country AS country,
       latitude AS lat, longitude AS lng, timezone,
       phone, website, social_links, menu->>'link' AS menu_url, link AS maps_url,
       review_count, review_rating AS rating, reviews_per_rating AS stars, price_range AS price,
       open_hours, popular_times, about,
       coalesce(jsonb_typeof(owner) = 'object', false) AS is_claimed,
       created_at AS observed_at
FROM business_listings
WHERE (created_at, id) > ($1, $2)       -- keyset: cursor terakhir
  AND created_at < $3                   -- batas atas = mulai import − 1 jam
ORDER BY created_at, id
LIMIT $4;                               -- 5.000
```

- **Koneksi:** `MSP_SOURCE_DATABASE_URL` (akun read-only, NFR-05), pool maks. 2 koneksi, sesi `default_transaction_read_only=on` dan `statement_timeout`. Keyset butuh index `(created_at, id)` di tabel sumber. Jalur jaringan dari server ke DB gmaps belum diputuskan (Q-12). Di test/CI, URL ini menunjuk database `gmaps_fixture` di service `postgres` yang sama, berisi tabel `business_listings` tiruan (06).
- **Satu import per environment.** Sebelum mulai, import mengambil `pg_try_advisory_lock` dengan kunci tetap `import` dan memegangnya di satu koneksi khusus sampai selesai, termasuk selama import pertama yang panjang. Production dan staging punya DB sendiri, jadi kuncinya per environment. Bila kunci sedang dipegang: `msp tool import` keluar dengan pesan "import sedang berjalan", dan job harian dilewati. Import yang sedang berjalan sudah membaca sampai batas atasnya, dan sisanya terbaca di run berikutnya.

**Langkah:**

1. **Cek kolom (fail fast).** Query sumber dijalankan sekali dengan `LIMIT 0`. Kolom yang hilang menghasilkan error Postgres, dan alias yang tidak cocok dengan `SourceRow` gagal saat scan. Keduanya diteruskan sebagai `source schema mismatch: <detail>; perbaiki apps/api/internal/ingest/gmaps/source.sql`. Import berstatus `failed` tanpa menulis apa pun (alert di 09).
2. **Batch keyset.** Cursor awal = cursor `imports` terakhir negara itu (belum ada = seluruh riwayat). Batas atas = waktu mulai − 1 jam, supaya baris yang masih dalam transaksi scraper tidak terlewat. Per batch: baca, lalu langkah 3–7 dalam **satu transaksi MSP** yang juga memajukan `imports.cursor_*`. Bila proses mati, run berikutnya melanjutkan dari batch terakhir yang di-commit, sehingga import pertama bisa dilanjutkan. Import selesai saat batch kosong.
3. **Validasi.** Baris ditolak bila `missing_cid` atau `missing_name` (tempat baru tanpa nama). Peringatan per field (field diabaikan, baris tetap dipakai): `unmapped_category`, `invalid_geo`, `invalid_phone`, `invalid_rating`, `invalid_stars`, `invalid_popular_times` (nilai di luar 0–100, BR-NORM-06), `unparsed_price`.
4. **Filter negara (D-28).** `--country` harus ada di `data/countries.yaml` (kode ISO 3166-1, bbox, `active`). Negara baris = `country` (kode ISO dari sumber); bila kosong, koordinat di dalam bbox negara. Baris lain **dilewati** (bukan ditolak) dan dihitung `skipped_country`.
5. **Normalisasi** (fungsi murni, unit test per aturan, FR-DATA-02):

| Alias sumber | Aturan | Hasil |
|---|---|---|
| `phone` | libphonenumber, region = negara baris → E.164; `phone_kind` dari tipe nomor (BR-NORM-01). Satu fungsi untuk semua negara | `phone_e164`, `phone_kind`, `phone_raw` |
| `website`, `social_links` | Host website dicocokkan ke `data/url-patterns.yaml` (BR-NORM-02); `social_links` disimpan sebagai peta platform → URL | `url_kind`, `url_raw`, `social_links` |
| `category`, `categories` | `ingest.MapCategory` dengan `data/categories.yaml` (huruf kecil + trim, label Inggris & Indonesia). `category` menentukan kategori MSP dan kelompok estimasinya. Bila `category` tidak dikenal atau `lainnya`, label pertama di `categories` (urutan sumber) yang dikenal dipakai. Tidak ada yang dikenal → `lainnya` + `unmapped_category` | `category_id` (tidak pernah NULL), `est_group`, `accom_class`, `category_raw`, `categories_raw` |
| `price` | `ingest.ParsePrice` (BR-NORM-04, semua format PRD §7.1). `$`/`$$` atau format tak dikenal → NULL + `unparsed_price` | `price_min`, `price_max`, `price_mid` (`estimate.PriceMid`), `price_raw` |
| `open_hours` | Kunci hari Indonesia, format "08.00–23.00", melewati tengah malam, buka 24 jam, tutup → jadwal Senin–Minggu, waktu lokal (BR-NORM-06) | `opening_hours` |
| `popular_times` | Kunci hari Inggris → `smallint[168]` (Senin = 0, jam lokal 0–23); hari yang tidak ada = 0 (tutup); `null`/`{}` → NULL | `popular_times` |
| `rating`, `review_count`, `stars` | Rating 0–5, ulasan ≥ 0, rating tanpa ulasan → NULL; `stars` kunci "1"–"5" → `int[5]` | `rating`, `review_count`, `stars` |
| `lat`, `lng`, `timezone` | Koordinat valid dan di dalam bbox negara; zona IANA valid | `geom`, `timezone` |
| `about`, `is_claimed`, `menu_url`, `maps_url`, `google_place_id` | Disimpan apa adanya. Di F1 hanya ditampilkan di profil (`about`, `is_claimed` juga sebagai fakta Narasi AI) | kolom sama |
| `observed_at` (`created_at`) | Waktu snapshot | `field_observed`, `place_snapshots.day` (tanggal lokal) |

6. **Gabung snapshot (BR-NORM-05).** `ingest.Merge(cur, snap) (next, changed)` adalah fungsi murni. Baris diproses urut `created_at`. Untuk tiap field, nilai tidak kosong dari snapshot dengan waktu ≥ `field_observed[field]` menggantikan nilai sekarang. Pembanding ≥ membuat baca ulang idempoten. `is_claimed` mengikuti snapshot terbaru. `observed_at` = snapshot terbaru, `first_observed_at` = tertua, `raw` = `SourceRow` terbaru. Bila koordinat berubah, `desa_id`…`prov_id` dikosongkan supaya dihitung ulang oleh `geo.assign`.
7. **Tulis.** Upsert `places` per `cid` (hanya yang baru atau berubah). Upsert `place_snapshots` per (`place_id`, `day`): baris hari yang sama hanya diganti oleh snapshot yang lebih baru, jadi yang tersimpan adalah snapshot terakhir hari itu. Majukan `imports.newest_at`.
8. **Laporan** ke stdout + `imports.counts`: snapshot baru (diterima di run ini), tempat baru, tempat berubah, dilewati (`skipped_country`), ditolak per alasan, peringatan per alasan (FR-DATA-01). Bila ada snapshot diterima, `geo.assign` di-enqueue.

**Jadwal.** Periodic job River `import.run` (queue `import`) berjalan **tiap hari 03.00 WIB** untuk setiap negara `active`. Jadwal ini sengaja tidak bertabrakan dengan backup 01.00 WIB (06), dan rantai hitung ulang selesai sebelum jam kerja (NFR-02). `msp tool import --country ID` menjalankan fungsi yang sama secara sinkron. Bila `MSP_SOURCE_DATABASE_URL` kosong (mis. `local` tanpa akses), job tidak didaftarkan dan CLI gagal dengan pesan jelas.

**Semantik baca ulang.** Membaca ulang rentang yang sama tidak membuat duplikat: `cid` unik, `place_snapshots` ber-PK (`place_id`, `day`), dan merge idempoten. Bila normalizer atau `data/categories.yaml` berubah (`norm_version` naik), jalankan `msp tool import --country ID --reset`. Perintah ini membaca ulang seluruh riwayat lewat jalur yang sama dengan import pertama, lalu import harian berlanjut dari cursor. Tempat yang tidak muncul lagi **tidak dihapus** di F1 (deteksi usaha tutup ada di F2). Mengaktifkan negara lain kelak = `geo-sync` + `import` negara itu, tanpa perubahan skema.

### 5.2 `msp tool geo-sync`

1. Metadata semua level, **hanya negara `--country`** (FR-DATA-03): `GET /v1/areas?country=<kode>&admin_level=L&limit=500&offset=k` → upsert `areas` dengan `country_code`. Untuk ID: ±180 request untuk 34/522/7.069/81.912 area.
2. Polygon **hanya level 4** (desa/kelurahan): `GET /v1/areas/{id}?format=geojson&simplify=<toleransi kecil>` → `ST_Multi(ST_GeomFromGeoJSON(...))`. Level 1–3 cukup lewat `parent_id`. Peta memakai vector tiles GeoAPI.
3. Sopan terhadap batas 120 req/menit/IP: token bucket 1,5 req/s, ikuti `429` + `Retry-After`, backoff untuk 5xx. **Resumable**: area yang sudah punya `geom` dilewati (kecuali `--refresh`).
4. Setelah selesai, jumlah per level dicek terhadap jumlah yang diharapkan (ID: angka di atas), lalu `geo.assign` penuh di-enqueue.

Sync polygon pertama butuh ±82 ribu request (±15 jam), jadi dijalankan sekali di background. Per 2026-10-10, mode GeoJSON untuk daftar area di GeoAPI dibatasi 10.000 fitur dan tidak memfilter `parent_id`, sehingga polygon harus diambil per area.

### 5.3 Tahap job & hitung ulang

| Job (queue) | Isi |
|---|---|
| `import.run` (`import`) | §5.1 |
| `geo.assign` (`import`) | `ST_Contains` untuk place yang `desa_id IS NULL`. Titik di luar semua polygon atau tanpa koordinat tetap tanpa wilayah: tampil "Tidak diketahui" dan tidak ikut metrik wilayah. Target ≥ 98% (FR-DATA-04), dipantau metrik |
| `est.compute` (`score`) | Estimasi semua tempat kelompok `makan`/`kafe`/`hotel` (§5.4) |
| `geo.metrics` (`score`) | `area_metrics` level 3–4 per kelompok estimasi (+ `'*'`) sesuai Lampiran C: jumlah, densitas, kompetisi, permintaan terbukti (dari `estimates.visitors_mid`), rentang harga (`price_mid`), `market_score`. Pembanding persentil = area selevel dalam kab/kota yang sama; area tanpa bisnis ikut sebagai pembanding (n = 0) |
| `intel.derive` (`score`) | `review_pctl` (`percent_rank()`, BR-SCORE-03), `peer_stats`, lalu `place_gaps`: stream place + peer → `intel.DetectGaps()` (Lampiran B, BR-OPP-01) → isi tabel diganti dalam satu transaksi |
| `score.all` (`score`) | Enqueue `score.icp` untuk setiap ICP (`icps_to_rescore()`) |

```sql
-- geo.assign (inti)
UPDATE places p SET desa_id = d.id, kec_id = k.id, kab_id = k.parent_id,
       prov_id = (SELECT parent_id FROM areas WHERE id = k.parent_id)
FROM areas d JOIN areas k ON k.id = d.parent_id
WHERE d.admin_level = 4 AND d.country_code = p.country_code AND p.desa_id IS NULL
  AND p.geom && d.geom AND ST_Contains(d.geom, p.geom);
-- geo.metrics (kelurahan; kecamatan sama dengan kec_id): satu agregat per (area, kelompok) dan (area)
SELECT p.desa_id, p.est_group, count(*) AS n, count(e.visitors_mid) AS est_n, sum(e.visitors_mid) AS demand_visits,
       percentile_cont(0.5) WITHIN GROUP (ORDER BY p.price_mid) AS price_median   -- juga p25, p75
FROM places p LEFT JOIN estimates e ON e.place_id = p.id
GROUP BY GROUPING SETS ((p.desa_id, p.est_group), (p.desa_id));
-- lalu densitas, percent_rank per kab/kota, ambang minimal, dan market_score sesuai Lampiran C (geo-3.0.0)
```

**Satu jalur hitung ulang** (README): `est.compute` → `geo.metrics` (permintaan terbukti butuh estimasi) → `intel.derive` (gap butuh metrik wilayah) → `score.all` → `score.icp` × N. Tidak ada periodic job rescore; yang periodik hanya import harian.

| Pemicu | Aksi |
|---|---|
| Import (`import.run` / `msp tool import`) atau `msp tool geo-sync` selesai | `geo.assign` → rantai penuh |
| Parameter `est.*` diubah (`PUT /ops/settings/{key}`) | `est.compute` → rantai (enqueue di transaksi yang sama dengan perubahan + audit) |
| ICP disimpan (dibuat/diubah, termasuk Offering-nya diubah) | API meng-enqueue `score.icp` untuk ICP itu saja (transaksi yang sama) |
| `msp tool rescore` (manual, ops; setelah versi `est-`/`geo-`/`priority-` berubah) | `est.compute` → rantai |

### 5.4 Estimasi (`est.compute`)

Rumus = PRD Lampiran E. Semua rumus ada di paket `estimate` sebagai fungsi murni: data dan parameter masuk sebagai argumen, tanpa I/O dan tanpa `time.Now()` (08 STD-GO-09).

```go
// internal/estimate — satu-satunya tempat rumus PRD Lampiran E dan F
const Version, LocVersion = "est-1.0.0", "loc-1.0.0" // naik setiap kali hasil bisa berubah
func Velocity(series []DayCount) Velocity                                      // E.1: laju setahun & terkini + titik yang dipakai
func Estimate(in Input, p Params) Result                                        // E.2/E.3 + penanda FR-EST-05 + alasan BR-EST-01
func Pattern(pt []int16, hours Hours, p Params, guestsPerOpenDay *float64) PatternSummary // E.4
func AnalyzeLocation(in LocInput, nb []Neighbour, cmp *Neighbour, p Params) LocResult      // Lampiran F
func PriceMid(min int64, max *int64) int64                                      // E.2: rentang terbuka → min
func RoundVisitors(v int) int; func RoundRupiah(v int64) int64                  // BR-EST-02, hanya di proyeksi
```

Golden test PRD §12 memanggil fungsi ini langsung. Signa Cafe diuji lewat `Estimate` (dua snapshot, harga hasil `ingest.ParsePrice`, parameter default) dan `Pattern` (pola 7 hari asli → pola datar). Warung Kita diuji lewat `AnalyzeLocation` (pola KR dan Joglo). Kasus ujinya ditulis di 05.

**Parameter (D-29, D-32).** Parameter disimpan sebagai kunci `est.*` di tabel `settings` yang sudah ada dan diatur lewat `/ops/settings`, tanpa mekanisme konfigurasi baru. Nilai awal = PRD Lampiran E §E.5, di-seed migrasi.

| Kunci | Bentuk nilai |
|---|---|
| `est.review_rate` | Per kelompok (`makan`, `kafe`, `hotel`): `{"low","mid","high"}` berupa pecahan. Hotel di-seed sama ketiganya (10%) |
| `est.spend_factor` | Angka |
| `est.price_benchmark` | Per kelompok: `{"min","max"}` Rupiah atau `null` |
| `est.hotel` | Per kelas akomodasi (`hotel_1`…`hotel_5`, `hotel`, `resort`, `villa`, `penginapan`): `{"stay_nights","rooms","adr"}`; `null` = belum diisi |
| `est.flags` | `{"five_star_share","velocity_change"}` (FR-EST-05) |
| `est.pattern` | `{"core_hour_ratio","flat_ratio"}`: ambang jam inti dan pola datar (E.4) |

`PUT /ops/settings/{key}` memvalidasi nilai lewat `estimate.Params.Validate()` (pecahan 0–1, `low ≤ mid ≤ high`, Rupiah bilangan bulat ≥ 0, `min ≤ max`), mencatat audit, lalu meng-enqueue `est.compute` di transaksi yang sama (FR-EST-06).

**Job `est.compute`:**

1. Baca `est.*` sekali → `Params`.
2. SQL: median `price_mid` per (kab/kota, kelompok) yang memenuhi ambang E.2 → sumber harga ke-2.
3. Stream tempat kelompok `makan`/`kafe`/`hotel` (keyset `id`) beserta deret (hari, jumlah ulasan) 365 hari terakhir dari `place_snapshots` (`array_agg`) → `estimate.Estimate` → upsert batch 2.000, commit per batch.
4. Hapus baris dengan `computed_at` < mulai job (tempat yang keluar dari kelompok), set metrik cakupan, lalu enqueue `geo.metrics`.

Kelompok `lainnya` tidak punya baris; API menampilkan "Belum dapat diperkirakan" dengan alasan kelompok belum didukung. Pola keramaian tidak bergantung pada kelompok: `estimate.Pattern` dihitung saat profil dibaca dari `places.popular_times` (168 nilai, murah) dengan parameter saat ini.

## 6. Skor & ICP

### 6.1 ICP

```json
{ "categories":  { "ids": ["fnb.restoran"], "mode": "required" },
  "areas":       { "ids": [3374], "mode": "required" },
  "digital":     { "website_kinds": ["none", "social", "aggregator"], "mode": "preferred" },
  "min_rating":  { "value": 4.0, "mode": "preferred" },  "min_reviews": { "value": 20, "mode": "preferred" } }
```

| Kriteria | Kolom yang dibandingkan | Tidak diketahui bila |
|---|---|---|
| `categories` | `category_id` (+ `categories.parent_id`) | — |
| `areas` (ID GeoAPI level 1–4) | `desa_id`, `kec_id`, `kab_id`, `prov_id` | `desa_id` NULL |
| `digital` | `url_kind` (NULL = `none`) | — |
| `min_rating` / `min_reviews` | `rating` / `review_count` | NULL |

- Nilai cocok (1), sebagian (0,5), dan tidak cocok (0) per kriteria, termasuk kategori **Lainnya**, ditetapkan di PRD Lampiran A dan tidak disalin di sini.
- ICP Fit = rata-rata kriteria yang diketahui × 100. Kriteria *wajib* bernilai 0 → bukan kandidat (tidak tampil). Kriteria *wajib* yang tidak diketahui tetap kandidat, tetapi dibatasi WARM + "Data terbatas" (BR-SCORE-02).
- Kandidat difilter di SQL dengan predikat statis berparameter (mis. `(rating IS NULL OR rating >= @rating_floor)`; batas "sebagian" dihitung di Go dari Lampiran A), tanpa SQL dinamis.

### 6.2 Perhitungan dan penyimpanan

`intel.Score(facts, icp, cfg) Result` adalah **fungsi murni**: faktor, renormalisasi, coverage, band, dan cap persis sesuai Lampiran A dan BR-SCORE-01/02. Input + versi yang sama selalu memberi hasil yang sama; tie diurutkan dengan `place_id`. `factors` menyimpan breakdown untuk "Kenapa bisnis ini?": `{"icp_fit":{"v":100,"w":30},"opportunity":{"v":70,"w":25,"code":"OPP-WEB-SOCIAL"},"area":null,…}` (`null` = tidak diketahui, bobotnya dibagi ulang). Faktor Wilayah = `area_metrics.market_score` kelurahan untuk kelompok estimasi bisnis itu; bila kelurahan "Data terbatas", dipakai kecamatan.

Job `score.icp {workspace_id, icp_id}`:

1. `pg_try_advisory_lock(icp)`. Bila ICP sedang dihitung → `JobSnooze(30 s)`, sehingga hanya satu job per ICP yang berjalan.
2. Satu SQL agregat kandidat per kelurahan/kecamatan → `icp_area_scores.matching_count`.
3. Stream kandidat (+ gap, area, peer) **diurutkan dengan proksi murah** (severity gap relevan tertinggi, lalu jumlah ulasan) → `intel.Score` → upsert batch 2.000 dengan **commit per batch**, lalu update `icps.scored`.
4. Hapus skor lama (`computed_at` < mulai job), isi `hot_warm_count`, lalu set `score_status='ready'`, `model_version`, dan `scored_at`.

**Onboarding ≤ 5 menit (FR-ICP-01).** Jalur "Analisis lokasi" dan "Kompetitor & omzet" langsung memakai data global yang sudah dihitung. Pada jalur "Cari prospek", batch pertama berisi kandidat terbaik menurut proksi dan langsung di-commit, jadi halaman pertama tampil dalam hitungan detik setelah ICP disimpan. UI menampilkan progres `scored/candidates` dan me-refresh sampai statusnya `ready`. Pemicu hitung ulang ada di §5.3.

### 6.3 Konfigurasi bobot

Satu file berversi di repo, di-embed ke binary. Setiap perubahan bobot atau aturan berarti versi dinaikkan (semver).

```yaml
# data/scoring/priority-2.0.0.yaml
version: priority-2.0.0
weights: { icp_fit: 30, opportunity: 25, area: 15, contact: 15, popularity: 10, completeness: 5 }  # Lampiran A
overrides: {}   # per tag primer Offering, mis. reputation_mgmt: { opportunity: 30, popularity: 5 }
bands: { hot: 80, warm: 60 }
```

## 7. Search, profil & analisis lokasi

```sql
CREATE TEXT SEARCH CONFIGURATION msp (COPY = simple);   -- butuh extension unaccent dan pg_trgm
ALTER TEXT SEARCH CONFIGURATION msp ALTER MAPPING FOR hword, hword_part, word WITH unaccent, simple;
CREATE INDEX ON places USING gin (search);   CREATE INDEX ON places USING gin (name gin_trgm_ops);
CREATE INDEX ON places USING gist (geom);    CREATE INDEX ON places USING gist ((geom::geography));  -- ST_DWithin (§7.3)
CREATE INDEX ON scores (icp_id, priority DESC, place_id DESC);
```

Konfigurasi `simple` + `unaccent` dipakai tanpa stemming, karena cocok untuk nama usaha Indonesia. Karena regconfig ditulis eksplisit, `to_tsvector('msp', …)` immutable dan bisa dipakai di kolom generated. Trigram menangani salah ketik dan potongan nama.

### 7.1 Search (FR-SRCH-01)

```sql
-- name: SearchPlaces  (RLS membatasi scores ke workspace; tanpa ICP aktif → icp_id NULL)
SELECT * FROM (
  SELECT p.id, p.name, p.category_id, p.est_group, p.address_text,
         e.visitors_mid, e.revenue_mid, e.velocity_year, s.priority, s.band, s.top_opp, s.limited,
         CASE @sort::text WHEN 'score' THEN s.priority WHEN 'visitors' THEN e.visitors_mid
                          WHEN 'revenue' THEN e.revenue_mid ELSE e.velocity_year END::numeric AS sort_key
  FROM places p
  LEFT JOIN estimates e ON e.place_id = p.id
  LEFT JOIN scores s ON s.place_id = p.id AND s.icp_id = sqlc.narg(icp_id)
  WHERE (@q::text = '' OR p.search @@ websearch_to_tsquery('msp', @q) OR p.name % @q)
    AND (sqlc.narg(cats)::text[]   IS NULL OR p.category_id = ANY(sqlc.narg(cats)))
    AND (sqlc.narg(groups)::text[] IS NULL OR p.est_group = ANY(sqlc.narg(groups)))
    AND (sqlc.narg(area)::bigint   IS NULL OR sqlc.narg(area) IN (p.desa_id, p.kec_id, p.kab_id, p.prov_id))
    AND (sqlc.narg(rev_min)::bigint IS NULL OR e.revenue_mid >= sqlc.narg(rev_min))   -- juga rev_max
    AND (NOT @has_pattern OR p.popular_times IS NOT NULL)
    AND (sqlc.narg(bands)::text[]  IS NULL OR s.band = ANY(sqlc.narg(bands)))
    AND (sqlc.narg(opps)::text[]   IS NULL OR EXISTS (SELECT 1 FROM place_gaps g
           WHERE g.place_id = p.id AND g.code = ANY(sqlc.narg(opps))))
    AND (NOT @has_phone OR p.phone_e164 IS NOT NULL)        -- juga has_website, has_social, min_rating
    AND (@sort <> 'score' OR s.place_id IS NOT NULL)        -- urut skor = kandidat ICP aktif saja
) x
WHERE (coalesce(x.sort_key, -1), x.id) < (@cur_key, @cur_id)   -- keyset; nilai tidak diketahui di akhir
ORDER BY coalesce(x.sort_key, -1) DESC, x.id DESC
LIMIT @lim;   -- lim = limit + 1 untuk mendeteksi halaman berikutnya
```

- **Cakupan:** semua bisnis yang lolos filter. Kolom skor terisi bila bisnis itu kandidat ICP aktif. Urut `score` atau filter band → hanya kandidat ICP aktif (Lampiran A: yang gagal kriteria *wajib* tidak tampil); tanpa ICP aktif, `sort=score` → 422. Setiap baris dibentuk lewat `intel.ProspectView`, sama dengan profil.
- **Urutan:** `score` (default bila ada ICP aktif), `visitors` (default lainnya), `revenue`, `velocity`. Semuanya menurun, dengan nilai tidak diketahui di akhir. Filter omzet memakai `revenue_mid` (belum dibulatkan).
- **Pagination:** keyset; `cursor` = base64 dari (`sort_key`, `id`) terakhir; `limit` maks. 100.
- **Jumlah:** `SELECT count(*) FROM (<filter yang sama> LIMIT 10001)` → tampil "10.000+" bila melewati batas. Hasil selalu konsisten dengan filter.
- Index per kunci urut ditambahkan hanya bila p95 melewati NFR-01 (`EXPLAIN ANALYZE` di MR).

### 7.2 Profil & peta

- **Profil (FR-PROF-01):** satu query (place + estimasi + gap + wilayah + metrik + skor + unlock + suppression) ditambah satu query riwayat ulasan (`place_snapshots`: hari, jumlah ulasan, rating) → `intel.ProspectView`. Estimasi tampil sebagai rentang + parameter (`estimates.params`) + sumber harga + "Data per `data_as_of`"; tabel sensitivitas dibentuk dari kolom `estimates`. Pola keramaian (blok waktu, jam puncak, jam inti, pola datar, keramaian rata-rata, tamu per jam bila `visitors_mid` ada) dihitung saat dibaca dengan `estimate.Pattern`, beserta label zona waktu (WIB/WITA/WIT dari `timezone`) dan label "relatif terhadap jam tersibuk". `about` dan `is_claimed` ikut ditampilkan. Target p95 ≤ 500 ms (NFR-01).
- **Peta & peringkat wilayah (FR-GEO-01):** browser memuat TileJSON GeoAPI (`/v1/tiles/areas.json`, layer `areas`, `area_labels`) langsung dengan MapLibre. Warna diambil dari `GET /areas/ranking?parent_id=&level=&group=` lewat `feature-state` (kunci = ID area). Peringkat membaca `area_metrics` untuk kelompok yang dipilih, ditambah `icp_area_scores.hot_warm_count` bila ada ICP aktif. Atribusi TileJSON tampil di peta, dan tabel peringkat selalu tersedia sebagai padanan.

### 7.3 Analisis lokasi (FR-LOC-01)

Aturan = PRD Lampiran F. Dihitung **sinkron di api** dengan target p95 ≤ 3 s (NFR-01).

```sql
-- name: LocationNeighbours  (data global)
SELECT p.id, p.name, p.category_id, p.est_group, c.visitor_source, p.review_count, p.popular_times,
       p.price_mid, p.field_observed, ST_Distance(p.geom::geography, pt.g) AS distance_m
FROM places p JOIN categories c ON c.id = p.category_id,
     (SELECT ST_SetSRID(ST_MakePoint(@lng, @lat), 4326)::geography AS g) pt
WHERE ST_DWithin(p.geom::geography, pt.g, @radius_m)
ORDER BY distance_m;
```

`POST /location-analyses`:

1. **Input** (divalidasi OpenAPI): titik `{lat, lng}` atau `place_id`; `radius_m` 100–2.000 (default 500); `group` `makan`\|`kafe`\|`hotel`; `planned_price {min, max}` opsional; `comparator_place_id` opsional. Input "alamat" di UI berarti memilih hasil pencarian bisnis atau wilayah (`/prospects?q=`, `/areas?q=`) yang memberi titik. F1 tidak memakai geocoder eksternal.
2. Bila `Idempotency-Key` sudah ada di workspace, analisis yang tersimpan dikembalikan (200) tanpa hitung ulang dan tanpa tagihan.
3. Cek saldo cepat (`credit` < 1 → 402) supaya tidak menghitung sia-sia.
4. `LocationNeighbours` + data pembanding → `estimate.AnalyzeLocation` (parameter `est.*` saat ini) → `result`. Isinya: tetangga sejenis dan F&B lain beserta tanggal data masing-masing, pola, jam inti, saran jam buka, pembanding, posisi harga, sumber pengunjung siang (non-F&B hanya jumlah per `visitor_source` + 3 nama terdekat), keyakinan + saran cek lapangan, dan parameter yang dipakai.
5. Dalam satu transaksi `InWorkspace`: `billing.Debit` (1 `credit`, `ref = location_analysis`, §12) + `INSERT location_analyses` (`data_as_of` = snapshot terbaru di antara data yang dipakai) → `201` + `Location`.

- **Buka ulang** (`GET /location-analyses/{id}`) gratis dan membaca `result` apa adanya, lengkap dengan versi dan `data_as_of`. Analisis baru di titik yang sama = POST baru (berbayar) dengan data terbaru.
- **Feedback** atas kesimpulan → `PUT /location-analyses/{id}/feedback` → `relevant` (FR-INTL-05). **Narasi AI** → `POST /location-analyses/{id}/brief` (§11), dengan fakta dari `result`.
- Durasi dipantau lewat `msp_http_request_duration_seconds{route}`. Ada rate limit in-process per user (§8).

## 8. API

| Aspek | Aturan |
|---|---|
| Kontrak | `contracts/openapi.yaml` adalah sumber tunggal. Go: oapi-codegen (`std-http-server` + `strict-server`) → `internal/api`. TS: `openapi-typescript` + `openapi-fetch` → `packages/api-client`. Keduanya dibuat oleh `make generate`; CI gagal bila hasil generate berbeda |
| Basis | `/api/v1`, JSON, same-origin lewat Caddy. Workspace dipilih dengan header `MSP-Workspace: <uuid>`, divalidasi terhadap membership. Health di luar prefix: `/healthz`, `/readyz`; `/metrics` hanya di listener internal (§14) |
| Role | Ditulis sekali per operasi di OpenAPI: `x-msp-role: member\|admin\|ops`. Middleware membacanya lewat operationId; test matriks otorisasi memakai sumber yang sama |
| Error | `application/problem+json` (RFC 9457) + `code` stabil + `request_id`. Kode: `UNAUTHENTICATED` 401 · `INSUFFICIENT_CREDITS` 402 (extension `required`, `available`) · `FORBIDDEN`, `EMAIL_NOT_VERIFIED`, `MFA_REQUIRED`, `SEAT_LIMIT` (kuota seat plan habis saat mengundang, §12) 403 · `NOT_FOUND` 404 (termasuk milik workspace lain) · `CONFLICT` 409 (mis. `Idempotency-Key` dipakai ulang dengan body berbeda) · `VALIDATION_FAILED` 422 (extension `errors[]`), `WEBHOOK_URL_REJECTED` 422 · `RATE_LIMITED` 429 · `INTERNAL` 500 (tanpa detail internal) · `FEATURE_DISABLED`, `AI_UNAVAILABLE` 503 |
| Idempotensi | Header `Idempotency-Key` **wajib** di POST yang bisa mengubah saldo credit (`/unlocks`, `/exports`, `/webhooks/{id}/send`, `/location-analyses`, `/prospects/{id}/brief`, `/location-analyses/{id}/brief`, grant ops). Disimpan sebagai kunci unik di `credit_ledger` / `exports` / `webhook_deliveries` / `location_analyses`. Request ulang mengembalikan hasil yang sama tanpa tagihan baru (§12) |
| Async | `202 Accepted` + `Location` ke resource. Resource membawa `{status: queued\|running\|done\|failed, progress: {done, total}, error}` (export, kirim webhook, Narasi AI, scoring ICP). Analisis lokasi sinkron (`201`) |
| Batas | Body ≤ 1 MB, `limit` ≤ 100, **export maks. 10.000 baris per job** (FR-ACT-03; lebih → 422), kirim webhook ≤ 100 bisnis, radius analisis lokasi 100–2.000 m. Rate limit in-process per user untuk Narasi AI, export, dan analisis lokasi (`x/time/rate`; cukup karena hanya satu instance api) |

Role: **M** = Member (Admin juga boleh), **A** = Admin, **O** = ops (`is_ops` + AAL2). **$** = bisa mengubah saldo credit (export/kirim hanya lewat `unlock_missing`) + `Idempotency-Key` wajib.

| Method | Path | FR | Role |
|---|---|---|---|
| GET · PUT | `/me` (user, membership, role, ICP aktif, saldo), `/me/active-icp` | AUTH-01/02, ICP-02 | login · M |
| POST | `/workspaces` (onboarding) | AUTH-02 | login, email terverifikasi |
| GET · POST · PATCH · DELETE | `/members`, `/members/{id}` (undang = POST, cek seat → `SEAT_LIMIT`); `/invites/accept` | AUTH-02, BILL-01 | M baca · A tulis |
| GET · POST · PUT | `/offerings`, `/offerings/{id}` | ICP-01 | M baca · A tulis |
| GET · POST · PUT · DELETE | `/icps`, `/icps/{id}` (status scoring ikut di body) | ICP-01/02 | M baca · A tulis |
| GET | `/categories`, `/areas?q=&level=&parent_id=` | ICP-01, SRCH-01, LOC-01 | M |
| GET | `/prospects?q=&filters…&sort=&cursor=` (filter & urutan §7.1) | SRCH-01 | M |
| GET | `/prospects/{id}` (+ estimasi, pola keramaian, riwayat ulasan), `/prospects/{id}/evidence` | PROF-01, EST-01..05, INTL-01 | M |
| PUT | `/prospects/{id}/feedback` (per kode opportunity) | INTL-05 | M |
| GET · POST · DELETE | `/saved-searches`, `/saved-searches/{id}` | SRCH-02 | M |
| GET | `/areas/ranking?parent_id=&level=&group=` | GEO-01 | M |
| POST | `/location-analyses` → 201 (§7.3) | LOC-01 | M $ |
| GET | `/location-analyses`, `/location-analyses/{id}` (buka ulang gratis) | LOC-01 | M |
| PUT | `/location-analyses/{id}/feedback` | INTL-05 | M |
| GET · POST · DELETE | `/lists`, `/lists/{id}`, `/lists/{id}/items` | ACT-01 | M |
| PUT · POST | `/prospects/{id}/tags`, `/prospects/{id}/owner`, `/prospects/{id}/notes` | ACT-01 | M |
| POST | `/unlocks` (`place_ids` ≤ 100) | ACT-02 | M $ |
| POST | `/unlocks/preview` (sebelum "Unlock & export/kirim": jumlah, sudah di-unlock, ter-suppress, biaya, saldo) | ACT-02/03/04 | M |
| POST · GET | `/exports` → 202, `/exports/{id}` (memuat `download_url`) | ACT-03 | M $ |
| GET | `/exports/{id}/file?exp=&sig=` | ACT-03 | tanda tangan (§13) |
| GET · POST · PATCH · DELETE | `/webhooks`, `/webhooks/{id}`, `/webhooks/{id}/test`, `/webhooks/{id}/deliveries` | ACT-04 | A |
| POST | `/webhooks/{id}/send` (`place_ids` ≤ 100) → 202 | ACT-04 | M $ |
| POST · GET | `/prospects/{id}/brief`, `/location-analyses/{id}/brief` → 202, `/briefs/{id}` | ACT-05 | M $ (`ai_credit`) |
| GET | `/credits` (saldo per meter, plan & seat terpakai; riwayat ledger untuk Admin) | BILL-01 | M |
| GET · PATCH | `/ops/workspaces`, `/ops/workspaces/{id}` (ganti plan) | OPS-01, BILL-01 | O |
| POST | `/ops/workspaces/{id}/credits` (grant/refund manual) | OPS-01, BILL-01 | O $ |
| GET · POST · PUT | `/ops/plans`, `/ops/plans/{id}` (tanpa DELETE; nonaktifkan dengan `active: false`) | OPS-01, BILL-01 | O |
| GET · PUT | `/ops/settings`, `/ops/settings/{key}` (setting AI + status API key §11; parameter estimasi `est.*` §5.4) | OPS-01/02, ACT-05, EST-06 | O |
| POST | `/ops/settings/ai/test` (Uji koneksi, §11) | OPS-01, ACT-05 | O |
| GET | `/ops/imports` (snapshot terbaru, jeda, run berjalan, riwayat + laporan), `/ops/data-health`, `/ops/audit` | OPS-01/02 | O |
| GET · PUT · POST · DELETE | `/ops/flags`, `/ops/flags/{key}`, `/ops/suppressions`, `/ops/suppressions/{id}` | OPS-01, PRIV-02 | O |

`/ops/data-health` dihitung saat diminta dengan satu agregat atas `places` ⟕ `estimates`: jumlah bisnis, % terisi per field (termasuk harga dan `popular_times`), % bisnis dengan ≥ 2 snapshot berjarak ≥ 30 hari (`observed_at − first_observed_at`), % terpetakan wilayah, dan cakupan estimasi per kelompok (FR-OPS-01).

## 9. Auth (Kratos)

- **Same domain.** Caddy me-route `/.ory/*` ke Kratos public API. Cookie sesi Kratos (`HttpOnly`, `Secure`, `SameSite=Lax`) otomatis terkirim ke `/api/v1`. Admin API Kratos hanya bisa diakses dari jaringan Docker internal.
- **Browser flows.** Next.js merender UI login, registrasi, verifikasi, recovery, dan settings dari flow Kratos (`/self-service/*/browser`, `@ory/client`). Metode: password, **Google OIDC**, TOTP. Kratos mengirim email lewat courier SMTP-nya sendiri.
- **Middleware api.** Cookie → `GET {MSP_KRATOS_PUBLIC_URL}/sessions/whoami` (cache in-process 30 s per hash cookie) → `Principal{identity_id, email, verified, aal}`. Baris `users` dibuat JIT. Email belum terverifikasi → `403 EMAIL_NOT_VERIFIED` (FR-AUTH-01).
- **Membership & role** dibaca dari DB kita pada setiap request (tanpa cache), jadi anggota yang dikeluarkan langsung kehilangan akses. Undangan memakai token acak (hanya hash-nya yang disimpan di `members`), berlaku 7 hari, dan diterima lewat `accept_invite()`.
- **Ops (FR-AUTH-03).** `/api/v1/ops/*` mensyaratkan `users.is_ops` **dan** sesi `aal2` (TOTP). Tanpa itu → `403 MFA_REQUIRED`. Kratos `session.whoami.required_aal: highest_available` memaksa akun yang punya TOTP untuk memakainya.
- **CSRF.** Flow Kratos punya token CSRF sendiri. Untuk API: `SameSite=Lax`, request yang mengubah data wajib `Content-Type: application/json`, dan `Origin` harus sama dengan `MSP_PUBLIC_URL`. Kratos memakai database `kratos` di instance Postgres yang sama, dengan backup yang sama.

## 10. Webhook

- **Event F1:** `prospects.sent` (anggota mengirim bisnis terpilih) dan `webhook.test` (data contoh, tanpa biaya). Payload memakai `intel.ProspectView` yang sama dengan export (DRY), termasuk kolom estimasi: website dan sosial selalu ada, telepon terisi hanya untuk bisnis yang sudah di-unlock, dan nilai ter-suppress tidak pernah dikirim.
- **Biaya (BR-CRED-01):** sama dengan export. Bisnis yang belum di-unlock di-unlock dulu lewat fungsi unlock (§12, `unlock_missing: true`); pengiriman dan retry tidak menyentuh credit. Bisnis ter-suppress tidak di-unlock dan tidak ditagih (FR-PRIV-02).
- **Signature:** `MSP-Signature: t=<unix>,v1=<hex(HMAC-SHA256(secret, t + "." + body))>`, plus `MSP-Event` dan `MSP-Delivery`. Penerima disarankan menolak `|now − t| > 300 s` dan men-dedupe `MSP-Delivery`. Secret 32 byte hanya ditampilkan sekali, saat dibuat.
- **Retry:** job River `webhook.deliver` (queue `webhook`), `MaxAttempts` 7, `NextRetry` kustom **1 m, 5 m, 30 m, 2 j, 6 j, 12 j**. Sukses = 2xx. **Redirect tidak diikuti: 3xx = percobaan gagal** (di-retry sesuai jadwal). Percobaan habis → `failed` di log pengiriman dan bisa dikirim ulang manual. Flag `webhook` off → job di-snooze.
- **SSRF:** URL yang didaftarkan wajib `https`, tanpa userinfo, dan semua IP hasil resolve-nya harus lolos cek di bawah (bila tidak → `422 WEBHOOK_URL_REJECTED`). Cek yang **sama** diulang saat dial, sehingga DNS rebinding tidak lolos. Body respons dibaca maksimal 64 KB lalu dibuang. Fixture uji SSRF (IP privat, nama yang resolve ke 127.0.0.1, redirect ke metadata, IPv4-mapped IPv6) ada di 05.

```go
func dialGuard(_, address string, _ syscall.RawConn) error { // net.Dialer.Control: dicek di setiap koneksi
	host, _, _ := net.SplitHostPort(address)
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return err
	}
	if ip = ip.Unmap(); !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || inBlocked(ip) {
		return errBlockedDestination // link-local mencakup metadata 169.254.169.254; inBlocked: 0/8, 100.64/10, 198.18/15, 64:ff9b::/96, …
	}
	return nil
}

var webhookHTTP = &http.Client{
	Timeout:       10 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, // redirect tidak diikuti
	Transport: &http.Transport{Proxy: nil, // tanpa proxy env: guard harus melihat IP tujuan
		DialContext:     (&net.Dialer{Timeout: 3 * time.Second, Control: dialGuard}).DialContext,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}},
}
```

## 11. Narasi AI

Narasi AI menjelaskan hasil hitungan untuk dua subjek: profil bisnis dan analisis lokasi (FR-ACT-05). Nama teknisnya tetap `brief` (tabel `ai_briefs`, job `ai.brief`, endpoint `…/brief`, prompt `brief-X.Y.Z`).

```go
type LLM interface{ Complete(ctx context.Context, req Request) (Response, error) } // keluaran selalu JSON
type Request struct{ System, User string; Schema json.RawMessage; MaxTokens int }
type Response struct{ JSON []byte; Provider, Model string; InTokens, OutTokens int }
type Fallback struct{ Primary, Secondary LLM; Timeout time.Duration } // dirakit per panggilan dari setting AI
```

- **Setting AI (D-29).** Provider dan model dibaca dari tabel `settings` setiap kali narasi dibuat (cache in-process 10 s, sama dengan flag), jadi perubahan dari halaman ops berlaku tanpa deploy:

| Kunci | Isi | Seed migrasi |
|---|---|---|
| `ai.provider` | Provider utama: `anthropic` \| `openai_compatible` | `anthropic` |
| `ai.model` | Model provider utama | `claude-opus-5-5` |
| `ai.fallback.enabled` | Fallback aktif | `false` (diaktifkan ops setelah base URL dan model diisi) |
| `ai.fallback.base_url` | Base URL OpenAI-compatible, wajib `https` | `null` |
| `ai.fallback.model` | Model fallback | `null` |

- `PUT /ops/settings/{key}` memvalidasi nilai per kunci (enum, `https`, model tidak kosong; `ai.fallback.enabled=true` butuh base URL dan model) lalu mencatat audit. Satu-satunya model ID di repo adalah nilai seed di atas.
- **Secret tetap di env:** `MSP_ANTHROPIC_API_KEY`, `MSP_LLM_FALLBACK_API_KEY`. Model dan base URL **tidak** diambil dari env (KISS: satu sumber, tabel `settings`, tanpa aturan env vs DB mana yang menang).
- `GET /ops/settings` juga mengembalikan status key per provider (`key_set: true|false`), tidak pernah nilainya. `PUT` ditolak `422` bila key provider yang dipilih belum diset.
- **Uji koneksi:** `POST /ops/settings/ai/test` menerima nilai form (belum disimpan), mengirim satu prompt uji tetap tanpa data bisnis ke provider utama dan fallback, lalu mengembalikan per provider `ok`, latensi, dan pesan error. Tidak memakai AI credit dan tidak menyimpan setting.
- **Claude (`anthropic`):** SDK resmi `github.com/anthropics/anthropic-sdk-go`, model `ai.model`, structured output (`output_config.format` JSON Schema), effort `low` demi latensi. Error, timeout, `stop_reason` selain `end_turn` (mis. `refusal`, `max_tokens`), atau JSON tidak valid → fallback (bila aktif).
- **OpenAI-compatible:** `net/http` biasa ke `{ai.fallback.base_url}/chat/completions` dengan `MSP_LLM_FALLBACK_API_KEY`, `response_format: json_object`. Dipakai sebagai fallback (model `ai.fallback.model`), atau sebagai provider utama bila `ai.provider = openai_compatible` (model `ai.model`).
- **Prompt** `brief-2.0.0`: `internal/action/ai/prompts/brief-2.0.0/{system.md,user.tmpl,schema.json}` (di-embed). Satu prompt dipakai untuk kedua subjek. Versi dinaikkan setiap kali isinya berubah.
- **Input = daftar fakta ber-ID**, tanpa nomor telepon (BR-AI-01; unlock tidak mengubah aturan ini). Fakta profil berasal dari `ProspectView`: estimasi berupa rentang + parameter + tanggal data, pola, `about`, dan `is_claimed`. Fakta analisis lokasi berasal dari `location_analyses.result`. Contoh: `{"subject":"place","offering":["website_dev"],"facts":[{"id":"f1","text":"Perkiraan kunjungan 13.800–27.600/tahun (data per 24 Sep 2026)","claim":"inferred"}]}`.
- **Output:** `{"sections":[{"kind":"problem|data|reading|conclusion|limits","sentences":[{"text":"…","fact_ids":["f1"]}]}]}`, sesuai struktur FR-ACT-05 (masalah, data, cara membaca, kesimpulan, batasan). Validator deterministik: lolos schema, kelima bagian ada, 1–3 kalimat per bagian, setiap kalimat punya ≥ 1 `fact_id` yang ada di input, dan tidak ada pola telepon/email. Gagal → **regenerate sekali** dengan daftar pelanggaran. Gagal lagi → `failed`, **tidak ditagih** (FR-ACT-05, BR-CRED-01).
- **Cache:** `input_hash = sha256(prompt_version ‖ subjek ‖ fakta kanonik ‖ tag offering)`. Narasi `done` dengan hash sama, di workspace yang sama, dan berumur < 7 hari dikembalikan langsung tanpa biaya. Cache tidak pernah dibagi lintas workspace (RLS).
- **Alur:** POST mengecek flag `ai` (off → `503 FEATURE_DISABLED`) dan saldo meter `ai_credit` (kurang → 402) → job `ai.brief` (queue `ai`) → narasi disimpan (`ai_briefs.place_id` atau `analysis_id`) dan 1 `ai_credit` didebit lewat `billing.Debit` **dalam satu transaksi**, hanya saat sukses. Timeout 15 s per provider, batas job 45 s. Target p95 ≤ 20 s (NFR-02) dipantau metrik.

## 12. Credit & plan

- **Satu ledger, dua meter** (`credit`, `ai_credit`), append-only: grant `+n`, debit `−n`, refund `+n`. Saldo = `SUM(delta)` per meter (index `workspace_id, meter`). Tidak ada kolom saldo yang bisa berbeda dari ledger (FR-BILL-01).
- **Satu fungsi debit:** `billing.Debit(ctx, tx, ws, meter, n, idemKey, ref)` adalah satu-satunya jalur debit. Dipakai oleh unlock, analisis lokasi (Lampiran D), dan Narasi AI.
- **Idempotensi:** `UNIQUE (workspace_id, idem_key)`. Request ulang menabrak kunci unik, lalu handler mengembalikan hasil yang sudah tersimpan. PK `unlocks` (`workspace_id`, `place_id`) menjamin satu bisnis tidak pernah ditagih dua kali, walau `idem_key`-nya berbeda.
- **Konkurensi:** setiap debit diawali `SELECT … FROM workspaces WHERE id = $ws FOR UPDATE`, jadi debit dalam satu workspace berjalan serial dan saldo tidak bisa minus.
- **Suppression (FR-PRIV-02):** bisnis dengan kontak ter-suppress tidak pernah masuk `unlocks` dan tidak pernah dihitung biaya, baik lewat unlock, export, maupun webhook.

```sql
-- billing.Debit, di dalam InWorkspace (transaksi pemanggil)
SELECT 1 FROM workspaces WHERE id = $ws FOR UPDATE;
-- SUM(delta) WHERE meter = $meter < $n → 402 INSUFFICIENT_CREDITS
INSERT INTO credit_ledger (id, workspace_id, meter, kind, delta, idem_key, ref, actor_id)
VALUES ($id, $ws, $meter, 'debit', -$n, $idem_key, $ref, $user);   -- ref: unlock | location_analysis | ai_brief

-- POST /unlocks: n = place_ids yang belum di-unlock DAN tidak ter-suppress (yang ter-suppress dikembalikan "suppressed"); n = 0 → selesai
-- billing.Debit(credit, n, idem_key, 'unlock'), lalu:
INSERT INTO unlocks (workspace_id, place_id, via, ledger_id, user_id)
SELECT $ws, unnest($chargeable::uuid[]), 'unlock', $id, $user ON CONFLICT DO NOTHING;
INSERT INTO audit_log (...) VALUES (...);
```

**Analisis lokasi (Lampiran D).** Debit 1 `credit` dan `INSERT location_analyses` dilakukan dalam satu transaksi setelah hasil dihitung (§7.3). Hitungannya hanya membaca data global, jadi lock workspace tidak dipegang selama komputasi. Membuka ulang analisis yang tersimpan tidak menyentuh ledger.

**Export (FR-ACT-03).** (1) `POST /unlocks/preview` menghitung jumlah bisnis, yang sudah di-unlock, yang akan di-unlock (belum di-unlock dan tidak ter-suppress), biaya, dan saldo. (2) `POST /exports` (seleksi > 10.000 baris → 422) dengan `unlock_missing: true` memanggil **fungsi unlock yang sama** dengan `POST /unlocks` (transaksi pendek di atas, ditagih, `idem_key` dari request), lalu menyimpan `exports` + job `export.run` → 202. Tanpa `unlock_missing`, bisnis yang belum di-unlock diekspor dengan telepon ter-mask. (3) Worker menulis file ke storage → `status='done'`. Gagal → `failed`, bisa diulang **tanpa biaya**: export sendiri tidak pernah menyentuh ledger.

**Kirim ke webhook (FR-ACT-04).** Sama dengan export (DRY): `POST /webhooks/{id}/send` dengan `unlock_missing: true` menjalankan fungsi unlock (ditagih), lalu menyimpan `webhook_deliveries` + job → 202. Pengiriman dan retry **tidak menyentuh credit**, jadi tidak ada transaksi atau lock yang terbuka selama panggilan HTTP.

Refund dilakukan manual oleh ops (`kind='refund'`, tercatat di audit).

**Plan (FR-BILL-01, D-29).** Plan ada di tabel `plans` (§4.2) dan dikelola ops lewat `/ops/plans`. Ops memasang plan ke workspace lewat `PATCH /ops/workspaces/{id}` setelah pembayaran manual (F1) atau kesepakatan beta. Harga hanya untuk tampilan; tidak ada penagihan otomatis. Setiap perubahan plan dicatat di audit.

- **Grant bulanan:** periodic job River `billing.monthly_grant` (queue `billing`, tiap jam, `RunOnStart`). Untuk setiap workspace dengan plan (`ops_workspaces()`), job memanggil **fungsi grant yang sama** dengan grant manual ops (ledger + audit, aktor = sistem) untuk tiap meter yang kuotanya > 0, dengan `idem_key = grant:<workspace_id>:<YYYY-MM>:<meter>` (bulan kalender Asia/Jakarta; meter ikut di key karena `idem_key` unik per workspace). Job idempoten: jalan ulang, restart, atau retry tidak pernah memberi grant ganda. Workspace yang baru mendapat plan menerima grant bulan berjalan paling lambat satu jam kemudian. Ganti plan di tengah bulan tidak memberi grant kedua; selisihnya diberikan ops lewat grant manual bila perlu.
- **Seat:** `POST /members` (undang) mengunci baris workspace (`FOR UPDATE`, pola yang sama dengan debit), lalu menghitung anggota `active` + `invited`. Jumlahnya ≥ `plans.seats` → `403 SEAT_LIMIT`. Workspace tanpa plan hanya punya 1 seat (pembuatnya). Plan diturunkan → anggota yang ada tetap, undangan baru ditolak.

## 13. Storage

`storage.Storage` = `Put(ctx, key, io.Reader)` (tulis ke `key.tmp` lalu rename atomik), `Open(ctx, key)`, `Delete(ctx, key)`.

- F1 memakai implementasi `local` di `MSP_STORAGE_DIR` (volume bersama api + worker). Key berbentuk `exports/<workspace_id>/<export_id>.<ext>` dan divalidasi (tanpa `..`). Adapter S3 baru di F3, tanpa mengubah pemanggil.
- **URL unduh:** `GET /exports/{id}` (sudah lolos auth + RLS) mengembalikan `download_url = /api/v1/exports/{id}/file?exp=<unix+300>&sig=<base64url(HMAC-SHA256(k_dl, id‖exp))>`. `k_dl` diturunkan dari `MSP_ENCRYPTION_KEY` (HKDF, label `download`). Endpoint file memverifikasi tanda tangan dan masa berlaku, lalu streaming dengan `Content-Disposition: attachment`.
- Retensi 7 hari lewat periodic job harian `export.cleanup`. Direktori storage tidak di-backup karena file export bisa dibuat ulang.

## 14. Konfigurasi & observability

| Env var | Isi |
|---|---|
| `MSP_ENV`, `MSP_LOG_LEVEL` | `local` \| `staging` \| `production`; `info` (default) |
| `MSP_PUBLIC_URL` | `https://app.<domain>` (cek `Origin`, tautan email) |
| `MSP_HTTP_ADDR`, `MSP_METRICS_ADDR` | `:8080`, `:9090` |
| `MSP_DATABASE_URL`, `MSP_MIGRATE_DATABASE_URL` | Role `msp_app` (api, worker, tool); role `msp_owner` (hanya `msp migrate`) |
| `MSP_SOURCE_DATABASE_URL` | DB gmaps, akun read-only (§5.1). Kosong → job `import.run` tidak didaftarkan dan `msp tool import` gagal dengan pesan jelas. CI: database `gmaps_fixture` |
| `MSP_KRATOS_PUBLIC_URL`, `MSP_GEOAPI_BASE_URL` | `http://kratos:4433`; `https://geoapi.mcpke.dev/api` |
| `MSP_ANTHROPIC_API_KEY`, `MSP_LLM_FALLBACK_API_KEY` | API key Claude dan OpenAI-compatible. Provider, model, dan base URL ada di setting ops (§11) |
| `MSP_STORAGE_DIR` | `/var/lib/msp/storage` |
| `MSP_ENCRYPTION_KEY` | 32 byte base64: AES-GCM untuk secret webhook + kunci HKDF untuk URL unduh |
| `MSP_SMTP_URL` | Email undangan anggota (Kratos punya konfigurasi courier sendiri) |

- `platform/config` membaca env ke satu struct dan memvalidasinya saat start (fail fast). Env hanya untuk secret dan pengaturan proses (alamat, URL, level log). Data yang di-embed berubah lewat MR: `data/` (negara + `active`, kategori & mapping sumber, pola URL, bobot skor), prompt, dan `ingest/gmaps/source.sql`. Nilai yang diatur ops saat runtime ada di DB: `flags` dan `settings` (AI, `est.*`) dibaca dengan cache in-process 10 s, `plans` dibaca langsung. Setiap perubahannya tercatat di audit.
- **Log:** `slog` JSON ke stdout. Field: `time`, `level`, `msg`, `request_id`, `user_id`, `workspace_id`, `route`, `status`, `duration_ms`, `job_id`, `queue`. `ReplaceAttr` me-mask pola nomor telepon (`+62…`/`08…` → `+62******1234`). Body request dan respons tidak pernah di-log.

| Metrik | Untuk |
|---|---|
| `msp_http_request_duration_seconds{route,method,status}` | NFR-01 (termasuk analisis lokasi), error rate |
| `msp_jobs_total{queue,kind,result}`, `msp_job_duration_seconds{kind}` | Kesehatan worker |
| `msp_queue_depth{queue,state}` | Antrean menumpuk / job `discarded` |
| `msp_import_records_total{result}` | Laporan import (`accepted`, `skipped_country`, `rejected`) |
| `msp_import_last_success_timestamp_seconds{country}` | Waktu selesai import `done` terakhir. AL-10: `time() − … > 26 j` |
| `msp_source_newest_snapshot_timestamp_seconds{country}` | Snapshot sumber terbaru yang diterima (`imports.newest_at`). Jeda sumber = `time() − …`; AL-11: > 2 hari |
| `msp_estimate_coverage_ratio{group}` | Porsi tempat per kelompok yang punya perkiraan kunjungan (di-set `est.compute`) |
| `msp_places_area_assigned_ratio` | FR-DATA-04 (≥ 0,98) |
| `msp_credits_total{meter,kind}` | Pemakaian credit |
| `msp_webhook_deliveries_total{result}` | Pengiriman webhook |
| `msp_llm_requests_total{provider,model,result}`, `msp_llm_duration_seconds` | NFR-02, frekuensi fallback |
| `msp_geoapi_requests_total{status}` | Rate limit GeoAPI |

Dua gauge timestamp import dibaca dari tabel `imports` saat scrape (satu query kecil), jadi nilainya tetap benar setelah restart. Selain itu ada metrik runtime Go dan statistik pgxpool. `/healthz` = proses hidup (tanpa dependensi). `/readyz` = DB bisa di-ping dan versi migrasi sesuai (DB gmaps tidak ikut dicek, supaya api tetap siap walau sumber tidak terjangkau). `/metrics` hanya di `MSP_METRICS_ADDR` dan tidak di-route Caddy. Prometheus (`/opt/ops`, server yang sama) mengambilnya lewat network Docker internal (detail di 06).

**Uptime.** Monitoring berada di server yang sama, jadi ikut mati bila server mati. Karena itu ketersediaan (NFR-03) dicek **layanan uptime monitor eksternal gratis** yang memanggil `https://app.<domain>/readyz` tiap ≤ 5 menit dan mengirim alert ke email/Telegram. Alert dan dashboard lain ada di 06/09.

## 15. Keamanan F1

1. **Permukaan jaringan:** server hanya membuka 80/443 (satu Caddy untuk semua domain) dan SSH (key-only). Postgres, admin API Kratos, dan metrik `:9090` hanya ada di jaringan Docker (§14). Production dan staging tidak berbagi database maupun secret milik MesinProspek. **Pengecualian:** keduanya membaca DB gmaps eksternal yang sama secara read-only (bisa dengan kredensial yang sama). Koneksi keluar ini memakai akun read-only + `default_transaction_read_only`, dan wajib TLS bila melewati jaringan publik (jalur: Q-12).
2. **Isolasi workspace:** RLS `FORCE` + role `msp_app` `NOBYPASSRLS` + `InWorkspace` sebagai satu-satunya jalan, termasuk `location_analyses`. Resource milik workspace lain → 404. Test lintas workspace mencakup setiap endpoint (05).
3. **Identitas:** Kratos, email wajib terverifikasi, ops wajib AAL2 (TOTP). Role diperiksa per request dari `x-msp-role`.
4. **Secret:** variabel CI terproteksi → `.env` per environment (izin 600), termasuk `MSP_SOURCE_DATABASE_URL`. Tidak ada secret di repo, log, atau tabel `settings`. Secret webhook dienkripsi AES-GCM.
5. **Outbound:** guard SSRF di dial time untuk webhook, tanpa redirect. Host GeoAPI dan DB gmaps berasal dari env, host LLM dari setting ops (wajib `https`, hanya akun ops AAL2, diaudit). Semuanya tidak pernah dari input user workspace.
6. **Data kontak & pribadi:** telepon di-mask sampai di-unlock, dan suppression diterapkan di satu fungsi `intel.ProspectView` (search, profil, evidence, export, webhook). Telepon tidak dikirim ke LLM dan di-redact dari log. Kolom sumber yang memuat data orang (`user_reviews`, isi `owner`) tidak di-import (§5.1).
7. **Audit:** `audit_log` append-only (`REVOKE UPDATE, DELETE`) untuk login, undangan, unlock, export, webhook, grant, flag, suppression, plan, setting AI, dan parameter estimasi.
8. **Dependensi & input:** `govulncheck` + `npm audit` di CI (06). Batas ukuran body/limit/export/radius, validasi OpenAPI di setiap request, dan query hanya lewat sqlc (berparameter) atau `source.sql` (berparameter).

## 16. Evolusi F2/F3 (hanya bila dibutuhkan)

Daftar fitur F2/F3 ada di PRD §6 dan tidak disalin di sini. Tabel ini hanya mencatat kapan komponen atau perubahan teknis ditambahkan. Fitur F2 memakai pola yang sudah ada bila bisa (pencocokan customer = pg_trgm + jarak geo, integrasi CRM = pola webhook §10, NL search = LLM → filter kanonik §7.1, perbandingan bebas = proyeksi `ProspectView` + `area_metrics`).

| Fase | Komponen / perubahan teknis | Pemicu |
|---|---|---|
| F2 | Pembanding BPS (TPK hotel, penjualan F&B per provinsi): tabel referensi per provinsi, ditampilkan di samping estimasi | PRD §6 (F2) |
| F2 | Estimasi kategori lain (ritel, jasa): kelompok baru di `data/categories.yaml` + fungsi di `estimate` + versi `est-` naik | PRD §6 (F2) |
| F2 | Signals perubahan (usaha tutup/berubah) dari `place_snapshots` dan `observed_at` | Watch/alert perubahan (PRD §6) |
| F2 | Remap kategori langsung di DB dari `category_raw`/`categories_raw`, tanpa `--reset` | `--reset` terukur terlalu lama untuk perubahan mapping yang sering |
| F2 | Geocoding alamat bebas untuk analisis lokasi | User butuh titik yang tidak bisa dipilih dari peta, bisnis, atau wilayah |
| F2 | Proxy + cache tiles GeoAPI di Caddy | Banyak user di balik satu IP mencapai 120 req/menit |
| F2 | Error tracking | Volume error mulai sulit dibaca dari log |
| F3 | Filter negara di query sumber, batch import lebih besar | Import harian tidak selesai sebelum jam kerja (NFR-02) |
| F3 | Rantai hitung ulang incremental (hanya tempat berubah), batch scoring lebih besar | Dataset > 5 juta atau rantai > 1 jam |
| F3 | Partisi `place_snapshots` | Ukuran tabel atau query riwayat terukur lambat |
| F3 | Cache (Redis) | p95 search/analisis lokasi melewati NFR-01 setelah tuning query/index |
| F3 | Adapter `storage` S3 (presigned URL menggantikan URL bertanda tangan) | Server aplikasi kedua |
| F3 | GitLab + runner dipindah ke server terpisah | Contention terukur: p95 production naik saat CI atau import berjalan (PRD §13) |
| F3 | Server aplikasi kedua, replika Postgres untuk baca, PgBouncer, deploy tanpa downtime | Target 99,5% (NFR-03) |
| F3 | Partisi `scores`, mesin search terpisah | Hanya bila Postgres terbukti tidak cukup |
