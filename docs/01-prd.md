# 01 — Product Requirements Document (PRD)

## MesinProspek — Sales Intelligence Engine (Sellable MVP)

| Field | Value |
|---|---|
| Dokumen | PRD |
| Versi | 0.1 (Draft) |
| Tanggal | 2026-10-10 |
| Basis | MVP Pack v1.2 + `00-analisis-dokumen-awal.md` |
| Status crawler | **PAUSED** (D-01). Input = dataset yang sudah ada + upload customer (D-02). |
| Infra | Dedicated server, CI/CD tanpa GitHub (D-18, D-19) |
| Audience | Founder, Product, Design, Engineering, QA, Sales |
| Menggantikan | Bagian MVP Pack yang bertentangan (I-01 s.d. I-14). Prinsip MVP Pack tetap berlaku. |

> Konvensi: `FR-<AREA>-nnn` functional requirement, `NFR-<CAT>-nn` non-functional, `BR-<AREA>-nn` business rule. Prioritas **P0** = wajib untuk release yang disebut, **P1** = sebaiknya ada, **P2** = nanti. Kolom *Rel* = release target (lihat §6). Keputusan berlabel D-xx masih **Proposed** kecuali disebut lain (lihat analisis §7).

---

## 1. Ringkasan produk

MesinProspek adalah platform **sales intelligence multi-tenant** untuk penjual B2B yang menargetkan bisnis lokal di Indonesia. Platform ini mengubah data bisnis yang sudah tersedia menjadi prospek yang **dapat dicari, dapat dipercaya, diprioritaskan, dan siap ditindaklanjuti**.

**Core promise:** *"Jangan hanya menemukan prospek. Pahami, prioritaskan, dan siapkan prospek tersebut untuk ditindaklanjuti."*

**Positioning:** pemain lead database seperti Coldlytics menjawab "siapa yang cocok dan bagaimana kontaknya". MesinProspek menjawab pertanyaan berikutnya: mengapa account ini layak dikejar, apa opportunity-nya, evidence apa yang mendukung, seberapa yakin datanya, dan tindakan apa yang masuk akal.

## 2. Masalah

| # | Masalah user | Dampak hari ini |
|---|---|---|
| P-1 | Daftar bisnis mentah (Maps, direktori, spreadsheet) tidak memberi tahu mana yang paling layak dihubungi. | Sales menghabiskan waktu untuk prospek yang tidak cocok. |
| P-2 | Kualitas kontak tidak jelas: nomor mati, email salah, data kedaluwarsa. | Outreach gagal, reputasi rusak, biaya terbuang. |
| P-3 | Riset sebelum menghubungi prospek dilakukan manual (cek website, ulasan, kompetitor sekitar). | 10–20 menit per prospek, tidak skalabel. |
| P-4 | Manajer tidak punya dasar data untuk memilih wilayah (kecamatan/kelurahan) yang paling potensial. | Territory dibagi berdasarkan intuisi. |
| P-5 | Hasil riset tidak mengalir ke workflow (CRM, WhatsApp, spreadsheet tim). | Copy-paste, data tercecer. |

## 3. Target user & persona

**Beachhead (D-16):** penjual B2B yang menargetkan bisnis lokal Indonesia yang punya lokasi fisik: agency digital (website, iklan, SEO lokal), vendor SaaS untuk UMKM (booking, POS, CRM), dan distributor. Vertikal contoh: klinik gigi/kesehatan, F&B, retail.

| ID | Persona | Konteks | Job-to-be-done | Fitur kunci |
|---|---|---|---|---|
| PER-01 | **Owner/BD agency digital** | Agency kecil (2–15 orang) di kota tier 1–2, menjual website/SEO/ads ke klinik & F&B | "Temukan bisnis di area saya yang jelas butuh layanan saya, lalu beri saya alasan untuk menghubungi mereka." | ICP + Offering, opportunity, brief, export ke WhatsApp/sheet |
| PER-02 | **SDR / sales rep vendor SaaS** | Target harian 30–60 akun baru | "Beri saya antrean prospek terprioritas dengan kontak yang bisa dipercaya." | Search, priority, contact confidence, segment, webhook ke CRM |
| PER-03 | **Sales manager / head of sales** | Membagi territory ke 3–20 rep | "Tunjukkan wilayah mana yang paling potensial dan siapa yang harus mengerjakannya." | Location intelligence, territory ranking, assignment |
| PER-04 | **Workspace owner/admin** | Pemilik akun pelanggan | "Kelola user, credit, dan integrasi tanpa ribet." | Users & roles, usage, webhook |
| PER-05 | **Ops admin MesinProspek (internal)** | Tim internal | "Pastikan data dan pipeline sehat, dan tangani masalah kualitas." | Ops console, merge review, dataset import |

## 4. Tujuan & non-tujuan

### 4.1 Tujuan (MVP 1)
- **OBJ-1 Relevansi.** Dalam ≤ 5 menit sejak sign-up, user baru melihat daftar prospek yang cocok dengan ICP-nya dan memahami mengapa urutan teratas berada di atas.
- **OBJ-2 Kepercayaan.** Setiap claim penting menampilkan jenis claim (observed/derived/inferred), confidence, freshness, dan evidence.
- **OBJ-3 Aksi.** User dapat memilih sekumpulan prospek dan menjalankan minimal satu aksi lanjutan (export, webhook, assign).
- **OBJ-4 Ekonomi.** Penggunaan berbayar tercatat akurat, idempotent, dan dapat di-refund.
- **OBJ-5 Operasional.** Tim internal dapat melihat penurunan kualitas data atau pipeline sebelum customer mengeluh.

### 4.2 Non-tujuan (eksplisit)
- Akuisisi data / crawler (**PAUSED**, D-01).
- CRM penuh, pipeline deal, forecasting.
- Email sequencing, outbound automation, outreach otonom.
- Marketplace riset manusia, browser extension (V2+).
- Aplikasi mobile native (web responsif cukup).
- Pembayaran self-serve di MVP 1 (invoice manual + grant credit oleh ops; self-serve di V1).

## 5. Success metrics

| Metrik | Definisi | Target beta (MVP 1) | Sumber |
|---|---|---|---|
| **North star: Actioned prospects / workspace / minggu** | Prospek unik yang di-export, dikirim ke webhook, atau di-assign | ≥ 50 | event `activity` |
| Activation rate | Workspace baru yang membuat ICP **dan** menyimpan segment/list dalam 7 hari | ≥ 60% | event |
| Time-to-first-result | Sign-up → hasil pencarian pertama dengan ICP aktif | median ≤ 5 menit | event |
| Search-to-save rate | Sesi pencarian yang berakhir dengan save/list/segment | ≥ 25% | event |
| Profile open rate | Profil dibuka / baris hasil ditampilkan | ≥ 8% | event |
| Opportunity acceptance | Opportunity bertanda "Relevan" / opportunity yang diberi feedback | ≥ 60% | FR-INTL-010 |
| Verified contact rate | Contact point VERIFIED + LIKELY_VALID / total contact point tampil | ≥ 70% | data |
| Duplicate merge precision | Merge benar / merge disampel (n ≥ 200 per release) | ≥ 98% | QA sampling |
| Duplicate recall | Duplikat terdeteksi / duplikat di golden set | ≥ 90% | golden set |
| Evidence coverage | Claim tampil yang punya evidence / total claim tampil | ≥ 95% | data |
| AI brief grounding | Brief yang lolos validasi sitasi / brief diminta | ≥ 98% | AI eval |
| Webhook success | Terkirim (≤ retry policy) / dicoba | ≥ 99% | delivery log |
| Week-4 retention | Workspace aktif di minggu ke-4 | ≥ 40% | event |
| Gross margin / enriched lead | (Pendapatan − biaya data/AI variabel) / lead | Ditetapkan setelah pricing | billing |

## 6. Scope per release

| Release | Tema | Isi utama | Tidak dimasukkan |
|---|---|---|---|
| **MVP 0** (internal alpha; memakai workspace internal + ICP/Offering fixture yang dibuat ops untuk menguji engine) | Canonical intelligence | Import dataset snapshot, normalisasi, ER/dedupe, verifikasi, evidence, area reference data, klasifikasi industri, skor (quality, contactability, commercial, geo, ICP fit engine), opportunity v0, priority score, ops console dasar | UI customer lengkap |
| **MVP 1** (closed beta → early paid) | Sellable workflow | Auth & workspace, Offering & ICP, search, profil, location intelligence view & territory ranking, list/segment/tag/note/assign, unlock & credits, export, webhook, AI brief (+ sales angle P1), manual refresh, audit, suppression, upload customer | CRM native, signals, watch |
| **V1** | Recurring value | Signals (diff snapshot), watch & alert, refresh terjadwal, compare (prospek & area), market opportunity report, NL search, satu integrasi CRM, pembayaran self-serve, API key | Enterprise suite |
| **V1.5** | Team workflow | Bobot skor kustom per tenant, territory management, kolaborasi, analytics tim | CRM penuh |
| **V2** | Platform | API publik matang, agency client workspace & white-label, SSO, multi-integrasi; **crawler dapat dihidupkan kembali sebagai source adapter** | Marketing stack |

## 7. Glosarium

