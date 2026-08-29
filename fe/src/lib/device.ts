/**
 * Device binding + at-rest protection for persisted auth data.
 *
 * - `device_id`: a stable per-browser UUID used to bind refresh tokens on the
 *   backend so a token copied to another device is rejected.
 * - Stored refresh token / user profile are NOT persisted as raw values.
 *   When Web Crypto is available (secure context) they are AES-GCM encrypted
 *   with a key derived (PBKDF2) from the device id; otherwise they are
 *   obfuscated with a device-id-derived XOR transform so plaintext is never
 *   written to localStorage.
 */

const DEVICE_ID_KEY = 'pos_device_id';

/** Base64-encode bytes (chunked to avoid call-stack limits). */
function bytesToBase64(bytes: Uint8Array): string {
  let bin = '';
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    bin += String.fromCharCode(...Array.from(bytes.subarray(i, i + chunk)));
  }
  return btoa(bin);
}

/** Base64-decode into bytes. */
function base64ToBytes(b64: string): Uint8Array {
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) {
    bytes[i] = bin.charCodeAt(i);
  }
  return bytes;
}

/** Return the stable per-browser device id (creating it on first use). */
export function getDeviceId(): string {
  if (typeof window === 'undefined') return '';
  let id = window.localStorage.getItem(DEVICE_ID_KEY);
  if (!id) {
    id =
      typeof crypto.randomUUID === 'function'
        ? crypto.randomUUID()
        : `dev-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
    window.localStorage.setItem(DEVICE_ID_KEY, id);
  }
  return id;
}

const subtleAvailable =
  typeof crypto !== 'undefined' && !!crypto.subtle;

async function deriveAesKey(deviceId: string, salt: Uint8Array): Promise<CryptoKey> {
  const material = await crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(deviceId),
    'PBKDF2',
    false,
    ['deriveKey']
  );
  return crypto.subtle.deriveKey(
    { name: 'PBKDF2', salt: salt as BufferSource, iterations: 100000, hash: 'SHA-256' },
    material,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );
}

/** XOR a byte array with a repeating key (device id bytes). */
function xorBytes(input: Uint8Array, key: Uint8Array): Uint8Array {
  const out = new Uint8Array(input.length);
  for (let i = 0; i < input.length; i++) {
    out[i] = input[i] ^ key[i % key.length];
  }
  return out;
}

/** Lightweight device-keyed XOR obfuscation (fallback when no Web Crypto). */
function xorObfuscate(plain: string, deviceId: string): string {
  const key = new TextEncoder().encode(deviceId);
  const bytes = new TextEncoder().encode(plain);
  return bytesToBase64(xorBytes(bytes, key));
}

function xorDeobfuscate(stored: string, deviceId: string): string {
  const key = new TextEncoder().encode(deviceId);
  return new TextDecoder().decode(xorBytes(base64ToBytes(stored), key));
}

export interface EncryptedPayload {
  v: number;
  // aes fields
  s?: string;
  i?: string;
  d?: string;
  // xor fallback field
  x?: string;
}

/**
 * Encrypt a plaintext string for at-rest storage using the device-derived key.
 */
export async function encryptValue(plain: string): Promise<string> {
  const deviceId = getDeviceId();
  if (!deviceId) return plain;

  if (subtleAvailable) {
    const salt = crypto.getRandomValues(new Uint8Array(16));
    const iv = crypto.getRandomValues(new Uint8Array(12));
    const key = await deriveAesKey(deviceId, salt);
    const encrypted = await crypto.subtle.encrypt(
      { name: 'AES-GCM', iv },
      key,
      new TextEncoder().encode(plain)
    );
    const payload: EncryptedPayload = {
      v: 1,
      s: bytesToBase64(salt),
      i: bytesToBase64(iv),
      d: bytesToBase64(new Uint8Array(encrypted)),
    };
    return JSON.stringify(payload);
  }

  return JSON.stringify({ v: 0, x: xorObfuscate(plain, deviceId) } as EncryptedPayload);
}

/**
 * Decrypt a value previously produced by encryptValue.
 * Returns null on failure (e.g. copied to a different device / tampered).
 */
export async function decryptValue(stored: string): Promise<string | null> {
  const deviceId = getDeviceId();
  if (!deviceId) return null;

  let payload: EncryptedPayload;
  try {
    payload = JSON.parse(stored) as EncryptedPayload;
  } catch {
    return null;
  }

  if (payload.v === 1 && payload.s && payload.i && payload.d) {
    if (!subtleAvailable) return null;
    try {
      const salt = base64ToBytes(payload.s);
      const iv = base64ToBytes(payload.i);
      const key = await deriveAesKey(deviceId, salt);
      const decrypted = await crypto.subtle.decrypt(
        { name: 'AES-GCM', iv: iv as BufferSource },
        key,
        base64ToBytes(payload.d) as BufferSource
      );
      return new TextDecoder().decode(decrypted);
    } catch {
      return null;
    }
  }

  if (payload.v === 0 && payload.x) {
    try {
      return xorDeobfuscate(payload.x, deviceId);
    } catch {
      return null;
    }
  }

  return null;
}
