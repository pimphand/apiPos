<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { useLocaleStore } from '@/stores/locale';
import { navigateTo } from '@/lib/navigation';
import { can } from '@/lib/rbac';
import { BarChart3, DollarSign, Hand, Package, ShoppingCart, TriangleAlert, Users } from '@lucide/vue';
import AppSidebar from './AppSidebar.vue';
import AppTopbar from './AppTopbar.vue';
import { SidebarProvider, SidebarInset } from '@/components/ui/sidebar';

const authStore = useAuthStore();
const uiStore = useUiStore();
const localeStore = useLocaleStore();
const t = localeStore.t;
const isReady = ref(false);

const userName = ref('User');

const stats = computed(() => [
  { label: t('dashboard.totalSales'), value: 'Rp 0', icon: DollarSign, href: '/admin/sales', permission: 'sales.read' as const },
  { label: t('dashboard.products'), value: '0', icon: Package, href: '/admin/products', permission: 'products.read' as const },
  { label: t('dashboard.customers'), value: '0', icon: Users, href: '/admin/customers', permission: 'customers.read' as const },
  { label: t('dashboard.lowStock'), value: '0', icon: TriangleAlert, href: '/admin/inventory', permission: 'inventory.read' as const },
].filter(s => can(authStore.currentUser, s.permission)));

onMounted(async () => {
  uiStore.loadTheme();

  // Restore a persisted session (reload case) before gating on auth.
  await authStore.restoreSession();

  if (!authStore.isAuthenticated) {
    navigateTo('/login');
    return;
  }

  if (authStore.currentUser) {
    userName.value = authStore.currentUser.full_name || authStore.currentUser.username || 'User';
  }

  isReady.value = true;
});
</script>

<template>
  <!-- While authenticating, render nothing but a spinner — never the
       dashboard shell (sidebar/stats) — otherwise the protected page flashes
       to unauthenticated visitors before the guard redirects to /login. -->
  <div v-if="!isReady" class="flex items-center justify-center min-h-dvh">
    <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary"></div>
  </div>
  <!-- SidebarProvider OUTERMOST so AppTopbar's SidebarTrigger has a provider. -->
  <SidebarProvider v-else class="block lg:flex lg:flex-row">
    <AppSidebar />
    <SidebarInset class="min-w-0 min-h-0 min-h-svh lg:min-h-0">
      <AppTopbar />
      <main class="min-w-0 flex-1 p-3 pb-20 sm:p-4 md:pb-8 lg:p-6 lg:h-svh lg:overflow-y-auto lg:overscroll-contain">
        <!-- Welcome Header -->
        <div class="mb-5 sm:mb-8">
          <h2 class="flex flex-wrap items-center gap-2 text-xl font-bold sm:text-2xl">{{ t('dashboard.welcome', { name: userName }) }} <Hand class="h-5 w-5 sm:h-6 sm:w-6 text-primary" aria-hidden="true" /></h2>
          <p class="text-muted-foreground mt-1 text-sm sm:text-base">{{ t('dashboard.subtitle') }}</p>
        </div>

        <!-- Stats Grid -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4 lg:gap-6 mb-6 sm:mb-8">
          <a
            v-for="stat in stats"
            :key="stat.label"
            :href="stat.href"
            class="p-4 sm:p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow min-w-0"
          >
            <div class="flex items-center justify-between gap-2 min-w-0">
              <div class="min-w-0">
                <p class="text-sm text-muted-foreground truncate">{{ stat.label }}</p>
                <p class="text-xl sm:text-2xl font-bold mt-1 truncate">{{ stat.value }}</p>
              </div>
              <component :is="stat.icon" class="h-7 w-7 sm:h-8 sm:w-8 shrink-0 text-muted-foreground" aria-hidden="true" />
            </div>
          </a>
        </div>

        <!-- Quick Actions -->
        <div class="mb-6 sm:mb-8">
          <h3 class="text-base sm:text-lg font-semibold mb-3 sm:mb-4">{{ t('dashboard.quickActions') }}</h3>
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
            <a
              v-if="can(authStore.currentUser, 'sales.create')"
              href="/pos"
              class="p-4 sm:p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow flex items-center gap-3 sm:gap-4 min-w-0"
            >
              <ShoppingCart class="h-6 w-6 sm:h-8 sm:w-8 shrink-0 text-muted-foreground" aria-hidden="true" />
              <div class="min-w-0">
                <h4 class="font-semibold truncate">{{ t('dashboard.posCheckout') }}</h4>
                <p class="text-sm text-muted-foreground">{{ t('dashboard.posCheckoutDesc') }}</p>
              </div>
            </a>

            <a
              v-if="can(authStore.currentUser, 'reports.read')"
              href="/reports"
              class="p-4 sm:p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow flex items-center gap-3 sm:gap-4 min-w-0"
            >
              <BarChart3 class="h-6 w-6 sm:h-8 sm:w-8 shrink-0 text-muted-foreground" aria-hidden="true" />
              <div class="min-w-0">
                <h4 class="font-semibold truncate">{{ t('dashboard.reports') }}</h4>
                <p class="text-sm text-muted-foreground">{{ t('dashboard.reportsDesc') }}</p>
              </div>
            </a>

            <a
              v-if="can(authStore.currentUser, 'products.read')"
              href="/admin/products"
              class="p-4 sm:p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow flex items-center gap-3 sm:gap-4 min-w-0"
            >
              <Package class="h-6 w-6 sm:h-8 sm:w-8 shrink-0 text-muted-foreground" aria-hidden="true" />
              <div class="min-w-0">
                <h4 class="font-semibold truncate">{{ t('dashboard.productsShort') }}</h4>
                <p class="text-sm text-muted-foreground">{{ t('dashboard.productsDesc') }}</p>
              </div>
            </a>
          </div>
        </div>

        <!-- Recent Activity Placeholder -->
        <div class="bg-card rounded-xl border shadow-sm p-4 sm:p-6">
          <h3 class="text-base sm:text-lg font-semibold mb-3 sm:mb-4">{{ t('dashboard.recentActivity') }}</h3>
          <div class="text-center py-6 sm:py-8 text-muted-foreground">
            <p>{{ t('dashboard.noActivity') }}</p>
            <p class="text-sm mt-2">{{ t('dashboard.activityHint') }}</p>
          </div>
        </div>
        </main>
    </SidebarInset>
  </SidebarProvider>

</template>
