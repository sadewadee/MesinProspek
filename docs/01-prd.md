# 01 — PRD (lean)

## MesinProspek — Sales Intelligence Engine

| Field | Value |
|---|---|
| Versi | 0.2 (lean). Menggantikan v0.1 yang terlalu besar untuk MVP |
| Tanggal | 2026-10-10 |
| Prinsip | **KISS**, **DRY**, **Make it work → make it right → make it fast** (D-23) |
| Input | Dataset bisnis JSON-LD yang sudah ada (D-24) + data wilayah GeoAPI (D-25). Crawler **PAUSED** (D-01) |
| Pembangun | Agent AI (maks. 8 paralel) diorkestrasi Claude. Founder menyetujui tiap gerbang fase (D-22) |

> Konvensi: `FR-<AREA>-nn`, `BR-<AREA>-nn`, `NFR-nn`. Kolom **Fase**: **F1** = make it work (rilis jual pertama), **F2** = make it right, **F3** = make it fast. Hanya F1 yang dirinci. Fitur masuk F1 hanya jika dibutuhkan alur inti: **temukan → pahami → prioritaskan → tindak lanjuti**.

---

## 1. Ringkasan

MesinProspek membantu penjual B2B menemukan **bisnis lokal Indonesia** yang paling layak dihubungi. Untuk tiap bisnis, produk menjelaskan **alasannya** (opportunity + evidence) dan langsung memungkinkan **tindak lanjut** (unlock kontak, export, webhook, AI brief).

## 2. Masalah

1. Daftar bisnis mentah (Maps, direktori) tidak memberi tahu mana yang paling cocok dan butuh layanan kita.
2. Riset manual sebelum menghubungi prospek (cek website, ulasan, kompetitor sekitar) makan 10–20 menit per prospek.
3. Manajer tidak punya dasar data untuk memilih wilayah yang paling potensial.

## 3. Persona

| ID | Persona | Kebutuhan utama |
|---|---|---|
| PER-01 | Owner/BD agency digital (website, sosmed, ads, SEO lokal) | "Bisnis mana di area saya yang jelas butuh layanan saya, dan kenapa?" |
| PER-02 | Sales/SDR vendor SaaS untuk UMKM | "Antrean prospek terprioritas dengan kontak yang bisa dihubungi." |
| PER-03 | Sales manager | "Kecamatan/kelurahan mana yang paling potensial?" |
| PER-04 | Ops internal MesinProspek | "Import data lancar, data sehat, customer terlayani." |

## 4. Tujuan & non-tujuan

**Tujuan F1:**
- User baru melihat prospek yang cocok dengan ICP-nya ≤ 5 menit setelah daftar dan paham alasan urutannya.
- Setiap nilai menunjukkan sumbernya dan jenisnya (teramati, turunan, perkiraan, tidak diketahui).
- User bisa unlock, export, atau kirim prospek ke webhook.
- Pemakaian credit tercatat benar.

**Non-tujuan:** crawler, CRM, email sequencing/outreach otomatis, aplikasi mobile native, dan isu hukum (ditangani tim legal, D-27).

## 5. Prinsip produk

1. **Unknown lebih baik daripada mengarang.** Field yang tidak ada di dataset tampil "Tidak diketahui", bukan ditebak.
2. **Evidence untuk setiap klaim.** Nilai bisa ditelusuri ke record sumber.
3. **Satu record kanonik, banyak tampilan** (DRY). Search, profil, export, dan webhook membaca model yang sama.
4. **Fitur kecil yang bekerja** lebih baik daripada fitur lengkap yang setengah jadi.

## 6. Fase rilis

| Fase | Tema | Isi |
|---|---|---|
| **F1 — Make it work** | Alur inti end-to-end, bisa dijual ke customer awal | Semua FR di §9 |
| **F2 — Make it right** | Kualitas, kelengkapan, kenyamanan | Upload daftar customer + pencocokan; re-import → signal perubahan + watch/alert; laporan data salah + koreksi ops; role Viewer/Manager; compare prospek & wilayah; natural-language search; satu integrasi CRM; billing self-serve; batas plan lengkap; pengelompokan multi-cabang; band revenue (bila ada data acuan); bahasa Inggris; error tracking; perluasan E2E |
| **F3 — Make it fast** | Skala & ketahanan | Server aplikasi kedua (HA), replika DB, object storage S3, cache, deploy tanpa downtime, optimasi query untuk dataset > 5 juta, mesin search terpisah hanya bila Postgres tidak cukup |

## 7. Data input

### 7.1 Dataset bisnis (JSON-LD schema.org, D-24)

