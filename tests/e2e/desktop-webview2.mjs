// Attach only to the WebView2 created by this disposable Windows desktop.
import { chromium, expect } from "@playwright/test";
import { spawn } from "node:child_process";
import { join, resolve } from "node:path";
import { setTimeout as delay } from "node:timers/promises";

if (process.platform !== "win32" || process.env.RUNNER_ENVIRONMENT !== "github-hosted")
  throw new Error("Requires a disposable Windows runner");
const app = spawn(resolve("clients/desktop/bin/edupilot.exe"), [], {
  env: {
    ...process.env,
    WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS: "--remote-debugging-port=19222",
    WEBVIEW2_USER_DATA_FOLDER: join(process.env.RUNNER_TEMP, "edupilot-webview2"),
  },
  stdio: "inherit",
});
let browser;
try {
  for (let attempt = 0; attempt < 100; attempt++) {
    if (app.exitCode !== null) throw new Error("Windows desktop exited during startup");
    try {
      browser = await chromium.connectOverCDP("http://127.0.0.1:19222", {timeout: 1500});
      break;
    } catch {
      await delay(400);
    }
  }
  if (!browser) throw new Error("Windows desktop WebView2 did not become ready");
  const context = browser.contexts()[0];
  let page;
  await expect.poll(() => {
    page = context.pages().find(p => new URL(p.url()).hostname === "wails.localhost");
    return Boolean(page);
  }, {timeout: 30000}).toBe(true);

  const screenshot = async (state) => {
    await page.evaluate(() => document.fonts.ready);
    await page.screenshot({path: join(process.env.DESKTOP_SCREEN_DIR, "windows-" + state + ".png")});
  };
  const school = page.getByLabel("School ID", {exact: true});
  await expect(school).toBeVisible({timeout: 30000});
  await screenshot("signin");
  await school.fill(process.env.EDUPILOT_SEED_SCHOOL);
  await page.getByLabel("Email", {exact: true}).fill(process.env.EDUPILOT_SEED_EMAIL);
  await page.getByLabel("Password", {exact: true}).fill(process.env.EDUPILOT_SEED_PASSWORD);
  await page.getByRole("button", {name: "Sign in", exact: true}).click();
  const students = page.getByRole("link", {name: "Students", exact: true});
  await expect(students).toBeVisible({timeout: 15000});
  await expect(page.locator(".school-label")).toHaveText(process.env.EDUPILOT_SEED_SCHOOL);
  await screenshot("directory");
  await students.click();
  await page.getByRole("button", {name: "Add student", exact: true}).click();
  await page.getByLabel("Given name", {exact: true}).fill("Jamie");
  await page.getByLabel("Family name", {exact: true}).fill("Demo");
  await page.getByRole("button", {name: "Save student", exact: true}).click();
  await expect(page.getByRole("rowheader", {name: "Jamie Demo"})).toBeVisible();
  await screenshot("students");
  const nav = page.getByRole("button", {name: "Navigation", exact: true});
  if (await nav.isVisible() && await nav.getAttribute("aria-expanded") === "false") await nav.click();
  await page.getByRole("button", {name: "Sign out", exact: true}).click();
  await expect(school).toBeVisible();
} finally {
  try { await browser?.close(); } finally { app.kill(); }
}
