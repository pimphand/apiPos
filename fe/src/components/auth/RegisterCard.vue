<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const authStore = useAuthStore();

const username = ref('');
const email = ref('');
const password = ref('');
const confirmPassword = ref('');
const fullName = ref('');
const isLoading = ref(false);
const error = ref('');

async function handleRegister() {
  if (!username.value || !email.value || !password.value || !fullName.value) {
    error.value = 'Please fill in all fields';
    return;
  }

  if (password.value !== confirmPassword.value) {
    error.value = 'Passwords do not match';
    return;
  }

  if (password.value.length < 6) {
    error.value = 'Password must be at least 6 characters';
    return;
  }

  isLoading.value = true;
  error.value = '';

  try {
    await authStore.register({
      username: username.value,
      email: email.value,
      password: password.value,
      full_name: fullName.value,
    });
    router.push('/dashboard');
  } catch (err: unknown) {
    const apiError = err as { message?: string };
    error.value = apiError.message || 'Registration failed';
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <div class="w-full max-w-md p-8 bg-card rounded-xl shadow-lg">
    <div class="text-center mb-8">
      <h1 class="text-2xl font-bold">Create Account</h1>
      <p class="text-muted-foreground mt-2">Sign up for a new account</p>
    </div>

    <form @submit.prevent="handleRegister" class="space-y-4">
      <div v-if="error" class="p-3 bg-destructive/10 text-destructive rounded-lg text-sm">
        {{ error }}
      </div>

      <div>
        <label for="fullName" class="block text-sm font-medium mb-1">
          Full Name
        </label>
        <input
          id="fullName"
          v-model="fullName"
          type="text"
          placeholder="Enter your full name"
          class="w-full px-3 py-2 border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-ring"
          :disabled="isLoading"
        />
      </div>

      <div>
        <label for="username" class="block text-sm font-medium mb-1">
          Username
        </label>
        <input
          id="username"
          v-model="username"
          type="text"
          placeholder="Choose a username"
          class="w-full px-3 py-2 border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-ring"
          :disabled="isLoading"
        />
      </div>

      <div>
        <label for="email" class="block text-sm font-medium mb-1">
          Email
        </label>
        <input
          id="email"
          v-model="email"
          type="email"
          placeholder="Enter your email"
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
          placeholder="Create a password"
          class="w-full px-3 py-2 border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-ring"
          :disabled="isLoading"
        />
      </div>

      <div>
        <label for="confirmPassword" class="block text-sm font-medium mb-1">
          Confirm Password
        </label>
        <input
          id="confirmPassword"
          v-model="confirmPassword"
          type="password"
          placeholder="Confirm your password"
          class="w-full px-3 py-2 border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-ring"
          :disabled="isLoading"
        />
      </div>

      <button
        type="submit"
        class="w-full py-2 px-4 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
        :disabled="isLoading"
      >
        <span v-if="isLoading">Creating account...</span>
        <span v-else>Create Account</span>
      </button>
    </form>

    <div class="mt-6 text-center text-sm text-muted-foreground">
      Already have an account?
      <a href="/login" class="text-primary hover:underline">
        Sign in
      </a>
    </div>
  </div>
</template>
