import { expect, test, type Page } from "@playwright/test";
import { captureWorkspace } from "./ui-evidence";
import { account, loginForm } from "./accounts";
import { expectAccessible, expectDialogDescriptionContrast, expectPageReflow } from "./accessibility";

async function openNavigation(page: Page) {
  const toggle = page.getByRole("button", { name: "Navigation", exact: true });
  if (await toggle.isVisible() && await toggle.getAttribute("aria-expanded") === "false") await toggle.click();
}
async function signOut(page: Page) {
  await openNavigation(page);
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
}

test("personal pins reorder, survive reload and reset only the current workspace", async ({ page }) => {
  test.setTimeout(90_000); // Full-page scans share this journey's existing sign-ins.
  const primary = account("NAV"), other = account("NAV_OTHER");
  await page.emulateMedia({ colorScheme: "light" });
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto("/#/settings");
  await expect(page.getByRole("button", { name: "Sign in", exact: true })).toBeVisible();
  await expect(page.getByRole("img", { name: "EduPilot OS", exact: true })).toBeVisible();
  await expectAccessible(page, "sign-in-desktop");
  await page.getByLabel("Language", { exact: true }).selectOption("fr");
  await page.emulateMedia({ colorScheme: "dark" });
  await page.setViewportSize({ width: 320, height: 780 });
  await expect(page.locator("html")).toHaveAttribute("lang", "fr");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expectPageReflow(page);
  await expectAccessible(page, "sign-in-narrow-fr");
  await page.getByLabel("Langue", { exact: true }).selectOption("en");
  await page.emulateMedia({ colorScheme: "light" });
  await page.setViewportSize({ width: 1280, height: 900 });
  await loginForm(page, other);
  await page.getByLabel("Accent", { exact: true }).selectOption("clay");
  await page.getByRole("link", { name: "Module directory", exact: true }).click();
  await page.getByRole("button", { name: "Pin Accounts and access", exact: true }).click();
  await page.getByRole("link", { name: "Preferences", exact: true }).click();
  await signOut(page);

  await loginForm(page, primary);
  await expect(page.getByLabel("Accent", { exact: true })).toHaveValue("garnet");
  await expect(page.getByRole("list", { name: "Pinned navigation", exact: true })).toHaveCount(0);
  await page.getByRole("link", { name: "Back to directory", exact: true }).click();
  await expect(page.getByRole("button", {name:"Pin Schools",exact:true})).toBeVisible();
  await captureWorkspace(page, "directory-desktop");
  await expect(page.getByRole("link", { name: "EduPilot OS", exact: true }).getByRole("img", { name: "EduPilot OS", exact: true })).toBeVisible();
  await expectAccessible(page, "directory-desktop");
  for (const title of ["Schools", "Accounts and access"]) {
    await page.getByRole("button", { name: "Pin " + title, exact: true }).click();
  }
  await page.getByRole("link", { name: "Preferences", exact: true }).click();
  const rail = page.locator(".pin-zone a");
  await expect(rail).toHaveText(["Schools", "Accounts and access"]);
  const up = page.getByRole("button", { name: "Move Accounts and access up", exact: true });
  await up.focus();
  await page.keyboard.press("Enter");
  await expect(rail).toHaveText(["Accounts and access", "Schools"]);
  await expect(page.getByRole("listitem", { name: "Accounts and access", exact: true })).toBeFocused();
  await expect(page.getByRole("button", { name: "Move Accounts and access up", exact: true })).toBeDisabled();
  await page.reload();
  await loginForm(page, primary);
  await expect(rail).toHaveText(["Accounts and access", "Schools"]);

  await captureWorkspace(page, "preferences-desktop");
  await expectAccessible(page, "preferences-desktop");
  await page.getByLabel("Appearance", { exact: true }).selectOption("dark");
  await page.getByLabel("Density", { exact: true }).selectOption("compact");
  await page.getByLabel("Reading width", { exact: true }).selectOption("wide");
  await page.getByLabel("Language", { exact: true }).selectOption("fr");
  await page.setViewportSize({ width: 320, height: 780 });
  await expect(page.getByRole("listitem", { name: "Comptes et accès", exact: true })).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.locator("html")).toHaveAttribute("data-density", "compact");
  await expectPageReflow(page);
  await captureWorkspace(page, "preferences-mobile-fr");
  await expectAccessible(page, "preferences-narrow-fr");
  await openNavigation(page);
  await page.getByRole("link", { name: "Répertoire des modules", exact: true }).click();
  await expect(page.getByRole("button", { name: "Désépingler Comptes et accès", exact: true })).toBeVisible();
  await expectPageReflow(page);
  await expectAccessible(page, "directory-narrow-fr");
  await openNavigation(page);
  await page.getByRole("link", { name: "Préférences", exact: true }).click();
  const reset = page.getByRole("button", { name: "Réinitialiser cet espace", exact: true });
  await reset.focus();
  await page.keyboard.press("Enter");
  let dialog = page.getByRole("dialog", { name: "Réinitialiser cet espace ?", exact: true });
  const close = dialog.getByRole("button", { name: "Annuler", exact: true }).first();
  await expect(close).toBeFocused();
  await page.keyboard.press("Shift+Tab");
  await expect(dialog.getByRole("button", { name: "Réinitialiser les préférences", exact: true })).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(close).toBeFocused();
  await expectPageReflow(page);
  await expectAccessible(page, "reset-dialog-narrow-fr");
  await expectDialogDescriptionContrast(page, dialog);
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(reset).toBeFocused();
  await expect(page.getByRole("list", { name: "Navigation épinglée", exact: true }).getByRole("listitem")).toHaveCount(2);
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
  await expect(page.getByRole("button", { name: "Pin Schools", exact: true })).toBeVisible();
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
  await expect(page.getByRole("listitem", { name: "Accounts and access", exact: true })).toBeVisible();
  await page.reload();
  await loginForm(page, other);
  await expect(page.getByLabel("Accent", { exact: true })).toHaveValue("clay");
  await expect(page.getByRole("listitem", { name: "Accounts and access", exact: true })).toBeVisible();
});
