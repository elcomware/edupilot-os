import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import { dirname } from "node:path";
import { expect, test, type Locator, type Page } from "@playwright/test";
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

async function writeFixtureEvidence(
  page: Page,
  checkpoint: Checkpoint | "reset-description-contrast",
  summary: object,
) {
  if (
    process.env.EDUPILOT_CAPTURE_FIXTURE_UI !== "1" ||
    process.env.GITHUB_ACTIONS !== "true"
  ) return;
  if (new URL(page.url()).hostname !== "127.0.0.1")
    throw new Error("Accessibility fixture artifacts require loopback");
  // Buffer attachments are reporter-owned and need not become files with
  // the list reporter. Persist only the already-redacted report explicitly.
  const path = test.info().outputPath("accessibility-" + checkpoint + ".json");
  await mkdir(dirname(path), { recursive: true, mode: 0o700 });
  await writeFile(path, JSON.stringify(summary, null, 2), { encoding: "utf8", mode: 0o600 });
  await test.info().attach("accessibility-" + checkpoint, { path, contentType: "application/json" });
}

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
    await writeFixtureEvidence(page, checkpoint, summary);
    if (summary.needsReview.length)
      test.info().annotations.push({
        type: "accessibility-manual-review",
        description: checkpoint + ": " + JSON.stringify(summary.needsReview),
      });
    expect(summary.passedRules, "Accessibility engine evaluated tagged rules").toBeGreaterThan(0);
    expect(summary.violations, "WCAG A/AA violations at " + checkpoint).toEqual([]);
  });
}

// This measures the actual opaque fixture pairing independently of axe's
// incomplete findings; it never edits or suppresses the original scan report.
export async function expectDialogDescriptionContrast(page: Page, dialog: Locator) {
  await test.step("Rendered reset dialog description contrast", async () => {
    const measurement = await dialog.evaluate((element) => {
      const id = element.getAttribute("aria-describedby");
      const description = id ? document.getElementById(id) : null;
      if (!description || description.parentElement !== element)
        throw new Error("Expected a direct dialog description reference");
      const text = getComputedStyle(description), surface = getComputedStyle(element);
      function rgba(value: string) {
        const channels = /^rgba?\(/.test(value) ? value.match(/[\d.]+/g)?.map(Number) : null;
        if (!channels || channels.length < 3 || channels.length > 4)
          throw new Error("Expected computed sRGB fixture colors");
        return [channels[0], channels[1], channels[2], channels[3] ?? 1];
      }
      function luminance(channels: number[]) {
        const linear = channels.slice(0, 3).map((value) => {
          const channel = value / 255;
          return channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4;
        });
        return linear[0] * 0.2126 + linear[1] * 0.7152 + linear[2] * 0.0722;
      }
      const foreground = rgba(text.color), background = rgba(surface.backgroundColor);
      const ink = luminance(foreground), plane = luminance(background);
      return {
        foreground,
        background,
        inheritsDialogInk: text.color === surface.color,
        descriptionBackgroundTransparent: rgba(text.backgroundColor)[3] === 0,
        noBackgroundImages: text.backgroundImage === "none" && surface.backgroundImage === "none",
        fullyVisible: text.visibility === "visible" && surface.visibility === "visible" &&
          Number(text.opacity) === 1 && Number(surface.opacity) === 1 &&
          description.getBoundingClientRect().width > 0 && description.getBoundingClientRect().height > 0,
        contrast: (Math.max(ink, plane) + 0.05) / (Math.min(ink, plane) + 0.05),
      };
    });
    await writeFixtureEvidence(page, "reset-description-contrast", measurement);
    expect(measurement.fullyVisible, "Description is visibly rendered").toBe(true);
    expect(measurement.inheritsDialogInk, "Description inherits dialog text color").toBe(true);
    expect(measurement.descriptionBackgroundTransparent, "Description exposes the dialog surface").toBe(true);
    expect(measurement.noBackgroundImages, "Measured fixture has no background image").toBe(true);
    expect(measurement.foreground[3], "Description text is opaque").toBe(1);
    expect(measurement.background[3], "Dialog surface is opaque").toBe(1);
    expect(measurement.contrast, "Rendered normal description text contrast").toBeGreaterThanOrEqual(4.5);
  });
}

export async function expectPageReflow(page: Page) {
  // Tables may scroll inside their named region; the page itself must reflow.
  expect(
    await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    "Page reflows without horizontal overflow",
  ).toBe(true);
}
