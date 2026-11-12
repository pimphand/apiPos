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
const isLoggingOut = ref(false);

async function handleLogout() {
  // Wait for the local session (and the desktop keystore entry) to be deleted
  // before navigating: a navigation that tears the page down mid-request left
  // the refresh token behind and the login page restored the session.
  if (isLoggingOut.value) return;
  isLoggingOut.value = true;
  showUserMenu.value = false;
  try {
    await authStore.logout();
  } finally {
    navigateTo('/login');
  }
}

function toggleDarkMode() {
  uiStore.toggleDarkMode();
}
</script>

<template>
  <!-- Click-away backdrop. Rendered BEFORE the header (z-30) so the header and
       its z-50 dropdown stay above it: as a later sibling at z-40 it painted
       over the menu, so clicking "Logout" only closed the menu. -->
  <div
    v-if="showUserMenu"
    class="fixed inset-0 z-30"
    @click="showUserMenu = false"
  />

  <!-- Sticky topbar: mobile shows burger, tablet still fits, desktop keeps title -->
  <header class="relative z-40 flex min-h-14 items-center gap-1 border-b bg-card px-2 sm:gap-2 sm:px-4 lg:sticky lg:top-0 lg:z-30 lg:bg-card/95 lg:backdrop-blur supports-[backdrop-filter]:lg:bg-card/80 lg:min-h-16">
    <SidebarTrigger class="-ml-1 shrink-0" />
    <Separator orientation="vertical" class="mr-1 h-4 sm:mr-2" />

    <div class="flex min-w-0 flex-1 items-center justify-between gap-2">
      <h1 class="min-w-0 truncate text-sm font-semibold sm:text-base md:text-lg">{{ t('topbar.appTitle') }}</h1>

      <div class="flex shrink-0 items-center gap-1 sm:gap-2">
        <!-- Language Toggle -->
        <button
          @click="localeStore.toggleLocale"
          class="h-9 min-w-9 rounded-lg border px-2 text-xs font-medium hover:bg-muted transition-colors sm:px-3 sm:text-sm"
          :title="localeStore.locale === 'en' ? 'Bahasa Indonesia' : 'English'"
        >
          {{ localeStore.locale === 'en' ? 'ID' : 'EN' }}
        </button>

        <!-- Dark Mode Toggle -->
        <button
          @click="toggleDarkMode"
          class="grid h-9 w-9 place-items-center rounded-lg hover:bg-muted transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring sm:h-10 sm:w-10"
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
            class="flex items-center gap-2 p-1.5 rounded-lg hover:bg-muted transition-colors sm:p-2"
          >
            <div class="w-8 h-8 shrink-0 rounded-full bg-primary flex items-center justify-center text-primary-foreground text-sm font-medium">
              {{ authStore.userName.charAt(0).toUpperCase() }}
            </div>
            <span class="hidden max-w-36 truncate text-sm font-medium lg:inline">{{ authStore.userName }}</span>
          </button>

          <div
            v-if="showUserMenu"
            class="absolute right-0 top-full mt-2 w-52 max-w-[calc(100vw-2rem)] bg-card border rounded-lg shadow-lg z-50"
          >
            <div class="p-2">
              <p class="px-2 py-1 text-sm font-medium truncate">{{ authStore.userName }}</p>
              <p class="px-2 py-1 text-xs text-muted-foreground truncate">{{ authStore.currentUser?.email }}</p>
            </div>
            <div class="border-t" />
            <div class="p-2">
              <a
                href="/admin/profile"
                class="flex items-center gap-2 px-2 py-2 text-sm rounded-lg hover:bg-muted transition-colors"
              >
                <UserRound class="h-4 w-4" aria-hidden="true" />
                <span>{{ t('topbar.profile') }}</span>
              </a>
              <button
                @click="handleLogout"
                :disabled="isLoggingOut"
                class="w-full flex items-center gap-2 px-2 py-2 text-sm rounded-lg hover:bg-muted transition-colors text-destructive disabled:opacity-60 disabled:pointer-events-none"
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
</template>
