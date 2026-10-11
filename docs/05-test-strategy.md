# 05 — Test Strategy (lean)

## MesinProspek — Apa yang diuji, di mana, dan kapan

| Field | Value |
|---|---|
| Versi | 0.3 (lean). Fixture DB sumber (DB gmaps), estimasi, analisis lokasi, golden test sample case founder |
| Tanggal | 2026-10-11 |
| Sumber requirement | [`01-prd.md`](01-prd.md) v0.3: FR/BR/NFR, §7 data input, Lampiran A–F, §12 gerbang rilis. Dokumen ini **merujuk**, tidak menyalin rumus |
| Scope | **F1** dirinci. F2/F3: PRD §6, plus tambahan khusus test di §8 |
| Terkait | [`04`](04-technical-design.md) (nama tabel/endpoint, pemetaan kolom sumber), [`06`](06-ci-cd.md) (job pipeline), [`09`](09-runbook-operasional.md) (prosedur import & restore) |

> Konvensi: `TC-<AREA>-nn` = test case kritis, `GT-nn` = golden test sample case founder (PRD §12), `EC-nn` = edge case data, `J-n` = journey E2E. Level: **U** = unit, **I** = integration, **E** = E2E. Dokumen ini menjelaskan **perilaku** yang diuji; nama tabel, endpoint, dan header mengikuti `04`.

---

## 1. Tujuan & prinsip

1. **Logika dulu.** Investasi terbesar di unit test logika inti: normalisasi, estimasi, analisis lokasi, skor, opportunity, metrik wilayah, credit (NFR-09). Murah, cepat, deterministik.
2. **Test adalah kontrak untuk agent.** Setiap paket kerja (WP) menyebut ID `TC-`/`GT-`/`EC-` yang harus lulus. ID ditulis di test (`// GT-01` di Go, `test('J-7 …')` di Playwright). Mengubah ekspektasi test wajib menyebut alasan + ID PRD di deskripsi MR. **Ekspektasi GT hanya boleh berubah bila PRD §12 berubah** (keputusan founder).
3. **Setiap MR hijau sebelum merge.** Tidak ada `t.Skip`/`test.skip` tanpa issue. Tidak ada retry otomatis: test flaky = bug, diperbaiki, bukan diulang.
4. **Proporsional.** Banyak unit, cukup integration, 7 journey E2E, 2 golden test. Tanpa mutation testing, soak test, atau load test di F1.
5. **Deterministik & tertutup.** CI tidak memanggil layanan luar: DB gmaps asli diganti fixture DB sumber (§6); GeoAPI, LLM, penerima webhook, dan Google di-fake. Waktu (`clock`) dan ID acak di-inject.
6. **DRY.** Ekspektasi merujuk Lampiran A–F. Data uji = seed SQL sumber + satu builder baris snapshot dengan default; tiap test hanya menimpa field yang relevan.

---

## 2. Level & tools

| Level | Tool | Yang diuji | Lokasi | Kapan |
|---|---|---|---|---|
| Unit (Go) | `go test -race`, **table-driven** | Aturan murni: BR-NORM, BR-EST, BR-SCORE, BR-OPP, BR-TRUST, Lampiran A–C, E (estimasi), F (analisis lokasi), validator SSRF, HMAC, validator rujukan fakta AI | `apps/api/internal/<modul>/*_test.go` | Setiap MR |
| Unit (frontend) | **Vitest** | Util saja: format Rupiah/angka/tanggal id-ID dan jam lokal + label WIB/WITA/WIT (NFR-07, BR-NORM-06), format rentang perkiraan, mask kontak, label band, claim type & keyakinan | `apps/web/**/*.test.ts` | Setiap MR |
| Integration | `go test -tags=integration` + **PostgreSQL 16 + PostGIS** sebagai GitLab CI `services:` | API (handler hasil oapi-codegen via `httptest`) + DB nyata: RLS, `percent_rank()`, point-in-polygon, radius analisis lokasi, ledger, job River (dijalankan inline), import end-to-end dari **DB sumber fixture** (database kedua `gmaps_fixture` di service yang sama, 06), rantai hitung ulang, golden test | `apps/api/internal/<modul>/*_integration_test.go` | Setiap MR |
| Kontrak API | oapi-codegen + generator client TS | `contracts/openapi.yaml` → kode Go & `packages/api-client` ter-generate ulang tanpa diff (`git diff --exit-code`) | CI job | Setiap MR |
| E2E | **Playwright** (Chromium di MR) | 7 journey kritis (tabel di bawah) di stack Docker Compose dengan data fixture | `apps/web/e2e/` | Setiap MR; smoke J-2 di staging setelah deploy `main` |
| Aksesibilitas | `@axe-core/playwright` di dalam journey E2E | Tiap halaman yang dikunjungi: 0 pelanggaran *serious/critical* (NFR-06). J-2 juga dijalankan hanya dengan keyboard; setiap chart punya padanan tabel | `apps/web/e2e/` | Bersama E2E |

**Journey E2E**

| ID | Journey | FR |
|---|---|---|
| J-1 | Daftar → verifikasi email (Mailpit, hanya di stack test CI, jadi J-1 tidak jalan di staging) → tujuan **Kompetitor & omzet** → daftar bisnis dengan perkiraan omzet tampil. Akun kedua: tujuan **Cari prospek** → wizard Offering + ICP → hasil pertama | FR-AUTH-01, FR-ICP-01 |
| J-2 | Search kata kunci + filter kelompok estimasi & punya pola keramaian + urut perkiraan omzet → profil: panel perkiraan (rentang, sumber harga, parameter, "Data per", penanda), pola keramaian + padanan tabel, riwayat ulasan, claim type tiap nilai, kontak ter-mask, tautan Google Maps; dengan ICP aktif: skor + "Kenapa bisnis ini?" | FR-SRCH-01, FR-PROF-01, FR-EST-01..05, FR-INTL-01 |
| J-3 | Unlock 1 bisnis (saldo −1) → tambah ke list → tampilan Estimasi di list → export CSV dengan preview biaya → file terunduh dengan kolom estimasi | FR-ACT-01..03 |
| J-4 | Peringkat wilayah per kelompok estimasi → tabel padanan peta tampil → drilldown ke daftar bisnis dan ke form analisis lokasi (tiles di-intercept `page.route`) | FR-GEO-01, FR-INTL-03 |
| J-5 | Admin daftar webhook (penerima test) → kirim bisnis terpilih → log pengiriman; buat narasi AI profil (LLM fake) dengan 5 bagian dan rujukan fakta | FR-ACT-04, FR-ACT-05 |
| J-6 | Login ops + TOTP → buat plan & tetapkan ke workspace → grant credit → saldo workspace bertambah → ubah parameter estimasi → estimasi di profil berubah → status import (snapshot terakhir, jeda) tampil → matikan flag AI → tambah suppression → kontak tersembunyi | FR-AUTH-03, FR-OPS-01, FR-EST-06, FR-BILL-01, FR-PRIV-02 |
| J-7 | Dari profil "Analisis lokasi di sini" → ubah radius, kelompok, harga rencana, pembanding → dialog 1 credit → hasil (CI: data GT-02, kesimpulan sesuai §4.1) + tabel padanan chart → keluar, buka ulang dari daftar (saldo tetap) → narasi AI (LLM fake) → feedback Relevan | FR-LOC-01, FR-INTL-05, FR-ACT-05, FR-BILL-01 |

**Kapan jalan**

| Pemicu | Yang jalan | Target |
|---|---|---|
| Setiap MR | Lint, unit Go + Vitest, integration (termasuk GT-01, GT-02), cek kontrak API, coverage paket inti ≥ 80% (NFR-09), E2E Chromium + axe | ≤ 15 menit. Konkurensi E2E dibatasi di runner (detail di `06`) |
| Merge ke `main` | Semua di atas → deploy `staging` → smoke E2E **J-2** ke staging dengan akun test yang sudah terverifikasi | Gagal → revert MR penyebab |
| Tag `vX.Y.Z` (sebelum rilis) | E2E J-2..J-7 di staging (berurutan) dengan data nyata & akun test terverifikasi (Chromium, Firefox, WebKit + viewport ponsel untuk profil & list, NFR-08), cek ID TC/GT/EC (§7), checklist gerbang §7 | Founder menyetujui |

**Staging satu host dengan production** (D-19). Test yang menyentuh staging wajib ringan: smoke dan E2E fungsional dijalankan berurutan (1 worker Playwright), pengukuran NFR memakai request berurutan (§7). **Tidak ada load test di staging pada F1**; load test baru di F3 di lingkungan terpisah (§8). Di staging, assertion E2E bersifat struktural (bagian tampil, claim type, tanggal data); angka persis hanya diuji di CI lewat fixture. J-7 di staging memakai 1 credit per jalan dari workspace test (credit diberi ops).

**Catatan teknis penting**
- **RLS hanya teruji bila koneksi test memakai role aplikasi**, bukan superuser/owner tabel (keduanya bypass RLS kecuali `FORCE ROW LEVEL SECURITY`).
- **DB sumber fixture dibaca dengan role read-only** (hanya `SELECT`), sama seperti akun DB gmaps di production (NFR-05). Helper test membuat database `gmaps_fixture`, menjalankan `schema.sql` + seed yang dibutuhkan (§6), lalu mengarahkan `MSP_SOURCE_DATABASE_URL` ke role itu.
- DB integration: `msp migrate` sekali ke database template, lalu tiap paket test membuat database dari template itu (cepat & terisolasi).
- Kratos di-fake di integration (respon *whoami* dengan status verifikasi & AAL). Kratos asli dipakai di E2E.

