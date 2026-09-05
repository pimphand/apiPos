import { test, expect } from '@playwright/test';

test('login then reload stays logged in', async ({ page }) => {
  const logs: Array<{ t: number; msg: string }> = [];
  const t0 = Date.now();
  const log = (msg: string) => logs.push({ t: Date.now() - t0, msg });

  page.on('console', (msg) => log(`[console] ${msg.text().slice(0, 200)}`));
  page.on('request', (req) => {
    if (req.url().includes('/api/')) {
      const f = req.frame();
      log(`[req] ${req.method()} ${req.url().replace('http://localhost:4000','')} frame=${f === page.mainFrame() ? 'MAIN' : 'SUB'}`);
    }
  });

  const uname = `reload_${Date.now()}`;
  const reg = await fetch('http://localhost:8990/api/v1/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      username: uname,
      email: `${uname}@test.com`,
      password: 'Password123!',
      full_name: 'Reload Test',
      device_id: 'browser-device-fixed',
      store_name: 'Reload Test Store',
    }),
  });
  expect(reg.status).toBe(200);

  await page.goto('/login');
  await page.waitForSelector('#username', { timeout: 10000 });
  await page.fill('#username', uname);
  await page.fill('#password', 'Password123!');
  await page.click('button[type="submit"]');
  await page.waitForURL((url) => !url.pathname.startsWith('/login'), { timeout: 15000 });
  await page.waitForTimeout(1200);

  // Decode the persisted refresh token in ALL frames
  const state = await page.evaluate(async () => {
    const raw = localStorage.getItem('pos_refresh_token') || '';
    let decoded = raw;
    try {
      const p = JSON.parse(raw);
      if (p && p.v === 1) {
        // decrypt with the same device-key approach is heavy; just show structure
        decoded = `encrypted-v1 salt=${p.s?.slice(0, 10)}...`;
      } else if (p && p.v === 0) {
        decoded = `xor-v0`;
      }
    } catch { decoded = `RAW:${raw.slice(0, 16)}...`; }
    return { len: raw.length, kind: decoded };
  });
  console.log('LS REFRESH BEFORE RELOAD:', JSON.stringify(state));

  log('PHASE: before reload');
  await page.reload();
  await page.waitForTimeout(3000);
  log(`PHASE: after reload URL=${page.url()}`);
  logs.forEach((l) => console.log(`[${l.t}ms] ${l.msg}`));
  expect(page.url().startsWith('http://localhost:4000/login')).toBe(false);
});
