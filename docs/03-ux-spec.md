# 03 — UX Spec (lean)

| Field | Value |
|---|---|
| Versi | 0.2 (lean). Menggantikan v0.1 (25 layar + V1) |
| Tanggal | 2026-10-10 |
| Basis | `01-prd.md` v0.2 (**sumber kebenaran**; bila bertentangan, PRD yang berlaku) · `02-design-system/` tokens `0.2.0` |
| Cakupan | Hanya **F1**. F2/F3 cukup di §10 |

> Konvensi: `SCR-nn` layar, `FLOW-nn` alur. Aturan bisnis **tidak disalin**, cukup dirujuk ke ID PRD (DRY). Nama token persis seperti di `02`. Dataset **tidak punya email maupun data orang** (PRD §7.1). Kontak = telepon, website, akun sosial (`sameAs`), dan tautan Google Maps.

---

## 1. Prinsip UX

1. **Satu layar, satu pertanyaan.** Setiap SCR menuliskan pertanyaan yang dijawabnya. Informasi di luar itu dipindah ke drawer atau tab.
2. **Alasan sebelum angka.** Skor selalu ditulis "untuk ICP X", dan "Kenapa prospek ini?" bisa dibuka dengan satu klik dari skor mana pun (FR-INTL-04).
3. **Trust terlihat, Unknown itu sah.** Setiap nilai membawa claim type dan evidence. Field kosong tampil "Tidak diketahui" (BR-TRUST-01, BR-TRUST-03).
4. **Biaya selalu eksplisit.** Tombol berbayar menyebut biayanya. Tidak ada debit tanpa dialog konfirmasi (BR-CRED-01).
5. **Gagal terlihat dan bisa diulang.** Error ditampilkan per bagian, lengkap dengan "Coba lagi". Aksi yang gagal tidak ditagih.

---

## 2. Navigasi & daftar layar F1

### 2.1 Kerangka

| Elemen | Isi & perilaku |
|---|---|
| Sidebar (≥ `breakpoint.lg`, di bawahnya jadi drawer) | Prospek · Wilayah · List · Export · Pengaturan. Item di luar role tidak dirender |
| Header | Nama workspace · **ICP aktif** (dropdown, FR-ICP-02) · saldo "Credit 184 · AI 42" · menu user (Akun, Keluar) |
| Banner (maks. 1) | Prioritas: fitur dimatikan > skor sedang dihitung > info |
| URL | Query, filter, urutan, dan ICP tersimpan di URL (`/prospects?q=&f=&sort=&icp=`) sehingga bisa dibagikan dan disimpan (FR-SRCH-02). Evidence drawer memakai `?evidence=<field>` agar tombol Back menutupnya |
| Ops console | Layout terpisah di `/ops`, khusus akun ops dengan MFA (FR-AUTH-03). Desktop saja |
| Responsive | Desain utama `breakpoint.xl`. Profil dan List wajib nyaman di 360 px (NFR-08); layar lain cukup tetap terbaca |

### 2.2 Daftar layar (dialog dan evidence drawer adalah pola §5, bukan layar)

| SCR | Layar | Route (usulan) | Role | FR |
|---|---|---|---|---|
| SCR-01 | Masuk & daftar (flow Kratos) | `/login`, `/signup`, `/verify`, `/recovery`, `/account`, `/invite/[token]` | Semua | FR-AUTH-01, -02, -03 |
| SCR-02 | Onboarding: Offering → ICP → hasil pertama | `/onboarding` | Admin | FR-ICP-01, FR-ICP-02, FR-AUTH-02 |
| SCR-03 | Prospek (search + filter + hasil) | `/prospects` | Admin, Member | FR-SRCH-01, -02, FR-INTL-04, FR-ACT-01..04 |
| SCR-04 | Profil prospek | `/prospects/[id]` | Admin, Member | FR-PROF-01, FR-INTL-01..05, FR-DATA-02, FR-ACT-01, -02, -04, -05, FR-PRIV-02 |
| SCR-05 | Wilayah (peringkat + peta + drilldown) | `/areas` | Admin, Member | FR-GEO-01, FR-INTL-03, FR-DATA-04 |
| SCR-06 | List & pencarian tersimpan | `/lists`, `/lists/[id]` | Admin, Member | FR-ACT-01, FR-SRCH-02 |
| SCR-07 | Export (dialog biaya + riwayat) | `/exports` | Admin, Member | FR-ACT-03, FR-BILL-01 |
| SCR-08 | Pengaturan › ICP & anggota | `/settings/icp`, `/settings/members` | Admin (Member: lihat ICP) | FR-ICP-02, FR-AUTH-02 |
| SCR-09 | Pengaturan › Webhook | `/settings/webhooks` | Admin | FR-ACT-04 |
| SCR-10 | Pengaturan › Credit & pemakaian | `/settings/usage` | Admin | FR-BILL-01 |
| SCR-11 | Ops › Workspace, credit & audit | `/ops/workspaces`, `/ops/audit` | Ops | FR-OPS-01, FR-OPS-02, FR-BILL-01 |
| SCR-12 | Ops › Data & kontrol (import, kesehatan data, flag, suppression) | `/ops/imports`, `/ops/health`, `/ops/flags`, `/ops/suppression` | Ops | FR-OPS-01, FR-DATA-01, -03, -04, FR-PRIV-02 |

### 2.3 Role (FR-AUTH-02; server tetap menegakkan, akses lewat URL di luar role → DITOLAK)