---

## 3. Cakupan per area

| ID | Yang diuji | Level | TC / GT / EC |
|---|---|---|---|
| **AUTH** | | | |
| FR-AUTH-01 | Email belum terverifikasi → data 403. Login email/password & verifikasi via E2E. Login Google: cek manual di staging (§7) | I, E | TC-AUTH-01 |
| FR-AUTH-02 | Workspace dibuat saat daftar; undang anggota; matriks role Admin vs Member → 403; batas seat plan saat undang | I | TC-AUTH-02, TC-AUTH-04 |
| FR-AUTH-03 | Akun ops tanpa MFA (AAL1) ditolak | I, E | TC-AUTH-03 |
| **DATA** | | | |
| FR-DATA-01 | Import inkremental dari DB sumber (read-only): import pertama membaca seluruh riwayat, upsert per ID tempat, riwayat per tempat per hari, import ulang rentang sama tanpa duplikat, filter negara, laporan (snapshot baru, tempat baru, berubah, dilewati, ditolak per alasan), cek kolom sumber, kolom yang tidak diimpor | U, I | TC-DATA-01, TC-DATA-05, TC-DATA-09, TC-DATA-12, EC-07, EC-12 |
| FR-DATA-02 | BR-NORM-01..06: telepon, URL & sosial, kategori dwibahasa, harga, jam buka, `popular_times`, distribusi bintang, zona waktu lokal, penggabungan snapshot, atribut & pemilik | U, I | TC-DATA-02..04, TC-DATA-06..08, TC-DATA-10, TC-DATA-11, EC-01..06, EC-09, EC-10, EC-14, EC-16 |
| FR-DATA-03 | `msp tool geo-sync --country ID` dari rekaman GeoAPI: 4 level, populasi, atribusi, idempoten, throttle ≤ 120 req/menit | U, I | TC-GEO-01 |
| FR-DATA-04 | Point-in-polygon → rantai kelurahan → provinsi; tanpa koordinat / di luar semua poligon → "Tidak diketahui" (tanpa fallback desa terdekat). ≥ 98% dicek di staging | I | TC-GEO-02, EC-08 |
| **EST** | | | |
| FR-EST-01 | Laju ulasan (Lampiran E §E.1): jendela 365/90 hari, syarat 30 hari, tidak negatif, tanpa tebakan umur usaha; riwayat untuk chart | U, I | TC-EST-01, EC-15, GT-01 |
| FR-EST-02 | Kunjungan & omzet `makan`/`kafe` (§E.2): rentang, tamu per hari buka, urutan sumber harga, sensitivitas, pembulatan BR-EST-02 | U, I | TC-EST-02, GT-01 |
| FR-EST-03 | Hotel (§E.3): okupasi > 100%, parameter kosong, kelas akomodasi | U | TC-EST-03, EC-17 |
| FR-EST-04 | Pola keramaian (§E.4): rata-rata & per hari, blok, puncak, jam inti, pola datar, keramaian rata-rata, tamu per jam, label wajib | U, E | TC-EST-05, GT-01, GT-02, J-2 |
| FR-EST-05 | Penanda bintang 5 dan laju berubah, di profil dan analisis lokasi | U, I | TC-EST-04, GT-01 |
| FR-EST-06 | Hitung ulang setelah import & perubahan parameter; versi, parameter, tanggal snapshot tersimpan; determinisme | I | TC-EST-06 |
| **LOC** | | | |
| FR-LOC-01 | Lampiran F: tetangga & data cukup, pola tetangga, jam inti & saran jam buka, pembanding, posisi harga, sumber pengunjung siang, keyakinan; simpan, buka ulang, credit | U, I, E | TC-LOC-01..06, GT-02, J-7 |
| **INTL** | | | |
| FR-INTL-01 | Claim type + sumber + tanggal data untuk setiap nilai, termasuk estimasi dan analisis lokasi; freshness (BR-TRUST-01..03) | U, I, E | TC-INTL-06, TC-INTL-07 |
| FR-INTL-02 | Tiap kode Lampiran B: kasus terpicu & tidak; relevansi per Offering; guard BR-OPP-01; teks alasan terisi | U | TC-INTL-01, TC-INTL-02 |
| FR-INTL-03 | Rumus Lampiran C per kelompok estimasi (kelompok `lainnya`: semua bisnis), "Data terbatas", ambang kompetisi, permintaan terbukti, rentang harga, label "Populasi: perkiraan model Kontur 2023" | U, I | TC-GEO-03, TC-GEO-04, EC-11 |
| FR-INTL-04 | Lampiran A, BR-SCORE-01..03: kecocokan ICP (termasuk Sebagian), renormalisasi, coverage, band & cap, persentil di Postgres, determinisme per versi `priority-X.Y.Z` | U, I | TC-INTL-03..05, TC-ICP-02 |
| FR-INTL-05 | Feedback Relevan/Tidak relevan pada opportunity dan kesimpulan analisis lokasi tersimpan per workspace, tidak bocor ke workspace lain | I, E | TC-PRIV-01, J-7 |
| **ICP** | | | |
| FR-ICP-01 | Onboarding per tujuan sampai hasil pertama | E | J-1 |
| FR-ICP-02 | Beberapa ICP per workspace, satu ICP aktif per user, ganti ICP → skor & urutan berubah | I | TC-ICP-01 |
| **SRCH / PROF / GEO** | | | |
| FR-SRCH-01 | Kata kunci + tiap filter (termasuk kelompok estimasi, rentang omzet, punya pola keramaian) + kombinasi; urut skor / kunjungan / omzet / laju ulasan; jumlah hasil = total item yang dipaginasi | I | TC-SRCH-01 |
| FR-SRCH-02 | Simpan pencarian → buka ulang → kata kunci & filter utuh, hasil dihitung ulang dari data terbaru (segment dinamis) | I | — |
| FR-PROF-01 | Semua bagian profil: panel perkiraan (dan varian hotel), pola keramaian + tabel, riwayat ulasan, harga, jam buka, atribut, penanda listing diklaim, tombol analisis lokasi; kontak ter-mask sampai unlock; tautan Maps dari `cid` | I, E | TC-INTL-06, TC-DATA-11, J-2 |
| FR-GEO-01 | Peringkat kecamatan/kelurahan per kelompok estimasi (urutan Lampiran C), peta selalu punya tabel, drilldown ke bisnis dan analisis lokasi | I, E | TC-GEO-04, J-4 |
| **ACT / BILL** | | | |
| FR-ACT-01 | List, tag, catatan, assign ke anggota workspace yang sama saja; tampilan Estimasi di list | I, E | J-3 |
| FR-ACT-02 | Unlock 1 credit, permanen, idempotent, juga saat paralel | I | TC-ACT-01 |
| FR-ACT-03 | Preview biaya, export tanpa unlock (telepon ter-mask, gratis), "Unlock & export" menagih lewat unlock, export gagal bisa diulang tanpa biaya, tanpa tagih ganda, batas 10.000 baris, kolom estimasi | I, E | TC-ACT-02, TC-ACT-05 |
| FR-ACT-04 | Validasi URL (SSRF), HMAC, retry, log pengiriman, batas 100 bisnis | U, I | TC-ACT-03..05 |
| FR-ACT-05 | Narasi profil dan analisis lokasi: 5 bagian, fakta saja + rujukan, tanpa telepon (BR-AI-01), fallback, cache 7 hari, kill switch, AI credit (Lampiran D); provider/model/fallback dari pengaturan ops berlaku ≤ 60 s, API key tidak pernah keluar lewat API | U, I | TC-AI-01..04 |
| FR-BILL-01 | Saldo = jumlah ledger; idempotency key; grant & refund oleh ops; plan dikelola ops; grant bulanan otomatis idempotent; batas seat; debit analisis lokasi (tidak ditagih bila "Tidak dapat dianalisis") | U, I | TC-ACT-01, TC-LOC-04, TC-BILL-01, TC-BILL-02, TC-AUTH-04 |
| **OPS / PRIV** | | | |
| FR-OPS-01 | Hanya akun ops; grant credit, plan, pengaturan AI, parameter estimasi, status import (snapshot terakhir, jeda, laporan), kesehatan data (% terisi termasuk harga & `popular_times`, % riwayat ≥ 30 hari, % terpetakan), flag, suppression | I, E | TC-AUTH-03, TC-BILL-01, TC-AI-04, TC-EST-06, TC-DATA-01, J-6 |
| FR-OPS-02 | Audit ditulis untuk tiap aksi (termasuk parameter estimasi); UPDATE/DELETE ditolak | I | TC-OPS-01 |
| FR-PRIV-01 | RLS lintas workspace (termasuk analisis lokasi tersimpan) + test katalog: setiap tabel data workspace ber-RLS | I | TC-PRIV-01 |
| FR-PRIV-02 | Suppression di profil, export, webhook | I | TC-PRIV-02 |
| **NFR** | | | |
| NFR-01, NFR-02 | Search, profil, analisis lokasi (p95 ≤ 3 s), export, narasi, durasi import harian: diukur sekali di staging sebelum rilis (§7), bukan di setiap MR | Manual + skrip | — |
| NFR-05 | RLS, SSRF, akun read-only ke DB sumber, pemindaian secret di pipeline (`06`) | I, CI | TC-PRIV-01, TC-ACT-03, TC-DATA-01 |
| NFR-06 | axe di setiap journey + J-2 keyboard-only; status tidak hanya warna; setiap chart punya padanan tabel | E | J-1..7 |
| NFR-07 | Format id-ID, waktu sistem Asia/Jakarta, jam buka & pola dalam zona lokal tempat | U (Vitest, Go) | TC-DATA-10 |
| NFR-09 | Coverage paket inti ≥ 80% (normalisasi, estimasi, analisis lokasi, skor, opportunity, metrik wilayah, credit) | CI | — |

