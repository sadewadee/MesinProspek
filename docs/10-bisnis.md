# 10 — Ringkasan Bisnis

| Field | Value |
|---|---|
| Versi | 0.1 (lean) |
| Tanggal | 2026-10-10 |
| Tujuan | Mengunci hal bisnis yang dibutuhkan sebelum menjual F1: harga & paket, unit economics, rencana beta |
| Rujukan | Positioning, monetisasi, dan benchmark Coldlytics ada di MVP Pack v1.2 (§01, §09, §16, Rev 1.1). Tidak ditulis ulang di sini (DRY) |

> **Semua angka di dokumen ini adalah hipotesis** untuk divalidasi saat beta. Angka yang perlu diisi founder ditandai **[isi]**.

---

## 1. Lean canvas

| Blok | Isi |
|---|---|
| **Masalah** | (1) Daftar bisnis mentah tidak memberi tahu siapa yang butuh layanan kita. (2) Riset manual 10–20 menit per prospek. (3) Tidak ada dasar data untuk memilih wilayah. |
| **Segmen** | Agency digital (website, sosmed, ads, SEO lokal) dan vendor SaaS UMKM yang menjual ke bisnis lokal Indonesia (D-16). *Early adopter:* agency kecil (2–15 orang) di kota tier 1–2. |
| **Nilai unik** | "Bukan sekadar daftar kontak: tahu **siapa** yang butuh layanan Anda, **kenapa**, dan **di wilayah mana** peluangnya terbesar." |
| **Solusi** | Opportunity berbasis evidence (mis. website hanya linktr.ee, ulasan sedikit), skor prioritas sesuai ICP, peringkat wilayah per kelurahan, unlock kontak WhatsApp, export/webhook, dan AI brief. |
| **Channel** | Jaringan founder, komunitas agency & UMKM, konten LinkedIn/Instagram (contoh peluang nyata per kota), kemitraan dengan komunitas pelatihan digital marketing. |
| **Pendapatan** | Langganan bulanan per plan (seat + kuota credit) ditambah top-up credit. |
| **Biaya** | Server (2 dedicated server), backup off-site, API Claude, domain/SMTP, waktu founder. Dataset sudah dimiliki. |
| **Metrik kunci** | Aktivasi (hasil pertama ≤ 5 menit), prospek yang ditindaklanjuti per workspace per minggu, % opportunity "Relevan", konversi beta → berbayar, churn bulanan. |
| **Keunggulan** | Data lokal Indonesia + konteks wilayah per kelurahan + aturan opportunity yang menjelaskan alasannya. Kompetitor global (mis. Coldlytics) kuat di kontak email, tetapi tidak di konteks lokal dan WhatsApp. |

## 2. Positioning singkat

| | Lead database global (mis. Coldlytics) | **MesinProspek** |
|---|---|---|
| Pertanyaan yang dijawab | "Siapa dan bagaimana kontaknya?" | "Siapa yang **butuh** layanan saya, **kenapa**, dan **di mana**?" |
| Kanal kontak utama | Email | Telepon/WhatsApp + sosial (sesuai kebiasaan UMKM Indonesia) |
| Konteks wilayah | Tidak ada | Peringkat kelurahan/kecamatan, kompetisi, populasi |
| Transparansi data | Status verifikasi | Claim type + evidence untuk setiap nilai |

## 3. Hipotesis harga & paket

Operasi yang memakai credit ditetapkan di PRD (Lampiran D, BR-CRED-01). Tabel ini hanya berisi harga dan kuota.

| Plan | Harga/bulan | Seat | Credit/bulan (unlock) | AI credit/bulan | Untuk siapa |
|---|---|---|---|---|---|
| Starter | Rp299.000 | 2 | 200 | 50 | Freelancer / agency sangat kecil |
| Growth | Rp999.000 | 5 | 1.000 | 300 | Agency kecil, tim sales vendor SaaS |
| Agency | Rp2.499.000 | 15 | 3.000 | 1.000 | Agency menengah, multi-klien |
| Top-up | Rp150.000 / 100 credit | — | +100 | — | Semua plan |

