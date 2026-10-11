# 02 — Design System & Design Tokens

| Field | Value |
|---|---|
| Dokumen | Design system & design tokens |
| Versi | 0.2 · tokens `0.2.0` (brand Forest & Mint) |
| Tanggal | 2026-10-10 |
| Basis | `01-prd.md` v0.2 (BR-TRUST, BR-NORM-01, BR-SCORE, NFR-06, NFR-07, FR-INTL-03) |
| Audience | Design, Frontend, QA |
| File | `tokens/design-tokens.json` (sumber kebenaran, DTCG) · `tokens/tokens.css` (hasil build) · `tokens/build.mjs` (build + validasi kontras, Node tanpa dependency) |

> **Brand: Forest & Mint (D-17, Accepted).** Forest `#064E3B` untuk tombol primer dan teks brand; Mint `#10B981` hanya untuk aksen non-teks (indikator nav aktif, progres) karena kontrasnya di atas putih < 3:1; Mint-light `#D1FAE5` untuk latar chip/nav aktif. Heading memakai **Plus Jakarta Sans**, body memakai **Inter**. Ketiga warna = skala `emerald` 900/500/100.

---

## 1. Prinsip visual

| Prinsip | Konsekuensi desain |
|---|---|
| **Trust terlihat** | Jenis claim, confidence, freshness, dan verifikasi punya token dan komponen sendiri, bukan teks abu-abu kecil. |
| **Tenang meski padat data** | Netral (slate) mendominasi. Warna dipakai untuk makna, tidak untuk dekorasi. Satu aksen brand. |
| **Warna tidak pernah sendirian** | Setiap status selalu tampil sebagai ikon + label teks. Jenis claim juga memakai gaya border (solid/dashed/dotted) sebagai encoding sekunder (NFR-06). |
| **Unknown itu sah** | Ada token khusus untuk Unknown/UNSCORED/no-data, sehingga "Tidak diketahui" tampil konsisten (BR-TRUST-03). |
| **Angka harus rata** | Kolom angka memakai `text.numeric` (tabular-nums) dan rata kanan. |
| **Satu sumber, banyak mode** | Light & dark dari token yang sama. Mode gelap adalah pilihan desain per token, bukan inversi otomatis. |

---

## 2. Arsitektur token

```
base.*  (primitive: palet hue, palet data-viz)      ← tidak dipakai langsung di komponen
   │ alias
   ▼
color.* / text.* / space.* / radius.* / …  (semantic) ← dipakai komponen & CSS
   │ CSS variables --msp-*
   ▼
komponen (packages/ui)                                ← tidak boleh hardcode hex/px
```

### 2.1 Format file
- Format **DTCG** (Design Tokens Community Group): setiap token memiliki `$value`, `$type`, dan opsional `$description`. Alias ditulis `{path.ke.token}`.
- Warna memakai hex string (`#rrggbb` / `#rrggbbaa`). Dimensi memakai string `px`.
- **Mode gelap**: `$value` = mode terang. Override gelap ditaruh di `$extensions["com.mesinprospek.mode"].dark`. Alias di-resolve per mode, jadi alias ke token semantik otomatis mengikuti mode.
- Line height disimpan sebagai dimensi absolut (px), bukan rasio, agar tinggi baris tabel presisi.

### 2.2 Penamaan → CSS variable
`color.score.hot.solid` → `--msp-color-score-hot-solid`. Token tipografi komposit menghasilkan beberapa variabel (`--msp-text-body-font-size`, dst.) ditambah kelas utilitas `.msp-text-body`.

Pemilihan tema:

```html
<html data-theme="light">   <!-- paksa terang -->
<html data-theme="dark">    <!-- paksa gelap -->
<html>                      <!-- ikut preferensi OS (prefers-color-scheme) -->
```

### 2.3 Build & validasi

```bash
node docs/02-design-system/tokens/build.mjs           # tulis tokens.css + cek kontras
node docs/02-design-system/tokens/build.mjs --check   # CI: gagal bila CSS basi atau kontras gagal
node docs/02-design-system/tokens/build.mjs --report  # tabel kontras lengkap (markdown)
```

