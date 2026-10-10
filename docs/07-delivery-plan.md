# 07 — Delivery Plan

## MesinProspek — Roadmap, Epic, Backlog, Rencana Sprint

| Field | Value |
|---|---|
| Dokumen | Delivery plan (roadmap, milestone, epic, backlog, sprint plan) |
| Versi | 0.1 (Draft) |
| Tanggal | 2026-10-10 |
| Basis | `01-prd.md` v0.1 (**sumber kebenaran** prioritas & release), `00-analisis-dokumen-awal.md` (D-01..D-21, Q-01..Q-10), `03-ux-spec.md` (SCR/FLOW), `04-technical-design.md` (modul, ADR), `05-test-strategy.md` (§5 entry/exit, §8 traceability), `06-ci-cd.md` (QG, environment, bootstrap) |
| Status crawler | **PAUSED** (D-01). Tidak ada story crawler. |
| Platform | GitLab CE self-hosted (D-18), dedicated server di DC Indonesia (D-19), Go + Next.js (D-21), object storage SeaweedFS (04 ADR-18) |
| Audience | Founder, Product, Tech lead (juga Engineering, QA, Design, DevOps) |

> **Konvensi.** `M-n` milestone, `EP-nn` epic, `US-<AREA>-nnn` story (AREA = area FR PRD + `PLAT` untuk platform/infra/kualitas/rilis), `SPR-nn` sprint (2 minggu), `Q-D-nn` pertanyaan terbuka dokumen ini, `RD-nn` risiko delivery. **SP** = story point (1 SP ≈ 1 hari-ideal engineer). **Lane** = kelompok kapasitas: `GO` (Go backend/data), `FE` (Next.js), `QA`, `OPS` (DevOps).
>
> **Semua tanggal adalah estimasi** yang bergantung pada konfirmasi ukuran tim dan tanggal mulai (Q-D-01). Timeline dinyatakan dalam sprint. Tanggal contoh memakai asumsi **SPR-00 dimulai Senin, 2 November 2026**. Libur nasional dan cuti bersama 2027 masih perkiraan dan perlu dicek ulang dengan SKB resmi.

### Ringkasan

| Item | Nilai (skenario baseline) |
|---|---|
| Milestone | M-0 Bootstrap (akhir SPR-00) → M-1 MVP 0 internal alpha (SPR-06) → M-2 MVP 1 closed beta wave 1 (SPR-12) → M-3 MVP 1 early paid (SPR-17) → M-4 V1 (± SPR-24) |
| Durasi ke early paid | 18 sprint = **36 minggu** (contoh: 2 Nov 2026 → 9 Jul 2027) |
| Backlog MVP 0 + MVP 1 | **190 story** (181 inti + 9 stretch P1), **756 SP inti** + 27 SP stretch, 30 epic MVP + 8 epic V1 dan sesudahnya |
| Cakupan PRD | **92/92 FR P0** tertutup story (25 khusus MVP 0, 66 khusus MVP 1, FR-AUTH-006: ops di MVP 0, tenant di MVP 1). 10/10 FR P1 MVP 1 tertutup sebagai stretch |
| Jalur kritis | Lane **Go** (420 SP) + jawaban dataset (Q-01..Q-05) + pengadaan server (Q-09) + perekrutan beta (Q-07) + `app-02` (R-08) |
| Tim baseline | 4 Go (termasuk Tech lead 50%), 2 Next.js, 1 QA, 0,5 DevOps (1,0 di SPR-00..01 dan SPR-14..17), 0,5 product designer, 1 PM/founder |

---

## 1. Pendekatan delivery

### 1.1 Prinsip

| # | Prinsip | Penerapan |
|---|---|---|
| P-1 | **Vertical slice** | Setiap sprint ditutup dengan demo ujung-ke-ujung di staging: data → API → UI → event/audit. Story API dan UI dipasangkan dalam slice yang sama. Bila UI dikerjakan lebih dulu (P-6), slice dianggap selesai saat story API-nya selesai. |
| P-2 | **Trunk-based + feature flag** | Satu branch `main`, branch fitur ≤ 2 hari (06 §1). Pekerjaan yang belum selesai masuk `main` di balik flag (`US-PLAT-014`, 06 §13), sehingga wave beta diaktifkan per workspace tanpa branch rilis. |
| P-3 | **Sprint 2 minggu** | Senin minggu 1 sampai Jumat minggu 2. Planning, review, dan retro di batas sprint (§10). |
| P-4 | **Sprint 0 = bootstrap** | Server (Ansible), GitLab CE + runner, skeleton monorepo Go + Next.js, pipeline MR, baseline DB + migrasi, observability, paket design tokens (06 §18 langkah 1–11). |
| P-5 | **Data-first** | MVP 0 membangun fondasi data dan engine intelligence tanpa UI customer (PRD §6). Jalur kritis ada di lane Go. Lane FE memakai MVP 0 untuk design system, ops console, dan UI-first. |
| P-6 | **Contract-first & UI-first** | Kontrak OpenAPI ditulis lebih dulu (ADR-08). FE boleh membangun layar terhadap mock kontrak sebelum API siap. Syaratnya: kontrak di-review TL, jarak UI→API ≤ 4 sprint, dan integrasi + E2E menjadi DoD story API (§9). Di backlog ditandai `↗nn`. |
| P-7 | **Beta-driven** | Closed beta dimulai saat *core loop* siap (wave 1). Fitur berikutnya diaktifkan bertahap lewat flag (wave 2). Feedback beta (FR-INTL-010, FR-DATA-014, FR-AI-007) masuk refinement mingguan. |
| P-8 | **Gate berbasis bukti** | Milestone lulus bila exit criteria 05 §5.2 dan RG-1..RG-7 (PRD §12.1) terbukti lewat QG di 06 §8, bukan karena tanggal. |
| P-9 | **Docs-as-code** | Perubahan scope, keputusan, dan kontrak diperbarui di `docs/` lewat MR yang sama (§10.3). |

### 1.2 Release → milestone → gate

| Release PRD | Milestone | Gate yang wajib lulus | Siapa yang memakai |
|---|---|---|---|
| — | **M-0** Bootstrap | Pipeline MR & main hijau, staging bisa di-deploy (06 §18) | Tim internal |
| MVP 0 (internal alpha) | **M-1** | Kolom MVP 0 di 05 §5.2: TC P0 FR MVP0 (termasuk MFA ops, FR-AUTH-006), RG-2 (evidence coverage), RG-5 parsial, RG-6 penuh, RG-7 (RLS), determinisme | Ops & founder lewat ops console + score inspector, memakai workspace internal + fixture ICP/Offering buatan ops (PRD §6) |
| MVP 1 (closed beta) | **M-2** Wave 1 | Subset (§3.1): RG-4, RG-6, RG-7 inti (QG-09, 0 Critical/High), backup + drill restore pertama, suppression aktif | 5 workspace beta |
| MVP 1 (early paid, sellable) | **M-3** | **RG-1..RG-7 lengkap** (QG-32) + kolom MVP 1 di 05 §5.2 + `app-02` aktif (R-08) | 5–10 workspace beta → berbayar |
| V1 | **M-4** | Acceptance FR V1 (PRD §6), RG tetap terjaga | Customer berbayar |

### 1.3 Wave closed beta (usulan, Q-D-10)

| Wave | Aktif | Isi (ringkas) | Alasan |
|---|---|---|---|
| Wave 1 | Akhir SPR-12 (M-2) | Auth/workspace/undangan, Offering + ICP + onboarding, search + saved search + bulk select, profil (why-panel, evidence, kontak + unlock, kartu), list/segment/tag/catatan/assign, credit & entitlement, export, peta wilayah, feedback, laporkan data, refresh, suppression, kill switch | *Core loop* PER-01/PER-02: Find → Understand → Qualify → Prioritize → Act (export/assign) |
| Wave 2 | SPR-14..SPR-15 (per flag, setelah story selesai) | Territory ranking & detail area, webhook, AI brief, upload customer, refund otomatis INVALID, DSR intake, penutupan workspace, koreksi ops, akses support | PER-03 (manager), integrasi, dan fitur yang bergantung pada D-15 (AI) dan egress proxy |

PRD §6 tidak mengatur apakah closed beta wajib memuat seluruh P0 MVP 1. Rencana ini memakai wave dan menunggu konfirmasi Product (Q-D-10). Gate M-3 tetap menuntut seluruh P0.

---

## 2. Asumsi tim & kapasitas

### 2.1 Tim baseline (usulan)

| Peran | FTE | Lane | SP efektif / sprint | Catatan |
|---|--:|---|--:|---|
| Tech lead (Go, hands-on) | 1 | GO | 5 | 50% coding. Sisanya arsitektur, review, keputusan ADR, Maintainer deploy (Q-CI-07) |
| Go backend engineer | 2 | GO | 20 | API, platform, billing, export/webhook, AI |
| Go data/pipeline engineer | 1 | GO | 10 | Input Contract, normalisasi, ER, scoring, geo |
| Next.js frontend engineer | 2 | FE | 20 | UI customer & ops console, design system |
| QA engineer (otomasi, bisa Go/TS) | 1 | QA | 5 | ±70% waktu untuk pengujian story (masuk estimasi story); 5 SP untuk story QA (golden set, harness, usability, soak) |
| DevOps/SRE | 0,5 (1,0 di SPR-00..01 & SPR-14..17) | OPS | 5 (10) | Server, GitLab, backup/DR, egress, HA `app-02` |
| Product designer | 0,5 | — | — | Dual-track, satu sprint di depan. Tidak dihitung SP |
| PM (dapat dirangkap founder) | 1 | — | — | Backlog, beta, keputusan produk |
| Founder | — | — | — | Data & lisensi, server, beta customer, pricing |

Baseline ini menambah satu Go engineer dibanding contoh di brief (2 Go backend + 1 Go data). Alasannya, lane Go adalah jalur kritis (420 SP, 56% backlog inti). Tanpa tambahan itu, early paid mundur ±5 sprint (§2.4 skenario B).

### 2.2 Model kapasitas

- **Kalibrasi awal:** 10 SP per engineer full-time per sprint, setelah ceremony, review, dan support. Nilai ini diukur ulang setelah SPR-02 dari velocity aktual (Q-D-02). Bila meleset > 15%, rencana disusun ulang.
- **Commit = 85% kapasitas** (buffer 15% untuk bug, support beta, dan pekerjaan tak terduga): GO 30, FE 17, QA 5, OPS 4 (8 saat DevOps 1,0 FTE).
- **Faktor kalender (perkiraan):** SPR-03 ×0,8 (Natal), SPR-04 ×0,8 (Tahun Baru), SPR-09 ×0,6 (Idul Fitri 1448 H dan cuti bersama), SPR-14 ×0,85 (Idul Adha, Waisak). Libur lain diserap buffer.
- **Sprint hardening (SPR-16..SPR-17):** batas per lane tidak berlaku. Engineer membantu QA dan DevOps (security review, drill, perbaikan bug).
- Kelebihan ≤ 2 SP di satu lane (SPR-02, SPR-03..05, SPR-06, SPR-10) diserap buffer atau dibantu lane GO (untuk OPS).

### 2.3 Kapasitas vs beban per sprint

| Sprint | Tanggal (contoh) | Faktor kalender | Kapasitas commit GO / FE / QA / OPS | Terencana GO / FE / QA / OPS | Total terencana (stretch) |
|---|---|--:|---|---|---|
| SPR-00 | 02 Nov – 13 Nov 2026 | 1.00 | 30 / 17 / 5 / 8 = 60 | 16 / 6 / 0 / 16 | 38 (0) |
| SPR-01 | 16 Nov – 27 Nov 2026 | 1.00 | 30 / 17 / 5 / 8 = 60 | 30 / 14 / 5 / 8 | 57 (0) |
| SPR-02 | 30 Nov – 11 Des 2026 | 1.00 | 30 / 17 / 5 / 4 = 56 | 31 / 13 / 6 / 2 | 52 (0) |
| SPR-03 | 14 Des – 25 Des 2026 | 0.80 | 24 / 14 / 4 / 3 = 45 | 24 / 10 / 5 / 0 | 39 (0) |
| SPR-04 | 28 Des – 08 Jan 2027 | 0.80 | 24 / 14 / 4 / 3 = 45 | 23 / 13 / 5 / 5 | 46 (0) |
| SPR-05 | 11 Jan – 22 Jan 2027 | 1.00 | 30 / 17 / 5 / 4 = 56 | 29 / 15 / 5 / 5 | 54 (0) |
| SPR-06 | 25 Jan – 05 Feb 2027 | 1.00 | 30 / 17 / 5 / 4 = 56 | 29 / 18 / 3 / 3 | 53 (0) |
| SPR-07 | 08 Feb – 19 Feb 2027 | 1.00 | 30 / 17 / 5 / 4 = 56 | 28 / 16 / 0 / 3 | 47 (0) |
| SPR-08 | 22 Feb – 05 Mar 2027 | 1.00 | 30 / 17 / 5 / 4 = 56 | 30 / 16 / 0 / 3 | 49 (0) |
| SPR-09 | 08 Mar – 19 Mar 2027 | 0.60 | 18 / 10 / 3 / 2 = 33 | 18 / 10 / 0 / 0 | 28 (0) |
| SPR-10 | 22 Mar – 02 Apr 2027 | 1.00 | 30 / 17 / 5 / 4 = 56 | 29 / 17 / 0 / 5 | 51 (0) |
| SPR-11 | 05 Apr – 16 Apr 2027 | 1.00 | 30 / 17 / 5 / 4 = 56 | 27 / 16 / 3 / 3 | 49 (0) |
| SPR-12 | 19 Apr – 30 Apr 2027 | 1.00 | 30 / 17 / 5 / 4 = 56 | 24 / 12 / 2 / 0 | 38 (0) |
| SPR-13 | 03 Mei – 14 Mei 2027 | 1.00 | 30 / 17 / 5 / 4 = 56 | 29 / 17 / 5 / 0 | 51 (0) |
| SPR-14 | 17 Mei – 28 Mei 2027 | 0.85 | 26 / 14 / 4 / 7 = 51 | 24 / 10 / 3 / 0 | 37 (0) |
| SPR-15 | 31 Mei – 11 Jun 2027 | 1.00 | 30 / 17 / 5 / 8 = 60 | 27 / 16 / 0 / 8 | 51 (14) |
| SPR-16 | 14 Jun – 25 Jun 2027 | 1.00 | 30 / 17 / 5 / 8 = 60 | 13 / 8 / 11 / 5 | 37 (13) |
| SPR-17 | 28 Jun – 09 Jul 2027 | 1.00 | 30 / 17 / 5 / 8 = 60 | 0 / 0 / 6 / 0 | 6 (0) |

Kolom "Total terencana" memuat story stretch dalam kurung. Stretch hanya diambil bila story inti sprint tersebut sudah *done*. Beban rendah di SPR-16..SPR-17 disengaja: kapasitas dipakai untuk perbaikan temuan usability, security, dan performa sebelum go/no-go.

### 2.4 Sensitivitas timeline terhadap ukuran tim

Dihitung dari beban lane Go (166 SP sampai M-1, 322 SP sampai M-2, 396 SP sampai feature complete) dibagi commit per sprint, ditambah ±1 sprint efek libur dan 2 sprint hardening. Lane FE tidak menjadi pembatas di skenario mana pun.

| Skenario | Komposisi | Commit GO / FE | M-1 | M-2 | M-3 (minggu sejak SPR-00) | Catatan |
|---|---|---|---|---|---|---|
| **A. Baseline** | §2.1 | 30 / 17 | SPR-06 | SPR-12 | **SPR-17 (36 mgg, 9 Jul 2027)** | Rencana di dokumen ini |
| B. Contoh brief | Tech lead merangkap salah satu dari 2 Go backend (3 orang Go), lain sama | 21 / 17 | SPR-09 | SPR-17 | SPR-22 (46 mgg, 17 Sep 2027) | FE punya slack ±25%, dipakai untuk E2E dan a11y |
| C. Tim minimal | 3 engineer: 1 Go (TL + DevOps), 1 Go data, 1 FE full-stack. Founder merangkap PM + QA | 12 / 8,5 | SPR-15 | SPR-28 | ± SPR-36 (74 mgg, Mar 2028) | Tidak direkomendasikan tanpa negosiasi scope P0 (Q-D-19) |
| D. Dipercepat | A + 1 Go backend (mulai SPR-02) + 1 FE (SPR-07..15) | 38 / 25 | SPR-05 | SPR-10 | SPR-14 (30 mgg, 28 Mei 2027) | Dibatasi dependensi eksternal (dataset, server, beta). Kompresi lebih jauh tidak realistis |
| E. Velocity −20% | Tim A, velocity aktual 8 SP/engineer | 24 / 14 | SPR-08 | SPR-15 | SPR-20 (42 mgg, 20 Agu 2027) | Pemicu replan otomatis setelah SPR-02 (Q-D-02) |

---

## 3. Milestone & timeline

### 3.1 Milestone

