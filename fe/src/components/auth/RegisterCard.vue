<script setup lang="ts">
import { ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { registerSchema } from '@/lib/validators';
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
  <Card class="w-full max-w-md">
    <CardHeader class="text-center">
      <CardTitle class="text-2xl">{{ t('auth.register.title') }}</CardTitle>
      <CardDescription>{{ t('auth.register.subtitle') }}</CardDescription>
    </CardHeader>
    <CardContent>
      <form @submit.prevent="handleRegister" class="space-y-4">
        <div v-if="error" class="p-3 bg-destructive/10 text-destructive rounded-lg text-sm">
          {{ error }}
        </div>

        <div class="space-y-2">
          <Label for="fullName">{{ t('auth.register.fullName') }}</Label>
          <Input
            id="fullName"
            v-model="fullName"
            type="text"
            :placeholder="t('auth.register.fullNamePlaceholder')"
            autocomplete="name"
            :disabled="isLoading"
          />
        </div>

        <div class="space-y-2">
          <Label for="storeName">{{ t('auth.register.storeName') }}</Label>
          <Input
            id="storeName"
            v-model="storeName"
            type="text"
            :placeholder="t('auth.register.storeNamePlaceholder')"
            autocomplete="organization"
            :disabled="isLoading"
          />
          <p class="text-xs text-muted-foreground">{{ t('auth.register.storeHint') }}</p>
        </div>

        <div class="space-y-2">
          <Label for="username">{{ t('auth.register.username') }}</Label>
          <Input
            id="username"
            v-model="username"
            type="text"
            :placeholder="t('auth.register.usernamePlaceholder')"
            autocomplete="username"
            :disabled="isLoading"
          />
        </div>

        <div class="space-y-2">
          <Label for="email">{{ t('auth.register.email') }}</Label>
          <Input
            id="email"
            v-model="email"
            type="email"
            :placeholder="t('auth.register.emailPlaceholder')"
            autocomplete="email"
            :disabled="isLoading"
          />
        </div>

        <div class="space-y-2">
          <Label for="password">{{ t('auth.register.password') }}</Label>
          <Input
            id="password"
            v-model="password"
            type="password"
            :placeholder="t('auth.register.passwordPlaceholder')"
            autocomplete="new-password"
            :disabled="isLoading"
          />
        </div>

        <div class="space-y-2">
          <Label for="confirmPassword">{{ t('auth.register.confirmPassword') }}</Label>
          <Input
            id="confirmPassword"
            v-model="confirmPassword"
            type="password"
            :placeholder="t('auth.register.confirmPasswordPlaceholder')"
            autocomplete="new-password"
            :disabled="isLoading"
          />
        </div>

        <Button type="submit" class="w-full" :disabled="isLoading">
          <span v-if="isLoading">{{ t('auth.register.creating') }}</span>
          <span v-else>{{ t('auth.register.create') }}</span>
        </Button>
      </form>
    </CardContent>
    <CardFooter class="justify-center">
      <p class="text-sm text-muted-foreground">
        {{ t('auth.register.hasAccount') }}
        <a href="/login" class="text-primary hover:underline">{{ t('auth.register.signIn') }}</a>
      </p>
    </CardFooter>
  </Card>

  <Toaster rich-colors position="bottom-right" />
</template>
