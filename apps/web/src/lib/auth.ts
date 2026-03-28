const ADMIN_KEY_STORAGE_KEY = "octomanger.admin_key";
let inMemoryAdminKey = "";

function readStoredAdminKey(): string {
  if (typeof window === "undefined") {
    return "";
  }
  try {
    return String(window.sessionStorage.getItem(ADMIN_KEY_STORAGE_KEY) ?? "").trim();
  } catch {
    return "";
  }
}

export function getAdminKey(): string {
  if (!inMemoryAdminKey) {
    inMemoryAdminKey = readStoredAdminKey();
  }
  return inMemoryAdminKey.trim();
}

export function setAdminKey(key: string): void {
  const normalized = key.trim();
  inMemoryAdminKey = normalized;

  if (typeof window === "undefined") {
    return;
  }

  try {
    if (!normalized) {
      window.sessionStorage.removeItem(ADMIN_KEY_STORAGE_KEY);
      return;
    }
    window.sessionStorage.setItem(ADMIN_KEY_STORAGE_KEY, normalized);
  } catch {
    // ignore storage failures in private/blocked contexts
  }
}
