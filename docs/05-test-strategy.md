# 05 — Test Strategy (lean)

## MesinProspek — Apa yang diuji, di mana, dan kapan

| Field | Value |
|---|---|
| Versi | 0.2 (lean). Menggantikan v0.1 yang terlalu besar untuk MVP |
| Tanggal | 2026-10-10 |
| Sumber requirement | [`01-prd.md`](01-prd.md): FR/BR/NFR, §7 dataset, Lampiran A–D, §12 gerbang rilis. Dokumen ini **merujuk**, tidak menyalin rumus |
| Scope | **F1** dirinci. F2/F3: PRD §6, plus tambahan khusus test di §8 |
| Terkait | [`04`](04-technical-design.md) (nama tabel/endpoint), [`06`](06-ci-cd.md) (job pipeline), [`09`](09-runbook-operasional.md) (prosedur restore) |

> Konvensi: `TC-<AREA>-nn` = test case kritis, `EC-nn` = edge case data, `J-n` = journey E2E. Level: **U** = unit, **I** = integration, **E** = E2E. Dokumen ini menjelaskan **perilaku** yang diuji; nama tabel, endpoint, dan header mengikuti `04`.

---

## 1. Tujuan & prinsip

1. **Logika dulu.** Investasi terbesar di unit test logika inti: normalisasi, skor, opportunity, metrik wilayah, credit (NFR-09). Murah, cepat, deterministik.
2. **Test adalah kontrak untuk agent.** Setiap paket kerja (WP) menyebut ID `TC-`/`EC-` yang harus lulus. ID ditulis di test (`// TC-ACT-01` di Go, `test('TC-ACT-01 …')` di Playwright). Mengubah ekspektasi test wajib menyebut alasan + ID PRD di deskripsi MR.
3. **Setiap MR hijau sebelum merge.** Tidak ada `t.Skip`/`test.skip` tanpa issue. Tidak ada retry otomatis: test flaky = bug, diperbaiki, bukan diulang.
4. **Proporsional.** Banyak unit, cukup integration, 6 journey E2E. Tanpa mutation testing, golden set, soak test, atau load test di F1.
5. **Deterministik & tertutup.** CI tidak memanggil layanan luar: GeoAPI, LLM, penerima webhook, dan Google di-fake. Waktu (`clock`) dan ID acak di-inject.
6. **DRY.** Ekspektasi merujuk Lampiran A–D. Fixture dibangun dari satu builder record dengan default, tiap test hanya menimpa field yang relevan.

---

## 2. Level & tools

| Level | Tool | Yang diuji | Lokasi | Kapan |
|---|---|---|---|---|
| Unit (Go) | `go test -race`, **table-driven** | Aturan murni: BR-NORM, BR-SCORE, BR-OPP, BR-TRUST, Lampiran A–C, validator SSRF, HMAC, validator rujukan fakta AI | `apps/api/internal/<modul>/*_test.go` | Setiap MR |
| Unit (frontend) | **Vitest** | Util saja: format Rupiah/angka/tanggal id-ID (NFR-07), mask kontak, label band & claim type | `apps/web/**/*.test.ts` | Setiap MR |
| Integration | `go test -tags=integration` + **PostgreSQL 16 + PostGIS** sebagai GitLab CI `services:` | API (handler hasil oapi-codegen via `httptest`) + DB nyata: RLS, `percent_rank()`, point-in-polygon, ledger, job River (dijalankan inline), import end-to-end | `apps/api/internal/<modul>/*_integration_test.go` | Setiap MR |
| Kontrak API | oapi-codegen + generator client TS | `contracts/openapi.yaml` → kode Go & `packages/api-client` ter-generate ulang tanpa diff (`git diff --exit-code`) | CI job | Setiap MR |
| E2E | **Playwright** (Chromium di MR) | 6 journey kritis (tabel di bawah) di stack Docker Compose dengan dataset fixture | `apps/web/e2e/` | Setiap MR; smoke J-2 di staging setelah deploy `main` |
| Aksesibilitas | `@axe-core/playwright` di dalam journey E2E | Tiap halaman yang dikunjungi: 0 pelanggaran *serious/critical* (NFR-06). J2 juga dijalankan hanya dengan keyboard | `apps/web/e2e/` | Bersama E2E |

**Journey E2E**