Satu record = satu tempat usaha (Google Place). Contoh field dan pemakaiannya:

| Field JSON-LD | Dipakai untuk | Ada? |
|---|---|---|
| `identifier[google_place_id]` | **Kunci unik** (dedupe & re-import) | ✓ |
| `name`, `description` | Identitas | ✓ |
| `@type`, `servesCuisine` | Kategori (mis. Restaurant → F&B / Restoran Asia) | ✓ |
| `telephone` | Kontak, deteksi seluler → kemungkinan WhatsApp | ✓ |
| `url` | Klasifikasi website (domain sendiri / link sosial / linktree / kosong) | ✓ |
| `sameAs` | Akun sosial (Instagram, TikTok, …) | ✓ |
| `address` (jalan, kota, provinsi, kode pos) | Alamat tampilan | ✓ |
| `geo` (lat, lng) | Penetapan desa/kelurahan | ✓ |
| `openingHoursSpecification` | Kelengkapan listing | ✓ |
| `aggregateRating` (rating, jumlah ulasan) | Reputasi, popularitas | ✓ |
| `hasMap` / `@id` | Tautan "Lihat di Google Maps" | ✓ |
| `image` | Tidak dipakai di F1 | — |
| Email, nama pemilik/person | Tidak ada → "Tidak diketahui" | ✗ |

Pertanyaan terbuka: jumlah record, cakupan wilayah, tanggal pengambilan, dan kadens pembaruan (Q-02, Q-03).

### 7.2 Data wilayah (GeoAPI, D-25)

`geoapi.mcpke.dev` menyediakan untuk Indonesia: provinsi, kab/kota, kecamatan, dan desa/kelurahan (81.912 area). Atribut tiap area: kode, nama, induk, luas, centroid, batas wilayah (GeoJSON & vector tiles), dan **populasi Kontur 2023 (hasil pemodelan)**. Data area di-sync ke database kita (batas 120 request/menit). Atribusi sumber wajib ditampilkan.

## 8. Glosarium

| Istilah | Arti |
|---|---|
| Prospek | Satu tempat usaha dari dataset (1 `google_place_id`). Pengelompokan multi-cabang ada di F2 |
| Workspace | Akun pelanggan. Data workspace terisolasi |
| Offering | Apa yang dijual workspace (tag: `website_dev`, `social_media`, `local_seo`, `digital_ads`, `reputation_mgmt`, `booking_system`, `other`) |
| ICP | Kriteria target workspace. Terikat ke satu Offering |
| Opportunity | Gap yang relevan dengan Offering, lengkap dengan alasan dan evidence |
| Claim type | Teramati (dari sumber) · Turunan (aturan) · Perkiraan (inferensi) · Tidak diketahui |
| Unlock | Membuka nilai kontak sebuah prospek untuk workspace, permanen |

---

## 9. Functional requirements (F1)

