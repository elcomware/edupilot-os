import * as auth from "../../modules/auth/resources/js/pages.web";
import * as tenancy from "../../modules/tenancy/resources/js/pages.web";

export const LoginPage = auth.Login;
export const workspacePages: ReadonlyMap<string, typeof tenancy.Workspace> =
  new Map<string, typeof tenancy.Workspace>([
    ["auth", auth.Workspace],
    ["tenancy", tenancy.Workspace],
  ]);
