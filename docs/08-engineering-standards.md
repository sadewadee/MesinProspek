# 08 — Engineering Standards

## MesinProspek — Standar Kode & Konvensi (Go backend, Next.js frontend)

| Field | Value |
|---|---|
| Dokumen | Engineering standards: standar kode, struktur repo, konvensi, dan checklist review |
| Versi | 0.1 (Draft) |
| Tanggal | 2026-10-10 |
| Basis | `README.md` (konvensi teknis & struktur monorepo), `01-prd.md` (BR-TRUST, BR-SCORE, BR-PRIV, BR-CRED, NFR), `02-design-system/`, `03-ux-spec.md` (§6–§8), `04-technical-design.md`, `05-test-strategy.md`, `06-ci-cd.md` |
| Keputusan acuan | D-11 (modular monolith), D-18 (GitLab CE), D-19 (dedicated server), D-20 (IdP, Open), **D-21 (Go + Next.js, Accepted)** |
| Audience | Engineering (backend, frontend, data), QA, DevOps, reviewer MR |
| Status | Aturan di dokumen ini adalah default kerja. Library/tool di luar README dan 04 §3.1 berlabel *Proposed* (Lampiran A). |

> **Cara membaca.** Setiap aturan punya ID `STD-<AREA>-nn` (AREA ∈ GEN, GO, DB, JOB, API, CONTRACT, SEC, OBS, DATA, TEST, FE, A11Y, I18N, GIT, DOC). Format: **aturan** · *alasan* · **penegakan**. Penegakan: **lint** (golangci-lint, ESLint, stylelint, Spectral, squawk, sqlc vet), **CI** (job/quality gate `QG-nn` di 06 §8), **test** (`TC-…` di 05), **review** (checklist §13.2). Penanda *(usulan)* = tooling belum ada di 06 dan perlu ditambahkan. "Wajib/dilarang" mengikat; pengecualian ditulis sebagai `//nolint:<linter> // <alasan> (#issue)` atau lewat ADR (§14).

**Daftar isi:** 1. Prinsip umum · 2. Struktur repo & modul · 3. Go · 4. Database · 5. Jobs (River) · 6. API & kontrak · 7. Keamanan · 8. Observability · 9. Data & intelligence · 10. Testing · 11. Frontend Next.js · 12. Aksesibilitas & i18n · 13. Git, MR & code review · 14. Dokumentasi & ADR · 15. Lampiran

---

## 1. Prinsip umum

| Prinsip | Artinya di kode |
|---|---|
| **Sederhana dulu** | Stdlib dan pola yang "membosankan" lebih baik daripada abstraksi pintar. Abstraksi baru dibuat setelah ada ≥ 2 pemakai nyata. |
| **Eksplisit** | Dependency di-inject lewat constructor, transaksi terlihat di service, konteks tenant terlihat di signature. Tidak ada global, `init()`, atau reflection untuk wiring. |
| **Deterministik** | Input + versi + `as_of` sama → output identik (BR-SCORE-13, NFR-DATA-01). Jam sistem, randomness, dan urutan iterasi map tidak bocor ke hasil. |
| **Unknown over invented** | "Tidak tahu" adalah nilai first-class di setiap layer, bukan `0`, `""`, atau `false` (BR-TRUST-06, 04 §1.2). |
| **Secure by default** | Default = tolak: RLS fail-closed, permission deny, egress deny, CSP ketat. Data pribadi diminimalkan sejak tipe datanya (BR-PRIV-01). |
| **Docs-as-code** | Kontrak, skema, keputusan, dan standar hidup di repo dan berubah dalam MR yang sama dengan kodenya. |

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-GEN-01 | Kode baru memakai stdlib atau library yang sudah tercantum di README, 04 §3.1, atau Lampiran A. Dependency runtime baru (Go module atau paket npm non-dev) butuh entri Lampiran A atau ADR (§14) di MR yang sama. | Supply chain (06 §15), beban upgrade, lisensi | review CODEOWNERS, QG-20 (license), Renovate |
| STD-GEN-02 | Wiring dependency hanya di composition root (`internal/app`, `apps/web/src/app/**/layout.tsx` + providers). Tanpa DI container, service locator, atau `init()` yang punya efek samping. | Alur kontrol terbaca dari kode | lint `gochecknoinits`, review |
| STD-GEN-03 | Fungsi yang menghasilkan skor, metrik, rationale, hash, export, atau golden output tidak bergantung pada jam sistem, urutan iterasi map, randomness, locale/zona waktu proses, atau urutan selesai goroutine. | BR-SCORE-13, NFR-DATA-01 | lint `forbidigo` (STD-GO-30), test golden 2× hash identik (QG-24) |
| STD-GEN-04 | Ketidaktahuan dimodelkan eksplisit di DB, Go, API, UI, dan export. Kegagalan sementara menghasilkan UNKNOWN, bukan nilai negatif palsu (BR-VER-02). | BR-TRUST-06 | review, test §9 |
| STD-GEN-05 | Tidak ada kegagalan diam-diam: error tidak ditelan, setiap operasi gagal berakhir dengan status ber-kode yang dapat di-retry (PRD §12.2). `_ =` untuk nilai error dilarang kecuali disertai komentar alasan. | DoD PRD §12.2 | lint `errcheck`, review |
| STD-GEN-06 | Perubahan perilaku, kontrak, skema, atau konfigurasi memperbarui `docs/`/`contracts/` di MR yang sama. | Dokumen tidak basi | MR template, QG-22 |
| STD-GEN-07 | Identifier, komentar kode, pesan log, dan pesan commit berbahasa Inggris. Teks yang dilihat user hanya lewat katalog i18n (id-ID default, en-US). Dokumen `docs/` berbahasa Indonesia. Nama domain mengikuti glosarium PRD §7 dan enum 04 §5.7. | Satu kosakata di DB, Go, OpenAPI, TS; tooling | review, lint i18n (STD-I18N-02) |

---

## 2. Struktur repo & modul

### 2.1 Monorepo

Mengikuti README §Struktur monorepo, diperinci:

