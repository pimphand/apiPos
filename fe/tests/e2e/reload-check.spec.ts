import { test, expect } from '@playwright/test';

// Regression test for the "reload logs the user out" bug.
//
// Root cause (fixed): Astro view transitions (`<ClientRouter />`) pre-render the
// destination page in a hidden sub-frame. That sub-frame boots the app again,
// restores the session, and refreshes with the only valid refresh token — the
// server rotates (revokes) it and issues a new one, but the sub-frame is torn
// down before the replacement is persisted. The next reload then refreshes with
// the revoked token → 401 → logout.
//
// Fix: no view transitions. All navigation is a plain full page load, so a
// single context owns the session.
test('login then reload stays logged in', async ({ page }) => {
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

  // Log any frame attachments — a sub-frame appearing here means the bug's
  // mechanism is back.
  const frames: string[] = [];
  page.on('frameattached', (f) => frames.push(`SUB-FRAME ${f.url()}`));

  await page.click('button[type="submit"]');
  await page.waitForURL((url) => !url.pathname.startsWith('/login'), { timeout: 15000 });
  await page.waitForTimeout(1500);

  // No sub-frame should have been created for the login → dashboard navigation.
  console.log('FRAMES:', frames.length ? frames.join(' | ') : 'none');

  // Reload — the persisted refresh token must still be valid.
  await page.reload();
  await page.waitForTimeout(2500);
  console.log('FINAL URL:', page.url());
  expect(page.url().startsWith('http://localhost:4000/login')).toBe(false);
});
