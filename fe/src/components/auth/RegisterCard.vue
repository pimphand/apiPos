<script setup lang="ts">
import { ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { registerSchema } from '@/lib/validators';
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
const email = ref('');
const password = ref('');
const confirmPassword = ref('');
const fullName = ref('');
const storeName = ref('');
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
  const parsed = registerSchema.safeParse({
    full_name: fullName.value,
    username: username.value,
    email: email.value,
    password: password.value,
    confirm_password: confirmPassword.value,
    store_name: storeName.value,
  });

  if (!parsed.success) {
    const issue = parsed.error.issues[0];
    showError(issue?.message ?? 'Please check your details');
    return;
  }

  isLoading.value = true;
  error.value = '';

  try {
    const { full_name, username, email, password, store_name } = parsed.data;
    await authStore.register({
      full_name,
      username,
      email,
      password,
      store_name,
    });
    uiStore.addToast('success', t('auth.register.success'));
    navigateTo('/dashboard');
  } catch (err: unknown) {
    const apiError = err as { response?: { data?: { message?: string } }; message?: string };
    const message = apiError.response?.data?.message || apiError.message || t('auth.register.failed');
    showError(message);
    uiStore.addToast('error', message, 3000);
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <main class="grid w-full max-w-6xl overflow-hidden rounded-3xl border bg-card shadow-2xl shadow-black/10 md:min-h-[680px] md:grid-cols-[0.78fr_1.22fr]">
    <aside class="relative flex min-h-56 flex-col justify-between overflow-hidden bg-foreground p-5 text-background sm:p-10 lg:p-12">
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

        <p class="mt-10 text-xs font-semibold tracking-[0.22em] text-background/60">{{ t('auth.panel.eyebrow') }}</p>
        <h1 class="mt-4 max-w-sm text-3xl font-semibold leading-tight tracking-tight sm:text-4xl">{{ t('auth.panel.headline') }}</h1>
        <p class="mt-4 max-w-sm text-sm leading-6 text-background/70">{{ t('auth.panel.description') }}</p>
      </div>

      <div class="relative mt-6 rounded-2xl border border-background/15 bg-background/[0.06] p-4 sm:mt-8 sm:p-5">
        <p class="text-[10px] font-semibold tracking-[0.18em] text-background/55">{{ t('auth.panel.accessLabel') }}</p>
        <p class="mt-2 text-sm leading-6 text-background/85">{{ t('auth.panel.registerNote') }}</p>
        <div class="mt-5 hidden h-8 items-end gap-[3px] opacity-60 sm:flex" aria-hidden="true">
          <span v-for="(bar, index) in [18, 28, 13, 23, 32, 16, 26, 12, 30, 20, 34, 15, 24, 18, 29, 12, 22, 32, 16, 27, 14, 30, 20, 34, 13, 24, 18, 31, 15, 26]" :key="index" class="flex-1 bg-background" :style="{ height: `${bar}px` }" />
        </div>
      </div>
    </aside>

    <section class="flex items-center justify-center px-6 py-8 sm:px-10 lg:px-12">
      <div class="w-full max-w-2xl">
        <div class="mb-6">
          <p class="text-sm font-medium text-primary">{{ t('auth.panel.eyebrow') }}</p>
          <h2 class="mt-2 text-3xl font-semibold tracking-tight">{{ t('auth.register.title') }}</h2>
          <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('auth.register.subtitle') }}</p>
        </div>

        <form @submit.prevent="handleRegister" class="grid gap-x-5 gap-y-4 sm:grid-cols-2">
          <div v-if="error" role="alert" class="rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive sm:col-span-2">
            {{ error }}
          </div>

          <div class="space-y-2">
            <Label for="storeName" class="text-sm font-medium">{{ t('auth.register.storeName') }}</Label>
            <Input
              id="storeName"
              v-model="storeName"
              type="text"
              :placeholder="t('auth.register.storeNamePlaceholder')"
              autocomplete="organization"
              :disabled="isLoading"
              class="h-12 rounded-xl px-4 text-base"
            />
            <p class="text-xs leading-5 text-muted-foreground sm:min-h-10">{{ t('auth.register.storeHint') }}</p>
          </div>

          <div class="space-y-2">
            <Label for="fullName" class="text-sm font-medium">{{ t('auth.register.fullName') }}</Label>
            <Input
              id="fullName"
              v-model="fullName"
              type="text"
              :placeholder="t('auth.register.fullNamePlaceholder')"
              autocomplete="name"
              :disabled="isLoading"
              class="h-12 rounded-xl px-4 text-base"
            />
          </div>

          <div class="space-y-2">
            <Label for="username" class="text-sm font-medium">{{ t('auth.register.username') }}</Label>
            <Input
              id="username"
              v-model="username"
              type="text"
              :placeholder="t('auth.register.usernamePlaceholder')"
              autocomplete="username"
              :disabled="isLoading"
              class="h-12 rounded-xl px-4 text-base"
            />
          </div>

          <div class="space-y-2">
            <Label for="email" class="text-sm font-medium">{{ t('auth.register.email') }}</Label>
            <Input
              id="email"
              v-model="email"
              type="email"
              :placeholder="t('auth.register.emailPlaceholder')"
              autocomplete="email"
              :disabled="isLoading"
              class="h-12 rounded-xl px-4 text-base"
            />
          </div>

          <div class="space-y-2">
            <Label for="password" class="text-sm font-medium">{{ t('auth.register.password') }}</Label>
            <Input
              id="password"
              v-model="password"
              type="password"
              :placeholder="t('auth.register.passwordPlaceholder')"
              autocomplete="new-password"
              :disabled="isLoading"
              class="h-12 rounded-xl px-4 text-base"
            />
          </div>

          <div class="space-y-2">
            <Label for="confirmPassword" class="text-sm font-medium">{{ t('auth.register.confirmPassword') }}</Label>
            <Input
              id="confirmPassword"
              v-model="confirmPassword"
              type="password"
              :placeholder="t('auth.register.confirmPasswordPlaceholder')"
              autocomplete="new-password"
              :disabled="isLoading"
              class="h-12 rounded-xl px-4 text-base"
            />
          </div>

          <div class="pt-2 sm:col-span-2">
            <Button type="submit" class="h-12 w-full rounded-xl text-sm font-semibold" :disabled="isLoading">
              <span v-if="isLoading">{{ t('auth.register.creating') }}</span>
              <span v-else>{{ t('auth.register.create') }}</span>
            </Button>
          </div>
        </form>

        <p class="mt-6 border-t pt-5 text-center text-sm text-muted-foreground">
          {{ t('auth.register.hasAccount') }}
          <a href="/login" class="ml-1 font-semibold text-foreground underline-offset-4 hover:underline">{{ t('auth.register.signIn') }}</a>
        </p>
      </div>
    </section>
  </main>

  <Toaster rich-colors position="bottom-right" />
</template>
