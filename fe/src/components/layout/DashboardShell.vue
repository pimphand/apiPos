<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { navigateTo } from '@/lib/navigation';
import AppSidebar from './AppSidebar.vue';
import AppTopbar from './AppTopbar.vue';

const authStore = useAuthStore();
const uiStore = useUiStore();
const isReady = ref(false);

onMounted(() => {
  // Load theme on mount
  uiStore.loadTheme();
  
  // Redirect to login if not authenticated
  if (!authStore.isAuthenticated) {
    navigateTo('/login');
    return;
  }
  
  isReady.value = true;
});
</script>

<template>
  <div v-if="isReady" class="flex h-screen overflow-hidden">
    <!-- Sidebar -->
    <AppSidebar />
    
    <!-- Main Content -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- Topbar -->
      <AppTopbar />
      
      <!-- Page Content -->
      <main class="flex-1 overflow-y-auto p-6">
        <slot />
      </main>
    </div>
  </div>
  
  <!-- Loading state -->
  <div v-else class="flex items-center justify-center h-screen">
    <div class="text-center">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
      <p class="mt-4 text-muted-foreground">Loading...</p>
    </div>
  </div>
</template>
