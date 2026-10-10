# 07 — Delivery Plan (lean)

## MesinProspek — Paket kerja agent AI & gerbang fase

| Field | Value |
|---|---|
| Versi | 0.2 (lean). Menggantikan v0.1 (roadmap sprint & backlog story) |
| Tanggal | 2026-10-10 |
| Basis | [PRD](01-prd.md) = sumber kebenaran scope · [04](04-technical-design.md) desain · [05](05-test-strategy.md) test · [06](06-ci-cd.md) pipeline & deploy |
| Pembangun | Agent AI, maks. 8 paralel, diorkestrasi Claude. Founder menyetujui gerbang fase dan rilis production (D-22) |
| Prinsip | KISS, DRY, make it work → right → fast (D-23). Urutan kerja ditentukan dependensi, bukan kalender. Progres = WP merged + gerbang lulus |

---

## 1. Model eksekusi

### 1.1 Peran

| Peran | Tugas |
|---|---|
| **Orchestrator** (Claude) | Memecah fase menjadi paket kerja `WP-nn`, menulis brief WP, menjaga graf dependensi, menugaskan agent (maks. 8 aktif), review & merge MR, menyiapkan laporan gerbang |
| **Agent** (lane A–H) | Mengerjakan satu WP di branch sendiri sampai acceptance test lulus. Lane hanya saran: WP yang siap diberikan ke agent yang sedang menganggur |
| **Founder** | Menyetujui gerbang fase dan tag rilis `v*` (deploy production). Menyediakan input di §6 |

### 1.2 Siklus satu WP

1. Orchestrator menulis brief WP (satu issue GitLab per WP): outcome, FR terkait, scope file, acceptance test, dependensi, bagian dokumen yang wajib dibaca. WP berstatus *siap* bila semua dependensinya sudah merged.
2. Agent membuat branch `wp/<nn>-<slug>` dari `main`, menulis acceptance test lebih dulu, lalu implementasi.
3. Agent membuka merge request berjudul Conventional Commits dengan WP sebagai scope, mis. `feat(WP-14): skor prioritas` (STD-GIT-02). Pipeline MR wajib hijau.
4. Orchestrator me-review terhadap PRD, 04, dan brief → minta perbaikan atau merge ke `main`.
5. `main` ter-deploy otomatis ke staging; orchestrator memastikan `/readyz` staging sehat dan smoke J-2 hijau.
6. Akhir fase: orchestrator menyerahkan laporan gerbang → founder menyetujui → tag `vX.Y.Z` → deploy production.

### 1.3 Aturan

| # | Aturan | Alasan |
|---|---|---|
| R1 | Satu WP = satu outcome yang jelas + acceptance test yang membuktikannya | Review cepat; selesai berarti terbukti |
| R2 | Agent hanya mengubah file dalam scope WP. Butuh perubahan di luar scope → lapor ke orchestrator | Mencegah bentrok antar agent paralel |
| R3 | Kontrak (`contracts/openapi.yaml`, `apps/api/db/migrations/`) hanya diubah di WP kontrak khusus, dikerjakan serial. Kode hasil generate ikut di MR yang sama | Kontrak = file bersama paling rawan konflik |
| R4 | Docs diperbarui di MR yang sama bila perilaku berubah. Perubahan requirement butuh keputusan founder | Satu sumber kebenaran (DRY) |
| R5 | MR kecil, rebase ke `main` sebelum merge. CI merah tidak pernah di-merge | `main` selalu bisa di-deploy |
| R6 | Tidak menambah komponen di luar Stack F1 ([README](README.md)) tanpa persetujuan orchestrator | KISS |

**Selesai (DoD WP):** acceptance test ada dan lulus di CI · test lintas workspace untuk tabel/endpoint workspace baru (FR-PRIV-01) · docs terkait diperbarui · review orchestrator lulus · staging sehat setelah merge.

Penomoran: Fase 0 = `WP-01..08`, F1 = `WP-10..25`, F2 dan seterusnya mulai `WP-30`. Perubahan kontrak di tengah fase = WP baru berlabel `kontrak`.

---

## 2. Fase 0 — Bootstrap

Sampai WP-06 selesai, pekerjaan berjalan di repo saat ini. Setelah itu repo utama pindah ke GitLab (D-18). Q-08 sudah diputuskan: GitLab adalah repo utama; mirror GitHub opsional dan tanpa CI.