```
MesinProspek/
├─ apps/
│  ├─ api/                    Go module (§2.2)
│  └─ web/                    Next.js App Router (§11.1)
├─ packages/
│  ├─ design-tokens/          build dari docs/02-design-system/tokens → CSS vars, preset Tailwind, TS
│  ├─ ui/                     komponen React di atas Radix (satu-satunya pengimpor @radix-ui/*)
│  ├─ api-client/             hasil openapi-typescript (di-commit) + factory openapi-fetch
│  ├─ eslint-config/          flat config bersama + rule kustom `msp/*`
│  └─ tsconfig/               tsconfig dasar (strict)
├─ contracts/{openapi,events,input}/   sumber kebenaran kontrak (§6)
├─ data/{taxonomy,reference,golden}/   data berversi; golden set aman-repo (05 §4.4)
├─ infra/{docker,ansible}/
├─ scripts/{ci,release}/      skrip yang dipanggil CI (YAML CI hanya orkestrasi, 06 §1.4)
├─ tests/{e2e,load,security,ops}/
├─ docs/  (+ docs/adr/ untuk ADR panjang, §14)
├─ .gitlab/{CODEOWNERS,merge_request_templates/}
└─ .gitlab-ci.yml  .editorconfig  turbo.json  pnpm-workspace.yaml  .nvmrc
```

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-GEN-08 | Satu Go module (`apps/api`) dan satu pnpm workspace. Module/paket workspace baru butuh ADR. | Satu graf dependency, satu rilis | review |
| STD-GEN-09 | Kode hasil generate (oapi-codegen, sqlc, openapi-typescript, design tokens) di-commit, berheader `Code generated … DO NOT EDIT.`, dan tidak pernah diedit manual. | Diff kontrak terlihat di MR | CI QG-11, QG-14, QG-15 (`git diff --exit-code`) |
| STD-GEN-10 | Toolchain di-pin: `go`/`toolchain` di `go.mod`; directive `tool` untuk oapi-codegen, govulncheck, gotestsum, gocover-cobertura; sqlc dan golangci-lint di image `ci-tools` (06 §1.5); Node 24 via `.nvmrc` + `engines`; pnpm via `packageManager`. Instalasi beku: `pnpm install --frozen-lockfile`, `go mod verify`, `go mod tidy -diff` kosong. | Build reproducible, parity lokal–CI (06 §3.1) | CI QG-03 |

### 2.2 Layout Go (`apps/api`)

Mengikuti 04 §3.1, dengan pembagian layer di dalam `internal/` tiap modul:

```
apps/api/
├─ go.mod  .golangci.yml  sqlc.yaml  oapi-codegen.yaml
├─ cmd/
│  ├─ msp/                    main → internal/app; subcommand: api | worker | migrate | tool <perintah>
│  └─ msp-synth/              generator data sintetis, dev-only, tidak masuk image (05 §4.3)
├─ db/
│  ├─ migrations/             goose SQL berurutan (§4.3)
│  ├─ queries/<modul>/        SQL sumber sqlc
│  └─ schema.sql              dump skema ter-commit (QG-11)
├─ internal/
│  ├─ app/                    composition root: config, pool pgx, client River, router strict-server, DAG
│  ├─ api/                    hasil oapi-codegen (DO NOT EDIT)
│  ├─ db/<modul>db/           hasil sqlc per schema (DO NOT EDIT; hanya diimpor store modul pemilik)
│  ├─ platform/               auth, tenancy, rbac, audit, flags, events, jobs, idempotency, netguard, ids,
│  │                          problem, pagination, i18n, telemetry (+ clock, apperr, db, buildinfo: ditetapkan dokumen ini)
│  └─ <modul>/                identity · ingest · intelligence · geo · query · action · ai · usage · ops
│     ├─ service.go           FACADE: interface Service, DTO publik, New(Deps) — satu-satunya API modul
│     ├─ http.go              handler strict-server untuk operasi OpenAPI milik modul
│     ├─ jobs.go              worker River milik modul (registrasi ke River client)
│     ├─ doc.go               tanggung jawab, tabel milik, event yang di-emit/dikonsumsi
│     └─ internal/
│        ├─ domain/           tipe & aturan murni; sub-domain murni boleh jadi paket sendiri
│        │                    (mis. intelligence/internal/scoring, identity/internal/normalize, identity/internal/er,
│        │                     geo/internal/geocalc)
│        ├─ service/          implementasi Service: authz, transaksi, orkestrasi, event, audit
│        ├─ jobargs/          args job River (dipakai service untuk InsertTx dan jobs.go; mencegah siklus impor)
│        └─ store/            repository di atas internal/db/<modul>db: row↔domain, pemetaan error pgx
└─ tools/                     checker CI berbahasa Go (gen-depguard, checkrls, …)
```

### 2.3 Layer di dalam modul

| Paket | Isi | Boleh mengimpor | Dilarang |
|---|---|---|---|
| `domain` dan sub-domain murni (`scoring`, `normalize`, `er`, `opportunity`, `geocalc`) | Tipe, invariant, fungsi murni, aturan BR | stdlib, `platform/{ids,clock,apperr}` | pgx, `internal/db`, `net/http`, River, I/O, `time.Now` |
| `service` | Use case: cek permission & aturan resource, transaksi, orkestrasi, `events.Emit`, `audit.Record` | domain, store, facade modul di bawahnya, platform | `net/http`, `internal/api` |
| `store` | Repository: memanggil sqlc, memetakan row↔domain dan error pgx | domain, `internal/db/<modul>db`, pgx, `platform/db` | service, facade modul lain |
| `http.go` (paket facade) | Implementasi operasi strict-server: DTO↔domain, panggil `Service` | `internal/api`, facade sendiri, `platform/problem` | pgx, store, logika bisnis |
| `jobs.go` (paket facade) + `jobargs` | Worker River: terima args dari `jobargs`, panggil `Service` | `platform/jobs`, `jobargs`, facade sendiri | store langsung, `net/http` |

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-GO-01 | Modul lain hanya mengimpor paket akar modul (`internal/<modul>`). Kode di `internal/<modul>/internal/…` tidak dapat diimpor dari luar modul. | Batas modul dijamin compiler (ADR-01) | compiler Go |
| STD-GO-02 | Urutan lapisan 04 §3.3: `app` > `ops` > `action` \| `ai` > `query` > `intelligence` > `geo` > `identity` > `ingest` > `usage` > `platform`. Modul hanya mengimpor facade modul **di bawahnya**; `action` dan `ai` tidak saling impor. | Graf dependency asiklik | lint `depguard`, rule dibangkitkan `tools/gen-depguard` dari tabel layer (04 §3.3) |
| STD-GO-03 | Kebutuhan bawah→atas (mis. merge di `identity` memicu rekey di `action`) lewat event (`events.Emit` → job River), bukan import atau callback yang didaftarkan global. | Menjaga asiklik, at-least-once | review |
| STD-GO-04 | Modul hanya menulis ke schema miliknya (04 §3.2). Membaca data modul lain lewat facade atau view publik `<schema>.v_*_public` (04 §3.3 butir 4). Paket `internal/db/<modul>db` hanya diimpor `store` modul pemilik. | Kepemilikan data jelas | lint `depguard`; CI `tools/check-sql-ownership` *(usulan)* |
| STD-GO-05 | Layer mengikuti tabel §2.3. Paket domain murni tidak melakukan I/O dan tidak membaca jam. | Testable, deterministik | lint `depguard` rule `pure` (Lampiran B), `forbidigo` |
| STD-GO-06 | Metode facade yang harus ikut transaksi pemanggil diberi sufiks `Tx` dan menerima `pgx.Tx` (mis. `usage.Service.HoldTx(ctx, tx, …)`). Modul tidak membuka transaksi baru di dalam transaksi milik modul lain. | Unlock + ledger + audit atomik (04 §2.3b) | review |
| STD-GO-07 | Tidak ada direktori `pkg/`. Kode bersama berada di `internal/platform/<paket>`. Nama paket `util`, `common`, `helpers`, `misc`, `base`, `shared` dilarang. | Tidak ada konsumen eksternal; paket "tong sampah" tumbuh tanpa batas | review, lint `revive` (`var-naming`/package name) |
| STD-GO-08 | `cmd/msp/main.go` hanya mem-parse subcommand (stdlib `flag`) lalu memanggil `internal/app`. Perintah ops = `msp tool <nama>` (04 §3.1, mis. `score-reproduce`). | Satu binary, satu image (06 P-2) | review |

---

## 3. Go

### 3.1 Format, import, naming, desain paket

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-GO-09 | `gofmt` + `goimports` dengan tiga grup import: stdlib, pihak ketiga, lalu internal (prefix module). | Diff minimal | lint formatter, QG-03 |
| STD-GO-10 | Konfigurasi golangci-lint tunggal `apps/api/.golangci.yml` (Lampiran B). `//nolint` wajib menyebut linter + alasan. | Pengecualian terlihat dan dapat diaudit | lint `nolintlint` |
| STD-GO-11 | Nama paket huruf kecil, satu kata, kata benda, tanpa stutter (`identity.Service`, bukan `identity.IdentityService`). Inisialisme konsisten huruf besar: `ID`, `URL`, `HTTP`, `ICP`, `JSON` (`CompanyID`, `ICPVersion`). | Idiom Go | lint `staticcheck` ST1003, `revive` |
| STD-GO-12 | Sentinel error `ErrX`, tipe error `XError`, constructor `NewX`, boolean `IsX`/`HasX`. Interface kecil didefinisikan di sisi pemakai; satu metode → akhiran *-er*. Receiver pendek dan konsisten per tipe. | Idiom Go | lint `errname`, `recvcheck`, review |
| STD-GO-13 | Satu konsep = satu nama di DB, Go, OpenAPI, dan TS (`Establishment`, bukan `Branch`; `ContactPoint`, bukan `Contact`). Label UI boleh berbeda (03 §7.2), kode tidak. | Traceability lintas layer | review |
| STD-GO-14 | Paket berorientasi domain (`normalize`, `ledger`), bukan teknis. Surface exported minimal; setiap identifier exported di facade dan `platform` punya doc comment. | API modul jelas | lint `revive` (`exported`), review |
| STD-GO-15 | Constructor menerima dependency eksplisit (struct `Deps` bila > 3 parameter) dan memvalidasinya. Panic di constructor hanya untuk kesalahan wiring yang terdeteksi saat start. | Gagal cepat saat start, bukan saat request | review |

### 3.2 Error

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-GO-16 | Error di-wrap dengan `fmt.Errorf("<operasi>: %w", err)`: huruf kecil, tanpa titik, tanpa "failed to". Pemeriksaan memakai `errors.Is`/`errors.As` (atau `errors.AsType` di Go 1.26), tidak pernah membandingkan string. | Rantai error utuh | lint `errorlint`, `staticcheck` ST1005 |
| STD-GO-17 | Error domain = `*apperr.Error{Code, Detail, Meta, Err}`. `Code` diambil dari katalog 04 §11.2; kode baru ditambahkan ke katalog di MR yang sama. `Detail` aman dibaca user, `Err` adalah penyebab internal yang tidak pernah diserialisasi. | Client menangani berdasarkan `code` (ADR-11) | review, test contract |
| STD-GO-18 | `store` memetakan `pgx.ErrNoRows` → `store.ErrNotFound` dan `*pgconn.PgError` (23505 unique, 23503 FK, 40001 serialization, 55P03 lock timeout, 57014 statement timeout) → error ber-kode. Tipe pgx tidak keluar dari `store`. | Domain tidak bergantung driver | lint `depguard`, review |
| STD-GO-19 | Pemetaan error → HTTP hanya di `platform/problem` (RFC 9457): `Code` → status, `type`, `title` terlokalisasi, extension `Meta`. Error tanpa kode → 500 `INTERNAL_ERROR` generik (04 §11.2) dengan `correlation_id`. `err.Error()` internal, stack trace, dan SQL tidak pernah dikirim ke client. | Tidak membocorkan internal (NFR-SEC-01) | test contract (Schemathesis), review |
| STD-GO-20 | Error di-log **satu kali** di boundary (middleware HTTP, worker River, `main`). "Log lalu return" dilarang. Panic hanya untuk bug programmer; middleware HTTP dan worker memasang recover yang mengubah panic menjadi 500/job error dan melaporkannya ke GlitchTip. | Log tidak ganda, crash tidak menjatuhkan proses | review |

```go
// BAIK — konteks operasi, rantai utuh, error domain ber-kode
job, err := s.store.GetExportJob(ctx, tx, id)
if errors.Is(err, store.ErrNotFound) {
	return Job{}, apperr.NotFound("export_job", id.String())
}
if err != nil {
	return Job{}, fmt.Errorf("get export job %s: %w", id, err)
}

// BURUK — logger global, log + return, rantai putus, detail internal bocor ke client
if err != nil {
	log.Println(err)
	return Job{}, errors.New("db error: " + err.Error())
}
```

### 3.3 Context, state, konfigurasi

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-GO-21 | `ctx context.Context` adalah parameter pertama setiap fungsi yang melakukan atau memicu I/O atau bisa berjalan lama. Context tidak disimpan di struct. `context.Background()`/`TODO()` hanya di `main`, `internal/app`, dan test (`t.Context()`). | Cancellation & tracing menyebar | lint `containedctx`, `contextcheck`, `noctx`, `forbidigo` |
| STD-GO-22 | Cancellation dihormati: loop batch memeriksa `ctx.Err()` per iterasi/chunk; setiap dependency punya timeout eksplisit (DB `statement_timeout`, HTTP keluar 10 s, LLM 30 s; 04 §10.8, §12.5, §13.6). | Tidak ada kerja yatim setelah client pergi | review, test |
| STD-GO-23 | Value di context hanya untuk data lintas-cutting request-scoped (principal, workspace, correlation_id, trace), dengan key unexported dan accessor bertipe (`auth.PrincipalFrom(ctx)`). Parameter bisnis tidak diselundupkan lewat context. | Signature jujur | review |
| STD-GO-24 | Tidak ada state global yang dapat berubah: logger global (`slog.Default`, `slog.Info`), `http.DefaultClient`, singleton, atau var paket yang diubah setelah start. Yang diizinkan: konstanta, sentinel error, tabel lookup literal read-only, `regexp.MustCompile`, `//go:embed`. | Test paralel aman, dependency eksplisit | lint `gochecknoglobals`, `sloglint` (`no-global`), `forbidigo` |
| STD-GO-25 | Konfigurasi 12-factor: semua variabel didefinisikan di 04 §18.1 dan di-parse **sekali** di `internal/app/config.go` ke struct bertipe (`time.Duration`, `*url.URL`, enum), lalu divalidasi saat start. Nilai wajib hilang atau invalid → exit non-zero dengan pesan yang menyebut nama variabel, tanpa mencetak nilai secret. Semua variabel berprefiks `MSP_` (kecuali standar `OTEL_*`). Secret hanya dibaca dari `MSP_<NAMA>_FILE` (file Compose secrets di tmpfs hasil dekripsi SOPS, 06 §12.3); nilai secret tidak pernah dikirim sebagai env biasa. Variabel non-`_FILE` untuk secret hanya diterima saat `MSP_ENV` = `local`/`ci` (nilai dummy); di staging/production → exit non-zero. Parser memakai stdlib (`os.LookupEnv`, `strconv`, `time.ParseDuration`). | Gagal cepat; secret tidak tampil di `docker inspect` | lint `forbidigo` (`os.Getenv`/`os.LookupEnv` hanya di `internal/app`), test config |
| STD-GO-26 | Konfigurasi domain berversi (model skor, katalog opportunity, taksonomi, daftar referensi, prompt) adalah file di repo yang di-`//go:embed` atau dibaca dari `data/`, bukan variabel env (04 §18.1). `.env.example` memuat semua variabel dengan nilai dummy aman. | Reproducibility (NFR-DATA-01) | review, QG-23 |

### 3.4 Konkurensi

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-GO-27 | Setiap goroutine punya pemilik, batas, dan jalur berhenti yang terikat ctx. Gunakan `errgroup.WithContext` + `SetLimit(n)`, `sync.WaitGroup.Go`, atau worker pool berbatas. `go func()` fire-and-forget di handler dilarang; kerja async = job River (§5). | Tidak ada goroutine leak atau kerja hilang saat deploy | review, `goleak` di test |
| STD-GO-28 | Semua test Go berjalan dengan `-race`. Paket yang memulai goroutine memakai `goleak.VerifyTestMain`. State bersama dijaga mutex (komentar menyebut field yang dijaga) atau dimiliki satu goroutine; `sync/atomic` tidak dipakai untuk invariant multi-field. | Data race = bug S2 | CI QG-05 |
| STD-GO-29 | Graceful shutdown: `signal.NotifyContext(SIGTERM)` → `/readyz` false → `http.Server.Shutdown` (≤ 25 s, drain blue/green 06 §6.3) → `river.Client.Stop` lalu `StopAndCancel` setelah timeout → tutup pool. `http.Server` selalu mengisi `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`. | Deploy tanpa 5xx (TC-OPS-015) | lint `gosec` G112, review |

### 3.5 Waktu, angka, ID, generics

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-GO-30 | Waktu "sekarang" hanya dari `clock.Clock` yang di-inject (`platform/clock`). `time.Now`, `time.Since`, `time.Until` dilarang di luar `platform/clock`; test memakai `clock.Fake`. | Determinisme, test waktu tanpa sleep | lint `forbidigo` |
| STD-GO-31 | Semua waktu disimpan dan dikirim dalam UTC (`timestamptz`, RFC 3339 `Z`); konversi ke zona user hanya di UI (NFR-L10N-01). Container berjalan dengan `TZ=UTC`, tetapi kode tetap memanggil `.UTC()` di boundary. Perbandingan memakai `Equal`/`Before`/`After`, bukan `==`. | Bebas zona waktu host | review |
| STD-GO-32 | Komputasi yang bergantung waktu (freshness, kedaluwarsa, stale, recency) menerima `asOf time.Time` sebagai parameter dan mencatatnya di output (`as_of`, `calculated_at`; BR-SCORE-13, 04 §7.2). | Skor reproducible | review, test TC-INTL-015 |
| STD-GO-33 | Credit, ai_credit, unit usage, rupiah, byte, dan hitungan memakai integer (`int64`). Biaya AI memakai `int64` micro-USD (`cost_micro_usd`, 04 §13.6). `float` dilarang untuk uang, credit, dan ledger. Confidence = integer 0–100. | Rekonsiliasi 100% (RG-4) | review, test properti ledger TC-BILL-015 |
| STD-GO-34 | ID: UUIDv7 (`uuid.NewV7`) di DB; di API dan log berupa string berprefiks gaya TypeID (prefix 04 §5.1 + 26 karakter base32 Crockford). Setiap entity punya tipe ID sendiri (`ids.CompanyID`) sehingga ID tidak bisa tertukar saat kompilasi. Parsing ID dari luar memvalidasi prefix. Prefix baru ditambahkan ke daftar 04 §5.1 di MR yang sama. Codec `platform/ids` diuji dengan test vector spesifikasi TypeID. | ADR-14, ID stabil & terbaca | test, review |
| STD-GO-35 | Generics untuk container dan helper type-safe (`Claim[T]`, `Page[T]`, `ids.ID[P]`), bukan untuk abstraksi bisnis: tidak ada `Repository[T]` atau service generik. Tipe parameter yang hanya punya satu instansiasi berarti generik tidak perlu. | Kode tetap terbaca | review |

```go
// BAIK — waktu di-inject, asOf eksplisit, ID bertipe
func (s *Service) RefreshFreshness(ctx context.Context, ws ids.WorkspaceID, est ids.EstablishmentID) error {
	asOf := s.clock.Now().UTC()
	return s.tenancy.WithTx(ctx, ws, func(tx pgx.Tx) error {
		c, err := s.store.ContactPoints(ctx, tx, est)
		if err != nil {
			return fmt.Errorf("load contact points: %w", err)
		}
		return s.store.SaveFreshness(ctx, tx, est, domain.Freshness(c, asOf)) // fungsi murni
	})
}

// BURUK — jam global di dalam logika, ID mentah yang bisa tertukar
func freshness(lastVerified time.Time, companyID, estID uuid.UUID) Class {
	if time.Since(lastVerified) > 90*24*time.Hour { /* … */ }
}
```

### 3.6 Performa

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-GO-36 | Optimasi berbasis bukti. Hot path (normalizer, kernel skor, builder SQL search, stream export) punya benchmark `BenchmarkX` dengan `b.Loop()`. MR yang mengubah hot path melampirkan output `benchstat` sebelum/sesudah. | NFR-PERF-05/07 | review |
| STD-GO-37 | Data besar diproses secara streaming (cursor server-side, `iter.Seq`, `pgx.CopyFrom`) dan per batch berukuran tetap (04 §6.1), tidak dimuat seluruhnya ke memori. Slice di loop batch dialokasikan di awal (prealokasi). | Memori dedicated server terbatas | review, test perf (05 §3.16) |
| STD-GO-38 | pprof hanya di listener admin internal (bind ke jaringan internal, tidak di-route Traefik). Handler didaftarkan eksplisit ke mux admin; import side-effect `net/http/pprof` dilarang. | Tidak membuka endpoint debug ke publik | lint `gosec` G108, review |

---

## 4. Database

### 4.1 sqlc & query

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-DB-01 | Semua SQL statis ditulis di `db/queries/<modul>/<aggregate>.sql` (satu direktori per modul, satu file per aggregate) dan di-generate sqlc ke `internal/db/<modul>db`. String SQL di kode Go dilarang, kecuali di builder search `query` (STD-DB-04) dan helper `platform` (`set_config`, advisory lock, `SET LOCAL`). | SQL direview sebagai kode (ADR-26), bebas injeksi | lint `gosec` G201/G202, review |
| STD-DB-02 | Nama query `-- name: <Verb><Object>[By<Key>] :one\|:many\|:exec\|:execrows\|:copyfrom`, dengan verb ∈ Get, List, Count, Exists, Create, Update, Upsert, Delete, Lock. Parameter bernama (`@workspace_id`, `sqlc.narg(x)`), kolom eksplisit (tanpa `SELECT *`). | Nama query menjadi nama span DB (§8) | `sqlc vet` rule kustom *(usulan)*, review |
| STD-DB-03 | Query pada tabel tenant selalu memuat `workspace_id = @workspace_id` secara eksplisit, walaupun RLS aktif (lapis kedua, 04 §5.8). | Isolasi ganda (NFR-SEC-03) | test TC-AUD-007/008, `sqlc vet` *(usulan)* |
| STD-DB-04 | SQL dinamis (kombinasi filter FR-SRCH-002) hanya ada di satu paket builder modul `query`: nama kolom dan arah sort diambil dari allowlist enum, nilai selalu placeholder. Builder diuji dengan fuzz, golden SQL, dan cek `EXPLAIN` tanpa seq scan pada tabel besar (05 §3.16). | Fleksibilitas filter tanpa injeksi | test, CODEOWNERS |
| STD-DB-05 | Query yang dipakai endpoint sinkron punya index pendukung dan `statement_timeout` (`SET LOCAL`; search 3 s, 04 §10.8). Query baru di jalur search/profil melampirkan `EXPLAIN (ANALYZE, BUFFERS)` pada dataset `medium` di deskripsi MR. | NFR-PERF-01/02 | review, nightly query plan check (05 §3.16) |

```sql
-- db/queries/action/list_item.sql
-- name: ListListItemsPage :many
SELECT li.list_id, li.company_id, li.added_at, li.added_by
FROM action.list_item AS li
WHERE li.workspace_id = @workspace_id                       -- lapis kedua di atas RLS
  AND li.list_id = @list_id
  AND (li.added_at, li.company_id) < (@after_added_at::timestamptz, @after_company_id::uuid)  -- keyset
