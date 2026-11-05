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
  <SidebarProvider>
    <AppSidebar />
    <SidebarInset>
      <AppTopbar />
      <main class="min-w-0 flex-1 overflow-y-auto p-3 sm:p-4 lg:p-6">
        <slot />
      </main>
    </SidebarInset>
  </SidebarProvider>


  <!-- Toaster lives in the same Vue island as the page components that push
       toasts, so the vue-sonner singleton is reliably shared. -->
  <Toaster rich-colors position="bottom-right" />
</template>