| ID | Requirement | Acceptance (ringkas) |
|---|---|---|
| **AUTH** | | |
| FR-AUTH-01 | Daftar/login via **Ory Kratos**: email + password, login Google, verifikasi email (D-20) | Tanpa verifikasi email, data tidak bisa diakses |
| FR-AUTH-02 | Workspace dibuat saat daftar. Admin bisa mengundang anggota. Role: **Admin** (kelola anggota, ICP, webhook, lihat usage) dan **Member** (pakai fitur) | Aksi di luar role → 403 |
| FR-AUTH-03 | Akun internal ops wajib MFA (TOTP) | Login ops tanpa MFA ditolak |
| **DATA** | | |
| FR-DATA-01 | Import file dataset JSON-LD lewat `msp tool import`. Upsert berdasarkan `google_place_id`. Record mentah disimpan sebagai evidence. Hasilnya laporan: baru, berubah, tidak berubah, ditolak (per alasan) | Import ulang file yang sama tidak membuat duplikat |
| FR-DATA-02 | Normalisasi sesuai BR-NORM: telepon, URL, kategori, jam buka, rating | Unit test per aturan |
| FR-DATA-03 | Sync area GeoAPI (4 level, populasi, batas sederhana, atribusi) lewat `msp tool geo-sync` | Semua area level 1–4 Indonesia tersimpan |
| FR-DATA-04 | Penetapan wilayah: koordinat prospek → desa/kelurahan (+ kecamatan, kab/kota, provinsi) dengan point-in-polygon di database | ≥ 98% prospek ber-koordinat mendapat kelurahan |
| **INTL** | | |
| FR-INTL-01 | Setiap nilai yang tampil punya claim type dan sumber (BR-TRUST) | Evidence bisa dibuka dari profil |
| FR-INTL-02 | Deteksi opportunity sesuai katalog Lampiran B | Tiap opportunity punya alasan yang bisa dibaca sales |
| FR-INTL-03 | Metrik wilayah sesuai Lampiran C | Label "Populasi: perkiraan model Kontur 2023" selalu tampil |
| FR-INTL-04 | ICP fit dan priority score sesuai Lampiran A, dengan breakdown faktor dan versi model | Hasil sama untuk input + versi yang sama |
| FR-INTL-05 | Tombol "Relevan / Tidak relevan" pada opportunity | Tersimpan per workspace |
| **ICP** | | |
| FR-ICP-01 | Wizard onboarding: pilih Offering → isi ICP (kategori, wilayah, kondisi digital, minimum rating/ulasan; tiap kriteria *wajib* atau *preferensi*) → hasil pertama | Hasil pertama ≤ 5 menit sejak daftar |
| FR-ICP-02 | Workspace bisa menyimpan beberapa ICP. Tiap user memilih satu ICP aktif | Ganti ICP → skor dan urutan berubah |
| **SRCH / PROF / GEO** | | |
| FR-SRCH-01 | Search kata kunci (nama, kategori, alamat) + filter (kategori, wilayah, band skor, tipe opportunity, ada telepon/website/sosial, rating) + urut skor | Jumlah hasil konsisten dengan filter |
| FR-SRCH-02 | Simpan pencarian (sekaligus berfungsi sebagai segment dinamis) | Pencarian tersimpan bisa dibuka ulang |
| FR-PROF-01 | Profil prospek: identitas, skor + "Kenapa prospek ini?", opportunity + alasan, kontak (ter-mask sampai unlock), kehadiran digital, konteks wilayah, evidence tiap nilai, tautan Google Maps | Setiap nilai menampilkan claim type |
| FR-GEO-01 | Peringkat wilayah (kecamatan dan kelurahan dalam kab/kota pilihan) untuk ICP aktif, peta berwarna dari tiles GeoAPI, drilldown ke daftar prospek | Peta selalu punya padanan tabel |
| **ACT** | | |
| FR-ACT-01 | List, tag, catatan, dan pemilik (assign ke anggota) untuk prospek | — |
| FR-ACT-02 | Unlock kontak: 1 credit per prospek, permanen per workspace. Yang di-mask hanya **nomor telepon**; website dan akun sosial selalu terlihat | Unlock ganda tidak menagih ulang |
| FR-ACT-03 | Export CSV/XLSX sebagai job async dengan preview biaya, maksimal 10.000 baris per job. Prospek yang belum di-unlock ikut di-unlock saat export **berhasil** | Gagal → tidak ditagih |
| FR-ACT-04 | Webhook: daftar URL HTTPS (ditolak bila mengarah ke IP privat/internal), payload ber-tanda tangan HMAC, retry otomatis, log pengiriman, kirim prospek terpilih | Uji SSRF lulus |
| FR-ACT-05 | AI brief per prospek: **Claude**, fallback ke API **OpenAI-compatible** (D-15). Hanya dari fakta profil, setiap kalimat merujuk fakta. Cache 7 hari. Ada kill switch | Brief tanpa rujukan fakta ditolak lalu dibuat ulang sekali |
| **BILL** | | |
| FR-BILL-01 | Satu ledger dengan dua meter: `credit` dan `ai_credit` (grant, debit, refund), idempotent. Ops memberi credit secara manual. Saldo terlihat di workspace | Saldo = jumlah ledger per meter |
| **OPS** | | |
| FR-OPS-01 | Halaman ops: daftar workspace + grant credit, riwayat import, ringkasan kesehatan data (jumlah prospek, % terisi per field, % terpetakan wilayah), feature flag (AI, export, webhook), suppression list | Hanya akun ops |
| FR-OPS-02 | Audit log append-only untuk login, undang anggota, unlock, export, webhook, grant credit, flag, suppression | Tidak ada update/delete |
| **PRIV** | | |
| FR-PRIV-01 | Isolasi data per workspace di database (row-level security) | Test lintas workspace lulus |
| FR-PRIV-02 | Nilai kontak di suppression list disembunyikan dari tampilan, export, dan webhook. Prospek yang kontaknya ter-suppress tidak bisa di-unlock dan tidak ditagih | — |

---

## 10. Business rules