`build.mjs` adalah satu-satunya build token (Node, tanpa dependency). `packages/design-tokens` memakai output-nya langsung (CSS variables + preset Tailwind). CI menjalankan `--check` sebagai quality gate (lihat `06-ci-cd.md`).

### 2.4 Versioning token
| Perubahan | Versi |
|---|---|
| Hapus/ganti nama token semantik | **MAJOR** |
| Tambah token, tambah mode | MINOR |
| Ubah nilai (warna, spasi) tanpa mengubah makna | PATCH. Untuk perubahan warna semantik, review visual wajib. |

---

## 3. Referensi token warna

### 3.1 Fondasi

| Token | Light | Dark | Pemakaian |
|---|---|---|---|
| `color.bg.canvas` | slate 50 `#f8fafc` | slate 950 `#020617` | Latar halaman |
| `color.bg.surface` | white | slate 900 `#0f172a` | Kartu, tabel, panel |
| `color.bg.surface-raised` | white | slate 800 | Popover, dropdown, dialog |
| `color.bg.surface-sunken` | slate 100 | slate 950 | Area inset |
| `color.bg.hover` / `selected` | slate 100 / emerald 50 | slate 800 / emerald 950 | Baris tabel |
| `color.bg.overlay` | slate 900 @50% | slate 950 @70% | Scrim dialog/drawer |
| `color.bg.inverse` | slate 900 | slate 50 | Tooltip, toast |
| `color.fg.default` / `muted` / `subtle` | slate 900 / 600 / 500 | slate 50 / 300 / 400 | Teks |
| `color.fg.link` | emerald 700 | emerald 300 | Tautan |
| `color.border.default` / `subtle` / `strong` | slate 200 / 100 / 400 | slate 700 / 800 / 600 | Pemisah |
| `color.border.input` | slate 500 | slate 500 | Batas kontrol form (≥ 3:1, WCAG 1.4.11) |
| `color.border.focus` | emerald 600 | emerald 400 | Focus ring 2px + offset 2px |
| `color.accent.default` / `hover` / `active` | forest `#064E3B` / emerald 800 / 950 | emerald 700 / 800 / 900 | Tombol primer (teks putih) |
| `color.accent.subtle` / `fg` | mint-light `#D1FAE5` / forest | forest / emerald 300 | Chip & nav aktif, teks brand |
| `color.accent.highlight` | mint `#10B981` | mint | Aksen dekoratif non-teks |
| `color.status.{success,warning,danger,info}.{fg,bg,border}` | green/amber/red/blue | — | Alert, toast, validasi form |

### 3.2 Token domain (trust & intelligence)

Semua token domain memiliki `fg`, `bg`, `border`. Label UI id-ID final ditetapkan di `03-ux-spec.md` (glosarium). Kolom ikon berisi saran nama dari set **Lucide** (lisensi ISC).

**Score band** (BR-SCORE-02). Token tambahan `solid` dan `on-solid` dipakai untuk badge skor di baris hasil.

| Token | Makna | Gaya badge | Ikon |
|---|---|---|---|
| `color.score.hot` | HOT ≥ 80 | solid orange 700 + teks putih | `flame` |
| `color.score.warm` | WARM 60–79 | solid amber 400 + teks amber 950 | `sun` |
| `color.score.cold` | COLD < 60 | solid sky 700 + teks putih | `snowflake` |
| `color.score.unscored` | UNSCORED ("Belum dinilai") | solid slate 200 + teks slate 700 | `circle-minus` |

**Verifikasi contact point** (F1: telepon "Format valid" → `likely-valid`, tidak valid → `invalid`, kosong → `unknown`; VERIFIED dan RISKY dipakai mulai F2)

