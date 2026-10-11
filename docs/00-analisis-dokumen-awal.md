# 00 — Analisis Dokumen Awal

| Field | Value |
|---|---|
| Dokumen | Analisis dokumen awal → dasar dokumen lanjutan |
| Status | v0.2 (lean): keputusan founder diterapkan, lihat §7 |
| Tanggal | 2026-10-10 |
| Input | (1) *MesinProspek Sales Intelligence Engine — Sellable MVP Documentation Pack v1.2 (Intelligence Domains)*, selanjutnya **MVP Pack**; (2) *MSP Website Acquisition & Crawler Engine Spec v1.0*, selanjutnya **Crawler Spec** (versi .docx dan .pdf identik) |
| Output | Daftar inkonsistensi (I-xx), gap (G-xx), dampak pause crawler, register keputusan (D-xx), peta dokumen lanjutan |

---

## 1. Ringkasan eksekutif

1. **MVP Pack adalah dokumen produk yang kuat secara prinsip.** Prinsip intinya sudah tepat sebagai fondasi: *evidence over magic*, *unknown over invented*, *one canonical record, many projections*, serta pemisahan observed/derived/inferred. Semua dokumen lanjutan mempertahankan prinsip ini tanpa perubahan.
2. **Dokumen ini belum siap dieksekusi engineering.** Revisi 1.1 dan 1.2 ditempel di akhir dokumen, tidak diintegrasikan. Akibatnya ada dua model skor prioritas, tiga mata uang credit, dan roadmap yang saling bertentangan. Daftarnya ada di §4.
3. **Kontrak input belum pernah didefinisikan.** Kotak "INPUT CONTRACT" muncul di diagram tetapi tidak dijabarkan. Karena crawler di-pause dan dataset sudah tersedia, kontrak inilah yang menghubungkan dataset ke engine. Dokumen lanjutan wajib mendefinisikannya.
4. **Crawler Spec di-PAUSE, tetapi sebagian polanya tetap dipakai.** Engine intelligence tidak bergantung pada crawler. Pola berikut diadopsi ulang: model evidence/provenance, kontrol SSRF (dipakai untuk webhook URL milik customer), retry/backoff/DLQ, konvensi logging, dan change detection berbasis hash/snapshot.
5. **Beberapa keputusan produk belum diambil**, misalnya tenancy data, model credit, dan definisi Offering. Keputusan ini memengaruhi skema data dan billing. Untuk masing-masing, dokumen ini mengusulkan default (§7) agar pekerjaan bisa berjalan. Default tersebut tetap perlu dikonfirmasi founder/product.

---

## 2. Status dokumen sumber

| Dokumen | Status baru | Catatan |
|---|---|---|
| MVP Pack v1.2 | **BASELINE** produk | Sumber kebenaran untuk prinsip, domain intelligence, dan scope. Isi yang bertentangan diselesaikan di PRD (`01-prd.md`). |
| Crawler Spec v1.0 | **PAUSED** | Tidak dibangun di MVP. Input engine berasal dari dataset yang sudah ada + upload customer. Crawler bisa dihidupkan lagi kelak sebagai *source adapter* yang memenuhi Input Contract yang sama (lihat D-01). |

### 2.1 Bagian Crawler Spec yang tetap diadopsi

| Bagian Crawler Spec | Diadopsi untuk | Dokumen tujuan |
|---|---|---|
| §11 Evidence & Provenance (source_url, captured_at, source_record_id, snippet_hash, extractor_version, confidence) | Skema `evidence` generik untuk semua sumber (dataset, upload, partner) | `04-technical-design.md` |
| §12 SSRF controls | Validasi URL webhook dan endpoint CRM milik customer (outbound request dari server) | `04-technical-design.md`, `05-test-strategy.md` |
| §8.3 / §13 Retry, backoff, DLQ | Job pipeline, export, delivery webhook | `04-technical-design.md` |
| §14.2 Change detection (hash) | Diff antar *dataset snapshot* → signal & refresh | `04-technical-design.md` |
| §19.2 Log fields (tenant_id, job_id, correlation_id, redaksi PII) | Standar logging seluruh service | `04-technical-design.md`, `06-ci-cd.md` |
| §24 ADR "Data quality separated from sales score" | Masukan untuk I-03 | `01-prd.md` |
| §29 Testing strategy (unit + fixture + load) | Pola pengujian pipeline | `05-test-strategy.md` |