| WP | Outcome | Acceptance | Butuh |
|---|---|---|---|
| WP-01 | Skeleton monorepo sesuai README: `apps/api` (Go, binary `msp` dengan subcommand `api`, `worker`, `migrate`, `tool`), `apps/web` (Next.js + TS + Tailwind), `packages/`, `data/`, `deploy/` | `msp api` melayani `/healthz`; `web` ter-build; lint Go & TS lulus | — |
| WP-02 | `contracts/openapi.yaml` skeleton (`/api/v1`, format error, health) + codegen oapi-codegen (Go) dan `packages/api-client` (TS) | Codegen tanpa diff di CI; handler health memakai kode hasil generate | 01 |
| WP-03 | Migrasi baseline goose (PostGIS, pg_trgm) + sqlc + helper RLS (konteks workspace per transaksi) + `msp migrate` | Integration test: query tanpa konteks workspace → 0 baris; migrasi jalan dari DB kosong | 01 |
| WP-04 | Docker Compose `local`: `caddy`, `web`, `api`, `worker`, `kratos`, `postgres` | `docker compose up` → `/readyz` OK | 01, 03 |
| WP-05 | Bootstrap server `ops` (stack `/opt/ops` sesuai README) dan `prod` (Docker, Caddy) | GitLab + registry hidup, runner terdaftar, Grafana bisa dibuka | Q-09, domain |
| WP-06 | GitLab CI `check` (lint, unit, integration dengan Postgres) dan `build` (image `<registry>/msp/api:<git-sha>`, `<registry>/msp/web:<git-sha>`) + deploy otomatis `main` → staging | Pipeline MR & `main` hijau; commit di `main` muncul di staging | 01, 04, 05 |
| WP-07 | Paket `packages/design-tokens` langsung dari output `docs/02-design-system/tokens/build.mjs` (CSS variables + preset Tailwind; tanpa migrasi ke Style Dictionary) | `web` memakai token; `make generate` tanpa diff | 01 |
| WP-08 | Integrasi Kratos: daftar/login email + password, verifikasi email, middleware sesi di `msp api`, halaman auth dasar di `web` | E2E: daftar → verifikasi → login → endpoint terproteksi 200; tanpa sesi 401 | 02, 04 |

Paralel: WP-01 dan WP-05 mulai bersamaan; setelah WP-01 → WP-02, 03, 07; lalu WP-04 → WP-06, 08.

**Gerbang Fase 0** (disetujui founder): pipeline `main` hijau, staging ter-deploy otomatis, `docker compose up` jalan di lokal, daftar + login berhasil di staging.

---

## 3. Fase F1 — Make it work

Target: semua FR PRD §9, lalu Gerbang F1 (§4). Lane A–F mengikuti area kerja; lane G dan H mengerjakan slice penuh (API + UI) karena fiturnya terpisah jelas.