| ID | Milestone | Target | Entry criteria | Exit criteria (bukti) |
|---|---|---|---|---|
| **M-0** | Bootstrap | Akhir SPR-00 (13 Nov 2026) | Server `ops-01`, `ci-01`, `stg-01` tersedia (Q-09). Keputusan migrasi repo (Q-08), domain (Q-CI-08), dan D-18 dikonfirmasi. Tim lengkap untuk SPR-00 | GitLab CE + runner aktif (2FA). Skeleton Go + Next.js ter-build. Pipeline MR hijau dengan QG-01..05 dan QG-18, merge diblokir `mr:gate`. Baseline goose + linter RLS. `make dev` jalan. Telemetri masuk Grafana. Paket design tokens terbit. Deploy staging boleh selesai di SPR-01 (`US-PLAT-011`) |
| **M-1** | MVP 0 internal alpha | Akhir SPR-06 (5 Feb 2027) | Snapshot dataset nyata (atau subset berlisensi), referensi wilayah, populasi, dan batas wilayah tersedia (Q-01..Q-05, Q-10). Golden set `gs-er`, `gs-cls`, `gs-geo` berlabel. D-20 diputuskan (SPR-01) | Kolom **MVP 0** 05 §5.2: semua TC P0 FR MVP0 lulus, termasuk FR-AUTH-006 untuk akun ops (login ops tanpa MFA ditolak). RG-2: evidence coverage ≥ 95%. RG-5: drill alert pipeline failure & backlog. **RG-6 penuh** (ER P ≥ 98% & R ≥ 90%, klasifikasi ≥ 90%, kelurahan ≥ 95%, ≥ 10 EC). RG-7: linter + suite RLS. TC-INTL-015/016 & TC-GEO-008 lulus, `priority-1.2.0` & `geo-1.0.0` dibekukan. TC-DATA-034 terukur (waiver TL bila meleset). 0 S1/S2. NFR-MAINT-01 untuk normalisasi/ER/scoring |
| **M-2** | MVP 1 closed beta (wave 1) | Akhir SPR-12 (30 Apr 2027) | M-1. Story wave 1 *done*. ≥ 5 workspace beta menandatangani perjanjian beta + pemberitahuan privasi (Q-07, Q-D-08). Credit awal beta disepakati (Q-D-13) | RG-7 inti: QG-09 hijau sejak SPR-07, 0 Critical/High (QG-19/27). RG-4: suite idempotensi + rekonsiliasi nightly 100%. RG-6 terjaga di snapshot aktif. Produksi: backup pgBackRest berjalan + drill restore pertama di `dr-01` (QG-35). Runbook + latihan on-call (`US-PLAT-031`). Suppression aktif. DSR ditangani manual (email + register) sampai `US-AUD-003`. Temuan S1/S2 usability formatif (SPR-11) ditutup. Flag wave 2 nonaktif |
| **M-3** | MVP 1 early paid (sellable) | Akhir SPR-17 (9 Jul 2027) | M-2. Feature freeze akhir SPR-15. Build RC di staging | **QG-32**: issue Release Readiness memuat bukti RG-1..RG-7. Kolom **MVP 1** 05 §5.2 lengkap: usability T2/T3 ≥ 4/5 + median TTFR ≤ 5 menit, E2E export & webhook + soak 24 jam ≥ 99%, rekonsiliasi 100%, drill 4 alert, sampel merge produksi n ≥ 200 ≥ 98%, ZAP full tanpa High, PERF-01..07, drill restore & rollback, axe 0 serious/critical, 0 S1/S2, S3 ≤ 10, sign-off UAT ≥ 3 workspace. **`app-02` aktif + drill failover** (R-08). Pricing & proses invoice manual siap (D-09). Tag `v1.0.0` |
| **M-4** | V1 | ± Akhir SPR-24 (15 Okt 2027), rentang SPR-23..26 | M-3. Target CRM dipilih dari survei beta (D-10). Payment gateway dipilih. Kadens update dataset diketahui (Q-03) untuk signal | Acceptance FR V1 (PRD §6, EP-31..EP-36). Regresi + RG tetap hijau. Availability 99,5% bulanan selama ≥ 2 bulan (NFR-AVAIL-01) |

### 3.2 Gantt per sprint

`##` = ada story inti di sprint itu, `s.` = hanya stretch, `..` = tidak ada. Isi V1 dirinci di §7.4.

```
Sprint                      00 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 17 | 18-24
Milestone                   M0                M1                M2             M3 |    M4
Fase                        B0 [----- MVP 0 ----][-- MVP 1 build: W1 > W2 -][hard]| V1
EP-01/02 Bootstrap, CI/CD   ## ## .. .. .. ## .. ## .. .. .. .. .. .. .. .. .. .. |
EP-03/04 Observ., DR        ## .. .. .. ## .. ## .. ## .. .. ## .. .. .. ## .. .. |
EP-05 Security              .. .. .. .. .. .. .. .. .. .. ## .. .. .. .. .. ## ## |
EP-06 Design system & FE    ## ## ## ## .. .. .. .. .. .. .. .. .. .. .. .. .. .. |
EP-07 Test & golden set     .. ## ## ## ## ## ## .. .. .. .. .. .. ## .. .. .. .. |
EP-08 HA app-02             .. .. .. .. .. .. .. .. .. .. .. .. .. .. .. .. ## .. |
EP-09 Input & snapshot      .. ## ## .. .. .. ## .. .. .. .. .. .. .. .. .. .. .. |
EP-10 Norm/verif/evidence   .. .. ## ## .. .. .. .. .. .. .. .. .. .. .. s. .. .. |
EP-11 Entity resolution     .. .. .. ## ## ## .. .. .. .. .. .. .. .. .. .. .. .. |
EP-12 Geo ref & metrik      .. ## .. ## ## ## .. .. .. .. .. .. .. .. .. .. .. .. |
EP-13 Faktor & opportunity  .. .. ## ## ## ## ## .. .. .. .. .. .. .. .. .. .. .. |
EP-14 Scoring & ICP fit     .. .. .. .. .. ## ## .. .. .. .. .. .. .. .. .. .. .. |
EP-15 Ops console dasar     .. ## .. ## ## ## ## .. .. .. .. .. .. .. .. .. .. .. |
EP-16 Auth/RBAC/RLS         .. .. ## .. .. ## .. ## .. .. ## .. ## .. .. .. .. .. |
EP-17 Offering/ICP/onb.     .. .. .. .. .. .. ## ## ## ## .. .. .. .. .. .. .. .. |
EP-18 Search                .. .. .. .. ## .. .. ## ## ## ## ## .. .. .. .. .. .. |
EP-19 Profil & trust UI     .. .. .. .. .. ## ## .. ## ## .. .. .. .. ## s. .. .. |
EP-20 Location intel.       .. .. .. .. .. .. .. .. .. .. .. ## ## ## .. .. s. .. |
EP-21 List/segment/assign   .. .. .. .. .. .. .. .. .. .. ## ## .. .. .. s. .. .. |
EP-22 Credit & unlock       .. .. .. .. .. .. .. .. ## ## ## ## ## ## .. .. s. .. |
EP-23/24 Export, webhook    .. .. .. .. .. .. .. .. .. .. .. ## .. ## ## .. .. .. |
EP-25 AI brief              .. .. .. .. .. .. .. .. .. .. .. .. .. ## ## s. s. .. |
EP-26/27 Feedback, upload   .. .. .. .. .. .. .. .. .. ## ## .. ## ## ## .. .. .. |
EP-28/29 Privasi, ops MVP1  .. .. .. .. .. .. .. ## .. .. ## ## ## ## .. ## s. .. |
EP-30 Beta & RG evidence    .. .. .. .. .. .. .. .. .. .. .. ## ## .. .. .. ## ## |
EP-31..36 V1 (ringkas)      .. .. .. .. .. .. .. .. .. .. .. .. .. .. .. .. .. .. | ######
Closed beta                                                        == == == == == | ==> paid
```

### 3.3 Kesiapan release gate per milestone

| Gate | M-1 | M-2 | M-3 (bukti final) |
|---|---|---|---|
| RG-1 Relevansi | — | Prototype formatif (`US-PLAT-042`, SPR-03) + uji formatif beta (`US-PLAT-032`, SPR-11). Event TTFR terpasang (`US-ICP-008`) | Usability sumatif 5 peserta (`US-PLAT-037`) |
| RG-2 Kepercayaan | Evidence coverage ≥ 95% (`US-DATA-006`, `US-OPS-004`) | Evidence drawer live (`US-PROF-003`) | Usability T3 (`US-PLAT-037`) + Data Health |
| RG-3 Aksi | — | E2E select → export (`US-EXP-004`, `US-PLAT-025`) | + select → webhook + soak 24 jam (`US-PLAT-036`) |
| RG-4 Usage | — | Suite idempotensi + rekonsiliasi (`US-BILL-002`, `US-BILL-009`) | + refund otomatis (`US-BILL-007`), laporan billing |
| RG-5 Operasional | Alert pipeline failure & backlog (`US-PLAT-026`) | + on-call & runbook (`US-PLAT-031`) | Drill 4 alert (`US-PLAT-033`) |
| RG-6 Kualitas data | Laporan holdout penuh (`US-PLAT-027`) | QG-26 nightly | + sampel merge produksi n ≥ 200 (`US-PLAT-039`) |
| RG-7 Keamanan | Linter & suite RLS (`US-PLAT-007`) | QG-09 penuh (`US-AUTH-005`) + 0 Critical/High | Security review + retest (`US-PLAT-038`, `US-PLAT-048`), suite SSRF (`US-EXP-005/006`, `US-PLAT-030`) |

---

## 4. Critical path & dependencies

### 4.1 Jalur kritis

```
Q-09 server ─► SPR-00 GitLab/CI/skeleton ─► SPR-01 Input Contract + import (Q-01/Q-05 sampel)
  ─► SPR-02 normalisasi/evidence/verifikasi + MFA ops (D-20) ─► SPR-03..05 ER + golden set (Q-D-03) ─► geo metrik (Q-04/Q-10)
  ─► SPR-05..06 opportunity + ICP fit + priority score ─► [M-1]
  ─► SPR-07 auth/RLS ─► SPR-08..09 projection + search + profil ─► SPR-08..10 ledger/unlock (D-08)
  ─► SPR-11 export ─► SPR-12 suppression + refresh ─► [M-2] beta wave 1 (Q-07)
  ─► SPR-13..14 webhook (egress) + AI (D-15) + upload ─► SPR-15 privasi (legal) + drill
  ─► SPR-16 app-02 (R-08) + usability + soak + security ─► SPR-17 [M-3]
```

Lane FE punya *float* ±2 sprint di MVP 0 (dipakai untuk UI-first). Lane Go tidak punya float. Keterlambatan lane Go langsung menggeser M-1..M-3.

### 4.2 Dependensi eksternal

| # | Dependensi | Memblokir | Default bila terlambat | Keputusan dibutuhkan paling lambat | Owner |
|---|---|---|---|---|---|
| 1 | **Q-09** spesifikasi & jumlah server, lokasi off-site (+ Q-CI-04) | EP-01 (`ops-01`, `ci-01`, `stg-01`), EP-04 produksi (`US-PLAT-023`) | Mulai di laptop + Docker Compose. Sewa sementara 1 server di DC Indonesia untuk GitLab | `ops-01`/`ci-01`/`stg-01`: sebelum SPR-00. Produksi data tier: SPR-03 | Founder, DevOps |
| 2 | **Q-08 / Q-CI-01** migrasi repo GitHub → GitLab | `US-PLAT-002` | GitLab primary, mirror GitHub read-only | Sebelum SPR-00 | Founder, TL |
| 3 | **Q-01** skema & fill-rate dataset, **Q-02** volume & cakupan | `US-DATA-001` (mapping), `US-INTL-001`, `US-INTL-004` (tipe opportunity aktif), sizing | `msp-synth` dengan profil fill-rate asumsi. Mapping disesuaikan saat data tiba | Sampel + kamus data: **SPR-01**. Dump penuh: SPR-04 | Founder (pemilik data) |
| 4 | **Q-05** lisensi tampil/export/staging/golden set (+ Q-CI-09) | `US-DATA-002` (tag lisensi), `US-PLAT-021` (golden set), export (EP-23) | Konservatif: export hanya field kanonik/derived. Golden set dari sampel yang diizinkan | SPR-01 (staging/golden set). SPR-10 (aturan export) | Founder, legal |
| 5 | **Q-04** sumber & periode populasi, sistem kode (D-13) | `US-DATA-013`, `US-GEO-001` | Kemendagri + crosswalk BPS. Fallback kecamatan | SPR-01 | Founder, data eng |
| 6 | **Q-10** poligon batas wilayah + lisensi | `US-DATA-014` (PIP), `US-GEO-003/004` (choropleth) | Pencocokan alamat + titik/cluster + tabel (OQ-UX-18) | SPR-02 (PIP). SPR-10 (tile peta) | Founder |
| 7 | **Q-03** kadens update dataset | Refresh (`US-MON-001`), signal V1 (EP-31) | Kuartalan, publish manual | SPR-10 (MVP 1). SPR-18 (V1) | Founder |
| 8 | **Q-06** ground truth revenue + TQ-08 tier kota | `US-INTL-003` | Prior manual per kategori × tier kota (`city_tier.yaml`) | SPR-03 | Product |
| 9 | **D-20** identity provider (TQ-01) | MFA ops MVP 0 (`US-PLAT-041`, `US-AUTH-001`, SPR-02), `US-AUTH-002` (UI, SPR-05) | Ory Kratos self-hosted | **SPR-01** | TL, Founder |
| 10 | **D-15** penyedia AI + DPA/region (TQ-02) + budget sandbox (OQ-T-02) | EP-25 (`US-AI-001`, SPR-13) | Abstraksi provider + fake provider. Wave 2 tanpa AI bila belum ada | **SPR-11** | Founder, TL, legal |
| 11 | **D-17** brand final | Swap token `color.accent.*` (tanpa ubah komponen) | Placeholder indigo (02 §README) | SPR-10 (sebelum beta). Paling lambat SPR-15 | Founder, Design |
| 12 | **D-08 / D-09** model credit & plan/harga | EP-22 (SPR-08..10), entitlement, pricing M-3 | Lampiran E PRD sebagai angka beta | D-08: SPR-07. Harga: SPR-15 | Founder, PM |
| 13 | Kapasitas **labeling golden set** (OQ-T-01, Q-D-03) | `US-PLAT-021/022`, RG-6 di M-1 | Mulai 1.000 pasangan ER. QA + 1 orang Product/ops | Nama pelabel: SPR-01 | PM |
| 14 | **Q-07** 5–10 customer beta + Offering mereka (D-16) | Uji formatif SPR-11, M-2, UAT, RG-1 | Rekrut dari jaringan founder. Wave 1 minimal 5 | Shortlist: SPR-08. Perjanjian: SPR-11 | Founder, PM |
| 15 | **TQ-17 / Q-CI-13** PoC SeaweedFS (object lock, enkripsi, pgBackRest S3, `filer.sync`) | `US-PLAT-040`, EP-04 | Garage. Fitur yang tidak didukung diganti salinan append-only off-site + LUKS | SPR-01 | DevOps |
| 16 | **TQ-12** provider email transaksional (Q-D-05) | Verifikasi email (`US-AUTH-002`), undangan (`US-AUTH-006`) | SMTP relay di Indonesia | SPR-04 | TL |
| 17 | **TQ-11** basemap (Q-D-06) | `US-GEO-004` | Tanpa basemap atau tile OSM self-hosted | SPR-10 | Design, TL |
| 18 | **Konsultan hukum** (SLA DSR, retensi, pseudonimisasi, ToS/DPA, perjanjian beta) (Q-D-08) | M-2 (perjanjian beta), `US-AUD-003/004` (SPR-15), M-3 | Default 05/04 (OQ-T-15, TQ-14) | Dokumen beta: SPR-11. Aturan DSR/retensi: SPR-14 | Founder |
| 19 | **Hardware `app-02` + VIP/LB** (R-08, Q-CI-05, Q-D-09) | `US-PLAT-045/046` (SPR-16) → **M-3** | Tidak ada: M-3 tidak boleh lulus tanpa `app-02` | Pesan: **SPR-12**. Siap pakai: SPR-15 | Founder, DevOps |
| 20 | Server perf setara produksi (OQ-T-03, Q-D-04) | TC-DATA-034 (SPR-06), PERF-01..07 (SPR-16) | Faktor skala didokumentasikan + k6 sekali di hardware produksi | SPR-04 | DevOps |

---

## 5. Epic

Ukuran: **S** ≤ 15 SP, **M** 16–30, **L** 31–45, **XL** > 45. Epic V1 ke atas memakai estimasi kasar (belum dipecah menjadi story).

