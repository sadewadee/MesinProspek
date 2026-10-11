# 01 — PRD (lean)

## MesinProspek — Market Intelligence untuk Bisnis Lokal Indonesia

| Field | Value |
|---|---|
| Versi | 0.3 (lean). Posisi berubah dari sales intelligence menjadi market intelligence (D-30) |
| Tanggal | 2026-10-11 |
| Prinsip | **KISS**, **DRY**, **Make it work → make it right → make it fast** (D-23) |
| Input | **DB gmaps**: kiriman mingguan ramping dari founder (hanya Indonesia, kolom yang dipakai, baris baru) dimuat ke server kita lalu dibaca read-only, berisi snapshot berulang per tempat (D-31, D-35, D-37) + data wilayah GeoAPI (D-25). **F1 melayani seluruh Indonesia** (D-28, D-34). MesinProspek tidak melakukan scraping; crawler website **PAUSED** (D-01) |
| Pembangun | Agent AI (maks. 8 paralel) diorkestrasi Claude. Founder menyetujui tiap gerbang fase (D-22) |

> Konvensi: `FR-<AREA>-nn`, `BR-<AREA>-nn`, `NFR-nn`. **F1** = make it work (rilis jual pertama), **F2** = make it right, **F3** = make it fast. Hanya F1 yang dirinci. Fitur masuk F1 hanya jika dibutuhkan alur inti: **temukan → pahami → prioritaskan → tindak lanjuti**.

---

## 1. Ringkasan

MesinProspek adalah **market intelligence untuk bisnis lokal Indonesia**. Dari data publik Google Maps yang diambil berulang dan data wilayah, produk:

- memperkirakan **kunjungan dan omzet** sebuah usaha, juga **okupasi** untuk hotel;
- menunjukkan **pola keramaian** per jam dan per hari;
- membantu memilih **lokasi, jam operasional, dan posisi harga** untuk cabang baru;
- membandingkan usaha dengan **kompetitor dan tetangganya**.

Setiap angka menunjukkan bukti, asumsi, rentang, dan tanggal datanya. Fitur prospek (unlock kontak, export, webhook) tetap ada sebagai salah satu cara menindaklanjuti.

Alur inti: **temukan** (bisnis atau wilayah) → **pahami** (estimasi, pola, bukti) → **prioritaskan** (peringkat wilayah, perbandingan, skor) → **tindak lanjuti** (keputusan lokasi/jam, narasi AI, unlock, export, webhook).

## 2. Masalah

1. Pemilik usaha yang mau buka cabang atau mengubah jam buka menebak dari pengamatan sesaat. Data keramaian tetangga dan kompetitor ada, tetapi tersebar dan tidak pernah dihitung.
2. Omzet kompetitor dan ukuran pasar tidak diketahui. Hitungan manual lambat dan mudah memakai data atau asumsi basi. Contoh Signa Cafe: tebakan laju ulasan dari umur cafe meleset sekitar 60%, sehingga omzet meleset sekitar 40%.
3. Penjual B2B tidak tahu bisnis mana yang paling layak dihubungi dan kenapa.

## 3. Persona

| ID | Persona | Kebutuhan utama |
|---|---|---|
| PER-01 | Pemilik usaha F&B atau multi-outlet | "Di mana dan jam berapa cabang baru sebaiknya buka? Bisakah konsep saya disalin ke sana?" |
| PER-02 | Investor, analis, franchisor | "Berapa kira-kira kunjungan dan omzet usaha ini atau kompetitornya, dan seberapa yakin angkanya?" |
| PER-03 | Penjual B2B (agency, vendor SaaS, sales manager) | "Bisnis dan wilayah mana yang paling layak dihubungi, dan kenapa?" |
| PER-04 | Ops internal MesinProspek | "Import lancar, parameter estimasi masuk akal, customer terlayani." |

PER-01 dan PER-02 adalah pengguna utama F1 (D-30). PER-03 tetap dilayani lewat fitur prospek.

## 4. Tujuan & non-tujuan

**Tujuan F1:**
- User melihat perkiraan kunjungan dan omzet sebuah usaha (okupasi untuk hotel) beserta rentang, asumsi, dan tanggal datanya.
- User melihat pola keramaian per jam dan per hari.
- User bisa menganalisis satu lokasi dan mendapat jam inti, saran jam buka, posisi harga, dan tingkat keyakinan. Analisis kedua sample case founder (Warung Kita, Signa Cafe) dapat direproduksi (§12).
- Setiap nilai menunjukkan sumber dan claim type. Estimasi yang datanya kurang tampil "Belum dapat diperkirakan".
- User bisa unlock kontak, export, atau mengirim ke webhook. Pemakaian credit tercatat benar.
- Hasil pertama tampil ≤ 5 menit setelah daftar.

**Non-tujuan:** scraping (data dikelola terpisah di DB gmaps), crawler website, CRM, outreach otomatis, aplikasi mobile native, dan isu hukum (ditangani tim legal, D-27).

## 5. Prinsip produk

1. **Unknown lebih baik daripada mengarang.** Field yang tidak ada tampil "Tidak diketahui", bukan ditebak.
2. **Evidence untuk setiap klaim.** Nilai bisa ditelusuri ke snapshot sumber.
3. **Ukur, jangan tebak.** Yang bisa diukur dari data diukur, misalnya laju ulasan dari snapshot. Asumsi hanya dipakai untuk yang tidak bisa diukur (review rate, faktor belanja, kapasitas hotel), dan semuanya diatur dari dashboard ops.
4. **Estimasi selalu rentang + asumsi + tanggal.** Tidak ada angka omzet tunggal tanpa rentang, parameter yang dipakai, dan tanggal data.
5. **Hanya janjikan yang bisa kita sediakan.** Semua fitur dibangun dari data yang kita punya (DB gmaps, GeoAPI, benchmark publik). Data internal usaha (omzet nyata, jumlah transaksi, kapasitas) tidak tersedia dan tidak dijanjikan.
6. **Satu record kanonik, banyak tampilan** (DRY). Search, profil, analisis lokasi, export, dan webhook membaca model yang sama.
7. **Fitur kecil yang bekerja** lebih baik daripada fitur lengkap yang setengah jadi.

## 6. Fase rilis

