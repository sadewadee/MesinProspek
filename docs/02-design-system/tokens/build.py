#!/usr/bin/env python3
"""Build & validate MesinProspek design tokens (stdlib only).

Source of truth: design-tokens.json (DTCG format). Light values live in
`$value`; dark overrides live in `$extensions["com.mesinprospek.mode"].dark`.

Usage:
  python3 build.py            # write tokens.css and run contrast checks
  python3 build.py --check    # fail if tokens.css is stale or contrast fails (CI)
  python3 build.py --report   # print the contrast table as markdown

Reference implementation until packages/design-tokens (Style Dictionary) exists;
the output contract (CSS variable names) must stay identical.
"""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
SRC = HERE / "design-tokens.json"
OUT = HERE / "tokens.css"
MODE_EXT = "com.mesinprospek.mode"
CSS_EXT = "com.mesinprospek.css"
PREFIX = "msp"
SKIP_ROOTS = {"base"}  # primitives are resolved into semantic tokens, not exported
ALIAS = re.compile(r"^\{([^}]+)\}$")

# (foreground, background, minimum ratio, note). Text pairs use WCAG 1.4.3 (4.5:1);
# non-text UI pairs use WCAG 1.4.11 (3:1).
TEXT, UI = 4.5, 3.0
CONTRAST_PAIRS: list[tuple[str, str, float, str]] = [
    ("color.fg.default", "color.bg.canvas", TEXT, "teks utama di canvas"),
    ("color.fg.default", "color.bg.surface", TEXT, "teks utama di surface"),
    ("color.fg.default", "color.bg.surface-raised", TEXT, "teks di popover"),
    ("color.fg.default", "color.bg.hover", TEXT, "teks di baris hover"),
    ("color.fg.default", "color.bg.selected", TEXT, "teks di baris terpilih"),
    ("color.fg.muted", "color.bg.canvas", TEXT, "teks sekunder di canvas"),
    ("color.fg.muted", "color.bg.surface", TEXT, "teks sekunder di surface"),
    ("color.fg.subtle", "color.bg.surface", TEXT, "placeholder/metadata"),
    ("color.fg.link", "color.bg.surface", TEXT, "tautan"),
    ("color.accent.fg", "color.bg.surface", TEXT, "teks brand"),
    ("color.accent.fg", "color.accent.subtle", TEXT, "nav aktif"),
    ("color.fg.on-accent", "color.accent.default", TEXT, "tombol primer"),
    ("color.fg.on-accent", "color.accent.hover", TEXT, "tombol primer hover"),
    ("color.fg.on-accent", "color.accent.active", TEXT, "tombol primer aktif"),
    ("color.fg.inverse", "color.bg.inverse", TEXT, "tooltip/toast"),
    ("color.border.focus", "color.bg.surface", UI, "focus ring di surface"),
    ("color.border.focus", "color.bg.canvas", UI, "focus ring di canvas"),
    ("color.border.input", "color.bg.surface", UI, "batas input"),
]
for fam in ("success", "warning", "danger", "info"):
    CONTRAST_PAIRS.append((f"color.status.{fam}.fg", f"color.status.{fam}.bg", TEXT, f"status {fam}"))
for band in ("hot", "warm", "cold", "unscored"):
    CONTRAST_PAIRS.append((f"color.score.{band}.fg", f"color.score.{band}.bg", TEXT, f"score {band} subtle"))
    CONTRAST_PAIRS.append((f"color.score.{band}.on-solid", f"color.score.{band}.solid", TEXT, f"score {band} solid"))
for st in ("verified", "likely-valid", "risky", "invalid", "unknown"):
    CONTRAST_PAIRS.append((f"color.verification.{st}.fg", f"color.verification.{st}.bg", TEXT, f"verifikasi {st}"))
for cl in ("observed", "derived", "inferred", "unknown"):
    CONTRAST_PAIRS.append((f"color.claim.{cl}.fg", f"color.claim.{cl}.bg", TEXT, f"claim {cl}"))
for fr in ("fresh", "aging", "stale"):
    CONTRAST_PAIRS.append((f"color.freshness.{fr}.fg", f"color.freshness.{fr}.bg", TEXT, f"freshness {fr}"))
for cf in ("high", "medium", "low"):
    CONTRAST_PAIRS.append((f"color.confidence.{cf}.fg", f"color.confidence.{cf}.bg", TEXT, f"confidence {cf}"))