---

## 4. Test case kritis

| ID | Skenario | Ekspektasi | Level | Ref |
|---|---|---|---|---|
| TC-DATA-01 | Seed DB sumber fixture lalu `msp tool import --country ID` dua kali; tambah snapshot dengan waktu scrape lebih baru lalu import lagi (sama dengan job `import.run`); lalu `--reset` | Run 1 membaca seluruh riwayat: tempat baru = jumlah tempat ID, snapshot baru = jumlah baris ID. Run 2: 0 snapshot baru, 0 tempat baru, jumlah bisnis & baris riwayat tetap. Run 3: hanya baris baru yang dibaca, tempat yang berubah dihitung "berubah". `--reset` membaca ulang seluruh riwayat → 0 duplikat bisnis maupun riwayat, lalu run berikutnya lanjut dari cursor. Koneksi sumber hanya `SELECT`. Isi `user_reviews` dan `status` tidak tersalin ke tabel MSP maupun evidence. Status import di `/ops/imports` (snapshot terakhir, jeda, laporan) ikut diperbarui | I | FR-DATA-01, NFR-05, PRD §7.1 |
| TC-DATA-02 | Normalisasi `phone`: `0812-0000-0001`, `+62 812 0000 0001`, `62812…`, `(024) 831-2345`, `021 555 1234`, `12345` | Seluler → `+62812…` + "Kemungkinan WhatsApp" (Perkiraan); tetap → `+6224…` (Turunan, tanpa WhatsApp); `12345` → tidak valid, tidak dihitung kontak. Nilai mentah selalu tersimpan | U | BR-NORM-01 |
| TC-DATA-03 | Klasifikasi `website`: `linktr.ee/x`, `instagram.com/x`, `WWW.Instagram.com/x?hl=id`, `wa.me/62812…`, `x.wixsite.com/y`, `warunguji.co.id`, `""`, NULL. `social_links` `{"instagram": "…"}` | aggregator, sosial, sosial, link pesan, subdomain gratis, domain sendiri, kosong, kosong. Akun sosial dari `social_links` tampil sebagai Instagram. Pola dibaca dari file data di `data/` (bukan hard-code) | U | BR-NORM-02 |
| TC-DATA-04 | Kategori dwibahasa: `category` "Restaurant" vs "Restoran"; "Cafe", "Kafe", "Coffee shop"; "Hotel", "Hotel bintang 3", "Resor", "Vila", "Penginapan"; "Klinik", "Apotek"; `category` "Restoran" dengan `categories` berisi "Kafe"; `category` "Xyz" dengan `categories` ["Xyz", "Kafe"]; "Xyz" tanpa label lain yang dikenal | Restaurant = Restoran (kategori MSP & kelompok `makan` sama); kafe → `kafe`; akomodasi → `hotel` + kelas akomodasi (bintang 3, resor, vila, penginapan, hotel umum); klinik & apotek → non-F&B jenis kesehatan (Lampiran F); "Restoran" + "Kafe" → tetap `makan` (kategori utama menentukan), "Kafe" tampil sebagai kategori tambahan; kategori utama tak dikenal → label pertama yang dikenal di daftar (`kafe`); tidak ada yang dikenal → "Lainnya" + peringatan `unmapped_category` di laporan (tidak ditolak). Mapping dibaca dari `data/categories.yaml` | U | BR-NORM-03 |
| TC-DATA-05 | Sumber berisi tempat `address_country` ID, SG, MY, dan kosong (koordinat di dalam / di luar bbox ID), di-import dengan `--country ID`; lalu snapshot lebih baru untuk tempat ID yang sama (1 berubah) + 1 tempat baru | Run 1: hanya tempat ID (termasuk yang kosong dengan koordinat di dalam bbox ID; aturan di 04) menjadi bisnis; sisanya tidak ditulis dan dihitung "dilewati (negara lain)" (bukan ditolak). Run 2: 1 tempat baru, 1 berubah, 0 duplikat; freshness tempat yang punya snapshot baru = tanggal snapshot itu (BR-TRUST-02) | I | FR-DATA-01, D-28 |
| TC-DATA-06 | Harga per orang: "Rp 25–50K", "Rp 75–200K", "Rp 75-200K" (tanda hubung biasa), "Rp 100–225 rb", "Rp 1–25.000", "Rp 250,000+", "$", "$$", "murah", "", NULL | 25.000–50.000; 75.000–200.000 (dua-duanya); 100.000–225.000; 1–25.000; min 250.000 tanpa max; sisanya "Tidak diketahui" (peringatan `unparsed_price` di laporan, baris tidak ditolak) | U | BR-NORM-04, PRD §7.1 |
| TC-DATA-07 | `open_hours` kunci hari Indonesia: "08.00–23.00", "17.00–02.00", "Tutup", format tak dikenal. `popular_times` kunci hari Inggris → jam "6".."23": lengkap; satu hari semua 0; satu nilai 101 / −1 / bukan angka. `reviews_per_rating` `{"1":48,…,"5":1319}` | Jadwal Senin–Minggu; lewat tengah malam → tutup 02.00 hari berikutnya; "Tutup" → tutup; tak dikenal → hari itu "Tidak diketahui". Matriks 7 × 24 (jam yang tidak ada = 0); hari semua 0 → tutup; nilai di luar 0–100 → field `popular_times` ditolak (Tidak diketahui) + peringatan `invalid_popular_times`, field lain tetap masuk. Distribusi bintang + persen per bintang | U | BR-NORM-06, FR-DATA-02 |
| TC-DATA-08 | Satu tempat dengan snapshot: (a) harga hanya di snapshot tertua; (b) telepon kosong di snapshot terbaru tetapi ada di sebelumnya; (c) dua snapshot di hari yang sama (09.00 dan 21.00); (d) kategori "Restaurant" lalu "Restoran" | (a) harga terkini = nilai tertua dengan tanggal snapshot-nya sendiri (freshness dari tanggal itu); (b) telepon = nilai sebelumnya dengan tanggalnya; (c) riwayat hanya 1 baris untuk hari itu, dari snapshot 21.00; riwayat menyimpan jumlah ulasan, rating, distribusi bintang, harga; (d) kelompok tetap `makan`. Field lain dari snapshot terbaru | U, I | BR-NORM-05 |
| TC-DATA-09 | Skema sumber berubah: kolom `price_range` di-rename di fixture; lalu kolom tambahan tak dikenal ditambahkan | Rename → import gagal **saat mulai** dengan pesan `source schema mismatch: <detail>; perbaiki apps/api/internal/ingest/gmaps/source.sql`, 0 baris ditulis, status `failed` (dasar alert 09 AL-10), data lama tetap dipakai. Kolom tambahan → diabaikan, import jalan. Kolom yang diperiksa = yang dipakai `source.sql` | I | FR-DATA-01, PRD §13 |
| TC-DATA-10 | Zona waktu lokal: Signa Cafe (`timezone` `Asia/Makassar`, jam buka "08.00–23.00", puncak jam 20) dan tempat `Asia/Jakarta` serta `Asia/Jayapura` dengan nilai jam yang sama | Jam ditafsirkan sebagai waktu lokal, tanpa konversi: API & UI menampilkan "08.00–23.00 WITA" dan puncak "20.00 WITA" (bukan 07.00–22.00 WIB); Jakarta → WIB; Jayapura → WIT. Jam inti, tamu per jam, dan saran jam buka analisis lokasi memakai jam lokal yang sama. Waktu sistem ("Data per", log) tetap Asia/Jakarta | U, E | BR-NORM-06, NFR-07 |
| TC-DATA-11 | `about` berisi grup atribut (populer untuk: Sarapan, Makan siang, Makan malam; tipe pengunjung: Turis); `owner` terisi vs NULL | Profil menampilkan atribut sebagai Teramati per grup. `owner` terisi → "Listing diklaim pemilik"; NULL → "Tidak diketahui". Hanya penanda yang dipakai; isi `owner` tidak ditampilkan | U, I | FR-PROF-01, PRD §7.1 |
| TC-DATA-12 | Satu import per environment: jalankan `msp tool import --country ID` saat import lain sedang memegang kunci (kunci dipegang test); lalu job harian `import.run` terpicu saat kunci dipegang; lalu kunci dilepas | CLI berhenti dengan pesan jelas "import sedang berjalan" tanpa membaca atau menulis apa pun; job harian dilewati (bukan gagal, tidak dicoba ulang terus); setelah kunci dilepas import berjalan normal dan melanjutkan dari cursor | I | FR-DATA-01, 04 §5.1 |
| TC-EST-01 | Laju ulasan, pasangan (hari, ulasan): (t−29, 100)–(t, 110); (t−30, 100)–(t, 110); (t−60, 1.200)–(t, 1.190); (t−400, 50), (t−200, 100), (t, 200); hanya 1 snapshot; snapshot (t−200), (t−20), (t) untuk laju terkini | "Belum cukup riwayat"; 10 ÷ 30 × 365 ≈ 121,7; laju 0 (tidak negatif); memakai t−200 (t−400 di luar 365 hari) → 182,5; "Belum cukup riwayat"; laju terkini rentang 20 hari < 30 → Tidak diketahui, penanda laju berubah tidak dievaluasi. Tidak ada jalur kode yang memakai umur usaha | U | Lampiran E §E.1, FR-EST-01, BR-EST-01 |
| TC-EST-02 | Kelompok `makan`, laju 100/tahun: (a) harga sendiri "Rp 25–50K"; (b) tanpa harga, 10 bisnis `makan` ber-harga di kab/kota yang sama; (c) seperti (b) tetapi 9 bisnis, asumsi harga ops `makan` Rp50.000–70.000; (d) seperti (c) tanpa asumsi harga. Variasi: buka 6 hari/minggu; jam buka tidak diketahui; harga "Rp 250,000+"; kelompok `lainnya` | Kunjungan 5.000 / 6.700 / 10.000 (BR-EST-02). (a) belanja 30.000 → omzet Rp150 jt / Rp200 jt / Rp300 jt, sumber "harga tempat ini"; (b) median titik tengah 10 bisnis, sumber menyebut median kab/kota; (c) titik tengah 60.000 → belanja 48.000 → Rp240 jt / Rp320 jt / Rp480 jt, sumber "asumsi harga" (`benchmark`); (d) omzet "Belum dapat diperkirakan — harga tidak diketahui", kunjungan tetap tampil. 6 hari → tamu per hari = kunjungan ÷ (365 × 6/7); jam buka tidak diketahui → 7 hari; rentang terbuka → titik tengah = 250.000; `lainnya` → "Belum dapat diperkirakan" + alasan kategori. Tabel sensitivitas berisi 3 baris review rate. Catatan metode pesan-antar selalu ada di respons | U, I | Lampiran E §E.2, BR-EST-01/02 |
| TC-EST-03 | Hotel kelas `hotel_3` (asumsi: lama menginap 2 malam, 20 kamar, ADR Rp500.000; review rate hotel 6,7% / 10% / 13,3%): laju 100 dan 500; lalu ADR kosong; jumlah kamar kosong; lama menginap kosong; kategori "Resor" tanpa asumsi kelas `resort` | Laju 100 → kunjungan menginap 800 / 1.000 / 1.500 (rendah = laju ÷ 13,3%), room-night (tengah) 2.000, okupasi (tengah) 27,4%, omzet kamar Rp750 jt / Rp1,0 M / Rp1,5 M. Laju 500 → kunjungan 3.800 / 5.000 / 7.500, room-night 10.000, okupasi 137% → okupasi **tidak ditampilkan** + `occupancy_inconsistent`; kunjungan, room-night, dan omzet kamar (Rp3,8 / 5,0 / 7,5 M) tetap tampil. ADR kosong → omzet kamar Belum dapat diperkirakan; kamar kosong → okupasi Belum dapat diperkirakan; lama menginap kosong → room-night, okupasi, omzet kamar Belum dapat diperkirakan, kunjungan menginap tetap; resor → `hotel_params_missing` | U | Lampiran E §E.3, §E.5, FR-EST-03 |
| TC-EST-04 | Distribusi bintang 5 = 84,9% / 85,0% / tidak diketahui; laju terkini vs setahun +30,0% / +30,1% / −30,1% / laju terkini tidak diketahui | Bintang 5: tidak / `five_star_heavy` (teks netral 03 §7) / tidak. Laju: tidak / `velocity_changed` / `velocity_changed` / tidak. Penanda ada di respons profil dan di baris tetangga analisis lokasi | U, I | FR-EST-05, §E.5 |
| TC-EST-05 | Pola KR dari GT-02 (jam 11–21, sama tiap hari, buka 11.00–22.00); variasi: dua jam dengan nilai puncak sama; satu hari semua 0; jam inti 8 dari 10 jam buka vs 11 dari 14; jam buka tidak diketahui; kunjungan diketahui vs tidak | Puncak jam 19 (73), jam inti 17–21, bentuk Siang 23% · Sore 23% · Malam 53% · Pagi & Larut 0%, keramaian rata-rata ≈ 47% (46,5), 5 dari 11 jam buka → bukan pola datar. Puncak seri → jam paling awal. Hari semua 0 → "Tutup", tidak ikut rata-rata 7 hari. 80% → "Ramai sepanjang hari", 78,6% → tidak. Jam buka tidak diketahui → keramaian rata-rata & pola datar "Tidak diketahui". Tamu per jam hanya bila kunjungan diketahui (rumus §E.4). Respons selalu memuat label "relatif terhadap jam tersibuk, bukan jumlah orang" | U | Lampiran E §E.4, FR-EST-04 |
| TC-EST-07 | Wilayah siap: fixture berisi tempat di Badung (siap) dan di kab/kota lain (tidak siap) dengan riwayat ≥ 30 hari; ops menambah kab/kota itu ke `est.ready_regions` | Sebelum: tempat di luar wilayah siap tanpa estimasi, `reason = region_not_ready`, tidak ikut metrik kawasan dan peringkat wilayah; profil, pola keramaian, dan analisis lokasi tetap jalan. Sesudah: estimasi dan metrik kawasan dihitung ulang dan muncul; perubahan tercatat di audit | I | BR-EST-04, FR-OPS-01 |
| TC-EST-06 | Hitung estimasi dua kali dengan input + parameter + versi sama; ops mengubah faktor belanja 0,8 → 0,7 lewat `/ops/settings` (kunci `est.*`); import snapshot baru; Admin customer memanggil API parameter; nilai tidak valid (review rate 0 atau > 100%, bawah > atas, negatif) | Hasil identik (termasuk pembulatan). Perubahan parameter → entri audit (lama → baru) + `est.compute` di-enqueue di transaksi yang sama → rantai hitung ulang; hasil menyimpan parameter baru. Import baru → estimasi dihitung ulang dengan "Data per" baru. Setiap hasil menyimpan `est-1.0.0`, parameter, dan tanggal snapshot. Admin → 403. Tidak valid → 422 tanpa perubahan | I | FR-EST-06, FR-OPS-01/02, §E.5 |
| TC-LOC-01 | Titik + radius 500 m; tempat pada 499 m dan 501 m; tetangga sejenis dengan (ulasan > 0 + pola), (ulasan 0 + pola), (ulasan > 0 tanpa pola); kafe; hotel, klinik, kantor, toko; radius 99 / 100 / 2.000 / 2.001 m | 501 m tidak ikut. Pengelompokan sejenis / F&B lain / non-F&B sesuai kelompok target. Hanya ulasan > 0 dan punya pola = "data cukup"; yang lain tetap terhitung dengan alasan. Tiap tetangga membawa tanggal datanya. Non-F&B dihitung per jenis (kesehatan, akomodasi, kantor/pendidikan, ritel) + 3 nama terdekat urut jarak. Radius di luar 100–2.000 → 422 tanpa debit | U, I | Lampiran F |
| TC-LOC-02 | 0, 1, 2, 4, 5 tetangga sejenis dengan data cukup (+ F&B lain dengan data cukup yang tidak boleh ikut dihitung) | "Tidak dapat dianalisis" (pola & jam inti tidak dihitung, F&B lain tetap tampil) · Rendah + saran cek lapangan · Sedang · Sedang · Tinggi | U | Lampiran F |
| TC-LOC-03 | Median titik tengah tetangga sejenis 100.000; harga rencana titik tengah 79.900 / 80.000 / 120.000 / 120.100; harga rencana "250.000+"; jumlah tetangga ber-harga genap; tanpa harga rencana; tanpa tetangga sejenis ber-harga | Lebih murah / setara / setara / lebih mahal; titik tengah rentang terbuka = min; median = rata-rata dua nilai tengah; dua kasus terakhir → posisi harga "Tidak diketahui" | U | Lampiran F |
| TC-LOC-04 | `POST /location-analyses`: analisis baru; ulangi dengan `Idempotency-Key` sama berurutan dan paralel; buka ulang analisis tersimpan; hitung gagal (fake error); saldo 0; titik dengan 0 tetangga sejenis ber-data cukup; analisis baru di titik yang sama | Tepat 1 debit `credit` + 1 analisis tersimpan per key (request ulang → hasil sama, 200). Buka ulang → 0 credit, hasil identik (tidak dihitung ulang). Gagal → tanpa debit, tidak tersimpan. Saldo 0 → 402 sebelum dihitung, tanpa entri ledger. **"Tidak dapat dianalisis" → 0 debit, analisis tetap tersimpan**. Analisis baru = 1 credit | I | FR-LOC-01, Lampiran D, BR-CRED-01 |
| TC-LOC-05 | Pola tetangga dari 2 tetangga sejenis; pola dengan jam inti 12–14 dan 17–20; pembanding dengan jam inti 10–11 dan 18–19; tetangga sejenis tanpa data cukup | Pola tetangga = rata-rata nilai per jam; saran jam buka 11.00–21.00; pembanding 10–11 "tidak didukung tetangga", 18–19 tidak ditandai; tetangga tanpa data cukup tidak ikut pola. Hasil + `loc-1.0.0` deterministik | U | Lampiran F, §E.4 |
| TC-LOC-06 | Input GT-02 yang sama **tanpa** bisnis pembanding | Analisis lengkap tanpa pembanding: tetangga, pola tetangga (seri tetangga sejenis + F&B lain), jam inti 17–21, saran jam buka 16.00–22.00, posisi harga lebih murah, sumber pengunjung siang, keyakinan Rendah + saran cek lapangan identik dengan GT-02. Bagian pembanding dan tanda "tidak didukung tetangga" tidak ada; kesimpulan tidak menyebut pembanding; tetap 1 credit | U, I | FR-LOC-01, Lampiran F § Input |
| TC-GEO-01 | `msp tool geo-sync` ke server GeoAPI fake (rekaman), dijalankan dua kali | 4 level tersimpan dengan populasi & atribusi; run ke-2 tanpa duplikat; client tidak melebihi 120 req/menit (clock fake) | U, I | FR-DATA-03 |
| TC-GEO-02 | Poligon kotak buatan: titik di dalam kelurahan A, tepat di batas A/B, di luar semua poligon (termasuk 100 m dari tepi A) | A + kecamatan/kab/prov induknya; titik batas → tepat satu kelurahan (deterministik); di luar → tidak terpetakan (wilayah "Tidak diketahui", tanpa fallback desa terdekat), masuk hitungan "% terpetakan" | I | FR-DATA-04 |
| TC-GEO-03 | Metrik area per kelompok estimasi: populasi 499 / 500, jumlah bisnis 4 / 5; persentil kompetisi 32,9 / 33 / 66 / 66,1 | <500 atau <5 → "Data terbatas", metrik tampil, `market_score` kosong; kompetisi Rendah/Sedang/Sedang/Tinggi; dihitung terpisah untuk `makan`, `kafe`, `hotel`; faktor Wilayah memakai kecamatan bila kelurahan terbatas | U | Lampiran C, A |
| TC-GEO-04 | Kelurahan dengan 2 vs 3 bisnis `makan` ber-estimasi (kunjungan tengah 4.000, 6.000, 8.000) dan 2 vs 3 ber-harga; kelurahan tanpa bisnis ber-estimasi; `GET /areas/ranking?group=` untuk `makan` dan bisnis `lainnya` di profil | 2 → permintaan terbukti & rentang harga "Tidak diketahui"; 3 → jumlah 18.000, rata-rata 6.000 per bisnis, median & sebaran titik tengah harga; `market_score` `geo-3.0.0` memakai bobot Lampiran C, faktor kunjungan yang tidak diketahui dikeluarkan dan bobotnya dibagi ulang. Peringkat urut `market_score`. Bisnis `lainnya` memakai metrik semua bisnis di area | U, I | Lampiran C, FR-GEO-01 |
| TC-INTL-01 | Guard BR-OPP-01 untuk OPP-WEB-NONE: kelompok 20 bisnis dengan 10 ber-website; 20 dengan 9; 19 dengan 18 | 50% & ≥ 20 → opportunity dibuat untuk yang tanpa website; 45% → tidak dibuat; 19 bisnis → tidak dibuat | U | BR-OPP-01 |
| TC-INTL-02 | Website = linktr.ee untuk Offering `website_dev`, `social_media`, `reputation_mgmt` | Skor OPP-WEB-SOCIAL = severity × relevansi (1 / 0,6 / 0). Relevansi 0 → bukan opportunity, gap tetap tampil di kartu digital | U | Lampiran B |
| TC-INTL-03 | Faktor Popularitas & Wilayah tidak diketahui; lalu hitung ulang input yang sama | Faktor dikeluarkan, bobot dibagi ulang, `coverage` = 0,75; breakdown menandai faktor yang dikeluarkan; hasil identik untuk input + `priority-X.Y.Z` sama | U | BR-SCORE-01, FR-INTL-04 |
| TC-INTL-04 | Band pada skor 59, 60, 79, 80 | COLD, WARM, WARM, HOT | U | BR-SCORE-02 |
| TC-INTL-05 | Skor 85 dengan coverage 0,59; coverage 0,60; coverage 1 tapi kriteria ICP *wajib* tidak diketahui | WARM + "Data terbatas"; HOT; WARM + "Data terbatas" | U | BR-SCORE-02 |
| TC-INTL-06 | Buka profil bisnis fixture lengkap, minim, dan hotel | Setiap nilai (termasuk estimasi, pola, riwayat, atribut) punya claim type + sumber + tanggal data, evidence bisa dibuka; estimasi tanpa data → "Belum dapat diperkirakan" + alasan; email/person → "Tidak diketahui"; tidak pernah kosong/"0"/"—" | I, E | FR-INTL-01, FR-PROF-01, BR-TRUST-01/03, BR-EST-01 |
| TC-INTL-07 | Umur data 180, 181, 365, 366 hari (clock fake) | Baru, Menua, Menua, Usang | U | BR-TRUST-02 |
| TC-ICP-01 | User ganti ICP aktif; ICP dengan kriteria *wajib* yang tidak cocok | Skor & urutan berubah sesuai ICP baru; bisnis yang gagal kriteria wajib tidak tampil | I | FR-ICP-02, Lampiran A |
| TC-ICP-02 | ICP kategori Restoran Asia, min. rating 4,0, min. ulasan 20. Bisnis: rating 4,0 / 3,7 / 3,6 / tidak ada; ulasan 20 / 10 / 9; kategori Restoran Asia / kategori lain dengan induk sama / Lainnya | Rating: Cocok / Sebagian / Tidak / Tidak diketahui; ulasan: Cocok / Sebagian / Tidak; kategori: Cocok / Sebagian / Tidak (Lainnya tidak pernah cocok) | U | Lampiran A |
| TC-SRCH-01 | Kata kunci + filter kategori, kelompok estimasi, kelurahan, rentang perkiraan omzet, punya pola keramaian, band, tipe opportunity, ada telepon, rating minimum; urut skor, kunjungan, omzet, laju ulasan; tanpa ICP dan dengan ICP | ID hasil = himpunan ekspektasi fixture; total = jumlah item seluruh halaman; filter omzet memakai nilai tengah; urut turun dengan tie-break stabil, "Belum dapat diperkirakan" selalu di akhir; semua bisnis yang lolos filter tampil, kolom skor terisi untuk kandidat ICP aktif; urut skor atau filter band → hanya kandidat ICP; tanpa ICP aktif `sort=score` → 422 | I | FR-SRCH-01, FR-ICP-02 |
| TC-AUTH-01 | Session dengan email belum terverifikasi memanggil endpoint data | 403; setelah verifikasi → 200 | I | FR-AUTH-01 |
| TC-AUTH-02 | Member: undang anggota, kelola ICP, daftar webhook, lihat usage. Admin: aksi yang sama | Member → 403 untuk semua; Admin → 2xx | I | FR-AUTH-02 |
| TC-AUTH-03 | Akun ops session AAL1; ops AAL2 (TOTP); Admin workspace customer ke halaman/API ops | 403; 200; 403 | I, E | FR-AUTH-03, FR-OPS-01 |
| TC-AUTH-04 | Plan dengan 2 seat: workspace berisi Admin + 1 undangan tertunda mengundang orang ke-3; lalu undangan tertunda dibatalkan dan undang ulang | Ditolak `SEAT_LIMIT` tanpa undangan maupun email; setelah dibatalkan → 2xx | I | FR-AUTH-02, FR-BILL-01 |
| TC-PRIV-01 | User workspace A membaca/mengubah list, ICP, catatan, feedback (opportunity & kesimpulan analisis), **analisis lokasi tersimpan**, unlock, ledger, export, webhook milik B lewat ID; query langsung dengan konteks A | 404 untuk akses by ID; 0 baris milik B. Test katalog: tiap tabel data workspace punya RLS aktif + policy (menangkap tabel baru tanpa RLS) | I | FR-PRIV-01, NFR-05 |
| TC-PRIV-02 | Telepon bisnis masuk suppression list setelah di-unlock | Tersembunyi di profil, tidak ada di file export dan payload webhook | I | FR-PRIV-02 |
| TC-ACT-01 | Unlock bisnis yang sama dua kali berurutan dan 2 request paralel; saldo 0 | Satu debit saja, saldo = jumlah ledger; saldo 0 → ditolak tanpa entri ledger | I | FR-ACT-02, FR-BILL-01, BR-CRED-01 |
| TC-ACT-02 | "Unlock & export" 10 bisnis (3 sudah di-unlock); lalu job export dibuat gagal (fake `storage` error) dan diulang; export 10 bisnis yang sama tanpa unlock | Tanpa unlock → 0 debit, 7 telepon ter-mask di file. Preview 7 credit; konfirmasi → 7 debit + 7 unlock (via unlock); export gagal → tidak ada debit tambahan; ulang → sukses, tetap total 7 debit. File memuat kolom estimasi (daftar di 04) dengan rentang, sumber harga, dan tanggal data | I | FR-ACT-02/03, BR-CRED-01 |
| TC-ACT-03 | Daftar URL: skema `http://`; `https://` ke `localhost`, `127.0.0.1`, `2130706433` (desimal), `[::1]`, `10.x`, `172.16.x`, `192.168.x`, `169.254.169.254`, `0.0.0.0`, `[fd00::1]`; host publik yang resolve ke IP privat, host publik yang redirect 302 ke `169.254.169.254` | Semua ditolak saat daftar; IP dicek ulang saat koneksi (resolver fake); redirect tidak diikuti → pengiriman gagal tercatat | U, I | FR-ACT-04, NFR-05 |
| TC-ACT-04 | Penerima `httptest` membalas 500, 500, 200; lalu selalu 500; body diubah 1 byte | Signature HMAC-SHA256 valid; 3 percobaan tercatat lalu sukses; selalu 500 → berhenti di batas retry, status gagal; body diubah → verifikasi gagal | I | FR-ACT-04 |
| TC-ACT-05 | Bulk unlock & kirim ke webhook 100 vs 101 bisnis; export 10.000 vs 10.001 baris | Tepat di batas → diterima; 101 / 10.001 → 422 tanpa debit maupun job | I | FR-ACT-02..04 |
| TC-AI-01 | Buat narasi profil (fixture GT-01) dan narasi analisis lokasi (fixture GT-02) dengan LLM fake; fake membalas kalimat tanpa rujukan fakta, lalu valid | Request ke LLM hanya berisi fakta level bisnis dan hasil hitungan (estimasi, pola, kesimpulan analisis), tanpa telepon (mentah/E.164), nilai kontak, maupun `user_reviews`; narasi punya 5 bagian (masalah, data, cara membaca, kesimpulan, batasan); jawaban tanpa rujukan ditolak → dibuat ulang sekali; gagal lagi → error jelas, tidak ditagih | U, I | FR-ACT-05, BR-AI-01 |
| TC-AI-02 | Fake Claude membalas 5xx/timeout; lalu keduanya gagal | Request pindah ke fake OpenAI-compatible (base URL & model dari pengaturan AI), provider, model & versi `brief-2.0.0` tercatat; keduanya gagal → error, tanpa narasi tersimpan | I | FR-ACT-05, D-15 |
| TC-AI-03 | Narasi yang sama (profil dan analisis lokasi) diminta lagi hari ke-6 dan hari ke-8 (clock fake); flag AI dimatikan | Hari ke-6: dari cache, 0 AI credit, 0 request ke LLM; hari ke-8: dibuat ulang, 1 AI credit; flag mati → ditolak tanpa panggilan LLM | I | FR-ACT-05, Lampiran D |
| TC-AI-04 | Ops mengubah model utama lewat API pengaturan AI; narasi diminta sebelum dan sesudah t+61 s (clock fake) tanpa restart proses. Fallback dinonaktifkan lalu Claude fake gagal; diaktifkan lagi. Baca, simpan, dan Uji koneksi dengan API key fake bernilai unik. Admin workspace customer memanggil API pengaturan | Pada t+61 s request ke LLM fake memakai model baru (tanpa redeploy); fallback nonaktif → error tanpa panggilan ke fallback; aktif → fallback dipanggil dengan base URL & model dari pengaturan; nilai key tidak pernah muncul di body respons maupun audit (hanya status terpasang/belum); perubahan tercatat di audit; Admin → 403 | I | FR-ACT-05, FR-OPS-01, D-29 |
| TC-BILL-01 | Ops membuat, mengubah, dan menonaktifkan plan lalu menetapkannya ke workspace; Admin/Member customer memanggil API yang sama; nama kosong atau angka negatif | Ops → 2xx + entri audit; plan nonaktif tidak bisa ditetapkan (422), workspace yang sudah memakainya tetap; customer → 403; validasi gagal → 422 | I | FR-BILL-01, FR-OPS-01 |
| TC-BILL-02 | Job grant bulanan dijalankan 2× berurutan dan 2× paralel dalam siklus yang sama (clock fake), lalu di siklus berikutnya; nilai plan diubah di tengah siklus; workspace tanpa plan | Tepat 1 grant `credit` + 1 grant `ai_credit` per workspace per siklus sesuai nilai plan saat grant; siklus berikutnya → grant baru; perubahan plan terpakai di grant berikutnya; tanpa plan → tidak ada grant; saldo = jumlah ledger | I | FR-BILL-01 |
| TC-OPS-01 | Jalankan tiap aksi teraudit (login, undang, unlock, export, webhook, grant credit, plan, pengaturan AI, parameter estimasi, flag, suppression); coba UPDATE/DELETE audit dengan role aplikasi | Satu entri per aksi (aktor, workspace, waktu, target; nilai lama → baru untuk pengaturan); UPDATE/DELETE → error izin | I | FR-OPS-02 |

