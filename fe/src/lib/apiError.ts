import type { ApiError } from '@/types';

/**
 * Extract a user-friendly message from a rejected API call.
 * The http interceptor normalizes failures into ApiError; fall back gracefully
 * for non-ApiError rejections (network errors, unexpected shapes).
 */
export function apiMessage(err: unknown, fallback = 'Something went wrong'): string {
  if (err && typeof err === 'object') {
    const e = err as Partial<ApiError>;
    if (typeof e.message === 'string' && e.message.trim()) {
      return e.message;
    }
  }
  if (err instanceof Error && err.message) {
    return err.message;
  }
  return fallback;
}
