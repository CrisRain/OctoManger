const KEY = "octo_admin_key";

export function getAdminKey(): string {
  return localStorage.getItem(KEY)?.trim() ?? "";
}

export function setAdminKey(key: string): void {
  localStorage.setItem(KEY, key);
}