| ID | Journey | FR |
|---|---|---|
| J-1 | Daftar → verifikasi email (Mailpit, hanya di stack test CI, jadi J-1 tidak jalan di staging) → wizard Offering + ICP → hasil pertama tampil | FR-AUTH-01, FR-ICP-01 |
| J-2 | Search kata kunci + filter → profil: skor + "Kenapa prospek ini?", opportunity, claim type tiap nilai, kontak ter-mask, tautan Google Maps | FR-SRCH-01, FR-PROF-01, FR-INTL-01 |
| J-3 | Unlock 1 prospek (saldo −1) → tambah ke list → export CSV dengan preview biaya → file terunduh | FR-ACT-01..03 |
| J-4 | Peringkat wilayah → tabel padanan peta tampil → drilldown ke daftar prospek (tiles di-intercept `page.route`) | FR-GEO-01, FR-INTL-03 |
| J-5 | Admin daftar webhook (penerima test) → kirim prospek terpilih → log pengiriman; buat AI brief (LLM fake) dengan rujukan fakta | FR-ACT-04, FR-ACT-05 |
| J-6 | Login ops + TOTP → buat plan & tetapkan ke workspace → grant credit → saldo workspace bertambah → matikan flag AI → tambah suppression → kontak tersembunyi | FR-AUTH-03, FR-OPS-01, FR-BILL-01, FR-PRIV-02 |

**Kapan jalan**

| Pemicu | Yang jalan | Target |
|---|---|---|
| Setiap MR | Lint, unit Go + Vitest, integration, cek kontrak API, coverage paket inti ≥ 80% (NFR-09), E2E Chromium + axe | ≤ 15 menit. Konkurensi E2E dibatasi di runner (detail di `06`) |
| Merge ke `main` | Semua di atas → deploy `staging` → smoke E2E **J-2** ke staging dengan akun test yang sudah terverifikasi | Gagal → revert MR penyebab |
| Tag `vX.Y.Z` (sebelum rilis) | E2E J-2..J-6 di staging (berurutan) dengan dataset nyata & akun test terverifikasi (Chromium, Firefox, WebKit + viewport ponsel untuk profil & list, NFR-08), cek ID TC/EC (§7), checklist gerbang §7 | Founder menyetujui |

**Staging satu host dengan production** (D-19). Test yang menyentuh staging wajib ringan: smoke dan E2E fungsional dijalankan berurutan (1 worker Playwright), pengukuran NFR memakai request berurutan (§7). **Tidak ada load test di staging pada F1**; load test baru di F3 di lingkungan terpisah (§8).

**Catatan teknis penting**
- **RLS hanya teruji bila koneksi test memakai role aplikasi**, bukan superuser/owner tabel (keduanya bypass RLS kecuali `FORCE ROW LEVEL SECURITY`).
- DB integration: `msp migrate` sekali ke database template, lalu tiap paket test membuat database dari template itu (cepat & terisolasi).
- Kratos di-fake di integration (respon *whoami* dengan status verifikasi & AAL). Kratos asli dipakai di E2E.

---

## 3. Cakupan per area

