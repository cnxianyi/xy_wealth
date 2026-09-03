export const TOKEN_STORAGE_KEY = "xy-wealth.x-token";
export const TOKEN_EXPIRY_STORAGE_KEY = "xy-wealth.x-token-expires-at";
export const THEME_STORAGE_KEY = "xy-wealth-theme";

function storageAvailable(): boolean {
  return typeof window !== "undefined" && typeof window.sessionStorage !== "undefined";
}

export function getStoredToken(): string | null {
  if (!storageAvailable()) return null;
  try {
    return window.sessionStorage.getItem(TOKEN_STORAGE_KEY);
  } catch {
    return null;
  }
}

export function storeToken(token: string, expiresAt?: string): void {
  if (!storageAvailable()) return;
  try {
    window.sessionStorage.setItem(TOKEN_STORAGE_KEY, token);
    if (expiresAt) window.sessionStorage.setItem(TOKEN_EXPIRY_STORAGE_KEY, expiresAt);
  } catch {
    // Private browsing can deny storage access. The in-memory auth state still works.
  }
}

export function clearStoredToken(): void {
  if (!storageAvailable()) return;
  try {
    window.sessionStorage.removeItem(TOKEN_STORAGE_KEY);
    window.sessionStorage.removeItem(TOKEN_EXPIRY_STORAGE_KEY);
  } catch {
    // Ignore storage errors while handling an expired session.
  }
}

export function getStoredTheme(): "light" | "dark" {
  if (typeof window === "undefined" || typeof window.localStorage === "undefined") return "light";
  try {
    return window.localStorage.getItem(THEME_STORAGE_KEY) === "dark" ? "dark" : "light";
  } catch {
    return "light";
  }
}

export function storeTheme(theme: "light" | "dark"): void {
  if (typeof window === "undefined" || typeof window.localStorage === "undefined") return;
  try {
    window.localStorage.setItem(THEME_STORAGE_KEY, theme);
  } catch {
    // Theme preference is best effort.
  }
}
