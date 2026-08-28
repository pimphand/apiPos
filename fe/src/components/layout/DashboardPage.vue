<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useUiStore } from '@/stores/ui';
import { formatCurrency } from '@/lib/utils';
import AppSidebar from './AppSidebar.vue';
import AppTopbar from './AppTopbar.vue';
import { navigateTo } from '@/lib/navigation';

const authStore = useAuthStore();
const uiStore = useUiStore();
const isReady = ref(false);

const userName = ref('User');

const stats = ref([
  { label: 'Total Sales', value: 'Rp 0', icon: '💰', href: '/admin/sales' },
  { label: 'Products', value: '0', icon: '📦', href: '/admin/products' },
  { label: 'Customers', value: '0', icon: '👥', href: '/admin/customers' },
  { label: 'Low Stock', value: '0', icon: '⚠️', href: '/admin/inventory' },
]);

onMounted(() => {
  uiStore.loadTheme();
  
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
  <div v-if="isReady" class="flex h-screen overflow-hidden">
    <!-- Sidebar -->
    <AppSidebar />
    
    <!-- Main Content -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- Topbar -->
      <AppTopbar />
      
      <!-- Page Content -->
      <main class="flex-1 overflow-y-auto p-6">
        <!-- Welcome Header -->
        <div class="mb-8">
          <h2 class="text-2xl font-bold">Welcome back, {{ userName }}! 👋</h2>
          <p class="text-muted-foreground mt-1">Here's what's happening with your store today.</p>
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
              <span class="text-3xl">{{ stat.icon }}</span>
            </div>
          </a>
        </div>

        <!-- Quick Actions -->
        <div class="mb-8">
          <h3 class="text-lg font-semibold mb-4">Quick Actions</h3>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
            <a
              href="/pos"
              class="p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow flex items-center gap-4"
            >
              <span class="text-3xl">🛒</span>
              <div>
                <h4 class="font-semibold">POS Checkout</h4>
                <p class="text-sm text-muted-foreground">Start a new sale transaction</p>
              </div>
            </a>

            <a
              href="/reports"
              class="p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow flex items-center gap-4"
            >
              <span class="text-3xl">📊</span>
              <div>
                <h4 class="font-semibold">Reports</h4>
                <p class="text-sm text-muted-foreground">View sales and inventory reports</p>
              </div>
            </a>

            <a
              href="/admin/products"
              class="p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow flex items-center gap-4"
            >
              <span class="text-3xl">📦</span>
              <div>
                <h4 class="font-semibold">Products</h4>
                <p class="text-sm text-muted-foreground">Manage products and inventory</p>
              </div>
            </a>
          </div>
        </div>

        <!-- Recent Activity Placeholder -->
        <div class="bg-card rounded-xl border shadow-sm p-6">
          <h3 class="text-lg font-semibold mb-4">Recent Activity</h3>
          <div class="text-center py-8 text-muted-foreground">
            <p>No recent activity to display.</p>
            <p class="text-sm mt-2">Activity will appear here as you use the system.</p>
          </div>
        </div>
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
