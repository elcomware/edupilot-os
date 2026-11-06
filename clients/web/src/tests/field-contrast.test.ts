// @vitest-environment jsdom
import { expect, it } from "vitest";
import { createRequire } from "node:module";
import { readFileSync } from "node:fs";

const resolve = createRequire(import.meta.url).resolve;
const style = document.createElement("style");
style.textContent = ["@edupilot/core/brand/palette.css", "@edupilot/core/css/tokens.css"]
  .map((name) => readFileSync(resolve(name), "utf8")).join("\n");
document.head.append(style);
const rules = Array.from(style.sheet!.cssRules).filter(
  (rule): rule is CSSStyleRule => rule.type === CSSRule.STYLE_RULE,
);
style.remove();

function token(theme: "light" | "dark", name: string): string {
  const selectors = theme === "dark" ? [':root[data-theme="dark"]', ":root"] : [":root"];
  for (const selector of selectors) {
    for (const rule of rules.filter((item) => item.selectorText === selector).reverse()) {
      const value = rule.style.getPropertyValue(name).trim();
      if (!value) continue;
      const ref = /^var\((--[a-z-]+)\)$/.exec(value);
      return ref ? token(theme, ref[1]) : value;
    }
  }
  throw new Error("Missing theme token " + name);
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
