# 08 — Engineering Standards (lean)

## MesinProspek — Aturan coding esensial untuk Go & Next.js

| Field | Value |
|---|---|
| Versi | 0.2 (lean). Menggantikan v0.1 |
| Tanggal | 2026-10-10 |
| Prinsip | **KISS**, **DRY**, **make it work → make it right → make it fast** (D-23) |
| Basis | `README.md` (stack F1, kosakata, struktur monorepo), `01-prd.md` v0.2 (BR-TRUST, BR-SCORE, BR-CRED, BR-AI-01, NFR-05/06/07/09), `02-design-system/` (tokens) |
| Pembaca | Agent AI pembangun (maks. 8 paralel), orchestrator (Claude), founder |

> **Cara membaca.** Satu aturan = satu baris dengan ID `STD-<AREA>-nn`. Kolom **Ditegakkan oleh**:
> **lint** = gagal di `make lint` · **test** = gagal di `make test` · **CI** = job pipeline GitLab gagal · **review** = dicek orchestrator di MR.
> Aturan yang sudah dicek tool tidak perlu dibahas lagi di review. Dokumen ini tidak menyalin aturan bisnis (PRD), nama (README), atau desain (04). Dokumen ini hanya merujuk ke sana.
>
> **Bahasa:** identifier, komentar kode, log, dan commit memakai bahasa Inggris. Teks UI dan `docs/` memakai bahasa Indonesia.

---

## 0. Perintah standar (lokal = CI)

Ada satu `Makefile` di root. Lokal dan CI memanggil target yang sama (pipeline di `06-ci-cd.md`).

| Target | Isi |
|---|---|
| `make generate` | oapi-codegen (server Go), openapi-typescript (`packages/api-client`), `sqlc generate`, build tokens |
| `make lint` | golangci-lint, `sqlc vet`, squawk (migrasi baru), oasdiff breaking vs `main`, ESLint + `tsc --noEmit`, cek hex/px (grep), gitleaks |
| `make test` | `go test -race ./...` dengan Postgres 16 + PostGIS asli (Compose `local` / service CI), gate coverage, test web (05) |

CI menambahkan satu cek: jalankan `make generate`, lalu `git diff --exit-code`. Tujuannya memastikan kode hasil generate tidak basi.

Konfigurasi lint:
- **Go** (`apps/api/.golangci.yml`): set `standard` + `errorlint`, `forbidigo`, `gochecknoglobals`, `gochecknoinits`, `nolintlint`. Formatter: `gofmt` + `goimports`.
- **Web**: ESLint flat config di `apps/web`.
- **Pengecualian** (`//nolint:<linter>`, `eslint-disable-next-line <rule>`) wajib menyebut nama rule dan alasannya.

---

## 1. Umum (GEN)

| ID | Aturan | Ditegakkan oleh |
|---|---|---|
| STD-GEN-01 | **KISS.** Dependency baru (Go module, paket npm runtime) atau komponen/service/lapisan baru wajib punya alasan tertulis di MR: masalah nyatanya apa, dan kenapa stdlib atau stack README tidak cukup | review (diff `go.mod` / `package.json`) |
| STD-GEN-02 | **DRY.** Satu sumber kebenaran: aturan → PRD, API → `contracts/openapi.yaml`, skema → migrasi, styling → tokens, nama → README. Logika yang sama ditulis sekali. File hasil generate tidak diedit tangan | CI (generate tanpa diff), review |
| STD-GEN-03 | **Make it work dulu.** Tulis kode paling sederhana yang benar. Optimasi, cache, atau konkurensi hanya untuk masalah terukur (benchmark, `EXPLAIN ANALYZE`, metrik) yang dicantumkan di MR. Abstraksi baru dibuat setelah ada ≥ 2 pemakai nyata | review |
| STD-GEN-04 | **Unknown tidak pernah jadi `0`, `""`, atau `false`.** Bentuknya `NULL` di DB, pointer/`pgtype` di Go, `null` di API, dan "Tidak diketahui" di UI. Faktor skor yang Unknown dikeluarkan dari perhitungan (BR-TRUST-03, BR-SCORE-01) | test, review |
| STD-GEN-05 | **Tanpa PII di log.** Nomor telepon dan nilai kontak tidak pernah masuk log, pesan error, atau prompt LLM (BR-AI-01). Log memakai ID | test (handler `slog` meredaksi kunci kontak), review |

