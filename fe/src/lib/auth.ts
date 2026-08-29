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

const REFRESH_KEY = 'pos_refresh_token';
const USER_KEY = 'pos_user';

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

  getAccessToken(): string | null {
    return authStorage._accessToken;
  },

  setAccessToken(token: string): void {
    authStorage._accessToken = token;
  },

  async getRefreshToken(): Promise<string | null> {
    if (authStorage._refreshToken) return authStorage._refreshToken;
    if (typeof window === 'undefined') return null;
    const stored = window.localStorage.getItem(REFRESH_KEY);
    if (!stored) return null;
    const decrypted = await decryptValue(stored);
    if (decrypted === null) return null;
    authStorage._refreshToken = decrypted;
    return decrypted;
  },

  async setRefreshToken(token: string | null): Promise<void> {
    authStorage._refreshToken = token;
    if (typeof window === 'undefined') return;
    if (token) {
      window.localStorage.setItem(REFRESH_KEY, await encryptValue(token));
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
   */
  clear(): void {
    authStorage._accessToken = null;
    authStorage._refreshToken = null;
    authStorage._user = null;
    if (typeof window !== 'undefined') {
      window.localStorage.removeItem(REFRESH_KEY);
      window.localStorage.removeItem(USER_KEY);
    }
  },

  /** Sync check for whether a refresh token is persisted (no decryption needed). */
  hasPersistedRefresh(): boolean {
    if (authStorage._refreshToken) return true;
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