| WP | Outcome | FR | Butuh | Acceptance (test yang membuktikan) | Lane |
|---|---|---|---|---|---|
| WP-10 | **Kontrak F1**: migrasi semua tabel F1 (data global + data workspace, D-03) dengan policy RLS dan `audit_log` append-only; OpenAPI semua endpoint F1 + codegen | PRIV-01, OPS-02 | Gerbang F0 | Integration: data workspace A tidak terbaca dari workspace B (semua tabel workspace); UPDATE/DELETE `audit_log` ditolak; lint OpenAPI + codegen tanpa diff | D |
| WP-11 | **Import & normalisasi**: `msp tool import --observed-at` upsert per `google_place_id`, record mentah → evidence, claim type + sumber per nilai, BR-NORM-01..03 (pola URL & mapping kategori di `data/`), laporan baru/berubah/tidak berubah/ditolak | DATA-01, DATA-02, INTL-01 | 10 | Unit per aturan BR-NORM; import fixture 2× → 0 duplikat & laporan benar; sampel dataset nyata (Q-02) ter-import dengan alasan tolak terlapor | A |
| WP-12 | **Geo-sync**: `msp tool geo-sync` 4 level area + populasi + batas sederhana + atribusi, patuh 120 req/menit, aman diulang | DATA-03 | 10 | Integration dengan respons GeoAPI rekaman; di staging semua area level 1–4 tersimpan (81.912 desa/kelurahan) | B |
| WP-13 | **Wilayah**: penetapan prospek → kelurahan (+ induk) dengan point-in-polygon PostGIS, metrik Lampiran C `geo-2.0.0`, API peringkat wilayah | DATA-04, INTL-03, GEO-01 | 11, 12 | ≥ 98% prospek ber-koordinat (sampel nyata) dapat kelurahan; unit per rumus + guard "Data terbatas"; respons API selalu memuat label populasi Kontur 2023 | B |
| WP-14 | **Opportunity & skor**: katalog Lampiran B + BR-OPP-01 + alasan; ICP fit + `priority-2.0.0` (renormalisasi, coverage, band + cap, `percent_rank`) dengan breakdown & versi; antrean `score`, `msp tool rescore` (metrik wilayah → skor) | INTL-02, INTL-04 | 11, 13 | Unit per kode OPP & per faktor; golden test: input + versi sama → hasil sama; cap BR-SCORE-02; ganti ICP → urutan berubah | C |
| WP-15 | **Auth & workspace**: workspace dibuat saat daftar, undang anggota, role Admin/Member, login Google, data terkunci sebelum verifikasi email, MFA TOTP untuk akun ops | AUTH-01, AUTH-02, AUTH-03 | 10 | Matriks role × endpoint → 403 di luar role; belum verifikasi → ditolak; ops tanpa TOTP ditolak; E2E daftar → undang → anggota login | D |
| WP-16 | **ICP API**: Offering, beberapa ICP per workspace (kriteria wajib/preferensi), ICP aktif per user, feedback Relevan/Tidak relevan | ICP-01, ICP-02, INTL-05 | 10 | Integration CRUD + isolasi workspace; ICP aktif per user; feedback tersimpan per workspace | D |
| WP-17 | **Search & profil API**: search kata kunci (FTS + pg_trgm) + filter + urut skor, saved search, list/tag/catatan/pemilik; search, profil, dan AI brief memakai `intel.ProspectView` (WP-21) | SRCH-01, SRCH-02, PROF-01, ACT-01 | 14, 16, 21 | Tabel kasus filter → jumlah hasil konsisten; saved search dibuka ulang; tiap nilai profil punya claim type + sumber; NFR-01 diukur dengan dataset nyata | D |
| WP-18 | **Web shell & onboarding**: layout app, halaman auth (+ Google), wizard Offering → ICP → hasil pertama, pemilih ICP aktif, kelola anggota (Admin) | AUTH-01, AUTH-02, ICP-01, ICP-02 | 15, 16 | E2E J-1 di stack test CI (Mailpit): daftar → verifikasi → wizard → hasil pertama; cek a11y otomatis dasar (NFR-06). ≤ 5 menit di staging dicek founder di Gerbang F1 | E |
| WP-19 | **Search UI**: kata kunci, filter, band, urut skor, saved search, list/tag/catatan/pemilik | SRCH-01, SRCH-02, ACT-01 | 17, 18 | E2E: filter → jumlah sama dengan API; simpan & buka ulang pencarian; list + tag + catatan + assign | E |
| WP-20 | **Profil & wilayah UI**: profil (skor + "Kenapa prospek ini?", opportunity + feedback, kontak ter-mask, evidence, tautan Google Maps); peringkat wilayah + peta tiles GeoAPI/OSM + tabel padanan + drilldown | PROF-01, INTL-01, INTL-03, INTL-05, GEO-01 | 17, 18 | E2E: tiap nilai tampil dengan claim type; "Tidak diketahui" tidak pernah kosong (BR-TRUST-03); peta selalu punya tabel; drilldown → search terfilter | F |
| WP-21 | **Unlock & credit**: ledger grant/debit/refund idempotent, saldo, unlock 1 credit permanen, `intel.ProspectView` sebagai satu-satunya fungsi proyeksi (mask → unlock → suppression; milik modul `intel`, bukan modul aksi) untuk search/profil/export/webhook, komponen tombol unlock + saldo | ACT-02, BILL-01, PRIV-02 | 10 | Unlock ganda/konkuren → 1 debit; saldo = Σ ledger; kontak tersupresi tidak pernah keluar dari `intel.ProspectView` | G |
| WP-22 | **Export & webhook**: export CSV/XLSX lewat antrean `export` + preview biaya, "Unlock & export/kirim": unlock dulu, export/kirim gratis; webhook HTTPS (tolak IP privat/internal), HMAC, retry, log, kirim prospek terpilih; UI keduanya | ACT-03, ACT-04 | 17, 21 | Export gagal → 0 debit; 10.000 baris ≤ 2 menit (NFR-02); suite SSRF (IP privat, loopback, redirect, DNS rebinding) lulus; HMAC terverifikasi | G |
| WP-23 | **AI brief**: antrean `ai`, Claude → fallback OpenAI-compatible, hanya fakta level bisnis tanpa telepon (BR-AI-01), rujukan fakta per kalimat, cache 7 hari, kill switch, 1 AI credit (cache gratis), prompt `brief-X.Y.Z`, panel di profil | ACT-05 | 17, 21 | Dengan LLM palsu: brief tanpa rujukan → dibuat ulang sekali lalu ditolak; fallback saat Claude gagal; payload tanpa nomor telepon; cache tidak menagih; ≤ 20 s | H |
| WP-24 | **Halaman ops**: daftar workspace + grant credit, riwayat import, kesehatan data, feature flag (AI, export, webhook), suppression list, tampilan audit log | OPS-01, OPS-02, PRIV-02, BILL-01 | 11, 15, 21 | Akun non-ops → 403; grant → entri ledger + audit; flag off → fitur menolak | H |
| WP-25 | **Monitoring & backup**: `/metrics` (lewat Caddy, allowlist IP + basic auth) + `node-exporter` + `blackbox-exporter` → Prometheus + dashboard Grafana + alert (email/Telegram); pgBackRest WAL kontinu ke S3-compatible off-site; drill restore dicatat di [09](09-runbook-operasional.md) | NFR-03, NFR-04 | 06, Q-09 | Restore ke server/DB kosong berhasil; umur WAL terakhir ≤ 15 menit; alert uji terkirim | H |

