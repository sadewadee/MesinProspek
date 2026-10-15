# 07 — Delivery Plan (lean)

## MesinProspek — Paket kerja agent AI & gerbang fase

| Field | Value |
|---|---|
| Versi | 0.4 (lean). F1 dirampingkan ke inti market intelligence (D-40): 17 WP F1, fitur prospek ke backlog F2 |
| Tanggal | 2026-10-15 |
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
3. Agent membuka merge request berjudul Conventional Commits dengan WP sebagai scope, mis. `feat(WP-26): estimasi kunjungan` (STD-GIT-02). Pipeline MR wajib hijau. WP besar boleh dipecah menjadi beberapa MR kecil (STD-GIT-01).
4. Orchestrator me-review terhadap PRD, 04, dan brief → minta perbaikan atau merge ke `main`.
5. `main` ter-deploy otomatis ke staging; orchestrator memastikan `/readyz` staging sehat dan smoke J-2 hijau.
6. Akhir fase: orchestrator menyerahkan laporan gerbang → founder menyetujui → tag `vX.Y.Z` → deploy production.

### 1.3 Aturan

| # | Aturan | Alasan |
|---|---|---|
| R1 | Satu WP = satu outcome yang jelas + acceptance test yang membuktikannya | Review cepat; selesai berarti terbukti |
| R2 | Agent hanya mengubah file dalam scope WP. Butuh perubahan di luar scope → lapor ke orchestrator | Mencegah bentrok antar agent paralel |
| R3 | Kontrak (`contracts/openapi.yaml`, `apps/api/db/migrations/`) hanya diubah di WP kontrak khusus, dikerjakan serial. Kode hasil generate ikut di MR yang sama | Kontrak = file bersama paling rawan konflik |
| R4 | Docs diperbarui di MR yang sama bila perilaku berubah. Perubahan requirement butuh keputusan founder. Ekspektasi golden test (05 §4.1) hanya berubah bila PRD §12 berubah | Satu sumber kebenaran (DRY) |
| R5 | MR kecil, rebase ke `main` sebelum merge. CI merah tidak pernah di-merge | `main` selalu bisa di-deploy |
| R6 | Tidak menambah komponen di luar Stack F1 ([README](README.md)) tanpa persetujuan orchestrator. Fitur hanya dari data yang kita punya (PRD §5 prinsip 5) | KISS, PRD §5 |

**Selesai (DoD WP):** acceptance test ada dan lulus di CI · test lintas workspace untuk tabel/endpoint workspace baru (FR-PRIV-01) · docs terkait diperbarui · review orchestrator lulus · staging sehat setelah merge.

Penomoran: Fase 0 = `WP-01..08`, F1 = `WP-10..29` (WP-26..28 untuk estimasi dan analisis lokasi; WP-29 cadangan), F2 dan seterusnya mulai `WP-30`. ID lama dipertahankan agar rujukan tidak putus: WP-14 dan WP-16 pindah ke backlog F2 (§5, D-40). Perubahan kontrak di tengah fase = WP baru berlabel `kontrak`.

---

## 2. Fase 0 — Bootstrap

Sampai WP-06 selesai, pekerjaan berjalan di repo saat ini. Setelah itu repo utama pindah ke GitLab (D-18). Q-08 sudah diputuskan: GitLab adalah repo utama; mirror GitHub opsional dan tanpa CI.