Bagian yang **tidak** diadopsi selama pause: frontier, fetch engine, robots, browser renderer, credit model crawler (§22.1), dan API `/v1/crawl/*`.

---

## 3. Bagian yang sudah kuat (dipertahankan apa adanya)

- Layer produk Find → Understand → Qualify → Prioritize → Recommend → Act → Monitor (Rev 1.1 §A).
- Klasifikasi claim **Observed / Derived / Inferred / Unknown** (Rev 1.1 §B) sebagai atribut wajib setiap nilai yang tampil ke user.
- Kontrak skor: setiap skor memiliki nilai, *factor breakdown*, versi model, `calculated_at`, dan status freshness (§06).
- Pipeline yang idempotent dan bisa diproses ulang per tahap (§05), termasuk perilaku saat gagal (*Unknown, not false*).
- Hierarki entity resolution dengan human-review queue untuk kandidat ambigu bernilai tinggi (§05).
- Lima domain intelligence (Rev 1.2) dan guardrail-nya (Rev 1.2 §8).
- Anti-clone boundary terhadap Coldlytics (Rev 1.1 §E).
- Acceptance criteria MVP-01 s.d. MVP-14 dan lima release gate (§10, §15).

---

## 4. Inkonsistensi antar bagian

| ID | Lokasi | Masalah | Resolusi yang diusulkan |
|---|---|---|---|
| I-01 | Document control vs judul | Tabel *Document control* menulis "Version 1.1", sedangkan judul file dan isi menyebut v1.2. | Dokumen lanjutan merujuk **MVP Pack v1.2**. PRD menjadi v1.3-equivalent dan menggantikan bagian yang bertentangan. |
| I-02 | §06 vs Rev 1.2 §9 | Dua model priority score. §06 memakai 6 faktor (ICP 30, DQ 15, Contactability 15, Activity 10, Opportunity 20, Signal 10). Rev 1.2 memakai 8 faktor (ICP 25, DQ 10, Contact 10, Commercial 15, Digital 10, Geo 15, Signal 5, Opportunity 10). Faktor "Activity" hilang di v1.2. | **Rev 1.2 §9 berlaku** (lebih baru). "Activity" dilebur ke Signal/Commercial. Ditambah aturan *missing factor* (G-05). PRD BR-SCORE. |
| I-03 | Rev 1.2 §9 vs Crawler Spec §24 | Data Quality menjadi faktor priority (10%), padahal ADR crawler menyatakan "data quality separated from sales score". | Data Quality tetap 10% di priority (keputusan produk v1.2) **dan** ditampilkan terpisah sebagai *confidence* skor. Pantau korelasinya setelah beta (D-06). |
| I-04 | §09 vs Rev 1.1 §D vs Crawler §22.1 | Model credit tidak konsisten. §09: "1 basic prospect = 1 credit". Rev 1.1: basic read "included or low-cost". Ada tiga mata uang (credit, AI credit, refresh credit), ditambah credit crawler. | Dua meter saja: `credit` dan `ai_credit`. Lihat/cari profil tidak memakai credit (fair-use per plan). Credit dikenakan saat *unlock* kontak, export/sync, enrichment, dan refresh (D-08). |
| I-05 | §09 vs §16 | Nama tier (All/Growth+/Agency/Enterprise) tidak dipetakan ke paket penjualan (Prospect/Intelligence/Workflow/Agency). | Satu tabel plan: Starter, Growth, Agency, Enterprise, dipetakan ke paket (D-09). |
| I-06 | §15 + Appendix A vs Rev 1.2 §10 | Rev 1.2 memasukkan Geographic dan Commercial baseline ke MVP, tetapi roadmap §15 dan backlog Appendix A belum memuatnya. Digital maturity ditaruh di V1, padahal faktor "Digital Opportunity" (10%) sudah dipakai di skor MVP. | PRD menyatukan release scope. Faktor Digital di MVP hanya memakai atribut digital yang tersedia di dataset; jika tidak ada, faktor = Unknown (aturan G-05). |
| I-07 | §06, §15, Appendix A | Status Signal Detection berbeda-beda: "PREMIUM-LEAN" (§06), V1 (§15), tidak ada di backlog, tetapi sudah menjadi faktor skor 5% di MVP. | Signal = **V1**. Di MVP faktor Signal = Unknown → di-renormalisasi. |
| I-08 | Rev 1.2 §4 vs §5 | Satuan densitas berbeda: formula memakai "per 1.000 penduduk", contoh memakai "3 dental clinics / 10K residents". | Satuan baku **per 1.000 penduduk** di seluruh produk. |
| I-09 | Rev 1.2 | Penomoran bagian Rev 1.2 ("# 2. Canonical Intelligence Model") bertabrakan dengan bagian utama 01–16. | Dokumen lanjutan memakai ID unik (FR-, BR-, TC-), bukan nomor bab. |
| I-10 | §08 vs §16 "Recommended MVP action set" vs Appendix A | CRM: §08 menyebut "Webhook + one CRM target" di MVP, action set rekomendasi tidak menyebut CRM, sedangkan backlog menaruh CRM di P1. | MVP 1 = webhook. **Satu integrasi CRM = V1** (D-10). |
| I-11 | Rev 1.1 §C vs Appendix A | "Compare" tercantum sebagai MVP action (Prepare) di Rev 1.1, tetapi P2 di backlog. | Compare prospect = V1. Compare Areas = V1 (Rev 1.2 §6). |
| I-12 | Rev 1.1 §A vs Appendix A | Natural-language search P1 di benchmark, tetapi tidak ada di backlog. | NL search = V1, sebagai lapisan di atas filter kanonik (filter hasil parsing ditampilkan ke user). |
| I-13 | §04 vs Rev 1.2 §2 | Entity `Opportunity` di §04 tidak punya `confidence`, sedangkan MVP-07 mewajibkannya. | Skema mengikuti Rev 1.2 (dengan `confidence`). |
| I-14 | §04 `Activity` vs §13 audit | `Activity` (aksi user/system) dan audit log tidak dibedakan. | Dua hal terpisah: `activity` (timeline produk, terlihat user) dan `audit_log` (immutable, untuk kepatuhan). |