| ID | Yang diuji | Level | TC / EC |
|---|---|---|---|
| **AUTH** | | | |
| FR-AUTH-01 | Email belum terverifikasi → data 403. Login email/password & verifikasi via E2E. Login Google: cek manual di staging (§7) | I, E | TC-AUTH-01 |
| FR-AUTH-02 | Workspace dibuat saat daftar; undang anggota; matriks role Admin vs Member → 403; batas seat plan saat undang | I | TC-AUTH-02, TC-AUTH-04 |
| FR-AUTH-03 | Akun ops tanpa MFA (AAL1) ditolak | I, E | TC-AUTH-03 |
| **DATA** | | | |
| FR-DATA-01 | Upsert by `google_place_id`, idempotent, laporan baru/berubah/tidak berubah/ditolak per alasan, record mentah tersimpan sebagai evidence, `.jsonl` dan `.jsonl.gz`; filter negara `--country ID` (negara lain dilewati & dihitung `skipped_country`); import ulang file lebih baru | U, I | TC-DATA-01, TC-DATA-05, EC-07, EC-12 |
| FR-DATA-02 | BR-NORM-01..03, jam buka, rating — satu tabel kasus per aturan | U | TC-DATA-02..04, EC-01..06, EC-09, EC-10 |
| FR-DATA-03 | `msp tool geo-sync --country ID` dari rekaman GeoAPI: 4 level, populasi, atribusi, idempoten, throttle ≤ 120 req/menit | U, I | TC-GEO-01 |
| FR-DATA-04 | Point-in-polygon → rantai kelurahan → provinsi; tanpa geo / di luar semua poligon → "Tidak diketahui" (tanpa fallback desa terdekat). ≥ 98% dicek di staging | I | TC-GEO-02, EC-08 |
| **INTL** | | | |
| FR-INTL-01 | Claim type + sumber untuk setiap nilai, freshness (BR-TRUST-01..03) | U, I, E | TC-INTL-06, TC-INTL-07 |
| FR-INTL-02 | Tiap kode Lampiran B: kasus terpicu & tidak; relevansi per Offering; guard BR-OPP-01; teks alasan terisi | U | TC-INTL-01, TC-INTL-02 |
| FR-INTL-03 | Rumus Lampiran C, "Data terbatas", ambang kompetisi, label "Populasi: perkiraan model Kontur 2023" | U, I | TC-GEO-03, EC-11 |
| FR-INTL-04 | Lampiran A, BR-SCORE-01..03: kecocokan ICP (termasuk Sebagian), renormalisasi, coverage, band & cap, persentil di Postgres, determinisme per versi `priority-X.Y.Z` | U, I | TC-INTL-03..05, TC-ICP-02 |
| FR-INTL-05 | Feedback Relevan/Tidak relevan tersimpan per workspace, tidak bocor ke workspace lain | I | TC-PRIV-01 |
| **ICP** | | | |
| FR-ICP-01 | Wizard sampai hasil pertama | E | J-1 |
| FR-ICP-02 | Beberapa ICP per workspace, satu ICP aktif per user, ganti ICP → skor & urutan berubah | I | TC-ICP-01 |
| **SRCH / PROF / GEO** | | | |
| FR-SRCH-01 | Kata kunci + tiap filter + kombinasi; jumlah hasil = total item yang dipaginasi; urut skor stabil | I | TC-SRCH-01 |
| FR-SRCH-02 | Simpan pencarian → buka ulang → kata kunci & filter utuh, hasil dihitung ulang dari data terbaru (segment dinamis) | I | — |
| FR-PROF-01 | Semua bagian profil; kontak ter-mask sampai unlock; tautan Maps dari `hasMap`/`@id` | I, E | TC-INTL-06, J-2 |
| FR-GEO-01 | Peringkat kecamatan/kelurahan untuk ICP aktif (urutan Lampiran C), peta selalu punya tabel | I, E | J-4 |
| **ACT / BILL** | | | |
| FR-ACT-01 | List, tag, catatan, assign ke anggota workspace yang sama saja | I | — |
| FR-ACT-02 | Unlock 1 credit, permanen, idempotent, juga saat paralel | I | TC-ACT-01 |
| FR-ACT-03 | Preview biaya, "Unlock & export" menagih lewat unlock, export gagal bisa diulang tanpa biaya, tanpa tagih ganda, batas 10.000 baris | I, E | TC-ACT-02, TC-ACT-05 |
| FR-ACT-04 | Validasi URL (SSRF), HMAC, retry, log pengiriman, batas 100 prospek | U, I | TC-ACT-03..05 |
| FR-ACT-05 | Fakta saja + rujukan, tanpa telepon (BR-AI-01), fallback, cache 7 hari, kill switch, AI credit (Lampiran D); provider/model/fallback dari pengaturan ops berlaku ≤ 60 s, API key tidak pernah keluar lewat API | U, I | TC-AI-01..04 |
| FR-BILL-01 | Saldo = jumlah ledger; idempotency key; grant & refund oleh ops; plan dikelola ops; grant bulanan otomatis idempotent; batas seat | U, I | TC-ACT-01, TC-BILL-01, TC-BILL-02, TC-AUTH-04 |
| **OPS / PRIV** | | | |
| FR-OPS-01 | Hanya akun ops; grant credit, plan, pengaturan AI, riwayat import, kesehatan data (% terisi, % terpetakan), flag, suppression | I, E | TC-AUTH-03, TC-BILL-01, TC-AI-04, J-6 |
| FR-OPS-02 | Audit ditulis untuk tiap aksi; UPDATE/DELETE ditolak | I | TC-OPS-01 |
| FR-PRIV-01 | RLS lintas workspace + test katalog: setiap tabel data workspace ber-RLS | I | TC-PRIV-01 |
| FR-PRIV-02 | Suppression di profil, export, webhook | I | TC-PRIV-02 |
| **NFR** | | | |
| NFR-01, NFR-02 | Diukur sekali di staging sebelum rilis (§7), bukan di setiap MR | Manual + skrip | — |
| NFR-05 | RLS, SSRF, pemindaian secret di pipeline (`06`) | I, CI | TC-PRIV-01, TC-ACT-03 |
| NFR-06 | axe di setiap journey + J-2 keyboard-only; status tidak hanya warna (band punya teks) | E | J-1..6 |
| NFR-07 | Format id-ID & zona Asia/Jakarta | U (Vitest, Go) | — |
| NFR-09 | Coverage paket inti ≥ 80% (normalisasi, skor, opportunity, metrik wilayah, credit) | CI | — |

