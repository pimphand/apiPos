/**
 * Auth helper utilities
 * 
 * Token Strategy (from Frontend.md):
 * - Access token (15m): Memory only (Pinia + module variable)
 * - Refresh token (30d): localStorage for persistence across page reloads
 * 
 * Security Note: 
 * - Access token is never persisted (XSS protection)
 * - Refresh token is persisted but only sent to /auth/refresh endpoint
 * - On page reload, user must re-authenticate if access token is lost
 */

import type { User } from '@/types';

const ACCESS_TOKEN_KEY = 'pos_access_token';
const REFRESH_TOKEN_KEY = 'pos_refresh_token';
const USER_KEY = 'pos_user';

/**
 * Token manager for auth state
 */
export const authStorage = {
  /**
   * Get access token (memory only in production)
   */
  getAccessToken(): string | null {
    if (typeof window === 'undefined') return null;
    return sessionStorage.getItem(ACCESS_TOKEN_KEY);
  },

  /**
   * Set access token (session storage for page refresh survival)
   */
  setAccessToken(token: string): void {
    if (typeof window === 'undefined') return;
    sessionStorage.setItem(ACCESS_TOKEN_KEY, token);
  },

  /**
   * Get refresh token (localStorage for long-term persistence)
   */
  getRefreshToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem(REFRESH_TOKEN_KEY);
  },

  /**
   * Set refresh token (localStorage)
   */
  setRefreshToken(token: string): void {
    if (typeof window === 'undefined') return;
    localStorage.setItem(REFRESH_TOKEN_KEY, token);
  },

  /**
   * Get stored user data
   */
  getUser(): User | null {
    if (typeof window === 'undefined') return null;
    const data = localStorage.getItem(USER_KEY);
    if (!data) return null;
    try {
      return JSON.parse(data) as User;
    } catch {
      return null;
    }
  },

  /**
   * Set user data
   */
  setUser(user: User): void {
    if (typeof window === 'undefined') return;
    localStorage.setItem(USER_KEY, JSON.stringify(user));
  },

  /**
   * Clear all auth data
   */
  clear(): void {
    if (typeof window === 'undefined') return;
    sessionStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
  },

  /**
   * Check if user is authenticated (has tokens)
   */
  isAuthenticated(): boolean {
    return !!this.getAccessToken() || !!this.getRefreshToken();
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