ORDER BY li.added_at DESC, li.company_id DESC
LIMIT @page_size;
```

### 4.2 Transaksi, RLS, locking

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-DB-06 | Setiap transaksi tenant dibuka lewat `WithTx(ctx, ws, fn)` milik `*tenancy.Tenancy` yang di-inject (04 §3.2: `tenancy.WithTx`), yang menjalankan `SELECT set_config('app.workspace_id', $1, true)` (setara `SET LOCAL`, menerima bind parameter, aman untuk PgBouncer transaction mode; 04 §5.8). `SET` tingkat sesi dilarang. | RLS fail-closed, tidak bocor antar koneksi pool | test TC-AUD-008, QG-09 |
| STD-DB-07 | Koneksi runtime memakai role `msp_app`, `msp_pipeline`, `msp_ops`, atau `msp_readonly_replica` sesuai 04 §5.8. `msp_owner`/superuser hanya untuk `msp migrate`. Satu `pgxpool.Pool` per role/target, dibuat di `internal/app`. | Least privilege, RLS tidak di-bypass | Ansible + test isolasi (06 M-7) |
| STD-DB-08 | Transaksi pendek: tidak ada panggilan HTTP, LLM, DNS, atau object storage di dalam transaksi. Search/profil memakai transaksi `ReadOnly` di replica. Isolasi default READ COMMITTED; SERIALIZABLE hanya bersama retry eksplisit pada SQLSTATE 40001. | Lock singkat, tidak ada koneksi tertahan | review |
| STD-DB-09 | Saldo credit: lock baris `usage.credit_balance … FOR UPDATE` dengan urutan tetap `(workspace_id, meter)`; ledger append-only (INSERT saja; 04 §14.5). Advisory lock hanya `pg_advisory_xact_lock` dengan key dari registry `platform/db/locks.go`. Advisory lock tingkat sesi hanya di koneksi langsung (migrasi). | Tidak overspend, tidak deadlock | test TC-BILL-003/004 |
| STD-DB-10 | Mode statement pgx untuk pool lewat PgBouncer diselaraskan dengan transaction mode (PgBouncer ≥ 1.21 dengan `max_prepared_statements`, atau `QueryExecModeDescribeExec`). Pilihan final ditetapkan saat bootstrap dan diuji di CI. Pool langsung (River, migrate; STD-JOB-11) memakai mode default pgx. `rows.Close()` dan `rows.Err()` selalu diperiksa. | Error "prepared statement already exists" di produksi | lint `rowserrcheck`, `sqlclosecheck`, test integrasi |

```go
// platform/tenancy — satu-satunya pintu transaksi tenant
func (t *Tenancy) WithTx(ctx context.Context, ws ids.WorkspaceID, fn func(pgx.Tx) error) error {
	return pgx.BeginTxFunc(ctx, t.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.workspace_id', $1, true)`, ws.UUID().String()); err != nil {
			return fmt.Errorf("set tenant context: %w", err)
		}
		return fn(tx)
	})
}
```

### 4.3 Migrasi (goose)

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-DB-11 | File `db/migrations/<NNNNN>_<verb>_<schema>_<objek>.sql` (goose SQL, versi berurutan `goose -s`, tanpa out-of-order; 06 M-9). Hanya SQL; backfill = job River (06 M-5). Header komentar memuat fase (`expand`/`contract`), ref FR/ADR, serta estimasi baris dan lock. `db/schema.sql` di-regenerate. | Riwayat linear, dapat direview | CI QG-11 |
| STD-DB-12 | Expand/contract (06 §9): tidak ada perubahan destruktif di rilis yang sama dengan kode yang berhenti memakainya. Fase contract hanya dengan label `db-contract`. Setiap migrasi kompatibel dengan aplikasi N-1 (blue/green). | Rollback aplikasi aman (06 P-8) | CI QG-11, review checklist 06 §9.4 |
| STD-DB-13 | Index pada tabel berisi: `CREATE INDEX CONCURRENTLY` dalam file `-- +goose NO TRANSACTION`. FK/CHECK baru: `NOT VALID`, lalu `VALIDATE CONSTRAINT` di migrasi terpisah. `lock_timeout` 5 s di-set oleh `msp migrate`. | Tidak ada lock panjang di produksi | lint squawk, QG-11 |
| STD-DB-14 | Tabel tenant: kolom pertama `workspace_id uuid NOT NULL`; PK dan semua index diawali `workspace_id`; `ENABLE` + `FORCE ROW LEVEL SECURITY` + policy `USING`/`WITH CHECK` + `GRANT` di migrasi yang sama dengan `CREATE TABLE` (04 §5.1, 06 M-6). | Tidak ada tabel tenant tanpa RLS | CI cek RLS (QG-11, TC-AUD-013) |
| STD-DB-15 | Bagian `Down` hanya untuk kenyamanan lokal. Staging/production memakai forward fix (06 §9.1). Migrasi yang sudah masuk `main` tidak pernah diedit. | Riwayat immutable | review, CI (checksum file lama) *(usulan)* |

```sql
-- db/migrations/00042_create_action_list.sql
-- phase: expand · ref: FR-SEG-001 · est: tabel baru, tanpa lock pada tabel lain
-- +goose Up
CREATE TABLE action.list (
  workspace_id uuid        NOT NULL,
  id           uuid        NOT NULL,
  name         text        NOT NULL CHECK (length(name) BETWEEN 1 AND 120),
  owner_id     uuid        NOT NULL,
  created_at   timestamptz NOT NULL,
  PRIMARY KEY (workspace_id, id)
);
ALTER TABLE action.list ENABLE ROW LEVEL SECURITY;
ALTER TABLE action.list FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON action.list
  USING      (workspace_id = current_setting('app.workspace_id', true)::uuid)
  WITH CHECK (workspace_id = current_setting('app.workspace_id', true)::uuid);
GRANT SELECT, INSERT, UPDATE, DELETE ON action.list TO msp_app;
-- +goose Down
DROP TABLE action.list;
```

### 4.4 Skema, tipe, PostGIS/FTS, pagination, bulk

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-DB-16 | Penamaan: snake_case; tabel singular di schema modul (`action.list_item`); PK `id`; FK `<entity>_id`; waktu `<kata_kerja>_at`; boolean `is_`/`has_`; index `<tabel>_<kolom>_idx`; constraint `<tabel>_<kolom>_{key,fkey,check}`. | Konsisten dengan 04 §5 | review, squawk |
| STD-DB-17 | Waktu `timestamptz` (UTC); tanggal referensi data = teks periode (`2025-S1`); uang/credit `bigint`; skor sesuai 04 §7.9. Enum stabil = PostgreSQL ENUM, daftar yang sering berubah = `text` + validasi aplikasi (04 §5.1). `NULL` hanya bermakna Unknown bila terdokumentasi di kolom tersebut; sentinel `0`/`''` untuk Unknown dilarang, kecuali representasi yang sudah ditetapkan 04 (`intel.score.value = -1` UNSCORED, faktor `-1` UNKNOWN). | BR-TRUST-06 | review |
| STD-DB-18 | Geometri `geometry(…, 4326)` + index GiST. Jarak dalam meter memakai `geography`/`ST_DWithin` (BR-ER-02 50 m), bukan `ST_Distance(...) < x` di `WHERE`. Point-in-polygon (`ST_Covers`) dikerjakan set-based di SQL, bukan loop di Go. | Memakai index, akurat dalam meter | review, `EXPLAIN` |
| STD-DB-19 | FTS: `tsvector` dihitung di projection (konfigurasi `simple` + wrapper `IMMUTABLE` atas `unaccent`, karena `unaccent()` tidak immutable sehingga tidak dapat dipakai di index expression). Trigram `pg_trgm` memakai index GIN; threshold di-set dengan `SET LOCAL`. | ADR-05, NFR-PERF-01 | review |
| STD-DB-20 | Pagination keyset `(sort_key, id) < ($1, $2)` + `LIMIT n+1`. `OFFSET` dilarang di endpoint publik (ADR-12). Cursor ber-HMAC dibentuk dan diverifikasi hanya di `platform/pagination` (04 §10.5). | Stabil pada 5 juta baris | review, test |
| STD-DB-21 | Bulk ≥ 1.000 baris memakai `COPY` (`pgx.CopyFrom` / sqlc `:copyfrom`) ke tabel staging atau epoch, lalu `INSERT … ON CONFLICT` set-based pada natural key. INSERT per baris di dalam loop dilarang. | NFR-PERF-05/07, idempotensi batch (04 §6.2) | review, benchmark |

---

## 5. Jobs (River)

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-JOB-01 | Semua kerja async (stage pipeline, scoring, export, webhook, AI, backfill, dispatch event, periodic) = job River di PostgreSQL. Redis tidak pernah dipakai sebagai antrean (README, ADR-06). | Job ikut transaksi, backup, dan PITR | review |
| STD-JOB-02 | Job yang menjadi akibat perubahan data **wajib** di-insert dengan `InsertTx(ctx, tx, …)` di transaksi yang sama. Event domain di-emit lewat `events.Emit(ctx, tx, evt)` (baris `platform.event` + job `event.dispatch`; ADR-07, 04 §12.3). `Insert` di luar transaksi hanya untuk pemicu tanpa perubahan data. | Outbox transaksional: rollback → tidak ada job | test `rivertest` (05 §3.7), TC-EXP-009 |
| STD-JOB-03 | Worker idempotent: menjalankan ulang job yang sama (retry, rescue setelah crash, duplikat) menghasilkan state akhir yang sama. Efek eksternal (webhook, LLM, capture ledger) dijaga status/kunci idempotensi di DB yang diperiksa dalam transaksi. | At-least-once (River rescuer, 04 §6.2) | test TC-OPS-008/009, TC-BILL-016 |
| STD-JOB-04 | Kind stabil `<area>.<aksi_snake>` (`event.dispatch`, `webhook.deliver`, `export.generate`, `scoring.recompute_icp`). Antrean mengikuti ADR-06: `pipeline`, `scoring`, `export`, `webhook`, `ai`, `maintenance`; prioritas 1–4 di dalam antrean (1 = interaktif). | Kind tersimpan di `river_job`; rename memutus job yang sedang antre | review, registry kind di `platform/jobs` |
| STD-JOB-05 | Args berupa struct kecil berisi ID dan versi, bukan payload besar atau PII (args tersimpan di DB dan tampil di ops console). Tag JSON snake_case. Args tenant selalu memuat `workspace_id`, dan worker membuka `tenancy.WithTx` dengan nilai tersebut. | Privasi (BR-PRIV-01), isolasi jalur async (TC-AUD-009) | lint `musttag`, review |
| STD-JOB-06 | Perubahan args hanya aditif (field baru opsional dengan default aman). Perubahan tak kompatibel = kind baru (`…_v2`); worker lama tetap terdaftar sampai tidak ada job lama tersisa (≥ jendela retry terpanjang + 1 hari). | Deploy blue/green: job lama diproses kode baru dan sebaliknya | review (06 §9.4 "job args River kompatibel") |
| STD-JOB-07 | Job yang tidak boleh ganda memakai `UniqueOpts{ByArgs: true}` dengan tag `river:"unique"` pada field kunci (batch pipeline per `idem_key`, recompute per ICP, delivery per `(endpoint, event)`). | Enqueue ganda tidak menjalankan kerja dua kali | test integrasi |
| STD-JOB-08 | `MaxAttempts` eksplisit per kind. Backoff mengikuti 04 (pipeline 5× eksponensial 2 s → 5 m + jitter; webhook 7× jadwal 1 m … 12 j via `NextRetry`). Error permanen (validasi, data rusak, `410`) → `river.JobCancel(err)`. Job yang kehabisan percobaan (`discarded`) = DLQ dan dapat di-redrive dari ops console (FR-OPS-002). Job yang memanggil jaringan mengisi `Timeout(job)`. | Retry hanya untuk kegagalan sementara | test resilience (05 §3.18) |
| STD-JOB-09 | Job > 1 menit atau > 10.000 baris diproses per batch dengan checkpoint di DB (`platform.stage_run`, `score_job.done/total`; 04 §6.2, §7.9) dan melanjutkan dari checkpoint setelah restart. Progres dilaporkan untuk UI (NFR-PERF-05). | Restart tidak mengulang dari nol | review, test TC-OPS-008 |
| STD-JOB-10 | Observability per job: span `job <kind>` (middleware OTel River), log dengan `job_id`, `job_kind`, `queue`, `attempt`, `workspace_id`, `duration_ms`; metrik antrean 04 §16.3. `correlation_id` dan `traceparent` dari request asal dibawa di metadata job. | RG-5 (backlog, failure rate) | review, alert drill |
| STD-JOB-11 | Trafik request `api` boleh lewat PgBouncer transaction mode (PgBouncer lokal di setiap app host; RLS memakai `SET LOCAL`; `InsertTx` River ikut transaksi aplikasi). Setiap proses yang menjalankan client River (`msp worker`: fetch, `LISTEN/NOTIFY`, leader election, periodic job) memakai **pool kecil khusus yang terhubung langsung** ke primary `db-primary.internal:5432` (`MSP_DATABASE_URL_DIRECT_FILE`, 04 §17.4, §18.1), karena `LISTEN` butuh koneksi tingkat sesi yang rusak di transaction pooling. Fallback terdokumentasi bila koneksi langsung tidak tersedia: River `PollOnly` (`MSP_RIVER_POLL_ONLY=true`). Jadwal berkala memakai periodic job River, bukan cron host. | Worker tidak "tidur" diam-diam | test integrasi, review infra |

```go
// action/internal/jobargs/export.go — args kecil, kind stabil, unik per export
type ExportGenerate struct {
	WorkspaceID ids.WorkspaceID `json:"workspace_id" river:"unique"`
	ExportID    ids.ExportID    `json:"export_id"    river:"unique"`
}

