// @vitest-environment jsdom
import { describe, it, expect } from "vitest";
import {
  usePreferences,
  preferenceKey,
  parsePreferences,
} from "../app/preferences";
import { readRoute } from "../app/router";
import { collectionAccess, normalizeText } from "../features/collections/model";
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
    store.update({ appearance: "dark", pins: ["students"] });
    store.activate(person("two"));
    expect(usePreferences.getState().value.pins).toEqual([]);
    store.activate(person("one", "another"));
    expect(usePreferences.getState().value.appearance).toBe("system");
    store.activate(person("one"));
    expect(usePreferences.getState().value.pins).toEqual(["students"]);
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
    expect(readRoute("#/modules/students/edit").page).toBe("unknown");
    expect(readRoute("#/modules/a%2Fb").page).toBe("unknown");
  });
  it("requires principal and descriptor permissions; archive is student-specific", () => {
    const module = {
      key: "students",
      group: "lifecycle",
      title: { en: "Students", fr: "Élèves" },
      capabilities: ["list", "archive"],
    };
    expect(
      collectionAccess(module, {
        ...person("one"),
        permissions: ["students.read"],
      }),
    ).toEqual({ list: true, edit: false, create: false, archive: false });
    expect(
      collectionAccess(
        { ...module, key: "library" },
        { ...person("one"), permissions: ["workspace.write"] },
      ).archive,
    ).toBe(false);
  });
  it("counts Unicode code points after trimming, and rejects control characters", () => {
    expect(normalizeText(" 𐐀 ", 1)).toBe("𐐀");
    expect(normalizeText("a\tb", 100)).toBeNull();
  });
});
