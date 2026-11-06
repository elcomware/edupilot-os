import { createRequire } from "node:module";
import { expect, test, type Page } from "@playwright/test";
import type * as axeCore from "axe-core";

const require = createRequire(import.meta.url);
type Checkpoint =
  | "sign-in-desktop"
  | "sign-in-narrow-fr"
  | "directory-desktop"
  | "directory-narrow-fr"
  | "preferences-desktop"
  | "preferences-narrow-fr"
  | "reset-dialog-narrow-fr"
  | "student-columns-desktop"
  | "student-columns-narrow-fr";

// Run against the whole real page, including exposed disclosures/dialogs.
// Return only rule IDs and structural targets: axe's HTML, selectors, text,
// accessible names and failure summaries can contain private record values.
export async function expectAccessible(page: Page, checkpoint: Checkpoint) {
  await test.step("WCAG A/AA: " + checkpoint, async () => {
    await page.addScriptTag({ path: require.resolve("axe-core/axe.min.js") });
    const summary = await page.evaluate(async () => {
      const axe = (window as typeof window & { axe: typeof axeCore }).axe;
      const tags = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];
      // Child positions identify the affected element without returning any
      // attributes (IDs, hrefs, values, classes or names) from the school page.
      function target(element?: HTMLElement) {
        if (!element) return "unresolved element";
        const path: string[] = [];
        let current: Element = element;
        while (current.parentElement) {
          const parent = current.parentElement;
          const position = Array.from(parent.children).indexOf(current) + 1;
          path.unshift(":nth-child(" + position + ")");
          current = parent;
        }
        return current === document.documentElement
          ? [":root", ...path].join(" > ")
          : "element outside document tree";
      }
      try {
        const results = await axe.run(document, {
          runOnly: { type: "tag", values: tags },
          elementRef: true,
        });
        const issues = (rules: typeof results.violations) =>
          rules.map((rule) => ({
            rule: rule.id,
            impact: rule.impact,
            targets: rule.nodes.map((node) => target(node.element)),
          }));
        return {
          version: axe.version,
          tags,
          passedRules: results.passes.length,
          violations: issues(results.violations),
          needsReview: issues(results.incomplete),
        };
      } catch {
        // Fail closed without serializing a browser exception's page content.
        throw new Error("Automated accessibility scan could not complete");
      }
    });
    await test.info().attach("accessibility-" + checkpoint + ".json", {
      body: JSON.stringify(summary, null, 2),
      contentType: "application/json",
    });
    if (summary.needsReview.length)
      test.info().annotations.push({
        type: "accessibility-manual-review",
        description: checkpoint + ": " + JSON.stringify(summary.needsReview),
      });
    expect(summary.passedRules, "Accessibility engine evaluated tagged rules").toBeGreaterThan(0);
    expect(summary.violations, "WCAG A/AA violations at " + checkpoint).toEqual([]);
  });
}

export async function expectPageReflow(page: Page) {
  // Tables may scroll inside their named region; the page itself must reflow.
  expect(
    await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    "Page reflows without horizontal overflow",
  ).toBe(true);
}
