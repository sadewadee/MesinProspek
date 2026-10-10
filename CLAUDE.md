# CLAUDE.md — MesinProspek

Panduan wajib untuk setiap sesi Claude dan setiap agent yang bekerja di repo ini. Bila ada konflik, file ini dan `docs/01-prd.md` menang atas kebiasaan atau asumsi lain.

## Goal produk (jangan melenceng)

MesinProspek membantu penjual B2B menemukan **bisnis lokal Indonesia** yang paling layak dihubungi, lalu menjelaskan **kenapa** (opportunity + evidence) dan memungkinkan **tindak lanjut** (unlock kontak, export, webhook, AI brief).

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
| Requirement, aturan bisnis, rumus skor/opportunity/wilayah, credit | `docs/01-prd.md` |
| Arsitektur, data model, nama modul/tabel | `docs/04-technical-design.md` |
| Kontrak API | `contracts/openapi.yaml` (kode server & client TS di-generate) |
| Skema database | migrasi goose di `apps/api/db/migrations` |
| Warna, tipografi, spasi | `docs/02-design-system/tokens/design-tokens.json` |
| Nama perintah, service, env var, antrean | `docs/README.md` § Kosakata teknis bersama |
| Keputusan | `docs/00-analisis-dokumen-awal.md` §7 (register D-xx) |

Logika yang sama (mis. klasifikasi URL, normalisasi telepon, perhitungan skor) ditulis **sekali** dan dipakai ulang.

### 3. Make it work → make it right → make it fast
- **F1 — make it work:** alur end-to-end berjalan dan benar untuk kasus utama. Scope = `docs/01-prd.md` §9, tidak lebih.
- **F2 — make it right:** kelengkapan, kualitas, kenyamanan (daftar di PRD §6).
- **F3 — make it fast:** skala & ketahanan. Optimasi **hanya** setelah ada pengukuran yang menunjukkan masalah.
- Jangan mengerjakan fitur F2/F3 saat F1 belum lolos gerbang rilis (PRD §12), kecuali founder meminta.

## Batasan tetap

- Crawler **PAUSED**. Input = dataset JSON-LD (kunci `google_place_id`) + data wilayah GeoAPI.
- **Unknown lebih baik daripada mengarang.** Field yang tidak ada tampil "Tidak diketahui". Jangan mengisi nilai tebakan tanpa label "Perkiraan".
- Isu hukum (PDP, lisensi data, retensi) ditangani **tim legal**. Jangan membuat keputusan atau dokumen hukum.
- CI/CD memakai **GitLab CE self-hosted**, tanpa GitHub Actions. Hosting di **dedicated server** di Indonesia.
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
