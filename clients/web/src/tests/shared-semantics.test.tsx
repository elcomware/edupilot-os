// @vitest-environment jsdom
import { afterEach, beforeAll, expect, it } from "vitest";
import { cleanup, render, screen, within } from "@testing-library/react";
import { Brand, Dialog } from "@edupilot/core/web/components/primitives";

beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function () { this.open = true; };
  HTMLDialogElement.prototype.close = function () { this.open = false; };
});
afterEach(cleanup);

it.each([true, false])("names the home link and logo when wordmark is %s", (wordmark) => {
  render(<a href="#/workspace"><Brand wordmark={wordmark} /></a>);
  const home = screen.getByRole("link", { name: "EduPilot OS" });
  expect(within(home).getByRole("img", { name: "EduPilot OS" })).toBeTruthy();
  expect(screen.getAllByRole("img")).toHaveLength(1);
});

it("describes the shared dialog with its optional description without a dangling reference", () => {
  const view = render(<Dialog title="Review changes" description="Check the consequences before confirming." onClose={() => {}} children={null} />);
  expect(screen.getByRole("dialog", {
    name: "Review changes",
    description: "Check the consequences before confirming.",
  })).toBeTruthy();
  view.rerender(<Dialog title="Review changes" onClose={() => {}} children={null} />);
  expect(screen.getByRole("dialog", { name: "Review changes" }).hasAttribute("aria-describedby")).toBe(false);
});