def flatten(node: dict, path: tuple[str, ...] = (), inherited_type: str | None = None, out: dict | None = None) -> dict:
    """Map 'a.b.c' -> token dict (with resolved $type) for every node that has $value."""
    out = {} if out is None else out
    ttype = node.get("$type", inherited_type)
    if "$value" in node:
        token = dict(node)
        token["$type"] = ttype
        out[".".join(path)] = token
        return out
    for key, child in node.items():
        if key.startswith("$") or not isinstance(child, dict):
            continue
        flatten(child, path + (key,), ttype, out)
    return out


class Resolver:
    def __init__(self, flat: dict):
        self.flat = flat

    def raw(self, path: str, mode: str):
        token = self.flat[path]
        if mode == "dark":
            dark = token.get("$extensions", {}).get(MODE_EXT, {}).get("dark")
            if dark is not None:
                return dark
        return token["$value"]

    def value(self, path: str, mode: str, seen: tuple[str, ...] = ()):
        if path in seen:
            raise ValueError(f"alias cycle: {' -> '.join(seen + (path,))}")
        if path not in self.flat:
            raise KeyError(f"unknown token reference: {path}")
        return self._resolve(self.raw(path, mode), mode, seen + (path,))

    def _resolve(self, v, mode: str, seen: tuple[str, ...]):
        if isinstance(v, str):
            m = ALIAS.match(v)
            return self.value(m.group(1), mode, seen) if m else v
        if isinstance(v, list):
            return [self._resolve(x, mode, seen) for x in v]
        if isinstance(v, dict):
            return {k: self._resolve(x, mode, seen) for k, x in v.items()}
        return v


def var_name(path: str) -> str:
    return f"--{PREFIX}-" + path.replace(".", "-")


def fmt_family(fams: list[str]) -> str:
    generic = {"serif", "sans-serif", "monospace", "system-ui", "ui-monospace", "cursive", "fantasy"}
    return ", ".join(f if f in generic or f.startswith("-") else f'"{f}"' for f in fams)


def fmt_shadow(v) -> str:
    layers = v if isinstance(v, list) else [v]
    return ", ".join(f"{l['offsetX']} {l['offsetY']} {l['blur']} {l['spread']} {l['color']}" for l in layers)


def css_decls(path: str, token: dict, value) -> list[tuple[str, str]]:
    """Return (var-name, css-value) pairs for one token."""
    t = token["$type"]
    name = var_name(path)
    if t == "typography":
        decls = [
            (f"{name}-font-family", fmt_family(value["fontFamily"])),
            (f"{name}-font-size", value["fontSize"]),
            (f"{name}-font-weight", str(value["fontWeight"])),
            (f"{name}-line-height", value["lineHeight"]),
            (f"{name}-letter-spacing", value["letterSpacing"]),
        ]
        for prop, css_val in token.get("$extensions", {}).get(CSS_EXT, {}).items():
            decls.append((f"{name}-{prop}", css_val))
        return decls
    if t == "fontFamily":
        return [(name, fmt_family(value))]
    if t == "cubicBezier":
        return [(name, "cubic-bezier(" + ", ".join(str(x) for x in value) + ")")]
    if t == "shadow":
        return [(name, fmt_shadow(value))]
    if t in ("fontWeight", "number"):
        return [(name, str(value))]
    if isinstance(value, str):
        return [(name, value.lower() if t == "color" else value)]
    raise TypeError(f"{path}: unsupported $type {t!r} / value {value!r}")


