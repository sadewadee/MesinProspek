# 04 — Technical Design (lean)

## MesinProspek — Sales Intelligence Engine

| Field | Value |
|---|---|
| Versi | 0.2 (lean). Menggantikan desain v0.1 yang terlalu berat untuk F1 |
| Tanggal | 2026-10-10 |
| Sumber kebenaran | Requirement: [PRD](01-prd.md) · Keputusan: [00 §7](00-analisis-dokumen-awal.md#7-register-keputusan-status-v02) · Nama teknis: [README](README.md#kosakata-teknis-bersama) |
| Cakupan | **F1 dirinci.** F2/F3 hanya daftar evolusi (§16) |

> Dokumen ini **tidak menyalin** rumus PRD. Rumus skor, katalog opportunity, dan metrik wilayah tetap di PRD Lampiran A–C. Di sini hanya dijelaskan cara menghitung dan menyimpannya.

## 1. Ringkasan & keputusan arsitektur

- Satu binary Go `msp` + satu PostgreSQL. Data, antrean, search, dan geo semuanya di Postgres. Tidak ada Redis, PgBouncer, object storage, atau search engine terpisah (D-26). Semuanya berjalan di **satu dedicated server** (D-19, §2). Data bisnis **global** (places, areas) dibaca semua workspace dan ditandai per negara (`country_code`); F1 hanya mengaktifkan `ID` (D-28). Data **workspace** diisolasi dengan RLS (D-03).
- Alur data: `msp tool geo-sync --country ID` (sekali) → `msp tool import --country ID` (berkala, tiap ada file dataset baru) → job River: wilayah → metrik → gap → skor per ICP. Logika inti (normalisasi, gap, skor) berupa **fungsi Go murni** yang mudah di-unit-test. SQL dipakai untuk agregat dan join.
- Nilai yang sering berubah (model AI, plan & kuota) disimpan di DB dan diatur dari halaman ops, bukan di kode atau env (D-29). Secret tetap di `.env`.

| ADR | Keputusan | Rujukan |
|---|---|---|
| ADR-01 | Modular monolith: satu binary `msp` (api, worker, migrate, tool), modul sedikit dan datar | D-11 |
| ADR-02 | PostgreSQL 16 + PostGIS + pg_trgm + unaccent untuk data, antrean River, FTS, dan point-in-polygon | D-12, D-26 |
| ADR-03 | Data global + data workspace dalam satu DB; tabel workspace wajib RLS `app.workspace_id` | D-03, FR-PRIV-01 |
| ADR-04 | Claim type diturunkan **saat dibaca** dari asal field; record JSON mentah disimpan sebagai evidence | BR-TRUST-01 |
| ADR-05 | Skor disimpan per (ICP, place), dihitung di Go, ditulis per batch (hasil pertama muncul cepat) | FR-INTL-04, FR-ICP-01 |
| ADR-06 | OpenAPI-first: `contracts/openapi.yaml` → server Go + client TS di-generate | README |
| ADR-07 | Identitas di Kratos; DB kita hanya menyimpan user, membership, dan role | D-20 |
| ADR-08 | Credit = ledger append-only, saldo = `SUM`, debit diserialkan dengan lock baris workspace | BR-CRED-01 |
| ADR-09 | Webhook keluar langsung dari worker; guard SSRF di dial time, tanpa egress proxy | FR-ACT-04, NFR-05 |
| ADR-10 | File export di disk lokal lewat interface `storage`; unduh lewat URL bertanda tangan berumur pendek | D-26 |
| ADR-11 | Satu server untuk production, staging, dan ops; isolasi lewat project Compose terpisah + batas CPU/RAM per container | D-19, D-26 |
| ADR-12 | Konfigurasi runtime milik ops (setting AI, plan) di tabel `settings` dan `plans`, dibaca dengan cache singkat; env hanya untuk secret dan pengaturan proses | D-29 |

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
Eksternal: worker → GeoAPI (geo-sync) · worker → LLM sesuai setting AI (§11)
           worker → URL webhook customer (§10) · kratos → Google OIDC, SMTP · api → SMTP (undangan)
```

| Proses | Perintah | Catatan |
|---|---|---|
| `api` | `msp api` | HTTP `:8080`, metrik `:9090` (listener internal) |
| `worker` | `msp worker` | Semua queue River. Concurrency: `import` 1, `score` 2, `export` 2, `webhook` 10, `ai` 4, `billing` 1 |
| migrasi | `msp migrate` | goose + migrasi River, dijalankan pipeline deploy sebelum `api`/`worker` diganti (06) |
| tool | `msp tool import\|geo-sync\|rescore` | `docker compose run --rm worker msp tool …` di `/opt/msp` (production) |

Satu `caddy` memegang port 80/443 dan me-route semua domain (README). Production dan staging adalah project Compose terpisah dengan network default, database, volume, dan `.env` sendiri, jadi staging tidak bisa menjangkau database production. Detail Compose, Caddyfile, dan bootstrap ada di 06.

### 2.1 Kapasitas server (perkiraan)

Saran awal untuk **dataset Indonesia saja**: **16 vCPU, 64 GB RAM, 2× NVMe ±2 TB (RAID1)**. Angka ini perkiraan dan diverifikasi setelah import pertama (Q-09). RAID1 melindungi dari disk rusak, bukan pengganti backup off-site.

| Kelompok container | Batas CPU | Batas RAM | Catatan |
|---|---|---|---|
| Production `postgres` | 4 | 16 GB | Sisa RAM host dipakai page cache Postgres |
| Production `api`, `worker`, `web`, `kratos` + `caddy` bersama | 3 | 6 GB | `worker` mendapat porsi terbesar (import, skor) |
| Staging (semua service) | 2 | 6 GB | Dataset nyata untuk gerbang rilis (PRD §12) |
| `gitlab` (+ registry) | 4 | 8 GB | Ukuran GitLab CE untuk tim kecil |
| `gitlab-runner` + job CI | 2 | 4 GB | `concurrent = 1` di `config.toml`: satu job sekaligus |
| Monitoring (`prometheus`, `alertmanager`, `grafana`, `node-exporter`) | 1 | 2 GB | — |
| **Total batas** | 16 | 42 GB | ±22 GB sisa untuk OS dan page cache |

- **Batas per container wajib** di setiap file Compose (`deploy.resources.limits.cpus` / `.memory`, pembagian per service di 06), sehingga build CI atau import besar tidak membuat production kehabisan CPU/RAM.
- **Disk (perkiraan):** DB production dan DB staging masing-masing puluhan sampai ±150 GB (raw JSON, index, polygon desa, skor), GitLab + registry ±200 GB dengan cleanup policy. Sisanya untuk tabel staging import, WAL lokal, dan pertumbuhan.
- **Bila contention terukur** (p95 naik saat pipeline CI atau import berjalan), GitLab/CI dipindah ke server lain (§16, PRD §13).

## 3. Modul Go

Tata letak: `apps/api/cmd/msp` (subcommand), `internal/<modul>`, `internal/api` (hasil generate oapi-codegen, jangan diedit), `db/migrations` (goose SQL), `db/queries` (sqlc → `internal/platform/db/q`). Target Makefile yang sama dipakai lokal dan CI: `make generate` (oapi-codegen, client TS, sqlc, design tokens), `make lint`, `make test`.

| Modul | Isi (satu baris) |
|---|---|
| `platform` | `config` (env), `db` (pgxpool, `InWorkspace`, sqlc), `httpx` (router, problem+json, middleware), `auth` (Kratos whoami → `Principal`), `storage`, `settings` (tabel `flags` + `settings`, cache in-process 10 s), `audit`, `crypto` (AES-GCM, HMAC), `jobs` (River client + semua struct args job) |
| `ingest` | Baca JSONL(.gz), validasi, normalisasi (BR-NORM), upsert `places`, laporan import |
| `geo` | Client GeoAPI + `geo-sync`, penetapan wilayah (PostGIS), metrik wilayah (Lampiran C) |
| `intel` | Tabel claim, deteksi gap (Lampiran B), skor (Lampiran A), search, profil, `ProspectView` (satu proyeksi claim + masking untuk search/profil/export/webhook) |
| `workspace` | Workspace, anggota & undangan (+ batas seat plan), offering, ICP, list, tag, pemilik, catatan, saved search, feedback |
| `action` | Unlock, export, webhook (+ guard SSRF), AI brief (interface `LLM`) |
| `billing` | Ledger credit, saldo, grant/refund, plan, grant bulanan (`billing.monthly_grant`) |
| `ops` | Endpoint ops: workspace (plan, grant), plan, setting AI, riwayat import, kesehatan data, flag, suppression, audit |

**Aturan dependensi.** `platform` tidak mengimpor modul domain. Urutan impor yang diizinkan: `geo` → `intel` → `workspace` → `billing` → `action` → `ops` (modul kanan boleh mengimpor modul kiri, tidak sebaliknya). `ingest` hanya mengimpor `intel`. Semua query lewat paket sqlc bersama. Struct args job ada di `platform/jobs`, jadi enqueue tidak perlu mengimpor modul pelaksana. Siklus impor ditolak compiler Go.

## 4. Data model

**Konvensi.** Satu schema `public`. ID entity kita **UUIDv7** (`uuid`, dibuat di Go `uuid.NewV7()`, terurut waktu). Pengecualian: `places.google_place_id` (kunci alami, `UNIQUE`), `areas.id` = ID integer GeoAPI (D-13), `categories.id` = slug teks. Waktu `timestamptz` (UTC), ditampilkan Asia/Jakarta. Tabel workspace punya `workspace_id uuid NOT NULL` + RLS.

### 4.1 Tabel inti `places`

```sql
CREATE TABLE places (
  id              uuid PRIMARY KEY,
  google_place_id text NOT NULL UNIQUE,
  country_code char(2) NOT NULL,                         -- ISO 3166-1 alpha-2, negara aktif saat import (D-28)
  name text NOT NULL, description text,
  schema_type text, cuisines text[],                     -- @type terpilih, servesCuisine
  category_id text NOT NULL REFERENCES categories, category_label text,
  phone_raw text, phone_e164 text, phone_kind text,      -- 'mobile' | 'fixed'
  url_raw text, url_kind text,  -- own_domain|social|aggregator|messaging|free_subdomain; NULL = kosong
  social_links text[], address jsonb, address_text text, maps_url text,
  geom geometry(Point, 4326), opening_hours jsonb,
  rating numeric(2,1), review_count int, review_pctl smallint,  -- pctl: percent_rank kategori × kab/kota
  desa_id bigint, kec_id bigint, kab_id bigint, prov_id bigint,  -- NULL = di luar semua polygon/tanpa koordinat
  raw jsonb NOT NULL, raw_hash bytea NOT NULL, norm_version text NOT NULL,       -- evidence (di-TOAST)
  import_id uuid NOT NULL, observed_at date NOT NULL,                            -- BR-TRUST-02
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('msp', coalesce(name, '')), 'A') ||
    setweight(to_tsvector('msp', coalesce(category_label, '')), 'B') ||
    setweight(to_tsvector('msp', coalesce(address_text, '')), 'C')) STORED
);
```

### 4.2 Tabel global (dibaca semua workspace)

| Tabel | Kolom kunci | Catatan |
|---|---|---|
| `users` | `id` (= ID identitas Kratos), `email`, `name`, `is_ops` | Dibuat JIT saat request pertama |
| `places` | §4.1 | Satu baris per `google_place_id` |
| `imports` | `id`, `country_code`, `file_name`, `file_sha256`, `observed_at`, `status`, `counts` jsonb, `rejects` jsonb (per alasan + 100 sampel baris), `health` jsonb, `started_at`, `finished_at` | Riwayat import + ringkasan kesehatan data (FR-OPS-01) |
| `areas` | `id` bigint (GeoAPI), `country_code`, `admin_level` 1–4, `name`, `external_code`, `parent_id`, `area_km2`, `population`, `density`, `centroid`, `geom` MultiPolygon (level 4 saja), `synced_at` | GIST(`geom`). Populasi = Kontur 2023 (perkiraan model) |
| `area_metrics` | PK (`area_id`, `category_id`; `'*'` = semua), `business_count`, `density_per_1k`, `competition_pct`, `competition_level`, `limited`, `model_version` (`geo-X.Y.Z`) | Level 3–4, tidak bergantung ICP |
| `categories` | `id` (mis. `fnb.restoran_asia`, `lainnya`), `parent_id`, `name` | Taksonomi 2 level, seed dari `data/categories.yaml` |
| `category_map` | PK (`schema_type`, `cuisine`; `'*'` = apa saja) → `category_id`, `type_depth` | BR-NORM-03. `type_depth` = kedalaman tipe di hierarki schema.org. Seed dari file yang sama (DRY) |
| `peer_stats` | PK (`category_id`, `kab_id`), `n`, `url_fill`, `own_domain_share`, `same_as_fill`, `review_p25` | Kelompok pembanding BR-OPP-01 dan teks alasan |
| `place_gaps` | PK (`place_id`, `code`), `severity`, `params` jsonb (mis. `{"peer_n":212,"own_domain_pct":61}`) | Gap global. Relevansi Offering dihitung saat skor |
| `suppressions` | `id`, `kind` (`phone`\|`url`\|`place`), `value` (ternormalisasi), `reason`, `created_by`; UQ(`kind`,`value`) | FR-PRIV-02 |
| `flags` | `key` (`ai`, `export`, `webhook`), `enabled`, `updated_by`, `updated_at` | Kill switch global |
| `settings` | `key` text PK, `value` jsonb, `updated_by`, `updated_at` | Setting ops (D-29). F1 hanya kunci AI (§11). Nilai awal di-seed migrasi. Setiap perubahan diaudit |
| `plans` | `id`, `name` (UQ), `price_idr` bigint (harga tampilan), `seats`, `monthly_credits`, `monthly_ai_credits`, `active`, `updated_by`, `updated_at` | Dikelola ops (§12), tidak di-seed: nilai awal diisi ops dari 10 §3. `active=false` = tidak bisa dipilih lagi, workspace yang sudah memakainya tetap jalan |
| `audit_log` | `id`, `at`, `actor_id`, `workspace_id` NULL, `action`, `target_type`, `target_id`, `meta` jsonb, `ip` | Append-only (FR-OPS-02) |

### 4.3 Tabel workspace (RLS)

| Tabel | Kolom kunci | Catatan |
|---|---|---|
| `workspaces` | `id`, `name`, `plan_id` (NULL = belum ada plan), `created_at` | Policy: workspace aktif atau milik membership user |
| `members` | `id`, `user_id` (NULL sampai undangan diterima), `email`, `role` (`admin`\|`member`), `status` (`invited`\|`active`), `invite_token_hash`, `active_icp_id` | UQ(`workspace_id`,`email`). ICP aktif per user (FR-ICP-02) |
| `offerings` | `id`, `name`, `tags` text[] (PRD §8), `description` | Tag pertama = tag primer untuk override bobot |
| `icps` | `id`, `offering_id`, `name`, `criteria` jsonb (§6.1), `score_status` (`pending`\|`scoring`\|`ready`), `candidates`, `scored`, `model_version`, `scored_at` | Progres scoring untuk UI |
| `icp_area_scores` | PK (`icp_id`, `area_id`), `level`, `market_score` (NULL = data terbatas), `matching_count`, `hot_warm_count` | Lampiran C bagian yang bergantung ICP |
| `scores` | PK (`icp_id`, `place_id`), `priority`, `band`, `fit`, `coverage`, `limited`, `factors` jsonb, `top_opp`, `model_version`, `computed_at` | Hanya prospek yang lolos kriteria *wajib* |
| `lists` / `list_items` | `id`, `name`, `created_by` / PK (`list_id`, `place_id`), `added_by` | FR-ACT-01 |
| `tags` · `assignments` | PK (`workspace_id`, `place_id`, `tag`) · PK (`workspace_id`, `place_id`), `owner_id` | Tag dan pemilik prospek (FR-ACT-01) |
| `notes` | `id`, `place_id`, `author_id`, `body`, `created_at` | — |
| `saved_searches` | `id`, `name`, `query` jsonb (`q`, `filters`, `icp_id`), `created_by` | FR-SRCH-02 |
| `feedback` | PK (`workspace_id`, `place_id`, `opp_code`), `relevant` bool, `user_id`, `at` | FR-INTL-05 |
| `unlocks` | PK (`workspace_id`, `place_id`), `via` (`unlock`\|`export`\|`webhook`), `ledger_id`, `user_id`, `at` | Permanen (FR-ACT-02). Biaya kontak hanya tercatat di sini (`ledger_id`) + `credit_ledger` |
| `credit_ledger` | `id`, `meter` (`credit`\|`ai_credit`), `kind` (`grant`\|`debit`\|`refund`), `delta` int, `idem_key`, `ref`, `actor_id`, `note`, `at`; UQ(`workspace_id`,`idem_key`) | Append-only (§12) |
| `exports` | `id`, `idem_key` (UQ per workspace), `format` (`csv`\|`xlsx`), `selection` jsonb, `status`, `rows`, `file_key`, `error`, `expires_at` | FR-ACT-03 |
| `webhooks` | `id`, `url`, `secret_enc` bytea, `active`, `created_by` | Secret dienkripsi AES-GCM (`MSP_ENCRYPTION_KEY`) |
| `webhook_deliveries` | `id`, `webhook_id`, `idem_key`, `event`, `place_ids` uuid[], `status` (`pending`\|`ok`\|`failed`), `attempts`, `last_code`, `last_error`, `done_at` | Log pengiriman. Payload dibangun ulang tiap percobaan, tidak disimpan |
| `ai_briefs` | `id`, `place_id`, `icp_id`, `input_hash`, `prompt_version`, `provider`, `model`, `status`, `output` jsonb, `error`, `expires_at` | Cache 7 hari per workspace |

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
// intel/claims.go — satu tabel untuk search, profil, export, dan webhook (DRY)
var Fields = map[string]Field{
	"name":         {Claim: Observed, Path: "name"},
	"telephone":    {Claim: Observed, Path: "telephone"},
	"whatsapp":     {Claim: Inferred, From: "telephone", Rule: "BR-NORM-01"},
	"category":     {Claim: Derived, From: "@type,servesCuisine", Rule: "BR-NORM-03"},
	"website_kind": {Claim: Derived, From: "url", Rule: "BR-NORM-02"},
	"desa":         {Claim: Derived, From: "geo", Rule: "FR-DATA-04"},
	"population":   {Claim: Inferred, Source: "GeoAPI · Kontur 2023 (perkiraan model)"},
} // ... opportunity selalu Inferred (BR-TRUST-01)
```

