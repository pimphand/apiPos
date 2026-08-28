<script setup lang="ts">
import { ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import Card from '@/components/ui/card/Card.vue';
import CardHeader from '@/components/ui/card/CardHeader.vue';
import CardTitle from '@/components/ui/card/CardTitle.vue';
import CardDescription from '@/components/ui/card/CardDescription.vue';
import CardContent from '@/components/ui/card/CardContent.vue';
import CardFooter from '@/components/ui/card/CardFooter.vue';
import { navigateTo } from '@/lib/navigation';

const authStore = useAuthStore();
const uiStore = useUiStore();

const username = ref('');
const email = ref('');
const password = ref('');
const confirmPassword = ref('');
const fullName = ref('');
const isLoading = ref(false);
const error = ref('');
let errorTimer: ReturnType<typeof setTimeout> | null = null;

function showError(message: string) {
  error.value = '';
  if (errorTimer) clearTimeout(errorTimer);
  requestAnimationFrame(() => {
    error.value = message;
    errorTimer = setTimeout(() => {
      error.value = '';
      errorTimer = null;
    }, 3000);
  });
}

async function handleRegister() {
  if (!username.value || !email.value || !password.value || !fullName.value) {
    showError('Please fill in all fields');
    return;
  }

  if (password.value !== confirmPassword.value) {
    showError('Passwords do not match');
    return;
  }

  if (password.value.length < 6) {
    showError('Password must be at least 6 characters');
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
    uiStore.addToast('success', 'Registration successful!');
    navigateTo('/dashboard');
  } catch (err: unknown) {
    const apiError = err as { response?: { data?: { message?: string } }; message?: string };
    const message = apiError.response?.data?.message || apiError.message || 'Registration failed';
    showError(message);
    uiStore.addToast('error', message, 3000);
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <Card class="w-full max-w-md">
    <CardHeader class="text-center">
      <CardTitle class="text-2xl">Create Account</CardTitle>
      <CardDescription>Sign up for a new account</CardDescription>
    </CardHeader>
    <CardContent>
      <form @submit.prevent="handleRegister" class="space-y-4">
        <div v-if="error" class="p-3 bg-destructive/10 text-destructive rounded-lg text-sm">
          {{ error }}
        </div>

        <div class="space-y-2">
          <Label for="fullName">Full Name</Label>
          <Input
            id="fullName"
            v-model="fullName"
            type="text"
            placeholder="Enter your full name"
            autocomplete="name"
            :disabled="isLoading"
          />
        </div>

        <div class="space-y-2">
          <Label for="username">Username</Label>
          <Input
            id="username"
            v-model="username"
            type="text"
            placeholder="Choose a username"
            autocomplete="username"
            :disabled="isLoading"
          />
        </div>

        <div class="space-y-2">
          <Label for="email">Email</Label>
          <Input
            id="email"
            v-model="email"
            type="email"
            placeholder="Enter your email"
            autocomplete="email"
            :disabled="isLoading"
          />
        </div>

        <div class="space-y-2">
          <Label for="password">Password</Label>
          <Input
            id="password"
            v-model="password"
            type="password"
            placeholder="Create a password"
            autocomplete="new-password"
            :disabled="isLoading"
          />
        </div>

        <div class="space-y-2">
          <Label for="confirmPassword">Confirm Password</Label>
          <Input
            id="confirmPassword"
            v-model="confirmPassword"
            type="password"
            placeholder="Confirm your password"
            autocomplete="new-password"
            :disabled="isLoading"
          />
        </div>

        <Button type="submit" class="w-full" :disabled="isLoading">
          <span v-if="isLoading">Creating account...</span>
          <span v-else>Create Account</span>
        </Button>
      </form>
    </CardContent>
    <CardFooter class="justify-center">
      <p class="text-sm text-muted-foreground">
        Already have an account?
        <a href="/login" class="text-primary hover:underline">Sign in</a>
      </p>
    </CardFooter>
  </Card>
</template>