### 4.1 Golden test sample case founder (PRD §12)

Dua golden test mereproduksi sample case founder dengan data aslinya, dibandingkan dengan hitungan manual founder (PRD §12). Masing-masing punya dua lapis dengan ekspektasi yang sama: **U** memanggil fungsi murni paket `estimate` langsung (`Velocity`, `Estimate`, `Pattern`, `AnalyzeLocation`; 04 §5.4) dengan input dari tabel di bawah; **I** men-seed `golden-*.sql` (§6) ke DB sumber fixture, menjalankan import + `est.compute`, lalu membaca angka utama dari `GET /prospects/{id}` dan `POST /location-analyses`. Lapis I membuktikan rantai import → gabung snapshot → estimasi → API. Parameter = default Lampiran E §E.5 (review rate `makan` 2% / 1,5% / 1%, faktor belanja 0,8, ambang 60% / 80% / 85% / 30%). Perbandingan ambang memakai nilai tanpa pembulatan; pembulatan hanya untuk tampilan (BR-EST-02).

**GT-01 — Signa Cafe** (FR-EST-01, -02, -04, -05; BR-NORM-04..06)

Fixture: satu tempat, tiga snapshot (`created_at` = `updated_at` = waktu scrape).

| Snapshot | `review_count` | `price_range` | `category` | Field lain |
|---|---|---|---|---|
| 10 Feb 2026 | 1.336 | "Rp 75–200K" | "Restaurant" | — |
| 8 Agu 2026 | 1.463 | kosong | "Restaurant" | — |
| 24 Sep 2026 | 1.507 | "Rp 100–225 rb" | "Restoran", `categories` memuat "Kafe" | `reviews_per_rating` `{"1":48,"2":36,"3":40,"4":64,"5":1319}`; `open_hours` setiap hari "08.00–23.00"; `popular_times` di bawah; `timezone` `Asia/Makassar`; `latitude` −8.8176127, `longitude` 115.1901369; `address_country` "ID"; `website` `https://signa.cafe/`; `social_links` Instagram; `owner` terisi; `phone` kosong |

