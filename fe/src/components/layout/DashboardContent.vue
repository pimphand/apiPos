<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import { formatCurrency } from '@/lib/utils';

const router = useRouter();
const authStore = useAuthStore();

const stats = ref([
  { label: 'Total Sales', value: 'Rp 0', icon: '💰' },
  { label: 'Products', value: '0', icon: '📦' },
  { label: 'Customers', value: '0', icon: '👥' },
  { label: 'Low Stock', value: '0', icon: '⚠️' },
]);

onMounted(() => {
  if (!authStore.isAuthenticated) {
    router.push('/login');
  }
});

function handleLogout() {
  authStore.logout();
  router.push('/login');
}
</script>

<template>
  <div class="min-h-screen bg-background">
    <!-- Header -->
    <header class="border-b bg-card">
      <div class="container mx-auto px-4 py-4 flex items-center justify-between">
        <h1 class="text-xl font-bold">POS Dashboard</h1>
        <div class="flex items-center gap-4">
          <span class="text-sm text-muted-foreground">
            {{ authStore.currentUser?.full_name || 'User' }}
          </span>
          <button
            @click="handleLogout"
            class="px-4 py-2 text-sm bg-destructive text-destructive-foreground rounded-lg hover:bg-destructive/90"
          >
            Logout
          </button>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="container mx-auto px-4 py-8">
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div
          v-for="stat in stats"
          :key="stat.label"
          class="p-6 bg-card rounded-xl border shadow-sm"
        >
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm text-muted-foreground">{{ stat.label }}</p>
              <p class="text-2xl font-bold mt-1">{{ stat.value }}</p>
            </div>
            <span class="text-3xl">{{ stat.icon }}</span>
          </div>
        </div>
      </div>

      <!-- Quick Actions -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <a
          href="/pos"
          class="p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow"
        >
          <h3 class="font-semibold text-lg mb-2">🛒 POS Checkout</h3>
          <p class="text-sm text-muted-foreground">
            Start a new sale transaction
          </p>
        </a>

        <a
          href="/reports"
          class="p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow"
        >
          <h3 class="font-semibold text-lg mb-2">📊 Reports</h3>
          <p class="text-sm text-muted-foreground">
            View sales and inventory reports
          </p>
        </a>

        <a
          href="/admin/products"
          class="p-6 bg-card rounded-xl border shadow-sm hover:shadow-md transition-shadow"
        >
          <h3 class="font-semibold text-lg mb-2">📦 Products</h3>
          <p class="text-sm text-muted-foreground">
            Manage products and inventory
          </p>
        </a>
      </div>
    </main>
  </div>
</template>