| ID | Aturan |
|---|---|
| BR-TRUST-01 | Claim type: **Teramati** (langsung dari dataset), **Turunan** (aturan deterministik, mis. kategori, tipe telepon), **Perkiraan** (hipotesis, mis. "kemungkinan WhatsApp", opportunity), **Tidak diketahui** (field kosong) |
| BR-TRUST-02 | Freshness dari tanggal pengambilan data: Baru ≤ 180 hari, Menua ≤ 365 hari, Usang > 365 hari. Record tanpa tanggal sendiri memakai tanggal pengambilan dataset yang diisi saat import |
| BR-TRUST-03 | "Tidak diketahui" tidak pernah ditampilkan sebagai kosong, "0", atau "—" |
| BR-NORM-01 | Telepon → E.164 (+62) dengan nilai mentah tetap disimpan. Nomor seluler (+628…) → "Kemungkinan WhatsApp" (Perkiraan) |
| BR-NORM-02 | URL website diklasifikasi: domain sendiri · profil sosial (instagram.com, facebook.com, tiktok.com) · link aggregator (linktr.ee, dll.) · link pesan (wa.me) · subdomain gratis (wixsite, blogspot, dll.) · kosong. Daftar pola disimpan sekali sebagai data (DRY) |
| BR-NORM-03 | Kategori: `@type` schema.org + `servesCuisine` dipetakan ke taksonomi MSP dua level lewat satu tabel mapping |
| BR-SCORE-01 | Rumus di Lampiran A. Faktor yang nilainya tidak diketahui dikeluarkan dan bobotnya dibagi ulang. `coverage` = bobot faktor yang diketahui / total bobot |
| BR-SCORE-02 | Band: **HOT** ≥ 80, **WARM** 60–79, **COLD** < 60. Bila coverage < 0,6 atau kriteria ICP *wajib* tidak diketahui → maksimal WARM, dengan label "Data terbatas" |
| BR-SCORE-03 | Persentil memakai `percent_rank()` Postgres dalam kelompok pembanding (kategori × kab/kota) |
| BR-OPP-01 | Klaim "tidak ada X" hanya dibuat bila field X terisi ≥ 50% di kategori × kab/kota yang sama (minimal 20 prospek). Bila tidak, opportunity tersebut tidak dibuat |
| BR-CRED-01 | Unlock 1 credit/prospek, permanen. Export **atau kirim ke webhook** untuk prospek yang belum di-unlock = auto-unlock, ditagih hanya bila berhasil. AI brief yang gagal tidak ditagih. Semua debit memakai idempotency key. Refund dilakukan manual oleh ops |
| BR-AI-01 | Ke LLM hanya dikirim fakta level bisnis. Nomor telepon tidak dikirim |

## 11. Non-functional requirements

| ID | Requirement |
|---|---|
| NFR-01 | Search p95 ≤ 1 s, profil p95 ≤ 500 ms, untuk ukuran dataset saat ini (Q-02) |
| NFR-02 | Export 10.000 baris ≤ 2 menit. AI brief ≤ 20 s |
| NFR-03 | Ketersediaan F1 99% per bulan (satu server). Naik ke 99,5% di F3 |
| NFR-04 | Backup: RPO ≤ 15 menit, RTO ≤ 8 jam (F1). Restore diuji sebelum rilis F1 |
| NFR-05 | HTTPS, isolasi workspace (RLS), secret tidak ada di repo, pengecekan SSRF untuk webhook |
| NFR-06 | WCAG 2.2 AA dasar: kontras, keyboard, status tidak hanya dengan warna |
| NFR-07 | Bahasa Indonesia (F1), format Rupiah/angka/tanggal id-ID, zona waktu Asia/Jakarta |
| NFR-08 | Browser: 2 versi terakhir Chrome, Edge, Firefox, Safari. Desktop utama; profil dan list bisa dipakai di ponsel |
| NFR-09 | Test otomatis untuk logika inti (normalisasi, skor, opportunity, credit, isolasi workspace). Coverage paket inti ≥ 80% |

## 12. Gerbang rilis F1

1. Alur end-to-end berjalan di staging dengan dataset nyata: import → wilayah → skor → search → profil → unlock → export/webhook → AI brief.
2. Setiap nilai di profil menampilkan claim type dan sumber.
3. Test otomatis lulus, termasuk isolasi workspace, SSRF webhook, dan idempotensi credit.
4. Restore backup sudah dicoba sekali dan berhasil.
5. Founder mencoba alur onboarding: hasil pertama ≤ 5 menit.

## 13. Risiko

| Risiko | Mitigasi |
|---|---|
| Dataset tidak punya email/person → nilai kontak lebih rendah dari kompetitor | Fokus ke telepon/WhatsApp + sosial; upload daftar customer di F2 |
| Populasi adalah perkiraan model, bukan sensus | Label jelas di UI dan laporan |
| Satu server = titik gagal tunggal | Backup off-site + RTO 8 jam di F1; HA di F3 |
| Opportunity dirasa tidak relevan | Offering + tombol feedback (FR-INTL-05), kalibrasi bobot |

