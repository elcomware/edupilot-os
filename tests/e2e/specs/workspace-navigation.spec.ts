import { expect, test, type Page } from "@playwright/test";
import { captureWorkspace } from "./ui-evidence";
import { account, loginForm } from "./accounts";

async function openNavigation(page: Page) {
  const toggle = page.getByRole("button", { name: "Navigation", exact: true });
  if (await toggle.isVisible() && await toggle.getAttribute("aria-expanded") === "false") await toggle.click();
}
async function signOut(page: Page) {
  await openNavigation(page);
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
}

test("personal pins reorder, survive reload and reset only the current workspace", async ({ page }) => {
  const primary = account("NAV"), other = account("NAV_OTHER");
  await page.goto("/#/settings");
  await loginForm(page, other);
  await page.getByLabel("Accent", { exact: true }).selectOption("clay");
  await page.getByRole("link", { name: "Module directory", exact: true }).click();
  await page.getByRole("button", { name: "Pin Library", exact: true }).click();
  await page.getByRole("link", { name: "Preferences", exact: true }).click();
  await signOut(page);

  await loginForm(page, primary);
  await expect(page.getByLabel("Accent", { exact: true })).toHaveValue("pine");
  await expect(page.getByRole("list", { name: "Pinned navigation", exact: true })).toHaveCount(0);
  await page.getByRole("link", { name: "Back to directory", exact: true }).click();
  await expect(page.getByRole("button", {name:"Pin Students",exact:true})).toBeVisible();
  await captureWorkspace(page, "directory-desktop");
  for (const title of ["Students", "Library", "Family and student portal"]) {
    await page.getByRole("button", { name: "Pin " + title, exact: true }).click();
  }
  await page.getByRole("link", { name: "Preferences", exact: true }).click();
  const rail = page.locator(".pin-zone a");
  await expect(rail).toHaveText(["Students", "Library", "Family and student portal"]);
  const up = page.getByRole("button", { name: "Move Library up", exact: true });
  await up.focus();
  await page.keyboard.press("Enter");
  await expect(rail).toHaveText(["Library", "Students", "Family and student portal"]);
  await expect(page.getByRole("listitem", { name: "Library", exact: true })).toBeFocused();
  await expect(page.getByRole("button", { name: "Move Library up", exact: true })).toBeDisabled();
  await page.reload();
  await loginForm(page, primary);
  await expect(rail).toHaveText(["Library", "Students", "Family and student portal"]);

  await captureWorkspace(page, "preferences-desktop");
  await page.getByLabel("Appearance", { exact: true }).selectOption("dark");
  await page.getByLabel("Density", { exact: true }).selectOption("compact");
  await page.getByLabel("Reading width", { exact: true }).selectOption("wide");
  await page.getByLabel("Language", { exact: true }).selectOption("fr");
  await page.setViewportSize({ width: 320, height: 780 });
  await expect(page.getByRole("listitem", { name: "Portail familles et élèves", exact: true })).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.locator("html")).toHaveAttribute("data-density", "compact");
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await captureWorkspace(page, "preferences-mobile-fr");
  const reset = page.getByRole("button", { name: "Réinitialiser cet espace", exact: true });
  await reset.click();
  let dialog = page.getByRole("dialog", { name: "Réinitialiser cet espace ?", exact: true });
  await dialog.getByRole("button", { name: "Annuler", exact: true }).last().click();
  await expect(reset).toBeFocused();
  await expect(page.getByRole("list", { name: "Navigation épinglée", exact: true }).getByRole("listitem")).toHaveCount(3);
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await reset.click();
  dialog = page.getByRole("dialog", { name: "Réinitialiser cet espace ?", exact: true });
  await dialog.getByRole("button", { name: "Réinitialiser les préférences", exact: true }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(page.getByLabel("Appearance", { exact: true })).toHaveValue("system");
  await expect(page.getByLabel("Density", { exact: true })).toHaveValue("comfortable");
  await expect(page.getByLabel("Reading width", { exact: true })).toHaveValue("standard");
  await expect(page.getByRole("list", { name: "Pinned navigation", exact: true })).toHaveCount(0);
  await expect(page.getByRole("status")).toHaveText("This workspace’s preferences were reset on this device.");
  // Reset leaves the authenticated session usable; a real school directory still opens.
  await page.getByRole("link", { name: "Back to directory", exact: true }).click();
  await expect(page.getByRole("button", { name: "Pin Students", exact: true })).toBeVisible();
  await page.reload();
  await loginForm(page, primary);
  await openNavigation(page);
  await page.getByRole("link", { name: "Preferences", exact: true }).click();
  await expect(page.getByLabel("Appearance", { exact: true })).toHaveValue("system");
  await expect(page.getByRole("list", { name: "Pinned navigation", exact: true })).toHaveCount(0);
  await signOut(page);
  await loginForm(page, other);
  await openNavigation(page);
  await page.getByRole("link", { name: "Preferences", exact: true }).click();
  await expect(page.getByLabel("Accent", { exact: true })).toHaveValue("clay");
  await expect(page.getByRole("listitem", { name: "Library", exact: true })).toBeVisible();
  await page.reload();
  await loginForm(page, other);
  await expect(page.getByLabel("Accent", { exact: true })).toHaveValue("clay");
  await expect(page.getByRole("listitem", { name: "Library", exact: true })).toBeVisible();
});
