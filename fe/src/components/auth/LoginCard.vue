<script setup lang="ts">
import { ref } from 'vue';
import { Eye, EyeOff } from '@lucide/vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { loginSchema } from '@/lib/validators';
import Button from '@/components/ui/button/Button.vue';
import Input from '@/components/ui/input/Input.vue';
import Label from '@/components/ui/label/Label.vue';
import { navigateTo } from '@/lib/navigation';
import { Toaster } from '@/components/ui/sonner';

const authStore = useAuthStore();
const uiStore = useUiStore();
const localeStore = useLocaleStore();
const t = localeStore.t;

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
    uiStore.addToast('success', t('auth.login.loginSuccessful'));
    navigateTo('/dashboard');
  } catch (err: unknown) {
    const apiError = err as { response?: { data?: { message?: string } }; message?: string };
    const message = apiError.response?.data?.message || apiError.message || t('auth.login.loginFailed');
    showError(message);
    uiStore.addToast('error', message, 3000);
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <main class="grid w-full max-w-5xl overflow-hidden rounded-3xl border bg-card shadow-2xl shadow-black/10 md:min-h-[620px] md:grid-cols-[0.95fr_1.05fr]">
    <aside class="relative flex min-h-64 flex-col justify-between overflow-hidden bg-foreground p-7 text-background sm:p-10 lg:p-12">
      <div class="pointer-events-none absolute -right-24 -top-24 h-72 w-72 rounded-full border border-background/10" />
      <div class="pointer-events-none absolute -right-12 -top-12 h-48 w-48 rounded-full border border-background/10" />

      <div class="relative">
        <a href="/" class="inline-flex items-center gap-3 rounded-md focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-background">
          <span class="grid h-11 w-11 place-items-center rounded-xl bg-background text-foreground">
            <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" class="h-6 w-6" stroke="currentColor" stroke-width="1.8">
              <path d="M4 8.5h16v11H4zM3 5h18v3.5H3zM8 12h3v3H8zM14 12h3" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </span>
          <span class="text-lg font-semibold tracking-tight">POS</span>
        </a>

        <p class="mt-12 text-xs font-semibold tracking-[0.22em] text-background/60">{{ t('auth.panel.eyebrow') }}</p>
        <h1 class="mt-4 max-w-sm text-3xl font-semibold leading-tight tracking-tight sm:text-4xl">{{ t('auth.panel.headline') }}</h1>
        <p class="mt-4 max-w-sm text-sm leading-6 text-background/70">{{ t('auth.panel.description') }}</p>
      </div>

      <div class="relative mt-10 rounded-2xl border border-background/15 bg-background/[0.06] p-5">
        <div class="flex items-center justify-between border-b border-background/15 pb-4">
          <div>
            <p class="text-[10px] font-semibold tracking-[0.18em] text-background/55">{{ t('auth.panel.accessLabel') }}</p>
            <p class="mt-1 text-sm font-medium">{{ t('auth.panel.loginNote') }}</p>
          </div>
          <span class="grid h-9 w-9 place-items-center rounded-full border border-background/20 text-background/80" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" class="h-4 w-4" stroke="currentColor" stroke-width="1.8"><path d="M5 12h14M13 6l6 6-6 6" stroke-linecap="round" stroke-linejoin="round" /></svg>
          </span>
        </div>
        <div class="mt-4 grid grid-cols-3 divide-x divide-background/15 text-center">
          <p class="px-1 text-xs text-background/75">{{ t('auth.panel.salesLabel') }}</p>
          <p class="px-1 text-xs text-background/75">{{ t('auth.panel.inventoryLabel') }}</p>
          <p class="px-1 text-xs text-background/75">{{ t('auth.panel.reportsLabel') }}</p>
        </div>
        <div class="mt-5 flex h-8 items-end gap-[3px] opacity-60" aria-hidden="true">
          <span v-for="(bar, index) in [18, 28, 13, 23, 32, 16, 26, 12, 30, 20, 34, 15, 24, 18, 29, 12, 22, 32, 16, 27, 14, 30, 20, 34, 13, 24, 18, 31, 15, 26]" :key="index" class="flex-1 bg-background" :style="{ height: `${bar}px` }" />
        </div>
      </div>
    </aside>

    <section class="flex items-center justify-center px-6 py-9 sm:px-10 lg:px-14">
      <div class="w-full max-w-sm">
        <div class="mb-8">
          <p class="text-sm font-medium text-primary">{{ t('auth.panel.eyebrow') }}</p>
          <h2 class="mt-2 text-3xl font-semibold tracking-tight">{{ t('auth.login.welcome') }}</h2>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('auth.login.subtitle') }}</p>
        </div>

        <form @submit.prevent="handleLogin" class="space-y-5">
          <div v-if="error" role="alert" class="rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
            {{ error }}
          </div>

          <div class="space-y-2">
            <Label for="username" class="text-sm font-medium">{{ t('auth.login.username') }}</Label>
            <Input
              id="username"
              v-model="username"
              type="text"
              :placeholder="t('auth.login.usernamePlaceholder')"
              autocomplete="username"
              :disabled="isLoading"
              class="h-12 rounded-xl px-4 text-base"
            />
          </div>

          <div class="space-y-2">
            <Label for="password" class="text-sm font-medium">{{ t('auth.login.password') }}</Label>
            <div class="relative">
              <Input
                id="password"
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                :placeholder="t('auth.login.passwordPlaceholder')"
                autocomplete="current-password"
                :disabled="isLoading"
                class="h-12 rounded-xl px-4 pr-12 text-base"
              />
              <button
                type="button"
                :aria-label="t('auth.login.togglePassword')"
                class="absolute inset-y-0 right-0 grid w-12 place-items-center rounded-r-xl text-muted-foreground transition-colors hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                @click="showPassword = !showPassword"
              >
                <Eye v-if="!showPassword" class="h-4 w-4" aria-hidden="true" />
                <EyeOff v-else class="h-4 w-4" aria-hidden="true" />
              </button>
            </div>
          </div>

          <Button type="submit" class="h-12 w-full rounded-xl text-sm font-semibold" :disabled="isLoading">
            <span v-if="isLoading">{{ t('auth.login.signingIn') }}</span>
            <span v-else>{{ t('auth.login.signIn') }}</span>
          </Button>
        </form>

        <p class="mt-8 border-t pt-6 text-center text-sm text-muted-foreground">
          {{ t('auth.login.noAccount') }}
          <a href="/register" class="ml-1 font-semibold text-foreground underline-offset-4 hover:underline">{{ t('auth.login.register') }}</a>
        </p>
      </div>
    </section>
  </main>

  <Toaster rich-colors position="bottom-right" />
</template>
