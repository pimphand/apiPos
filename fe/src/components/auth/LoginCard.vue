<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const authStore = useAuthStore();

const username = ref('');
const password = ref('');
const isLoading = ref(false);
const error = ref('');

async function handleLogin() {
  if (!username.value || !password.value) {
    error.value = 'Please enter username and password';
    return;
  }

  isLoading.value = true;
  error.value = '';

  try {
    await authStore.login({
      username: username.value,
      password: password.value,
    });
    router.push('/dashboard');
  } catch (err: unknown) {
    const apiError = err as { message?: string };
    error.value = apiError.message || 'Login failed';
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <div class="w-full max-w-md p-8 bg-card rounded-xl shadow-lg">
    <div class="text-center mb-8">
      <h1 class="text-2xl font-bold">Welcome Back</h1>
      <p class="text-muted-foreground mt-2">Sign in to your account</p>
    </div>

    <form @submit.prevent="handleLogin" class="space-y-4">
      <div v-if="error" class="p-3 bg-destructive/10 text-destructive rounded-lg text-sm">
        {{ error }}
      </div>

      <div>
        <label for="username" class="block text-sm font-medium mb-1">
          Username
        </label>
        <input
          id="username"
          v-model="username"
          type="text"
          placeholder="Enter your username"
          class="w-full px-3 py-2 border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-ring"
          :disabled="isLoading"
        />
      </div>

      <div>
        <label for="password" class="block text-sm font-medium mb-1">
          Password
        </label>
        <input
          id="password"
          v-model="password"
          type="password"
          placeholder="Enter your password"
          class="w-full px-3 py-2 border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-ring"
          :disabled="isLoading"
        />
      </div>

      <button
        type="submit"
        class="w-full py-2 px-4 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
        :disabled="isLoading"
      >
        <span v-if="isLoading">Signing in...</span>
        <span v-else>Sign In</span>
      </button>
    </form>

    <div class="mt-6 text-center text-sm text-muted-foreground">
      Don't have an account?
      <a href="/register" class="text-primary hover:underline">
        Register
      </a>
    </div>
  </div>
</template>