| ID | Epic | FR / NFR | Release | Pri | Ukuran (SP inti) | Dependensi |
|---|---|---|---|---|---|---|
| EP-01 | Bootstrap infrastruktur, repo & fondasi platform | NFR-SEC-02/04, NFR-PRIV-03, FR-AUD-006 (fondasi) | M-0 | P0 | L (33) | Q-09, Q-08, TQ-17 |
| EP-02 | CI/CD & quality gate | QG-01..37, NFR-SEC-06 | M-0..M-1 | P0 | M (23) | EP-01 |
| EP-03 | Observability & alerting | NFR-OBS-01, RG-5 | M-0..M-3 | P0 | S (11) | EP-01 |
| EP-04 | Backup, DR & runbook | NFR-AVAIL-02, BR-PRIV-03 | M-1..M-3 | P0 | M (16) | Q-09, Q-CI-04, EP-01 |
| EP-05 | Security hardening & kontrol egress | NFR-SEC-01/05/06, RG-7 | M-2..M-3 | P0 | S (13) | EP-02 |
| EP-06 | Design system & fondasi frontend | NFR-A11Y-01, NFR-L10N-01, BR-TRUST | M-0..M-1 | P0 | L (32) | `02-design-system`, D-17 |
| EP-07 | Test harness, data sintetis & golden set | RG-6, PRD §12.2 | M-1 | P0 | L (34) | Q-D-03, Q-05 |
| EP-08 | HA aplikasi `app-02` | R-08, NFR-AVAIL-01/02 | M-3 | P0 | S (8) | Q-D-09, Q-CI-05 |
| EP-09 | Input Contract & import snapshot | FR-DATA-001, 007 | MVP0 | P0 | M (23) | Q-01, Q-02, Q-05 |
| EP-10 | Normalisasi, verifikasi, evidence & digital lite | FR-DATA-002, 005, 006, 010 (011 stretch) | MVP0 | P0 | M (23) | EP-09 |
| EP-11 | Entity resolution & keputusan merge | FR-DATA-003, 004, 007 | MVP0 | P0 | M (21) | EP-10, golden set |
| EP-12 | Referensi wilayah, penetapan area & metrik geo | FR-DATA-008, 009, FR-GEO-001 | MVP0 | P0 | M (23) | Q-04, Q-10 |
| EP-13 | Faktor intelligence & opportunity | FR-INTL-001..007 | MVP0 | P0 | L (31) | EP-10, EP-12, Q-06 |
| EP-14 | Scoring engine & ICP fit | FR-ICP-003, FR-INTL-008, 009, 011 | MVP0 | P0 | M (23) | EP-13 |
| EP-15 | Ops console dasar | FR-OPS-001, 002, 005 | MVP0 | P0 | L (35) | EP-09, EP-11 |
| EP-16 | Auth, workspace, RBAC & isolasi tenant | FR-AUTH-001..006, FR-AUD-006 | MVP0 (MFA ops) + MVP1 | P0 | L (37) | D-20, TQ-12 |
| EP-17 | Offering, ICP & onboarding | FR-ICP-001, 002, 004, 005, 006 | MVP1 | P0 | M (30) | EP-14, EP-16 |
| EP-18 | Search & discovery | FR-SRCH-001..008 | MVP1 | P0 | XL (47) | EP-14, EP-17 |
| EP-19 | Profil prospek & UI trust | FR-PROF-001..009 (010 stretch) | MVP1 | P0 | L (34) | EP-18, EP-06 |
| EP-20 | Location intelligence & territory | FR-GEO-002, 003, 004, 009 (005 stretch) | MVP1 | P0 | M (23) | EP-12, Q-10, TQ-11 |
| EP-21 | List, segment, tag, catatan, assignment | FR-SEG-001..005 (006 stretch) | MVP1 | P0 | M (22) | EP-18 |
| EP-22 | Usage, credit, unlock & entitlement | FR-BILL-001..006 (007 stretch) | MVP1 | P0 | L (39) | D-08, D-09 |
| EP-23 | Export | FR-EXP-001, 002, 005, 006 | MVP1 | P0 | S (15) | EP-22 |
| EP-24 | Webhook | FR-EXP-003, 004, 005 | MVP1 | P0 | M (23) | EP-05 (egress), EP-23 |
| EP-25 | AI brief | FR-AI-001, 003..006 (002, 007 stretch) | MVP1 | P0 | M (27) | D-15, TQ-02 |
| EP-26 | Refresh, feedback & laporan data | FR-MON-001, FR-INTL-010, FR-DATA-014 | MVP1 | P0 | M (17) | EP-19, EP-22 |
| EP-27 | Upload customer | FR-DATA-012 | MVP1 | P0 | S (15) | EP-16, EP-11 |
| EP-28 | Audit, privasi & governance | FR-AUD-001, 003, 004, 005 (002 stretch) | MVP1 | P0 | M (26) | Konsultan hukum |
| EP-29 | Ops console MVP 1 | FR-OPS-006..009 (003, 004 stretch) | MVP1 | P0 | L (33) | EP-22, EP-28 |
| EP-30 | Program beta & release readiness | RG-1..RG-7, PRD §5 | M-2..M-3 | P0 | M (19) | Q-07 |
| EP-31 | Signal & change detection | FR-DATA-013, FR-INTL-012, FR-PROF-011 | V1 | P1 | L (±40) | Q-03 (snapshot kedua) |
| EP-32 | Watch, alert, refresh terjadwal, notifikasi | FR-MON-002..004, FR-GEO-008 | V1 | P1/P2 | L (±45) | EP-31 |
| EP-33 | Compare & laporan | FR-PROF-012, FR-GEO-006, 007, FR-RPT-001 | V1 | P1 | L (±35) | — |
| EP-34 | NL search & peta hasil | FR-SRCH-009, 010 | V1 | P1 | L (±35) | D-15 |
| EP-35 | Integrasi CRM & API key | FR-EXP-007, FR-AUTH-007 | V1 | P1 | L (±40) | D-10 (survei beta) |
| EP-36 | Billing self-serve, retensi, antrean kampanye | FR-BILL-008, FR-AUD-007, FR-SEG-007 | V1 | P1 | L (±40) | Payment gateway |
| EP-37 | Bobot kustom & team workflow | FR-ICP-007 | V1.5 | P2 | — | M-4 |
| EP-38 | Platform V2 (SSO, agency workspace, white-label, crawler adapter) | FR-AUTH-008, 009, FR-RPT-002, D-01 | V2 | P2 | — | 04 §19 |

---

## 6. Backlog

### 6.1 Konvensi backlog

- Satu story ≤ 8 SP dan selesai dalam satu sprint. Story "Sebagai …" adalah user story. Story "Teknis:" adalah technical story.
- Kolom **SPR** = sprint target. Kolom **EP** = epic. Kolom **Lane** dipakai untuk perencanaan kapasitas (§2.3).
- `↗nn` pada *Depends on* = story **UI-first** (P-6): UI dibangun terhadap mock kontrak, dan API-nya selesai di SPR-nn. Integrasi + E2E menjadi bagian DoD story API.
- *(Stretch, P1)* = FR P1 MVP 1. Hanya diambil bila story inti sprint tersebut sudah *done*. Bila tidak sempat, pindah ke awal V1 (SPR-18).
- FR yang diberi tanda "(fondasi)" atau "(interim)" berarti story tersebut menyiapkan prasyarat, bukan memenuhi acceptance FR tersebut.

### 6.2 Story per area

#### PLAT — platform, infrastruktur, kualitas & rilis (48 story, 191 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-PLAT-001 | Teknis: provisioning `ops-01`, `ci-01`, `stg-01` dengan Ansible (role base, users, firewall, WireGuard, LUKS) + kunci SOPS/age per environment | NFR-SEC-02, NFR-SEC-04, NFR-PRIV-03 | — | 5 | OPS | SPR-00 | EP-01 | Q-09, Q-CI-08 |
| US-PLAT-002 | Teknis: GitLab CE Omnibus di `ops-01` (registry aktif, 2FA wajib, sign-up nonaktif, backup harian), grup CODEOWNERS, migrasi repo dari GitHub | — | — | 3 | OPS | SPR-00 | EP-01 | US-PLAT-001, Q-08 |
| US-PLAT-003 | Teknis: GitLab Runner `ci-docker`/`ci-build` di `ci-01` + runner `deploy` (protected) di `ops-01`, cache S3 ke SeaweedFS lokal | — | — | 3 | OPS | SPR-00 | EP-01 | US-PLAT-002 |
| US-PLAT-004 | Teknis: skeleton Go `apps/api` — `cmd/msp` (`api`, `worker`, `migrate`), `internal/<modul>` untuk 10 modul + aturan `depguard`, `/healthz`, `/version`, log `slog` JSON | NFR-MAINT-01 | — | 3 | GO | SPR-00 | EP-01 | US-PLAT-002 |
| US-PLAT-005 | Teknis: skeleton web — Next.js App Router + TS strict, pnpm workspaces + Turborepo, `packages/ui`, `packages/api-client`, route `/api` via Traefik | NFR-COMP-01 | — | 3 | FE | SPR-00 | EP-01 | US-PLAT-002 |
| US-PLAT-006 | Teknis: pipeline MR awal — commitlint, golangci-lint (gosec, staticcheck, depguard), `go test -race`, ESLint/Prettier/`tsc`, Vitest, gitleaks, `mr:gate` (QG-01..05, QG-18) | NFR-SEC-04, NFR-MAINT-01 | — | 5 | GO | SPR-00 | EP-02 | US-PLAT-003..005 |
| US-PLAT-007 | Teknis: baseline DB — PostgreSQL 16 + PostGIS + `pg_trgm` + `unaccent` (local, CI, staging), migrasi goose, schema per modul, role `msp_app` `NOBYPASSRLS`, linter RLS & cek migrasi (QG-09 awal, QG-11) | FR-AUD-006 (fondasi), NFR-SEC-03 | — | 5 | GO | SPR-00 | EP-01 | US-PLAT-004 |
| US-PLAT-008 | Teknis: observability baseline — OTel Collector, Prometheus, Grafana, Loki, Tempo, Alertmanager, GlitchTip di `ops-01`; instrumentasi Go/Next.js dengan `correlation_id`; helper redaksi PII | NFR-OBS-01, NFR-PRIV-01 | — | 5 | OPS | SPR-00 | EP-03 | US-PLAT-001 |
| US-PLAT-009 | Teknis: paket `packages/design-tokens` (DTCG → CSS variables, preset Tailwind, TS) dari `docs/02-design-system/tokens` + QG-15 | NFR-A11Y-01 | — | 3 | FE | SPR-00 | EP-06 | US-PLAT-005, D-17 (placeholder OK) |
| US-PLAT-010 | Teknis: dev lokal Docker Compose (PostGIS, Redis, SeaweedFS `weed server -s3`, Kratos/auth stub, Mailpit, Smokescreen, fake-llm, webhook-receiver) + seed minimal | — | — | 3 | GO | SPR-00 | EP-01 | US-PLAT-004 |
| US-PLAT-011 | Teknis: pipeline `main` — image distroless (`CGO_ENABLED=0`), Trivy (QG-27), SBOM + cosign (QG-28), `deploy:staging` (Ansible + Traefik blue/green di `stg-01`), smoke | NFR-SEC-06, NFR-AVAIL-01 | — | 5 | OPS | SPR-01 | EP-02 | US-PLAT-006 |
| US-PLAT-012 | Teknis: kerangka job River — antrean `pipeline`/`scoring`/`export`/`webhook`/`ai`/`maintenance`, retry/backoff, unique & periodic job, job `discarded` sebagai DLQ, outbox via `InsertTx` + `platform.event` | FR-OPS-002 (fondasi), FR-EXP-003 (fondasi) | — | 5 | GO | SPR-01 | EP-01 | US-PLAT-007 |
| US-PLAT-013 | Teknis: contract-first — `contracts/openapi` 3.1, oapi-codegen strict server, error RFC 9457 + katalog `code`, client TS (openapi-typescript/openapi-fetch), Spectral + oasdiff (QG-12..14) | PRD §12.2 | — | 3 | GO | SPR-01 | EP-02 | US-PLAT-004 |
| US-PLAT-014 | Teknis: registry feature flag backend (global + override tenant) untuk trunk-based/dark launch | FR-OPS-007 (fondasi) | — | 2 | GO | SPR-01 | EP-02 | US-PLAT-007 |
| US-PLAT-015 | Teknis: komponen inti `packages/ui` (Radix): button, input, select, table, tabs, dialog, drawer, toast, state baku (loading/empty/error/denied) + Storybook + axe (QG-16) | NFR-A11Y-01 | 03 §4.0 | 8 | FE | SPR-01 | EP-06 | US-PLAT-009 |
| US-PLAT-016 | Teknis: i18n id-ID/en-US + formatter Rupiah, angka, tanggal, zona Asia/Jakarta | NFR-L10N-01 | 03 §7.3 | 3 | FE | SPR-01 | EP-06 | US-PLAT-005 |
| US-PLAT-017 | Teknis: generator `msp-synth` (`apps/api/cmd/msp-synth`) profil tiny/small/medium/perf, kode Kemendagri, injeksi EC berlabel ground truth, canary PII | RG-6 (alat ukur) | — | 5 | QA | SPR-01 | EP-07 | US-PLAT-004 |
| US-PLAT-018 | Teknis: panduan labeling + alat labeling + korpus `gs-norm` | FR-DATA-002 (alat ukur) | — | 3 | QA | SPR-02 | EP-07 | US-PLAT-017, Q-D-03 |
| US-PLAT-019 | Teknis: komponen pola trust P-01..P-13 (claim type, confidence, freshness, verifikasi, band, coverage, factor bars, kontak ter-mask, perkiraan, guardrail, kartu opportunity, Unknown, tombol berbayar) | BR-TRUST-01..06, NFR-A11Y-01 | 03 §6 | 8 | FE | SPR-02 | EP-06 | US-PLAT-015 |
| US-PLAT-020 | Teknis: harness integration test (`services:` PostGIS/Redis/SeaweedFS, fixture EC-xx) + job `traceability` TC↔FR | PRD §12.2 | — | 3 | QA | SPR-02 | EP-07 | US-PLAT-006 |
| US-PLAT-021 | Teknis: labeling `gs-er` batch 1 (1.000 pasangan, 2 pelabel independen, κ ≥ 0,80) | FR-DATA-003, RG-6 | — | 5 | QA | SPR-03 | EP-07 | US-PLAT-018, Q-05, Q-D-03 |
| US-PLAT-022 | Teknis: labeling `gs-cls` (≥ 400) dan `gs-geo` (≥ 300) | FR-INTL-001, FR-DATA-009, RG-6 | — | 5 | QA | SPR-04 | EP-07 | US-PLAT-018, Q-D-03 |
| US-PLAT-023 | Teknis: provisioning produksi data tier — `db-01`/`db-02` (replika streaming, PgBouncer), `stor-01` (SeaweedFS + `weed filer.sync` off-site), pgBackRest repo1 `stor-01` + repo2 off-site | NFR-AVAIL-02, NFR-PRIV-03 | — | 5 | OPS | SPR-04 | EP-04 | Q-09, Q-CI-04 |
| US-PLAT-024 | Teknis: `app-01` produksi (Traefik, `msp_app`) + pipeline tag `v*` (deploy manual blue/green + canary) + rilis uji `v0.1.0` + uji rollback | NFR-AVAIL-01 | — | 5 | OPS | SPR-05 | EP-02 | US-PLAT-023 |
| US-PLAT-025 | Teknis: harness E2E Playwright + axe di CI (stack penuh di runner) dan suite `@smoke` (QG-17) | PRD §12.2, NFR-A11Y-01 | — | 5 | QA | SPR-05 | EP-07 | US-PLAT-011 |
| US-PLAT-026 | Teknis: alert RG-5 bagian MVP 0 (pipeline failure > 5%, backlog > 1 jam), dashboard SLO, job `alert-drill` awal | RG-5, NFR-OBS-01 | — | 3 | OPS | SPR-06 | EP-03 | US-PLAT-008, US-DATA-015 |
| US-PLAT-027 | Teknis: laporan RG-6 MVP 0 pada holdout (ER P/R, akurasi klasifikasi & kelurahan, ≥ 10 EC) + uji determinisme | RG-6, NFR-DATA-01 | — | 3 | QA | SPR-06 | EP-07 | US-PLAT-021, US-PLAT-022 |
| US-PLAT-028 | Teknis: scheduled pipeline `nightly` (regresi, ZAP baseline, data quality, rescan) + Renovate | NFR-SEC-06 | — | 3 | OPS | SPR-07 | EP-02 | US-PLAT-011 |
| US-PLAT-029 | Teknis: app shell customer (sidebar, header, workspace switcher, ICP switcher, saldo, indikator job, banner sistem) dengan data mock dari kontrak | FR-AUTH-002, FR-ICP-006 (shell) | 03 §2.3 | 5 | FE | SPR-02 | EP-06 | US-PLAT-019, US-PLAT-013 |
| US-PLAT-030 | Teknis: egress proxy (Smokescreen) + nftables di `app-01` untuk worker webhook/AI | NFR-SEC-05, FR-EXP-003 | — | 5 | OPS | SPR-10 | EP-05 | US-PLAT-024 |
| US-PLAT-031 | Teknis: runbook on-call + latihan (deploy, rollback, failover DB, insiden + notifikasi PDP 3×24 jam) sebelum closed beta | BR-PRIV-03, NFR-AVAIL-02 | — | 3 | OPS | SPR-11 | EP-04 | US-PLAT-024 |
| US-PLAT-032 | Teknis: usability test formatif (3 peserta calon beta) pada alur onboarding → search → profil | RG-1, RG-2 (persiapan) | FLOW-01, FLOW-02 | 3 | QA | SPR-11 | EP-30 | US-ICP-008, US-PROF-002, Q-07 |
| US-PLAT-033 | Teknis: alert AI failure > 5% dan webhook failure > 5% + kanal on-call (Q-CI-06); drill 4 alert di staging | RG-5 | — | 3 | OPS | SPR-15 | EP-03 | US-PLAT-026, US-EXP-006, US-AI-003 |
| US-PLAT-034 | Teknis: harness k6 + dataset `perf` 5 juta di server perf; PERF-01..07 | NFR-PERF-01..07, NFR-SCAL-01 | — | 5 | QA | SPR-13 | EP-07 | US-SRCH-003, Q-D-04 |
| US-PLAT-035 | Teknis: drill pre-release — restore PITR dari repo off-site ke `dr-01` terisolasi (RTO ≤ 4 jam), tabletop kehilangan DC (RTO ≤ 24 jam), restore objek SeaweedFS, drill rollback blue/green (TC-OPS-011/016) | NFR-AVAIL-02 | — | 5 | OPS | SPR-15 | EP-04 | US-PLAT-023, US-PLAT-024 |
| US-PLAT-036 | Teknis: soak test webhook 24 jam di staging (≥ 99%, TC-EXP-015) | RG-3 | — | 3 | QA | SPR-16 | EP-30 | US-EXP-006 |
| US-PLAT-037 | Teknis: usability test sumatif RG-1/RG-2 (5 peserta, T1–T5, 05 §3.23) | RG-1, RG-2 | FLOW-01..04 | 5 | QA | SPR-16 | EP-30 | build RC, Q-07 |
| US-PLAT-038 | Teknis: security review RG-7 — ZAP full (QG-30), checklist ASVS L2, review threat model 04 §15.11, uji isolasi & SSRF pada build RC | RG-7, NFR-SEC-01, NFR-SEC-06 | — | 5 | GO | SPR-16 | EP-05 | build RC |
| US-PLAT-039 | Teknis: UAT beta (skenario per persona) + sign-off ≥ 3 workspace; sampel merge produksi n ≥ 200 (RG-6); kompilasi bukti RG-1..RG-7 di issue Release Readiness (QG-32), go/no-go | RG-1..RG-7 | — | 3 | QA | SPR-17 | EP-30 | US-PLAT-033..038 |
| US-PLAT-040 | Teknis: PoC SeaweedFS (TQ-17/Q-CI-13) — object lock/WORM, enkripsi volume, presigned PUT/GET + multipart via SDK S3 Go, repo S3 pgBackRest, `weed filer.sync` off-site; hasil → keputusan | NFR-SEC-02, NFR-PRIV-03, NFR-AVAIL-02 | — | 3 | OPS | SPR-01 | EP-01 | US-PLAT-001, stg-01 |
| US-PLAT-041 | Teknis: deploy identity provider (D-20, default Ory Kratos) di `stg-01` via Ansible (produksi ikut `US-PLAT-024`), courier email ke Mailpit/SMTP | FR-AUTH-006, FR-AUTH-001 (fondasi) | — | 2 | OPS | SPR-02 | EP-16 | US-PLAT-001, D-20 |
| US-PLAT-042 | Teknis/Design: prototype interaktif (Storybook + mock kontrak) FLOW-01/FLOW-02 untuk uji formatif awal panel "Kenapa prospek ini?" dan evidence | RG-1, RG-2 (mitigasi dini) | SCR-02, SCR-04, SCR-05, SCR-06 | 5 | FE | SPR-03 | EP-06 | US-PLAT-019 |
| US-PLAT-043 | Teknis: drill restore pertama (manual) dari repo off-site ke `dr-01`, lalu schedule `monthly` (QG-35) | NFR-AVAIL-02 | — | 3 | OPS | SPR-08 | EP-04 | US-PLAT-023, Q-CI-04 |
| US-PLAT-044 | Teknis: eksekusi PERF-01..07 pada build RC (QG-31) + k6 sekali di hardware produksi (data sintetis) sebelum early paid | NFR-PERF-01..07 | — | 3 | QA | SPR-16 | EP-30 | US-PLAT-034 |
| US-PLAT-045 | Teknis: `app-02` — reuse role Ansible `app-01`, Traefik di dua host di belakang VIP (keepalived/VRRP atau LB provider), Redis replica + Sentinel, playbook deploy `serial: 1` | R-08, NFR-AVAIL-01, NFR-AVAIL-02 | — | 5 | OPS | SPR-16 | EP-08 | Q-D-09 (hardware app-02), Q-CI-05, US-PLAT-024 |
| US-PLAT-046 | Teknis: kesiapan multi-host — sesi & invalidasi cache membership lewat Redis bersama (pub/sub), worker River aktif di dua host (leader election periodic job), Kratos di dua host; drill failover `app-01` → `app-02` | R-08, FR-AUTH-005, NFR-AVAIL-01 | — | 3 | GO | SPR-16 | EP-08 | US-PLAT-045 |
| US-PLAT-047 | Teknis: kesiapan beta — workspace beta + grant credit, workspace sintetis `msp-synthetic-monitor` untuk smoke produksi (`msp-synthetic` hanya untuk data uji staging), kanal feedback beta | FR-OPS-006, FR-INTL-010 | — | 2 | QA | SPR-12 | EP-30 | US-OPS-011 |
| US-PLAT-048 | Teknis: retest temuan security + pentest eksternal (opsional) + laporan RG-7 (0 Critical/High) | RG-7, NFR-SEC-06 | — | 3 | QA | SPR-17 | EP-05 | US-PLAT-038 |

