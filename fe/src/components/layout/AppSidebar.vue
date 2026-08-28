<script setup lang="ts">
import { ref, computed } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { can, type Permission } from '@/lib/rbac';

const authStore = useAuthStore();
const isCollapsed = ref(false);

interface MenuItem {
  label: string;
  icon: string;
  href: string;
  permission?: Permission;
  divider?: boolean;
}

const menuItems = computed<MenuItem[]>(() => [
  { label: 'Dashboard', icon: '📊', href: '/dashboard' },
  { label: 'POS Checkout', icon: '🛒', href: '/pos', permission: 'sales.create' },
  { divider: true, label: '', icon: '', href: '' },
  { label: 'Products', icon: '📦', href: '/admin/products', permission: 'products.read' },
  { label: 'Categories', icon: '🏷️', href: '/admin/categories', permission: 'categories.read' },
  { label: 'Inventory', icon: '📋', href: '/admin/inventory', permission: 'inventory.read' },
  { divider: true, label: '', icon: '', href: '' },
  { label: 'Customers', icon: '👥', href: '/admin/customers', permission: 'customers.read' },
  { label: 'Sales', icon: '💰', href: '/admin/sales', permission: 'sales.read' },
  { label: 'Payments', icon: '💳', href: '/admin/payments', permission: 'payments.read' },
  { divider: true, label: '', icon: '', href: '' },
  { label: 'Cashier Shifts', icon: '🕐', href: '/admin/cashier', permission: 'cashier.view_shift' },
  { label: 'Reports', icon: '📈', href: '/reports', permission: 'reports.read' },
  { divider: true, label: '', icon: '', href: '' },
  { label: 'Users', icon: '👤', href: '/admin/users', permission: 'users.read' },
  { label: 'Roles', icon: '🔐', href: '/admin/roles', permission: 'roles.read' },
  { label: 'Activity Logs', icon: '📝', href: '/admin/activity-logs', permission: 'activity_logs.read' },
]);

const filteredMenuItems = computed(() => {
  return menuItems.value.filter(item => {
    if (item.divider) return true;
    if (!item.permission) return true;
    return can(authStore.currentUser, item.permission);
  });
});

function toggleSidebar() {
  isCollapsed.value = !isCollapsed.value;
}

// Check if current page matches menu item
function isActive(href: string): boolean {
  if (typeof window === 'undefined') return false;
  return window.location.pathname === href || window.location.pathname.startsWith(href + '/');
}
</script>

<template>
  <aside
    :class="[
      'h-screen bg-card border-r transition-all duration-300 flex flex-col',
      isCollapsed ? 'w-16' : 'w-64'
    ]"
  >
    <!-- Logo -->
    <div class="h-16 flex items-center justify-between px-4 border-b">
      <span v-if="!isCollapsed" class="font-bold text-lg">POS App</span>
      <button
        @click="toggleSidebar"
        class="p-2 rounded-lg hover:bg-muted transition-colors"
      >
        <span v-if="isCollapsed">→</span>
        <span v-else>←</span>
      </button>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 overflow-y-auto py-4">
      <template v-for="item in filteredMenuItems" :key="item.href">
        <!-- Divider -->
        <div v-if="item.divider" class="my-2 border-t" />
        
        <!-- Menu Item -->
        <a
          v-else
          :href="item.href"
          :class="[
            'flex items-center gap-3 px-4 py-2 mx-2 rounded-lg transition-colors',
            isActive(item.href)
              ? 'bg-primary text-primary-foreground'
              : 'hover:bg-muted text-muted-foreground hover:text-foreground'
          ]"
        >
          <span class="text-lg">{{ item.icon }}</span>
          <span v-if="!isCollapsed" class="text-sm font-medium">{{ item.label }}</span>
        </a>
      </template>
    </nav>

    <!-- User Info -->
    <div class="p-4 border-t">
      <div class="flex items-center gap-3">
        <div class="w-8 h-8 rounded-full bg-primary flex items-center justify-center text-primary-foreground text-sm font-medium">
          {{ authStore.userName.charAt(0).toUpperCase() }}
        </div>
        <div v-if="!isCollapsed" class="flex-1 min-w-0">
          <p class="text-sm font-medium truncate">{{ authStore.userName }}</p>
          <p class="text-xs text-muted-foreground truncate">{{ (authStore.currentUser as any)?.role_name || 'User' }}</p>
        </div>
      </div>
    </div>
  </aside>
</template>
