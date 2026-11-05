import type { ApiClient, Credentials, Principal } from '@edupilot/api-client';

export interface TokenStore {
  read(): Promise<string | null>;
  write(token: string): Promise<void>;
  remove(): Promise<void>;
}
export type Reason =
  | 'credentials'
  | 'expired'
  | 'network'
  | 'storage'
  | 'remoteLogout'
  | 'forbidden'
  | 'invalid'
  | 'conflict';
type LoginState = { kind: 'guest'; reason?: Reason };
export type SessionState =
  | LoginState
  | { kind: 'checking' | 'signingIn' | 'signingOut' | 'retry' | 'erase' }
  | { kind: 'member'; principal: Principal };
export function statusOf(error: unknown) {
  return typeof error === 'object' && error !== null && 'status' in error
    ? Number(error.status)
    : 0;
}
export function feedback(error: unknown): Reason {
  switch (statusOf(error)) {
    case 401:
      return 'expired';
    case 403:
      return 'forbidden';
    case 400:
    case 422:
      return 'invalid';
    case 409:
      return 'conflict';
    default:
      return 'network';
  }
}

type AuthApi = Pick<ApiClient, 'session' | 'logout' | 'login'>;
/** A single authority for in-memory identity and secured storage. During
 * recovery and logout, the UI never renders private data from a cached principal. */
export class SessionModel {
  private token: string | null = null;
  private state: SessionState = { kind: 'checking' };
  private listeners = new Set<() => void>();
  private epoch = 0;
  private api: AuthApi;
  constructor(
    private store: TokenStore,
    makeApi: (getToken: () => string | null, onExpired: () => void) => AuthApi,
  ) {
    this.api = makeApi(this.getToken, this.onExpired);
  }
  getToken = () => this.token;
  snapshot = () => this.state;
  subscribe = (fn: () => void) => {
    this.listeners.add(fn);
    return () => {
      this.listeners.delete(fn);
    };
  };
  private set(next: SessionState) {
    this.state = next;
    this.listeners.forEach((fn) => fn());
  }
  private async forget(reason?: Reason) {
    this.epoch++;
    this.token = null;
    this.set({ kind: 'signingOut' });
    try {
      await this.store.remove();
      this.set({ kind: 'guest', reason });
    } catch {
      this.set({ kind: 'erase' });
    }
  }
  onExpired = () => {
    // Recovery handles its own 401. During sign-out the captured DELETE must
    // complete before deciding what to tell the user about revocation.
    if (this.state.kind === 'member') void this.forget('expired');
  };
  async restore() {
    const epoch = ++this.epoch;
    this.set({ kind: 'checking' });
    try {
      const token = await this.store.read();
      if (this.epoch !== epoch) return;
      if (!token) {
        this.token = null;
        this.set({ kind: 'guest' });
        return;
      }
      this.token = token;
      const principal = await this.api.session();
      if (this.epoch === epoch) this.set({ kind: 'member', principal });
    } catch (error) {
      if (this.epoch !== epoch) return;
      if (statusOf(error) === 401) await this.forget('expired');
      else {
        this.token = null;
        this.set({ kind: 'retry' });
      }
    }
  }
  async login(form: Credentials) {
    if (this.state.kind !== 'guest') return;
    this.set({ kind: 'signingIn' });
    try {
      const { token, principal } = await this.api.login(form);
      this.token = token;
      try {
        await this.store.write(token);
      } catch {
        try {
          await this.api.logout();
        } catch {
          /* expires at the server */
        }
        await this.forget('storage');
        return;
      }
      this.set({ kind: 'member', principal });
    } catch (error) {
      this.token = null;
      this.set({
        kind: 'guest',
        reason: statusOf(error) === 401 ? 'credentials' : feedback(error),
      });
    }
  }
  async logout() {
    if (this.state.kind === 'retry' || this.state.kind === 'erase') {
      // No verified live session. Only the local token can be discarded.
      await this.forget(
        this.state.kind === 'retry' ? 'remoteLogout' : undefined,
      );
      return;
    }
    if (this.state.kind !== 'member') return;
    this.set({ kind: 'signingOut' });
    let warning: Reason | undefined;
    try {
      await this.api.logout();
    } catch (error) {
      if (statusOf(error) !== 401) warning = 'remoteLogout';
    }
    await this.forget(warning);
  }
}
