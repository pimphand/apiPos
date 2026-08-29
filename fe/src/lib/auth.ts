/**
 * Auth helper utilities
 *
 * Token Strategy (from Frontend.md §2.2 / §5.1):
 * - Access token (15m): Memory only (Pinia + module variable). Never persisted.
 * - Refresh token (30d): Memory only.
 *
 * Security Note:
 * The backend grpc-gateway returns tokens in the JSON body (no HttpOnly cookie),
 * so to avoid XSS token theft we keep BOTH tokens in JS memory only. Nothing is
 * written to localStorage/sessionStorage. Consequence: a page reload clears the
 * session and the user must re-authenticate — an accepted trade-off documented
 * in Frontend.md.
 */

import type { User } from '@/types';

/**
 * In-memory auth state. Scope is the page session; cleared on reload.
 */
export const authStorage = {
  /** Memory-backed access token (never persisted). */
  _accessToken: null as string | null,
  /** Memory-backed refresh token (never persisted). */
  _refreshToken: null as string | null,
  /** Memory-backed current user (never persisted). */
  _user: null as User | null,

  getAccessToken(): string | null {
    return authStorage._accessToken;
  },

  setAccessToken(token: string): void {
    authStorage._accessToken = token;
  },

  getRefreshToken(): string | null {
    return authStorage._refreshToken;
  },

  setRefreshToken(token: string): void {
    authStorage._refreshToken = token;
  },

  getUser(): User | null {
    return authStorage._user;
  },

  setUser(user: User): void {
    authStorage._user = user;
  },

  /**
   * Clear all auth state (both tokens and cached user).
   */
  clear(): void {
    authStorage._accessToken = null;
    authStorage._refreshToken = null;
    authStorage._user = null;
  },

  /**
   * Check if user is authenticated (has tokens).
   */
  isAuthenticated(): boolean {
    return !!authStorage.getAccessToken() || !!authStorage.getRefreshToken();
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