Catatan:
- Audit log (FR-OPS-02) ditulis oleh WP pemilik aksinya: 15 (login, undang), 21 (unlock), 22 (export, webhook), 24 (grant credit, flag, suppression).
- WP UI (18–20) boleh mulai sejak WP-10 merged dengan mock dari contoh di kontrak; merge hanya setelah dependensinya merged dan E2E lulus.

### 3.1 Graf dependensi

```
WP-10 Kontrak F1  (setelah Gerbang F0)
 ├─► 11 Import ─────┐
 ├─► 12 Geo-sync ───┴─► 13 Wilayah ─► 14 Opp + skor ─┐
 ├─► 16 ICP API ──────────────────────────────────────┤
 ├─► 21 Unlock + credit ──────────────────────────────┴─► 17 Search/profil API ─┬─► 19 Search UI   (+18)
 ├─► 15 Auth ─┬─► 18 Shell + onboarding (+16)                                   ├─► 20 Profil UI   (+18)
 │            └─► 24 Ops (+11, +21)                                             ├─► 22 Export + webhook
 │                                                                              └─► 23 AI brief
WP-06 (F0) ─► 25 Monitoring + backup
```

- **Paralel maksimum** setelah WP-10: 11, 12, 15, 16, 21, 25 + 18 (mock) = 7 agent; slot ke-8 untuk WP perbaikan atau WP kontrak.
- **Jalur kritis:** 10 → 11/12 → 13 → 14 → 17 → 19/20/22/23 → Gerbang F1. Agent yang selesai lebih dulu mengambil WP siap berikutnya, apa pun lane-nya.

### 3.2 Cek cakupan FR PRD §9

| Area | FR → WP |
|---|---|
| AUTH | 01 → 08, 15, 18 · 02 → 15, 18 · 03 → 15 |
| DATA | 01 → 11 · 02 → 11 · 03 → 12 · 04 → 13 |
| INTL | 01 → 11, 17, 20 · 02 → 14 · 03 → 13, 20 · 04 → 14 · 05 → 16, 20 |
| ICP | 01 → 16, 18 · 02 → 16, 18 |
| SRCH / PROF / GEO | SRCH-01 → 17, 19 · SRCH-02 → 17, 19 · PROF-01 → 17, 20 · GEO-01 → 13, 20 |
| ACT | 01 → 17, 19 · 02 → 21 · 03 → 22 · 04 → 22 · 05 → 23 |
| BILL | 01 → 21, 24 |
| OPS | 01 → 24 · 02 → 10 (append-only) + 15, 21, 22, 24 (penulisan) |
| PRIV | 01 → 10 (+ test lintas workspace di tiap WP) · 02 → 21, 24 |