| Istilah | Definisi |
|---|---|
| Workspace / Tenant | Akun pelanggan. Semua data tenant-layer terisolasi per workspace. |
| Global layer | Data bisnis milik MesinProspek yang dibaca semua tenant (D-03). |
| Company | Entitas bisnis/brand. Memiliki 1..N Establishment (D-05). |
| Establishment | Lokasi fisik/cabang sebuah Company. Unit geo dan unit skor. |
| Prospect | Baris hasil pencarian: Company beserta Establishment terbaik yang cocok dengan ICP aktif. |
| Contact Point | Email, telepon, WhatsApp, atau kanal lain, milik Company, Establishment, atau Person. |
| Claim | Nilai atribut beserta `claim_type`, `confidence`, `freshness`, dan `evidence_ids`. |
| Evidence | Rujukan sumber sebuah claim (snapshot dataset, baris upload, rule, model). |
| Dataset snapshot | Versi immutable dari dataset yang di-import (`ds-YYYY.MM.n`). |
| ICP | Ideal Customer Profile milik tenant (kriteria target). |
| Offering | Apa yang dijual tenant (D-04). Menentukan relevansi opportunity. |
| Gap fact | Kondisi faktual global, misalnya "tidak ada website pada sumber". |
| Opportunity | Gap fact yang relevan dengan Offering tenant, beserta skor, rationale, dan evidence. |
| Signal | Perubahan atau peristiwa yang meningkatkan relevansi (V1). |
| Area | Unit wilayah administratif: provinsi, kab/kota, kecamatan, kelurahan/desa (kode Kemendagri, D-13). |
| Unlock | Membuka nilai contact point yang ter-mask untuk satu Company, permanen untuk workspace tersebut (D-08). |
| Coverage (skor) | Proporsi bobot faktor yang nilainya diketahui. |

---

## 8. Functional requirements

### 8.1 AUTH — Workspace, autentikasi, role

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-AUTH-001 | Sign-up/login dengan email + password dan Google OAuth melalui identity provider (D-20). Verifikasi email wajib. | P0 | MVP1 | User tidak bisa mengakses data sebelum email terverifikasi. |
| FR-AUTH-002 | Sign-up pertama membuat workspace. Satu user dapat menjadi anggota >1 workspace dan berpindah workspace. | P0 | MVP1 | Data workspace A tidak pernah tampil saat konteks workspace B aktif. |
| FR-AUTH-003 | Undang anggota via email dengan role Owner, Admin, Manager, Member, atau Viewer. Undangan kedaluwarsa dalam 7 hari. | P0 | MVP1 | Role diterapkan segera setelah undangan diterima. |
| FR-AUTH-004 | Permission sesuai matriks RBAC (§8.1.1), ditegakkan di server. | P0 | MVP1 | Request API di luar permission → 403 dan tercatat di audit. |
| FR-AUTH-005 | Manajemen sesi: idle timeout 12 jam, logout semua sesi, cabut sesi saat user dikeluarkan. | P0 | MVP1 | User yang dikeluarkan kehilangan akses ≤ 1 menit. |
| FR-AUTH-006 | MFA wajib untuk akun internal (ops), opsional untuk user tenant. | P0 | MVP0 (ops) / MVP1 (tenant) | Login ops tanpa MFA ditolak. |
| FR-AUTH-007 | API key per workspace (scoped, hashed, dapat dicabut). | P1 | V1 | Key hanya ditampilkan sekali. Key dicabut → 401. |
| FR-AUTH-008 | SSO (SAML/OIDC) untuk Enterprise. | P2 | V2 | — |
| FR-AUTH-009 | Agency client workspace (sub-workspace). | P2 | V2 | — |

#### 8.1.1 Matriks RBAC

| Permission | Owner | Admin | Manager | Member | Viewer |
|---|:-:|:-:|:-:|:-:|:-:|
| Search & lihat profil | ✓ | ✓ | ✓ | ✓ | ✓ |
| Unlock kontak (memakai credit) | ✓ | ✓ | ✓ | ✓ | — |
| Kelola list/segment/tag/note sendiri | ✓ | ✓ | ✓ | ✓ | — |
| Assign prospek ke orang lain | ✓ | ✓ | ✓ | — | — |
| Export / kirim ke webhook | ✓ | ✓ | ✓ | ✓* | — |
| Kelola ICP & Offering | ✓ | ✓ | ✓ | — | — |
| Generate AI brief | ✓ | ✓ | ✓ | ✓ | — |
| Refresh prospek (memakai credit) | ✓ | ✓ | ✓ | ✓ | — |
| Upload daftar sendiri (FR-DATA-012) | ✓ | ✓ | ✓ | ✓ | — |
| Lihat list/segment bersama & AI brief yang sudah ada | ✓ | ✓ | ✓ | ✓ | ✓ |
| Lihat saldo credit workspace | ✓ | ✓ | ✓ | ✓ | ✓ |
| Lihat job (export/upload/refresh) milik sendiri / semua job workspace | ✓ / ✓ | ✓ / ✓ | ✓ / ✓ | ✓ / — | — / — |
| Feedback opportunity/skor/AI, laporkan data | ✓ | ✓ | ✓ | ✓ | — |
| Kelola webhook & integrasi | ✓ | ✓ | — | — | — |
| Kelola anggota & role | ✓ | ✓ | — | — | — |
| Lihat usage & audit log | ✓ | ✓ | Usage saja | — | — |
| Billing & plan, hapus workspace | ✓ | — | — | — | — |

\* Bisa dibatasi Admin melalui pengaturan "Member boleh export" (default: aktif).

#### 8.1.2 Peran internal (ops MesinProspek)

Semua akun internal wajib MFA (FR-AUTH-006) dan setiap aksi tercatat di audit (FR-AUD-001). Satu orang boleh memegang lebih dari satu peran.

| Permission | Ops Engineer (on-call) | Data Steward | Privacy & Billing | Ops Admin |
|---|:-:|:-:|:-:|:-:|
| Lihat dashboard data/pipeline/intelligence/cost | ✓ | ✓ | ✓ | ✓ |
| Feature flag & kill switch, redrive DLQ | ✓ | — | — | ✓ |
| Merge review, koreksi data (FR-OPS-009), triase laporan data | — | ✓ | — | ✓ |
| Review laporan validasi snapshot (publish via pipeline rilis data, `06`) | — | ✓ | — | ✓ |
| Suppression & permintaan subjek data | — | — | ✓ | ✓ |
| Grant/refund credit, ubah plan | — | — | ✓ | ✓ |
| Akses support read-only ke data tenant (dengan alasan) | ✓ | ✓ | ✓ | ✓ |
| Baca audit log internal (tanpa edit/hapus) | ✓ | ✓ | ✓ | ✓ |
| Kelola akun & peran internal | — | — | — | ✓ |

### 8.2 DATA — Fondasi data & pipeline

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-DATA-001 | Ops dapat meng-import **dataset snapshot** yang dipetakan ke Input Contract. Snapshot memiliki versi, sumber, tanggal pengambilan, periode referensi, dan tag lisensi. Import menghasilkan laporan validasi: jumlah baris, reject per alasan, fill-rate per field. | P0 | MVP0 | Snapshot immutable. Baris reject tidak memblokir baris valid. Laporan tersimpan. |
| FR-DATA-002 | Normalisasi field sesuai BR-NORM (telepon E.164, email, domain, nama, alamat → hierarki wilayah). Nilai mentah selalu disimpan. | P0 | MVP0 | Unit test BR-NORM lulus. Nilai mentah dapat ditampilkan di evidence. |
| FR-DATA-003 | Entity resolution Company & Establishment sesuai BR-ER (exact → composite → fuzzy → review queue). | P0 | MVP0 | Precision ≥ 98%, recall ≥ 90% pada golden set. |
| FR-DATA-004 | Kandidat merge ambigu bernilai tinggi masuk review queue. Keputusan ops (merge/split/keep) persisten lintas re-run. | P0 | MVP0 | Re-run tidak membatalkan keputusan manual. |
| FR-DATA-005 | Verifikasi contact point sesuai BR-VER (email: syntax + MX; telepon: format + tipe; domain: DNS). | P0 | MVP0 | Setiap contact point memiliki status, confidence, dan `verified_at` (MVP-02). |
| FR-DATA-006 | Setiap nilai observed memiliki Evidence. Nilai derived/inferred merujuk input dan versi rule/model-nya. | P0 | MVP0 | Evidence coverage ≥ 95% (MVP-03). |
| FR-DATA-007 | Re-run pipeline pada snapshot yang sama atau lebih baru tidak membuat entity duplikat. ID entity stabil. | P0 | MVP0 | Re-run 2× → jumlah entity sama, ID sama (MVP-11). |
| FR-DATA-008 | Import data referensi wilayah: kode wilayah Kemendagri, batas wilayah (wajib untuk peta choropleth FR-GEO-002; tanpa batas, peta turun ke titik/cluster + tampilan tabel), dan populasi per kelurahan/desa beserta periode referensi. Crosswalk ke kode BPS bila diperlukan. | P0 | MVP0 | Setiap metrik populasi memiliki `reference_period` dan `source`. |
| FR-DATA-009 | Penetapan area: setiap Establishment dipetakan ke kelurahan/desa melalui point-in-polygon (bila ada batas) atau pencocokan alamat, dengan `geo_confidence`. | P0 | MVP0 | ≥ 90% establishment ber-koordinat terpetakan ke kelurahan. |
| FR-DATA-010 | Digital presence lite: klasifikasi nilai website (domain sendiri / subdomain platform gratis / profil sosial / link aggregator / link pesan / tidak ada) dan klasifikasi email (domain sendiri / webmail gratis). | P0 | MVP0 | Unit test klasifikasi lulus untuk daftar pola (SDD). |
| FR-DATA-011 | Identifikasi penyedia dari DNS: host MX → penyedia email (mis. Google Workspace, Microsoft 365) dan include SPF → tool email marketing. Memakai cache DNS yang sama dengan pengecekan MX untuk status verifikasi di FR-DATA-005 (FR-DATA-005 hanya menentukan ada/tidaknya MX). | P1 | MVP1 | Hasil DNS di-cache dengan TTL dan dicatat sebagai evidence. |
| FR-DATA-012 | **Upload customer** CSV/XLSX (≤ 50.000 baris per file di MVP 1): mapping kolom, preview validasi, proses async, pencocokan ke entity global (link, bukan merge), laporan match-rate. Data upload privat untuk workspace. Baris yang tidak cocok dengan entity global menjadi **prospek privat** workspace: tampil di search workspace tersebut dengan label "Dari upload Anda" dan dinilai dengan aturan yang sama. Kandidat **"mungkin cocok"** (skor 0,80–0,92, BR-ER-04) ditampilkan untuk dikonfirmasi user (tautkan / jadikan prospek privat); tanpa konfirmasi diperlakukan sebagai prospek privat. | P0 | MVP1 | Data upload tidak pernah terbaca workspace lain. Laporan menampilkan matched/new/rejected. |
| FR-DATA-013 | Diff antar snapshot → change record (input untuk signal). | P1 | V1 | Change record merujuk kedua snapshot. |
| FR-DATA-014 | User dapat melaporkan data salah ("Laporkan data") dari profil. Laporan masuk antrean ops. | P0 | MVP1 | Laporan tercatat dengan claim yang dirujuk. |