func (ExportGenerate) Kind() string { return "export.generate" }
func (ExportGenerate) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueExport, Priority: 2, MaxAttempts: 5,
		UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// action/internal/service — job di-insert di transaksi yang sama dengan hold credit
return s.tenancy.WithTx(ctx, ws, func(tx pgx.Tx) error {
	if err := s.usage.HoldTx(ctx, tx, hold); err != nil {
		return fmt.Errorf("hold credits: %w", err)
	}
	if err := s.store.CreateExportJob(ctx, tx, job); err != nil {
		return fmt.Errorf("create export job: %w", err)
	}
	_, err := s.jobs.InsertTx(ctx, tx, jobargs.ExportGenerate{WorkspaceID: ws, ExportID: job.ID}, nil)
	return err
})
```

---

## 6. API & kontrak

### 6.1 Alur OpenAPI-first (ADR-08)

```
1. Ubah contracts/openapi/openapi.yaml (+ contoh)       → review kontrak dulu bila perubahan besar
2. go generate ./... (oapi-codegen strict-server → apps/api/internal/api)
   pnpm --filter @msp/api-client generate (openapi-typescript → packages/api-client)
3. Implementasi di <modul>/http.go → Service              → compiler memaksa semua operasi diimplementasi
4. Test unit/integrasi + Schemathesis lokal
5. Commit spec + kode generate + implementasi dalam satu MR (QG-12/13/14)
```

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-CONTRACT-01 | Spec adalah sumber kebenaran. Server stub Go dan client TS di-generate; type request/response tidak ditulis tangan. Regenerasi yang menghasilkan diff = gagal. | Kontrak dan kode tidak bisa menyimpang | CI QG-14 |
| STD-CONTRACT-02 | Spec ditulis dalam OpenAPI 3.1, tetapi hanya memakai subset yang didukung seluruh toolchain (oapi-codegen, openapi-typescript, oasdiff, Spectral, Schemathesis). oapi-codegen baru mendukung 3.1 secara awal (sejak v2.8.0, butuh Go ≥ 1.25), sehingga subset ditetapkan oleh spike toolchain Sprint 0 (04 TQ-18, 06 §18); konstruksi di luar subset ditolak rule Spectral kustom. Fallback bila spike gagal: spec 3.0.3, atau 3.1 di-*down-convert* lewat overlay untuk generator. | Dukungan 3.1 di generator Go belum penuh | lint Spectral (QG-12) |
| STD-CONTRACT-03 | Breaking change (hapus/rename field atau operasi, field opsional menjadi wajib, enum request menyempit, ubah tipe/format/semantik) diblok kecuali berlabel `api-breaking` + rencana versi. Di dalam `/v1` hanya perubahan aditif. | Integrasi customer stabil | CI QG-13 (oasdiff) |
| STD-CONTRACT-04 | Deprecation: `deprecated: true` di spec + header `Deprecation` (RFC 9745) dan `Sunset` (RFC 8594) di respons, minimal satu rilis minor dan 90 hari sebelum penghapusan *(usulan, berlaku sejak API key V1)*. | Waktu migrasi untuk integrator | review |
| STD-CONTRACT-05 | Event & webhook: JSON Schema 2020-12 di `contracts/events/`, yaitu envelope `envelope-<major>.<minor>.json` (04 §12.1) + satu file per tipe & versi `<type>/<major>.<minor>.json`. Versi yang sudah terbit immutable; perubahan aditif = minor, breaking = major baru yang dikirim paralel selama masa transisi *(usulan)*. | Konsumen men-dedupe & mem-parse dengan aman | CI QG-14 (ajv, immutability) |
| STD-CONTRACT-06 | Contoh payload valid dan invalid di `contracts/**/examples/` diuji di CI. Helper test memvalidasi setiap event yang di-enqueue terhadap schema (05 §3.8). Input Contract (`contracts/input/`, `observation-record/X.Y.Z`) mengikuti aturan versi yang sama. | Schema dan kode tidak menyimpang | CI QG-14, test |

### 6.2 Konvensi desain API

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-API-01 | Path `/v1/<resource-jamak-kebab>` (`/v1/webhook-endpoints/{id}`). Aksi non-CRUD sebagai sub-resource kata benda (`/v1/exports/preview`). `operationId` camelCase `verbNoun` unik (`listProspects`) dan menjadi nama metode Go. Field JSON dan query param snake_case; enum UPPER_SNAKE (04 §5.7); ID string berprefiks; waktu RFC 3339 UTC; uang integer rupiah (04 §11.1). | Konsisten untuk integrator & generator | lint Spectral |
| STD-API-02 | Setiap operasi wajib punya `x-msp-permission`, `x-msp-billable`, tag modul pemilik, `security`, dan respons error `application/problem+json`. Operasi tanpa deklarasi permission membuat CI gagal (04 §15.2). | Matriks RBAC dibangkitkan dari spec (TC-AUTH-006) | lint Spectral kustom |
| STD-API-03 | POST berbayar mendeklarasikan header `Idempotency-Key` sebagai *required*; middleware `platform/idempotency` menegakkan semantik 04 §11.3 (replay, 409 in-progress, 422 key reuse). | BR-CRED-04 | lint Spectral, test TC-BILL-002 |
| STD-API-04 | Operasi dengan p95 > 1 s atau berbayar massal bersifat async: `202 Accepted` + `Location` + body job `{id,type,status,progress{done,total},result,error}`; client polling `GET /v1/jobs/{id}`. | Request tidak timeout di Traefik | review |
| STD-API-05 | List memakai cursor pagination `?cursor=&limit=` (maks 100) → `{items, next_cursor, count?}`; `count` membawa flag eksak/estimasi (FR-SRCH-006). | ADR-12 | lint Spectral |
| STD-API-06 | Error RFC 9457 dengan `code` dari katalog 04 §11.2 + `correlation_id`; `type` = `https://docs.mesinprospek.id/errors/<code-kebab>`; `title` dilokalkan dari `Accept-Language`. Error validasi 422 `VALIDATION_FAILED` membawa extension member `errors[]` (RFC 9457) berisi `{pointer | field, code, detail}` per pelanggaran (04 §11.2); error tak terduga = 500 `INTERNAL_ERROR` tanpa detail internal. Resource milik workspace lain → 404, bukan 403. | Client bercabang pada `code`; keberadaan resource tidak bocor | test contract, TC-AUD-007 |
| STD-API-07 | Body request `additionalProperties: false` dengan batas panjang string/array di schema. Client wajib toleran terhadap field baru dan nilai enum baru di respons (tolerant reader). | Typo tertangkap; evolusi aditif aman | lint Spectral, review FE |
| STD-API-08 | Header baku: `MSP-Workspace` (endpoint tenant), `MSP-CSRF` (non-GET via cookie), `MSP-Correlation-Id` (diterima/dibangkitkan, selalu dikembalikan), `RateLimit-*`/`Retry-After`, `ETag`/`If-Match` untuk PUT ICP/Offering (04 §11.1). | ADR-25, lost update | test contract |
| STD-API-09 | Handler tipis: validasi struktural dilakukan tipe hasil generate + middleware validator OpenAPI; validasi semantik dan aturan resource di service. Handler tidak mengakses DB. | Logika bisnis teruji tanpa HTTP | lint `depguard`, review |

---