#### DATA — fondasi data & pipeline (22 story, 96 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-DATA-001 | Teknis: Input Contract — JSON Schema `ObservationRecord`, Source Registry, format mapping spec (`contracts/input`) | FR-DATA-001 | — | 5 | GO | SPR-01 | EP-09 | Q-01 (sampel), D-02 |
| US-DATA-002 | Sebagai ops admin, saya meng-import dataset snapshot immutable (`ds-YYYY.MM.n`) beserta sumber, tanggal pengambilan, periode referensi, dan tag lisensi | FR-DATA-001 | — | 5 | GO | SPR-01 | EP-09 | US-DATA-001, Q-05 |
| US-DATA-003 | Sebagai ops admin, saya mendapat laporan validasi (jumlah baris, reject per alasan, fill-rate per field); baris reject tidak memblokir baris valid | FR-DATA-001 | — | 3 | GO | SPR-02 | EP-09 | US-DATA-002 |
| US-DATA-004 | Teknis: normalisasi telepon E.164, email, domain (PSL termasuk `.co.id`, `.my.id`), nilai mentah disimpan (BR-NORM-01..03) | FR-DATA-002 | — | 5 | GO | SPR-02 | EP-10 | US-DATA-002, US-PLAT-018 |
| US-DATA-005 | Teknis: normalisasi nama (token badan usaha/gelar → alias) dan alamat → hierarki wilayah Kemendagri (BR-NORM-04/05) | FR-DATA-002 | — | 5 | GO | SPR-02 | EP-10 | US-DATA-013 |
| US-DATA-006 | Teknis: model evidence & `claim_meta` (claim_type, confidence, observed_at, evidence_ids, alternates) + checker evidence coverage | FR-DATA-006, BR-TRUST-01/02 | — | 5 | GO | SPR-02 | EP-10 | US-DATA-002 |
| US-DATA-007 | Teknis: ER tahap 1 — blocking + exact + composite untuk Company/Establishment (BR-ER-01..03, 08, 09) | FR-DATA-003 | — | 8 | GO | SPR-03 | EP-11 | US-DATA-004, US-DATA-005 |
| US-DATA-008 | Teknis: ER tahap 2 — skor fuzzy gabungan, auto-merge ≥ 0,92, kandidat review bernilai tinggi, identifier bertentangan tidak pernah merge (BR-ER-04, 05, 10) | FR-DATA-003, FR-DATA-004 | — | 5 | GO | SPR-04 | EP-11 | US-DATA-007, US-PLAT-021 |
| US-DATA-009 | Teknis: ID stabil (UUIDv7 berprefiks), `entity_source_link`, `entity_redirect`; re-run 2× → jumlah & ID identik | FR-DATA-007 | — | 5 | GO | SPR-04 | EP-11 | US-DATA-007 |
| US-DATA-010 | Teknis: keputusan merge/split/keep persisten lintas re-run (BR-ER-06) | FR-DATA-004 | — | 3 | GO | SPR-05 | EP-11 | US-DATA-008, US-DATA-009 |
| US-DATA-011 | Teknis: verifikasi contact point — email syntax + keberadaan MX (Unbound lokal + cache DNS bersama), telepon format + tipe, domain DNS; gagal sementara → UNKNOWN (BR-VER) | FR-DATA-005 | — | 5 | GO | SPR-02 | EP-10 | US-DATA-004 |
| US-DATA-012 | Teknis: digital presence lite — klasifikasi tipe website & tipe email dari daftar pola `data/reference` | FR-DATA-010 | — | 3 | GO | SPR-03 | EP-10 | US-DATA-004 |
| US-DATA-013 | Sebagai ops admin, saya meng-import referensi wilayah (kode Kemendagri, populasi + periode + sumber, crosswalk BPS, batas wilayah bila ada) | FR-DATA-008 | FLOW-20 | 5 | GO | SPR-01 | EP-12 | Q-04, Q-10, D-13 |
| US-DATA-014 | Teknis: penetapan area — point-in-polygon (`ST_Subdivide` + GiST) atau pencocokan alamat, dengan `geo_confidence` | FR-DATA-009 | — | 5 | GO | SPR-03 | EP-12 | US-DATA-013, US-DATA-005 |
| US-DATA-015 | Teknis: DAG pipeline per snapshot (normalize → resolve → verify → enrich → classify → score → opportunity → project), re-run per stage, gate "Terbitkan" | FR-DATA-001, FR-DATA-007, FR-OPS-002 | — | 5 | GO | SPR-01 | EP-09 | US-PLAT-012 |
| US-DATA-016 | Teknis: throughput re-run penuh 5 juta establishment ≤ 12 jam (TC-DATA-034): batch, `COPY`, paralelisme, skip record tak berubah | FR-DATA-007, NFR-PERF-07 | — | 5 | GO | SPR-06 | EP-09 | US-DATA-015, Q-D-04 |
| US-DATA-017 | Sebagai SDR, saya mengunggah CSV/XLSX (≤ 50.000 baris), memetakan kolom, dan melihat preview validasi | FR-DATA-012 | SCR-16, FLOW-05 | 5 | GO | SPR-14 | EP-27 | US-AUTH-005 |
| US-DATA-018 | Teknis: proses upload async — link ke entity global (BR-ER-07), private prospect untuk NEW, laporan matched/new/rejected, uji isolasi | FR-DATA-012, FR-AUD-006 | SCR-15 | 5 | GO | SPR-14 | EP-27 | US-DATA-017 |
| US-DATA-019 | Sebagai SDR, saya memakai wizard upload dan melihat hasil pencocokan di job center | FR-DATA-012 | SCR-16, SCR-15, FLOW-05 | 5 | FE | SPR-13 | EP-27 | US-DATA-017 ↗14 |
| US-DATA-020 | Teknis: API "Laporkan data" (claim yang dirujuk, jenis masalah) → antrean ops (daftar read-only minimal sampai US-OPS-015) | FR-DATA-014 | — | 2 | GO | SPR-12 | EP-26 | US-PROF-001 |
| US-DATA-021 | Sebagai SDR, saya melaporkan data yang salah dari profil | FR-DATA-014 | SCR-23, FLOW-09 | 2 | FE | SPR-09 | EP-26 | US-DATA-020 ↗12 |
| US-DATA-022 | *(Stretch, P1)* Teknis: identifikasi penyedia dari host MX (mis. Google Workspace, Microsoft 365) & include SPF (tool email marketing) memakai cache DNS yang sama dengan US-DATA-011; evidence; aktifkan OPP-DNS-NO-MX | FR-DATA-011 | — | 3 | GO | SPR-15 | EP-10 | US-DATA-011 |

#### INTL — intelligence engine (11 story, 53 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-INTL-001 | Teknis: taksonomi industri dua level (`data/taxonomy`) + mapping kategori sumber dengan confidence; tidak terpetakan → "Lainnya" | FR-INTL-001 | — | 5 | GO | SPR-02 | EP-13 | Q-01, US-DATA-004 |
| US-INTL-002 | Teknis: skor Data Quality dan Contactability (Lampiran A) | FR-INTL-002, FR-INTL-003 | — | 5 | GO | SPR-03 | EP-13 | US-DATA-011, US-DATA-006 |
| US-INTL-003 | Teknis: commercial potential — band karyawan & band revenue v0 (prior kategori × tier kota, confidence ≤ 70, BR-REV) | FR-INTL-004 | — | 5 | GO | SPR-04 | EP-13 | TQ-08, Q-06 |
| US-INTL-004 | Teknis: katalog opportunity sebagai data + gap fact global per snapshot + guard fill-rate BR-OPP-02 | FR-INTL-007, FR-INTL-005 | — | 8 | GO | SPR-05 | EP-13 | US-DATA-012, US-INTL-001 |
| US-INTL-005 | Teknis: relevansi opportunity per Offering saat dibaca (ADR-15), `opportunity_score`, rationale id/en dengan konteks peer | FR-INTL-007 | — | 5 | GO | SPR-06 | EP-13 | US-INTL-004, US-ICP-002 |
| US-INTL-006 | Teknis: faktor Digital Opportunity & Geographic Opportunity (fallback kelurahan → kecamatan → Unknown) | FR-INTL-005, FR-INTL-006 | — | 3 | GO | SPR-06 | EP-13 | US-INTL-004, US-GEO-002 |
| US-INTL-007 | Teknis: engine priority score — renormalisasi Unknown, coverage, band + cap, UNSCORED, rollup company, `as_of` deterministik | FR-INTL-008 | — | 8 | GO | SPR-06 | EP-14 | US-ICP-001, US-INTL-002..006 |
| US-INTL-008 | Teknis: versioning model skor (`priority-1.2.0`, `versions.lock`, QG-23) + job recompute per versi; versi lama tetap tersimpan | FR-INTL-009 | — | 5 | GO | SPR-06 | EP-14 | US-INTL-007 |
| US-INTL-009 | Teknis: penanda stale (input lebih baru, kelas freshness berpindah) + antrean recompute | FR-INTL-011 | — | 3 | GO | SPR-06 | EP-14 | US-INTL-008 |
| US-INTL-010 | Teknis: API feedback opportunity (Relevan/Tidak + alasan) & skor (terlalu tinggi/rendah) + laporan ops sederhana (agregat acceptance rate per tipe & versi model) | FR-INTL-010 | — | 3 | GO | SPR-12 | EP-26 | US-PROF-001 |
| US-INTL-011 | Sebagai agency owner, saya menandai opportunity relevan/tidak relevan dan memberi feedback skor | FR-INTL-010 | SCR-05 (P-11, P-07) | 3 | FE | SPR-10 | EP-26 | US-INTL-010 ↗12 |

#### ICP — Offering & ICP (10 story, 37 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-ICP-001 | Teknis: engine ICP fit 0–100 (match/partial/no/unknown, wajib vs preferensi, BR-SCORE-12) + breakdown per kriteria | FR-ICP-003 | — | 5 | GO | SPR-05 | EP-14 | US-INTL-001, US-DATA-014 |
| US-ICP-002 | Teknis: workspace internal + Offering & ICP fixture yang dibuat ops (`msp tool`) untuk menguji engine di MVP 0 (PRD §6) | FR-ICP-003, FR-INTL-008 | — | 2 | GO | SPR-05 | EP-14 | US-ICP-001 |
| US-ICP-003 | Teknis: API Offering + katalog offering tag | FR-ICP-001 | — | 3 | GO | SPR-07 | EP-17 | US-AUTH-003 |
| US-ICP-004 | Teknis: API ICP berversi, terikat tepat satu Offering, kriteria wajib/preferensi, pemicu re-score | FR-ICP-002 | — | 5 | GO | SPR-07 | EP-17 | US-ICP-003 |
| US-ICP-005 | Sebagai workspace admin, saya mengelola Offering dan ICP | FR-ICP-001, FR-ICP-002 | SCR-17 | 5 | FE | SPR-06 | EP-17 | US-ICP-004 ↗07 |
| US-ICP-006 | Teknis: skor progresif saat ICP berubah — pre-ranking ≤ 50 kandidat ≤ 30 detik, seluruh kandidat ≤ 10 menit, status progres | FR-ICP-002, FR-INTL-008, NFR-PERF-05 | — | 5 | GO | SPR-09 | EP-17 | US-ICP-004, US-INTL-007 |
| US-ICP-007 | Teknis: preset bobot per tipe Offering, tercatat di `model_version` | FR-ICP-004 | — | 2 | GO | SPR-08 | EP-17 | US-INTL-008 |
| US-ICP-008 | Sebagai agency owner baru, saya menyelesaikan wizard Offering → ICP → hasil pertama ≤ 5 menit (event TTFR) | FR-ICP-005 | SCR-02, FLOW-01 | 5 | FE | SPR-07 | EP-17 | US-ICP-005, US-SRCH-002 ↗08 |
| US-ICP-009 | Teknis: ICP aktif per user per workspace + batas ICP tersimpan per plan | FR-ICP-006 | — | 2 | GO | SPR-08 | EP-17 | US-ICP-004 |
| US-ICP-010 | Sebagai SDR, saya berganti ICP aktif dan melihat status "Menghitung n%" sampai skor siap | FR-ICP-006 | Header switcher, FLOW-11 | 3 | FE | SPR-08 | EP-17 | US-ICP-009, US-PLAT-029 |