- **Bentuk nilai di API:** `{"value": "Restoran Asia", "claim": "derived", "source": {"import_id": "…", "path": "@type,servesCuisine", "rule": "BR-NORM-03", "observed_at": "2026-08-01"}}`. F1 tidak punya field `confidence` (claim type sudah cukup; confidence baru di F2). Kategori **Lainnya** cukup claim `derived` dengan nilai "Lainnya".
- **Masking (FR-ACT-02, FR-PRIV-02):** yang di-mask hanya **nomor telepon** (`+62 812-****-**90`, `"locked": true`) sampai di-unlock. Website dan akun sosial selalu terlihat. Nilai ter-suppress disembunyikan di semua tampilan dan tidak bisa di-unlock. Proyeksi claim + masking ada di **satu** fungsi `intel.ProspectView`, dipakai search, profil, export, dan webhook. Evidence (`raw`) melewati mask yang sama.
- **Freshness (BR-TRUST-02)** dihitung dari `places.observed_at`.

## 5. Pipeline F1

```
msp tool geo-sync --country ID                               (sekali; ulang bila GeoAPI berubah)
msp tool import --country ID --observed-at 2026-08-01 [--dry-run] file.jsonl.gz   (tiap file baru)
   └─► geo.assign ─► geo.metrics ─► intel.derive ─► score.all ─► score.icp × N
       └──────────── queue import ──────────────┘   └──── queue score ────┘
```

