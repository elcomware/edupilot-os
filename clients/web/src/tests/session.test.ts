// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, createApiClient } from "@edupilot/api-client";
import { api, signOut, useAuth, queries } from "@edupilot/core/web/app/session";
import { parsePreferences } from "@edupilot/core/web/app/preferences";
afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  queries.clear();
});
describe("school-specific workflows", () => {
  it("distinguishes loss of write confirmation from an offline rejection before dispatch", async () => {
    const fetcher = vi.fn(async () => {
      throw new TypeError("connection lost");
    });
    const api = createApiClient({ baseUrl: "/api/v1", fetcher });
    await expect(
      api.createEntry("tenancy", { title: "Reading lists" }),
    ).rejects.toMatchObject({ code: "write_unknown" });
    vi.stubGlobal("navigator", { onLine: false });
    await expect(
      api.createEntry("tenancy", { title: "Reading lists" }),
    ).rejects.toMatchObject({ code: "offline" });
    expect(fetcher).toHaveBeenCalledTimes(1);
  });
  it("clears the local session and school caches even if revocation fails", async () => {
    const p = { id: "staff", school: "A", permissions: ["workspace.read"] };
    useAuth.getState().start({ token: "token", principal: p });
    queries.setQueryData(["tenancy", "A"], ["private"]);
    vi.spyOn(api, "logout").mockRejectedValue(
      new ApiError(0, "offline", "Offline"),
    );
    await signOut();
    expect(useAuth.getState()).toMatchObject({
      session: null,
      revocationUnknown: true,
    });
    expect(queries.getQueryData(["tenancy", "A"])).toBeUndefined();
  });
  it("prefers saved staff language and otherwise preserves the language chosen at sign-in", () => {
    expect(parsePreferences(null, "fr").locale).toBe("fr");
    expect(parsePreferences('{"locale":"en"}', "fr").locale).toBe("en");
  });
});