#### GEO — geographic & market intelligence (8 story, 36 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-GEO-001 | Teknis: metrik area global L1–L4 × kategori (Lampiran C: densitas, competition_index, coverage, white_space) reproducible per `geo-1.0.0` | FR-GEO-001 | — | 8 | GO | SPR-04 | EP-12 | US-DATA-014, US-INTL-001 |
| US-GEO-002 | Teknis: metrik tenant per ICP (target_business_count, target_density, market_opportunity_score) di job scoring | FR-GEO-001, FR-INTL-006 | — | 5 | GO | SPR-05 | EP-12 | US-GEO-001, US-ICP-001 |
| US-GEO-003 | Teknis: vector tile batas wilayah (PMTiles per `area_set_version`) + endpoint metrik area JSON per ICP | FR-GEO-002 | — | 5 | GO | SPR-11 | EP-20 | US-GEO-002, Q-10 |
| US-GEO-004 | Sebagai sales manager, saya melihat peta choropleth kelurahan/kecamatan dengan metrik terpilih, legend, periode, guardrail, dan alternatif tabel | FR-GEO-002, FR-GEO-009 | SCR-08, FLOW-03 | 8 | FE | SPR-12 | EP-20 | US-GEO-003, Q-D-06 (basemap) |
| US-GEO-005 | Teknis: API territory ranking (territory_score + breakdown) untuk ICP + Offering aktif | FR-GEO-003 | — | 3 | GO | SPR-13 | EP-20 | US-GEO-002 |
| US-GEO-006 | Sebagai sales manager, saya melihat peringkat wilayah, membuka detail area, dan drilldown ke prospek area tersebut | FR-GEO-003, FR-GEO-004 | SCR-09, SCR-10, FLOW-03 | 5 | FE | SPR-13 | EP-20 | US-GEO-005, US-SRCH-006 |
| US-GEO-007 | Teknis: copy guardrail & label "Data terbatas" (kunci i18n 03 §7.7) di tooltip, kartu, dan laporan | FR-GEO-009 | P-10, P-06 | 2 | FE | SPR-12 | EP-20 | US-PLAT-019 |
| US-GEO-008 | *(Stretch, P1)* Sebagai sales manager, saya membuat segment dinamis dari area/opportunity zone | FR-GEO-005 | SCR-10 | 3 | FE | SPR-16 | EP-20 | US-SEG-001 |

#### OPS — konsol internal (19 story, 68 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-OPS-001 | Teknis: shell ops console `/ops` (layout compact, banner environment, navigasi SCR-30..39) | FR-OPS-001 (fondasi) | 03 §2.2 | 3 | FE | SPR-01 | EP-15 | US-PLAT-015 |
| US-OPS-002 | Sebagai ops admin, saya meninjau laporan validasi snapshot lalu menerbitkan atau menolaknya | FR-DATA-001, FR-DATA-008 | SCR-34, FLOW-20 | 5 | FE | SPR-03 | EP-15 | US-DATA-003, US-AUTH-001 |
| US-OPS-003 | Teknis: API metrik Data Health per snapshot (verifikasi, stale, kandidat duplikat, evidence coverage, mapped rate, klasifikasi) | FR-OPS-001 | — | 3 | GO | SPR-03 | EP-15 | US-DATA-006 |
| US-OPS-004 | Sebagai ops admin, saya melihat dashboard Data Health per snapshot dengan delta dan drilldown | FR-OPS-001 | SCR-30 | 5 | FE | SPR-04 | EP-15 | US-OPS-003 |
| US-OPS-005 | Teknis: API merge review (antrean, keputusan merge/same-company/split/skip, undo) | FR-OPS-005, FR-DATA-004 | — | 3 | GO | SPR-05 | EP-15 | US-DATA-008 |
| US-OPS-006 | Sebagai ops admin, saya memutuskan kandidat merge secara berdampingan dengan shortcut keyboard | FR-OPS-005 | SCR-33, FLOW-21 | 5 | FE | SPR-05 | EP-15 | US-OPS-005 |
| US-OPS-007 | Teknis: API Pipeline Health (selesai/gagal/retry/latensi/backlog per stage) + redrive/buang DLQ | FR-OPS-002 | — | 3 | GO | SPR-05 | EP-15 | US-DATA-015 |
| US-OPS-008 | Sebagai ops admin, saya memantau Pipeline Health dan me-redrive DLQ | FR-OPS-002 | SCR-31, FLOW-23 | 5 | FE | SPR-06 | EP-15 | US-OPS-007 |
| US-OPS-009 | Teknis: score inspector ops (alpha) — breakdown skor & opportunity per establishment untuk ICP seed | FR-ICP-003, FR-INTL-008, FR-INTL-007 | SCR-31 (tab) | 3 | FE | SPR-06 | EP-15 | US-INTL-007 |
| US-OPS-010 | Teknis: admin tenant — daftar workspace, plan, grant credit/plan manual (invoice manual) + audit | FR-OPS-006, FR-BILL-006 | — | 5 | GO | SPR-10 | EP-29 | US-BILL-001, US-AUD-001 |
| US-OPS-011 | Sebagai ops admin, saya mengelola workspace beta, plan, dan memberi kredit | FR-OPS-006, FR-BILL-006 | SCR-37, FLOW-24 | 5 | FE | SPR-11 | EP-29 | US-OPS-010 |
| US-OPS-012 | Teknis: akses support read-only (alasan wajib, banner, tercatat audit, tanpa impersonation) | FR-OPS-006 | SCR-37 | 3 | GO | SPR-15 | EP-29 | US-OPS-010 |
| US-OPS-013 | Sebagai ops admin, saya mengubah feature flag dan kill switch (AI, export, webhook, upload) dengan alasan | FR-OPS-007 | SCR-38 | 3 | FE | SPR-11 | EP-29 | US-OPS-014 |
| US-OPS-014 | Teknis: kill switch backend + audit perubahan flag | FR-OPS-007 | — | 2 | GO | SPR-11 | EP-29 | US-PLAT-014, US-AUD-001 |
| US-OPS-015 | Teknis: antrean ops untuk laporan data & permintaan subjek data (status, catatan, tindakan) | FR-OPS-008 | — | 3 | GO | SPR-15 | EP-29 | US-DATA-020, US-AUD-003 |
| US-OPS-016 | Sebagai ops admin, saya menangani suppression, permintaan subjek data, dan laporan data dari satu konsol | FR-OPS-008, FR-AUD-003, FR-AUD-004 | SCR-35, SCR-36, FLOW-22 | 5 | FE | SPR-15 | EP-29 | US-OPS-015, US-AUD-002 |
| US-OPS-017 | Teknis: koreksi data ops (evidence `ops_correction` merujuk laporan, menang sampai observasi lebih baru, memicu recompute, tercatat audit) | FR-OPS-009 | — | 5 | GO | SPR-15 | EP-29 | US-OPS-015, US-INTL-009 |
| US-OPS-018 | Sebagai ops admin, saya menelusuri log audit internal lintas tenant (tanpa edit/hapus) | FR-AUD-001 | SCR-39 | 2 | FE | SPR-13 | EP-29 | US-AUD-001 |
| US-OPS-019 | *(Stretch, P1)* Teknis + UI: Intelligence Health & Cost (distribusi skor per versi, acceptance, kegagalan AI, biaya per brief/record) | FR-OPS-003, FR-OPS-004 | SCR-32 | 5 | GO | SPR-16 | EP-29 | US-INTL-010, US-AI-006 |

#### AUTH — workspace, autentikasi, role (9 story, 35 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-AUTH-001 | Teknis: identitas ops di identity provider (D-20) + TOTP MFA wajib (aal2) + sesi aplikasi untuk ops console; login ops tanpa MFA ditolak | FR-AUTH-006 (ops, MVP 0), FR-AUTH-001 (fondasi) | Login ops (SCR-30..39) | 3 | GO | SPR-02 | EP-16 | US-PLAT-041, D-20 |
| US-AUTH-002 | Sebagai calon user, saya membuat akun (email + password / Google), memverifikasi email, masuk, dan mengatur ulang password | FR-AUTH-001 | SCR-01, FLOW-01 | 5 | FE | SPR-05 | EP-16 | US-PLAT-041, D-20, Q-D-05 |
| US-AUTH-003 | Teknis: sign-up → workspace, membership multi-workspace, header `MSP-Workspace` (ADR-25), gate email terverifikasi | FR-AUTH-001, FR-AUTH-002 | — | 5 | GO | SPR-07 | EP-16 | US-AUTH-001 |
| US-AUTH-004 | Teknis: RBAC dari matriks §8.1.1 (ter-generate), ditegakkan di server, 403 + audit | FR-AUTH-004 | — | 5 | GO | SPR-07 | EP-16 | US-AUTH-003, US-AUD-001 |
| US-AUTH-005 | Teknis: RLS di semua tabel tenant via `tenancy.WithTx` + suite lintas-tenant di CI (QG-09, tidak bisa di-waive) | FR-AUD-006 | — | 5 | GO | SPR-07 | EP-16 | US-AUTH-003 |
| US-AUTH-006 | Teknis: undangan anggota (role, kedaluwarsa 7 hari) + email transaksional | FR-AUTH-003 | — | 3 | GO | SPR-10 | EP-16 | US-AUTH-004, Q-D-05 |
| US-AUTH-007 | Sebagai workspace admin, saya mengundang anggota dan mengubah role; penerima menerima undangan | FR-AUTH-003 | SCR-19, SCR-01, FLOW-10 | 3 | FE | SPR-07 | EP-16 | US-AUTH-006 ↗10 |
| US-AUTH-008 | Teknis: sesi — idle timeout 12 jam, logout semua sesi, akses dicabut ≤ 1 menit saat anggota dikeluarkan | FR-AUTH-005 | — | 3 | GO | SPR-12 | EP-16 | US-AUTH-003 |
| US-AUTH-009 | Sebagai user, saya berpindah workspace (cache klien lama dikosongkan) dan mengelola akun/sesi saya | FR-AUTH-002, FR-AUTH-005 | SCR-18, SCR-24, header | 3 | FE | SPR-07 | EP-16 | US-AUTH-003, US-PLAT-029 |

#### SRCH — search & discovery (11 story, 47 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-SRCH-001 | Teknis: projection `query.prospect_search` (shadow table + swap; establishment aktif, tidak ter-suppress) | FR-SRCH-001, FR-SRCH-004 | — | 5 | GO | SPR-08 | EP-18 | US-INTL-007 |
| US-SRCH-002 | Teknis: keyword search FTS (`simple` + `unaccent`) + `pg_trgm`, toleran PT/CV/tanda baca | FR-SRCH-001 | — | 5 | GO | SPR-08 | EP-18 | US-SRCH-001 |
| US-SRCH-003 | Teknis: filter lengkap + sortir (default priority ↓) + join skor tenant (`score_rank`) | FR-SRCH-002, FR-SRCH-003 | — | 8 | GO | SPR-09 | EP-18 | US-SRCH-002, US-ICP-006 |
| US-SRCH-004 | Teknis: keyset pagination (cursor ber-HMAC terikat epoch) + hitungan eksak ≤ 10.000 / perkiraan "±" | FR-SRCH-006 | — | 3 | GO | SPR-09 | EP-18 | US-SRCH-003 |
| US-SRCH-005 | Sebagai SDR, saya melihat baris hasil berisi skor + band, coverage, opportunity teratas, ikon kontak ter-mask, freshness, dan "+N cabang" | FR-SRCH-004 | SCR-04, FLOW-02 | 8 | FE | SPR-04 | EP-18 | US-SRCH-002 ↗08, US-PLAT-019 |
| US-SRCH-006 | Sebagai SDR, saya memfilter, menyortir, dan berbagi URL hasil; saat hasil nol saya mendapat saran pelonggaran filter | FR-SRCH-002, FR-SRCH-003, FR-SRCH-008 | SCR-04 | 5 | FE | SPR-07 | EP-18 | US-SRCH-003 ↗09 |
| US-SRCH-007 | Teknis: API zero-result guidance (filter mana yang paling membatasi) | FR-SRCH-008 | — | 2 | GO | SPR-09 | EP-18 | US-SRCH-003 |
| US-SRCH-008 | Teknis: API saved search (query + filter + ICP) | FR-SRCH-005 | — | 2 | GO | SPR-10 | EP-18 | US-SRCH-003 |
| US-SRCH-009 | Sebagai SDR, saya menyimpan pencarian dan menjalankannya ulang | FR-SRCH-005 | SCR-11 (tab tersimpan) | 3 | FE | SPR-11 | EP-18 | US-SRCH-008 |
| US-SRCH-010 | Teknis: objek seleksi (halaman ini / semua hasil hingga batas plan) untuk aksi massal | FR-SRCH-007 | — | 3 | GO | SPR-10 | EP-18 | US-SRCH-004 |
| US-SRCH-011 | Sebagai SDR, saya memilih banyak prospek (halaman/semua) dan melihat batas sebelum aksi massal | FR-SRCH-007 | SCR-04 | 3 | FE | SPR-08 | EP-18 | US-SRCH-010 ↗10 |

#### PROF — prospect profile (9 story, 34 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-PROF-001 | Teknis: API profil (claim, masking BR-CRED-02, suppression, p95 ≤ 500 ms) | FR-PROF-001, FR-PROF-003 | — | 5 | GO | SPR-08 | EP-19 | US-SRCH-001 |
| US-PROF-002 | Sebagai SDR, saya melihat header profil dan panel "Kenapa prospek ini?" (nilai, bobot, kontribusi, claim type, versi model) | FR-PROF-001, FR-PROF-002 | SCR-05, P-05, P-07 | 5 | FE | SPR-05 | EP-19 | US-PROF-001 ↗08, US-PROF-006 ↗08 |
| US-PROF-003 | Sebagai SDR, saya membuka evidence drawer untuk setiap claim (dataset + snapshot, upload, rule, model/prompt) | FR-PROF-004 | SCR-06 | 5 | FE | SPR-06 | EP-19 | US-PROF-006 ↗08 |
| US-PROF-004 | Sebagai SDR, saya melihat kontak ter-mask beserta status/confidence, membuka kontak, lalu copy/klik-WA/klik-telepon | FR-PROF-003 | SCR-05, P-08 | 5 | FE | SPR-08 | EP-19 | US-BILL-003 ↗10 |
| US-PROF-005 | Sebagai agency owner, saya melihat daftar cabang serta kartu konteks lokasi, komersial (perkiraan), dan digital presence | FR-PROF-005, FR-PROF-006, FR-PROF-007, FR-PROF-008 | SCR-05 | 5 | FE | SPR-08 | EP-19 | US-PROF-001 |
| US-PROF-006 | Teknis: API penjelasan skor (rekonstruksi dari faktor + config) dan lookup evidence per claim | FR-PROF-002, FR-PROF-004 | — | 3 | GO | SPR-08 | EP-19 | US-PROF-001, US-INTL-008 |
| US-PROF-007 | Sebagai Viewer/Member, saya hanya melihat aksi yang diizinkan role di bar aksi profil | FR-PROF-009 | SCR-05 | 3 | FE | SPR-09 | EP-19 | US-AUTH-004 |
| US-PROF-008 | Teknis: profil, list, dan Tugas saya dapat dipakai di 360 px (NFR-COMP-01) | FR-PROF-001, FR-SEG-005 | SCR-05, SCR-12, SCR-13 | 3 | FE | SPR-14 | EP-19 | US-PROF-005 |
| US-PROF-009 | *(Stretch, P1)* Sebagai SDR, saya melihat timeline aktivitas prospek (privat workspace) | FR-PROF-010 | SCR-05 | 3 | FE | SPR-15 | EP-19 | US-SEG-005 |

#### SEG — list, segment, tag, catatan, assignment (7 story, 22 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-SEG-001 | Teknis: API list statis (owner, berbagi) dan segment dinamis (jumlah, distribusi skor, pembaruan otomatis) | FR-SEG-001, FR-SEG-002 | — | 5 | GO | SPR-11 | EP-21 | US-SRCH-010 |
| US-SEG-002 | Sebagai SDR, saya membuat list, menambah/menghapus prospek, dan menyimpan segment dinamis | FR-SEG-001, FR-SEG-002 | SCR-11, SCR-12 | 5 | FE | SPR-10 | EP-21 | US-SEG-001 ↗11 |
| US-SEG-003 | Teknis: API tag workspace (termasuk bulk) dan catatan prospek | FR-SEG-003, FR-SEG-004 | — | 3 | GO | SPR-10 | EP-21 | US-AUTH-005 |
| US-SEG-004 | Sebagai SDR, saya memberi tag (massal) dan catatan pada prospek | FR-SEG-003, FR-SEG-004 | SCR-04, SCR-05 | 3 | FE | SPR-10 | EP-21 | US-SEG-003 |
| US-SEG-005 | Teknis: assignment prospek/list ke anggota + pencatatan `activity` | FR-SEG-005 | — | 3 | GO | SPR-11 | EP-21 | US-SEG-001 |
| US-SEG-006 | Sebagai sales manager, saya menugaskan prospek/list; rep melihatnya di Tugas saya | FR-SEG-005 | SCR-13 | 3 | FE | SPR-10 | EP-21 | US-SEG-005 ↗11 |
| US-SEG-007 | *(Stretch, P1)* Sebagai SDR, saya memberi status kerja ringan (Baru … Didiskualifikasi) | FR-SEG-006 | SCR-05, SCR-13 | 2 | FE | SPR-15 | EP-21 | US-SEG-005 |

