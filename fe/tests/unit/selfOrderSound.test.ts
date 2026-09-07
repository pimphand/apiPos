import { describe, it, expect, beforeEach } from 'vitest';
import {
  isSelfOrderSoundEnabled,
  setSelfOrderSoundEnabled,
} from '@/lib/selfOrderSound';

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
    key: (i: number) => Object.keys(store)[i] ?? null,
    removeItem: (k: string) => {
      delete store[k];
    },
    setItem: (k: string, v: string) => {
      store[k] = String(v);
    },
  };
}

describe('selfOrderSound', () => {
  beforeEach(() => {
    // The module guards on `typeof window === 'undefined'` and reads the
    // global `localStorage`, so stub both in the node test environment.
    (globalThis as { window?: unknown }).window = {};
    (globalThis as { localStorage?: Storage }).localStorage = createStorage();
  });

  afterEach(() => {
    delete (globalThis as { window?: unknown }).window;
    delete (globalThis as { localStorage?: Storage }).localStorage;
  });

  it('is enabled by default', () => {
    expect(isSelfOrderSoundEnabled()).toBe(true);
  });

  it('persists the mute toggle', () => {
    expect(setSelfOrderSoundEnabled(false)).toBe(false);
    expect(isSelfOrderSoundEnabled()).toBe(false);

    expect(setSelfOrderSoundEnabled(true)).toBe(true);
    expect(isSelfOrderSoundEnabled()).toBe(true);
  });
});