### 8.3 ICP — Offering & Ideal Customer Profile

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-ICP-001 | Workspace membuat ≥ 1 **Offering**: nama dan *offering tags* dari katalog (website_dev, local_seo, digital_ads, reputation_mgmt, booking_system, crm, pos, email_service, other). | P0 | MVP1 | Offering wajib ada sebelum opportunity ditampilkan. |
| FR-ICP-002 | Workspace membuat **ICP**: kategori/industri, wilayah (provinsi→kelurahan), band karyawan, band revenue, kondisi digital, tipe kontak wajib, dan pengecualian. Setiap kriteria dapat ditandai *wajib* atau *preferensi*. Setiap ICP terikat ke **tepat satu Offering**, sehingga relevansi opportunity selalu memakai Offering milik ICP aktif. | P0 | MVP1 | ICP tersimpan berversi. Perubahan memicu re-score (NFR-PERF-05). |
| FR-ICP-003 | Engine menghitung **ICP fit** 0–100 dengan breakdown per kriteria (match/partial/no/unknown). | P0 | MVP0 | Breakdown tampil di profil. |
| FR-ICP-004 | **Preset bobot skor per tipe Offering** dapat dipilih. Default mengikuti Lampiran A. | P0 | MVP1 | Preset tercatat di `model_version` skor. |
| FR-ICP-005 | Onboarding wizard: Offering → ICP → hasil pertama. | P0 | MVP1 | Median time-to-first-result ≤ 5 menit. |
| FR-ICP-006 | Beberapa ICP tersimpan per workspace (batas per plan). Setiap user memilih satu ICP aktif sebagai konteks pencarian. | P0 | MVP1 | Mengganti ICP aktif mengubah skor & urutan hasil. |
| FR-ICP-007 | Bobot faktor kustom per tenant. | P2 | V1.5 | — |

### 8.4 SRCH — Search & discovery

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-SRCH-001 | Pencarian kata kunci pada nama, alias, kategori, dan alamat. Toleran terhadap variasi penulisan umum (PT/CV, huruf besar, tanda baca). | P0 | MVP1 | "klinik gigi abc" menemukan "ABC Dental Clinic" bila alias cocok. |
| FR-SRCH-002 | Filter: kategori, wilayah (hierarki), band karyawan, band revenue, ketersediaan kontak (telepon/email/website/WA-likely), status verifikasi, band prioritas, rentang ICP fit, tipe opportunity, freshness, status bisnis, rentang rating, "belum ada di list saya", status unlock. | P0 | MVP1 | Kombinasi filter menghasilkan hitungan yang konsisten dengan hasil. |
| FR-SRCH-003 | Sortir: prioritas (default), ICP fit, kualitas data, nama. | P0 | MVP1 | Default = priority ↓. |
| FR-SRCH-004 | Baris hasil menampilkan nama, lokasi establishment, skor + band, badge coverage, opportunity teratas, ikon ketersediaan kontak (ter-mask), dan badge freshness/verifikasi. Multi-cabang ditampilkan "+N cabang". | P0 | MVP1 | Sesuai UX spec. |
| FR-SRCH-005 | Saved search: simpan query + filter + ICP, dapat dijalankan ulang. | P0 | MVP1 | — |
| FR-SRCH-006 | Pagination dengan total hitungan (eksak ≤ 10.000, perkiraan di atasnya, diberi label "±"). | P0 | MVP1 | — |
| FR-SRCH-007 | Bulk select (halaman ini / semua hasil hingga batas plan) → aksi massal. | P0 | MVP1 | Batas ditampilkan sebelum aksi. |
| FR-SRCH-008 | Zero-result guidance: saran melonggarkan filter tertentu. | P0 | MVP1 | — |
| FR-SRCH-009 | Natural-language search → diterjemahkan menjadi filter kanonik yang ditampilkan sebagai chip dan dapat diedit. | P1 | V1 | User selalu melihat filter hasil parsing. |
| FR-SRCH-010 | Tampilan peta untuk hasil (clustered). | P1 | V1 | — |

### 8.5 PROF — Prospect profile

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-PROF-001 | Header: identitas (nama, kategori, status bisnis, alamat, jumlah cabang), priority score + band + coverage, ICP fit, opportunity teratas. | P0 | MVP1 | Hierarki informasi mengikuti MVP Pack §07. |
| FR-PROF-002 | Panel **"Kenapa prospek ini?"**: setiap faktor menampilkan nilai, bobot, kontribusi, jenis claim, serta model version dan `calculated_at`. | P0 | MVP1 | MVP-06. Faktor Unknown ditampilkan sebagai "Tidak diketahui" (tidak dihitung). |
| FR-PROF-003 | Kontak: contact point level company/establishment/person beserta tipe, status, confidence, dan terakhir diverifikasi. Nilai ter-mask sampai unlock (D-08). Setelah unlock tersedia copy, klik-WhatsApp, dan klik-telepon. | P0 | MVP1 | MVP-02. Suppressed contact tidak tampil. |
| FR-PROF-004 | **Evidence drawer** untuk setiap claim: sumber (nama dataset + versi snapshot + ref record / nama file upload + baris / rule + input / model + versi prompt) dan `observed_at`. | P0 | MVP1 | MVP-03. Dapat dibuka tanpa pengetahuan teknis. |
| FR-PROF-005 | Daftar establishment (cabang) beserta lokasi dan konteks area. | P0 | MVP1 | — |
| FR-PROF-006 | Kartu konteks lokasi: kelurahan, populasi (dengan periode), densitas kategori, level kompetisi, skor opportunity area, label guardrail. | P0 | MVP1 | Label periode referensi selalu tampil. |
| FR-PROF-007 | Kartu komersial: band karyawan, band revenue (Inferred: confidence, basis, as-of). | P0 | MVP1 | Tidak pernah ditampilkan sebagai fakta terverifikasi. |
| FR-PROF-008 | Kartu digital presence lite: tipe website, domain sendiri, tipe email, penyedia email (bila diketahui), rating & jumlah ulasan. | P0 | MVP1 | Field yang tidak tersedia = "Tidak diketahui". |
| FR-PROF-009 | Bar aksi: simpan ke list, tag, catatan, assign, unlock, export, kirim ke webhook, AI brief, refresh, laporkan data. | P0 | MVP1 | Aksi yang tidak diizinkan role disembunyikan. |
| FR-PROF-010 | Timeline aktivitas (privat workspace). | P1 | MVP1 | — |
| FR-PROF-011 | Bagian signal. | P1 | V1 | — |
| FR-PROF-012 | Compare 2–5 prospek berdampingan (skor, faktor, kontak, konteks area). | P1 | V1 | — |

### 8.6 INTL — Intelligence engine

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-INTL-001 | Klasifikasi industri: kategori sumber dipetakan ke taksonomi MSP dua level (sektor → kategori) dengan confidence. Tidak terpetakan → "Lainnya". | P0 | MVP0 | ≥ 95% establishment aktif terpetakan ke kategori non-"Lainnya" (target, divalidasi Q-01). |
| FR-INTL-002 | Skor Data Quality (Lampiran A). | P0 | MVP0 | — |
| FR-INTL-003 | Skor Contactability (Lampiran A). | P0 | MVP0 | — |
| FR-INTL-004 | Commercial potential: band karyawan dan band revenue v0 (BR-REV). | P0 | MVP0 | Confidence revenue inferred ≤ 70 (BR-REV-03). |
| FR-INTL-005 | Faktor Digital Opportunity dari digital presence lite + opportunity keluarga DIGITAL. | P0 | MVP0 | Unknown bila data digital tidak ada. |
| FR-INTL-006 | Faktor Geographic Opportunity dari metrik area (Lampiran C). Memakai kelurahan. Bila coverage kelurahan LOW, fallback ke kecamatan. Bila keduanya LOW → Unknown. | P0 | MVP0 | Sumber level (kelurahan/kecamatan) tercatat di breakdown. |
| FR-INTL-007 | **Opportunity detection** sesuai katalog v0 (Lampiran B): gap fact global + relevansi per Offering, lengkap dengan rationale, evidence_ids, dan confidence. | P0 | MVP0 | MVP-07. Setiap opportunity bisa dibuka ke evidence. |
| FR-INTL-008 | **Priority score** per (workspace, ICP, establishment) + rollup company, beserta breakdown, versi, coverage, dan band. | P0 | MVP0 | MVP-06. Deterministik untuk input + versi yang sama. |
| FR-INTL-009 | Versioning model skor. Perubahan konfigurasi → versi baru + job recompute. Versi lama tetap tersimpan untuk audit/perbandingan. | P0 | MVP0 | Skor lama dapat direproduksi dari versi + snapshot. |
| FR-INTL-010 | **Feedback user**: opportunity "Relevan / Tidak relevan (+alasan)", skor "Terlalu tinggi / Terlalu rendah". | P0 | MVP1 | Feedback tersimpan per workspace/versi model, dan agregat acceptance rate tersedia lewat laporan ops sederhana. Dashboard lengkap ada di FR-OPS-003 (P1). |
| FR-INTL-011 | Skor yang lebih tua dari input-nya ditandai **stale** dan diantrekan untuk recompute. | P0 | MVP0 | Badge stale tampil sampai recompute selesai. |
| FR-INTL-012 | Signal detection dari change record (katalog signal V1). | P1 | V1 | — |