---

## 5. Gap: hal yang wajib ada sebelum build

Severity: **C** = Critical (memblokir desain skema/arsitektur), **H** = High (memblokir fitur P0), **M** = Medium.

| ID | Sev | Gap | Ditutup di |
|---|---|---|---|
| G-01 | C | **Input Contract belum didefinisikan.** Belum ada skema observation, cara memetakan dataset yang sudah ada, versioning snapshot, maupun format upload customer. | PRD FR-DATA, SDD §Input Contract |
| G-02 | C | **Tenancy data belum diputuskan.** Apakah data bisnis dipakai bersama (global) atau per tenant? ICP fit dan priority bergantung pada ICP tiap tenant, sehingga skor tidak bisa disimpan sebagai atribut global Company. Upload customer juga harus tetap privat. | D-03, SDD §Tenancy |
| G-03 | H | **Belum ada objek "Offering".** Opportunity adalah "hipotesis kebutuhan sales", dan relevansinya bergantung pada apa yang dijual tenant. "Tidak ada online booking" hanya bernilai bagi tenant yang menjual sistem booking. | D-04, PRD FR-ICP |
| G-04 | H | **Company vs cabang (establishment) belum dibedakan.** Rev 1.2 memakai `branch_count` dan konteks geo per lokasi. Aturan ER "same canonical domain → merge" akan menggabungkan dua cabang dan kehilangan salah satu lokasi. | D-05, SDD §Data Model |
| G-05 | H | **Perlakuan nilai Unknown di skor belum ada.** Prinsip "Unknown over invented" belum diterjemahkan ke rumus skor. | PRD BR-SCORE-03 |
| G-06 | H | **Rumus confidence dan freshness belum didefinisikan.** Belum ada cara agregasi multi-evidence, decay waktu, maupun ambang Fresh/Aging/Stale. | PRD BR-TRUST, SDD |
| G-07 | H | **Katalog opportunity belum ada.** Belum ada daftar tipe, aturan deteksi, evidence minimum, dan pemetaan ke Offering. | PRD Lampiran B |
| G-08 | H | **Rumus geo belum operasional.** Belum ada definisi *peer area*, guard populasi kecil, dan bias cakupan data ("white space" bisa saja berarti data belum lengkap). Kode wilayah kanonik (Kemendagri vs BPS) dan periode referensi data populasi juga belum ditentukan. | PRD Lampiran C, SDD |
| G-09 | H | **Model revenue v0 belum punya basis.** Input dan tabel prior per kategori belum ada, begitu pula ground truth untuk kalibrasi. | PRD BR-REV, Open question |
| G-10 | H | **NFR belum ada.** Belum ada target performa, availability, skala data, browser support, aksesibilitas, maupun lokalisasi. | PRD §NFR |
| G-11 | M | **Persona dan segmen beachhead belum eksplisit.** | PRD §3 |
| G-12 | H | **Role/RBAC belum didefinisikan** (owner, admin, member, dsb.). | PRD FR-AUTH |
| G-13 | H | **Keamanan webhook belum dispesifikasikan.** Belum ada signing, SSRF pada URL customer, retry, maupun replay protection. | SDD, Test |
| G-14 | H | **AI belum punya kriteria grounding/eval, batas biaya, dan aturan data** (data pribadi tidak dikirim ke LLM). | PRD FR-AI, SDD, Test |
| G-15 | M | **Metrik belum punya angka target.** Release gate masih kualitatif. | PRD §Success metrics |
| G-16 | H | **Belum ada design system/brand.** | `02-design-system/` |
| G-17 | H | **Belum ada strategi pengujian, CI/CD, environment, dan migrasi DB.** | `05-test-strategy.md`, `06-ci-cd.md` |
| G-18 | M | **Mekanisme suppression dan permintaan subjek data belum operasional.** Aturan governance §13 (UU PDP) belum diterjemahkan ke mekanisme. | PRD FR-AUD, SDD |
| G-19 | M | **Versioning dataset belum ada.** Agregasi geo harus *reproducible from the underlying input snapshot* (Rev 1.2 §8). | SDD §Dataset Snapshot |
| G-20 | M | **Kebutuhan feedback user belum ada.** Metrik "opportunity acceptance" butuh tombol terima/tolak di UI. | PRD FR-INTL |