#### BILL — usage, credit, plan (10 story, 39 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-BILL-001 | Teknis: credit ledger double-entry append-only (grant/debit/refund/expire; available/held/consumed), saldo = Σ ledger, urutan kedaluwarsa | FR-BILL-002 | — | 8 | GO | SPR-08 | EP-22 | US-AUTH-003, D-08 |
| US-BILL-002 | Teknis: usage event ber-idempotency key + hold → capture/release + anti-overspend konkuren | FR-BILL-001 | — | 5 | GO | SPR-10 | EP-22 | US-BILL-001 |
| US-BILL-003 | Teknis: unlock company (1 credit, permanen per workspace, unlock konkuren aman) | FR-BILL-001, FR-PROF-003 | — | 3 | GO | SPR-10 | EP-22 | US-BILL-002 |
| US-BILL-004 | Teknis: entitlement & fair-use server-side (seat, ICP, profil, export, AI credit, webhook) dengan error ber-kode | FR-BILL-003 | — | 5 | GO | SPR-10 | EP-22 | US-BILL-001, D-09 |
| US-BILL-005 | Sebagai SDR, saya melihat pratinjau biaya sebelum aksi berbayar dan diblokir dengan pesan jelas bila kredit tidak cukup | FR-BILL-004, FR-EXP-002 | SCR-14, FLOW-06 | 5 | FE | SPR-09 | EP-22 | US-BILL-006 ↗11 |
| US-BILL-006 | Teknis: API pratinjau biaya untuk aksi massal (unlock, export, webhook, refresh) | FR-BILL-004 | — | 2 | GO | SPR-11 | EP-22 | US-BILL-002, US-SRCH-010 |
| US-BILL-007 | Teknis: refund otomatis (operasi gagal/tidak terkirim; kontak INVALID ≤ 30 hari via refresh atau laporan terkonfirmasi) | FR-BILL-005 | — | 5 | GO | SPR-13 | EP-22 | US-BILL-002, US-MON-001 |
| US-BILL-008 | Sebagai workspace owner, saya melihat saldo di header dan halaman Pemakaian & kredit (riwayat ledger, masa berlaku) | FR-BILL-002 | SCR-21, header | 3 | FE | SPR-10 | EP-22 | US-BILL-001 |
| US-BILL-009 | Teknis: rekonsiliasi nightly ledger ↔ usage event + alert selisih | FR-BILL-001, FR-BILL-002, RG-4 | — | 3 | GO | SPR-12 | EP-22 | US-BILL-002 |
| US-BILL-010 | *(Stretch, P1)* Sebagai workspace owner, saya melihat dashboard usage per operasi dan per anggota | FR-BILL-007 | SCR-21 | 3 | FE | SPR-16 | EP-22 | US-BILL-008 |

#### EXP — export & webhook (9 story, 38 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-EXP-001 | Teknis: export job async CSV/XLSX/JSON, pilih field, kolom claim_type/confidence/verified_at, link presigned SeaweedFS 24 jam, retry | FR-EXP-001 | — | 5 | GO | SPR-11 | EP-23 | US-SRCH-010, US-PLAT-012 |
| US-EXP-002 | Teknis: billing export — reserve → capture/release, auto-unlock hanya saat sukses | FR-EXP-002 | — | 3 | GO | SPR-11 | EP-23 | US-EXP-001, US-BILL-002 |
| US-EXP-003 | Teknis: batas export per plan & per job (≤ 50.000) dan pengecualian record ter-suppress | FR-EXP-005, FR-EXP-006 | — | 2 | GO | SPR-11 | EP-23 | US-EXP-001, US-BILL-004 |
| US-EXP-004 | Sebagai SDR, saya meng-export seleksi/list/segment, memantau progres di job center, dan mengunduh hasil | FR-EXP-001 | SCR-15, FLOW-04 | 5 | FE | SPR-11 | EP-23 | US-EXP-001 |
| US-EXP-005 | Teknis: registrasi webhook endpoint (HTTPS, validasi SSRF, secret AES-GCM, langganan event, tombol uji) | FR-EXP-003 | — | 5 | GO | SPR-13 | EP-24 | US-PLAT-030 |
| US-EXP-006 | Teknis: delivery webhook — HMAC-SHA256 + timestamp, retry backoff ≤ 24 jam, DLQ + kirim ulang manual, log; lewat egress proxy | FR-EXP-003 | — | 8 | GO | SPR-13 | EP-24 | US-EXP-005, US-PLAT-012 |
| US-EXP-007 | Sebagai workspace admin, saya mengatur, menguji, dan memantau log pengiriman webhook | FR-EXP-003 | SCR-20, FLOW-07 | 5 | FE | SPR-13 | EP-24 | US-EXP-005 |
| US-EXP-008 | Teknis: aksi "Kirim ke webhook" untuk seleksi + billing + pengecualian suppressed | FR-EXP-004, FR-EXP-005 | — | 3 | GO | SPR-14 | EP-24 | US-EXP-006 |
| US-EXP-009 | Sebagai SDR, saya mengirim seleksi ke webhook dari bar aksi massal dan profil | FR-EXP-004 | SCR-04, SCR-05 | 2 | FE | SPR-14 | EP-24 | US-EXP-008 |

#### AI — brief & sales angle (9 story, 27 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-AI-001 | Teknis: port `LLMProvider` + alias model + fake provider CI + key sandbox | FR-AI-001 | — | 3 | GO | SPR-13 | EP-25 | D-15, TQ-02 |
| US-AI-002 | Teknis: prompt registry `brief-1.0.0` + input builder + redaksi kontak person (person = role) + test redaksi | FR-AI-001, FR-AI-005 | — | 5 | GO | SPR-13 | EP-25 | US-AI-001, US-PROF-001 |
| US-AI-003 | Teknis: grounding validator (sitasi claim/evidence ID), regenerate maks 1×, error eksplisit + refund | FR-AI-004 | — | 5 | GO | SPR-14 | EP-25 | US-AI-002 |
| US-AI-004 | Teknis: cache brief per workspace (kunci ws, ICP/Offering, hash input, versi prompt, bahasa) 7 hari; generate ulang berbayar | FR-AI-003 | — | 3 | GO | SPR-14 | EP-25 | US-AI-002, US-BILL-002 |
| US-AI-005 | Sebagai agency owner, saya membuat brief AI berlabel "Dibuat AI" dalam bahasa Indonesia/Inggris dengan sitasi yang dapat dibuka | FR-AI-001 | SCR-07, FLOW-04 | 5 | FE | SPR-14 | EP-25 | US-AI-003 |
| US-AI-006 | Teknis: batas biaya AI per workspace + kill switch global (tombol AI nonaktif dengan pesan) | FR-AI-006 | — | 3 | GO | SPR-14 | EP-25 | US-AI-003, US-OPS-014 |
| US-AI-007 | Teknis: eval set `gs-ai` (≥ 60 profil) + QG-25 di CI | FR-AI-001, FR-AI-004, FR-AI-005 | — | 3 | QA | SPR-14 | EP-25 | US-AI-002 |
| US-AI-008 | *(Stretch, P1)* Teknis: sales angle (Inferred) terkait Offering + opportunity | FR-AI-002 | SCR-07 | 3 | GO | SPR-15 | EP-25 | US-AI-003 |
| US-AI-009 | *(Stretch, P1)* Sebagai agency owner, saya memberi 👍/👎 + alasan pada brief | FR-AI-007 | SCR-07 | 2 | FE | SPR-16 | EP-25 | US-AI-005 |

#### MON — refresh (2 story, 7 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-MON-001 | Teknis: refresh manual — re-verify + re-score pada snapshot terbaru, 1 credit, tanpa entity duplikat | FR-MON-001 | — | 5 | GO | SPR-12 | EP-26 | US-DATA-011, US-INTL-009, US-BILL-002 |
| US-MON-002 | Sebagai SDR, saya me-refresh prospek dengan konfirmasi biaya dan melihat statusnya | FR-MON-001 | SCR-05, SCR-14, FLOW-08 | 2 | FE | SPR-12 | EP-26 | US-MON-001 |

#### AUD — audit, privasi, governance (6 story, 26 SP inti)

| ID | Story | FR / NFR | SCR / FLOW | SP | Lane | SPR | EP | Depends on |
|---|---|---|---|--:|---|---|---|---|
| US-AUD-001 | Teknis: audit log immutable (INSERT-only, tanpa endpoint update/delete) + emitter untuk login, role, export, unlock, webhook, suppression, merge, grant, support, flag | FR-AUD-001 | — | 5 | GO | SPR-07 | EP-28 | US-PLAT-007 |
| US-AUD-002 | Teknis: suppression list global + propagasi ≤ 24 jam ke projection/search/profil/export/webhook, regenerasi file export aktif, event `suppression.applied` | FR-AUD-003 | — | 8 | GO | SPR-12 | EP-28 | US-SRCH-001, US-EXP-001 |
| US-AUD-003 | Teknis: intake permintaan subjek data (form publik + email), workflow SLA, penerusan ke tenant untuk data upload | FR-AUD-004 | — | 5 | GO | SPR-15 | EP-28 | US-AUD-001, Q-D-08 (legal) |
| US-AUD-004 | Teknis: penutupan workspace — tenggang 30 hari, purge data tenant, pseudonimisasi audit & ledger | FR-AUD-005 | — | 5 | GO | SPR-15 | EP-28 | US-AUD-001, Q-D-08 (legal) |
| US-AUD-005 | Sebagai subjek data, saya mengajukan permintaan lewat formulir publik; sebagai Owner, saya dapat menutup workspace | FR-AUD-004, FR-AUD-005 | SCR-25, SCR-18 | 3 | FE | SPR-15 | EP-28 | US-AUD-003 |
| US-AUD-006 | *(Stretch, P1)* Sebagai workspace admin, saya melihat log audit workspace | FR-AUD-002 | SCR-22 | 3 | FE | SPR-15 | EP-28 | US-AUD-001 |

### 6.3 Cek cakupan FR

**Hasil:** semua **92 FR P0** MVP 0/MVP 1 muncul di ≥ 1 story inti: 25 khusus MVP 0 (tanda ⁰), 66 khusus MVP 1, dan FR-AUTH-006 (ops di MVP 0, tenant di MVP 1). Semua 10 FR P1 MVP 1 tertutup sebagai stretch. Format: nomor FR → story (sprint). Prefix `FR-<AREA>-` dan `US-` dihilangkan.

| Area | FR P0 → story (sprint) |
|---|---|
| AUTH | 001 → AUTH-001 (02), PLAT-041 (02), AUTH-002 (05), AUTH-003 (07) · 002 → PLAT-029 (02), AUTH-003 (07), AUTH-009 (07) · 003 → AUTH-007 (07), AUTH-006 (10) · 004 → AUTH-004 (07) · 005 → AUTH-009 (07), AUTH-008 (12), PLAT-046 (16) · 006⁰ (ops) / tenant MVP 1 → AUTH-001 (02), PLAT-041 (02) |
| DATA | 001⁰ → DATA-001 (01), DATA-002 (01), DATA-015 (01), DATA-003 (02), OPS-002 (03) · 002⁰ → DATA-004 (02), DATA-005 (02), PLAT-018 (02) · 003⁰ → DATA-007 (03), PLAT-021 (03), DATA-008 (04) · 004⁰ → DATA-008 (04), DATA-010 (05), OPS-005 (05) · 005⁰ → DATA-011 (02) · 006⁰ → DATA-006 (02) · 007⁰ → DATA-015 (01), DATA-009 (04), DATA-016 (06) · 008⁰ → DATA-013 (01), OPS-002 (03) · 009⁰ → DATA-014 (03), PLAT-022 (04) · 010⁰ → DATA-012 (03) · 012 → DATA-019 (13), DATA-017 (14), DATA-018 (14) · 014 → DATA-021 (09), DATA-020 (12) |
| ICP | 001 → ICP-005 (06), ICP-003 (07) · 002 → ICP-005 (06), ICP-004 (07), ICP-006 (09) · 003⁰ → ICP-001 (05), ICP-002 (05), OPS-009 (06) · 004 → ICP-007 (08) · 005 → ICP-008 (07) · 006 → PLAT-029 (02), ICP-009 (08), ICP-010 (08) |
| SRCH | 001 → SRCH-001 (08), SRCH-002 (08) · 002 → SRCH-006 (07), SRCH-003 (09) · 003 → SRCH-006 (07), SRCH-003 (09) · 004 → SRCH-005 (04), SRCH-001 (08) · 005 → SRCH-008 (10), SRCH-009 (11) · 006 → SRCH-004 (09) · 007 → SRCH-011 (08), SRCH-010 (10) · 008 → SRCH-006 (07), SRCH-007 (09) |
| PROF | 001 → PROF-002 (05), PROF-001 (08), PROF-008 (14) · 002 → PROF-002 (05), PROF-006 (08) · 003 → PROF-001 (08), PROF-004 (08), BILL-003 (10) · 004 → PROF-003 (06), PROF-006 (08) · 005 → PROF-005 (08) · 006 → PROF-005 (08) · 007 → PROF-005 (08) · 008 → PROF-005 (08) · 009 → PROF-007 (09) |
| INTL | 001⁰ → INTL-001 (02), PLAT-022 (04) · 002⁰ → INTL-002 (03) · 003⁰ → INTL-002 (03) · 004⁰ → INTL-003 (04) · 005⁰ → INTL-004 (05), INTL-006 (06) · 006⁰ → GEO-002 (05), INTL-006 (06) · 007⁰ → INTL-004 (05), INTL-005 (06), OPS-009 (06) · 008⁰ → ICP-002 (05), INTL-007 (06), OPS-009 (06), ICP-006 (09) · 009⁰ → INTL-008 (06) · 010 → INTL-011 (10), INTL-010 (12), PLAT-047 (12) · 011⁰ → INTL-009 (06) |
| GEO | 001⁰ → GEO-001 (04), GEO-002 (05) · 002 → GEO-003 (11), GEO-004 (12) · 003 → GEO-005 (13), GEO-006 (13) · 004 → GEO-006 (13) · 009 → GEO-004 (12), GEO-007 (12) |
| SEG | 001 → SEG-002 (10), SEG-001 (11) · 002 → SEG-002 (10), SEG-001 (11) · 003 → SEG-003 (10), SEG-004 (10) · 004 → SEG-003 (10), SEG-004 (10) · 005 → SEG-006 (10), SEG-005 (11), PROF-008 (14) |
| EXP | 001 → EXP-001 (11), EXP-004 (11) · 002 → BILL-005 (09), EXP-002 (11) · 003 → PLAT-012 (01), PLAT-030 (10), EXP-005 (13), EXP-006 (13), EXP-007 (13) · 004 → EXP-008 (14), EXP-009 (14) · 005 → EXP-003 (11), EXP-008 (14) · 006 → EXP-003 (11) |
| AI | 001 → AI-001 (13), AI-002 (13), AI-005 (14), AI-007 (14) · 003 → AI-004 (14) · 004 → AI-003 (14), AI-007 (14) · 005 → AI-002 (13), AI-007 (14) · 006 → AI-006 (14) |
| MON | 001 → MON-001 (12), MON-002 (12) |
| BILL | 001 → BILL-002 (10), BILL-003 (10), BILL-009 (12) · 002 → BILL-001 (08), BILL-008 (10), BILL-009 (12) · 003 → BILL-004 (10) · 004 → BILL-005 (09), BILL-006 (11) · 005 → BILL-007 (13) · 006 → OPS-010 (10), OPS-011 (11) |
| AUD | 001 → AUD-001 (07), OPS-018 (13) · 003 → AUD-002 (12), OPS-016 (15) · 004 → AUD-003 (15), AUD-005 (15), OPS-016 (15) · 005 → AUD-004 (15), AUD-005 (15) · 006 → PLAT-007 (00), AUTH-005 (07), DATA-018 (14) |
| OPS | 001⁰ → OPS-001 (01), OPS-003 (03), OPS-004 (04) · 002⁰ → DATA-015 (01), PLAT-012 (01), OPS-007 (05), OPS-008 (06) · 005⁰ → OPS-005 (05), OPS-006 (05) · 006 → OPS-010 (10), OPS-011 (11), PLAT-047 (12), OPS-012 (15) · 007 → PLAT-014 (01), OPS-013 (11), OPS-014 (11) · 008 → OPS-015 (15), OPS-016 (15) · 009 → OPS-017 (15) |

