import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';
import type { ApiError } from '@/types';
import { navigateTo } from '@/lib/navigation';
import { getDeviceId } from '@/lib/device';
import { authStorage } from '@/lib/auth';

// Access token stored in memory (synced with auth store)
let accessToken: string | null = null;
let refreshToken: string | null = null;

// Queue for concurrent requests during token refresh
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value: unknown) => void;
  reject: (reason?: unknown) => void;
}> = [];

const processQueue = (error: Error | null, result: RefreshResult | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(result);
    }
  });
  failedQueue = [];
};

// Create axios instance
const http = axios.create({
  baseURL: import.meta.env.PUBLIC_API_BASE_URL || '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Token management (used by auth store)
export const tokenManager = {
  getAccessToken: () => accessToken,
  getRefreshToken: () => refreshToken,

  setTokens: (access: string, refresh: string) => {
    accessToken = access;
    refreshToken = refresh;
  },

  /** Set only the refresh token (used when restoring a persisted session so
   * concurrent 401-handlers during restore can piggyback on the same refresh). */
  setRefreshToken: (refresh: string) => {
    refreshToken = refresh;
  },

  clearTokens: () => {
    accessToken = null;
    refreshToken = null;
  },

  isAuthenticated: () => !!accessToken,
};

/** Result of a token refresh. */
export interface RefreshResult {
  accessToken: string;
  /** Permission names embedded by the backend in the refresh payload. */
  permissions?: string[];
}

/**
 * Perform a token refresh and persist the rotated refresh token.
 *
 * This is the SINGLE refresh path for both the axios 401 interceptor and the
 * auth store (`refreshAccessToken`). It is single-flight: concurrent callers
 * (e.g. multiple tabs/components restoring a session at once) share one refresh
 * request instead of racing and revoking each other's tokens — the backend
 * rotates (revokes) the presented refresh token on every use.
 *
 * Returns the new access token (plus embedded permissions), or null if no
 * refresh token exists / refresh failed. On success the rotated refresh token
 * is persisted to authStorage so a page reload after an interceptor-driven
 * refresh does NOT log the user out.
 */
export async function refreshTokens(): Promise<RefreshResult | null> {
  console.log('[DBG refreshTokens] entry isRefreshing=', isRefreshing, 'moduleRT=', refreshToken ? refreshToken.slice(0, 12) + '...' : null);
  if (isRefreshing) {
    return new Promise<RefreshResult | null>((resolve, reject) => {
      failedQueue.push({
        resolve: (value) => resolve(value as RefreshResult | null),
        reject,
      });
    });
  }

  isRefreshing = true;
  try {
    // Fall back to the persisted refresh token (e.g. right after a page reload
    // before restoreSession has run) so a stray 401 doesn't force a logout.
    let rt = refreshToken;
    if (!rt) {
      rt = await authStorage.getRefreshToken();
      if (!rt) return null;
      refreshToken = rt;
    }
    console.log('[DBG refreshTokens] using rt=', rt.slice(0, 12) + '...');

    const response = await axios.post(
      `${import.meta.env.PUBLIC_API_BASE_URL || '/api'}/v1/auth/refresh`,
      { refresh_token: rt, device_id: getDeviceId() }
    );

    const resData = response.data ?? {};
    const access_token = resData.access_token ?? resData.accessToken;
    const newRefreshToken = resData.refresh_token ?? resData.refreshToken;
    const permissions = Array.isArray(resData.permissions)
      ? resData.permissions.map(String)
      : undefined;

    accessToken = access_token;
    refreshToken = newRefreshToken;
    console.log('[DBG refreshTokens] SUCCESS newRT=', newRefreshToken.slice(0, 12) + '...');

    // CRITICAL: persist the rotated refresh token. Without this, an
    // interceptor-driven refresh leaves the old (revoked) token in storage and
    // the next page reload fails to refresh -> logout.
    await authStorage.setRefreshToken(newRefreshToken);
    authStorage.setAccessToken(access_token);

    const result: RefreshResult = { accessToken: access_token, permissions };
    processQueue(null, result);
    return result;
  } catch (err) {
    console.log('[DBG refreshTokens] FAILED', (err as Error)?.message);
    processQueue(err as Error, null);
    return null;
  } finally {
    isRefreshing = false;
  }
}

// Request interceptor - add auth header
http.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    if (accessToken) {
      config.headers.Authorization = `Bearer ${accessToken}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor - handle 401 and refresh token
http.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & {
      _retry?: boolean;
    };

    // 401 on public auth endpoints means invalid credentials, NOT an expired
    // token. Let the error bubble up so the UI can show it instead of treating
    // it as a session expiry and redirecting.
    const isAuthEndpoint = (url?: string) =>
      !!url &&
      (url.includes('/v1/auth/login') ||
        url.includes('/v1/auth/register') ||
        url.includes('/v1/auth/refresh'));

    // If 401 and not already retrying
    if (
      error.response?.status === 401 &&
      !originalRequest._retry &&
      !isAuthEndpoint(originalRequest.url)
    ) {
      console.log('[DBG interceptor] 401 on', originalRequest.url, 'retrying');
      originalRequest._retry = true;

      // Single shared refresh (single-flight — safe for concurrent callers).
      const refreshed = await refreshTokens();

      if (!refreshed) {
        // Refresh failed - clear state and send to login
        tokenManager.clearTokens();
        if (typeof window !== 'undefined') {
          navigateTo('/login');
        }
        return Promise.reject(error);
      }

      // Retry original request
      if (originalRequest.headers) {
        originalRequest.headers.Authorization = `Bearer ${refreshed.accessToken}`;
      }
      return http(originalRequest);
    }

    // Map error to typed format
    const apiError: ApiError = {
      code: 'INTERNAL',
      message: 'An unexpected error occurred',
      status: error.response?.status,
    };

    if (error.response?.data) {
      const data = error.response.data as Record<string, unknown>;
      if (typeof data.message === 'string') {
        apiError.message = data.message;
      }
      if (typeof data.code === 'string') {
        apiError.code = data.code;
      }
    }

    // Map HTTP status to gRPC-style code
    switch (error.response?.status) {
      case 400:
        apiError.code = 'INVALID_ARGUMENT';
        break;
      case 401:
        apiError.code = 'UNAUTHENTICATED';
        break;
      case 403:
        apiError.code = 'PERMISSION_DENIED';
        break;
      case 404:
        apiError.code = 'NOT_FOUND';
        break;
      case 409:
        apiError.code = 'ALREADY_EXISTS';
        break;
      case 500:
        apiError.code = 'INTERNAL';
        break;
    }

    return Promise.reject(apiError);
  }
);

export default http;