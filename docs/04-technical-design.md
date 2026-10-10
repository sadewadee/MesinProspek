# 04 — Technical Design (lean)

## MesinProspek — Sales Intelligence Engine

| Field | Value |
|---|---|
| Versi | 0.2 (lean). Menggantikan desain v0.1 yang terlalu berat untuk F1 |
| Tanggal | 2026-10-10 |
| Sumber kebenaran | Requirement: [PRD](01-prd.md) · Keputusan: [00 §7](00-analisis-dokumen-awal.md#7-register-keputusan-status-v02) · Nama teknis: [README](README.md#kosakata-teknis-bersama) |
| Cakupan | **F1 dirinci.** F2/F3 hanya daftar evolusi (§16) |

> Dokumen ini **tidak menyalin** rumus PRD. Rumus skor, katalog opportunity, dan metrik wilayah tetap di PRD Lampiran A–C; di sini hanya cara menghitung dan menyimpannya.

Isi: 1 Ringkasan · 2 Arsitektur · 3 Modul Go · 4 Data model · 5 Pipeline · 6 Skor & ICP · 7 Search · 8 API · 9 Auth · 10 Webhook · 11 AI · 12 Credit · 13 Storage · 14 Konfigurasi & observability · 15 Keamanan · 16 Evolusi

---

## 1. Ringkasan & keputusan arsitektur

- Satu binary Go `msp` + satu PostgreSQL. Semua (data, antrean, search, geo) ada di Postgres. Tidak ada Redis, PgBouncer, object storage, atau search engine terpisah (D-26).
- Data bisnis **global** (places, areas) dibaca semua workspace. Data **workspace** (ICP, skor, list, unlock, ledger) diisolasi dengan RLS (D-03).
- Alur data F1: `msp tool geo-sync` (sekali) → `msp tool import` → job River: wilayah → metrik → gap → skor per ICP.
- Logika inti (normalisasi, gap, skor) berupa **fungsi Go murni** yang mudah di-unit-test. SQL dipakai untuk agregat dan join.

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

---

## 2. Arsitektur F1

```
                     Browser (HTTPS)
                           │
┌─ server prod ────────────▼───────────────────────────────────────────────┐
│ caddy  (TLS otomatis, satu domain app.<domain>)                           │
│   ├─ /             → web     Next.js (SSR memanggil api di jaringan internal)
│   ├─ /api/v1/*     → api     msp api ── /sessions/whoami ──► kratos       │
│   └─ /.ory/*       → kratos  Ory Kratos (public API saja; admin API tidak di-route)
│                                                                           │
│ worker   msp worker  (River: import, score, export, webhook, ai)          │
│                                                                           │
│ postgres 16 + PostGIS + pg_trgm + unaccent                                │
│   ├─ db msp     data global · data workspace (RLS) · tabel River          │
│   └─ db kratos  identitas                                                 │
│ volume /var/lib/msp/storage  (file export; dipakai api + worker)          │
└──────┬─────────────────────────────┬──────────────────────────────────────┘
       │ pgBackRest (WAL kontinu)    │ scrape /metrics (IP allowlist)
       ▼                             ▼
 S3-compatible off-site       ┌─ server ops ─────────────────────────────────┐
 (Indonesia)                  │ gitlab · gitlab-runner · prometheus ·        │
                              │ alertmanager · grafana · staging stack       │
                              │ (caddy, web, api, worker, kratos, postgres)  │
                              └──────────────────────────────────────────────┘
Eksternal: worker → GeoAPI (geo-sync) · worker → Claude, fallback OpenAI-compatible
           worker → URL webhook customer (§10) · kratos → Google OIDC, SMTP · api → SMTP (undangan)
```

| Proses | Perintah | Catatan |
|---|---|---|
| `api` | `msp api` | HTTP `:8080`, metrik `:9090` (listener internal) |
| `worker` | `msp worker` | Semua queue River. Concurrency: `import` 1, `score` 2, `export` 2, `webhook` 10, `ai` 4 |
| migrasi | `msp migrate` | goose + migrasi River, dijalankan pipeline deploy sebelum `api`/`worker` diganti (06) |
| tool | `msp tool import\|geo-sync\|rescore` | `docker compose run --rm api msp tool …` di server prod |

---

## 3. Modul Go

```
apps/api/
├─ cmd/msp/          main + subcommand (api, worker, migrate, tool …)
├─ internal/
│  ├─ platform/      fondasi teknis, tanpa logika bisnis
│  ├─ ingest/  geo/  intel/  workspace/  action/  billing/  ops/
│  └─ api/           hasil generate oapi-codegen (jangan diedit)
└─ db/migrations/ (goose SQL) · db/queries/ (sqlc → internal/platform/db/q)
```

| Modul | Isi (satu baris) |
|---|---|
| `platform` | `config` (env), `db` (pgxpool, `InWorkspace`, sqlc), `httpx` (router, problem+json, middleware), `auth` (Kratos whoami → `Principal`), `storage`, `flags`, `audit`, `crypto` (AES-GCM, HMAC), `jobs` (River client + semua struct args job) |
| `ingest` | Baca JSONL(.gz), validasi, normalisasi (BR-NORM), upsert `places`, laporan import |
| `geo` | Client GeoAPI + `geo-sync`, penetapan wilayah (PostGIS), metrik wilayah (Lampiran C) |
| `intel` | Tabel claim, deteksi gap (Lampiran B), skor (Lampiran A), search, profil, `ProspectView` (satu bentuk untuk profil/export/webhook) |
| `workspace` | Workspace, anggota & undangan, offering, ICP, list, tag, pemilik, catatan, saved search, feedback |
| `action` | Unlock, export, webhook (+ guard SSRF), AI brief (interface `LLM`) |
| `billing` | Ledger credit, saldo, grant/refund |
| `ops` | Endpoint ops: workspace & grant, riwayat import, kesehatan data, flag, suppression, audit |

**Aturan dependensi.** `platform` tidak mengimpor modul domain. Urutan impor yang diizinkan: `geo` → `intel` → `workspace` → `billing` → `action` → `ops` (modul kanan boleh mengimpor modul kiri, tidak sebaliknya); `ingest` hanya mengimpor `intel`. Semua query lewat paket sqlc bersama `platform/db/q`. Struct args job River ada di `platform/jobs`, jadi enqueue tidak perlu mengimpor modul pelaksana. Siklus impor ditolak compiler Go.

---

## 4. Data model

**Konvensi.** Satu schema `public`. ID entity kita **UUIDv7** (`uuid`, dibuat di Go `uuid.NewV7()`, terurut waktu). Pengecualian: `places.google_place_id` (kunci alami, `UNIQUE`), `areas.id` = ID integer GeoAPI (D-13), `categories.id` = slug teks. Waktu `timestamptz` (UTC), ditampilkan Asia/Jakarta. Tabel workspace punya `workspace_id uuid NOT NULL` + RLS.

### 4.1 Tabel inti `places`

```sql
CREATE TABLE places (
  id              uuid PRIMARY KEY,
  google_place_id text NOT NULL UNIQUE,
  name text NOT NULL, description text,
  schema_type text, cuisines text[],                     -- @type, servesCuisine (mentah)
  category_id text REFERENCES categories, category_label text,
  phone_raw text, phone_e164 text, phone_kind text,      -- 'mobile' | 'fixed'
  url_raw text, url_kind text,  -- own_domain|social|aggregator|messaging|free_subdomain; NULL = kosong
  social_links text[], address jsonb, address_text text, maps_url text,
  geom geometry(Point, 4326), opening_hours jsonb,
  rating numeric(2,1), review_count int, review_pctl smallint,  -- pctl: percent_rank kategori × kab/kota
  desa_id bigint, kec_id bigint, kab_id bigint, prov_id bigint, area_method text, -- 'contains'|'nearest'
  raw jsonb NOT NULL, raw_hash bytea NOT NULL, norm_version text NOT NULL,       -- evidence
  import_id uuid NOT NULL, observed_at date NOT NULL,         -- BR-TRUST-02
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  search tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('msp', coalesce(name, '')), 'A') ||
    setweight(to_tsvector('msp', coalesce(category_label, '')), 'B') ||
    setweight(to_tsvector('msp', coalesce(address_text, '')), 'C')) STORED
);
```

`raw` disimpan di baris yang sama (Postgres men-TOAST nilai besar di luar baris; query search memilih kolom eksplisit, jadi `raw` tidak ikut terbaca).

### 4.2 Tabel global (dibaca semua workspace)

| Tabel | Kolom kunci | Catatan |
|---|---|---|
| `users` | `id` (= ID identitas Kratos), `email`, `name`, `is_ops` | Dibuat JIT saat request pertama |
| `places` | §4.1 | Satu baris per `google_place_id` |
| `imports` | `id`, `file_name`, `file_sha256`, `observed_at`, `status`, `counts` jsonb, `rejects` jsonb (per alasan + 100 sampel baris), `health` jsonb, `started_at`, `finished_at` | Riwayat import + ringkasan kesehatan data (FR-OPS-01) |
| `areas` | `id` bigint (GeoAPI), `admin_level` 1–4, `name`, `external_code`, `parent_id`, `area_km2`, `population`, `density`, `centroid`, `geom` MultiPolygon (level 4 saja), `synced_at` | GIST(`geom`). Populasi = Kontur 2023 (perkiraan model) |
| `area_metrics` | PK (`area_id`, `category_id`; `'*'` = semua), `business_count`, `density_per_1k`, `competition_pct`, `competition_level`, `limited`, `model_version` (`geo-X.Y.Z`) | Level 3–4, global (tidak bergantung ICP) |
| `categories` | `id` (mis. `fnb.restoran_asia`, `lainnya`), `parent_id`, `name` | Taksonomi 2 level, seed dari `data/categories.yaml` |
| `category_map` | PK (`schema_type`, `cuisine`; `'*'` = apa saja) → `category_id` | BR-NORM-03, seed dari file yang sama (DRY) |
| `peer_stats` | PK (`category_id`, `kab_id`), `n`, `url_fill`, `own_domain_share`, `same_as_fill`, `review_p25` | Kelompok pembanding BR-OPP-01 dan teks alasan |
| `place_gaps` | PK (`place_id`, `code`), `severity`, `params` jsonb (mis. `{"peer_n":212,"own_domain_pct":61}`) | Gap global; relevansi Offering dihitung saat skor |
| `suppressions` | `id`, `kind` (`phone`\|`url`\|`place`), `value` (ternormalisasi), `reason`, `created_by`; UQ(`kind`,`value`) | FR-PRIV-02 |
| `flags` | `key` (`ai`, `export`, `webhook`), `enabled`, `updated_by`, `updated_at` | Kill switch global |
| `audit_log` | `id`, `at`, `actor_id`, `workspace_id` NULL, `action`, `target_type`, `target_id`, `meta` jsonb, `ip` | Append-only (FR-OPS-02) |

### 4.3 Tabel workspace (RLS)

| Tabel | Kolom kunci | Catatan |
|---|---|---|
| `workspaces` | `id`, `name`, `plan`, `created_at` | Policy: baris workspace aktif atau milik membership user |
| `members` | `id`, `user_id` (NULL sampai undangan diterima), `email`, `role` (`admin`\|`member`), `status` (`invited`\|`active`), `invite_token_hash`, `active_icp_id` | UQ(`workspace_id`,`email`). ICP aktif per user (FR-ICP-02) |
| `offerings` | `id`, `name`, `tags` text[] (PRD §8), `description` | Tag pertama = tag primer untuk override bobot |
| `icps` | `id`, `offering_id`, `name`, `criteria` jsonb (§6.1), `score_status` (`pending`\|`scoring`\|`ready`), `candidates`, `scored`, `model_version`, `scored_at`, `updated_at` | Status dipakai UI untuk progres |
| `icp_area_scores` | PK (`icp_id`, `area_id`), `level`, `market_score` (NULL = data terbatas), `matching_count`, `hot_warm_count` | Lampiran C bagian yang bergantung ICP |
| `scores` | PK (`icp_id`, `place_id`), `priority`, `band`, `fit`, `coverage`, `limited`, `factors` jsonb, `top_opp`, `model_version`, `computed_at` | Hanya prospek yang lolos kriteria *wajib* |
| `lists` / `list_items` | `id`, `name`, `created_by` / PK (`list_id`, `place_id`), `added_by` | FR-ACT-01 |
| `tags` | PK (`workspace_id`, `place_id`, `tag`) | FR-ACT-01 |
| `assignments` | PK (`workspace_id`, `place_id`), `owner_id` | Pemilik prospek |
| `notes` | `id`, `place_id`, `author_id`, `body`, `created_at` | — |
| `saved_searches` | `id`, `name`, `query` jsonb (`q`, `filters`, `icp_id`), `created_by` | FR-SRCH-02 |
| `feedback` | PK (`workspace_id`, `place_id`, `opp_code`), `relevant` bool, `user_id`, `at` | FR-INTL-05 |
| `unlocks` | PK (`workspace_id`, `place_id`), `via` (`unlock`\|`export`\|`webhook`), `ledger_id`, `user_id`, `at` | Permanen (FR-ACT-02) |
| `credit_ledger` | `id`, `meter` (`credit`\|`ai_credit`), `kind` (`grant`\|`debit`\|`refund`), `delta` int, `idem_key`, `ref`, `actor_id`, `note`, `at`; UQ(`workspace_id`,`idem_key`) | Append-only (§12) |
| `exports` | `id`, `idem_key` (UQ per workspace), `format` (`csv`\|`xlsx`), `selection` jsonb, `status`, `rows`, `charged`, `file_key`, `error`, `expires_at` | FR-ACT-03 |
| `webhooks` | `id`, `url`, `secret_enc` bytea, `active`, `created_by` | Secret dienkripsi AES-GCM (`MSP_ENCRYPTION_KEY`) |
| `webhook_deliveries` | `id`, `webhook_id`, `event`, `payload` jsonb, `status` (`pending`\|`ok`\|`failed`), `attempts`, `last_code`, `last_error`, `done_at` | Log pengiriman |
| `ai_briefs` | `id`, `place_id`, `icp_id`, `input_hash`, `prompt_version`, `provider`, `model`, `status`, `output` jsonb, `error`, `expires_at` | Cache 7 hari per workspace |

### 4.4 Pola RLS

```sql
ALTER TABLE scores ENABLE ROW LEVEL SECURITY;
ALTER TABLE scores FORCE ROW LEVEL SECURITY;
CREATE POLICY ws ON scores
  USING      (workspace_id = current_setting('app.workspace_id', true)::uuid)
  WITH CHECK (workspace_id = current_setting('app.workspace_id', true)::uuid);
-- members: tambahan OR user_id = current_setting('app.user_id', true)::uuid  (agar /me bisa daftar workspace)
```

```go
// platform/db — satu-satunya jalan ke tabel workspace
func (d *DB) InWorkspace(ctx context.Context, ws, user uuid.UUID, fn func(q *q.Queries, tx pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, d.pool, func(tx pgx.Tx) error {
		// set_config(..., true) = SET LOCAL, tapi menerima parameter bind
		if _, err := tx.Exec(ctx, `SELECT set_config('app.workspace_id', $1, true),
		                                 set_config('app.user_id', $2, true)`, ws.String(), user.String()); err != nil {
			return err
		}
		return fn(q.New(tx), tx)
	})
}
```

- Setting tidak ada → `NULL` → tidak ada baris (fail closed).
- Dua role DB: `msp_owner` (pemilik objek, hanya `msp migrate`) dan `msp_app` (api, worker, tool; `NOBYPASSRLS`, bukan owner). `UPDATE`/`DELETE` pada `audit_log` dan `credit_ledger` di-`REVOKE` dari `msp_app`.
- Akses lintas workspace hanya lewat sedikit fungsi `SECURITY DEFINER` yang eksplisit: `accept_invite(token_hash, user_id)`, `icps_to_rescore()`, `ops_workspaces()`.

### 4.5 Claim type diturunkan saat dibaca (KISS)

Claim type tidak disimpan per nilai. Satu tabel di Go memetakan **field → claim type + asal**. Nilai `NULL` selalu menjadi **Tidak diketahui** (BR-TRUST-03). Tabel ini dipakai profil, export, dan webhook (DRY).

```go
// intel/claims.go
var Fields = map[string]Field{
	"name":         {Claim: Observed, Path: "name"},
	"telephone":    {Claim: Observed, Path: "telephone"},
	"phone_kind":   {Claim: Derived, From: "telephone", Rule: "BR-NORM-01"},
	"whatsapp":     {Claim: Inferred, From: "telephone", Rule: "BR-NORM-01"},
	"category":     {Claim: Derived, From: "@type,servesCuisine", Rule: "BR-NORM-03"},
	"website_kind": {Claim: Derived, From: "url", Rule: "BR-NORM-02"},
	"desa":         {Claim: Derived, From: "geo", Rule: "FR-DATA-04"},
	"population":   {Claim: Inferred, Source: "GeoAPI · Kontur 2023 (perkiraan model)"},
	// ... opportunity selalu Inferred (BR-TRUST-01)
}
```

Bentuk nilai di API: `{"value": "Restoran Asia", "claim": "derived", "source": {"import_id": "…", "path": "@type,servesCuisine", "rule": "BR-NORM-03", "observed_at": "2026-08-01"}}`. Nilai fallback diberi `"confidence": "low"`: kategori **Lainnya** (tipe tak dikenal, tetap Turunan) dan desa hasil `nearest`.

- **Masking (FR-ACT-02, FR-PRIV-02):** yang di-mask hanya **nomor telepon** (`+62 812-****-**90`, `"locked": true`) sampai di-unlock. Website dan akun sosial selalu terlihat. Nilai ter-suppress disembunyikan di semua tampilan dan tidak bisa di-unlock. Aturan ini ada di satu fungsi `intel.ProspectView`. Evidence (`raw`) melewati mask yang sama.
- **Freshness (BR-TRUST-02)** dihitung dari `places.observed_at`.

---

## 5. Pipeline F1

```
msp tool geo-sync                      (sekali; ulang bila GeoAPI berubah)
msp tool import --observed-at 2026-08-01 [--dry-run] file.jsonl.gz
   └─► geo.assign ─► geo.metrics ─► intel.derive ─► score.all ─► score.icp × N
       └──────────── queue import ──────────────┘   └──── queue score ────┘
```

Setiap tahap adalah job River yang meng-enqueue tahap berikutnya di transaksi yang sama (`InsertTx`). Setiap tahap **menghitung ulang penuh** dan idempoten: aman diulang. Incremental baru di F3.

### 5.1 `msp tool import`

1. Stream baris (`bufio.Reader`, gzip dideteksi otomatis, baris maks. 1 MB). Tidak memuat file utuh ke memori.
2. Validasi. Alasan tolak: `invalid_json`, `missing_place_id`, `missing_name`, `duplicate_in_file`. Peringatan (record tetap masuk): `unmapped_type`, `invalid_geo`, `invalid_phone`.
3. Normalisasi (fungsi murni, unit test per aturan, FR-DATA-02):

| Input | Aturan | Hasil |
|---|---|---|
| `telephone` | libphonenumber (region ID) → E.164; `+628…` = seluler (BR-NORM-01) | `phone_e164`, `phone_kind`; nilai mentah tetap |
| `url`, `sameAs` | Parse host, cocokkan pola di `data/url-patterns.yaml` (BR-NORM-02) | `url_kind`, `social_links` |
| `@type`, `servesCuisine` | `@type` berupa array → pakai tipe schema.org **paling spesifik** yang ada di `category_map` (kedalaman hierarki terbesar; `["LocalBusiness","Restaurant"]` → `Restaurant`). Lalu cuisine spesifik dulu, baru `'*'` (BR-NORM-03). Tipe tak dikenal → kategori **Lainnya** (`lainnya`) + peringatan `unmapped_type` | `category_id` (tidak pernah NULL) |
| `aggregateRating` | rating 0–5, ulasan ≥ 0; rating tanpa ulasan → rating NULL | `rating`, `review_count` |
| `geo` | Lat/lng valid dan di dalam bbox Indonesia | `geom` (NULL bila tidak valid) |
| `openingHoursSpecification` | Disimpan ternormalisasi (hari, buka, tutup) | `opening_hours` |
| tanggal record (mis. `dateModified`) | Bila ada dipakai, bila tidak → `--observed-at` (BR-TRUST-02) | `observed_at` |

4. Setiap baris valid di-`COPY` (dengan nomor baris) ke tabel staging sementara. Setelah file habis, merge per potongan (`hashtext(google_place_id) % 16`). **Baris terakhir menang**: untuk `google_place_id` yang muncul lebih dari sekali dalam satu file, hanya baris dengan nomor terbesar yang dipakai. Kemunculan sebelumnya dihitung `duplicate_in_file` di laporan.

```sql
INSERT INTO places AS p (...)
SELECT DISTINCT ON (google_place_id) ... FROM import_stage WHERE chunk = $1
ORDER BY google_place_id, line_no DESC                              -- baris terakhir menang
ON CONFLICT (google_place_id) DO UPDATE SET ... , desa_id = NULL    -- wilayah dihitung ulang bila berubah
WHERE p.raw_hash <> EXCLUDED.raw_hash OR p.norm_version <> EXCLUDED.norm_version
RETURNING (xmax = 0) AS inserted;   -- true = baru, false = berubah; tidak kembali = tidak berubah
```

5. Laporan ke stdout + `imports`: baru, berubah, tidak berubah, ditolak per alasan (FR-DATA-01). Lalu enqueue `geo.assign`.

**Semantik re-import.** File yang sama → semua "tidak berubah", tanpa duplikat. Normalizer berubah (naikkan `norm_version`) → import ulang file yang sama menormalisasi ulang semua record. Record yang hilang dari file baru **tidak dihapus** di F1 (`import_id` lama tetap terlihat). Deteksi tutup/berubah = signals F2. `--observed-at` wajib diisi (tanggal pengambilan dataset).

### 5.2 `msp tool geo-sync`

1. Metadata semua level: `GET /v1/areas?country=ID&admin_level=L&limit=500&offset=k` (±180 request untuk 34/522/7.069/81.912 area). Upsert ke `areas`.
2. Polygon **hanya level 4** (desa/kelurahan): `GET /v1/areas/{id}?format=geojson&simplify=<toleransi kecil>` → `ST_Multi(ST_GeomFromGeoJSON(...))`. Level 1–3 cukup lewat `parent_id`; peta memakai vector tiles GeoAPI.
3. Sopan terhadap batas 120 req/menit/IP: token bucket 1,5 req/s, ikuti `429` + `Retry-After`, backoff untuk 5xx. **Resumable**: area yang sudah punya `geom` dilewati (kecuali `--refresh`).
4. Selesai: cek jumlah per level sesuai angka di atas, lalu enqueue `geo.assign` penuh.

Catatan: sync polygon pertama butuh ±82 ribu request (±15 jam), jadi dijalankan sekali di background. Per 2026-10-10, mode GeoJSON daftar GeoAPI dibatasi 10.000 fitur dan tidak memfilter `parent_id`, sehingga fetch per area diperlukan.

### 5.3 Tahap job

| Job (queue) | Isi |
|---|---|
| `geo.assign` (`import`) | `ST_Contains` untuk place yang `desa_id IS NULL`; fallback desa terdekat ≤ 500 m (`area_method='nearest'`). Target ≥ 98% (FR-DATA-04), dipantau metrik |
| `geo.metrics` (`import`) | `area_metrics` untuk level 3–4 sesuai Lampiran C (bagian yang tidak bergantung ICP) |
| `intel.derive` (`import`) | `review_pctl` (`percent_rank()`, BR-SCORE-03), `peer_stats`, lalu `place_gaps`: stream place + peer → `intel.DetectGaps()` (Lampiran B, BR-OPP-01) → ganti isi tabel dalam satu transaksi. Tulis `imports.health` |
| `score.all` (`score`) | Enqueue `score.icp` untuk setiap ICP (`icps_to_rescore()`) |

```sql
-- geo.assign (inti)
UPDATE places p SET desa_id = d.id, kec_id = k.id, kab_id = k.parent_id,
       prov_id = (SELECT parent_id FROM areas WHERE id = k.parent_id), area_method = 'contains'
FROM areas d JOIN areas k ON k.id = d.parent_id
WHERE d.admin_level = 4 AND p.desa_id IS NULL AND p.geom && d.geom AND ST_Contains(d.geom, p.geom);

-- geo.metrics (kelurahan; kecamatan sama dengan kec_id). Area tanpa bisnis ikut sebagai pembanding (n = 0)
INSERT INTO area_metrics (area_id, category_id, business_count, density_per_1k, competition_pct, limited, model_version)
SELECT area_id, cat, n, n * 1000.0 / nullif(pop, 0),
       round(100 * percent_rank() OVER (PARTITION BY kab_id, cat ORDER BY n * 1.0 / nullif(pop, 0))),
       (pop < 500 OR n < 5), 'geo-2.0.0'
FROM area_category_counts;   -- CTE: LEFT JOIN areas ← places, GROUP BY GROUPING SETS ((desa, kategori), (desa))
```

---

## 6. Skor & ICP

### 6.1 ICP

```json
{ "categories":  { "ids": ["fnb.restoran"], "mode": "required" },
  "areas":       { "ids": [3374], "mode": "required" },
  "digital":     { "website_kinds": ["none", "social", "aggregator"], "mode": "preferred" },
  "min_rating":  { "value": 4.0, "mode": "preferred" },
  "min_reviews": { "value": 20, "mode": "preferred" } }
```

| Kriteria | Cocok (1) | Sebagian (0,5) | Tidak diketahui |
|---|---|---|---|
| `categories` | `category_id` ∈ ids | Induk sama | — (`lainnya` = tidak cocok, kecuali dipilih) |
| `areas` (ID GeoAPI level 1–4) | desa/kec/kab/prov ∈ ids | — | `desa_id` NULL |
| `digital` | `url_kind` (NULL = `none`) ∈ daftar | — | — |
| `min_rating` / `min_reviews` | ≥ nilai | rating ≥ nilai − 0,3 / ulasan ≥ nilai ÷ 2 | NULL |

- ICP Fit = rata-rata kriteria yang diketahui × 100. Kriteria *wajib* bernilai 0 → prospek bukan kandidat (tidak tampil). Kriteria *wajib* yang tidak diketahui tetap kandidat, tetapi dibatasi WARM + "Data terbatas" (BR-SCORE-02).
- Kandidat difilter di SQL dengan predikat statis berparameter (mis. `(rating IS NULL OR rating >= $min - 0.3)`), tanpa SQL dinamis.

### 6.2 Perhitungan dan penyimpanan

`intel.Score(facts, icp, cfg) Result` adalah **fungsi murni**: faktor, renormalisasi, coverage, band, dan cap persis sesuai Lampiran A dan BR-SCORE-01/02. Input + versi yang sama selalu memberi hasil yang sama. Hasil tie diurutkan dengan `place_id`. `factors` menyimpan breakdown untuk "Kenapa prospek ini?": `{"icp_fit":{"v":100,"w":30},"opportunity":{"v":70,"w":25,"code":"OPP-WEB-SOCIAL"},"area":null,…}` (`null` = tidak diketahui, bobot dibagi ulang).

Job `score.icp {workspace_id, icp_id}`:

1. `pg_try_advisory_lock(icp)`: bila ICP sedang dihitung → `JobSnooze(30 s)`. Satu job per ICP pada satu waktu.
2. Satu SQL agregat kandidat per kelurahan/kecamatan → `icp_area_scores.market_score` (Lampiran C, bagian ICP; pembanding = area selevel dalam kab/kota yang sama).
3. Stream kandidat (+ gap, area, peer) **berurut proksi murah** (severity gap relevan tertinggi, lalu jumlah ulasan) → `intel.Score` → upsert batch 2.000 dan **commit per batch**, lalu update `icps.scored`.
4. Hapus skor lama (`computed_at` < mulai job), isi `hot_warm_count`, set `score_status='ready'`, `model_version`, dan `scored_at`.

**Onboarding ≤ 5 menit (FR-ICP-01).** Karena batch pertama berisi kandidat terbaik menurut proksi dan langsung di-commit, halaman pertama sudah tampil dalam hitungan detik setelah ICP disimpan. UI menampilkan progres `scored/candidates` dan me-refresh sampai `ready`.

### 6.3 Pemicu recompute

| Pemicu | Aksi |
|---|---|
| ICP dibuat/diubah, Offering diubah | API meng-enqueue `score.icp` untuk ICP terkait (transaksi yang sama) |
| Pipeline import selesai | `score.all` |
| Versi model berubah (deploy dengan `priority-X.Y.Z` baru) | Periodic job River per jam: ICP dengan `model_version` ≠ versi aktif atau `scored_at` < import terakhir → enqueue. Manual: `msp tool rescore [--icp=<id>]` |

### 6.4 Konfigurasi bobot

Satu file berversi di repo, di-embed ke binary. Perubahan bobot atau aturan = naikkan versi (semver).

```yaml
# data/scoring/priority-2.0.0.yaml
version: priority-2.0.0
weights: { icp_fit: 30, opportunity: 25, area: 15, contact: 15, popularity: 10, completeness: 5 }  # Lampiran A
overrides: {}   # per tag primer Offering, mis. reputation_mgmt: { opportunity: 30, popularity: 5 }
bands: { hot: 80, warm: 60 }
```

---

## 7. Search

```sql
CREATE EXTENSION IF NOT EXISTS unaccent;  CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE TEXT SEARCH CONFIGURATION msp (COPY = simple);
ALTER TEXT SEARCH CONFIGURATION msp ALTER MAPPING FOR hword, hword_part, word WITH unaccent, simple;
CREATE INDEX ON places USING gin (search);
CREATE INDEX ON places USING gin (name gin_trgm_ops);
CREATE INDEX ON places USING gist (geom);
CREATE INDEX ON scores (icp_id, priority DESC, place_id DESC);
```

Konfigurasi `simple` + `unaccent` (tanpa stemming, cocok untuk nama usaha Indonesia). Karena regconfig ditulis eksplisit, `to_tsvector('msp', …)` immutable dan bisa dipakai di kolom generated. Trigram menangani salah ketik dan potongan nama.

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

- **Cakupan:** default `scope=icp` (driver = index `scores`). `scope=all` mencari di semua `places` dengan `LEFT JOIN scores`, urut `priority DESC NULLS LAST`, lalu `place_id` (prospek di luar ICP tampil "Tidak cocok ICP").
- **Pagination:** keyset, `cursor` = base64 dari kunci urut terakhir, `limit` maks. 100.
- **Jumlah:** `SELECT count(*) FROM (<filter yang sama> LIMIT 10001)` → tampil "10.000+" bila melewati batas. Konsisten dengan filter (FR-SRCH-01).
- **Profil:** satu query (place + gap + wilayah + metrik + skor + unlock + suppression) → `intel.ProspectView`. Target p95 ≤ 500 ms (NFR-01).
- **Peta (FR-GEO-01):** browser memuat TileJSON GeoAPI (`/v1/tiles/areas.json`, layer `areas`, `area_labels`) langsung dengan MapLibre. Warna diambil dari `GET /areas/ranking` lewat `feature-state` (kunci = ID area). Atribusi TileJSON tampil di peta, dan tabel peringkat selalu tersedia.

---

## 8. API

### 8.1 Konvensi

| Aspek | Aturan |
|---|---|
| Kontrak | `contracts/openapi.yaml` adalah sumber tunggal. Go: oapi-codegen (`std-http-server` + `strict-server`) → `internal/api`. TS: `openapi-typescript` + `openapi-fetch` → `packages/api-client`. Target Makefile yang sama di lokal dan CI: `make generate` (juga sqlc + design tokens), `make lint`, `make test`. CI gagal bila hasil generate berbeda |
| Basis | `/api/v1`, JSON, same-origin lewat Caddy. Workspace dipilih dengan header `MSP-Workspace: <uuid>`, divalidasi terhadap membership |
| Role | Ditulis sekali per operasi di OpenAPI: `x-msp-role: member\|admin\|ops`. Middleware membacanya via operationId, dan test matriks otorisasi memakai sumber yang sama |
| Error | `application/problem+json` (RFC 9457) + `code` stabil + `request_id`. 500 tidak pernah membawa detail internal |
| Idempotensi | Header `Idempotency-Key` **wajib** di POST berbayar (`/unlocks`, `/exports`, `/prospects/{id}/brief`, grant ops). Disimpan sebagai kunci unik di `credit_ledger`/`exports`. Request ulang mengembalikan hasil yang sama tanpa tagihan baru (§12) |
| Async | `202 Accepted` + `Location` ke resource. Resource membawa `{status: queued\|running\|done\|failed, progress: {done, total}, error}` (export, brief, scoring ICP) |
| Batas | Body ≤ 1 MB, `limit` ≤ 100, export maks. 10.000 baris per job (FR-ACT-03; lebih → 422). Rate limit in-process per user untuk brief dan export (`x/time/rate`; cukup karena satu instance api) |

| HTTP | `code` |
|---|---|
| 401 | `UNAUTHENTICATED` |
| 402 | `INSUFFICIENT_CREDITS` (extension `required`, `available`) |
| 403 | `FORBIDDEN`, `EMAIL_NOT_VERIFIED`, `MFA_REQUIRED` |
| 404 | `NOT_FOUND` (termasuk resource milik workspace lain) |
| 409 | `CONFLICT` (mis. `Idempotency-Key` dipakai ulang dengan body berbeda) |
| 422 | `VALIDATION_FAILED` (extension `errors[]` berisi `pointer`/`field` + pesan), `WEBHOOK_URL_REJECTED` |
| 429 | `RATE_LIMITED` |
| 500 | `INTERNAL` |
| 503 | `FEATURE_DISABLED` (kill switch), `AI_UNAVAILABLE` |

### 8.2 Endpoint F1

Role: **M** = Member (Admin juga boleh), **A** = Admin, **O** = ops (`is_ops` + AAL2). **$** = memakai credit + `Idempotency-Key`.

| Method | Path | FR | Role |
|---|---|---|---|
| GET | `/me` (user, membership, role, ICP aktif, saldo) | AUTH-01/02 | login |
| POST | `/workspaces` (onboarding) | AUTH-02 | login, email terverifikasi |
| GET · POST · PATCH · DELETE | `/members`, `/members/{id}` (undang = POST) | AUTH-02 | M baca · A tulis |
| POST | `/invites/accept` | AUTH-02 | login |
| GET · POST · PUT | `/offerings`, `/offerings/{id}` | ICP-01 | M baca · A tulis |
| GET · POST · PUT · DELETE | `/icps`, `/icps/{id}` (status scoring ikut di body) | ICP-01/02 | M baca · A tulis |
| PUT | `/me/active-icp` | ICP-02 | M |
| GET | `/categories`, `/areas?q=&level=&parent_id=` | ICP-01, SRCH-01 | M |
| GET | `/prospects?q=&filters…&scope=&cursor=` | SRCH-01 | M |
| GET | `/prospects/{id}`, `/prospects/{id}/evidence` | PROF-01, INTL-01 | M |
| PUT | `/prospects/{id}/feedback` (per kode opportunity) | INTL-05 | M |
| GET · POST · DELETE | `/saved-searches`, `/saved-searches/{id}` | SRCH-02 | M |
| GET | `/areas/ranking?parent_id=&level=` | GEO-01 | M |
| GET · POST · DELETE | `/lists`, `/lists/{id}`, `/lists/{id}/items` | ACT-01 | M |
| PUT · POST | `/prospects/{id}/tags`, `/prospects/{id}/owner`, `/prospects/{id}/notes` | ACT-01 | M |
| POST | `/unlocks` (`place_ids` ≤ 100) | ACT-02 | M $ |
| POST | `/charges/preview` (export & webhook: jumlah, sudah di-unlock, ter-suppress, biaya, saldo) | ACT-03/04 | M |
| POST · GET | `/exports` → 202, `/exports/{id}` (memuat `download_url`) | ACT-03 | M $ |
| GET | `/exports/{id}/file?exp=&sig=` | ACT-03 | tanda tangan (§13) |
| GET · POST · PATCH · DELETE | `/webhooks`, `/webhooks/{id}` | ACT-04 | A |
| POST · GET | `/webhooks/{id}/test`, `/webhooks/{id}/deliveries` | ACT-04 | A |
| POST | `/webhooks/{id}/send` (`place_ids` ≤ 100) → 202 | ACT-04 | M $ |
| POST · GET | `/prospects/{id}/brief` → 202, `/briefs/{id}` | ACT-05 | M $ (AI credit) |
| GET | `/credits` (saldo; riwayat ledger untuk Admin) | BILL-01 | M |
| GET · POST | `/ops/workspaces`, `/ops/workspaces/{id}/credits` (grant/refund) | OPS-01, BILL-01 | O $ |
| GET | `/ops/imports`, `/ops/data-health` | OPS-01 | O |
| GET · PUT | `/ops/flags`, `/ops/flags/{key}` | OPS-01 | O |
| GET · POST · DELETE | `/ops/suppressions`, `/ops/suppressions/{id}` | OPS-01, PRIV-02 | O |
| GET | `/ops/audit` | OPS-02 | O |

Health di luar `/api/v1`: `/healthz`, `/readyz`. `/metrics` hanya di listener internal (§14).

---

## 9. Auth (Kratos)

- **Same domain.** Caddy me-route `/.ory/*` ke Kratos public API. Cookie sesi Kratos (`HttpOnly`, `Secure`, `SameSite=Lax`) otomatis terkirim ke `/api/v1`. Admin API Kratos hanya bisa diakses dari jaringan Docker internal.
- **Browser flows.** Next.js merender UI login, registrasi, verifikasi, recovery, dan settings dari flow Kratos (`/self-service/*/browser`, `@ory/client`). Metode: password, **Google OIDC**, TOTP. Kratos mengirim email lewat courier SMTP-nya sendiri.
- **Middleware api.** Cookie → `GET {MSP_KRATOS_PUBLIC_URL}/sessions/whoami` (cache in-process 30 s per hash cookie) → `Principal{identity_id, email, verified, aal}`. Baris `users` dibuat JIT. Email belum terverifikasi → `403 EMAIL_NOT_VERIFIED` (FR-AUTH-01).
- **Membership & role** dibaca dari DB kita pada setiap request (tanpa cache), jadi anggota yang dikeluarkan langsung kehilangan akses. Undangan: token acak, hanya hash yang disimpan di `members`, berlaku 7 hari, diterima lewat `accept_invite()`.
- **Ops (FR-AUTH-03).** `/api/v1/ops/*` mensyaratkan `users.is_ops` **dan** sesi `aal2` (TOTP). Tanpa itu → `403 MFA_REQUIRED`. Kratos `session.whoami.required_aal: highest_available` memaksa akun yang punya TOTP untuk memakainya.
- **CSRF.** Flow Kratos punya token CSRF sendiri. Untuk API: `SameSite=Lax`, request yang mengubah data wajib `Content-Type: application/json`, dan `Origin` harus sama dengan `MSP_PUBLIC_URL`.
- Kratos memakai database `kratos` di instance Postgres yang sama, dengan backup yang sama.

---

## 10. Webhook

- **Event F1:** `prospects.sent` (anggota mengirim prospek terpilih) dan `webhook.test`. Payload memakai `intel.ProspectView` yang sama dengan export (DRY). Kontak hanya terisi bila prospek sudah di-unlock dan tidak ter-suppress. Webhook tidak memakai credit.
- **Signature:** `MSP-Signature: t=<unix>,v1=<hex(HMAC-SHA256(secret, t + "." + body))>`, plus `MSP-Event` dan `MSP-Delivery`. Penerima disarankan menolak `|now − t| > 300 s` dan men-dedupe `MSP-Delivery`. Secret 32 byte ditampilkan sekali saat dibuat.
- **Retry:** job River `webhook.deliver` (queue `webhook`), `MaxAttempts` 7, `NextRetry` kustom **1 m, 5 m, 30 m, 2 j, 6 j, 12 j**. Sukses = 2xx. Percobaan habis → `failed` di log pengiriman, bisa dikirim ulang manual. Flag `webhook` off → job di-snooze.
- **SSRF:** saat URL didaftarkan, hanya `https`, tanpa userinfo, dan semua IP hasil resolve harus lolos cek di bawah (bila tidak → `422 WEBHOOK_URL_REJECTED`). Cek yang **sama** diulang saat dial, sehingga DNS rebinding tidak lolos.

```go
func dialGuard(_, address string, _ syscall.RawConn) error {
	host, _, _ := net.SplitHostPort(address)
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return err
	}
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || inBlocked(ip) {
		return errBlockedDestination // inBlocked: 0/8, 100.64/10, 169.254.169.254, 192.0.0/24, 198.18/15, 64:ff9b::/96, …
	}
	return nil
}

var webhookHTTP = &http.Client{
	Timeout:       10 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, // 3xx = gagal
	Transport: &http.Transport{
		Proxy:           nil, // jangan lewat proxy env: guard harus melihat IP tujuan
		DialContext:     (&net.Dialer{Timeout: 3 * time.Second, Control: dialGuard}).DialContext,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	},
}
```

Body respons dibaca maksimal 64 KB lalu dibuang. Fixture uji SSRF (IP privat, nama yang resolve ke 127.0.0.1, redirect ke metadata, IPv4-mapped IPv6) ada di 05.

---

## 11. AI brief

```go
type LLM interface {
	Complete(ctx context.Context, req Request) (Response, error) // keluaran selalu JSON
}
type Request struct{ System, User string; Schema json.RawMessage; MaxTokens int }
type Response struct{ JSON []byte; Provider, Model string; InTokens, OutTokens int }

// Fallback: Claude dulu; error, timeout, refusal, atau JSON tidak valid → client OpenAI-compatible.
type Fallback struct{ Primary, Secondary LLM; Timeout time.Duration }
```

- **Claude:** SDK resmi `github.com/anthropics/anthropic-sdk-go`, model dari `MSP_ANTHROPIC_MODEL` (default `claude-opus-5-5`), structured output (`output_config.format` JSON Schema), effort rendah demi latensi. `stop_reason` selain `end_turn` (mis. `refusal`, `max_tokens`) dianggap error → fallback.
- **OpenAI-compatible:** `net/http` biasa ke `{MSP_LLM_FALLBACK_BASE_URL}/chat/completions`, `response_format: json_object`, model `MSP_LLM_FALLBACK_MODEL`.
- **Prompt** `brief-1.0.0`: `internal/action/ai/prompts/brief-1.0.0/{system.md,user.tmpl,schema.json}` (di-embed). Versi naik setiap kali isinya berubah.
- **Input = daftar fakta ber-ID** dari `ProspectView`, tanpa nilai kontak (BR-AI-01; unlock tidak mengubah aturan ini): `{"offering":["website_dev"],"facts":[{"id":"f1","text":"Kategori: Restoran Asia","claim":"derived"},{"id":"f2","text":"Website mengarah ke linktr.ee","claim":"observed"}]}`.
- **Output:** `{"sentences":[{"text":"…","fact_ids":["f2"]}]}`. Validator deterministik: lolos schema, 3–8 kalimat, setiap kalimat punya ≥ 1 `fact_id` yang ada di input, dan tidak ada pola telepon/email/URL. Gagal → **regenerate sekali** dengan daftar pelanggaran. Gagal lagi → `failed`, tanpa tagihan (FR-ACT-05).
- **Cache:** `input_hash = sha256(prompt_version ‖ fakta kanonik ‖ tag offering)`. Brief `done` dengan hash sama di workspace yang sama dan umur < 7 hari dikembalikan langsung tanpa biaya. Cache tidak pernah dibagi lintas workspace (RLS).
- **Alur:** POST cek flag `ai` (off → `503 FEATURE_DISABLED`) dan saldo `ai_credit` (kurang → 402) → job `ai.brief` (queue `ai`) → simpan brief + debit 1 `ai_credit` **dalam satu transaksi** saat sukses. Timeout 15 s per provider, batas job 45 s. Target p95 ≤ 20 s (NFR-02) dipantau metrik.

---

## 12. Credit

- **Ledger append-only** (`credit_ledger`): grant `+n`, debit `−n`, refund `+n`. Saldo = `SUM(delta)` per `meter` (index `workspace_id, meter`). Tidak ada kolom saldo yang bisa berbeda dari ledger (FR-BILL-01).
- **Idempotensi:** `UNIQUE (workspace_id, idem_key)`. Request ulang → pelanggaran unik → handler mengembalikan hasil yang sudah tersimpan. `unlocks` PK (`workspace_id`, `place_id`) menjamin satu prospek tidak pernah ditagih dua kali, walau `idem_key`-nya berbeda.
- **Konkurensi:** setiap debit diawali `SELECT … FROM workspaces WHERE id = $ws FOR UPDATE`, jadi debit dalam satu workspace berjalan serial dan saldo tidak bisa minus.

```sql
-- POST /unlocks, dalam InWorkspace (satu transaksi)
SELECT 1 FROM workspaces WHERE id = $ws FOR UPDATE;
-- n = jumlah place_ids yang belum ada di unlocks; n = 0 → selesai tanpa tagihan
-- SUM(delta) WHERE meter = 'credit' < n → 402 INSUFFICIENT_CREDITS
INSERT INTO credit_ledger (id, workspace_id, meter, kind, delta, idem_key, ref, actor_id)
VALUES ($id, $ws, 'credit', 'debit', -$n, $idem_key, 'unlock', $user);
INSERT INTO unlocks (workspace_id, place_id, via, ledger_id, user_id)
SELECT $ws, unnest($place_ids::uuid[]), 'unlock', $id, $user ON CONFLICT DO NOTHING;
INSERT INTO audit_log (...) VALUES (...);
```

**Export (BR-CRED-01).** (1) `POST /exports/preview` menghitung biaya = jumlah prospek yang belum di-unlock. (2) `POST /exports` mengecek saldo lebih dulu (402 bila kurang), lalu menyimpan `exports` + job `export.run` dalam satu transaksi → 202. (3) Worker menulis file ke storage sebagai file sementara. (4) **Satu transaksi**: lock workspace → hitung ulang biaya → cek saldo (kurang → `failed`, tanpa tagihan) → insert `unlocks` (`via='export'`) → debit ledger `idem_key = 'export:<id>'` → `status='done'`, `file_key`. Transaksi gagal → file dihapus dan tidak ada tagihan. Refund dilakukan manual oleh ops (`kind='refund'`, tercatat di audit).

---

## 13. Storage

```go
type Storage interface {
	Put(ctx context.Context, key string, r io.Reader) error // tulis ke key.tmp lalu rename (atomik)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}
```

- F1: implementasi `local` di `MSP_STORAGE_DIR` (volume bersama api + worker). Key berbentuk `exports/<workspace_id>/<export_id>.<ext>` dan divalidasi (tanpa `..`). Adapter S3 baru di F3, tanpa mengubah pemanggil.
- **URL unduh:** `GET /exports/{id}` (sudah lolos auth + RLS) mengembalikan `download_url = /api/v1/exports/{id}/file?exp=<unix+300>&sig=<base64url(HMAC-SHA256(k_dl, id‖exp))>`. `k_dl` diturunkan dari `MSP_ENCRYPTION_KEY` (HKDF, label `download`). Endpoint file memverifikasi tanda tangan + masa berlaku, lalu streaming dengan `Content-Disposition: attachment`.
- Retensi 7 hari lewat periodic job harian `export.cleanup`. Direktori storage tidak di-backup karena file export bisa dibuat ulang.

---

## 14. Konfigurasi & observability

| Env var | Isi |
|---|---|
| `MSP_ENV` | `local` \| `staging` \| `production` |
| `MSP_PUBLIC_URL` | `https://app.<domain>` (cek `Origin`, tautan email) |
| `MSP_HTTP_ADDR`, `MSP_METRICS_ADDR` | `:8080`, `:9090` |
| `MSP_DATABASE_URL` | Role `msp_app` (api, worker, tool) |
| `MSP_MIGRATE_DATABASE_URL` | Role `msp_owner` (hanya `msp migrate`) |
| `MSP_KRATOS_PUBLIC_URL` | `http://kratos:4433` |
| `MSP_GEOAPI_BASE_URL` | `https://geoapi.mcpke.dev/api` |
| `MSP_ANTHROPIC_API_KEY`, `MSP_ANTHROPIC_MODEL` | Claude |
| `MSP_LLM_FALLBACK_BASE_URL`, `MSP_LLM_FALLBACK_API_KEY`, `MSP_LLM_FALLBACK_MODEL` | Provider OpenAI-compatible |
| `MSP_STORAGE_DIR` | `/var/lib/msp/storage` |
| `MSP_ENCRYPTION_KEY` | 32 byte base64: AES-GCM untuk secret webhook + kunci HKDF untuk URL unduh |
| `MSP_SMTP_URL` | Email undangan anggota (Kratos punya konfigurasi courier sendiri) |
| `MSP_LOG_LEVEL` | `info` (default) |

- `platform/config` membaca env ke satu struct dan memvalidasi saat start (fail fast). Tidak ada file konfigurasi runtime selain data yang di-embed (`data/`: kategori, pola URL, bobot skor, prompt).
- Flag (`flags`) dibaca dengan cache in-process 10 s. Setiap perubahan tercatat di audit.
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
| `msp_llm_requests_total{provider,result}`, `msp_llm_duration_seconds` | NFR-02, frekuensi fallback |
| `msp_geoapi_requests_total{status}` | Rate limit GeoAPI |

Ditambah metrik runtime Go dan statistik pgxpool. `/healthz` = proses hidup (tanpa dependensi). `/readyz` = DB bisa di-ping dan versi migrasi sesuai. `/metrics` hanya di `MSP_METRICS_ADDR`; Prometheus di server ops mengambilnya lewat Caddy dengan IP allowlist + basic auth. Alert dan dashboard ada di 06/09.

---

## 15. Keamanan F1

1. **Permukaan jaringan:** server prod hanya membuka 80/443 (Caddy) dan SSH (key-only). Postgres, admin API Kratos, dan metrik hanya ada di jaringan Docker.
2. **Isolasi workspace:** RLS `FORCE` + role `msp_app` `NOBYPASSRLS` + `InWorkspace` sebagai satu-satunya jalan. Resource milik workspace lain → 404. Test lintas workspace mencakup setiap endpoint (05).
3. **Identitas:** Kratos, email wajib terverifikasi, ops wajib AAL2 (TOTP). Role diperiksa per request dari `x-msp-role`.
4. **Secret:** variabel CI terproteksi → `.env` (izin 600). Tidak ada secret di repo atau log. Secret webhook dienkripsi AES-GCM.
5. **Outbound:** guard SSRF di dial time untuk webhook. Host GeoAPI dan LLM tetap (dari konfigurasi), tidak pernah diambil dari input user.
6. **Data kontak:** di-mask sampai di-unlock, dan suppression diterapkan di satu fungsi `ProspectView` (profil, evidence, export, webhook). Telepon tidak dikirim ke LLM dan di-redact dari log.
7. **Audit:** `audit_log` append-only (`REVOKE UPDATE, DELETE`) untuk login, undangan, unlock, export, webhook, grant, flag, dan suppression.
8. **Dependensi & input:** `govulncheck` + `npm audit` di CI (06). Batas ukuran body/limit/export, validasi OpenAPI di setiap request, dan query hanya lewat sqlc (berparameter).

---

## 16. Evolusi F2/F3 (hanya bila dibutuhkan)

| Fase | Perubahan | Pemicu |
|---|---|---|
| F2 | Upload daftar customer + pencocokan (pg_trgm nama + jarak geo) | Fitur PRD F2 |
| F2 | Riwayat raw per import (`place_versions`) → diff → signals, watch/alert | Re-import berkala (Q-03) |
| F2 | Role Viewer/Manager, laporan data salah + overlay koreksi ops | Fitur PRD F2 |
| F2 | Billing self-serve, grant bulanan otomatis per plan, batas plan lengkap | Pricing final (D-09) |
| F2 | Satu integrasi CRM (pola sama dengan webhook), NL search (LLM → filter kanonik) | Fitur PRD F2 |
| F2 | Proxy + cache tiles GeoAPI di Caddy | Banyak user di balik satu IP mencapai 120 req/menit |
| F2 | Error tracking | Volume error mulai sulit dibaca dari log |
| F3 | Pipeline incremental (hanya place berubah), batch scoring lebih besar | Dataset > 5 juta atau import > 1 jam |
| F3 | Cache (Redis) | p95 search melewati NFR-01 setelah tuning query/index |
| F3 | Adapter `storage` S3 (presigned URL menggantikan URL bertanda tangan) | Server aplikasi kedua |
| F3 | Server aplikasi kedua, replika Postgres untuk baca, PgBouncer, deploy tanpa downtime | Target 99,5% (NFR-03) |
| F3 | Partisi `scores`, mesin search terpisah | Hanya bila Postgres terbukti tidak cukup |