| WP | Outcome | Acceptance | Butuh |
|---|---|---|---|
| WP-01 | Skeleton monorepo sesuai README: `apps/api` (Go, binary `msp` dengan subcommand `api`, `worker`, `migrate`, `tool`), `apps/web` (Next.js + TS + Tailwind), `packages/`, `data/`, `deploy/` | `msp api` melayani `/healthz`; `web` ter-build; lint Go & TS lulus | — |
| WP-02 | `contracts/openapi.yaml` skeleton (`/api/v1`, format error, health) + codegen oapi-codegen (Go) dan `packages/api-client` (TS) | Codegen tanpa diff di CI; handler health memakai kode hasil generate | 01 |
| WP-03 | Migrasi baseline goose (PostGIS, pg_trgm) + sqlc + helper RLS (konteks workspace per transaksi) + `msp migrate` | Integration test: query tanpa konteks workspace → 0 baris; migrasi jalan dari DB kosong | 01 |
| WP-04 | Docker Compose `local`: `caddy`, `web`, `api`, `worker`, `kratos`, `postgres` | `docker compose up` → `/readyz` OK | 01, 03 |
| WP-05 | Bootstrap **satu** server (D-19, D-26) dengan skrip di `deploy/`: Docker, satu Caddy untuk semua domain, stack `/opt/ops`, direktori `/opt/msp-staging` dan `/opt/msp` (README), batas CPU/RAM per container, runner 1 job sekaligus; uptime monitor eksternal untuk `app.`, `staging.`, `gitlab.` | GitLab + registry hidup, runner terdaftar, Grafana bisa dibuka, uptime monitor eksternal mengirim alert uji; skrip bootstrap aman dijalankan ulang (dasar rebuild, RTO 8 jam) | Server (✓), domain |
| WP-06 | GitLab CI `check` (lint, unit, integration dengan Postgres) dan `build` (image `<registry>/msp/api:<git-sha>`, `<registry>/msp/web:<git-sha>`) + deploy otomatis `main` → staging | Pipeline MR & `main` hijau; commit di `main` muncul di staging | 01, 04, 05 |
| WP-07 | Paket `packages/design-tokens` langsung dari output `docs/02-design-system/tokens/build.mjs` (CSS variables + preset Tailwind; tanpa migrasi ke Style Dictionary) | `web` memakai token; `make generate` tanpa diff | 01 |
| WP-08 | Integrasi Kratos: daftar/login email + password, verifikasi email, middleware sesi di `msp api`, halaman auth dasar di `web` | E2E: daftar → verifikasi → login → endpoint terproteksi 200; tanpa sesi 401 | 02, 04 |

Paralel: WP-01 dan WP-05 mulai bersamaan; setelah WP-01 → WP-02, 03, 07; lalu WP-04 → WP-06, 08.

**Status 2026-10-15:** WP-01..08 selesai di repo: `make generate lint test` hijau, test integrasi jalan di Postgres 16 + PostGIS, dan journey auth (daftar → verifikasi → masuk, `/api/v1/me` 401/200) lulus dengan Playwright terhadap Kratos, Mailpit, Caddy, `msp api`, dan Next sungguhan. Yang hanya bisa dibuktikan di server founder: `bootstrap.sh` dan stack `/opt/ops` (WP-05), pipeline GitLab termasuk job `e2e` di dind dan anggaran 15 menit MR (WP-06, WP-08), `docker compose up`/build image (WP-04), serta email verifikasi lewat SMTP sungguhan. Gerbang Fase 0 di bawah menunggu langkah itu.

**Gerbang Fase 0** (disetujui founder): pipeline `main` hijau, staging ter-deploy otomatis di server yang sama dengan GitLab, `docker compose up` jalan di lokal, daftar + login berhasil di staging, uptime monitor eksternal aktif.

---

## 3. Fase F1 — Make it work

Target: semua FR PRD §9, lalu Gerbang F1 (§4). Lane A–F mengikuti area kerja; lane G dan H mengerjakan slice penuh (API + UI) karena fiturnya terpisah jelas. Nama tabel, job, endpoint, dan file mengikuti 04.

