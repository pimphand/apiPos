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
      permissions: Array.isArray(raw.permissions) ? raw.permissions : undefined,
      created_at: raw.created_at ?? raw.createdAt,
      updated_at: raw.updated_at ?? raw.updatedAt,
    };
  }

  /**
   * Fetch the actual permissions for the user's role from the backend and
   * attach them to the user object so that `can()` uses the real
   * role-permission assignment instead of the hardcoded defaults.
   */
  async function fetchAndAttachPermissions(): Promise<void> {
    if (!user.value?.role_id) return;
    try {
      const { roleService } = await import('@/services/role');
      const perms = await roleService.getPermissions(user.value.role_id);
      user.value = {
        ...user.value,
        permissions: perms.map((p) => p.name),
      };
      await authStorage.setUser(user.value);
    } catch (err) {
      // Log for debugging — the RBAC system depends on these permissions.
      // If this fails, can() falls back to hardcoded ROLE_PERMISSIONS which
      // may grant more access than intended.
      console.warn('[auth] fetchAndAttachPermissions failed:', err);
      // Set empty permissions array so can() knows we tried but got nothing,
      // rather than falling back to the hardcoded role map.
      if (user.value) {
        user.value = { ...user.value, permissions: [] };
        await authStorage.setUser(user.value);
      }
    }
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

      // Fetch actual permissions from backend so can() uses real role-permission assignment
      await fetchAndAttachPermissions();

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

      // Fetch actual permissions from backend so can() uses real role-permission assignment
      await fetchAndAttachPermissions();

      return userData;
    } catch (err: unknown) {
      const apiError = err as { response?: { data?: { message?: string } }; message?: string };
      error.value = apiError.response?.data?.message || apiError.message || 'Registration failed';
      throw err;
    } finally {
      isLoading.value = false;
    }
  }

  let refreshPromise: Promise<string | null> | null = null;
  let restorePromise: Promise<User | null> | null = null;

  function refreshAccessToken() {
    // Single-flight: if a refresh is already in progress, share it. The
    // backend rotates (revokes) the refresh token on use, so concurrent
    // refresh calls with the same token would race and the second one would
    // fail with 401 -> logout. Multiple guarded components (DashboardShell +
    // page guards) call restoreSession at once on reload, so this dedup is
    // required to survive a page reload.
    if (refreshPromise) {
      return refreshPromise;
    }

    if (!refreshToken.value) {
      refreshPromise = Promise.reject(new Error('No refresh token available'));
      refreshPromise = null;
      throw new Error('No refresh token available');
    }

    const pending = (async () => {
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

        // Re-fetch the current user so cached role/permissions stay in sync
        // with the backend (e.g. a role change or the initial role assignment
        // on a stale reload) instead of serving the stale persisted profile.
        const currentId = user.value?.id;
        if (currentId) {
          try {
            const { userService } = await import('@/services/user');
            const previousRole = user.value?.role_name;
            const fresh = await userService.get(currentId);
            const freshUser = normalizeUser(fresh as Record<string, any>);
            // Defensive: if the fetched profile omits the role, keep the
            // existing one so we never downgrade to "no permission".
            if (!freshUser.role_name && previousRole) {
              freshUser.role_name = previousRole;
            }
            user.value = freshUser;
            await authStorage.setUser(user.value);
          } catch (err) {
            // Profile refresh is best-effort; token refresh above still succeeded.
            // Keep the cached user so the app is still usable.
            console.warn('[auth] Failed to refresh user profile:', err);
          }
        }

        // Fetch actual permissions from backend so can() uses real
        // role-permission assignment instead of hardcoded defaults.
        // If this fails (e.g. network), fetchAndAttachPermissions sets
        // permissions to [] so can() correctly denies access rather than
        // falling back to the hardcoded role map.
        await fetchAndAttachPermissions();

        return access_token;
      } catch (err) {
        // Refresh failed - logout
        logout();
        throw err;
      }
    })();

    refreshPromise = pending;
    pending.finally(() => {
      if (refreshPromise === pending) {
        refreshPromise = null;
      }
    }).catch(() => {});

    return pending;
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

    // Single-flight: multiple guarded components (DashboardShell + page guard)
    // call restoreSession at once on reload. Share one restore result so they
    // don't each trigger their own refresh (the backend rotates refresh tokens).
    if (restorePromise) {
      return restorePromise;
    }

    restorePromise = (async () => {
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
    })();

    try {
      return await restorePromise;
    } finally {
      restorePromise = null;
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
