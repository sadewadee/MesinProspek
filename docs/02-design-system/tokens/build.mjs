#!/usr/bin/env node
// Build & validate MesinProspek design tokens (Node stdlib only).
//
// Source of truth: design-tokens.json (DTCG format). Light values live in
// `$value`; dark overrides live in `$extensions["com.mesinprospek.mode"].dark`.
//
// Usage:
//   node build.mjs            # write tokens.css and run contrast checks
//   node build.mjs --check    # fail if tokens.css is stale or contrast fails (CI)
//   node build.mjs --report   # print the contrast table as markdown
//   node build.mjs --out <path> [--check]   # write/check <path> instead of tokens.css
//
// This is the only token build: packages/design-tokens ships its output via --out
// (`make generate`). Renaming a CSS variable is a MAJOR token change (README §2.4).

import { readFileSync, writeFileSync, existsSync } from "node:fs";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const SRC = join(HERE, "design-tokens.json");
const MODE_EXT = "com.mesinprospek.mode";
const CSS_EXT = "com.mesinprospek.css";
const PREFIX = "msp";
const SKIP_ROOTS = new Set(["base"]); // primitives are resolved into semantic tokens, not exported
const ALIAS = /^\{([^}]+)\}$/;

// [foreground, background, minimum ratio, note]. Text pairs use WCAG 1.4.3 (4.5:1);
// non-text UI pairs use WCAG 1.4.11 (3:1).
const TEXT = 4.5;
const UI = 3.0;
const CONTRAST_PAIRS = [
  ["color.fg.default", "color.bg.canvas", TEXT, "teks utama di canvas"],
  ["color.fg.default", "color.bg.surface", TEXT, "teks utama di surface"],
  ["color.fg.default", "color.bg.surface-raised", TEXT, "teks di popover"],
  ["color.fg.default", "color.bg.hover", TEXT, "teks di baris hover"],
  ["color.fg.default", "color.bg.selected", TEXT, "teks di baris terpilih"],
  ["color.fg.muted", "color.bg.canvas", TEXT, "teks sekunder di canvas"],
  ["color.fg.muted", "color.bg.surface", TEXT, "teks sekunder di surface"],
  ["color.fg.subtle", "color.bg.surface", TEXT, "placeholder/metadata"],
  ["color.fg.link", "color.bg.surface", TEXT, "tautan"],
  ["color.accent.fg", "color.bg.surface", TEXT, "teks brand"],
  ["color.accent.fg", "color.accent.subtle", TEXT, "nav aktif"],
  ["color.fg.on-accent", "color.accent.default", TEXT, "tombol primer"],
  ["color.fg.on-accent", "color.accent.hover", TEXT, "tombol primer hover"],
  ["color.fg.on-accent", "color.accent.active", TEXT, "tombol primer aktif"],
  ["color.fg.inverse", "color.bg.inverse", TEXT, "tooltip/toast"],
  ["color.border.focus", "color.bg.surface", UI, "focus ring di surface"],
  ["color.border.focus", "color.bg.canvas", UI, "focus ring di canvas"],
  ["color.border.input", "color.bg.surface", UI, "batas input"],
];
for (const fam of ["success", "warning", "danger", "info"]) {
  CONTRAST_PAIRS.push([`color.status.${fam}.fg`, `color.status.${fam}.bg`, TEXT, `status ${fam}`]);
}
for (const band of ["hot", "warm", "cold", "unscored"]) {
  CONTRAST_PAIRS.push([`color.score.${band}.fg`, `color.score.${band}.bg`, TEXT, `score ${band} subtle`]);
  CONTRAST_PAIRS.push([`color.score.${band}.on-solid`, `color.score.${band}.solid`, TEXT, `score ${band} solid`]);
}
for (const st of ["verified", "likely-valid", "risky", "invalid", "unknown"]) {
  CONTRAST_PAIRS.push([`color.verification.${st}.fg`, `color.verification.${st}.bg`, TEXT, `verifikasi ${st}`]);
}
for (const cl of ["observed", "derived", "inferred", "unknown"]) {
  CONTRAST_PAIRS.push([`color.claim.${cl}.fg`, `color.claim.${cl}.bg`, TEXT, `claim ${cl}`]);
}
for (const fr of ["fresh", "aging", "stale"]) {
  CONTRAST_PAIRS.push([`color.freshness.${fr}.fg`, `color.freshness.${fr}.bg`, TEXT, `freshness ${fr}`]);
}
for (const cf of ["high", "medium", "low"]) {
  CONTRAST_PAIRS.push([`color.confidence.${cf}.fg`, `color.confidence.${cf}.bg`, TEXT, `confidence ${cf}`]);
}

const isObject = (v) => v !== null && typeof v === "object" && !Array.isArray(v);

