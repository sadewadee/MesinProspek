// tailwind.css is hand-written; tokens.css is generated. A renamed or removed
// token must fail here instead of silently producing an empty utility.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

const read = (file) => readFileSync(new URL(file, import.meta.url), "utf8");

test("tailwind.css only references --msp-* variables defined in tokens.css", () => {
  const defined = new Set(read("./tokens.css").match(/--msp-[\w-]+(?=:)/g));
  const used = read("./tailwind.css").match(/--msp-[\w-]+/g);
  assert.ok(used.length > 0);
  assert.deepEqual(used.filter((name) => !defined.has(name)), []);
});