| WP | Outcome | FR | Butuh | Acceptance (test yang membuktikan) | Lane |
|---|---|---|---|---|---|
| WP-10 | **Kontrak F1**: migrasi **hanya tabel F1** (04 §4 dan tabel "Di luar F1", D-40): `places`, `place_snapshots`, `estimates`, `imports` + cursor, `areas`, `area_metrics`, `categories`, data workspace (`workspaces`, `members`, `lists`/`list_items`, catatan, `saved_searches`, `location_analyses`, `exports`, `ai_briefs`), `credit_ledger`, `plans`, `settings` (seed `est.*` = default PRD §E.5, kunci AI, `billing.signup_grant`), `flags`, `suppressions`, `audit_log` append-only, dengan policy RLS; OpenAPI semua endpoint F1 + codegen | PRIV-01, OPS-02 | Gerbang F0 | Integration: data workspace A tidak terbaca dari workspace B (semua tabel workspace); UPDATE/DELETE `audit_log` ditolak; lint OpenAPI + codegen tanpa diff; tidak ada tabel F2 di migrasi | D |
| WP-11 | **Import dari DB gmaps**: `msp tool import --country ID [--reset]` + job harian `import.run` (03.00 WIB); baca `gmaps-db` inkremental dengan cursor + tumpang-tindih 24 jam; satu-satunya pemetaan kolom di `apps/api/internal/ingest/gmaps/source.sql` (`export`, `create_table`, `read_batch`) + **cek kolom fail-fast**; advisory lock; filter negara; normalisasi BR-NORM-03..06 (kategori dwibahasa via `data/categories.yaml`, harga, jam buka, `popular_times`, distribusi bintang, zona waktu lokal, atribut & pemilik); telepon, website, dan sosial disimpan apa adanya; `newest_review_dates` dari snapshot `reviews_sort = 'newest'` (D-38); **gabung snapshot** BR-NORM-05 → `places` + `place_snapshots`; claim type + sumber; laporan import; fixture `gmaps_fixture` | DATA-01, DATA-02, INTL-01 | 10 | TC-DATA (import & normalisasi F1), EC import, bagian import TC-EST-07; import fixture 2× → 0 duplikat; setelah kiriman pertama dimuat (09 RB-15): import data nyata di staging dengan alasan tolak terlapor | A |
| WP-12 | **Geo-sync**: `msp tool geo-sync --country ID` 4 level area + populasi + batas sederhana + atribusi, patuh 120 req/menit, aman diulang | DATA-03 | 10 | TC-GEO-01; di staging semua area level 1–4 tersimpan (81.912 desa/kelurahan) | B |
| WP-13 | **Wilayah**: `geo.assign` (point-in-polygon PostGIS) + metrik Lampiran C `geo-3.0.0` per kelompok estimasi (semua bisnis untuk `lainnya`): permintaan terbukti dari `estimates`, rentang harga, `market_score`; `GET /areas/ranking?group=`; rantai F1 `geo.assign → est.compute → geo.metrics` | DATA-04, INTL-03, GEO-01 | 11, 12, 26 | TC-GEO-02..04; ≥ 98% bisnis ber-koordinat (sampel nyata) dapat kelurahan; respons API selalu memuat label populasi Kontur 2023 | B |
| WP-15 | **Auth & workspace**: workspace dibuat saat daftar, undang anggota (dibatasi seat plan), role Admin/Member, login Google, data terkunci sebelum verifikasi email, MFA TOTP untuk akun ops | AUTH-01, AUTH-02, AUTH-03, BILL-01 (seat) | 10 | TC-AUTH-01..04; E2E daftar → undang → anggota login | D |
| WP-17 | **Search & profil API**: search kata kunci (FTS + pg_trgm) + filter (kelompok estimasi, wilayah, rentang omzet, punya pola keramaian, ada website/sosial, rating) + urut `visitors`/`revenue`/`velocity`; `intel.ProspectView` sebagai satu-satunya proyeksi (suppression bisnis/URL; **tanpa telepon**) untuk search, profil, dan export; profil: estimasi (rentang, asumsi, sumber harga, `data_as_of`, penanda), pola keramaian (`estimate.Pattern`, zona lokal), riwayat ulasan dari `place_snapshots`, atribut & `is_claimed`, website & sosial; saved search; list + catatan | SRCH-01, SRCH-02, PROF-01, ACT-01, PRIV-02, EST-01..05 (API) | 21, 26 | TC-SRCH-01, TC-INTL-06/07, TC-PRIV-02, TC-DATA-10 (API); GT-01 lapis I (profil API); tidak ada telepon di respons mana pun; saved search dibuka ulang; NFR-01 diukur dengan data nyata | D |
| WP-18 | **Web shell & onboarding**: layout app, halaman auth (+ Google), **onboarding 2 tujuan** (FR-ONB-01: Analisis lokasi → SCR-13, Kompetitor & omzet → SCR-03), kelola anggota (Admin, termasuk info & pesan seat), Pengaturan › Plan, credit & pemakaian (SCR-10, hanya baca) | AUTH-01, AUTH-02, ONB-01, BILL-01 | 15, 21 | E2E J-1 di stack test CI (Mailpit); cek a11y otomatis dasar (NFR-06). ≤ 5 menit di staging dicek founder di Gerbang F1 | E |
| WP-19 | **Search UI & list**: SCR-03 dengan filter & urut F1, saved search; SCR-06 list + catatan dengan kolom estimasi (tabel perbandingan kompetitor) | SRCH-01, SRCH-02, ACT-01 | 17, 18 | E2E: filter → jumlah sama dengan API; simpan & buka ulang pencarian; list + kolom estimasi + catatan (J-3, J-5) | E |
| WP-20 | **Profil & wilayah UI**: SCR-04 dengan panel perkiraan kunjungan & omzet (varian hotel, sensitivitas, catatan metode, penanda), pola keramaian, riwayat ulasan, atribut, website & sosial (tanpa telepon), evidence, "Analisis lokasi di sini"; komponen chart 02 §3.3 + tabel padanan di `packages/ui` (dipakai ulang WP-28); SCR-05 peringkat per kelompok + peta tiles GeoAPI/OSM + tabel padanan + drilldown ke bisnis dan analisis lokasi | PROF-01, EST-01..05 (UI), INTL-01, INTL-03, GEO-01 | 17, 18 | E2E J-2, J-4; "Tidak diketahui"/"Belum dapat diperkirakan" tidak pernah kosong (BR-TRUST-03); setiap chart dan peta punya tabel; axe | F |
| WP-21 | **Credit & plan**: ledger grant/debit/refund idempotent dengan **satu fungsi debit** `billing.Debit` (dipakai analisis lokasi dan narasi AI), saldo per meter, grant bulanan otomatis sesuai plan, grant awal workspace baru (D-36), komponen saldo | BILL-01 | 10 | TC-BILL-02, TC-BILL-03; saldo = Σ ledger; debit ganda dengan `Idempotency-Key` sama ditolak | G |
| WP-22 | **Export CSV**: export sebagai job async di antrean `export`, maks. 10.000 baris, lewat `intel.ProspectView` (identitas, wilayah, website & sosial, kolom estimasi; **tanpa telepon**); unduh lewat URL bertanda tangan; UI tombol export di SCR-03/SCR-06 | ACT-03 | 17 | TC-ACT-02 (CSV tanpa kolom telepon, tanpa entri ledger), TC-ACT-05 (batas 10.000 baris); 10.000 baris ≤ 2 menit (NFR-02); export gagal bisa diulang; bagian J-3 | G |
| WP-23 | **Narasi AI**: antrean `ai`, narasi profil (`/prospects/{id}/brief`) dan analisis lokasi (`/location-analyses/{id}/brief`) dengan 5 bagian (masalah, data, cara membaca, kesimpulan, batasan), prompt `brief-2.0.0`; provider/model utama + fallback OpenAI-compatible dari setting AI (berlaku ≤ 60 s tanpa redeploy; API key hanya dari secret server); hanya fakta tanpa telepon (BR-AI-01), rujukan fakta per kalimat, cache 7 hari, kill switch, 1 AI credit (cache gratis); panel narasi di profil; halaman ops **Pengaturan AI** + Uji koneksi | ACT-05, OPS-01 | 17, 21, 24, 27 | TC-AI-01..04; ≤ 20 s; bagian J-7 | H |
| WP-24 | **Halaman ops**: layout `/ops`, daftar workspace + tetapkan plan + grant credit, **Plan** + grant awal, **status import DB gmaps** (`/ops/imports`), kesehatan data (% harga, % `popular_times`, % laju ulasan per metode A/B), API `/ops/settings` + tab **Parameter estimasi** (`est.*`, `estimate.Params.Validate()`, audit, enqueue `est.compute`), feature flag (AI, export), suppression list (bisnis/URL), tampilan audit log | OPS-01, OPS-02, PRIV-02, BILL-01, EST-06 (parameter) | 11, 15, 21, 26 | Akun non-ops → 403; TC-BILL-01, TC-EST-06 (API parameter), TC-OPS-01; flag off → fitur menolak; J-6 | H |
| WP-25 | **Monitoring & backup**: `/metrics` (listener internal, README) + `node-exporter` → Prometheus + dashboard Grafana + alert (email/Telegram), termasuk AL-10, AL-11, AL-12; pgBackRest WAL kontinu (repo lokal terenkripsi) + backup GitLab, disalin rclone tiap 5 menit ke SFTP/FTPS off-site di Indonesia (cron `msp-offsite`, 06 §8; salinan B2 nonaktif sampai Q-19); drill restore dicatat di [09](09-runbook-operasional.md) | NFR-03, NFR-04 | 06, D-39 | Restore **dari salinan off-site** ke instance Postgres kosong berhasil; umur WAL terakhir ≤ 15 menit di lokal dan off-site; alert uji (AL-10, AL-12) terkirim | H |
| WP-26 | **Estimasi**: paket `estimate` (Lampiran E `est-1.0.0`, fungsi murni): `Velocity` (metode A + B, cek silang, `short_window`; D-38), `Estimate` (`makan`/`kafe`/`hotel` + penanda FR-EST-05 + alasan BR-EST-01), `Pattern` (E.4 termasuk pola datar), pembulatan BR-EST-02, `Params.Validate()`; job `est.compute` (antrean `score`) yang menulis `estimates` dengan versi, asumsi, dan `data_as_of`, termasuk median harga kab/kota | EST-01..06 | 10 | TC-EST-01..07, GT-01 lapis U; coverage paket ≥ 80% | C |
| WP-27 | **Analisis lokasi (API)**: `estimate.AnalyzeLocation` (Lampiran F `loc-1.0.0`) + query tetangga PostGIS + `POST/GET /location-analyses`, `GET /location-analyses/{id}`, `PUT /location-analyses/{id}/feedback`; debit 1 credit lewat `billing.Debit` (tidak ditagih bila "Tidak dapat dianalisis"), `Idempotency-Key`; tersimpan dan dibuka ulang tanpa hitung ulang; RLS `location_analyses` | LOC-01, INTL-05, BILL-01, PRIV-01 | 11, 21, 26 | TC-LOC-01..06, GT-02 (U + I), TC-PRIV-01 (analisis); p95 ≤ 3 s (NFR-01) diukur dengan data nyata di staging | A |
| WP-28 | **Analisis lokasi (UI)**: SCR-13 (daftar tersimpan, form dengan peta + pencarian bisnis/wilayah, dialog biaya, hasil 9 bagian dengan chart perbandingan + tabel padanan, ubah nama/hapus/jalankan ulang, panel Narasi AI) + titik masuk dari sidebar, onboarding, profil, dan wilayah | LOC-01, INTL-05, ACT-05 (panel) | 18, 20, 23, 27 | E2E J-7; hasil tanpa pembanding tampil lengkap (TC-LOC-06); axe + keyboard; tidak ada debit tanpa dialog | F |