---

## 4. Test case kritis

| ID | Skenario | Ekspektasi | Level | Ref |
|---|---|---|---|---|
| TC-DATA-01 | Import file fixture dua kali; lalu import versi dengan `telephone` satu record diubah; import tanpa `--observed-at` atau tanpa `--country` | Run ke-2: 0 baru, 0 berubah, N tidak berubah, jumlah prospek tetap. Run ke-3: 1 berubah, evidence mentah ikut diperbarui. Tanpa `--observed-at`/`--country` → error, tidak ada yang ditulis | I | FR-DATA-01 |
| TC-DATA-02 | Normalisasi telepon: `0812-0000-0001`, `+62 812 0000 0001`, `62812…`, `(024) 831-2345`, `021 555 1234`, `12345` | Seluler → `+62812…` + "Kemungkinan WhatsApp" (Perkiraan); tetap → `+6224…` (Turunan, tanpa WhatsApp); `12345` → tidak valid, tidak dihitung kontak. Nilai mentah selalu tersimpan | U | BR-NORM-01 |
| TC-DATA-03 | Klasifikasi `url`: `linktr.ee/x`, `instagram.com/x`, `WWW.Instagram.com/x?hl=id`, `wa.me/62812…`, `x.wixsite.com/y`, `warunguji.co.id`, `""`, field tidak ada | aggregator, sosial, sosial, link pesan, subdomain gratis, domain sendiri, kosong, kosong. Pola dibaca dari file data di `data/` (bukan hard-code) | U | BR-NORM-02 |
| TC-DATA-04 | Kategori: `Restaurant` + `servesCuisine: Japanese`; `@type` tak dikenal; `@type` berupa array | F&B / Restoran Asia; "Lainnya" + dihitung "kategori tak terpetakan" di laporan import (tidak ditolak); array → tipe paling spesifik | U | BR-NORM-03 |
| TC-DATA-05 | File campuran record ID + negara lain (mis. SG, MY) + record tanpa `addressCountry` (geo di dalam / di luar bbox ID) di-import dengan `--country ID`; lalu file lebih baru (`--observed-at` lebih baru) dengan record ID yang sama, 1 di antaranya berubah, + 1 record baru | Run 1: hanya record ID (termasuk tanpa `addressCountry` dengan geo di dalam bbox ID) menjadi prospek; sisanya tidak ditulis dan dihitung `skipped_country` (bukan ditolak; aturan di `04`). Run 2: 1 baru, 1 berubah, sisanya tidak berubah, 0 duplikat; tanggal pengambilan record yang ada di file baru = `--observed-at` baru (BR-TRUST-02) | I | FR-DATA-01, D-28 |
| TC-GEO-01 | `msp tool geo-sync` ke server GeoAPI fake (rekaman), dijalankan dua kali | 4 level tersimpan dengan populasi & atribusi; run ke-2 tanpa duplikat; client tidak melebihi 120 req/menit (clock fake) | U, I | FR-DATA-03 |
| TC-GEO-02 | Poligon kotak buatan: titik di dalam kelurahan A, tepat di batas A/B, di luar semua poligon (termasuk 100 m dari tepi A) | A + kecamatan/kab/prov induknya; titik batas → tepat satu kelurahan (deterministik); di luar → tidak terpetakan (wilayah "Tidak diketahui", tanpa fallback desa terdekat), masuk hitungan "% terpetakan" | I | FR-DATA-04 |
| TC-GEO-03 | Metrik area: populasi 499 / 500, jumlah bisnis 4 / 5; persentil kompetisi 32,9 / 33 / 66 / 66,1 | <500 atau <5 → "Data terbatas", metrik tampil, `market_score` kosong; kompetisi Rendah/Sedang/Sedang/Tinggi; faktor Wilayah memakai kecamatan bila kelurahan terbatas | U | Lampiran C, A |
| TC-INTL-01 | Guard BR-OPP-01 untuk OPP-WEB-NONE: kelompok 20 prospek dengan 10 ber-`url`; 20 prospek dengan 9 ber-`url`; 19 prospek dengan 18 ber-`url` | 50% & ≥ 20 → opportunity dibuat untuk yang tanpa `url`; 45% → tidak dibuat; 19 prospek → tidak dibuat | U | BR-OPP-01 |
| TC-INTL-02 | `url` = linktr.ee untuk Offering `website_dev`, `social_media`, `reputation_mgmt` | Skor OPP-WEB-SOCIAL = severity × relevansi (1 / 0,6 / 0). Relevansi 0 → bukan opportunity, gap tetap tampil di kartu digital | U | Lampiran B |
| TC-INTL-03 | Faktor Popularitas & Wilayah tidak diketahui; lalu hitung ulang input yang sama | Faktor dikeluarkan, bobot dibagi ulang, `coverage` = 0,75; breakdown menandai faktor yang dikeluarkan; hasil identik untuk input + `priority-X.Y.Z` sama | U | BR-SCORE-01, FR-INTL-04 |
| TC-INTL-04 | Band pada skor 59, 60, 79, 80 | COLD, WARM, WARM, HOT | U | BR-SCORE-02 |
| TC-INTL-05 | Skor 85 dengan coverage 0,59; coverage 0,60; coverage 1 tapi kriteria ICP *wajib* tidak diketahui | WARM + "Data terbatas"; HOT; WARM + "Data terbatas" | U | BR-SCORE-02 |
| TC-INTL-06 | Buka profil prospek fixture lengkap & minim | Setiap nilai punya claim type + sumber, evidence bisa dibuka; email/person → "Tidak diketahui", tidak pernah kosong/"0"/"—" | I, E | FR-INTL-01, FR-PROF-01, BR-TRUST-01/03 |
| TC-INTL-07 | Umur data 180, 181, 365, 366 hari (clock fake) | Baru, Menua, Menua, Usang | U | BR-TRUST-02 |
| TC-ICP-01 | User ganti ICP aktif; ICP dengan kriteria *wajib* yang tidak cocok | Skor & urutan berubah sesuai ICP baru; prospek yang gagal kriteria wajib tidak tampil | I | FR-ICP-02, Lampiran A |
| TC-ICP-02 | ICP kategori Restoran Asia, min. rating 4,0, min. ulasan 20. Prospek: rating 4,0 / 3,7 / 3,6 / tidak ada; ulasan 20 / 10 / 9; kategori Restoran Asia / kategori lain dengan induk sama / Lainnya | Rating: Cocok / Sebagian / Tidak / Tidak diketahui; ulasan: Cocok / Sebagian / Tidak; kategori: Cocok / Sebagian / Tidak (Lainnya tidak pernah cocok) | U | Lampiran A |
| TC-SRCH-01 | Kata kunci + filter kategori, kelurahan, band, tipe opportunity, ada telepon, rating minimum | ID hasil = himpunan ekspektasi fixture; total = jumlah item seluruh halaman; urut skor turun dengan tie-break stabil | I | FR-SRCH-01 |
| TC-AUTH-01 | Session dengan email belum terverifikasi memanggil endpoint data | 403; setelah verifikasi → 200 | I | FR-AUTH-01 |
| TC-AUTH-02 | Member: undang anggota, kelola ICP, daftar webhook, lihat usage. Admin: aksi yang sama | Member → 403 untuk semua; Admin → 2xx | I | FR-AUTH-02 |
| TC-AUTH-03 | Akun ops session AAL1; ops AAL2 (TOTP); Admin workspace customer ke halaman/API ops | 403; 200; 403 | I, E | FR-AUTH-03, FR-OPS-01 |
| TC-AUTH-04 | Plan dengan 2 seat: workspace berisi Admin + 1 undangan tertunda mengundang orang ke-3; lalu undangan tertunda dibatalkan dan undang ulang | Ditolak `SEAT_LIMIT` tanpa undangan maupun email; setelah dibatalkan → 2xx | I | FR-AUTH-02, FR-BILL-01 |
| TC-PRIV-01 | User workspace A membaca/mengubah list, ICP, catatan, feedback, unlock, ledger, export, webhook milik B lewat ID; query langsung dengan konteks A | 404 untuk akses by ID; 0 baris milik B. Test katalog: tiap tabel data workspace punya RLS aktif + policy (menangkap tabel baru tanpa RLS) | I | FR-PRIV-01, NFR-05 |
| TC-PRIV-02 | Telepon prospek masuk suppression list setelah di-unlock | Tersembunyi di profil, tidak ada di file export dan payload webhook | I | FR-PRIV-02 |
| TC-ACT-01 | Unlock prospek yang sama dua kali berurutan dan 2 request paralel; saldo 0 | Satu debit saja, saldo = jumlah ledger; saldo 0 → ditolak tanpa entri ledger | I | FR-ACT-02, FR-BILL-01, BR-CRED-01 |
| TC-ACT-02 | "Unlock & export" 10 prospek (3 sudah di-unlock); lalu job export dibuat gagal (fake `storage` error) dan diulang | Preview 7 credit; konfirmasi → 7 debit + 7 unlock (via unlock); export gagal → tidak ada debit tambahan; ulang → sukses, tetap total 7 debit | I | FR-ACT-02/03, BR-CRED-01 |
| TC-ACT-03 | Daftar URL: skema `http://`; `https://` ke `localhost`, `127.0.0.1`, `2130706433` (desimal), `[::1]`, `10.x`, `172.16.x`, `192.168.x`, `169.254.169.254`, `0.0.0.0`, `[fd00::1]`; host publik yang resolve ke IP privat, host publik yang redirect 302 ke `169.254.169.254` | Semua ditolak saat daftar; IP dicek ulang saat koneksi (resolver fake); redirect tidak diikuti → pengiriman gagal tercatat | U, I | FR-ACT-04, NFR-05 |
| TC-ACT-04 | Penerima `httptest` membalas 500, 500, 200; lalu selalu 500; body diubah 1 byte | Signature HMAC-SHA256 valid; 3 percobaan tercatat lalu sukses; selalu 500 → berhenti di batas retry, status gagal; body diubah → verifikasi gagal | I | FR-ACT-04 |
| TC-ACT-05 | Bulk unlock & kirim ke webhook 100 vs 101 prospek; export 10.000 vs 10.001 baris | Tepat di batas → diterima; 101 / 10.001 → 422 tanpa debit maupun job | I | FR-ACT-02..04 |
| TC-AI-01 | Buat brief dengan LLM fake; fake membalas kalimat tanpa rujukan fakta, lalu valid | Request ke LLM hanya berisi fakta bisnis, tanpa telepon (mentah/E.164) dan nilai kontak; jawaban tanpa rujukan ditolak → dibuat ulang sekali; gagal lagi → error jelas | U, I | FR-ACT-05, BR-AI-01 |
| TC-AI-02 | Fake Claude membalas 5xx/timeout; lalu keduanya gagal | Request pindah ke fake OpenAI-compatible (base URL & model dari pengaturan AI), provider, model & versi `brief-X.Y.Z` tercatat; keduanya gagal → error, tanpa brief tersimpan | I | FR-ACT-05, D-15 |
| TC-AI-03 | Brief yang sama diminta lagi hari ke-6 dan hari ke-8 (clock fake); flag AI dimatikan | Hari ke-6: dari cache, 0 AI credit, 0 request ke LLM; hari ke-8: dibuat ulang, 1 AI credit; flag mati → ditolak tanpa panggilan LLM | I | FR-ACT-05, Lampiran D |
| TC-AI-04 | Ops mengubah model utama lewat API pengaturan AI; brief diminta sebelum dan sesudah t+61 s (clock fake) tanpa restart proses. Fallback dinonaktifkan lalu Claude fake gagal; diaktifkan lagi. Baca, simpan, dan Uji koneksi dengan API key fake bernilai unik. Admin workspace customer memanggil API pengaturan | Pada t+61 s request ke LLM fake memakai model baru (tanpa redeploy); fallback nonaktif → error tanpa panggilan ke fallback; aktif → fallback dipanggil dengan base URL & model dari pengaturan; nilai key tidak pernah muncul di body respons maupun audit (hanya status terpasang/belum); perubahan tercatat di audit; Admin → 403 | I | FR-ACT-05, FR-OPS-01, D-29 |
| TC-BILL-01 | Ops membuat, mengubah, dan menonaktifkan plan lalu menetapkannya ke workspace; Admin/Member customer memanggil API yang sama; nama kosong atau angka negatif | Ops → 2xx + entri audit; plan nonaktif tidak bisa ditetapkan (422), workspace yang sudah memakainya tetap; customer → 403; validasi gagal → 422 | I | FR-BILL-01, FR-OPS-01 |
| TC-BILL-02 | Job grant bulanan dijalankan 2× berurutan dan 2× paralel dalam siklus yang sama (clock fake), lalu di siklus berikutnya; nilai plan diubah di tengah siklus; workspace tanpa plan | Tepat 1 grant `credit` + 1 grant `ai_credit` per workspace per siklus sesuai nilai plan saat grant; siklus berikutnya → grant baru; perubahan plan terpakai di grant berikutnya; tanpa plan → tidak ada grant; saldo = jumlah ledger | I | FR-BILL-01 |
| TC-OPS-01 | Jalankan tiap aksi teraudit (login, undang, unlock, export, webhook, grant credit, plan, pengaturan AI, flag, suppression); coba UPDATE/DELETE audit dengan role aplikasi | Satu entri per aksi (aktor, workspace, waktu, target); UPDATE/DELETE → error izin | I | FR-OPS-02 |

