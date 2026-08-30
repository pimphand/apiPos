import { describe, it, expect } from 'vitest';
import { loginSchema, registerSchema, paginationSchema } from '@/lib/validators';

describe('loginSchema', () => {
  it('accepts valid credentials', () => {
    const r = loginSchema.safeParse({ username: 'admin', password: 'secret123' });
    expect(r.success).toBe(true);
  });

  it('rejects empty username/password', () => {
    expect(loginSchema.safeParse({ username: '', password: 'x' }).success).toBe(false);
    expect(loginSchema.safeParse({ username: 'u', password: '' }).success).toBe(false);
  });

  it('rejects missing fields', () => {
    expect(loginSchema.safeParse({}).success).toBe(false);
  });
});

describe('registerSchema', () => {
  it('accepts valid registration', () => {
    const r = registerSchema.safeParse({
      full_name: 'John Doe',
      store_name: 'Johns Store',
      username: 'johndoe',
      email: 'john@example.com',
      password: 'secret123',
      confirm_password: 'secret123',
    });
    expect(r.success).toBe(true);
  });

  it('rejects missing store name', () => {
    const r = registerSchema.safeParse({
      full_name: 'John Doe',
      store_name: '',
      username: 'johndoe',
      email: 'john@example.com',
      password: 'secret123',
      confirm_password: 'secret123',
    });
    expect(r.success).toBe(false);
    expect(r.error?.issues[0].path).toContain('store_name');
  });

  it('rejects mismatched passwords', () => {
    const r = registerSchema.safeParse({
      full_name: 'John Doe',
      store_name: 'Johns Store',
      username: 'johndoe',
      email: 'john@example.com',
      password: 'secret123',
      confirm_password: 'different',
    });
    expect(r.success).toBe(false);
    expect(r.error?.issues[0].path).toContain('confirm_password');
  });

  it('rejects invalid email', () => {
    const r = registerSchema.safeParse({
      full_name: 'John Doe',
      store_name: 'Johns Store',
      username: 'johndoe',
      email: 'not-an-email',
      password: 'secret123',
      confirm_password: 'secret123',
    });
    expect(r.success).toBe(false);
  });

  it('rejects short passwords and invalid usernames', () => {
    expect(
      registerSchema.safeParse({
        full_name: 'J',
        store_name: 'Johns Store',
        username: 'jd',
        email: 'a@b.co',
        password: '123',
        confirm_password: '123',
      }).success
    ).toBe(false);
    expect(
      registerSchema.safeParse({
        full_name: 'John',
        store_name: 'Johns Store',
        username: 'bad user!',
        email: 'a@b.co',
        password: 'secret123',
        confirm_password: 'secret123',
      }).success
    ).toBe(false);
  });
});

describe('paginationSchema', () => {
  it('defaults missing values to page=1, per_page=20', () => {
    expect(paginationSchema.parse({}).page).toBe(1);
    expect(paginationSchema.parse({}).per_page).toBe(20);
  });

  it('rejects out-of-bounds per_page and page', () => {
    expect(paginationSchema.safeParse({ per_page: 0 }).success).toBe(false);
    expect(paginationSchema.safeParse({ per_page: 500 }).success).toBe(false);
    expect(paginationSchema.safeParse({ page: 0 }).success).toBe(false);
  });
});