| Fase | Tema | Isi |
|---|---|---|
| **F1 — Make it work** | Alur inti end-to-end, bisa dijual ke customer awal | Semua FR di §9 |
| **F2 — Make it right** | Kualitas, kelengkapan, kenyamanan | **Insight teks ulasan** (`user_reviews`, tanpa nama/foto pengulas): tema pujian/keluhan dan sebutan harga per orang, dikumpulkan dari ulasan unik lintas snapshot mingguan, sebagai pembanding belanja per tamu; pembanding BPS (TPK hotel, penjualan F&B per provinsi); estimasi kategori lain (ritel, jasa); perbandingan bebas antar bisnis dan antar wilayah; watch/alert perubahan; upload daftar customer + pencocokan; laporan data salah + koreksi ops; role Viewer/Manager; natural-language search; satu integrasi CRM; billing self-serve; pengelompokan multi-cabang; bahasa Inggris; error tracking |
| **F3 — Make it fast** | Skala & ketahanan | Server aplikasi kedua (HA), replika DB, object storage S3, cache, deploy tanpa downtime, optimasi untuk data > 5 juta tempat, mesin search terpisah hanya bila Postgres tidak cukup |

## 7. Data input

### 7.1 DB gmaps (D-31)

DB gmaps adalah sumber dataset kita. Export JSON-LD-nya (D-24) membuang `popular_times`, harga, distribusi bintang, dan riwayat snapshot, padahal semuanya dibutuhkan estimasi. Karena itu MesinProspek membaca isi DB gmaps itu sendiri: founder menyerahkan **kiriman mingguan ramping** dari DB gmaps (baris Indonesia, hanya kolom yang dipakai import, hanya baris baru sejak kiriman sebelumnya; kiriman pertama berisi seluruh riwayat), kiriman dimuat ke server kita, lalu MesinProspek membacanya lewat akun **read-only**, secara inkremental berdasarkan waktu scrape (D-35, D-37). Riwayat snapshot tersimpan di database MesinProspek sendiri. **Satu baris = satu snapshot satu tempat.** Satu tempat bisa punya ratusan snapshot (Signa Cafe: 171 snapshot, Februari–September 2026). Snapshot inilah yang membuat laju ulasan bisa diukur, bukan ditebak.

| Data | Dipakai untuk | Catatan dari data nyata (sampel Signa Cafe) |
|---|---|---|
| ID tempat: `cid`, `place_id` | **Kunci unik** `cid` (D-05) | Banyak baris per tempat |
| Nama, kategori utama, daftar kategori | Identitas, taksonomi, kelompok estimasi | Dwibahasa ("Restaurant" / "Restoran"). Daftar kategori membantu, mis. "Restoran" yang juga "Kafe" |
| Alamat (lengkap & per bagian), kode negara | Alamat tampilan, filter negara | Kode negara ISO ("ID") |
| Koordinat, zona waktu | Wilayah, jam lokal | Zona waktu IANA, mis. Bali = `Asia/Makassar` (WITA) |
| Telepon, website, akun sosial, tautan menu | Kontak, kehadiran digital, opportunity | — |
| Jumlah ulasan, rating | Reputasi, **laju ulasan** antar snapshot | Bisa turun sedikit (ulasan dihapus) |
| Distribusi bintang 1–5 | Penanda pola ulasan janggal | — |
| Harga per orang | Belanja per tamu | Terisi ±0,8% baris secara global (Bali jauh lebih tinggi, lihat cakupan di bawah). Format bervariasi: "Rp 75–200K", "Rp 100–225 rb", "Rp 250,000+", "$$" |
| Jam buka | Hari & jam buka, kelengkapan | Kunci hari berbahasa Indonesia, format "08.00–23.00", waktu lokal |
| `popular_times` | **Pola keramaian** 7 hari × 24 jam | Terisi ±13% baris secara global. Nilai 0–100 **relatif terhadap jam tersibuk tempat itu sendiri, bukan jumlah orang**. Jam tutup bernilai 0. Hotel jarang punya |
| Atribut tempat (`about`) | Profil dan fakta narasi AI | Mis. "Populer untuk: Sarapan, Makan siang, Makan malam", "Tipe pengunjung: Turis", opsi layanan, parkir, pembayaran |
| Pemilik (`owner`) | Penanda "listing diklaim pemilik" | — |
| Waktu scrape | Tanggal snapshot (`observed_at`) | — |
| Status, foto, deskripsi | Tidak dipakai di F1 | Status tidak andal (sampel berisi "Anjing boleh masuk") |
| Contoh ulasan (`user_reviews`) | F1: hanya **tanggal ulasan terbaru** dari snapshot yang di-scrape dengan urutan "Terbaru", untuk laju ulasan (E.1 metode B, D-38). Teks, nama, dan foto tidak diimpor; insight teks di F2 (§6) | Snapshot lama berisi ±8 ulasan urutan "paling relevan" (sampel Signa berisi ulasan 2024–2026), jadi tidak bisa dipakai untuk laju ulasan dan tidak mewakili sebaran rating. Scraper mengambil **20 ulasan terbaru** per tempat dan menandai snapshot itu dengan `reviews_sort = 'newest'` (Q-18); snapshot tanpa penanda ini tidak dipakai untuk metode B. Proporsi ulasan positif sudah lengkap dari distribusi bintang (semua ulasan). Isinya berguna untuk hal lain: sebutan harga per orang ("berdua di bawah 250 rb"), ukuran rombongan, dan tema pujian/keluhan. Memuat nama dan foto pengulas yang tidak kita butuhkan |
| Email, nama pemilik/person | Tidak ada → "Tidak diketahui" | — |

Nama kolom persis dan cara membacanya ditulis sekali di 04 (DRY). Snapshot yang berbeda bisa menyimpan atribut berbeda. Contoh: harga Signa hanya ada di sebagian snapshot, dan kategorinya kadang "Restaurant", kadang "Restoran". Aturan penggabungannya ada di BR-NORM-05.

Data mencakup **seluruh dunia**. F1 hanya memproses **Indonesia** (D-28); tempat negara lain dilewati dan dihitung di laporan.

**Cakupan terukur (Q-14, F&B).** Riwayat ≥ 30 hari ada di 44–52% tempat di Badung, Tabanan, dan Karangasem, tetapi hanya 0–0,3% di kota lain. Pola jam 42–51% di Bali dan 20–39% di kota lain; harga 40–53% di Bali dan 5–29% di kota lain. Menurut founder, seluruh Indonesia sudah ter-scrape dan scraper mengambil semua bisnis di area yang di-scrape, jadi metrik kawasan mencerminkan bisnis yang terdaftar di Google Maps. Estimasi muncul per tempat begitu riwayatnya ≥ 30 hari (BR-EST-01); saat diukur, itu baru terpenuhi di Bali. Ukur ulang Q-14 sebelum beta untuk melihat kota mana yang riwayatnya sudah cukup.

### 7.2 Data wilayah (GeoAPI, D-25)