| Aksi | Admin | Member | Ops |
|---|:-:|:-:|:-:|
| Search, profil, wilayah, unlock, list/tag/catatan, export, kirim ke webhook, AI brief | ✓ | ✓ | — |
| Pilih ICP aktif | ✓ | ✓ | — |
| Kelola ICP, anggota, endpoint webhook; lihat pemakaian credit | ✓ | — | — |
| Console ops (grant credit, flag, suppression, audit) | — | — | ✓ (MFA) |

---

## 3. Alur F1

### FLOW-01 — Onboarding: daftar → hasil pertama ≤ 5 menit (FR-ICP-01, PRD §4)

| # | Layar | Langkah | Catatan |
|---|---|---|---|
| 1 | SCR-01 | Daftar (email + password atau Google) → verifikasi email | Data tidak bisa diakses sebelum email terverifikasi |
| 2 | SCR-02 | Workspace dibuat otomatis → pilih **Offering** (chip tag) | Preset kriteria ICP diisi dari tag. Anggota undangan melewati langkah 2–4 dan langsung ke SCR-03 |
| 3 | SCR-02 | Isi **ICP**: kategori, wilayah, kondisi digital, minimum rating/ulasan. Tiap kriteria *wajib* atau *preferensi* | Jumlah prospek cocok diperbarui saat kriteria berubah |
| 4 | SCR-02 | Simpan → progres "Menghitung skor…" | State MENGHITUNG |
| 5 | SCR-03 | Hasil pertama + coach mark: (1) skor & band, (2) "Kenapa prospek ini?", (3) kontak ter-mask & biaya unlock | Event `first_result_viewed` |

### FLOW-02 — Prospecting harian

| # | Layar | Langkah | Catatan |
|---|---|---|---|
| 1 | SCR-03 | Cari kata kunci + filter (atau buka pencarian tersimpan) | Urut skor untuk ICP aktif |
| 2 | SCR-04 | Buka profil → baca "Kenapa prospek ini?" dan opportunity → beri feedback Relevan/Tidak relevan | FR-INTL-05 |
| 3 | SCR-04 | Buka evidence nilai yang diragukan | Drawer §5 |
| 4 | SCR-04 | **Unlock · 1 credit** → dialog → nomor terbuka → Salin / Telepon / WhatsApp | Unlock ulang tidak ditagih (FR-ACT-02) |
| 5 | SCR-04/03 | Simpan ke list, tag, catatan, tugaskan ke anggota | FR-ACT-01 |
| 6 | SCR-03/06 | Pilih banyak → **Export** (SCR-07) atau **Kirim ke webhook** | Dialog biaya menampilkan jumlah yang perlu di-unlock |

### FLOW-03 — Wilayah → drilldown (PER-03)

| # | Layar | Langkah | Catatan |
|---|---|---|---|
| 1 | SCR-05 | Pilih kab/kota → level Kecamatan atau Kelurahan | Default: kab/kota dari ICP aktif |
| 2 | SCR-05 | Baca peringkat (tabel) dan peta, lalu pilih area dari baris tabel atau poligon | Area "Data terbatas" diarsir dan tidak diberi peringkat (Lampiran C) |
| 3 | SCR-05 | Panel area: metrik + "Lihat N prospek" | Populasi berlabel "perkiraan model Kontur 2023" |
| 4 | SCR-03 | Hasil prospek dengan filter wilayah terisi → simpan pencarian | FR-SRCH-02 |

### FLOW-04 — AI brief (FR-ACT-05)

| # | Layar | Langkah | Catatan |
|---|---|---|---|
| 1 | SCR-04 | Panel AI brief: bila ada cache (≤ 7 hari) langsung tampil, gratis | "Dibuat 3 hari lalu · berlaku sampai 13 Okt" |
| 2 | SCR-04 | Bila belum ada: **Buat brief · 1 AI credit** → progres (≤ 20 s, NFR-02) | Nomor telepon tidak dikirim (BR-AI-01) |
| 3 | SCR-04 | Brief tampil per bagian. Tiap kalimat punya rujukan fakta `[n]` yang membuka evidence | Label "Dibuat AI dari data profil ini" |
| 4 | SCR-04 | Gagal validasi → sistem membuat ulang sekali secara otomatis. Bila masih gagal → error, tidak ditagih | Kill switch aktif → state FITUR DIMATIKAN |

### FLOW-05 — Ops import dataset (FR-DATA-01, PER-04)

| # | Layar / alat | Langkah | Catatan |
|---|---|---|---|
| 1 | CLI | Ops menjalankan `msp tool import <file.jsonl.gz>` (bila area belum ada: `msp tool geo-sync`) | Import ulang file yang sama tidak membuat duplikat |
| 2 | SCR-12 Import | Job `import` tampil di riwayat: status, progres, lalu laporan baru / berubah / tidak berubah / ditolak (per alasan) | Laporan bisa diunduh sebagai CSV |
| 3 | SCR-12 Import | Penetapan wilayah + job `score` berjalan → status "Skor diperbarui" | Customer melihat banner MENGHITUNG bila skor ICP-nya berubah |
| 4 | SCR-12 Kesehatan | Cek jumlah prospek, % terisi per field, % terpetakan wilayah | Target ≥ 98% terpetakan (FR-DATA-04). Di bawah target → indikator peringatan |

---

## 4. Spesifikasi layar

### 4.0 State baku (berlaku di semua layar; tiap SCR hanya menulis yang khusus)