Setiap tahap adalah job River yang meng-enqueue tahap berikutnya di transaksi yang sama (`InsertTx`). Setiap tahap **menghitung ulang penuh** dan idempoten, jadi aman diulang. Versi incremental baru di F3. Semua pemicu rantai ini ada di §6.2.

### 5.1 `msp tool import`

1. Stream baris (`bufio.Reader`, gzip dideteksi otomatis, baris maks. 1 MB). File tidak dimuat utuh ke memori.
2. Validasi. Alasan tolak: `invalid_json`, `missing_place_id`, `missing_name`, `duplicate_in_file`. Peringatan (record tetap masuk): `unmapped_type`, `invalid_geo`, `invalid_phone`.
3. **Filter negara (D-28).** `--country` wajib dan harus ada di `data/countries.yaml` (kode ISO 3166-1, alias nama, bbox; satu file untuk semua negara). Negara record = `address.addressCountry` (kode atau nama, dicocokkan ke alias). Bila kosong, dipakai `geo` di dalam bbox negara aktif. Record negara lain, atau tanpa keduanya, **dilewati** (bukan ditolak) dan dihitung sebagai `skipped_country`. Record yang lolos diberi `country_code`.
4. Normalisasi (fungsi murni, unit test per aturan, FR-DATA-02):