### 8.7 GEO — Geographic & market intelligence

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-GEO-001 | Metrik area per kelurahan/kecamatan/kab-kota sesuai Lampiran C, dengan periode referensi dan indikator coverage. | P0 | MVP0 | Reproducible dari snapshot + `geo_version`. |
| FR-GEO-002 | **Location Intelligence View**: peta choropleth (kelurahan/kecamatan) dengan metrik yang dapat dipilih + panel metrik area. | P0 | MVP1 | Legend, periode, dan guardrail selalu tampil. |
| FR-GEO-003 | **Territory ranking**: peringkat area dalam wilayah pilihan untuk ICP + Offering aktif (territory score, Lampiran C). | P0 | MVP1 | Setiap baris dapat dibuka ke breakdown. |
| FR-GEO-004 | Drilldown area → prospek (search terfilter ke area, urut prioritas). | P0 | MVP1 | — |
| FR-GEO-005 | Buat segment dinamis dari area / opportunity zone. | P1 | MVP1 | — |
| FR-GEO-006 | Compare areas (2–5 area). | P1 | V1 | — |
| FR-GEO-007 | Market opportunity report (dapat dibagikan). | P1 | V1 | — |
| FR-GEO-008 | Watch area + alert saat supply bisnis target berubah material. | P2 | V1 | — |
| FR-GEO-009 | Guardrail tampilan: densitas populasi = konteks, bukan demand; densitas bisnis = proxy supply, bukan market share; label "Data terbatas" untuk coverage LOW. | P0 | MVP1 | Copy guardrail ada di tooltip dan laporan. |

### 8.8 SEG — Organize: list, segment, tag, catatan, assignment

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-SEG-001 | **List statis**: tambah/hapus prospek, beri nama, pemilik, dibagikan ke workspace. | P0 | MVP1 | MVP-08. |
| FR-SEG-002 | **Segment dinamis**: filter + ICP tersimpan. Menampilkan jumlah dan distribusi skor, dan diperbarui otomatis. | P0 | MVP1 | MVP-08. |
| FR-SEG-003 | Tag level workspace, termasuk bulk tag. | P0 | MVP1 | — |
| FR-SEG-004 | Catatan pada prospek (privat workspace). | P0 | MVP1 | — |
| FR-SEG-005 | Assign prospek atau list ke anggota. Tersedia tampilan "Tugas saya". | P0 | MVP1 | Assignment tercatat di activity. |
| FR-SEG-006 | Status kerja ringan per prospek (Baru, Diriset, Siap dihubungi, Sudah dihubungi, Didiskualifikasi). Ini **bukan** pipeline deal. | P1 | MVP1 | — |
| FR-SEG-007 | Antrean siap-kampanye dari segment untuk rep tertentu, urut prioritas. | P1 | V1 | — |

### 8.9 EXP — Export & integrasi

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-EXP-001 | Export CSV/XLSX/JSON dari seleksi/list/segment sebagai job async dengan progres. Field dapat dipilih. Kolom `claim_type`, `confidence`, dan `verified_at` disertakan. Link unduhan kedaluwarsa 24 jam. | P0 | MVP1 | MVP-09. Gagal → status eksplisit + retry. |
| FR-EXP-002 | Biaya export mengikuti BR-CRED. Pratinjau biaya sebelum konfirmasi. | P0 | MVP1 | Tidak ada potongan credit tanpa konfirmasi. |
| FR-EXP-003 | **Webhook endpoint**: daftarkan URL (HTTPS, validasi SSRF), secret, dan langganan event. Mencakup tombol uji, payload bertanda tangan HMAC-SHA256, retry dengan backoff hingga 24 jam, DLQ + kirim ulang manual, dan log pengiriman. | P0 | MVP1 | MVP-09. URL ke IP privat/metadata ditolak. |
| FR-EXP-004 | Aksi "Kirim ke webhook" untuk seleksi. | P0 | MVP1 | — |
| FR-EXP-005 | Record ter-suppress dikecualikan dari export dan webhook. | P0 | MVP1 | — |
| FR-EXP-006 | Batas export per plan dan per job (≤ 50.000 baris). | P0 | MVP1 | — |
| FR-EXP-007 | Satu integrasi CRM native. | P1 | V1 | — |

### 8.10 AI — Brief & sales angle

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-AI-001 | **Prospect brief** dari profil terstruktur. Setiap kalimat mengutip claim/evidence ID. Tidak ada claim di luar data yang tersedia. Diberi label "Dibuat AI" beserta versi model/prompt. Bahasa Indonesia (default) atau Inggris. | P0 | MVP1 | MVP-10. Validator sitasi lulus ≥ 98%. |
| FR-AI-002 | **Sales angle**: hipotesis pendekatan yang terkait Offering + opportunity, beserta rationale. Berlabel Inferred. | P1 | MVP1 | — |
| FR-AI-003 | Cache **per workspace** dengan kunci (workspace, ICP/Offering, hash input, versi prompt, bahasa). Input sama dalam 7 hari → dipakai ulang tanpa biaya. "Generate ulang" dikenai biaya. Cache tidak pernah dibagi lintas workspace. | P0 | MVP1 | Uji isolasi cache lintas workspace. |
| FR-AI-004 | Grounding validator: output dengan klaim tanpa sitasi ditolak, lalu di-regenerate maksimal 1×. Bila tetap gagal → error eksplisit dan refund. | P0 | MVP1 | — |
| FR-AI-005 | Nilai contact point pribadi (email/telepon person) **tidak dikirim** ke LLM. Person cukup direpresentasikan dengan role. | P0 | MVP1 | Diuji otomatis (test redaksi prompt). |
| FR-AI-006 | Batas biaya AI per workspace + kill switch global. | P0 | MVP1 | Kill switch aktif → tombol AI nonaktif dengan pesan. |
| FR-AI-007 | Feedback 👍/👎 + alasan. | P1 | MVP1 | — |

### 8.11 MON — Refresh & monitoring

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-MON-001 | **Refresh manual**: re-verify + re-score pada snapshot terbaru. 1 credit. Tidak membuat entity duplikat. | P0 | MVP1 | MVP-11. |
| FR-MON-002 | Watch company + alert signal (in-app + digest email). | P1 | V1 | — |
| FR-MON-003 | Refresh terjadwal untuk record yang di-watch. | P1 | V1 | — |
| FR-MON-004 | Pusat notifikasi in-app. | P1 | V1 | — |

### 8.12 BILL — Usage, credit, plan

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-BILL-001 | Setiap operasi berbayar menghasilkan **usage event** (idempotency key, workspace, tipe operasi, unit, referensi). | P0 | MVP1 | MVP-14. Event ganda dengan key sama → satu tagihan. |
| FR-BILL-002 | **Credit ledger** append-only (grant, debit, refund, expire), tampilan saldo, dan riwayat. | P0 | MVP1 | Saldo = Σ ledger (rekonsiliasi 100%). |
| FR-BILL-003 | Entitlement plan ditegakkan di server: seat, jumlah ICP, fair-use profil, batas export, AI credit, jumlah webhook. | P0 | MVP1 | Pelanggaran → error ber-kode, bukan hanya disembunyikan di UI. |
| FR-BILL-004 | Pratinjau biaya sebelum aksi massal berbayar. Credit tidak cukup → aksi diblokir dengan pesan jelas. | P0 | MVP1 | — |
| FR-BILL-005 | Refund otomatis sesuai BR-CRED. | P0 | MVP1 | — |
| FR-BILL-006 | Ops dapat memberi credit/plan secara manual (invoice manual di MVP 1). | P0 | MVP1 | Tercatat di audit. |
| FR-BILL-007 | Dashboard usage workspace (per operasi, per anggota). | P1 | MVP1 | — |
| FR-BILL-008 | Pembayaran self-serve melalui payment gateway Indonesia + invoice. | P1 | V1 | — |