Catatan:
- Audit log (FR-OPS-02) ditulis oleh WP pemilik aksinya: 15 (login, undang), 22 (export), 23 (setting AI), 24 (grant credit, plan, parameter estimasi, flag, suppression), 27 (analisis lokasi).
- WP UI (18–20, 28) boleh mulai sejak WP-10 merged dengan mock dari contoh di kontrak; merge hanya setelah dependensinya merged dan E2E lulus.
- WP-11 dan WP-26 bisa jalan paralel: rumus estimasi diuji dengan input langsung, dan test integrasi `est.compute` men-seed `places`/`place_snapshots` sendiri.

### 3.1 Graf dependensi

```
WP-10 Kontrak F1  (setelah Gerbang F0)
 ├─► 11 Import DB gmaps ──┬─► 13 Wilayah (+12, +26) ─────────────────────────────┐
 ├─► 12 Geo-sync ─────────┘                                                     │
 ├─► 26 Estimasi ───────────────────────────────────────────────────────────────┤
 ├─► 21 Credit & plan ──────────────────────────────────────────────────────────┴─► 17 Search/profil API ─┬─► 19 Search UI & list (+18)
 ├─► 15 Auth ─┬─► 18 Shell + onboarding (+21)                                                            ├─► 20 Profil/wilayah UI (+18)
 │            └─► 24 Ops (+11, +21, +26)                                                                 ├─► 22 Export CSV
 │                                                                                                       └─► 23 Narasi AI (+24, +27)
 └─► 27 Analisis lokasi API (+11, +21, +26) ───────────────────────────► 28 Analisis lokasi UI (+18, +20, +23)
WP-06 (F0) ─► 25 Monitoring + backup
```

