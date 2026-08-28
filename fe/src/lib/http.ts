import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';
import type { ApiError, RefreshTokenResponse } from '@/types';

// Access token stored in memory only (not localStorage for XSS protection)
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

// Token management
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

    // If 401 and not already retrying
    if (error.response?.status === 401 && !originalRequest._retry) {
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
        window.location.href = '/login';
        return Promise.reject(error);
      }

      try {
        // Try to refresh the token
        const response = await axios.post<RefreshTokenResponse>(
          `${import.meta.env.PUBLIC_API_BASE_URL || '/api'}/v1/auth/refresh`,
          { refresh_token: refreshToken }
        );

        const { access_token, refresh_token: newRefreshToken } = response.data;
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
        window.location.href = '/login';
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
