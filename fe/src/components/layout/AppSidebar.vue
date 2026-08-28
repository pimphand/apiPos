<script setup lang="ts">
import { computed, ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { can, type Permission } from '@/lib/rbac';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
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
  Users,
  DollarSign,
  CreditCard,
  Clock,
  BarChart3,
  UserCog,
  Shield,
  FileText,
  Store,
} from '@lucide/vue';

const authStore = useAuthStore();

interface SubMenuItem {
  label: string;
  href: string;
  icon?: any;
}

interface MenuItem {
  label: string;
  icon: any;
  href?: string;
  permission?: Permission;
  children?: SubMenuItem[];
}

const menuGroups = computed(() => {
  const groups: { label: string; items: MenuItem[] }[] = [
    {
      label: 'Main',
      items: [
        { label: 'Dashboard', icon: LayoutDashboard, href: '/dashboard' },
        { label: 'POS Checkout', icon: ShoppingCart, href: '/pos', permission: 'sales.create' },
      ],
    },
    {
      label: 'Catalog',
      items: [
        { label: 'Products', icon: Package, href: '/admin/products', permission: 'products.read' },
        { label: 'Categories', icon: Tag, href: '/admin/categories', permission: 'categories.read' },
        { label: 'Inventory', icon: ClipboardList, href: '/admin/inventory', permission: 'inventory.read' },
      ],
    },
    {
      label: 'Transactions',
      items: [
        { label: 'Customers', icon: Users, href: '/admin/customers', permission: 'customers.read' },
        { label: 'Sales', icon: DollarSign, href: '/admin/sales', permission: 'sales.read' },
        { label: 'Payments', icon: CreditCard, href: '/admin/payments', permission: 'payments.read' },
      ],
    },
    {
      label: 'Operations',
      items: [
        { label: 'Cashier Shifts', icon: Clock, href: '/admin/cashier', permission: 'cashier.view_shift' },
        { label: 'Reports', icon: BarChart3, href: '/reports', permission: 'reports.read' },
      ],
    },
    {
      label: 'Administration',
      items: [
        { label: 'Users', icon: UserCog, href: '/admin/users', permission: 'users.read' },
        { label: 'Roles', icon: Shield, href: '/admin/roles', permission: 'roles.read' },
        { label: 'Activity Logs', icon: FileText, href: '/admin/activity-logs', permission: 'activity_logs.read' },
      ],
    },
  ];

  return groups.map(group => ({
    ...group,
    items: group.items.filter(item => {
      if (!item.permission) return true;
      return can(authStore.currentUser, item.permission);
    }),
  })).filter(group => group.items.length > 0);
});

function isActive(href: string): boolean {
  if (typeof window === 'undefined') return false;
  return window.location.pathname === href || window.location.pathname.startsWith(href + '/');
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
              <span class="truncate font-semibold">POS App</span>
              <span class="truncate text-xs">Point of Sale</span>
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
              <span class="truncate text-xs">{{ (authStore.currentUser as any)?.role_name || 'User' }}</span>
            </div>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarFooter>

    <SidebarRail />
  </Sidebar>
</template>
