<script setup lang="ts">
import { ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { navigateTo } from '@/lib/navigation';
import { SidebarTrigger } from '@/components/ui/sidebar';
import { Separator } from '@/components/ui/separator';
import { LogOut, Moon, Sun, UserRound } from '@lucide/vue';

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
  <header class="flex min-h-16 items-center gap-1 border-b bg-card px-2 sm:gap-2 sm:px-4">
    <SidebarTrigger class="-ml-1" />
    <Separator orientation="vertical" class="mr-1 h-4 sm:mr-2" />

    <div class="flex min-w-0 flex-1 items-center justify-between gap-2">
      <h1 class="min-w-0 truncate text-base font-semibold sm:text-lg">{{ t('topbar.appTitle') }}</h1>

      <div class="flex shrink-0 items-center gap-1 sm:gap-2">
        <!-- Language Toggle -->
        <button
          @click="localeStore.toggleLocale"
          class="h-9 rounded-lg border px-2 text-xs font-medium hover:bg-muted transition-colors sm:px-3 sm:text-sm"
          :title="localeStore.locale === 'en' ? 'Bahasa Indonesia' : 'English'"
        >
          {{ localeStore.locale === 'en' ? 'ID' : 'EN' }}
        </button>

        <!-- Dark Mode Toggle -->
        <button
          @click="toggleDarkMode"
          class="grid h-10 w-10 place-items-center rounded-lg hover:bg-muted transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          :title="t('topbar.toggleDark')"
          :aria-label="t('topbar.toggleDark')"
        >
          <Sun v-if="uiStore.isDarkMode" class="h-5 w-5" aria-hidden="true" />
          <Moon v-else class="h-5 w-5" aria-hidden="true" />
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
            <span class="hidden max-w-36 truncate text-sm font-medium md:inline">{{ authStore.userName }}</span>
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
                <UserRound class="h-4 w-4" aria-hidden="true" />
                <span>{{ t('topbar.profile') }}</span>
              </a>
              <button
                @click="handleLogout"
                class="w-full flex items-center gap-2 px-2 py-1.5 text-sm rounded-lg hover:bg-muted transition-colors text-destructive"
              >
                <LogOut class="h-4 w-4" aria-hidden="true" />
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
