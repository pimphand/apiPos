import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import axios from 'axios';
import http, { tokenManager, refreshTokens, cancelRefresh } from '@/lib/http';
import { API_BASE_URL } from '@/lib/apiBase';
import { authStorage } from '@/lib/auth';
import { realtime } from '@/lib/realtime';
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
  /**
   * True when the app booted without a connection and is running on the
   * encrypted profile cached on this device. Cleared as soon as a real token
   * refresh succeeds. Used to keep the POS usable offline (see
   * docs/pos-offline-sync.md); sensitive actions still require the server.
   */
  const offlineSession = ref(false);

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
   * Attach the permission names carried by the auth payload (login/register/
   * refresh responses) to the user object so that `can()` uses the real
   * role-permission assignment instead of hardcoded defaults.
   *
   * The backend embeds `permissions` in the auth responses deliberately — this
   * avoids a separate `GET /permissions/my` request from the browser, which
   * would otherwise be visible in the network tab (F12).
   *
   * Returns true when permissions were attached from the payload.
   */
  function attachPermissionsFromPayload(perms: string[] | undefined): boolean {
    if (!Array.isArray(perms) || !user.value) {
      return false;
    }
    user.value = { ...user.value, permissions: perms };
    authStorage.setUser(user.value);
    return true;
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

      // Permissions are embedded in the login payload (no separate request).
      attachPermissionsFromPayload((data.user ?? {})?.permissions);
      offlineSession.value = false;

      // Open the realtime socket now that we hold a valid access token.
      realtime.connect();

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

      // Permissions are embedded in the register payload (no separate request).
      attachPermissionsFromPayload((resData.user ?? {})?.permissions);
      offlineSession.value = false;

      // Open the realtime socket now that we hold a valid access token.
      realtime.connect();

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

  /**
   * Bumped on every logout. A refresh/restore that was started before the
   * user signed out compares this after its await and discards the result, so
   * an in-flight rotation cannot write the session back.
   */
  let sessionEpoch = 0;

  function refreshAccessToken() {
    if (refreshPromise) {
      return refreshPromise;
    }

    const pending = (async () => {
      const epoch = sessionEpoch;
      try {
        const refreshed = await refreshTokens();
        if (epoch !== sessionEpoch) {
          throw new Error('Session ended');
        }
        if (!refreshed) {
          // No refresh token available — this is a genuine session end, not a
          // race. Clear state but do NOT attempt a server logout (nothing to
          // revoke) and let the caller decide on navigation.
          throw new Error('No refresh token available');
        }
        const access_token = refreshed.accessToken;
        offlineSession.value = false;

        // refreshTokens() already persisted the rotated pair and updated
        // tokenManager/authStorage; mirror them into the store's refs.
        const newRefreshToken = tokenManager.getRefreshToken();
        accessToken.value = access_token;
        refreshToken.value = newRefreshToken;

        // Permissions are embedded in the refresh payload (no separate request).
        if (refreshed.permissions) {
          attachPermissionsFromPayload(refreshed.permissions);
        }

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
            // The profile endpoint does not carry permissions — keep the ones
            // from the refresh payload so can() keeps using the real assignment.
            if (refreshed.permissions) {
              freshUser.permissions = refreshed.permissions;
            }
            user.value = freshUser;
            await authStorage.setUser(user.value);
          } catch (err) {
            // Profile refresh is best-effort; token refresh above still succeeded.
            // Keep the cached user so the app is still usable.
            console.warn('[auth] Failed to refresh user profile:', err);
          }
        }

        return access_token;
      } catch (err) {
        // Refresh failed due to a race (token rotated by another context and
        // the fallback candidate also failed). Do NOT logout immediately —
        // the persisted token may still be valid and restoreSession() on the
        // next navigation will pick it up. Only logout on an explicit
        // "no refresh token" (genuine session end).
        const msg = err instanceof Error ? err.message : '';
        if (
          epoch === sessionEpoch &&
          (msg.includes('No refresh token') || msg.includes('No refresh token available'))
        ) {
          logout();
        }
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
      const epoch = sessionEpoch;
      const persistedRefresh = await authStorage.getRefreshToken();
      const cachedUser = await authStorage.getUser();

      if (!persistedRefresh) {
        return null;
      }

      // Stage the refresh token + cached profile so refreshAccessToken can run
      // and the guard has a user while the access token is being minted. The
      // module-level token is also set so that any stray 401 during restore
      // (e.g. a page request firing before the refresh completes) shares the
      // same single-flight refresh instead of force-logging-out.
      refreshToken.value = persistedRefresh;
      tokenManager.setRefreshToken(persistedRefresh);
      if (cachedUser) {
        user.value = cachedUser;
      }

      // The user signed out while the persisted token was being read.
      if (epoch !== sessionEpoch) return null;

      try {
        await refreshAccessToken();
        // Session restored: the socket may have been started before the token
        // existed (onMounted races restoreSession), so (re)connect now.
        realtime.connect();
        return user.value;
      } catch {
        // Logout raced the restore — never bring the session back.
        if (epoch !== sessionEpoch) return null;
        // No connection: a transport failure cannot prove the token is
        // invalid, so keep the cached profile and let the POS run offline.
        // When the browser is online and the refresh was rejected, the session
        // is genuinely over and we fall through to the login redirect.
        if (typeof navigator !== 'undefined' && navigator.onLine === false && cachedUser) {
          offlineSession.value = true;
          return user.value;
        }
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

  async function logout(): Promise<void> {
    // 1. Invalidate anything in flight, then clear local state. Callers await
    //    this before navigating, so the desktop keystore deletion completes
    //    while the page is still alive.
    sessionEpoch++;
    cancelRefresh();

    const access = accessToken.value ?? tokenManager.getAccessToken();
    const rt = refreshToken.value;

    user.value = null;
    accessToken.value = null;
    refreshToken.value = null;
    error.value = null;
    offlineSession.value = false;
    tokenManager.clearTokens();

    await authStorage.clear();

    // Close the realtime socket: without an access token the server would
    // reject a reconnect, and the user should not keep receiving store events.
    realtime.disconnect();

    // 2. Revoke the refresh token server-side. This runs after local cleanup
    //    (so a failing/slow request can never leave the user half-logged-in)
    //    with the captured Bearer token, and is bounded so logout stays snappy
    //    when the gateway is unreachable.
    if (rt && (typeof navigator === 'undefined' || navigator.onLine !== false)) {
      try {
        await axios.post(
          `${API_BASE_URL}/v1/auth/logout`,
          { refresh_token: rt, device_id: getDeviceId() },
          {
            headers: access ? { Authorization: `Bearer ${access}` } : undefined,
            timeout: 3000,
          },
        );
      } catch {
        // Offline, timed out, or already revoked: the local session is gone.
      }
    }
  }

  async function setUser(userData: User) {
    user.value = normalizeUser(userData);
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
    offlineSession,
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
