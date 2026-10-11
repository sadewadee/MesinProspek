# 10 — Ringkasan Bisnis

| Field | Value |
|---|---|
| Versi | 0.3 (lean) · 2026-10-11. Posisi baru: market intelligence untuk bisnis lokal Indonesia (D-30) |
| Tujuan | Mengunci hal bisnis yang dibutuhkan sebelum menjual F1: posisi, harga & paket, unit economics, rencana beta |
| Rujukan | Masalah, persona, dan scope ada di [PRD](01-prd.md) §1–§4. Operasi yang memakai credit ada di PRD Lampiran D. Keduanya tidak ditulis ulang di sini (DRY). Posisi lama (sales intelligence, MVP Pack v1.2) digantikan D-30 |

> **Semua angka di dokumen ini adalah hipotesis** untuk divalidasi saat beta. Angka yang perlu diisi founder ditandai **[isi]**.

---

## 1. Lean canvas

| Blok | Isi |
|---|---|
| **Masalah** | (1) Pemilik usaha memilih lokasi dan jam buka cabang dari pengamatan sesaat. (2) Omzet kompetitor dan ukuran pasar tidak diketahui; hitungan manual lambat dan mudah memakai asumsi basi. (3) Penjual B2B tidak tahu bisnis mana yang layak dihubungi. Rincian di PRD §2 |
| **Segmen** | **Utama (D-30):** pemilik usaha F&B dan multi-outlet (PER-01); investor, analis, dan franchisor (PER-02). **Sekunder:** penjual B2B ke bisnis lokal (PER-03). *Early adopter:* pemilik F&B 2–10 outlet di kota tier 1–2 yang sedang merencanakan cabang, dan franchisor yang menilai lokasi calon mitra |
| **Nilai unik** | "Perkiraan kunjungan dan omzet usaha mana pun, lengkap dengan rentang, asumsi, dan bukti, plus analisis lokasi dan jam buka untuk cabang berikutnya." |
| **Solusi** | Estimasi kunjungan dan omzet (okupasi untuk hotel) dari laju ulasan yang diukur; pola keramaian per jam; analisis lokasi (jam inti, saran jam buka, posisi harga, tingkat keyakinan); peringkat wilayah per kelurahan; narasi AI. Fitur prospek (unlock, export, webhook) untuk PER-03 |
| **Channel** | Jaringan founder; komunitas dan asosiasi pengusaha kuliner dan waralaba; konsultan bisnis/waralaba dan broker properti komersial sebagai perujuk; konten edukasi per kota (cara membaca keramaian, memilih jam buka); komunitas agency untuk segmen sekunder |
| **Pendapatan** | Langganan bulanan per plan (seat + kuota credit) ditambah top-up credit. Credit dipakai untuk analisis lokasi baru dan unlock kontak |
| **Biaya** | 1 dedicated server (semua layanan), backup off-site, API AI, domain/SMTP, waktu founder. Data dari DB gmaps yang dikelola founder terpisah (§4.2) |
| **Metrik kunci** | Aktivasi (hasil pertama ≤ 5 menit); analisis lokasi per workspace per minggu; kredibilitas estimasi (§5); % kesimpulan analisis lokasi dan opportunity yang dinilai "Relevan" (FR-INTL-05); konversi beta → berbayar; churn bulanan |
| **Keunggulan** | Riwayat snapshot berulang per tempat (Signa Cafe: 171 snapshot, Februari–September 2026) yang tidak bisa dibuat ulang ke belakang oleh pendatang baru, sehingga laju ulasan diukur, bukan ditebak. Pola keramaian, harga, dan wilayah per kelurahan ada dalam satu model. Setiap angka menunjukkan asumsi dan buktinya |

## 2. Positioning singkat

