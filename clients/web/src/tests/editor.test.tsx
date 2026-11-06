// @vitest-environment jsdom
import { beforeAll, afterEach, describe, it, expect, vi } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError } from "@edupilot/api-client";
import { api } from "../../../../resources/js/app/session";
import { RecordDialog } from "../../../../resources/js/features/collections/RecordDialog";
beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function () {
    this.setAttribute("open", "");
  };
  HTMLDialogElement.prototype.close = function () {
    this.removeAttribute("open");
  };
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});
describe("versioned workspace edits", () => {
  it("retains a rejected draft and adopts the latest version only when requested", async () => {
    const row = {
      id: "r1",
      title: "Jo",
      status: "open",
      version: 2,
    };
    const latest = vi.fn(async () => ({
      ...row,
      title: "Joseph",
      version: 3,
    }));
    const update = vi
      .spyOn(api, "updateEntry")
      .mockRejectedValueOnce(new ApiError(409, "version_conflict", "Changed"))
      .mockResolvedValue({ ...row, version: 4 });
    const onSaved = vi.fn(),
      user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <RecordDialog
          module="tenancy"
          editor={{ kind: "edit", row }}
          onSaved={onSaved}
          onClose={() => {}}
          latest={latest}
        />
      </QueryClientProvider>,
    );
    await user.clear(screen.getByLabelText("Title"));
    await user.type(screen.getByLabelText("Title"), "Joe");
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    await screen.findByRole("alert");
    expect(
      (screen.getByLabelText("Title") as HTMLInputElement).value,
    ).toBe("Joe");
    expect(onSaved).not.toHaveBeenCalled();
    await user.click(
      screen.getByRole("button", { name: "Load latest version" }),
    );
    await screen.findByDisplayValue("Joseph");
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    expect(update).toHaveBeenLastCalledWith("tenancy", "r1", {
      status: "open",
      title: "Joseph",
      version: 3,
    });
  });
});