---

## 6. Dampak pause crawler

Dengan crawler di-pause, **ketersediaan fitur bergantung pada field yang ada di dataset**. Prinsipnya: fitur tetap dibangun di atas Input Contract. Jika field sumbernya tidak ada, output-nya `Unknown`, bukan fitur yang dihapus. Begitu crawler aktif kembali (atau dataset bertambah), fitur langsung terisi tanpa perubahan engine.

| Kapabilitas | Sumber saat crawler aktif | Sumber selama pause | Perlakuan MVP |
|---|---|---|---|
| Identitas company (nama, alamat, kategori, koordinat, telepon, website) | Crawler + dataset | **Dataset** | P0, penuh |
| Status bisnis, jam operasional, rating, jumlah ulasan | Dataset | **Dataset** (jika ada) | P0 bila field tersedia |
| Email | Crawler (mailto, teks) | Dataset (jika ada), upload customer | P0. Coverage mungkin rendah dan ditampilkan apa adanya. |
| Person / decision maker | Crawler (team page, JSON-LD Person) | Upload customer, dataset (jika ada) | P0 untuk model data. Coverage terbatas. |
| WhatsApp | Crawler (wa.me) | Derived: nomor seluler (08xx/+628xx) → "kemungkinan WhatsApp" | Label **Inferred**, bukan Verified |
| Technology stack, booking/ecommerce/CRM presence | Crawler | Field dataset bila ada. Selain itu derived ringan tanpa fetch halaman: hostname website (platform gratisan, link sosial/linktree) dan DNS (MX, SPF) | Sebagian: "Digital presence lite" |
| Signal perubahan | Crawler change detection | **Diff antar snapshot dataset** | V1 |
| Refresh | Re-crawl | Re-run pipeline pada snapshot terbaru + re-verify (syntax/DNS/format) + re-score | MVP 1 (manual), V1 (terjadwal) |
| Verifikasi kontak | Crawler + verifier | Email: syntax + MX/DNS (opsional: provider verifikasi). Telepon: validasi format & tipe (libphonenumber). | P0 |

