import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import http from '@/lib/http';
import { authStorage } from '@/lib/auth';
import type { User, LoginRequest, RegisterRequest } from '@/types';

export const useAuthStore = defineStore('auth', () => {
  // State - initialize from storage if available
  const user = ref<User | null>(authStorage.getUser());
  const accessToken = ref<string | null>(authStorage.getAccessToken());
  const refreshToken = ref<string | null>(authStorage.getRefreshToken());
  const isLoading = ref(false);
  const error = ref<string | null>(null);

  // Getters
  const isAuthenticated = computed(() => {
    return (!!accessToken.value || !!refreshToken.value) && user.value !== null;
  });
  const currentUser = computed(() => user.value);
  const userRole = computed(() => user.value?.role_id || null);
  const userName = computed(() => user.value?.full_name || user.value?.username || 'User');

  // Actions
  async function login(credentials: LoginRequest) {
    isLoading.value = true;
    error.value = null;

    try {
      const response = await http.post('/v1/auth/login', credentials);
      const { access_token, refresh_token, user: userData } = response.data;

      // Store tokens
      accessToken.value = access_token;
      refreshToken.value = refresh_token;
      authStorage.setAccessToken(access_token);
      authStorage.setRefreshToken(refresh_token);

      // Store user data
      user.value = userData;
      authStorage.setUser(userData);

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
      const response = await http.post('/v1/auth/register', data);
      const { access_token, refresh_token, user: userData } = response.data;

      // Store tokens
      accessToken.value = access_token;
      refreshToken.value = refresh_token;
      authStorage.setAccessToken(access_token);
      authStorage.setRefreshToken(refresh_token);

      // Store user data
      user.value = userData;
      authStorage.setUser(userData);

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
      });

      const { access_token, refresh_token: newRefreshToken } = response.data;

      // Update tokens
      accessToken.value = access_token;
      refreshToken.value = newRefreshToken;
      authStorage.setAccessToken(access_token);
      authStorage.setRefreshToken(newRefreshToken);

      return access_token;
    } catch (err) {
      // Refresh failed - logout
      logout();
      throw err;
    }
  }

  function logout() {
    // Clear state
    user.value = null;
    accessToken.value = null;
    refreshToken.value = null;
    error.value = null;

    // Clear storage
    authStorage.clear();

    // Call logout endpoint (fire and forget)
    http.post('/v1/auth/logout', {}).catch(() => {});
  }

  function setUser(userData: User) {
    user.value = userData;
    authStorage.setUser(userData);
  }

  function setTokens(access: string, refresh: string) {
    accessToken.value = access;
    refreshToken.value = refresh;
    authStorage.setAccessToken(access);
    authStorage.setRefreshToken(refresh);
  }

  function clearError() {
    error.value = null;
  }

  // Initialize from storage on store creation
  function init() {
    const storedUser = authStorage.getUser();
    const storedAccessToken = authStorage.getAccessToken();
    const storedRefreshToken = authStorage.getRefreshToken();

    if (storedUser) user.value = storedUser;
    if (storedAccessToken) accessToken.value = storedAccessToken;
    if (storedRefreshToken) refreshToken.value = storedRefreshToken;
  }

  // Run init
  init();

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
    logout,
    setUser,
    setTokens,
    clearError,
  };
});