| | Database lead umum | Tool analitik foot traffic | **MesinProspek** |
|---|---|---|---|
| Pertanyaan yang dijawab | "Siapa dan bagaimana kontaknya?" | "Berapa orang datang ke lokasi ini?" | "Seberapa ramai dan berapa kira-kira omzet usaha ini? Di mana dan jam berapa cabang saya sebaiknya buka?" |
| Sumber data | Direktori dan listing bisnis | Umumnya data lokasi perangkat seluler (panel) yang dimodelkan | Snapshot Google Maps berulang (ulasan, harga, `popular_times`) + wilayah GeoAPI |
| Estimasi untuk usaha lokal | Umumnya tidak ada | Umumnya jumlah kunjungan, bukan omzet | Rentang kunjungan dan omzet + asumsi + tanggal data; okupasi hotel |
| Keputusan lokasi & jam buka | Tidak ada | Analisis area dagang, umumnya untuk ritel dan properti skala besar | Analisis per titik: jam inti tetangga, saran jam buka, posisi harga, tingkat keyakinan |
| Transparansi | Status verifikasi kontak | Bervariasi per penyedia | Claim type, sumber, dan parameter untuk setiap angka |
| Tindak lanjut | Kontak, umumnya email | Laporan dan dashboard | Narasi AI; unlock telepon/WhatsApp, export, webhook |

**Batas yang kami sampaikan terus terang:** estimasi bergantung pada asumsi review rate (1% vs 2% = 2×), pola keramaian hanya ada di ±13% tempat, dan harga di ±0,8% (PRD §13). Karena itu yang dijual adalah rentang dan bukti, bukan angka pasti. Analisis lokasi dengan keyakinan Rendah selalu disertai saran cek lapangan.

## 3. Hipotesis harga & paket

> Harga, seat, dan kuota **diatur dari dashboard ops** (D-29), jadi bisa diubah kapan saja tanpa deploy. Tabel ini hanya nilai awal untuk diisi ke dashboard dan divalidasi saat beta.

Credit dipakai untuk analisis lokasi baru dan unlock kontak, AI credit untuk narasi AI. Tarifnya hanya ditetapkan di PRD Lampiran D (BR-CRED-01). Tabel ini hanya berisi harga dan kuota.

| Plan | Harga/bulan | Seat | Credit/bulan | AI credit/bulan | Untuk siapa |
|---|---|---|---|---|---|
| Starter | Rp299.000 | 2 | 200 | 50 | Pemilik usaha 1–2 outlet, analis perorangan |
| Growth | Rp999.000 | 5 | 1.000 | 300 | F&B multi-outlet, investor/analis kecil, tim sales B2B |
| Agency | Rp2.499.000 | 15 | 3.000 | 1.000 | Franchisor dan jaringan multi-outlet besar, konsultan atau agency multi-klien |
| Top-up | Rp150.000 / 100 credit | — | +100 | — | Semua plan |

Prinsip:
- Estimasi dan pola keramaian di profil tidak memakai credit (Lampiran D). Nilai langganan ada di akses ke data itu; credit membatasi operasi bernilai tinggi.
- Satu credit efektif ≈ Rp800–1.500, jauh di bawah nilai satu keputusan lokasi atau satu prospek yang relevan. Validasi saat beta lewat wawancara kesediaan bayar per segmen dan konversi.
- Nama "Agency" kurang cocok untuk segmen utama. Nama final ikut D-09 (§6).

## 4. Unit economics (asumsi)

### 4.1 Biaya variabel: narasi AI

Harga API Claude per 1 juta token, sumber referensi resmi per 2026-10-06. Asumsi satu narasi (profil atau analisis lokasi): ±1.500 token input dan ±1.000 token output (termasuk thinking). Narasi analisis lokasi memuat lebih banyak fakta, jadi input-nya bisa lebih besar **[sesuaikan setelah prompt `brief-` final]**. Kurs asumsi **Rp16.500/USD [sesuaikan]**.

| Model | Input / output per 1 juta token | Biaya per narasi | Biaya 300 narasi (kuota Growth) |
|---|---|---|---|
| Claude Opus 5.5 | $4 / $20 | ≈ $0,026 ≈ Rp430 | ≈ Rp129.000 |
| Claude Sonnet 5.5 | $2 / $10 | ≈ $0,013 ≈ Rp215 | ≈ Rp64.000 |
| Claude Haiku 5.5 | $0,10 / $0,50 | ≈ $0,00065 ≈ Rp11 | ≈ Rp3.200 |