## 2. Go backend (GO)

| ID | Aturan | Ditegakkan oleh |
|---|---|---|
| STD-GO-01 | Kode lolos `gofmt`/`goimports` dan golangci-lint (konfigurasi §0) | lint |
| STD-GO-02 | Error dibungkus `fmt.Errorf("op: %w", err)` dan dicek dengan `errors.Is`/`errors.As`. Pemetaan ke problem+json hanya terjadi di tepi HTTP (satu fungsi mapper), tidak di kode domain | lint (`errorlint`, `errcheck`), review |
| STD-GO-03 | `ctx context.Context` adalah parameter pertama setiap fungsi yang melakukan I/O atau bisa berjalan lama. `ctx` tidak disimpan di struct | review |
| STD-GO-04 | Tanpa state global yang bisa berubah dan tanpa `init()` yang punya efek samping. Dependency masuk lewat constructor `NewX(...)` dan dirakit di `cmd/msp` | lint (`gochecknoglobals`, `gochecknoinits`) |
| STD-GO-05 | Konfigurasi hanya dari env `MSP_*` (README). Env dibaca sekali ke struct `Config` dan divalidasi saat start. Nilai salah → proses berhenti dengan pesan jelas. `os.Getenv` hanya boleh di paket config | lint (`forbidigo`), test |
| STD-GO-06 | Log hanya lewat `log/slog` dengan handler JSON dari `cmd/msp`. `fmt.Print*` dan `log.*` dilarang. Sertakan `request_id`, `workspace_id`, atau `job_id` bila ada | lint (`forbidigo`) |
| STD-GO-07 | Logika yang harus deterministik (skor, opportunity, metrik wilayah, freshness BR-TRUST-02, cache AI) menerima waktu sebagai parameter (`asOf time.Time`) atau lewat clock yang di-inject. Logika ini tidak memanggil `time.Now()` (FR-INTL-04) | test (input + versi sama → output identik), review |
| STD-GO-08 | Paket dibuat datar: `apps/api/internal/<modul>` (nama modul di 04). Tidak ada paket `util`, `common`, atau `helpers`. Sub-paket hanya dibuat dengan alasan di MR | review |

## 3. Database (DB)

| ID | Aturan | Ditegakkan oleh |
|---|---|---|
| STD-DB-01 | Skema hanya diubah lewat migrasi goose bernomor urut (`goose create -s`) di `apps/api/db/migrations`. Migrasi yang sudah ada di `main` tidak pernah diedit; perbaikan = migrasi baru. Perubahan harus backward-compatible dengan versi app yang sedang jalan: tambah dulu, hapus di rilis berikutnya | CI (migrasi lama tidak berubah, nomor unik), lint (squawk) |
| STD-DB-02 | Query hanya ditulis di `apps/api/db/queries/*.sql` lewat sqlc. Tanpa string SQL di kode Go, tanpa ORM, tanpa query builder | CI (generate tanpa diff), lint (`sqlc vet`), review |
| STD-DB-03 | Query data workspace hanya berjalan di dalam helper transaksi workspace (04). Helper ini menjalankan `set_config('app.workspace_id', …, true)` untuk RLS. Role DB runtime bukan owner dan tanpa `BYPASSRLS` (NFR-05) | test (isolasi lintas workspace, FR-PRIV-01) |
| STD-DB-04 | Uang dan credit disimpan sebagai integer (`bigint`, satuan terkecil) di DB, Go (`int64`), dan API. Float dilarang | review |
| STD-DB-05 | Waktu disimpan sebagai `timestamptz` (UTC). Konversi ke Asia/Jakarta hanya saat ditampilkan (STD-FE-05) | lint (squawk `prefer-timestamptz`) |
| STD-DB-06 | Job River harus idempotent (unique key `UniqueOpts`, aman dijalankan ulang). Job di-enqueue dengan `InsertTx` di transaksi yang sama dengan perubahan datanya | test (job dijalankan 2× → hasil sama), review |