**Konsekuensi untuk positioning:** contoh unggulan di MVP Pack ("Opportunity: Online booking", "Digital maturity 58/100") membutuhkan bukti dari website. Selama pause, katalog opportunity MVP difokuskan pada tipe yang bisa dibuktikan dari dataset: tidak ada website, web presence hanya media sosial, listing tidak lengkap, reputasi/rating, kepadatan kompetisi, dan email non-domain. Lihat PRD Lampiran B.

---

## 7. Register keputusan (status v0.2)

Revisi 0.2 menerapkan arahan founder: **KISS, DRY, make it work → right → fast**. Tabel ini satu-satunya register keputusan (DRY).

| ID | Keputusan | Status |
|---|---|---|
| D-01 | Crawler di-PAUSE. Engine source-agnostic | Accepted |
| D-02 | Input F1 = DB gmaps read-only (D-31) + data wilayah GeoAPI (D-25). Upload daftar customer di F2 | Accepted (diperbarui D-31) |
| D-03 | Dua lapis data: data bisnis global (milik MSP, dibaca semua workspace) + data workspace privat (ICP, skor, list, unlock, ledger) | Accepted |
| D-04 | Offering menentukan relevansi opportunity | Accepted |
| D-05 | Bisnis (prospek) = satu tempat usaha, kunci `cid` Google (`google_place_id` disimpan bila ada, Q-13). Pengelompokan multi-cabang di F2 | Accepted (diperbarui D-31) |
| D-06 | Priority score disederhanakan ke 6 faktor yang datanya ada (PRD Lampiran A) | Accepted |
| D-07 | Band HOT ≥ 80 / WARM 60–79 / COLD < 60 | Accepted |
| D-08 | Credit: **unlock satu-satunya operasi berbayar** untuk kontak (1 credit, permanen). Export & kirim webhook gratis ("Unlock & export/kirim" menjalankan unlock dulu). Refund manual oleh ops | Accepted (disederhanakan) |
| D-09 | Plan Starter / Growth / Agency | Proposed (menunggu pricing) |
| D-10 | Integrasi CRM di F2. F1 cukup export + webhook | Accepted |
| D-11 | Modular monolith, satu binary Go | Accepted |
| D-12 | PostgreSQL + PostGIS + full-text search Postgres. Tanpa mesin search terpisah | Accepted |
| D-13 | Kunci wilayah = ID & kode area GeoAPI. Tanpa crosswalk Kemendagri/BPS di F1 | Accepted (menggantikan usulan lama) |
| D-14 | Data utama dan backup disimpan di Indonesia | Accepted |
| D-15 | AI: Claude, fallback ke API OpenAI-compatible | Accepted |
| D-16 | Beachhead: penjual B2B (agency digital, vendor SaaS UMKM) ke bisnis lokal | Diganti D-30 |
| D-17 | Brand Forest & Mint; heading Plus Jakarta Sans, body Inter | Accepted |
| D-18 | GitLab CE self-hosted, tanpa GitHub | Accepted |
| D-19 | Hosting: **satu dedicated server** untuk semuanya (production, staging, GitLab CE, CI runner, monitoring) — pilihan founder | Accepted |
| D-20 | Identity: Ory Kratos | Accepted |
| D-21 | Go (backend) + Next.js (frontend) | Accepted |
| D-22 | Dibangun agent AI (maks. 8 paralel) yang diorkestrasi Claude. Founder menyetujui gerbang fase dan deploy produksi | Accepted |
| D-23 | Prinsip KISS, DRY, make it work → right → fast | Accepted |
| D-24 | Dataset = JSON-LD schema.org per tempat usaha, kunci `google_place_id` | Diganti D-31 |
| D-25 | Wilayah & populasi dari GeoAPI (Kontur 2023, perkiraan model), di-sync ke PostGIS | Accepted |
| D-26 | Infra F1 minimal: 1 server, tanpa Redis, PgBouncer, object storage internal, maupun HA. Mitigasi: batas resource per container, runner 1 job, uptime monitor eksternal, backup off-site | Accepted |
| D-27 | Isu hukum (PDP, lisensi data, retensi) ditangani tim legal | Accepted |
| D-28 | Dataset global & diperbarui berkala; **F1 melayani Indonesia** (filter negara saat import). Negara lain diaktifkan kelak per negara | Accepted |
| D-29 | Model AI (provider, model, fallback) dan plan (harga tampilan, seat, kuota) **diatur dari dashboard ops**, tidak mengikat | Accepted |
| D-30 | Posisi: **market intelligence untuk bisnis lokal Indonesia**. Pengguna utama F1: pemilik usaha F&B/multi-outlet dan investor/analis. Fitur prospek (unlock, export, webhook) tetap ada untuk penjual B2B | Accepted |
| D-31 | Input = **DB gmaps dibaca langsung** (read-only, inkremental per waktu scrape). Export JSON-LD tidak dipakai karena membuang `popular_times`, harga, distribusi bintang, dan riwayat snapshot | Accepted |
| D-32 | Estimasi di F1 (PRD Lampiran E): laju ulasan dari snapshot → kunjungan → omzet untuk F&B, plus okupasi & omzet kamar hotel. Selalu rentang + asumsi + tanggal. Parameter yang tidak terukur diatur dari dashboard ops | Accepted |
| D-33 | Analisis lokasi di F1 (PRD Lampiran F): pola keramaian tetangga, jam inti, posisi harga, keyakinan. Divalidasi dengan dua sample case founder (Warung Kita, Signa Cafe) sebagai golden test | Accepted |

