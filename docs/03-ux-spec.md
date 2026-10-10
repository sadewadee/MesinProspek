# 03 — UX Specification

## MesinProspek — Sales Intelligence Engine (Sellable MVP)

| Field | Value |
|---|---|
| Dokumen | UX Specification |
| Versi | 0.1 (Draft) |
| Tanggal | 2026-10-10 |
| Basis | `01-prd.md` v0.1 (sumber kebenaran), `00-analisis-dokumen-awal.md`, MVP Pack v1.2 §07 Presentation, §08 Action, Rev 1.2 §6 Presentation Additions |
| Bergantung pada | `02-design-system/` (nama token semantik; nilai token tidak didefinisikan di sini) |
| Status crawler | **PAUSED** (D-01). Semua layar bekerja di atas dataset snapshot + upload customer. Field yang tidak ada di sumber tampil "Tidak diketahui", fiturnya tidak dihapus. |
| Audience | Design, Frontend, Product, QA |

> Konvensi: `SCR-nn` layar, `FLOW-nn` user flow, `WF-nn` wireframe, `P-nn` pola UI, `OQ-UX-nn` pertanyaan terbuka UX. Rentang nomor sengaja diberi celah: SCR-01..25 aplikasi customer, SCR-30..40 ops console, SCR-50..59 V1; FLOW-01..11 customer, FLOW-20..24 ops. Setiap layar dan flow mencantumkan FR/BR yang diimplementasikan. Bila dokumen ini tampak bertentangan dengan PRD, **PRD yang berlaku**; konflik dan asumsi dicatat di §10.

---

## 1. Prinsip UX

| ID | Prinsip | Asal | Konsekuensi desain |
|---|---|---|---|
| UX-P1 | **Setiap layar menjawab satu pertanyaan** | MVP Pack §07 | Setiap SCR menuliskan "pertanyaan yang dijawab". Informasi yang tidak membantu menjawabnya dipindah ke drawer, tab, atau layar lain. |
| UX-P2 | **Action over dashboard** | MVP Pack App. B | Setiap kartu, tabel, dan metrik punya langkah lanjut (buka, simpan, tugaskan, ekspor). Tidak ada angka "pajangan" tanpa aksi, termasuk di ops console. |
| UX-P3 | **Trust terlihat** | Rev 1.1 §F, PRD OBJ-2 | Jenis data, keyakinan, kemutakhiran, dan akses ke bukti tampil di samping nilai, bukan di halaman terpisah. |
| UX-P4 | **Unknown over invented** | MVP Pack App. B, BR-TRUST-06 | "Tidak diketahui" adalah state kelas satu, tidak pernah kosong, "0", atau "-". Ketiadaan data ditulis "tidak ditemukan pada data sumber", bukan "tidak punya". |
| UX-P5 | **Progressive disclosure bukti** | MVP Pack §07 (hierarki profil) | Level 1: badge ringkas. Level 2: tooltip (angka keyakinan, tanggal). Level 3: evidence drawer (sumber, versi, nilai mentah, nilai alternatif). |
| UX-P6 | **Biaya selalu eksplisit** | FR-EXP-002, FR-BILL-004, D-08, BR-CRED-07 | Tombol berbayar memuat biayanya. Tidak ada potongan kredit tanpa konfirmasi. Kredit ditahan, ditagih, dilepas, dan refund terlihat di ledger dan di objek terkait. |
| UX-P7 | **Skor bisa dijelaskan dalam 10 detik** | RG-1 | Baris hasil memuat alasan teratas (peluang teratas). Panel "Kenapa prospek ini?" berjarak satu klik dari skor mana pun. |
| UX-P8 | **Hipotesis bukan janji** | BR-OPP-05, BR-TRUST-05, Rev 1.2 §8 | Peluang, sales angle, revenue, dan metrik geo memakai label "Hipotesis"/"Perkiraan" beserta copy guardrail. |
| UX-P9 | **Konteks selalu eksplisit** | FR-AUTH-002, FR-ICP-006 | Workspace dan ICP aktif selalu terlihat di header. Skor selalu disebut "untuk ICP X". |
| UX-P10 | **Gagal itu eksplisit dan bisa diulang** | PRD §12.2, MVP Pack §10 DoD | Setiap aksi async punya status, alasan gagal, tombol coba lagi, dan info kredit (dilepas/refund). Tidak ada kegagalan diam-diam. |

---

## 2. Information architecture

### 2.1 Sitemap aplikasi customer

```
Publik
|- Masuk / Buat akun / Verifikasi email / Atur ulang password ...... SCR-01
|- Terima undangan .................................................. SCR-01
'- Formulir permintaan subjek data .................................. SCR-25
Aplikasi (/app, butuh login + email terverifikasi)
|- Onboarding (sekali per workspace baru) ........................... SCR-02
|- Beranda .......................................................... SCR-03
|- Cari prospek ..................................................... SCR-04
|  '- Profil prospek ................................................ SCR-05
|     |- Evidence drawer ............................................ SCR-06
|     |- Panel AI brief ............................................. SCR-07
|     '- Dialog laporkan data ....................................... SCR-23
|- Peta wilayah ..................................................... SCR-08
|- Peringkat wilayah ................................................ SCR-09
|  '- Detail area ................................................... SCR-10
|- List & segmen (termasuk pencarian tersimpan) ..................... SCR-11
|  '- Detail list / segmen .......................................... SCR-12
|- Tugas saya ....................................................... SCR-13
|- Upload ........................................................... SCR-16
|- Ekspor & proses (job center) ..................................... SCR-15
|- ICP & penawaran .................................................. SCR-17
|- Pengaturan
|  |- Workspace ..................................................... SCR-18
|  |- Anggota & role ................................................ SCR-19
|  |- Webhook & log pengiriman ...................................... SCR-20
|  |- Pemakaian & kredit ............................................ SCR-21
|  '- Log audit ..................................................... SCR-22
|- Akun saya ........................................................ SCR-24
'- (lintas layar) Dialog konfirmasi biaya ........................... SCR-14
```

### 2.2 Sitemap ops console

```
Ops console (/ops, akun internal, MFA wajib - FR-AUTH-006; menu per peran internal, §2.5)
|- Data Health ...................................................... SCR-30
|- Pipeline Health & DLQ ............................................ SCR-31
|- Intelligence Health & Cost (P1) .................................. SCR-32
|- Merge review ..................................................... SCR-33
|- Dataset snapshot (status & laporan validasi) ..................... SCR-34
|- Privasi: suppression & permintaan subjek data .................... SCR-35
|- Laporan data dari user ........................................... SCR-36
|- Tenant & kredit (termasuk akses support read-only) ............... SCR-37
|- Feature flag & kill switch ....................................... SCR-38
|- Log audit internal ............................................... SCR-39
'- Akun & peran internal ............................................ SCR-40
```

### 2.3 Model navigasi

| Elemen | Posisi | Isi & perilaku | FR |
|---|---|---|---|
| Sidebar kiri | Tetap di ≥ `breakpoint.lg` (lebar 240 px, dapat diciutkan ke 64 px ikon + tooltip). Di bawahnya menjadi drawer via tombol menu. | Grup *Kerja*: Beranda, Cari prospek, Peta wilayah, Peringkat wilayah, List & segmen, Tugas saya (badge jumlah tugas terbuka). Grup *Data*: Upload, Ekspor & proses (badge job berjalan). Grup bawah: ICP & penawaran, Pengaturan, Bantuan. Item yang tidak diizinkan role tidak dirender. | FR-AUTH-004 |
| Workspace switcher | Puncak sidebar | Nama workspace + plan. Dropdown: daftar workspace user. Berpindah workspace → kembali ke Beranda workspace tujuan dan cache klien workspace lama dikosongkan. | FR-AUTH-002 |
| ICP context switcher | Header kiri | Label "ICP aktif" + nama ICP + penawaran + versi. Dropdown: daftar ICP workspace dengan status skor (Siap / Menghitung n%), tautan "Kelola ICP". **ICP aktif menentukan skor, urutan, peluang, dan peringkat wilayah di semua layar.** | FR-ICP-006, BR-SCORE-01 |
| Pencarian cepat | Header tengah | Shortcut `/`. Enter → SCR-04 dengan query. Saran: 5 pencarian tersimpan terakhir. | FR-SRCH-001, FR-SRCH-005 |
| Saldo kredit | Header kanan | "Kredit 184 · AI 42" (= saldo **tersedia**). Klik → popover: tersedia, **ditahan** untuk proses berjalan (mis. "35 ditahan · ekspor berjalan", BR-CRED-07), masa berlaku (BR-CRED-06), tautan Pemakaian (Owner/Admin/Manager). Saldo tersedia ≤ 10% grant bulanan → ikon peringatan + teks "Saldo rendah". | FR-BILL-002, BR-CRED-07 |
| Indikator job | Header kanan | Ikon + jumlah job berjalan → popover 5 job terakhir → SCR-15. | FR-EXP-001 |
| Menu user | Header paling kanan | Akun saya, Bahasa (Indonesia/English), Bantuan, Keluar. | FR-AUTH-005, NFR-L10N-01 |
| Banner sistem | Di bawah header (`z.sticky`) | Mode pemeliharaan read-only (`maintenance.read_only`, state READ-ONLY §4.0), kill switch aktif, skor sedang dihitung ulang, fair-use ≥ 80%, mode support. Maksimal 1 banner tampil; sisanya diantrekan (error > peringatan > info; READ-ONLY selalu menang). | FR-OPS-007, NFR-PERF-05, `04` §18.3 |

Aturan konteks:
- ICP aktif disimpan **per user per workspace** (OQ-UX-03) dan tercermin di URL sebagai `icp=<id>` pada layar yang bergantung skor, agar tautan dan pencarian tersimpan dapat direproduksi (FR-SRCH-005).
- URL hasil mereproduksi **query, filter, sortir, dan ICP**, bukan posisi halaman. Daftar hasil memakai cursor keyset yang terikat epoch skor (`04` §10.5); cursor disimpan di state klien/riwayat browser saja, sehingga tautan yang dibagikan selalu mulai dari hasil teratas.
- Workspace aktif berasal dari sesi. Tautan buatan sistem (undangan, ekspor siap, deep link) membawa `ws=<slug>`. Bila `ws` ≠ workspace aktif → interstitial "Tautan ini milik workspace *X*. Pindah ke *X*?". Data workspace lain tidak pernah tampil tanpa perpindahan eksplisit (FR-AUTH-002, OQ-UX-20).

### 2.4 Daftar route (usulan)