---

## Lampiran A — Priority score `priority-2.0.0`

Disederhanakan dari MVP Pack Rev 1.2 §9. Faktor tanpa data di dataset (signal, revenue) ditunda ke F2.

| Faktor | Bobot | Nilai 0–100 |
|---|---|---|
| ICP Fit | 30 | Rata-rata kecocokan kriteria ICP (cocok 1, sebagian 0,5, tidak 0). Kriteria *wajib* yang tidak cocok → prospek tidak tampil |
| Opportunity | 25 | Skor opportunity relevan tertinggi (Lampiran B) |
| Wilayah | 15 | `market_score` kelurahan. Bila kelurahan "Data terbatas", pakai kecamatan |
| Kontak | 15 | Seluler 80 · telepon tetap 60 · hanya akun sosial 30 · tidak ada 0 |
| Popularitas | 10 | Persentil jumlah ulasan dalam kategori × kab/kota |
| Kelengkapan data | 5 | % terisi dari 5 field: telepon, website, jam buka, rating, koordinat |

`priority = Σ(bobot × nilai) / Σ(bobot faktor yang diketahui)`. Bobot bisa diubah per Offering lewat satu file konfigurasi berversi.

## Lampiran B — Katalog opportunity v0

`skor = severity × relevansi`. Relevansi = 1 bila tag Offering primer cocok, 0,6 bila sekunder, 0 bila tidak relevan (gap tetap terlihat di kartu digital, tapi tidak disebut opportunity).

| Kode | Aturan | Severity | Offering primer / sekunder |
|---|---|---|---|
| OPP-WEB-NONE | `url` kosong (dengan syarat BR-OPP-01) | 80 | website_dev / local_seo, digital_ads |
| OPP-WEB-SOCIAL | `url` berupa profil sosial, link aggregator, atau link pesan | 70 | website_dev / social_media |
| OPP-WEB-FREE | `url` di subdomain gratis | 55 | website_dev / — |
| OPP-SOCIAL-NONE | `sameAs` kosong (dengan syarat BR-OPP-01) | 50 | social_media / digital_ads |
| OPP-LIST-INCOMPLETE | ≥ 2 dari {telepon, website, jam buka} kosong | 50 | local_seo / website_dev |
| OPP-REP-LOW-RATING | Rating < 4,0 dengan ≥ 20 ulasan | 70 | reputation_mgmt / local_seo |
| OPP-REP-FEW-REVIEWS | Ulasan < persentil 25 kategori × kab/kota (≥ 10 pembanding) | 55 | reputation_mgmt, local_seo / digital_ads |
| OPP-MKT-COMPETITIVE | Kompetisi kelurahan = Tinggi (Lampiran C) | 50 | digital_ads, local_seo / reputation_mgmt |

Contoh alasan: *"Website mengarah ke linktr.ee, bukan domain sendiri. 61% restoran di Kota Semarang memiliki website sendiri."*

## Lampiran C — Metrik wilayah `geo-2.0.0`

Dihitung per kelurahan dan kecamatan. Satuan densitas **per 1.000 penduduk**.

| Metrik | Rumus |
|---|---|
| Populasi, luas | Dari GeoAPI (Kontur 2023, perkiraan model) |
| Jumlah bisnis / kategori | Jumlah prospek di area (total / kategori ICP) |
| Densitas bisnis / kategori | jumlah / populasi × 1.000 |
| Kompetisi | Persentil densitas kategori di antara kelurahan dalam kab/kota yang sama. Rendah < 33, Sedang 33–66, Tinggi > 66 |
| Data terbatas | Populasi < 500 atau jumlah bisnis < 5 → metrik tetap tampil, tetapi `market_score` tidak dihitung |
| `market_score` | 0,40·pct(populasi) + 0,35·pct(densitas prospek cocok ICP) + 0,25·(100 − kompetisi) |
| Peringkat wilayah | Urut `market_score`, lalu jumlah prospek HOT + WARM |

## Lampiran D — Credit & plan (angka awal, divalidasi pricing)

| Operasi | Biaya |
|---|---|
| Search, profil, peta | Gratis |
| Unlock kontak / export prospek yang belum di-unlock | 1 credit per prospek |
| AI brief | 1 AI credit (gratis bila diambil dari cache) |

| Plan | Seat | Credit/bulan | AI credit/bulan |
|---|---|---|---|
| Starter | 2 | 200 | 50 |
| Growth | 5 | 1.000 | 300 |
| Agency | 15 | 3.000 | 1.000 |