def build_css(flat: dict, res: Resolver) -> str:
    light: list[tuple[str, str]] = []
    dark: list[tuple[str, str]] = []
    typo_paths: list[str] = []
    for path, token in flat.items():
        if path.split(".")[0] in SKIP_ROOTS:
            continue
        lv = css_decls(path, token, res.value(path, "light"))
        dv = css_decls(path, token, res.value(path, "dark"))
        light.extend(lv)
        dark.extend(d for d, l in zip(dv, lv) if d != l)
        if token["$type"] == "typography":
            typo_paths.append(path)

    def block(selector: str, decls: list[tuple[str, str]], indent: str = "") -> str:
        body = "\n".join(f"{indent}  {k}: {v};" for k, v in decls)
        return f"{indent}{selector} {{\n{body}\n{indent}}}"

    parts = [
        "/* AUTO-GENERATED by docs/02-design-system/tokens/build.py from design-tokens.json. Do not edit. */",
        block(":root", [("color-scheme", "light")] + light),
        "@media (prefers-color-scheme: dark) {\n"
        + block(':root:not([data-theme="light"])', [("color-scheme", "dark")] + dark, "  ")
        + "\n}",
        block(':root[data-theme="dark"]', [("color-scheme", "dark")] + dark),
    ]
    for path in typo_paths:
        n = var_name(path)
        cls = "." + PREFIX + "-" + path.replace(".", "-")
        props = [
            ("font-family", f"var({n}-font-family)"),
            ("font-size", f"var({n}-font-size)"),
            ("font-weight", f"var({n}-font-weight)"),
            ("line-height", f"var({n}-line-height)"),
            ("letter-spacing", f"var({n}-letter-spacing)"),
        ]
        for prop in flat[path].get("$extensions", {}).get(CSS_EXT, {}):
            props.append((prop, f"var({n}-{prop})"))
        parts.append(block(cls, props))
    parts.append(
        "@media (prefers-reduced-motion: reduce) {\n  :root {\n"
        + "\n".join(f"    {var_name('motion.duration.' + k)}: 0ms;" for k in ("fast", "normal", "slow"))
        + "\n  }\n}"
    )
    return "\n\n".join(parts) + "\n"


def _channel(c: float) -> float:
    c /= 255
    return c / 12.92 if c <= 0.03928 else ((c + 0.055) / 1.055) ** 2.4


def _rgba(hex_: str) -> tuple[float, float, float, float]:
    h = hex_.lstrip("#")
    if len(h) not in (6, 8):
        raise ValueError(f"expected #rrggbb or #rrggbbaa, got {hex_}")
    r, g, b = (int(h[i:i + 2], 16) for i in (0, 2, 4))
    a = int(h[6:8], 16) / 255 if len(h) == 8 else 1.0
    return r, g, b, a


def contrast(fg: str, bg: str) -> float:
    fr, fg_, fb, fa = _rgba(fg)
    br, bg_, bb, _ = _rgba(bg)
    # composite a translucent foreground over the background
    fr, fg_, fb = (fa * f + (1 - fa) * b for f, b in ((fr, br), (fg_, bg_), (fb, bb)))
    lf = 0.2126 * _channel(fr) + 0.7152 * _channel(fg_) + 0.0722 * _channel(fb)
    lb = 0.2126 * _channel(br) + 0.7152 * _channel(bg_) + 0.0722 * _channel(bb)
    hi, lo = max(lf, lb), min(lf, lb)
    return (hi + 0.05) / (lo + 0.05)


def check_contrast(res: Resolver) -> list[dict]:
    rows = []
    for fg, bg, minimum, note in CONTRAST_PAIRS:
        for mode in ("light", "dark"):
            f, b = res.value(fg, mode), res.value(bg, mode)
            ratio = contrast(f, b)
            rows.append({"fg": fg, "bg": bg, "mode": mode, "fg_hex": f, "bg_hex": b,
                         "ratio": ratio, "min": minimum, "ok": ratio >= minimum, "note": note})
    return rows


def main(argv: list[str]) -> int:
    tokens = json.loads(SRC.read_text(encoding="utf-8"))
    flat = flatten(tokens)
    res = Resolver(flat)
    css = build_css(flat, res)
    rows = check_contrast(res)
    failures = [r for r in rows if not r["ok"]]

    if "--report" in argv:
        print("| Pasangan | Mode | Rasio | Min | Status |")
        print("|---|---|---|---|---|")
        for r in rows:
            fg, bg = r["fg"].removeprefix("color."), r["bg"].removeprefix("color.")
            print(f"| `{fg}` / `{bg}` | {r['mode']} | {r['ratio']:.2f} | {r['min']} | {'✓' if r['ok'] else '✗'} |")
        return 1 if failures else 0

    if "--check" in argv:
        stale = not OUT.exists() or OUT.read_text(encoding="utf-8") != css
        if stale:
            print("tokens.css is out of date: run `python3 build.py`", file=sys.stderr)
    else:
        OUT.write_text(css, encoding="utf-8")
        stale = False
        print(f"wrote {OUT.relative_to(HERE.parent.parent.parent)} ({css.count(chr(10))} lines)")

    for r in failures:
        print(f"CONTRAST FAIL [{r['mode']}] {r['fg']} on {r['bg']}: {r['ratio']:.2f} < {r['min']} ({r['note']})", file=sys.stderr)
    print(f"contrast: {len(rows) - len(failures)}/{len(rows)} pairs pass")
    return 1 if failures or stale else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