| Kode | Kapan | Perilaku UI |
|---|---|---|
| LOADING | Data pertama dimuat | Skeleton sesuai bentuk konten. Lebih dari 10 s → teks "Masih memuat…" |
| KOSONG | Objek belum pernah dibuat | 1 kalimat manfaat + 1 CTA (`EmptyState`) |
| NOL HASIL | Filter tanpa hasil | Daftar chip filter aktif + "Hapus filter" per chip + "Reset semua" |
| TIDAK DIKETAHUI | Field tidak ada di dataset | "Tidak diketahui" + ikon + tooltip alasan. Tidak pernah kosong, "0", atau "—" (BR-TRUST-03) |
| DATA TERBATAS | Skor: BR-SCORE-02 · Area: Lampiran C | Badge "Data terbatas" + tooltip (§5) |
| MENGHITUNG | ICP baru/diubah sedang dinilai | Banner progres. Prospek yang belum dinilai tampil "Belum dinilai" |
| ERROR | Request gagal | Error per bagian (bukan seluruh halaman) + "Coba lagi" + kode referensi (§7) |
| CREDIT KURANG | Saldo < biaya | Tombol tetap terlihat dengan biayanya. Klik → dialog: kebutuhan vs saldo + "Hubungi Admin" (Member) / "Hubungi tim MesinProspek" (Admin). Top-up manual oleh ops (FR-BILL-01) |
| FITUR DIMATIKAN | Flag AI / export / webhook off (FR-OPS-01) | Kontrol **nonaktif, tidak disembunyikan** + tooltip. Bila server membalas 503 → toast |
| DITOLAK | Di luar role (403) | Menjelaskan apa yang tidak bisa diakses dan siapa Admin-nya |

### SCR-01 — Masuk & daftar

| | |
|---|---|
| Pertanyaan | "Bagaimana saya masuk ke workspace saya?" |
| Isi | Flow Ory Kratos (login, registration, verification, recovery, settings) yang dirender Next.js dengan token kita: kartu tengah maks. 400 px di atas `color.bg.canvas`. Tombol "Lanjut dengan Google", email + password. Akun: ganti password, tautkan Google, MFA TOTP |
| Undangan | `/invite/[token]` menampilkan nama workspace + pengundang → daftar/masuk → bergabung sebagai Member |
| State khusus | Email belum terverifikasi → hanya layar verifikasi. Undangan kedaluwarsa. Pesan login gagal tidak membocorkan apakah email terdaftar. Ops tanpa TOTP → diarahkan ke pendaftaran TOTP (FR-AUTH-03) |

### SCR-02 — Onboarding (Offering → ICP → hasil pertama)

| | |
|---|---|
| Pertanyaan | "Apa yang saya jual, dan ke siapa?" |
| Langkah 1 | **Offering**: chip tag `website_dev` (Pembuatan website), `social_media`, `local_seo`, `digital_ads`, `reputation_mgmt`, `booking_system`, `other` (label id-ID + 1 baris deskripsi). Boleh lebih dari 1; tag pertama = primer |
| Langkah 2 | **ICP**: kategori (taksonomi 2 level), wilayah (provinsi → kab/kota → kecamatan), kondisi digital (mis. "Tanpa website domain sendiri", "Ada telepon seluler"), minimum rating & ulasan. Toggle *Wajib/Preferensi* per kriteria. Panel kanan: "± 418 prospek cocok" |
| Langkah 3 | Progres skor → masuk SCR-03 dengan coach mark (FLOW-01) |
| Aksi | Kembali (isian tetap tersimpan) · Simpan. Tidak ada "Lewati" karena ICP dibutuhkan untuk skor |
| State khusus | Prospek cocok = 0 → saran melonggarkan kriteria *wajib*. Workspace baru dengan saldo 0 → coach mark (3) menyebut "Credit belum tersedia — hubungi tim kami" |

### SCR-03 — Prospek

| | |
|---|---|
| Pertanyaan | "Prospek mana yang paling layak saya hubungi sekarang?" |
| Isi | Search kata kunci (nama, kategori, alamat). Panel filter: kategori, wilayah, band, tipe opportunity, ada telepon / website / sosial, rating (FR-SRCH-01). Header hasil: jumlah tepat + "untuk ICP X" + urutan. Baris: `ScoreBadge` · nama, kategori, kelurahan, freshness · opportunity teratas (1 baris alasan) · indikator kontak & digital (teks, bukan ikon saja) · rating. "Muat 50 lagi" |
| Aksi | Buka profil · Simpan pencarian · Pilih banyak → bar aksi: Simpan ke list, Tag, Tugaskan, Unlock (n credit), Export, Kirim ke webhook. "Pilih semua hasil" maks. 10.000 |
| State khusus | NOL HASIL · MENGHITUNG (urutan bisa berubah → "Muat ulang dari atas") · DATA TERBATAS pada baris · FITUR DIMATIKAN (Export/Webhook di bar aksi) |

### SCR-04 — Profil prospek

