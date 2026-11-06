import { create } from "zustand";
import type { Locale, ModuleDescriptor, Principal } from "@edupilot/api-client";
export interface Preferences {
  locale: Locale;
  appearance: "system" | "light" | "dark";
  accent: "pine" | "indigo" | "clay";
  density: "comfortable" | "compact";
  width: "standard" | "wide";
  pins: string[];
  columns: Record<string, string[]>;
}
const defaults = (): Preferences => ({
  locale:
    typeof navigator !== "undefined" && navigator.language.startsWith("fr")
      ? "fr"
      : "en",
  appearance: "system",
  accent: "pine",
  density: "comfortable",
  width: "standard",
  pins: [],
  columns: {},
});
export function preferenceKey(p: Pick<Principal, "school" | "id">) {
  return "edupilot:prefs:v1:" + JSON.stringify([p.school, p.id]);
}
const viewKey = /^(?:collection:[a-z][a-z0-9_]{0,39}|enrollment:placements)$/;
function allowedColumns(view: string, value: unknown): string[] {
  if (!viewKey.test(view) || !Array.isArray(value)) return [];
  const allowed = view === "enrollment:placements" ? ["status", "year", "grade"] : ["status"];
  return allowed.filter(key => value.includes(key));
}
export function parsePreferences(
  raw: string | null,
  fallback = defaults().locale,
): Preferences {
  const prefs = { ...defaults(), locale: fallback };
  try {
    const value = JSON.parse(raw || "{}");
    const choices = {
      locale: ["en", "fr"],
      appearance: ["system", "light", "dark"],
      accent: ["pine", "indigo", "clay"],
      density: ["comfortable", "compact"],
      width: ["standard", "wide"],
    };
    for (const [k, valid] of Object.entries(choices))
      if (valid.includes(value?.[k])) Object.assign(prefs, { [k]: value[k] });
    if (value?.columns && typeof value.columns === "object" && !Array.isArray(value.columns)) {
      for (const [view, keys] of Object.entries(value.columns).slice(0, 128)) {
        const hidden = allowedColumns(view, keys);
        if (hidden.length) prefs.columns[view] = hidden;
      }
    }
    if (Array.isArray(value?.pins))
      prefs.pins = [
        ...new Set<string>(
          value.pins.filter(
            (x: unknown) => typeof x === "string" && x.length <= 100,
          ),
        ),
      ].slice(0, 100);
  } catch {
    /* Malformed local settings must not block sign-in. */
  }
  return prefs;
}
/** Saved pin order is cosmetic; descriptors always come from the authorized catalog. */
export function orderedPinnedModules(modules: ModuleDescriptor[], pins: string[]) {
  const catalog = new Map(modules.map(module => [module.key, module]));
  return [...new Set(pins)].flatMap(key => {
    const module = catalog.get(key);
    return module ? [module] : [];
  });
}
export function movePinnedModule(pins: string[], modules: ModuleDescriptor[], key: string, direction: -1 | 1) {
  const visible = orderedPinnedModules(modules, pins);
  const index = visible.findIndex(module => module.key === key);
  const neighbor = visible[index + direction];
  if (index < 0 || !neighbor) return pins;
  const next = [...pins], from = pins.indexOf(key), to = pins.indexOf(neighbor.key);
  [next[from], next[to]] = [next[to], next[from]];
  return next;
}
interface Settings {
  scope: string | null;
  value: Preferences;
  stored: boolean;
  activate: (p: Principal | null) => void;
  update: (changes: Partial<Preferences>) => void;
  pin: (key: string) => void;
  reset: () => boolean;
  setColumns: (view: string, hidden: string[]) => void;
}
export const usePreferences = create<Settings>((set, get) => ({
  scope: null,
  value: defaults(),
  stored: true,
  activate(p) {
    const scope = p ? preferenceKey(p) : null;
    // Only an explicit language choice on the signed-out screen carries into a
    // new workspace. An authenticated account's locale belongs to that account.
    const previous = get();
    const fallback = previous.scope ? defaults().locale : previous.value.locale;
    let value = { ...defaults(), locale: fallback },
      stored = true;
    if (scope) {
      try {
        value = parsePreferences(localStorage.getItem(scope), fallback);
      } catch {
        stored = false;
      }
    }
    set({ scope, value, stored });
  },
  update(changes) {
    const { scope, value } = get(),
      next = { ...value, ...changes };
    let stored = true;
    if (scope)
      try {
        localStorage.setItem(scope, JSON.stringify(next));
      } catch {
        stored = false;
      }
    set({ value: next, stored });
  },
  setColumns(view, hidden) {
    if (!viewKey.test(view)) return;
    const columns = { ...get().value.columns };
    const keys = allowedColumns(view, hidden);
    if (keys.length) columns[view] = keys;
    else delete columns[view];
    get().update({ columns });
  },
  reset() {
    const { scope } = get();
    try {
      if (scope) localStorage.removeItem(scope);
    } catch {
      set({ stored: false });
      return false;
    }
    set({ value: defaults(), stored: true });
    return true;
  },
  pin(key) {
    const {
      value: { pins },
      update,
    } = get();
    update({
      pins: pins.includes(key)
        ? pins.filter((id) => id !== key)
        : [...pins, key],
    });
  },
}));
export const useLocale = () => usePreferences((s) => s.value.locale);