`popular_times` asli (jam 8..22; jam 6, 7, dan 23 bernilai 0):

| Hari | 8 | 9 | 10 | 11 | 12 | 13 | 14 | 15 | 16 | 17 | 18 | 19 | 20 | 21 | 22 |
|---|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|
| Monday | 50 | 70 | 81 | 84 | 81 | 77 | 74 | 73 | 73 | 72 | 77 | 83 | 88 | 86 | 74 |
| Tuesday | 49 | 67 | 76 | 83 | 92 | 100 | 94 | 91 | 86 | 86 | 85 | 88 | 90 | 81 | 67 |
| Wednesday | 62 | 78 | 87 | 90 | 91 | 93 | 92 | 90 | 85 | 80 | 82 | 80 | 78 | 74 | 66 |
| Thursday | 50 | 64 | 69 | 68 | 67 | 72 | 74 | 72 | 67 | 68 | 79 | 83 | 81 | 74 | 63 |
| Friday | 45 | 59 | 69 | 69 | 68 | 68 | 74 | 76 | 80 | 81 | 83 | 84 | 86 | 82 | 70 |
| Saturday | 46 | 65 | 79 | 83 | 78 | 71 | 70 | 69 | 76 | 78 | 82 | 85 | 85 | 84 | 73 |
| Sunday | 58 | 70 | 75 | 72 | 72 | 68 | 65 | 66 | 68 | 74 | 80 | 81 | 78 | 73 | 62 |