Prinsip: harga per unlock yang efektif (Rp1.000–1.500) dijaga di bawah nilai satu prospek yang relevan bagi agency. Validasi saat beta lewat wawancara kesediaan bayar dan konversi.

## 4. Unit economics (asumsi)

### 4.1 Biaya variabel: AI brief

Harga API Claude per 1 juta token, sumber referensi resmi per 2026-10-06. Asumsi satu brief: ±1.500 token input dan ±1.000 token output (termasuk thinking). Kurs asumsi **Rp16.500/USD [sesuaikan]**.

| Model | Input / output per 1 juta token | Biaya per brief | Biaya 300 brief (kuota Growth) |
|---|---|---|---|
| Claude Opus 5.5 | $4 / $20 | ≈ $0,026 ≈ Rp430 | ≈ Rp129.000 |
| Claude Sonnet 5.5 | $2 / $10 | ≈ $0,013 ≈ Rp215 | ≈ Rp64.000 |
| Claude Haiku 5.5 | $0,10 / $0,50 | ≈ $0,00065 ≈ Rp11 | ≈ Rp3.200 |

Model dipilih founder lewat konfigurasi (`MSP_*`). Cache 7 hari (FR-ACT-05) menurunkan biaya untuk prospek yang dibuka berulang. Kuota AI credit membatasi biaya maksimum per workspace.

### 4.2 Biaya tetap per bulan

| Pos | Asumsi |
|---|---|
| 2 dedicated server di Indonesia (prod + ops) | **[isi]**, mis. Rp4–8 juta |
| Object storage off-site untuk backup | **[isi]**, mis. Rp200–500 ribu |
| Domain, SMTP, lain-lain | **[isi]**, mis. Rp200 ribu |
| GeoAPI | Gratis (kunci API gratis). Wajib mencantumkan atribusi sumber |

### 4.3 Contoh perhitungan (plan Growth, skenario terburuk Opus 5.5)

| Item | Per workspace/bulan |
|---|---|
| Pendapatan | Rp999.000 |
| Biaya AI maksimum (300 brief, Opus 5.5) | − Rp129.000 |
| Porsi infra (biaya tetap Rp6 juta dibagi 30 workspace) | − Rp200.000 |
| **Margin kotor** | **≈ Rp670.000 (≈ 67%)** |

Titik impas infra ≈ biaya tetap ÷ (harga rata-rata − biaya AI rata-rata). Dengan biaya tetap Rp6 juta dan campuran plan rata-rata Rp800 ribu, impas tercapai di **±9 workspace berbayar**.

## 5. Rencana beta → berbayar

| Langkah | Isi |
|---|---|
| Kriteria peserta | 5–10 agency/vendor SaaS (Q-07) di 1–2 kota dengan cakupan dataset terbaik, yang sudah aktif mencari klien UMKM |
| Penawaran | Gratis 30 hari, plan Growth, plus sesi onboarding 30 menit. Diskon 50% selama 3 bulan pertama bila lanjut berbayar |
| Umpan balik | Tombol "Relevan / Tidak relevan" (FR-INTL-05), wawancara singkat mingguan, dan catatan dari ops |
| Metrik sukses beta | ≥ 70% peserta mendapat hasil pertama ≤ 5 menit · ≥ 50 prospek ditindaklanjuti per workspace per minggu · ≥ 60% opportunity dinilai "Relevan" · **≥ 50% peserta lanjut berbayar** |
| Keputusan setelah beta | Kunci harga (D-09), sesuaikan bobot skor dan katalog opportunity, tentukan prioritas F2 dari permintaan nyata |

## 6. Yang perlu diputuskan founder

| ID | Keputusan | Kapan |
|---|---|---|
| D-09 | Harga & kuota plan final | Setelah beta |
| — | Model Claude default untuk AI brief | Sebelum WP AI brief (07) |
| — | Biaya server & backup aktual (mengisi §4.2) | Saat server disewa (Q-09) |
| Q-07 | Daftar calon peserta beta | Sebelum gerbang F1 |
