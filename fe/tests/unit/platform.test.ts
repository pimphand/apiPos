import { describe, it, expect, afterEach, vi } from 'vitest';

type Invoke = (cmd: string, args?: Record<string, unknown>) => Promise<unknown>;

/**
 * Loads a fresh copy of the module with a stubbed global window, because the
 * Tauri detection is evaluated once at import time.
 */
async function loadPlatform(invoke?: Invoke) {
  vi.resetModules();
  const win: Record<string, unknown> = { location: { origin: 'http://localhost' } };
  if (invoke) win.__TAURI_INTERNALS__ = { invoke };
  (globalThis as { window?: unknown }).window = win;
  return import('@/lib/platform');
}

afterEach(() => {
  delete (globalThis as { window?: unknown }).window;
});

describe('platform notifications (web)', () => {
  it('reports no native notifications and is a silent no-op', async () => {
    // Browser runtime: no __TAURI_INTERNALS__ on window.
    const { isNativeNotificationAvailable, notifyNative } = await loadPlatform();

    expect(isNativeNotificationAvailable()).toBe(false);
    await expect(notifyNative('New order', 'Table 4')).resolves.toBe(false);
  });
});

describe('platform notifications (Tauri)', () => {
  it('invokes the native command with trimmed text', async () => {
    const invoke = vi.fn<Invoke>().mockResolvedValue(undefined);
    const { isNativeNotificationAvailable, notifyNative } = await loadPlatform(invoke);

    expect(isNativeNotificationAvailable()).toBe(true);
    await expect(notifyNative('  New order ', ' Table 4 ')).resolves.toBe(true);
    expect(invoke).toHaveBeenCalledWith('desktop_notify', {
      title: 'New order',
      body: 'Table 4',
    });
  });

  it('skips empty notifications', async () => {
    const invoke = vi.fn<Invoke>().mockResolvedValue(undefined);
    const { notifyNative } = await loadPlatform(invoke);

    await expect(notifyNative('   ', '  ')).resolves.toBe(false);
    expect(invoke).not.toHaveBeenCalled();
  });

  it('reports failure instead of throwing when the OS refuses', async () => {
    const invoke = vi.fn<Invoke>().mockRejectedValue(new Error('permission denied'));
    const { notifyNative } = await loadPlatform(invoke);

    await expect(notifyNative('New order', 'Table 4')).resolves.toBe(false);
  });
});

describe('platform file dialogs (web)', () => {
  it('keeps the browser import input and download path', async () => {
    const { isNativeFileDialogAvailable, pickImportFile, saveExportFile } = await loadPlatform();

    expect(isNativeFileDialogAvailable()).toBe(false);
    await expect(pickImportFile()).resolves.toBeNull();
    await expect(saveExportFile(new Blob(['x']), 'report.csv')).resolves.toBe('unsupported');
  });
});

describe('platform file dialogs (Tauri)', () => {
  it('rebuilds the picked import file from the native payload', async () => {
    const bytes = new TextEncoder().encode('sku,name\nA-1,Kopi\n');
    const invoke = vi.fn<Invoke>().mockResolvedValue({
      name: 'produk.csv',
      size: bytes.length,
      data: Buffer.from(bytes).toString('base64'),
    });
    const { pickImportFile } = await loadPlatform(invoke);

    const file = await pickImportFile();
    expect(invoke).toHaveBeenCalledWith('pick_import_file');
    expect(file?.name).toBe('produk.csv');
    expect(await file!.text()).toBe('sku,name\nA-1,Kopi\n');
  });

  it('returns null when the import dialog is cancelled', async () => {
    const invoke = vi.fn<Invoke>().mockResolvedValue(null);
    const { pickImportFile } = await loadPlatform(invoke);

    await expect(pickImportFile()).resolves.toBeNull();
  });

  it('sends the export bytes to the native save dialog', async () => {
    const invoke = vi.fn<Invoke>().mockResolvedValue('/Users/kasir/Downloads/report.pdf');
    const { saveExportFile } = await loadPlatform(invoke);

    const blob = new Blob(['PDFDATA']);
    await expect(saveExportFile(blob, 'report.pdf')).resolves.toBe('saved');
    expect(invoke).toHaveBeenCalledWith('save_export_file', {
      suggestedName: 'report.pdf',
      contents: Buffer.from('PDFDATA').toString('base64'),
    });
  });

  it('distinguishes a cancelled save from an unavailable dialog', async () => {
    const cancelled = vi.fn<Invoke>().mockResolvedValue(null);
    await expect((await loadPlatform(cancelled)).saveExportFile(new Blob(['x']), 'r.pdf')).resolves.toBe(
      'cancelled'
    );

    const failing = vi.fn<Invoke>().mockRejectedValue(new Error('no dialog'));
    await expect((await loadPlatform(failing)).saveExportFile(new Blob(['x']), 'r.pdf')).resolves.toBe(
      'unsupported'
    );
  });
});
