import { expect, test } from "@playwright/test";
import { account, loginForm } from "./accounts";

test("Goravel serves the web sign-in and keeps API requests JSON", async ({ page, request }) => {
  const pageResponse = await page.goto("/login");
  expect(pageResponse?.status()).toBe(200);
  expect(pageResponse?.headers()["content-type"]).toContain("text/html");
  expect(pageResponse?.headers()["vary"]).toContain("Accept");
  await loginForm(page, account("AUTH"));
  await expect(page.getByRole("heading", { name: "Module directory", exact: true })).toBeVisible();

  await page.getByRole("link", { name: "Accounts and access", exact: true }).click();
  await expect(page.getByRole("heading", { name: "My account" })).toBeVisible();
  await expect(page.getByText(account("AUTH").email, { exact: true })).toBeVisible();
  await expect(page.getByText("School administrator", { exact: true })).toBeVisible();

  for (const [path, accept] of [
    ["/login", "application/json"],
    ["/api/v1/session", "text/html"],
  ]) {
    const response = await request.get(path, { headers: { Accept: accept } });
    expect(response.status()).toBe(401);
    expect(response.headers()["content-type"]).toContain("application/json");
    expect((await response.json()).error.code).toBe("unauthenticated");
  }
});