---

## 5. Edge case data dari dataset

Setiap EC = satu baris di `edge-cases.jsonl` (§6), field `description` diawali ID-nya. Test import fixture memeriksa ekspektasi semua baris.

| ID | Kasus di record | Ekspektasi |
|---|---|---|
| EC-01 | `url` = `https://linktr.ee/…` | Link aggregator, tidak dihitung "punya website"; OPP-WEB-SOCIAL |
| EC-02 | `url` tidak ada vs `""` | Keduanya kosong; website "Tidak diketahui"; OPP-WEB-NONE hanya bila BR-OPP-01 terpenuhi |
| EC-03 | `url` = `https://wa.me/62812…` | Link pesan, bukan website; OPP-WEB-SOCIAL |
| EC-04 | `telephone` telepon tetap vs seluler, dengan spasi/tanda hubung/kurung | Tipe benar; hanya seluler yang "Kemungkinan WhatsApp"; faktor Kontak 60 vs 80 (Lampiran A) |
| EC-05 | `telephone` tidak ada; `sameAs` ada / tidak ada | Kontak "Tidak diketahui"; faktor Kontak 30 (hanya sosial) atau 0 |
| EC-06 | `aggregateRating` punya `ratingValue` tanpa `reviewCount` | Rating Teramati, jumlah ulasan "Tidak diketahui"; Popularitas dikeluarkan dari skor; OPP-REP-LOW-RATING tidak terpicu |
| EC-07 | `google_place_id` sama muncul 2× dalam satu file | Satu prospek, baris terakhir dipakai, laporan mencatat duplikat dalam file |
| EC-08 | `geo` tidak ada; record negara ID dengan koordinat di luar Indonesia (mis. Singapura) | Prospek tetap diimpor, wilayah "Tidak diketahui", tidak masuk metrik area; faktor Wilayah dikeluarkan |
| EC-09 | `latitude`/`longitude` berupa string (`"-6.99"`) | Diparse sebagai angka; hasil sama dengan versi numerik |
| EC-10 | `@type` tak dikenal (`"XyzStore"`) atau array `["Restaurant","LocalBusiness"]` | "Lainnya" + tercatat di laporan; array → `Restaurant` |
| EC-11 | Kelurahan dengan populasi < 500 | "Data terbatas", `market_score` tidak dihitung, faktor Wilayah memakai kecamatan |
| EC-12 | Baris JSON rusak (terpotong); record tanpa `google_place_id` | Baris ditolak dengan alasan + nomor baris; import lanjut ke baris berikutnya |
| EC-13 | Nama sama ("Warung Uji"), `google_place_id` berbeda | Dua prospek terpisah, tidak digabung (D-05) |