| Token | Status | Gaya | Ikon |
|---|---|---|---|
| `color.verification.verified` | VERIFIED | **solid** green 700, teks putih | `badge-check` |
| `color.verification.likely-valid` | LIKELY_VALID | subtle green | `check` |
| `color.verification.risky` | RISKY | subtle amber | `triangle-alert` |
| `color.verification.invalid` | INVALID | subtle red | `circle-x` |
| `color.verification.unknown` | UNKNOWN | subtle slate | `circle-help` |

VERIFIED dan LIKELY_VALID memakai hue yang sama. Keduanya dibedakan lewat **bobot visual (solid vs subtle) + ikon + label**, karena perbedaannya adalah tingkat kepastian, bukan kategori.

**Jenis claim** (BR-TRUST-01)

| Token | Claim | Label id-ID | Border (`border-style.claim.*`) | Ikon |
|---|---|---|---|---|
| `color.claim.observed` | OBSERVED | Teramati | solid | `eye` |
| `color.claim.derived` | DERIVED | Turunan | solid | `calculator` |
| `color.claim.inferred` | INFERRED | Perkiraan | **dashed** | `sparkles` |
| `color.claim.unknown` | UNKNOWN | Tidak diketahui | **dotted** | `circle-help` |

**Freshness** (BR-TRUST-02) dan **confidence** (dipakai mulai F2) adalah alias ke `status.*`, sehingga palet tetap kecil:

| Token | Alias ke | Ikon |
|---|---|---|
| `color.freshness.fresh` / `aging` / `stale` | success / warning / danger | `clock` / `clock-alert` / `history` |
| `color.confidence.high` / `medium` / `low` *(F2)* | success / warning / danger | meter 3 bar (terisi 3/2/1) |

> Karena freshness, confidence, dan verifikasi berbagi hue, komponen **wajib** menampilkan label teks ("Tinggi", "Usang", dst.). Dua badge berwarna sama dalam satu baris dibedakan oleh label dan ikonnya.

### 3.3 Peta & chart (data-viz)

Palet data-viz memakai palet referensi skill dataviz yang **sudah divalidasi ulang terhadap surface aplikasi** (`#ffffff` terang, `#0f172a` gelap):

| Uji | Hasil |
|---|---|
| Categorical 8 slot, light, adjacent | PASS — CVD ΔE terburuk 9,1; normal-vision 19,6 |
| Categorical 8 slot, dark, adjacent | PASS — CVD ΔE terburuk 8,4; normal-vision 19,3; semua ≥ 3:1 |
| Slot 1–3, all-pairs (peta/scatter), light & dark | PASS — CVD 9,2 / 9,4; normal-vision 24,0 / 20,9 |
| `geo.competition` ordinal, light & dark | PASS — monoton, ΔL ≥ 0,06, ujung terang ≥ 2:1 |

| Token | Isi | Aturan |
|---|---|---|
| `color.chart.categorical.1..8` *(F1 memakai slot 1–3: pola keramaian dan perbandingan; slot 4–8 untuk F2)* | blue, orange, aqua, yellow, magenta, green, violet, red | **Urutan tetap**, diberikan berurutan, tidak pernah di-cycle. Seri ke-9 dilebur ke "Lainnya". Di mode terang, slot 3, 4, dan 5 berada di bawah 3:1 terhadap surface, sehingga **wajib** ada label langsung atau tampilan tabel. |
| — (batas) | Chart all-pairs (peta, scatter, small multiples) | Maksimal **3 seri**. Lebih dari itu: lebur ke "Lainnya" atau pecah menjadi facet. |
| `color.geo.sequential.1..7` | Satu hue biru, terang → gelap | Untuk metrik area kontinu (densitas, skor). Mode gelap **membalik anchor**: nilai kecil mendekati warna surface. Tidak pernah rainbow. |
| `color.geo.competition.{low,medium,high}` | Ramp ordinal oranye | Legend selalu menyertakan label Rendah/Sedang/Tinggi. |
| `color.geo.no-data` | Slate + **pola arsir 45°** | Untuk coverage LOW/"Data terbatas" (FR-INTL-03). Warna polos tanpa arsir dilarang. |
| `color.geo.boundary` / `selected` | Garis batas / area terpilih | — |
| `color.chart.status.{good,warning,serious,critical}` | Status data-viz tetap | Tidak ikut tema, tidak dipakai sebagai "seri ke-n". Selalu ikon + label. |
| `color.chart.grid` / `axis` / `label` | Chrome chart | Grid samar, label memakai warna teks, bukan warna seri. |

