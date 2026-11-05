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
  <SidebarProvider>
    <AppSidebar />
    <SidebarInset>
      <AppTopbar />
      <main class="min-w-0 flex-1 overflow-y-auto p-3 sm:p-4 lg:p-6">
        <!-- Welcome Header -->
        <div class="mb-8">
          <h2 class="flex items-center gap-2 text-2xl font-bold">{{ t('dashboard.welcome', { name: userName }) }} <Hand class="h-6 w-6 text-primary" aria-hidden="true" /></h2>
          <p class="text-muted-foreground mt-1">{{ t('dashboard.subtitle') }}</p>
        </div>

        <!-- Stats Grid -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <a
            v-for="stat in stats"
            :key="stat.label"
            :href="stat.href"
            class="p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow"
          >
            <div class="flex items-center justify-between">
              <div>
                <p class="text-sm text-muted-foreground">{{ stat.label }}</p>
                <p class="text-2xl font-bold mt-1">{{ stat.value }}</p>
              </div>
              <component :is="stat.icon" class="h-8 w-8 text-muted-foreground" aria-hidden="true" />
            </div>
          </a>
        </div>

        <!-- Quick Actions -->
        <div class="mb-8">
          <h3 class="text-lg font-semibold mb-4">{{ t('dashboard.quickActions') }}</h3>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
            <a
              v-if="can(authStore.currentUser, 'sales.create')"
              href="/pos"
              class="p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow flex items-center gap-4"
            >
              <ShoppingCart class="h-8 w-8 text-muted-foreground" aria-hidden="true" />
              <div>
                <h4 class="font-semibold">{{ t('dashboard.posCheckout') }}</h4>
                <p class="text-sm text-muted-foreground">{{ t('dashboard.posCheckoutDesc') }}</p>
              </div>
            </a>

            <a
              v-if="can(authStore.currentUser, 'reports.read')"
              href="/reports"
              class="p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow flex items-center gap-4"
            >
              <BarChart3 class="h-8 w-8 text-muted-foreground" aria-hidden="true" />
              <div>
                <h4 class="font-semibold">{{ t('dashboard.reports') }}</h4>
                <p class="text-sm text-muted-foreground">{{ t('dashboard.reportsDesc') }}</p>
              </div>
            </a>

            <a
              v-if="can(authStore.currentUser, 'products.read')"
              href="/admin/products"
              class="p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow flex items-center gap-4"
            >
              <Package class="h-8 w-8 text-muted-foreground" aria-hidden="true" />
              <div>
                <h4 class="font-semibold">{{ t('dashboard.productsShort') }}</h4>
                <p class="text-sm text-muted-foreground">{{ t('dashboard.productsDesc') }}</p>
              </div>
            </a>
          </div>
        </div>

        <!-- Recent Activity Placeholder -->
        <div class="bg-card rounded-xl border shadow-sm p-6">
          <h3 class="text-lg font-semibold mb-4">{{ t('dashboard.recentActivity') }}</h3>
          <div class="text-center py-8 text-muted-foreground">
            <p>{{ t('dashboard.noActivity') }}</p>
            <p class="text-sm mt-2">{{ t('dashboard.activityHint') }}</p>
          </div>
        </div>
      </main>
    </SidebarInset>
  </SidebarProvider>

</template>
