// @vitest-environment jsdom
import { describe, it, expect } from "vitest";
import {
  usePreferences,
  preferenceKey,
  parsePreferences,
} from "../../../../resources/js/app/preferences";
import { readRoute } from "../../../../resources/js/app/router";
import { collectionAccess, normalizeText } from "../../../../resources/js/features/collections/model";
const person = (school: string, id = "staff") => ({
  school,
  id,
  permissions: [],
});
describe("shared workstation", () => {
  it("keeps an administrator's language out of the next account's defaults", () => {
    const store = usePreferences.getState();
    store.activate(person("lang-reset"));
    store.update({ locale: "fr" });
    store.reset();
    store.activate(person("lang-a"));
    store.update({ locale: "fr" });
    store.activate(person("lang-reset"));
    expect(usePreferences.getState().value.locale).toBe("en");
    store.activate(person("lang-b"));
    expect(usePreferences.getState().value.locale).toBe("en");
    store.activate(person("lang-a"));
    expect(usePreferences.getState().value.locale).toBe("fr");
    store.activate(null);
    expect(usePreferences.getState().value.locale).toBe("en");
    // An explicit signed-out choice may seed a new workspace.
    store.update({ locale: "fr" });
    store.activate(person("lang-c"));
    expect(usePreferences.getState().value.locale).toBe("fr");
    store.activate(null);
  });
  it("isolates pins and appearance by both school and staff member", () => {
    const store = usePreferences.getState();
    store.activate(person("one"));
    store.update({ appearance: "dark", pins: ["tenancy"] });
    store.activate(person("two"));
    expect(usePreferences.getState().value.pins).toEqual([]);
    store.activate(person("one", "another"));
    expect(usePreferences.getState().value.appearance).toBe("system");
    store.activate(person("one"));
    expect(usePreferences.getState().value.pins).toEqual(["tenancy"]);
    expect(preferenceKey(person("a:b", "c"))).not.toBe(
      preferenceKey(person("a", "b:c")),
    );
  });
  it("ignores malformed persisted preferences", () => {
    expect(parsePreferences("{").appearance).toBe("system");
    expect(
      parsePreferences('{"pins":[false,"a","a"],"density":"tiny"}'),
    ).toMatchObject({ density: "comfortable", pins: ["a"] });
  });
});
describe("route and action boundaries", () => {
  it("rejects nested or encoded module paths", () => {
    expect(readRoute("#/modules/tenancy/edit").page).toBe("unknown");
    expect(readRoute("#/modules/a%2Fb").page).toBe("unknown");
  });
  it("requires principal and descriptor permissions for list and edit", () => {
    const module = {
      key: "tenancy",
      group: "platform",
      title: { en: "Schools", fr: "Établissements" },
      capabilities: ["list", "archive"],
    };
    expect(
      collectionAccess(module, {
        ...person("one"),
        permissions: ["workspace.read"],
      }),
    ).toEqual({ list: true, edit: false, create: false });
    expect(
      collectionAccess(
        { ...module, key: "auth" },
        { ...person("one"), permissions: ["workspace.write"] },
      ).edit,
    ).toBe(false);
  });
  it("counts Unicode code points after trimming, and rejects control characters", () => {
    expect(normalizeText(" 𐐀 ", 1)).toBe("𐐀");
    expect(normalizeText("a\tb", 100)).toBeNull();
  });
});
