<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useLocaleStore } from '@/stores/locale';
import { navigateTo } from '@/lib/navigation';
import { LockKeyhole } from '@lucide/vue';

const authStore = useAuthStore();
const t = useLocaleStore().t;
const isReady = ref(false);

// Remember the last visited URL so "Back" can return there. Only pages behind
// the auth guard record it (see usePageGuard), so this is never the /403 page
// itself. If there is no recorded URL (e.g. direct visit), fall back to
// /dashboard.
const LAST_URL_KEY = 'pos_last_url';

function goBack() {
  let last = '';
  try {
    last = window.sessionStorage.getItem(LAST_URL_KEY) || '';
  } catch {
    /* sessionStorage unavailable */
  }
  navigateTo(last && last !== '/403' ? last : '/dashboard');
}

onMounted(async () => {
  // Only authenticated users may see this page. If the session is missing or
  // cannot be restored, send the user to /login instead — never show them the
  // permission-denied screen.
  await authStore.restoreSession();
  if (!authStore.isAuthenticated) {
    navigateTo('/login');
    return;
  }
  isReady.value = true;
});
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-muted/50 p-4">
    <div v-if="isReady" class="w-full max-w-md bg-card rounded-xl border shadow-sm p-8 text-center">
      <div class="mx-auto flex size-16 items-center justify-center rounded-full bg-destructive/10 mb-4">
        <LockKeyhole class="h-8 w-8 text-destructive" aria-hidden="true" />
      </div>
      <h1 class="text-2xl font-bold">{{ t('permissionDenied.title') }}</h1>
      <p class="text-muted-foreground mt-2">{{ t('permissionDenied.description') }}</p>
      <p class="text-sm text-muted-foreground mt-1">{{ t('permissionDenied.hint') }}</p>

      <div class="mt-6 flex items-center justify-center gap-3">
        <button
          type="button"
          @click="navigateTo('/dashboard')"
          class="inline-flex items-center gap-2 px-4 py-2 border rounded-lg text-sm font-medium hover:bg-muted transition-colors"
        >
          {{ t('common.backToDashboard') }}
        </button>
      </div>
    </div>

    <div v-else class="text-center">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
      <p class="mt-4 text-sm text-muted-foreground">{{ t('common.loading') }}</p>
    </div>
  </div>
</template>