## 7. Keamanan

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-SEC-01 | Semua input divalidasi di boundary (schema + domain). Body dibatasi `http.MaxBytesReader` (default 1 MiB; upload memakai batas eksplisit). Parser file tidak mengeksekusi formula/macro (04 §15.10). | OWASP ASVS L2 (NFR-SEC-01) | lint `gosec`, test TC-DATA-030 |
| STD-SEC-02 | Otorisasi berlapis: middleware menolak berdasarkan `x-msp-permission` sebelum handler (04 §15.2), dan **service** memeriksa ulang permission + aturan tingkat resource (kepemilikan, "Member boleh export", assign ke orang lain, job milik sendiri vs semua) sesuai matriks PRD §8.1.1, karena service juga dipanggil dari job dan `msp tool`. ID di path di-resolve ke workspace pemilik (berbeda → 404). RLS adalah lapis berikutnya, bukan pengganti. Penolakan menulis `audit_log(authz.denied)`. | FR-AUTH-004, NFR-SEC-03 | test TC-AUTH-006, TC-AUD-007, QG-09 |
| STD-SEC-03 | Feature flag dan entitlement bukan pengganti otorisasi (06 §13.1). Tombol yang disembunyikan di UI tetap ditegakkan di server. | Flag bisa salah konfigurasi | review |
| STD-SEC-04 | HTTP keluar ke URL milik customer hanya lewat `netguard.Client(role)` (04 §12.5): validasi URL + resolve saat registrasi dan sebelum setiap delivery; `net.Dialer.Control` memeriksa ulang IP di setiap koneksi (di produksi satu-satunya tujuan yang diizinkan adalah alamat Smokescreen); redirect dimatikan; TLS ≥ 1.2; timeout connect 3 s/total 10 s; respons ≤ 64 KB; tanpa dekompresi. `http.DefaultClient`, `http.Get`, dan `&http.Client{}` ad-hoc dilarang di luar `netguard`/adapter yang di-inject. | NFR-SEC-05, RG-7 | lint `forbidigo`, test TC-EXP-005/020/022 |
| STD-SEC-05 | Secret tidak pernah ada di kode, test, log, pesan error, atribut span, atau URL. Secret disimpan dengan SOPS + age (06 §12) dan dibaca dari file. Di Go, secret dibungkus `secret.String` yang `String()`, `LogValue()`, dan `MarshalJSON()`-nya mengembalikan `[REDACTED]`. | NFR-SEC-04 | CI QG-18 (gitleaks, non-waivable), review |
| STD-SEC-06 | Data pribadi kelas P1 (04 §15.6) memakai tipe khusus (`pii.Value`) yang teredaksi saat di-log atau di-format. Korelasi memakai HMAC 8 hex (`h:<…>`) dari `platform/telemetry`. Key log terlarang ditolak lint. | BR-PRIV-01, NFR-PRIV-01 | lint `sloglint` `forbidden-keys`, test TC-AUD-010 (QG-10) |
| STD-SEC-07 | Tidak ada nilai contact point, nama person, isi upload, atau catatan tenant yang dikirim ke LLM (FR-AI-005). Input builder `ai` hanya menerima `ai.Fact` dari field allowlist; `pii.Value` tidak punya konversi ke `ai.Fact`; pemindai regex berjalan sebelum setiap panggilan (04 §13.3). | NFR-PRIV-04, D-14 | test TC-AI-003 (QG-10) |
| STD-SEC-08 | Masking kontak dilakukan di server (BR-CRED-02, 04 §10.7). DTO kontak ter-mask tidak memiliki field untuk nilai asli. | Nilai asli tidak pernah sampai ke browser | test TC-PROF-003 |
| STD-SEC-09 | Kriptografi hanya dari stdlib `crypto/*` dan `golang.org/x/crypto`. Webhook memakai HMAC-SHA256 `MSP-Signature: t=…,v1=…` (04 §12.4) dan verifikasi dengan `hmac.Equal`. Token acak dari `crypto/rand` (sesi 256-bit). Token ber-entropi tinggi (API key, sesi) disimpan sebagai SHA-256; secret ber-entropi rendah (bila kelak ada password lokal) memakai argon2id dengan parameter minimum OWASP (m ≥ 19 MiB, t ≥ 2, p = 1). Secret webhook dienkripsi AES-256-GCM dengan key ring berversi (ADR-22). `math/rand` v1 dilarang; `math/rand/v2` hanya untuk jitter. | Primitif teruji, tanpa kripto buatan sendiri | lint `gosec`, `depguard` |
| STD-SEC-10 | Path dari input user tidak pernah diteruskan langsung ke `os.Open`; gunakan `os.Root` atau nama file acak buatan server. Export memitigasi formula injection CSV dan XLSX tanpa formula (04 §15.10). Objek tenant di object storage berada di prefix `{workspace_id}/` dan hanya diakses lewat presigned URL. | Path traversal, injeksi | lint `gosec` G304, test TC-EXP-013 |
| STD-SEC-11 | Key cache Redis selalu diawali `ws:{workspace_id}:` untuk data tenant; Redis bukan sumber kebenaran (04 §15.3). | Isolasi lapis ketiga | test TC-AUD-009 |
| STD-SEC-12 | Sesi web memakai cookie `__Host-msp_sess` (HttpOnly, Secure, SameSite=Lax); metode non-GET wajib header `MSP-CSRF` (double-submit). CORS tidak diaktifkan karena same-origin (ADR-08). | Spoofing/CSRF (04 §15.11) | test, ZAP (QG-30) |
| STD-SEC-13 | Hygiene dependency: govulncheck (reachable = gagal), OSV-Scanner, `pnpm audit --audit-level high`, license allowlist (QG-19/20). Update lewat Renovate (06 §15.4). `replace` di `go.mod` ke fork butuh ADR. | NFR-SEC-06 | CI QG-19, QG-20 |

```go
// platform/netguard — cuplikan: IP diperiksa ulang saat connect, redirect mati
dialer := &net.Dialer{Timeout: 3 * time.Second, Control: func(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return err
	}
	return g.policy.CheckDial(ip.Unmap()) // produksi: hanya alamat Smokescreen yang lolos
}}
client := &http.Client{
	Timeout:       10 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, // 3xx = gagal
	Transport: &http.Transport{
		Proxy:              http.ProxyURL(g.egressProxy),
		DialContext:        dialer.DialContext,
		TLSClientConfig:    &tls.Config{MinVersion: tls.VersionTLS12},
		DisableCompression: true,
	},
}
```

---

## 8. Observability

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-OBS-01 | Log hanya lewat `*slog.Logger` (handler JSON dari `platform/telemetry`) yang di-inject, selalu dengan varian `…Context(ctx, …)`. Handler menambahkan `correlation_id`, `request_id`, `trace_id`, `span_id`, `workspace_id`, dan `user_id` (pseudonim) dari ctx secara otomatis. | NFR-OBS-01, korelasi log–trace | lint `sloglint` (`context: all`, `no-global: all`) |
| STD-OBS-02 | Field mengikuti 04 §16.2. Key snake_case; atribut bertipe (`slog.Int`, `slog.String`), grup `http` dan `error` memakai `slog.Group`. Pesan log statis tanpa interpolasi; variabel masuk atribut. | Query LogQL konsisten | lint `sloglint` (`attr-only`, `static-msg`, `key-naming-case: snake`) |
| STD-OBS-03 | Level: **ERROR** = butuh perhatian manusia (dan dikirim ke GlitchTip); **WARN** = degradasi atau retry yang masih tertangani; **INFO** = peristiwa lifecycle/bisnis (request selesai, job selesai, snapshot publish); **DEBUG** = mati di production. Satu log akses per request, di middleware. | Volume log dan alert bermakna | review |
| STD-OBS-04 | Tanpa PII atau secret di log, span, metrik, dan breadcrumb (STD-SEC-05/06). Query search mentah tidak di-log; cukup `has_query` dan `query_length` (03 §9). `ReplaceAttr` handler menghapus key terlarang sebagai jaring pengaman (04 §15.6), bukan sebagai mekanisme utama. | BR-PRIV-01 | test TC-AUD-010 |
| STD-OBS-05 | Tracing OTel: span server HTTP bernama `"{METHOD} {route}"` (route pattern, bukan path berisi ID); span DB dari tracer pgx dinamai sesuai nama query sqlc; span job `job <kind>`; span internal `<modul>.<operasi_snake>` (mis. `intelligence.score_chunk`). Atribut domain berprefiks `msp.` (`msp.workspace_id`, `msp.icp_id`, `msp.snapshot_id`, `msp.model_version`). Setiap span di-`End()`, error dicatat dengan `RecordError` + status. | Trace dapat dicari & di-join | lint `spancheck`, review |
| STD-OBS-06 | Metrik: instrumen OTel bernama `msp.<domain>.<nama>` dengan unit UCUM sehingga nama Prometheus hasil ekspor sama dengan katalog 04 §16.3 (`msp_stage_duration_seconds`, counter `_total`). Unit dasar (detik, byte). Label berkardinalitas rendah (≤ 1.000 seri per metrik); label `user_id`, `company_id`, URL mentah, dan pesan error dilarang; `workspace_id` hanya untuk metrik billing/AI yang disetujui. | Biaya & performa Prometheus | review |
| STD-OBS-07 | Error reporting GlitchTip (SDK Sentry) hanya untuk error tak terduga (5xx, panic, job gagal final). `release` = `msp@vX.Y.Z` (06 §16.1); `BeforeSend` memakai redaksi yang sama dengan log. | Sinyal bersih, tanpa PII | test TC-AUD-010 |
| STD-OBS-08 | Fitur yang menyentuh SLO atau alert RG-5 menambahkan metrik, alert, dan panel dashboard di MR yang sama. | NFR-OBS-01 | review checklist |
| STD-OBS-09 | Endpoint `/healthz` (liveness, tanpa dependency), `/readyz` (DB, River), dan `/version` (versi, commit) tidak butuh auth dan tidak menampilkan detail internal. | Blue/green & smoke (06 §6.3) | test smoke |

---

## 9. Data & intelligence

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-DATA-01 | Setiap nilai atribut yang ditulis ke global layer membawa `claim_type`, `confidence`, waktu observasi, dan `evidence_ids` (04 §5.6). Claim builder adalah satu-satunya jalur penulisan `claim_meta`. Nilai DERIVED/INFERRED mencatat versi rule/model (`ver`). | BR-TRUST-01/02, FR-DATA-006 (coverage ≥ 95%) | test, metrik evidence coverage |
| STD-DATA-02 | Unknown tidak pernah dikoersi menjadi `0`, `""`, `false`, atau nilai default. API mengembalikan `{ "claim_type": "UNKNOWN" }` dan export menulis `UNKNOWN` (03 P-12). Faktor skor Unknown dikeluarkan dan bobotnya di-renormalisasi (BR-SCORE-03). Alasan Unknown disimpan (`reason`). | BR-TRUST-06 | test TC-INTL-013, review |
| STD-DATA-03 | INFERRED tidak pernah dipromosikan menjadi OBSERVED. Confidence revenue inferred ≤ 70 (BR-REV-03). WhatsApp dari nomor seluler selalu INFERRED (BR-VER-01). Constructor `Claim` menolak kombinasi yang melanggar. | BR-TRUST-05 | test |
| STD-DATA-04 | Output deterministik: iterasi map tidak boleh memengaruhi hasil. Gunakan `slices.Sorted(maps.Keys(m))` atau slice berurutan dari config (urutan faktor = urutan di YAML model, 04 §7.1). Tie-break selalu eksplisit (04 §7.6). Persentil mid-rank punya satu implementasi (BR-SCORE-09). | BR-SCORE-13 | test properti (urutan input tidak berpengaruh, 05 §3.5) |
| STD-DATA-05 | Aritmetika skor memakai float64 dengan urutan penjumlahan tetap. Hasil perkalian yang dijumlahkan ditulis dengan konversi eksplisit `float64(w*s)`, karena spesifikasi Go mengizinkan compiler memfusikan `x*y+z` (FMA) di arsitektur seperti arm64 dan konversi eksplisit mencegahnya. Pembulatan hanya lewat `scoring.RoundHalfUp` (BR-SCORE-10); band dihitung dari nilai tampil. Golden skor wajib lulus di CI (amd64) dan disarankan dijalankan juga di laptop arm64. | Hasil identik lintas arsitektur | lint `forbidigo` (`math.Round` di paket scoring), test golden |
| STD-DATA-06 | Setiap output intelligence menyimpan versi inputnya: `model_version` (+preset), `geo_version`, `population_version`, versi snapshot, `prompt_version` + `provider_model`, `as_of`/`calculated_at` (BR-SCORE-06, FR-AI-001). Konfigurasi versi aktif immutable (`versions.lock`); perubahan bobot, aturan, atau prompt = versi baru + golden baru. | NFR-DATA-01, FR-INTL-009 | CI QG-23/24/25 |
| STD-DATA-07 | Skor dan metrik area dapat dihitung ulang dengan `msp tool score-reproduce …` (04 §7.8); nightly recompute sampel menghasilkan hash identik. | Reproducibility | CI nightly (06 §7) |
| STD-DATA-08 | Nilai mentah sumber selalu disimpan (FR-DATA-002). Normalizer murni, idempoten, tidak pernah panic, dan mengembalikan error ber-kode (05 §3.5). Daftar referensi (`data/reference/`, taksonomi) berversi dan versinya masuk `stage_version` (04 §6.2); tidak ada daftar hardcode di Go. | BR-NORM, re-run idempotent | test fuzz/rapid |
| STD-DATA-09 | Setiap projection, cache, file export, event, atau cache AI baru didaftarkan di registry projection beserta handler suppression-nya (05 §3.20). Data upload tenant tidak pernah menulis ke global layer (BR-ER-07). | FR-AUD-003 (≤ 24 jam), BR-PRIV-02 | test TC-AUD-003 |

```go
// BAIK — Unknown eksplisit, tidak bisa dibaca sebagai nilai tanpa cek
type Claim[T any] struct {
	value      T
	known      bool
	Type       ClaimType // OBSERVED | DERIVED | INFERRED | UNKNOWN
	Confidence uint8     // 0..99 (BR-TRUST-02)
	ObservedAt time.Time
	Evidence   []ids.EvidenceID
	Reason     UnknownReason // terisi bila !known: NOT_IN_SOURCE, NOT_CHECKED, LIMITED_DATA
}

func (c Claim[T]) Get() (T, bool) { return c.value, c.known }

// BURUK — 0 berarti "tidak ada data" sekaligus "nol karyawan"
type Commercial struct{ EmployeeCount int }
```

---

