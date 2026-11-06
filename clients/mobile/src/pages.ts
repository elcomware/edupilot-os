import * as auth from "@edupilot/auth/native";
import * as tenancy from "@edupilot/tenancy/native";

export const LoginPage = auth.Login;
export const workspacePages: ReadonlyMap<string, typeof tenancy.Workspace> =
  new Map<string, typeof tenancy.Workspace>([
    ["auth", auth.Workspace],
    ["tenancy", tenancy.Workspace],
  ]);

export const directoryIcons = new Map<string, number>([
  ["auth", auth.directoryIcon],
  ["tenancy", tenancy.directoryIcon],
]);
