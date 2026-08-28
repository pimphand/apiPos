import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';
import type { ApiError } from '@/types';
import { navigateTo } from '@/lib/navigation';

// Access token stored in memory (synced with auth store)
let accessToken: string | null = null;
let refreshToken: string | null = null;

// Queue for concurrent requests during token refresh
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (value: unknown) => void;
  reject: (reason?: unknown) => void;
}> = [];

const processQueue = (error: Error | null, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token);
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

  clearTokens: () => {
    accessToken = null;
    refreshToken = null;
  },

  isAuthenticated: () => !!accessToken,
};

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
      // If already refreshing, queue this request
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        })
          .then((token) => {
            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${token}`;
            }
            return http(originalRequest);
          })
          .catch((err) => Promise.reject(err));
      }

      originalRequest._retry = true;
      isRefreshing = true;

      // If no refresh token, reject immediately
      if (!refreshToken) {
        isRefreshing = false;
        tokenManager.clearTokens();
        if (typeof window !== 'undefined') {
          navigateTo('/login');
        }
        return Promise.reject(error);
      }

      try {
        // Try to refresh the token
        const response = await axios.post(
          `${import.meta.env.PUBLIC_API_BASE_URL || '/api'}/v1/auth/refresh`,
          { refresh_token: refreshToken }
        );

        const resData = response.data ?? {};
        const access_token = resData.access_token ?? resData.accessToken;
        const newRefreshToken = resData.refresh_token ?? resData.refreshToken;
        tokenManager.setTokens(access_token, newRefreshToken);

        // Retry original request
        if (originalRequest.headers) {
          originalRequest.headers.Authorization = `Bearer ${access_token}`;
        }

        processQueue(null, access_token);
        return http(originalRequest);
      } catch (refreshError) {
        // Refresh failed - logout
        processQueue(refreshError as Error, null);
        tokenManager.clearTokens();
        if (typeof window !== 'undefined') {
          navigateTo('/login');
        }
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
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
