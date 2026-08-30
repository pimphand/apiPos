<script setup lang="ts">
import { ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { navigateTo } from '@/lib/navigation';
import { SidebarTrigger } from '@/components/ui/sidebar';
import { Separator } from '@/components/ui/separator';

const authStore = useAuthStore();
const uiStore = useUiStore();
const localeStore = useLocaleStore();
const t = localeStore.t;

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
  <header class="h-16 border-b bg-card flex items-center gap-2 px-4">
    <SidebarTrigger class="-ml-1" />
    <Separator orientation="vertical" class="mr-2 h-4" />

    <div class="flex-1 flex items-center justify-between">
      <h1 class="text-lg font-semibold">{{ t('topbar.appTitle') }}</h1>

      <div class="flex items-center gap-2">
        <!-- Language Toggle -->
        <button
          @click="localeStore.toggleLocale"
          class="px-3 h-9 rounded-lg border text-sm font-medium hover:bg-muted transition-colors"
          :title="localeStore.locale === 'en' ? 'Bahasa Indonesia' : 'English'"
        >
          {{ localeStore.locale === 'en' ? 'ID' : 'EN' }}
        </button>

        <!-- Dark Mode Toggle -->
        <button
          @click="toggleDarkMode"
          class="p-2 rounded-lg hover:bg-muted transition-colors"
          :title="t('topbar.toggleDark')"
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
                <span>{{ t('topbar.profile') }}</span>
              </a>
              <button
                @click="handleLogout"
                class="w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-lg hover:bg-muted transition-colors text-destructive"
              >
                <span>🚪</span>
                <span>{{ t('topbar.logout') }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </header>

  <div
    v-if="showUserMenu"
    class="fixed inset-0 z-40"
    @click="showUserMenu = false"
  />
</template>
