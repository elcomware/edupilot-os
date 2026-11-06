import * as auth from "../../modules/auth/resources/js/pages.native";
import * as tenancy from "../../modules/tenancy/resources/js/pages.native";

export const LoginPage = auth.Login;
export const workspacePages: ReadonlyMap<string, typeof tenancy.Workspace> =
  new Map<string, typeof tenancy.Workspace>([
    ["auth", auth.Workspace],
    ["tenancy", tenancy.Workspace],
  ]);
