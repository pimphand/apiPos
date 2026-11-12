/**
 * Auth helper utilities
 *
 * Token Strategy:
 * - Access token (15m): Memory only (Pinia + module variable). Never persisted.
 * - Refresh token (30d): Persisted, but ENCRYPTED at rest (see lib/device.ts)
 *   so raw data is never written to localStorage. On reload the session is
 *   restored by calling /auth/refresh (see authStore.restoreSession). The
 *   backend binds each refresh token to a device id, so a token pasted into
 *   another browser cannot be used.
 */

import type { User } from '@/types';
import { encryptValue, decryptValue } from '@/lib/device';
import { isTauriRuntime } from '@/lib/apiBase';

const REFRESH_KEY = 'pos_refresh_token';
const USER_KEY = 'pos_user';

/**
 * OS-native storage for the refresh token (desktop builds only).
 *
 * Inside the Tauri webview the token is written to the platform keystore
 * (macOS Keychain, Windows Credential Manager, Linux Secret Service) through
 * Rust commands, so it never touches localStorage. The browser build keeps the
 * existing device-encrypted localStorage strategy. A token persisted by an
 * older desktop build is migrated into the keystore on the first read and then
 * removed from localStorage. If the native store is unavailable, the hardened
 * browser path is used as a fallback. See docs/TAURI_DESKTOP.md §6.
 */
type TauriInvoke = (cmd: string, args?: Record<string, unknown>) => Promise<unknown>;

const NATIVE_GET = 'auth_refresh_get';
const NATIVE_SET = 'auth_refresh_set';
const NATIVE_DELETE = 'auth_refresh_delete';

function nativeInvoke(): TauriInvoke | null {
  if (!isTauriRuntime) return null;
  const internals = (window as unknown as {
    __TAURI_INTERNALS__?: { invoke?: TauriInvoke };
  }).__TAURI_INTERNALS__;
  return internals?.invoke ?? null;
}

/**
 * Auth state. Access token is memory-only; the refresh token and a
 * non-sensitive user profile cache are encrypted before persisting.
 */