---

## 6. Data uji

| Jenis | Isi | Lokasi |
|---|---|---|
| Fixture dataset kecil | ±60 record sintetis berbentuk JSON-LD seperti PRD §7.1, mencakup 2 kab/kota × 2 kategori dengan ≥ 20 prospek per kelompok (agar BR-OPP-01 & persentil bisa diuji), + 3 record negara lain (filter `--country ID`) | `apps/api/testdata/dataset/prospects-small.jsonl` |
| Fixture edge case | Satu baris per `EC-nn` | `apps/api/testdata/dataset/edge-cases.jsonl` (+ versi `.gz`) |
| Builder record (Go) | Fungsi pembuat record dengan default valid; test menimpa field tertentu (DRY, tanpa generator terpisah) | paket test helper di `apps/api` |
| Rekaman GeoAPI | Respon nyata untuk satu kab/kota (kecamatan + kelurahan, populasi, batas disederhanakan), diputar ulang oleh server fake. Diperbarui manual dengan flag `-update` | `apps/api/testdata/geoapi/` |
| Poligon buatan | Kotak sederhana untuk TC-GEO-02 (lebih mudah dibaca daripada batas nyata) | di dalam test |
| Fake eksternal | `httptest` untuk GeoAPI, Claude, OpenAI-compatible, penerima webhook, Kratos *whoami*; resolver DNS fake untuk SSRF; Mailpit untuk email di E2E | di dalam test / stack E2E |

