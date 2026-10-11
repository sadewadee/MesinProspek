# 03 — UX Spec (lean)

| Field | Value |
|---|---|
| Versi | 0.3 (lean). Posisi market intelligence (D-30): analisis lokasi, panel estimasi & pola keramaian, onboarding per tujuan |
| Tanggal | 2026-10-11 |
| Basis | `01-prd.md` v0.3 (**sumber kebenaran**; bila bertentangan, PRD yang berlaku) · `02-design-system/` tokens `0.2.0` |
| Cakupan | Hanya **F1**. F2/F3: PRD §6 |

> Konvensi: `SCR-nn` layar, `FLOW-nn` alur, `WF-nn` wireframe. Aturan bisnis dan rumus **tidak disalin**, cukup dirujuk ke ID PRD (DRY). Nama token persis seperti di `02`; endpoint dan tabel di `04`. Data = snapshot DB gmaps (PRD §7.1): **tidak ada email maupun data orang**. Dari contoh ulasan (`user_reviews`) hanya tanggal ulasan terbaru yang diimpor (laju ulasan metode B, D-38); teks, nama, dan foto tidak. Kontak = telepon, website, akun sosial, dan tautan Google Maps.

---

## 1. Prinsip UX

1. **Satu layar, satu pertanyaan.** Setiap SCR menuliskan pertanyaan yang dijawabnya. Informasi di luar itu dipindah ke drawer atau tab.
2. **Alasan sebelum angka.** Kesimpulan analisis lokasi tampil di atas buktinya. Skor selalu ditulis "untuk ICP X", dan "Kenapa bisnis ini?" bisa dibuka dengan satu klik dari skor mana pun (FR-INTL-04).
3. **Perkiraan selalu rentang + asumsi + tanggal.** Kunjungan dan omzet tampil rendah–tengah–tinggi (BR-EST-02), dengan sumber harga dan parameter terlihat tanpa klik, plus "Data per <tanggal>" (BR-TRUST-02). Tidak ada angka omzet tunggal.
4. **Trust terlihat, Unknown itu sah.** Setiap nilai membawa claim type dan evidence. Field kosong tampil "Tidak diketahui"; estimasi yang datanya kurang tampil "Belum dapat diperkirakan" + alasannya (BR-TRUST-01/03, BR-EST-01).
5. **Biaya selalu eksplisit.** Credit dipakai untuk unlock dan analisis lokasi baru; narasi AI memakai AI credit (Lampiran D). Tombol berbayar menyebut biayanya. Tidak ada debit tanpa dialog konfirmasi.
6. **Gagal terlihat dan bisa diulang.** Error ditampilkan per bagian, lengkap dengan "Coba lagi". Aksi yang gagal tidak ditagih.

---

## 2. Navigasi & daftar layar F1

### 2.1 Kerangka

| Elemen | Isi & perilaku |
|---|---|
| Sidebar (≥ `breakpoint.lg`, di bawahnya jadi drawer) | Bisnis · Analisis lokasi · Wilayah · List · Export · Pengaturan. Item di luar role tidak dirender |
| Header | Nama workspace · **ICP aktif** (dropdown, hanya bila workspace punya ICP; pilihan "Tanpa ICP" = semua bisnis tanpa skor, FR-ICP-02) · saldo "Credit 184 · AI 42" · menu user (Akun, Keluar) |
| Banner (maks. 1) | Prioritas: fitur dimatikan > skor sedang dihitung > info |
| URL | Query, filter, urutan, tampilan, dan ICP tersimpan di URL (`/businesses?q=&f=&sort=&view=&icp=`) sehingga bisa dibagikan dan disimpan (FR-SRCH-02). Evidence drawer memakai `?evidence=<field>` agar tombol Back menutupnya |
| Waktu | Waktu sistem (tanggal data, riwayat, log) dalam WIB (NFR-07). **Jam buka dan pola keramaian dalam waktu lokal tempat** dengan label WIB/WITA/WIT (BR-NORM-06), mis. "08.00–23.00 WITA" untuk Bali |
| Ops console | Layout terpisah di `/ops`, khusus akun ops dengan MFA (FR-AUTH-03). Desktop saja |
| Responsive | Desain utama `breakpoint.xl`. Profil dan List wajib nyaman di 360 px (NFR-08); layar lain cukup tetap terbaca |

### 2.2 Daftar layar (dialog dan evidence drawer adalah pola §5, bukan layar)

| SCR | Layar | Route (usulan) | Role | FR |
|---|---|---|---|---|
| SCR-01 | Masuk & daftar (flow Kratos) | `/login`, `/signup`, `/verify`, `/recovery`, `/account`, `/invite/[token]` | Semua | FR-AUTH-01, -02, -03 |
| SCR-02 | Onboarding: pilih tujuan → hasil pertama | `/onboarding` | Admin | FR-ICP-01, FR-ICP-02, FR-AUTH-02 |
| SCR-03 | Bisnis (search + filter + hasil) | `/businesses` | Admin, Member | FR-SRCH-01, -02, FR-EST-02/03 (kolom), FR-INTL-04, FR-ACT-01..04 |
| SCR-04 | Profil bisnis | `/businesses/[id]` | Admin, Member | FR-PROF-01, FR-EST-01..05, FR-INTL-01..05, FR-DATA-02, FR-ACT-01, -02, -04, -05, FR-PRIV-02 |
| SCR-05 | Wilayah (peringkat per kelompok estimasi + peta + drilldown) | `/areas` | Admin, Member | FR-GEO-01, FR-INTL-03, FR-DATA-04 |
| SCR-06 | List & pencarian tersimpan | `/lists`, `/lists/[id]` | Admin, Member | FR-ACT-01, FR-SRCH-02 |
| SCR-07 | Export (dialog unlock & export + riwayat) | `/exports` | Admin, Member | FR-ACT-03, FR-BILL-01 |
| SCR-08 | Pengaturan › ICP & anggota | `/settings/icp`, `/settings/members` | Admin (Member: lihat ICP) | FR-ICP-02, FR-AUTH-02, FR-BILL-01 (seat) |
| SCR-09 | Pengaturan › Webhook | `/settings/webhooks` | Admin | FR-ACT-04 |
| SCR-10 | Pengaturan › Plan, credit & pemakaian | `/settings/usage` | Admin | FR-BILL-01 |
| SCR-11 | Ops › Workspace, plan, credit & audit | `/ops/workspaces`, `/ops/plans`, `/ops/audit` | Ops | FR-OPS-01, FR-OPS-02, FR-BILL-01 |
| SCR-12 | Ops › Data & kontrol (import, kesehatan data, parameter estimasi, flag, pengaturan AI, suppression) | `/ops/imports`, `/ops/health`, `/ops/estimation`, `/ops/flags`, `/ops/ai`, `/ops/suppression` | Ops | FR-OPS-01, FR-OPS-02, FR-DATA-01, -04, FR-EST-06, FR-ACT-05, FR-PRIV-02 |
| SCR-13 | **Analisis lokasi** (daftar tersimpan, form, hasil) | `/locations`, `/locations/new`, `/locations/[id]` | Admin, Member | FR-LOC-01, FR-EST-04/05, FR-INTL-05, FR-ACT-05, FR-BILL-01 |

### 2.3 Role (FR-AUTH-02; server tetap menegakkan, akses lewat URL di luar role → DITOLAK)

| Aksi | Admin | Member | Ops |
|---|:-:|:-:|:-:|
| Search, profil, wilayah, analisis lokasi (jalankan, buka ulang, ubah nama, hapus), unlock, list/tag/catatan, export, kirim ke webhook, narasi AI | ✓ | ✓ | — |
| Pilih ICP aktif | ✓ | ✓ | — |
| Kelola ICP, anggota, endpoint webhook; lihat plan & pemakaian credit | ✓ | — | — |
| Console ops (grant credit, plan, parameter estimasi, pengaturan AI, flag, suppression, audit) | — | — | ✓ (MFA) |

---

## 3. Alur F1

### FLOW-01 — Onboarding: daftar → pilih tujuan → hasil pertama ≤ 5 menit (FR-ICP-01, PRD §4)

