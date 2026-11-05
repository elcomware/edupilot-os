// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, createApiClient } from "@edupilot/api-client";
import { validYear } from "../features/specialized/EnrollmentPanel";
import {
  validDocumentId,
  allowedDocument,
} from "../features/specialized/DocumentsPanel";
import { api, signOut, useAuth, queries } from "../app/session";
import { parsePreferences } from "../app/preferences";
afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  queries.clear();
});
describe("school-specific workflows", () => {
  it("requires consecutive academic years and bounded supported file types", () => {
    expect(validYear("2026-2027")).toBe(true);
    expect(validYear("2026-2028")).toBe(false);
    expect(
      allowedDocument({ size: 8 * 1024 * 1024, type: "application/pdf" }),
    ).toBe(true);
    expect(
      allowedDocument({ size: 8 * 1024 * 1024 + 1, type: "application/pdf" }),
    ).toBe(false);
    expect(validDocumentId("../not-a-document")).toBe(false);
  });
  it("sends the raw file once and rejects credential redirects", async () => {
    const file = new Blob(["staff"], { type: "text/plain" });
    const fetcher = vi.fn(
      async () => new Response(JSON.stringify({ id: "doc" }), { status: 201 }),
    );
    const client = createApiClient({
      baseUrl: "/api/v1",
      getToken: () => "bearer",
      fetcher,
    });
    expect(await client.uploadDocument(file)).toEqual({ id: "doc" });
    expect(fetcher).toHaveBeenCalledWith(
      "/api/v1/documents/files",
      expect.objectContaining({
        body: file,
        redirect: "error",
        headers: expect.objectContaining({
          "Content-Type": "text/plain",
          Authorization: "Bearer bearer",
        }),
      }),
    );
  });
  it("distinguishes loss of write confirmation from an offline rejection before dispatch", async () => {
    const fetcher = vi.fn(async () => {
      throw new TypeError("connection lost");
    });
    const api = createApiClient({ baseUrl: "/api/v1", fetcher });
    await expect(
      api.createEntry("library", { title: "Reading lists" }),
    ).rejects.toMatchObject({ code: "write_unknown" });
    vi.stubGlobal("navigator", { onLine: false });
    await expect(
      api.createEntry("library", { title: "Reading lists" }),
    ).rejects.toMatchObject({ code: "offline" });
    expect(fetcher).toHaveBeenCalledTimes(1);
  });
  it("clears the local session and school caches even if revocation fails", async () => {
    const p = { id: "staff", school: "A", permissions: ["students.read"] };
    useAuth.getState().start({ token: "token", principal: p });
    queries.setQueryData(["students", "A"], ["private"]);
    vi.spyOn(api, "logout").mockRejectedValue(
      new ApiError(0, "offline", "Offline"),
    );
    await signOut();
    expect(useAuth.getState()).toMatchObject({
      session: null,
      revocationUnknown: true,
    });
    expect(queries.getQueryData(["students", "A"])).toBeUndefined();
  });
  it("prefers saved staff language and otherwise preserves the language chosen at sign-in", () => {
    expect(parsePreferences(null, "fr").locale).toBe("fr");
    expect(parsePreferences('{"locale":"en"}', "fr").locale).toBe("en");
  });
});
