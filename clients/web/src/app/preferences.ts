import { create } from "zustand";
import type { Locale, Principal } from "@edupilot/api-client";
export interface Preferences {
  locale: Locale;
  appearance: "system" | "light" | "dark";
  accent: "pine" | "indigo" | "clay";
  density: "comfortable" | "compact";
  width: "standard" | "wide";
  pins: string[];
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
});
export function preferenceKey(p: Pick<Principal, "school" | "id">) {
  return "edupilot:prefs:v1:" + JSON.stringify([p.school, p.id]);
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
interface Settings {
  scope: string | null;
  value: Preferences;
  stored: boolean;
  activate: (p: Principal | null) => void;
  update: (changes: Partial<Preferences>) => void;
  pin: (key: string) => void;
}
export const usePreferences = create<Settings>((set, get) => ({
  scope: null,
  value: defaults(),
  stored: true,
  activate(p) {
    const scope = p ? preferenceKey(p) : null;
    const fallback = get().value.locale;
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
