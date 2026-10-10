# 00 — Analisis Dokumen Awal

| Field | Value |
|---|---|
| Dokumen | Analisis dokumen awal → dasar dokumen lanjutan |
| Status | Draft v0.1 |
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

## 7. Register keputusan (usulan default, butuh konfirmasi)

Status: **Proposed** = default yang dipakai dokumen lanjutan sampai dikonfirmasi; **Open** = belum ada default.

| ID | Keputusan | Default yang diusulkan | Alternatif | Dampak | Status |
|---|---|---|---|---|---|
| D-01 | Status crawler | PAUSED. Engine source-agnostic. Crawler kelak masuk sebagai source adapter yang memenuhi Input Contract. | — | Scope MVP | **Accepted** (arahan founder) |
| D-02 | Input MVP | (a) Dataset bisnis yang sudah ada, di-import sebagai **dataset snapshot** berversi. (b) Dataset populasi/wilayah per kelurahan. (c) Upload CSV/XLSX oleh customer. | Partner API | Skema, pipeline | Proposed |
| D-03 | Tenancy data | **Dua lapis.** *Global layer* (milik MSP): company, establishment, kontak publik, atribut, metrik area, dibaca semua tenant. *Tenant layer* (privat): ICP, Offering, skor ICP/priority, opportunity relevance, segment, tag, note, assignment, upload, unlock, activity. Record hasil upload hanya di-*link* ke entity global, tidak di-merge ke global. | Semua per tenant (mahal, tanpa network effect) | Skema, RLS, biaya | Proposed |
| D-04 | Objek Offering | Tenant mendefinisikan ≥1 **Offering** (apa yang dijual: website, booking system, SEO lokal, dsb.). Opportunity global = "gap fact". Relevansi gap ke Offering = opportunity tenant. | Opportunity global tanpa konteks | Intelligence, UX onboarding | Proposed |
| D-05 | Model entity | `Company` (entitas bisnis/brand) 1—N `Establishment` (lokasi fisik/cabang). Geo dan kontak lokasi menempel pada Establishment. | Satu tabel Company | Skema, ER | Proposed |
| D-06 | Priority score | Bobot Rev 1.2 §9 sebagai default. Faktor Unknown dikeluarkan dan bobot di-renormalisasi. `coverage` < 0,6 → badge "Data terbatas" dan tidak boleh berlabel HOT. Bobot dapat diatur per Offering preset (MVP) dan per tenant (V1.5). | Rumus §06 | Scoring | Proposed |
| D-07 | Score band | HOT ≥ 80, WARM 60–79, COLD < 60, UNSCORED. Dapat dikonfigurasi. | 4 band | UX, tokens | Proposed |
| D-08 | Model credit | Meter `credit` + `ai_credit`. Search/profil tidak memakai credit (fair-use per plan). Semua nilai contact point dan nama person ditampilkan ter-mask sampai **unlock** (1 credit/record, permanen per tenant). Export/sync record yang sudah di-unlock gratis. Refund otomatis untuk operasi gagal dan untuk kontak yang terbukti INVALID ≤ 30 hari setelah unlock. | Charge per export | Billing, UX | Proposed |
| D-09 | Plan | Starter (≈ paket Prospect), Growth (≈ Intelligence + Workflow), Agency, Enterprise. Harga di luar dokumen ini. | — | Billing | Proposed |
| D-10 | Integrasi CRM | MVP 1: webhook + export. V1: satu CRM (kandidat ditentukan dari survei customer beta). | CRM di MVP | Scope | Proposed |
| D-11 | Arsitektur | **Modular monolith** (satu codebase backend dengan modul sesuai service boundary §11) + worker terpisah. Pecah menjadi service hanya bila ada alasan skala. | 6 microservice | Ops, CI/CD | Proposed |
| D-12 | Stack data | PostgreSQL + **PostGIS** + Postgres FTS/pg_trgm (tanpa OpenSearch di MVP), storage S3-compatible. Bahasa backend: lihat D-21. | OpenSearch sejak awal | Infra | Proposed |
| D-13 | Kode wilayah kanonik | Kode **Kemendagri** (Kepmendagri kode & data wilayah terbaru), plus crosswalk ke kode BPS bila sumber populasi memakai kode BPS. Setiap metrik populasi wajib menyimpan periode referensi. | Kode BPS | Geo | Proposed |
| D-14 | Residensi data | Semua penyimpanan utama (DB, storage, backup utama) di **data center Indonesia**. Backup off-site juga di Indonesia. Panggilan AI hanya membawa data level company, tidak membawa nilai kontak pribadi. | Luar negeri | Kepatuhan, infra | Proposed |
| D-15 | Penyedia AI | Abstraksi provider. Default diusulkan: model Claude (kelas Sonnet untuk brief, kelas Haiku untuk klasifikasi). | Provider lain | Biaya, eval | Open |
| D-16 | Beachhead | Penjual B2B yang menargetkan **bisnis lokal Indonesia yang punya lokasi fisik**: agency digital (website, ads, SEO lokal), vendor SaaS SMB (booking, POS, CRM), distributor. Contoh vertikal awal: klinik gigi/kesehatan, F&B, retail. | Enterprise B2B | Persona, opportunity catalog | Proposed |
| D-17 | Brand | Token warna brand bersifat **placeholder** sampai identitas visual final. Struktur token tidak berubah saat brand diganti. | — | Design tokens | Open |
| D-18 | Platform CI/CD & host kode | **Tanpa GitHub** (arahan founder). Default: **GitLab CE self-hosted** (Git, merge request, GitLab CI, Container Registry, runner Docker di server CI terpisah). | Forgejo/Gitea + Actions, Woodpecker, Jenkins | CI/CD | **Accepted** (tanpa GitHub) / Proposed (GitLab CE) |
| D-19 | Hosting & runtime | **Dedicated server** (arahan founder). Default: Docker Compose + Traefik (blue/green), PostgreSQL + PostGIS bare-metal dengan replika streaming + pgBackRest, Redis, SeaweedFS (S3-compatible; MinIO community sudah diarsipkan, alternatif Garage), Ansible untuk provisioning, SOPS + age untuk secret, monitoring self-hosted. | Docker Swarm / k3s saat > 2 app server | Infra, CI/CD, SDD | **Accepted** (dedicated) / Proposed (stack) |
| D-20 | Identity provider | Disarankan self-hosted (**Ory Kratos**: verifikasi email, login Google via OIDC, TOTP MFA) agar data identitas user tetap di Indonesia. Aplikasi dibuat provider-agnostic. | Managed OIDC SaaS, Keycloak | Auth, kepatuhan | Open |
| D-21 | Bahasa backend | **Go** untuk API dan worker, **Next.js + TypeScript** untuk UI (keputusan founder). Menggantikan FastAPI/Python di MVP Pack §11. Alasan: pipeline data jutaan baris, query SQL/PostGIS berat (sqlc), worker konkuren, efisiensi resource di dedicated server. Python boleh dipakai offline (notebook) untuk kalibrasi model kelak, di luar service. | NestJS + Next.js; Python + Next.js | Semua dokumen engineering | **Accepted** |

