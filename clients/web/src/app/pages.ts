import * as auth from "@edupilot/auth/web";
import * as tenancy from "@edupilot/tenancy/web";

export const LoginPage = auth.Login;
export const workspacePages: ReadonlyMap<string, typeof tenancy.Workspace> =
  new Map<string, typeof tenancy.Workspace>([
    ["auth", auth.Workspace],
    ["tenancy", tenancy.Workspace],
  ]);

export const directoryIcons = new Map<string, string>([
  ["auth", auth.directoryIcon],
  ["tenancy", tenancy.directoryIcon],
]);
