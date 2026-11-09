/**
 * Playwright regression test for the sidebar navigation bug:
 * "after login, clicking the sidebar sends the user back to /login".
 *
 * Root cause (regression from b990711): the latest commit re-introduced
 * Astro view transitions (<ClientRouter /> + navigate() from
 * 'astro:transitions/client') after they had been deliberately removed
 * (commit 39d42169c) because the transition machinery boots the destination
 * page in a hidden sub-frame. That second context independently restores the
 * session and consumes (rotates) the only valid refresh token, then is torn
 * down before persisting the replacement. The visible context is left with a
 * revoked token; the next sidebar navigation's page guard / API call fails
 * with 401 and redirects to /login.
 *
 * Fix: revert to plain full-page navigation (window.location.href) and remove
 * <ClientRouter /> from both layouts, restoring AuthBootstrap on protected
 * pages. The backend now also uses long-lived refresh tokens without
 * rotation, and authStorage persists the raw token synchronously, so even
 * if a sub-frame ever consumed a token the reload path still recovers —
 * but single-context navigation remains the correct defence.
 */

import { test, expect } from '@playwright/test';

test('login then sidebar navigation stays logged in', async ({ page }) => {
  const uname = `sidebar_${Date.now()}`;
  const reg = await fetch('http://localhost:8990/api/v1/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      username: uname,
      email: `${uname}@test.com`,
      password: 'Password123!',
      full_name: 'Sidebar Test',
      device_id: 'browser-device-fixed',
      store_name: 'Sidebar Test Store',
    }),
  });
  expect(reg.status).toBe(200);

  await page.goto('/login');
  await page.waitForSelector('#username', { timeout: 10000 });
  await page.fill('#username', uname);
  await page.fill('#password', 'Password123!');

  const frames: string[] = [];
  page.on('frameattached', (f) => frames.push(`SUB-FRAME ${f.url()}`));

  await page.click('button[type="submit"]');
  await page.waitForURL((url) => !url.pathname.startsWith('/login'), {
    timeout: 15000,
  });
  await page.waitForTimeout(1500);

  // 1. No sub-frame should have been created for view transitions.
  console.log('FRAMES:', frames.length ? frames.join(' | ') : 'none');
  expect(frames.length).toBe(0);

  // 2. Sidebar navigation must not bounce back to /login.
  //    Click a sidebar link (e.g. Products) and verify we stay authenticated.
  const currentUrl = page.url();
  expect(currentUrl.startsWith('http://localhost:4000/login')).toBe(false);

  // Navigate through the sidebar links like a user would.
  const sidebarLinks = await page.$$('a[href^="/admin/"], a[href^="/pos"], a[href^="/dashboard"]');
  if (sidebarLinks.length > 0) {
    // Click the first sidebar link and ensure we don't land on /login.
    const href = await sidebarLinks[0].getAttribute('href');
    await sidebarLinks[0].click().catch(() => {});
    await page.waitForTimeout(2500);
    console.log('AFTER SIDEBAR CLICK:', page.url(), 'href=', href);
    expect(page.url().startsWith('http://localhost:4000/login')).toBe(false);
  }

  // 3. Reload — the persisted refresh token must still be valid.
  await page.reload();
  await page.waitForTimeout(3000);
  console.log('FINAL URL:', page.url());
  expect(page.url().startsWith('http://localhost:4000/login')).toBe(false);
});