| Input | Aturan | Hasil |
|---|---|---|
| `telephone` | libphonenumber dengan region = `country_code` record → E.164. `phone_kind` dari tipe nomor libphonenumber (untuk ID: `+628…` = seluler, BR-NORM-01). Satu fungsi untuk semua negara | `phone_e164`, `phone_kind`; nilai mentah tetap |
| `url`, `sameAs` | Parse host, cocokkan pola di `data/url-patterns.yaml` (BR-NORM-02) | `url_kind`, `social_links` |
| `@type`, `servesCuisine` | `@type` berupa array → pakai tipe yang ada di `category_map` dengan `type_depth` **terbesar** (paling spesifik; `["LocalBusiness","Restaurant"]` → `Restaurant`). Lalu cuisine spesifik dulu, baru `'*'` (BR-NORM-03). Tidak ada tipe yang dikenal → kategori **Lainnya** (`lainnya`) + peringatan `unmapped_type` | `category_id` (tidak pernah NULL) |
| `aggregateRating` | rating 0–5, ulasan ≥ 0; rating tanpa ulasan → rating NULL | `rating`, `review_count` |
| `geo` | Lat/lng valid dan di dalam bbox negara aktif (`data/countries.yaml`) | `geom` (NULL bila tidak valid) |
| `openingHoursSpecification` | Disimpan ternormalisasi (hari, buka, tutup) | `opening_hours` |
| Tanggal record (mis. `dateModified`) | Dipakai bila ada; bila tidak → `--observed-at` (BR-TRUST-02) | `observed_at` |

5. Setiap baris valid di-`COPY` (beserta nomor baris) ke tabel staging. Setelah file habis, merge per potongan (`hashtext(google_place_id) % 16`). **Baris terakhir menang**: bila `google_place_id` muncul lebih dari sekali dalam satu file, hanya baris dengan nomor terbesar yang dipakai. Kemunculan sebelumnya dihitung sebagai `duplicate_in_file` di laporan.

```sql
INSERT INTO places AS p (...)
SELECT DISTINCT ON (google_place_id) ... FROM import_stage WHERE chunk = $1
ORDER BY google_place_id, line_no DESC                              -- baris terakhir menang
ON CONFLICT (google_place_id) DO UPDATE SET ... , desa_id = NULL    -- wilayah dihitung ulang bila berubah
WHERE p.raw_hash <> EXCLUDED.raw_hash OR p.norm_version <> EXCLUDED.norm_version
RETURNING (xmax = 0) AS inserted;   -- true = baru, false = berubah; tidak kembali = tidak berubah
```

6. Laporan ke stdout + `imports`: baru, berubah, tidak berubah, ditolak per alasan, `skipped_country` (FR-DATA-01). Lalu enqueue `geo.assign`.

**Semantik re-import.** File yang sama → semua "tidak berubah", tanpa duplikat. Bila normalizer berubah (`norm_version` dinaikkan), import ulang file yang sama menormalisasi ulang semua record. Record yang hilang dari file baru **tidak dihapus** di F1; `import_id` lama tetap terlihat. Deteksi usaha tutup/berubah masuk signals F2. `--country` dan `--observed-at` (tanggal pengambilan dataset) wajib diisi.

**Pembaruan berkala (PRD §7.1).** Dataset baru = jalankan `msp tool import` lagi dengan file terbaru dan `--observed-at` baru; tidak ada mekanisme lain. Upsert per `google_place_id` menghasilkan hitungan baru/berubah/tidak berubah. Record yang tidak berubah tetapi muncul lagi di file yang lebih baru hanya dimajukan `observed_at` dan `import_id`-nya (satu `UPDATE` per potongan setelah merge, bila `observed_at` baru lebih besar), sehingga freshness (BR-TRUST-02) tetap benar tanpa me-reset wilayah. Mengaktifkan negara lain kelak = `geo-sync` + `import` dengan `--country` negara itu, tanpa perubahan skema.

### 5.2 `msp tool geo-sync`

1. Metadata semua level, **hanya negara `--country`** (FR-DATA-03): `GET /v1/areas?country=<kode>&admin_level=L&limit=500&offset=k` → upsert `areas` dengan `country_code`. Untuk ID: ±180 request untuk 34/522/7.069/81.912 area.
2. Polygon **hanya level 4** (desa/kelurahan): `GET /v1/areas/{id}?format=geojson&simplify=<toleransi kecil>` → `ST_Multi(ST_GeomFromGeoJSON(...))`. Level 1–3 cukup lewat `parent_id`. Peta memakai vector tiles GeoAPI.
3. Sopan terhadap batas 120 req/menit/IP: token bucket 1,5 req/s, ikuti `429` + `Retry-After`, backoff untuk 5xx. **Resumable**: area yang sudah punya `geom` dilewati (kecuali `--refresh`).
4. Setelah selesai, jumlah per level dicek terhadap jumlah yang diharapkan (ID: angka di atas), lalu `geo.assign` penuh di-enqueue.

Sync polygon pertama butuh ±82 ribu request (±15 jam), jadi dijalankan sekali di background. Per 2026-10-10, mode GeoJSON untuk daftar area di GeoAPI dibatasi 10.000 fitur dan tidak memfilter `parent_id`, sehingga polygon harus diambil per area.

### 5.3 Tahap job

| Job (queue) | Isi |
|---|---|
| `geo.assign` (`import`) | `ST_Contains` untuk place yang `desa_id IS NULL`. Titik di luar semua polygon atau tanpa koordinat tetap tanpa wilayah: tampil "Tidak diketahui" dan tidak ikut metrik wilayah. Target ≥ 98% (FR-DATA-04), dipantau metrik |
| `geo.metrics` (`import`) | `area_metrics` level 3–4 sesuai Lampiran C (bagian yang tidak bergantung ICP). Area tanpa bisnis ikut sebagai pembanding (n = 0) |
| `intel.derive` (`import`) | `review_pctl` (`percent_rank()`, BR-SCORE-03), `peer_stats`, lalu `place_gaps`: stream place + peer → `intel.DetectGaps()` (Lampiran B, BR-OPP-01) → isi tabel diganti dalam satu transaksi. Hasil ditulis ke `imports.health` |
| `score.all` (`score`) | Enqueue `score.icp` untuk setiap ICP (`icps_to_rescore()`) |

```sql
-- geo.assign (inti)
UPDATE places p SET desa_id = d.id, kec_id = k.id, kab_id = k.parent_id,
       prov_id = (SELECT parent_id FROM areas WHERE id = k.parent_id)
FROM areas d JOIN areas k ON k.id = d.parent_id
WHERE d.admin_level = 4 AND d.country_code = p.country_code AND p.desa_id IS NULL
  AND p.geom && d.geom AND ST_Contains(d.geom, p.geom);
-- geo.metrics (kelurahan; kecamatan sama dengan kec_id). counts = LEFT JOIN areas ← places, GROUPING SETS (desa × kategori, desa)
INSERT INTO area_metrics (area_id, category_id, business_count, density_per_1k, competition_pct, limited, model_version)
SELECT area_id, cat, n, n * 1000.0 / nullif(pop, 0),
       round(100 * percent_rank() OVER (PARTITION BY kab_id, cat ORDER BY n * 1.0 / nullif(pop, 0))),
       (pop < 500 OR n < 5), 'geo-2.0.0'
FROM counts;
```

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
- ICP Fit = rata-rata kriteria yang diketahui × 100. Kriteria *wajib* bernilai 0 → prospek bukan kandidat (tidak tampil). Kriteria *wajib* yang tidak diketahui tetap kandidat, tetapi dibatasi WARM + "Data terbatas" (BR-SCORE-02).
- Kandidat difilter di SQL dengan predikat statis berparameter (mis. `(rating IS NULL OR rating >= @rating_floor)`; batas "sebagian" dihitung di Go dari Lampiran A), tanpa SQL dinamis.