---

## 8. Pertanyaan terbuka (butuh jawaban pemilik dataset/founder)

| ID | Pertanyaan | Kenapa penting |
|---|---|---|
| Q-01 | Skema dan field dataset bisnis yang tersedia (daftar kolom, tipe, % terisi)? | Menentukan fitur mana yang aktif dan mana yang Unknown (§6). |
| Q-02 | Volume (jumlah record), cakupan wilayah, dan tanggal pengambilan data? | Sizing NFR dan informasi freshness. |
| Q-03 | Apakah dataset akan diperbarui? Dengan kadens berapa? | Desain snapshot, refresh, dan signal (V1). |
| Q-04 | Sumber dan periode dataset populasi per kelurahan, serta sistem kode wilayah yang dipakai? | D-13, label periode referensi. |
| Q-05 | Ketentuan lisensi/penggunaan dataset untuk penyimpanan, tampilan, dan export ke customer (guardrail Rev 1.2 §8)? | Aturan export dan tampilan. |
| Q-06 | Apakah ada data ground truth revenue/karyawan untuk kalibrasi model revenue? | Confidence revenue (G-09). |
| Q-07 | Siapa 5–10 calon customer beta dan Offering mereka? | Prioritas opportunity catalog dan preset bobot. |
| Q-08 | Apakah source code ikut pindah sepenuhnya dari GitHub ke GitLab self-hosted, atau GitHub tetap ada sebagai mirror? Default usulan: GitLab primary, mirror GitHub opsional (= `06` Q-CI-01). | Setup repo, CI/CD (D-18). |
| Q-09 | Spesifikasi dan jumlah dedicated server yang tersedia/dianggarkan, serta lokasi backup off-site? | Topologi, HA, DR (D-19). |
| Q-10 | Apakah tersedia data batas wilayah (poligon kelurahan/kecamatan) beserta lisensinya? | Peta choropleth FR-GEO-002 (A-03). |

---

## 9. Peta dokumen lanjutan

| No | Dokumen | Isi | Bergantung pada |
|---|---|---|---|
| 01 | `01-prd.md` | Product Requirements: persona, scope per release, FR/NFR ber-ID, business rules, metrik | Analisis ini |
| 02 | `02-design-system/` | Design tokens (DTCG JSON + CSS variables), pedoman visual, komponen inti | PRD |
| 03 | `03-ux-spec.md` | Information architecture, user flow, spesifikasi layar dan state | PRD, 02 |
| 04 | `04-technical-design.md` | Arsitektur, Input Contract, data model, pipeline, scoring engine, API & events, keamanan | PRD |
| 05 | `05-test-strategy.md` | Strategi dan rencana pengujian, test case, edge case, traceability ke FR | PRD, 04 |
| 06 | `06-ci-cd.md` | Branching, pipeline GitLab CI, quality gates, environment dedicated server, deployment, migrasi DB, backup/DR, release & rollback | 04, 05 |

Konvensi ID yang dipakai di semua dokumen: `D-` keputusan, `I-` inkonsistensi, `G-` gap, `Q-` pertanyaan, `FR-<AREA>-nnn` functional requirement, `NFR-<CAT>-nn`, `BR-<AREA>-nn` business rule, `TC-<AREA>-nnn` test case, `EC-nn` edge case, `QG-nn` quality gate.