| FR P1 MVP 1 | Story (sprint) | Status |
|---|---|---|
| FR-DATA-011 | US-DATA-022 (15) | Stretch |
| FR-PROF-010 | US-PROF-009 (15) | Stretch |
| FR-GEO-005 | US-GEO-008 (16) | Stretch |
| FR-SEG-006 | US-SEG-007 (15) | Stretch |
| FR-AI-002 | US-AI-008 (15) | Stretch |
| FR-AI-007 | US-AI-009 (16) | Stretch |
| FR-BILL-007 | US-BILL-010 (16) | Stretch |
| FR-AUD-002 | US-AUD-006 (15) | Stretch |
| FR-OPS-003 | US-OPS-019 (16) | Stretch |
| FR-OPS-004 | US-OPS-019 (16) | Stretch |

---

## 7. Rencana sprint

Kolom *Story kunci* menulis ID tanpa prefix `US-`. Daftar lengkap per sprint ada di kolom SPR §6.2.

### 7.1 Sprint 0 & MVP 0 (SPR-00..SPR-06)

| Sprint | Goal | Story kunci | Demo di akhir sprint | Risiko |
|---|---|---|---|---|
| **SPR-00** Bootstrap | Infrastruktur & repo siap dipakai seluruh tim | PLAT-001..010 | MR percobaan: pipeline hijau lalu diblokir `mr:gate` sampai di-approve. `make dev` menjalankan stack lokal. Grafana menampilkan metrik host. Paket design tokens ter-build | Server belum tersedia (Q-09). Migrasi repo (Q-08). DevOps 0,5 FTE tidak cukup (Q-D-18) |
| **SPR-01** Input Contract | Snapshot pertama masuk lewat kontrak resmi, staging bisa di-deploy | DATA-001/002/013/015, PLAT-011..017, PLAT-040, OPS-001 | Sampel dataset (Q-01) di-import sebagai `ds-2026.11.1` immutable di staging lewat pipeline `main`. Referensi wilayah ter-import. Komponen UI di Storybook | Sampel dataset & lisensi terlambat. PoC SeaweedFS gagal → Garage |
| **SPR-02** Normalisasi & evidence | Nilai mentah → kanonik dengan evidence dan status verifikasi | DATA-003/004/005/006/011, INTL-001, AUTH-001, PLAT-018/019/020/029/041 | Laporan validasi + fill-rate per field. Nilai mentah vs ternormalisasi beserta evidence. Status verifikasi email/telepon. Login ops console dengan TOTP; tanpa MFA ditolak (FR-AUTH-006) | Kualitas alamat rendah sehingga mapping wilayah lemah. D-20 belum diputuskan |
| **SPR-03** ER tahap 1 | Deduplikasi dasar dan penetapan kelurahan | DATA-007/012/014, INTL-002, OPS-002/003, PLAT-021/042 | Ops menerbitkan snapshot dari SCR-34. Jumlah duplikat terdeteksi. % establishment terpetakan ke kelurahan. Prototype klik diuji internal | Kapasitas Natal. Labeling `gs-er` (Q-D-03) |
| **SPR-04** ER tahap 2 & metrik area | ID stabil, auto-merge, metrik geo global | DATA-008/009, GEO-001, INTL-003, OPS-004, PLAT-022/023, SRCH-005 (UI-first) | Re-run 2× → jumlah & ID identik. Dashboard Data Health. Metrik area per kecamatan dengan label coverage | Kapasitas Tahun Baru. Server produksi (Q-09). Prior revenue (Q-06) |
| **SPR-05** Opportunity & ICP fit | Gap fact + ICP fit untuk ICP seed, merge review di UI | INTL-004, ICP-001/002, GEO-002, DATA-010, OPS-005/006/007, PLAT-024/025, AUTH-002 (UI), PROF-002 (UI-first) | Opportunity beserta rationale & evidence. ICP fit untuk ICP seed. Keputusan merge bertahan setelah re-run. Rilis uji `v0.1.0` ke produksi | Guard fill-rate membuat banyak tipe opportunity inactive (Q-01) |
| **SPR-06** Priority score → **M-1** | Priority score berversi + exit MVP 0 | INTL-005..009, DATA-016, OPS-008/009, PLAT-026/027, PROF-003, ICP-005 (UI-first) | Score inspector: breakdown + coverage + band. Recompute 2× → hash identik. Laporan RG-6. Waktu re-run 5 juta. Drill alert pipeline. **Go/no-go M-1** | TC-DATA-034 meleset (waiver TL + rencana). RG-6 di bawah ambang → M-1 mundur 1 sprint |

### 7.2 MVP 1 build (SPR-07..SPR-15)

| Sprint | Goal | Story kunci | Demo di akhir sprint | Risiko |
|---|---|---|---|---|
| **SPR-07** Tenant foundation | User, workspace, RBAC, RLS, audit | AUTH-003/004/005, AUD-001, ICP-003/004, PLAT-028; FE: AUTH-007/009, ICP-008, SRCH-006 | Sign-up → verifikasi email → workspace → pindah workspace. Viewer mendapat 403 yang tercatat di audit. Suite lintas-tenant hijau di CI. API Offering & ICP | Provider email (Q-D-05) |
| **SPR-08** ICP & search dasar | Offering/ICP nyata, projection, keyword search, API profil | ICP-007/009, SRCH-001/002, PROF-001/006, BILL-001, PLAT-043; FE: ICP-010, PROF-004/005, SRCH-011 | Wizard Offering → ICP → hasil pertama (keyword) dengan skor nyata. Profil dengan panel "Kenapa?" + evidence drawer. Ledger credit dasar (grant, saldo). Drill restore pertama di `dr-01` | Waktu rebuild projection |
| **SPR-09** Filter & skor progresif | Filter lengkap, sort, pagination, re-skor progresif | ICP-006, SRCH-003/004/007; FE: BILL-005, PROF-007, DATA-021 | Ganti ICP → halaman pertama ber-skor ≤ 30 detik. Hitungan filter konsisten. Saran zero-result | Kapasitas 60% (Idul Fitri). NFR-PERF-05 |
| **SPR-10** Credit & unlock | Ledger, usage, unlock, entitlement, organize UI | BILL-002..004, OPS-010, SEG-003, AUTH-006, SRCH-008/010, PLAT-030; FE: SEG-002/004/006, BILL-008, INTL-011 | Buka kontak (1 credit), saldo = Σ ledger, unlock konkuren aman, error entitlement ber-kode. Undangan anggota. Egress proxy aktif | Konkurensi unlock & overspend (TC-BILL-003/004) |
| **SPR-11** Export & organize | Export dengan pratinjau biaya, list/segment/assign, admin tenant | EXP-001..004, SEG-001/005, BILL-006, OPS-011/013/014, GEO-003, PLAT-031/032, SRCH-009 | Select → pratinjau biaya → export XLSX → unduh (separuh RG-3). Ops memberi credit ke workspace beta. Kill switch export. Uji formatif dengan 3 calon beta | NFR-PERF-04. Rekrutmen beta (Q-07) |
| **SPR-12** Beta-ready → **M-2** | Suppression, refresh, feedback, peta | AUD-002, AUTH-008, MON-001/002, INTL-010, DATA-020, BILL-009, GEO-004/007, PLAT-047 | Suppression terpropagasi ke search/profil/export. Refresh 1 credit. Feedback opportunity + laporan acceptance di ops. Peta choropleth. Workspace beta disiapkan. **Go/no-go M-2** | Kompleksitas suppression. Perjanjian beta & privasi (Q-D-08) |
| **SPR-13** Webhook & AI dasar | Webhook end-to-end, fondasi AI, territory | EXP-005/006/007, AI-001/002, GEO-005/006, BILL-007, DATA-019 (UI), OPS-018, PLAT-034 | Registrasi webhook menolak `169.254.169.254`. Delivery bertanda tangan HMAC → retry → DLQ → kirim ulang. Territory ranking + drilldown. Dry-run prompt dengan fake provider | Konfigurasi egress. D-15 belum final |
| **SPR-14** AI brief & upload | Brief ber-sitasi, upload customer, webhook massal | AI-003..007, DATA-017/018, EXP-008/009, PROF-008 | Brief id/en dengan sitasi, cache hit, kill switch. Upload CSV → matched/new/rejected. Kirim seleksi ke webhook. Wave 2 aktif per workspace | Grounding < 98%. Biaya AI (R-05). Kapasitas libur |
| **SPR-15** Privasi & feature freeze | Seluruh P0 selesai, hardening 1 | AUD-003/004/005, OPS-012/015/016/017, PLAT-033, PLAT-035, stretch | DSR masuk → SLA → diteruskan ke tenant. Koreksi ops memicu recompute. Drill 4 alert. Drill restore `dr-01` + rollback. **Feature freeze** | Input konsultan hukum terlambat |

### 7.3 Hardening & rilis (SPR-16..SPR-17)

| Sprint | Goal | Story kunci | Demo / bukti | Risiko |
|---|---|---|---|---|
| **SPR-16** Bukti RG | Kumpulkan bukti RG-1/2/3/7 dan HA | PLAT-036 (soak), PLAT-037 (usability RG-1/RG-2), PLAT-038 (security), PLAT-044 (PERF), PLAT-045/046 (`app-02`), stretch bila RG hijau | Laporan soak ≥ 99%. Hasil usability T1–T5. ZAP full. PERF-01..07. Drill failover `app-01` → `app-02` | Usability gagal → perbaikan di SPR-17. Hardware `app-02` belum siap |
| **SPR-17** Rilis → **M-3** | Release Readiness & go/no-go | PLAT-039 (UAT + RR + sampel merge n ≥ 200), PLAT-048 (retest security), perbaikan bug | Issue Release Readiness ditutup (QG-32). Tag `v1.0.0` ke produksi. Invoice manual pertama | S1/S2 terlambat ditemukan → M-3 mundur. Cadangan: SPR-18 sebagai sprint buffer sebelum V1 |

### 7.4 V1 (SPR-18..SPR-24, ringkas)

| Sprint | Fokus | Epic |
|---|---|---|
| SPR-18..19 | Konversi beta → berbayar, sisa stretch P1, billing self-serve, API key | EP-36, EP-35 (API key) |
| SPR-19..21 | Signal dari diff snapshot, watch & alert, refresh terjadwal, pusat notifikasi | EP-31, EP-32 |
| SPR-21..23 | Compare & laporan, NL search & peta hasil, satu integrasi CRM | EP-33, EP-34, EP-35 |
| SPR-24 | Hardening V1 → **M-4** | — |

Backlog V1 dipecah menjadi story pada refinement SPR-15..SPR-17, setelah survei CRM (D-10) dan data beta tersedia.

---

## 8. Program beta

### 8.1 Kriteria seleksi (Q-07, D-16)

| Kriteria | Target | Alasan |
|---|---|---|
| Komposisi persona | ≥ 3 agency digital (PER-01), ≥ 2 vendor SaaS SMB (PER-02/PER-03). Distributor opsional | Beachhead D-16. Usability RG-1 butuh ≥ 2 PER-01, ≥ 2 PER-02, ≥ 1 PER-03 (05 §3.23) |
| Vertikal target | Gabungan klinik gigi/kesehatan, F&B, retail | Validasi taksonomi & katalog opportunity (Lampiran B) |
| Cakupan wilayah | Area target customer berada di cakupan dataset (Q-02), ≥ 2 kota tier 1–2 | Hindari "Data terbatas" massal (R-03) |
| Offering | Offering dapat dipetakan ke offering tag dengan tipe opportunity aktif | R-04. Preset bobot (FR-ICP-004) |
| Aktivitas sales | ≥ 2 user aktif, outreach ≥ 30 prospek/minggu | North star ≥ 50 actioned/minggu terukur |
| Kebutuhan integrasi | ≥ 2 workspace punya target webhook nyata (CRM/Sheet/automation) | RG-3 di dunia nyata + survei CRM V1 (D-10) |
| Komitmen | Sesi feedback 30 menit/minggu, bersedia ikut usability, menandatangani perjanjian beta + DPA, LOI harga | Feedback loop dan konversi (Q-D-14) |

### 8.2 Rencana onboarding

| Waktu | Aktivitas | Owner |
|---|---|---|
| SPR-06..SPR-08 | Rekrut kandidat, wawancara discovery (Offering, wilayah, alur kerja), shortlist 10–12 | Founder, PM |
| SPR-10..SPR-11 | Perjanjian beta + pemberitahuan privasi (Q-D-08). Uji formatif dengan 3 kandidat (`US-PLAT-032`) | Founder, Design, QA |
| Awal SPR-13 (setelah M-2) | **Wave 1** (5 workspace): kickoff 60 menit, grant credit beta (FLOW-24), setup Offering + ICP bersama (FLOW-01) sambil mengukur TTFR, kanal chat dukungan | PM, Ops |
| SPR-14..SPR-15 | **Wave 2**: aktifkan webhook, AI brief, upload, territory per workspace. Tambah hingga 10 workspace | PM, TL |
| SPR-15..SPR-17 | UAT per persona (05 §3.24), rekrut 5 peserta usability sumatif yang tidak ikut uji formatif, negosiasi harga | QA, Design, Founder |

### 8.3 Feedback loop

| Kanal | Sumber | Diproses | Keluaran |
|---|---|---|---|
| Relevansi opportunity & skor | FR-INTL-010 (`US-INTL-010/011`) | Review mingguan metrik acceptance per tipe & offering tag | Kalibrasi severity/preset → bump `priority-X.Y.Z` lewat QG-24 (R-04) |
| Kualitas data | "Laporkan data" FR-DATA-014 | Triase ops (SCR-36), koreksi FR-OPS-009 | Koreksi, merge review, catatan untuk snapshot berikutnya |
| Kualitas AI brief | FR-AI-007 (stretch). Bila belum ada: form in-app | Review mingguan sampel + eval QG-25 | Bump `brief-X.Y.Z` |
| Usability & workflow | Sesi mingguan, rekaman (izin), tiket | Triase defect 2×/minggu (05 §5.4) | Story baru / perbaikan masuk refinement |
| Integrasi | Survei CRM & webhook | PM | Pilihan CRM V1 (D-10) |

### 8.4 Metrik yang dipantau (PRD §5)

| Metrik | Target beta | Mulai diukur | Tindakan bila meleset |
|---|---|---|---|
| North star: actioned prospects / workspace / minggu | ≥ 50 | SPR-13 | Wawancara workspace di bawah target. Cek friksi export/assign |
| Activation rate (ICP + save dalam 7 hari) | ≥ 60% | SPR-13 | Perbaiki onboarding (FLOW-01) |
| Time-to-first-result (median) | ≤ 5 menit | SPR-12 | Tuning pre-ranking (NFR-PERF-05), wizard |
| Search-to-save / profile open | ≥ 25% / ≥ 8% | SPR-13 | Review relevansi urutan & baris hasil |
| Opportunity acceptance | ≥ 60% | SPR-13 | Kalibrasi katalog (R-04) |
| Verified contact rate | ≥ 70% | M-1 (data) | Komunikasi coverage jujur (R-01), upload customer |
| Evidence coverage | ≥ 95% | M-1 | Blokir publish snapshot (QG-34) |
| Duplicate precision / recall | ≥ 98% / ≥ 90% | M-1 | Tuning ER, review queue |
| AI brief grounding | ≥ 98% | SPR-14 | Rollback versi prompt |
| Webhook success | ≥ 99% | SPR-13 | Investigasi DLQ & endpoint customer |
| Week-4 retention | ≥ 40% | SPR-16 (wave 1 minggu ke-4) | Exit interview, prioritas ulang backlog |
| Gross margin / enriched lead | Setelah pricing | M-3 | Biaya AI (FR-OPS-004) |

### 8.5 Exit beta → berbayar

1. M-3 lulus (RG-1..RG-7, §3.1).
2. Sign-off UAT ≥ 3 workspace (05 §3.24) dan ≥ 3 komitmen bayar/LOI (Q-D-14).
3. Activation ≥ 60% dan retention minggu ke-4 ≥ 40% pada workspace wave 1. Bila meleset, founder memutuskan lanjut atau perpanjang beta. Metrik ini tidak termasuk release gate PRD.
4. Harga, paket (D-09), dan proses invoice manual + grant credit (FR-BILL-006) siap. Workspace beta dimigrasi ke plan berbayar tanpa kehilangan data.

---

## 9. Definition of Ready / Definition of Done

### 9.1 Definition of Ready (story)

- [ ] Ada ID, epic, dan referensi FR/NFR/BR. Acceptance criteria ditulis atau merujuk acceptance FR.
- [ ] ID TC baru/terdampak dicatat di katalog 05 (05 §5.1 entry "Dev testing").
- [ ] UI: SCR/FLOW dirujuk, desain siap (state baku 03 §4.0, copy id-ID & en-US, catatan a11y).
- [ ] API: draf kontrak OpenAPI/event sudah di-review. **Wajib untuk UI-first.**
- [ ] Data/fixture tersedia (golden set, `msp-synth`, EC-xx). Dampak migrasi (expand/contract) diidentifikasi.
- [ ] Dependensi selesai atau terjadwal lebih awal. Keputusan D-xx/Q-xx terkait sudah ditutup, atau default Proposed diterima tertulis.
- [ ] Estimasi ≤ 8 SP. Bila lebih, story dipecah.
- [ ] Rencana flag (nama, owner, kedaluwarsa) bila dark launch. Catatan privasi/keamanan (RLS, PII, SSRF) bila relevan.