**Chart F1 (market intelligence):**

| Chart | Bentuk | Aturan |
|---|---|---|
| Pola keramaian satu tempat | Bar per jam, satu seri `categorical.1` | Jam inti diberi penanda di sumbu X (bukan warna kedua). Sumbu Y 0–100 berlabel "relatif terhadap jam tersibuk" |
| Perbandingan pola (analisis lokasi) | Garis per jam, maksimal 3 seri: tetangga sejenis `categorical.1`, pembanding `categorical.2`, F&B lain `categorical.3` | Yang dibandingkan adalah **bentuk** (BR-EST-03); seri ke-4 dst. masuk tabel |
| Rentang estimasi | Bar horizontal rendah–tinggi dengan penanda titik tengah | Warna `accent`, bukan warna status. Angka rendah/tengah/tinggi selalu tertulis |
| Riwayat ulasan | Sparkline satu seri | Tanpa sumbu; tanggal awal/akhir dan nilainya tertulis |

Aturan chart tambahan: satu sumbu Y (tidak ada dual-axis); legend selalu ada untuk ≥ 2 seri; setiap chart dan peta punya **tampilan tabel** (wajib untuk peta choropleth, lihat `03-ux-spec.md`); tooltip saat hover; garis 2px; ujung bar membulat 4px; celah 2px antar segmen.

---

## 4. Tipografi

| Token | Family | Size / line-height | Weight | Pemakaian |
|---|---|---|---|---|
| `text.display` | display | 36 / 44 | 700 | Angka/judul besar di laporan |
| `text.h1` | display | 30 / 36 | 600 | Judul halaman |
| `text.h2` | display | 24 / 32 | 600 | Judul seksi |
| `text.h3` | display | 18 / 28 | 600 | Judul kartu/panel |
| `text.body-lg` | sans | 16 / 24 | 400 | Teks panjang (AI brief) |
| `text.body` | sans | 14 / 20 | 400 | **Default UI** |
| `text.body-sm` | sans | 13 / 18 | 400 | Sel tabel compact |
| `text.caption` | sans | 12 / 16 | 400 | Sumber, timestamp |
| `text.label` | sans | 12 / 16 | 600, +0,02em | Label form, header tabel, badge |
| `text.numeric` | sans | 14 / 20 | 500, **tabular-nums** | Kolom angka |
| `text.mono` | mono | 13 / 20 | 400 | ID, JSON payload |

Font: **Plus Jakarta Sans** (semua heading), **Inter** (body, tabel, data), dan **JetBrains Mono** (mono). Ketiganya berlisensi SIL OFL dan di-self-host. Tidak ada huruf kapital penuh untuk label berbahasa Indonesia; gunakan sentence case.

---

## 5. Spasi, bentuk, elevasi, gerak