## 4. API & kontrak (API)

| ID | Aturan | Ditegakkan oleh |
|---|---|---|
| STD-API-01 | OpenAPI-first: ubah `contracts/openapi.yaml` lebih dulu, lalu jalankan `make generate`. Handler mengimplementasikan interface hasil oapi-codegen. Endpoint di luar spec dilarang | review |
| STD-API-02 | Kode hasil generate (server Go, `packages/api-client`, sqlc) di-commit dan tidak diedit tangan | CI (`make generate` + `git diff --exit-code`) |
| STD-API-03 | Error memakai `application/problem+json` (RFC 9457) dengan `code` stabil (snake_case) yang tercantum di spec. Client bercabang berdasarkan `code`, tidak berdasarkan teks pesan | test (kontrak), review |
| STD-API-04 | Setiap POST berbayar (unlock, export, AI brief) wajib membawa header `Idempotency-Key`. Key yang sama → respons sama, tanpa debit ganda (BR-CRED-01) | test (idempotensi credit, gerbang rilis PRD §12) |
| STD-API-05 | Perubahan breaking hanya lewat versi path baru (`/api/v2/…`). `/api/v1` hanya boleh berubah secara kompatibel (tambah endpoint atau field opsional) | lint (oasdiff breaking vs `main`) |

## 5. Frontend Next.js (FE)

| ID | Aturan | Ditegakkan oleh |
|---|---|---|
| STD-FE-01 | Styling hanya lewat token (CSS variable `--msp-*`) dan preset Tailwind dari `packages/design-tokens`. Dilarang hex/rgb/px mentah dan arbitrary value Tailwind (`[…]`) | lint (grep hex/px/`[…]` di `apps/web`) |
| STD-FE-02 | Pakai Server Component secara default. `"use client"` hanya untuk interaktivitas (state, event, browser API) dan ditaruh sedalam mungkin di pohon komponen | review |
| STD-FE-03 | Data API hanya diambil lewat client hasil generate (`packages/api-client`). Di client component, client ini dipanggil lewat TanStack Query. Tanpa `fetch` manual ke API | lint (ESLint `no-restricted-globals: fetch` di luar wrapper) |
| STD-FE-04 | Form memakai react-hook-form + zod (`zodResolver`). Error ditampilkan per field dalam bahasa Indonesia. Error server (problem+json) dipetakan ke field | review |
| STD-FE-05 | Angka, Rupiah, dan tanggal diformat hanya lewat satu helper `Intl` (`id-ID`, `Asia/Jakarta`, NFR-07) | lint (ESLint `no-restricted-properties`: `toLocaleString`, `toFixed` di luar helper) |
| STD-FE-06 | Status tidak pernah ditunjukkan hanya dengan warna: selalu ikon + label teks (NFR-06, 02 §1). Prop `label` wajib di komponen badge/status. Unknown tampil sebagai "Tidak diketahui" | TypeScript (prop wajib), review |

## 6. Testing (TEST)

| ID | Aturan | Ditegakkan oleh |
|---|---|---|
| STD-TEST-01 | Paket logika (normalisasi, skor, opportunity, wilayah, credit) memakai table-driven test (`[]struct{…}` + `t.Run`). Kasus Unknown wajib ikut diuji | review |
| STD-TEST-02 | Setiap bug fix menyertakan test yang gagal sebelum fix diterapkan | review |
| STD-TEST-03 | Test integrasi (query, RLS, migrasi, job) memakai Postgres 16 + PostGIS asli, bukan mock DB | CI |
| STD-TEST-04 | Test tidak memanggil API eksternal (GeoAPI, Claude, OpenAI-compatible). Gunakan fixture di `testdata/` + `httptest.Server`; base URL diambil dari config | CI (job test tanpa API key provider), review |
| STD-TEST-05 | Coverage paket inti NFR-09 (normalisasi, skor, opportunity, credit, isolasi workspace) ≥ 80% | CI (gate coverage) |

