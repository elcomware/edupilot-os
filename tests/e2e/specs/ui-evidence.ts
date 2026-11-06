import { expect, test, type Page } from "@playwright/test";

// Only the disposable native CI workflow enables these explicit successful
// states. Standard Playwright failure screenshots, videos and traces stay off.
export async function captureWorkspace(
  page: Page,
  kind: "directory-desktop" | "preferences-desktop" | "preferences-mobile-fr",
) {
  if (
    process.env.EDUPILOT_CAPTURE_FIXTURE_UI !== "1" ||
    process.env.GITHUB_ACTIONS !== "true"
  )
    return;
  const url = new URL(page.url());
  const expected = kind === "directory-desktop" ? "#/workspace" : "#/settings";
  if (url.hostname !== "127.0.0.1" || url.hash !== expected)
    throw new Error("Unexpected fixture capture location");
  await expect(page.locator(".app-shell")).toBeVisible();
  await expect(page.locator('input[type="password"]')).toHaveCount(0);
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({
    path: test.info().outputPath("qa-" + kind + ".png"),
    fullPage: kind === "preferences-mobile-fr",
    animations: "disabled",
    style: ".school-label { visibility: hidden !important; }",
  });
}