### 8.13 AUD — Audit, privasi, governance

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-AUD-001 | **Audit log immutable** untuk login, perubahan role, export, unlock, konfigurasi webhook/API key, suppression, keputusan merge, grant credit, dan akses support. Setiap entri memuat aktor, waktu, IP, dan target. | P0 | MVP1 | MVP-13. Tidak ada endpoint update/delete audit. |
| FR-AUD-002 | Penampil audit log untuk Owner/Admin. | P1 | MVP1 | — |
| FR-AUD-003 | **Suppression list** global (dikelola ops): person/contact point ter-suppress hilang dari search, profil, export, dan webhook ≤ 24 jam, termasuk projection. File export yang belum kedaluwarsa dan memuat record tersebut dibatalkan lalu dibuat ulang tanpa record itu. Workspace yang pernah menerima record (export/webhook) menerima event `suppression.applied` (tanpa nilai kontak) agar dapat menghapus di sistem hilir. | P0 | MVP1 | Uji propagasi ke semua projection, file export aktif, dan event. |
| FR-AUD-004 | Intake **permintaan subjek data** (akses, koreksi, hapus, keberatan) via form/email + workflow internal dengan SLA terpantau. Pengakuan penerimaan ≤ 1 hari kerja. Tenggat pemenuhan mengikuti UU PDP dan aturan pelaksananya (beberapa hak bertenggat 3×24 jam; divalidasi konsultan hukum). Untuk data upload tenant (tenant = pengendali), permintaan diteruskan ke tenant. Verifikasi identitas pemohon memakai OTP email (SMTP). Bila pemohon hanya punya kontak telepon, ops memverifikasi manual lewat panggilan balik ke nomor yang tercatat (tanpa provider SMS di MVP). | P0 | MVP1 | Permintaan tercatat dari masuk hingga selesai, termasuk penerusan ke tenant. |
| FR-AUD-005 | Penutupan workspace → data tenant (termasuk upload) dihapus setelah masa tenggang 30 hari. Pengecualian: `audit_log` dan `credit_ledger`/invoice disimpan sesuai masa retensi hukum dan **dipseudonimkan saat purge hari ke-30** (identitas aktor diganti ID ter-hash, tanpa data pribadi). Masa retensi divalidasi konsultan hukum. | P0 | MVP1 | Job penghapusan terverifikasi. Audit & ledger tetap utuh tetapi tanpa PII. |
| FR-AUD-006 | **Isolasi tenant** ditegakkan di layer DB (RLS) dan aplikasi. | P0 | MVP1 | MVP-12. Uji lintas-tenant otomatis di CI. |
| FR-AUD-007 | Kebijakan retensi per kelas data dapat dikonfigurasi. | P1 | V1 | — |

### 8.14 OPS — Konsol internal

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-OPS-001 | Dashboard **Data Health** (MVP Pack §13): total company, rasio verifikasi, rasio stale, rasio kandidat duplikat, evidence coverage. Tersedia per snapshot. | P0 | MVP0 | — |
| FR-OPS-002 | Dashboard **Pipeline Health**: job selesai/gagal, retry, latensi per stage, backlog. DLQ dapat di-redrive. | P0 | MVP0 | — |
| FR-OPS-003 | **Intelligence Health**: distribusi skor per versi, opportunity acceptance, tingkat kegagalan AI. | P1 | MVP1 | — |
| FR-OPS-004 | **Cost**: biaya per enriched record, per aksi AI. | P1 | MVP1 | — |
| FR-OPS-005 | UI merge review queue (FR-DATA-004). | P0 | MVP0 | — |
| FR-OPS-006 | Admin tenant: daftar workspace, plan, grant credit. Akses support read-only dengan alasan wajib dan tercatat di audit (tanpa impersonation). | P0 | MVP1 | — |
| FR-OPS-007 | Feature flag & kill switch (AI, export, webhook, upload). | P0 | MVP1 | Perubahan flag tercatat di audit. |
| FR-OPS-008 | Antrean laporan data dari user (FR-DATA-014) dan permintaan subjek data (FR-AUD-004). | P0 | MVP1 | — |
| FR-OPS-009 | Koreksi data oleh ops: membuat evidence `ops_correction` (merujuk laporan) yang menang atas data dataset sampai ada observasi lebih baru dari snapshot berikutnya. Koreksi memicu recompute skor terkait. | P0 | MVP1 | Koreksi bertahan saat re-run snapshot yang sama dan tercatat di audit. |

### 8.15 RPT — Laporan

| ID | Requirement | Pri | Rel | Acceptance (ringkas) |
|---|---|---|---|---|
| FR-RPT-001 | Ringkasan segment (jumlah, kualitas, top prospek) → PDF/tautan. | P1 | V1 | — |
| FR-RPT-002 | Laporan white-label. | P2 | V2 | — |

---

## 9. Business rules

### 9.1 BR-TRUST — Claim, confidence, freshness

| ID | Aturan |
|---|---|
| BR-TRUST-01 | Setiap nilai yang tampil memiliki `claim_type` ∈ {OBSERVED, DERIVED, INFERRED, UNKNOWN}. Observed dan inferred tidak boleh tercampur tanpa label. |
| BR-TRUST-02 | `confidence` 0–100 per claim. Untuk beberapa evidence independen yang sepakat: `c = 1 − Π(1 − cᵢ)`, dibatasi maksimum 99. Bila evidence bertentangan: nilai utama = evidence dengan (confidence × kebaruan) tertinggi, sisanya disimpan sebagai *alternate*, dan confidence utama dikurangi 20 poin. |
| BR-TRUST-03 | Level confidence untuk tampilan: **Tinggi** ≥ 80, **Sedang** 50–79, **Rendah** < 50. Angka ditampilkan di tooltip. |
| BR-TRUST-04 | Freshness dihitung dari `observed_at`/`verified_at` terakhir. Contact point: Fresh ≤ 90 hari, Aging 91–180, Stale > 180. Atribut company: Fresh ≤ 180, Aging 181–365, Stale > 365. Data populasi: Fresh bila periode referensi = periode terbit terbaru yang tersedia di sistem, selain itu Aging. |
| BR-TRUST-05 | Nilai INFERRED selalu memakai label "Perkiraan" atau "Hipotesis" beserta basis. Tidak pernah diformat sebagai fakta terverifikasi. |
| BR-TRUST-06 | UNKNOWN ditampilkan sebagai "Tidak diketahui". Tidak pernah kosong, "0", atau "-". |

### 9.2 BR-VER — Status verifikasi contact point

| Status | Email | Telepon | WhatsApp |
|---|---|---|---|
| VERIFIED | Lolos verifikasi provider (deliverable) | Dikonfirmasi sumber terpercaya (V1) | Tidak dipakai di MVP |
| LIKELY_VALID | Syntax valid + domain memiliki MX | Format valid untuk Indonesia + tipe terdeteksi | — |
| RISKY | Catch-all, role-based (info@, admin@), domain baru/tidak stabil, atau tanpa MX tetapi memiliki A/AAAA (implicit MX, RFC 5321) | Format valid tetapi tipe tidak dikenal | — |
| INVALID | Syntax gagal / domain tidak ada (NXDOMAIN) atau tanpa MX maupun A/AAAA / provider undeliverable | Format tidak valid | — |
| UNKNOWN | Belum dicek / gagal sementara | Belum dicek | Default |

- **BR-VER-01:** Ketersediaan WhatsApp dari nomor seluler (+628…) berstatus **INFERRED** ("Kemungkinan WhatsApp"), tidak pernah VERIFIED di MVP.
- **BR-VER-02:** Kegagalan verifikasi sementara (timeout DNS) → UNKNOWN, bukan INVALID (*Unknown, not false*).

### 9.3 BR-NORM — Normalisasi (ringkas; detail di SDD)

- **BR-NORM-01:** Telepon → E.164 (+62). Menangani awalan `0`, `62`, `+62`, kode area tetap, dan pemisah. Nilai mentah disimpan.
- **BR-NORM-02:** Email: domain lowercase, local-part dipertahankan untuk tampilan. Domain webmail gratis ditandai.
- **BR-NORM-03:** Domain → registrable domain berdasarkan Public Suffix List (termasuk `.co.id`, `.my.id`, `.web.id`, `.or.id`, `.ac.id`, `.sch.id`, `.go.id`). URL platform (instagram.com, linktr.ee, wa.me, subdomain platform gratis) **bukan** identitas company.
- **BR-NORM-04:** Nama: normalisasi spasi/tanda baca. Bentuk badan usaha (PT, CV, UD, Tbk, Persero, Koperasi) dan gelar (dr., drg.) dipisahkan menjadi token alias. Nilai asli tidak pernah dihapus.
- **BR-NORM-05:** Alamat dipertahankan untuk tampilan, lalu dipetakan ke kode wilayah Kemendagri (D-13).

### 9.4 BR-ER — Entity resolution

| ID | Aturan |
|---|---|
| BR-ER-01 | Identifier kuat: registrable domain milik sendiri, telepon E.164 yang bukan nomor bersama, ID record sumber. Webmail gratis, nomor call-center bersama, dan URL platform **tidak** dipakai sebagai identifier kuat. |
| BR-ER-02 | Establishment sama: (identifier kuat sama **atau** kemiripan nama ≥ 0,90) **dan** jarak ≤ 50 m (atau alamat ternormalisasi sama). |
| BR-ER-03 | Company sama, establishment berbeda: domain sendiri sama, tetapi lokasi berjarak > 200 m. |
| BR-ER-04 | Skor fuzzy gabungan ≥ 0,92 dan tidak ada identifier kuat yang bertentangan → auto-merge. 0,80–0,92 → review queue bila bernilai tinggi (cocok dengan ≥ 1 ICP aktif atau punya aktivitas tenant), selain itu tetap terpisah. < 0,80 → terpisah. |
| BR-ER-05 | Identifier kuat yang bertentangan (dua domain sendiri berbeda) → tidak pernah auto-merge. |
| BR-ER-06 | Keputusan manual bersifat final terhadap re-run sampai ops mengubahnya. |
| BR-ER-07 | Record upload customer hanya di-**link** ke entity global (confidence link disimpan). Data upload tidak pernah mengubah global layer. |
| BR-ER-08 | Telepon dianggap **nomor bersama** (bukan identifier kuat) bila muncul di ≥ 3 establishment milik company berbeda (nama dan domain berbeda) dalam satu snapshot. |
| BR-ER-09 | Domain sendiri sama dengan jarak 50–200 m → tetap satu Company. Establishment di-merge hanya bila kemiripan nama ≥ 0,90 **dan** alamat ternormalisasi sama. Selain itu dibuat establishment terpisah dengan flag `geo_ambiguous`, dan masuk review queue bila bernilai tinggi. |
| BR-ER-10 | Pengukuran recall: pasangan duplikat yang berakhir di review queue dihitung *terdeteksi*. Auto-recall (tanpa review) dilaporkan terpisah. Rasio review terhadap total kandidat ≤ 5% (target operasional). |