- **Paralel maksimum** setelah WP-10: 11, 12, 15, 21, 26 + 18 (mock) + 25 = 7 agent; slot ke-8 untuk WP perbaikan atau WP kontrak. Maks. 8 (D-22).
- **Jalur kritis:** 10 → 11/12/26 → 13 → 17 → 20 → 28 → Gerbang F1. WP-27 jalan paralel begitu 11, 21, dan 26 selesai. Agent yang selesai lebih dulu mengambil WP siap berikutnya, apa pun lane-nya.

### 3.2 Cek cakupan FR PRD §9

| Area | FR → WP |
|---|---|
| AUTH | 01 → 08, 15, 18 · 02 → 15 (+ seat), 18 · 03 → 15 |
| DATA | 01 → 11 (+ status di 24) · 02 → 11 · 03 → 12 · 04 → 13 |
| EST | 01 → 26, 17, 20 · 02 → 26, 17, 20 · 03 → 26, 17, 20 · 04 → 26, 17, 20, 28 · 05 → 26, 17, 20, 27 · 06 → 26 (+ parameter di 24) |
| LOC | 01 → 27, 28 |
| INTL | 01 → 11, 17, 20 · 03 → 13, 20 · 05 → 27, 28 |
| ONB | 01 → 18 |
| SRCH / PROF / GEO | SRCH-01 → 17, 19 · SRCH-02 → 17, 19 · PROF-01 → 17, 20 · GEO-01 → 13, 20 |
| ACT | 01 → 17, 19 · 03 → 22 · 05 → 23 (+ setting AI), 28 (panel) |
| BILL | 01 → 21 (ledger, debit, grant), 27 (debit analisis), 23 (debit AI), 18 (SCR-10), 24 (plan, grant manual), 15 (seat) |
| OPS | 01 → 24 (+ status import, parameter estimasi), 23 (Pengaturan AI) · 02 → 10 (append-only) + 15, 22, 23, 24, 27 (penulisan) |
| PRIV | 01 → 10 (+ test lintas workspace di tiap WP, `location_analyses` di 27) · 02 → 17 (ProspectView), 24 (daftar) |

