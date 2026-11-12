import { describe, it, expect, afterEach, vi } from 'vitest';

const REFRESH_KEY = 'pos_refresh_token';

type Invoke = (cmd: string, args?: Record<string, unknown>) => Promise<unknown>;

function createStorage(): Storage {
  let store: Record<string, string> = {};
  return {
    get length() {
      return Object.keys(store).length;
    },
    clear: () => {
      store = {};
    },
    getItem: (k: string) => store[k] ?? null,
    key: (i: string | number) => Object.keys(store)[Number(i)] ?? null,
    removeItem: (k: string) => {
      delete store[k];
    },
    setItem: (k: string, v: string) => {
      store[k] = String(v);
    },
  };
}

/**
 * Loads a fresh auth module because `isTauriRuntime` is resolved at import
 * time. `seed` writes values into the fake localStorage before import.
 */
async function loadAuth(opts: { invoke?: Invoke; seed?: Record<string, string> }) {
  vi.resetModules();
  const store = createStorage();
  for (const [key, value] of Object.entries(opts.seed ?? {})) store.setItem(key, value);
  const win: Record<string, unknown> = {
    location: { origin: 'http://localhost' },
    localStorage: store,
  };
  if (opts.invoke) win.__TAURI_INTERNALS__ = { invoke: opts.invoke };
  (globalThis as { window?: unknown }).window = win;
  (globalThis as { localStorage?: Storage }).localStorage = store;
  const mod = await import('@/lib/auth');
  return { authStorage: mod.authStorage, store };
}

afterEach(() => {
  delete (globalThis as { window?: unknown }).window;
  delete (globalThis as { localStorage?: Storage }).localStorage;
});

describe('authStorage on desktop (OS keystore)', () => {
  it('reads the refresh token from the keystore, not localStorage', async () => {
    const invoke = vi.fn<Invoke>().mockImplementation(async (cmd) =>
      cmd === 'auth_refresh_get' ? 'native-token' : undefined
    );
    const { authStorage, store } = await loadAuth({ invoke });

    await expect(authStorage.getRefreshToken()).resolves.toBe('native-token');
    expect(invoke).toHaveBeenCalledWith('auth_refresh_get');
    expect(store.getItem(REFRESH_KEY)).toBeNull();
    expect(authStorage.hasPersistedRefresh()).toBe(true);
  });

  it('writes the rotated token to the keystore and never to localStorage', async () => {
    const invoke = vi.fn<Invoke>().mockResolvedValue(undefined);
    const { authStorage, store } = await loadAuth({ invoke });

    await authStorage.setRefreshToken('rotated-token');
    expect(invoke).toHaveBeenCalledWith('auth_refresh_set', { token: 'rotated-token' });
    expect(store.getItem(REFRESH_KEY)).toBeNull();
  });

  it('migrates a token left by an older desktop build', async () => {
    const invoke = vi.fn<Invoke>().mockImplementation(async (cmd) =>
      cmd === 'auth_refresh_get' ? null : undefined
    );
    const { authStorage, store } = await loadAuth({
      invoke,
      seed: { [REFRESH_KEY]: 'legacy-token' },
    });

    await expect(authStorage.getRefreshToken({ ignoreCache: true })).resolves.toBe('legacy-token');
    expect(invoke).toHaveBeenCalledWith('auth_refresh_set', { token: 'legacy-token' });
    expect(store.getItem(REFRESH_KEY)).toBeNull();
  });

  it('deletes the keystore item on logout', async () => {
    const invoke = vi.fn<Invoke>().mockImplementation(async (cmd) =>
      cmd === 'auth_refresh_get' ? 'native-token' : undefined
    );
    const { authStorage, store } = await loadAuth({ invoke });

    await authStorage.getRefreshToken();
    await authStorage.clear();

    expect(invoke).toHaveBeenCalledWith('auth_refresh_delete');
    expect(store.getItem(REFRESH_KEY)).toBeNull();
    expect(store.getItem('pos_user')).toBeNull();
  });
});

describe('authStorage fallback (keystore unavailable)', () => {
  it('falls back to the hardened localStorage path', async () => {
    const invoke = vi.fn<Invoke>().mockRejectedValue(new Error('ipc unavailable'));
    const { authStorage, store } = await loadAuth({ invoke });

    await expect(authStorage.getRefreshToken()).resolves.toBeNull();
    await authStorage.setRefreshToken('fallback-token');
    expect(store.getItem(REFRESH_KEY)).not.toBeNull();
  });
});