| Grup | Token |
|---|---|
| `space.*` (basis 4px) | `0`=0 · `0-5`=2 · `1`=4 · `1-5`=6 · `2`=8 · `3`=12 · `4`=16 · `5`=20 · `6`=24 · `8`=32 · `10`=40 · `12`=48 · `16`=64 |
| `radius.*` | `none` 0 · `sm` 4 (badge) · `md` 6 (input, tombol) · `lg` 8 (kartu) · `xl` 12 (dialog, drawer) · `full` |
| `border-width.*` | `thin` 1 · `thick` 2 · `focus` 2 |
| `shadow.*` | `sm` (kartu) · `md` (dropdown) · `lg` (drawer) · `overlay` (dialog). Di mode gelap bayangan lebih pekat; elevasi utama datang dari `bg.surface-raised`. |
| `motion.duration.*` | `instant` 0 · `fast` 120ms · `normal` 200ms · `slow` 320ms. Saat `prefers-reduced-motion`, semua durasi → 0 (sudah ada di `tokens.css`). |
| `motion.easing.*` | `standard` (0.2,0,0,1) · `enter` (0,0,0,1) · `exit` (0.3,0,1,1) |
| `z.*` | base 0 · dropdown 1000 · sticky 1100 · overlay 1200 · modal 1300 · toast 1400 · tooltip 1500 |
| `size.control.*` | `sm` 32 · `md` 36 (default) · `lg` 44 (target sentuh, mobile) |
| `size.icon.*` | 16 / 20 / 24 |
| `size.target-min` | 24px (WCAG 2.5.8) |
| `size.sidebar` / `drawer` / `content-max` | 248 / 480 / 1440 |
| `breakpoint.*` | sm 640 · md 768 · lg 1024 · **xl 1280 (target desktop)** · 2xl 1536 |
| `density.*` | `compact` 36px · `comfortable` 44px (tinggi baris tabel) |

---

## 6. Komponen inti

Komponen dibangun di `packages/ui` di atas Radix primitives. Komponen hanya boleh membaca token semantik; hex atau px yang di-hardcode akan ditolak lint.

| Komponen | Anatomi | Token | State & aturan | A11y |
|---|---|---|---|---|
| **Button** | label, ikon opsional | `accent.*`, `fg.on-accent`, `radius.md`, `size.control.*` | primary / secondary / ghost / danger; loading; disabled. Tombol berbayar menampilkan biaya ("Unlock · 1 credit"). | Fokus `border.focus`. Aksi berbayar wajib lewat dialog konfirmasi. |
| **Badge** | ikon + label | `color.{score,verification,claim,freshness,confidence,status}.*`, `radius.sm`, `text.label` | Varian `solid` / `subtle`. Badge claim memakai `border-style.claim.*`. | Label teks wajib. Tooltip berisi angka (confidence) dan tanggal. |
| **ScoreBadge** | angka skor + band + (opsional) badge coverage | `color.score.*.solid/on-solid`, `text.numeric` | UNSCORED menampilkan label "Belum dinilai" tanpa angka (tidak pernah "—" atau "0", BR-TRUST-03). `coverage` < 0,6 menampilkan badge "Data terbatas" (`status.warning`). | `aria-label` lengkap, misalnya "Skor prioritas 92, HOT". |
| **ConfidenceMeter** *(F2)* | 3 bar + label | `color.confidence.*` | Tinggi 3 bar, Sedang 2, Rendah 1. Angka di tooltip. | Bukan warna saja: jumlah bar + label. |
| **FactorBar** (penjelasan skor) | nama faktor, bar kontribusi, bobot, nilai, badge claim | `color.accent.default` (bar), `color.claim.*`, `text.numeric` | Faktor Unknown: bar kosong bergaris putus + "Tidak dihitung". | Tersedia sebagai tabel untuk screen reader. |
| **MaskedValue + Unlock** | nilai ter-mask, tombol unlock | `fg.muted`, `text.mono` | Ter-mask → dialog biaya → terbuka (copy, WhatsApp, telepon). | Nilai ter-mask dibacakan "disembunyikan". |
| **EvidenceDrawer** | judul claim, daftar sumber, metadata | `bg.surface-raised`, `shadow.lg`, `size.drawer`, `color.claim.*` | Loading, kosong (claim UNKNOWN), error + retry. | Focus trap. Esc menutup. Fokus kembali ke pemicu. |
| **DataTable** | header sticky, checkbox, kolom angka kanan | `density.*`, `bg.hover/selected`, `text.label`, `text.numeric` | Skeleton, kosong, zero-result, error. Bulk select. | Navigasi keyboard baris. `aria-sort`. Checkbox ≥ 24px. |
| **MapLegend** | ramp/ordinal, label, periode referensi, no-data | `color.geo.*`, `text.caption` | Selalu menampilkan periode data & guardrail (FR-INTL-03). | Peta selalu punya padanan tabel. |
| **Dialog / Drawer** | judul, isi, aksi | `bg.overlay`, `shadow.overlay`, `radius.xl`, `motion.*` | Dialog berbayar: biaya, saldo sebelum/sesudah. | Focus trap, `aria-modal`. |
| **Toast** | ikon status, pesan, aksi | `bg.inverse`, `fg.inverse`, `z.toast` | Error yang bisa di-retry selalu punya tombol "Coba lagi". | `role=status` / `alert`. |
| **EmptyState** | ilustrasi ringan, judul, saran, CTA | `fg.muted`, `space.*` | Copy menjelaskan langkah berikutnya. | — |