| Hasil | Ekspektasi |
|---|---|
| Laju ulasan | Snapshot terbaru 24 Sep + tertua dalam 365 hari 10 Feb: (1.507 − 1.336) ÷ 226 hari × 365 ≈ **276/tahun** (276,2; Turunan) |
| Laju terkini (90 hari) | Jendela mulai 26 Jun 2026 → pasangan 8 Agu & 24 Sep: 44 ÷ 47 hari × 365 ≈ 342/tahun (341,7). Selisih +23,7% ≤ 30% → penanda "Laju ulasan berubah" **tidak** tampil |
| Harga | Nilai tidak kosong terbaru "Rp 100–225 rb" (24 Sep) → 100.000–225.000, bukan "Rp 75–200K". Sumber "harga tempat ini" (`own`); belanja per tamu 162.500 × 0,8 = 130.000 |
| Kunjungan/tahun | **13.800 / 18.400 / 27.600** |
| Tamu per hari buka | ≈ 37,8 / 50,4 / 75,7 (buka 7 hari) |
| Omzet/tahun | **Rp1,8 / 2,4 / 3,6 miliar**; tabel sensitivitas review rate 1% / 1,5% / 2% → Rp3,6 / 2,4 / 1,8 M |
| Penanda bintang 5 | 1.319 ÷ 1.507 = 87,5% ≥ 85% → `five_star_heavy` dengan teks netral (03 §7) |
| Kategori | "Restaurant" = "Restoran" → kelompok `makan`; `categories` berisi "Kafe" tidak mengubah kelompok (BR-NORM-03), "Kafe" tampil sebagai kategori tambahan |
| Pola: rata-rata 7 hari jam 8..22 | 51 68 77 78 78 78 78 77 76 77 81 83 84 79 68 (dibulatkan) |
| Jam puncak | Rata-rata: **jam 20** (83,7). Per hari: Sen 20, **Sel 13 (= 100)**, Rab 13, Kam 19, Jum 20, Min 19; Sab seri jam 19 & 20 → 19 (aturan seri TC-EST-05) |
| Jam inti | Ambang 60% × 83,7 = 50,2 → **jam 8–22** (jam 8 = 51,4, tepat di atas ambang) = 15 dari 15 jam buka → **"Ramai sepanjang hari"** |
| Keramaian rata-rata | ≈ 76% (75,6) dari jam tersibuknya |
| Bentuk per blok | Pagi 17% · Siang 28% · Sore 20% · Malam 29% · Larut 6% |
| Zona & tanggal | "08.00–23.00 WITA", puncak "20.00 WITA" (TC-DATA-10); "Data per 24 Sep 2026"; `est-1.0.0` tersimpan bersama parameter |
| Lain-lain | Catatan metode pesan-antar tampil; website domain sendiri, Instagram, "Listing diklaim pemilik"; telepon "Tidak diketahui" |