### 6.2 Perhitungan dan penyimpanan

`intel.Score(facts, icp, cfg) Result` adalah **fungsi murni**: faktor, renormalisasi, coverage, band, dan cap persis sesuai Lampiran A dan BR-SCORE-01/02. Input + versi yang sama selalu memberi hasil yang sama; tie diurutkan dengan `place_id`. `factors` menyimpan breakdown untuk "Kenapa prospek ini?": `{"icp_fit":{"v":100,"w":30},"opportunity":{"v":70,"w":25,"code":"OPP-WEB-SOCIAL"},"area":null,…}` (`null` = tidak diketahui, bobotnya dibagi ulang).

Job `score.icp {workspace_id, icp_id}`:

1. `pg_try_advisory_lock(icp)`. Bila ICP sedang dihitung → `JobSnooze(30 s)`, sehingga hanya satu job per ICP yang berjalan.
2. Satu SQL agregat kandidat per kelurahan/kecamatan → `icp_area_scores.market_score` (Lampiran C, bagian ICP; pembanding = area selevel dalam kab/kota yang sama).
3. Stream kandidat (+ gap, area, peer) **diurutkan dengan proksi murah** (severity gap relevan tertinggi, lalu jumlah ulasan) → `intel.Score` → upsert batch 2.000 dengan **commit per batch**, lalu update `icps.scored`.
4. Hapus skor lama (`computed_at` < mulai job), isi `hot_warm_count`, lalu set `score_status='ready'`, `model_version`, dan `scored_at`.

**Onboarding ≤ 5 menit (FR-ICP-01).** Batch pertama berisi kandidat terbaik menurut proksi dan langsung di-commit, jadi halaman pertama tampil dalam hitungan detik setelah ICP disimpan. UI menampilkan progres `scored/candidates` dan me-refresh sampai statusnya `ready`.

**Satu jalur hitung ulang.** Rantai tunggal: `geo.metrics` (metrik wilayah) → `intel.derive` (gap bergantung pada metrik wilayah) → `score.all` → `score.icp` × N (§5). Tidak ada periodic job rescore.

| Pemicu | Aksi |
|---|---|
| `msp tool import` atau `msp tool geo-sync` selesai | `geo.assign` → rantai di atas |
| ICP disimpan (dibuat/diubah, termasuk Offering-nya diubah) | API meng-enqueue `score.icp` untuk ICP itu saja (transaksi yang sama) |
| `msp tool rescore` (manual, ops; setelah versi model/bobot `priority-`/`geo-` berubah) | Enqueue `geo.metrics` → rantai di atas (metrik wilayah ikut dihitung ulang) |

### 6.3 Konfigurasi bobot

Satu file berversi di repo, di-embed ke binary. Setiap perubahan bobot atau aturan berarti versi dinaikkan (semver).

```yaml
# data/scoring/priority-2.0.0.yaml
version: priority-2.0.0
weights: { icp_fit: 30, opportunity: 25, area: 15, contact: 15, popularity: 10, completeness: 5 }  # Lampiran A
overrides: {}   # per tag primer Offering, mis. reputation_mgmt: { opportunity: 30, popularity: 5 }
bands: { hot: 80, warm: 60 }
```

## 7. Search

```sql
CREATE TEXT SEARCH CONFIGURATION msp (COPY = simple);   -- butuh extension unaccent dan pg_trgm
ALTER TEXT SEARCH CONFIGURATION msp ALTER MAPPING FOR hword, hword_part, word WITH unaccent, simple;
CREATE INDEX ON places USING gin (search);   CREATE INDEX ON places USING gin (name gin_trgm_ops);
CREATE INDEX ON places USING gist (geom);    CREATE INDEX ON scores (icp_id, priority DESC, place_id DESC);
```

Konfigurasi `simple` + `unaccent` dipakai tanpa stemming, karena cocok untuk nama usaha Indonesia. Karena regconfig ditulis eksplisit, `to_tsvector('msp', …)` immutable dan bisa dipakai di kolom generated. Trigram menangani salah ketik dan potongan nama.

```sql
-- name: SearchProspects  (ICP aktif; RLS membatasi workspace)
SELECT p.id, p.name, p.category_label, p.address_text, s.priority, s.band, s.top_opp, s.limited
FROM scores s JOIN places p ON p.id = s.place_id
WHERE s.icp_id = @icp_id
  AND (@q::text = '' OR p.search @@ websearch_to_tsquery('msp', @q) OR p.name % @q)
  AND (sqlc.narg(cats)::text[] IS NULL OR p.category_id = ANY(sqlc.narg(cats)))
  AND (sqlc.narg(area)::bigint IS NULL OR sqlc.narg(area) IN (p.desa_id, p.kec_id, p.kab_id, p.prov_id))
  AND (sqlc.narg(bands)::text[] IS NULL OR s.band = ANY(sqlc.narg(bands)))
  AND (sqlc.narg(opps)::text[] IS NULL OR EXISTS (SELECT 1 FROM place_gaps g
         WHERE g.place_id = p.id AND g.code = ANY(sqlc.narg(opps))))
  AND (NOT @has_phone OR p.phone_e164 IS NOT NULL)          -- juga has_website, has_social, min_rating
  AND (s.priority, s.place_id) < (@cur_priority, @cur_id)   -- keyset
ORDER BY s.priority DESC, s.place_id DESC
LIMIT @lim;   -- lim = limit + 1 untuk mendeteksi halaman berikutnya
```

- **Cakupan:** selalu ICP aktif (driver = index `scores`), jadi hanya kandidat ICP yang tampil. Setiap baris hasil dibentuk lewat `intel.ProspectView`, sama dengan profil.
- **Pagination:** keyset; `cursor` = base64 dari kunci urut terakhir; `limit` maks. 100.
- **Jumlah:** `SELECT count(*) FROM (<filter yang sama> LIMIT 10001)` → tampil "10.000+" bila melewati batas. Hasil selalu konsisten dengan filter (FR-SRCH-01).
- **Profil:** satu query (place + gap + wilayah + metrik + skor + unlock + suppression) → `intel.ProspectView`. Target p95 ≤ 500 ms (NFR-01).
- **Peta (FR-GEO-01):** browser memuat TileJSON GeoAPI (`/v1/tiles/areas.json`, layer `areas`, `area_labels`) langsung dengan MapLibre. Warna diambil dari `GET /areas/ranking` lewat `feature-state` (kunci = ID area). Atribusi TileJSON tampil di peta, dan tabel peringkat selalu tersedia sebagai padanan.

## 8. API

