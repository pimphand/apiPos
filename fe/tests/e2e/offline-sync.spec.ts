/**
 * End-to-end test for the offline POS flow (docs/pos-offline-sync.md):
 *
 *   login → open POS → add a product → cut the network → pay → the sale is
 *   queued with an OFFLINE- receipt → reconnect → Sinkron Data → the queued
 *   sale is replayed against POST /v1/sales and appears on the server.
 *
 * Requires the real gateway on :8990 with a migrated database (same contract
 * as the other e2e specs in this folder). The store is created through
 * /auth/register, verified by the seeded superadmin, then stocked with one
 * product/SKU so the test never depends on demo data.
 */
import { test, expect } from '@playwright/test';

const API = 'http://localhost:8990/api/v1';
const DEVICE_ID = 'browser-device-fixed';

// The app defaults to English (locale store falls back to 'en' for a fresh
// profile), so the labels below are the deterministic English ones.

interface ApiResult {
  status: number;
  data: Record<string, any>;
}

async function api(
  method: 'GET' | 'POST',
  path: string,
  body?: unknown,
  token?: string
): Promise<ApiResult> {
  const res = await fetch(`${API}${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await res.text();
  let data: Record<string, any> = {};
  try {
    data = text ? JSON.parse(text) : {};
  } catch {
    data = { raw: text };
  }
  return { status: res.status, data };
}

async function seedStore(productName: string, skuCode: string, storeName: string) {
  const stamp = Date.now();
  const username = `offline_${stamp}`;
  const password = 'Password123!';

  const reg = await api('POST', '/auth/register', {
    username,
    email: `${username}@test.com`,
    password,
    full_name: 'Offline E2E',
    device_id: DEVICE_ID,
    store_name: storeName,
  });
  expect(reg.status, `register: ${JSON.stringify(reg.data)}`).toBe(200);

  const superLogin = await api('POST', '/auth/login', {
    username: 'superadmin',
    password: 'SuperAdmin123!',
    device_id: DEVICE_ID,
  });
  expect(superLogin.status, `superadmin login: ${JSON.stringify(superLogin.data)}`).toBe(200);
  const superToken = superLogin.data.access_token as string;

  let storeId = (reg.data.user?.store_id ?? reg.data.user?.storeId ?? '') as string;
  if (!storeId) {
    const pending = await api('GET', '/stores?is_verified=false&per_page=100', undefined, superToken);
    const match = (pending.data.stores ?? []).find(
      (s: Record<string, any>) => s.name === storeName
    );
    storeId = match?.id ?? '';
  }
  expect(storeId, 'registered store id').toBeTruthy();

  const verify = await api('POST', `/stores/${storeId}/verify`, {}, superToken);
  if (verify.status !== 200) {
    // Registration may already activate the store in some builds; accept that
    // only when the store really is verified.
    const store = await api('GET', `/stores/${storeId}`, undefined, superToken);
    const verified = store.data.store?.is_verified ?? store.data.is_verified;
    console.log('verify fallback:', verify.status, JSON.stringify(store.data).slice(0, 200));
    expect(String(verified)).toBe('true');
  }

  const login = await api('POST', '/auth/login', {
    username,
    password,
    device_id: DEVICE_ID,
  });
  expect(login.status, `owner login: ${JSON.stringify(login.data)}`).toBe(200);
  const token = login.data.access_token as string;

  const category = await api('POST', '/categories', { name: `Offline Cat ${stamp}` }, token);
  expect(category.status, `category: ${JSON.stringify(category.data)}`).toBe(200);
  const categoryId = (category.data.category?.id ?? category.data.id) as string;
  expect(categoryId).toBeTruthy();

  const product = await api(
    'POST',
    '/products',
    {
      name: productName,
      category_id: categoryId,
      cost_price: 10000,
      selling_price: 15000,
      stock: 50,
      minimum_stock: 5,
      sku: skuCode,
      unit: 'pcs',
    },
    token
  );
  expect(product.status, `product: ${JSON.stringify(product.data)}`).toBe(200);

  const shift = await api(
    'POST',
    '/cashier/shifts/open',
    { store_id: storeId, opening_cash: 100000 },
    token
  );
  expect(shift.status, `shift: ${JSON.stringify(shift.data)}`).toBe(200);

  return { username, password, storeId };
}

test('offline checkout is queued and synced from Sinkron Data', async ({ page, context }) => {
  test.setTimeout(120_000);

  const stamp = Date.now();
  const productName = `Kopi Offline ${stamp}`;
  const skuCode = `OFF-${stamp}`;
  const storeName = `Offline E2E Store ${stamp}`;
  const { username, password, storeId } = await seedStore(productName, skuCode, storeName);

  // 1. Login through the real form.
  await page.goto('/login');
  await page.waitForSelector('#username', { timeout: 15000 });
  await page.fill('#username', username);
  await page.fill('#password', password);
  await page.click('button[type="submit"]');
  await page.waitForURL((url) => !url.pathname.startsWith('/login'), { timeout: 20000 });

  // 2. POS loads the product grid + an OPEN shift, and caches both for offline.
  await page.goto('/pos');
  await expect(page.getByText(productName, { exact: false }).first()).toBeVisible({
    timeout: 20000,
  });

  const cacheBefore = await page.evaluate(() => {
    const raw = localStorage.getItem('pos.offline_cache.v1');
    return raw ? JSON.parse(raw) : null;
  });
  expect(cacheBefore?.stores?.[storeId]?.products?.length ?? 0).toBeGreaterThan(0);
  expect(cacheBefore?.stores?.[storeId]?.shift?.status).toBe('OPEN');

  // 3. Add the SKU to the cart and open the payment dialog.
  const skuRow = page.locator('tr').filter({ hasText: skuCode }).first();
  await skuRow.getByRole('button', { name: 'Add' }).click();
  await page.getByRole('button', { name: /^Pay/ }).first().click();

  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  // Pay the exact total (15.000 + 11% tax). NOTE: the backend accounting
  // posting currently rejects overpaid cash (it debits the tendered amount but
  // credits only the sale total, so change makes the journal unbalanced).
  // Exact payment keeps this test focused on the offline queue, and the bug is
  // recorded in docs/progress.md as a backend finding.
  await dialog.locator('input[type="number"]').fill('16650');

  // 4. Cut the network and finish the payment: the POST cannot reach the
  //    gateway, so the sale must be stored locally with an OFFLINE receipt.
  await context.setOffline(true);
  await page.waitForTimeout(400);
  await dialog.getByRole('button', { name: 'Pay', exact: true }).click();

  await expect(page.getByText(/OFFLINE-[A-Z0-9]{8}/)).toBeVisible({ timeout: 20000 });

  const queued = await page.evaluate(() => {
    const raw = localStorage.getItem('pos.offline_queue.v1');
    return raw ? JSON.parse(raw) : null;
  });
  expect(queued?.entries ?? []).toHaveLength(1);
  const record = queued.entries[0];
  expect(record.status).toBe('pending');
  expect(record.store_id).toBe(storeId);
  expect(record.payload.items).toHaveLength(1);
  expect(record.payload.items[0].quantity).toBe(1);

  // 5. Close the receipt, reconnect, and sync from the menu.
  // Close the receipt via its last footer action (label-independent).
  await page.getByRole('dialog').getByRole('button').last().click();
  await context.setOffline(false);
  await page.waitForTimeout(300);

  await page.goto('/admin/offline-sync');
  await expect(page.getByRole('heading', { name: 'Transaction Sync' })).toBeVisible({
    timeout: 20000,
  });
  await expect(page.getByText(/OFFLINE-[A-Z0-9]{8}/)).toBeVisible();
  await page.getByRole('button', { name: 'Sync All' }).click();

  await expect(page.getByText('No sales waiting to sync')).toBeVisible({
    timeout: 25000,
  });

  const queueAfter = await page.evaluate(() => {
    const raw = localStorage.getItem('pos.offline_queue.v1');
    return raw ? JSON.parse(raw) : { entries: [] };
  });
  expect(queueAfter.entries).toHaveLength(0);

  // 6. The server must now hold the sale: total 15.000 + 11% tax = 16.650.
  const login = await api('POST', '/auth/login', {
    username,
    password,
    device_id: DEVICE_ID,
  });
  const sales = await api('GET', '/sales?per_page=100', undefined, login.data.access_token as string);
  expect(sales.status, JSON.stringify(sales.data)).toBe(200);
  const synced = (sales.data.sales ?? []).find(
    (s: Record<string, any>) =>
      Number(s.grand_total ?? s.grandTotal) === 16650 &&
      String(s.status).toUpperCase() === 'PAID'
  );
  expect(synced, `synced sale not found: ${JSON.stringify(sales.data).slice(0, 300)}`).toBeTruthy();
  console.log('SYNCED INVOICE:', synced.invoice_number ?? synced.invoiceNumber);
});