---

## 7. Pemakaian di kode

**CSS**

```css
@import "tokens.css";

.prospect-row:hover { background: var(--msp-color-bg-hover); }
.badge--hot {
  background: var(--msp-color-score-hot-solid);
  color: var(--msp-color-score-hot-on-solid);
  border-radius: var(--msp-radius-sm);
}
.badge--inferred {
  color: var(--msp-color-claim-inferred-fg);
  background: var(--msp-color-claim-inferred-bg);
  border: var(--msp-border-width-thin) var(--msp-border-style-claim-inferred) var(--msp-color-claim-inferred-border);
}
```

**Tailwind preset** (ringkas; akan dihasilkan oleh `packages/design-tokens`)

```js
// tailwind preset — warna menunjuk CSS variable agar mode gelap otomatis
export default {
  theme: {
    extend: {
      colors: {
        canvas: "var(--msp-color-bg-canvas)",
        surface: "var(--msp-color-bg-surface)",
        fg: { DEFAULT: "var(--msp-color-fg-default)", muted: "var(--msp-color-fg-muted)" },
        score: {
          hot: { DEFAULT: "var(--msp-color-score-hot-solid)", fg: "var(--msp-color-score-hot-on-solid)" },
        },
      },
      spacing: { 1: "var(--msp-space-1)", 2: "var(--msp-space-2)" /* … */ },
      borderRadius: { md: "var(--msp-radius-md)" },
    },
  },
};
```

---

## 8. Aksesibilitas: hasil validasi kontras

`build.mjs` memeriksa **90 pasangan** (45 pasangan × 2 mode). Teks memakai ambang 4,5:1 dan elemen UI non-teks 3:1. Hasil saat ini: **90/90 lolos**. Rasio terendah:

| Pasangan | Mode | Rasio | Min |
|---|---|---|---|
| `border.input` / `bg.surface` | dark | 3,75 | 3,0 |
| `fg.subtle` / `bg.surface` | light | 4,76 | 4,5 |
| `verification.verified.fg` / `.bg` | light & dark | 5,02 | 4,5 |
| `score.hot.on-solid` / `.solid` | light & dark | 5,18 | 4,5 |

Aturan tambahan:
- Focus ring: 2px `border.focus` dengan offset 2px, selalu tampil pada `:focus-visible`.
- Target interaktif ≥ 24px (desktop) dan ≥ 44px (sentuh).
- Status tidak pernah hanya warna (ikon + label, plus gaya border untuk claim).
- Palet categorical slot 3–5 di mode terang di bawah 3:1 terhadap surface, sehingga label langsung atau tabel wajib ada (aturan relief data-viz).

---

## 9. Item terbuka

| ID | Item | Status |
|---|---|---|
| D-17 | Brand final | **Selesai:** Forest & Mint, Plus Jakarta Sans + Inter. Logo menyusul. |
| DS-01 | Set ikon: Lucide diusulkan (ISC) | Proposed |
| DS-02 | Basemap peta & gaya peta dasar | Menunggu Q-05 (lisensi dataset) |
| DS-03 | `packages/design-tokens` memakai output `build.mjs` langsung (tanpa Style Dictionary) | Saat bootstrap monorepo (WP-07) |
| DS-04 | Ilustrasi empty state | Setelah brand final |
