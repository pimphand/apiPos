/**
 * Native desktop integration helpers.
 *
 * The web build has no native layer, so every helper here is a no-op unless the
 * app runs inside the Tauri webview. Callers keep their existing browser
 * behaviour (in-app toast, `<input type=file>`, anchor download) and only take
 * the native path when a helper reports success. See docs/TAURI_DESKTOP.md §6.
 */

import { isTauriRuntime } from '@/lib/apiBase';

type TauriInvoke = (cmd: string, args?: Record<string, unknown>) => Promise<unknown>;

const NOTIFY_COMMAND = 'desktop_notify';
const TRAY_BADGE_COMMAND = 'tray_set_offline_count';
const PICK_IMPORT_COMMAND = 'pick_import_file';
const SAVE_EXPORT_COMMAND = 'save_export_file';

/** Last count sent to the tray, so re-mounts do not repaint the same icon. */
let lastTrayOfflineCount: number | null = null;

function nativeInvoke(): TauriInvoke | null {
  if (!isTauriRuntime || typeof window === 'undefined') return null;
  const internals = (window as unknown as { __TAURI_INTERNALS__?: { invoke?: TauriInvoke } })
    .__TAURI_INTERNALS__;
  return internals?.invoke ?? null;
}

/** Whether native notifications can be sent from this runtime. */
export function isNativeNotificationAvailable(): boolean {
  return nativeInvoke() !== null;
}

/** Whether the app can open/save files through OS dialogs. */
export function isNativeFileDialogAvailable(): boolean {
  return nativeInvoke() !== null;
}

/**
 * Show a native OS notification (macOS Notification Center, Windows toast,
 * Linux notification daemon). Resolves to `true` only when the OS accepted it,
 * so callers can log a real failure instead of assuming delivery. Never throws.
 */
export async function notifyNative(title: string, body: string): Promise<boolean> {
  const invoke = nativeInvoke();
  const cleanTitle = title.trim();
  const cleanBody = body.trim();
  if (!invoke || (!cleanTitle && !cleanBody)) return false;
  try {
    await invoke(NOTIFY_COMMAND, { title: cleanTitle, body: cleanBody });
    return true;
  } catch {
    // The OS may refuse notifications (permission revoked, no daemon). The
    // in-app toast the caller already shows stays the source of truth.
    return false;
  }
}

/**
 * Show the number of queued offline sales on the desktop tray icon and
 * tooltip. No-op in the browser build. Never throws; resolves `true` only when
 * the shell accepted the update.
 */
export async function setTrayOfflineCount(count: number): Promise<boolean> {
  const invoke = nativeInvoke();
  if (!invoke) return false;
  const safe = Number.isFinite(count) ? Math.max(0, Math.floor(count)) : 0;
  if (lastTrayOfflineCount === safe) return true;
  try {
    await invoke(TRAY_BADGE_COMMAND, { count: safe });
    lastTrayOfflineCount = safe;
    return true;
  } catch {
    // Tray unavailable (e.g. Linux without appindicator) is not fatal.
    return false;
  }
}

function base64ToBytes(value: string): Uint8Array {
  const binary = atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes;
}

function bytesToBase64(bytes: Uint8Array): string {
  let binary = '';
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...Array.from(bytes.subarray(i, i + chunk)));
  }
  return btoa(binary);
}

interface NativePickedFile {
  name: string;
  size: number;
  data: string;
}

/**
 * Open the OS file dialog for a product import. Returns the chosen file (bytes
 * already read by Rust) or `null` when unavailable/cancelled, in which case the
 * browser `<input type=file>` path keeps working.
 */
export async function pickImportFile(): Promise<File | null> {
  const invoke = nativeInvoke();
  if (!invoke) return null;
  try {
    const picked = (await invoke(PICK_IMPORT_COMMAND)) as NativePickedFile | null;
    if (!picked) return null;
    const bytes = base64ToBytes(picked.data);
    // Copy into a plain ArrayBuffer-backed view so it is a valid BlobPart.
    const copy = Uint8Array.from(bytes);
    return new File([copy], picked.name, { type: 'application/octet-stream' });
  } catch {
    return null;
  }
}

export type SaveExportResult = 'saved' | 'cancelled' | 'unsupported';

/**
 * Save an exported report through the OS save dialog. The browser build returns
 * `'unsupported'` so the caller can fall back to an anchor download; a user
 * cancelling the dialog returns `'cancelled'` and must not trigger a download.
 */
export async function saveExportFile(
  blob: Blob,
  suggestedName: string
): Promise<SaveExportResult> {
  const invoke = nativeInvoke();
  if (!invoke) return 'unsupported';
  try {
    const contents = bytesToBase64(new Uint8Array(await blob.arrayBuffer()));
    const path = await invoke(SAVE_EXPORT_COMMAND, { suggestedName, contents });
    return typeof path === 'string' && path.length > 0 ? 'saved' : 'cancelled';
  } catch {
    return 'unsupported';
  }
}
