/** Platform-independent EduPilot HTTP contract; web and Expo supply their own token storage. */
export type Locale = 'en' | 'fr'
export interface Principal { id: string; school: string; permissions: string[] }
export interface Credentials { school: string; email: string; password: string }
export interface Session { token: string; principal: Principal }
export interface ModuleDescriptor {
  key: string; group: string; title: Record<Locale, string>; capabilities: string[]
}
export interface StudentNames { givenName: string; familyName: string }
export interface Student extends StudentNames { id: string; version: number; status: string }
export interface WorkspaceEntry { id: string; title: string; status: string; version: number }
export interface Page<T> { items: T[]; total: number }
export interface Paging { limit?: number; offset?: number }
export class ApiError extends Error {
  constructor(public readonly status: number, public readonly code: string, message: string) {
    super(message); this.name = 'ApiError'
  }
}
type ClientOptions = {
  /** Versioned base, e.g. /api/v1 or https://school.example/api/v1. */
  baseUrl: string; getToken?: () => string | null | undefined
  onUnauthorized?: () => void; fetcher?: typeof fetch
}
export function createApiClient({baseUrl, getToken, onUnauthorized, fetcher = fetch}: ClientOptions) {
  const root = baseUrl.replace(/\/+$/, '')
  const key = encodeURIComponent
  async function request<T>(path: string, method = 'GET', body?: unknown, signal?: AbortSignal, authenticate = true): Promise<T> {
    const token = authenticate ? getToken?.() : null
    const headers: Record<string, string> = {Accept: 'application/json'}
    if (token) headers.Authorization = 'Bearer ' + token
    if (body !== undefined) headers['Content-Type'] = 'application/json'
    const response = await fetcher(root + path, {method, headers, signal,
      ...(body === undefined ? {} : {body: JSON.stringify(body)})})
    if (!response.ok) {
      if (response.status === 401 && token && token === getToken?.()) onUnauthorized?.()
      const payload = await response.json().catch(() => null)
      throw new ApiError(response.status, payload?.error?.code ?? 'http_error', payload?.error?.message ?? 'Request failed')
    }
    if (response.status === 204) return undefined as T
    return await response.json() as T
  }
  const paging = ({limit = 25, offset = 0}: Paging = {}) => '?limit=' + encodeURIComponent(limit) + '&offset=' + encodeURIComponent(offset)
  const workspacePath = (module: string) => '/modules/' + key(module) + '/workspace'
  return {
    login: (credentials: Credentials) => request<Session>('/session', 'POST', credentials, undefined, false),
    session: (signal?: AbortSignal) => request<Principal>('/session','GET',undefined,signal),
    logout: () => request<void>('/session','DELETE'),
    modules: (signal?: AbortSignal) => request<{items:ModuleDescriptor[]}>('/modules','GET',undefined,signal),
    module: (module: string,signal?: AbortSignal) => request<ModuleDescriptor>('/modules/' + key(module),'GET',undefined,signal),
    workspace: (module: string, page?: Paging, signal?: AbortSignal) =>
      request<Page<WorkspaceEntry> & {module: string}>(workspacePath(module) + paging(page),'GET',undefined,signal),
    createEntry: (module: string, data: {title: string}) => request<WorkspaceEntry>(workspacePath(module),'POST',data),
    updateEntry: (module: string, id: string, data: Pick<WorkspaceEntry,'title'|'version'|'status'>) =>
      request<WorkspaceEntry>(workspacePath(module) + '/' + key(id),'PATCH',data),
    students: (page?: Paging, signal?: AbortSignal) => request<Page<Student>>('/students' + paging(page),'GET',undefined,signal),
    student: (id:string, signal?:AbortSignal) => request<Student>('/students/' + key(id),'GET',undefined,signal),
    createStudent: (names: StudentNames) => request<Student>('/students','POST',names),
    updateStudent: (id: string, names: StudentNames & {version:number}) => request<Student>('/students/' + key(id),'PATCH',names),
    archiveStudent: (id: string, version: number) => request<void | Student>('/students/' + key(id) + '?version=' + encodeURIComponent(version),'DELETE')
  }
}
export type ApiClient = ReturnType<typeof createApiClient>
