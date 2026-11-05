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

/** POST the refresh token and return the raw response data (throws on failure). */
async function postRefresh(rt: string): Promise<Record<string, unknown>> {
  const response = await axios.post(
    `${import.meta.env.PUBLIC_API_BASE_URL || '/api'}/v1/auth/refresh`,
    { refresh_token: rt, device_id: getDeviceId() }
  );
  return (response.data ?? {}) as Record<string, unknown>;
}

/**
 * Perform a token refresh and persist the rotated refresh token.
 *
 * The backend rotates (revokes) the presented refresh token on EVERY use, so a
 * stale in-memory copy fails with 401 once it has been consumed elsewhere
 * (another tab, or a short-lived context that booted the app and restored the
 * session). To stay correct we try the in-memory token first and, if it was
 * already rotated away, fall back to the persisted token — the latest rotation
 * is always persisted synchronously first (see authStorage), so the persisted
 * value is the most recent valid one across contexts.
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
    // Candidate tokens, most-recent first. The in-memory token is tried first;
    // if it was already rotated away (revoked) by another context, the
    // persisted one — always kept current by setRefreshToken — is tried next.
    const candidates: string[] = [];
    if (refreshToken) candidates.push(refreshToken);
    // Read the persisted token ignoring the in-memory mirror: another JS
    // context (e.g. the Astro transition iframe) may have rotated it and
    // updated localStorage while this context's mirror stayed stale.
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

        // CRITICAL: persist the rotated refresh token. setRefreshToken writes
        // the raw value synchronously BEFORE the async encryption, so the write
        // survives even if this JS context is torn down mid-refresh (another
        // tab closing, a page navigation) — otherwise the next page reload
        // refreshes with the now-revoked previous token and logs the user out.
        await authStorage.setRefreshToken(newRefreshToken);
        authStorage.setAccessToken(access_token);

        const result: RefreshResult = { accessToken: access_token, permissions };
        processQueue(null, result);
        return result;
      } catch (err) {
        // Token rejected (e.g. already rotated/revoked) → try the next
        // candidate. Any other error (network, 5xx) also falls through — if no
        // candidate succeeds the refresh is treated as failed below.
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