| Aspek | Aturan |
|---|---|
| Kontrak | `contracts/openapi.yaml` adalah sumber tunggal. Go: oapi-codegen (`std-http-server` + `strict-server`) → `internal/api`. TS: `openapi-typescript` + `openapi-fetch` → `packages/api-client`. Keduanya dibuat oleh `make generate`; CI gagal bila hasil generate berbeda |
| Basis | `/api/v1`, JSON, same-origin lewat Caddy. Workspace dipilih dengan header `MSP-Workspace: <uuid>`, divalidasi terhadap membership. Health di luar prefix: `/healthz`, `/readyz`; `/metrics` hanya di listener internal (§14) |
| Role | Ditulis sekali per operasi di OpenAPI: `x-msp-role: member\|admin\|ops`. Middleware membacanya lewat operationId; test matriks otorisasi memakai sumber yang sama |
| Error | `application/problem+json` (RFC 9457) + `code` stabil + `request_id`. Kode: `UNAUTHENTICATED` 401 · `INSUFFICIENT_CREDITS` 402 (extension `required`, `available`) · `FORBIDDEN`, `EMAIL_NOT_VERIFIED`, `MFA_REQUIRED`, `SEAT_LIMIT` (kuota seat plan habis saat mengundang, §12) 403 · `NOT_FOUND` 404 (termasuk milik workspace lain) · `CONFLICT` 409 (mis. `Idempotency-Key` dipakai ulang dengan body berbeda) · `VALIDATION_FAILED` 422 (extension `errors[]`), `WEBHOOK_URL_REJECTED` 422 · `RATE_LIMITED` 429 · `INTERNAL` 500 (tanpa detail internal) · `FEATURE_DISABLED`, `AI_UNAVAILABLE` 503 |
| Idempotensi | Header `Idempotency-Key` **wajib** di POST yang bisa mengubah saldo credit (`/unlocks`, `/exports`, `/webhooks/{id}/send`, `/prospects/{id}/brief`, grant ops). Disimpan sebagai kunci unik di `credit_ledger` / `exports` / `webhook_deliveries`. Request ulang mengembalikan hasil yang sama tanpa tagihan baru (§12) |
| Async | `202 Accepted` + `Location` ke resource. Resource membawa `{status: queued\|running\|done\|failed, progress: {done, total}, error}` (export, kirim webhook, brief, scoring ICP) |
| Batas | Body ≤ 1 MB, `limit` ≤ 100, **export maks. 10.000 baris per job** (FR-ACT-03; lebih → 422), kirim webhook ≤ 100 prospek. Rate limit in-process per user untuk brief dan export (`x/time/rate`; cukup karena hanya satu instance api) |

Role: **M** = Member (Admin juga boleh), **A** = Admin, **O** = ops (`is_ops` + AAL2). **$** = bisa mengubah saldo credit (export/kirim hanya lewat `unlock_missing`) + `Idempotency-Key` wajib.

| Method | Path | FR | Role |
|---|---|---|---|
| GET · PUT | `/me` (user, membership, role, ICP aktif, saldo), `/me/active-icp` | AUTH-01/02, ICP-02 | login · M |
| POST | `/workspaces` (onboarding) | AUTH-02 | login, email terverifikasi |
| GET · POST · PATCH · DELETE | `/members`, `/members/{id}` (undang = POST, cek seat → `SEAT_LIMIT`); `/invites/accept` | AUTH-02, BILL-01 | M baca · A tulis |
| GET · POST · PUT | `/offerings`, `/offerings/{id}` | ICP-01 | M baca · A tulis |
| GET · POST · PUT · DELETE | `/icps`, `/icps/{id}` (status scoring ikut di body) | ICP-01/02 | M baca · A tulis |
| GET | `/categories`, `/areas?q=&level=&parent_id=` | ICP-01, SRCH-01 | M |
| GET | `/prospects?q=&filters…&cursor=` | SRCH-01 | M |
| GET | `/prospects/{id}`, `/prospects/{id}/evidence` | PROF-01, INTL-01 | M |
| PUT | `/prospects/{id}/feedback` (per kode opportunity) | INTL-05 | M |
| GET · POST · DELETE | `/saved-searches`, `/saved-searches/{id}` | SRCH-02 | M |
| GET | `/areas/ranking?parent_id=&level=` | GEO-01 | M |
| GET · POST · DELETE | `/lists`, `/lists/{id}`, `/lists/{id}/items` | ACT-01 | M |
| PUT · POST | `/prospects/{id}/tags`, `/prospects/{id}/owner`, `/prospects/{id}/notes` | ACT-01 | M |
| POST | `/unlocks` (`place_ids` ≤ 100) | ACT-02 | M $ |
| POST | `/unlocks/preview` (sebelum "Unlock & export/kirim": jumlah, sudah di-unlock, ter-suppress, biaya, saldo) | ACT-02/03/04 | M |
| POST · GET | `/exports` → 202, `/exports/{id}` (memuat `download_url`) | ACT-03 | M $ |
| GET | `/exports/{id}/file?exp=&sig=` | ACT-03 | tanda tangan (§13) |
| GET · POST · PATCH · DELETE | `/webhooks`, `/webhooks/{id}`, `/webhooks/{id}/test`, `/webhooks/{id}/deliveries` | ACT-04 | A |
| POST | `/webhooks/{id}/send` (`place_ids` ≤ 100) → 202 | ACT-04 | M $ |
| POST · GET | `/prospects/{id}/brief` → 202, `/briefs/{id}` | ACT-05 | M $ (`ai_credit`) |
| GET | `/credits` (saldo per meter, plan & seat terpakai; riwayat ledger untuk Admin) | BILL-01 | M |
| GET · PATCH | `/ops/workspaces`, `/ops/workspaces/{id}` (ganti plan) | OPS-01, BILL-01 | O |
| POST | `/ops/workspaces/{id}/credits` (grant/refund manual) | OPS-01, BILL-01 | O $ |
| GET · POST · PUT | `/ops/plans`, `/ops/plans/{id}` (tanpa DELETE; nonaktifkan dengan `active: false`) | OPS-01, BILL-01 | O |
| GET · PUT | `/ops/settings`, `/ops/settings/{key}` (setting AI + status API key, §11) | OPS-01, ACT-05 | O |
| POST | `/ops/settings/ai/test` (Uji koneksi, §11) | OPS-01, ACT-05 | O |
| GET | `/ops/imports`, `/ops/data-health`, `/ops/audit` | OPS-01/02 | O |
| GET · PUT · POST · DELETE | `/ops/flags`, `/ops/flags/{key}`, `/ops/suppressions`, `/ops/suppressions/{id}` | OPS-01, PRIV-02 | O |

## 9. Auth (Kratos)

