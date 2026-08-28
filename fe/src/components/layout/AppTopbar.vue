<script setup lang="ts">
import { ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { navigateTo } from '@/lib/navigation';

const authStore = useAuthStore();
const uiStore = useUiStore();

const showUserMenu = ref(false);

function handleLogout() {
  authStore.logout();
  navigateTo('/login');
}

function toggleDarkMode() {
  uiStore.toggleDarkMode();
}
</script>

<template>
  <header class="h-16 border-b bg-card flex items-center justify-between px-6">
    <!-- Left: Page Title -->
    <div>
      <h1 class="text-lg font-semibold">POS Application</h1>
    </div>

    <!-- Right: Actions -->
    <div class="flex items-center gap-4">
      <!-- Dark Mode Toggle -->
      <button
        @click="toggleDarkMode"
        class="p-2 rounded-lg hover:bg-muted transition-colors"
        title="Toggle dark mode"
      >
        <span v-if="uiStore.isDarkMode">☀️</span>
        <span v-else>🌙</span>
      </button>

      <!-- User Menu -->
      <div class="relative">
        <button
          @click="showUserMenu = !showUserMenu"
          class="flex items-center gap-2 p-2 rounded-lg hover:bg-muted transition-colors"
        >
          <div class="w-8 h-8 rounded-full bg-primary flex items-center justify-center text-primary-foreground text-sm font-medium">
            {{ authStore.userName.charAt(0).toUpperCase() }}
          </div>
          <span class="text-sm font-medium hidden sm:inline">{{ authStore.userName }}</span>
        </button>

        <!-- Dropdown Menu -->
        <div
          v-if="showUserMenu"
          class="absolute right-0 top-full mt-2 w-48 bg-card border rounded-lg shadow-lg z-50"
        >
          <div class="p-2">
            <p class="px-2 py-1 text-sm font-medium">{{ authStore.userName }}</p>
            <p class="px-2 py-1 text-xs text-muted-foreground">{{ authStore.currentUser?.email }}</p>
          </div>
          <div class="border-t" />
          <div class="p-2">
            <a
              href="/admin/profile"
              class="flex items-center gap-2 px-2 py-1.5 text-sm rounded-lg hover:bg-muted transition-colors"
            >
              <span>👤</span>
              <span>Profile</span>
            </a>
            <button
              @click="handleLogout"
              class="w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-lg hover:bg-muted transition-colors text-destructive"
            >
              <span>🚪</span>
              <span>Logout</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </header>

  <!-- Overlay to close menu -->
  <div
    v-if="showUserMenu"
    class="fixed inset-0 z-40"
    @click="showUserMenu = false"
  />
</template>