### 9.5 BR-SCORE — Skor

| ID | Aturan |
|---|---|
| BR-SCORE-01 | Unit skor = Establishment dalam konteks (workspace, ICP aktif, Offering). Priority Company = maksimum priority dari establishment yang lolos filter ICP. Baris hasil menampilkan establishment terbaik tersebut. |
| BR-SCORE-02 | Rumus, bobot default, dan definisi faktor ada di **Lampiran A** (bobot Rev 1.2 §9, I-02). |
| BR-SCORE-03 | Faktor UNKNOWN dikeluarkan dan bobotnya di-renormalisasi. `coverage` = Σ bobot faktor yang diketahui / Σ semua bobot. |
| BR-SCORE-04 | `coverage` < 0,60 → badge "Data terbatas" dan band maksimal WARM. ICP Fit UNKNOWN → UNSCORED. |
| BR-SCORE-05 | Band: **HOT** ≥ 80, **WARM** 60–79, **COLD** < 60, **UNSCORED** (D-07). |
| BR-SCORE-06 | Setiap skor menyimpan `value`, `factors[]` (nilai, bobot, kontribusi, claim_type), `model_version`, `calculated_at`, `coverage`, dan `stale`. |
| BR-SCORE-07 | Establishment dengan status tutup permanen dan record ter-suppress dikecualikan dari hasil secara default. |
| BR-SCORE-08 | Agar tidak dihitung ganda: faktor *Digital Opportunity* hanya memakai opportunity keluarga DIGITAL, sedangkan *Opportunity Strength* memakai keluarga lain. |
| BR-SCORE-09 | Persentil memakai metode **mid-rank**: `pct(x) = 100 × (jumlah nilai < x + 0,5 × jumlah nilai = x) / n`. Nilai seri mendapat persentil yang sama. |
| BR-SCORE-10 | Skor disimpan dengan 1 desimal dan ditampilkan sebagai bilangan bulat (pembulatan half-up). **Band ditentukan dari bilangan bulat yang ditampilkan**, sehingga UI tidak pernah menampilkan "80" dengan band WARM. |
| BR-SCORE-11 | Coverage < 0,60 hanya membatasi **band** (maks WARM). Nilai numerik tidak dipotong. |
| BR-SCORE-12 | Kriteria ICP *wajib* yang nilainya Unknown: establishment tetap masuk hasil, kriteria dikeluarkan dari ICP Fit, flag "Kriteria wajib belum terkonfirmasi" ditampilkan, dan band maksimal WARM. |
| BR-SCORE-13 | Determinisme: input yang sama + `model_version` yang sama + `as_of` yang sama → skor identik. Komponen yang bergantung waktu (freshness) dihitung terhadap `as_of` (= `calculated_at`). Job harian menandai skor stale bila freshness berpindah kelas. |

### 9.6 BR-OPP — Opportunity

| ID | Aturan |
|---|---|
| BR-OPP-01 | Katalog, aturan deteksi, dan severity ada di **Lampiran B**. Tipe yang field sumbernya tidak ada di dataset berstatus *inactive*, bukan dihapus. |
| BR-OPP-02 | Klaim "tidak ada X" hanya valid bila fill-rate field X pada sumber untuk kategori × kab/kota tersebut ≥ 50%. Bila di bawah itu, confidence dikurangi 30 poin. Bila < 20%, tipe tidak dideteksi untuk segmen tersebut. Fill-rate hanya dihitung bila segmen berisi ≥ 20 establishment. Bila kurang, naik ke kategori × provinsi, lalu kategori × nasional. |
| BR-OPP-03 | `opportunity_score = severity × (confidence/100) × relevance`. `relevance` = 1,0 bila tag Offering cocok primer, 0,6 bila sekunder, dan 0 bila tidak relevan (gap fact tetap terlihat di kartu digital/reputasi, tetapi tidak disebut opportunity). |
| BR-OPP-04 | Setiap opportunity memiliki rationale yang dapat dibaca sales, termasuk konteks peer bila ada. Contoh: "Tidak ada website pada data sumber (snapshot 2026.10). 68% klinik gigi di Kota Denpasar memiliki website." |
| BR-OPP-05 | Opportunity adalah hipotesis, bukan jaminan hasil penjualan (guardrail Rev 1.2 §8). |

### 9.7 BR-REV — Commercial / revenue

| ID | Aturan |
|---|---|
| BR-REV-01 | Band karyawan mengikuti kelas BPS: 1–4, 5–19, 20–99, 100–499, 500+. |
| BR-REV-02 | Band revenue tahunan mengikuti batas UMKM (PP 7/2021) dengan sub-band: R1 ≤ Rp2 M (Mikro), R2 > Rp2–5 M, R3 > Rp5–15 M (R2–R3 = Kecil), R4 > Rp15–50 M (Menengah), R5 > Rp50–250 M, R6 > Rp250 M (Besar). Di UI, "M" = miliar. |
| BR-REV-03 | Revenue v0 berstatus INFERRED, dihitung dari prior kategori × kota dan pengali ukuran (jumlah cabang, persentil volume ulasan, price level bila ada). Confidence maksimum 70 kecuali ada evidence finansial observed. |
| BR-REV-04 | Point estimate tidak ditampilkan di MVP. Yang ditampilkan hanya band + confidence + basis + as-of. |
| BR-REV-05 | Basis harus dapat dibaca. Contoh: "Kategori klinik gigi, 3 cabang, volume ulasan persentil 85 di Kota Denpasar." |
| BR-REV-06 | **Tier kota** (dipakai prior revenue dan persentil komersial) berdasarkan populasi kab/kota: Tier 1 ≥ 1.000.000, Tier 2 250.000–999.999, Tier 3 < 250.000. |

### 9.8 BR-CRED — Credit & refund (D-08)

| ID | Aturan |
|---|---|
| BR-CRED-01 | Search dan lihat profil tidak memakai credit, tetapi dibatasi fair-use per plan (Lampiran E). Kuota menghitung **profil unik** per siklus (membuka profil yang sama berulang kali dihitung satu). Saat kuota 100%: hasil search tetap tampil, profil yang belum dibuka di siklus ini diblokir dengan pesan upgrade, sedangkan profil yang sudah dibuka, di-unlock, atau di-assign tetap dapat diakses. |
| BR-CRED-02 | **Semua** nilai contact point (telepon, WhatsApp, email) dan nama person ditampilkan ter-mask (contoh `+62 812-****-7890`, `h****@abcdental.co.id`) sampai di-**unlock**. Nama company, kategori, alamat, website, dan role person tetap terlihat. Unlock = 1 credit per Company dan berlaku permanen untuk workspace. Kontak yang berasal dari upload workspace itu sendiri tidak di-mask. |
| BR-CRED-03 | Export/webhook record yang sudah di-unlock gratis. Record yang belum di-unlock otomatis di-unlock saat export (biaya ditampilkan di pratinjau). |
| BR-CRED-04 | Operasi berbayar idempotent per `idempotency_key`. Retry tidak menagih ulang. |
| BR-CRED-05 | Refund otomatis bila (a) operasi gagal atau tidak terkirim, atau (b) seluruh contact point Company yang di-unlock terbukti INVALID ≤ 30 hari setelah unlock. Di MVP 1, "terbukti INVALID" berarti hasil re-verifikasi (refresh) INVALID, atau laporan data user (FR-DATA-014) yang dikonfirmasi ops. |
| BR-CRED-06 | Credit grant bulanan kedaluwarsa di akhir siklus. Credit top-up berlaku 12 bulan (default, menunggu keputusan pricing). |
| BR-CRED-07 | Operasi berbayar async (export, webhook massal, refresh massal) memakai pola **reserve → capture/release**: credit ditahan saat job dibuat, ditagih saat sukses, dan dilepas bila gagal. Auto-unlock dari export baru berlaku saat export sukses. |
| BR-CRED-08 | Urutan pemakaian credit: yang paling cepat kedaluwarsa dipakai lebih dulu (grant bulanan, lalu top-up FIFO berdasarkan tanggal kedaluwarsa). |
| BR-CRED-09 | Bila dua Company yang sama-sama sudah di-unlock oleh satu workspace kemudian di-merge (BR-ER), satu credit unlock dikembalikan otomatis. |

### 9.9 BR-PRIV — Privasi

| ID | Aturan |
|---|---|
| BR-PRIV-01 | Data pribadi (nama person, email/telepon pribadi) diminimalkan di log, analytics, dan payload debug (hash/redaksi). |
| BR-PRIV-02 | Suppression atas data global layer berlaku untuk semua workspace dan mengalahkan semua sumber lain. Untuk data upload tenant (tenant = pengendali, MSP = prosesor), permintaan subjek data diteruskan ke tenant dan dieksekusi atas instruksinya. Nilai yang sama dari global layer tetap di-suppress. |
| BR-PRIV-03 | Kepatuhan mengacu UU 27/2022 tentang Pelindungan Data Pribadi dan aturan pelaksananya. Kewajiban notifikasi kegagalan pelindungan data 3×24 jam dimasukkan ke runbook insiden. Detail hukum divalidasi konsultan hukum. |
| BR-PRIV-04 | Penggunaan, penyimpanan, tampilan, dan export data mengikuti lisensi/ketentuan sumber dataset (guardrail Rev 1.2 §8, Q-05). |

---

## 10. Non-functional requirements