| | |
|---|---|
| Pertanyaan | "Kenapa prospek ini, dan bagaimana saya mendekatinya?" |
| Header | Nama · kategori (Turunan) · alamat · kelurahan › kecamatan · freshness · "Lihat di Google Maps" · `ScoreBadge` besar "untuk ICP X" · "Data terbatas" bila berlaku |
| Kenapa prospek ini? | `FactorBar` 6 faktor Lampiran A: nilai, bobot, kontribusi, claim type. Faktor tidak diketahui: "Tidak dihitung" (BR-SCORE-01). Footer: coverage, `priority-X.Y.Z`. Expand ICP Fit → kriteria Cocok / Sebagian / Tidak / Tidak diketahui + Wajib/Preferensi |
| Opportunity | Kartu per opportunity relevan (Lampiran B): judul, alasan + pembanding, badge Perkiraan, relevansi primer/sekunder, "Lihat evidence", **Relevan / Tidak relevan** (FR-INTL-05). Gap yang tidak relevan dengan Offering hanya tampil di Kehadiran digital |
| Kontak | Telepon ter-mask + "Kemungkinan WhatsApp" bila seluler (BR-NORM-01). **Unlock · 1 credit** → nilai penuh + Salin / Telepon / WhatsApp (`wa.me`) + "Di-unlock oleh Rina · 10 Okt". Nilai yang di-suppress: "Kontak tidak tersedia", tombol Unlock tidak muncul (FR-PRIV-02) |
| Kehadiran digital | Website + klasifikasi BR-NORM-02 (domain sendiri / sosial / link aggregator / `wa.me` / subdomain gratis / kosong) · akun sosial dari `sameAs` · jam buka · rating & jumlah ulasan. Website, sosial, dan Maps **tidak** ter-mask |
| Konteks wilayah | Kelurahan: populasi (perkiraan model Kontur 2023), jumlah & densitas kategori per 1.000, kompetisi Rendah/Sedang/Tinggi, `market_score` → "Lihat wilayah" (SCR-05) |
| AI brief | FLOW-04. Teks `text.body-lg`, rujukan `[n]` |
| Aksi & kerja tim | Unlock · Simpan ke list · Tugaskan · Kirim ke webhook · ‹ › prospek berikutnya. Panel list, tag, catatan, pemilik (FR-ACT-01) |
| State khusus | Setiap nilai punya claim type + evidence (§5). AI dimatikan → FITUR DIMATIKAN. Di 360 px: satu kolom akordeon + bar bawah sticky [Unlock · 1 credit] [Aksi lain] |

### SCR-05 — Wilayah

| | |
|---|---|
| Pertanyaan | "Kecamatan/kelurahan mana yang paling potensial untuk ICP saya?" |
| Isi | Pemilih kab/kota + level (Kecamatan / Kelurahan). **Tabel peringkat** (urut `market_score`, lalu jumlah HOT + WARM; Lampiran C): peringkat, area, `market_score`, prospek cocok, HOT/WARM, kompetisi, populasi. **Peta** choropleth `market_score` dari vector tiles GeoAPI (`color.geo.sequential.1..7`, kuantil dalam kab/kota). Panel area terpilih: semua metrik Lampiran C |
| Aksi | Pilih area (baris atau poligon, tersinkron) · "Lihat N prospek" → SCR-03 dengan filter wilayah · zoom/reset |
| Wajib tampil | Legend + "Populasi: perkiraan model Kontur 2023" + atribusi "Batas & populasi: GeoAPI · Peta dasar © kontributor OpenStreetMap" (PRD §7.2, FR-INTL-03) |
| State khusus | Area Data terbatas: `color.geo.no-data` + arsir, metrik tetap tampil, tanpa peringkat. Tiles gagal → peta error + tabel tetap jalan. Di < `breakpoint.lg`: tab Tabel / Peta (default Tabel) |

### SCR-06 — List & pencarian tersimpan

| | |
|---|---|
| Pertanyaan | "Prospek apa yang sedang dikerjakan tim saya?" |
| Isi | Tab **List**: nama, jumlah, pemilik, diubah. Detail list = tabel baris SCR-03 + kolom pemilik, tag, catatan. Tab **Pencarian tersimpan**: nama, ringkasan filter, ICP, jumlah hasil saat ini (dinamis, FR-SRCH-02) |
| Aksi | Buat/ganti nama/hapus list · hapus dari list · tugaskan · tag · Export · Kirim ke webhook · Jalankan pencarian |
| State khusus | KOSONG: "Simpan prospek ke list untuk dikerjakan bersama tim." CTA: Cari prospek. Di 360 px: daftar kartu |

### SCR-07 — Export

| | |
|---|---|
| Pertanyaan | "Berapa biayanya, dan di mana file saya?" |
| Dialog biaya | Jumlah prospek · sudah di-unlock (gratis) · akan di-unlock (n credit) · saldo sebelum → sesudah · format CSV/XLSX · "Credit hanya ditagih bila export berhasil" (FR-ACT-03, BR-CRED-01). Prospek yang di-suppress dikecualikan dan disebutkan jumlahnya |
| Riwayat | Tabel job: dibuat, oleh, jumlah baris, credit, status (Antre / Diproses / Selesai / Gagal), Unduh / Coba lagi. Status job diperbarui otomatis |
| State khusus | CREDIT KURANG · FITUR DIMATIKAN · Gagal → "Tidak ada credit yang ditagih" · lebih dari 10.000 baris → minta persempit filter |

### SCR-08 — Pengaturan › ICP & anggota

| | |
|---|---|
| Pertanyaan | "Siapa tim saya dan target apa yang kami kejar?" |
| ICP | Daftar ICP (nama, Offering, jumlah prospek cocok). Buat/ubah memakai form langkah 1–2 SCR-02. Simpan → MENGHITUNG. Member: hanya lihat & pilih ICP aktif |
| Anggota | Tabel anggota (nama, email, role, status undangan) · Undang (email + role) · ubah role · keluarkan · kirim ulang undangan |
| State khusus | Admin terakhir tidak bisa diturunkan atau dikeluarkan. DITOLAK untuk Member di tab Anggota |