Catatan: pembacaan founder "ramai dari jam 10 sampai 21" lebih sempit daripada hasil aturan §E.4 (jam inti 8–22). Test mengikuti aturan PRD.

**GT-02 — Warung Kita, Jl. Pulau Kawe** (FR-LOC-01, Lampiran F)

Fixture: titik analisis di Jl. Pulau Kawe. Semua tempat `timezone` `Asia/Makassar`, satu snapshot 20 Sep 2026 dengan ulasan > 0, `popular_times` sama untuk ketujuh hari (jam lain 0). Koordinat fiktif dipilih agar jaraknya sesuai tabel.

| Tempat | Kategori → kelompok | Jarak | Ulasan | Harga | `popular_times` |
|---|---|---|---|---|---|
| Kampoeng Renon (KR) | Restoran → `makan` | ±200 m | 687 | "Rp 50–100K" | jam 11..21: 25 27 33 35 35 36 48 64 73 71 65 (buka 11.00–22.00) |
| Joglo (pembanding) | Restoran → `makan` | ±2 km (di luar radius) | 1.628 | "Rp 25–50K" | jam 11..21: 40 55 58 52 44 44 57 75 83 71 47 |
| Berkarier Coffee | Kafe → `kafe` | ±400 m | 304 | "Rp 25–50K" | jam 11..23: 15 17 17 18 18 19 22 29 40 55 71 79 73 |
| Klinik (fiktif) | Klinik → non-F&B, kesehatan | ±150 m | > 0 | — | — |
| Apotek 24 jam (fiktif) | Apotek → non-F&B, kesehatan | ±300 m | > 0 | — | — |
| Hotel (fiktif) | Hotel → non-F&B, akomodasi | ±450 m | > 0 | — | — |

Input: radius 500 m, kelompok `makan`, harga rencana Rp25.000–50.000, pembanding Joglo.

| Hasil | Ekspektasi |
|---|---|
| Tetangga | Sejenis: KR (data cukup). F&B lain: Berkarier Coffee (data cukup). Non-F&B: 3. Joglo bukan tetangga (di luar radius) tetapi tampil sebagai pembanding. Setiap tetangga membawa "Data per 20 Sep 2026" |
| Jam inti tetangga | Puncak 73 (jam 19), ambang 43,8 → **jam 17–21** |
| Saran jam buka | **16.00–22.00** WITA |
| Pembanding Joglo | Puncak 83 (jam 19), ambang 49,8 → jam inti **12–14 dan 17–20**; **12–14 ditandai "tidak didukung tetangga"**, 17–20 tidak ditandai |
| Bentuk per blok | KR: Siang 23% · Sore 23% · Malam 53%. Joglo: Siang 33% · Sore 23% · Malam 44%. Pagi & Larut 0% |
| F&B lain | Berkarier Coffee di bagian F&B lain dengan jam inti **20–23** (puncak 79 jam 22, ambang 47,4). Tidak ikut pola tetangga sejenis maupun keyakinan |
| Keyakinan | 1 tetangga sejenis dengan data cukup → **Rendah** + saran cek lapangan |
| Posisi harga | Titik tengah rencana 37.500 ÷ median tetangga sejenis 75.000 = 50% < 80% → **lebih murah** |
| Sumber pengunjung siang | Kesehatan 2 · Akomodasi 1 · Kantor/pendidikan 0 · Ritel 0; terdekat: klinik (150 m), apotek (300 m), hotel (450 m) |
| Pola datar (kontras GT-01) | KR: jam inti 5 dari 11 jam buka (45%) → **tidak** berlabel "Ramai sepanjang hari" |
| Versi, biaya & kesimpulan | `loc-1.0.0`; 1 credit (bukan "Tidak dapat dianalisis"); kesimpulan deterministik memuat saran jam buka, posisi harga, jam pembanding yang tidak didukung, dan keyakinan. Varian tanpa pembanding: TC-LOC-06 |

---

## 5. Edge case data dari sumber

Setiap EC = satu tempat di `edge-cases.sql` (§6), dengan nama tempat diawali ID-nya. Satu tempat bisa punya beberapa snapshot. Test import fixture memeriksa ekspektasi semua EC.

| ID | Kasus di sumber | Ekspektasi |
|---|---|---|
| EC-01 | `website` = `https://linktr.ee/…` | Link aggregator, tidak dihitung "punya website"; OPP-WEB-SOCIAL |
| EC-02 | `website` NULL vs `""` | Keduanya kosong; website "Tidak diketahui"; OPP-WEB-NONE hanya bila BR-OPP-01 terpenuhi |
| EC-03 | `website` = `https://wa.me/62812…` | Link pesan, bukan website; OPP-WEB-SOCIAL |
| EC-04 | `phone` telepon tetap vs seluler, dengan spasi/tanda hubung/kurung | Tipe benar; hanya seluler yang "Kemungkinan WhatsApp"; faktor Kontak 60 vs 80 (Lampiran A) |
| EC-05 | `phone` kosong; `social_links` ada / tidak ada | Kontak "Tidak diketahui"; faktor Kontak 30 (hanya sosial) atau 0 |
| EC-06 | `review_rating` ada tanpa `review_count` | Rating Teramati, jumlah ulasan "Tidak diketahui"; laju ulasan "Belum dapat diperkirakan"; Popularitas dikeluarkan dari skor; OPP-REP-LOW-RATING tidak terpicu |
| EC-07 | Satu `cid` dengan banyak baris (snapshot), termasuk dua baris di hari yang sama | **Bukan duplikat dan bukan error**: satu bisnis, riwayat satu baris per hari (snapshot terakhir hari itu), laporan menghitung snapshot baru (BR-NORM-05) |
| EC-08 | Koordinat kosong; tempat ID dengan koordinat di luar Indonesia (mis. Singapura) | Bisnis tetap diimpor, wilayah "Tidak diketahui", tidak masuk metrik area; faktor Wilayah dikeluarkan; "Analisis lokasi di sini" nonaktif bila koordinat kosong |
| EC-09 | `price_range` = "$$" atau format tak dikenal | Harga "Tidak diketahui"; omzet memakai sumber harga berikutnya (Lampiran E §E.2) dan menyebut sumbernya |
| EC-10 | `category` tak dikenal ("XyzStore"); `category` Inggris vs Indonesia untuk tempat yang sama di snapshot berbeda | "Lainnya" + tercatat di laporan; kategori dwibahasa → kategori MSP sama |
| EC-11 | Kelurahan dengan populasi < 500 | "Data terbatas", `market_score` tidak dihitung, faktor Wilayah memakai kecamatan |
| EC-12 | Baris tanpa `cid`; baris tempat baru tanpa `title` | Ditolak dengan alasan `missing_cid` / `missing_name` + `id` baris sumber; import lanjut ke baris berikutnya |
| EC-13 | Nama sama ("Warung Uji"), `cid` berbeda | Dua bisnis terpisah, tidak digabung (D-05) |
| EC-14 | `price_range` hanya ada di snapshot lama; snapshot terbaru kosong | Harga = nilai lama dengan tanggal snapshot-nya sendiri (BR-NORM-05); freshness harga dari tanggal itu |
| EC-15 | `review_count` turun antar snapshot (ulasan dihapus) | Laju 0, tidak negatif (Lampiran E §E.1) |
| EC-16 | `popular_times` berisi nilai 120; tempat lain dengan satu hari semua 0 | Field `popular_times` ditolak (Tidak diketahui) + peringatan, field lain masuk; hari semua 0 → "Tutup" |
| EC-17 | Hotel bintang 3 tanpa `popular_times` | Pola keramaian "Belum dapat diperkirakan"; estimasi hotel tetap dihitung bila parameter kelas ada |