| ID | Kategori | Requirement |
|---|---|---|
| NFR-PERF-01 | Performa | API search p95 ≤ 800 ms, p99 ≤ 2 s pada dataset sizing NFR-SCAL-01 dengan 50 user konkuren. |
| NFR-PERF-02 | Performa | API profil p95 ≤ 500 ms. |
| NFR-PERF-03 | Performa | Web: LCP p75 ≤ 2,5 s, INP p75 ≤ 200 ms (desktop). |
| NFR-PERF-04 | Performa | Export 10.000 baris ≤ 2 menit. 50.000 baris ≤ 10 menit. |
| NFR-PERF-05 | Performa | Simpan/ubah ICP → **skor progresif**: halaman pertama (≤ 50 kandidat teratas hasil pre-ranking) ber-skor ≤ 30 detik; seluruh kandidat (≤ 200.000 establishment) ≤ 10 menit. Selama proses, UI menampilkan status "Menghitung skor" beserta progres. |
| NFR-PERF-06 | Performa | AI brief p95 ≤ 20 s. |
| NFR-PERF-07 | Performa | Re-run pipeline penuh untuk 5 juta establishment ≤ 12 jam. |
| NFR-SCAL-01 | Skala | Sizing MVP: 5 juta establishment, ±84 ribu kelurahan/desa, 200 workspace, 2.000 user, 50 user konkuren. Revisi setelah Q-02 dijawab. |
| NFR-AVAIL-01 | Ketersediaan | 99,5% bulanan (MVP 1). Target 99,9% untuk Enterprise (V2). |
| NFR-AVAIL-02 | Ketersediaan | RPO ≤ 15 menit (PITR). RTO ≤ 4 jam untuk kegagalan server/DB tunggal. Untuk kehilangan seluruh data center: RTO ≤ 24 jam (restore dari backup off-site ke lokasi cadangan), ditinjau ulang sebelum Enterprise. |
| NFR-SEC-01 | Keamanan | Baseline OWASP ASVS Level 2. |
| NFR-SEC-02 | Keamanan | TLS 1.2+ untuk semua trafik. Enkripsi at-rest untuk DB, storage, dan backup. |
| NFR-SEC-03 | Keamanan | Isolasi tenant ganda (RLS + aplikasi), diuji otomatis di CI (FR-AUD-006). |
| NFR-SEC-04 | Keamanan | Secret hanya di penyimpanan secret terenkripsi (SOPS + age, variabel CI terproteksi). Repo dipindai secret di CI. |
| NFR-SEC-05 | Keamanan | Request keluar ke URL milik customer (webhook/CRM) melewati kontrol SSRF (Crawler Spec §12, diadopsi). |
| NFR-SEC-06 | Keamanan | Tidak ada temuan Critical/High yang terbuka saat rilis (SAST, dependency, container, DAST baseline). |
| NFR-PRIV-01 | Privasi | Redaksi PII di log (BR-PRIV-01). |
| NFR-PRIV-02 | Privasi | Propagasi suppression ≤ 24 jam (FR-AUD-003). |
| NFR-PRIV-03 | Privasi | Penyimpanan utama dan backup di data center Indonesia (D-14, D-19). |
| NFR-PRIV-04 | Privasi | Tidak ada nilai contact point pribadi yang dikirim ke LLM (FR-AI-005). |
| NFR-A11Y-01 | Aksesibilitas | WCAG 2.2 AA. Status (band, verifikasi, claim type) tidak boleh dikomunikasikan hanya dengan warna. |
| NFR-L10N-01 | Lokalisasi | id-ID default, en-US sekunder. Format Rupiah & angka id-ID. Zona waktu tampilan Asia/Jakarta (dapat diubah). Penyimpanan UTC. |
| NFR-COMP-01 | Kompatibilitas | 2 versi terakhir Chrome, Edge, Firefox, Safari. Desktop ≥ 1280 px sebagai target utama. Profil, list, dan tugas tetap dapat dipakai di 360 px. |
| NFR-OBS-01 | Observability | Log terstruktur, metrik, dan trace dengan `correlation_id`. Dashboard + alert untuk setiap SLO. |
| NFR-DATA-01 | Reproducibility | Setiap skor dan metrik area dapat direproduksi dari `snapshot_version` + `model_version`. |
| NFR-MAINT-01 | Maintainability | Coverage test modul normalisasi, ER, dan scoring ≥ 85% (statement coverage Go; line coverage untuk TypeScript). Keseluruhan backend ≥ 75%. |
| NFR-COST-01 | Biaya | Batas biaya AI per workspace. Biaya per brief dipantau (FR-OPS-004). |

---

## 11. Kebutuhan data (ringkas)

Kontrak lengkap ada di `04-technical-design.md` §Input Contract.

| Data | Sumber | Field minimum | Wajib untuk |
|---|---|---|---|
| Business records | Dataset yang sudah ada (snapshot) | source_record_id, nama, kategori sumber, alamat, koordinat, telepon, website, status bisnis, observed_at | Semua P0 |
| Atribut tambahan | Dataset (bila ada) | jam operasional, rating, jumlah ulasan, price level, email, person/role | Faktor & opportunity terkait (Unknown bila tidak ada) |
| Wilayah | Referensi Kemendagri (+ batas wilayah) | kode, nama, level, parent, geometri (opsional) | GEO, filter wilayah |
| Populasi | Dataset populasi per kelurahan | kode wilayah, populasi, luas, periode referensi, sumber | GEO |
| Upload customer | CSV/XLSX | nama + (domain atau telepon atau alamat) minimal | FR-DATA-012 |

---

## 12. Release gate & Definition of Done

### 12.1 Gate Sellable MVP (MVP 1), versi terukur dari MVP Pack §15

| Gate | Kriteria terukur | Bukti |
|---|---|---|
| RG-1 Relevansi | ≥ 4 dari 5 user uji (sales) mampu menjelaskan alasan peringkat 3 hasil teratas tanpa bantuan. Median time-to-first-result ≤ 5 menit. | Laporan usability test |
| RG-2 Kepercayaan | ≥ 4/5 user uji dapat menunjukkan sumber dan confidence sebuah kontak. Evidence coverage ≥ 95%. | Usability + metrik data |
| RG-3 Aksi | Skenario E2E select → export dan select → webhook lulus. Webhook success ≥ 99% pada soak test 24 jam di staging. | Laporan E2E + soak |
| RG-4 Usage | Rekonsiliasi ledger vs usage event 100%. Tidak ada tagihan ganda pada uji idempotensi. | Laporan test billing |
| RG-5 Operasional | Alert pipeline failure > 5%, backlog > 1 jam, AI failure > 5%, dan webhook failure > 5% terpicu pada drill di staging. | Laporan drill |
| RG-6 Kualitas data | ER precision ≥ 98% dan recall ≥ 90% pada golden set. Akurasi klasifikasi industri ≥ 90% dan akurasi penetapan kelurahan ≥ 95% (sampel berlabel n ≥ 300). ≥ 10 edge case data (duplikat, hilang, konflik, stale, ambigu) lulus. | Laporan QA |
| RG-7 Keamanan | Uji isolasi tenant lulus. Tidak ada Critical/High terbuka. Uji SSRF webhook lulus. | Laporan security |

### 12.2 Definition of Done (per fitur)
- Acceptance criteria FR terkait lulus dan tertaut ke test case (`05-test-strategy.md`).
- Status gagal bersifat eksplisit dan dapat di-retry. Tidak ada aksi yang gagal secara diam-diam.
- Event analytics & audit (bila relevan) terpasang.
- Copy id-ID & en-US tersedia, dan aksesibilitas dicek (axe tanpa pelanggaran serious/critical).
- Dokumentasi API/event diperbarui. Perubahan breaking terdeteksi di CI.

---

## 13. Dependensi, asumsi, risiko

| ID | Tipe | Deskripsi | Mitigasi |
|---|---|---|---|
| A-01 | Asumsi | Dataset bisnis memuat field minimum §11 untuk mayoritas record. | Validasi via laporan fill-rate FR-DATA-001 (Q-01). |
| A-02 | Asumsi | Data populasi per kelurahan tersedia dengan periode referensi jelas. | Q-04. Fallback ke level kecamatan. |
| A-03 | Asumsi | Data batas wilayah (poligon) kelurahan/kecamatan tersedia dengan lisensi yang mengizinkan tampilan. | Q-10. Tanpa poligon, peta memakai titik/cluster + tabel. |
| R-01 | Risiko | Fill-rate rendah untuk email/person → nilai Contact Intelligence turun. | Tampilkan coverage secara jujur. Upload customer. Crawler dapat dihidupkan kembali di V2. |
| R-02 | Risiko | Revenue v0 tidak terkalibrasi. | Hanya band + confidence ≤ 70. Kumpulkan ground truth (Q-06). |
| R-03 | Risiko | "White space" sebenarnya mencerminkan data yang belum lengkap. | Indikator coverage dan guard populasi (Lampiran C). |
| R-04 | Risiko | Opportunity dianggap tidak relevan oleh user. | Offering + feedback FR-INTL-010. Kalibrasi severity per beta. |
| R-05 | Risiko | Biaya AI tidak terkendali. | Cache, batas per workspace, kill switch. |
| R-06 | Risiko | Kebocoran data antar tenant. | RLS + uji otomatis + review keamanan. |
| R-07 | Risiko | Dataset tidak diperbarui → freshness menurun dari waktu ke waktu. | Label freshness jujur. Q-03. Roadmap refresh. |
| R-08 | Risiko | `app-01` adalah single point of failure. Kerusakan hardware bisa menghabiskan anggaran downtime 99,5% (±3,6 jam/bulan). | Restore cepat via Ansible + image registry. **`app-02` wajib aktif sebelum M-3 (early paid)**. |

---

## Lampiran A — Priority score v1.2.0 (default)

```
priority = Σ_{k∈K} wₖ·sₖ / Σ_{k∈K} wₖ          K = faktor yang diketahui
coverage = Σ_{k∈K} wₖ / Σ_{semua} wₖ
```

