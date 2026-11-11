/**
 * Single source of truth for the backend location.
 *
 * Web (nginx / Vite dev server): relative `/api` through the existing proxy
 * (`astro.config.mjs` in dev, `Dockerfile.fe.nginx` in production).
 *
 * Desktop (Tauri): the webview is served from the `tauri://` (or
 * `http://tauri.localhost` on Windows) origin and there is NO proxy, so a
 * packaged build must carry an absolute `PUBLIC_API_BASE_URL`.
 * `desktop/scripts/build-frontend.mjs` rejects a desktop build without it.
 *
 * `tauri dev` loads the Vite dev server (`devUrl`) and therefore keeps using
 * the relative base and its `/api` + `/ws` proxy.
 *
 * There is intentionally no localhost fallback: a misconfigured packaged build
 * fails loudly instead of silently calling the app's own origin.
 */

const RAW_BASE = (import.meta.env.PUBLIC_API_BASE_URL || '').trim();

/** True inside the Tauri webview (set before any app script runs). */
export const isTauriRuntime =
  typeof window !== 'undefined' &&
  '__TAURI_INTERNALS__' in (window as unknown as Record<string, unknown>);

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '');
}

function isAbsoluteHttp(value: string): boolean {
  return /^https?:\/\//i.test(value);
}

/** Axios `baseURL` — absolute when configured, relative in browser/dev. */
export const API_BASE_URL = trimTrailingSlash(RAW_BASE || '/api');

/** Origin of the API (e.g. `https://pos.example.com`), for native integrations. */
export const API_ORIGIN = isAbsoluteHttp(API_BASE_URL)
  ? new URL(API_BASE_URL).origin
  : typeof window !== 'undefined'
    ? window.location.origin
    : '';

if (
  typeof console !== 'undefined' &&
  isTauriRuntime &&
  !import.meta.env.DEV &&
  !isAbsoluteHttp(API_BASE_URL)
) {
  // A packaged desktop build bypassed desktop/scripts/build-frontend.mjs.
  console.error(
    `[apiBase] Packaged desktop build has no absolute API base ("${API_BASE_URL}"). ` +
      'Rebuild with PUBLIC_API_BASE_URL=https://api.example.com/api (see desktop/scripts/build-frontend.mjs).'
  );
}

/** Absolute WebSocket URL for `/ws` (no query string; token added by caller). */
export const WS_URL = (() => {
  if (isAbsoluteHttp(API_BASE_URL)) {
    const url = new URL(API_BASE_URL);
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    url.pathname = `${url.pathname.replace(/\/api$/, '')}/ws`;
    url.search = '';
    url.hash = '';
    return url.toString().replace(/\/$/, '');
  }

  if (typeof window === 'undefined') return '/ws';

  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}/ws`;
})();
