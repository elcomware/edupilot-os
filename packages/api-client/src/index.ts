/** Platform-independent EduPilot HTTP contract; web and Expo supply their own token storage. */
export type Locale = "en" | "fr";
export interface Principal {
  id: string;
  school: string;
  permissions: string[];
}
export interface Credentials {
  school: string;
  email: string;
  password: string;
}
export interface Session {
  token: string;
  principal: Principal;
}
export interface ModuleDescriptor {
  key: string;
  group: string;
  title: Record<Locale, string>;
  capabilities: string[];
}
export interface StudentNames {
  givenName: string;
  familyName: string;
}
export interface Student extends StudentNames {
  id: string;
  version: number;
  status: string;
}
export interface WorkspaceEntry {
  id: string;
  title: string;
  status: string;
  version: number;
}
export interface PlacementInput {
  studentId: string;
  academicYear: string;
  grade: string;
  course?: string;
}
export interface Placement extends PlacementInput {
  id: string;
  version: number;
  status: "active" | "withdrawn";
}
export interface Page<T> {
  items: T[];
  total: number;
}
export interface Paging {
  limit?: number;
  offset?: number;
}
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}
type ClientOptions = {
  /** Versioned base, e.g. /api/v1 or https://school.example/api/v1. */
  baseUrl: string;
  getToken?: () => string | null | undefined;
  onUnauthorized?: () => void;
  fetcher?: typeof fetch;
};
export function createApiClient({
  baseUrl,
  getToken,
  onUnauthorized,
  fetcher = fetch,
}: ClientOptions) {
  const root = baseUrl.replace(/\/+$/, "");
  const key = encodeURIComponent;
  async function send<T>(
    path: string,
    method: string,
    body: RequestInit["body"],
    contentType: string | undefined,
    consume: (r: Response) => Promise<T>,
    signal?: AbortSignal,
    authenticate = true,
    timeoutMs = 15000,
  ): Promise<T> {
    const token = authenticate ? getToken?.() : null;
    const headers: Record<string, string> = { Accept: "application/json" };
    if (token) headers.Authorization = "Bearer " + token;
    if (contentType) headers["Content-Type"] = contentType;
    const write = method !== "GET";
    if (
      write &&
      typeof navigator !== "undefined" &&
      (navigator as { onLine?: boolean }).onLine === false
    )
      throw new ApiError(0, "offline", "No request was sent while offline.");
    const controller = new AbortController(),
      cancel = () => controller.abort();
    if (signal?.aborted) cancel();
    else signal?.addEventListener("abort", cancel, { once: true });
    let timedOut = false;
    const timer = setTimeout(() => {
      timedOut = true;
      cancel();
    }, timeoutMs);
    try {
      const response = await fetcher(root + path, {
        method,
        headers,
        signal: controller.signal,
        body,
        redirect: "error",
        credentials: "omit",
      });
      if (!response.ok) {
        if (response.status === 401 && token && token === getToken?.())
          onUnauthorized?.();
        const payload = await response.json().catch(() => null);
        throw new ApiError(
          response.status,
          payload?.error?.code ?? "http_error",
          payload?.error?.message ?? "Request failed",
        );
      }
      return await consume(response);
    } catch (error) {
      if (error instanceof ApiError || signal?.aborted) throw error;
      const uncertain = write && authenticate;
      throw new ApiError(
        0,
        uncertain ? "write_unknown" : timedOut ? "timeout" : "network",
        uncertain
          ? "No confirmation received for this write."
          : timedOut
            ? "The request timed out."
            : "Could not reach the server.",
      );
    } finally {
      clearTimeout(timer);
      signal?.removeEventListener("abort", cancel);
    }
  }
  async function request<T>(
    path: string,
    method = "GET",
    body?: unknown,
    signal?: AbortSignal,
    authenticate = true,
  ): Promise<T> {
    return send(
      path,
      method,
      body === undefined ? undefined : JSON.stringify(body),
      body === undefined ? undefined : "application/json",
      async (r) =>
        r.status === 204 ? (undefined as T) : ((await r.json()) as T),
      signal,
      authenticate,
    );
  }
  const paging = ({ limit = 25, offset = 0 }: Paging = {}) =>
    "?limit=" +
    encodeURIComponent(limit) +
    "&offset=" +
    encodeURIComponent(offset);
  const workspacePath = (module: string) =>
    "/modules/" + key(module) + "/workspace";
  return {
    placements: (
      page: Paging & { studentId?: string } = {},
      signal?: AbortSignal,
    ) =>
      request<Page<Placement>>(
        "/enrollment/placements" +
          paging(page) +
          (page.studentId ? "&studentId=" + key(page.studentId) : ""),
        "GET",
        undefined,
        signal,
      ),
    place: (data: PlacementInput) =>
      request<Placement>("/enrollment/placements", "POST", data),
    withdraw: (id: string, version: number) =>
      request<Placement>("/enrollment/placements/" + key(id), "PATCH", {
        status: "withdrawn",
        version,
      }),
    uploadDocument: async (
      file: Blob,
      contentType = file.type,
    ): Promise<{ id: string }> => {
      return send(
        "/documents/files",
        "POST",
        file,
        contentType,
        (r) => r.json(),
        undefined,
        true,
        60000,
      );
    },
    downloadDocument: async (
      id: string,
      signal?: AbortSignal,
    ): Promise<Blob> => {
      return send(
        "/documents/files/" + key(id),
        "GET",
        undefined,
        undefined,
        (r) => r.blob(),
        signal,
        true,
        60000,
      );
    },
    login: (credentials: Credentials) =>
      request<Session>("/session", "POST", credentials, undefined, false),
    session: (signal?: AbortSignal) =>
      request<Principal>("/session", "GET", undefined, signal),
    logout: () => request<void>("/session", "DELETE"),
    modules: (signal?: AbortSignal) =>
      request<{ items: ModuleDescriptor[] }>(
        "/modules",
        "GET",
        undefined,
        signal,
      ),
    module: (module: string, signal?: AbortSignal) =>
      request<ModuleDescriptor>(
        "/modules/" + key(module),
        "GET",
        undefined,
        signal,
      ),
    workspace: (module: string, page?: Paging, signal?: AbortSignal) =>
      request<Page<WorkspaceEntry> & { module: string }>(
        workspacePath(module) + paging(page),
        "GET",
        undefined,
        signal,
      ),
    entry: (module: string, id: string, signal?: AbortSignal) =>
      request<WorkspaceEntry>(
        workspacePath(module) + "/" + key(id),
        "GET",
        undefined,
        signal,
      ),
    createEntry: (module: string, data: { title: string }) =>
      request<WorkspaceEntry>(workspacePath(module), "POST", data),
    updateEntry: (
      module: string,
      id: string,
      data: Pick<WorkspaceEntry, "title" | "version" | "status">,
    ) =>
      request<WorkspaceEntry>(
        workspacePath(module) + "/" + key(id),
        "PATCH",
        data,
      ),
    students: (page?: Paging, signal?: AbortSignal) =>
      request<Page<Student>>(
        "/students" + paging(page),
        "GET",
        undefined,
        signal,
      ),
    student: (id: string, signal?: AbortSignal) =>
      request<Student>("/students/" + key(id), "GET", undefined, signal),
    createStudent: (names: StudentNames) =>
      request<Student>("/students", "POST", names),
    updateStudent: (id: string, names: StudentNames & { version: number }) =>
      request<Student>("/students/" + key(id), "PATCH", names),
    archiveStudent: (id: string, version: number) =>
      request<void | Student>(
        "/students/" + key(id) + "?version=" + encodeURIComponent(version),
        "DELETE",
      ),
  };
}
export type ApiClient = ReturnType<typeof createApiClient>;