/** Map "a.b.c" -> token (with resolved $type) for every node that has $value. */
function flatten(node, path = [], inheritedType = undefined, out = new Map()) {
  const type = node.$type ?? inheritedType;
  if ("$value" in node) {
    out.set(path.join("."), { ...node, $type: type });
    return out;
  }
  for (const [key, child] of orderedEntries(node)) {
    if (key.startsWith("$") || !isObject(child)) continue;
    flatten(child, [...path, key], type, out);
  }
  return out;
}

// JSON.parse moves integer-like keys ("0", "50") ahead of others ("0-5"), so scale
// groups are re-sorted numerically ("0-5" = 0.5) to keep a stable, logical order.
const SCALE_KEY = /^\d+(-\d+)?$/;
function orderedEntries(node) {
  const entries = Object.entries(node).filter(([k]) => !k.startsWith("$"));
  if (entries.length && entries.every(([k]) => SCALE_KEY.test(k))) {
    entries.sort(([a], [b]) => parseFloat(a.replace("-", ".")) - parseFloat(b.replace("-", ".")));
  }
  return entries;
}

class Resolver {
  constructor(flat) {
    this.flat = flat;
  }

  raw(path, mode) {
    const token = this.flat.get(path);
    const dark = token.$extensions?.[MODE_EXT]?.dark;
    return mode === "dark" && dark !== undefined ? dark : token.$value;
  }

  value(path, mode, seen = []) {
    if (seen.includes(path)) throw new Error(`alias cycle: ${[...seen, path].join(" -> ")}`);
    if (!this.flat.has(path)) throw new Error(`unknown token reference: ${path}`);
    return this.#resolve(this.raw(path, mode), mode, [...seen, path]);
  }