| # | Layar | Langkah | Catatan |
|---|---|---|---|
| 1 | SCR-01 | Daftar (email + password atau Google) → verifikasi email | Data tidak bisa diakses sebelum email terverifikasi |
| 2 | SCR-02 | Workspace dibuat otomatis → **pilih tujuan**: Analisis lokasi · Kompetitor & omzet · Cari prospek | Anggota undangan melewati onboarding dan langsung ke SCR-03 |
| 3a | SCR-13 | *Analisis lokasi*: form analisis baru (FLOW-06) | Saldo 0 → CREDIT KURANG + tautan "Coba Kompetitor & omzet (gratis)" |
| 3b | SCR-03 | *Kompetitor & omzet*: pilih kelompok (Makan / Kafe / Hotel) + kab/kota → hasil diurutkan perkiraan omzet | Gratis, tanpa ICP |
| 3c | SCR-02 → SCR-03 | *Cari prospek*: Offering → ICP (kriteria *wajib*/*preferensi*) → progres "Menghitung skor…" → hasil | State MENGHITUNG. Jumlah bisnis cocok diperbarui saat kriteria berubah |
| 4 | SCR-03/13 | Hasil pertama + coach mark: (1) rentang perkiraan & "Data per", (2) claim type & evidence, (3) biaya (unlock / analisis). Jalur prospek: skor & "Kenapa bisnis ini?" | Waktu ≤ 5 menit diukur manual (gerbang di `05` §7) |

### FLOW-02 — Prospecting harian (PER-03)

| # | Layar | Langkah | Catatan |
|---|---|---|---|
| 1 | SCR-03 | Cari kata kunci + filter (atau buka pencarian tersimpan) | Tampilan Prospek, urut skor untuk ICP aktif |
| 2 | SCR-04 | Buka profil → baca "Kenapa bisnis ini?" dan opportunity → beri feedback Relevan/Tidak relevan | FR-INTL-05 |
| 3 | SCR-04 | Buka evidence nilai yang diragukan | Drawer §5 |
| 4 | SCR-04 | **Unlock · 1 credit** → dialog → nomor terbuka → Salin / Telepon / WhatsApp | Unlock ulang tidak ditagih (FR-ACT-02) |
| 5 | SCR-04/03 | Simpan ke list, tag, catatan, tugaskan ke anggota | FR-ACT-01 |
| 6 | SCR-03/06 | Pilih banyak → **Export** (SCR-07) atau **Kirim ke webhook** | Export/kirim gratis; dialog menampilkan bisnis yang perlu di-unlock dulu (BR-CRED-01). Batas per aksi: SCR-03 |

### FLOW-03 — Wilayah → drilldown (PER-01, PER-02, PER-03)

| # | Layar | Langkah | Catatan |
|---|---|---|---|
| 1 | SCR-05 | Pilih kab/kota, level (Kecamatan/Kelurahan), dan **kelompok estimasi** | Default kelompok: Makan |
| 2 | SCR-05 | Baca peringkat (tabel) dan peta, lalu pilih area dari baris tabel atau poligon | Area "Data terbatas" diarsir dan tidak diberi peringkat (Lampiran C) |
| 3 | SCR-05 | Panel area: metrik Lampiran C + "Lihat N bisnis" + "Analisis lokasi di area ini" | Populasi berlabel "perkiraan model Kontur 2023" |
| 4 | SCR-03 / SCR-13 | Hasil bisnis dengan filter wilayah + kelompok terisi, atau form analisis lokasi dengan peta terpusat di area itu | Titik tetap dipilih user (tidak otomatis di centroid) |

### FLOW-04 — Narasi AI (FR-ACT-05)

| # | Layar | Langkah | Catatan |
|---|---|---|---|
| 1 | SCR-04 / SCR-13 | Panel Narasi AI: bila ada cache (≤ 7 hari) langsung tampil, gratis | "Dibuat 3 hari lalu · berlaku sampai 13 Okt" |
| 2 | SCR-04 / SCR-13 | Bila belum ada: **Buat narasi · 1 AI credit** → progres (≤ 20 s, NFR-02) | Nomor telepon tidak dikirim (BR-AI-01) |
| 3 | SCR-04 / SCR-13 | Narasi tampil per bagian: **Masalah · Data · Cara membaca · Kesimpulan · Batasan**. Tiap kalimat punya rujukan fakta `[n]` yang membuka evidence | Label "Dibuat AI dari angka di halaman ini". Narasi hanya menjelaskan hasil hitungan (Lampiran F) |
| 4 | SCR-04 / SCR-13 | Gagal validasi → dibuat ulang sekali secara otomatis. Bila masih gagal → error, tidak ditagih | Kill switch aktif → state FITUR DIMATIKAN |

### FLOW-05 — Ops import dari DB gmaps (FR-DATA-01, PER-04)

| # | Layar / alat | Langkah | Catatan |
|---|---|---|---|
| 1 | Job harian / CLI | Job `import.run` (tiap hari 03.00 WIB) membaca snapshot baru dari DB gmaps (read-only, inkremental per waktu scrape). Manual: `msp tool import --country ID`, atau `--reset` untuk membaca ulang seluruh riwayat setelah normalisasi berubah (09 RB-03). Bila area belum ada: `msp tool geo-sync --country ID` | Import pertama membaca seluruh riwayat. F1 hanya negara aktif `ID` (D-28); negara lain dilewati dan dihitung. Satu import per environment: bila import sedang berjalan, CLI berhenti dengan pesan "import sedang berjalan" dan job harian dilewati (04 §5.1) |
| 2 | SCR-12 Import | Status sumber (snapshot terakhir, jeda) + riwayat job: status, laporan snapshot baru / tempat baru / berubah / dilewati (negara lain) / ditolak dan peringatan per alasan | "Dilewati (negara lain)" bukan error. Skema sumber tidak cocok → job gagal dengan pesan yang menyebut kolomnya, tidak ada data yang ditulis |
| 3 | SCR-12 Import | Rantai hitung ulang berjalan: wilayah → estimasi → metrik wilayah → opportunity → skor → status "Diperbarui" | Customer melihat banner MENGHITUNG bila skor ICP-nya berubah. Estimasi baru tampil dengan "Data per" barunya |
| 4 | SCR-12 Kesehatan | Cek jumlah bisnis, % terisi per field, % bisnis ber-riwayat cukup, % terpetakan | Target ≥ 98% terpetakan (FR-DATA-04). Di bawah target → indikator peringatan |

### FLOW-06 — Analisis lokasi (FR-LOC-01, PER-01)

| # | Layar | Langkah | Catatan |
|---|---|---|---|
| 1 | SCR-13 | Masuk dari sidebar, onboarding, profil ("Analisis lokasi di sini"), atau wilayah ("Analisis lokasi di area ini") | Dari profil: titik = koordinat bisnis itu, kelompok = kelompoknya |
| 2 | SCR-13 | Isi input: titik, radius, kelompok target, harga per orang rencana (opsional), bisnis pembanding (opsional) | Lampiran F § Input. Pembanding = bisnis mana pun di data (mis. konsep yang ingin ditiru), tidak harus milik user. Tanpa pembanding analisis tetap lengkap |
| 3 | SCR-13 | **Jalankan analisis · 1 credit** → dialog: biaya, saldo sebelum → sesudah | Hasil "Tidak dapat dianalisis" tidak ditagih (Lampiran D) |
| 4 | SCR-13 | Hasil (≤ 3 s, NFR-01): kesimpulan → keyakinan → pola → jam inti & pembanding → posisi harga → sumber pengunjung siang → tetangga | Tersimpan otomatis. Feedback Relevan/Tidak relevan pada kesimpulan (FR-INTL-05) |
| 5 | SCR-13 | Opsional: Narasi AI (FLOW-04) | 1 AI credit |
| 6 | SCR-13 | Buka ulang dari daftar analisis tersimpan → gratis, hasil sama dengan saat dijalankan | Data lebih baru tersedia → info + "Jalankan ulang · 1 credit" (Lampiran D) |

### FLOW-07 — Kompetitor & omzet (PER-02)

| # | Layar | Langkah | Catatan |
|---|---|---|---|
| 1 | SCR-03 | Filter kelompok estimasi + wilayah (+ rentang perkiraan omzet, punya pola keramaian) → urut perkiraan omzet / kunjungan / laju ulasan | Tampilan Estimasi |
| 2 | SCR-04 | Buka profil → panel perkiraan (rentang, sumber harga, parameter, tabel sensitivitas), pola keramaian, riwayat ulasan | Penanda FR-EST-05 dibaca sebelum angka dipakai |
| 3 | SCR-06 | Simpan beberapa bisnis ke list → tampilan Estimasi = tabel perbandingan sederhana | FR-ACT-01 |
| 4 | SCR-07 | Export list dengan kolom estimasi | FR-ACT-03; aturan unlock tetap BR-CRED-01 |

---

## 4. Spesifikasi layar

### 4.0 State baku (berlaku di semua layar; tiap SCR hanya menulis yang khusus)

| Kode | Kapan | Perilaku UI |
|---|---|---|
| LOADING | Data pertama dimuat | Skeleton sesuai bentuk konten. Lebih dari 10 s → teks "Masih memuat…" |
| KOSONG | Objek belum pernah dibuat | 1 kalimat manfaat + 1 CTA (`EmptyState`) |
| NOL HASIL | Filter tanpa hasil | Daftar chip filter aktif + "Hapus filter" per chip + "Reset semua" |
| TIDAK DIKETAHUI | Field tidak ada di sumber | "Tidak diketahui" + ikon + tooltip alasan. Tidak pernah kosong, "0", atau "—" (BR-TRUST-03) |
| BELUM DAPAT DIPERKIRAKAN | Data minimum estimasi tidak ada (BR-EST-01) | "Belum dapat diperkirakan" + alasan **tertulis di samping**, bukan hanya tooltip (mis. "belum cukup data ulasan", "harga tidak diketahui", "parameter hotel belum diisi ops", "kategori belum didukung estimasi"). Bagian lain yang datanya ada tetap tampil |
| DATA TERBATAS | Skor: BR-SCORE-02 · Area: Lampiran C | Badge "Data terbatas" + tooltip (§5) |
| MENGHITUNG | ICP baru/diubah sedang dinilai | Banner progres. Bisnis yang belum dinilai tampil "Belum dinilai" |
| ERROR | Request gagal | Error per bagian (bukan seluruh halaman) + "Coba lagi" + kode referensi (§7) |
| CREDIT KURANG | Saldo < biaya (unlock, analisis lokasi, narasi AI) | Tombol tetap terlihat dengan biayanya. Klik → dialog: kebutuhan vs saldo + "Hubungi Admin" (Member) / "Hubungi tim MesinProspek" (Admin). Top-up manual oleh ops (FR-BILL-01) |
| FITUR DIMATIKAN | Flag AI / export / webhook off (FR-OPS-01) | Kontrol **nonaktif, tidak disembunyikan** + tooltip. Bila server membalas 503 → toast |
| DITOLAK | Di luar role (403) | Menjelaskan apa yang tidak bisa diakses dan siapa Admin-nya |

### SCR-01 — Masuk & daftar

| | |
|---|---|
| Pertanyaan | "Bagaimana saya masuk ke workspace saya?" |
| Isi | Flow Ory Kratos (login, registration, verification, recovery, settings) yang dirender Next.js dengan token kita: kartu tengah maks. 400 px di atas `color.bg.canvas`. Tombol "Lanjut dengan Google", email + password. Akun: ganti password, tautkan Google, MFA TOTP |
| Undangan | `/invite/[token]` menampilkan nama workspace + pengundang → daftar/masuk → bergabung sebagai Member |
| State khusus | Email belum terverifikasi → hanya layar verifikasi. Undangan kedaluwarsa. Pesan login gagal tidak membocorkan apakah email terdaftar. Ops tanpa TOTP → diarahkan ke pendaftaran TOTP (FR-AUTH-03) |

### SCR-02 — Onboarding (pilih tujuan → hasil pertama)

| | |
|---|---|
| Pertanyaan | "Apa yang ingin saya ketahui lebih dulu?" |
| Langkah 1 | **Tujuan** (3 kartu, pilih satu): **Analisis lokasi** — "Di mana dan jam berapa usaha baru sebaiknya buka?" · **Kompetitor & omzet** — "Berapa kira-kira kunjungan dan omzet usaha di sekitar saya?" · **Cari prospek** — "Bisnis mana yang paling layak saya hubungi?" |
| Kompetitor & omzet | Satu langkah: kelompok (Makan / Kafe / Hotel) + kab/kota → SCR-03 tampilan Estimasi, urut perkiraan omzet |
| Cari prospek | **Offering**: chip tag `website_dev` (Pembuatan website), `social_media`, `local_seo`, `digital_ads`, `reputation_mgmt`, `booking_system`, `other` (label id-ID + 1 baris deskripsi; boleh > 1, tag pertama = primer) → **ICP**: kategori (taksonomi 2 level), wilayah (provinsi → kab/kota → kecamatan), kondisi digital, minimum rating & ulasan, toggle *Wajib/Preferensi* per kriteria; panel kanan "± 418 bisnis cocok" → progres skor → SCR-03 tampilan Prospek |
| Analisis lokasi | Langsung ke `/locations/new` (SCR-13) |
| Aksi | Kembali (isian tetap tersimpan) · Ganti tujuan. Tujuan tidak disimpan; ketiga jalur tetap bisa dibuka dari sidebar |
| State khusus | Bisnis cocok = 0 → saran melonggarkan kriteria *wajib*. Workspace baru dengan saldo 0 → coach mark biaya menyebut "Credit belum tersedia — hubungi tim kami"; jalur Analisis lokasi menawarkan "Coba Kompetitor & omzet (gratis)" |

### SCR-03 — Bisnis

| | |
|---|---|
| Pertanyaan | "Bisnis mana yang paling layak saya perhatikan, dan kenapa?" |
| Isi | Search kata kunci (nama, kategori, alamat). Panel filter (FR-SRCH-01): kategori, **kelompok estimasi**, wilayah, **rentang perkiraan omzet/tahun** (dibandingkan dengan nilai tengah), **punya pola keramaian**, rating, ada telepon / website / sosial, dan bila ICP aktif: band, tipe opportunity. Header hasil: jumlah tepat + "untuk ICP X" (bila ada) + tampilan + urutan. "Muat 50 lagi" |
| Tampilan | **Estimasi** (default tanpa ICP): nama, kategori, kelurahan, "Data per" · perkiraan kunjungan/tahun (tengah + rentang) · perkiraan omzet/tahun (tengah + rentang + sumber harga) · laju ulasan · pola (label pola datar atau jam puncak) · penanda FR-EST-05 sebagai ikon + teks tooltip. **Prospek** (default dengan ICP): `ScoreBadge` · nama, kategori, kelurahan, freshness · opportunity teratas (1 baris alasan) · indikator kontak & digital (teks, bukan ikon saja) · rating. Hotel: kolom omzet berisi omzet kamar |
| Urut | Skor (default bila ICP aktif) · perkiraan kunjungan (default tanpa ICP) · perkiraan omzet (nilai tengah) · laju ulasan. Nilai "Belum dapat diperkirakan" selalu di akhir. Urut skor atau filter band → hanya kandidat ICP aktif; tanpa ICP keduanya tidak tersedia |
| Aksi | Buka profil · Simpan pencarian · Pilih banyak → bar aksi: Simpan ke list, Tag, Tugaskan, Unlock (n credit), Export, Kirim ke webhook. "Pilih semua hasil" maks. 10.000. **Batas per aksi:** Unlock dan Kirim ke webhook maks. 100 bisnis; Export maks. 10.000 baris per job. Di atas batas → tombol nonaktif + tooltip batasnya |
| State khusus | NOL HASIL · MENGHITUNG (urutan bisa berubah → "Muat ulang dari atas") · DATA TERBATAS pada baris · BELUM DAPAT DIPERKIRAKAN per sel · FITUR DIMATIKAN (Export/Webhook di bar aksi). Di 360 px: daftar kartu |

### SCR-04 — Profil bisnis

| | |
|---|---|
| Pertanyaan | "Seberapa ramai dan besar usaha ini, seberapa yakin angkanya, dan apa langkah saya berikutnya?" |
| Header | Nama · kategori (Turunan; kategori tambahan dari daftar kategori) · alamat · kelurahan › kecamatan · freshness · "Listing diklaim pemilik" bila ada (Teramati) · "Lihat di Google Maps" · **Analisis lokasi di sini** (nonaktif + tooltip bila koordinat tidak diketahui) · bila ICP aktif: `ScoreBadge` besar "untuk ICP X" + "Data terbatas" bila berlaku |
| Perkiraan kunjungan & omzet (`makan`, `kafe`; FR-EST-02, Lampiran E §E.2) | Badge Perkiraan · "Data per <snapshot terbaru>" · `est-X.Y.Z`. Tabel 3 kolom **Rendah · Tengah · Tinggi** (tengah ditebalkan): kunjungan/tahun, tamu per hari buka, omzet/tahun, dibulatkan per BR-EST-02; baris kunjungan dan omzet diberi chart "Rentang estimasi" (02 §3.3). Baris **Dasar** (selalu terlihat): laju ulasan + **sumber laju** (FR-EST-01, copy §7): riwayat snapshot (2 snapshot pembentuknya) atau N ulasan terbaru, selalu dengan jendela tanggalnya; penanda "jendela pendek" bila berlaku; baris cek silang bila kedua metode tersedia. Lalu **sumber harga** (harga tempat ini · median N bisnis sekelompok di kab/kota · asumsi harga ops · tidak diketahui), harga per orang, faktor belanja dan review rate (disebut **asumsi**), hari buka per minggu ("jam buka tidak diketahui → dianggap 7 hari"). **Catatan metode** (selalu tampil, §E.2): pesanan aplikasi pesan-antar umumnya tidak tercakup. Expand **Tabel sensitivitas**: review rate bawah/tengah/atas → tamu per hari, omzet/tahun. Penanda FR-EST-05 di bawah angka (copy §7). "Lihat evidence" membuka rumus langkah demi langkah |
| Varian hotel (FR-EST-03, Lampiran E §E.3) | Kunjungan menginap/tahun dan omzet kamar/tahun ("tidak termasuk F&B dan acara") sebagai rentang Rendah · Tengah · Tinggi (review rate hotel atas/tengah/bawah); room-night/tahun dan okupasi tahunan dari nilai tengah (04 §4.1). Dasar: laju ulasan, kelas akomodasi, dan asumsi ops (review rate hotel, lama menginap, jumlah kamar, ADR; Perkiraan). Okupasi > 100% → baris okupasi diganti badge **"Asumsi tidak konsisten"** + 1 kalimat penjelasan, nilai okupasi tidak ditampilkan; nilai lain tetap tampil. Asumsi kelas belum diisi ops → nilai yang bergantung padanya BELUM DAPAT DIPERKIRAKAN (copy §7) |
| Pola keramaian (FR-EST-04, Lampiran E §E.4) | Label tetap (bukan tooltip): **"Relatif terhadap jam tersibuk tempat ini, bukan jumlah orang."** · "Data per <tanggal>" · zona waktu lokal. Tab **Rata-rata 7 hari** · Sen … Min. Chart "Pola keramaian satu tempat" (02 §3.3 Chart F1); jam puncak diberi label teks. Ringkasan: jam puncak (juga per hari), jam inti, **"Ramai sepanjang hari"** bila pola datar, keramaian rata-rata ("rata-rata 76% dari jam tersibuknya"), bentuk per blok waktu (Pagi/Siang/Sore/Malam/Larut, %). **Lihat sebagai tabel** (NFR-06): jam · nilai · perkiraan tamu per jam (tengah, hanya bila kunjungan diketahui). Hari dengan semua nilai 0 → "Tutup". Tanpa `popular_times` → BELUM DAPAT DIPERKIRAKAN ("data pola keramaian tidak tersedia untuk tempat ini") |
| Riwayat ulasan (FR-EST-01) | Chart "Riwayat ulasan" (02 §3.3): jumlah ulasan per hari snapshot · laju setahun dan laju 90 hari terakhir. "Lihat data" → tabel tanggal, jumlah ulasan, rating. Laju ditulis dengan sumbernya, sama seperti di panel perkiraan. Riwayat < 30 hari → catatan "Riwayat snapshot sejak <tanggal>"; laju dari ulasan terbaru bila ada, laju 90 hari "Tidak diketahui". Kedua metode tidak tersedia → copy `review_data_insufficient` (§7) |
| Harga, jam buka & atribut | Harga per orang (Teramati + tanggal snapshot-nya, BR-NORM-05) · jam buka per hari + zona · atribut tempat (Populer untuk, Tipe pengunjung, opsi layanan; Teramati) |
| Kenapa bisnis ini? (hanya bila ICP aktif) | `FactorBar` 6 faktor Lampiran A: nilai, bobot, kontribusi, claim type. Faktor tidak diketahui: "Tidak dihitung" (BR-SCORE-01). Footer: coverage, `priority-X.Y.Z`. Expand ICP Fit → kriteria Cocok / Sebagian / Tidak / Tidak diketahui + Wajib/Preferensi |
| Opportunity (hanya bila ICP aktif) | Kartu per opportunity relevan (Lampiran B): judul, alasan + pembanding, badge Perkiraan, relevansi primer/sekunder, "Lihat evidence", **Relevan / Tidak relevan** (FR-INTL-05). Gap yang tidak relevan dengan Offering hanya tampil di Kehadiran digital |
| Kontak | Telepon ter-mask + "Kemungkinan WhatsApp" bila seluler (BR-NORM-01). **Unlock · 1 credit** → nilai penuh + Salin / Telepon / WhatsApp (`wa.me`) + "Di-unlock oleh Rina · 10 Okt". Nilai yang di-suppress: "Kontak tidak tersedia", tombol Unlock tidak muncul (FR-PRIV-02) |
| Kehadiran digital | Website + klasifikasi BR-NORM-02 · akun sosial · rating & jumlah ulasan. Website, sosial, dan Maps **tidak** ter-mask |
| Konteks wilayah | Kelurahan: populasi (perkiraan model Kontur 2023), jumlah & densitas bisnis sekelompok per 1.000 (kelompok `lainnya`: semua bisnis, Lampiran C), kompetisi, permintaan terbukti, `market_score` → "Lihat wilayah" (SCR-05) |
| Narasi AI | FLOW-04. Teks `text.body-lg`, rujukan `[n]` |
| Aksi & kerja tim | Unlock · Simpan ke list · Tugaskan · Kirim ke webhook · ‹ › bisnis berikutnya. Panel list, tag, catatan, pemilik (FR-ACT-01) |
| State khusus | Setiap nilai punya claim type + evidence (§5). Kelompok `lainnya` → panel perkiraan BELUM DAPAT DIPERKIRAKAN ("kategori belum didukung estimasi"), pola keramaian tetap tampil bila ada. Tanpa koordinat / di luar semua poligon → wilayah dan Konteks wilayah "Tidak diketahui" (tanpa tebakan desa terdekat). AI dimatikan → FITUR DIMATIKAN. Di 360 px: satu kolom akordeon (urutan: perkiraan, pola, riwayat, harga & jam, lainnya) + bar bawah sticky [Unlock · 1 credit] [Aksi lain]; grafik pola default ke tampilan tabel |

### SCR-05 — Wilayah

| | |
|---|---|
| Pertanyaan | "Kecamatan/kelurahan mana yang paling potensial untuk kelompok usaha ini?" |
| Isi | Pemilih kab/kota + level (Kecamatan / Kelurahan) + **kelompok estimasi** (Makan / Kafe / Hotel). **Tabel peringkat** (urut `market_score`; Lampiran C): peringkat, area, `market_score`, populasi, jumlah & densitas bisnis sekelompok, kompetisi, permintaan terbukti (kunjungan rata-rata per bisnis, tengah), rentang harga (median titik tengah), dan bila ICP aktif: bisnis cocok + HOT/WARM. **Peta** choropleth `market_score` dari vector tiles GeoAPI (`color.geo.sequential.1..7`, kuantil dalam kab/kota). Panel area terpilih: semua metrik Lampiran C |
| Aksi | Pilih area (baris atau poligon, tersinkron) · "Lihat N bisnis" → SCR-03 dengan filter wilayah + kelompok · "Analisis lokasi di area ini" → SCR-13 · zoom/reset |
| Wajib tampil | Legend + "Populasi: perkiraan model Kontur 2023" + atribusi "Batas & populasi: GeoAPI · Peta dasar © kontributor OpenStreetMap" (PRD §7.2, FR-INTL-03) + `geo-X.Y.Z` |
| State khusus | Area Data terbatas: `color.geo.no-data` + arsir, metrik tetap tampil, tanpa peringkat. Permintaan terbukti / rentang harga dengan < 3 bisnis → "Tidak diketahui" (Lampiran C). Tiles gagal → peta error + tabel tetap jalan. Di < `breakpoint.lg`: tab Tabel / Peta (default Tabel) |

### SCR-06 — List & pencarian tersimpan

| | |
|---|---|
| Pertanyaan | "Bisnis apa yang sedang kami pantau atau kerjakan?" |
| Isi | Tab **List**: nama, jumlah, pemilik, diubah. Detail list = tabel baris SCR-03 dengan tampilan **Estimasi** (tabel perbandingan sederhana, FR-ACT-01) atau **Prospek**, + kolom pemilik, tag, catatan. Tab **Pencarian tersimpan**: nama, ringkasan filter, ICP, jumlah hasil saat ini (dinamis, FR-SRCH-02) |
| Aksi | Buat/ganti nama/hapus list · hapus dari list · tugaskan · tag · Export · Kirim ke webhook · Jalankan pencarian |
| State khusus | KOSONG: "Simpan bisnis ke list untuk dibandingkan atau dikerjakan bersama tim." CTA: Cari bisnis. Di 360 px: daftar kartu |

### SCR-07 — Export

| | |
|---|---|
| Pertanyaan | "Berapa bisnis yang perlu di-unlock dulu, dan di mana file saya?" |
| Dialog | Export gratis. Bila ada bisnis yang belum di-unlock, dua pilihan: **Export tanpa unlock** (telepon ter-mask, gratis) atau **Unlock & export** dengan preview yang sama dengan bulk Unlock: jumlah bisnis · sudah di-unlock (gratis) · akan di-unlock (n credit) · saldo sebelum → sesudah · "Unlock dijalankan sekarang; export yang gagal bisa diulang gratis" (FR-ACT-03, BR-CRED-01). Format CSV/XLSX. Bisnis yang kontaknya di-suppress tidak di-unlock/ditagih dan disebutkan jumlahnya. File memuat kolom estimasi (rentang, sumber harga, "Data per", versi; daftar kolom di 04) |
| Riwayat | Tabel job: dibuat, oleh, jumlah baris, status (Antre / Diproses / Selesai / Gagal), Unduh / Coba lagi. Status job diperbarui otomatis |
| State khusus | CREDIT KURANG · FITUR DIMATIKAN · Gagal → ulangi gratis, unlock yang sudah jalan tetap berlaku · lebih dari 10.000 baris → minta persempit filter |

### SCR-08 — Pengaturan › ICP & anggota

| | |
|---|---|
| Pertanyaan | "Siapa tim saya dan target apa yang kami kejar?" |
| ICP | Daftar ICP (nama, Offering, jumlah bisnis cocok). Buat/ubah memakai form jalur Cari prospek SCR-02. Simpan → MENGHITUNG. Member: hanya lihat & pilih ICP aktif. KOSONG: "ICP dipakai untuk skor prospek. Opsional bila Anda tidak mencari prospek." |
| Anggota | "Seat 4 dari 5 terpakai · plan Growth" (seat terpakai = anggota + undangan tertunda) · tabel anggota (nama, email, role, status undangan) · Undang (email + role) · ubah role · keluarkan · kirim ulang / batalkan undangan |
| State khusus | Admin terakhir tidak bisa diturunkan atau dikeluarkan. Seat plan penuh → tombol Undang nonaktif + tooltip; bila server tetap menolak (`SEAT_LIMIT`) → copy "Seat penuh" (§7). DITOLAK untuk Member di tab Anggota |

### SCR-09 — Pengaturan › Webhook

| | |
|---|---|
| Pertanyaan | "Apakah bisnis terkirim ke sistem saya?" |
| Isi | Daftar endpoint (nama, URL, status aktif). Tambah: URL HTTPS → validasi. Secret HMAC tampil sekali saat dibuat + "Salin". Log pengiriman: waktu, jumlah bisnis, kode respons, percobaan, retry berikutnya (FR-ACT-04) |
| Aksi | Tambah · Kirim uji · nonaktifkan · hapus · rotasi secret · kirim ulang dari log |
| Kirim bisnis | Dari SCR-03/04/06 (maks. 100 bisnis): pilih endpoint → dialog seperti SCR-07. Kirim gratis; bisnis yang belum di-unlock di-unlock dulu (n credit) |
| State khusus | URL ke IP privat/internal ditolak (copy §7). Uji gagal → kode respons + saran. FITUR DIMATIKAN |

### SCR-10 — Pengaturan › Plan, credit & pemakaian

| | |
|---|---|
| Pertanyaan | "Credit kami terpakai untuk apa?" |
| Isi | **Plan** (hanya baca, diatur ops di SCR-11): nama, harga/bulan, seat terpakai/maks., credit & AI credit per bulan, tanggal grant bulanan berikutnya · saldo credit & AI credit (= jumlah ledger, FR-BILL-01) · tabel ledger: tanggal, jenis (Grant bulanan / Grant manual / Debit / Refund), jumlah, oleh ("Sistem" untuk grant bulanan), objek (unlock bisnis / analisis lokasi / narasi AI), filter per bulan |
| State khusus | KOSONG: "Belum ada pemakaian." Workspace tanpa plan: "Belum ada plan — hubungi tim MesinProspek." Info: "Perubahan plan dan penambahan credit dilakukan oleh tim MesinProspek" + kontak |

### SCR-11 — Ops › Workspace, plan, credit & audit

| | |
|---|---|
| Pertanyaan | "Customer mana yang perlu dilayani, dan siapa melakukan apa?" |
| Workspace | Tabel: nama, plan, anggota/seat, saldo credit/AI, aktivitas terakhir. Detail: **Plan** (pilih dari plan aktif; kuota plan baru dipakai pada grant awal siklus berikutnya, kebutuhan segera → Grant credit) + ledger (grant bulanan tercatat "Sistem") + form **Grant credit** (jenis, jumlah, alasan wajib) + refund manual per debit (BR-CRED-01) |
| Plan | Tabel: nama, harga tampilan (Rp), seat, credit/bulan, AI credit/bulan, aktif, jumlah workspace pemakai. Buat/ubah lewat form (semua field wajib, angka ≥ 0). Tidak ada hapus: plan dinonaktifkan (tidak bisa dipilih lagi; workspace yang sudah memakainya tetap). Simpan perubahan → konfirmasi "Dipakai N workspace, berlaku mulai grant berikutnya". Harga hanya tampilan; pembayaran tetap manual (FR-BILL-01). Di atas tabel: **Grant awal workspace baru** (credit dan AI credit, angka ≥ 0; default 1.000 / 0, D-36), berlaku untuk workspace yang dibuat setelah disimpan, tercatat di audit |
| Audit | Tabel append-only (FR-OPS-02): waktu, aktor, workspace, aksi, objek, nilai lama → baru untuk pengaturan. Filter jenis/aktor/workspace/tanggal. Tanpa edit maupun hapus |
| State khusus | Grant ganda dicegah idempotency (tombol nonaktif saat proses). Ganti plan workspace dan ubah plan masuk audit. Hanya akun ops dengan MFA |

### SCR-12 — Ops › Data & kontrol

| | |
|---|---|
| Pertanyaan | "Apakah data sehat, estimasi masuk akal, dan sistem terkendali?" |
| Import | **Status sumber DB gmaps**: snapshot terakhir yang terbaca (waktu scrape, Indonesia), jeda sejak itu, import sukses terakhir; ambang peringatan mengikuti alert 09 AL-10/AL-11. **Riwayat job** `import`: pemicu (harian / manual), mulai–selesai, rentang waktu scrape yang dibaca, status, snapshot baru, tempat baru, berubah, dilewati (negara lain), ditolak per alasan, lalu status rantai hitung ulang. Job gagal → pesan utuh (mis. kolom sumber yang hilang) + "Tidak ada data yang ditulis". Laporan tampil di layar saja (FR-DATA-01) |
| Kesehatan data | Jumlah bisnis (total dan per kelompok estimasi) · % terisi per field: telepon, website, sosial, jam buka, rating, koordinat, **harga**, **`popular_times`**, distribusi bintang · **% bisnis dengan laju ulasan per metode (A: riwayat snapshot ≥ 30 hari; B: ulasan terbaru)** · % terpetakan kelurahan (target ≥ 98%) · distribusi band · versi model aktif (`est-`, `loc-`, `geo-`, `priority-`) |
| Parameter estimasi (PRD Lampiran E §E.5; kunci `est.*` di 04) | Judul tab dan label menyebut nilai ini **asumsi**, bukan benchmark. Form per grup, setiap field menampilkan default PRD sebagai petunjuk: **Review rate** `makan`, `kafe`, `hotel` (bawah / tengah / atas) · **faktor belanja** · **asumsi harga per orang** per kelompok (Rp min–max) · **hotel per kelas akomodasi** (bintang 1–5, hotel umum, resor, vila, penginapan): lama menginap, jumlah kamar, ADR · **laju ulasan metode B**: minimum ulasan terbaru (N) dan batas jendela pendek (hari) · **ambang**: penanda bintang 5, laju berubah, jam inti, pola datar. Field kosong = estimasi yang bergantung padanya "Belum dapat diperkirakan". Validasi: persen 0 < x ≤ 100, bawah ≤ tengah ≤ atas, angka ≥ 0, N bilangan bulat ≥ 2, jendela pendek bilangan bulat ≥ 1 hari. Simpan (alasan wajib) → konfirmasi "Estimasi semua bisnis akan dihitung ulang" → status hitung ulang (FR-EST-06); tercatat di audit (nilai lama → baru) |
| Flag | Toggle AI / export / webhook + alasan wajib → banner FITUR DIMATIKAN di app customer |
| Pengaturan AI | Form (FR-ACT-05, D-29): provider utama (Claude / OpenAI-compatible) + model utama · fallback aktif/nonaktif + base URL (HTTPS) + model · status API key per provider "Terpasang" / "Belum diset" (hanya status; key adalah secret server, tidak bisa dilihat atau diubah di UI). **Uji koneksi**: prompt uji tanpa data bisnis ke provider utama & fallback memakai nilai di form → OK/gagal + latensi + pesan error, tanpa AI credit. Simpan → berlaku untuk narasi baru ≤ 60 s tanpa redeploy; tercatat di audit (nilai lama → baru). Kill switch tetap di Flag |
| Suppression | Cari / tambah / hapus nilai (telepon, URL) + alasan (FR-PRIV-02). Efek: hilang dari tampilan, export, dan webhook |
| State khusus | Semua aksi tulis masuk audit. Import sedang berjalan → status "Berjalan" + progres batch. Persentase di bawah target ditandai ikon + teks "Di bawah target". Pengaturan AI: API key provider terpilih belum diset → Simpan nonaktif + "API key belum diset di server". Parameter: hitung ulang sedang berjalan → Simpan nonaktif sampai selesai |

### SCR-13 — Analisis lokasi

| | |
|---|---|
| Pertanyaan | "Apakah titik ini cocok untuk usaha saya, jam berapa sebaiknya buka, dan di posisi harga mana?" |
| Daftar (`/locations`) | Analisis tersimpan: nama, titik, kelompok, radius, keyakinan, "Data per", dibuat oleh, tanggal. Ubah nama · hapus · buka (gratis). CTA **Analisis baru** |
| Form (`/locations/new`) | Peta (MapLibre, basemap OSM + tiles GeoAPI) + panel input (Lampiran F § Input): **Titik** = klik peta, atau cari alamat = pilih bisnis atau wilayah dari pencarian (F1 tanpa geocoder eksternal, 04 §7.3), lalu titik bisa digeser dengan klik peta · **Radius** slider + input angka 100–2.000 m (default 500, lingkaran tampil di peta) · **Kelompok target** Makan / Kafe / Hotel · **Harga per orang rencana** (opsional): Rp min – Rp max (max boleh kosong = rentang terbuka) · **Bisnis pembanding (opsional)**: cari bisnis mana pun di data, mis. konsep yang ingin ditiru; tidak harus milik Anda. Nama otomatis "<jalan/kelurahan> · <kelompok> · <radius>" |
| Biaya | **Jalankan analisis · 1 credit** → dialog: biaya, saldo sebelum → sesudah, "Hasil tersimpan dan bisa dibuka ulang gratis. Tidak ditagih bila hasilnya Tidak dapat dianalisis" (Lampiran D). Mengubah input apa pun = analisis baru |
| Hasil (`/locations/[id]`) — kepala | Ringkasan input · "Data per <snapshot tetangga terbaru>" · `loc-X.Y.Z` · dibuat oleh, tanggal. Aksi: Ubah nama · **Jalankan ulang · 1 credit** (input sama, data terbaru) · Ubah input (analisis baru) · Hapus. Peta: titik, lingkaran radius, penanda tetangga per kelompok (bentuk + label, tabel Tetangga sebagai padanan) |
| 1. Kesimpulan | Deterministik (Lampiran F § Kesimpulan): saran jam buka, posisi harga, jam inti pembanding yang tidak didukung tetangga (bila ada pembanding), keyakinan. Badge Turunan. **Relevan / Tidak relevan** (FR-INTL-05) |
| 2. Keyakinan | `ConfidenceMeter` + label Tinggi/Sedang/Rendah + "N tetangga sejenis dengan data cukup". Rendah → kotak **saran cek lapangan** (teks Lampiran F, dengan nama tetangga dan jam puncak/sepi terisi). 0 → state TIDAK DAPAT DIANALISIS |
| 3. Pola keramaian tetangga vs pembanding | Chart "Perbandingan pola" (02 §3.3): **bentuk** per jam (% dari total harian, BR-EST-03) untuk rata-rata tetangga sejenis (data cukup), pembanding (bila dipilih), dan F&B lain (konteks, mis. kafe yang ramai larut malam). Label "Relatif terhadap jam tersibuk, bukan jumlah orang" + zona waktu lokal. **Lihat sebagai tabel** (NFR-06): jam × seri |
| 4. Jam inti & saran jam buka | Jam inti tetangga sejenis → saran jam buka + aturan singkatnya ("1 jam sebelum jam inti pertama sampai 1 jam setelah awal jam inti terakhir") |
| 5. Pembanding berdampingan | Tabel 2 kolom Tetangga sejenis · Pembanding: jam inti, bentuk per blok waktu (%). Jam inti pembanding yang tidak ada di tetangga ditandai `triangle-alert` + "tidak didukung tetangga". Tanpa pembanding: kolom Tetangga sejenis saja + tautan "Tambah pembanding di analisis berikutnya" |
| 6. Posisi harga | Titik tengah harga rencana vs median titik tengah tetangga sejenis → **Lebih murah / Setara / Lebih mahal** + rasio + ambang (Lampiran F). Harga rencana kosong → "Isi harga rencana di analisis berikutnya untuk melihat posisi harga". Tidak ada tetangga sejenis berharga → "Tidak diketahui" |
| 7. Sumber pengunjung siang | Jumlah bisnis non-F&B dalam radius per jenis (kesehatan, akomodasi, kantor/pendidikan, ritel; 0 tetap ditulis 0 karena terhitung) + 3 nama terdekat dengan jarak |
| 8. Tetangga | Tab **Sejenis · F&B lain · Non-F&B** (dengan jumlah). Kolom: nama (→ profil), kategori, jarak, ulasan, rating, harga, pola keramaian (Ada/Tidak ada), data cukup (Ya / Tidak + alasan), jam inti, penanda FR-EST-05, **Data per** |
| 9. Narasi AI | FLOW-04: Masalah · Data · Cara membaca · Kesimpulan · Batasan |
| State khusus | TIDAK DAPAT DIANALISIS (0 tetangga sejenis dengan data cukup): **tidak ditagih**, tetap tersimpan; bagian 3–5 diganti pesan + saran memperbesar radius; F&B lain, posisi harga (bila ada harga tetangga), sumber pengunjung siang, dan tabel Tetangga tetap tampil. Dibuka ulang setelah ada import baru → info "Ada data lebih baru sejak analisis ini" + Jalankan ulang · 1 credit. CREDIT KURANG. Gagal hitung → error, tidak ditagih, tidak tersimpan. KOSONG (daftar): "Analisis satu titik untuk melihat keramaian tetangga, saran jam buka, dan posisi harga." CTA: Analisis baru · 1 credit |

---

## 5. Pola UI trust

Aturan umum: **status tidak pernah ditampilkan hanya dengan warna.** Selalu pakai ikon (Lucide) + label teks; claim type juga dibedakan dengan gaya border (NFR-06). Trust di F1 = claim type + freshness + tanggal data. Confidence hanya dipakai untuk **keyakinan analisis lokasi** (Lampiran F). Komponen di `packages/ui`, hanya memakai token semantik `02`.

| Pola | Tampilan (ikon · label) | Token / komponen | Aturan |
|---|---|---|---|
| Claim: Teramati | `eye` · "Teramati" | `color.claim.observed.*`, `border-style.claim.observed` (solid) | Nilai langsung dari snapshot (BR-TRUST-01), mis. harga, jam buka, atribut. Di tabel padat boleh ikon + tooltip |
| Claim: Turunan | `calculator` · "Turunan" | `color.claim.derived.*` (solid) | Hasil aturan, mis. kategori, klasifikasi URL, laju ulasan, jam inti, kesimpulan analisis lokasi |
| Claim: Perkiraan | `sparkles` · "Perkiraan" | `color.claim.inferred.*` (**dashed**) | Selalu label penuh. Dipakai untuk kunjungan, omzet, okupasi, tamu per jam, asumsi ops, opportunity, "Kemungkinan WhatsApp", populasi |
| Claim: Tidak diketahui | `circle-help` · "Tidak diketahui" | `color.claim.unknown.*` (**dotted**) | Selalu label penuh + tooltip alasan (BR-TRUST-03) |
| Belum dapat diperkirakan | `circle-help` · "Belum dapat diperkirakan" + alasan | `color.claim.unknown.*` (dotted) | Alasan tertulis di samping label (BR-EST-01). Tidak pernah "0" atau kosong |
| Rentang perkiraan | Rendah · **Tengah** · Tinggi | `text.numeric`, badge Perkiraan, chart "Rentang estimasi" (02 §3.3) | Tidak pernah satu angka saja (BR-EST-02). Selalu didampingi "Data per <tanggal>", sumber harga, asumsi yang dipakai, dan catatan metode. Tidak pernah disebut omzet pasti |
| Tanggal data | `calendar` · "Data per 24 Sep 2026" | `text.caption`, `color.fg.muted` | Wajib di setiap estimasi, pola keramaian, tetangga, dan analisis (BR-TRUST-02) |
| Penanda kualitas estimasi | `triangle-alert` · kalimat FR-EST-05 utuh | `color.status.warning.*` | Netral, tidak menuduh (copy §7). Tampil di profil dan analisis lokasi. Di tabel padat: ikon + tooltip berisi kalimat utuh |
| Asumsi tidak konsisten | `triangle-alert` · "Asumsi tidak konsisten" | `color.status.warning.*` | Hotel, okupasi > 100% (FR-EST-03). Nilai okupasi tidak ditampilkan |
| Label pola keramaian | `info` · "Relatif terhadap jam tersibuk, bukan jumlah orang" | `text.caption` | Selalu terlihat di setiap grafik/tabel pola (FR-EST-04) |
| Keyakinan analisis | Meter 3 bar · "Tinggi" / "Sedang" / "Rendah"; `ban` · "Tidak dapat dianalisis" | `ConfidenceMeter`, `color.confidence.*` | Hanya untuk Lampiran F. Jumlah bar + label, bukan warna saja |
| Freshness | `clock` Baru · `clock-alert` Menua · `history` Usang | `color.freshness.fresh / aging / stale` | Ambang BR-TRUST-02. Format "Baru · 45 hari lalu"; tooltip berisi tanggal snapshot. Tanggal tidak ada → Tidak diketahui |
| Band | `flame` HOT · `sun` WARM · `snowflake` COLD · `circle-minus` Belum dinilai | `ScoreBadge`, `color.score.{hot,warm,cold,unscored}.solid / on-solid` | Band per BR-SCORE-02. Angka + kata band selalu tampil. "Belum dinilai" tanpa angka |
| Data terbatas | `triangle-alert` · "Data terbatas" | `color.status.warning.*` · peta: `color.geo.no-data` + arsir 45° | Bisnis: BR-SCORE-02 (band dibatasi WARM, tooltip menyebut coverage). Area: Lampiran C (tanpa `market_score`) |
| Kontak ter-mask | `lock` · "+62 812-****-**90" | `MaskedValue`, `color.fg.muted`, `text.mono` | Belum di-unlock → tidak ada Salin / WhatsApp. Screen reader: "Nomor disembunyikan" |
| Chart | Pola keramaian · perbandingan pola · rentang estimasi · riwayat ulasan | 02 §3.3 "Chart F1" (bentuk, seri, token) | Tidak didefinisikan ulang di sini. Padanan tabel wajib (NFR-06) |
| Evidence drawer | `file-search` · "Lihat evidence" | `EvidenceDrawer`, `color.bg.surface-raised`, `shadow.lg`, `size.drawer` | Isi: nilai, claim type, sumber (`cid` + waktu snapshot asal nilai), nilai mentah sumber (`text.mono`), aturan atau versi model (`est-` / `loc-` / `priority-` / `geo-` / `brief-X.Y.Z`), freshness. Untuk estimasi: rumus langkah demi langkah dengan input (snapshot & jumlah ulasan, sumber harga, parameter). Untuk opportunity: field + pembanding |

---

## 6. Wireframes

Wireframe bersifat struktural. WF-02 memakai data golden test Signa Cafe dan WF-04 memakai Warung Kita (05 §4.1). Nama lain fiktif. Alamat dan nilai yang tidak ada di fixture ditulis sebagai `<…>`.

### WF-01 — Bisnis, tampilan Estimasi tanpa ICP (SCR-03, ≥ 1280 px)

```
+------------------------------------------------------------------------------------------------------------+
| MesinProspek  Kopi Nusantara   ICP aktif: [Tanpa ICP v]                         Credit 12 · AI 20   [RA v] |
+-------------+------------------------+---------------------------------------------------------------------+
| > Bisnis    | FILTER         [Reset] | [Cari nama, kategori, alamat...                ] [Simpan pencarian] |
|   Analisis  | Kelompok estimasi      | Makan (x)  Provinsi Bali (x)  Punya pola keramaian (x)              |
|   lokasi    |  (o) Makan  ( ) Kafe   |---------------------------------------------------------------------|
|   Wilayah   |  ( ) Hotel             | 214 bisnis     Tampilan: [Estimasi | Prospek]   Urut: [Omzet v]     |
|   List      | Wilayah                |---------------------------------------------------------------------|
|   Export    |  Bali > [Semua kab v]  | [ ] BISNIS & LOKASI      KUNJUNGAN/THN   OMZET/THN       POLA       |
|   Pengatur. | Perkiraan omzet/thn    |---------------------------------------------------------------------|
|             |  [Rp min] – [Rp max]   | [ ] Signa Cafe           18.400          Rp2,4 M         Ramai      |
|             | Pola keramaian         |     Restoran · <kel.>    13.800–27.600   1,8–3,6 M       sepanjang  |
|             |  [x] Punya pola        |     Data per 24 Sep 2026 laju 276/thn    harga sendiri   hari       |
|             | Kategori, rating,      |                                          [!] ulasan bintang 5       |
|             | kontak & digital   [+] |---------------------------------------------------------------------|
|             |                        | [ ] Rumah Makan Contoh   6.700           Rp270 jt        Puncak     |
|             |                        |     Restoran · <kel.>    5.000–10.000    200–400 jt      19.00 WITA |
|             |                        |     Data per 20 Sep 2026 laju 100/thn    median kab/kota            |
|             |                        |---------------------------------------------------------------------|
|             |                        | [ ] Kedai Contoh Dua     4.000           Rp190 jt        Puncak     |
|             |                        |     Restoran · <kel.>    3.000–6.000     140–290 jt      12.00 WITA |
|             |                        |     Data per 18 Sep 2026 laju 60/thn     asumsi harga               |
|             |                        |---------------------------------------------------------------------|
|             |                        | Menampilkan 50 dari 214                           [Muat 50 lagi]    |
+-------------+------------------------+---------------------------------------------------------------------+
| BAR AKSI (sticky, muncul bila >= 1 dipilih): 2 dipilih · [Simpan ke list] [Tag] [Tugaskan]                 |
|   [Unlock · 2 credit] [Export] [Kirim ke webhook]  [Batal]                                                 |
+------------------------------------------------------------------------------------------------------------+
```

Urut omzet memakai nilai tengah; bisnis yang omzetnya "Belum dapat diperkirakan" tampil di akhir. Sumber harga ditulis sebagai teks. `[!]` dirender sebagai ikon `triangle-alert` dengan `aria-label` berisi kalimat FR-EST-05 utuh. Tampilan Prospek sama dengan baris lama: `ScoreBadge`, opportunity teratas, indikator kontak.

### WF-02 — Profil bisnis (SCR-04, ≥ 1280 px)

```
+------------------------------------------------------------------------------------------------------------+
| < Kembali ke hasil   1 dari 214  [<] [>]                                                                   |
| Signa Cafe                                        [Analisis lokasi di sini] [Simpan ke list] [Google Maps] |
| Restoran (Turunan) · juga Kafe · <alamat>, Bali · Listing diklaim pemilik (Teramati) · Baru · 17 hari lalu |
| (Tanpa ICP aktif: "Kenapa bisnis ini?" dan opportunity tidak tampil)                                       |
+---------------------------------------------------------------------+--------------------------------------+
| PERKIRAAN KUNJUNGAN & OMZET  [Perkiraan]   Data per 24 Sep 2026     | HARGA, JAM BUKA & ATRIBUT            |
|                          Rendah      Tengah      Tinggi             | Harga  Rp100.000–225.000 per orang   |
| Kunjungan/tahun          13.800      18.400      27.600             |        Teramati · 24 Sep 2026    [i] |
| Tamu per hari buka       37,8        50,4        75,7               | Jam    Setiap hari 08.00–23.00 WITA  |
| Omzet/tahun              Rp1,8 M     Rp2,4 M     Rp3,6 M            |        Teramati                  [i] |
| Dasar: laju ulasan 276/tahun · dari riwayat snapshot                | Populer untuk: Sarapan, Makan siang, |
|   (10 Feb–24 Sep 2026; 1.336 → 1.507 ulasan) · harga tempat ini     |   Makan malam · Pengunjung: Turis    |
|   · asumsi: faktor belanja 0,8, review rate 2% / 1,5% / 1%          |                                      |
|   · buka 7 hari/minggu                                              |--------------------------------------|
| Pesanan aplikasi pesan-antar umumnya tidak tercakup.                | KONTAK              Belum di-unlock  |
| [!] Proporsi bintang 5 sangat tinggi (87,5%); bisa jadi lebih       | Telepon [lock] <ter-mask>            |
|     banyak tamu menulis ulasan dari biasanya, sehingga omzet        |                   [Unlock · 1 credit]|
|     cenderung di sisi bawah rentang.                                |                                      |
| [Tabel sensitivitas v]  [Lihat evidence]                est-1.0.0   |                                      |
|---------------------------------------------------------------------|--------------------------------------|
| POLA KERAMAIAN   Data per 24 Sep 2026 · waktu lokal WITA            | KEHADIRAN DIGITAL                    |
| Relatif terhadap jam tersibuk tempat ini, bukan jumlah orang.       | Website signa.cafe  Domain sendiri   |
| [Rata-rata 7 hari] [Sen] [Sel] [Rab] [Kam] [Jum] [Sab] [Min]        | Sosial  Instagram (Teramati)         |
|      |                               #  #  #                        | Rating  4,7 · 1.507 ulasan (Teramati)|
|      |       #  #  #  #  #  #  #  #  #  #  #  #                     |--------------------------------------|
|      | #  #  #  #  #  #  #  #  #  #  #  #  #  #  #                  | KONTEKS WILAYAH      [Lihat wilayah] |
|      +---------------------------------------------                 | <kelurahan> · populasi, kompetisi,   |
|        08 09 10 11 12 13 14 15 16 17 18 19 20 21 22                 | permintaan terbukti (Lampiran C)     |
|       [------------- jam inti 08–22 --------------]                 |--------------------------------------|
| Ramai sepanjang hari · puncak 20.00 · rata-rata 76% dari jam        | NARASI AI                            |
| tersibuknya                                  [Lihat sebagai tabel]  | Masalah · Data · Cara membaca ·      |
|---------------------------------------------------------------------| Kesimpulan · Batasan                 |
| RIWAYAT ULASAN                                       [Lihat data]   | [Buat narasi · 1 AI credit]          |
| 1.336  ___.....-----'''''  1.507        10 Feb 2026 → 24 Sep 2026   |--------------------------------------|
| Laju setahun 276/tahun · dari riwayat snapshot                      | KERJA TIM                            |
| 90 hari terakhir 342/tahun                                          | List: Kompetitor Bali · Tag: kafe    |
+---------------------------------------------------------------------+--------------------------------------+
```

Tab Sel menunjukkan puncak jam 13 (= 100). `[i]` membuka evidence drawer. Setelah unlock, baris telepon menjadi nomor penuh + [Salin] [Telepon] [WhatsApp] + "Di-unlock oleh Rina A. · 10 Okt 2026". Dengan ICP aktif, panel "Kenapa bisnis ini?" (`FactorBar`) dan Opportunity tampil di bawah Riwayat ulasan.

### WF-03 — Wilayah (SCR-05, ≥ 1280 px)

```
+------------------------------------------------------------------------------------------------------------+
| Wilayah [Jateng > Kota Semarang v]  Level: [Kecamatan | (Kelurahan)]  Kelompok: [(Makan) | Kafe | Hotel]   |
+-----------------------------------------------------+------------------------------------------------------+
| PERINGKAT · Makan (177 kelurahan: 169 diperingkat,  | [+] [-] [Reset]                                      |
| 8 Data terbatas)                                    |   +-----------+-----------+-----------+              |
| #  Kelurahan    Skor  Bisnis  Kompet.  Kunj./bisnis |   | Sekayu    | Pleburan  | Peterongan|              |
| 1  Pleburan       78     42   Sedang         9.100  |   |   [6]     | ==[7]==   |   [5]     |              |
| 2  Sekayu         74     35   Sedang         8.400  |   +-----------+-----------+-----------+              |
| 3  Peterongan     71     51   Tinggi         7.900  |   | Jomblang  | Lamper    | Bugangan  |              |
| …                                                   |   |  [////]   |   [4]     |   [3]     |              |
| -  Jomblang      Data terbatas · tanpa peringkat    |   +-----------+-----------+-----------+              |
|-----------------------------------------------------|   (skematik; peta asli = poligon GeoAPI)             |
| PLEBURAN · Kec. Semarang Selatan            [x]     | LEGEND: Skor pasar (kuantil dalam Kota Semarang)     |
| Skor pasar 78 / 100 · peringkat 1 dari 169          | [1][2][3][4][5][6][7] rendah -> tinggi               |
| Populasi ±13.500 · perkiraan model Kontur 2023 (i)  | [////] Data terbatas    ==[ ]== area terpilih        |
| Luas 1,2 km2 · Bisnis 186 · Makan 42                |------------------------------------------------------|
| Densitas makan 3,1 per 1.000 pddk              (i)  | Batas & populasi: GeoAPI (Kontur 2023, perkiraan     |
| Kompetisi Sedang (P48)                         (i)  | model) · Peta dasar © kontributor OpenStreetMap ·    |
| Permintaan terbukti ±382.000 kunjungan/thn     (i)  | geo-3.0.0                                            |
|   (31 bisnis ber-estimasi, rata-rata 9.100)         |                                                      |
| Harga median Rp45.000 per orang (18 bisnis)    (i)  |                                                      |
| [Lihat 42 bisnis]  [Analisis lokasi di area ini]    |                                                      |
+-----------------------------------------------------+------------------------------------------------------+
```

Tabel dan peta tersinkron: memilih baris menyorot poligon, dan sebaliknya. Tabel adalah padanan peta yang aksesibel (FR-GEO-01). Dengan ICP aktif, tabel menambah kolom bisnis cocok dan HOT/WARM.

### WF-04 — Hasil analisis lokasi (SCR-13, ≥ 1280 px)

```
+------------------------------------------------------------------------------------------------------------+
| < Analisis lokasi   Jl. Pulau Kawe · Makan · 500 m      [Ubah nama] [Jalankan ulang · 1 credit] [Hapus]    |
| Titik: Jl. Pulau Kawe (klik peta) · radius 500 m · kelompok Makan · harga rencana Rp25.000–50.000          |
| Pembanding: Joglo · Data per 20 Sep 2026 · loc-1.0.0 · dibuat Rina A., 11 Okt 2026                         |
+---------------------------------------------------------------------+--------------------------------------+
| KESIMPULAN                                            [Turunan]     | PETA                                 |
| - Saran jam buka 16.00–22.00 WITA (jam inti tetangga 17–21)         |   .klinik            .hotel          |
| - Posisi harga: lebih murah (50% dari median tetangga sejenis)      |          (x) titik   [KR]            |
| - Jam inti pembanding 12–14 tidak didukung tetangga                 |     .apotek            <Berkarier>   |
| - Keyakinan rendah: cek lapangan dulu                               |   ( lingkaran radius 500 m )         |
| Relevan untuk Anda?  [Relevan] [Tidak relevan]                      | [] sejenis  <> F&B lain  . non-F&B   |
|---------------------------------------------------------------------|--------------------------------------|
| KEYAKINAN  [# . .] Rendah · 1 tetangga sejenis dengan data cukup    | SUMBER PENGUNJUNG SIANG              |
| Cek lapangan: hitung kendaraan parkir di Kampoeng Renon pada jam    | Kesehatan 2 · Akomodasi 1 ·          |
| puncak (19.00) dan jam sepi (11.00) selama 2–3 hari.                | Kantor/pendidikan 0 · Ritel 0        |
|---------------------------------------------------------------------| Terdekat: <klinik> 150 m,            |
| POLA KERAMAIAN (bentuk, % dari total harian) · waktu lokal WITA     | <apotek 24 jam> 300 m,               |
| Relatif terhadap jam tersibuk, bukan jumlah orang.                  | <hotel> 450 m                        |
| [grafik garis 3 seri: tetangga sejenis · Joglo · F&B lain]          |--------------------------------------|
|   jam 11 … 23                                [Lihat sebagai tabel]  | NARASI AI                            |
|---------------------------------------------------------------------| Masalah · Data · Cara membaca ·      |
| JAM INTI & PEMBANDING     Tetangga sejenis   Joglo (pembanding)     | Kesimpulan · Batasan                 |
| Jam inti                  17–21              12–14 [!], 17–20       | [Buat narasi · 1 AI credit]          |
| Bentuk Siang (11–14)      23%                33%                    |                                      |
| Bentuk Sore (15–17)       23%                23%                    |                                      |
| Bentuk Malam (18–21)      53%                44%                    |                                      |
| [!] 12–14 tidak didukung tetangga                                   |                                      |
| Saran jam buka 16.00–22.00 = 1 jam sebelum jam inti pertama s.d.    |                                      |
| 1 jam setelah awal jam inti terakhir                                |                                      |
|---------------------------------------------------------------------|                                      |
| POSISI HARGA  Lebih murah                                           |                                      |
| Rencana Rp37.500 (titik tengah) vs median tetangga sejenis Rp75.000 |                                      |
| = 50% (< 80% lebih murah · 80–120% setara · > 120% lebih mahal)     |                                      |
+---------------------------------------------------------------------+--------------------------------------+
| TETANGGA (5 dalam 500 m)   [Sejenis 1] [F&B lain 1] [Non-F&B 3]                                            |
| Nama             Kategori   Jarak  Ulasan  Harga         Pola   Data cukup  Jam inti  Data per             |
| Kampoeng Renon   Restoran   200 m     687  Rp50–100 rb   Ada    Ya          17–21     20 Sep 2026          |
| (tab F&B lain: Berkarier Coffee · Kafe · 400 m · 304 · Rp25–50 rb · Ada · Ya · 20–23 · ramai larut malam)  |
+------------------------------------------------------------------------------------------------------------+
```

---

## 7. Copy & format id-ID

Nada: lugas, sopan, tanpa menyalahkan user. Istilah produk mengikuti PRD: *credit*, *unlock*, *export*, *webhook*, *Offering*, *ICP*, *opportunity*, *estimasi*, *analisis lokasi*, *narasi AI*, *jam inti*. Sentence case, tanpa huruf kapital penuh kecuali band HOT/WARM/COLD.

| Jenis | Format | Catatan |
|---|---|---|
| Angka · desimal · persen | 12.400 · 4,6 · 61% | `Intl.NumberFormat('id-ID')`. Maks. 1 desimal di UI |
| Rupiah | Rp1.500.000 · ringkas Rp640 jt / Rp2,4 M | Tanpa spasi setelah "Rp". "M" = miliar. Pembulatan estimasi: BR-EST-02 |
| Rentang perkiraan | 13.800–27.600 · Rp1,8–3,6 M | Tengah ditulis terpisah dan ditebalkan |
| Perkiraan tunggal | ±13.500 | Untuk populasi |
| Densitas | 3,1 per 1.000 penduduk | Lampiran C |
| Laju ulasan | 276 ulasan/tahun · dari riwayat snapshot (10 Feb–24 Sep 2026) | Selalu dengan sumber dan jendelanya (Lampiran E §E.1). Copy sumber di tabel teks estimasi di bawah |
| Tanggal · tanggal + jam | 10 Okt 2026 · 10 Okt 2026, 14.30 WIB | Waktu sistem `Asia/Jakarta` (NFR-07) |
| Jam buka & jam lokal | 08.00–23.00 WITA · puncak 20.00 WITA | Zona waktu lokal tempat + label WIB/WITA/WIT (BR-NORM-06). Label zona boleh ditulis sekali per panel |
| Jam pola (blok 1 jam) | jam 17–21 · jam 12–14 dan 17–20 | "jam 21" = 21.00–21.59. Saran jam buka ditulis sebagai jam operasional (16.00–22.00) |
| Waktu relatif | baru saja · 5 menit lalu · 3 hari lalu | Lebih dari 30 hari → tanggal absolut, kecuali umur data di badge freshness ("Menua · 210 hari lalu"). `<time datetime>` selalu ada |
| Telepon | +62 812-3456-7890 | Disimpan E.164 (BR-NORM-01) |
| Credit | 1 credit · 35 credit · 1 AI credit | Tanpa bentuk jamak |

**Pola error:** *[Apa yang terjadi]. [Kenapa, bila diketahui]. [Apa yang bisa dilakukan].* + tombol aksi + `Kode <8 karakter>` (dari correlation ID). Kode teknis tidak dipakai sebagai kalimat utama.

| Situasi | Copy |
|---|---|
| Dialog unlock | "Unlock kontak Rumah Makan Sari Rasa? Biaya 1 credit · saldo 184 → 183. Kontak terbuka permanen untuk workspace ini." |
| Dialog Unlock & export | "Export 120 bisnis (CSV). 85 sudah di-unlock (gratis), 35 akan di-unlock · 35 credit. Saldo 184 → 149. Unlock dijalankan sekarang; bila export gagal, ulangi tanpa biaya." |
| Dialog analisis lokasi | "Jalankan analisis lokasi? Biaya 1 credit · saldo 184 → 183. Hasil tersimpan dan bisa dibuka ulang gratis. Tidak ditagih bila hasilnya Tidak dapat dianalisis." |
| Credit kurang | "Credit tidak cukup. Aksi ini butuh 35 credit, saldo Anda 20. Minta Admin workspace menambah credit." |
| Tidak diketahui | "Tidak diketahui — data sumber tidak memuat jam buka." |
| Keyakinan rendah | "Keyakinan rendah: hanya 1 tetangga sejenis dengan data keramaian. Cek lapangan: hitung kendaraan parkir di Kampoeng Renon pada jam puncak dan jam sepi selama 2–3 hari." |
| Tidak dapat dianalisis | "Tidak dapat dianalisis: tidak ada tetangga sejenis dengan data keramaian dalam radius 500 m. Perbesar radius atau lihat F&B lain di bawah." |
| Tidak didukung tetangga | "Jam inti pembanding 12–14 tidak didukung tetangga: tetangga sejenis tidak ramai di jam ini." |
| WhatsApp | "Kemungkinan WhatsApp karena nomor seluler. Belum dikonfirmasi." |
| Export gagal | "Export gagal diproses. Layanan penyimpanan tidak merespons. Kontak yang sudah di-unlock tetap terbuka; coba lagi tanpa biaya. [Coba lagi] · Kode 7F3A21C0" |
| URL webhook ditolak | "URL tidak dapat dipakai. Alamat ini mengarah ke jaringan privat atau internal. Gunakan URL HTTPS publik." |
| Seat penuh | "Seat plan Growth sudah terpakai 5 dari 5. Keluarkan anggota atau batalkan undangan, atau hubungi tim MesinProspek untuk menambah seat." |
| Pengaturan AI disimpan (ops) | "Pengaturan AI disimpan. Narasi baru memakai model ini dalam 1 menit." |
| Parameter estimasi disimpan (ops) | "Parameter estimasi disimpan. Estimasi semua bisnis sedang dihitung ulang." |
| Import gagal: skema sumber (ops) | "Import gagal: skema DB gmaps tidak cocok (kolom `price_range` tidak ditemukan). Tidak ada data yang ditulis; aplikasi tetap memakai data terakhir. Perbaiki `source.sql`, lihat 09 RB-03." + pesan teknis utuh di bawahnya |
| Import sedang berjalan (ops, CLI) | "Import sedang berjalan. Tunggu sampai selesai; sisa data terbaca di run berikutnya." |

**Teks estimasi.** 04 §4.1 menyimpan kode `flags`/`reason` yang stabil; teks Indonesianya hanya di tabel ini (DRY). Angka di contoh diisi dari data.

| Kode / situasi | Tampil di | Copy |
|---|---|---|
| `five_star_heavy` | Panel perkiraan, baris tetangga | "Proporsi bintang 5 sangat tinggi (87,5%); bisa jadi lebih banyak tamu menulis ulasan dari biasanya, sehingga omzet cenderung di sisi bawah rentang." |
| `velocity_changed` | Panel perkiraan, baris tetangga | "Laju ulasan berubah: 90 hari terakhir 410 ulasan/tahun, setahun 276 ulasan/tahun." |
| `price_unknown` | Baris omzet | "Omzet belum dapat diperkirakan — harga per orang tidak diketahui, dan belum ada median wilayah atau asumsi harga untuk kelompok ini. Perkiraan kunjungan tetap ditampilkan." |
| `hotel_params_missing` | Panel hotel | "Belum dapat diperkirakan — asumsi hotel kelas <kelas> belum diisi tim MesinProspek." |
| `occupancy_inconsistent` | Baris okupasi | "Okupasi tidak ditampilkan: hasil hitungnya melebihi 100%, jadi asumsi jumlah kamar atau lama menginap kemungkinan tidak cocok untuk tempat ini." |
| Sumber laju: riwayat snapshot (`velocity_method` `snapshots`) | Baris Dasar, riwayat ulasan | "Laju ulasan 276/tahun · dari riwayat snapshot (10 Feb–24 Sep 2026)." |
| Sumber laju: ulasan terbaru (`velocity_method` `newest_reviews`) | Baris Dasar, riwayat ulasan | "Laju ulasan 289/tahun · dari 20 ulasan terbaru (31 Agu–24 Sep 2026)." |
| `short_window` | Di samping sumber laju atau baris cek silang | "Jendela pendek: 20 ulasan terbaru hanya mencakup 24 hari, jadi laju bisa terpengaruh musim atau promo." |
| Cek silang (`velocity_crosscheck`) | Di bawah sumber laju, bila kedua metode tersedia | "Cek silang: dari 20 ulasan terbaru (15 Agu–24 Sep 2026) 173/tahun, 5% lebih rendah dari riwayat snapshot." |
| `review_data_insufficient` | Panel perkiraan, riwayat ulasan | "Belum dapat diperkirakan — belum cukup data ulasan: riwayat snapshot kurang dari 30 hari (sejak 2 Okt 2026) dan ulasan terbaru kurang dari 10." Tanpa tanggal ulasan terbaru, akhir kalimatnya menjadi "…dan tanggal ulasan terbaru belum tersedia." |
| Jumlah ulasan tidak diketahui | Panel perkiraan | "Belum dapat diperkirakan — jumlah ulasan tidak diketahui." |
| Kelompok `lainnya` | Panel perkiraan | "Belum dapat diperkirakan — estimasi belum mendukung kategori ini." |
| Tanpa `popular_times` | Panel pola keramaian | "Belum dapat diperkirakan — data pola keramaian tidak tersedia untuk tempat ini." |
| Catatan metode (selalu) | Panel perkiraan | "Kunjungan dihitung dari ulasan Google Maps, sehingga pesanan lewat aplikasi pesan-antar umumnya tidak tercakup." |
| Label pola (selalu) | Setiap chart/tabel pola | "Relatif terhadap jam tersibuk tempat ini, bukan jumlah orang." |
| Pola datar | Ringkasan pola | "Ramai sepanjang hari: jam inti mencakup hampir semua jam buka." |

---

## 8. Aksesibilitas (NFR-06, WCAG 2.2 AA dasar)

- Status tidak pernah hanya warna: ikon + label teks; claim type juga memakai gaya border (§5). Jam inti ditandai di sumbu X (02 §3.3) dan jam puncak diberi label teks.
- Hanya token semantik `02`. Kontras dijaga oleh `build.mjs --check` di CI; warna hardcode ditolak lint.
- Semua aksi bisa dijalankan dengan keyboard dalam urutan fokus yang logis. Focus ring 2px `color.border.focus` pada `:focus-visible`. Shortcut `/` memfokuskan search.
- Dialog dan drawer (Radix): focus trap, Esc menutup, dan fokus kembali ke pemicu.
- Tabel: `<th scope>`, `aria-sort` pada kolom urut. Checkbox dan target ≥ 24px (`size.target-min`); 44px di layar sentuh.
- **Setiap chart punya padanan tabel** (02 §3.3): pola keramaian (profil & analisis), perbandingan pola, rentang estimasi (tabel Rendah · Tengah · Tinggi), dan riwayat ulasan. Sparkline punya ringkasan teks (nilai awal, akhir, laju).
- Peta wilayah selalu punya padanan tabel peringkat; peta analisis lokasi punya tabel Tetangga. Titik analisis bisa dipilih tanpa peta (pilih bisnis atau wilayah dari pencarian). Radius bisa diisi lewat input angka. Legend berlabel teks.
- Label screen reader: `ScoreBadge` "Skor prioritas 85, HOT"; `MaskedValue` "Nomor disembunyikan, unlock 1 credit"; `ConfidenceMeter` "Keyakinan rendah, 1 dari 3"; rentang "Perkiraan omzet per tahun, rendah 1,8 miliar, tengah 2,4 miliar, tinggi 3,6 miliar"; `FactorBar` disajikan sebagai `<table>`.
- `lang="id"`. Progres skor/export/narasi/analisis diumumkan lewat `role="status"`. `prefers-reduced-motion` dihormati (durasi 0 di `tokens.css`).

---

## 9. Di luar F1

- Analytics produk: F2 (lihat PRD §6).
- Perbandingan bebas antar bisnis dan antar wilayah, estimasi kategori lain, watch/alert: F2.
- F2/F3 lainnya: lihat [PRD §6](01-prd.md).