Contoh bentuk satu baris fixture (disesuaikan bila sample dataset nyata berbeda):

```json
{"@context":"https://schema.org","@type":"Restaurant","@id":"https://maps.google.com/?cid=1001","identifier":[{"@type":"PropertyValue","propertyID":"google_place_id","value":"TEST-PLACE-0001"}],"name":"Warung Uji Satu","description":"EC-01: url linktr.ee","servesCuisine":"Japanese","telephone":"0812-0000-0001","url":"https://linktr.ee/warunguji","sameAs":["https://instagram.com/warunguji"],"address":{"@type":"PostalAddress","streetAddress":"Jl. Contoh 1","addressLocality":"Kota Semarang","addressRegion":"Jawa Tengah","postalCode":"50132","addressCountry":"ID"},"geo":{"@type":"GeoCoordinates","latitude":-6.9932,"longitude":110.4203},"aggregateRating":{"@type":"AggregateRating","ratingValue":4.3,"reviewCount":57}}
```

**Aturan**
- Tidak ada data customer nyata (akun, ICP, list, catatan, unlock) di `local`, CI, atau `staging`. Tidak ada dump database produksi ke luar `production`.
- Fixture memakai nama, nomor (`+62812000…`), dan domain fiktif. `staging` boleh memakai **dataset bisnis** nyata (milik MSP, sesuai PRD §12) dengan akun test.
- CI tanpa akses jaringan ke layanan luar: semua `MSP_*_BASE_URL` diarahkan ke fake.

