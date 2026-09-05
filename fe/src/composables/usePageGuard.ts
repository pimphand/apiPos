import { onMounted, ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { navigateTo } from '@/lib/navigation';
import { can, type Permission } from '@/lib/rbac';

// Session key recording the last successfully visited authenticated page. The
// /403 page's "Back" button uses it to return the user to where they came from.
const LAST_URL_KEY = 'pos_last_url';

/**
 * Page-level guard for Vue islands on protected routes.
 *
 * This is UX/defense-in-depth ONLY — the Astro static pages are public HTML and
 * the Go gateway is the real authorization boundary.
 *
 * Behavior:
 * - Not authenticated  → redirect to /login (never the 403 page).
 * - Authenticated but missing `permission` → redirect to /403. The 403 page is
 *   only ever reached by logged-in users without the required permission.
 * - Authenticated with permission → `allowed = true` and the current URL is
 *   recorded as the "last URL" so the 403 Back button can return here.
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
      if (can(authStore.currentUser, permission)) {
        allowed.value = true;
        recordLastUrl();
      } else {
        // Logged in but not permitted → permission denied page.
        allowed.value = false;
        navigateTo('/403');
        return;
      }
    } else {
      recordLastUrl();
    }

    isReady.value = true;
  });

  return { isReady, allowed, authStore };
}

/** Record the current page as the last visited URL (for the 403 Back button). */
function recordLastUrl() {
  if (typeof window === 'undefined') return;
  try {
    const path = window.location.pathname;
    if (path && path !== '/403' && path !== '/login' && path !== '/register') {
      window.sessionStorage.setItem(LAST_URL_KEY, path);
    }
  } catch {
    /* sessionStorage unavailable — Back falls back to /dashboard */
  }
}