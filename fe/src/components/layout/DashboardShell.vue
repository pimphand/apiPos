<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { navigateTo } from '@/lib/navigation';
import AppSidebar from './AppSidebar.vue';
import AppTopbar from './AppTopbar.vue';
import { SidebarProvider, SidebarInset } from '@/components/ui/sidebar';
import { Toaster } from '@/components/ui/sonner';

const authStore = useAuthStore();
const uiStore = useUiStore();
const isReady = ref(false);

onMounted(async () => {
  uiStore.loadTheme();

  // Restore a persisted session (reload case) before gating on auth.
  await authStore.restoreSession();

  if (!authStore.isAuthenticated) {
    navigateTo('/login');
    return;
  }

  isReady.value = true;
});
</script>

<template>
  <!-- SidebarProvider OUTERMOST so AppTopbar's SidebarTrigger (useSidebar
       injection) has a provider. -->
  <SidebarProvider v-if="isReady" class="block lg:flex lg:flex-row">
    <AppSidebar />
    <SidebarInset class="min-w-0 min-h-svh lg:h-svh lg:min-h-0 lg:overflow-hidden">
      <AppTopbar />
      <div class="min-w-0 flex-1 min-h-0 overflow-y-auto p-3 pb-20 sm:p-4 md:pb-8 lg:p-6 lg:overscroll-contain">
        <slot />
      </div>
    </SidebarInset>
  </SidebarProvider>

  <div v-else class="flex items-center justify-center min-h-dvh">
    <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary"></div>
  </div>


  <!-- Toaster lives in the same Vue island as the page components that push
       toasts, so the vue-sonner singleton is reliably shared. -->
  <Toaster rich-colors position="bottom-right" />
</template>