| Faktor | Bobot | Definisi sₖ (0–100) | MVP |
|---|---|---|---|
| ICP Fit | 25 | Rata-rata tertimbang kecocokan kriteria ICP: match = 1, partial = 0,5, no = 0. Kriteria berstatus Unknown dikeluarkan. Kriteria *wajib* yang `no` → establishment tidak masuk hasil. | ✓ |
| Data Quality | 10 | 0,4·completeness (field relevan per offering tag, didefinisikan di konfigurasi model; default: telepon, website, alamat, kategori, jam operasional) + 0,3·verifikasi (status kontak terbaik) + 0,2·freshness + 0,1·evidence coverage | ✓ |
| Contactability | 10 | Kanal terbaik: email person VERIFIED 100; email person LIKELY_VALID 90; email company LIKELY_VALID 80; seluler LIKELY_VALID (WA-likely) 80; telepon tetap LIKELY_VALID 60; hanya profil sosial 30; tidak ada 0. Bonus +10 bila ada kontak decision-maker (maks 100). | ✓ |
| Commercial Potential | 15 | Persentil proxy skala (band revenue, band karyawan, cabang, volume ulasan) dalam kategori × tier kota, dikali (0,5 + 0,5·confidence/100) | ✓ |
| Digital Opportunity | 10 | Maks `opportunity_score` keluarga DIGITAL yang relevan. 0 bila data digital diketahui tetapi tidak ada gap relevan. Unknown bila data digital tidak ada. | ✓ (lite) |
| Geographic Opportunity | 15 | `market_opportunity_score` kelurahan establishment. Fallback kecamatan bila kelurahan LOW. Unknown bila keduanya LOW (FR-INTL-006). | ✓ |
| Signal Strength | 5 | Maks kekuatan signal relevan dalam 90 hari, dengan decay | Unknown sampai V1 (coverage maksimum di MVP = 0,95) |
| Opportunity Strength | 10 | Maks `opportunity_score` keluarga non-DIGITAL yang relevan | ✓ |

- Versi: `priority-1.2.0`. Setiap perubahan bobot/aturan menaikkan versi (FR-INTL-009).
- Preset per Offering (FR-ICP-004) hanya mengubah bobot. Contoh preset `reputation_mgmt`: Opportunity Strength 20, Digital 5, Geographic 10; faktor lain proporsional.

## Lampiran B — Katalog opportunity v0

| Kode | Keluarga | Aturan deteksi (ringkas) | Field sumber | Severity | Offering tag (primer / sekunder) | Status selama pause |
|---|---|---|---|---|---|---|
| OPP-WEB-NONE | DIGITAL | Website kosong pada sumber (dengan syarat BR-OPP-02) | website | 80 | website_dev / local_seo, digital_ads | Aktif |
| OPP-WEB-SOCIAL-ONLY | DIGITAL | Nilai website = profil sosial, link aggregator, atau link pesan | website | 70 | website_dev / digital_ads | Aktif |
| OPP-WEB-FREE-SUBDOMAIN | DIGITAL | Website berada di subdomain platform gratis | website | 55 | website_dev / email_service | Aktif |
| OPP-EMAIL-NON-DOMAIN | DIGITAL | Hanya email webmail gratis yang diketahui | email | 50 | email_service / website_dev | Aktif bila field email ada |
| OPP-DNS-NO-MX | DIGITAL | Domain sendiri tanpa record MX | DNS | 45 | email_service / — | MVP 1 (FR-DATA-011) |
| OPP-BOOKING-NONE | DIGITAL | `has_online_booking = false` (observed) | has_online_booking | 75 | booking_system / website_dev | Inactive kecuali dataset memiliki field ini |
| OPP-REP-LOW-RATING | REPUTATION | Rating < 4,0 dengan ≥ 20 ulasan | rating, review_count | 70 | reputation_mgmt / local_seo | Aktif bila field ada |
| OPP-REP-LOW-VOLUME | REPUTATION | Jumlah ulasan < P25 peer kategori di kab/kota yang sama (≥ 10 peer) | review_count | 55 | reputation_mgmt, local_seo / digital_ads | Aktif bila field ada |
| OPP-LIST-INCOMPLETE | LISTING | ≥ 2 dari {telepon, website, jam operasional} kosong | phone, website, hours | 50 | local_seo / website_dev | Aktif |
| OPP-MKT-HIGH-COMPETITION | MARKET | Competition index area = High | metrik area | 50 | digital_ads, local_seo / reputation_mgmt | Aktif |
| OPP-OPS-MULTI-BRANCH | OPERATIONS | Company memiliki ≥ 3 establishment | establishment | 60 | crm, pos, booking_system / — | Aktif |
| OPP-SIG-NEW-BUSINESS | MARKET | Pertama kali muncul di snapshot terbaru | diff snapshot | 60 | semua / — | V1 |

## Lampiran C — Metrik geografis v1.0.0

Level area: L1 provinsi, L2 kab/kota, L3 kecamatan, L4 kelurahan/desa. Satuan densitas **per 1.000 penduduk** (I-08).

| Metrik | Rumus | Catatan |
|---|---|---|
| population | dari dataset populasi | wajib `reference_period` |
| population_density | population / luas_km² | — |
| business_count | jumlah establishment aktif di area | tutup permanen tidak dihitung |
| business_density | business_count / population × 1.000 | proxy supply |
| category_count | establishment kategori X di area | — |
| category_density | category_count / population × 1.000 | — |
| peer group | area satu level dalam kab/kota yang sama (L4) atau provinsi yang sama (L3). Bila < 10 area, diperluas satu level ke atas. | — |
| competition_index | persentil category_density dalam peer group (0–100). Low < 33, Medium 33–66, High > 66. | Faktor geo establishment memakai kategori establishment itu sendiri. Territory ranking untuk ICP multi-kategori memakai gabungan kategori ICP (jumlah count). |
| coverage | **LOW** bila population < 500, atau business_count < 5, atau business_density < 0,25 × median peer padahal population_density ≥ median peer. Selain itu OK. | LOW → white space & opportunity area = Unknown, label "Data terbatas" |
| white_space_score | pct(population) × (100 − pct(category_density)) / 100 | hanya bila population ≥ 1.000 dan coverage OK |
| target_business_count | establishment yang lolos filter wajib ICP di area | tenant-specific |
| target_density | target_business_count / population × 1.000 | — |
| activity | median jumlah ulasan per establishment di area (persentil) | Unknown bila tidak ada data ulasan |
| market_opportunity_score | 0,35·pct(population) + 0,25·pct(target_density) + 0,15·pct(activity) + 0,25·(100 − competition_index) | faktor Unknown di-renormalisasi |
| territory_score | 0,5·market_opportunity_score + 0,3·pct(jumlah prospek HOT+WARM) + 0,2·rata-rata data quality prospek target | tenant & ICP-specific |

`pct()` = persentil dalam peer group. Versi `geo-1.0.0`.

## Lampiran D — Traceability MVP Pack → PRD

| MVP Pack | Requirement PRD |
|---|---|
| MVP-01 Canonical Company | FR-DATA-002, FR-DATA-003, FR-DATA-007 |
| MVP-02 Contact Point | FR-DATA-005, FR-PROF-003, BR-VER |
| MVP-03 Evidence | FR-DATA-006, FR-PROF-004 |
| MVP-04 Deduplication | FR-DATA-003, FR-DATA-004, BR-ER |
| MVP-05 Search | FR-SRCH-001 s.d. 008 |
| MVP-06 Scoring | FR-INTL-008, FR-INTL-009, FR-PROF-002, Lampiran A |
| MVP-07 Opportunity | FR-INTL-007, Lampiran B |
| MVP-08 Segment | FR-SEG-001, FR-SEG-002 |
| MVP-09 Action | FR-EXP-001, FR-EXP-003, FR-EXP-004 |
| MVP-10 AI Brief | FR-AI-001, FR-AI-004, FR-AI-005 |
| MVP-11 Refresh | FR-DATA-007, FR-MON-001 |
| MVP-12 Tenant isolation | FR-AUD-006, NFR-SEC-03 |
| MVP-13 Auditability | FR-AUD-001 |
| MVP-14 Usage | FR-BILL-001, FR-BILL-002 |

## Lampiran E — Katalog credit & plan (angka awal beta, divalidasi pricing)

| Operasi | Meter | Unit | Catatan |
|---|---|---|---|
| Search, lihat profil | — | 0 | Fair-use per plan |
| Unlock kontak Company | credit | 1 | Permanen per workspace |
| Export/webhook record yang sudah di-unlock | — | 0 | — |
| Export/webhook record yang belum di-unlock | credit | 1 | Auto-unlock, tampil di pratinjau |
| Refresh prospek | credit | 1 | — |
| Re-verifikasi kontak via provider | credit | 1 per kontak | V1 |
| Enrichment / deep intelligence | credit | 2–5 | V1+, butuh sumber tambahan |
| AI brief | ai_credit | 1 | Cache 7 hari gratis |
| AI sales angle | ai_credit | 1 | — |
| Watch | entitlement | per plan | V1 |

| Entitlement | Starter | Growth | Agency | Enterprise |
|---|---|---|---|---|
| Seat | 2 | 5 | 15 | Kontrak |
| ICP tersimpan | 1 | 5 | 20 | Kontrak |
| Lihat profil / bulan (fair-use) | 1.000 | 10.000 | 30.000 | Kontrak |
| Credit / bulan | 200 | 1.000 | 3.000 | Kontrak |
| AI credit / bulan | 50 | 300 | 1.000 | Kontrak |
| Webhook endpoint | — | 3 | 10 | Kontrak |
| Maks baris per export | 1.000 | 10.000 | 50.000 | 50.000+ |
| Paket penjualan (MVP Pack §16) | Prospect | Intelligence + Workflow | Agency | Agency + API/SSO/audit |