### SCR-09 — Pengaturan › Webhook

| | |
|---|---|
| Pertanyaan | "Apakah prospek terkirim ke sistem saya?" |
| Isi | Daftar endpoint (nama, URL, status aktif). Tambah: URL HTTPS → validasi. Secret HMAC tampil sekali saat dibuat + "Salin". Log pengiriman: waktu, jumlah prospek, kode respons, percobaan, retry berikutnya (FR-ACT-04) |
| Aksi | Tambah · Kirim uji · nonaktifkan · hapus · rotasi secret · kirim ulang dari log |
| Kirim prospek | Dari SCR-03/04/06: pilih endpoint → dialog. Prospek yang belum di-unlock ikut di-unlock (n credit) sebelum dikirim |
| State khusus | URL ke IP privat/internal ditolak (copy §7). Uji gagal → kode respons + saran. FITUR DIMATIKAN |

### SCR-10 — Pengaturan › Credit & pemakaian

| | |
|---|---|
| Pertanyaan | "Credit kami terpakai untuk apa?" |
| Isi | Saldo credit & AI credit (= jumlah ledger, FR-BILL-01) · plan & kuota (Lampiran D, info saja) · tabel ledger: tanggal, jenis (Grant / Debit / Refund), jumlah, oleh, objek (prospek / export / brief), filter per bulan |
| State khusus | KOSONG: "Belum ada pemakaian." Info: "Penambahan credit dilakukan oleh tim MesinProspek" + kontak |

### SCR-11 — Ops › Workspace, credit & audit

| | |
|---|---|
| Pertanyaan | "Customer mana yang perlu dilayani, dan siapa melakukan apa?" |
| Workspace | Tabel: nama, plan, anggota, saldo credit/AI, aktivitas terakhir. Detail: ledger + form **Grant credit** (jenis, jumlah, alasan wajib) + refund manual per debit (BR-CRED-01) |
| Audit | Tabel append-only (FR-OPS-02): waktu, aktor, workspace, aksi, objek. Filter jenis/aktor/workspace/tanggal. Tanpa edit maupun hapus |
| State khusus | Grant ganda dicegah idempotency (tombol nonaktif saat proses). Hanya akun ops dengan MFA |

### SCR-12 — Ops › Data & kontrol

| | |
|---|---|
| Pertanyaan | "Apakah data sehat dan sistem terkendali?" |
| Import | Riwayat job `import`: file, waktu, status, baru/berubah/tidak berubah/ditolak per alasan, unduh laporan (FR-DATA-01). Riwayat `geo-sync` (FR-DATA-03) |
| Kesehatan data | Jumlah prospek · % terisi per field (telepon, `url`, `sameAs`, jam buka, rating, `geo`) · % terpetakan kelurahan (target ≥ 98%) · distribusi band · versi model aktif |
| Flag | Toggle AI / export / webhook + alasan wajib → banner FITUR DIMATIKAN di app customer |
| Suppression | Cari / tambah / hapus nilai (telepon, URL) + alasan (FR-PRIV-02). Efek: hilang dari tampilan, export, dan webhook |
| State khusus | Semua aksi tulis masuk audit. Persentase di bawah target ditandai ikon + teks "Di bawah target" |

---

## 5. Pola UI trust

Aturan umum: **status tidak pernah ditampilkan hanya dengan warna.** Selalu pakai ikon (Lucide) + label teks; claim type juga dibedakan dengan gaya border (NFR-06). Komponen di `packages/ui`, hanya memakai token semantik `02`.

| Pola | Tampilan (ikon · label) | Token / komponen | Aturan |
|---|---|---|---|
| Claim: Teramati | `eye` · "Teramati" | `color.claim.observed.*`, `border-style.claim.observed` (solid) | Nilai langsung dari dataset (BR-TRUST-01). Di tabel padat boleh ikon + tooltip |
| Claim: Turunan | `calculator` · "Turunan" | `color.claim.derived.*` (solid) | Hasil aturan, mis. kategori, tipe telepon, klasifikasi URL |
| Claim: Perkiraan | `sparkles` · "Perkiraan" | `color.claim.inferred.*` (**dashed**) | Selalu label penuh. Dipakai untuk opportunity, "Kemungkinan WhatsApp", populasi |
| Claim: Tidak diketahui | `circle-help` · "Tidak diketahui" | `color.claim.unknown.*` (**dotted**) | Selalu label penuh + tooltip alasan (BR-TRUST-03) |
| Freshness | `clock` Baru · `clock-alert` Menua · `history` Usang | `color.freshness.fresh / aging / stale` | Ambang BR-TRUST-02. Format "Baru · 45 hari lalu"; tooltip berisi tanggal pengambilan. Tanggal tidak ada → Tidak diketahui |
| Band | `flame` HOT · `sun` WARM · `snowflake` COLD · `circle-minus` Belum dinilai | `ScoreBadge`, `color.score.{hot,warm,cold,unscored}.solid / on-solid` | Band per BR-SCORE-02. Angka + kata band selalu tampil. "Belum dinilai" tanpa angka |
| Data terbatas | `triangle-alert` · "Data terbatas" | `color.status.warning.*` · peta: `color.geo.no-data` + arsir 45° | Prospek: BR-SCORE-02 (band dibatasi WARM, tooltip menyebut coverage). Area: Lampiran C (tanpa `market_score`) |
| Kontak ter-mask | `lock` · "+62 812-****-**90" | `MaskedValue`, `color.fg.muted`, `text.mono` | Belum di-unlock → tidak ada Salin / WhatsApp. Screen reader: "Nomor disembunyikan" |
| Evidence drawer | `file-search` · "Lihat evidence" | `EvidenceDrawer`, `color.bg.surface-raised`, `shadow.lg`, `size.drawer` | Isi: nilai, claim type, sumber (import + tanggal), nilai mentah JSON-LD (`text.mono`), aturan atau versi model (`priority-` / `geo-` / `brief-X.Y.Z`), freshness. Untuk opportunity: field + pembanding (mis. 61% restoran di Kota Semarang) |

