<script setup lang="ts">
import { computed, type Component } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useLocaleStore } from '@/stores/locale';
import { can, type Permission } from '@/lib/rbac';
import { useOfflineQueueCount } from '@/composables/useOfflineQueueCount';
import { useOfflineAutoSync } from '@/composables/useOfflineAutoSync';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
  SidebarSeparator,
} from '@/components/ui/sidebar';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
import {
  LayoutDashboard,
  ShoppingCart,
  Package,
  Tag,
  ClipboardList,
  Table2,
  Users,
  DollarSign,
  CreditCard,
  Clock,
  BarChart3,
  Calculator,
  UserCog,
  Shield,
  KeyRound,
  FileText,
  Store,
  Settings,
  RefreshCw,
} from '@lucide/vue';

const authStore = useAuthStore();
const localeStore = useLocaleStore();
const t = localeStore.t;
/** Unsynced offline sales → badge on the "Sinkron Data" item. */
const { count: offlineCount } = useOfflineQueueCount();
// Replays the queue when the connection returns (mounted on every page).
useOfflineAutoSync();

interface SubMenuItem {
  label: string;
  href: string;
  icon?: Component;
}

interface MenuItem {
  label: string;
  icon: Component;
  href?: string;
  permission?: Permission;
  /** Only visible to the system-wide superadmin role (e.g. Store management). */
  superadminOnly?: boolean;
  /** Only visible to store admin & manager roles (store settings page). */
  adminOrManagerOnly?: boolean;
  children?: SubMenuItem[];
}

const menuGroups = computed(() => {
  const groups: { label: string; items: MenuItem[] }[] = [
    {
      label: t('sidebar.groupMain'),
      items: [
        { label: t('sidebar.dashboard'), icon: LayoutDashboard, href: '/dashboard' },
        { label: t('sidebar.pos'), icon: ShoppingCart, href: '/pos', permission: 'sales.create' },
      ],
    },
    {
      label: t('sidebar.groupCatalog'),
      items: [
        { label: t('sidebar.products'), icon: Package, href: '/admin/products', permission: 'products.read' },
        { label: t('sidebar.categories'), icon: Tag, href: '/admin/categories', permission: 'categories.read' },
        { label: t('sidebar.inventory'), icon: ClipboardList, href: '/admin/inventory', permission: 'inventory.read' },
        { label: t('sidebar.tables'), icon: Table2, href: '/admin/tables', permission: 'tables.read' },
      ],
    },
    {
      label: t('sidebar.groupTransactions'),
      items: [
        { label: t('sidebar.customers'), icon: Users, href: '/admin/customers', permission: 'customers.read' },
        { label: t('sidebar.sales'), icon: DollarSign, href: '/admin/sales', permission: 'sales.read' },
        { label: t('sidebar.payments'), icon: CreditCard, href: '/admin/payments', permission: 'payments.read' },
      ],
    },
    {
      label: t('sidebar.groupOperations'),
      items: [
        { label: t('sidebar.cashier'), icon: Clock, href: '/admin/cashier', permission: 'cashier.view_shift' },
        { label: t('sidebar.sync'), icon: RefreshCw, href: '/admin/offline-sync', permission: 'sales.create' },
        { label: t('sidebar.reports'), icon: BarChart3, href: '/reports', permission: 'reports.read' },
        { label: t('sidebar.accounting'), icon: Calculator, href: '/reports/accounting', permission: 'accounting.read' },
      ],
    },
    {
      label: t('sidebar.groupAdministration'),
      items: [
        { label: t('sidebar.users'), icon: UserCog, href: '/admin/users', permission: 'users.read' },
        { label: t('sidebar.roles'), icon: Shield, href: '/admin/roles', permission: 'roles.read' },
        { label: t('sidebar.permissions'), icon: KeyRound, href: '/admin/permissions', permission: 'permissions.read' },
        { label: t('sidebar.stores'), icon: Store, href: '/admin/stores', superadminOnly: true },
        { label: t('sidebar.activityLogs'), icon: FileText, href: '/admin/activity-logs', permission: 'activity_logs.read' },
        { label: t('sidebar.settings'), icon: Settings, href: '/admin/settings', adminOrManagerOnly: true },
      ],
    },
  ];

  return groups.map(group => ({
    ...group,
    items: group.items.filter(item => {
      if (item.superadminOnly) {
        return authStore.currentUser?.role_name === 'superadmin';
      }
      if (item.adminOrManagerOnly) {
        const role = authStore.currentUser?.role_name;
        return role === 'admin' || role === 'manager';
      }
      if (!item.permission) return true;
      return can(authStore.currentUser, item.permission);
    }),
  })).filter(group => group.items.length > 0);
});

function isActive(href: string): boolean {
  if (typeof window === 'undefined') return false;
  const pathname = window.location.pathname;
  const normalized = (str: string) => str.replace(/\/+$/, '').replace(/^\/+/, '');
  const p = normalized(pathname);
  const h = normalized(href);
  if (p === h || p.startsWith(h + '/')) return true;
  // Accounting sits under /reports — highlight both items from /reports/accounting.
  if (h === 'reports' && p === 'reports/accounting') return true;
  return false;
}
</script>

<template>
  <Sidebar collapsible="icon" variant="inset">
    <SidebarHeader>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton size="lg" as="a" href="/dashboard">
            <div class="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg">
              <Store class="size-4" />
            </div>
            <div class="grid flex-1 text-left text-sm leading-tight">
              <span class="truncate font-semibold">{{ t('sidebar.appName') }}</span>
              <span class="truncate text-xs">{{ t('sidebar.appTagline') }}</span>
            </div>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarHeader>

    <SidebarSeparator />

    <SidebarContent>
      <SidebarGroup v-for="group in menuGroups" :key="group.label">
        <SidebarGroupLabel>{{ group.label }}</SidebarGroupLabel>
        <SidebarGroupContent>
          <SidebarMenu>
            <SidebarMenuItem v-for="item in group.items" :key="item.label">
              <SidebarMenuButton as="a" :href="item.href" :tooltip="item.label" :is-active="isActive(item.href!)">
                <component :is="item.icon" />
                <span>{{ item.label }}</span>
              </SidebarMenuButton>
              <SidebarMenuBadge
                v-if="item.href === '/admin/offline-sync' && offlineCount > 0"
                class="bg-primary text-primary-foreground"
                :title="t('offlineSync.pendingBadge', { count: offlineCount })"
              >
                {{ offlineCount > 99 ? '99+' : offlineCount }}
              </SidebarMenuBadge>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>

    <SidebarFooter>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton size="lg">
            <div class="bg-primary text-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg">
              {{ authStore.userName.charAt(0).toUpperCase() }}
            </div>
            <div class="grid flex-1 text-left text-sm leading-tight">
              <span class="truncate font-semibold">{{ authStore.userName }}</span>
              <span class="truncate text-xs">{{ authStore.currentUser?.role_name || 'User' }}</span>
            </div>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarFooter>

    <SidebarRail />
  </Sidebar>
</template>