**Hasil: 28/28 FR tercakup.**

---

## 4. Gerbang F1

Kriteria = [PRD §12](01-prd.md#12-gerbang-rilis-f1); cek konkret dan bukti = checklist [05 §7](05-test-strategy.md#7-gerbang-rilis-f1). Orchestrator mengisi checklist itu untuk laporan gerbang.

**Demo founder** (di staging, dataset nyata):
1. Orchestrator menyerahkan laporan gerbang: checklist 05 §7 terisi, link pipeline, daftar WP merged, isu terbuka.
2. Founder daftar akun baru → verifikasi email → wizard → hasil pertama (diukur ≤ 5 menit).
3. Search → profil ("Kenapa prospek ini?", evidence) → unlock → export → kirim ke webhook uji → AI brief.
4. Halaman ops: grant credit, cek kesehatan data, matikan flag AI → brief ditolak.
5. Temuan → WP perbaikan; hanya item yang gagal diulang.
6. Founder menyetujui → tag `vX.Y.Z` → deploy production → onboarding customer beta (Q-07).

---

## 5. Fase F2 — Make it right dan Fase F3 — Make it fast

Kandidat WP = isi F2/F3 di [PRD §6](01-prd.md#6-fase-rilis). Dipecah menjadi `WP-nn` oleh orchestrator setelah Gerbang F1, diurutkan berdasarkan feedback customer beta. Tiap fase punya gerbang sendiri yang disetujui founder. Contoh kandidat awal F2: re-import → signal perubahan (Q-03), billing self-serve, error tracking.

F3 hanya dikerjakan bila pengukuran menunjukkan kebutuhannya.

---

## 6. Input yang dibutuhkan dari founder

Secret (key, password) dimasukkan founder sebagai variabel CI terproteksi GitLab, tidak lewat chat atau repo.

| Input | Ref | Dipakai di | Dibutuhkan sebelum |
|---|---|---|---|
| 2 dedicated server (`prod`, `ops`) + object storage S3-compatible off-site di Indonesia | Q-09, D-14 | WP-05, WP-25 | WP-05 mulai; paling lambat akhir Fase 0 |
| Domain + akses DNS | — | WP-05 (TLS Caddy), WP-08 (email) | WP-05 mulai |
| SMTP (email verifikasi, undangan, alert) | — | WP-08, WP-15, WP-25 | WP-08 diuji di staging |
| Google OAuth client (ID, secret, redirect URI) | — | WP-15 | WP-15 mulai |
| GeoAPI key (opsional) | D-25 | WP-12 | WP-12 mulai |
| File dataset lengkap, atau sampel representatif + jumlah record & cakupan wilayah | Q-02 | WP-11, WP-13, NFR-01 | WP-11 selesai (sampel idealnya saat WP-11 mulai) |
| API key Anthropic + endpoint, key & model OpenAI-compatible (fallback) | D-15 | WP-23 | WP-23 mulai |
| Calon customer beta + Offering mereka | Q-07 | Demo gerbang, kalibrasi bobot | Gerbang F1 |
| Kadens pembaruan dataset | Q-03 | Re-import & signal (F2) | Mulai F2 |

---

## 7. Risiko eksekusi

| Risiko | Mitigasi |
|---|---|
| Agent paralel bentrok di file bersama (kontrak, migrasi, layout UI) | Kontrak-first (WP-10), WP kontrak serial (R3), scope file per WP (R2), MR kecil + rebase |
| Kejutan dataset (format, volume, field kosong) | Import sampel nyata sedini mungkin di WP-11; laporan tolak per alasan; ukur % terisi per field sebelum kalibrasi skor |
| Agent menyimpang dari spec atau menambah fitur | Brief WP merujuk FR/BR; acceptance test ditulis dulu dari PRD; review orchestrator terhadap PRD/04; R6 |
| Test hijau tetapi lemah | Orchestrator mencocokkan test dengan acceptance PRD; golden test skor; coverage paket inti ≥ 80%; demo founder dengan data nyata |
| Input founder terlambat (server, dataset, key) | Tabel §6 dengan batas "dibutuhkan sebelum"; WP yang tidak terblokir tetap jalan di lokal dengan fixture |
| Performa buruk pada dataset nyata | NFR-01/02 diukur di WP-17 dan WP-22 dengan data nyata; optimasi hanya bila terukur (F3) |
