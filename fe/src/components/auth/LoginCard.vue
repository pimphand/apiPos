<script setup lang="ts">
import { ref } from 'vue';
import { Eye, EyeOff } from '@lucide/vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { loginSchema } from '@/lib/validators';
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
const password = ref('');
const showPassword = ref(false);
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

async function handleLogin() {
  const parsed = loginSchema.safeParse({
    username: username.value,
    password: password.value,
  });

  if (!parsed.success) {
    const issue = parsed.error.issues[0];
    showError(issue?.message ?? 'Please enter valid credentials');
    return;
  }

  isLoading.value = true;
  error.value = '';

  try {
    await authStore.login(parsed.data);
    uiStore.addToast('success', 'Login successful!');
    navigateTo('/dashboard');
  } catch (err: unknown) {
    const apiError = err as { response?: { data?: { message?: string } }; message?: string };
    const message = apiError.response?.data?.message || apiError.message || 'Login failed';
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
      <CardTitle class="text-2xl">Welcome Back</CardTitle>
      <CardDescription>Sign in to your account</CardDescription>
    </CardHeader>
    <CardContent>
      <form @submit.prevent="handleLogin" class="space-y-4">
        <div v-if="error" class="p-3 bg-destructive/10 text-destructive rounded-lg text-sm">
          {{ error }}
        </div>

        <div class="space-y-2">
          <Label for="username">Username</Label>
          <Input
            id="username"
            v-model="username"
            type="text"
            placeholder="Enter your username"
            autocomplete="username"
            :disabled="isLoading"
          />
        </div>

        <div class="space-y-2">
          <Label for="password">Password</Label>
          <div class="relative">
            <Input
              id="password"
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              placeholder="Enter your password"
              autocomplete="current-password"
              :disabled="isLoading"
              class="pr-10"
            />
            <button
              type="button"
              aria-label="Toggle password visibility"
              class="absolute inset-y-0 right-0 flex items-center pr-3 text-muted-foreground hover:text-foreground"
              @click="showPassword = !showPassword"
            >
              <Eye v-if="!showPassword" class="h-4 w-4" />
              <EyeOff v-else class="h-4 w-4" />
            </button>
          </div>
        </div>

        <Button type="submit" class="w-full" :disabled="isLoading">
          <span v-if="isLoading">Signing in...</span>
          <span v-else>Sign In</span>
        </Button>
      </form>
    </CardContent>
    <CardFooter class="justify-center">
      <p class="text-sm text-muted-foreground">
        Don't have an account?
        <a href="/register" class="text-primary hover:underline">Register</a>
      </p>
    </CardFooter>
  </Card>
</template>
