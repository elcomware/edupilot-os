// @vitest-environment jsdom
import { expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

// Read the canonical stylesheet, rather than copying its colors into fixtures.
const style = document.createElement("style");
style.textContent = readFileSync(resolve(import.meta.dirname, "../../../../resources/css/tokens.css"), "utf8");
document.head.append(style);
const rules = Array.from(style.sheet!.cssRules).filter(
  (rule): rule is CSSStyleRule => rule.type === CSSRule.STYLE_RULE,
);
style.remove();

function token(theme: "light" | "dark", name: string) {
  const base = rules.find((rule) => rule.selectorText === ":root");
  const dark = rules.find(
    (rule) => rule.selectorText === ':root[data-theme="dark"]',
  );
  if (!base || !dark) throw new Error("Missing canonical theme declarations");
  return (
    (theme === "dark" && dark.style.getPropertyValue(name)) ||
    base.style.getPropertyValue(name)
  ).trim();
}

function luminance(hex: string) {
  // These actual opaque palette tokens use CSS hexadecimal colors.
  expect(hex).toMatch(/^#(?:[\da-f]{3}|[\da-f]{6})$/i);
  const digits = hex.slice(1);
  const expanded = digits.length === 3
    ? [...digits].map((digit) => digit + digit).join("")
    : digits;
  const channels = [0, 2, 4].map((offset) => {
    const channel = parseInt(expanded.slice(offset, offset + 2), 16) / 255;
    return channel <= 0.04045
      ? channel / 12.92
      : ((channel + 0.055) / 1.055) ** 2.4;
  });
  return channels[0] * 0.2126 + channels[1] * 0.7152 + channels[2] * 0.0722;
}

it.each([
  ["light", "--paper"],
  ["light", "--surface"],
  ["dark", "--paper"],
  ["dark", "--surface"],
] as const)("keeps active field borders at 3:1 against %s %s", (theme, background) => {
  // Inputs use --field-line with --surface inside, and sit on both page planes.
  // This checks those token pairs, not full rendered-page WCAG conformance.
  const border = luminance(token(theme, "--field-line"));
  const plane = luminance(token(theme, background));
  const contrast = (Math.max(border, plane) + 0.05) / (Math.min(border, plane) + 0.05);
  expect(contrast).toBeGreaterThanOrEqual(3);
});