---

## 6. Data uji

| Jenis | Isi | Lokasi |
|---|---|---|
| Skema DB sumber fixture | Tabel `business_listings` dengan kolom persis seperti DB gmaps (PRD §7.1; pemetaan di 04): `id` bigint, `place_id`, `cid`, `title`, `category`, `categories` text[], `category_id`, `address`, `address_city`, `address_state`, `address_country`, `latitude`, `longitude`, `timezone`, `phone`, `phones`, `website`, `social_links` jsonb, `review_count`, `review_rating`, `reviews_per_rating` jsonb, `price_range`, `open_hours` jsonb, `popular_times` jsonb, `about` jsonb, `owner` jsonb, `status`, `user_reviews` jsonb, `created_at`, `updated_at`. Satu baris = satu snapshot. Bila pemetaan di 04 berubah, file ini ikut | `apps/api/testdata/gmaps/schema.sql` |
| Seed kecil | ±60 tempat sintetis Indonesia, 2–5 snapshot per tempat dengan rentang ≥ 30 hari; 2 kab/kota × kelompok `makan` dan `kafe` dengan ≥ 20 tempat per kelompok (BR-OPP-01, persentil) dan ≥ 10 ber-harga (median harga, §E.2); sebagian punya `popular_times`; beberapa hotel; + 3 tempat negara lain | `apps/api/testdata/gmaps/seed-small.sql` |
| Edge case | Satu tempat per `EC-nn` (§5) | `apps/api/testdata/gmaps/edge-cases.sql` |
| Golden | GT-01, GT-02 (§4.1) | `apps/api/testdata/gmaps/golden-signa.sql`, `golden-warung-kita.sql` |
| Builder (Go) | Fungsi pembuat baris snapshot dengan default valid untuk unit test; test menimpa field tertentu (DRY, tanpa generator terpisah) | paket test helper di `apps/api` |
| Rekaman GeoAPI | Respon nyata untuk satu kab/kota (kecamatan + kelurahan, populasi, batas disederhanakan), diputar ulang oleh server fake. Diperbarui manual dengan flag `-update` | `apps/api/testdata/geoapi/` |
| Poligon buatan | Kotak sederhana untuk TC-GEO-02 (lebih mudah dibaca daripada batas nyata) | di dalam test |
| Fake eksternal | `httptest` untuk GeoAPI, Claude, OpenAI-compatible, penerima webhook, Kratos *whoami*; resolver DNS fake untuk SSRF; Mailpit untuk email di E2E | di dalam test / stack E2E |

Setup: helper test membuat database `gmaps_fixture` di service Postgres yang sama (06), menjalankan `schema.sql` + seed yang dibutuhkan, membuat role read-only, lalu mengarahkan `MSP_SOURCE_DATABASE_URL` ke role itu. Stack E2E dan `local` memakai cara yang sama. `created_at` dan `updated_at` fixture diisi waktu scrape yang sama, sehingga fixture tetap benar apa pun kolom waktu yang dipilih 04.

Contoh satu baris (ilustratif; JSON dipersingkat):

```sql
INSERT INTO business_listings (id, place_id, cid, title, category, categories, address, address_city, address_state,
  address_country, latitude, longitude, timezone, phone, website, social_links, review_count, review_rating,
  reviews_per_rating, price_range, open_hours, popular_times, about, owner, created_at, updated_at)
VALUES (1001, 'TEST-PLACE-0001', '1001', 'EC-01 Warung Uji Satu', 'Restoran', '{Restoran}', 'Jl. Contoh 1',
  'Kota Semarang', 'Jawa Tengah', 'ID', -6.9932, 110.4203, 'Asia/Jakarta', '0812-0000-0001',
  'https://linktr.ee/warunguji', '{"instagram": "https://instagram.com/warunguji"}', 57, 4.3,
  '{"1":2,"2":1,"3":4,"4":10,"5":40}', 'Rp 25–50K', '{"Senin": "10.00–21.00", "Minggu": "Tutup"}',
  '{"Monday": {"6": 0, "10": 35, "12": 80, "19": 100}}', '{"popular_for": ["Makan siang"]}', NULL,
  '2026-08-01 10:00+07', '2026-08-01 10:00+07');
```

**Aturan**
- Tidak ada data customer nyata (akun, ICP, list, catatan, unlock, analisis) di `local`, CI, atau `staging`. Tidak ada dump database produksi ke luar `production`.
- Fixture memakai nama, nomor (`+62812000…`), dan domain fiktif. **Pengecualian:** fixture golden memakai nama dan angka asli sample case founder (data publik tempat usaha, tanpa telepon); tempat pelengkap di fixture golden tetap fiktif. `staging` membaca data nyata dari DB gmaps (read-only) dengan akun test.
- `user_reviews` di fixture boleh berisi teks dummy agar TC-DATA-01 bisa membuktikan kolom ini tidak diimpor.
- CI tanpa akses jaringan ke layanan luar: semua `MSP_*_BASE_URL` diarahkan ke fake, `MSP_SOURCE_DATABASE_URL` ke `gmaps_fixture`.

---

## 7. Gerbang rilis F1

**Satu-satunya checklist gerbang F1** (dokumen lain merujuk ke sini): satu baris per item PRD §12, masing-masing dengan bukti.

| # | Item PRD §12 | Cek konkret | Bukti |
|---|---|---|---|
| 1 | Alur end-to-end di staging dengan data nyata | Import dari DB gmaps di staging (job `import.run` atau `msp tool import --country ID`, 09 RB-03) + `msp tool geo-sync --country ID` tanpa error, laporan import mencatat "dilewati (negara lain)"; kesehatan data ops: ≥ 98% bisnis ber-koordinat terpetakan (FR-DATA-04), % harga, % `popular_times`, dan % riwayat ≥ 30 hari terisi; E2E J-2..J-7 hijau di staging; walkthrough manual: login Google, Uji koneksi di Pengaturan AI OK, narasi AI dengan Claude asli (profil dan analisis lokasi), webhook ke penerima milik tim | Link pipeline tag + checklist terisi |
| 2 | Sample case founder direproduksi (golden test) | GT-01 dan GT-02 hijau di pipeline tag, dengan ekspektasi persis §4.1 (tidak ada perubahan ekspektasi tanpa perubahan PRD §12) | Log job CI |
| 3 | Setiap nilai di profil dan analisis lokasi menampilkan claim type, sumber, dan tanggal data | TC-INTL-06, J-2, J-7 hijau; cek manual 10 profil acak di staging (termasuk yang datanya minim dan satu hotel) + 3 analisis lokasi | Checklist + screenshot |
| 4 | Test otomatis lulus (isolasi, SSRF, idempotensi credit) | Pipeline tag hijau; TC-PRIV-01, TC-ACT-01..03, TC-LOC-04 ada dan lulus; coverage paket inti ≥ 80%; 0 test di-skip; **cek ID**: skrip kecil memastikan setiap `TC-`/`GT-`/`EC-` di dokumen ini ditemukan di kode test | Log job CI |
| 5 | Restore backup berhasil sekali | Restore pgBackRest ke instance Postgres terpisah, bukan database production (prosedur `09`), Postgres hasil restore bisa diquery, jumlah baris tabel utama cocok; catat waktu (RTO ≤ 8 jam) dan titik pulih (RPO ≤ 15 menit) (NFR-04). Setelah itu bulanan per `09` | Catatan restore |
| 6 | Onboarding founder ≤ 5 menit | Founder daftar akun baru di staging dan memilih tujuan; waktu dari submit daftar sampai hasil pertama tampil. Jalur Analisis lokasi butuh credit, jadi ops memberi credit ke workspace itu lebih dulu (FR-BILL-01) | Waktu tercatat |

Cek pendukung (sekali, di staging, sebelum tag; ringan sesuai §2): NFR-01 dengan skrip latensi berurutan, bukan load test (mis. 200 search + 200 profil acak + 30 analisis lokasi di workspace test, satu request sekaligus; p95 dari hasil skrip); NFR-02 dengan export 10.000 baris dan satu narasi AI diukur waktunya, serta durasi import harian dari riwayat `/ops/imports` (selesai sebelum jam kerja); NFR-06 axe 0 pelanggaran *serious/critical* di semua journey.

---

## 8. Tambahan F2 / F3

Cakupan fitur: [PRD §6](01-prd.md). Tambahan khusus test:
- F2: E2E per fitur baru; set evaluasi kecil untuk kualitas narasi AI & NL search; E2E semua browser di `main`.
- F3: load test data > 5 juta tempat di lingkungan terpisah dari production; test deploy tanpa downtime (migrasi kompatibel mundur); cek regresi `EXPLAIN` untuk query utama.
