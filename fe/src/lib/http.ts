import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';
import type { ApiError } from '@/types';
import { API_BASE_URL } from '@/lib/apiBase';
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

const http = axios.create({
  baseURL: API_BASE_URL,
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

/** POST the refresh token and return the raw response data (throws on failure). */
async function postRefresh(rt: string): Promise<Record<string, unknown>> {
  const response = await axios.post(
    `${API_BASE_URL}/v1/auth/refresh`,
    { refresh_token: rt, device_id: getDeviceId() }
  );
  return (response.data ?? {}) as Record<string, unknown>;
}

/**
 * Perform a token refresh and persist the rotated refresh token.
 *
 * NOTE: the backend rotates (revokes) the presented refresh token on EVERY
 * use. Two concurrent refreshes with the same token would race: the winner
 * issues a new token, the loser gets 401 and must NOT trigger logout —
 * otherwise a harmless double-refresh (e.g. AuthBootstrap + page guard firing
 * RestoreSession at once on a full-page navigation away from another tab)
 * logs the user out randomly.
 *
 * Strategy here: single in-flight refresh is shared via the queue, and
 * candidates are retried in order. The in-memory token is tried first; the
 * persisted token is tried next (the persisted value is always the most
 * recent valid token, because setRefreshToken writes it synchronously).
 */
export async function refreshTokens(): Promise<RefreshResult | null> {
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
    const candidates: string[] = [];
    if (refreshToken) candidates.push(refreshToken);
    const persisted = await authStorage.getRefreshToken({ ignoreCache: true });
    if (persisted && !candidates.includes(persisted)) candidates.push(persisted);
    if (candidates.length === 0) return null;

    let lastErr: unknown = null;
    for (const rt of candidates) {
      try {
        const resData = await postRefresh(rt);
        const access_token = (resData.access_token ?? resData.accessToken) as string | undefined;
        const newRefreshToken = (resData.refresh_token ?? resData.refreshToken) as string | undefined;
        const permissions = Array.isArray(resData.permissions)
          ? resData.permissions.map(String)
          : undefined;
        if (!access_token || !newRefreshToken) {
          throw new Error('Refresh response missing tokens');
        }

        accessToken = access_token;
        refreshToken = newRefreshToken;

        // CRITICAL: persist the rotated refresh token SYNCHRONOUSLY FIRST
        // (before the async encryption finishes) so that a page navigation
        // or tab close that tears down this JS context mid-refresh cannot
        // lose the only valid token. See authStorage.setRefreshToken.
        await authStorage.setRefreshToken(newRefreshToken);
        authStorage.setAccessToken(access_token);

        const result: RefreshResult = { accessToken: access_token, permissions };
        processQueue(null, result);
        return result;
      } catch (err) {
        lastErr = err;
      }
    }

    processQueue(lastErr as Error, null);
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
      originalRequest._retry = true;

      // Single shared refresh (single-flight — safe for concurrent callers).
      const refreshed = await refreshTokens();

      if (!refreshed) {
        // Refresh failed: the refresh token was already rotated away (common
        // when a second tab or a navigation races the first refresh). Do NOT
        // navigate to /login or clear tokens here — the auth store's
        // restoreSession() will re-run once the rotation settles and will
        // either restore the session or log out cleanly. Navigating here
        // causes the random "clicked 5x -> logout" bug: the first failed
        // refresh wipes state while a second, valid token is still in
        // localStorage waiting to be picked up on the next navigation.
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

    // No HTTP response at all: the gateway is unreachable (offline, DNS,
    // timeout). Tagged separately so the POS can queue the sale locally
    // instead of dropping it, while real 4xx/5xx errors keep their code.
    if (!error.response) {
      apiError.code = 'UNAVAILABLE';
      apiError.message = 'Network unavailable';
      apiError.status = undefined;
      return Promise.reject(apiError);
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