  #resolve(v, mode, seen) {
    if (typeof v === "string") {
      const m = ALIAS.exec(v);
      return m ? this.value(m[1], mode, seen) : v;
    }
    if (Array.isArray(v)) return v.map((x) => this.#resolve(x, mode, seen));
    if (isObject(v)) return Object.fromEntries(Object.entries(v).map(([k, x]) => [k, this.#resolve(x, mode, seen)]));
    return v;
  }
}

const varName = (path) => `--${PREFIX}-` + path.replaceAll(".", "-");

const GENERIC_FAMILIES = new Set(["serif", "sans-serif", "monospace", "system-ui", "ui-monospace", "cursive", "fantasy"]);
const fmtFamily = (fams) => fams.map((f) => (GENERIC_FAMILIES.has(f) || f.startsWith("-") ? f : `"${f}"`)).join(", ");

const fmtShadow = (v) =>
  (Array.isArray(v) ? v : [v]).map((l) => `${l.offsetX} ${l.offsetY} ${l.blur} ${l.spread} ${l.color}`).join(", ");

/** [var-name, css-value] pairs for one token. */
function cssDecls(path, token, value) {
  const t = token.$type;
  const name = varName(path);
  switch (t) {
    case "typography": {
      const decls = [
        [`${name}-font-family`, fmtFamily(value.fontFamily)],
        [`${name}-font-size`, value.fontSize],
        [`${name}-font-weight`, String(value.fontWeight)],
        [`${name}-line-height`, value.lineHeight],
        [`${name}-letter-spacing`, value.letterSpacing],
      ];
      for (const [prop, cssVal] of Object.entries(token.$extensions?.[CSS_EXT] ?? {})) {
        decls.push([`${name}-${prop}`, cssVal]);
      }
      return decls;
    }
    case "fontFamily":
      return [[name, fmtFamily(value)]];
    case "cubicBezier":
      return [[name, `cubic-bezier(${value.join(", ")})`]];
    case "shadow":
      return [[name, fmtShadow(value)]];
    case "fontWeight":
    case "number":
      return [[name, String(value)]];
    default:
      if (typeof value === "string") return [[name, t === "color" ? value.toLowerCase() : value]];
      throw new TypeError(`${path}: unsupported $type ${t} / value ${JSON.stringify(value)}`);
  }
}

function buildCss(flat, res) {
  const light = [];
  const dark = [];
  const typoPaths = [];
  for (const [path, token] of flat) {
    if (SKIP_ROOTS.has(path.split(".")[0])) continue;
    const lv = cssDecls(path, token, res.value(path, "light"));
    const dv = cssDecls(path, token, res.value(path, "dark"));
    light.push(...lv);
    dv.forEach((d, i) => {
      if (d[0] !== lv[i][0] || d[1] !== lv[i][1]) dark.push(d);
    });
    if (token.$type === "typography") typoPaths.push(path);
  }

  const block = (selector, decls, indent = "") =>
    `${indent}${selector} {\n${decls.map(([k, v]) => `${indent}  ${k}: ${v};`).join("\n")}\n${indent}}`;

  const parts = [
    "/* AUTO-GENERATED by docs/02-design-system/tokens/build.mjs from design-tokens.json. Do not edit. */",
    block(":root", [["color-scheme", "light"], ...light]),
    "@media (prefers-color-scheme: dark) {\n" +
      block(':root:not([data-theme="light"])', [["color-scheme", "dark"], ...dark], "  ") +
      "\n}",
    block(':root[data-theme="dark"]', [["color-scheme", "dark"], ...dark]),
  ];
  for (const path of typoPaths) {
    const n = varName(path);
    const props = [
      ["font-family", `var(${n}-font-family)`],
      ["font-size", `var(${n}-font-size)`],
      ["font-weight", `var(${n}-font-weight)`],
      ["line-height", `var(${n}-line-height)`],
      ["letter-spacing", `var(${n}-letter-spacing)`],
    ];
    for (const prop of Object.keys(flat.get(path).$extensions?.[CSS_EXT] ?? {})) {
      props.push([prop, `var(${n}-${prop})`]);
    }
    parts.push(block(`.${PREFIX}-${path.replaceAll(".", "-")}`, props));
  }
  parts.push(
    "@media (prefers-reduced-motion: reduce) {\n  :root {\n" +
      ["fast", "normal", "slow"].map((k) => `    ${varName(`motion.duration.${k}`)}: 0ms;`).join("\n") +
      "\n  }\n}",
  );
  return parts.join("\n\n") + "\n";
}

function rgba(hex) {
  const h = hex.replace(/^#/, "");
  if (h.length !== 6 && h.length !== 8) throw new Error(`expected #rrggbb or #rrggbbaa, got ${hex}`);
  const [r, g, b] = [0, 2, 4].map((i) => parseInt(h.slice(i, i + 2), 16));
  const a = h.length === 8 ? parseInt(h.slice(6, 8), 16) / 255 : 1;
  return [r, g, b, a];
}

const channel = (c) => {
  const s = c / 255;
  return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
};
const luminance = ([r, g, b]) => 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);

/** WCAG contrast ratio; a translucent foreground is composited over the background. */
function contrast(fg, bg) {
  const [fr, fgc, fb, fa] = rgba(fg);
  const [br, bgc, bb] = rgba(bg);
  const comp = [
    [fr, br],
    [fgc, bgc],
    [fb, bb],
  ].map(([f, b]) => fa * f + (1 - fa) * b);
  const lf = luminance(comp);
  const lb = luminance([br, bgc, bb]);
  return (Math.max(lf, lb) + 0.05) / (Math.min(lf, lb) + 0.05);
}

function checkContrast(res) {
  const rows = [];
  for (const [fg, bg, min, note] of CONTRAST_PAIRS) {
    for (const mode of ["light", "dark"]) {
      const ratio = contrast(res.value(fg, mode), res.value(bg, mode));
      rows.push({ fg, bg, mode, ratio, min, ok: ratio >= min, note });
    }
  }
  return rows;
}

function main(argv) {
  const tokens = JSON.parse(readFileSync(SRC, "utf8"));
  const flat = flatten(tokens);
  const res = new Resolver(flat);
  const css = buildCss(flat, res);
  const rows = checkContrast(res);
  const failures = rows.filter((r) => !r.ok);
  const outArg = argv.indexOf("--out");
  if (outArg !== -1 && !argv[outArg + 1]) throw new Error("--out needs a file path");
  const out = outArg === -1 ? join(HERE, "tokens.css") : resolve(argv[outArg + 1]);

  if (argv.includes("--report")) {
    console.log("| Pasangan | Mode | Rasio | Min | Status |");
    console.log("|---|---|---|---|---|");
    for (const r of rows) {
      const strip = (p) => p.replace(/^color\./, "");
      console.log(`| \`${strip(r.fg)}\` / \`${strip(r.bg)}\` | ${r.mode} | ${r.ratio.toFixed(2)} | ${r.min.toFixed(1)} | ${r.ok ? "✓" : "✗"} |`);
    }
    return failures.length ? 1 : 0;
  }

  let stale = false;
  if (argv.includes("--check")) {
    stale = !existsSync(out) || readFileSync(out, "utf8") !== css;
    if (stale) console.error(`${relative(process.cwd(), out)} is out of date: run \`make generate\``);
  } else {
    writeFileSync(out, css, "utf8");
    console.log(`wrote ${relative(join(HERE, "..", "..", ".."), out)} (${css.split("\n").length - 1} lines)`);
  }

  for (const r of failures) {
    console.error(`CONTRAST FAIL [${r.mode}] ${r.fg} on ${r.bg}: ${r.ratio.toFixed(2)} < ${r.min} (${r.note})`);
  }
  console.log(`contrast: ${rows.length - failures.length}/${rows.length} pairs pass`);
  return failures.length || stale ? 1 : 0;
}

process.exitCode = main(process.argv.slice(2));
