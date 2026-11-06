// @vitest-environment jsdom
import { afterEach, beforeAll, expect, it } from "vitest";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Dialog } from "@edupilot/core/web/components/primitives";

beforeAll(() => {
  // jsdom has no native showModal/inert implementation. Exercise the shared
  // component's endpoint handling; hosted Chromium verifies real modal focus.
  HTMLDialogElement.prototype.showModal = function () { this.open = true; };
  HTMLDialogElement.prototype.close = function () { this.open = false; };
});
afterEach(cleanup);

function dialog() {
  render(<>
    <button>Outside before</button>
    <Dialog title="Review changes" onClose={() => {}}>
      <input aria-label="Reason" />
      <button>Confirm</button>
      <button disabled>Disabled action</button>
      <fieldset disabled><button>Disabled by fieldset</button></fieldset>
      <div hidden><button>Hidden action</button></div>
      <div style={{ display: "none" }}><button>Collapsed action</button></div>
      <div style={{ visibility: "hidden" }}><button>Invisible action</button></div>
      <div inert><button>Inert action</button></div>
      <button tabIndex={-1}>Programmatic action</button>
    </Dialog>
    <button>Outside after</button>
  </>);
  const modal = screen.getByRole("dialog", { name: "Review changes" });
  return {
    first: within(modal).getByRole("button", { name: "Cancel" }),
    last: within(modal).getByRole("button", { name: "Confirm" }),
    middle: within(modal).getByRole("textbox", { name: "Reason" }),
  };
}

it("wraps Shift+Tab from the first exposed control to the last enabled control", async () => {
  const { first, last } = dialog();
  first.focus();
  await userEvent.setup().tab({ shift: true });
  expect(document.activeElement).toBe(last);
});

it("wraps Tab from the last exposed control to the first without leaving the dialog", async () => {
  const { first, last } = dialog();
  last.focus();
  await userEvent.setup().tab();
  expect(document.activeElement).toBe(first);
});

it("leaves ordinary middle-control Tab events to the browser", async () => {
  const { first, middle, last } = dialog();
  middle.focus();
  expect(middle.dispatchEvent(new KeyboardEvent("keydown", {
    key: "Tab", bubbles: true, cancelable: true,
  }))).toBe(true);
  const user = userEvent.setup();
  await user.tab();
  expect(document.activeElement).toBe(last);
  middle.focus();
  await user.tab({ shift: true });
  expect(document.activeElement).toBe(first);
});