## 10. Testing

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-TEST-01 | Unit test table-driven dengan `t.Run(name, …)`. Nama subtest memuat ID TC/BR/EC (`"TC-DATA-021/rerun-snapshot-sama"`, `"EC-09/landline-separator"`; 05 §3.2). Default `t.Parallel()`. Assertion memakai stdlib + `go-cmp` (`cmp.Diff`), tanpa testify. | Traceability otomatis; satu gaya assertion | CI job traceability, lint `tparallel`, `depguard` |
| STD-TEST-02 | Test black-box (paket `<pkg>_test`) untuk API paket; white-box hanya untuk invariant internal. Helper memanggil `t.Helper()`. | Test tidak rapuh terhadap refactor | lint `thelper`, review |
| STD-TEST-03 | Integration test berada di `*_integration_test.go` dengan `//go:build integration`, memakai DB nyata (GitLab `services:` / testcontainers-go), role `msp_app` `NOBYPASSRLS`, dan helper `tenancy.WithTx` yang sama dengan produksi; satu workspace baru per test (05 §3.7). Data quality memakai tag `dataquality`. | Superuser menyembunyikan kebocoran RLS | CI QG-05, QG-09 |
| STD-TEST-04 | Golden file di `testdata/` (`*.golden`, JSON kanonik dengan key terurut, tanpa `calculated_at`), diperbarui hanya lewat flag `-update` dan direview. Golden skor disimpan per versi: `internal/intelligence/internal/scoring/testdata/priority-X.Y.Z/` (05 §3.10). | FR-INTL-009 | CI QG-23/24, CODEOWNERS |
| STD-TEST-05 | Property test `pgregory.net/rapid` untuk invariant (normalizer, confidence, persentil, priority, ledger stateful) dan fuzzing native `FuzzX` untuk parser/normalizer. Counterexample dan input pemicu bug di-commit ke `testdata/rapid/` dan `testdata/fuzz/` (05 §3.5). | Menemukan edge case yang tidak terpikir | CI QG-05 (fuzz smoke), nightly |
| STD-TEST-06 | Waktu dan randomness dikontrol: `clock.Fake`, seed tetap. `time.Sleep` untuk sinkronisasi dilarang; gunakan `testing/synctest` atau sinkronisasi eksplisit. | Test cepat dan tidak flaky | review |
| STD-TEST-07 | Fixture hanya sintetis (`msp-synth` dengan seed tetap) atau golden set yang disetujui; tidak ada PII produksi (05 §4.5). Test redaksi memakai nilai canary PII. | BR-PRIV, UU PDP | CI QG-18, review |
| STD-TEST-08 | Coverage: paket normalisasi, ER, scoring (termasuk opportunity & kalkulasi geo) ≥ 85% statement; backend ≥ 75% tanpa kode generate (NFR-MAINT-01); diff coverage ≥ 80% *(usulan, 05 §9.1)*. Coverage adalah syarat minimum, bukan target desain test. | NFR-MAINT-01 | CI QG-06, QG-07 |
| STD-TEST-09 | Perbaikan bug S1/S2 disertai test regresi bertanda TC sebelum issue ditutup (05 §5.4). Unit, property, dan integration test tidak pernah di-retry otomatis; quarantine hanya dengan issue + owner, maks 7 hari, total ≤ 1% suite, lewat penanda `testutil.Quarantine`/tag `@quarantine` (05 §5.4, 06 §16.3). Suite isolasi tenant, RBAC, SSRF, billing, suppression, redaksi, dan scoring regression tidak pernah di-quarantine. | Flaky tidak boleh menyembunyikan bug kritis | CI quarantine-check |
| STD-TEST-10 | Frontend: Vitest + Testing Library; query berdasarkan role/label/teks (bukan class; `data-testid` hanya bila terpaksa). Setiap komponen inti punya story untuk semua state (05 §3.6) yang dipakai ulang di test (`composeStories`) + `vitest-axe`. Tidak ada snapshot DOM besar. | Test mengikuti cara user (dan screen reader) | CI QG-05, QG-16 |
| STD-TEST-11 | Mock API frontend di level jaringan memakai MSW dengan tipe `paths` dari `@msp/api-client`. Fixture respons divalidasi terhadap OpenAPI. | Mock tidak menyimpang dari kontrak | test |
| STD-TEST-12 | Playwright: selector role/label, data disiapkan lewat API/seed, satu workspace baru per test, tanpa sleep tetap, anotasi `tc`, tag `@smoke` untuk critical path (05 §3.14). | E2E stabil | CI QG-17 |

---

## 11. Frontend Next.js

### 11.1 Struktur App Router

Route group memisahkan layout dan batas auth. Segmen di dalam group menghasilkan URL sesuai 03 §2.4 (`/app/…`, `/ops/…`).

```
apps/web/src/
├─ app/
│  ├─ layout.tsx                    root: <html lang>, tema (data-theme), font self-host, providers
│  ├─ global-error.tsx  not-found.tsx
│  ├─ (public)/                     tanpa sesi
│  │  ├─ auth/{login,signup,verify,reset}/page.tsx      SCR-01 (adapter IdP, D-20)
│  │  ├─ invite/[token]/page.tsx
│  │  └─ privacy/request/page.tsx                       SCR-25
│  ├─ (app)/app/                    URL /app/*; layout: sesi, workspace, ICP aktif, sidebar
│  │  ├─ layout.tsx  loading.tsx  error.tsx
│  │  ├─ page.tsx                   SCR-03
│  │  ├─ search/page.tsx            SCR-04
│  │  ├─ prospects/[companyId]/page.tsx                 SCR-05 (+ ?evidence= untuk SCR-06)
│  │  └─ map/  territories/  areas/  lists/  tasks/  jobs/  uploads/  icp/  settings/  account/
│  └─ (ops)/ops/                    URL /ops/*; layout: sesi ops (audience terpisah, MFA)
├─ features/<fitur>/                search, prospect, evidence, export, webhook, icp, upload, ops-…
│  └─ components/  queries.ts  schemas.ts  *.test.tsx
├─ lib/{api,auth,format,i18n,url-state}/
├─ messages/{id-ID,en-US}.json
└─ proxy.ts                         nonce CSP + header (nama di Next.js 16; sebelumnya middleware.ts)
```

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-FE-01 | Route group `(public)`, `(app)`, `(ops)` memisahkan layout dan batas auth. Kode `(ops)` tidak diimpor dari `(app)` dan sebaliknya. | Konsol ops punya audience & MFA sendiri (04 §15.1) | ESLint `no-restricted-imports` |
| STD-FE-02 | Logika per fitur berada di `features/<fitur>`; `app/**/page.tsx` hanya komposisi dan pemuatan data. Komponen generik dan pola trust berada di `packages/ui`. | Halaman tipis, fitur dapat diuji | review |
| STD-FE-03 | Next.js hanya UI (README): tidak ada logika bisnis, akses DB, Server Actions untuk mutasi, atau Route Handler selain health/CSP report. Semua data lewat API Go `/api/v1`. | Satu sumber aturan bisnis (Go) | ESLint (larang `"use server"`), review |

### 11.2 Server vs client component & data fetching

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-FE-04 | Default Server Component. `"use client"` hanya di daun interaktif (form, tabel interaktif, drawer, peta). Modul yang membaca cookie/header/env server mengimpor `server-only`. Hanya env `NEXT_PUBLIC_*` yang ada di allowlist boleh masuk bundle client. | Bundle kecil, tidak ada secret di client | build gagal (`server-only`), ESLint |
| STD-FE-05 | Props dari server ke client hanya data serializable hasil API; objek besar tidak dioper (kirim ID, biarkan query client mengambil sisanya). | Payload RSC kecil | review |
| STD-FE-06 | Semua panggilan API memakai client hasil generate (`@msp/api-client`, openapi-fetch); `fetch` manual ke `/api` dilarang. Middleware client menambahkan `MSP-Workspace`, `MSP-CSRF` (non-GET), `MSP-Correlation-Id`, dan `Idempotency-Key` (mutasi berbayar; dibuat sekali per niat user dan dipakai ulang saat retry). | Kontrak bertipe, header konsisten | ESLint `msp/no-raw-fetch` *(usulan)* |
| STD-FE-07 | Server state memakai TanStack Query. Query key factory per fitur diawali `["ws", workspaceId, …]`; saat pindah workspace dipanggil `queryClient.clear()` (03 §2.3, TC-AUTH-003). Data awal halaman di-prefetch di Server Component lalu di-hydrate (`HydrationBoundary`) bila mempercepat LCP. | Tidak ada data workspace lain di cache | ESLint `@tanstack/query`, test TC-AUTH-003 |
| STD-FE-08 | Data tenant tidak pernah masuk cache Next (Data Cache, Full Route Cache, `"use cache"`, `unstable_cache`). Fetch di server memakai `cache: "no-store"`; layout `(app)`/`(ops)` dinamis. | Isolasi tenant, data segar | review, ESLint *(usulan)* |
| STD-FE-09 | Error API ditangani berdasarkan `code` problem+json, bukan teks atau status HTTP saja. Pesan untuk user diambil dari katalog i18n mengikuti pola 03 §7.4, ditambah kode referensi 8 karakter dari `correlation_id`. | Copy konsisten, dapat dilacak | review, test |

```ts
// apps/web/src/features/search/queries.ts
export const prospectKeys = {
  all: (ws: string) => ["ws", ws, "prospects"] as const,
  search: (ws: string, p: SearchParams) => [...prospectKeys.all(ws), "search", p] as const,
};

export const searchQuery = (api: ApiClient, ws: string, p: SearchParams) =>
  queryOptions({
    queryKey: prospectKeys.search(ws, p),
    queryFn: async ({ signal }) => {
      const { data, error } = await api.GET("/v1/prospects", { params: { query: p }, signal });
      if (error) throw toAppError(error); // bercabang pada error.code
      return data;
    },
  });
```

### 11.3 Auth, form, state

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-FE-10 | Sesi dimiliki API Go (`__Host-msp_sess`, HttpOnly). Web tidak membaca atau menyimpan token di JS, `localStorage`, maupun `sessionStorage`. Layout `(app)` memvalidasi sesi lewat `GET /v1/me` di server dan redirect ke `/auth/login` bila 401. Flow login/sign-up/verify berada di balik adapter `lib/auth` yang agnostik terhadap IdP (Kratos atau OIDC, ADR-19); halaman tidak memanggil SDK IdP langsung. | D-20 masih Open; token tidak terekspos XSS | ESLint `no-restricted-globals` (`localStorage` di `lib/auth`), review |
| STD-FE-11 | Gating UI per role/plan memakai `/v1/me` + `/v1/entitlements`. Item yang tidak diizinkan tidak dirender (03 §2.5), tetapi server tetap menegakkan. | FR-PROF-009, FR-AUTH-004 | test RBAC E2E (J-10) |
| STD-FE-12 | Form memakai **react-hook-form + zod** (`@hookform/resolvers`). Alasan: input uncontrolled meminimalkan re-render pada form panjang (wizard ICP, mapping upload, webhook); skema zod sekaligus menjadi tipe TS; pesan validasi berupa kunci i18n. Server tetap validator final; error 422 `errors[].pointer`/`field` dipetakan ke `setError` (04 §11.2). | INP ≤ 200 ms (NFR-PERF-03), satu sumber validasi client | review |
| STD-FE-13 | Aksi berbayar selalu melewati dialog konfirmasi biaya (SCR-14, P-13). Tombol submit nonaktif selama mutasi berjalan; retry memakai `Idempotency-Key` yang sama. | BR-CRED-04, FR-BILL-004 | test E2E J-04/J-06 |
| STD-FE-14 | State yang menentukan hasil (`q`, `f`, `icp`, `sort`, `view`, `evidence`, …) ada di URL (03 §2.4) lewat helper `lib/url-state` yang mem-parse `searchParams` dengan zod, dan tidak diduplikasi ke state lokal. State UI sementara memakai `useState`/`useReducer`. Global store (Redux, Zustand, Context besar) tidak dipakai tanpa ADR. | Back/forward & berbagi tautan bekerja (FR-SRCH-005) | review |

### 11.4 Styling, komponen, boundary, performa, keamanan

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-FE-15 | Styling hanya lewat token (02): utilitas Tailwind dari preset `@msp/design-tokens` (palet dan skala default Tailwind tidak tersedia) atau `var(--msp-*)`. Hex, `rgb()`/`hsl()`, px literal, dan arbitrary value Tailwind (`text-[#c2410c]`, `p-[13px]`) dilarang di `apps/web` dan `packages/ui`. | Tema & brand (D-17) cukup diganti di token | ESLint `msp/no-raw-design-values` *(usulan)*, stylelint (`color-no-hex`, `unit-disallowed-list: [px]`), QG-03 |
| STD-FE-16 | Komponen interaktif dibangun di `packages/ui` di atas Radix; `@radix-ui/*` hanya boleh diimpor dari `packages/ui`. Pola trust P-01..P-13 (03 §6) hanya dirender lewat komponen `packages/ui` (`ClaimBadge`, `ConfidenceMeter`, `FreshnessBadge`, `ScoreBadge`, `ClaimValue`, `MaskedValue`, …). Ikon Lucide (DS-01) lewat wrapper ikon dengan `aria-hidden` default. | Satu implementasi a11y & trust | ESLint `no-restricted-imports` |
| STD-FE-17 | Setiap segmen route di `(app)`/`(ops)` punya `loading.tsx` (skeleton sesuai layout akhir, tanpa layout shift) dan `error.tsx` (state ERROR 03 §4.0 dengan "Coba lagi"). Bagian independen dalam satu halaman memakai error boundary lokal ("Bagian ini gagal dimuat", 03 §7.4). | Kegagalan satu kartu tidak menjatuhkan halaman | review, story state error |
| STD-FE-18 | Budget performa: LCP p75 ≤ 2,5 s dan INP p75 ≤ 200 ms (NFR-PERF-03); first-load JS per route ≤ 180 KB gzip *(usulan)*. Komponen berat (MapLibre, chart) di-lazy-load (`next/dynamic`, `ssr: false`) dan tidak masuk bundle route lain. Font self-host via `next/font/local` (02 §4). Tabel dipaginasi server (≤ 100 baris per halaman). | NFR-PERF-03 | Lighthouse CI nightly (06), cek bundle di `web:build` *(usulan)*, RUM saat beta |
| STD-FE-19 | CSP berbasis nonce dibuat di `proxy.ts`: `default-src 'self'; script-src 'self' 'nonce-…' 'strict-dynamic'; style-src 'self' 'unsafe-inline'` (atribut style dari Radix/React); `img-src 'self' data: blob:`; `connect-src 'self'` + endpoint GlitchTip; `worker-src 'self' blob:` (MapLibre); `frame-ancestors 'none'; base-uri 'none'; object-src 'none'; form-action 'self'`. Header statis (HSTS, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, `Permissions-Policy` minimal) dipasang di Traefik. | XSS & clickjacking (NFR-SEC-01) | ZAP (QG-30), test header di smoke |
| STD-FE-20 | `dangerouslySetInnerHTML` dilarang. Output AI brief dirender sebagai struktur (kalimat + sitasi) dari JSON, bukan HTML atau Markdown mentah. | XSS dari konten model | ESLint `react/no-danger` |