`geoapi.mcpke.dev` menyediakan untuk Indonesia: provinsi, kab/kota, kecamatan, dan desa/kelurahan (81.912 area). Atribut tiap area: kode, nama, induk, luas, centroid, batas wilayah (GeoJSON & vector tiles), dan **populasi Kontur 2023 (hasil pemodelan)**. Data area di-sync ke database kita (batas 120 request/menit). Atribusi sumber wajib ditampilkan.

### 7.3 Parameter estimasi (dashboard ops)

Nilai yang tidak bisa diukur dari data diatur ops dari dashboard (D-29, D-32): review rate, faktor belanja, benchmark harga, dan benchmark hotel. Daftar dan nilai awalnya ada di Lampiran E §E.5. Setiap estimasi menyimpan parameter yang dipakainya.

## 8. Glosarium

| Istilah | Arti |
|---|---|
| Bisnis | Satu tempat usaha (1 ID tempat). Dalam konteks penjualan disebut **prospek** |
| Snapshot | Satu pengambilan data satu tempat pada satu waktu |
| Laju ulasan | Ulasan baru per tahun, diukur dari snapshot (Lampiran E §E.1) |
| Review rate | Persentase pengunjung yang menulis ulasan. Asumsi per kelompok estimasi, diatur ops |
| Kelompok estimasi | `makan` (restoran, warung, rumah makan), `kafe` (kafe, coffee shop, bar), `hotel` (hotel, resor, vila, penginapan), `lainnya` (belum didukung estimasi) |
| Pola keramaian | `popular_times`: nilai 0–100 per jam, relatif terhadap jam tersibuk tempat itu sendiri |
| Jam inti | Jam dengan keramaian ≥ 60% dari puncaknya (Lampiran E §E.4) |
| Analisis lokasi | Analisis satu titik + radius untuk keputusan cabang (Lampiran F) |
| Workspace | Akun pelanggan. Data workspace terisolasi |
| Offering, ICP | Apa yang dijual workspace, dan kriteria target penjualannya (dipakai PER-03) |
| Opportunity | Gap digital yang relevan dengan Offering, lengkap dengan alasan dan evidence |
| Claim type | Teramati (dari sumber) · Turunan (aturan) · Perkiraan (inferensi/estimasi) · Tidak diketahui |
| Unlock | Membuka nilai kontak sebuah bisnis untuk workspace, permanen |

---

## 9. Functional requirements (F1)

