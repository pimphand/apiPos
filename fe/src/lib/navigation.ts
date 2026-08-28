import { navigate } from 'astro:transitions/client';

export function navigateTo(path: string) {
  if (typeof navigate === 'function') {
    navigate(path);
    return;
  }
  if (typeof window !== 'undefined') {
    window.location.href = path;
  }
}