## 7. Git & merge request (GIT)

| ID | Aturan | Ditegakkan oleh |
|---|---|---|
| STD-GIT-01 | MR kecil: satu WP atau satu perubahan, ±400 baris diubah (tidak termasuk kode generate dan fixture). Lebih besar dari itu → pecah | review |
| STD-GIT-02 | Merge memakai squash, sehingga judul MR = pesan commit. Format Conventional Commits dengan WP sebagai scope bila ada (`feat(WP-14): …`, `fix`, `test`, `docs`, `refactor`, `chore`) | CI (regex judul MR) |
| STD-GIT-03 | Merge hanya bila pipeline hijau. Test flaky diperbaiki, tidak di-retry sampai lolos | CI (setting GitLab "Pipelines must succeed") |
| STD-GIT-04 | Deskripsi MR menyebut ID `WP-` dan ID PRD (`FR-`/`BR-`/`NFR-`) yang dikerjakan, plus alasan bila ada dependency/komponen baru (STD-GEN-01) | CI (regex deskripsi MR), review |

## 8. Agent AI (AGENT)

| ID | Aturan | Ditegakkan oleh |
|---|---|---|
| STD-AGENT-01 | Kerjakan hanya scope WP (07). File bersama (`contracts/openapi.yaml`, migrasi, tokens, `go.mod`, `package.json`) hanya diubah bila WP menyebutnya. Bila tidak disebut, minta lewat orchestrator | review (diff vs scope WP) |
| STD-AGENT-02 | Sebelum menulis kode, baca ID PRD dan bagian 04 yang dirujuk WP, plus dokumen ini | review (rujukan di MR, STD-GIT-04) |
| STD-AGENT-03 | Untuk logika, tulis atau sesuaikan test lebih dulu (merah → hijau), baru tulis kode | review (test di MR yang sama), CI (coverage) |
| STD-AGENT-04 | Jalankan `make generate lint test` sebelum membuka MR. Jangan mematikan lint atau test (`//nolint`, `t.Skip`, `eslint-disable`) hanya agar MR lolos | CI, lint (`nolintlint`), review |
| STD-AGENT-05 | Bila perilaku, kontrak, atau skema berubah, perbarui dokumen terkait (spec, 03, 04, 09) di MR yang sama. Aturan bisnis PRD hanya diubah lewat orchestrator (persetujuan founder) | review |
| STD-AGENT-06 | Jangan commit secret atau data pribadi nyata. Pakai `.env.example` dengan nilai dummy dan fixture sintetis | CI (gitleaks) |
| STD-AGENT-07 | Requirement ambigu atau saling bertentangan → berhenti dan tanya orchestrator dengan opsi konkret. Jangan menebak. Jawabannya dicatat sebagai `Q-`/`D-` | review |

---

## 9. Template deskripsi MR

Simpan di `.gitlab/merge_request_templates/Default.md`:

```markdown
## Apa & kenapa
WP-xx · FR-/BR-/NFR-xx · 04 §x

## Dependency/komponen baru (hapus bila tidak ada)
Masalah nyata: … · Kenapa stack yang ada tidak cukup: …

## Checklist
- [ ] Test baru/diubah (bug fix → test yang gagal sebelum fix)
- [ ] `make generate lint test` hijau di lokal
- [ ] Dokumen terkait diperbarui (bila perilaku/kontrak/skema berubah)
```

## 10. Sengaja tidak diatur di F1

Item berikut belum diatur: aturan batas modul via `depguard`, Spectral, Schemathesis, mutation testing, Storybook, visual regression, ADR formal, CODEOWNERS, dan tracing OpenTelemetry. Masing-masing baru ditambahkan bila ada masalah nyata (STD-GEN-01).

Perubahan dokumen ini dilakukan lewat MR dengan persetujuan orchestrator. Aturan baru harus satu baris dan sebisa mungkin ditegakkan oleh tool.
