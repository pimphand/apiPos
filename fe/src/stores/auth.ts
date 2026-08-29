import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import http, { tokenManager } from '@/lib/http';
import { authStorage } from '@/lib/auth';
import { getDeviceId } from '@/lib/device';
import type { User, LoginRequest, RegisterRequest } from '@/types';

export const useAuthStore = defineStore('auth', () => {
  // State - memory only (no persistence). On page reload the session is lost
  // and the user must re-authenticate (see lib/auth.ts security note).
  const user = ref<User | null>(null);
  const accessToken = ref<string | null>(null);
  const refreshToken = ref<string | null>(null);
  const isLoading = ref(false);
  const error = ref<string | null>(null);

  // Getters
  const isAuthenticated = computed(() => {
    return (!!accessToken.value || !!refreshToken.value) && user.value !== null;
  });
  const currentUser = computed(() => user.value);
  const userRole = computed(() => user.value?.role_id || null);
  const userName = computed(() => user.value?.full_name || user.value?.username || 'User');

  // Normalize user object from API (accepts camelCase or snake_case)
  function normalizeUser(raw: Record<string, any>): User {
    return {
      id: raw.id ?? '',
      username: raw.username ?? '',
      email: raw.email ?? '',
      full_name: raw.full_name ?? raw.fullName ?? '',
      role_id: raw.role_id ?? raw.roleId ?? '',
      role_name: raw.role_name ?? raw.roleName ?? '',
      store_id: raw.store_id ?? raw.storeId ?? '',
      is_active: raw.is_active ?? raw.isActive ?? true,
      created_at: raw.created_at ?? raw.createdAt,
      updated_at: raw.updated_at ?? raw.updatedAt,
    };
  }

  // Actions
  async function login(credentials: LoginRequest) {
    isLoading.value = true;
    error.value = null;

    try {
      const response = await http.post('/v1/auth/login', {
        ...credentials,
        device_id: getDeviceId(),
      });
      const data = response.data ?? {};
      const access_token = data.access_token ?? data.accessToken;
      const refresh_token = data.refresh_token ?? data.refreshToken;
      const userData = normalizeUser(data.user ?? {});

      // Store tokens (refresh persisted encrypted)
      accessToken.value = access_token;
      refreshToken.value = refresh_token;
      tokenManager.setTokens(access_token, refresh_token);
      authStorage.setAccessToken(access_token);
      await authStorage.setRefreshToken(refresh_token);

      // Store user data
      user.value = userData;
      await authStorage.setUser(userData);

      return userData;
    } catch (err: unknown) {
      const apiError = err as { response?: { data?: { message?: string } }; message?: string };
      error.value = apiError.response?.data?.message || apiError.message || 'Login failed';
      throw err;
    } finally {
      isLoading.value = false;
    }
  }

  async function register(data: RegisterRequest) {
    isLoading.value = true;
    error.value = null;

    try {
      const response = await http.post('/v1/auth/register', {
        ...data,
        device_id: getDeviceId(),
      });
      const resData = response.data ?? {};
      const access_token = resData.access_token ?? resData.accessToken;
      const refresh_token = resData.refresh_token ?? resData.refreshToken;
      const userData = normalizeUser(resData.user ?? {});

      // Store tokens
      accessToken.value = access_token;
      refreshToken.value = refresh_token;
      tokenManager.setTokens(access_token, refresh_token);
      authStorage.setAccessToken(access_token);
      await authStorage.setRefreshToken(refresh_token);

      // Store user data
      user.value = userData;
      await authStorage.setUser(userData);

      return userData;
    } catch (err: unknown) {
      const apiError = err as { response?: { data?: { message?: string } }; message?: string };
      error.value = apiError.response?.data?.message || apiError.message || 'Registration failed';
      throw err;
    } finally {
      isLoading.value = false;
    }
  }

  async function refreshAccessToken() {
    if (!refreshToken.value) {
      throw new Error('No refresh token available');
    }

    try {
      const response = await http.post('/v1/auth/refresh', {
        refresh_token: refreshToken.value,
        device_id: getDeviceId(),
      });

      const resData = response.data ?? {};
      const access_token = resData.access_token ?? resData.accessToken;
      const newRefreshToken = resData.refresh_token ?? resData.refreshToken;

      // Update tokens (mirrors the rotated refresh token to localStorage)
      accessToken.value = access_token;
      refreshToken.value = newRefreshToken;
      tokenManager.setTokens(access_token, newRefreshToken);
      authStorage.setAccessToken(access_token);
      await authStorage.setRefreshToken(newRefreshToken);

      return access_token;
    } catch (err) {
      // Refresh failed - logout
      logout();
      throw err;
    }
  }

  /**
   * Restore a persisted session on page reload.
   *
   * Only the refresh token is persisted (see lib/auth.ts). This reads it back,
   * stages the cached user profile, then calls /auth/refresh to mint a fresh
   * in-memory access token. Idempotent — safe to call on every guarded page.
   */
  async function restoreSession(): Promise<User | null> {
    // Already have a live session in memory; nothing to restore.
    if (isAuthenticated.value && !!accessToken.value) {
      return user.value;
    }

    const persistedRefresh = await authStorage.getRefreshToken();
    const cachedUser = await authStorage.getUser();

    if (!persistedRefresh) {
      return null;
    }

    // Stage the refresh token + cached profile so refreshAccessToken can run
    // and the guard has a user while the access token is being minted.
    refreshToken.value = persistedRefresh;
    if (cachedUser) {
      user.value = cachedUser;
    }

    try {
      await refreshAccessToken();
      return user.value;
    } catch {
      // refreshAccessToken already clears local state on failure.
      return null;
    }
  }

  function logout() {
    // Revoke the refresh token server-side BEFORE clearing local state so the
    // request still carries the Bearer access token (backend requires auth).
    http
      .post('/v1/auth/logout', {
        refresh_token: refreshToken.value,
        device_id: getDeviceId(),
      })
      .catch(() => {});

    // Clear state
    user.value = null;
    accessToken.value = null;
    refreshToken.value = null;
    error.value = null;
    tokenManager.clearTokens();

    // Clear memory storage
    authStorage.clear();
  }

  async function setUser(userData: User) {
    user.value = normalizeUser(userData as any);
    await authStorage.setUser(user.value);
  }

  async function setTokens(access: string, refresh: string) {
    accessToken.value = access;
    refreshToken.value = refresh;
    tokenManager.setTokens(access, refresh);
    authStorage.setAccessToken(access);
    await authStorage.setRefreshToken(refresh);
  }

  function clearError() {
    error.value = null;
  }

  return {
    // State
    user,
    accessToken,
    refreshToken,
    isLoading,
    error,
    // Getters
    isAuthenticated,
    currentUser,
    userRole,
    userName,
    // Actions
    login,
    register,
    refreshAccessToken,
    restoreSession,
    logout,
    setUser,
    setTokens,
    clearError,
  };
});
