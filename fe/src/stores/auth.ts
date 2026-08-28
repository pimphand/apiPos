import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import http, { tokenManager } from '@/lib/http';
import type { User, LoginRequest, RegisterRequest } from '@/types';

export const useAuthStore = defineStore('auth', () => {
  // State
  const user = ref<User | null>(null);
  const isLoading = ref(false);
  const error = ref<string | null>(null);

  // Getters
  const isAuthenticated = computed(() => tokenManager.isAuthenticated() && user.value !== null);
  const currentUser = computed(() => user.value);
  const userRole = computed(() => user.value?.role_id || null);

  // Actions
  async function login(credentials: LoginRequest) {
    isLoading.value = true;
    error.value = null;

    try {
      const response = await http.post('/v1/auth/login', credentials);
      const { access_token, refresh_token, user: userData } = response.data;

      // Store tokens in memory only
      tokenManager.setTokens(access_token, refresh_token);

      // Store user data
      user.value = userData;

      return userData;
    } catch (err: unknown) {
      const apiError = err as { message?: string };
      error.value = apiError.message || 'Login failed';
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

      // Store tokens in memory only
      tokenManager.setTokens(access_token, refresh_token);

      // Store user data
      user.value = userData;

      return userData;
    } catch (err: unknown) {
      const apiError = err as { message?: string };
      error.value = apiError.message || 'Registration failed';
      throw err;
    } finally {
      isLoading.value = false;
    }
  }

  async function logout() {
    try {
      await http.post('/v1/auth/logout', {});
    } catch {
      // Ignore logout errors
    } finally {
      tokenManager.clearTokens();
      user.value = null;
    }
  }

  function setUser(userData: User) {
    user.value = userData;
  }

  function clearError() {
    error.value = null;
  }

  return {
    // State
    user,
    isLoading,
    error,
    // Getters
    isAuthenticated,
    currentUser,
    userRole,
    // Actions
    login,
    register,
    logout,
    setUser,
    clearError,
  };
});
