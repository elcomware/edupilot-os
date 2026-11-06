// @vitest-environment jsdom
import { beforeAll, beforeEach, afterEach, describe, it, expect, vi } from "vitest";
import { render, screen, within, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ModuleDescriptor } from "@edupilot/api-client";
import { usePreferences, preferenceKey } from "../../../../resources/js/app/preferences";
import { PreferencesPage } from "../../../../resources/js/features/preferences/PreferencesPage";
import { TenantAdminLayout } from "../../../../resources/js/layouts/TenantAdminLayout";

const person = { school: "north", id: "staff", permissions: ["modules.read"] };
const modules: ModuleDescriptor[] = [
  { key: "tenancy", group: "platform", title: { en: "Schools", fr: "Établissements" }, capabilities: [] },
  { key: "auth", group: "platform", title: { en: "Accounts and access", fr: "Comptes et accès" }, capabilities: [] },
];
function workspace(catalogStatus: "ready" | "loading" | "unavailable" = "ready") {
  return render(<TenantAdminLayout principal={person} modules={modules}><PreferencesPage modules={modules} catalogStatus={catalogStatus} /></TenantAdminLayout>);
}
function railTitles() {
  return Array.from(document.querySelectorAll(".pin-zone a")).map(a => a.textContent);
}
beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function () { this.setAttribute("open", ""); };
  HTMLDialogElement.prototype.close = function () { this.removeAttribute("open"); };
});
beforeEach(() => {
  localStorage.clear();
  usePreferences.getState().activate(person);
  usePreferences.getState().update({ locale: "en", pins: ["auth", "unavailable-private", "tenancy"] });
  window.location.hash = "#/settings";
});
afterEach(() => { cleanup(); vi.restoreAllMocks(); });

describe("personal pinned navigation", () => {
  it("renders saved order in both settings and rail using only authorized descriptors", () => {
    workspace();
    expect(railTitles()).toEqual(["Accounts and access", "Schools"]);
    const list = screen.getByRole("list", { name: "Pinned navigation" });
    expect(within(list).getAllByRole("listitem").map(row => row.getAttribute("aria-label"))).toEqual(["Accounts and access", "Schools"]);
    expect(screen.queryByText("unavailable-private")).toBeNull();
  });
  it("moves across hidden pins, updates the rail immediately and retains logical focus", async () => {
    const user = userEvent.setup();
    workspace();
    expect((screen.getByRole("button", { name: "Move Accounts and access up" }) as HTMLButtonElement).disabled).toBe(true);
    expect((screen.getByRole("button", { name: "Move Schools down" }) as HTMLButtonElement).disabled).toBe(true);
    await user.click(screen.getByRole("button", { name: "Move Accounts and access down" }));
    expect(usePreferences.getState().value.pins).toEqual(["tenancy", "unavailable-private", "auth"]);
    expect(railTitles()).toEqual(["Schools", "Accounts and access"]);
    expect(document.activeElement).toBe(screen.getByRole("listitem", { name: "Accounts and access" }));
    expect(screen.getByRole("status").textContent).toBe("Preferences saved on this device.");
    const saved = JSON.parse(localStorage.getItem(preferenceKey(person))!);
    expect(saved.pins).toEqual(["tenancy", "unavailable-private", "auth"]);
  });
  it("distinguishes unavailable catalog from empty pins and offers the directory when empty", () => {
    const view = workspace("unavailable");
    expect(screen.getByText("Pinned navigation is unavailable. Reload this workspace to try again.")).toBeTruthy();
    expect(screen.queryByRole("list", { name: "Pinned navigation" })).toBeNull();
    expect(screen.queryByRole("link", { name: "Back to directory" })).toBeNull();
    view.unmount();
    usePreferences.getState().update({ pins: [] });
    workspace();
    expect(screen.getByRole("link", { name: "Back to directory" }).getAttribute("href")).toBe("#/workspace");
  });
  it("names localized keyboard controls with the full real module title", async () => {
    usePreferences.getState().update({ locale: "fr" });
    workspace();
    const user = userEvent.setup();
    const down = screen.getByRole("button", { name: "Descendre Comptes et accès" });
    down.focus();
    await user.keyboard("{Enter}");
    expect(railTitles()).toEqual(["Établissements", "Comptes et accès"]);
    expect(screen.getByRole("status").textContent).toBe("Préférences enregistrées sur cet appareil.");
  });
  it("announces session-only ordering when device storage rejects a save", async () => {
    workspace();
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new DOMException("Denied", "SecurityError"); });
    await userEvent.setup().click(screen.getByRole("button", { name: "Move Accounts and access down" }));
    expect(railTitles()).toEqual(["Schools", "Accounts and access"]);
    expect(screen.getByRole("status").textContent).toContain("session only");
    expect(screen.getByRole("status").textContent).not.toContain("Preferences saved");
  });
});

describe("scoped workspace reset", () => {
  it("cancels without changes, then resets only the current subject and school", async () => {
    const store = usePreferences.getState();
    const otherUser = { ...person, id: "other" }, otherSchool = { ...person, school: "south" };
    for (const principal of [otherUser, otherSchool]) {
      store.activate(principal);
      store.update({ appearance: "dark", pins: ["auth"] });
    }
    const savedOther = [otherUser, otherSchool].map(p => localStorage.getItem(preferenceKey(p)));
    localStorage.setItem("unrelated-session-marker", "untouched");
    store.activate(person);
    store.update({ appearance: "dark", accent: "clay", density: "compact", width: "wide" });
    workspace();
    const user = userEvent.setup();
    const liveRegion = screen.getByRole("status");
    const trigger = screen.getByRole("button", { name: "Reset this workspace" });
    await user.click(trigger);
    let dialog = screen.getByRole("dialog", { name: "Reset this workspace?" });
    await user.click(within(dialog).getAllByRole("button", { name: "Cancel" })[1]);
    expect(usePreferences.getState().value.appearance).toBe("dark");
    expect(document.activeElement).toBe(trigger);
    await user.selectOptions(screen.getByLabelText("Language", { exact: true }), "fr");
    await user.click(screen.getByRole("button", { name: "Réinitialiser cet espace" }));
    dialog = screen.getByRole("dialog", { name: "Réinitialiser cet espace ?" });
    await user.click(within(dialog).getByRole("button", { name: "Réinitialiser les préférences" }));
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(usePreferences.getState().value).toMatchObject({ locale: "en", appearance: "system", accent: "pine", density: "comfortable", width: "standard", pins: [] });
    expect(localStorage.getItem(preferenceKey(person))).toBeNull();
    expect([otherUser, otherSchool].map(p => localStorage.getItem(preferenceKey(p)))).toEqual(savedOther);
    expect(localStorage.getItem("unrelated-session-marker")).toBe("untouched");
    expect(screen.getByRole("status")).toBe(liveRegion);
    expect(screen.getByRole("status").textContent).toBe("This workspace’s preferences were reset on this device.");
  });
  it("keeps settings and confirmation available when removal is denied", async () => {
    workspace();
    const before = usePreferences.getState().value;
    vi.spyOn(Storage.prototype, "removeItem").mockImplementation(() => { throw new DOMException("Denied", "SecurityError"); });
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Reset this workspace" }));
    const dialog = screen.getByRole("dialog", { name: "Reset this workspace?" });
    await user.click(within(dialog).getByRole("button", { name: "Reset preferences" }));
    expect(usePreferences.getState().value).toEqual(before);
    expect(within(dialog).getByRole("alert").textContent).toBe("Couldn’t remove saved preferences. Your settings are unchanged. Try again.");
    expect(screen.getAllByRole("status")).toHaveLength(1);
  });
});