Provider dan model **diatur dari dashboard ops** dan bisa diganti kapan saja (D-29), termasuk fallback ke API OpenAI-compatible. Cache 7 hari (FR-ACT-05) menurunkan biaya untuk bisnis yang dibuka berulang. Kuota AI credit membatasi biaya maksimum per workspace.

### 4.2 Biaya tetap per bulan

| Pos | Asumsi |
|---|---|
| 1 dedicated server di Indonesia (production, staging, GitLab, CI, monitoring): sudah ada, 56 core, 128 GB RAM, disk 1 TB | **[isi]** biaya bulanannya |
| Backup off-site: SFTP/FTPS milik founder di Indonesia (sudah ada); Backblaze B2 opsional setelah Q-19 | **[isi]** biaya bulanannya |
| Domain, SMTP, lain-lain | **[isi]**, mis. Rp200 ribu |
| GeoAPI | Gratis (kunci API gratis). Wajib mencantumkan atribusi sumber |
| DB gmaps dan scraper (dikelola founder terpisah) | **[isi]** bila dibebankan ke MesinProspek |

### 4.3 Contoh perhitungan (plan Growth, skenario terburuk Opus 5.5)

| Item | Per workspace/bulan |
|---|---|
| Pendapatan | Rp999.000 |
| Biaya AI maksimum (300 narasi, Opus 5.5) | − Rp129.000 |
| Porsi infra (biaya tetap Rp5 juta dibagi 30 workspace) | ≈ − Rp167.000 |
| **Margin kotor** | **≈ Rp703.000 (≈ 70%)** |

Titik impas infra ≈ biaya tetap ÷ (harga rata-rata − biaya AI rata-rata). Dengan biaya tetap Rp5 juta dan campuran plan rata-rata Rp800 ribu, impas tercapai di **±7 workspace berbayar**.

## 5. Rencana beta → berbayar

| Langkah | Isi |
|---|---|
| Peserta | **Sudah ada** (Q-07). Seluruh Indonesia (D-34); rekrut peserta di kota yang data ulasannya sudah cukup: saat ini Bali (riwayat snapshot), dan seluruh Indonesia setelah scraper mengambil ulasan terbaru (D-38). Pastikan peserta mencakup segmen utama D-30 (pemilik F&B/multi-outlet, investor/franchisor), bukan hanya penjual B2B: **[isi jumlah per segmen]** |
| Penawaran | Gratis 30 hari, plan Growth, plus sesi onboarding 30 menit. Diskon 50% selama 3 bulan pertama bila lanjut berbayar |
| Umpan balik | Tombol "Relevan / Tidak relevan" pada opportunity dan kesimpulan analisis lokasi (FR-INTL-05), wawancara singkat mingguan, dan catatan dari ops. MesinProspek tidak meminta atau menyimpan omzet nyata peserta |
| Metrik sukses beta | ≥ 70% peserta mendapat hasil pertama ≤ 5 menit · **Kredibilitas estimasi:** ≥ 70% peserta menilai estimasi dan analisis lokasi "masuk akal" (wawancara + tombol feedback) · ≥ 60% kesimpulan analisis lokasi dan opportunity dinilai "Relevan" · workspace penjual B2B: ≥ 50 prospek ditindaklanjuti per minggu · **≥ 50% peserta lanjut berbayar** |
| Keputusan setelah beta | Kunci harga dan nama plan (D-09); putuskan apakah estimasi cukup kredibel untuk dijual ke investor/franchisor; tentukan prioritas F2 dari permintaan nyata (pembanding BPS, kategori lain, pantau kompetitor); sesuaikan bobot skor dan katalog opportunity untuk PER-03 |

## 6. Yang perlu diputuskan founder

| ID | Keputusan | Kapan |
|---|---|---|
| D-09 | Harga, kuota, dan nama plan final (diubah langsung di dashboard ops) | Setelah beta |
| — | Model AI default (bisa diganti kapan saja di dashboard ops) | Saat narasi AI pertama aktif |
| — | Biaya server & backup aktual (mengisi §4.2) | Sekarang (server dan SFTP sudah ada) |
| — | Biaya DB gmaps dan scraper dibebankan ke MesinProspek atau tidak (§4.2) | Sebelum beta berbayar |