---

## 6. Wireframes

Wireframe bersifat struktural. Contoh memakai data yang konsisten: ICP "Restoran Semarang · Website" (Offering `website_dev`). Skor dihitung dengan Lampiran A.

### WF-01 — Hasil prospek (SCR-03, ≥ 1280 px)

```
+------------------------------------------------------------------------------------------------------------+
| MesinProspek  Agensi Maju   ICP aktif: [Restoran Semarang · Website v]          Credit 184 · AI 42  [RA v] |
+------------+------------------------+----------------------------------------------------------------------+
| > Prospek  | FILTER         [Reset] | [Cari nama, kategori, alamat...                 ] [Simpan pencarian] |
|   Wilayah  | Kategori               | Restoran (x)  Kota Semarang (x)  HOT, WARM (x)                       |
|   List     |  [x] Restoran          |----------------------------------------------------------------------|
|   Export   | Wilayah                | 126 prospek · untuk ICP Restoran Semarang · Website   Urut: [Skor v] |
|   Pengatur.|  Jateng > Kota Semarang|----------------------------------------------------------------------|
|            |  [Semua kecamatan v]   | [ ] SKOR      BISNIS & LOKASI          OPPORTUNITY TERATAS KONTAK    |
|            | Band                   |----------------------------------------------------------------------|
|            |  [x] HOT  [x] WARM     | [ ] 85 HOT    Rumah Makan Sari Rasa    Website hanya       Telp      |
|            |  [ ] COLD              |               Restoran · Pleburan      linktr.ee           seluler   |
|            | Tipe opportunity       |               Baru · 45 hari lalu      (Perkiraan)         Web · IG  |
|            |  [ ] Website (semua)   |----------------------------------------------------------------------|
|            | Kontak & digital       | [ ] 78 WARM   Kedai Bakmi Pak Darto    Website tidak       Telp      |
|            |  [x] Ada telepon       |               Restoran · Peterongan    ditemukan           tetap     |
|            |  [ ] Ada website       |               Menua · 210 hari lalu    (Perkiraan)         IG · Web ?|
|            |  [ ] Ada sosial        |----------------------------------------------------------------------|
|            | Rating minimum         | [ ] 64 WARM   Soto Ayam Bu Tini        Website di          Telp      |
|            |  [Semua v]             |     [Data     Restoran · Jomblang      blogspot.com        seluler   |
|            |                        |     terbatas] Baru · 30 hari lalu      (Perkiraan)         Web       |
|            |                        |----------------------------------------------------------------------|
|            |                        | Menampilkan 50 dari 126                          [Muat 50 lagi]      |
+------------+------------------------+----------------------------------------------------------------------+
| BAR AKSI (sticky, muncul bila >= 1 dipilih): 3 dipilih · [Simpan ke list] [Tag] [Tugaskan]                 |
|   [Unlock · 3 credit] [Export] [Kirim ke webhook]  [Batal]                                                 |
+------------------------------------------------------------------------------------------------------------+
```

Kolom KONTAK hanya menunjukkan **ketersediaan** (nilainya ter-mask). "Web ?" dirender sebagai teks "Website tidak diketahui". Setiap indikator punya teks di tooltip dan `aria-label`.

### WF-02 — Profil prospek (SCR-04, ≥ 1280 px)

