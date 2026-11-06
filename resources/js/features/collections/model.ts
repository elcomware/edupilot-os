import type {
  ModuleDescriptor,
  Principal,
  WorkspaceEntry,
} from "@edupilot/api-client";
export type RegisterRow = WorkspaceEntry;
export const rowName = (row: RegisterRow) => row.title;
export function collectionAccess(
  module: ModuleDescriptor,
  principal: Principal,
) {
  const can = (verb: string, permission: string) =>
    module.capabilities.includes(verb) &&
    principal.permissions.includes("workspace." + permission);
  return {
    list: can("list", "read"),
    create: can("create", "write"),
    edit: can("update", "write"),
  };
}
export const PAGE_SIZE = 25;
export function normalizeText(
  value: string,
  limit: number,
  { allowControls = false } = {},
) {
  const clean = value.trim();
  return clean &&
    Array.from(clean).length <= limit &&
    (allowControls || !/[\u0000-\u001f\u007f-\u009f]/u.test(clean))
    ? clean
    : null;
}
