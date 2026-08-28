<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { formatCurrency } from '@/lib/utils';

const authStore = useAuthStore();

const userName = ref('User');

const stats = ref([
  { label: 'Total Sales', value: 'Rp 0', icon: '💰', href: '/admin/sales' },
  { label: 'Products', value: '0', icon: '📦', href: '/admin/products' },
  { label: 'Customers', value: '0', icon: '👥', href: '/admin/customers' },
  { label: 'Low Stock', value: '0', icon: '⚠️', href: '/admin/inventory' },
]);

onMounted(() => {
  if (authStore.currentUser) {
    userName.value = authStore.currentUser.full_name || authStore.currentUser.username || 'User';
  }
});
</script>

<template>
  <div>
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
  </div>
</template>
