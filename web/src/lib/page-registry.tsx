import { lazy, type ComponentType, type LazyExoticComponent } from "react";

export type PreloadablePage<T extends ComponentType<any> = ComponentType<any>> =
  LazyExoticComponent<T> & {
    preload: () => Promise<void>;
  };

function createPreloadablePage<TModule, TComponent extends ComponentType<any>>(
  loader: () => Promise<TModule>,
  pick: (module: TModule) => TComponent,
): PreloadablePage<TComponent> {
  let pending: Promise<TModule> | undefined;

  const loadModule = () => {
    if (!pending) {
      pending = loader().catch((error) => {
        pending = undefined;
        throw error;
      });
    }
    return pending;
  };

  const Page = lazy(async () => ({
    default: pick(await loadModule()),
  }));

  return Object.assign(Page, {
    preload: async () => {
      await loadModule();
    },
  });
}

const loadDashboardPage = () => import("@/pages/dashboard-page");
const loadAccountTypesPage = () => import("@/pages/account-types-page");
const loadAccountsPage = () => import("@/pages/accounts-page");
const loadEmailAccountsOutlookPage = () => import("@/pages/email-accounts-outlook-page");
const loadJobsPage = () => import("@/pages/jobs-page");
const loadLogsPage = () => import("@/pages/logs-page");
const loadOctoModulesPage = () => import("@/pages/octo-modules-page");
const loadApiKeysPage = () => import("@/pages/api-keys-page");
const loadTriggersPage = () => import("@/pages/triggers-page");
const loadSettingsPage = () => import("@/pages/settings-page");
const loadSSLCertificatePage = () => import("@/pages/ssl-certificate-page");
const loadOAuthCallbackPage = () => import("@/pages/oauth-callback-page");
const loadSetupPage = () => import("@/pages/setup-page");
const loadAuthPage = () => import("@/pages/auth-page");

export const DashboardPage = createPreloadablePage(loadDashboardPage, (module) => module.DashboardPage);
export const AccountTypesPage = createPreloadablePage(loadAccountTypesPage, (module) => module.AccountTypesPage);
export const AccountsPage = createPreloadablePage(loadAccountsPage, (module) => module.AccountsPage);
export const EmailAccountsOutlookPage = createPreloadablePage(loadEmailAccountsOutlookPage, (module) => module.EmailAccountsOutlookPage);
export const JobsPage = createPreloadablePage(loadJobsPage, (module) => module.JobsPage);
export const LogsPage = createPreloadablePage(loadLogsPage, (module) => module.LogsPage);
export const OctoModulesPage = createPreloadablePage(loadOctoModulesPage, (module) => module.OctoModulesPage);
export const ApiKeysPage = createPreloadablePage(loadApiKeysPage, (module) => module.ApiKeysPage);
export const TriggersPage = createPreloadablePage(loadTriggersPage, (module) => module.TriggersPage);
export const SettingsPage = createPreloadablePage(loadSettingsPage, (module) => module.SettingsPage);
export const SSLCertificatePage = createPreloadablePage(loadSSLCertificatePage, (module) => module.SSLCertificatePage);
export const OAuthCallbackPage = createPreloadablePage(loadOAuthCallbackPage, (module) => module.OAuthCallbackPage);
export const SetupPage = createPreloadablePage(loadSetupPage, (module) => module.SetupPage);
export const AuthPage = createPreloadablePage(loadAuthPage, (module) => module.AuthPage);

export const routePreloads = {
  dashboard: DashboardPage.preload,
  accountTypes: AccountTypesPage.preload,
  accounts: AccountsPage.preload,
  emailAccounts: EmailAccountsOutlookPage.preload,
  jobs: JobsPage.preload,
  logs: LogsPage.preload,
  modules: OctoModulesPage.preload,
  apiKeys: ApiKeysPage.preload,
  triggers: TriggersPage.preload,
  settings: SettingsPage.preload,
  ssl: SSLCertificatePage.preload,
  oauthCallback: OAuthCallbackPage.preload,
  setup: SetupPage.preload,
  auth: AuthPage.preload,
} as const;

export const coreProtectedRoutePreloads = [
  routePreloads.dashboard,
  routePreloads.accounts,
  routePreloads.jobs,
  routePreloads.logs,
  routePreloads.triggers,
];