```
+------------------------------------------------------------------------------------------------------------+
| < Kembali ke hasil   1 dari 126  [<] [>]                                                                   |
| Rumah Makan Sari Rasa                                         +-----------+  untuk ICP Restoran Semarang   |
| Restoran (Turunan) · Jl. Pleburan Raya 8, Kel. Pleburan,      | 85   HOT  |  · Website                     |
| Kec. Semarang Selatan, Kota Semarang · Baru · 45 hari lalu    +-----------+  Coverage 100%                 |
| [Unlock kontak · 1 credit] [Simpan ke list] [Tugaskan] [Kirim ke webhook] [Google Maps]   (bar sticky)     |
+---------------------------------------------------------------------+--------------------------------------+
| KENAPA PROSPEK INI?                         priority-2.0.0          | AI BRIEF                             |
| Faktor             Nilai  Bobot  Kontribusi                         | Ringkasan, opportunity, dan hal yang |
| ICP Fit              100   30    ########## +30,0   Turunan         | belum diketahui, dengan rujukan.     |
| Opportunity           70   25    ######     +17,5   Perkiraan       | [Buat brief · 1 AI credit]           |
| Kontak                80   15    ####       +12,0   Turunan         |--------------------------------------|
| Wilayah               78   15    ####       +11,7   Perkiraan       | KONTEKS WILAYAH       [Lihat wilayah]|
| Popularitas           85   10    ###        +8,5    Turunan         | Kel. Pleburan · Kec. Semarang Selatan|
| Kelengkapan data     100    5    #          +5,0    Turunan         | Populasi   ±13.500 (i)               |
| Total 84,7 -> 85 HOT                                                |   perkiraan model Kontur 2023        |
|---------------------------------------------------------------------| Restoran   42 · 3,1 per 1.000 pddk   |
| OPPORTUNITY (1)                                 untuk Offering: Web | Kompetisi  Sedang                    |
| +-----------------------------------------------------------------+ | Skor pasar 78 / 100                  |
| | Website hanya linktr.ee        [Perkiraan]   Relevansi: primer  | |--------------------------------------|
| | Website mengarah ke linktr.ee, bukan domain sendiri. 61%        | | KERJA TIM                            |
| | restoran di Kota Semarang memiliki website sendiri.             | | List: Kuliner Smg Q4 · Tag: lunch    |
| | [Lihat evidence]          Relevan untuk Anda? [Relevan] [Tidak] | | Pemilik: Budi S.                     |
| +-----------------------------------------------------------------+ | Catatan (1): "Owner aktif di IG,     |
|---------------------------------------------------------------------|  DM dulu sebelum telepon." [+]       |
| KONTAK                                     Belum di-unlock          |                                      |
| Telepon  [lock] +62 812-****-**90   Seluler (Turunan)               |                                      |
|          Kemungkinan WhatsApp [Perkiraan]      [Unlock · 1 credit]  |                                      |
|---------------------------------------------------------------------|                                      |
| KEHADIRAN DIGITAL                                                   |                                      |
| Website    linktr.ee/sarirasa   Link aggregator (Turunan)    [i]    |                                      |
| Sosial     Instagram @sarirasa.smg (Teramati)                [i]    |                                      |
| Jam buka   Setiap hari 10.00–21.00 (Teramati)                [i]    |                                      |
| Rating     4,6 · 312 ulasan (Teramati)                       [i]    |                                      |
+---------------------------------------------------------------------+--------------------------------------+
```

Setelah unlock, baris telepon menjadi `+62 812-3456-7890 [Salin] [Telepon] [WhatsApp]` + "Di-unlock oleh Rina A. · 10 Okt 2026". `[i]` membuka evidence drawer.

### WF-03 — Wilayah (SCR-05, ≥ 1280 px)

```
+------------------------------------------------------------------------------------------------------------+
| Wilayah   [Jawa Tengah > Kota Semarang v]   Level: [Kecamatan | (Kelurahan)]   untuk ICP Restoran Semarang |
+-----------------------------------------------------+------------------------------------------------------+
| PERINGKAT (177 kelurahan: 169 diperingkat, 8 Data   | [+] [-] [Reset]                                      |
| terbatas)                                           |   +-----------+-----------+-----------+              |
| #  Kelurahan       Skor  Cocok  HOT/WARM  Kompet.   |   | Sekayu    | Pleburan  | Peterongan|              |
| 1  Pleburan          78    42    9 / 14   Sedang    |   |   [6]     | ==[7]==   |   [5]     |              |
| 2  Sekayu            74    35    6 / 11   Sedang    |   +-----------+-----------+-----------+              |
| 3  Peterongan        71    51    5 / 17   Tinggi    |   | Jomblang  | Lamper    | Bugangan  |              |
| …                                                   |   |  [////]   |   [4]     |   [3]     |              |
| -  Jomblang         Data terbatas · tanpa peringkat |   +-----------+-----------+-----------+              |
|-----------------------------------------------------|   (skematik; peta asli = poligon GeoAPI)             |
| PLEBURAN · Kec. Semarang Selatan            [x]     | LEGEND: Skor pasar (kuantil dalam Kota Semarang)     |
| Skor pasar 78 / 100 · peringkat 1 dari 169          | [1][2][3][4][5][6][7] rendah -> tinggi               |
| Populasi ±13.500 · perkiraan model Kontur 2023 (i)  | [////] Data terbatas    ==[ ]== area terpilih        |
| Luas 1,2 km2 · Bisnis 186 · Restoran 42             |------------------------------------------------------|
| Densitas restoran 3,1 per 1.000 pddk           (i)  | Batas & populasi: GeoAPI (Kontur 2023, perkiraan     |
| Kompetisi Sedang (P48)                         (i)  | model) · Peta dasar © kontributor OpenStreetMap ·    |
| [Lihat 23 prospek HOT/WARM]                         | geo-2.0.0                                            |
+-----------------------------------------------------+------------------------------------------------------+
```

Tabel dan peta tersinkron: memilih baris menyorot poligon, dan sebaliknya. Tabel adalah padanan peta yang aksesibel (FR-GEO-01).

---

## 7. Copy & format id-ID

Nada: lugas, sopan, tanpa menyalahkan user. Istilah produk mengikuti PRD: *credit*, *unlock*, *export*, *webhook*, *Offering*, *ICP*, *opportunity*. Sentence case, tanpa huruf kapital penuh kecuali band HOT/WARM/COLD.

| Jenis | Format | Catatan |
|---|---|---|
| Angka · desimal · persen | 12.400 · 4,6 · 61% | `Intl.NumberFormat('id-ID')`. Maks. 1 desimal di UI |
| Rupiah | Rp1.500.000 · ringkas Rp1,5 jt / Rp2 M | Tanpa spasi setelah "Rp". "M" = miliar |
| Perkiraan | ±13.500 | Untuk populasi dan hitungan perkiraan |
| Densitas | 3,1 per 1.000 penduduk | Lampiran C |
| Tanggal · tanggal + jam | 10 Okt 2026 · 10 Okt 2026, 14.30 WIB | `Asia/Jakarta` (NFR-07) |
| Waktu relatif | baru saja · 5 menit lalu · 3 hari lalu | Lebih dari 30 hari → tanggal absolut, kecuali umur data di badge freshness ("Menua · 210 hari lalu"). `<time datetime>` selalu ada |
| Telepon | +62 812-3456-7890 | Disimpan E.164 (BR-NORM-01) |
| Credit | 1 credit · 35 credit · 1 AI credit | Tanpa bentuk jamak |