---

## 7. Gerbang rilis F1

**Satu-satunya checklist gerbang F1** (dokumen lain merujuk ke sini): satu baris per item PRD §12, masing-masing dengan bukti.

| # | Item PRD §12 | Cek konkret | Bukti |
|---|---|---|---|
| 1 | Alur end-to-end di staging dengan dataset nyata | `msp tool import --country ID --observed-at <tanggal>` + `msp tool geo-sync --country ID` + `msp tool rescore` di staging tanpa error, laporan import mencatat `skipped_country`; halaman kesehatan data ops menunjukkan ≥ 98% prospek ber-koordinat terpetakan (FR-DATA-04); E2E J-2..J-6 hijau di staging; walkthrough manual: login Google, Uji koneksi di Pengaturan AI OK, AI brief dengan Claude asli, webhook ke penerima milik tim | Link pipeline tag + checklist terisi |
| 2 | Setiap nilai di profil punya claim type & sumber | TC-INTL-06 + J-2 hijau; cek manual 10 profil acak di staging (termasuk yang datanya minim) | Checklist + screenshot |
| 3 | Test otomatis lulus (isolasi, SSRF, idempotensi credit) | Pipeline tag hijau; TC-PRIV-01, TC-ACT-01..03 ada dan lulus; coverage paket inti ≥ 80%; 0 test di-skip; **cek ID**: skrip kecil memastikan setiap `TC-`/`EC-` di dokumen ini ditemukan di kode test | Log job CI |
| 4 | Restore backup berhasil sekali | Restore pgBackRest ke instance Postgres terpisah, bukan database production (prosedur `09`), `/readyz` OK, jumlah baris tabel utama cocok; catat waktu (RTO ≤ 8 jam) dan titik pulih (RPO ≤ 15 menit) (NFR-04). Setelah itu bulanan per `09` | Catatan restore |
| 5 | Onboarding founder ≤ 5 menit | Founder daftar akun baru di staging; waktu dari submit daftar sampai hasil pertama tampil | Waktu tercatat |

Cek pendukung (sekali, di staging, sebelum tag; ringan sesuai §2): NFR-01 dengan skrip latensi berurutan, bukan load test (mis. 200 search + 200 profil acak, satu request sekaligus; p95 dari hasil skrip); NFR-02 dengan export 10.000 baris dan satu AI brief diukur waktunya; NFR-06 axe 0 pelanggaran *serious/critical* di semua journey.

---

## 8. Tambahan F2 / F3

Cakupan fitur: [PRD §6](01-prd.md). Tambahan khusus test:
- F2: E2E per fitur baru; set evaluasi kecil untuk kualitas AI brief & NL search; test diff antar import (ambang perubahan wajar); E2E semua browser di `main`.
- F3: load test dataset > 5 juta di lingkungan terpisah dari production; test deploy tanpa downtime (migrasi kompatibel mundur); cek regresi `EXPLAIN` untuk query utama.