| Route | SCR | Catatan |
|---|---|---|
| `/auth/login`, `/auth/signup`, `/auth/verify`, `/auth/reset` | SCR-01 | Managed auth (FR-AUTH-001) |
| `/invite/[token]` | SCR-01 | Terima undangan (FR-AUTH-003) |
| `/privacy/request` | SCR-25 | Publik, tanpa login (FR-AUD-004) |
| `/app/onboarding?step=offering\|icp\|results` | SCR-02 | Redirect otomatis bila workspace belum punya ICP |
| `/app` | SCR-03 | Beranda |
| `/app/search?q=&f=&icp=&sort=` | SCR-04 | `f` = filter ter-encode (termasuk filter sumber "upload Anda"). Query, filter, sortir, dan ICP ada di URL (back/forward, berbagi tautan); posisi muat/cursor tidak (§2.3). |
| `/app/prospects/[companyId]?est=&icp=&evidence=` | SCR-05, SCR-06 | `est` = establishment terpilih; `evidence=<claimId>` membuka drawer |
| `/app/map?region=&level=&metric=&area=&view=map\|table` | SCR-08 | `view=table` = alternatif aksesibel peta |
| `/app/territories?region=&level=` | SCR-09 | |
| `/app/areas/[areaCode]` | SCR-10 | Kode wilayah Kemendagri (D-13) |
| `/app/lists?tab=lists\|segments\|saved` | SCR-11 | |
| `/app/lists/[listId]`, `/app/segments/[segmentId]` | SCR-12 | |
| `/app/tasks` | SCR-13 | |
| `/app/jobs`, `/app/jobs/[jobId]` | SCR-15 | |
| `/app/uploads`, `/app/uploads/new`, `/app/uploads/[uploadId]`, `/app/uploads/[uploadId]?tab=possible` | SCR-16 | `tab=possible` = daftar "Mungkin cocok" untuk dikonfirmasi (FLOW-05 #6) |
| `/app/icp`, `/app/icp/new`, `/app/icp/[icpId]`, `/app/offerings/[offeringId]` | SCR-17 | |
| `/app/settings/{workspace,members,webhooks,usage,audit}` | SCR-18..22 | `/app/settings/webhooks/[endpointId]` untuk log pengiriman |
| `/app/account` | SCR-24 | |
| `/ops/data-health`, `/ops/pipeline`, `/ops/pipeline/dlq`, `/ops/intelligence` | SCR-30..32 | |
| `/ops/merge-review`, `/ops/merge-review/[candidateId]` | SCR-33 | |
| `/ops/datasets`, `/ops/datasets/[snapshotId]` | SCR-34 | |
| `/ops/privacy`, `/ops/privacy/[requestId]` | SCR-35 | |
| `/ops/data-reports`, `/ops/tenants`, `/ops/tenants/[workspaceId]` | SCR-36, SCR-37 | |
| `/ops/flags`, `/ops/audit` | SCR-38, SCR-39 | |
| `/ops/accounts` | SCR-40 | Ops Admin saja |

Dialog (SCR-14, SCR-23) tidak punya route. Evidence drawer memakai query param agar tombol Back menutupnya.

### 2.5 Visibilitas per role

Turunan langsung matriks RBAC PRD §8.1.1. "—" = menu/tombol tidak dirender (FR-PROF-009); akses lewat URL → state DENIED (§4.0). Server tetap menegakkan semuanya (FR-AUTH-004).

| Layar / kontrol | Owner | Admin | Manager | Member | Viewer |
|---|:-:|:-:|:-:|:-:|:-:|
| SCR-03/04/05/08/09/10 (lihat) | ✓ | ✓ | ✓ | ✓ | ✓ |
| Tombol Buka kontak | ✓ | ✓ | ✓ | ✓ | — |
| Simpan ke list, tag, catatan, status kerja | ✓ | ✓ | ✓ | ✓ | — |
| SCR-11/12 kelola list/segmen/pencarian tersimpan milik sendiri | ✓ | ✓ | ✓ | ✓ | Lihat list yang dibagikan (OQ-UX-05) |
| Tugaskan ke orang lain | ✓ | ✓ | ✓ | — | — |
| SCR-13 Tugas saya | ✓ | ✓ | ✓ | ✓ | — |
| Ekspor / Kirim ke webhook | ✓ | ✓ | ✓ | ✓ bila "Member boleh ekspor" aktif (OQ-UX-30) | — |
| SCR-16 Upload | ✓ | ✓ | ✓ | ✓ | — (OQ-UX-04) |
| SCR-17 kelola ICP & penawaran | ✓ | ✓ | ✓ | Lihat + ganti ICP aktif | Lihat + ganti ICP aktif |
| SCR-07 Buat AI brief | ✓ | ✓ | ✓ | ✓ | Baca brief yang sudah ada (OQ-UX-06) |
| SCR-20 Webhook | ✓ | ✓ | — | — | — |
| SCR-19 Anggota & role | ✓ | ✓ | — | — | — |
| SCR-21 Pemakaian & kredit | ✓ | ✓ | ✓ | Saldo di header & dialog biaya saja (OQ-UX-31) | — |
| SCR-22 Log audit | ✓ | ✓ | — | — | — |
| SCR-18 umum & "Member boleh ekspor" | ✓ | ✓ | — | — | — |
| SCR-18 plan & tutup workspace | ✓ | — | — | — | — |
| SCR-15 Ekspor & proses | Semua job | Semua job | Semua job | Job sendiri | — |

**Ops console — peran internal (PRD §8.1.2).** Semua akun internal wajib MFA (FR-AUTH-006); satu orang boleh memegang > 1 peran (hak = gabungan). "—" = menu/tombol tidak dirender; akses lewat URL → DENIED. Server tetap menegakkan semuanya.

| Layar / kontrol ops | Ops Engineer (on-call) | Data Steward | Privacy & Billing | Ops Admin |
|---|:-:|:-:|:-:|:-:|
| SCR-30/31/32 lihat dashboard data, pipeline, intelligence, cost | ✓ | ✓ | ✓ | ✓ |
| SCR-31 redrive/buang DLQ · SCR-38 ubah flag & kill switch | ✓ | Lihat | Lihat | ✓ |
| SCR-33 merge review · SCR-36 triase laporan data & koreksi (FR-OPS-009) | — | ✓ | — | ✓ |
| SCR-34 status, riwayat & laporan validasi snapshot; tinjau dan beri rekomendasi publish/tolak (eksekusi = job pipeline rilis data, `06`) | — (status via SCR-30) | ✓ | — | ✓ |
| SCR-34 import referensi wilayah/populasi | — | ✓ | — | ✓ |
| SCR-35 suppression & permintaan subjek data | — | — | ✓ | ✓ |
| SCR-37 grant/refund kredit, ubah plan | — | — | ✓ | ✓ |
| SCR-37 akses support read-only ke data tenant (alasan wajib) | ✓ | ✓ | ✓ | ✓ |
| SCR-40 kelola akun & peran internal | — | — | — | ✓ |

Pembagian ini sama dengan kolom peran endpoint `/v1/ops/*` di `04` §11.5 (daftar tenant SCR-37 dan panel versi model SCR-32: semua peran internal). Akses baca SCR-39 belum diatur → asumsi OQ-UX-34. Semua aksi yang mengubah data (grant/refund kredit, suppression, keputusan merge, koreksi, flag, redrive, import referensi, akses support, perubahan peran) mewajibkan isian **alasan** dan tercatat di audit (FR-AUD-001).

---

## 3. User flows

Format tiap flow: tabel langkah. Kolom *Gagal / edge* menyebut penanganan khusus; state baku (LOADING, ERROR, NO-CREDIT, dst.) didefinisikan di §4.0.

### FLOW-01 — Onboarding: sign-up → hasil pertama ≤ 5 menit
Persona PER-01, PER-02, PER-04. Implements: FR-AUTH-001, FR-AUTH-002, FR-ICP-001, FR-ICP-002, FR-ICP-004, FR-ICP-005, FR-ICP-006, FR-SRCH-004. Metrik: Time-to-first-result (median ≤ 5 menit), Activation rate.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-01 | Buat akun (email + password, atau Google) | Akun dibuat; email verifikasi dikirim. Google dengan email terverifikasi → lewati #2. | FR-AUTH-001 | Email sudah terdaftar → "Masuk saja?" + tautan atur ulang password. |
| 2 | SCR-01 | Klik tautan / tempel kode verifikasi | Email terverifikasi; workspace dibuat dari isian "Nama usaha Anda" (dapat diubah) | FR-AUTH-001/002 | Tautan kedaluwarsa → "Kirim ulang". Sebelum terverifikasi tidak ada data yang dapat diakses. |
| 3 | SCR-02 langkah 1 | Pilih tag penawaran (chip katalog); nama terisi otomatis | Penawaran tersimpan; preset bobot disarankan dari tag | FR-ICP-001/004 | Tag "Lainnya" → nama wajib diisi; preset default Lampiran A. |
| 4 | SCR-02 langkah 2 | Pilih kategori + wilayah (wajib); band dan kontak opsional | Pratinjau live: "±8.900 bisnis cocok" + 3 contoh nama | FR-ICP-002 | 0 kandidat → saran melonggarkan (wilayah induk, kategori sejenis). > 200.000 kandidat → saran mempersempit (NFR-PERF-05). |
| 5 | SCR-02 langkah 2 | "Simpan & lihat hasil" | ICP v1 tersimpan dan aktif. Job skor mulai; batch kandidat dengan kecocokan tertinggi dihitung lebih dulu. | FR-ICP-005/006 | Gagal simpan → ERROR; isian tidak hilang. |
| 6 | SCR-02 langkah 3 | Menunggu (target ≤ 30 detik ke halaman pertama ber-skor: ≤ 50 kandidat teratas hasil pre-ranking) | Hasil tampil bertahap: "Menghitung skor: 3.100 dari 8.900". Coach mark pada skor, "Kenapa prospek ini?", dan kontak ter-mask. | FR-SRCH-004, NFR-PERF-05 | > 30 detik → tampilkan yang sudah dihitung + state SCORING. Job gagal → ERROR + "Hitung ulang". |
| 7 | SCR-02 → SCR-04/05 | "Mulai mencari" atau buka profil teratas | Event `first_result_viewed`. Beranda menampilkan checklist aktivasi (simpan list/segmen pertama). | PRD §5 | Anggota undangan tidak melewati wizard (workspace sudah punya ICP) → langsung SCR-03. |

Anggaran waktu: langkah 1–2 ±60 detik, 3 ±30 detik, 4–5 ±90 detik, 6 ≤ 30 detik → total ±3,5 menit (OQ-UX-01, ditutup).

### FLOW-02 — Prospecting harian
Persona PER-02. Implements: FR-SRCH-001..008, FR-PROF-001..004, FR-PROF-009, FR-SEG-001, FR-SEG-003..006, BR-CRED-02, BR-CRED-04. Metrik: Profile open rate, Search-to-save rate, North star.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-03 | Jalankan pencarian tersimpan atau buka kartu Tugas saya | SCR-04 dengan query + filter + ICP tersimpan | FR-SRCH-005 | ICP pencarian tersimpan sudah dihapus → jalankan dengan ICP aktif + banner info. |
| 2 | SCR-04 | Ubah filter / sortir | Hitungan dan facet diperbarui (debounce 300 ms) | FR-SRCH-002/003/006 | 0 hasil → ZERO + saran (FR-SRCH-008). > 10.000 → hitungan "±". |
| 3 | SCR-04 | Pindai baris; buka profil (klik / Enter) | SCR-05; posisi hasil dipertahankan; navigasi sebelumnya/berikutnya | FR-SRCH-004, FR-PROF-001 | Fair-use profil habis → FAIR-USE. |
| 4 | SCR-05 | Baca "Kenapa prospek ini?"; buka bukti kontak | Panel faktor; SCR-06 | FR-PROF-002/004 | Faktor Unknown → "Tidak diketahui — tidak dihitung". |
| 5 | SCR-05 | "Buka kontak · 1 kredit" | SCR-14 (mode buka kontak) → debit → nilai tampil + aksi salin/telepon/WhatsApp/email | FR-PROF-003, BR-CRED-02/04 | Saldo kurang → NO-CREDIT. Sudah dibuka rekan → nilai langsung tampil tanpa biaya. Klik ganda → satu debit. Flag `unlock.enabled` off → tombol nonaktif + penjelasan (FLAG-OFF). Prospek privat dari upload → kontak sudah terlihat, tanpa tombol buka. |
| 6 | SCR-05 | Simpan ke list, tag, status "Siap dihubungi", catatan | Toast + entri timeline | FR-SEG-001/003/004/006 | Gagal simpan → toast error + "Coba lagi"; perubahan optimistik dibatalkan. |
| 7 | SCR-05 → SCR-04 | Kembali / prospek berikutnya | Seleksi dan posisi scroll hasil dipertahankan | — | — |
| 8 | SCR-04 | Bulk select → "Tugaskan" (Manager ke atas) | Pilih anggota → assignment tercatat → muncul di SCR-13 penerima | FR-SRCH-007, FR-SEG-005 | Seleksi melebihi batas plan → batas ditampilkan sebelum aksi. Member: tombol tidak dirender. |

### FLOW-03 — Perencanaan territory
Persona PER-03. Implements: FR-GEO-001..005, FR-GEO-009, FR-SEG-001, FR-SEG-002, FR-SEG-005. Metrik: Activation rate, North star (assign).

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-08 | Pilih wilayah induk (kab/kota), level (kecamatan/kelurahan), metrik | Choropleth + legenda + periode data + guardrail | FR-GEO-002/009 | Batas wilayah tidak tersedia → otomatis `view=table` + info (OQ-UX-18). Basemap gagal → choropleth tanpa basemap. |
| 2 | SCR-08 | Hover / fokus / klik area | Panel area: metrik, cakupan, jumlah prospek HOT/WARM | FR-GEO-001 | Cakupan LOW → arsir + "Data terbatas"; celah pasokan dan skor area "Tidak diketahui". |
| 3 | SCR-09 | Buka "Peringkat wilayah" | Area diurutkan skor wilayah untuk ICP + penawaran aktif | FR-GEO-003 | Area cakupan LOW → grup "Tidak diperingkat — data terbatas". ICP sedang dihitung → SCORING. |
| 4 | SCR-09 | Expand baris | Breakdown skor wilayah (3 komponen) dan skor peluang pasar (4 komponen) | FR-GEO-003 | Komponen Unknown → abu-abu, "tidak dihitung". |
| 5 | SCR-10 | Buka detail area | Metrik, posisi di area pembanding, distribusi band, prospek teratas | FR-GEO-001/004 | — |
| 6 | SCR-10 → SCR-04 | "Lihat prospek di area ini" | Hasil terfilter area, urut prioritas | FR-GEO-004 | — |
| 7 | SCR-09 | Centang beberapa area → "Buat segmen dari area" | Segmen dinamis (filter wilayah ∈ pilihan + ICP aktif) → SCR-12 | FR-GEO-005, FR-SEG-002 | Nama duplikat → sufiks otomatis "(2)". |
| 8 | SCR-12 | Pilih semua (≤ batas plan) → Tugaskan; atau "Simpan sebagai list" lalu tugaskan list | Assignment tercatat di activity | FR-SEG-001/005 | Area tidak dapat ditugaskan langsung (territory management = V1.5). |

### FLOW-04 — Siapkan & bertindak
Persona PER-01, PER-02. Implements: FR-AI-001..007, FR-PROF-003, FR-EXP-001, FR-EXP-004, FR-SEG-004, FR-SEG-006. Metrik: AI brief grounding, North star.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-05/07 | "Buat brief · 1 kredit AI" (pilih ID/EN) | Cache ≤ 7 hari dan input sama → tampil langsung, "Dari cache — gratis". Selain itu generate (p95 ≤ 20 detik). | FR-AI-001/003, NFR-PERF-06 | Kredit AI habis → NO-CREDIT. Kill switch / batas biaya → FLAG-OFF. |
| 2 | SCR-07 | Baca brief; klik sitasi [n] | SCR-06 membuka claim yang dirujuk | FR-AI-001 | Validator gagal 2× → "Brief tidak dapat dibuat … kredit AI tidak ditagih" (kredit AI yang ditahan dilepas, BR-CRED-07) + "Coba lagi" (FR-AI-004). |
| 3 | SCR-07 | (P1) "Buat sales angle · 1 kredit AI" | Hipotesis pendekatan berlabel "Hipotesis" + rationale | FR-AI-002 | — |
| 4 | SCR-07 | "Salin brief" | Teks + daftar sumber ke clipboard; toast "Disalin" | — | Clipboard ditolak browser → teks terseleksi + petunjuk Ctrl/Cmd+C. |
| 5 | SCR-05 | Klik WhatsApp / Telepon / Email pada kontak yang sudah dibuka | Membuka `https://wa.me/<nomor>` / `tel:` / `mailto:` | FR-PROF-003, BR-VER-01 | Belum dibuka → tombol diganti "Buka kontak". WhatsApp selalu berlabel "Kemungkinan WhatsApp". |
| 6 | SCR-05 | Ubah status "Sudah dihubungi" + catatan | Entri timeline | FR-SEG-004/006 | — |
| 7 | SCR-05 | "Kirim ke webhook" atau "Ekspor" | SCR-14 (0 kredit bila sudah dibuka) → job → SCR-15 | FR-EXP-001/002/004, BR-CRED-03 | Belum ada endpoint aktif → Owner/Admin: "Atur webhook"; lainnya: "Minta admin mengatur webhook". |
| 8 | SCR-07 | Feedback "Membantu" / "Tidak membantu" + alasan | Tersimpan | FR-AI-007 | — |

### FLOW-05 — Upload & cocokkan list sendiri
Implements: FR-DATA-012, BR-ER-04, BR-ER-07, FR-SEG-001, BR-PRIV-03.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-16 langkah 1 | Unggah CSV/XLSX; pilih sheet | Parse header + hitung baris | FR-DATA-012 | > 50.000 baris → ditolak + saran memecah file. Format/encoding tidak dikenali → pesan + unduh template. Upload dimatikan → FLAG-OFF. |
| 2 | Langkah 2 | Petakan kolom (saran otomatis dari header id/en) | Validasi minimum: nama + (domain atau telepon atau alamat) | PRD §11 | Minimum tidak terpenuhi → "Lanjut" nonaktif + penjelasan kolom yang kurang. |
| 3 | Langkah 3 | Tinjau pratinjau 20 baris + ringkasan masalah; centang pernyataan hak memproses data | Hitungan valid dan akan-ditolak per alasan | FR-DATA-012 | — |
| 4 | Langkah 4 | "Proses" | Job async; user boleh meninggalkan halaman; progres di SCR-15 & header | FR-DATA-012 | Gagal → status eksplisit + "Proses ulang" (file tetap tersimpan). |
| 5 | Langkah 5 | Baca laporan: Cocok / Mungkin cocok / Baru / Ditolak | Match rate, distribusi keyakinan link, unduh baris ditolak (CSV + kolom alasan). Baris Baru = **prospek privat** workspace: dinilai dengan aturan yang sama dan tampil di SCR-04 secara default berlabel "Dari upload Anda". | FR-DATA-012 | Match rate < 30% → tips melengkapi kolom domain/telepon. |
| 6 | Langkah 6 (`?tab=possible`) | Tinjau "Mungkin cocok": per baris, data upload berdampingan dengan kandidat bisnis global (skor link 0,80–0,92, BR-ER-04; nama, alamat, telepon/domain yang cocok disorot) → **Tautkan** atau **Jadikan prospek privat**; aksi massal untuk baris terpilih | Tautkan → baris di-link ke bisnis global (link, bukan merge; global tidak berubah). Jadikan prospek privat → tampil sebagai prospek privat. Hitungan laporan diperbarui. | FR-DATA-012, BR-ER-04, BR-ER-07 | Baris yang **belum dikonfirmasi** diperlakukan sebagai prospek privat (default tidak di-link). Langkah boleh dilewati ("Nanti saja"); daftar tetap dapat dibuka dari indeks upload. |
| 7 | Langkah 5/6 → SCR-12 | "Buka list" | List otomatis "Upload: <nama file> — <tanggal>" berisi baris Cocok (termasuk yang ditautkan) + prospek privat | FR-SEG-001 | Data upload privat workspace dan tidak pernah mengubah data global (BR-ER-07). |

### FLOW-06 — Ekspor dengan pratinjau biaya & kredit tidak cukup
Implements: FR-EXP-001, FR-EXP-002, FR-EXP-005, FR-EXP-006, FR-BILL-001, FR-BILL-004, FR-BILL-005, BR-CRED-03..05, BR-CRED-07. Wireframe: WF-06.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-04/12 | Pilih prospek → "Ekspor" | SCR-14 mode ekspor, langkah "Atur" | FR-SRCH-007 | — |
| 2 | SCR-14 | Pilih format (CSV/XLSX/JSON) dan field | Kolom `claim_type`, `confidence`, `verified_at` terkunci ikut | FR-EXP-001 | Field yang dibatasi lisensi sumber → nonaktif + ikon gembok (BR-PRIV-04). |
| 3 | SCR-14 | "Lanjut ke biaya" | Pratinjau: total baris, sudah dibuka (0), dibuka otomatis (1/baris), dikecualikan, total kredit, saldo sebelum → sesudah | FR-EXP-002, BR-CRED-03 | Melebihi batas baris per ekspor plan → kelebihan masuk "dikecualikan" + penjelasan (FR-EXP-006). |
| 4a | SCR-14 | Saldo cukup → "Ekspor · 35 kredit" | Job dibuat (idempotency key per dialog); 35 kredit **ditahan** (saldo tersedia 184 → 149, ledger "Ditahan") → toast → SCR-15 | FR-BILL-001, BR-CRED-04, BR-CRED-07 | Klik ganda → satu job, satu penahanan. |
| 4b | SCR-14 | Saldo kurang | Tombol utama berubah; opsi: "Ekspor hanya yang sudah dibuka", "Batasi ke N prioritas tertinggi sesuai saldo", "Minta tambahan kredit" | FR-BILL-004 | Owner: "Hubungi tim MesinProspek" (invoice manual MVP 1). Role lain: "Salin pesan untuk Pemilik". |
| 5 | SCR-15 | Pantau progres → "Unduh" | Sukses: kredit ditagih hanya untuk record yang benar-benar diekspor, kontak record itu dibuka permanen, sisa penahanan dilepas. Tautan berlaku 24 jam | FR-EXP-001, NFR-PERF-04, BR-CRED-07 | Gagal → "Gagal" + alasan + "35 kredit yang ditahan dilepas" (tidak ada tagihan, tidak ada kontak yang dibuka) + "Coba lagi". Tautan kedaluwarsa → "Buat ulang" (0 kredit, sudah dibuka). |

### FLOW-07 — Atur & uji webhook
Persona PER-04 (Owner/Admin). Implements: FR-EXP-003, FR-EXP-004, FR-EXP-005, NFR-SEC-05, FR-BILL-003.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-20 | "Tambah endpoint" | Form: nama, URL, langganan event | FR-EXP-003 | Plan tanpa webhook (Starter) → UPSELL "Webhook tersedia di paket Growth ke atas". Kuota endpoint penuh → pesan berkode. |
| 2 | SCR-20 | Isi URL | Validasi klien (HTTPS) lalu server (SSRF) | NFR-SEC-05 | URL ke IP privat/metadata → "URL ini mengarah ke jaringan privat atau internal dan tidak diizinkan." |
| 3 | SCR-20 | Simpan | Secret dibuat dan **ditampilkan sekali** + Salin + panduan verifikasi HMAC-SHA256 | FR-EXP-003 | Dialog secret baru bisa ditutup setelah centang "Saya sudah menyimpan secret ini". |
| 4 | SCR-20 | "Kirim uji" | Payload uji bertanda tangan (event `webhook.test`) → hasil: kode HTTP, latensi, cuplikan respons (≤ 1 KB) | FR-EXP-003 | Timeout (10 detik) / 4xx / 5xx / TLS gagal → pesan spesifik + saran (dari `test.error_code`: `TIMEOUT`, `HTTP_STATUS`, `TLS_ERROR`, `CONNECTION_ERROR`, `URL_REJECTED`). Endpoint tetap `ACTIVE`; badge uji berubah dari "Belum diuji" (`test.status = UNTESTED`) menjadi "Uji gagal" (`FAILED`) atau "Lolos uji" (`PASSED`) (`04` §12.4). Uji tidak memblokir pengiriman. |
| 5 | SCR-04/05/12 | "Kirim ke webhook" (pilih endpoint) | SCR-14 → pengiriman diantrekan | FR-EXP-004, BR-CRED-03 | Record ter-suppress dikecualikan (FR-EXP-005). |
| 6 | SCR-20 log | Pantau | Status per pengiriman: Terkirim / Mencoba ulang (ke-n, jadwal berikutnya) / Gagal (DLQ) | FR-EXP-003 | Gagal setelah retry ≤ 24 jam → DLQ → "Kirim ulang" manual (tanpa tagihan baru). Kredit yang ditahan untuk record yang tidak terkirim → dilepas (BR-CRED-07). Balasan `410 Gone` → endpoint langsung dinonaktifkan otomatis. |

### FLOW-08 — Perbarui data prospek (refresh)
Implements: FR-MON-001, FR-DATA-007, FR-INTL-011, BR-CRED-05, BR-CRED-07.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-05 | Melihat badge "Usang"/"Mulai usang"; klik "Perbarui data" | SCR-14: 1 kredit, "Verifikasi ulang kontak dan hitung ulang skor pada versi dataset terbaru" | FR-MON-001 | Diperbarui < 24 jam lalu dan tidak ada snapshot baru → peringatan "Hasil kemungkinan sama" sebelum konfirmasi (OQ-UX-09). |
| 2 | SCR-05 | Konfirmasi | Status "Memperbarui…" di header; bagian terdampak memakai skeleton tipis; aksi lain tetap aktif | — | — |
| 3 | SCR-05 | — | Ringkasan "3 nilai berubah" (lama → baru, disorot selama sesi); skor baru; entri timeline | FR-MON-001 | Tanpa perubahan → "Tidak ada perubahan. Verifikasi dan skor sudah terbaru." |
| 4 | SCR-05 | — | Kredit ditahan saat konfirmasi, ditagih saat refresh selesai. Gagal → status + "1 kredit yang ditahan dilepas" + "Coba lagi" | BR-CRED-07 | ID prospek tetap; tidak ada entity duplikat (FR-DATA-007). |

### FLOW-09 — Laporkan data salah
Implements: FR-DATA-014, FR-OPS-008, FR-OPS-009.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-05/06 | Menu "…" pada claim → "Laporkan data", atau dari bar aksi | SCR-23 dengan claim terisi | FR-DATA-014 | Dari bar aksi (tanpa claim) → user memilih field. |
| 2 | SCR-23 | Pilih jenis masalah; opsional nilai yang benar + catatan | Validasi | — | "Ini data pribadi saya" → diarahkan ke SCR-25 (FR-AUD-004). |
| 3 | SCR-23 | Kirim | Toast "Laporan terkirim"; entri timeline; chip "Dilaporkan · sedang ditinjau" pada claim | FR-DATA-014 | Laporan sama dari user yang sama masih terbuka → tampilkan statusnya, bukan form baru. |
| 4 | SCR-36 | (Data Steward / Ops Admin) triase; bila laporan benar → "Koreksi nilai" | Perubahan status terlihat di chip claim. Nilai koreksi tampil dengan sumber "Koreksi tim MesinProspek" di evidence drawer. Skor terkait dihitung ulang. | FR-OPS-008, FR-OPS-009 | Koreksi ditolak bila nilai baru gagal validasi format (BR-NORM) |

### FLOW-10 — Undang anggota workspace
Implements: FR-AUTH-002..005, FR-BILL-003, FR-AUD-001.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-19 | "Undang anggota": satu atau beberapa email + role (dengan deskripsi hak tiap role) | Cek kuota seat | FR-AUTH-003, FR-BILL-003 | Seat penuh → diblokir: "Seat paket Growth: 5 dari 5 terpakai." |
| 2 | SCR-19 | Kirim | Undangan tertunda, berlaku 7 hari; aksi kirim ulang / batalkan | FR-AUTH-003 | Email sudah menjadi anggota → pesan inline. |
| 3 | Email → SCR-01 | Buka tautan; masuk atau buat akun | Bergabung dengan role; workspace muncul di switcher | FR-AUTH-002/003 | Kedaluwarsa → "Minta admin mengirim ulang undangan." Login dengan email lain → tawarkan ganti akun. |
| 4 | SCR-19 | Ubah role / keluarkan anggota | Role berlaku segera; anggota yang dikeluarkan kehilangan sesi ≤ 1 menit | FR-AUTH-005, FR-AUD-001 | Pemilik terakhir tidak dapat dikeluarkan/diturunkan. Tugas milik anggota yang keluar ditandai "Penerima tidak aktif" (OQ-UX-10). |

### FLOW-11 — Ganti atau ubah ICP → hitung ulang skor
Implements: FR-ICP-002, FR-ICP-006, FR-INTL-009, FR-INTL-011, NFR-PERF-05.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | Header | Pilih ICP lain | Skor dan urutan di semua layar mengikuti ICP baru; `icp=` di URL diperbarui | FR-ICP-006 | ICP belum pernah dihitung → SCORING. |
| 2 | SCR-17 | Ubah kriteria → Simpan | Versi baru (mis. v4) + job hitung ulang (≤ 10 menit) | FR-ICP-002, FR-INTL-009 | Kriteria kontradiktif (wilayah wajib sekaligus dikecualikan) → error inline. |
| 3 | Layar berskor | Tetap bekerja | Skor versi lama tetap tampil dengan label "Menunggu hitung ulang" + banner progres | FR-INTL-011 | Urutan **tidak** berubah sendiri di bawah kursor user. |
| 4 | Layar berskor | Toast "Skor ICP v4 siap" → "Muat skor baru" | Hasil dimuat ulang | — | Job gagal → banner ERROR + "Hitung ulang"; skor lama tetap berlabel. |

### FLOW-20 (ops) — Rilis & tinjau dataset snapshot
Persona PER-05 (Data Steward), Maintainer (`06`). Implements: FR-DATA-001, FR-DATA-008, FR-OPS-001, FR-OPS-002, BR-PRIV-04, PRD §8.1.2. Kanonik: pipeline rilis data `06` §10.2 dan siklus hidup snapshot `04` §4.2. **Konsol ops tidak menerbitkan snapshot**; publish dan re-publish hanya lewat job manual di pipeline tag terproteksi `ds-*` yang dijalankan Maintainer setelah gate QG-34.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | Di luar konsol (`06` §10.2) | Ops mengunggah file mentah + manifest (sumber, tanggal pengambilan, periode referensi, tag lisensi, checksum) → MR manifest → Maintainer membuat tag `ds-YYYY.MM.n` | Pipeline tag: register (STAGED) → validasi (VALIDATING). SCR-34 menampilkan versi baru beserta status dan tautan ke pipeline tag | FR-DATA-001, BR-PRIV-04 | Checksum tidak cocok → pipeline berhenti; SCR-34 menampilkan status gagal + tautan log pipeline. Tag lisensi dan periode referensi wajib di manifest. Versi immutable: koreksi = versi baru. |
| 2 | SCR-34 detail | Data Steward meninjau laporan validasi (VALIDATED) | Jumlah baris, valid, reject per alasan, fill-rate per field vs snapshot aktif (delta), hasil gate QG-34 per cek, ringkasan rehearsal staging | FR-DATA-001, PRD §8.1.2 | Baris reject tidak memblokir baris valid. Gate gagal → status "Gate gagal" + cek yang gagal. |
| 3 | SCR-34 → issue rilis data | Data Steward menulis rekomendasi **publish** atau **tolak** (+ alasan/catatan QG-34) di issue rilis data yang ditautkan panel "Cara menerbitkan" (`06` §10.2) | Konsol hanya membaca: panel menampilkan tautan pipeline tag + instruksi job manual `publish` (`msp tool snapshot-publish`) / `reject` (`msp tool snapshot-reject`) untuk Maintainer | FR-DATA-001, PRD §8.1.2 | Tidak ada tombol "Terbitkan" atau "Tolak" di konsol (`04` §4.2, §11.5). QG-34 gagal tanpa waiver → REJECTED otomatis. Kill switch `pipeline.publish` off → banner "Publish ditahan oleh kill switch". |
| 4 | Pipeline tag (`06`) → SCR-34 | Maintainer menjalankan job manual `publish` (atau `reject` sesuai rekomendasi) | PUBLISHED; snapshot aktif sebelumnya → SUPERSEDED; skor stale dihitung ulang (atau REJECTED). Riwayat SCR-34 mencatat transisi, aktor, waktu, pipeline | `04` §4.2 | — |
| 5 | SCR-31 → SCR-30 | Pantau | Progres per stage; Data Health per snapshot | FR-OPS-001/002 | Kegagalan > 5% → alert (RG-5). |
| 6 | SCR-34 riwayat | Rollback: Maintainer menjalankan job manual `republish` (alasan wajib) di pipeline tag `ds-*` snapshot SUPERSEDED yang dituju | Snapshot itu kembali PUBLISHED dan riwayat menandainya "Diterbitkan ulang"; snapshot yang tadinya aktif → SUPERSEDED; recompute + rebuild projection | `04` §4.2, `06` §10.2 | Partisi yang sudah diarsip dipasang ulang lebih dulu; status "Menyiapkan re-publish" tampil sampai selesai. |

Referensi wilayah/populasi (FR-DATA-008) di-import lewat SCR-34 tab "Referensi" (`POST /v1/ops/reference/*`, `04` §4.7) dan mengikuti validasi yang sama; aktivasi versi model skor/geo/prompt (`priority-*`, `geo-*`, `brief-*`/`angle-*`) juga hanya lewat pipeline tag (`06` §10.3–10.4) dan ditampilkan read-only di SCR-32.

### FLOW-21 (ops) — Merge review
Persona: Data Steward atau Ops Admin (PRD §8.1.2). Implements: FR-DATA-004, FR-OPS-005, BR-ER-04..06.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-33 | Buka antrean | Kandidat skor fuzzy 0,80–0,92 bernilai tinggi, urut nilai (cocok ICP aktif / ada aktivitas tenant) lalu skor | BR-ER-04 | — |
| 2 | SCR-33 detail | Bandingkan berdampingan | Field per kandidat; identifier cocok/bertentangan disorot; jarak (m); peta mini | FR-OPS-005 | Identifier kuat bertentangan → "Gabung" nonaktif + alasan (BR-ER-05). |
| 3 | SCR-33 | Putuskan: Gabung (establishment sama) / Company sama, cabang berbeda / Pisahkan / Lewati | Keputusan persisten lintas re-run; audit; kandidat berikutnya terbuka | BR-ER-06, FR-AUD-001 | Undo 10 detik via toast; setelahnya diubah lewat riwayat entity. |

### FLOW-22 (ops) — Suppression & permintaan subjek data
Persona: Privacy & Billing atau Ops Admin (PRD §8.1.2). Implements: FR-AUD-003, FR-AUD-004, FR-OPS-008, BR-PRIV-02, NFR-PRIV-02.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-25 / email | Subjek data mengisi formulir (atau ops membuat tiket dari email) | Tiket + nomor referensi + email konfirmasi penerimaan (≤ 1 hari kerja) | FR-AUD-004 | — |
| 2 | SCR-35 | Verifikasi identitas: **OTP email** (SMTP) ke email pemohon. Pemohon yang hanya punya kontak telepon → ops melakukan **panggilan balik manual** ke nomor yang tercatat lalu menandai hasilnya (catatan wajib) | Status "Identitas terverifikasi"; timer SLA berjalan sejak diterima | FR-AUD-004 | Tanpa OTP SMS/WhatsApp di MVP. Panggilan balik gagal → "Menunggu pemohon". Nilai SLA = TBD konsultan hukum (BR-PRIV-03, OQ-UX-13). |
| 3 | SCR-35 | Cari entity/contact point terkait → pilih tindakan: Suppress, Koreksi, Laporan akses, Keberatan | Suppression **global layer** dibuat dan berlaku di semua workspace | FR-AUD-003, BR-PRIV-02 | Kecocokan ambigu → minta info tambahan (status "Menunggu pemohon"). |
| 3b | SCR-35 | Bila subjek juga ditemukan di **data upload tenant** → "Teruskan ke tenant" | Status "Diteruskan ke tenant"; Owner/Admin workspace tersebut diberi notifikasi; data upload **tidak** disembunyikan otomatis (tenant = pengendali) dan dieksekusi atas instruksi tenant. Nilai yang sama di global layer tetap di-suppress | BR-PRIV-02, FR-AUD-004 | Panel kasus menampilkan workspace tujuan tanpa nilai kontak. |
| 4 | SCR-35 | Pantau propagasi | Checklist projection: pencarian, profil, ekspor (termasuk file aktif yang dibuat ulang), webhook, event `suppression.applied` (target ≤ 24 jam) | NFR-PRIV-02, FR-AUD-003 | Lewat 24 jam → alert. |
| 5 | SCR-35 | Tutup dengan respons (template) | Audit + email ke pemohon | FR-AUD-001 | — |

### FLOW-23 (ops) — Redrive DLQ
Persona: Ops Engineer atau Ops Admin (PRD §8.1.2). Implements: FR-OPS-002, BR-CRED-04, BR-CRED-07, BR-PRIV-01.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-31 | Menerima alert (kegagalan > 5% / backlog > 1 jam) | Tab DLQ dikelompokkan per tipe job × kelas error | FR-OPS-002 | — |
| 2 | SCR-31 | Inspeksi pesan | Payload ter-redaksi PII, error ringkas, `correlation_id` | BR-PRIV-01 | — |
| 3 | SCR-31 | Pilih → "Redrive" (konfirmasi jumlah + alasan) | Diantrekan ulang dengan idempotency key yang sama → tidak ada tagihan ganda | BR-CRED-04 | Gagal lagi → kembali ke DLQ, `attempt` bertambah. |
| 4 | SCR-31 | Atau "Buang" (alasan wajib) | Audit | FR-AUD-001 | Job berbayar yang dibuang → kredit yang ditahan dilepas otomatis (BR-CRED-07). |

### FLOW-24 (ops) — Grant kredit & plan (invoice manual)
Persona: Privacy & Billing atau Ops Admin (PRD §8.1.2). Implements: FR-BILL-006, FR-OPS-006, BR-CRED-06.

| # | Layar | Aksi user | Respons sistem | FR/BR | Gagal / edge |
|---|---|---|---|---|---|
| 1 | SCR-37 | Buka tenant → "Beri kredit" | Form: meter (kredit / kredit AI), jumlah, jenis (bulanan / top-up → masa berlaku), ref invoice, alasan | FR-BILL-006, BR-CRED-06 | Jumlah di atas ambang wajar → konfirmasi ganda. |
| 2 | SCR-37 | Konfirmasi | Entri ledger `grant` + audit; saldo tenant berubah | FR-BILL-002, FR-AUD-001 | — |

---

## 4. Spesifikasi layar

### 4.0 State baku (berlaku untuk semua layar)

Setiap layar di §4.1–4.3 merujuk kode state ini dan hanya menuliskan perilaku khususnya.

| Kode | Kapan | Perilaku UI | Contoh copy | Aturan |
|---|---|---|---|---|
| LOADING | Data pertama dimuat | Skeleton berbentuk konten (bukan spinner halaman). > 10 detik → teks "Masih memuat…". | — | NFR-PERF-03 |
| EMPTY | Objek belum pernah dibuat | 1 kalimat manfaat + 1 CTA utama | §7.5 | UX-P2 |
| ZERO | Query/filter tanpa hasil | Saran pelonggaran spesifik dengan hitungan alternatif | "Hapus filter Rating ≥ 4,5 → 37 hasil" | FR-SRCH-008 |
| UNKNOWN | Nilai tidak diketahui | "Tidak diketahui" + ikon + tooltip alasan (P-12) | "Tidak diketahui — data sumber tidak memuat jam operasional" | BR-TRUST-06 |
| LIMITED | Cakupan skor < 0,60 atau cakupan area LOW | Badge "Data terbatas" + tooltip (P-06) | — | BR-SCORE-04, FR-GEO-009 |
| STALE | Data Aging/Stale atau skor stale | Badge kemutakhiran (P-03) / "Menunggu hitung ulang"; CTA "Perbarui data" bila relevan | — | BR-TRUST-04, FR-INTL-011 |
| SCORING | ICP baru/berubah sedang dihitung | Banner progres "Menghitung skor ICP v4 · 62%"; hasil parsial ditandai | — | NFR-PERF-05 |
| ERROR | Request gagal | Error per bagian (bukan seluruh halaman) + "Coba lagi" + kode referensi | Pola §7.4 | PRD §12.2 |
| DENIED | Role tidak diizinkan (403) | Apa yang tidak dapat diakses + siapa yang dapat memberi akses | "Halaman ini khusus Pemilik dan Admin. Pemilik workspace: Rina A." | FR-AUTH-004 |
| NO-CREDIT | Saldo < biaya | Tombol berbayar tetap terlihat dengan biayanya; klik → dialog dengan opsi (WF-06b) | §7.6 | FR-BILL-004 |
| FAIR-USE | Batas lihat profil bulanan | ≥ 80%: banner info. 100%: profil baru diblokir dengan pesan; profil yang kontaknya sudah dibuka atau ditugaskan tetap dapat dilihat (OQ-UX-08) | "Batas 1.000 profil bulan ini tercapai. Kuota pulih 1 Nov 2026." | BR-CRED-01, FR-BILL-003 |
| FLAG-OFF | Kill switch / flag mati / batas biaya AI (flag `04` §18.3: `ai.generation`, `export.create`, `webhook.delivery`, `upload.create`, `unlock.enabled`, dst.) | Kontrol **nonaktif** (bukan disembunyikan) + tooltip + banner. API membalas `503 FEATURE_DISABLED` | "Ekspor sedang dinonaktifkan sementara oleh tim MesinProspek. Data Anda aman." · `unlock.enabled` off: tombol "Buka kontak" nonaktif + "Buka kontak sedang dinonaktifkan sementara. Kontak yang sudah dibuka tetap dapat dilihat; ekspor dan kirim webhook mengikuti statusnya masing-masing." | FR-OPS-007, FR-AI-006 |
| READ-ONLY | Flag `maintenance.read_only` on (PITR, switchover, DR; `04` §18.3) | Banner global (prioritas tertinggi, tidak dapat ditutup); **semua aksi tulis nonaktif** (buka kontak, simpan/tag/catatan/status, tugaskan, ekspor, webhook, upload, AI brief, perbarui data, simpan ICP/pengaturan); baca (cari, profil, peta, bukti) tetap jalan. Aksi tulis yang sudah terkirim dan ditolak server dengan `503 MAINTENANCE_READ_ONLY` → toast error + perubahan optimistik dibatalkan | "MesinProspek sedang dalam pemeliharaan. Anda tetap dapat mencari dan membaca data; menyimpan dan aksi berbayar dinonaktifkan sementara." | FR-OPS-007, `04` §11.2, §18.3 |
| UPSELL | Fitur di luar entitlement plan | Kontrol nonaktif + "Tersedia di paket X" + "Hubungi kami" | — | FR-BILL-003 |
| OFFLINE | Koneksi terputus | Banner; aksi tulis diantrekan atau ditolak dengan pesan | "Koneksi terputus. Perubahan yang belum tersimpan akan dicoba lagi." | — |

**Responsive baku.** Breakpoint `breakpoint.{sm,md,lg,xl,2xl}`. Desain utama untuk `xl` (≥ 1280 px). Wajib dapat dipakai penuh di 360 px (NFR-COMP-01): SCR-01, SCR-03, SCR-05 (beserta SCR-06, SCR-07, SCR-14, SCR-23), SCR-11, SCR-12, SCR-13, SCR-24, SCR-25. Layar lain di < `breakpoint.md` menampilkan versi baca yang disederhanakan + saran membuka di desktop; SCR-08 memakai `view=table`. Ops console hanya desktop (≥ `breakpoint.lg`).

**Format spesifikasi.** Tiap layar: tabel meta (pertanyaan, persona, route, implements), zona layout, data/komponen, interaksi, state khusus, responsive, event analytics (detail properti di §9).

### 4.1 Aplikasi customer

#### SCR-01 — Autentikasi (Masuk, Buat akun, Verifikasi, Atur ulang password, Terima undangan)

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Bagaimana saya masuk ke workspace saya?" | Semua | `/auth/*`, `/invite/[token]` | FR-AUTH-001, FR-AUTH-002, FR-AUTH-003, FR-AUTH-006 |

- **Layout:** kartu tunggal terpusat (maks 400 px) di atas `color.bg.canvas`; logo; pemilih bahasa di pojok.
- **Komponen:** tombol "Lanjut dengan Google"; email + password (tampilkan/sembunyikan, mendukung password manager dan tempel); "Buat akun" meminta nama, email, password, "Nama usaha Anda"; layar verifikasi (kirim ulang setelah 60 detik, tempel kode); MFA (TOTP) bila diaktifkan user, wajib untuk akun ops.
- **Terima undangan:** menampilkan nama workspace, pengundang, dan role sebelum user menerima. Bila user punya > 1 workspace dan tidak ada yang terakhir dipakai → pemilih workspace.
- **State khusus:** email belum terverifikasi → hanya layar verifikasi yang dapat diakses; undangan kedaluwarsa/dibatalkan; percobaan login berlebihan → jeda + pesan tanpa membocorkan apakah email terdaftar.
- **Responsive:** sama di semua lebar. **Analytics:** `signup_completed`, `email_verified`, `login_succeeded`, `invite_accepted`.

#### SCR-02 — Onboarding wizard (3 langkah)

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Apa yang saya jual dan ke siapa?" | PER-01, PER-02, PER-04 | `/app/onboarding` | FR-ICP-001, FR-ICP-002, FR-ICP-004, FR-ICP-005, FR-ICP-006 |

| Langkah | Isi | Aturan |
|---|---|---|
| 1. Penawaran | Grid chip tag katalog dengan label id-ID + deskripsi singkat: Pembuatan website (`website_dev`), SEO lokal (`local_seo`), Iklan digital (`digital_ads`), Manajemen reputasi (`reputation_mgmt`), Sistem booking (`booking_system`), CRM (`crm`), POS / kasir (`pos`), Layanan email (`email_service`), Lainnya (`other`). Boleh > 1. Nama penawaran terisi otomatis. Info preset bobot: "Skor akan menekankan peluang yang relevan dengan penawaran ini." | FR-ICP-001/004. Tanpa penawaran, peluang tidak ditampilkan. |
| 2. ICP | Wajib: kategori (combobox taksonomi 2 level, saran berdasarkan tag penawaran) dan wilayah (combobox hierarki provinsi → kelurahan). "Kriteria lanjutan" (tertutup): band karyawan, band revenue, kondisi digital, kontak wajib. Panel pratinjau kanan: jumlah kandidat, 3 contoh nama, distribusi kategori. | FR-ICP-002. Default semua kriteria = "Preferensi" kecuali kategori & wilayah = "Wajib". |
| 3. Hasil pertama | 10 hasil teratas (komponen baris SCR-04) + coach mark berurutan: (1) skor & band, (2) "Kenapa prospek ini?", (3) kontak tersembunyi & biaya buka. CTA "Mulai mencari". | FR-ICP-005, NFR-PERF-05 |

- **Interaksi:** indikator langkah 1-2-3; "Kembali" mempertahankan isian; "Keluar" menyimpan draf. Tidak ada tombol lewati (ICP dibutuhkan untuk skor).
- **State khusus:** SCORING bertahap (FLOW-01 #6); kandidat 0; kandidat > 200.000; saldo kredit 0 pada workspace baru → coach mark (3) menyebut "Kredit belum tersedia — hubungi tim kami untuk aktivasi" (OQ-UX-25).
- **Responsive:** langkah 1–2 satu kolom di < `breakpoint.lg`; panel pratinjau pindah ke bawah. **Analytics:** `onboarding_step_completed`, `offering_created`, `icp_saved`, `first_result_viewed`.

#### SCR-03 — Beranda

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Apa yang perlu saya kerjakan hari ini?" | PER-02, PER-03, PER-01 | `/app` | FR-SEG-005, FR-SRCH-005, FR-SEG-002, FR-BILL-002, FR-EXP-001 |

| Kartu (urutan) | Isi | Aksi lanjut | Role |
|---|---|---|---|
| Checklist aktivasi (workspace baru) | Buat penawaran ✓, Buat ICP ✓, Simpan list/segmen pertama, Undang rekan, Atur webhook. Hilang setelah lengkap atau ditutup. | Tiap item → layar terkait | Owner/Admin untuk undang & webhook |
| Tugas saya | Hitungan per status kerja + 5 tugas prioritas tertinggi (nama, skor, peluang teratas) | → SCR-13 / SCR-05 | Member ke atas |
| Pencarian tersimpan | 5 terakhir: nama, ringkasan filter, ICP, jumlah hasil saat ini | "Jalankan" → SCR-04 | Member ke atas |
| Pembaruan segmen | Segmen dengan perubahan jumlah sejak kunjungan terakhir: "+12 prospek sejak 3 Okt" | → SCR-12 | Member ke atas |
| Kredit | Saldo tersedia kredit & kredit AI (+ jumlah ditahan untuk proses berjalan, bila ada), tanggal kedaluwarsa grant, pemakaian siklus ini (bar + angka) | "Lihat pemakaian" (Owner/Admin/Manager) | Member ke atas |
| Ekspor & proses terbaru | 3 job terakhir + status | "Unduh" / "Coba lagi" → SCR-15 | Member ke atas |

- **State khusus:** EMPTY per kartu (§7.5); SCORING tampil sebagai banner global; Viewer hanya melihat sapaan + konteks ICP + CTA "Cari prospek".
- **Responsive:** grid 3 kolom (`xl`) → 2 (`lg`) → 1 (< `md`, urutan sesuai tabel). **Analytics:** `home_viewed`, `saved_search_run`, `segment_opened`.

#### SCR-04 — Cari & hasil

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Mana yang paling menarik untuk saya kerjakan?" | PER-02 (utama), PER-01, PER-03 | `/app/search` | FR-SRCH-001..008, FR-DATA-012, FR-SEG-001, FR-SEG-003, FR-SEG-005, FR-EXP-001, FR-EXP-004, BR-SCORE-01, BR-SCORE-04, BR-SCORE-07, BR-SCORE-12 |

Wireframe: **WF-01**.

**Zona layout**

| Zona | Isi |
|---|---|
| A. Bar pencarian | Input kata kunci (nama, alias, kategori, alamat; toleran PT/CV, kapitalisasi, tanda baca), tombol "Simpan pencarian" (nama + query + filter + ICP). |
| B. Chip filter aktif | Satu chip per filter, dapat dihapus satu per satu, "Reset". Chip sumber "Termasuk upload Anda" tampil **aktif secara default** bila workspace punya prospek privat; menghapusnya mengecualikan prospek privat dari hasil (FR-DATA-012; parameter API `source=ALL\|GLOBAL\|PRIVATE`, default `ALL`; chip dihapus → `GLOBAL`, `04` §10.9). "Reset" mengembalikannya ke default (aktif). |
| C. Panel filter (kiri, 280 px, dapat diciutkan) | Grup sesuai FR-SRCH-002: Kategori, Wilayah (hierarki), Band prioritas (HOT/WARM/COLD/Belum dinilai), Kecocokan ICP (rentang), Ketersediaan kontak (telepon, email, website, kemungkinan WA), Status verifikasi, Tipe peluang, Kemutakhiran, Status bisnis (default tanpa "Tutup permanen", BR-SCORE-07), Rating (rentang), Band karyawan, Band revenue, Status buka kontak, "Belum ada di list saya". Ditambah grup **Sumber data**: "Termasuk upload Anda" (default aktif, FR-DATA-012; bukan filter FR-SRCH-002). Tidak ada filter lain di MVP (usulan V1 di OQ-UX-22). Hitungan facet di samping opsi; "±" bila perkiraan. |
| D. Header hasil | "182 prospek cocok" (total hitungan FR-SRCH-006: eksak ≤ 10.000, "±12.400" di atasnya), sortir (Prioritas ↓ default, Kecocokan ICP, Kualitas data, Nama), kepadatan tabel (`density.compact` 36 px / `density.comfortable` 44 px). |
| E. Tabel hasil | Baris per prospek (lihat tabel kolom). |
| F. Bar aksi massal | Sticky bawah saat ≥ 1 baris dipilih: jumlah terpilih, "Pilih semua N hasil" (hingga batas plan, batas ditampilkan), Simpan ke list, Tag, Tugaskan, Buka kontak, Ekspor, Kirim ke webhook. |
| G. Muat lebih banyak | Pagination cursor keyset (`04` §10.5), **tanpa nomor halaman**: 25 baris per muat (50/100 opsional, maks 100). Di bawah tabel: "Menampilkan 50 dari 182" + tombol "Muat 25 lagi" (hilang bila tidak ada `next_cursor`). |

**Kolom baris hasil (FR-SRCH-004)**

| Kolom | Isi | Aturan / pola |
|---|---|---|
| Pilih | Checkbox, label SR "Pilih <nama bisnis>" | — |
| Skor | Angka + band; badge "Data terbatas"; chip "Kriteria wajib belum terkonfirmasi" (band maks WARM); "Menunggu hitung ulang" bila stale | P-05, P-06, BR-SCORE-04/05/12, FR-INTL-011 |
| Bisnis & lokasi | Nama Company; kategori; kelurahan, kecamatan establishment terbaik; "+N cabang"; kemutakhiran record; label "Dari upload Anda" untuk prospek privat | BR-SCORE-01, P-03, FR-DATA-012 |
| Peluang teratas | Label peluang dengan opportunity_score tertinggi untuk penawaran aktif + keyakinan. Tanpa peluang relevan → "Belum ada peluang relevan" | P-11 ringkas |
| Kontak | Ikon ketersediaan T/E/W/Web dengan status verifikasi terbaik per kanal; nilai tidak ditampilkan; ikon gembok terbuka bila sudah dibuka (prospek privat: selalu terbuka, tanpa biaya) | P-04, P-08, BR-CRED-02 |
| Status kerja / penerima | (opsional, tampil bila ada) chip status + avatar penerima | FR-SEG-005/006 |

**Interaksi**
- Klik baris / Enter → SCR-05 (`icp` ikut). Ctrl/Cmd+klik → tab baru. Hover baris → aksi cepat "Simpan ke list" dan "Buka profil".
- Filter berubah → URL diperbarui, hasil di-debounce 300 ms, fokus tetap di kontrol filter; pembaca layar mendapat pengumuman "182 prospek cocok" (aria-live polite).
- Checkbox header memilih semua baris yang sudah dimuat → tautan "Pilih semua 182 hasil" muncul di bar aksi. Seleksi (berbasis ID) bertahan saat memuat lebih banyak.
- "Muat 25 lagi" menambahkan baris di bawah tanpa memindahkan fokus; pembaca layar mendapat "25 prospek lagi dimuat, 50 dari 182". Kembali dari profil memulihkan baris yang sudah dimuat dan posisi scroll dari state riwayat (bukan URL).
- Baris UNSCORED diurutkan paling bawah saat sortir prioritas. Prospek privat diurutkan bersama hasil global dengan kunci sortir yang sama.

**State khusus**

| State | Perilaku |
|---|---|
| ZERO | Panel saran: (1) filter paling membatasi + hitungan bila dihapus, (2) naikkan wilayah satu level, (3) bila penyebabnya kriteria **wajib** ICP: "Kriteria wajib 'Email tersedia' mengecualikan 1.240 bisnis" + "Ubah ICP" (Manager ke atas) (FR-SRCH-008). |
| SCORING | Banner + baris yang belum dihitung untuk versi ICP aktif berlabel "Menunggu hitung ulang"; toast "Muat skor baru" saat selesai (FLOW-11). |
| LIMITED | Baris dengan cakupan < 0,60: band maksimal WARM, tooltip menjelaskan pembatasan (P-06). |
| Kriteria wajib belum terkonfirmasi | Kriteria ICP *wajib* bernilai Tidak diketahui (termasuk kategori "Lainnya" dengan keyakinan < 50): baris tetap tampil, kriteria tidak dihitung di Kecocokan ICP, chip "Kriteria wajib belum terkonfirmasi" + tooltip menyebut kriterianya ("Wilayah belum diketahui"), band maks WARM (BR-SCORE-12, `04` §7.4). |
| Prospek privat (upload) | Label "Dari upload Anda" (+ chip "Mungkin cocok · tinjau" → SCR-16 `?tab=possible` bila baris belum dikonfirmasi); kontak tidak di-mask dan tidak dikenai biaya; skor untuk ICP aktif dengan aturan yang sama. Flag `search.include_private` off → hasil hanya data global dan chip sumber tidak dirender (`04` §10.9, §18.3). |
| Hasil diperbarui (cursor kedaluwarsa) | "Muat lagi" mendapat `409 CURSOR_STALE` (cursor terikat epoch skor yang sudah tidak aktif > 1 jam setelah hitung ulang, `04` §10.5) → banner info "Urutan hasil sudah diperbarui. [Muat ulang dari atas]"; baris yang sudah tampil tetap terlihat sampai user memilih; seleksi berbasis ID dipertahankan. |
| Hasil > batas bulk plan | "Pilih semua" dibatasi; teks "Batas paket Anda: 10.000 baris per aksi". |
| Pencarian timeout | ERROR khusus: "Pencarian terlalu lama. Persempit wilayah atau kurangi kata kunci." |

- **Responsive:** < `breakpoint.lg` panel filter menjadi sheet "Filter (5)"; < `breakpoint.md` baris menjadi kartu (skor, nama, lokasi, peluang teratas, ikon kontak); bulk action via menu "Aksi". 
- **Analytics:** `search_performed`, `search_results_rendered`, `zero_result_shown`, `zero_result_suggestion_applied`, `filter_applied`, `saved_search_created`, `prospects_bulk_selected`, `prospect_profile_opened`.

#### SCR-05 — Profil prospek

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Apa yang perlu saya tahu, dan apa langkah berikutnya?" | PER-01, PER-02 | `/app/prospects/[companyId]` | FR-PROF-001..010, FR-ICP-003, FR-INTL-007, FR-INTL-010, FR-INTL-011, FR-SEG-001..006, FR-MON-001, FR-DATA-014, BR-CRED-02, BR-TRUST-01..06, BR-REV-01..05 |

Wireframe: **WF-02** (desktop), **WF-03** (360 px). Hierarki informasi mengikuti MVP Pack §07: identitas → kenapa penting → kontak → (sinyal, V1) → bukti → aksi.

**Zona layout**

| Zona | Isi | FR |
|---|---|---|
| A. Navigasi | "← Kembali ke hasil", "3 dari 182", sebelumnya/berikutnya; konteks "Skor untuk ICP: <nama> v<n>" | FR-ICP-006 |
| B. Header identitas | Nama Company (`text.h1`); kategori (P-01 Turunan); status bisnis (P-01); alamat establishment terpilih; pemilih cabang "+N cabang"; status kerja (dropdown); penerima tugas; tag; label "Dari upload Anda" + nama file bila prospek privat | FR-PROF-001, FR-PROF-005, FR-SEG-003/005/006, FR-DATA-012 |
| C. Ringkasan skor | Badge skor besar (P-05) + cakupan (P-06); chip "Kriteria wajib belum terkonfirmasi" bila berlaku (klik → E1 kriteria terkait); Kecocokan ICP n/100; peluang teratas; kemutakhiran | FR-PROF-001, BR-SCORE-12 |
| D. Bar aksi (sticky di bawah header, `z.sticky`) | Tombol utama adaptif: belum dibuka → "Buka kontak · 1 kredit"; sudah dibuka → "Hubungi" (menu kanal terbaik). Lalu: Simpan ke list, Tugaskan, AI brief, Ekspor, Kirim ke webhook, Lainnya (Tag, Catatan, Perbarui data · 1 kredit, Laporkan data). Aksi yang tidak diizinkan role tidak dirender. | FR-PROF-009 |
| E1. Kenapa prospek ini? | Pola P-07: 8 faktor, bobot efektif, kontribusi, jenis data, bukti; sub-breakdown Kecocokan ICP per kriteria (Cocok/Sebagian/Tidak cocok/Tidak diketahui), kriteria *wajib* yang Tidak diketahui ditandai "Wajib · belum terkonfirmasi" dan catatan "band dibatasi maksimal WARM"; versi model + preset + `calculated_at`; feedback skor | FR-PROF-002, FR-ICP-003, FR-INTL-010, BR-SCORE-12 |
| E2. Peluang | Kartu P-11, maksimal 3 terlihat + "Lihat semua (n)"; urut opportunity_score | FR-INTL-007, FR-INTL-010 |
| E3. Kontak | Tabel kontak level Bisnis / Cabang / Orang (lihat di bawah) | FR-PROF-003 |
| E4. Cabang | Tabel establishment: nama/alamat, kelurahan, skor + band (atau alasan tidak lolos ICP), kemutakhiran; klik → ganti establishment terpilih | FR-PROF-005 |
| E5. Aktivitas | Timeline privat workspace: buka kontak, simpan, tugaskan, status, catatan, ekspor, webhook, brief, refresh, laporan data; filter jenis | FR-PROF-010 |
| F1. AI brief | SCR-07 | FR-AI-001/002 |
| F2. Konteks lokasi | Kelurahan, kecamatan, kab/kota; populasi + periode + sumber; kepadatan penduduk; jumlah & kepadatan kategori per 1.000 penduduk; kompetisi (Rendah/Sedang/Tinggi + persentil); skor peluang pasar; cakupan area; `geo_confidence` ("Lokasi dari koordinat" / "Lokasi dari alamat"); guardrail (P-10); "Lihat area" → SCR-10 | FR-PROF-006, FR-GEO-009 |
| F3. Komersial | Band karyawan (BR-REV-01); band revenue "Rp5–15 miliar" + label Perkiraan + keyakinan + basis + per tanggal (P-09). Tidak pernah point estimate. | FR-PROF-007, BR-REV-02..05 |
| F4. Kehadiran digital | Tipe website (Domain sendiri / Subdomain platform gratis / Profil media sosial / Link aggregator / Link pesan / Tidak ditemukan pada data); domain; tipe email (Domain sendiri / Webmail gratis); penyedia email (P1, Turunan dari MX); rating & jumlah ulasan (Teramati + tanggal) | FR-PROF-008, FR-DATA-010/011 |
| F5. Catatan | Daftar catatan + composer | FR-SEG-004 |

**Kontak (E3).** Kolom: Kanal, Nilai (ter-mask sampai dibuka, P-08), Milik (Bisnis / Cabang <nama> / Orang: role), Status verifikasi (P-04), Keyakinan (P-02), Terakhir dicek (P-03), Bukti. Baris kanal terbaik (berdasarkan Contactability) diberi label "Kanal terbaik". Nomor seluler menampilkan baris turunan "Kemungkinan WhatsApp" berlabel Perkiraan (BR-VER-01). Orang: role selalu tampil; nama di-mask sampai dibuka (OQ-UX-14). Contact point global layer yang ter-suppress tidak dirender sama sekali, tanpa placeholder (FR-PROF-003). Kontak yang berasal dari upload workspace sendiri tampil tanpa mask (OQ-UX-11) dan tidak disembunyikan otomatis oleh suppression global (tenant = pengendali, BR-PRIV-02; OQ-UX-28).

**Interaksi**
- Setiap nilai claim: hover/fokus menampilkan ikon bukti; klik → SCR-06 (`?evidence=<claimId>`).
- Ganti cabang → zona B (alamat), C, E3 (kontak cabang), F2 diperbarui; skor mengikuti establishment terpilih (BR-SCORE-01).
- Shortcut (dapat dimatikan di SCR-24, WCAG 2.1.4): `j`/`k` prospek berikutnya/sebelumnya, `s` simpan ke list, `b` AI brief.
- Feedback peluang dan skor tersimpan inline dengan undo 5 detik.

**State khusus**

| State | Perilaku |
|---|---|
| Kontak sudah dibuka | Nilai tampil; aksi Salin/Telepon/WhatsApp/Email; teks "Dibuka oleh Rina A. · 3 Okt 2026" |
| Refund kontak INVALID | Banner info: "Semua kontak yang Anda buka terbukti tidak valid. 1 kredit dikembalikan pada 20 Okt 2026." (BR-CRED-05) |
| UNSCORED | Badge "Belum dinilai"; panel E1 menjelaskan "Kecocokan ICP tidak dapat dihitung karena semua kriteria ICP tidak diketahui untuk bisnis ini"; faktor tetap tampil tanpa total |
| Kriteria wajib belum terkonfirmasi | Chip di zona C + tooltip "Kriteria wajib 'Wilayah' belum diketahui untuk bisnis ini. Kriteria ini tidak dihitung dan band dibatasi maksimal WARM." (BR-SCORE-12) |
| Prospek privat (upload) | Label "Dari upload Anda"; kontak tampil tanpa mask dan tanpa tombol buka; evidence menunjuk "Upload: <file> baris N"; aksi Ekspor/webhook 0 kredit untuk kontak upload; tidak terlihat oleh workspace lain (FR-DATA-012) |
| Buka kontak dinonaktifkan | Flag `unlock.enabled` off → tombol "Buka kontak · 1 kredit" tetap terlihat tetapi nonaktif + tooltip/banner FLAG-OFF (§4.0); kontak yang sudah dibuka tetap tampil |
| READ-ONLY | §4.0: bar aksi menonaktifkan semua aksi tulis; baca profil, bukti, dan brief yang sudah ada tetap jalan |
| Tutup permanen | Banner peringatan "Tutup permanen menurut data sumber (diamati 12 Sep 2026)" + bukti; aksi berbayar tetap tersedia dengan peringatan di dialog |
| Entity digabung (ER) | ID lama → redirect ke ID kanonik + info "Prospek ini telah digabung dengan <nama>" |
| Prospek tidak tersedia | Company ter-suppress / bukan milik workspace → halaman "Prospek ini tidak tersedia" tanpa alasan rinci |
| Refresh berjalan | FLOW-08 #2 |
| Dilaporkan | Chip "Dilaporkan · sedang ditinjau" pada claim terkait |
| FAIR-USE | Lihat §4.0 |

- **Responsive:** `xl` 2 kolom (8/4 grid); `lg` kolom samping turun di bawah E2; < `md` satu kolom accordion (WF-03): Kontak terbuka default, bar aksi menjadi bar bawah sticky (tombol utama + "Aksi lain" sheet), evidence drawer menjadi sheet layar penuh.
- **Analytics:** `prospect_profile_opened`, `score_explanation_viewed`, `evidence_drawer_opened`, `prospect_unlocked`, `contact_action_clicked`, `opportunity_feedback_submitted`, `score_feedback_submitted`, `prospects_added_to_list`, `work_status_changed`, `note_added`, `prospect_refresh_requested`.

#### SCR-06 — Evidence drawer

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Kenapa MesinProspek percaya pada nilai ini?" | PER-01, PER-02 | `?evidence=<claimId>` | FR-PROF-004, FR-DATA-006, BR-TRUST-01..05, BR-VER |

Wireframe: **WF-07**. Drawer kanan 480 px (`z.modal`, `shadow.overlay`), modal (fokus terkunci).

| Bagian | Isi |
|---|---|
| Ringkasan | Nama field + pemilik (Bisnis/Cabang/Orang), nilai (ter-mask bila kontak belum dibuka), jenis data (P-01), keyakinan (P-02), kemutakhiran (P-03), status verifikasi bila kontak (P-04) |
| Dari mana nilai ini? | Daftar evidence dalam bahasa awam, per jenis sumber: **Dataset** — nama tampilan dataset + versi snapshot + ref record + `observed_at` + nilai mentah (FR-DATA-002); **Upload** — nama file + baris + pengunggah + tanggal (hanya terlihat oleh workspace pemilik); **Aturan** — nama + versi aturan + input yang dipakai (tautan ke claim input); **Model/AI** — nama model/prompt + versi + basis; **DNS** — jenis lookup + tanggal + hasil |
| Verifikasi | Metode (email: syntax + MX, dengan cek A/AAAA bila MX tidak ada; telepon: format + tipe nomor), tanggal, hasil + alasan status (P-04) |
| Nilai lain yang ditemukan | Nilai alternatif (BR-TRUST-02) + sumber + alasan tidak dipakai + catatan pengurangan keyakinan 20 poin |
| Bagaimana keyakinan dihitung? | Collapsible: penjelasan rumus dalam bahasa awam |
| Aksi | "Laporkan data ini" → SCR-23; "Tutup" |

State khusus: claim UNKNOWN → bagian "Dari mana" menjelaskan alasan ("Field jam operasional tidak ada di dataset sumber versi ds-2026.09.1"); nama dataset dibatasi lisensi → nama generik "Dataset bisnis MesinProspek" (OQ-UX-19). Analytics: `evidence_drawer_opened`, `data_issue_report_started`.

#### SCR-07 — Panel AI brief

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Apa ringkasan dan sudut pendekatan untuk prospek ini?" | PER-01, PER-02 | Panel di SCR-05 (F1) | FR-AI-001..007, NFR-PERF-06 |

| Bagian brief (usulan, final di prompt `brief-x.y.z`, 04) | Isi |
|---|---|
| Ringkasan | 2–3 kalimat identitas, skala, lokasi |
| Yang perlu diketahui | Poin: kehadiran digital, reputasi, konteks area |
| Peluang relevan untuk penawaran Anda | Ringkasan peluang + rationale |
| Yang belum diketahui | Daftar data penting yang Unknown (UX-P4) |
| Sales angle (P1, terpisah) | Label "Hipotesis", rationale, terkait peluang |

- **Anatomi:** header "Dibuat AI" + model + versi prompt + waktu; setiap kalimat diakhiri chip sitasi `[n]` (focusable) → SCR-06; footer "Brief ini hanya memakai data yang tersedia dan tidak memakai data kontak pribadi." (FR-AI-005).
- **Aksi:** Salin (teks + daftar sumber), Bahasa (ID/EN, default bahasa UI), "Buat ulang · 1 kredit AI", feedback Membantu / Tidak membantu + alasan (FR-AI-007).

| State | Perilaku |
|---|---|
| Belum dibuat | Deskripsi + "Buat brief · 1 kredit AI" |
| Dari cache | Langsung tampil; label "Dari cache · dibuat 3 hari lalu · gratis" (FR-AI-003) |
| Generating | Skeleton paragraf + "Biasanya selesai dalam 20 detik"; dapat ditutup, hasil tetap tersimpan |
| Validasi gagal | "Brief tidak dapat dibuat karena sistem tidak dapat memastikan setiap kalimat didukung data. Kredit AI tidak ditagih." + "Coba lagi" (FR-AI-004; 1 kredit AI ditahan saat diminta lalu dilepas, BR-CRED-07) |
| FLAG-OFF | Kill switch: "Fitur AI sedang dinonaktifkan sementara." Batas biaya workspace: "Batas pemakaian AI workspace bulan ini tercapai." (FR-AI-006) |
| NO-CREDIT | Tombol tetap dengan biaya → dialog saldo |
| Viewer | Baca brief yang ada; tanpa tombol buat (OQ-UX-06) |

Analytics: `ai_brief_requested`, `ai_brief_generated`, `ai_brief_failed`, `ai_brief_copied`, `ai_brief_feedback_submitted`, `sales_angle_generated`.

#### SCR-08 — Peta wilayah (Location Intelligence View)

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Di wilayah mana pasar untuk penawaran saya paling menarik?" | PER-03, PER-01 | `/app/map` | FR-GEO-001, FR-GEO-002, FR-GEO-004, FR-GEO-005, FR-GEO-009, FR-DATA-008 |

Wireframe: **WF-04**.

**Zona layout:** (A) toolbar: pemilih wilayah (breadcrumb provinsi › kab/kota), level (Kecamatan / Kelurahan), metrik, tampilan (Peta / Tabel), konteks ICP; (B) kanvas peta; (C) legenda (kiri bawah, selalu tampil); (D) panel area (kanan 360 px); (E) footnote guardrail + metadata (`geo_version`, snapshot, periode populasi).

**Metrik yang dapat dipilih (Lampiran C)**

| Label UI | Metrik | Skala warna | Catatan |
|---|---|---|---|
| Skor peluang pasar (default) | market_opportunity_score | `color.geo.sequential.1..7` (kuantil dalam wilayah tampil) | Unknown bila cakupan LOW |
| Kompetisi | competition_index | `color.geo.competition.{low,medium,high}` | 3 kelas: Rendah < 33, Sedang 33–66, Tinggi > 66 |
| Kepadatan kategori | category_density (kategori ICP) | sequential | per 1.000 penduduk |
| Bisnis target | target_business_count | sequential | spesifik tenant & ICP |
| Celah pasokan (white space) | white_space_score | sequential | hanya bila populasi ≥ 1.000 dan cakupan OK |
| Populasi / Kepadatan penduduk | population / population_density | sequential | guardrail "konteks, bukan permintaan" |
| Kepadatan bisnis | business_density | sequential | guardrail "proxy supply, bukan market share" |

**Peta.** Vector map generik dengan choropleth (join kode wilayah) dan dukungan clustering (dipakai V1, FR-SRCH-010). **Basemap provider: TBD (Q-05).** Level kelurahan hanya tersedia bila wilayah induk = kab/kota (membatasi jumlah poligon). Area cakupan LOW diarsir (pola diagonal) di atas `color.geo.no-data`; area tanpa data = `color.geo.no-data` polos. Area terpilih: outline tebal `color.border.focus`.

**Interaksi:** hover/fokus → tooltip (nama, nilai, kelas); klik → panel area + `area=` di URL; dobel klik / "Perbesar ke kelurahan" → drill level; tombol zoom/pan (alternatif drag, WCAG 2.5.7); kotak "Cari kelurahan"; panel area berisi CTA "Lihat N prospek" (FR-GEO-004), "Detail area" (SCR-10), "Buat segmen" (FR-GEO-005), "Lihat di peringkat" (SCR-09).

**Tampilan tabel (`view=table`)** — wajib, setara informasi: kolom area, induk, metrik terpilih + kelas (teks), cakupan, prospek HOT/WARM; dapat diurutkan; baris → panel area yang sama.

| State | Perilaku |
|---|---|
| Batas wilayah tidak tersedia | Otomatis `view=table` + info "Peta tidak tersedia untuk wilayah ini; data ditampilkan sebagai tabel." (OQ-UX-18) |
| Basemap gagal dimuat | Choropleth tetap tampil di latar `color.bg.surface` + info kecil |
| Semua area LIMITED | Banner "Sebagian besar area di wilayah ini memiliki data terbatas" + saran naik satu level |
| SCORING | Metrik spesifik tenant (Bisnis target, prospek HOT/WARM) berlabel "Menghitung" |

- **Responsive:** < `breakpoint.lg` default `view=table`; peta tersedia sebagai toggle layar penuh. **Analytics:** `area_map_viewed`, `area_selected`, `geo_guardrail_viewed`, `area_drilldown_clicked`, `segment_created`.

#### SCR-09 — Peringkat wilayah (Territory ranking)

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Wilayah mana yang paling layak dikerjakan tim saya untuk ICP ini?" | PER-03 | `/app/territories` | FR-GEO-003, FR-GEO-004, FR-GEO-005, FR-GEO-009 |

Wireframe: **WF-05**.

- **Toolbar:** wilayah cakupan (kab/kota atau provinsi), level (kelurahan/kecamatan), pengingat konteks "Untuk ICP … · Penawaran …", ringkasan "43 kelurahan: 39 diperingkat, 4 data terbatas", toggle "Sembunyikan area data terbatas", sortir.
- **Kolom:** pilih, peringkat, area (+ induk), Skor wilayah (angka + bar), Skor peluang pasar, Prospek HOT+WARM, Bisnis target, Kompetisi (teks + ikon), Rata-rata kualitas data, cakupan. Default urut skor wilayah ↓; seri → jumlah HOT+WARM.
- **Expand baris:** breakdown skor wilayah (0,5 · peluang pasar + 0,3 · persentil HOT+WARM + 0,2 · rata-rata kualitas data) dan skor peluang pasar (4 komponen) dengan kontribusi; komponen Unknown abu-abu "tidak dihitung"; metadata versi; CTA "Lihat N prospek", "Detail area" (FR-GEO-003 "setiap baris dapat dibuka ke breakdown").
- **Grup "Tidak diperingkat — data terbatas":** area cakupan LOW, kolom skor "Tidak diketahui", tooltip kriteria LOW (Lampiran C).
- **Aksi massal:** "Buat segmen dari area" (FR-GEO-005), "Lihat prospek di area terpilih" (SCR-04 terfilter). Compare areas = V1 (SCR-51), tidak dirender.
- **State khusus:** SCORING (skor wilayah bergantung ICP); ZERO bila ICP tidak punya kandidat di wilayah → saran ganti wilayah.
- **Responsive:** < `breakpoint.lg` kolom prioritas: peringkat, area, skor wilayah, HOT+WARM; kolom lain di baris expand. **Analytics:** `territory_ranking_viewed`, `territory_row_expanded`, `area_drilldown_clicked`, `segment_created`.

#### SCR-10 — Detail area

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Seperti apa pasar di area ini, dan siapa prospek terbaiknya?" | PER-03, PER-01 | `/app/areas/[areaCode]` | FR-GEO-001, FR-GEO-004, FR-GEO-005, FR-GEO-009 |

| Zona | Isi |
|---|---|
| Header | Nama area, level, hierarki induk, cakupan (P-06), periode data, aksi: "Lihat semua prospek", "Buat segmen", "Kembali ke peta/peringkat" |
| Kartu metrik | Populasi, kepadatan penduduk, bisnis (total), bisnis target, kepadatan kategori, kompetisi, celah pasokan, skor peluang pasar, skor wilayah; masing-masing dengan guardrail (P-10) |
| Posisi di area pembanding | Per metrik: strip chart distribusi peer group + titik area ini + teks "Persentil 72 dari 43 kelurahan di Kota Denpasar" (alternatif teks wajib) |
| Prospek | Distribusi band (bar bertumpuk + angka), 10 prospek teratas (komponen baris SCR-04) |
| Peluang dominan | Jumlah prospek target per tipe peluang di area (tabel) → klik = SCR-04 terfilter area + tipe peluang |
| Metadata | `geo_version`, snapshot, sumber & periode populasi, definisi peer group |

State khusus: LIMITED → kartu skor area "Tidak diketahui" + penjelasan; area tanpa bisnis target → "Belum ada bisnis yang cocok dengan ICP di area ini". Analytics: `area_detail_viewed`, `area_drilldown_clicked`.

#### SCR-11 — List & segmen (indeks)

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Kumpulan prospek mana yang sedang saya kerjakan?" | PER-02, PER-03 | `/app/lists` | FR-SEG-001, FR-SEG-002, FR-SRCH-005 |

- **Tab:** List (statis) · Segmen (dinamis) · Pencarian tersimpan.
- **Kolom:** nama, jenis, pemilik, dibagikan ke workspace (ya/tidak), jumlah prospek, distribusi band (bar mini + teks "12 HOT · 30 WARM · 8 COLD"), ICP (segmen), diperbarui, penerima (list).
- **Aksi:** buat list; buat segmen (→ SCR-04 lalu "Simpan sebagai segmen"); duplikat; ganti nama; bagikan/berhenti berbagi; hapus (pemilik objek + Owner/Admin) dengan konfirmasi.
- **Filter:** "Milik saya" / "Dibagikan"; cari nama. **State:** EMPTY per tab (§7.5). **Responsive:** < `md` kartu. **Analytics:** `list_created`, `segment_opened`, `saved_search_run`.

#### SCR-12 — Detail list / segmen

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Siapa saja di kumpulan ini dan apa tindakan berikutnya?" | PER-02, PER-03 | `/app/lists/[id]`, `/app/segments/[id]` | FR-SEG-001..006, FR-SRCH-007, FR-EXP-001, FR-EXP-004, FR-GEO-005 |

- **Header:** nama, badge jenis (List statis / Segmen dinamis), pemilik, status berbagi, jumlah, distribusi skor (bar bertumpuk + angka, FR-SEG-002).
- **Segmen:** chip filter + ICP (read-only), "Diperbarui otomatis · terakhir 10 Okt 2026, 09.12 WIB", "+12 sejak kunjungan terakhir", "Ubah filter" (→ SCR-04 dengan filter segmen → "Perbarui segmen"), "Simpan sebagai list" (snapshot statis).
- **List:** tambah (dari pencarian), hapus baris, "Tugaskan list" ke anggota (Manager ke atas).
- **Tabel:** komponen baris SCR-04 + kolom status kerja & penerima; bar aksi massal yang sama (termasuk Buka kontak massal → SCR-14).
- **State khusus:** segmen dengan ICP yang sudah dihapus → banner "ICP segmen ini sudah dihapus; skor memakai ICP aktif Anda"; list berisi prospek yang kemudian ter-suppress → baris hilang + catatan jumlah "2 prospek tidak lagi tersedia".
- **Responsive:** 360 px kartu + aksi per kartu (NFR-COMP-01). **Analytics:** `segment_opened`, `prospects_assigned`, `export_previewed`, `webhook_send_requested`.

#### SCR-13 — Tugas saya

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Prospek mana yang harus saya hubungi sekarang?" | PER-02 | `/app/tasks` | FR-SEG-005, FR-SEG-006, FR-PROF-003 |

- **Kontrol:** segmented control status kerja (Baru · Diriset · Siap dihubungi · Sudah dihubungi · Didiskualifikasi) dengan hitungan; filter list asal; Manager ke atas: filter "Ditugaskan oleh saya" dan pemilih anggota.
- **Item (kartu/baris):** nama, skor + band, peluang teratas, kanal terbaik (ter-mask atau tombol langsung WhatsApp/Telepon bila sudah dibuka), ditugaskan oleh + tanggal, ubah status cepat (dropdown), buka profil. Urut prioritas ↓.
- **State:** EMPTY "Belum ada tugas untuk Anda"; penerima tidak aktif (untuk Manager) → badge "Penerima tidak aktif" + "Tugaskan ulang".
- **Responsive:** mobile-first; 360 px kartu dengan target sentuh 44 px. **Analytics:** `tasks_viewed`, `work_status_changed`, `contact_action_clicked`.

#### SCR-14 — Dialog konfirmasi biaya

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Berapa biayanya, dan apa yang saya dapat?" | Semua yang dapat membelanjakan kredit | Dialog | FR-EXP-001, FR-EXP-002, FR-EXP-004, FR-EXP-006, FR-BILL-001, FR-BILL-004, FR-MON-001, FR-AI-003, BR-CRED-02..05 |

Wireframe: **WF-06**. Radix Dialog, lebar 560 px, `z.modal`.

| Mode | Isi pratinjau | Tombol utama |
|---|---|---|
| Buka kontak (1 prospek) | Biaya 1 kredit (didebit langsung), saldo sebelum → sesudah, jumlah kontak yang akan terbuka per jenis, "berlaku permanen untuk workspace", aturan refund INVALID ≤ 30 hari | "Buka kontak · 1 kredit" |
| Buka kontak (massal) | Total dipilih, sudah dibuka (0), akan dibuka (1/prospek), dikecualikan, total, saldo | "Buka N kontak · N kredit" |
| Ekspor | Langkah 1 Atur: format, field (grup: Identitas, Lokasi, Skor & peluang, Kontak, Komersial, Digital, Metadata trust — terkunci), cakupan (seleksi/list/segmen). Langkah 2 Biaya: seperti WF-06a | "Ekspor · N kredit" |
| Kirim ke webhook | Pilih endpoint aktif + event; rincian biaya sama dengan ekspor | "Kirim · N kredit" |
| Perbarui data | 1 kredit; apa yang dilakukan (verifikasi ulang, hitung ulang skor pada versi dataset terbaru); peringatan bila baru diperbarui | "Perbarui · 1 kredit" |
| Buat ulang brief | 1 kredit AI; catatan bahwa brief cache masih tersedia gratis | "Buat ulang · 1 kredit AI" |

Aturan: idempotency key dibuat saat dialog terbuka dan dipakai ulang untuk retry (BR-CRED-04); buka kontak langsung = debit, sedangkan ekspor, kirim webhook, perbarui data, dan (buat ulang) brief memakai pola **tahan → tagih/lepas** (BR-CRED-07): dialog menulis "N kredit ditahan selama proses; hanya ditagih untuk yang berhasil, sisanya dilepas", saldo yang ditampilkan = saldo tersedia, dan kontak dari ekspor/webhook baru dibuka saat proses sukses; biaya 0 → tombol tetap menyebut "0 kredit"; FLAG-OFF `unlock.enabled` → mode buka kontak tidak dapat dikonfirmasi (tombol nonaktif + penjelasan §4.0); READ-ONLY → semua mode nonaktif; dikecualikan = ter-suppress, tutup permanen bila difilter, atau melebihi batas plan — rincian di tooltip tanpa membuka identitas record ter-suppress; NO-CREDIT → WF-06b; "Jangan tanya lagi" hanya untuk mode buka kontak tunggal (OQ-UX-15). Fokus awal pada tombol "Batal"; ringkasan biaya terhubung lewat `aria-describedby`. Analytics: `export_previewed`, `insufficient_credit_shown`, `export_requested`, `prospect_unlocked`, `webhook_send_requested`, `prospect_refresh_requested`.

#### SCR-15 — Ekspor & proses (job center)

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Apakah ekspor, upload, dan pengiriman saya sudah selesai?" | Semua kecuali Viewer | `/app/jobs` | FR-EXP-001, FR-EXP-003, FR-DATA-012, FR-MON-001, FR-BILL-005, BR-CRED-07 |

- **Kolom:** jenis (Ekspor / Upload / Kirim webhook / Buka kontak massal / Perbarui), deskripsi ("Ekspor 182 prospek · CSV"), status (Antre · Diproses n% · Selesai · Sebagian gagal · Gagal · Dijeda), dibuat oleh, waktu, kredit (ditahan → ditagih / dilepas; debit untuk buka kontak massal; refund bila ada), aksi (Unduh — sisa waktu tautan; Coba lagi; Lihat laporan).
- **Detail job:** timeline status, rincian baris berhasil/gagal + alasan, entri ledger terkait (Ditahan, Ditagih, Dilepas), `correlation_id` (untuk support).
- **State:** tautan kedaluwarsa → "Buat ulang" (0 kredit untuk record yang sudah dibuka); job gagal → "N kredit yang ditahan dilepas" terlihat di baris dan ledger; FLAG-OFF ekspor/upload (`export.create`, `upload.create`) → job baru tidak dapat dibuat, job yang sudah berjalan tetap selesai; FLAG-OFF `webhook.delivery` → pengiriman berstatus "Dijeda" (bukan "Ditahan", yang khusus untuk kredit) dan dilanjutkan saat flag aktif lagi (`04` §12.4, §18.3).
- **Analytics:** `export_downloaded`, `job_retry_clicked`.

#### SCR-16 — Wizard upload

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Mana dari daftar saya yang sudah dikenal MesinProspek, dan apa intelijennya?" | PER-01, PER-02 | `/app/uploads/new`, `/app/uploads/[uploadId]` | FR-DATA-012, BR-ER-04, BR-ER-07, BR-PRIV-03 |

| Langkah | Isi | Validasi |
|---|---|---|
| 1. Unggah | Drop zone CSV/XLSX, unduh template, pilih sheet (XLSX), catatan privasi "Data upload hanya terlihat oleh workspace ini" | ≤ 50.000 baris; encoding UTF-8/Windows-1252 terdeteksi |
| 2. Petakan kolom | Tabel kolom file → field (Nama, Domain/Website, Telepon, Email, Alamat, Kota, Kategori, Nama kontak, Jabatan, ID eksternal, Abaikan); saran otomatis dari header id/en ("No HP", "Nama Usaha"); contoh 3 nilai per kolom | Minimum: nama + (domain/telepon/alamat) |
| 3. Pratinjau | 20 baris dengan ikon masalah per sel; ringkasan "1.820 valid · 34 akan ditolak (telepon tidak valid 20, nama kosong 14)"; centang "Saya berhak memproses data ini sesuai UU PDP" | Centang wajib |
| 4. Proses | Progres; boleh tinggalkan halaman | — |
| 5. Laporan cocok | Cocok (ter-link ke entity global + distribusi keyakinan link) · Mungkin cocok (menunggu konfirmasi, CTA "Tinjau n baris") · Baru (prospek privat: tampil di Cari prospek berlabel "Dari upload Anda") · Ditolak (alasan, unduh CSV) | — |
| 6. Tinjau "Mungkin cocok" | Daftar baris dengan skor link 0,80–0,92 (BR-ER-04): kolom kiri data upload, kolom kanan kandidat bisnis global (nama, alamat, kelurahan, telepon/domain ter-mask bila belum dibuka) + keyakinan link + field yang cocok/berbeda (ikon + teks). Aksi per baris & massal: **Tautkan** · **Jadikan prospek privat**. Ringkasan "12 ditautkan · 3 dijadikan privat · 5 belum diputuskan" | Baris yang belum diputuskan diperlakukan sebagai prospek privat (default tidak di-link). Menautkan = link, bukan merge; data global tidak berubah (BR-ER-07). Keputusan dapat diubah selama upload belum dihapus (`04` §4.6). |

- **Indeks upload:** daftar upload (file, tanggal, pengunggah, baris, match rate, jumlah "Mungkin cocok" yang belum diputuskan, list hasil).
- **State khusus:** FLAG-OFF upload; file > batas; kolom duplikat dipetakan ke field yang sama → error inline.
- **Analytics:** `upload_started`, `upload_mapping_confirmed`, `upload_completed`, `upload_possible_match_decided`.

#### SCR-17 — ICP & penawaran

| Pertanyaan | Persona | Route | Implements |
|---|---|---|---|
| "Siapa target saya dan apa yang saya tawarkan?" | PER-01, PER-03, PER-04 | `/app/icp` | FR-ICP-001, FR-ICP-002, FR-ICP-004, FR-ICP-006, FR-INTL-009, FR-BILL-003 |

- **Indeks:** bagian Penawaran (kartu: nama, tag) dan ICP (tabel: nama, penawaran terkait, versi, jumlah kandidat, status skor, "Aktif untuk saya", diperbarui oleh/kapan). Batas jumlah ICP per plan ditampilkan "2 dari 5 ICP tersimpan" (OQ-UX-17).
- **Editor ICP:** nama; penawaran terkait (OQ-UX-07); kriteria per baris dengan segmented control **Wajib / Preferensi**: kategori, wilayah, band karyawan (1–4, 5–19, 20–99, 100–499, 500+), band revenue (R1–R6, label §7.3), kondisi digital (mis. website tidak ditemukan / hanya media sosial / domain sendiri), tipe kontak wajib, pengecualian (kategori, wilayah). Preset bobot (read-only di MVP; tabel bobot Lampiran A + perubahan preset). Panel pratinjau: jumlah kandidat, 5 contoh, peringatan kandidat > 200.000.
- **Simpan:** dialog "Simpan sebagai versi 4? Skor untuk ICP ini dihitung ulang (biasanya ≤ 10 menit)." → FLOW-11. Riwayat versi (read-only) dalam drawer.
- **Role:** Member/Viewer melihat dalam mode baca + dapat menjadikan ICP aktif untuk dirinya.
- **State:** UPSELL saat batas ICP plan tercapai; ICP dipakai segmen → peringatan sebelum hapus ("Dipakai 3 segmen"). **Analytics:** `offering_created`, `icp_saved`, `icp_switched`.

#### SCR-18 — Pengaturan › Workspace

Pertanyaan: "Bagaimana workspace ini dikonfigurasi?" · Route `/app/settings/workspace` · Implements: FR-AUTH-002, FR-AUD-005, FR-BILL-003, PRD §8.1.1 (catatan "Member boleh ekspor"), NFR-L10N-01.

| Bagian | Isi | Role |
|---|---|---|
| Umum | Nama workspace, bahasa default (id-ID / en-US), zona waktu default (Asia/Jakarta) | Owner, Admin |
| Kebijakan | Toggle "Anggota (Member) boleh ekspor dan kirim ke webhook" (default aktif) | Owner, Admin |
| Plan & entitlement | Plan, seat, ICP, fair-use profil, kredit/bulan, kredit AI/bulan, endpoint webhook, batas baris ekspor (Lampiran E); "Ubah plan: hubungi kami" (self-serve = V1) | Owner |
| Zona bahaya | "Tutup workspace": ketik nama workspace untuk konfirmasi; info data dihapus setelah 30 hari, dapat dibatalkan selama masa tenggang | Owner |

#### SCR-19 — Pengaturan › Anggota & role

Pertanyaan: "Siapa yang punya akses dan apa haknya?" · Route `/app/settings/members` · Implements: FR-AUTH-003, FR-AUTH-004, FR-AUTH-005, FR-BILL-003.

- Tabel anggota: nama, email, role (dropdown inline untuk Owner/Admin), terakhir aktif, aksi (keluarkan). Tab "Undangan tertunda": email, role, kedaluwarsa (hitung mundur 7 hari), kirim ulang, batalkan.
- Dialog undang: multi-email (chip), role dengan ringkasan hak (diturunkan dari matriks RBAC), kuota seat "4 dari 5 seat terpakai".
- Konfirmasi keluarkan: "Akses <nama> dicabut dalam 1 menit. Tugas miliknya tetap ada dan perlu ditugaskan ulang."
- Analytics: `member_invited`, `member_role_changed`, `member_removed`.

#### SCR-20 — Pengaturan › Webhook & log pengiriman

Pertanyaan: "Apakah data terkirim ke sistem saya dengan andal?" · Route `/app/settings/webhooks` · Implements: FR-EXP-003, FR-EXP-004, FR-EXP-005, NFR-SEC-05, FR-AUD-001.

- **Daftar endpoint:** nama, URL (dipotong tengah), event berlangganan; **status** (lifecycle, enum `webhook_endpoint_status`, `04` §12.4): Aktif (`ACTIVE`) / Dinonaktifkan (`DISABLED_MANUAL`) / Dinonaktifkan otomatis — gagal beruntun (`DISABLED_FAILING`: ≥ 20 pengiriman berturut-turut masuk DLQ, atau balasan `410 Gone`) + tombol "Aktifkan lagi"; **uji** (dimensi terpisah, dari field `test.status`, enum `webhook_test_status` = `UNTESTED` | `PASSED` | `FAILED`, tidak pernah null, default `UNTESTED`): badge "Belum diuji" bila `test.status = UNTESTED` (belum pernah diuji, juga setelah URL diubah), "Uji gagal · <kode>" bila `FAILED`, "Lolos uji · <tanggal>" bila `PASSED`; badge ini informatif dan tidak memblokir pengiriman; tingkat sukses 7 hari (`stats_7d`); pengiriman terakhir (`last_delivery_at`).
- **Form endpoint:** URL (HTTPS wajib, validasi SSRF server), event (daftar dari katalog event di `04-technical-design.md`), secret (dibuat sistem, tampil sekali, "Rotasi secret" dengan masa transisi), tombol "Kirim uji" + panel hasil (kode, latensi, cuplikan respons ≤ 1 KB, kode error uji) yang di-polling sampai selesai (≤ ±12 detik).
- **Log pengiriman:** waktu, event, jumlah record, status (Menunggu · Dikirim · Terkirim · Mencoba ulang · Gagal (DLQ) · Dibatalkan, enum `delivery_status`), percobaan ke-n dari 7, jadwal retry berikutnya (1 m, 5 m, 30 m, 2 j, 6 j, 12 j; `04` §12.4), kode respons; filter status; detail → header bertanda tangan (tanpa secret), payload (PII sesuai kontrak), riwayat percobaan. DLQ: "Kirim ulang" per item / massal.
- **State:** UPSELL (Starter); kuota endpoint penuh; FLAG-OFF webhook (pengiriman ditahan, banner). Analytics: `webhook_endpoint_created`, `webhook_test_sent`, `webhook_redelivery_requested`.

#### SCR-21 — Pengaturan › Pemakaian & kredit

Pertanyaan: "Ke mana kredit saya terpakai?" · Route `/app/settings/usage` · Implements: FR-BILL-001..005, FR-BILL-007, BR-CRED-01..08.

- **Ringkasan:** per meter (kredit & kredit AI): **Tersedia** (dapat dipakai) dan **Ditahan** (dicadangkan untuk proses yang sedang berjalan, dengan tautan ke job-nya; BR-CRED-07); rincian per grant (jumlah, sisa, kedaluwarsa: bulanan di akhir siklus, top-up 12 bulan; urutan pemakaian = yang paling cepat kedaluwarsa, BR-CRED-08); meter entitlement (seat, ICP, profil fair-use bulan ini, endpoint webhook, batas baris ekspor).
- **Pemakaian (P1, FR-BILL-007):** per operasi (buka kontak, ekspor auto-unlock, refresh, AI brief, sales angle) dan per anggota; grafik batang + tabel setara.
- **Ledger:** waktu, jenis (Grant · Debit · **Ditahan** (HOLD) · **Ditagih** (CAPTURE) · **Dilepas** (RELEASE) · Refund · Kedaluwarsa · Penyesuaian; enum `ledger_entry_type` `04` §5.7), meter, jumlah (+/−) pada saldo tersedia dan saldo ditahan, saldo tersedia setelah, operasi (tautan ke prospek/job), oleh, ref usage event. Entri Ditahan, Ditagih, dan Dilepas milik satu job dikelompokkan (expand) agar terbaca "35 ditahan → 33 ditagih, 2 dilepas". Filter jenis, meter, anggota, tanggal. Footer: Saldo tersedia dan Saldo ditahan = Σ ledger per akun (FR-BILL-002).
- Role: Owner/Admin penuh; Manager hanya pemakaian & saldo (tanpa log audit). Analytics: `usage_viewed`.

#### SCR-22 — Pengaturan › Log audit

Pertanyaan: "Siapa melakukan apa, kapan?" · Route `/app/settings/audit` · Implements: FR-AUD-001, FR-AUD-002.

- Tabel immutable: waktu, aktor (termasuk "Tim MesinProspek (support)" untuk akses support, OQ-UX-33), aksi (login, perubahan role, ekspor, buka kontak, konfigurasi webhook, dsb.), target, IP. Filter aktor/aksi/tanggal; detail di drawer. Tidak ada aksi edit/hapus. Analytics: `audit_log_viewed`.

#### SCR-23 — Dialog laporkan data

Pertanyaan: "Bagaimana saya memberi tahu bahwa data ini salah?" · Implements: FR-DATA-014, FR-AUD-004.

- Field: claim yang dirujuk (terisi; dapat dipilih bila dari bar aksi), jenis masalah (Nilai salah · Tidak berlaku lagi (tutup/pindah) · Duplikat dengan bisnis lain · Bukan bisnis ini · Ini data pribadi saya · Lainnya), nilai yang benar (opsional), catatan (opsional, maks 500 karakter).
- "Ini data pribadi saya" → teks + tautan ke SCR-25, tombol kirim berubah menjadi "Buka formulir privasi".
- Setelah kirim: "Terima kasih. Tim kami meninjau laporan dalam 5 hari kerja." (SLA ops = placeholder, OQ-UX-24). Laporan yang diterima dapat menghasilkan koreksi nilai oleh tim MesinProspek (FR-OPS-009, FLOW-09 #4). Analytics: `data_issue_reported`.

#### SCR-24 — Akun saya

Route `/app/account` · Implements: FR-AUTH-005, FR-AUTH-006, NFR-L10N-01, NFR-A11Y-01. Isi: nama, email, password, MFA (opsional untuk user tenant), bahasa, zona waktu, sesi aktif + "Keluar dari semua sesi", preferensi: shortcut keyboard (aktif/nonaktif), kepadatan tabel default, "Konfirmasi setiap buka 1 kontak" (OQ-UX-15).

#### SCR-25 — Formulir permintaan subjek data (publik)

Route `/privacy/request` · Implements: FR-AUD-004, BR-PRIV-03. Tanpa login, dengan captcha. Field: jenis permintaan (Akses · Koreksi · Hapus · Keberatan), nama pemohon, email/telepon yang menjadi subjek, nama bisnis terkait (opsional), deskripsi, email pemohon untuk verifikasi.

- **Verifikasi identitas:** **OTP via email** (SMTP) ke email pemohon; formulir terkirim setelah kode benar (kirim ulang setelah 60 detik). Pemohon yang **hanya punya kontak telepon** memilih "Saya tidak punya email" → isi nomor yang tercatat + waktu yang nyaman dihubungi; tidak ada OTP SMS/WhatsApp di MVP. Copy: "Tim kami akan menelepon balik ke nomor ini untuk memastikan Anda pemilik data sebelum permintaan diproses." Verifikasi dilakukan manual oleh ops (FLOW-22 #2).
- **Setelah kirim:** nomor referensi + estimasi waktu (nilai SLA TBD, OQ-UX-13) + pengakuan penerimaan ≤ 1 hari kerja. Bahasa id-ID default; tersedia en-US. Dapat dipakai penuh di 360 px.

### 4.2 Ops console

Semua layar ops: desktop ≥ `breakpoint.lg`, `density.compact`, banner lingkungan (staging/production), aksi pengubah data wajib alasan dan tercatat audit. Setiap metrik dapat diklik ke daftar record penyebabnya (UX-P2). Menu dan tombol mengikuti peran internal (§2.5, PRD §8.1.2); kolom *Aksi* menyebut peran yang boleh menjalankannya.

| SCR | Pertanyaan | Komponen utama | Aksi | Implements |
|---|---|---|---|---|
| SCR-30 Data Health | "Apakah kualitas data turun?" | Pemilih snapshot; tile: total company, total establishment, rasio verifikasi (VERIFIED + LIKELY_VALID), rasio stale, rasio kandidat duplikat, evidence coverage, rasio establishment terpetakan ke kelurahan, rasio terklasifikasi non-"Lainnya" — masing-masing dengan target PRD dan delta vs snapshot sebelumnya; tabel fill-rate per field; tren antar snapshot | Klik tile → daftar record; "Buka merge review"; "Lihat import" | FR-OPS-001, FR-DATA-009, FR-INTL-001 |
| SCR-31 Pipeline Health & DLQ | "Apakah pipeline berjalan dan apa yang macet?" | Tabel stage DAG snapshot sesuai `04` §6.1 (`ingest.load`, `identity.normalize`, `identity.resolve`, `identity.verify`, `identity.classify`, `geo.assign`, `intel.derive`, `geo.area_metrics`, `intel.fill_rate`, `intel.gap_facts`, `intel.base_factors`, `query.project`) + antrean `scoring` (fan-out re-skor, `04` §7.9) + DAG upload (`ingest.parse_upload`, `identity.normalize_link`, `intel.score_private`, `query.project_private`): selesai, gagal, retry, latensi p50/p95, backlog; daftar job; panel alert (kegagalan > 5%, backlog > 1 jam, AI gagal > 5%, webhook gagal > 5%); tab DLQ dikelompokkan tipe job × kelas error | Redrive (pilih / per grup), Buang (alasan), buka `correlation_id` — Ops Engineer, Ops Admin | FR-OPS-002, RG-5 |
| SCR-32 Intelligence Health & Cost (P1) | "Apakah skor, peluang, dan AI berguna dan terjangkau?" | Distribusi skor per versi model (histogram + tabel), opportunity acceptance per tipe & per tag penawaran, feedback skor (terlalu tinggi/rendah), tingkat gagal & grounding AI, biaya per brief dan per enriched record; panel **Versi model** read-only (`GET /v1/ops/model-versions?kind=score\|geo\|prompt`, `04` §11.5): versi `priority-*`, `geo-*`, `brief-*`/`angle-*` dengan status, riwayat aktivasi, epoch yang disimpan selama bake, tautan pipeline tag | Drilldown ke contoh feedback dan prospek terkait. **Tidak ada tombol aktivasi/rollback versi**: aktivasi hanya lewat job manual Maintainer di pipeline tag (`06` §10.3–10.4); panel menautkan pipeline + instruksi | FR-OPS-003, FR-OPS-004, FR-INTL-009, FR-INTL-010 |
| SCR-33 Merge review | "Apakah dua record ini bisnis yang sama?" | Antrean (alasan bernilai tinggi, skor fuzzy, umur); tampilan berdampingan: nama, alias, alamat, koordinat + jarak, telepon, domain, kategori, sumber; identifier cocok (ikon cek + `color.status.success.*`) / bertentangan (ikon silang + `color.status.danger.*`); peta mini | Gabung · Company sama, cabang berbeda · Pisahkan · Lewati (shortcut G/C/P/L); undo 10 detik; riwayat keputusan per entity — Data Steward, Ops Admin | FR-DATA-004, FR-OPS-005, BR-ER-04..06 |
| SCR-34 Dataset snapshot | "Apakah snapshot ini layak diterbitkan, dan versi mana yang aktif?" | Daftar snapshot per sumber: versi `ds-YYYY.MM.n`, sumber, tanggal pengambilan, periode referensi, lisensi, status sesuai `04` §4.2 (Terdaftar `STAGED` · Validasi `VALIDATING` · Siap ditinjau `VALIDATED` · Diterbitkan/aktif `PUBLISHED` · Digantikan `SUPERSEDED` · Ditolak `REJECTED`), tautan pipeline tag; detail: laporan validasi (`04` §4.5), reject per alasan (unduh sampel), fill-rate per field + delta vs snapshot aktif, hasil gate QG-34 per cek, mapping Input Contract; **riwayat** per sumber: publish, digantikan, **diterbitkan ulang (re-publish/rollback)** + aktor, alasan, pipeline; panel "Cara menerbitkan / re-publish" (instruksi + tautan pipeline `ds-*`, `06` §10.2); tab Referensi (wilayah, batas, populasi, crosswalk) | Baca status, riwayat, dan laporan; buka issue rilis data untuk rekomendasi publish/tolak — Data Steward, Ops Admin. **Read-only untuk transisi snapshot**: tanpa tombol Terbitkan, Tolak, atau re-publish (publish, reject, re-publish = job manual Maintainer di pipeline tag setelah QG-34; `04` §11.5). Import referensi (`04` §4.7) — Data Steward, Ops Admin | FR-DATA-001, FR-DATA-008, PRD §8.1.2 |
| SCR-35 Privasi | "Apakah permintaan privasi tertangani tepat waktu?" | Tab Suppression list (tipe, nilai ter-mask, alasan, sumber tiket, pembuat, status propagasi per projection) dan tab Permintaan subjek data (no. ref, jenis, diterima, sisa SLA, status: Diterima · Verifikasi · Diproses · Diteruskan ke tenant · Selesai · Ditolak, metode verifikasi: OTP email / panggilan balik, penanggung jawab) | Verifikasi identitas (lihat hasil OTP email; catat hasil panggilan balik untuk pemohon tanpa email), cari entity, suppress (global layer), teruskan ke tenant (data upload), koreksi, buat laporan akses, tutup dengan template — Privacy & Billing, Ops Admin | FR-AUD-003, FR-AUD-004, FR-OPS-008, BR-PRIV-02, NFR-PRIV-02 |
| SCR-36 Laporan data | "Data mana yang dilaporkan salah oleh user?" | Antrean: claim, jenis masalah, workspace pelapor, jumlah laporan untuk entity yang sama, umur, status (Baru · Ditinjau · Diterima · Ditolak · Selesai) | Ubah status + catatan; **Koreksi nilai** (form nilai baru + alasan → membuat evidence `ops_correction` yang menang atas data dataset sampai ada observasi lebih baru, memicu hitung ulang skor, tercatat di audit); buka merge review (duplikat); buat suppression (privasi; diteruskan ke Privacy & Billing); tandai untuk snapshot berikutnya — Data Steward, Ops Admin | FR-DATA-014, FR-OPS-008, FR-OPS-009 |
| SCR-37 Tenant & kredit | "Bagaimana kondisi akun pelanggan?" | Daftar workspace (plan, seat terpakai, saldo tersedia & ditahan, dibuat, aktif terakhir, status); detail: plan & entitlement, ledger, anggota (read-only) | Beri kredit (FLOW-24), refund manual per usage event, ubah plan — Privacy & Billing, Ops Admin; "Akses support read-only" (alasan wajib → sesi baca ≤ 4 jam dengan banner "Mode support — <tenant> — tercatat di audit"; tanpa impersonation) — semua peran internal | FR-OPS-006, FR-BILL-006 |
| SCR-38 Feature flag & kill switch | "Fitur mana yang aktif untuk siapa?" | Bagian **kill switch** di atas dengan toggle besar + status untuk semua flag `04` §18.3: `ai.generation`, `export.create`, `webhook.delivery`, `upload.create`, `unlock.enabled`, `pipeline.publish`, `dns.lookup`, `search.include_private`, `scoring.bulk_rescore`, dan `maintenance.read_only` (default **off**; menyalakannya = mode read-only global, konfirmasi ganda dengan ketik "READ-ONLY"); masing-masing dengan default dan efeknya (teks dari `04` §18.3); tabel flag (nama, cakupan GLOBAL → PLAN → WORKSPACE, nilai efektif, diubah oleh/kapan) | Ubah (konfirmasi + alasan); override per plan/tenant — Ops Engineer, Ops Admin (peran lain: lihat) | FR-OPS-007, FR-AI-006 |
| SCR-39 Log audit internal | "Siapa (internal) melakukan apa?" | Sama dengan SCR-22, lintas tenant, filter jenis aktor (ops/support/sistem) dan peran internal | Filter + detail entri; tanpa edit/hapus (akses baca: OQ-UX-34) | FR-AUD-001 |
| SCR-40 Akun & peran internal | "Siapa punya akses internal apa?" | Daftar akun internal (nama, email, peran: Ops Engineer · Data Steward · Privacy & Billing · Ops Admin, status MFA, aktif terakhir) | Undang akun, ubah peran (multi-pilih), nonaktifkan; alasan wajib, teraudit; akun tanpa MFA tidak dapat login (FR-AUTH-006) — Ops Admin | PRD §8.1.2, FR-AUTH-006, FR-AUD-001 |

### 4.3 V1 — ringkas (tidak dibangun di MVP 1; kontrol tidak dirender sampai flag aktif)

| SCR | Layar | FR | Catatan UX |
|---|---|---|---|
| SCR-50 | Compare prospek (2–5) | — (PRD §6 V1, belum ber-ID; OQ-UX-23) | Kolom per prospek; baris = faktor skor, peluang, kontak, konteks lokasi |
| SCR-51 | Compare areas (2–5) | FR-GEO-006 | Dari bar aksi SCR-09 |
| SCR-52 | Sinyal (bagian profil + feed) | FR-PROF-011, FR-INTL-012, FR-DATA-013 | Mengisi slot "Sinyal" hierarki profil; faktor Kekuatan sinyal tidak lagi Unknown |
| SCR-53 | Watch & pusat notifikasi | FR-MON-002..004, FR-GEO-008 | Ikon lonceng di header; watch prospek dan area |
| SCR-54 | NL search | FR-SRCH-009 | Lapisan di SCR-04; hasil parsing selalu tampil sebagai chip yang dapat diedit |
| SCR-55 | Peta hasil (clustered) | FR-SRCH-010 | Toggle tampilan di SCR-04 |
| SCR-56 | Market opportunity report & ringkasan segmen | FR-GEO-007, FR-RPT-001 | Tautan/PDF; guardrail wajib ikut |
| SCR-57 | Billing self-serve | FR-BILL-008 | Menggantikan "Hubungi kami" |
| SCR-58 | API key | FR-AUTH-007 | Key tampil sekali |
| SCR-59 | Integrasi CRM | FR-EXP-007 | Target ditentukan survei beta (D-10) |

---

## 5. Wireframes

Wireframe ASCII bersifat struktural (hierarki, zona, isi), bukan visual final. Angka contoh konsisten antar wireframe (ABC Dental Clinic, ICP "Klinik gigi Bali v3", penawaran Website) dan dihitung dengan rumus Lampiran A dan C PRD.

### WF-01 — Hasil pencarian (SCR-04, desktop ≥ 1280 px)

```
+---------------------------------------------------------------------------------------------------------------+
| [=] Agensi Maju v | ICP aktif: [Klinik gigi Bali - Website - v3 v] | [Cari cepat  /] | Kredit 184 AI 42 | RA  |
+--------------+-----------------------+------------------------------------------------------------------------+
| Beranda      | FILTER         [Reset]| [ Cari nama, kategori, alamat...               ] [Simpan pencarian]    |
|>Cari prospek | v Kategori            | Klinik gigi (x) Kota Denpasar (x) HOT, WARM (x) Telepon (x)  Reset     |
| Peta wilayah |   [x] Klinik gigi 182 |------------------------------------------------------------------------|
| Peringkat    | v Wilayah             | 182 prospek cocok         Urut: [Prioritas v]   Kepadatan: [36|44]     |
|  wilayah     |   Bali > Kota Denpasar|------------------------------------------------------------------------|
| List & segmen| v Band prioritas      | [ ]  SKOR      BISNIS & LOKASI              PELUANG TERATAS    KONTAK  |
| Tugas saya 4 |   [x] HOT (61)        |------------------------------------------------------------------------|
| Upload       |   [x] WARM (121)      | [ ]  84 HOT    ABC Dental Clinic +2 cabang  Website tidak      T+ E+ W~|
| Ekspor &     |   [ ] COLD            |                Klinik gigi - Renon,         ditemukan                  |
|  proses (1)  |   [ ] Belum dinilai   |                Denpasar Selatan             Keyakinan Tinggi           |
|              | v Ketersediaan kontak |                Mutakhir - 12 hari lalu                                 |
| ICP &        |   [x] Telepon         |------------------------------------------------------------------------|
|  penawaran   |   [ ] Email           | [ ]  81 WARM   Senyum Sehat Dental          Rating rendah      T+ E? W~|
| Pengaturan   |   [ ] Kemungkinan WA  |      [Data     Klinik gigi - Sanur Kaja,    3,6 - 41 ulasan            |
| Bantuan      | > Kecocokan ICP       |      terbatas] Denpasar Selatan             Keyakinan Sedang           |
|              | > Tipe peluang        |                Mulai usang - 140 hari lalu                             |
|              | > Kemutakhiran        |------------------------------------------------------------------------|
|              | > Status verifikasi   | [ ]  76 WARM   Klinik drg. Putu             Website hanya      T+ E- W~|
|              | > Band karyawan       |                Klinik gigi - Panjer,        media sosial               |
|              | > Band revenue        |                Denpasar Selatan             Keyakinan Tinggi           |
|              | > Status bisnis       |                Mutakhir - 30 hari lalu                                 |
|              | > Rating              |------------------------------------------------------------------------|
|              | > Status buka kontak  | Menampilkan 25 dari 182                         [Muat 25 lagi]         |
|              | [ ] Belum ada di list |                                                                        |
|              |     saya              |                                                                        |
|              | v Sumber data         |                                                                        |
|              |   [x] Termasuk upload |                                                                        |
|              |       Anda (default)  |                                                                        |
+--------------+-----------------------+------------------------------------------------------------------------+
+---------------------------------------------------------------------------------------------------------------+
| BAR AKSI MASSAL (muncul saat >= 1 baris dipilih, sticky bawah, z.sticky)                                      |
| 3 dipilih - Pilih semua 182 hasil (batas paket: 10.000)  [Simpan ke list] [Tag] [Tugaskan]                    |
| [Buka kontak] [Ekspor] [Kirim ke webhook]  [x Batal]                                                          |
+---------------------------------------------------------------------------------------------------------------+

Legenda kontak: T telepon, E email, W WhatsApp. + Terverifikasi/Kemungkinan valid, ! Berisiko,
x Tidak valid, ? Status tidak diketahui, - tidak ditemukan di data sumber, ~ Perkiraan (dari nomor seluler).
Nilai kontak tidak tampil di hasil (ter-mask). Ikon selalu disertai teks di tooltip & aria-label (P-04).
Tidak ada di contoh ini: label "Dari upload Anda" (prospek privat, kolom bisnis) dan chip
"Kriteria wajib belum terkonfirmasi" (kolom skor, BR-SCORE-12). Tanpa nomor halaman: cursor + "Muat 25 lagi".
```

### WF-02 — Profil prospek (SCR-05, desktop ≥ 1280 px)

```
+---------------------------------------------------------------------------------------------------------------+
| < Kembali ke hasil   3 dari 182  [<] [>]                       Skor untuk ICP: Klinik gigi Bali v3            |
|                                                                                                               |
| ABC Dental Clinic                                  +--------------+   Kecocokan ICP 97/100                    |
| Klinik gigi (Turunan) - Buka (Teramati)            |  84    HOT   |   Peluang teratas:                        |
| Jl. Raya Puputan No. 12, Renon, Denpasar Selatan   |Skor prioritas|   Website tidak ditemukan                 |
| Cabang: [Renon - utama v]  +2 cabang lain          +--------------+   Cakupan data 95% - Mutakhir             |
| Status kerja: [Siap dihubungi v]   Ditugaskan ke: Budi S.   Tag: premium, renon                               |
+---------------------------------------------------------------------------------------------------------------+
| [Buka kontak - 1 kredit] [Simpan ke list] [Tugaskan] [AI brief] [Ekspor] [Kirim ke webhook] [Lainnya v]       |
| (Lainnya: Tag, Catatan, Perbarui data - 1 kredit, Laporkan data)                    sticky saat scroll        |
+------------------------------------------------------------------------+--------------------------------------+
| KENAPA PROSPEK INI?        priority-1.2.0+website_dev - 10 Okt         | AI BRIEF                       [ID v]|
| Faktor                Nilai  Bobot*              Kontrib.              | Ringkasan bisnis, peluang, dan hal   |
| Kecocokan ICP         97     26,3%   ##########  +25,5                 | yang belum diketahui, dengan sumber. |
| Potensi komersial ~   88     15,8%   #####       +13,9                 | [Buat brief - 1 kredit AI]           |
| Peluang pasar wilayah 80     15,8%   #####       +12,6                 |--------------------------------------|
| Peluang digital       72     15,8%   ####        +11,4                 | KONTEKS LOKASI           [Lihat area]|
| Kualitas data         92     10,5%   ####        +9,7                  | Kel. Renon - Kec. Denpasar Selatan   |
| Kemudahan dihubungi   90     10,5%   ####        +9,5                  | Populasi       31.420 jiwa (i)       |
| Kekuatan peluang      27     5,3%    #           +1,4                  |                periode data 2024     |
| Kekuatan sinyal ?     Tidak diketahui - tidak dihitung (bobot 5%)      | Klinik gigi    4 - 0,13/1.000 pddk(i)|
| Total 84,0 -> 84 HOT. * bobot efektif setelah faktor tidak             | Kompetisi      Sedang (P52)     (i)  |
| diketahui dikeluarkan. Cakupan data 95%. ~ Perkiraan ? Tidak diketahui | Peluang pasar  80/100           (i)  |
| Menurut Anda skor ini:  [Terlalu tinggi]  [Terlalu rendah]             | Lokasi dari koordinat - Tinggi       |
|------------------------------------------------------------------------|--------------------------------------|
| PELUANG (3)                                  untuk penawaran: Website  | KOMERSIAL                            |
| +--------------------------------------------------------------------+ | Karyawan  20-99 orang (Turunan)      |
| | Website tidak ditemukan   DIGITAL - [Hipotesis] - Keyakinan Tinggi | | Revenue   Rp5-15 miliar  [Perkiraan] |
| | Tidak ada website pada data sumber (versi 2026.09). 68% klinik     | |           Keyakinan Sedang           |
| | gigi di Kota Denpasar memiliki website.                            | | Basis: klinik gigi, 3 cabang, volume |
| | Relevansi: primer (Pembuatan website)         [Lihat bukti (2)]    | | ulasan P85 di Kota Denpasar.         |
| | Relevan untuk Anda?   [Relevan]  [Tidak relevan]                   | | Per 10 Okt 2026                      |
| +--------------------------------------------------------------------+ |--------------------------------------|
| | Email belum pakai domain sendiri - DIGITAL - Hipotesis - Sedang    | | KEHADIRAN DIGITAL                    |
| +--------------------------------------------------------------------+ | Website   Tidak ditemukan pada data  |
| | Listing tidak lengkap - LISTING - Hipotesis - Tinggi (sekunder)    | | Email     Webmail gratis (Turunan)   |
| +--------------------------------------------------------------------+ | Penyedia  Tidak diketahui            |
|------------------------------------------------------------------------| Rating    4,7 - 212 ulasan           |
| KONTAK (4)                          Belum dibuka - [Buka - 1 kredit]   |           (Teramati, 28 Sep 2026)    |
| Kanal    Nilai                  Milik        Status          Dicek     |--------------------------------------|
| Telepon  +62 812-****-7890      Bisnis       + Kemungkinan   12 hr     | CATATAN (2)              [+ Catatan] |
|          ~ Kemungkinan WhatsApp                valid                   | Rina: owner aktif di IG, coba DM     |
| Telepon  +62 361-***-456        Cabang Renon ! Berisiko      40 hr     | dulu sebelum telepon.                |
| Email    a****@gmail.com        Bisnis       + Kemungkinan   12 hr     |                                      |
|                                                valid                   |                                      |
| Email    d****@gmail.com        Pemilik      ? Tidak         -         |                                      |
|          (nama: d** A***)                      diketahui               |                                      |
| Kredit kembali otomatis bila semua kontak terbukti tidak valid         |                                      |
| dalam 30 hari setelah dibuka.                                          |                                      |
|------------------------------------------------------------------------|                                      |
| CABANG (3)                                                             |                                      |
| Renon (utama)  Denpasar Selatan  84 HOT   Mutakhir                     |                                      |
| Sanur          Denpasar Selatan  77 WARM  Mulai usang                  |                                      |
| Kuta           Kab. Badung       Tidak lolos kriteria wajib: wilayah   |                                      |
|------------------------------------------------------------------------|                                      |
| AKTIVITAS (privat workspace)                          [Semua jenis v]  |                                      |
| 10 Okt 09.12  Budi S. mengubah status Diriset -> Siap dihubungi        |                                      |
| 9 Okt 16.40   Rina A. menugaskan ke Budi S.                            |                                      |
| 9 Okt 16.38   Rina A. menyimpan ke list "Klinik Denpasar Q4"           |                                      |
+------------------------------------------------------------------------+--------------------------------------+
```

Catatan: ICP contoh memakai penawaran Website sehingga preset bobot = `website_dev` (`04` §7.1: Peluang digital 15, Kekuatan peluang 5, faktor lain sama dengan default Lampiran A; Σ = 100). Skor 84 = Σ kontribusi faktor yang diketahui = (25·97 + 15·88 + 15·80 + 15·72 + 10·92 + 10·90 + 5·27) / 95 = 7.980 / 95 = 84,0; bobot efektif = bobot preset / 95 karena Kekuatan sinyal (bobot 5) Unknown sampai V1 (cakupan 0,95). Dengan bobot default nilainya 81,6 → 82. Peluang pasar wilayah 80 = skor peluang pasar Kel. Renon (WF-05; fallback kecamatan bila kelurahan tidak tersedia). Peluang digital 72 = Website tidak ditemukan (severity 80 × keyakinan 0,90 × relevansi primer 1,0). Kekuatan peluang 27 = Listing tidak lengkap (50 × 0,90 × relevansi sekunder 0,6). Semua mengikuti Lampiran A/B PRD.

### WF-03 — Profil prospek (SCR-05, 360 px)

```
+----------------------------------------+
| <  ABC Dental Clinic          [...]    |
|----------------------------------------|
| 84 HOT - Kecocokan ICP 97              |
| Klinik gigi - Renon, Denpasar Sel.     |
| +2 cabang     Mutakhir - 12 hari lalu  |
| Peluang: Website tidak ditemukan       |
|----------------------------------------|
| v Kontak (4)            Belum dibuka   |
|   Telepon  +62 812-****-7890           |
|            + Kemungkinan valid         |
|   Email    a****@gmail.com             |
|            + Kemungkinan valid         |
|   ... 2 lainnya                        |
| > Kenapa prospek ini?                  |
| > Peluang (3)                          |
| > Konteks lokasi                       |
| > Komersial                            |
| > Kehadiran digital                    |
| > Cabang (3)                           |
| > AI brief                             |
| > Catatan & aktivitas                  |
|----------------------------------------|
| [Buka kontak - 1 kredit] [Aksi lain]   |
| (bar bawah sticky, tinggi 56 px,       |
|  target sentuh 44 px)                  |
+----------------------------------------+
```

### WF-04 — Peta wilayah (SCR-08)

```
+---------------------------------------------------------------------------------------------------------------+
| Peta wilayah   Wilayah: [Bali > Kota Denpasar v]   Level: [Kecamatan|(Kelurahan)]                             |
| Metrik: [Skor peluang pasar v]   Tampilan: [(Peta)|Tabel]   Konteks: ICP Klinik gigi Bali v3 - Website        |
+-------------------------------------------------------------------------+-------------------------------------+
| [+] [-] [Reset] [Cari kelurahan...]           Basemap: TBD (Q-05)       | SESETAN                        [x]  |
|                                                                         | Kelurahan - Kec. Denpasar Selatan   |
|    +--------------+--------------+--------------+                       | Skor peluang pasar  78/100  (i)     |
|    | Peguyangan   | Dangin Puri  | Kesiman      |                       | Peringkat wilayah  2 dari 43        |
|    |     [5]      |     [6]      |     [3]      |                       |-------------------------------------|
|    +--------------+--------------+--------------+                       | Populasi        41.902 (2024) (i)   |
|    | Pemecutan    | Dauh Puri    | Sumerta      |                       | Kepadatan       9.870/km2     (i)   |
|    |   [//////]   |     [4]      |     [7]      |                       | Klinik gigi     5 - 0,12/1.000 (i)  |
|    +--------------+--------------+--------------+                       | Kompetisi       Rendah (P28)  (i)   |
|    | Pedungan     | Sesetan      | Renon        |                       | Celah pasokan   72/100        (i)   |
|    |     [5]      |  ==[7]==     |     [7]      |                       | Bisnis target   11                  |
|    +--------------+--------------+--------------+                       | Prospek HOT/WARM  4 / 5             |
|    (skematik; peta asli = poligon kelurahan)                            | Cakupan data    Memadai             |
|                                                                         |-------------------------------------|
| LEGENDA - Skor peluang pasar (kuantil dalam Kota Denpasar)              | [Lihat 11 prospek]                  |
| [1] [2] [3] [4] [5] [6] [7]   rendah -> tinggi (color.geo.sequential)   | [Detail area] [Buat segmen]         |
| [////] Data terbatas  [    ] Tidak ada data  ==[ ]== area terpilih      |-------------------------------------|
| Periode populasi 2024 - geo-1.0.0 - ds-2026.09.1                        | (i) Kepadatan penduduk adalah       |
| (i) Skor relatif terhadap area pembanding, bukan prediksi penjualan.    | konteks pasar, bukan ukuran         |
|                                                                         | permintaan atau pendapatan.         |
+-------------------------------------------------------------------------+-------------------------------------+
```

### WF-05 — Peringkat wilayah (SCR-09)

```
+---------------------------------------------------------------------------------------------------------------+
| Peringkat wilayah   Wilayah: [Bali > Kota Denpasar v]   Level: [(Kelurahan)|Kecamatan]                        |
| Untuk ICP: Klinik gigi Bali v3 - Penawaran: Website     43 kelurahan: 39 diperingkat, 4 data terbatas         |
| [ ] Sembunyikan area data terbatas                              Urut: [Skor wilayah v]                        |
|---------------------------------------------------------------------------------------------------------------|
| [ ]  #   Area                    Skor wilayah Peluang   HOT+WARM Bisnis  Kompetisi  Kualitas                  |
|                                               pasar              target             data                      |
|---------------------------------------------------------------------------------------------------------------|
| [x]  1   Sumerta (Dps Timur)     82 ########  81        9        14      Rendah     78                        |
| [x]  2   Sesetan (Dps Selatan)   79 ########  78        9        11      Rendah     74                        |
| [ ]v 3   Renon (Dps Selatan)     78 ########  80        8        13      Sedang     81                        |
|       Skor wilayah 78 = 0,5 x peluang pasar 80 (+40,0) + 0,3 x persentil HOT+WARM 74 (+22,2)                  |
|                         + 0,2 x rata-rata kualitas data 81 (+16,2)                                            |
|       Peluang pasar 80 = 0,35 x populasi P88 (+30,8) + 0,25 x kepadatan target P92 (+23,0)                    |
|                          + 0,15 x aktivitas P95 (+14,3) + 0,25 x (100 - kompetisi 52) (+12,0)                 |
|       [Lihat 13 prospek]  [Detail area]                  geo-1.0.0 - populasi 2024 - ds-2026.09.1             |
| [ ]  4   Panjer (Dps Selatan)    70 #######   66        7        10      Sedang     80                        |
| ...                                                                                                           |
|---------------------------------------------------------------------------------------------------------------|
| TIDAK DIPERINGKAT - DATA TERBATAS (4)                              (i) kenapa tidak diperingkat?              |
| [ ]      Pemecutan (Dps Barat)   Tidak        Tidak     2        3       Tidak      70                        |
|          [Data terbatas]         diketahui    diketahui                  diketahui                            |
|---------------------------------------------------------------------------------------------------------------|
| 2 area dipilih   [Buat segmen dari area]   [Lihat prospek di area terpilih]      [x Batal]                    |
+---------------------------------------------------------------------------------------------------------------+
```

### WF-06 — Dialog konfirmasi biaya (SCR-14)

```
(a) Ekspor - saldo cukup
+------------------------------------------------------------------+
| Ekspor 182 prospek                                         [x]   |
|------------------------------------------------------------------|
| Langkah 2 dari 2 - Pratinjau biaya                               |
|                                                                  |
|  Total dipilih                           182 baris               |
|  Sudah dibuka (gratis)                   145         0 kredit    |
|  Akan dibuka otomatis                    35          35 kredit   |
|  Dikecualikan, tidak dapat diekspor (i)  2           0 kredit    |
|  --------------------------------------------------------------  |
|  Total biaya                                         35 kredit   |
|  Saldo tersedia                                      184 -> 149  |
|                                                                  |
|  Format CSV - 14 kolom, termasuk jenis data, keyakinan, dan      |
|  tanggal verifikasi. Tautan unduhan berlaku 24 jam.              |
|  Kontak yang dibuka berlaku permanen untuk workspace ini.        |
|  35 kredit ditahan selama proses: ditagih hanya untuk            |
|  baris yang berhasil diekspor, dilepas utuh bila gagal.          |
|  Kontak baru dibuka saat ekspor berhasil.                        |
|                                                                  |
|                    [Kembali]   [Batal]   [Ekspor - 35 kredit]    |
+------------------------------------------------------------------+

(b) Ekspor - kredit tidak cukup (state NO-CREDIT)
+------------------------------------------------------------------+
| Ekspor 182 prospek                                         [x]   |
|------------------------------------------------------------------|
|  Total biaya                                         35 kredit   |
|  Saldo                                               20 kredit   |
|                                                                  |
|  [!] Kredit tidak cukup. Kurang 15 kredit.                       |
|  Pilih salah satu:                                               |
|  (o) Ekspor hanya yang sudah dibuka: 145 baris - 0 kredit        |
|  ( ) Ekspor 165 baris: 145 sudah dibuka + 20 prioritas           |
|      tertinggi yang belum dibuka - 20 kredit                     |
|  Atau minta tambahan kredit ke Pemilik workspace. [Salin pesan]  |
|                                                                  |
|                       [Batal]   [Ekspor 145 baris - 0 kredit]    |
+------------------------------------------------------------------+

(c) Buka kontak - satu prospek
+------------------------------------------------------+
| Buka kontak ABC Dental Clinic?                 [x]   |
|------------------------------------------------------|
|  Biaya            1 kredit                           |
|  Saldo            184 -> 183 kredit                  |
|  Yang terbuka     4 kontak di semua cabang           |
|                   (2 telepon, 2 email)               |
|                                                      |
|  Berlaku permanen untuk workspace Agensi Maju.       |
|  Kredit kembali otomatis bila semua kontak           |
|  terbukti tidak valid dalam 30 hari.                 |
|                                                      |
|  [ ] Jangan tanya lagi untuk buka 1 kontak           |
|                                                      |
|                 [Batal]   [Buka kontak - 1 kredit]   |
+------------------------------------------------------+
```

### WF-07 — Evidence drawer (SCR-06)

```
+----------------------------------------------------------+
| Bukti: Telepon (Bisnis)                             [x]  |
|----------------------------------------------------------|
| +62 812-****-7890                                        |
| [Teramati]  Keyakinan Sedang (i)  Mutakhir - 12 hari     |
| Status: Kemungkinan valid - format Indonesia valid,      |
| tipe seluler.                                            |
|----------------------------------------------------------|
| DARI MANA NILAI INI?                                     |
| 1. Dataset bisnis MesinProspek - versi ds-2026.09.1      |
|    Ref. record sumber 48213 - diamati 28 Sep 2026        |
|    Nilai mentah: 0812-****-7890                          |
|    [Teramati] keyakinan sumber 86                        |
|----------------------------------------------------------|
| VERIFIKASI                                               |
| Cek format & tipe nomor - 9 Okt 2026 - Kemungkinan valid |
| Kemungkinan WhatsApp: [Perkiraan] dari nomor seluler;    |
| belum pernah dikonfirmasi.                               |
|----------------------------------------------------------|
| NILAI LAIN YANG DITEMUKAN (1)                            |
| +62 812-****-1122 - ds-2026.03.1 - diamati 4 Mar 2026    |
| Tidak dipakai: lebih lama, keyakinan lebih rendah.       |
| Karena ada nilai bertentangan, keyakinan nilai utama     |
| dikurangi 20 poin (86 -> 66 = Sedang).                   |
|----------------------------------------------------------|
| > Bagaimana keyakinan dihitung?                          |
|                                                          |
| [Laporkan data ini]                             [Tutup]  |
+----------------------------------------------------------+
```

---

## 6. Pola UI trust & intelligence

Semua pola dibangun di `packages/ui` di atas Radix primitives dan hanya memakai token semantik `02-design-system/`. Aturan umum aksesibilitas: **status tidak pernah hanya dengan warna** — selalu ikon (bentuk berbeda per nilai) + teks (NFR-A11Y-01).

### P-01 — Badge jenis data (claim type)

| Nilai | Label id-ID | Label en-US | Ikon | Token |
|---|---|---|---|---|
| OBSERVED | Teramati | Observed | mata | `color.claim.observed.{fg,bg,border}` |
| DERIVED | Turunan | Derived | cabang/fungsi | `color.claim.derived.{fg,bg,border}` |
| INFERRED | Perkiraan (nilai) / Hipotesis (peluang, sales angle) | Estimated / Hypothesis | gelombang (≈) | `color.claim.inferred.{fg,bg,border}` |
| UNKNOWN | Tidak diketahui | Unknown | tanda tanya | `color.claim.unknown.{fg,bg,border}` |

- **Anatomi:** pill `radius.full`, padding `space.0-5`/`space.2`, ikon `size.icon.sm`, teks `text.caption`. Varian *full* (ikon + teks) dan *compact* (ikon saja + tooltip + `aria-label`).
- **Aturan:** setiap nilai di profil, drawer, dan ekspor membawa jenis data (BR-TRUST-01). Varian compact hanya untuk OBSERVED/DERIVED di tabel padat yang punya legenda; INFERRED dan UNKNOWN **selalu** varian full (BR-TRUST-05/06). Tooltip: definisi singkat + "Lihat bukti".
- **A11y:** `aria-label="Jenis data: Perkiraan"`; ikon `aria-hidden`.

### P-02 — Indikator keyakinan (confidence)

- **Anatomi:** meter 3 segmen (terisi 3/2/1) + teks Tinggi/Sedang/Rendah; tooltip "Keyakinan 86 dari 100 · 1 sumber · lihat cara hitung". Token `color.confidence.{high,medium,low}.{fg,bg}`, teks `text.caption`.
- **Aturan:** Tinggi ≥ 80, Sedang 50–79, Rendah < 50; angka hanya di tooltip/drawer (BR-TRUST-03). Revenue inferred maksimal 70 → maksimal "Sedang" (BR-REV-03). Ada nilai bertentangan → tooltip menyebut pengurangan 20 poin (BR-TRUST-02).
- **A11y:** jumlah segmen terisi = level (bukan warna saja); `aria-label="Keyakinan tinggi, 86 dari 100"`.

### P-03 — Badge kemutakhiran (freshness)

| Status | Label id-ID | en-US | Ikon | Token |
|---|---|---|---|---|
| Fresh | Mutakhir | Fresh | jam + centang | `color.freshness.fresh.{fg,bg}` |
| Aging | Mulai usang | Aging | jam | `color.freshness.aging.{fg,bg}` |
| Stale | Usang | Stale | jam + tanda seru | `color.freshness.stale.{fg,bg}` |

- **Ambang (BR-TRUST-04):** contact point Mutakhir ≤ 90 hari, Mulai usang 91–180, Usang > 180; atribut company ≤ 180 / 181–365 / > 365; data populasi Mutakhir bila periode = periode terbit terbaru di sistem, selain itu Mulai usang.
- **Anatomi:** ikon + label + waktu relatif ("Mutakhir · 12 hari lalu"); tooltip tanggal absolut + ambang yang berlaku. "Usang" pada prospek yang boleh di-refresh menampilkan tautan "Perbarui data · 1 kredit".
- **Pilihan label:** "Mutakhir → Mulai usang → Usang" membentuk skala bertahap yang mudah dibaca; "Kedaluwarsa" dihindari karena menyiratkan data tidak berlaku. Skor stale (FR-INTL-011) memakai label terpisah "Menunggu hitung ulang" agar tidak tertukar dengan kemutakhiran data.

### P-04 — Badge status verifikasi

| Status | Label id-ID | en-US | Ikon | Token | Tooltip (contoh) |
|---|---|---|---|---|---|
| VERIFIED | Terverifikasi | Verified | perisai + centang | `color.verification.verified.{fg,bg}` | "Lolos verifikasi penyedia" |
| LIKELY_VALID | Kemungkinan valid | Likely valid | centang | `color.verification.likely-valid.{fg,bg}` | "Format valid dan domain punya server email (MX)" |
| RISKY | Berisiko | Risky | segitiga peringatan | `color.verification.risky.{fg,bg}` | Sesuai alasan: "Alamat umum (info@) atau catch-all" · "Domain tidak punya server email khusus (MX), tetapi punya alamat server (A/AAAA) yang mungkin menerima email" · "Domain baru/tidak stabil" · telepon: "Format valid, tipe nomor tidak dikenal" |
| INVALID | Tidak valid | Invalid | lingkaran silang | `color.verification.invalid.{fg,bg}` | Sesuai alasan: "Format email tidak valid" · "Domain tidak ditemukan (tidak terdaftar)" · "Domain tidak punya server email (MX) maupun alamat server (A/AAAA)" · telepon: "Format nomor tidak valid" |
| UNKNOWN | Tidak diketahui | Unknown | tanda tanya | `color.verification.unknown.{fg,bg}` | "Belum dicek atau pengecekan gagal sementara" |

Aturan: tooltip memakai alasan spesifik dari hasil verifikasi (BR-VER). Domain tanpa MX tetapi ber-A/AAAA = **Berisiko**, bukan Tidak valid (implicit MX, RFC 5321); Tidak valid hanya untuk syntax gagal, domain tidak ada (NXDOMAIN), atau domain tanpa MX maupun A/AAAA. Kegagalan sementara = Tidak diketahui, bukan Tidak valid (BR-VER-02). WhatsApp tidak pernah "Terverifikasi" di MVP; tampil sebagai "Kemungkinan WhatsApp" + P-01 Perkiraan (BR-VER-01). A11y: `aria-label="Status verifikasi: Berisiko — alamat umum"`.

### P-05 — Badge skor & band

- **Anatomi:** komponen `ScoreBadge` (`02-design-system/`): angka 0–100 (`text.numeric`, tabular) + kata band. Ukuran *sm* (tabel, tinggi 24 px, varian subtle): bg `color.score.{band}.bg`, teks `color.score.{band}.fg`, border `color.score.{band}.border`. Ukuran *lg* (header profil, varian solid): bg `color.score.{band}.solid`, teks `color.score.{band}.on-solid` (pasangan kontras yang divalidasi per band; **bukan** `color.fg.on-accent`, karena WARM memakai teks gelap di atas amber), `radius.md`.
- **Band (BR-SCORE-05):** HOT ≥ 80, WARM 60–79, COLD < 60, UNSCORED. UNSCORED: teks "Belum dinilai" **tanpa angka** (tidak memakai "0", "—", atau "-", BR-TRUST-06); *sm* memakai border putus-putus `color.score.unscored.border`, *lg* memakai `color.score.unscored.solid` + `color.score.unscored.on-solid`.
- **Pembatasan:** cakupan < 0,60 → band maksimal WARM walau nilai ≥ 80; tooltip "Nilai 81 masuk rentang HOT, tetapi band dibatasi WARM karena cakupan data 52%" (BR-SCORE-04). Kriteria ICP *wajib* bernilai Tidak diketahui → band maksimal WARM + chip terpisah "Kriteria wajib belum terkonfirmasi" (ikon tanda tanya + teks, `color.status.warning.{fg,bg,border}`); tooltip "Nilai 85 masuk rentang HOT, tetapi band dibatasi WARM karena kriteria wajib 'Wilayah' belum terkonfirmasi" (BR-SCORE-12).
- **Keputusan bahasa band:** **HOT/WARM/COLD dipertahankan dalam bahasa Inggris** di id-ID dan en-US karena (1) istilah "hot/warm/cold lead" lazim dipakai tim sales Indonesia, (2) satu kosakata yang sama di UI, filter, kolom ekspor, dan payload webhook sehingga tidak perlu pemetaan saat data masuk CRM, (3) pendek dan stabil untuk tabel. UNSCORED dilokalkan menjadi "Belum dinilai" karena ia adalah *state*, bukan jargon band (nilai ekspor tetap `UNSCORED`). Divalidasi pada usability test RG-1 (OQ-UX-16).
- **A11y:** `aria-label="Skor prioritas 81 dari 100, band WARM, dibatasi karena data terbatas"` / "… dibatasi karena kriteria wajib belum terkonfirmasi". Band dibedakan oleh kata, bukan warna.

### P-06 — Badge cakupan "Data terbatas"

- **Anatomi:** ikon lingkaran setengah terisi + teks "Data terbatas"; token `color.status.warning.{fg,bg,border}`, `text.caption`.
- **Tooltip prospek:** "Cakupan data 52%: hanya sebagian faktor skor yang datanya tersedia. Faktor yang tidak diketahui tidak dihitung, dan band dibatasi maksimal WARM."
- **Tooltip area:** "Data terbatas: populasi < 500, bisnis tercatat < 5, atau kepadatan bisnis jauh di bawah area sejenis. Celah pasokan dan skor peluang area tidak dihitung." (Lampiran C, FR-GEO-009)
- **A11y:** teks selalu tampil (bukan ikon saja).

### P-07 — Penjelasan skor (factor bars)

- **Anatomi per baris:** label faktor · nilai sₖ (0–100) · bobot efektif (tooltip: bobot default) · bar kontribusi · kontribusi (+n,n poin) · P-01 · tautan bukti. Footer: total (dengan catatan pembulatan), cakupan, `model_version` + preset penawaran, `calculated_at`, status stale, feedback skor (FR-INTL-010).
- **Label faktor:** Kecocokan ICP, Kualitas data, Kemudahan dihubungi, Potensi komersial, Peluang digital, Peluang pasar wilayah, Kekuatan sinyal, Kekuatan peluang (Lampiran A).
- **Aturan:** urut kontribusi ↓; faktor UNKNOWN di bawah, baris abu-abu `color.fg.subtle` dengan teks "Tidak diketahui — tidak dihitung" dan bobot dicoret, catatan "bobotnya dibagi ke faktor lain" (BR-SCORE-03). Kontribusi = bobot efektif × sₖ; jumlah kontribusi = skor. Expand Kecocokan ICP → kriteria: Cocok / Sebagian / Tidak cocok / Tidak diketahui (ikon + teks) dengan label Wajib/Preferensi.
- **Token:** bar `color.accent.default` di atas track `color.bg.surface-sunken`; baris Unknown `color.claim.unknown.bg`; angka `text.numeric`; metadata `text.caption` `color.fg.muted`.
- **A11y:** disajikan sebagai `<table>` dengan caption "Rincian skor prioritas"; bar `aria-hidden` karena nilai sudah tertulis.

### P-08 — Kontak ter-mask & buka kontak

- **Pola mask (BR-CRED-02):** telepon `+62 812-****-7890` (kode negara + 3 digit awal + 4 digit akhir); telepon tetap `+62 361-***-456`; email `h****@abcdental.co.id` (huruf pertama + domain); nama orang `d** A***` (OQ-UX-14). Nilai mentah di drawer ikut ter-mask.
- **Anatomi:** nilai `text.mono` + ikon gembok + status P-04; tombol "Buka kontak · 1 kredit" (satu per Company, berlaku untuk semua kontak dan cabang). Setelah dibuka: nilai penuh, tombol Salin, Telepon, WhatsApp, Email (ikon + teks), keterangan "Dibuka oleh <nama> · <tanggal>".
- **Aturan:** unlock permanen per workspace; prospek yang sudah dibuka anggota lain tampil tanpa biaya; Salin tidak tersedia untuk nilai ter-mask; kontak dari upload milik workspace tidak di-mask (OQ-UX-11); nilai global layer yang ter-suppress tidak dirender (nilai dari upload workspace sendiri tidak disembunyikan otomatis, BR-PRIV-02); flag `unlock.enabled` off → tombol nonaktif dengan penjelasan (FLAG-OFF §4.0), tidak disembunyikan.
- **A11y:** `aria-label="Telepon tersembunyi, berakhiran 7890. Buka kontak untuk melihat."`; tombol aksi kontak berlabel lengkap ("Hubungi via WhatsApp +62 812 3456 7890").
- **Token:** `color.bg.surface-sunken` untuk nilai ter-mask, `color.fg.muted`; tombol `color.accent.*`.

### P-09 — Label nilai perkiraan (inferred)

- **Anatomi:** nilai (gaya `text.body`, tidak pernah ditebalkan seperti fakta) + P-01 "Perkiraan" + P-02 + baris "Basis: …" + "Per <tanggal>".
- **Aturan:** BR-TRUST-05, BR-REV-03..05. Revenue hanya band (tanpa point estimate). WhatsApp: "Kemungkinan WhatsApp — dari nomor seluler". Sales angle & peluang: "Hipotesis" + rationale.
- **Token:** `color.claim.inferred.*`.

### P-10 — Tooltip guardrail metrik geo

- **Anatomi:** ikon info (`size.icon.sm`, `color.status.info.fg`) di samping label metrik → Popover (bukan tooltip hover saja, agar dapat dibuka dengan sentuh dan keyboard) berisi: definisi, guardrail (§7.7), periode data + sumber, versi `geo-x.y.z`. Teks guardrail yang sama juga tampil sebagai footnote permanen di legenda peta, panel area, dan detail area.
- **Aturan:** FR-GEO-009, FR-PROF-006 (periode selalu tampil), Rev 1.2 §8.
- **A11y:** trigger berupa `button` dengan `aria-label="Tentang metrik kepadatan penduduk"`; Esc menutup; fokus kembali ke trigger.

### P-11 — Kartu peluang (opportunity)

- **Anatomi:** judul tipe peluang (label §7.2) · chip keluarga (DIGITAL, REPUTASI, LISTING, PASAR, OPERASIONAL) · P-01 "Hipotesis" · P-02 · rationale (BR-OPP-04) · relevansi ("primer/sekunder untuk <penawaran>") · tautan "Lihat bukti (n)" · feedback **Relevan** / **Tidak relevan** (alasan: Data salah · Bukan kebutuhan penawaran kami · Mereka sudah punya solusi · Lainnya).
- **Aturan:** hanya peluang dengan relevance > 0 untuk penawaran ICP aktif yang disebut peluang; gap fact yang tidak relevan tetap terlihat di kartu digital/reputasi (BR-OPP-03). Setelah feedback: chip "Anda menandai: Relevan" + undo. Footer kecil: "Peluang adalah hipotesis berbasis data, bukan jaminan hasil penjualan." (BR-OPP-05)
- **Token:** `color.bg.surface`, `color.border.default`, `radius.lg`, `space.4`; feedback terpilih `color.accent.subtle`.
- **A11y:** tombol feedback `aria-pressed`; kartu adalah `article` dengan heading.

### P-12 — Nilai tidak diketahui

- **Anatomi:** teks "Tidak diketahui" (`color.claim.unknown.fg`) + ikon tanda tanya + tooltip alasan spesifik.
- **Aturan:** tidak pernah kosong, "0", "-", atau "N/A" (BR-TRUST-06), termasuk di ekspor (`UNKNOWN`). Alasan dibedakan: "tidak ada di data sumber", "belum dicek", "data terbatas di area ini".

### P-13 — Tombol berbayar

- **Anatomi:** label kerja + pemisah + biaya ("Buka kontak · 1 kredit"); chip biaya `color.bg.surface-sunken`, `text.caption`.
- **Aturan:** biaya selalu tampil sebelum klik; klik selalu membuka SCR-14 kecuali preferensi OQ-UX-15; biaya 0 tetap ditulis bila aksi biasanya berbayar. NO-CREDIT tidak menyembunyikan tombol.

---

## 7. Konten & microcopy (id-ID)

### 7.1 Nada

| Aturan | Contoh benar | Hindari |
|---|---|---|
| Sapaan formal "Anda"; kalimat aktif, pendek | "Simpan prospek ke list." | "Prospek akan dapat disimpan oleh user ke dalam list." |
| Faktual, tanpa hiperbola atau tanda seru | "84 prospek cocok dengan ICP Anda." | "Wow! Banyak prospek hebat!" |
| Jujur soal ketidakpastian | "Website tidak ditemukan pada data sumber." | "Bisnis ini tidak punya website." |
| Sebut biaya dan akibat | "Buka kontak · 1 kredit. Berlaku permanen." | "Buka sekarang!" |
| Istilah teknis dipertahankan bila lazim di sales | ICP, HOT/WARM/COLD, webhook, list | Terjemahan dipaksakan ("kait web") |
| Angka spesifik daripada kata sifat | "Kurang 15 kredit." | "Kredit Anda tidak mencukupi." |

### 7.2 Glosarium UI

| Istilah PRD/kode | Label UI id-ID | Label en-US | Definisi untuk user |
|---|---|---|---|
| Prospect | Prospek | Prospect | Bisnis beserta cabang terbaik yang cocok dengan ICP aktif |
| Company | Bisnis | Business | Entitas bisnis/brand |
| Establishment | Cabang | Location | Lokasi fisik sebuah bisnis |
| Contact point | Kontak | Contact | Telepon, email, atau WhatsApp |
| Unlock | Buka kontak | Unlock contacts | Menampilkan kontak tersembunyi, 1 kredit per bisnis, permanen |
| Credit / AI credit | Kredit / Kredit AI | Credits / AI credits | Satuan pemakaian berbayar |
| Priority score | Skor prioritas | Priority score | 0–100, seberapa layak dikejar untuk ICP aktif |
| Score band | HOT · WARM · COLD · Belum dinilai | HOT · WARM · COLD · Unscored | Kelompok skor (P-05) |
| ICP | ICP (profil pelanggan ideal) | ICP | Kriteria target Anda |
| ICP fit | Kecocokan ICP | ICP fit | Seberapa cocok dengan kriteria ICP |
| Offering | Penawaran | Offering | Produk/layanan yang Anda jual |
| Opportunity | Peluang | Opportunity | Hipotesis kebutuhan yang relevan dengan penawaran Anda |
| Signal (V1) | Sinyal | Signal | Perubahan yang meningkatkan relevansi |
| Evidence | Bukti | Evidence | Sumber sebuah nilai |
| Claim type | Jenis data: Teramati · Turunan · Perkiraan · Tidak diketahui | Data type: Observed · Derived · Estimated · Unknown | P-01 |
| Confidence | Keyakinan: Tinggi · Sedang · Rendah | Confidence: High · Medium · Low | P-02 |
| Freshness | Kemutakhiran: Mutakhir · Mulai usang · Usang | Freshness: Fresh · Aging · Stale | P-03 |
| Coverage (LOW / < 0,6) | Cakupan data; "Data terbatas" | Data coverage; "Limited data" | P-06 |
| Verification status | Status verifikasi | Verification status | P-04 |
| List / Segment | List / Segmen | List / Segment | Kumpulan statis / dinamis |
| Saved search | Pencarian tersimpan | Saved search | Query + filter + ICP |
| Assign / Assignment | Tugaskan / Tugas | Assign / Task | Menyerahkan prospek ke anggota |
| Work status | Status kerja: Baru · Diriset · Siap dihubungi · Sudah dihubungi · Didiskualifikasi | Work status | Bukan pipeline deal (FR-SEG-006) |
| Export | Ekspor | Export | — |
| Refresh | Perbarui data | Refresh | Verifikasi ulang + hitung ulang skor, 1 kredit |
| Dataset snapshot | Versi dataset | Dataset version | `ds-YYYY.MM.n` |
| Area | Wilayah / area | Area | Provinsi, kab/kota, kecamatan, kelurahan/desa |
| market_opportunity_score | Skor peluang pasar | Market opportunity score | Lampiran C |
| territory_score | Skor wilayah | Territory score | Lampiran C, spesifik ICP |
| Geographic Opportunity (faktor skor) | Peluang pasar wilayah | Geographic opportunity | Skor peluang pasar kelurahan tempat cabang berada (Lampiran A) |
| white_space_score | Celah pasokan | White space | Potensi kekurangan pasokan relatif |
| competition_index | Kompetisi: Rendah · Sedang · Tinggi | Competition: Low · Medium · High | Persentil kepadatan kategori di area pembanding |
| Peer group | Area pembanding | Peer areas | Lampiran C |
| Reference period | Periode data | Reference period | Periode data populasi |
| Roles | Pemilik · Admin · Manajer · Anggota · Hanya lihat | Owner · Admin · Manager · Member · Viewer | PRD §8.1.1 |
| Peran internal (ops console) | Ops Engineer · Data Steward · Privacy & Billing · Ops Admin | sama | PRD §8.1.2; hanya di ops console |
| Private prospect | Prospek privat; label baris "Dari upload Anda" | Private prospect; "From your upload" | Baris upload yang tidak ditautkan ke bisnis global; hanya terlihat oleh workspace pengunggah (FR-DATA-012) |
| Possible match (0,80–0,92) | Mungkin cocok · Tautkan · Jadikan prospek privat | Possible match · Link · Keep as private | Kandidat link upload yang menunggu konfirmasi user (FR-DATA-012) |
| HOLD / CAPTURE / RELEASE | Ditahan · Ditagih · Dilepas | Held · Charged · Released | Pola kredit proses async: ditahan saat dibuat, ditagih saat sukses, dilepas bila gagal (BR-CRED-07) |
| Unconfirmed required criterion | Kriteria wajib belum terkonfirmasi | Required criterion unconfirmed | Kriteria ICP wajib yang nilainya tidak diketahui; band maks WARM (BR-SCORE-12) |
| Job | Proses (menu "Ekspor & proses") | Jobs | Pekerjaan async |
| Business status | Buka · Tutup sementara · Tutup permanen · Tidak diketahui | Open · Temporarily closed · Permanently closed · Unknown | — |

**Label tipe peluang (Lampiran B).** Label menyatakan kondisi pada data, bukan vonis:

| Kode | Label | Kode | Label |
|---|---|---|---|
| OPP-WEB-NONE | Website tidak ditemukan | OPP-REP-LOW-RATING | Rating rendah |
| OPP-WEB-SOCIAL-ONLY | Website hanya media sosial | OPP-REP-LOW-VOLUME | Ulasan lebih sedikit dari pesaing |
| OPP-WEB-FREE-SUBDOMAIN | Website di subdomain gratis | OPP-LIST-INCOMPLETE | Listing tidak lengkap |
| OPP-EMAIL-NON-DOMAIN | Email belum pakai domain sendiri | OPP-MKT-HIGH-COMPETITION | Kompetisi area tinggi |
| OPP-DNS-NO-MX | Domain tanpa layanan email | OPP-OPS-MULTI-BRANCH | Multi-cabang (≥ 3 lokasi) |
| OPP-BOOKING-NONE | Booking online tidak ditemukan | OPP-SIG-NEW-BUSINESS (V1) | Bisnis baru |

Template rationale (BR-OPP-04): "{Kondisi} pada data sumber (versi {snapshot}). {persen}% {kategori} di {kab/kota} {memiliki atribut}." Contoh: "Tidak ada website pada data sumber (versi 2026.09). 68% klinik gigi di Kota Denpasar memiliki website."

### 7.3 Format angka, mata uang, tanggal

| Jenis | id-ID | en-US | Aturan |
|---|---|---|---|
| Bilangan | 12.400 | 12,400 | Pemisah ribuan "." |
| Desimal | 4,2 | 4.2 | Desimal ","; maks 1 desimal di UI |
| Persen | 72% | 72% | Tanpa spasi |
| Hitungan perkiraan | ±12.400 | ~12,400 | FR-SRCH-006 |
| Skor | 84 | 84 | Bilangan bulat; kontribusi 1 desimal ("+25,5") |
| Rupiah (prosa) | Rp5–15 miliar | IDR 5–15 billion | Tanpa spasi setelah "Rp"; en dash tanpa spasi |
| Rupiah (ringkas) | Rp5–15 M · Rp500 jt | IDR 5–15B · IDR 500M | Di id-ID "M" = miliar (BR-REV-02), "jt" = juta. "M" tidak pernah dipakai untuk juta. |
| Band revenue | R3 · Rp5–15 miliar (Kecil) | R3 · IDR 5–15B (Small) | Label kelas UMKM dalam kurung |
| Band karyawan | 20–99 orang | 20–99 employees | BR-REV-01 |
| Kepadatan | 1,3 per 1.000 penduduk | 1.3 per 1,000 residents | Satuan baku I-08 |
| Tanggal | 10 Okt 2026 | Oct 10, 2026 | Bulan singkatan: Jan, Feb, Mar, Apr, Mei, Jun, Jul, Agu, Sep, Okt, Nov, Des |
| Tanggal + jam | 10 Okt 2026, 14.30 WIB | Oct 10, 2026, 2:30 PM WIB | Zona waktu tampilan user (default Asia/Jakarta), selalu dengan singkatan zona |
| Waktu relatif | baru saja · 5 menit lalu · 2 jam lalu · kemarin · 2 hari lalu | just now · 5 min ago | > 30 hari → tanggal absolut; hover/`<time datetime>` memuat tanggal absolut |
| Telepon | +62 812-3456-7890 | sama | Disimpan E.164 |
| Kredit | 1 kredit · 37 kredit | 1 credit · 37 credits | Tanpa bentuk jamak di id-ID |

### 7.4 Pola pesan error

Struktur: **[Apa yang terjadi]. [Kenapa, bila diketahui]. [Apa yang bisa dilakukan].** + tombol aksi + kode referensi (8 karakter dari `correlation_id`). Tidak menyalahkan user, tidak memakai kode teknis sebagai kalimat utama.

| Situasi | Copy |
|---|---|
| Ekspor gagal | "Ekspor gagal diproses. Layanan penyimpanan tidak merespons. 35 kredit yang ditahan sudah dilepas ke saldo Anda dan tidak ada kontak yang dibuka; coba lagi dalam beberapa menit. [Coba lagi] · Kode 7F3A21C0" |
| URL webhook ditolak | "URL tidak dapat dipakai. Alamat ini mengarah ke jaringan privat atau internal. Gunakan URL HTTPS publik milik sistem Anda." |
| Uji webhook timeout | "Uji gagal: sistem Anda tidak merespons dalam 10 detik. Pastikan endpoint aktif dan dapat diakses dari internet, lalu kirim uji lagi." |
| Upload terlalu besar | "File berisi 64.210 baris. Batas per upload 50.000 baris. Pecah file menjadi dua, lalu unggah lagi." |
| Pencarian timeout | "Pencarian terlalu lama. Kombinasi filter ini berat untuk diproses. Persempit wilayah atau kurangi kata kunci, lalu coba lagi." |
| Brief gagal validasi | "Brief tidak dapat dibuat. Sistem tidak dapat memastikan setiap kalimat didukung data. Kredit AI tidak ditagih. [Coba lagi]" |
| Mode pemeliharaan (`503 MAINTENANCE_READ_ONLY`) | "Perubahan belum tersimpan. MesinProspek sedang dalam pemeliharaan sehingga hanya dapat dibaca. Coba lagi setelah pemeliharaan selesai." (banner global READ-ONLY tetap tampil) |
| Buka kontak dinonaktifkan (`503 FEATURE_DISABLED`, flag `unlock.enabled`) | "Buka kontak sedang dinonaktifkan sementara oleh tim MesinProspek. Kontak yang sudah dibuka tetap dapat dilihat, dan tidak ada kredit yang terpakai." |
| Hasil diperbarui (`409 CURSOR_STALE`) | "Urutan hasil sudah diperbarui karena skor dihitung ulang. Muat ulang dari atas untuk melihat urutan terbaru. [Muat ulang dari atas]" |
| 403 | "Anda tidak punya akses ke halaman ini. Halaman ini khusus Pemilik dan Admin. Minta akses ke Rina A. (Pemilik)." |
| Bagian gagal dimuat | "Bagian ini gagal dimuat. Bagian lain tetap dapat dipakai. [Coba lagi]" |

### 7.5 Contoh copy empty state

| Layar | Judul | Isi | CTA |
|---|---|---|---|
| SCR-03 Tugas saya | Belum ada tugas | Prospek yang ditugaskan kepada Anda akan muncul di sini. | Cari prospek |
| SCR-11 List | Belum ada list | Simpan prospek ke list untuk dikerjakan bersama tim. | Cari prospek |
| SCR-11 Segmen | Belum ada segmen | Segmen memperbarui isinya otomatis saat data berubah. Buat dari hasil pencarian atau dari peta wilayah. | Buka pencarian |
| SCR-11 Pencarian tersimpan | Belum ada pencarian tersimpan | Simpan filter yang sering Anda pakai agar bisa dijalankan dengan satu klik. | Buka pencarian |
| SCR-15 | Belum ada proses | Ekspor, upload, dan pengiriman webhook akan tercatat di sini. | — |
| SCR-16 | Belum ada upload | Unggah daftar bisnis Anda sendiri untuk melihat skor, peluang, dan kontaknya. | Unggah file |
| SCR-20 | Belum ada webhook | Kirim prospek otomatis ke CRM atau sistem Anda. | Tambah endpoint |
| SCR-05 Peluang | Belum ada peluang relevan | Kami belum menemukan kondisi yang relevan dengan penawaran "Website" untuk bisnis ini. | Lihat kehadiran digital |
| SCR-04 ZERO | Tidak ada prospek yang cocok | Filter berikut paling membatasi hasil: Rating ≥ 4,5 (hapus → 37 hasil). | Hapus filter |

### 7.6 Copy konfirmasi aksi berbayar

Selalu memuat: biaya, saldo tersedia sebelum → sesudah, apa yang didapat, sifat permanen, aturan refund (buka kontak) atau tahan → tagih/lepas (proses async, BR-CRED-07).

| Mode | Judul | Isi | Tombol |
|---|---|---|---|
| Buka 1 kontak | Buka kontak ABC Dental Clinic? | Biaya 1 kredit · Saldo 184 → 183. 4 kontak di semua cabang akan terbuka permanen untuk workspace ini. Kredit kembali otomatis bila semua kontak terbukti tidak valid dalam 30 hari. | Buka kontak · 1 kredit |
| Buka massal | Buka kontak 40 prospek? | 12 sudah dibuka (gratis), 28 akan dibuka · 28 kredit · Saldo 184 → 156. | Buka 28 kontak · 28 kredit |
| Ekspor | Ekspor 182 prospek | 145 sudah dibuka (gratis), 35 akan dibuka otomatis · 35 kredit, 2 dikecualikan. Saldo tersedia 184 → 149. 35 kredit ditahan selama proses dan hanya ditagih untuk baris yang berhasil diekspor; bila gagal, kredit dilepas dan kontak tidak dibuka. Tautan unduhan berlaku 24 jam. | Ekspor · 35 kredit |
| Kirim ke webhook | Kirim 50 prospek ke "CRM Utama"? | 50 sudah dibuka · 0 kredit. Pengiriman dicoba ulang otomatis hingga 24 jam bila gagal. | Kirim · 0 kredit |
| Perbarui data | Perbarui data ABC Dental Clinic? | Biaya 1 kredit. Kami memverifikasi ulang kontak dan menghitung ulang skor memakai versi dataset terbaru. Kredit ditahan selama proses dan hanya ditagih bila berhasil. | Perbarui · 1 kredit |
| Buat ulang brief | Buat ulang brief? | Biaya 1 kredit AI. Brief saat ini (dibuat 3 hari lalu) tetap dapat dibaca gratis. | Buat ulang · 1 kredit AI |
| Kredit tidak cukup | Kredit tidak cukup | Aksi ini butuh 35 kredit; saldo Anda 20. Pilih opsi di bawah atau minta tambahan kredit ke Pemilik workspace. | (opsi WF-06b) |
| Pesan ke Pemilik (disalin) | — | "Halo, saldo kredit MesinProspek workspace Agensi Maju tinggal 20. Saya butuh ±35 kredit untuk ekspor prospek klinik gigi Denpasar. Bisa ditambah?" | Salin pesan |

### 7.7 Copy guardrail (kunci i18n)

| Kunci | Teks id-ID |
|---|---|
| `guardrail.population_density` | Kepadatan penduduk adalah konteks pasar, bukan ukuran permintaan atau pendapatan. |
| `guardrail.business_density` | Kepadatan bisnis adalah gambaran jumlah pemain (pasokan), bukan pangsa pasar. |
| `guardrail.category_density` | Jumlah bisnis kategori ini per 1.000 penduduk berdasarkan data yang tercatat. Bisnis yang tidak tercatat tidak terhitung. |
| `guardrail.white_space` | Celah pasokan menunjukkan potensi kekurangan pasokan dibanding area sejenis. Nilai tinggi juga bisa berarti data di area ini belum lengkap. |
| `guardrail.market_opportunity` | Skor relatif terhadap area pembanding di {wilayah induk}. Bukan prediksi penjualan. |
| `guardrail.territory_score` | Peringkat untuk ICP dan penawaran aktif Anda. Berubah bila ICP berubah. |
| `guardrail.coverage_low` | Data terbatas: populasi atau jumlah bisnis tercatat di area ini terlalu kecil, atau data belum lengkap. Celah pasokan dan skor peluang area tidak dihitung. |
| `guardrail.reference_period` | Populasi: {sumber}, periode {periode}. |
| `guardrail.opportunity` | Peluang adalah hipotesis berbasis data, bukan jaminan hasil penjualan. |
| `guardrail.revenue` | Perkiraan dari kategori, jumlah cabang, dan volume ulasan. Bukan data keuangan resmi. |
| `guardrail.whatsapp` | Kemungkinan WhatsApp karena nomor seluler. Belum dikonfirmasi. |
| `guardrail.ai_brief` | Dibuat AI hanya dari data di profil ini. Periksa sumber sebelum dipakai. |

---

## 8. Aksesibilitas (WCAG 2.2 AA, NFR-A11Y-01)

### 8.1 Kriteria dan penerapan

| Kriteria | Penerapan di MesinProspek |
|---|---|
| 1.4.1 Use of Color | Semua badge P-01..P-06 = ikon berbentuk beda + teks. Choropleth: legenda berlabel, arsir untuk data terbatas, tampilan tabel setara. Identifier cocok/bertentangan di SCR-33 memakai ikon cek/silang. |
| 1.4.3 / 1.4.11 Contrast | Pasangan fg/bg token divalidasi di `02-design-system/` (teks ≥ 4,5:1, komponen & fokus ≥ 3:1), termasuk warna band dan kelas geo. |
| 1.4.10 Reflow | Layar daftar NFR-COMP-01 tanpa scroll dua arah di 320–360 px; tabel data dan peta dikecualikan tetapi punya tampilan kartu/tabel. |
| 1.4.13 Content on Hover/Focus | Tooltip dapat ditutup (Esc), dapat di-hover, tidak hilang saat pointer berpindah ke isinya. Info penting (guardrail) memakai Popover + footnote. |
| 2.1.1 Keyboard / 2.1.2 No Trap | Semua fungsi, termasuk pemilihan area peta, dapat dijalankan dengan keyboard; drawer/dialog menjebak fokus hanya selama terbuka. |
| 2.1.4 Character Key Shortcuts | Shortcut satu huruf (`/`, `j`, `k`, `s`, `b`, G/C/P/L ops) dapat dimatikan di SCR-24 dan tidak aktif saat fokus di input. |
| 2.4.3 Focus Order / 2.4.7 Focus Visible | Urutan: header → bar aksi → konten. Ring fokus 2 px `color.border.focus`. |
| 2.4.11 Focus Not Obscured | `scroll-padding` mengimbangi header, bar aksi profil, dan bar aksi massal yang sticky. |
| 2.5.7 Dragging Movements | Peta: tombol zoom/pan dan tombol panah; tidak ada fitur yang hanya bisa dengan drag (tidak ada drag-reorder). |
| 2.5.8 Target Size | Minimum 24 × 24 px; < `breakpoint.md` kontrol utama `size.control.lg` (44 px). |
| 3.2.6 Consistent Help | Tautan Bantuan di lokasi sama (sidebar bawah & menu user). |
| 3.3.7 Redundant Entry | Wizard onboarding & upload mengingat isian saat kembali; mapping upload terakhir disarankan ulang. |
| 3.3.8 Accessible Authentication | Mendukung password manager, tempel kode, login Google; tanpa tes kognitif. |
| 4.1.3 Status Messages | `aria-live="polite"`: jumlah hasil, jumlah terpilih, progres job, toast sukses. `assertive` hanya untuk error yang memblokir. |

### 8.2 Alur keyboard

| Konteks | Tombol | Aksi |
|---|---|---|
| Tabel hasil / list (roving tabindex per baris) | ↑ / ↓ | Pindah baris |
| | Spasi | Pilih/batal pilih baris |
| | Shift + ↑ / ↓ | Perluas seleksi |
| | Ctrl/Cmd + A (fokus di tabel) | Pilih semua baris yang sudah dimuat |
| | Enter | Buka profil |
| | Esc | Kosongkan seleksi / tutup bar aksi massal |
| Header kolom | Enter / Spasi | Sortir; `aria-sort` diperbarui |
| Bar aksi massal | Tab | Masuk dari baris terakhir yang dipilih; tombol pertama = "Simpan ke list" |
| Drawer & dialog | Esc | Tutup; fokus kembali ke pemicu |
| | Tab / Shift+Tab | Berputar di dalam drawer/dialog |
| Dialog biaya | (fokus awal) | Tombol "Batal"; ringkasan biaya dibacakan via `aria-describedby` |
| Peta | Tab | Toolbar → kanvas peta → daftar area |
| | Panah / + / − (fokus di kanvas) | Geser / perbesar / perkecil |
| | ↑ / ↓ di daftar area | Pilih area, peta menyorot area yang sama |
| Profil | `j` / `k` | Prospek berikutnya / sebelumnya (dapat dimatikan) |

### 8.3 Fokus drawer & dialog

- Buka: fokus ke judul drawer (evidence) atau tombol "Batal" (dialog biaya); `aria-labelledby` ke judul.
- Tutup (Esc, tombol, klik overlay): fokus kembali ke elemen pemicu; bila pemicu hilang (mis. baris terhapus) → ke heading bagian terdekat.
- Drawer evidence di < `breakpoint.md` = sheet layar penuh dengan tombol "Tutup" di atas.
- Toast tidak mengambil fokus; toast error tidak hilang otomatis.

### 8.4 Label pembaca layar

| Komponen | Contoh `aria-label` / teks tersembunyi |
|---|---|
| Badge skor | "Skor prioritas 84 dari 100, band HOT" |
| Badge skor dibatasi | "Skor prioritas 81 dari 100, band WARM, dibatasi karena data terbatas" |
| Badge belum dinilai | "Belum dinilai: kecocokan ICP tidak diketahui" |
| Chip kriteria wajib | "Kriteria wajib belum terkonfirmasi: wilayah. Band dibatasi maksimal WARM." |
| Label prospek privat | "Dari upload Anda: leads-okt.xlsx" |
| Keyakinan | "Keyakinan sedang, 66 dari 100" |
| Kemutakhiran | "Mutakhir, terakhir diamati 28 September 2026" |
| Verifikasi | "Status verifikasi: Kemungkinan valid" |
| Kontak ter-mask | "Telepon tersembunyi, berakhiran 7890. Buka kontak untuk melihat." |
| Ikon kontak di hasil | "Telepon tersedia, kemungkinan valid. Email tidak ditemukan. Kemungkinan WhatsApp." |
| Bar kontribusi | `aria-hidden`; nilai dibaca dari sel tabel |
| Area peta | "Sesetan, skor peluang pasar 78, kelas 7 dari 7, cakupan memadai" |
| Sitasi brief | "Sumber 2: rating dan jumlah ulasan" |

### 8.5 Gerak & preferensi

- `prefers-reduced-motion`: tanpa animasi fly-to peta (lompat langsung), skeleton tanpa shimmer, transisi drawer memakai `motion.duration.instant`. Default: `motion.duration.normal` + `motion.easing.enter/exit` untuk drawer, `motion.duration.fast` untuk tooltip.
- Tidak ada konten yang berkedip > 3 kali per detik. Progres memakai angka, tidak hanya animasi.
- Atribut `lang="id"` / `lang="en"` mengikuti bahasa UI; brief berbahasa lain diberi `lang` sendiri.

---

## 9. Analytics event taxonomy

**Aturan penamaan:** `<objek>_<kata kerja lampau>` snake_case; properti snake_case. Event berbayar, assignment, ekspor, dan webhook **otoritatif dari server** (sumber yang sama dengan `activity`/usage); event klien dipakai untuk funnel dan diagnostik UX.

**Properti umum (semua event):** `event_id` (UUID, untuk dedupe), `occurred_at` (UTC), `workspace_id`, `user_id` (pseudonim), `role`, `plan`, `locale`, `app_version`, `session_id`, `screen` (SCR-nn), `viewport_class` (sm/md/lg/xl), dan bila relevan `icp_id` + `icp_version`.

**Privasi (BR-PRIV-01):** tidak pernah mengirim nilai kontak, nama orang, nama bisnis, atau teks query mentah ke analytics produk; query hanya `has_query` + `query_length` (OQ-UX-29). Identitas bisnis memakai `company_id`/`establishment_id`.

| Event | Pemicu | Properti kunci | Sumber | Metrik PRD §5 |
|---|---|---|---|---|
| `signup_completed` | Akun dibuat | `method` (password/google) | server | Time-to-first-result (t0) |
| `email_verified` | Verifikasi sukses | — | server | Time-to-first-result |
| `workspace_created` | Workspace baru | — | server | Activation rate (penyebut) |
| `onboarding_step_completed` | Tiap langkah SCR-02 | `step`, `duration_ms` | klien | Time-to-first-result (diagnostik) |
| `offering_created` | Penawaran disimpan | `offering_tags[]`, `preset` | server | Activation (diagnostik) |
| `icp_saved` | ICP disimpan | `icp_version`, `criteria_count`, `required_count`, `candidate_estimate`, `is_first` | server | Activation rate (pembilang, bagian 1) |
| `icp_switched` | ICP aktif diganti | `from_icp_id`, `to_icp_id` | klien | — |
| `scoring_completed` | Job skor selesai | `candidates`, `duration_s`, `trigger` (create/edit) | server | NFR-PERF-05, Time-to-first-result |
| `search_performed` | Query/filter/sortir dijalankan | `search_session_id`, `source` (manual/saved/area/segment/onboarding), `has_query`, `query_length`, `filter_keys[]`, `sort`, `result_count`, `count_is_estimate`, `latency_ms` | klien | Search-to-save rate (penyebut) |
| `search_results_rendered` | Hasil dirender (muat awal atau "Muat lagi") | `search_session_id`, `rows_rendered`, `load_index` (0 = muat awal), `private_rows` | klien | Profile open rate (penyebut) |
| `first_result_viewed` | Baris hasil pertama dengan ICP aktif dirender (sekali per workspace) | `seconds_since_signup` | klien | **Time-to-first-result** |
| `zero_result_shown` | ZERO tampil | `filter_keys[]`, `suggestion_types[]` | klien | — |
| `zero_result_suggestion_applied` | Saran dipakai | `suggestion_type`, `new_result_count` | klien | — |
| `saved_search_created` | Pencarian disimpan | `search_session_id`, `filter_keys[]` | server | Search-to-save rate |
| `saved_search_run` | Pencarian tersimpan dijalankan | `saved_search_id` | klien | Week-4 retention (diagnostik) |
| `prospect_profile_opened` | Profil dibuka | `company_id`, `establishment_id`, `source`, `rank_position`, `band`, `coverage_limited`, `search_session_id` | klien | **Profile open rate** (pembilang) |
| `score_explanation_viewed` | Panel "Kenapa prospek ini?" dibuka/expand | `company_id`, `band`, `coverage` | klien | RG-1 (diagnostik) |
| `evidence_drawer_opened` | Drawer bukti dibuka | `claim_field`, `claim_type`, `source` (profile/brief/opportunity/contact) | klien | RG-2, Evidence coverage (diagnostik) |
| `prospects_bulk_selected` | Seleksi massal | `count`, `scope` (loaded/all) | klien | — |
| `prospect_unlocked` | Debit buka kontak sukses | `company_id`, `credit_cost`, `source` (profile/bulk/export_auto/webhook_auto) | server | Gross margin / enriched lead |
| `contact_action_clicked` | Salin/Telepon/WhatsApp/Email | `channel`, `contact_type`, `verification_status` | klien | — (insight kanal) |
| `list_created` | List dibuat | `source` (search/upload/segment) | server | **Activation rate**, Search-to-save |
| `prospects_added_to_list` | Prospek masuk list | `list_id`, `count`, `search_session_id` | server | **Search-to-save rate** |
| `segment_created` | Segmen dibuat | `source` (search/area/territory), `filter_keys[]` | server | **Activation rate**, Search-to-save |
| `segment_opened` | Detail segmen dibuka | `segment_id`, `delta_since_last_visit` | klien | Week-4 retention (diagnostik) |
| `tag_applied` / `note_added` | Tag/catatan | `count` | server | — |
| `prospects_assigned` | Penugasan | `count`, `source`, `assignee_role` | server | **North star** (assign) |
| `work_status_changed` | Status kerja diubah | `from`, `to` | server | — |
| `export_previewed` | Pratinjau biaya tampil | `row_count`, `unlocked_count`, `to_unlock_count`, `excluded_count`, `credit_cost`, `format` | klien | — |
| `insufficient_credit_shown` | NO-CREDIT tampil | `operation`, `required`, `balance`, `option_chosen` | klien | Gross margin (insight pricing) |
| `export_requested` | Ekspor dikonfirmasi | `job_id`, `row_count`, `credit_cost`, `format` | server | — |
| `export_completed` | Job ekspor selesai | `job_id`, `row_count`, `unique_prospects`, `duration_s` | server | **North star** (ekspor) |
| `export_failed` | Job ekspor gagal | `job_id`, `error_class`, `credit_released` | server | — |
| `export_downloaded` | File diunduh | `job_id` | klien | — |
| `webhook_endpoint_created` | Endpoint dibuat | `events[]` | server | — |
| `webhook_test_sent` | Uji dikirim | `result` (ok/timeout/http_4xx/http_5xx/tls/ssrf_blocked), `latency_ms` | server | — |
| `webhook_send_requested` | "Kirim ke webhook" dikonfirmasi | `endpoint_id`, `prospect_count`, `credit_cost` | server | — |
| `webhook_delivery_completed` | Status akhir pengiriman | `endpoint_id`, `status` (delivered/failed/dlq), `attempts` | server | **Webhook success**, **North star** (delivered) |
| `ai_brief_requested` | Tombol brief diklik | `company_id`, `language`, `cached`, `regenerate` | klien | AI brief grounding (penyebut) |
| `ai_brief_generated` | Brief tampil | `latency_ms`, `grounding_passed`, `regenerated_once`, `prompt_version` | server | **AI brief grounding** |
| `ai_brief_failed` | Brief gagal | `reason` (grounding/timeout/kill_switch/budget/no_credit), `credit_released` | server | AI brief grounding |
| `ai_brief_copied` | Brief disalin | — | klien | — |
| `ai_brief_feedback_submitted` | Feedback brief | `rating` (up/down), `reason` | server | — |
| `sales_angle_generated` | Sales angle tampil | `latency_ms` | server | — |
| `opportunity_feedback_submitted` | Relevan / Tidak relevan | `opportunity_type`, `verdict`, `reason`, `offering_tags[]` | server | **Opportunity acceptance** |
| `score_feedback_submitted` | Terlalu tinggi / rendah | `verdict`, `band`, `coverage` | server | Intelligence Health (FR-OPS-003) |
| `prospect_refresh_requested` / `prospect_refresh_completed` | Refresh | `changed_fields_count`, `credit_released` | server | — (MVP Pack "Refresh usage") |
| `data_issue_reported` | Laporan data dikirim | `claim_field`, `issue_type` | server | — (Data Health) |
| `upload_started` / `upload_completed` | Upload | `row_count`, `matched`, `possible`, `new`, `rejected`, `match_rate` | server | — |
| `area_map_viewed` | SCR-08 dibuka / metrik diganti | `region_code`, `level`, `metric`, `view` | klien | — |
| `area_selected` | Area dipilih | `area_code`, `level`, `coverage` | klien | — |
| `territory_ranking_viewed` | SCR-09 dibuka | `region_code`, `level`, `ranked_count`, `unranked_count` | klien | — |
| `territory_row_expanded` | Breakdown dibuka | `area_code` | klien | RG-1 (diagnostik) |
| `area_drilldown_clicked` | Area → prospek | `area_code`, `source` (map/ranking/area_detail) | klien | — |
| `geo_guardrail_viewed` | Popover guardrail dibuka | `metric` | klien | FR-GEO-009 (diagnostik) |
| `member_invited` / `invite_accepted` | Undangan | `role` | server | Week-4 retention (diagnostik tim) |
| `workspace_switched` | Ganti workspace | — | klien | — |
| `app_session_started` | Sesi aplikasi dimulai | — | klien | **Week-4 retention** |
| `permission_denied_shown` | DENIED tampil | `permission` | klien | — |
| `feature_disabled_shown` | FLAG-OFF / UPSELL tampil | `feature`, `reason` (kill_switch/budget/plan) | klien | — |
| `fair_use_limit_reached` | Kuota profil 80% / 100% | `percent` | server | — |
| `filter_applied` | Filter ditambah/diubah/dihapus | `filter_key`, `action` (add/change/remove), `search_session_id` | klien | — (diagnostik relevansi filter) |
| `upload_mapping_confirmed` | Langkah mapping upload selesai | `mapped_fields[]`, `auto_mapped_ratio` | klien | — |
| `upload_possible_match_decided` | Baris "Mungkin cocok" ditautkan / dijadikan prospek privat | `upload_id`, `decision` (link/private), `count`, `bulk` | server | — (kualitas linking upload) |
| `webhook_redelivery_requested` | "Kirim ulang" dari log/DLQ | `endpoint_id`, `count` | server | Webhook success (diagnostik) |
| `job_retry_clicked` | "Coba lagi" pada job gagal | `job_type`, `error_class` | klien | — |
| `data_issue_report_started` | Dialog laporkan data dibuka | `claim_field`, `source` | klien | — |
| `login_succeeded` | Login berhasil | `method`, `mfa_used` | server | Week-4 retention (diagnostik) |
| `member_role_changed` / `member_removed` | Perubahan anggota | `from_role`, `to_role` | server | — (audit dicatat terpisah, FR-AUD-001) |
| `home_viewed`, `tasks_viewed`, `area_detail_viewed`, `usage_viewed`, `audit_log_viewed` | Layar terkait dibuka | `screen` | klien | Week-4 retention (diagnostik) |

Catatan: *Verified contact rate*, *Duplicate merge precision/recall*, dan *Evidence coverage* adalah metrik data (dihitung dari tabel data dan sampling QA), bukan event analytics. *North star* = prospek unik dari `export_completed` + `webhook_delivery_completed` (delivered) + `prospects_assigned` per workspace per minggu, dihitung dari `activity` server.

---

## 10. Pertanyaan terbuka & asumsi UX

Asumsi berlaku sebagai default desain sampai diputuskan. Kolom *Pemilik keputusan* menunjuk siapa yang perlu menjawab.

| ID | Pertanyaan / temuan | Asumsi yang dipakai | Pemilik keputusan |
|---|---|---|---|
| OQ-UX-01 | **Ditutup.** Time-to-first-result median ≤ 5 menit (OBJ-1, FR-ICP-005) vs hitung ulang skor ≤ 10 menit untuk ≤ 200.000 kandidat. | NFR-PERF-05 kini menetapkan skor progresif: halaman pertama (≤ 50 kandidat teratas hasil pre-ranking) ber-skor ≤ 30 detik, seluruh kandidat ≤ 10 menit; didukung `04` §7.9. FLOW-01 #6 memakai target 30 detik. | — |
| OQ-UX-02 | **Inkonsistensi:** D-08 menyebut mask untuk "kontak person/email", BR-CRED-02 memakai contoh telepon. | Mengikuti BR-CRED-02: **semua** contact point ter-mask sampai dibuka. | Product |
| OQ-UX-03 | ICP aktif per user atau per workspace? | Per user per workspace; tercermin di URL. | Product |
| OQ-UX-04 | Matriks RBAC tidak mencakup **upload customer**. | Sama dengan "Kelola list" (Member ke atas). | Product |
| OQ-UX-05 | Apakah Viewer dapat melihat list/segmen yang dibagikan? | Ya, baca saja. | Product |
| OQ-UX-06 | Apakah Viewer dapat membaca AI brief yang sudah dibuat? | Ya, tanpa biaya; tidak dapat membuat. | Product |
| OQ-UX-07 | Relasi ICP ↔ Penawaran tidak eksplisit, padahal konteks skor = (workspace, ICP, Offering) (BR-SCORE-01). | Setiap ICP terhubung ke tepat satu penawaran; switcher menampilkan keduanya. | Product |
| OQ-UX-08 | Perilaku fair-use profil di 100% tidak dirinci. | Blokir profil baru; profil yang sudah dibuka kontaknya atau ditugaskan tetap terlihat. | Product, Pricing |
| OQ-UX-09 | Refresh berulang dalam waktu singkat. | Peringatan bila < 24 jam dan tidak ada snapshot baru; tetap boleh dilanjutkan. | Product |
| OQ-UX-10 | Nasib tugas milik anggota yang dikeluarkan. | Tetap ada, ditandai "Penerima tidak aktif", dapat ditugaskan ulang. | Product |
| OQ-UX-11 | Kontak dari **upload workspace sendiri**: perlu dibuka dengan kredit? | Tidak; tampil tanpa mask untuk workspace pengunggah. | Product, Pricing |
| OQ-UX-12 | **Ditutup.** Role internal ops belum didefinisikan (G-12 hanya tenant). | PRD §8.1.2 menetapkan Ops Engineer, Data Steward, Privacy & Billing, Ops Admin (MFA wajib, boleh > 1 peran); diterapkan di §2.5 dan §4.2 (SCR-30..40). Sisa asumsi di OQ-UX-34. | — |
| OQ-UX-13 | SLA permintaan subjek data dan aturan jeda SLA. | Nilai TBD konsultan hukum; UI mendukung timer dan status "Menunggu pemohon". | Legal |
| OQ-UX-14 | Nama orang sebelum unlock tidak diatur PRD. | Role selalu tampil; nama di-mask (`d** A***`) sampai dibuka. | Product, Legal |
| OQ-UX-15 | Perlukah dialog untuk setiap buka 1 kontak (SDR membuka 30–60/hari)? | Dialog pada pertama kali; user dapat memilih "Jangan tanya lagi" (tombol tetap menampilkan biaya). Massal & ekspor selalu dialog. | Product |
| OQ-UX-16 | Bahasa label band. | HOT/WARM/COLD tetap Inggris; UNSCORED → "Belum dinilai". Divalidasi pada usability test RG-1. | Design, Product |
| OQ-UX-17 | **Istilah ganda:** Lampiran E "ICP aktif" (batas jumlah) vs FR-ICP-006 "satu ICP aktif" (konteks). | UI memakai "ICP tersimpan" untuk batas plan dan "ICP aktif" untuk konteks. PRD sebaiknya diselaraskan. | Product |
| OQ-UX-18 | Batas wilayah bersifat opsional (FR-DATA-008 "bila ada"), padahal FR-GEO-002 mensyaratkan choropleth. | Fallback otomatis ke tampilan tabel bila poligon tidak tersedia. | Data owner (Q-04) |
| OQ-UX-19 | Nama dataset di evidence drawer mungkin dibatasi lisensi (Q-05). | Nama tampilan dikonfigurasi ops per snapshot; fallback "Dataset bisnis MesinProspek". | Founder (Q-05) |
| OQ-UX-20 | Konteks workspace di URL. | `/app/...` + workspace dari sesi; tautan sistem membawa `ws=`; interstitial bila berbeda. Alternatif: segmen path `/w/[slug]/...`. | Tech lead |
| OQ-UX-21 | Notifikasi "ekspor siap" di MVP (pusat notifikasi = V1). | In-app saja: toast, badge job di header, kartu Beranda. Tanpa email. | Product |
| OQ-UX-22 | **Ditutup** (perilaku). Kriteria **wajib** ICP yang nilainya Unknown: masuk hasil atau tidak? | BR-SCORE-12 / `04` §7.4: masuk hasil, kriteria dikeluarkan dari Kecocokan ICP, chip "Kriteria wajib belum terkonfirmasi", band maks WARM; kategori "Lainnya" dengan keyakinan < 50 dihitung Unknown. Filter "Hanya yang semua kriteria wajibnya terkonfirmasi" **tidak** ada di FR-SRCH-002 → tidak dibangun di MVP; dicatat sebagai **usulan V1**. | Product (usulan V1) |
| OQ-UX-23 | Compare prospek (V1, PRD §6) belum punya FR ID. | Dicatat sebagai SCR-50 tanpa FR. | Product |
| OQ-UX-24 | **Ditutup** (mekanisme). Koreksi data global dari laporan user. | FR-OPS-009 + SCR-36 "Koreksi nilai" (evidence `ops_correction`) + FLOW-09 #4. Sisa: SLA tinjauan laporan belum diatur PRD; copy "5 hari kerja" tetap placeholder. | Product, Ops (SLA) |
| OQ-UX-25 | Kredit awal workspace baru (beta) tidak diatur. | Onboarding tetap menjelaskan biaya; bila saldo 0 tampil "Kredit belum tersedia — hubungi tim kami". | Founder, Pricing |
| OQ-UX-26 | "Pembaruan segmen" di Beranda butuh riwayat jumlah per segmen. | Dicatat untuk `04` (simpan jumlah per kunjungan user). | Tech lead |
| OQ-UX-27 | Basemap provider. | TBD (Q-05); desain tidak bergantung vendor. | Founder |
| OQ-UX-28 | **Ditutup.** Apakah nilai ter-suppress yang juga ada di upload milik tenant tetap tampil bagi pengunggah? | BR-PRIV-02 / `04` §15.7: suppression global menyembunyikan nilai **global layer** di semua workspace; nilai dari upload tenant **tidak** disembunyikan otomatis (tenant = pengendali, MSP = prosesor). Permintaan subjek data atas data upload diteruskan ke tenant (FLOW-22 #3b) dan dieksekusi atas instruksinya. | — |
| OQ-UX-29 | Teks query di analytics produk. | Tidak dikirim; hanya `has_query` + `query_length`. Log query server diatur retensinya di `04`. | Product, Security |
| OQ-UX-30 | Apakah "Member boleh ekspor" juga mengatur "Kirim ke webhook"? | Ya, mengikuti baris RBAC "Export / kirim ke webhook". | Product |
| OQ-UX-31 | Member membelanjakan kredit tetapi RBAC "Lihat usage" = —. | Member melihat saldo di header dan dialog biaya (dibutuhkan untuk persetujuan informasi), tidak melihat SCR-21. | Product |
| OQ-UX-32 | **Ditutup.** Langkah "tinjau sebelum terbitkan" snapshot. | PRD §8.1.2 + `06` §10.2: Data Steward meninjau laporan validasi di SCR-34; publish/re-publish hanya lewat job manual Maintainer di pipeline tag `ds-*` setelah QG-34. Konsol tidak punya tombol Terbitkan (FLOW-20). | — |
| OQ-UX-33 | Apakah akses support terlihat oleh tenant? | Ya, entri "Tim MesinProspek (support)" tampil di SCR-22 tenant. | Product, Security |
| OQ-UX-34 | **Ditutup.** PRD §8.1.2 dan `04` §11.5 tidak mengatur siapa yang boleh membaca log audit internal (SCR-39). → PRD §8.1.2 kini menetapkan baca untuk semua peran internal. | Baca untuk semua peran internal (investigasi on-call dan audit silang); tanpa edit/hapus. | Tech lead, Security |

---

## Lampiran A — Traceability FR → layar/flow

| Area FR | ID | Layar | Flow |
|---|---|---|---|
| AUTH | FR-AUTH-001..006 | SCR-01, SCR-18, SCR-19, SCR-24 | FLOW-01, FLOW-10 |
| AUTH | PRD §8.1.2 peran internal | §2.5, SCR-30..40 | FLOW-20..24 |
| DATA | FR-DATA-001, 008 | SCR-34 | FLOW-20 |
| DATA | FR-DATA-002, 006 | SCR-06 | — |
| DATA | FR-DATA-004 | SCR-33 | FLOW-21 |
| DATA | FR-DATA-009, 010, 011 | SCR-05 (F2, F4), SCR-30 | — |
| DATA | FR-DATA-012 | SCR-16, SCR-15, SCR-04 (prospek privat), SCR-05 | FLOW-05 |
| DATA | FR-DATA-014 | SCR-23, SCR-36 | FLOW-09 |
| ICP | FR-ICP-001..006 | SCR-02, SCR-17, header | FLOW-01, FLOW-11 |
| SRCH | FR-SRCH-001..008 | SCR-04, SCR-11 | FLOW-02 |
| PROF | FR-PROF-001..010 | SCR-05, SCR-06 | FLOW-02, FLOW-04, FLOW-08 |
| INTL | FR-INTL-007, 008, 010, 011 | SCR-04, SCR-05 (P-05, P-07, P-11), SCR-32 | FLOW-02, FLOW-11 |
| GEO | FR-GEO-001..005, 009 | SCR-08, SCR-09, SCR-10, SCR-05 (F2) | FLOW-03 |
| SEG | FR-SEG-001..006 | SCR-04, SCR-05, SCR-11, SCR-12, SCR-13 | FLOW-02, FLOW-03 |
| EXP | FR-EXP-001..006 | SCR-14, SCR-15, SCR-20 | FLOW-04, FLOW-06, FLOW-07 |
| AI | FR-AI-001..007 | SCR-07, SCR-38 | FLOW-04 |
| MON | FR-MON-001 | SCR-05, SCR-14 | FLOW-08 |
| BILL | FR-BILL-001..007 | SCR-14, SCR-21, header, SCR-37 | FLOW-06, FLOW-24 |
| AUD | FR-AUD-001..005 | SCR-22, SCR-25, SCR-35, SCR-39, SCR-18 | FLOW-22 |
| OPS | FR-OPS-001..008 | SCR-30..38, SCR-40 | FLOW-20..24 |
| OPS | FR-OPS-009 | SCR-36 | FLOW-09 |
| V1/V1.5/V2 | FR-SRCH-009/010, FR-GEO-006..008, FR-PROF-011, FR-MON-002..004, FR-EXP-007, FR-BILL-008, FR-AUTH-007, FR-RPT-001 | SCR-50..59 | — |
