import { expect, type APIRequestContext, type Page } from "@playwright/test";

type Slot = "AUTH" | "A" | "B" | "ROSTER" | "PREFS" | "PREFS_OTHER" | "NAV" | "NAV_OTHER";
type Account = { school: string; email: string; password: string };

export function account(slot: Slot): Account {
  const required = (name: string) => {
    const key = "E2E_" + name + "_" + slot;
    if (!process.env[key])
      throw new Error("Missing " + key + "; see tests/e2e/README.md");
    return process.env[key]!;
  };
  return {
    school: required("SCHOOL"),
    email: required("EMAIL"),
    password: required("PASSWORD"),
  };
}

export async function loginForm(page: Page, user: Account) {
  await page.getByLabel("School ID", { exact: true }).fill(user.school);
  await page.getByLabel("Email", { exact: true }).fill(user.email);
  await page.getByLabel("Password", { exact: true }).fill(user.password);
  const submit = page.getByRole("button", { name: "Sign in", exact: true });
  const [response] = await Promise.all([
    page.waitForResponse((r) =>
      new URL(r.url()).pathname === "/api/v1/session" && r.request().method() === "POST",
    ),
    submit.click(),
  ]);
  // Never include the session body, request data or authorization headers in reports.
  expect(response.status(), "browser sign-in HTTP status").toBe(200);
  await expect(page.getByLabel("School ID", { exact: true })).toHaveCount(0);
}

export async function session(request: APIRequestContext, user: Account) {
  const response = await request.post("/api/v1/session", { data: user });
  expect(response.status(), "API sign-in HTTP status").toBe(200);
  const body = await response.json();
  expect(body.principal.school).toBe(user.school);
  expect(typeof body.token).toBe("string");
  expect(body.token.length).toBeGreaterThan(0);
  return { Authorization: "Bearer " + body.token };
}
