<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { navigateTo } from '@/lib/navigation';
import AppSidebar from './AppSidebar.vue';
import AppTopbar from './AppTopbar.vue';
import { SidebarProvider, SidebarInset } from '@/components/ui/sidebar';

const authStore = useAuthStore();
const uiStore = useUiStore();
const isReady = ref(false);

onMounted(() => {
  uiStore.loadTheme();

  if (!authStore.isAuthenticated) {
    navigateTo('/login');
    return;
  }

  isReady.value = true;
});
</script>

<template>
  <SidebarProvider v-if="isReady">
    <AppSidebar />
    <SidebarInset>
      <AppTopbar />
      <main class="flex-1 overflow-y-auto p-6">
        <slot />
      </main>
    </SidebarInset>
  </SidebarProvider>

  <div v-else class="flex items-center justify-center h-screen">
    <div class="text-center">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
      <p class="mt-4 text-muted-foreground">Loading...</p>
    </div>
  </div>
</template>
