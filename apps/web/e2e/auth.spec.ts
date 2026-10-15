// J-1, auth part (WP-08, FR-AUTH-01, docs/05 §2): sign up, verify the email with the code Kratos sends
// to Mailpit, sign in, sign out; /api/v1/me answers 200 only with a session. Needs Mailpit: @ci-only.
import { expect, test, type APIRequestContext } from "@playwright/test";

// Mailpit HTTP API: local stack http://localhost:8025, CI stack http://mailpit:8025.
const MAILPIT_URL = process.env.MSP_E2E_MAILPIT_URL ?? "http://localhost:8025";
const PASSWORD = "Kopi-Susu-Gula-Aren-2026!";

test("J-1 sign up, verify email, sign in, sign out", { tag: "@ci-only" }, async ({ page, request }) => {
  const email = `j1-${Date.now()}-${Math.floor(Math.random() * 1e6)}@example.test`;
  const me = () => page.request.get("/api/v1/me");

  // Sign up: the verification screen comes next, without a session.
  await page.goto("/signup");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Kata sandi").fill(PASSWORD);
  await page.getByRole("button", { name: "Daftar" }).click();
  await expect(page).toHaveURL(/\/verify\?flow=/);
  expect((await me()).status()).toBe(401);

  // Resend once: a second code arrives; only the newest is used.
  await expect.poll(() => codes(request, email)).toHaveLength(1);
  await page.getByRole("button", { name: "Kirim ulang kode" }).click();
  await expect.poll(() => codes(request, email)).toHaveLength(2);
  const [code] = await codes(request, email);

  // Verify: the address is verified, still without a session.
  await page.getByLabel("Kode verifikasi").fill(code);
  await page.getByRole("button", { name: "Lanjut" }).click();
  await expect(page.getByText("Email Anda sudah terverifikasi.")).toBeVisible();
  await page.getByRole("link", { name: "Lanjut" }).click();
  await expect(page).toHaveURL("/");
  expect((await me()).status()).toBe(401);

  // Sign in: the protected endpoint answers 200 with the verified identity.
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Kata sandi").fill(PASSWORD);
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page).toHaveURL("/");
  const res = await me();
  expect(res.status()).toBe(200);
  expect(await res.json()).toMatchObject({ email, verified: true, aal: "aal1" });

  // Sign out: 401 again.
  await page.goto("/account");
  await page.getByRole("link", { name: "Keluar" }).click();
  await expect(page).toHaveURL(/\/login/);
  expect((await me()).status()).toBe(401);
});

/** Verification codes sent to address, newest first. */
async function codes(request: APIRequestContext, address: string): Promise<string[]> {
  const search = await request.get(`${MAILPIT_URL}/api/v1/search`, { params: { query: `to:"${address}"` } });
  expect(search.ok()).toBe(true);
  const { messages } = (await search.json()) as { messages: { ID: string }[] };
  const result: string[] = [];
  for (const { ID } of messages) {
    const message = await request.get(`${MAILPIT_URL}/api/v1/message/${ID}`);
    const { Text } = (await message.json()) as { Text: string };
    const code = /\b\d{6}\b/.exec(Text)?.[0];
    if (code) result.push(code);
  }
  return result;
}
