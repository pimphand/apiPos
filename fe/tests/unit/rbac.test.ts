import { describe, it, expect } from 'vitest';
import { can, canAny, isAdmin, isManagerOrAbove } from '@/lib/rbac';
import type { User } from '@/types';

function user(role_name: string): User {
  return {
    id: 'u1',
    username: role_name,
    email: `${role_name}@example.com`,
    full_name: '',
    role_id: role_name,
    role_name,
    store_id: 's1',
    is_active: true,
  };
}

describe('rbac can()', () => {
  it('denies when user is null', () => {
    expect(can(null, 'products.create')).toBe(false);
    expect(canAny(null, ['sales.read', 'reports.read'])).toBe(false);
  });

  it('gives admin full access', () => {
    const admin = user('admin');
    expect(can(admin, 'users.delete')).toBe(true);
    expect(can(admin, 'activity_logs.read')).toBe(true);
    expect(can(admin, 'products.read')).toBe(true);
  });

  it('gives manager product/inventory/sales/reports access but not user admin', () => {
    const manager = user('manager');
    expect(can(manager, 'products.create')).toBe(true);
    expect(can(manager, 'inventory.adjust')).toBe(true);
    expect(can(manager, 'reports.read')).toBe(true);
    expect(can(manager, 'users.delete')).toBe(false);
  });

  it('gives cashier only sales/customer/shift access', () => {
    const cashier = user('cashier');
    expect(can(cashier, 'sales.create')).toBe(true);
    expect(can(cashier, 'customers.read')).toBe(true);
    expect(can(cashier, 'products.read')).toBe(true);
    expect(can(cashier, 'reports.read')).toBe(false);
    expect(can(cashier, 'users.read')).toBe(false);
  });

  it('denies unknown roles', () => {
    expect(can(user('guest'), 'sales.read')).toBe(false);
  });
});

describe('rbac role helpers', () => {
  it('isAdmin true only for admin', () => {
    expect(isAdmin(user('admin'))).toBe(true);
    expect(isAdmin(user('manager'))).toBe(false);
  });

  it('isManagerOrAbove true for admin and manager', () => {
    expect(isManagerOrAbove(user('admin'))).toBe(true);
    expect(isManagerOrAbove(user('manager'))).toBe(true);
    expect(isManagerOrAbove(user('cashier'))).toBe(false);
  });
});
