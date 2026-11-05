import { navigate } from 'astro:transitions/client';

export function navigateTo(path: string) {
  if (typeof window !== 'undefined') {
    void navigate(path);
  }
}
