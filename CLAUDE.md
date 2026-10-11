# CLAUDE.md — MesinProspek

Panduan wajib untuk setiap sesi Claude dan setiap agent yang bekerja di repo ini. Bila ada konflik, file ini dan `docs/01-prd.md` menang atas kebiasaan atau asumsi lain.

## Goal produk (jangan melenceng)

MesinProspek adalah **market intelligence untuk bisnis lokal Indonesia** (D-30). Dari data Google Maps yang diambil berulang dan data wilayah, produk memperkirakan **kunjungan, omzet, dan okupasi** usaha, menunjukkan **pola keramaian**, dan membantu memilih **lokasi, jam buka, dan harga** cabang baru. Setiap angka disertai bukti, asumsi, rentang, dan tanggal data. Pengguna utama: pemilik usaha dan investor. Fitur prospek (unlock kontak, export, webhook) tetap ada untuk penjual B2B.

Alur inti: **temukan → pahami → prioritaskan → tindak lanjuti**. Pekerjaan yang tidak memperkuat alur ini **tidak dikerjakan** tanpa persetujuan founder.

## Prinsip WAJIB

### 1. KISS — Keep It Simple
- Pilih solusi paling sederhana yang memenuhi requirement. Jangan membangun untuk kebutuhan hipotetis.
- Komponen, layanan, library, atau lapisan abstraksi baru hanya boleh ditambahkan bila ada **masalah nyata yang terukur**. Alasannya wajib ditulis di deskripsi MR.
- Sengaja **tidak dipakai di F1**: Redis, PgBouncer, object storage internal, server kedua/HA, blue/green/canary, Ansible, SOPS, image signing/SBOM, mutation testing, mesin search terpisah, microservice. Butuh salah satunya? Tanya dulu.

### 2. DRY — Don't Repeat Yourself
Satu sumber kebenaran per hal. Rujuk, jangan salin:

| Hal | Sumber kebenaran |
|---|---|
| Requirement, aturan bisnis, rumus estimasi/analisis lokasi/skor/opportunity/wilayah, credit | `docs/01-prd.md` |
| Harga & kuota plan, unit economics | `docs/10-bisnis.md` |
| Arsitektur, data model, nama modul/tabel | `docs/04-technical-design.md` |
| Kontrak API | `contracts/openapi.yaml` (kode server & client TS di-generate) |
| Skema database | migrasi goose di `apps/api/db/migrations` |
| Warna, tipografi, spasi | `docs/02-design-system/tokens/design-tokens.json` |
| Nama perintah, service, env var, antrean | `docs/README.md` § Kosakata teknis bersama |
| Keputusan | `docs/00-analisis-dokumen-awal.md` §7 (register D-xx) |

Logika yang sama (mis. klasifikasi URL, normalisasi telepon/harga, laju ulasan, jam inti, perhitungan skor) ditulis **sekali** dan dipakai ulang.

### 3. Make it work → make it right → make it fast
- **F1 — make it work:** alur end-to-end berjalan dan benar untuk kasus utama. Scope = `docs/01-prd.md` §9, tidak lebih.
- **F2 — make it right:** kelengkapan, kualitas, kenyamanan (daftar di PRD §6).
- **F3 — make it fast:** skala & ketahanan. Optimasi **hanya** setelah ada pengukuran yang menunjukkan masalah.
- Jangan mengerjakan fitur F2/F3 saat F1 belum lolos gerbang rilis (PRD §12), kecuali founder meminta.

## Batasan tetap

- Kita **tidak scraping**; crawler website **PAUSED**. Input = **DB gmaps**: dump dari founder di-restore ke server kita dan dibaca read-only (global, snapshot berulang per tempat, D-31, D-35) + data wilayah GeoAPI. **F1 melayani seluruh Indonesia** (D-34); negara disimpan per record agar negara lain bisa diaktifkan kelak tanpa ubah desain.
- **Unknown lebih baik daripada mengarang.** Field yang tidak ada tampil "Tidak diketahui". Jangan mengisi nilai tebakan tanpa label "Perkiraan".
- **Hanya janjikan yang bisa kita sediakan.** Fitur hanya dibangun dari data yang kita punya (DB gmaps, GeoAPI, benchmark publik). Data internal usaha (omzet nyata, transaksi, kapasitas) tidak tersedia: jangan dijadikan fitur, metrik, atau syarat.
- **Ukur, jangan tebak.** Estimasi mengikuti PRD Lampiran E/F: yang bisa diukur dari snapshot diukur; asumsi hanya untuk yang tidak terukur, diatur dari dashboard ops, dan selalu ditampilkan sebagai rentang + asumsi + tanggal data.
- Isu hukum (PDP, lisensi data, retensi) ditangani **tim legal**. Jangan membuat keputusan atau dokumen hukum.
- CI/CD memakai **GitLab CE self-hosted**, tanpa GitHub Actions. Hosting di **satu dedicated server** di Indonesia (production, staging, GitLab, CI, monitoring).
- Nilai yang sering berubah (model AI, plan, harga, kuota, parameter estimasi) **dikonfigurasi dari dashboard ops**, tidak di-hardcode.
- Stack: Go (backend, satu binary `msp`) + Next.js (frontend) + PostgreSQL/PostGIS + Ory Kratos + River. AI: Claude, fallback ke OpenAI-compatible. Detail di `docs/README.md`.
- Jangan pernah commit secret. Jangan kirim nomor telepon ke LLM.

## Cara kerja agent

1. Kerjakan hanya scope paket kerja (WP) yang ditugaskan (`docs/07-delivery-plan.md`). Jangan mengubah file di luar scope.
2. Baca bagian PRD dan `docs/04` yang dirujuk WP sebelum menulis kode.
3. Untuk logika, tulis atau sesuaikan test lebih dulu. Setiap bug fix disertai test.
4. Jalankan lint dan test sebelum membuka merge request. MR kecil, Conventional Commits, CI harus hijau.
5. Bila perilaku atau kontrak berubah, perbarui dokumen terkait di MR yang sama (DRY).
6. Requirement ambigu? Tanya orchestrator/founder, jangan menebak.

## Bahasa

Dokumen dan teks UI: Bahasa Indonesia (istilah teknis boleh bahasa Inggris). Identifier dan komentar kode: bahasa Inggris.