### 9.2 Definition of Done (story) — selaras PRD §12.2

- [ ] Acceptance criteria FR lulus dan tertaut ke TC berotomasi (QG-37).
- [ ] Pipeline MR hijau: QG-01..QG-22 yang relevan, termasuk coverage (QG-06/07/08), isolasi tenant (QG-09), dan privasi (QG-10).
- [ ] Status gagal eksplisit dan dapat di-retry. Tidak ada aksi yang gagal secara diam-diam.
- [ ] Event analytics (03 §9) dan audit (FR-AUD-001) terpasang bila relevan.
- [ ] Copy id-ID & en-US tersedia. axe tanpa pelanggaran serious/critical (QG-16).
- [ ] Dokumentasi API/event diperbarui. oasdiff tanpa breaking change yang tidak direncanakan (QG-13). Migrasi mengikuti 06 §9.
- [ ] Ter-deploy di staging dan `staging-verified` (QG-29). Bisa didemokan.
- [ ] Dokumen terkait (04/05/03) diperbarui di MR yang sama bila perilaku berubah (§10.3).
- [ ] Story API dari pasangan UI-first: integrasi UI + E2E Playwright lulus.

### 9.3 Definition of Done (release / milestone)

| Level | Kriteria |
|---|---|
| Epic | Semua story inti *done*. Acceptance FR epic lulus end-to-end di staging. Semua TC P0 epic berotomasi |
| Milestone | Exit criteria §3.1 terpenuhi dan dibuktikan di issue Release Readiness. Regresi pre-release (05 §5.1). Waiver hanya untuk S3/S4 dan TC P1, ditandatangani Product + Tech lead. **Tidak ada waiver** untuk RG-4, RG-7, S1, atau kebocoran PII (05 §5.1) |
| Rilis produksi | Pipeline tag `v*` (QG-27/28/33), canary sehat, smoke pasca-deploy, catatan rilis, runbook diperbarui |

---

## 10. Ceremonies & governance

### 10.1 Kadens

| Ceremony | Kapan | Durasi | Peserta | Keluaran |
|---|---|---|---|---|
| Sprint planning | Senin, hari 1 | 2 jam | Tim inti | Sprint goal, commit per lane (§2.3), cek dependensi §4.2 yang jatuh tempo |
| Daily standup | Harian | 15 menit | Tim inti | Hambatan |
| Backlog refinement | Rabu minggu 1 | 1 jam | PM, TL, Design, QA (+ engineer rotasi) | Story memenuhi DoR untuk 2 sprint ke depan. Kontrak UI-first di-review |
| Design review (dual-track) | Mingguan | 45 menit | Design, PM, FE, TL | Desain siap 1 sprint lebih dulu |
| Sprint review / demo | Jumat minggu 2 | 1 jam | Tim + founder (+ beta mulai SPR-12, opsional) | Demo di staging. Update velocity & tabel §2.3 |
| Retro | Setelah review | 45 menit | Tim inti | Maks. 2 aksi perbaikan |
| Triase defect | 2×/minggu, harian saat hardening | 30 menit | QA (lead), PM, TL | Severity & prioritas (05 §5.3–5.4) |
| Review keputusan & risiko | Dua mingguan (saat planning) | 30 menit | Founder, PM, TL | Status D-xx/Q-xx/Q-D-xx, update §4.2 & §12 |
| Review ops | Mingguan mulai SPR-05 | 30 menit | DevOps, TL | Alert, backup, drift, DORA (06 §16) |
| Beta sync | Mingguan mulai SPR-12 | 30 menit/workspace | PM, CS/founder | Feedback → refinement |
| Go/no-go | M-1, M-2, M-3 | 1 jam | Founder, PM, TL, QA | Keputusan tertulis + waiver |

### 10.2 Proses keputusan

1. Pertanyaan baru dicatat sebagai `Q-xx` (00 §8) atau `Q-<doc>-nn` lokal, beserta owner dan sprint "dibutuhkan paling lambat".
2. Bila belum diputuskan sampai sprint tersebut, default berlabel *Proposed* dipakai. Risiko dicatat di §12.
3. Keputusan dicatat di **register D-xx (00 §7)** lewat MR berlabel `decision`: status Proposed → Accepted/Rejected, alasan, tanggal. Keputusan teknis juga di ADR 04 §1.3. Pertanyaan `Q-CI-nn` dan `Q-D-nn` dipindahkan ke 00 §7 setelah diputuskan.
4. Dampak ke backlog (story baru, estimasi ulang, pindah sprint) diterapkan di dokumen ini pada MR yang sama atau sprint review berikutnya.

### 10.3 Aturan dokumen (docs-as-code)

- Semua dokumen di `docs/` diubah lewat MR dan lolos QG-22 (markdownlint + link).
- Prioritas/release FR hanya berubah lewat MR ke PRD yang di-approve PM + Founder. Dokumen lain mengikuti PRD.
- MR fitur yang mengubah perilaku juga memperbarui kontrak (`contracts/`), TC (05), dan desain teknis (04) bila terdampak.
- Dokumen ini diperbarui setiap sprint review (velocity, §2.3, status story) dan setiap milestone (timeline, risiko). Versi dinaikkan per milestone.
- Permintaan scope baru di tengah sprint masuk triase PM. Bila P0 untuk release berjalan, story lain dengan SP setara dikeluarkan atau milestone digeser secara eksplisit.

---

## 11. RACI

R = Responsible, A = Accountable, C = Consulted, I = Informed.

| Deliverable / keputusan | Founder | PM | Tech lead | Backend | Frontend | QA | DevOps | Design |
|---|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|
| Prioritas backlog & scope release | C | A/R | C | I | I | C | I | C |
| Keputusan produk (D-03..D-10, D-16) | A | R | C | I | I | I | I | C |
| Keputusan teknis (ADR, D-11/12/18..21) | C | C | A/R | C | C | C | C | I |
| Brand final (D-17) | A | C | I | — | C | — | — | R |
| Pengadaan server & DC (Q-09, `app-02`) | A | I | C | — | — | — | R | — |
| Jawaban dataset & lisensi (Q-01..Q-05, Q-10) | A/R | C | C | C | — | C | — | — |
| Sprint 0 bootstrap (EP-01/02) | I | I | A | R | R | C | R | — |
| Pipeline data & ER (EP-09..EP-12) | I | C | A | R | — | C | C | — |
| Scoring, opportunity & bump versi skor | C | A | R | R | — | C | — | — |
| Ops console | I | A | C | R | R | C | C | C |
| UI customer (EP-16..EP-21) | I | A | C | R | R | C | — | R |
| Credit, ledger & entitlement (EP-22) | C | A | R | R | R | C | — | C |
| Export, webhook & kontrol SSRF | I | C | A | R | R | C | R | — |
| AI brief & eval (EP-25) | C | A | R | R | R | R | — | C |
| Labeling golden set | I | A | C | C | — | R | — | — |
| Usability RG-1/RG-2 | I | C | I | — | C | R | — | A/R |
| Security review RG-7 | I | I | A | R | R | R | R | — |
| Drill RG-5, DR & HA `app-02` | I | I | A | C | — | C | R | — |
| Program beta (rekrut, onboarding, feedback) | A/R | R | C | I | I | C | I | C |
| Go/no-go M-1..M-3 | A | R | R | I | I | R | C | C |
| Pricing & proses invoice (D-09) | A/R | R | I | C | — | — | — | — |
| Update dokumen (docs-as-code) | I | A | R | R | R | R | R | R |

---

## 12. Risiko delivery

| ID | Risiko | L | I | Mitigasi | Owner | Ref |
|---|---|:-:|:-:|---|---|---|
| RD-01 | Jawaban dataset (skema, volume, lisensi) terlambat | T | T | `msp-synth` dengan profil fill-rate. Mapping spec terisolasi. Batas SPR-01 untuk sampel | Founder | Q-01..Q-05, R-01, R-07 |
| RD-02 | Server terlambat sehingga Sprint 0 tertahan | S | T | Mulai lokal + server sewa sementara di Indonesia. Produksi baru dibutuhkan SPR-04 | Founder, DevOps | Q-09 |
| RD-03 | Lane Go kelebihan beban (jalur kritis tanpa float) | T | T | Kalibrasi velocity SPR-02. Opsi skenario D (+1 Go). Stretch dikorbankan lebih dulu. Negosiasi scope | TL, PM | §2.4 |
| RD-04 | ER di bawah ambang RG-6 saat M-1 | S | T | Golden set sejak SPR-03, holdout 30%, review queue, buffer tuning di SPR-06 | Data eng, QA | RG-6 |
| RD-05 | Kapasitas labeling golden set tidak ada | S | S | Pelabel ditetapkan SPR-01. Mulai 1.000 pasangan | PM | OQ-T-01 |
| RD-06 | RG-1/RG-2 (usability) gagal di akhir | S | T | Prototype formatif SPR-03, uji formatif SPR-11, sumatif SPR-16 dengan 1 sprint perbaikan | Design | RG-1, RG-2 |
| RD-07 | D-20 terlambat sehingga MFA ops (MVP 0) dan UI auth tertunda | S | S | Default Kratos. Port provider-agnostic (ADR-19). Batas SPR-01 | TL | D-20 |
| RD-08 | D-15 / DPA provider AI terlambat sehingga AI brief (P0) tertunda dan M-3 terblokir | S | T | Abstraksi + fake provider. AI di wave 2. Batas SPR-11 | Founder | D-15, R-05 |
| RD-09 | Kebocoran antar tenant atau temuan security terlambat | R | T | QG-09 sejak SPR-07 (tanpa waiver), ZAP nightly, review SPR-16 + retest SPR-17 | TL | R-06, RG-7 |
| RD-10 | Beta customer kurang atau tidak aktif | S | T | Rekrut sejak SPR-06, LOI, wave 1 minimal 5 | Founder | Q-07 |
| RD-11 | Hardware `app-02` terlambat sehingga M-3 terblokir | S | T | Pesan SPR-12. Role Ansible sama dengan `app-01` | DevOps | R-08 |
| RD-12 | PoC SeaweedFS gagal (object lock/enkripsi) | S | S | Garage sebagai alternatif. Keputusan SPR-01 | DevOps | TQ-17, Q-CI-13 |
| RD-13 | Libur/cuti bersama mengurangi kapasitas | T | S | Faktor kalender §2.2. Cek SKB 2027 | PM | — |
| RD-14 | Kontrak bergeser setelah UI-first sehingga ada rework FE | S | S | Kontrak di-review TL. Jarak ≤ 4 sprint. Perubahan = estimasi ulang | TL, FE | P-6 |
| RD-15 | DevOps 0,5 FTE menjadi bottleneck di bootstrap & launch | T | S | 1,0 FTE di SPR-00..01 & SPR-14..17. TL backup (Q-D-18) | Founder | — |
| RD-16 | Performa 5 juta (NFR-PERF-01/07) tidak tercapai | S | T | TC-DATA-034 di SPR-06, k6 nightly, harness SPR-13, server perf (Q-D-04) | TL | NFR-PERF |
| RD-17 | Opportunity dianggap tidak relevan oleh beta | S | S | Feedback di wave 1, kalibrasi mingguan | PM | R-04 |
| RD-18 | Ketentuan hukum PDP (DSR, retensi, ToS/DPA) terlambat | S | T | Konsultan dikontrak SPR-08. DSR manual di wave 1 | Founder | BR-PRIV-03, Q-D-08 |
| RD-19 | Lisensi melarang data nyata di staging/golden set | S | S | Data sintetis + sampel yang diizinkan. Bukti RG-6 di produksi | Founder, QA | Q-05, Q-CI-09 |
| RD-20 | Ketergantungan pada satu orang (data eng: ER/scoring) | S | T | Pairing, ADR & runbook, review oleh TL | TL | — |

L = likelihood, I = impact. T = tinggi, S = sedang, R = rendah.

---

## 13. Pertanyaan terbuka

| ID | Pertanyaan | Default sementara | Dibutuhkan paling lambat | Owner |
|---|---|---|---|---|
| Q-D-01 | Ukuran & komposisi tim aktual, serta tanggal mulai SPR-00. **Semua tanggal di dokumen ini estimasi** | Tim baseline §2.1, mulai 2 Nov 2026 | Sebelum SPR-00 | Founder |
| Q-D-02 | Kalibrasi velocity: apakah 10 SP/engineer realistis? | Diukur setelah SPR-02. Replan bila deviasi > 15% | SPR-03 | TL, PM |
| Q-D-03 | Siapa pelabel golden set dan berapa jamnya (≥ 2 orang × ±50 jam)? | QA + 1 orang Product/ops | SPR-01 | PM |
| Q-D-04 | Server perf setara produksi untuk TC-DATA-034 dan PERF-01..07 | Pinjam hardware produksi sebelum go-live + faktor skala | SPR-04 | DevOps |
| Q-D-05 | Provider email transaksional (TQ-12) | SMTP relay di Indonesia | SPR-04 | TL |
| Q-D-06 | Basemap peta (TQ-11) | Tanpa basemap / OSM self-hosted | SPR-10 | Design, TL |
| Q-D-07 | ~~FR-DATA-014 P1 padahal P0 bergantung padanya~~ | **Resolved:** PRD menaikkan FR-DATA-014 ke P0 MVP 1 (`US-DATA-020/021`) | — | PM |
| Q-D-08 | Konsultan hukum: perjanjian beta, pemberitahuan privasi, SLA DSR, retensi & pseudonimisasi, ToS/DPA, DPA provider LLM | Default 04/05 | Beta: SPR-11. DSR/retensi: SPR-14 | Founder |
| Q-D-09 | Anggaran & jadwal pengadaan `app-02` + VIP/LB (R-08) | Spesifikasi identik `app-01`. Keepalived/VRRP | Pesan SPR-12 | Founder, DevOps |
| Q-D-10 | Bolehkah closed beta dimulai dengan subset P0 (wave 1, §1.3)? | Ya. M-3 tetap menuntut seluruh P0 | SPR-08 | PM, Founder |
| Q-D-11 | ~~FR-INTL-010 butuh metrik di ops console, padahal FR-OPS-003 P1~~ | **Resolved:** acceptance PRD kini cukup laporan ops sederhana (`US-INTL-010`). Dashboard penuh tetap stretch (`US-OPS-019`) | — | PM |
| Q-D-12 | ~~Akses aman ops console MVP 0 sebelum MFA ops~~ | **Resolved:** PRD menetapkan FR-AUTH-006 untuk akun ops di MVP 0. MFA ops dibangun di SPR-02 (`US-PLAT-041`, `US-AUTH-001`), tanpa akses interim | — | TL |
| Q-D-13 | Credit awal beta (OQ-UX-25) dan harga final (D-09) | Lampiran E sebagai grant beta | Beta: SPR-11. Harga: SPR-15 | Founder |
| Q-D-14 | Target konversi early paid: berapa workspace berbayar / LOI di M-3? | ≥ 3 komitmen | SPR-12 | Founder |
| Q-D-15 | ~~04 §17.5 masih menulis "`app-02` di V1", berbeda dengan PRD R-08 (sebelum M-3)~~ **Resolved:** 04 sudah memakai "`app-02` aktif sebelum M-3 (R-08)" | Ikuti PRD. 04 diperbarui | SPR-12 | TL |
| Q-D-16 | ~~05 §8.4 masih menyebut 90 FR P0, padahal PRD kini punya 92. **FR-OPS-009** belum punya TC, FR-DATA-014 (kini P0) belum di §8.4, dan FR-AUTH-006 (ops) belum di kolom MVP 0 §5.2~~ **Resolved:** 05 kini mencakup 92 FR P0 (TC-OPS-017 untuk FR-OPS-009, TC-DATA-033 P0) | Tambah TC & perbarui 05 | SPR-02 (FR-AUTH-006), SPR-12 (lainnya) | QA |
| Q-D-17 | ~~FR MVP 0 butuh konteks workspace/ICP sebelum UI MVP 1~~ | **Resolved:** PRD §6 menetapkan MVP 0 memakai workspace internal + fixture ICP/Offering buatan ops (`US-ICP-002`, `US-OPS-009`) | — | PM |
| Q-D-18 | DevOps 1,0 FTE di SPR-00..01 dan SPR-14..17? | Ya, atau kontrak DevOps sementara | Sebelum SPR-00 | Founder |
| Q-D-19 | Bila tim hanya 3 engineer (§2.4 C), P0 mana yang dinegosiasikan ulang lewat PRD? | Belum ada. Kandidat: upload (FR-DATA-012), territory ranking (FR-GEO-003) | Sebelum SPR-00 | Founder, PM |
| Q-D-20 | ~~FR-DATA-005 vs FR-DATA-011 tumpang tindih (DNS/MX)~~ | **Resolved:** FR-DATA-005 = keberadaan MX untuk verifikasi (`US-DATA-011`). FR-DATA-011 = identifikasi penyedia dari MX/SPF dengan cache DNS yang sama (`US-DATA-022`, stretch) | — | PM, TL |
