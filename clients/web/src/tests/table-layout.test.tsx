// @vitest-environment jsdom
import { afterEach, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { WorkspaceTable } from "../../../../resources/js/components/WorkspaceTable";
import {
  parsePreferences,
  preferenceKey,
  usePreferences,
} from "../../../../resources/js/app/preferences";
const staff = { school: "north", id: "staff", permissions: [] };
afterEach(() => {
  cleanup();
  localStorage.clear();
  usePreferences.getState().activate(null);
});
it("persists permitted column keys per register without storing rows or altering actions", async () => {
  const prefs = usePreferences.getState();
  prefs.activate(staff);
  const rows = [
    { id: "private-record", name: "Private student", status: "Active" },
  ];
  const table = (view: string) => (
    <WorkspaceTable
      view={view}
      label="Register"
      rows={rows}
      rowKey={(r) => r.id}
      columns={[
        { key: "name", label: "Name", rowHeader: true, render: (r) => r.name },
        {
          key: "status",
          label: "Status",
          optional: true,
          render: (r) => r.status,
        },
        {
          key: "actions",
          label: "Actions",
          render: () => <button>Open</button>,
        },
      ]}
    />
  );
  const user = userEvent.setup();
  const rendered = render(table("collection:tenancy"));
  await user.click(screen.getByText("Columns", { selector: "summary" }));
  await user.click(screen.getByRole("checkbox", { name: "Status" }));
  expect(screen.queryByRole("columnheader", { name: "Status" })).toBeNull();
  expect(
    screen.getByRole("rowheader", { name: "Private student" }),
  ).toBeTruthy();
  expect(screen.getByRole("button", { name: "Open" })).toBeTruthy();
  const raw = localStorage.getItem(preferenceKey(staff))!;
  expect(raw).not.toContain("private-record");
  expect(raw).not.toContain("Private student");
  expect(parsePreferences(raw).columns["collection:tenancy"]).toEqual([
    "status",
  ]);
  rendered.rerender(table("collection:auth"));
  expect(screen.getByRole("columnheader", { name: "Status" })).toBeTruthy();
  cleanup();
  prefs.activate({ ...staff, school: "south" });
  render(table("collection:tenancy"));
  expect(screen.getByRole("columnheader", { name: "Status" })).toBeTruthy();
});
it("ignores arbitrary views and unrecognized or required column names in persisted settings", () => {
  const raw = JSON.stringify({
    columns: {
      "collection:tenancy": ["status", "status", "name", "actions", "secret"],
      __private: ["status"],
    },
  });
  expect(parsePreferences(raw).columns).toEqual({
    "collection:tenancy": ["status"],
  });
});
