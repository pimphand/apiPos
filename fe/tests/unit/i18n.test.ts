import { describe, it, expect } from 'vitest';
import { translate, messages } from '@/i18n/messages';

describe('i18n messages', () => {
  it('returns the key when a translation is missing', () => {
    expect(translate('en', 'nonexistent.key')).toBe('nonexistent.key');
  });

  it('translates deep nested keys', () => {
    expect(translate('en', 'sidebar.dashboard')).toBe('Dashboard');
    expect(translate('en', 'auth.login.welcome')).toBe('Welcome Back');
    expect(translate('en', 'topbar.logout')).toBe('Logout');
  });

  it('translates to Indonesian', () => {
    expect(translate('id', 'sidebar.dashboard')).toBe('Dashboard');
    expect(translate('id', 'sidebar.products')).toBe('Produk');
    expect(translate('id', 'sidebar.inventory')).toBe('Stok');
    expect(translate('id', 'auth.login.welcome')).toBe('Selamat Datang Kembali');
    expect(translate('id', 'topbar.logout')).toBe('Keluar');
  });

  it('falls back to english for unknown locale', () => {
    // @ts-expect-error testing unknown locale fallback
    expect(translate('xx', 'sidebar.dashboard')).toBe('Dashboard');
  });

  it('supports parameter interpolation', () => {
    expect(translate('en', 'dashboard.welcome', { name: 'Budi' })).toBe('Welcome back, Budi! 👋');
    expect(translate('id', 'dashboard.welcome', { name: 'Budi' })).toBe('Selamat Datang kembali, Budi! 👋');
    expect(translate('en', 'placeholder.comingSoon', { title: 'Sales' })).toBe('Sales coming soon');
  });

  it('has matching keys across en and id', () => {
    const flatKeys = (obj: Record<string, unknown>, prefix = ''): string[] =>
      Object.entries(obj).flatMap(([k, v]) => {
        const key = prefix ? `${prefix}.${k}` : k;
        return typeof v === 'string' ? [key] : flatKeys(v as Record<string, unknown>, key);
      });

    const enKeys = flatKeys(messages.en as unknown as Record<string, unknown>);
    const idKeys = flatKeys(messages.id as unknown as Record<string, unknown>);

    const enSet = new Set(enKeys);
    const idSet = new Set(idKeys);

    const missingInId = enSet.difference(idSet);
    const missingInEn = idSet.difference(enSet);

    expect([...missingInId]).toEqual([]);
    expect([...missingInEn]).toEqual([]);
  });
});