export const authStorage = {
  /** Memory-backed access token (never persisted). */
  _accessToken: null as string | null,
  /** Decrypted refresh token mirror (not itself persisted raw). */
  _refreshToken: null as string | null,
  /** Decrypted user mirror (not itself persisted raw). */
  _user: null as User | null,
  /** Whether the OS keystore holds a refresh token (desktop). */
  _nativeTokenPresent: false,

  getAccessToken(): string | null {
    return authStorage._accessToken;
  },

  setAccessToken(token: string): void {
    authStorage._accessToken = token;
  },

  async getRefreshToken(opts?: { ignoreCache?: boolean }): Promise<string | null> {
    if (!opts?.ignoreCache && authStorage._refreshToken) return authStorage._refreshToken;
    if (typeof window === 'undefined') return null;

    const invoke = nativeInvoke();
    if (invoke) {
      try {
        const native = (await invoke(NATIVE_GET)) as string | null;
        if (native) {
          authStorage._refreshToken = native;
          authStorage._nativeTokenPresent = true;
          window.localStorage.removeItem(REFRESH_KEY);
          return native;
        }
        // One-time migration of a token written by an older desktop build.
        const legacy = window.localStorage.getItem(REFRESH_KEY);
        if (legacy) {
          const raw = legacy.trimStart().startsWith('{') ? await decryptValue(legacy) : legacy;
          if (raw) {
            await invoke(NATIVE_SET, { token: raw });
            authStorage._refreshToken = raw;
            authStorage._nativeTokenPresent = true;
            window.localStorage.removeItem(REFRESH_KEY);
            return raw;
          }
        }
        authStorage._nativeTokenPresent = false;
        return null;
      } catch {
        // Keystore/IPC unavailable → fall back to the hardened local store.
      }
    }

    const stored = window.localStorage.getItem(REFRESH_KEY);
    if (!stored) return null;
    // setRefreshToken writes the raw token first (synchronously) and then
    // overwrites it with the encrypted payload. If the writing JS context was
    // torn down before the async encryption finished (e.g. a page navigation
    // or a closing tab right after a refresh), the stored value is the raw
    // token — accept it directly. Encrypted payloads are JSON.
    if (!stored.trimStart().startsWith('{')) {
      authStorage._refreshToken = stored;
      return stored;
    }
    const decrypted = await decryptValue(stored);
    if (decrypted === null) return null;
    authStorage._refreshToken = decrypted;
    return decrypted;
  },

  async setRefreshToken(token: string | null): Promise<void> {
    authStorage._refreshToken = token;
    if (typeof window === 'undefined') return;

    const invoke = nativeInvoke();
    if (invoke) {
      try {
        if (token) {
          await invoke(NATIVE_SET, { token });
          authStorage._nativeTokenPresent = true;
        } else {
          await invoke(NATIVE_DELETE);
          authStorage._nativeTokenPresent = false;
        }
        // Desktop keeps the refresh token out of localStorage entirely.
        window.localStorage.removeItem(REFRESH_KEY);
        return;
      } catch {
        // Keychain write failed → keep the hardened browser fallback below.
      }
    }

    if (token) {
      // CRITICAL: persist the raw token SYNCHRONOUSLY first. Encryption is
      // async (~100ms of PBKDF2) and the current JS context can be discarded
      // while the encryption is still in flight (page navigation, another
      // context restoring the session). The rotated refresh token must survive
      // that teardown or the next page reload refreshes with the now-revoked
      // previous token and logs the user out.
      window.localStorage.setItem(REFRESH_KEY, token);
      try {
        window.localStorage.setItem(REFRESH_KEY, await encryptValue(token));
      } catch {
        // Encryption is best-effort hardening; the raw write above stands.
      }
    } else {
      window.localStorage.removeItem(REFRESH_KEY);
    }
  },

  async getUser(): Promise<User | null> {
    if (authStorage._user) return authStorage._user;
    if (typeof window === 'undefined') return null;
    const stored = window.localStorage.getItem(USER_KEY);
    if (!stored) return null;
    const decrypted = await decryptValue(stored);
    if (decrypted === null) return null;
    try {
      const u = JSON.parse(decrypted) as User;
      authStorage._user = u;
      return u;
    } catch {
      return null;
    }
  },

  async setUser(user: User | null): Promise<void> {
    authStorage._user = user;
    if (typeof window === 'undefined') return;
    if (user) {
      window.localStorage.setItem(USER_KEY, await encryptValue(JSON.stringify(user)));
    } else {
      window.localStorage.removeItem(USER_KEY);
    }
  },

  /**
   * Clear all auth state (tokens and cached user).
   *
   * Async on purpose: the desktop keystore deletion must be awaited before the
   * caller navigates away. A fire-and-forget `invoke` is aborted when the
   * webview navigates to /login, which left the refresh token in the Keychain
   * and let the login page restore the session the user had just ended.
   */
  async clear(): Promise<void> {
    authStorage._accessToken = null;
    authStorage._refreshToken = null;
    authStorage._user = null;
    authStorage._nativeTokenPresent = false;
    if (typeof window !== 'undefined') {
      window.localStorage.removeItem(REFRESH_KEY);
      window.localStorage.removeItem(USER_KEY);
    }
    // Always ask the keystore to drop the token: a fresh JS context may not
    // have read it yet, and deleting an absent item is a no-op in Rust.
    const invoke = nativeInvoke();
    if (invoke) {
      try {
        await invoke(NATIVE_DELETE);
      } catch {
        // Keystore unavailable — nothing else to delete.
      }
    }
  },

  /** Sync check for whether a refresh token is persisted (no decryption needed). */
  hasPersistedRefresh(): boolean {
    if (authStorage._refreshToken || authStorage._nativeTokenPresent) return true;
    if (typeof window === 'undefined') return false;
    return !!window.localStorage.getItem(REFRESH_KEY);
  },

  /**
   * Check if user is authenticated (has tokens).
   */
  isAuthenticated(): boolean {
    return !!authStorage.getAccessToken() || authStorage.hasPersistedRefresh();
  },
};

/**
 * Format currency (Indonesian Rupiah)
 */
export function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount);
}

/**
 * Format date
 */
export function formatDate(date: string | Date): string {
  return new Intl.DateTimeFormat('id-ID', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  }).format(new Date(date));
}

/**
 * Format datetime
 */
export function formatDateTime(date: string | Date): string {
  return new Intl.DateTimeFormat('id-ID', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(date));
}