## 8. Pertanyaan terbuka

| ID | Pertanyaan | Status |
|---|---|---|
| Q-01 | Skema dataset | **Terjawab:** DB gmaps (PRD §7.1, D-31) |
| Q-02 | Jumlah record & cakupan wilayah dataset | **Terjawab:** full, seluruh dunia (F1: Indonesia, D-28) |
| Q-03 | Kadens pembaruan dataset | **Terjawab:** scrape berulang; import inkremental harian (D-31) |
| Q-04 / Q-10 | Populasi & batas wilayah | **Terjawab:** GeoAPI |
| Q-05 | Aturan pakai/tampil/export data | Tim legal |
| Q-06 | Data acuan revenue | **Terjawab:** benchmark di dashboard ops (D-32). Omzet nyata usaha tidak tersedia dan tidak dijanjikan |
| Q-07 | Calon customer beta & Offering mereka | **Terjawab:** peserta sudah ada |
| Q-08 | GitHub tetap jadi mirror atau ditinggalkan? | **Terjawab:** GitLab repo utama; mirror GitHub opsional, tanpa CI |
| Q-09 | Spesifikasi satu server & lokasi backup off-site | Terbuka |
| Q-11 | Validasi hukum PDP | Tim legal |
| Q-12 | Akses jaringan dari server MesinProspek ke DB gmaps (lokasi DB, cara koneksi, akun read-only), plus index (`created_at`, `id`) di tabel sumber untuk import inkremental | Terbuka |
| Q-14 | Cakupan data Indonesia per kota & kelompok: % tempat dengan riwayat ≥ 30 hari, dengan `popular_times`, dengan harga | Terbuka (query disiapkan untuk founder) |
| Q-13 | Skema lengkap tabel sumber di DB gmaps (nama kolom ID tempat, telepon, website, sosial, koordinat) | **Terjawab:** sampel record lengkap dari founder (PRD §7.1, pemetaan kolom di 04) |

## 9. Peta dokumen lanjutan

Lihat [README](README.md).
