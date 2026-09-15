"use client";

// The real session tokens live in httpOnly cookies set by link-admin and are
// never readable from JavaScript. `auth_state` is a separate, non-sensitive
// marker cookie (no token material) the backend sets alongside them purely
// so the frontend can tell whether a session is active.
const AUTH_STATE_COOKIE = "auth_state";

export function isAuthenticated(): boolean {
  if (typeof document === "undefined") return false;
  return document.cookie
    .split("; ")
    .some((c) => c.startsWith(`${AUTH_STATE_COOKIE}=`));
}

// Clears the client-readable marker immediately for UI purposes. The
// httpOnly cookies themselves can only be cleared by the server — call the
// /auth/logout endpoint to actually end the session.
export function clearAuthState(): void {
  if (typeof document === "undefined") return;
  document.cookie = `${AUTH_STATE_COOKIE}=; path=/; max-age=0`;
}