**Hasil: 30/30 FR F1 tercakup.** FR di PRD §9.1 (F2) sengaja tidak dipetakan.

---

## 4. Gerbang F1

Kriteria = [PRD §12](01-prd.md#12-gerbang-rilis-f1), termasuk golden test **GT-01 Signa Cafe** dan **GT-02 Warung Kita** ([05 §4.1](05-test-strategy.md#41-golden-test-sample-case-founder-prd-12)); cek konkret dan bukti = checklist [05 §7](05-test-strategy.md#7-gerbang-rilis-f1). Orchestrator mengisi checklist itu untuk laporan gerbang.

**Demo founder** (di staging, data nyata dari DB gmaps):
1. Orchestrator menyerahkan laporan gerbang: checklist 05 §7 terisi (termasuk log GT-01 dan GT-02), link pipeline, daftar WP merged, isu terbuka, ringkasan cakupan data (Q-14), dan catatan tinjauan ambang §E.5 terhadap data nyata.
2. Founder daftar akun baru → verifikasi email → pilih tujuan → hasil pertama (diukur ≤ 5 menit). Grant awal 1.000 credit (D-36) membuat analisis lokasi bisa langsung dicoba.
3. Kompetitor & omzet: search → profil (perkiraan, pola keramaian, riwayat ulasan, evidence; tanpa telepon) → **analisis lokasi** di titik nyata pilihan founder (dengan dan tanpa pembanding) → narasi AI → simpan ke list + catatan → export CSV (cek tidak ada kolom telepon).
4. Halaman ops: status import DB gmaps (snapshot terakhir, jeda), ubah satu asumsi estimasi → estimasi berubah setelah hitung ulang, buat plan & tetapkan ke workspace, grant credit, ganti model di Pengaturan AI → narasi berikutnya memakai model baru tanpa redeploy, matikan flag AI → narasi ditolak.
5. Temuan → WP perbaikan; hanya item yang gagal diulang.
6. Founder menyetujui → tag `vX.Y.Z` → deploy production → onboarding customer beta.

---

## 5. Fase F2 — Make it right dan Fase F3 — Make it fast

Kandidat WP = isi F2/F3 di [PRD §6](01-prd.md#6-fase-rilis). Dipecah menjadi `WP-nn` oleh orchestrator setelah Gerbang F1, diurutkan berdasarkan feedback customer beta. Tiap fase punya gerbang sendiri yang disetujui founder.

**Backlog F2 yang sudah dirancang (D-40):** WP-14 opportunity & skor prioritas (Lampiran A, B), WP-16 ICP API + onboarding "Cari prospek", unlock kontak + tampilan telepon (bagian lama WP-21), webhook + guard SSRF dan export XLSX (bagian lama WP-22), tag & assign (bagian lama WP-19). Desainnya ada di PRD §9.1 dan 04 (tabel "Di luar F1"). Kandidat F2 lain: insight teks ulasan, pembanding BPS, perbandingan bebas antar bisnis dan wilayah, watch/alert perubahan dari `place_snapshots`, billing self-serve, error tracking.

F3 hanya dikerjakan bila pengukuran menunjukkan kebutuhannya.

---

## 6. Input yang dibutuhkan dari founder

Secret (key, password, URL koneksi) dimasukkan founder sebagai variabel CI terproteksi GitLab, tidak lewat chat atau repo.

| Input | Ref | Dipakai di | Dibutuhkan sebelum | Status (2026-10-15) |
|---|---|---|---|---|
| **1** dedicated server di Indonesia (56 core, 128 GB RAM, 1 TB) untuk production, staging, GitLab/CI, dan monitoring | Q-09, D-19, D-37 | WP-05 dan seterusnya | WP-05 dijalankan | ✓ Tersedia |
| Akun SFTP/FTPS off-site milik founder di Indonesia; kredensial diisi founder langsung ke `/opt/ops/rclone.conf` di server (06 §7), bukan variabel CI. Passphrase pgBackRest disimpan juga di password manager founder | D-14, D-39 | WP-05, WP-25 | Akhir Fase 0 | ✓ Tersedia; snapshot di sisi SFTP masih Q-20 |
| Domain + akses DNS | — | WP-05 (TLS Caddy), WP-08 (email) | WP-05 dijalankan | Belum diketahui |
| Layanan uptime monitor eksternal (akun + kanal alert) | D-26 | WP-05 | WP-05 selesai | Belum diketahui |
| SMTP (email verifikasi, undangan, alert) | — | WP-08, WP-15, WP-25 | WP-08 diuji di staging | Belum diketahui |
| Google OAuth client (ID, secret, redirect URI) | — | WP-15 | WP-15 mulai | Belum diketahui |
| GeoAPI key (opsional) | D-25 | WP-12 | WP-12 mulai | Opsional |
| Skema sumber `business_listings` | Q-13 | WP-10, WP-11 (`source.sql`, fixture 05 §6) | WP-11 mulai | ✓ Dari sampel record |
| **Kiriman pertama DB gmaps** (riwayat penuh, ramping, lewat statement `export` di `source.sql`) + **ukurannya** diukur dulu (2 × S ≤ ±300 GB, 04 §2.1); lalu kiriman mingguan | D-35, D-37 | WP-11 (uji di staging), WP-27 (NFR-01), Gerbang F1 | WP-11 diuji di staging (WP-11 sendiri dikerjakan dengan fixture) | Menunggu |
| **Ulasan terbaru dari scraper**: 20 ulasan terbaru per tempat, snapshot bertanda `reviews_sort = 'newest'`; kolom ini harus ada di DB gmaps sebelum `export` versi ini dijalankan | Q-18, D-38 | WP-11, WP-26 (metode B), laporan cakupan Gerbang F1 | Sebelum metode B diuji di staging | Keputusan ✓; rollout scraper menunggu |
| Cakupan data per kota (Q-14) | Q-14, PRD §13 | Pemilihan kota beta, tinjauan ambang §E.5 | Sebelum beta (diukur ulang) | ✓ Diukur sekali; ukur ulang sebelum beta |
| Nilai asumsi yang default-nya kosong (harga per orang per kelompok; hotel per kelas akomodasi), diisi ops lewat tab Parameter estimasi. Tanpa nilai ini estimasi terkait tampil "Belum dapat diperkirakan" | D-32, PRD §E.5 | WP-24, WP-26, demo | Demo Gerbang F1 | Opsional |
| API key Anthropic + API key OpenAI-compatible (fallback). Provider, model, dan base URL diatur kemudian di Pengaturan AI | D-15, D-29 | WP-23 | WP-23 mulai | Belum diketahui |

---

## 7. Risiko eksekusi

| Risiko | Mitigasi |
|---|---|
| Agent paralel bentrok di file bersama (kontrak, migrasi, layout UI) | Kontrak-first (WP-10), WP kontrak serial (R3), scope file per WP (R2), MR kecil + rebase |
| Isi atau skema DB gmaps berbeda dari dugaan, berubah, atau cakupannya rendah | Nama kolom hanya di `source.sql` + cek kolom fail-fast (TC-DATA-09); fixture mengikuti DDL asli (05 §6); import sampel nyata sedini mungkin di WP-11; cakupan diukur sebelum build (Q-14). Fitur tidak diubah untuk menutupi cakupan rendah (PRD §13) |
| Import pertama lama dan membebani `gmaps-db` serta Postgres kita di server yang sama | Jalan di jam sepi, bisa dilanjutkan dari cursor, satu import per environment (04 §5.1, 09 RB-03) |
| Agent menyimpang dari spec atau menambah fitur | Brief WP merujuk FR/BR; acceptance test ditulis dulu dari PRD; review orchestrator terhadap PRD/04; R6 |
| Test hijau tetapi lemah | Orchestrator mencocokkan test dengan acceptance PRD; golden test sample case founder (GT-01, GT-02) di setiap MR; coverage paket inti ≥ 80%; demo founder dengan data nyata |
| Input founder terlambat (server, akses DB gmaps, key) | Tabel §6 dengan batas "dibutuhkan sebelum"; WP yang tidak terblokir tetap jalan di lokal dengan fixture |
| Performa buruk pada data nyata | NFR-01/02 diukur di WP-17 (search, profil), WP-22 (export), dan WP-27 (analisis lokasi ≤ 3 s) dengan data nyata; optimasi hanya bila terukur (F3) |
| Satu server: antrean CI panjang (runner 1 job, 8 agent paralel) atau CI/staging membebani production | MR kecil, pipeline MR ≤ 15 menit ([05](05-test-strategy.md) §2), test ke staging ringan tanpa load test; ukur waktu antre & p95 production. Mitigasi lain dan pemisahan GitLab/CI bila contention terukur: PRD §13 |
