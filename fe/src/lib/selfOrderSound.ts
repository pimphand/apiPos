/**
 * Self-order sound notification (docs/pos-self-order.md §8.2).
 *
 * Uses the Web Audio API (no audio asset needed). Browsers require a user
 * gesture before AudioContext can produce sound, so the context is created /
 * resumed lazily on the first user interaction on the POS page.
 */

let audioCtx: AudioContext | null = null;
let unlocked = false;
const STORAGE_KEY = 'pos.self_order_sound.enabled.v1';

/** Whether sound notifications are enabled (persisted per browser). */
export function isSelfOrderSoundEnabled(): boolean {
  if (typeof window === 'undefined') return false;
  return localStorage.getItem(STORAGE_KEY) !== '0';
}

/** Toggle sound notifications; returns the new state. */
export function setSelfOrderSoundEnabled(enabled: boolean): boolean {
  if (typeof window === 'undefined') return false;
  localStorage.setItem(STORAGE_KEY, enabled ? '1' : '0');
  return enabled;
}

/** Call from a user-gesture handler (click/keydown) to unlock audio. */
export function unlockSelfOrderAudio(): void {
  if (typeof window === 'undefined') return;
  unlocked = true;
  const ctx = ensureCtx();
  if (ctx && ctx.state === 'suspended') {
    void ctx.resume();
  }
}

function ensureCtx(): AudioContext | null {
  if (typeof window === 'undefined') return null;
  if (!audioCtx) {
    const Ctor =
      window.AudioContext ??
      (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
    if (!Ctor) return null;
    audioCtx = new Ctor();
  }
  return audioCtx;
}

function beep(ctx: AudioContext, freq: number, start: number, duration: number): void {
  const osc = ctx.createOscillator();
  const gain = ctx.createGain();
  osc.type = 'sine';
  osc.frequency.value = freq;
  gain.gain.setValueAtTime(0.0001, ctx.currentTime + start);
  gain.gain.exponentialRampToValueAtTime(0.25, ctx.currentTime + start + 0.02);
  gain.gain.exponentialRampToValueAtTime(0.0001, ctx.currentTime + start + duration);
  osc.connect(gain);
  gain.connect(ctx.destination);
  osc.start(ctx.currentTime + start);
  osc.stop(ctx.currentTime + start + duration + 0.05);
}

/**
 * Play a short 3-beep alert (like a kitchen bell). No-op when disabled or
 * when audio is not unlocked yet.
 */
export function playSelfOrderSound(): void {
  if (!isSelfOrderSoundEnabled() || !unlocked) return;
  const ctx = ensureCtx();
  if (!ctx || ctx.state !== 'running') return;
  // Ascending triple beep: E5 → G5 → A5.
  beep(ctx, 659.25, 0, 0.18);
  beep(ctx, 783.99, 0.22, 0.18);
  beep(ctx, 880.0, 0.44, 0.3);
}