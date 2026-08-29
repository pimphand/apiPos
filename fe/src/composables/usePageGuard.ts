import { onMounted, ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { navigateTo } from '@/lib/navigation';
import { can, type Permission } from '@/lib/rbac';

/**
 * Page-level guard for Vue islands on protected routes.
 *
 * This is UX/defense-in-depth ONLY — the Astro static pages are public HTML and
 * the Go gateway is the real authorization boundary. Use `permission` to gate
 * actions and show the "access restricted" state when missing.
 */
export function usePageGuard(permission?: Permission) {
  const authStore = useAuthStore();

  const isReady = ref(false);
  const allowed = ref(true);

  onMounted(async () => {
    // Try to restore a persisted session (reload case) before gating.
    await authStore.restoreSession();

    if (!authStore.isAuthenticated) {
      navigateTo('/login');
      return;
    }
    if (permission) {
      allowed.value = can(authStore.currentUser, permission);
    }
    isReady.value = true;
  });

  return { isReady, allowed, authStore };
}
