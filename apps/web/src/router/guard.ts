import { PATHS } from "./route-definitions";
import { getSetupStatus } from "@/api";

export type GuardDecision = true | { path: string; query?: { redirect: string } };

const publicPaths = new Set<string>([PATHS.oauthCallback, PATHS.auth, PATHS.setup]);
const setupStatusCacheTTL = 3000;

let setupStatusCache: { needsSetup: boolean; at: number } | null = null;
let setupStatusPending: Promise<boolean | null> | null = null;

function redirectTarget(fullPath: string): string {
  if (!fullPath.startsWith("/")) {
    return PATHS.dashboard;
  }
  if (fullPath === PATHS.auth || fullPath === PATHS.setup) {
    return PATHS.dashboard;
  }
  return fullPath;
}

export function invalidateSetupStatusCache(): void {
  setupStatusCache = null;
  setupStatusPending = null;
}

async function loadNeedsSetup(): Promise<boolean | null> {
  const now = Date.now();
  if (setupStatusCache && now - setupStatusCache.at <= setupStatusCacheTTL) {
    return setupStatusCache.needsSetup;
  }
  if (setupStatusPending) {
    return setupStatusPending;
  }

  setupStatusPending = (async () => {
    try {
      const status = await getSetupStatus();
      setupStatusCache = { needsSetup: Boolean(status.needs_setup), at: Date.now() };
      return setupStatusCache.needsSetup;
    } catch {
      return null;
    } finally {
      setupStatusPending = null;
    }
  })();

  return setupStatusPending;
}

export async function evaluateNavigationGuard(toPath: string, fullPath: string, adminKey: string): Promise<GuardDecision> {
  const redirect = redirectTarget(fullPath);
  const needsSetup = await loadNeedsSetup();

  if (needsSetup === true) {
    if (toPath === PATHS.setup) {
      return true;
    }
    return { path: PATHS.setup, query: { redirect } };
  }

  if (toPath === PATHS.setup && needsSetup === false) {
    if (adminKey.trim()) {
      return { path: redirect };
    }
    return { path: PATHS.auth, query: { redirect } };
  }

  if (toPath === PATHS.auth) {
    if (adminKey.trim()) {
      return { path: redirect };
    }
    return true;
  }

  if (publicPaths.has(toPath)) {
    return true;
  }

  if (adminKey.trim()) {
    return true;
  }
  return { path: PATHS.auth, query: { redirect } };
}
