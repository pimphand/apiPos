/**
 * Programmatic navigation.
 *
 * This is a plain MPA full-page navigation. Astro view transitions
 * (`<ClientRouter />`) are deliberately NOT used: Astro's transition machinery
 * pre-renders the target page in a hidden sub-frame, which boots the app again
 * and independently restores the session. That second context can consume
 * (rotate) the only valid refresh token and be torn down before persisting the
 * replacement — the next reload then refreshes with the now-revoked token and
 * the user is logged out. A single full-page navigation keeps exactly one
 * context in charge of session state.
 */
export function navigateTo(path: string) {
  if (typeof window !== 'undefined') {
    window.location.href = path;
  }
}
