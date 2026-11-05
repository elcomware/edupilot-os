import * as SecureStore from 'expo-secure-store';
import type { TokenStore } from './session';

const key = 'edupilot.session.v1';
const options: SecureStore.SecureStoreOptions = {
  keychainAccessible: SecureStore.WHEN_UNLOCKED_THIS_DEVICE_ONLY,
  keychainService: 'org.edupilot.mobile.session',
};

// A changed API origin must never inherit another server's bearer token.
export function tokenStore(apiRoot: string): TokenStore {
  return {
    async read() {
      const raw = await SecureStore.getItemAsync(key, options);
      if (!raw) return null;
      let old: unknown;
      try {
        old = JSON.parse(raw);
      } catch {
        /* discard malformed records */
      }
      if (
        typeof old === 'object' &&
        old !== null &&
        'root' in old &&
        old.root === apiRoot &&
        'token' in old &&
        typeof old.token === 'string' &&
        old.token
      )
        return old.token;
      await SecureStore.deleteItemAsync(key, options);
      return null;
    },
    write: (token) =>
      SecureStore.setItemAsync(
        key,
        JSON.stringify({ root: apiRoot, token }),
        options,
      ),
    remove: () => SecureStore.deleteItemAsync(key, options),
  };
}