**Pola error:** *[Apa yang terjadi]. [Kenapa, bila diketahui]. [Apa yang bisa dilakukan].* + tombol aksi + `Kode <8 karakter>` (dari correlation ID). Kode teknis tidak dipakai sebagai kalimat utama.

| Situasi | Copy |
|---|---|
| Dialog unlock | "Unlock kontak Rumah Makan Sari Rasa? Biaya 1 credit · saldo 184 → 183. Kontak terbuka permanen untuk workspace ini." |
| Dialog export | "Export 120 prospek (CSV). 85 sudah di-unlock (gratis), 35 akan di-unlock · 35 credit. Saldo 184 → 149. Credit hanya ditagih bila export berhasil." |
| Credit kurang | "Credit tidak cukup. Aksi ini butuh 35 credit, saldo Anda 20. Minta Admin workspace menambah credit." |
| Tidak diketahui | "Tidak diketahui — data sumber tidak memuat jam buka." |
| Data terbatas | "Data terbatas: sebagian faktor skor tidak diketahui (coverage 52%), jadi band maksimal WARM." |
| WhatsApp | "Kemungkinan WhatsApp karena nomor seluler. Belum dikonfirmasi." |
| Export gagal | "Export gagal diproses. Layanan penyimpanan tidak merespons. Tidak ada credit yang ditagih. [Coba lagi] · Kode 7F3A21C0" |
| URL webhook ditolak | "URL tidak dapat dipakai. Alamat ini mengarah ke jaringan privat atau internal. Gunakan URL HTTPS publik." |

---

## 8. Aksesibilitas (NFR-06, WCAG 2.2 AA dasar)

- Status tidak pernah hanya warna: ikon + label teks; claim type juga memakai gaya border (§5).
- Hanya token semantik `02`. Kontras dijaga oleh `build.mjs --check` di CI; warna hardcode ditolak lint.
- Semua aksi bisa dijalankan dengan keyboard dalam urutan fokus yang logis. Focus ring 2px `color.border.focus` pada `:focus-visible`. Shortcut `/` memfokuskan search.
- Dialog dan drawer (Radix): focus trap, Esc menutup, dan fokus kembali ke pemicu.
- Tabel: `<th scope>`, `aria-sort` pada kolom urut. Checkbox dan target ≥ 24px (`size.target-min`); 44px di layar sentuh.
- Peta selalu punya padanan tabel peringkat. Area bisa dipilih tanpa peta. Legend berlabel teks.
- Label screen reader: `ScoreBadge` "Skor prioritas 85, HOT"; `MaskedValue` "Nomor disembunyikan, unlock 1 credit"; `FactorBar` disajikan sebagai `<table>`.
- `lang="id"`. Progres skor/export/brief diumumkan lewat `role="status"`. `prefers-reduced-motion` dihormati (durasi 0 di `tokens.css`).

---

## 9. Event analytics minimal

Aturan: nama `<objek>_<kata kerja lampau>`; properti umum `workspace_id`, `user_id`, `role`, `icp_id`, `occurred_at`. Event berbayar dicatat **server-side** (sumber yang sama dengan ledger). Jangan pernah mengirim nomor telepon, teks query mentah, atau nama bisnis; pakai ID.

| Event | Sumber | Properti kunci | Tujuan PRD §4 |
|---|---|---|---|
| `signup_completed` | server | `method` (password/google) | Hasil pertama ≤ 5 menit (t0) |
| `icp_saved` | server | `offering_tags[]`, `criteria_count`, `required_count`, `is_first` | Hasil pertama ≤ 5 menit (funnel) |
| `first_result_viewed` | klien | `seconds_since_signup` | **Hasil pertama ≤ 5 menit** |
| `score_explanation_viewed` | klien | `band`, `coverage` | Paham alasan urutan |
| `evidence_opened` | klien | `field`, `claim_type` | Setiap nilai menunjukkan sumber |
| `opportunity_feedback_given` | server | `opp_code`, `relevant` | Paham alasan; kalibrasi (PRD §13) |
| `prospect_unlocked` | server | `source` (profile/bulk/export/webhook), `credits` | Unlock · credit tercatat benar |
| `export_completed` | server | `rows`, `credits_charged`, `status` | Export · credit tercatat benar |
| `webhook_sent` | server | `prospects`, `status`, `attempts` | Kirim ke webhook |
| `ai_brief_generated` | server | `cached`, `provider` (claude/fallback), `status` | Tindak lanjut (AI brief) |

---

## 10. F2/F3 (ringkas; detail di PRD §6)

- **F2:** upload daftar customer + pencocokan · signal perubahan setelah re-import + watch/alert · laporkan data salah · role Viewer/Manager · compare prospek & wilayah · natural-language search · satu integrasi CRM · billing self-serve (top-up di SCR-10) · batas plan lengkap · pengelompokan multi-cabang di profil · band revenue · bahasa Inggris.
- **F3:** tidak ada layar baru. Hanya penyesuaian performa (mis. hitungan hasil perkiraan "±" untuk dataset > 5 juta) dan status sistem saat deploy tanpa downtime.