- **Same domain.** Caddy me-route `/.ory/*` ke Kratos public API. Cookie sesi Kratos (`HttpOnly`, `Secure`, `SameSite=Lax`) otomatis terkirim ke `/api/v1`. Admin API Kratos hanya bisa diakses dari jaringan Docker internal.
- **Browser flows.** Next.js merender UI login, registrasi, verifikasi, recovery, dan settings dari flow Kratos (`/self-service/*/browser`, `@ory/client`). Metode: password, **Google OIDC**, TOTP. Kratos mengirim email lewat courier SMTP-nya sendiri.
- **Middleware api.** Cookie → `GET {MSP_KRATOS_PUBLIC_URL}/sessions/whoami` (cache in-process 30 s per hash cookie) → `Principal{identity_id, email, verified, aal}`. Baris `users` dibuat JIT. Email belum terverifikasi → `403 EMAIL_NOT_VERIFIED` (FR-AUTH-01).
- **Membership & role** dibaca dari DB kita pada setiap request (tanpa cache), jadi anggota yang dikeluarkan langsung kehilangan akses. Undangan memakai token acak (hanya hash-nya yang disimpan di `members`), berlaku 7 hari, dan diterima lewat `accept_invite()`.
- **Ops (FR-AUTH-03).** `/api/v1/ops/*` mensyaratkan `users.is_ops` **dan** sesi `aal2` (TOTP). Tanpa itu → `403 MFA_REQUIRED`. Kratos `session.whoami.required_aal: highest_available` memaksa akun yang punya TOTP untuk memakainya.
- **CSRF.** Flow Kratos punya token CSRF sendiri. Untuk API: `SameSite=Lax`, request yang mengubah data wajib `Content-Type: application/json`, dan `Origin` harus sama dengan `MSP_PUBLIC_URL`. Kratos memakai database `kratos` di instance Postgres yang sama, dengan backup yang sama.

## 10. Webhook

- **Event F1:** `prospects.sent` (anggota mengirim prospek terpilih) dan `webhook.test` (data contoh, tanpa biaya). Payload memakai `intel.ProspectView` yang sama dengan export (DRY): website dan sosial selalu ada, telepon terisi hanya untuk prospek yang sudah di-unlock, dan nilai ter-suppress tidak pernah dikirim.
- **Biaya (BR-CRED-01):** sama dengan export. Prospek yang belum di-unlock di-unlock dulu lewat fungsi unlock (§12, `unlock_missing: true`); pengiriman dan retry tidak menyentuh credit. Prospek ter-suppress tidak di-unlock dan tidak ditagih (FR-PRIV-02).
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

## 11. AI brief

```go
type LLM interface{ Complete(ctx context.Context, req Request) (Response, error) } // keluaran selalu JSON
type Request struct{ System, User string; Schema json.RawMessage; MaxTokens int }
type Response struct{ JSON []byte; Provider, Model string; InTokens, OutTokens int }
type Fallback struct{ Primary, Secondary LLM; Timeout time.Duration } // dirakit per panggilan dari setting AI
```

- **Setting AI (D-29).** Provider dan model dibaca dari tabel `settings` setiap kali brief dibuat (cache in-process 10 s, sama dengan flag), jadi perubahan dari halaman ops berlaku tanpa deploy:

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
- **Uji koneksi:** `POST /ops/settings/ai/test` menerima nilai form (belum disimpan), mengirim satu prompt uji tetap tanpa data prospek ke provider utama dan fallback, lalu mengembalikan per provider `ok`, latensi, dan pesan error. Tidak memakai AI credit dan tidak menyimpan setting.
- **Claude (`anthropic`):** SDK resmi `github.com/anthropics/anthropic-sdk-go`, model `ai.model`, structured output (`output_config.format` JSON Schema), effort `low` demi latensi. Error, timeout, `stop_reason` selain `end_turn` (mis. `refusal`, `max_tokens`), atau JSON tidak valid → fallback (bila aktif).
- **OpenAI-compatible:** `net/http` biasa ke `{ai.fallback.base_url}/chat/completions` dengan `MSP_LLM_FALLBACK_API_KEY`, `response_format: json_object`. Dipakai sebagai fallback (model `ai.fallback.model`), atau sebagai provider utama bila `ai.provider = openai_compatible` (model `ai.model`).
- **Prompt** `brief-1.0.0`: `internal/action/ai/prompts/brief-1.0.0/{system.md,user.tmpl,schema.json}` (di-embed). Versi dinaikkan setiap kali isinya berubah.
- **Input = daftar fakta ber-ID** dari `ProspectView`, tanpa nomor telepon (BR-AI-01; unlock tidak mengubah aturan ini): `{"offering":["website_dev"],"facts":[{"id":"f1","text":"Kategori: Restoran Asia","claim":"derived"},{"id":"f2","text":"Website mengarah ke linktr.ee","claim":"observed"}]}`.
- **Output:** `{"sentences":[{"text":"…","fact_ids":["f2"]}]}`. Validator deterministik: lolos schema, 3–8 kalimat, setiap kalimat punya ≥ 1 `fact_id` yang ada di input, dan tidak ada pola telepon/email. Gagal → **regenerate sekali** dengan daftar pelanggaran. Gagal lagi → `failed`, **tidak ditagih** (FR-ACT-05, BR-CRED-01).
- **Cache:** `input_hash = sha256(prompt_version ‖ fakta kanonik ‖ tag offering)`. Brief `done` dengan hash sama, di workspace yang sama, dan berumur < 7 hari dikembalikan langsung tanpa biaya. Cache tidak pernah dibagi lintas workspace (RLS).
- **Alur:** POST mengecek flag `ai` (off → `503 FEATURE_DISABLED`) dan saldo meter `ai_credit` (kurang → 402) → job `ai.brief` (queue `ai`) → brief disimpan dan 1 `ai_credit` didebit **dalam satu transaksi**, hanya saat sukses. Timeout 15 s per provider, batas job 45 s. Target p95 ≤ 20 s (NFR-02) dipantau metrik.

## 12. Credit & plan

- **Satu ledger, dua meter** (`credit`, `ai_credit`), append-only: grant `+n`, debit `−n`, refund `+n`. Saldo = `SUM(delta)` per meter (index `workspace_id, meter`). Tidak ada kolom saldo yang bisa berbeda dari ledger (FR-BILL-01).
- **Idempotensi:** `UNIQUE (workspace_id, idem_key)`. Request ulang menabrak kunci unik, lalu handler mengembalikan hasil yang sudah tersimpan. PK `unlocks` (`workspace_id`, `place_id`) menjamin satu prospek tidak pernah ditagih dua kali, walau `idem_key`-nya berbeda.
- **Konkurensi:** setiap debit diawali `SELECT … FROM workspaces WHERE id = $ws FOR UPDATE`, jadi debit dalam satu workspace berjalan serial dan saldo tidak bisa minus.
- **Suppression (FR-PRIV-02):** prospek dengan kontak ter-suppress tidak pernah masuk `unlocks` dan tidak pernah dihitung biaya, baik lewat unlock, export, maupun webhook.

```sql
-- POST /unlocks, dalam InWorkspace (satu transaksi)
SELECT 1 FROM workspaces WHERE id = $ws FOR UPDATE;
-- n = place_ids yang belum di-unlock DAN tidak ter-suppress (yang ter-suppress dikembalikan "suppressed"); n = 0 → selesai
-- SUM(delta) WHERE meter = 'credit' < n → 402 INSUFFICIENT_CREDITS
INSERT INTO credit_ledger (id, workspace_id, meter, kind, delta, idem_key, ref, actor_id)
VALUES ($id, $ws, 'credit', 'debit', -$n, $idem_key, 'unlock', $user);
INSERT INTO unlocks (workspace_id, place_id, via, ledger_id, user_id)
SELECT $ws, unnest($chargeable::uuid[]), 'unlock', $id, $user ON CONFLICT DO NOTHING;
INSERT INTO audit_log (...) VALUES (...);
```