| ID | Requirement | Acceptance (ringkas) |
|---|---|---|
| **AUTH** | | |
| FR-AUTH-01 | Daftar/login via **Ory Kratos**: email + password, login Google, verifikasi email (D-20) | Tanpa verifikasi email, data tidak bisa diakses |
| FR-AUTH-02 | Workspace dibuat saat daftar. Admin bisa mengundang anggota, dibatasi seat plan (anggota aktif + undangan tertunda; tanpa plan = 1 seat). Role: **Admin** (kelola anggota, ICP, webhook, lihat usage) dan **Member** (pakai fitur) | Aksi di luar role → 403; undangan melebihi seat ditolak |
| FR-AUTH-03 | Akun internal ops wajib MFA (TOTP) | Login ops tanpa MFA ditolak |
| **DATA** | | |
| FR-DATA-01 | Import dari DB gmaps (read-only, D-31): job `import` harian + `msp tool import --country ID` manual. Inkremental berdasarkan waktu scrape; import pertama membaca seluruh riwayat. Tempat di-upsert berdasarkan ID tempat. Snapshot disimpan per tempat per hari untuk riwayat. Laporan: snapshot baru, tempat baru, berubah, dilewati (negara lain), ditolak (per alasan) | Import ulang rentang waktu yang sama tidak membuat duplikat. Kolom sumber yang hilang → import gagal dengan pesan jelas |
| FR-DATA-02 | Normalisasi sesuai BR-NORM: telepon, URL, kategori dwibahasa, jam buka, harga per orang, `popular_times`, distribusi rating, penggabungan snapshot | Unit test per aturan, termasuk semua format harga di §7.1 |
| FR-DATA-03 | Sync area GeoAPI (4 level, populasi, batas sederhana, atribusi) lewat `msp tool geo-sync` untuk negara aktif | Semua area level 1–4 Indonesia tersimpan |
| FR-DATA-04 | Penetapan wilayah: koordinat → desa/kelurahan (+ kecamatan, kab/kota, provinsi) dengan point-in-polygon di database | ≥ 98% bisnis ber-koordinat mendapat kelurahan |
| **EST — estimasi** | | |
| FR-EST-01 | Laju ulasan (Lampiran E §E.1): metode A dari riwayat snapshot, metode B dari tanggal ulasan terbaru. Metode yang dipakai dan jendela waktunya selalu tampil, plus grafik riwayat jumlah ulasan | Data kedua metode kurang → "Belum cukup data ulasan"; tidak pernah ditebak dari umur usaha |
| FR-EST-02 | Perkiraan kunjungan dan omzet untuk kelompok `makan` dan `kafe` (Lampiran E §E.2): rentang rendah–tengah–tinggi, tabel sensitivitas review rate, sumber harga, parameter, dan tanggal data | Fixture Signa Cafe menghasilkan angka §12 |
| FR-EST-03 | Perkiraan hotel (Lampiran E §E.3): kunjungan menginap, room-night, okupasi tahunan, dan omzet kamar | Parameter belum diisi ops → "Belum dapat diperkirakan"; okupasi > 100% → penanda "asumsi tidak konsisten", okupasi tidak ditampilkan |
| FR-EST-04 | Pola keramaian (Lampiran E §E.4): grafik rata-rata 7 hari dan per hari, blok waktu, jam puncak, jam inti, keramaian rata-rata, perkiraan tamu per jam bila kunjungan diketahui | Label "relatif terhadap jam tersibuk, bukan jumlah orang" selalu tampil |
| FR-EST-05 | Penanda kualitas data estimasi: ulasan bintang 5 ≥ 85% → "Proporsi bintang 5 sangat tinggi; bisa jadi lebih banyak tamu menulis ulasan dari biasanya, sehingga omzet cenderung di sisi bawah rentang" (pernyataan netral, tidak menuduh); laju ulasan terkini berbeda > 30% dari laju setahun → "Laju ulasan berubah" | Penanda tampil di profil dan analisis lokasi |
| FR-EST-06 | Estimasi dihitung ulang setelah setiap import dan setiap perubahan parameter. Setiap hasil menyimpan versi `est-X.Y.Z`, parameter yang dipakai, dan tanggal snapshot | Input + parameter + versi sama → hasil sama |
| **LOC — analisis lokasi** | | |
| FR-LOC-01 | Analisis lokasi (Lampiran F). Input: titik di peta atau alamat, radius, kelompok target, harga per orang rencana (opsional), dan bisnis pembanding (opsional: bisnis mana pun di data, mis. konsep yang ingin ditiru; tidak harus milik user). Analisis tetap lengkap tanpa pembanding. Output: tetangga dengan data cukup, pola keramaian tetangga vs pembanding, jam inti + saran jam buka, posisi harga, sumber pengunjung siang, dan tingkat keyakinan + saran cek lapangan. Analisis bisa disimpan dan dibuka ulang | Fixture Warung Kita menghasilkan saran jam buka 16.00–22.00 dan keyakinan "Rendah" (§12) |
| **INTL** | | |
| FR-INTL-01 | Setiap nilai yang tampil punya claim type dan sumber (BR-TRUST) | Evidence bisa dibuka dari profil |
| FR-INTL-02 | Deteksi opportunity sesuai katalog Lampiran B | Tiap opportunity punya alasan yang bisa dibaca |
| FR-INTL-03 | Metrik wilayah sesuai Lampiran C | Label "Populasi: perkiraan model Kontur 2023" selalu tampil |
| FR-INTL-04 | ICP fit dan priority score sesuai Lampiran A, dengan breakdown faktor dan versi model | Hasil sama untuk input + versi yang sama |
| FR-INTL-05 | Tombol "Relevan / Tidak relevan" pada opportunity dan pada kesimpulan analisis lokasi | Tersimpan per workspace |
| **ICP** | | |
| FR-ICP-01 | Onboarding: pilih tujuan → **Analisis lokasi** (langsung ke FR-LOC-01), **Kompetitor & omzet** (search + profil), atau **Cari prospek** (Offering → ICP dengan kriteria *wajib*/*preferensi* → hasil pertama) | Hasil pertama ≤ 5 menit sejak daftar |
| FR-ICP-02 | Workspace bisa menyimpan beberapa ICP. Tiap user memilih satu ICP aktif | Ganti ICP → skor dan urutan berubah |
| **SRCH / PROF / GEO** | | |
| FR-SRCH-01 | Search kata kunci (nama, kategori, alamat) + filter (kategori, kelompok estimasi, wilayah, rentang perkiraan omzet, punya pola keramaian, band skor, tipe opportunity, ada telepon/website/sosial, rating). Urut: skor, perkiraan kunjungan, perkiraan omzet, atau laju ulasan | Jumlah hasil konsisten dengan filter |
| FR-SRCH-02 | Simpan pencarian (sekaligus berfungsi sebagai segment dinamis) | Pencarian tersimpan bisa dibuka ulang |
| FR-PROF-01 | Profil bisnis: identitas, **perkiraan kunjungan & omzet** (atau okupasi untuk hotel), **pola keramaian**, riwayat ulasan, harga, jam buka, atribut tempat (populer untuk, tipe pengunjung, opsi layanan), penanda listing diklaim pemilik, skor + "Kenapa bisnis ini?" (bila ada ICP aktif), opportunity, kontak (ter-mask sampai unlock), kehadiran digital, konteks wilayah, evidence tiap nilai, tautan Google Maps, tombol "Analisis lokasi di sini" | Setiap nilai menampilkan claim type dan tanggal data |
| FR-GEO-01 | Peringkat wilayah (kecamatan dan kelurahan dalam kab/kota) untuk satu kelompok estimasi (Lampiran C): populasi, jumlah & densitas bisnis sejenis, kompetisi, permintaan terbukti, rentang harga. Bila ada ICP aktif, ditambah jumlah prospek HOT/WARM. Peta berwarna dari tiles GeoAPI + tabel, drilldown ke daftar bisnis dan ke analisis lokasi | Peta selalu punya padanan tabel |
| **ACT** | | |
| FR-ACT-01 | List, tag, catatan, dan pemilik (assign ke anggota) untuk bisnis. Kolom list bisa menampilkan estimasi, sehingga list juga berfungsi sebagai tabel perbandingan sederhana | — |
| FR-ACT-02 | Unlock kontak: 1 credit per bisnis, permanen per workspace. Yang di-mask hanya **nomor telepon**; website dan akun sosial selalu terlihat | Unlock ganda tidak menagih ulang |
| FR-ACT-03 | Export CSV/XLSX sebagai job async, maksimal 10.000 baris per job, termasuk kolom estimasi. Bila ada bisnis yang belum di-unlock, user ditawari **"Unlock & export"** dengan preview biaya | Export gagal bisa diulang tanpa biaya |
| FR-ACT-04 | Webhook: daftar URL HTTPS (ditolak bila mengarah ke IP privat/internal), payload ber-tanda tangan HMAC, retry otomatis, log pengiriman, kirim bisnis terpilih | Uji SSRF lulus |
| FR-ACT-05 | **Narasi AI** untuk profil bisnis dan untuk analisis lokasi, dengan struktur: masalah, data, cara membaca, kesimpulan, batasan. **Claude**, fallback ke API **OpenAI-compatible** (D-15). Provider, model utama, dan model fallback **diatur dari halaman ops** (API key tetap secret di server, D-29). Hanya dari fakta yang sudah dihitung; setiap kalimat merujuk fakta. Cache 7 hari. Ada kill switch | Narasi tanpa rujukan fakta ditolak lalu dibuat ulang sekali |
| **BILL** | | |
| FR-BILL-01 | Satu ledger dengan dua meter: `credit` dan `ai_credit` (grant, debit, refund), idempotent. **Plan** (nama, harga tampilan, seat, credit & AI credit bulanan) dikelola dari halaman ops (D-29); credit bulanan diberikan otomatis tiap awal siklus sesuai plan workspace; ops juga bisa memberi credit manual. Workspace baru mendapat **grant awal satu kali** (default 1.000 credit dan 0 AI credit, diatur dari halaman ops, D-36), supaya analisis lokasi bisa langsung dicoba. Pembayaran tetap manual di F1. Saldo terlihat di workspace | Saldo = jumlah ledger per meter |
| **OPS** | | |
| FR-OPS-01 | Halaman ops: daftar workspace + grant credit, pengaturan plan, AI, **parameter estimasi**, dan **grant awal workspace baru**; status import dari DB gmaps (snapshot terakhir, jeda, laporan); kesehatan data (jumlah bisnis, % terisi per field termasuk harga dan `popular_times`, % bisnis dengan ≥ 2 snapshot ≥ 30 hari, % terpetakan wilayah); feature flag (AI, export, webhook); suppression list | Hanya akun ops |
| FR-OPS-02 | Audit log append-only untuk login, undang anggota, unlock, export, webhook, grant credit, plan, pengaturan AI, parameter estimasi, flag, suppression | Tidak ada update/delete |
| **PRIV** | | |
| FR-PRIV-01 | Isolasi data per workspace di database (row-level security), termasuk analisis lokasi yang disimpan | Test lintas workspace lulus |
| FR-PRIV-02 | Nilai kontak di suppression list disembunyikan dari tampilan, export, dan webhook. Bisnis yang kontaknya ter-suppress tidak bisa di-unlock dan tidak ditagih | — |

---

## 10. Business rules

| ID | Aturan |
|---|---|
| BR-TRUST-01 | Claim type: **Teramati** (langsung dari snapshot), **Turunan** (aturan deterministik, mis. kategori, laju ulasan, jam inti), **Perkiraan** (memakai asumsi, mis. kunjungan, omzet, okupasi, "kemungkinan WhatsApp", opportunity), **Tidak diketahui** (field kosong) |
| BR-TRUST-02 | Freshness dari tanggal snapshot tiap nilai: Baru ≤ 180 hari, Menua ≤ 365 hari, Usang > 365 hari. Estimasi selalu menampilkan "Data per <tanggal snapshot terbaru>" |
| BR-TRUST-03 | "Tidak diketahui" tidak pernah ditampilkan sebagai kosong, "0", atau "—" |
| BR-NORM-01 | Telepon → E.164 sesuai negara tempat, nilai mentah tetap disimpan. Nomor seluler Indonesia (+628…) → "Kemungkinan WhatsApp" (Perkiraan) |
| BR-NORM-02 | URL website diklasifikasi: domain sendiri · profil sosial (instagram.com, facebook.com, tiktok.com) · link aggregator (linktr.ee, dll.) · link pesan (wa.me) · subdomain gratis (wixsite, blogspot, dll.) · kosong. Daftar pola disimpan sekali sebagai data (DRY) |
| BR-NORM-03 | Kategori sumber (Inggris atau Indonesia) dipetakan lewat **satu tabel mapping** ke taksonomi MSP dua level dan ke kelompok estimasi (`makan`, `kafe`, `hotel`, `lainnya`). Kategori utama yang menentukan; daftar kategori hanya dipakai bila kategori utama tidak dikenal (mis. "Restoran" yang juga "Kafe" tetap `makan`) |
| BR-NORM-04 | Harga per orang → rentang Rupiah `min`–`max`. Contoh: "Rp 25–50K" → 25.000–50.000, "Rp 100–225 rb" → 100.000–225.000, "Rp 1–25.000" → 1–25.000. Rentang terbuka "Rp 250,000+" → min 250.000 tanpa max. Simbol "$"/"$$" atau format tak dikenal → "Tidak diketahui" |
| BR-NORM-05 | Penggabungan snapshot: nilai terkini tiap field = nilai **tidak kosong terbaru** di antara snapshot, dengan tanggal snapshot-nya sendiri. Riwayat disimpan satu baris per tempat per hari (snapshot terakhir hari itu) untuk jumlah ulasan, rating, distribusi bintang, dan harga |
| BR-NORM-06 | Jam buka dan `popular_times` memakai **zona waktu lokal tempat** (kolom zona waktu sumber) dan ditampilkan dengan labelnya (WIB/WITA/WIT). Jam buka → jadwal per hari (Senin–Minggu). `popular_times` → matriks 7 × 24 nilai 0–100; hari dengan semua nilai 0 dianggap tutup. Data di luar 0–100 → ditolak per field |
| BR-EST-01 | Estimasi hanya dihitung bila data minimumnya ada (Lampiran E). Bila tidak, tampil "Belum dapat diperkirakan" beserta alasannya (mis. "riwayat < 30 hari dan ulasan terbaru < 10", "harga tidak diketahui"). Tidak ada fallback ke tebakan umur usaha |
| BR-EST-02 | Kunjungan dan omzet selalu ditampilkan sebagai rentang rendah–tengah–tinggi, dibulatkan (kunjungan ke ratusan, omzet ke Rp0,1 miliar bila ≥ Rp1 miliar, selain itu ke Rp10 juta) |
| BR-EST-03 | `popular_times` antar tempat hanya dibandingkan **bentuknya** (pola dan proporsi), bukan angka mentahnya |
| BR-SCORE-01 | Rumus di Lampiran A. Faktor yang nilainya tidak diketahui dikeluarkan dan bobotnya dibagi ulang. `coverage` = bobot faktor yang diketahui / total bobot |
| BR-SCORE-02 | Band: **HOT** ≥ 80, **WARM** 60–79, **COLD** < 60. Bila coverage < 0,6 atau kriteria ICP *wajib* tidak diketahui → maksimal WARM, dengan label "Data terbatas" |
| BR-SCORE-03 | Persentil memakai `percent_rank()` Postgres dalam kelompok pembanding (kategori × kab/kota) |
| BR-OPP-01 | Klaim "tidak ada X" hanya dibuat bila field X terisi ≥ 50% di kategori × kab/kota yang sama (minimal 20 bisnis). Bila tidak, opportunity tersebut tidak dibuat |
| BR-CRED-01 | Operasi berbayar ada di Lampiran D. Unlock kontak: 1 credit/bisnis, permanen. Export dan kirim ke webhook gratis; untuk bisnis yang belum di-unlock, unlock dijalankan dulu (dengan preview biaya), lalu export/kirim berjalan, dan bila gagal cukup diulang tanpa biaya. Narasi AI yang gagal dan analisis lokasi yang hasilnya "Tidak dapat dianalisis" tidak ditagih. Semua debit memakai idempotency key. Refund dilakukan manual oleh ops |
| BR-AI-01 | Ke LLM hanya dikirim fakta level bisnis dan hasil hitungan. Nomor telepon tidak dikirim |

## 11. Non-functional requirements

| ID | Requirement |
|---|---|
| NFR-01 | Search p95 ≤ 1 s, profil p95 ≤ 500 ms, analisis lokasi p95 ≤ 3 s, untuk ukuran data saat ini |
| NFR-02 | Export 10.000 baris ≤ 2 menit. Narasi AI ≤ 20 s. Import harian inkremental selesai sebelum jam kerja |
| NFR-03 | Ketersediaan F1 99% per bulan (satu server untuk semua, D-19). Naik ke 99,5% di F3 |
| NFR-04 | Backup: RPO ≤ 15 menit, RTO ≤ 8 jam (F1). Restore diuji sebelum rilis F1 |
| NFR-05 | HTTPS, isolasi workspace (RLS), secret tidak ada di repo, koneksi ke DB gmaps memakai akun read-only, pengecekan SSRF untuk webhook |
| NFR-06 | WCAG 2.2 AA dasar: kontras, keyboard, status tidak hanya dengan warna. Grafik keramaian punya padanan tabel |
| NFR-07 | Bahasa Indonesia (F1), format Rupiah/angka/tanggal id-ID. Waktu sistem Asia/Jakarta; jam buka dan pola keramaian dalam zona waktu lokal tempat (BR-NORM-06) |
| NFR-08 | Browser: 2 versi terakhir Chrome, Edge, Firefox, Safari. Desktop utama; profil dan list bisa dipakai di ponsel |
| NFR-09 | Test otomatis untuk logika inti (normalisasi, estimasi, analisis lokasi, skor, opportunity, credit, isolasi workspace). Coverage paket inti ≥ 80% |

## 12. Gerbang rilis F1

1. Alur end-to-end berjalan di staging dengan data nyata: import dari DB gmaps → wilayah → estimasi → search → profil → analisis lokasi → narasi AI → unlock → export/webhook.
2. **Sample case founder direproduksi** sebagai test otomatis (golden test) dengan data dari kasus aslinya:
   - **Signa Cafe:** snapshot 1.336 ulasan (10 Feb 2026) dan 1.507 ulasan (24 Sep 2026) → laju ≈ 276/tahun. Harga "Rp 100–225 rb", faktor belanja 0,8, review rate 2% / 1,5% / 1% → kunjungan ≈ 13.800 / 18.400 / 27.600 per tahun dan omzet ≈ **Rp1,8 / 2,4 / 3,6 miliar** per tahun. Bintang 5 = 87,5% → penanda FR-EST-05 tampil. Pola keramaian (data asli 7 hari): puncak rata-rata jam 20, jam inti 08–22 → "Ramai sepanjang hari"; puncak Selasa jam 13 (= 100).
   - **Warung Kita (Jl. Pulau Kawe):** pola KR dan Joglo dari kasus asli, dengan Joglo sebagai pembanding. Jam inti tetangga 17–21 → saran jam buka **16.00–22.00**. Jam inti pembanding mencakup 12–14 tetapi tetangga tidak. Satu tetangga sejenis → keyakinan **Rendah** + saran cek lapangan. Harga rencana Rp25–50 rb vs KR Rp50–100 rb → **lebih murah**.
3. Setiap nilai di profil dan analisis lokasi menampilkan claim type, sumber, dan tanggal data.
4. Test otomatis lulus, termasuk isolasi workspace, SSRF webhook, dan idempotensi credit.
5. Restore backup sudah dicoba sekali dan berhasil.
6. Founder mencoba onboarding: hasil pertama ≤ 5 menit.

## 13. Risiko

| Risiko | Mitigasi |
|---|---|
| Review rate tidak bisa diukur dari data dan sangat menentukan omzet (1% vs 2% = 2×) | Selalu rentang + tabel sensitivitas + parameter yang dipakai, tidak pernah disebut omzet pasti. Parameter memakai benchmark dan diatur dari dashboard. MesinProspek tidak punya data omzet nyata usaha dan tidak menjanjikannya |
| `popular_times` hanya ada di ±13% tempat dan harga di ±0,8% | Pola dan omzet tampil "Belum dapat diperkirakan" bila data kurang; harga memakai median wilayah atau benchmark berlabel (Lampiran E) |
| **Riwayat snapshot belum merata** (Q-14: ≥ 30 hari di 44–52% tempat di Bali, 0–0,3% di kota lain saat diukur) | Metode B (tanggal ulasan terbaru, D-38) mengukur laju dari satu snapshot, sehingga estimasi tersedia di seluruh Indonesia setelah scraper mulai mengambil 20 ulasan terbaru (Q-18). Sebelum itu, di luar Bali tampil "Belum cukup data ulasan". Ukur ulang cakupan sebelum beta. Fitur tidak diubah untuk menutupi data yang kurang |
| Metode B memakai jendela pendek untuk tempat ramai (mis. 20 ulasan dalam 3 minggu), sehingga terpengaruh musim atau promo | Metode A didahulukan bila riwayat ≥ 30 hari. Jendela metode B selalu ditampilkan; jendela < 30 hari diberi penanda "jendela pendek". Bila kedua metode tersedia, selisihnya ditampilkan sebagai cek silang |
| Skema DB gmaps berubah dan merusak import | Pemetaan kolom ditulis di satu tempat (04); import memeriksa kolom saat mulai dan gagal dengan alert yang jelas |
| Data basi (contoh: catatan lama tentang Pulau Kawe ternyata keliru) | Estimasi selalu dihitung dari snapshot terbaru dan menampilkan tanggal data; dihitung ulang setelah setiap import |
| Populasi adalah perkiraan model, bukan sensus | Label jelas di UI dan laporan |
| Satu server untuk production, staging, GitLab, CI, dan monitoring (D-19) | Batas CPU/RAM per container, runner CI 1 job sekaligus, uptime monitor eksternal, backup DB & GitLab off-site, rebuild dari skrip bootstrap (RTO 8 jam). Pisahkan GitLab/CI ke server lain bila contention terukur (F3) |
| Opportunity atau kesimpulan dirasa tidak relevan | Tombol feedback (FR-INTL-05), penyesuaian parameter |

---

## Lampiran A — Priority score `priority-2.0.0` (fitur prospek, PER-03)

Disederhanakan dari MVP Pack Rev 1.2 §9.

| Faktor | Bobot | Nilai 0–100 |
|---|---|---|
| ICP Fit | 30 | Rata-rata kecocokan kriteria ICP (cocok 1, sebagian 0,5, tidak 0). **Sebagian**: rating ≥ minimum − 0,3, jumlah ulasan ≥ separuh minimum, atau kategori satu induk dengan kategori ICP. Kategori "Lainnya" tidak pernah cocok. Kriteria *wajib* yang tidak cocok → bisnis tidak tampil |
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
| OPP-WEB-NONE | Website kosong (dengan syarat BR-OPP-01) | 80 | website_dev / local_seo, digital_ads |
| OPP-WEB-SOCIAL | Website berupa profil sosial, link aggregator, atau link pesan | 70 | website_dev / social_media |
| OPP-WEB-FREE | Website di subdomain gratis | 55 | website_dev / — |
| OPP-SOCIAL-NONE | Akun sosial kosong (dengan syarat BR-OPP-01) | 50 | social_media / digital_ads |
| OPP-LIST-INCOMPLETE | ≥ 2 dari {telepon, website, jam buka} kosong | 50 | local_seo / website_dev |
| OPP-REP-LOW-RATING | Rating < 4,0 dengan ≥ 20 ulasan | 70 | reputation_mgmt / local_seo |
| OPP-REP-FEW-REVIEWS | Ulasan < persentil 25 kategori × kab/kota (≥ 10 pembanding) | 55 | reputation_mgmt, local_seo / digital_ads |
| OPP-MKT-COMPETITIVE | Kompetisi kelurahan = Tinggi (Lampiran C) | 50 | digital_ads, local_seo / reputation_mgmt |

Contoh alasan: *"Website mengarah ke linktr.ee, bukan domain sendiri. 61% restoran di Kota Semarang memiliki website sendiri."*

## Lampiran C — Metrik wilayah `geo-3.0.0`

Dihitung per kelurahan dan kecamatan, per kelompok estimasi. Satuan densitas **per 1.000 penduduk**. Jumlah bisnis selalu dilabeli "bisnis yang terdaftar di Google Maps".

| Metrik | Rumus |
|---|---|
| Populasi, luas | Dari GeoAPI (Kontur 2023, perkiraan model) |
| Jumlah bisnis | Total dan per kelompok estimasi |
| Densitas | jumlah / populasi × 1.000 |
| Kompetisi | Persentil densitas kelompok di antara kelurahan dalam kab/kota yang sama. Rendah < 33, Sedang 33–66, Tinggi > 66 |
| Permintaan terbukti | Jumlah perkiraan kunjungan **tengah** bisnis sekelompok di area (hanya yang punya estimasi), plus rata-ratanya per bisnis. Dihitung bila ≥ 3 bisnis sekelompok punya estimasi; bila tidak, "Tidak diketahui" |
| Rentang harga | Median dan sebaran titik tengah harga bisnis sekelompok (bila ≥ 3 punya harga) |
| Data terbatas | Populasi < 500 atau jumlah bisnis < 5 → metrik tetap tampil, tetapi `market_score` tidak dihitung |
| `market_score` | 0,40·pct(kunjungan rata-rata per bisnis sekelompok) + 0,35·pct(populasi) + 0,25·(100 − kompetisi). Faktor yang tidak diketahui dikeluarkan dan bobotnya dibagi ulang (pola BR-SCORE-01) |
| Peringkat wilayah | Urut `market_score`. Bila ada ICP aktif, juga ditampilkan jumlah prospek HOT + WARM |

Kunjungan rata-rata per bisnis yang tinggi dengan kompetisi rendah berarti permintaan sudah terbukti tetapi belum jenuh. Untuk bisnis kelompok `lainnya` (mis. prospek PER-03), metrik wilayahnya dihitung dari semua bisnis di area.

## Lampiran D — Operasi yang memakai credit

| Operasi | Biaya |
|---|---|
| Search, profil (termasuk estimasi dan pola keramaian), peta, peringkat wilayah, membuka ulang analisis yang sudah disimpan, export & kirim webhook untuk bisnis yang sudah di-unlock | Gratis |
| Menjalankan analisis lokasi baru | 1 credit. **Tidak ditagih** bila hasilnya "Tidak dapat dianalisis" |
| Unlock kontak (termasuk "Unlock & export/kirim") | 1 credit per bisnis |
| Grant awal workspace baru | +1.000 credit satu kali (default, diatur ops; D-36) |
| Narasi AI | 1 AI credit (gratis bila diambil dari cache, tidak ditagih bila gagal) |

Nilai plan (harga, seat, kuota) diatur di halaman ops (FR-BILL-01). Hipotesis awal ada di [`10-bisnis.md`](10-bisnis.md) §3.

## Lampiran E — Estimasi `est-1.0.0`

Rantai: **snapshot → laju ulasan → kunjungan → omzet**. Pola keramaian memberi bentuk, bukan volume. Omzet dihitung langsung dari kunjungan, tidak lewat okupasi, karena kapasitas (kursi) tidak ada di data.

### E.1 Laju ulasan

Dua metode. **A didahulukan**; B dipakai bila A tidak memenuhi syarat. Bila keduanya tidak memenuhi syarat → "Belum cukup data ulasan".

**Metode A (riwayat snapshot):**
- Pakai snapshot terbaru dan snapshot tertua dalam 365 hari sebelumnya. Syarat: rentang ≥ 30 hari.
- `laju = max(0, ulasan_terbaru − ulasan_terlama) ÷ hari × 365`.

**Metode B (tanggal ulasan terbaru, D-38):**
- Sumber: tanggal N ulasan terbaru (scraper mengambil 20) dari snapshot terbaru bertanda `reviews_sort = 'newest'` (Q-18). Tanggal setelah waktu scrape diabaikan. Syarat: N ≥ 10.
- Bila ulasan ke-N lebih tua dari 365 hari: `laju = jumlah ulasan dalam 365 hari terakhir` (hitungan tepat).
- Selain itu: `laju = (N − 1) ÷ (waktu scrape − tanggal ulasan ke-N, dalam hari) × 365`.
- Jendela (tanggal ulasan ke-N sampai waktu scrape) selalu ditampilkan. Jendela < 30 hari → penanda "jendela pendek".
- Ketelitian bergantung pada N: simpangan relatif sekitar 1/√(N − 2), yaitu ±40% untuk N = 8 dan ±24% untuk N = 20. Karena itu scraper sebaiknya mengambil ≥ 20 ulasan.

Bila metode A dipakai dan metode B juga tersedia, laju B ditampilkan sebagai cek silang.
- **Laju terkini** dihitung dengan cara yang sama untuk 90 hari terakhir (syarat rentang ≥ 30 hari). Laju ini hanya pembanding dan dasar penanda FR-EST-05; estimasi memakai laju setahun.

### E.2 Kunjungan & omzet: kelompok `makan` dan `kafe`

| Langkah | Rumus |
|---|---|
| Kunjungan/tahun (rendah, tengah, tinggi) | laju ÷ review rate (atas, tengah, bawah). Default 2% / 1,5% / 1% |
| Tamu per hari buka | kunjungan ÷ (365 × hari buka per minggu ÷ 7). Jam buka tidak diketahui → 7 hari |
| Belanja per tamu | titik tengah harga per orang × faktor belanja (default 0,8). Rentang terbuka: titik tengah = min |
| Sumber harga, urut prioritas | (1) harga tempat itu sendiri, (2) median titik tengah harga bisnis sekelompok di kab/kota yang sama (≥ 10 bisnis), (3) benchmark ops per kelompok, (4) tidak diketahui → omzet tidak dihitung, kunjungan tetap tampil. Sumber yang dipakai selalu ditampilkan |
| Omzet/tahun (rendah, tengah, tinggi) | kunjungan (rendah, tengah, tinggi) × belanja per tamu |
| Sensitivitas | Tabel review rate (bawah, tengah, atas) → tamu per hari dan omzet per tahun |
| Catatan metode (selalu tampil) | Kunjungan dihitung dari ulasan Google Maps, sehingga pesanan lewat aplikasi pesan-antar umumnya tidak tercakup |

### E.3 Hotel

| Langkah | Rumus |
|---|---|
| Kunjungan menginap/tahun | laju ÷ review rate hotel (default 10%, dari sample case founder) |
| Room-night/tahun (rendah, tengah, tinggi) | kunjungan menginap × lama menginap rata-rata |
| Okupasi tahunan (rendah, tengah, tinggi), **Perkiraan** | room-night ÷ (jumlah kamar × 365). Jumlah kamar adalah asumsi per kelas dan selalu ditulis di samping angkanya. Nilai > 100% → penanda "asumsi tidak konsisten", okupasi tidak ditampilkan |
| Omzet kamar/tahun | room-night × ADR (tarif rata-rata per malam). Tidak termasuk F&B dan acara |

Lama menginap, jumlah kamar, dan ADR diambil dari benchmark ops per **kelas akomodasi** (hotel bintang 1–5 bila kategori menyebutnya, hotel umum, resor, vila, penginapan). Nilai yang belum diisi → hasil yang bergantung padanya tampil "Belum dapat diperkirakan".

### E.4 Pola keramaian

- **Rata-rata 7 hari** per jam, plus tampilan per hari.
- **Blok waktu:** Pagi 06–10 · Siang 11–14 · Sore 15–17 · Malam 18–21 · Larut 22–02. Nilai blok = jumlah nilai jam di blok itu.
- **Bentuk:** nilai jam ÷ jumlah nilai semua jam. Inilah yang dibandingkan antar tempat (BR-EST-03).
- Hari tutup (semua nilai 0) tidak ikut dirata-ratakan.
- **Jam puncak:** jam dengan nilai rata-rata tertinggi, juga per hari. Bila sama, jam paling awal.
- **Jam inti:** semua jam dengan nilai rata-rata ≥ 60% dari nilai puncak. Bisa lebih dari satu rentang, mis. 12–14 dan 17–20.
- **Pola datar:** jam inti mencakup ≥ 80% jam buka → label "Ramai sepanjang hari" (contoh: Signa Cafe, jam inti 08–22 dari jam buka 08–23).
- **Keramaian rata-rata:** rata-rata nilai selama jam buka ÷ 100, ditampilkan sebagai "rata-rata X% dari jam tersibuknya". Jam buka tidak diketahui → keramaian rata-rata dan label pola datar "Tidak diketahui".
- **Tamu per jam (Perkiraan):** tamu per hari buka (tengah) × nilai jam ÷ jumlah nilai hari itu.

### E.5 Parameter (dashboard ops)

| Parameter | Default awal |
|---|---|
| Review rate `makan`, `kafe` (bawah / tengah / atas) | 1% / 1,5% / 2% |
| Review rate `hotel` (bawah / tengah / atas) | 6,7% / 10% / 13,3% (tengah ± ⅓, sebaran relatif sama dengan F&B) |
| Faktor belanja | 0,8 |
| Benchmark harga per orang per kelompok | Kosong (diisi ops) |
| Hotel: lama menginap, jumlah kamar, ADR per kelas akomodasi | Kosong (diisi ops) |
| Penanda bintang 5 | ≥ 85% |
| Penanda laju berubah | Selisih > 30% |
| Metode B: minimum ulasan terbaru (N) | 10 |
| Metode B: batas jendela pendek | 30 hari |
| Ambang jam inti | 60% dari puncak |
| Ambang pola datar | 80% jam buka |

**Semua default di atas adalah asumsi awal, bukan angka bersumber.** Review rate dan faktor belanja diambil dari sample case founder; riset publik tidak menemukan angka review rate yang bisa dikutip. Ambang jam inti, pola datar, dan penanda bintang 5 dipilih agar cocok dengan dua sample case, sehingga berisiko terlalu pas dengan dua contoh itu. Ambang ditinjau ulang setelah distribusinya diukur di data nyata (Q-14), sebelum rilis F1. UI menyebut parameter ini "asumsi", bukan "benchmark".

Mengubah parameter memicu hitung ulang (FR-EST-06) dan tercatat di audit.

## Lampiran F — Analisis lokasi `loc-1.0.0`

| Bagian | Aturan |
|---|---|
| Input | Titik (klik peta, alamat, atau bisnis yang sudah ada). Radius default 500 m (100–2.000 m). Kelompok target (`makan` / `kafe` / `hotel`). Harga per orang rencana (opsional). Bisnis pembanding (opsional): bisnis mana pun di data, tidak harus milik user. Semua bagian di bawah tetap dihitung tanpa pembanding |
| Tetangga | Bisnis dalam radius, dikelompokkan: **sejenis** (kelompok sama), **F&B lain** (kelompok makan/kafe lainnya), dan **non-F&B**. Tetangga "data cukup" = ulasan > 0 dan punya `popular_times`. Tetangga lain tetap dihitung, tetapi tidak dipakai untuk pola. Setiap tetangga menampilkan tanggal datanya |
| Pola tetangga | Rata-rata nilai per jam tetangga sejenis yang datanya cukup. F&B lain ditampilkan terpisah sebagai konteks (mis. kafe yang ramai larut malam) |
| Jam inti & saran jam buka | Jam inti pola tetangga (Lampiran E §E.4). Saran jam buka = 1 jam sebelum jam inti pertama sampai 1 jam setelah awal jam inti terakhir |
| Pembanding | Pembanding tidak dihitung sebagai tetangga meskipun berada di dalam radius. Jam inti dan bentuk per blok waktu pembanding ditampilkan berdampingan dengan tetangga. Jam inti pembanding yang tidak ada di tetangga ditandai "tidak didukung tetangga" |
| Posisi harga | Titik tengah harga rencana vs median titik tengah tetangga sejenis: < 80% → lebih murah, > 120% → lebih mahal, selain itu setara |
| Sumber pengunjung siang | Jumlah bisnis non-F&B dalam radius per jenis (kesehatan, akomodasi, kantor/pendidikan, ritel) dan 3 nama terdekat |
| Keyakinan | Jumlah tetangga sejenis dengan data cukup: 0 → "Tidak dapat dianalisis" (F&B lain tetap ditampilkan), 1 → **Rendah** + saran cek lapangan (mis. hitung kendaraan parkir pada jam puncak dan jam sepi selama 2–3 hari), 2–4 → Sedang, ≥ 5 → Tinggi |
| Kesimpulan | Deterministik dari aturan di atas. Narasi AI opsional (FR-ACT-05) hanya menjelaskan hasil hitungan ini |
