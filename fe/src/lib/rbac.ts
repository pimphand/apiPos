/**
 * RBAC (Role-Based Access Control) helper
 * 
 * Permissions format: {resource}.{action}
 * Examples: products.create, sales.read, inventory.adjust
 */

import type { User } from '@/types';

// Permission resources
export type Resource = 
  | 'users' | 'roles' | 'permissions' 
  | 'categories' | 'products' | 'inventory'
  | 'customers' | 'sales' | 'payments' 
  | 'cashier' | 'reports' | 'activity_logs';

// Permission actions
export type Action = 
  | 'create' | 'read' | 'update' | 'delete' 
  | 'assign' | 'adjust' | 'cancel' | 'refund'
  | 'open_shift' | 'close_shift' | 'view_shift';

// Full permission string
export type Permission = `${Resource}.${Action}`;

// Role-permission mapping (default roles)
const ROLE_PERMISSIONS: Record<string, Permission[]> = {
  superadmin: [
    // Full access to everything (system-wide)
    'users.create', 'users.read', 'users.update', 'users.delete',
    'roles.create', 'roles.read', 'roles.update', 'roles.delete',
    'permissions.create', 'permissions.read', 'permissions.update', 'permissions.delete', 'permissions.assign',
    'categories.create', 'categories.read', 'categories.update', 'categories.delete',
    'products.create', 'products.read', 'products.update', 'products.delete',
    'inventory.read', 'inventory.adjust',
    'customers.create', 'customers.read', 'customers.update', 'customers.delete',
    'sales.create', 'sales.read', 'sales.cancel', 'sales.refund',
    'payments.create', 'payments.read',
    'cashier.open_shift', 'cashier.close_shift', 'cashier.view_shift',
    'reports.read', 'activity_logs.read',
  ],
  admin: [
    // Full access to everything
    'users.create', 'users.read', 'users.update', 'users.delete',
    'roles.create', 'roles.read', 'roles.update', 'roles.delete',
    'permissions.create', 'permissions.read', 'permissions.update', 'permissions.delete', 'permissions.assign',
    'categories.create', 'categories.read', 'categories.update', 'categories.delete',
    'products.create', 'products.read', 'products.update', 'products.delete',
    'inventory.read', 'inventory.adjust',
    'customers.create', 'customers.read', 'customers.update', 'customers.delete',
    'sales.create', 'sales.read', 'sales.cancel', 'sales.refund',
    'payments.create', 'payments.read',
    'cashier.open_shift', 'cashier.close_shift', 'cashier.view_shift',
    'reports.read',
    'activity_logs.read',
  ],
  manager: [
    // Products, inventory, sales, reports
    'products.create', 'products.read', 'products.update', 'products.delete',
    'categories.create', 'categories.read', 'categories.update', 'categories.delete',
    'inventory.read', 'inventory.adjust',
    'customers.create', 'customers.read', 'customers.update', 'customers.delete',
    'sales.create', 'sales.read', 'sales.cancel', 'sales.refund',
    'payments.create', 'payments.read',
    'cashier.open_shift', 'cashier.close_shift', 'cashier.view_shift',
    'reports.read',
  ],
  cashier: [
    // Sales, customers, current shift only
    'customers.read',
    'sales.create', 'sales.read',
    'payments.create', 'payments.read',
    'cashier.open_shift', 'cashier.close_shift', 'cashier.view_shift',
    'products.read',
  ],
};

/**
 * Check if a user has a specific permission.
 *
 * Priority:
 * 1. Dynamic permissions fetched from the backend (user.permissions).
 *    This reflects the actual role-permission assignment and should
 *    always be used when available.
 * 2. Hardcoded ROLE_PERMISSIONS fallback for users whose profile
 *    hasn't been enriched yet (e.g. stale cache before refresh).
 */
export function can(user: User | null, permission: Permission): boolean {
  if (!user) return false;

  // 1. Use the dynamic permissions array when present.
  //    An explicit empty array [] means the role has NO permissions —
  //    do NOT fall back to the hardcoded map.
  if (user.permissions !== undefined) {
    return user.permissions.includes(permission);
  }

  // 2. Fallback to hardcoded defaults ONLY when permissions haven't been
  //    fetched from the backend yet (undefined = not loaded).
  const roleName = (user as any).role_name || '';
  const rolePerms = ROLE_PERMISSIONS[roleName] || [];
  return rolePerms.includes(permission);
}

/**
 * Check if a user has any of the specified permissions
 */
export function canAny(user: User | null, permissions: Permission[]): boolean {
  return permissions.some(p => can(user, p));
}

/**
 * Check if a user has all of the specified permissions
 */
export function canAll(user: User | null, permissions: Permission[]): boolean {
  return permissions.every(p => can(user, p));
}

/**
 * Get all permissions for a role
 */
export function getRolePermissions(roleName: string): Permission[] {
  return ROLE_PERMISSIONS[roleName] || [];
}

/**
 * Check if user is admin
 */
export function isAdmin(user: User | null): boolean {
  if (!user) return false;
  const role = (user as any).role_name;
  return role === 'admin' || role === 'superadmin';
}

/**
 * Check if user is manager or admin
 */
export function isManagerOrAbove(user: User | null): boolean {
  if (!user) return false;
  const role = (user as any).role_name;
  return role === 'superadmin' || role === 'admin' || role === 'manager';
}