```tsx
// BURUK — warna & ukuran mentah, Unknown jadi "-", status hanya warna
<span className="text-[#c2410c] text-[13px]">{band}</span>
<td>{revenueBand ?? "-"}</td>

// BAIK — token & komponen pola trust (ikon + label, Unknown → "Tidak diketahui")
<ScoreBadge value={84} band="HOT" coverage={0.95} />
<ClaimValue claim={revenueBand} format={fmt.revenueBand} />
```

---

## 12. Aksesibilitas & i18n

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-A11Y-01 | Target WCAG 2.2 AA (NFR-A11Y-01): axe 0 pelanggaran serious/critical di story dan halaman E2E, ditambah checklist manual per rilis (05 §3.15). | DoD PRD §12.2 | CI QG-16 |
| STD-A11Y-02 | Status tidak pernah hanya warna: band, claim type, confidence, freshness, dan verifikasi selalu berupa ikon (bentuk berbeda per nilai) + label teks; claim juga memakai gaya border (02 §1, 03 §6). Test komponen memastikan teks/role ada, bukan hanya kelas warna (05 §3.6). | NFR-A11Y-01 | test component |
| STD-A11Y-03 | HTML semantik lebih dulu (`button`, `a`, `table` + `th scope`, heading berurutan); ARIA hanya bila semantik native tidak cukup. Elemen klik non-interaktif (`div onClick`) dilarang. | Screen reader & keyboard gratis | ESLint `jsx-a11y` (strict) |
| STD-A11Y-04 | Fokus: ring `:focus-visible` dari token `color.border.focus`; drawer/dialog menjebak fokus, Esc menutup, dan fokus kembali ke pemicu (03 §8.3); elemen sticky tidak menutupi fokus (`scroll-padding`, WCAG 2.4.11). | 03 §8 | test component, checklist manual |
| STD-A11Y-05 | Semua fungsi dapat dijalankan dengan keyboard, termasuk peta dan tabel (03 §8.2). Shortcut satu huruf dapat dimatikan dan tidak aktif saat fokus di input (WCAG 2.1.4). | 03 §8.1 | test E2E, checklist |
| STD-A11Y-06 | Live region: hitungan hasil, jumlah terpilih, progres job, dan toast sukses memakai `aria-live="polite"`; hanya error yang memblokir memakai `assertive`. Toast error tidak hilang otomatis. Label pembaca layar mengikuti 03 §8.4 dan berasal dari katalog i18n; ikon dekoratif `aria-hidden`. | WCAG 4.1.3 | test component |
| STD-A11Y-07 | Target interaktif ≥ 24 px (≥ 44 px di bawah `breakpoint.md`); `prefers-reduced-motion` dihormati lewat token motion (02 §5). | WCAG 2.5.8 | review, token |
| STD-I18N-01 | Library i18n: **next-intl**. Alasan: mendukung App Router dan Server Components, memakai ICU MessageFormat, formatter berbasis `Intl`, kunci bertipe lewat augmentasi TS, dan dapat berjalan **tanpa prefix locale di URL** sesuai route 03 §2.4. Locale diambil dari preferensi user (profil + cookie), default `id-ID`; `<html lang>` mengikuti locale. | NFR-L10N-01 | review |
| STD-I18N-02 | Tidak ada string UI hardcode di JSX/TS; semua teks (termasuk `aria-label`) dari katalog. | Dua bahasa lengkap (DoD) | ESLint `i18next/no-literal-string` (mode `jsx-text-only`) |
| STD-I18N-03 | Kunci: segmen snake_case huruf kecil dipisah titik, diawali namespace fitur/layar (`search.zero_result.title`, `guardrail.coverage_low` seperti 03 §7.7). Kunci id-ID = kunci en-US. Kunci tidak dibangun dari string bebas; enum dipetakan ke kunci lewat map eksplisit. | Katalog dapat dicek statis | CI cek parity katalog *(usulan, 05 §3.22)* |
| STD-I18N-04 | Copy id-ID mengikuti 03 §7 (sapaan "Anda", glosarium §7.2). HOT/WARM/COLD tidak diterjemahkan; UNSCORED → "Belum dinilai" (03 P-05). | Konsistensi produk | review Product |
| STD-I18N-05 | Semua format angka, uang, tanggal, dan waktu relatif lewat satu modul `lib/format` yang diuji per locale (05 §3.22). `toLocaleString()` dan `new Intl.*` langsung di komponen dilarang. | Satu tempat untuk aturan 03 §7.3 | ESLint `msp/no-direct-intl` *(usulan)* |
| STD-I18N-06 | Angka memakai `Intl.NumberFormat('id-ID')` (12.400; 4,2; maks 1 desimal di UI). Rupiah tidak memakai `style: 'currency'` (menghasilkan spasi dan desimal): prosa "Rp5–15 miliar", ringkas "Rp500 jt" / "Rp5–15 M" ("M" = miliar, tidak pernah juta); en-US "IDR 5–15B" (03 §7.3, BR-REV-02). | Format sesuai 03 | test unit formatter (100%) |
| STD-I18N-07 | Tanggal dari API (UTC) ditampilkan di zona user (default `Asia/Jakarta`) dengan singkatan WIB/WITA/WIT dari map eksplisit, karena ICU en-US menampilkan "GMT+7" bukan "WIB". Format: "10 Okt 2026, 14.30 WIB" / "Oct 10, 2026, 2:30 PM WIB"; waktu relatif hanya ≤ 30 hari; `<time dateTime>` selalu berisi ISO. | NFR-L10N-01, 03 §7.3 | test unit formatter |
| STD-I18N-08 | Nilai Unknown ditampilkan "Tidak diketahui" lewat `ClaimValue` (P-12), tidak pernah "-", "0", atau "N/A", termasuk di tabel dan export. Backend melokalkan `title` error dan rationale dari `Accept-Language` dengan template berversi di katalog backend, bukan string di kode Go. | BR-TRUST-06, 04 §8.4 | test component, review |

---

## 13. Git, MR & code review

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-GIT-01 | Trunk-based development: branch `feat/`, `fix/`, `chore/`, atau `docs/` + `<scope>-<deskripsi>`, umur ≤ 2 hari (maks 5); tanpa branch `develop`. Pekerjaan yang belum selesai masuk `main` di balik flag `rel.*` (06 §2.1). | Integrasi sering, konflik kecil | review |
| STD-GIT-02 | Judul MR = Conventional Commits dengan scope modul (06 §2.2), squash + fast-forward merge. Deskripsi memuat `Refs: FR-…/BR-…` dan `Test: TC-…`. | Changelog otomatis, traceability | CI QG-01 |
| STD-GIT-03 | Ukuran MR ≤ 400 baris diubah, tidak termasuk kode generate, golden, lockfile, dan data referensi. Di atas 800 baris wajib alasan di deskripsi atau dipecah (stacked MR, flag). Satu MR = satu perubahan logis; refactor dipisah dari perubahan perilaku. | Review efektif menurun drastis pada MR besar | review |
| STD-GIT-04 | MR memakai template (Lampiran E). Checklist yang tidak relevan dicoret dengan alasan singkat, tidak dihapus. | Reviewer tahu apa yang sudah dicek | review |
| STD-GIT-05 | Approval: ≥ 1 owner area yang tersentuh (2 untuk path berisiko saat tim ≥ 4), approver ≠ author, approval setelah commit terakhir, semua thread resolved. Karena GitLab CE tidak menegakkan required approvals/Code Owners, aturan ini ditegakkan job manual blocking `mr:gate` (06 §1.3, §2.3). | Four-eyes di CE | CI QG-02 |
| STD-GIT-06 | Reviewer memakai checklist §13.2. Komentar diberi prefiks `blocking:`, `nit:`, `question:`, atau `suggestion:`. Respons pertama ≤ 1 hari kerja *(usulan)*. | Ekspektasi jelas, MR tidak menggantung | review |
| STD-GIT-07 | Commit tidak memuat secret, PII, binary besar, atau output build (`.env`, `node_modules`, `.next`, `coverage`, `*.out` di `.gitignore`). History `main` tidak ditulis ulang; pembatalan lewat MR `revert:`. | NFR-SEC-04, audit | CI QG-18 + server hook pre-receive |

### 13.1 CODEOWNERS (format GitLab)

Mengikuti 06 §2.3, dengan path paket kritis per modul. Sintaks GitLab: **satu pola per baris**, diikuti owner (`@user`, `@grup`, atau email); di dalam satu section (atau seluruh file bila tanpa header `[Section]`), pola terakhir yang cocok menang, sehingga pola umum ditulis sebelum pola spesifik.

```
# .gitlab/CODEOWNERS
*                                                           @msp/backend @msp/web
/docs/                                                      @msp/product
/docs/02-design-system/                                     @msp/design
/docs/08-engineering-standards.md                           @msp/backend @msp/web
/apps/web/                                                  @msp/web
/packages/ui/                                               @msp/web @msp/design
/packages/design-tokens/                                    @msp/design @msp/web
/apps/api/                                                  @msp/backend
/apps/api/internal/identity/                                @msp/data-intel
/apps/api/internal/intelligence/                            @msp/data-intel
/apps/api/internal/intelligence/internal/scoring/testdata/  @msp/data-intel @msp/product
/apps/api/internal/ai/                                      @msp/data-intel
/apps/api/internal/platform/                                @msp/backend @msp/security
/apps/api/internal/usage/                                   @msp/backend @msp/security
/apps/api/db/migrations/                                    @msp/backend @msp/platform
/contracts/                                                 @msp/api-owners
/data/                                                      @msp/data-intel
/infra/                                                     @msp/platform
/.gitlab-ci.yml                                             @msp/platform
/ci/                                                        @msp/platform
/scripts/ci/                                                @msp/platform
/scripts/release/                                           @msp/platform
```

### 13.2 Checklist review

| Area | Pertanyaan reviewer |
|---|---|
| Kebenaran | Acceptance FR terpenuhi? Edge case (Unknown, kosong, batas, konkurensi, retry, duplikat) tertangani? Kegagalan eksplisit dan dapat di-retry? |
| Test | Level sesuai 05? Subtest ber-TC? Golden berubah hanya bersama bump versi? Tidak ada sleep, randomness tak terkontrol, atau PII? |
| Keamanan | Permission dicek di service (bukan hanya middleware)? Input divalidasi dan dibatasi? HTTP keluar lewat `netguard`? Secret aman? |
| Tenant/RLS | Transaksi lewat `tenancy.WithTx`? `workspace_id` eksplisit di query? Tabel baru ber-RLS + `FORCE`? Key cache `ws:`? Args job membawa `workspace_id`? ID milik workspace lain → 404? |
| PII | Tidak ada PII di log/span/metrik/args job/LLM/fixture? Masking di server? Projection baru terdaftar di registry suppression? |
| Data/intelligence | `claim_type`/confidence/evidence ditulis? Unknown tidak dikoersi? Deterministik (map, jam, float)? Versi model/geo/prompt/snapshot dicatat? |
| DB/migrasi | Fase expand/contract? `CONCURRENTLY`, `NOT VALID`? Estimasi lock? Backfill sebagai job? `EXPLAIN` untuk query baru? `schema.sql` & sqlc di-regenerate? |
| Kontrak | Spec diubah lebih dulu? Kode generate di-commit? `x-msp-permission`/`x-msp-billable`? Breaking → label + rencana? Schema event versi baru, bukan edit versi terbit? |
| Jobs | Idempotent? `InsertTx` di transaksi yang sama? Unique opts? Error permanen → `JobCancel`? Args kompatibel N-1? |
| Observability | Log terstruktur tanpa PII? Span & metrik baru sesuai konvensi? Alert/dashboard untuk SLO terkait? |
| Frontend/a11y | Hanya token? Komponen dari `packages/ui`? Ikon + label untuk status? Keyboard & fokus? Loading/error state? Copy id-ID & en-US? Screenshot 1280 px + 360 px? |
| Docs | `docs/`, register keputusan, ADR, runbook diperbarui bila relevan? |

---

## 14. Dokumentasi & ADR