**Export (FR-ACT-03).** (1) `POST /unlocks/preview` menghitung jumlah prospek, yang sudah di-unlock, yang akan di-unlock (belum di-unlock dan tidak ter-suppress), biaya, dan saldo. (2) `POST /exports` (seleksi > 10.000 baris → 422) dengan `unlock_missing: true` memanggil **fungsi unlock yang sama** dengan `POST /unlocks` (transaksi pendek di atas, ditagih, `idem_key` dari request), lalu menyimpan `exports` + job `export.run` → 202. Tanpa `unlock_missing`, prospek yang belum di-unlock diekspor dengan telepon ter-mask. (3) Worker menulis file ke storage → `status='done'`. Gagal → `failed`, bisa diulang **tanpa biaya**: export sendiri tidak pernah menyentuh ledger.

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
| `MSP_KRATOS_PUBLIC_URL`, `MSP_GEOAPI_BASE_URL` | `http://kratos:4433`; `https://geoapi.mcpke.dev/api` |
| `MSP_ANTHROPIC_API_KEY`, `MSP_LLM_FALLBACK_API_KEY` | API key Claude dan OpenAI-compatible. Provider, model, dan base URL ada di setting ops (§11) |
| `MSP_STORAGE_DIR` | `/var/lib/msp/storage` |
| `MSP_ENCRYPTION_KEY` | 32 byte base64: AES-GCM untuk secret webhook + kunci HKDF untuk URL unduh |
| `MSP_SMTP_URL` | Email undangan anggota (Kratos punya konfigurasi courier sendiri) |

- `platform/config` membaca env ke satu struct dan memvalidasinya saat start (fail fast). Env hanya untuk secret dan pengaturan proses (alamat, URL, level log). Data yang di-embed (`data/`: negara, kategori, pola URL, bobot skor, prompt) berubah lewat MR. Nilai yang diatur ops saat runtime ada di DB: `flags` dan `settings` dibaca dengan cache in-process 10 s, `plans` dibaca langsung. Setiap perubahannya tercatat di audit.
- **Log:** `slog` JSON ke stdout. Field: `time`, `level`, `msg`, `request_id`, `user_id`, `workspace_id`, `route`, `status`, `duration_ms`, `job_id`, `queue`. `ReplaceAttr` me-mask pola nomor telepon (`+62…`/`08…` → `+62******1234`). Body request dan respons tidak pernah di-log.

| Metrik | Untuk |
|---|---|
| `msp_http_request_duration_seconds{route,method,status}` | NFR-01, error rate |
| `msp_jobs_total{queue,kind,result}`, `msp_job_duration_seconds{kind}` | Kesehatan worker |
| `msp_queue_depth{queue,state}` | Antrean menumpuk / job `discarded` |
| `msp_import_records_total{result}` | Laporan import |
| `msp_places_area_assigned_ratio` | FR-DATA-04 (≥ 0,98) |
| `msp_credits_total{meter,kind}` | Pemakaian credit |
| `msp_webhook_deliveries_total{result}` | Pengiriman webhook |
| `msp_llm_requests_total{provider,model,result}`, `msp_llm_duration_seconds` | NFR-02, frekuensi fallback |
| `msp_geoapi_requests_total{status}` | Rate limit GeoAPI |

Selain itu ada metrik runtime Go dan statistik pgxpool. `/healthz` = proses hidup (tanpa dependensi). `/readyz` = DB bisa di-ping dan versi migrasi sesuai. `/metrics` hanya di `MSP_METRICS_ADDR` dan tidak di-route Caddy. Prometheus (`/opt/ops`, server yang sama) mengambilnya lewat network Docker internal (detail di 06).

**Uptime.** Monitoring berada di server yang sama, jadi ikut mati bila server mati. Karena itu ketersediaan (NFR-03) dicek **layanan uptime monitor eksternal gratis** yang memanggil `https://app.<domain>/readyz` tiap ≤ 5 menit dan mengirim alert ke email/Telegram. Alert dan dashboard lain ada di 06/09.

## 15. Keamanan F1

1. **Permukaan jaringan:** server hanya membuka 80/443 (satu Caddy untuk semua domain) dan SSH (key-only). Postgres, admin API Kratos, dan metrik `:9090` hanya ada di jaringan Docker (§14). Production dan staging tidak berbagi database maupun secret.
2. **Isolasi workspace:** RLS `FORCE` + role `msp_app` `NOBYPASSRLS` + `InWorkspace` sebagai satu-satunya jalan. Resource milik workspace lain → 404. Test lintas workspace mencakup setiap endpoint (05).
3. **Identitas:** Kratos, email wajib terverifikasi, ops wajib AAL2 (TOTP). Role diperiksa per request dari `x-msp-role`.
4. **Secret:** variabel CI terproteksi → `.env` per environment (izin 600). Tidak ada secret di repo, log, atau tabel `settings`. Secret webhook dienkripsi AES-GCM.
5. **Outbound:** guard SSRF di dial time untuk webhook, tanpa redirect. Host GeoAPI berasal dari env, host LLM dari setting ops (wajib `https`, hanya akun ops AAL2, diaudit). Keduanya tidak pernah dari input user workspace.
6. **Data kontak:** telepon di-mask sampai di-unlock, dan suppression diterapkan di satu fungsi `intel.ProspectView` (search, profil, evidence, export, webhook). Telepon tidak dikirim ke LLM dan di-redact dari log.
7. **Audit:** `audit_log` append-only (`REVOKE UPDATE, DELETE`) untuk login, undangan, unlock, export, webhook, grant, flag, suppression, plan, dan setting.
8. **Dependensi & input:** `govulncheck` + `npm audit` di CI (06). Batas ukuran body/limit/export, validasi OpenAPI di setiap request, dan query hanya lewat sqlc (berparameter).

## 16. Evolusi F2/F3 (hanya bila dibutuhkan)

Daftar fitur F2/F3 ada di PRD §6 dan tidak disalin di sini. Tabel ini hanya mencatat kapan komponen atau perubahan teknis ditambahkan. Fitur F2 memakai pola yang sudah ada bila bisa (pencocokan customer = pg_trgm + jarak geo, integrasi CRM = pola webhook §10, NL search = LLM → filter kanonik §7).

| Fase | Komponen / perubahan teknis | Pemicu |
|---|---|---|
| F2 | Riwayat raw per import (`place_versions`) → diff → signals | Founder butuh deteksi usaha tutup/berubah dari import berkala |
| F2 | Proxy + cache tiles GeoAPI di Caddy | Banyak user di balik satu IP mencapai 120 req/menit |
| F2 | Error tracking | Volume error mulai sulit dibaca dari log |
| F3 | Pipeline incremental (hanya place berubah), batch scoring lebih besar | Dataset > 5 juta atau import > 1 jam |
| F3 | Cache (Redis) | p95 search melewati NFR-01 setelah tuning query/index |
| F3 | Adapter `storage` S3 (presigned URL menggantikan URL bertanda tangan) | Server aplikasi kedua |
| F3 | GitLab + runner dipindah ke server terpisah | Contention terukur: p95 production naik saat CI atau import berjalan (PRD §13) |
| F3 | Server aplikasi kedua, replika Postgres untuk baca, PgBouncer, deploy tanpa downtime | Target 99,5% (NFR-03) |
| F3 | Partisi `scores`, mesin search terpisah | Hanya bila Postgres terbukti tidak cukup |