| ID | Aturan | Alasan | Penegakan |
|---|---|---|---|
| STD-DOC-01 | Dokumentasi adalah Markdown di `docs/` dan ikut review MR. Diagram berupa teks (ASCII/Mermaid), bukan gambar biner tanpa sumber. | Dapat di-diff & direview | CI QG-22 (markdownlint, lychee) |
| STD-DOC-02 | ID (FR, NFR, BR, TC, QG, ADR, D, STD) tidak pernah dipakai ulang atau diubah artinya; yang tidak berlaku ditandai *deprecated*/dicoret. Kode dan test merujuk ID di nama subtest atau komentar (`// BR-SCORE-10`). | Traceability PRD ↔ kode ↔ test | CI cek ID (QG-22, advisory), job traceability (05 §3.2) |
| STD-DOC-03 | ADR wajib bila: menambah dependency runtime atau komponen infra baru; mengubah batas modul/layering; mengubah model data, tenancy, atau model keamanan; mengubah kontrak publik secara breaking; memilih alternatif yang mahal dibalik; atau menyimpang dari aturan STD tanpa batas waktu. | Alasan keputusan tidak hilang | review |
| STD-DOC-04 | Register ADR tetap di 04 §1.3 (satu baris per ADR). ADR yang butuh lebih dari satu baris ditulis di `docs/adr/ADR-nn-<slug>.md` dengan template Lampiran F dan ditautkan dari register. Status: Proposed → Accepted → Superseded/Deprecated; ADR tidak dihapus. | Satu indeks, detail bila perlu | review |
| STD-DOC-05 | Perubahan status keputusan `D-nn` dicatat di register 00 §7 dalam MR yang sama dengan dokumen yang terdampak; dokumen yang bertentangan diperbarui atau diberi catatan eksplisit. | README §Status dokumen | review |
| STD-DOC-06 | Perubahan pada dokumen ini lewat MR berlabel `docs` dengan approval tech lead. Aturan baru yang dapat diotomasi masuk linter/CI dalam ≤ 2 sprint; sampai saat itu ditandai *(usulan)*. | Standar yang tidak ditegakkan akan diabaikan | review |
| STD-DOC-07 | Setiap facade modul punya `doc.go` berisi tanggung jawab, tabel milik, event yang di-emit/dikonsumsi, dan rujukan ke 04 §3.2. | Onboarding & review lintas modul | lint `revive` (`package-comments`) |

---

## 15. Lampiran

### Lampiran A — Library & tool di luar README (Proposed)

Library di 04 §3.1 (mis. `nyaruka/phonenumbers`, `excelize`, `miekg/dns`, `google/uuid`, SDK Sentry Go) dan tool di 06 §1.5 tetap berlaku. Tambahan dari dokumen ini:

| Area | Pilihan | Alasan |
|---|---|---|
| Konkurensi | `golang.org/x/sync/errgroup` | Goroutine berbatas + propagasi error/cancel (STD-GO-27) |
| Test Go | `github.com/google/go-cmp` (sudah disebut 05), `go.uber.org/goleak`, `testcontainers-go` (05), `golang.org/x/perf/cmd/benchstat` | Assertion diff, deteksi leak, DB lokal, perbandingan benchmark |
| Observability Go | `go.opentelemetry.io/contrib/…/otelhttp`, `github.com/exaring/otelpgx`, middleware OTel River (`rivercontrib/otelriver`) | Span HTTP/DB/job tanpa instrumentasi manual |
| Validasi request | `github.com/oapi-codegen/nethttp-middleware` (kin-openapi) | Validasi struktural dari spec (STD-API-09) |
| Kripto (bila perlu) | `golang.org/x/crypto/argon2` | Hanya bila ada secret ber-entropi rendah (STD-SEC-09) |
| Form | `react-hook-form`, `zod`, `@hookform/resolvers` | STD-FE-12 |
| i18n | `next-intl` | STD-I18N-01 |
| Test frontend | `msw` | Mock level jaringan bertipe (STD-TEST-11) |
| Lint frontend | `eslint-plugin-jsx-a11y`, `@tanstack/eslint-plugin-query`, `eslint-plugin-i18next` (hanya rule `no-literal-string`), `stylelint` + `stylelint-config-standard`, `prettier-plugin-tailwindcss` | A11y, TanStack, string literal, CSS mentah, urutan class |
| Util frontend | `clsx` + `tailwind-merge`, `lucide-react` (DS-01), `server-only` | Komposisi class, ikon, batas server/client |

### Lampiran B — `apps/api/.golangci.yml` (golangci-lint v2, cuplikan)

```yaml
version: "2"
run:
  timeout: 5m
  build-tags: [integration, dataquality]
linters:
  default: standard            # errcheck, govet, ineffassign, staticcheck, unused
  enable:
    - bodyclose          # body respons HTTP ditutup
    - containedctx       # context tidak disimpan di struct
    - contextcheck       # context diteruskan, bukan dibuat ulang
    - depguard           # batas modul & paket terlarang (STD-GO-02/05)
    - errname            # ErrX / XError
    - errorlint          # %w, errors.Is/As
    - exhaustive         # switch enum lengkap (claim_type, band, status)
    - forbidigo          # time.Now, os.Getenv, http.DefaultClient, fmt.Print
    - forcetypeassert    # type assertion selalu dua nilai
    - gochecknoglobals   # tanpa state global (STD-GO-24)
    - gochecknoinits     # tanpa init() (STD-GEN-02)
    - gocritic
    - gosec              # keamanan (QG-19)
    - misspell
    - musttag            # struct yang di-marshal punya tag (args job, event)
    - nilerr
    - noctx              # request HTTP dengan context
    - nolintlint         # //nolint wajib spesifik + alasan
    - recvcheck
    - revive             # exported doc, naming, package-comments
    - rowserrcheck
    - sloglint           # gaya & keamanan slog (STD-OBS-01/02, STD-SEC-06)
    - spancheck          # span OTel di-End
    - sqlclosecheck
    - thelper
    - tparallel
    - unconvert
    - unparam
    - usetesting         # t.Context(), t.TempDir(), t.Setenv()
  settings:
    nolintlint: { require-explanation: true, require-specific: true }
    errorlint: { errorf: true }
    exhaustive: { default-signifies-exhaustive: false }
    sloglint:
      no-global: all
      context: all
      attr-only: true
      static-msg: true
      key-naming-case: snake
      forbidden-keys: [email, phone, person_name, value, raw_value, password, token, secret, cookie, authorization]
    forbidigo:
      analyze-types: true
      forbid:
        - { pattern: '^time\.(Now|Since|Until)$', msg: "pakai clock.Clock yang di-inject (STD-GO-30)" }
        - { pattern: '^os\.(Getenv|LookupEnv)$', msg: "konfigurasi hanya di internal/app (STD-GO-25)" }
        - { pattern: '^http\.(Get|Head|Post|PostForm|DefaultClient)$', msg: "HTTP keluar lewat netguard/klien ter-inject (STD-SEC-04)" }
        - { pattern: '^fmt\.Print', msg: "pakai slog (STD-OBS-01)" }
        - { pattern: '^context\.(Background|TODO)$', msg: "teruskan ctx dari pemanggil (STD-GO-21)" }
    depguard:
      rules:
        global:
          deny:
            - { pkg: "math/rand$", desc: "crypto/rand, atau math/rand/v2 untuk jitter (STD-SEC-09)" }
            - { pkg: "log$", desc: "pakai log/slog" }
            - { pkg: "github.com/pkg/errors", desc: "pakai errors + fmt.Errorf %w" }
            - { pkg: "database/sql", desc: "pakai pgx v5 + sqlc (ADR-26)" }
            - { pkg: "github.com/lib/pq", desc: "pakai pgx v5" }
            - { pkg: "github.com/stretchr/testify", desc: "pakai go-cmp (STD-TEST-01)" }
            - { pkg: "net/http/pprof", desc: "daftarkan handler pprof eksplisit di mux admin (STD-GO-38)" }
        pure:   # domain & sub-domain murni (STD-GO-05)
          files: ["**/internal/*/internal/domain/**", "**/internal/*/internal/scoring/**",
                  "**/internal/*/internal/normalize/**", "**/internal/*/internal/er/**",
                  "**/internal/*/internal/opportunity/**", "**/internal/*/internal/geocalc/**"]
          deny:
            - { pkg: "github.com/jackc/pgx", desc: "domain murni tanpa I/O" }
            - { pkg: "net/http", desc: "domain murni tanpa I/O" }
            - { pkg: "github.com/riverqueue/river", desc: "domain murni tanpa I/O" }
        # aturan layer per modul dibangkitkan tools/gen-depguard dari tabel 04 §3.3 (contoh di 04 §3.3)
  exclusions:
    generated: strict
    rules:
      - { path: '_test\.go', linters: [gochecknoglobals, forcetypeassert, unparam] }
      - { path: 'internal/platform/clock/', linters: [forbidigo], text: 'time\.' }
      - { path: '(cmd/|internal/app/)', linters: [forbidigo], text: '(context\.|os\.)' }
formatters:
  enable: [gofmt, goimports]
  settings:
    goimports:
      local-prefixes: ["<module-path>"]   # path module apps/api (ditetapkan saat bootstrap, Q-08)
```

### Lampiran C — ESLint, Prettier, tsconfig (setelan kunci)

```js
// packages/eslint-config/index.js (cuplikan ilustratif; nama preset mengikuti versi plugin)
export default [
  ...tseslint.configs.strictTypeChecked,
  nextPlugin.configs["core-web-vitals"],
  jsxA11y.flatConfigs.strict,
  ...tanstackQuery.configs["flat/recommended"],
  {
    plugins: { msp, i18next },
    rules: {
      "no-restricted-imports": ["error", { patterns: [
        { group: ["@radix-ui/*"], message: "Pakai komponen @msp/ui (STD-FE-16)" },
      ] }],
      "react/no-danger": "error",
      "i18next/no-literal-string": ["error", { mode: "jsx-text-only" }],
      "msp/no-raw-design-values": "error",   // hex/px/arbitrary value (STD-FE-15)
      "msp/no-direct-intl": "error",         // format lewat lib/format (STD-I18N-05)
      "msp/no-raw-fetch": "error",           // API lewat @msp/api-client (STD-FE-06)
      "@typescript-eslint/no-floating-promises": "error",
      "@typescript-eslint/consistent-type-imports": "error",
      "@typescript-eslint/switch-exhaustiveness-check": "error",
    },
  },
];
```

```json
// .prettierrc
{ "printWidth": 100, "trailingComma": "all", "plugins": ["prettier-plugin-tailwindcss"] }
```

```jsonc
// packages/tsconfig/base.json
{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "noFallthroughCasesInSwitch": true,
    "exactOptionalPropertyTypes": false,   // kompatibilitas tipe hasil openapi-typescript
    "verbatimModuleSyntax": true,
    "isolatedModules": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "target": "es2022",
    "skipLibCheck": true
  }
}
```

### Lampiran D — `.editorconfig`

```ini
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true
indent_style = space
indent_size = 2

[*.go]
indent_style = tab

[{Makefile,*.mk}]
indent_style = tab

[*.md]
trim_trailing_whitespace = false
```

### Lampiran E — Template MR (`.gitlab/merge_request_templates/Default.md`)

Versi lengkap dari 06 §2.4:

```markdown
## Ringkasan
<!-- Apa dan kenapa. Judul MR = Conventional Commits, mis. feat(action): ekspor XLSX streaming -->

## Requirement & test
Refs: FR-___ / NFR-___ / BR-___      Test: TC-___ (05)      ADR/D: ___ (bila ada)

## Checklist
- [ ] Test sesuai level di 05; subtest ber-TC; tanpa data pribadi nyata di test/log/screenshot
- [ ] Permission dicek di service; tenant lewat `tenancy.WithTx`; ID milik workspace lain → 404
- [ ] Unknown tidak dikoersi; claim_type/confidence/evidence ditulis; output deterministik
- [ ] Migrasi: fase `expand`/`contract`, kompatibel N-1, `CONCURRENTLY`/`NOT VALID`, RLS untuk tabel tenant, `schema.sql` & sqlc di-regenerate
- [ ] Kontrak: OpenAPI/JSON Schema diubah lebih dulu; kode Go & client TS di-regenerate; breaking → label `api-breaking`
- [ ] Job: idempotent, `InsertTx`, args kompatibel dengan versi sebelumnya
- [ ] Skor/geo/prompt: versi dinaikkan, golden baru, laporan regresi/eval terlampir (06 §10)
- [ ] Feature flag terdaftar di registry (nama, tipe, owner, kedaluwarsa)
- [ ] UI: token saja, komponen `packages/ui`, ikon + label, keyboard/fokus, screenshot 1280 px + 360 px, copy id-ID & en-US
- [ ] Observability: log/metrik/span tanpa PII; alert/dashboard bila menyentuh SLO
- [ ] Dokumen, register keputusan, atau runbook diperbarui (bila relevan)

## Dampak performa
<!-- EXPLAIN untuk query baru di jalur search/profil; benchstat untuk hot path -->

## Catatan migrasi / rollout
<!-- urutan deploy, flag, backfill, rencana rollback -->

## Rencana versi API (wajib bila `api-breaking`)
```

### Lampiran F — Template ADR (`docs/adr/ADR-nn-<slug>.md`)

```markdown
# ADR-nn — <Judul keputusan dalam bentuk kalimat aktif>

| Field | Value |
|---|---|
| Status | Proposed / Accepted / Superseded oleh ADR-mm / Deprecated |
| Tanggal | YYYY-MM-DD |
| Pengambil keputusan | <nama/peran> |
| Ref | D-nn, FR-/NFR-/BR-…, ADR terkait |

## Konteks
<!-- Masalah, batasan (D-19 dedicated server, D-14 residensi, tim kecil), dan data pendukung -->

## Keputusan
<!-- Apa yang dipilih, dalam 1–3 kalimat yang dapat diverifikasi -->

## Alternatif yang dipertimbangkan
| Opsi | Kelebihan | Kekurangan | Alasan ditolak |
|---|---|---|---|

## Konsekuensi
<!-- Positif, negatif, risiko, biaya operasional, dampak ke modul/kontrak/migrasi -->

## Penegakan & verifikasi
<!-- Lint/CI/test yang memastikan keputusan diikuti; kriteria untuk meninjau ulang -->
```
