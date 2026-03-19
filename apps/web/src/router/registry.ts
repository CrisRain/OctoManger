import type { RouteRecordRaw } from "vue-router";

export type NavGroup = "primary";
export type SearchType = "page" | "action";

export type IconKey =
  | "dashboard"
  | "layers"
  | "link"
  | "email"
  | "schedule"
  | "robot"
  | "apps"
  | "file"
  | "settings"
  | "thunderbolt";

export interface RouteShortcut {
  key: string;
  label: string;
  description: string;
}

export interface AppRouteMeta {
  label?: string;
  navGroup?: NavGroup;
  order?: number;
  iconKey?: IconKey;
  navParent?: RouteName;
  searchKeywords?: string[];
  searchType?: SearchType;
  shortcuts?: RouteShortcut[];
}

export type AppRouteRecord = RouteRecordRaw & {
  meta?: AppRouteMeta;
};

export const routeNames = {
  oauthCallback: "oauth.callback",
  dashboard: "dashboard",

  agentsList: "agents.list",
  agentCreate: "agents.create",
  agentDetail: "agents.detail",
  agentEdit: "agents.edit",

  jobsList: "jobs.list",
  jobCreate: "jobs.create",
  jobDetail: "jobs.detail",
  jobEdit: "jobs.edit",
  jobExecutionsList: "jobs.executions",
  jobExecutionDetail: "jobs.execution-detail",

  accountTypesList: "account-types.list",
  accountTypeCreate: "account-types.create",
  accountTypeEdit: "account-types.edit",

  accountsList: "accounts.list",
  accountsByType: "accounts.by-type",
  accountCreate: "accounts.create",
  accountDetail: "accounts.detail",
  accountEdit: "accounts.edit",

  triggersList: "triggers.list",
  triggerCreate: "triggers.create",
  triggerEdit: "triggers.edit",

  emailAccountsList: "email-accounts.list",
  emailAccountCreate: "email-accounts.create",
  emailAccountEdit: "email-accounts.edit",
  emailAccountPreview: "email-accounts.preview",

  pluginsList: "plugins.list",
  pluginDetail: "plugins.detail",

  settingsRoot: "settings.root",
  logsRoot: "logs.root",
} as const;

export type RouteName = (typeof routeNames)[keyof typeof routeNames];

const PATHS = {
  oauthCallback: "/oauth/callback",
  dashboard: "/dashboard",

  agents: {
    list: "/agents",
    create: "/agents/create",
    detail: "/agents/:id",
    edit: "/agents/:id/edit",
  },
  jobs: {
    list: "/jobs",
    create: "/jobs/create",
    executions: "/jobs/executions",
    executionDetail: "/jobs/executions/:id",
    detail: "/jobs/:id",
    edit: "/jobs/:id/edit",
  },
  accountTypes: {
    list: "/account-types",
    create: "/account-types/create",
    edit: "/account-types/:id/edit",
  },
  accounts: {
    list: "/accounts",
    byType: "/accounts/type/:typeKey",
    create: "/accounts/create",
    detail: "/accounts/:id",
    edit: "/accounts/:id/edit",
  },
  triggers: {
    list: "/triggers",
    create: "/triggers/create",
    edit: "/triggers/:id/edit",
  },
  emailAccounts: {
    list: "/email-accounts",
    create: "/email-accounts/create",
    edit: "/email-accounts/:id/edit",
    preview: "/email-accounts/:id/preview",
  },
  plugins: {
    list: "/plugins",
    detail: "/plugins/:id",
  },
  settings: {
    root: "/settings",
  },
  logs: {
    root: "/logs",
  },
} as const;

const childPath = (path: string) => path.replace(/^\//, "");
const encodeParam = (value: string | number) => encodeURIComponent(String(value));

export const to = {
  oauthCallback: () => PATHS.oauthCallback,
  dashboard: () => PATHS.dashboard,

  agents: {
    list: () => PATHS.agents.list,
    create: () => PATHS.agents.create,
    detail: (id: string | number) => `/agents/${encodeParam(id)}`,
    edit: (id: string | number) => `/agents/${encodeParam(id)}/edit`,
  },
  jobs: {
    list: () => PATHS.jobs.list,
    create: () => PATHS.jobs.create,
    executions: () => PATHS.jobs.executions,
    executionDetail: (id: string | number) => `/jobs/executions/${encodeParam(id)}`,
    detail: (id: string | number) => `/jobs/${encodeParam(id)}`,
    edit: (id: string | number) => `/jobs/${encodeParam(id)}/edit`,
  },
  accountTypes: {
    list: () => PATHS.accountTypes.list,
    create: () => PATHS.accountTypes.create,
    edit: (id: string | number) => `/account-types/${encodeParam(id)}/edit`,
  },
  accounts: {
    list: () => PATHS.accounts.list,
    byType: (typeKey: string) => `/accounts/type/${encodeParam(typeKey)}`,
    create: () => PATHS.accounts.create,
    detail: (id: string | number) => `/accounts/${encodeParam(id)}`,
    edit: (id: string | number) => `/accounts/${encodeParam(id)}/edit`,
  },
  triggers: {
    list: () => PATHS.triggers.list,
    create: () => PATHS.triggers.create,
    edit: (id: string | number) => `/triggers/${encodeParam(id)}/edit`,
  },
  emailAccounts: {
    list: () => PATHS.emailAccounts.list,
    create: () => PATHS.emailAccounts.create,
    edit: (id: string | number) => `/email-accounts/${encodeParam(id)}/edit`,
    preview: (id: string | number) => `/email-accounts/${encodeParam(id)}/preview`,
  },
  plugins: {
    list: () => PATHS.plugins.list,
    detail: (id: string | number) => `/plugins/${encodeParam(id)}`,
  },
  settings: {
    root: () => PATHS.settings.root,
  },
  logs: {
    root: () => PATHS.logs.root,
  },
} as const;

const AppShell = () => import("@/components/AppShell.vue");

const DashboardPage = () => import("@/pages/DashboardPage.vue");

const AgentsListPage = () => import("@/pages/AgentsListPage.vue");
const AgentCreatePage = () => import("@/pages/AgentCreatePage.vue");
const AgentDetailPage = () => import("@/pages/AgentDetailPage.vue");
const AgentEditPage = () => import("@/pages/AgentEditPage.vue");

const JobsListPage = () => import("@/pages/JobsListPage.vue");
const JobCreatePage = () => import("@/pages/JobCreatePage.vue");
const JobDetailPage = () => import("@/pages/JobDetailPage.vue");
const JobEditPage = () => import("@/pages/JobEditPage.vue");
const JobExecutionsListPage = () => import("@/pages/JobExecutionsListPage.vue");
const JobExecutionDetailPage = () => import("@/pages/JobExecutionDetailPage.vue");

const AccountTypesListPage = () => import("@/pages/AccountTypesListPage.vue");
const AccountTypeCreatePage = () => import("@/pages/AccountTypeCreatePage.vue");
const AccountTypeEditPage = () => import("@/pages/AccountTypeEditPage.vue");

const AccountsListPage = () => import("@/pages/AccountsListPage.vue");
const AccountCreatePage = () => import("@/pages/AccountCreatePage.vue");
const AccountDetailPage = () => import("@/pages/AccountDetailPage.vue");
const AccountEditPage = () => import("@/pages/AccountEditPage.vue");

const TriggersListPage = () => import("@/pages/TriggersListPage.vue");
const TriggerCreatePage = () => import("@/pages/TriggerCreatePage.vue");
const TriggerEditPage = () => import("@/pages/TriggerEditPage.vue");

const EmailAccountsListPage = () => import("@/pages/EmailAccountsListPage.vue");
const EmailAccountCreatePage = () => import("@/pages/EmailAccountCreatePage.vue");
const EmailAccountEditPage = () => import("@/pages/EmailAccountEditPage.vue");
const EmailAccountPreviewPage = () => import("@/pages/EmailAccountPreviewPage.vue");

const PluginsListPage = () => import("@/pages/PluginsListPage.vue");
const PluginDetailPage = () => import("@/pages/PluginDetailPage.vue");

const SettingsPage = () => import("@/pages/SettingsPage.vue");
const LogsPage = () => import("@/pages/LogsPage.vue");

const OAuthCallbackPage = () => import("@/pages/OAuthCallbackPage.vue");

const appShellChildren: AppRouteRecord[] = [
  { path: "", redirect: PATHS.dashboard },
  {
    path: childPath(PATHS.dashboard),
    name: routeNames.dashboard,
    component: DashboardPage,
    meta: {
      label: "控制台",
      navGroup: "primary",
      order: 1,
      iconKey: "dashboard",
      searchType: "page",
      searchKeywords: ["dashboard", "控制台", "首页"],
      shortcuts: [
        { key: "G then D", label: "前往控制台", description: "跳转到Dashboard" },
      ],
    },
  },

  // Agents
  {
    path: childPath(PATHS.agents.list),
    name: routeNames.agentsList,
    component: AgentsListPage,
    meta: {
      label: "后台任务",
      navGroup: "primary",
      order: 6,
      iconKey: "robot",
      searchType: "page",
      searchKeywords: ["agent", "后台", "任务"],
    },
  },
  {
    path: childPath(PATHS.agents.create),
    name: routeNames.agentCreate,
    component: AgentCreatePage,
    meta: {
      label: "新建后台任务",
      searchType: "action",
      searchKeywords: ["agent", "创建", "新建", "后台"],
    },
  },
  {
    path: childPath(PATHS.agents.detail),
    name: routeNames.agentDetail,
    component: AgentDetailPage,
  },
  {
    path: childPath(PATHS.agents.edit),
    name: routeNames.agentEdit,
    component: AgentEditPage,
  },

  // Jobs
  {
    path: childPath(PATHS.jobs.list),
    name: routeNames.jobsList,
    component: JobsListPage,
    meta: {
      label: "定时任务",
      navGroup: "primary",
      order: 5,
      iconKey: "schedule",
      searchType: "page",
      searchKeywords: ["job", "任务", "定时"],
      shortcuts: [
        { key: "G then J", label: "前往任务管理", description: "跳转到任务列表" },
      ],
    },
  },
  {
    path: childPath(PATHS.jobs.create),
    name: routeNames.jobCreate,
    component: JobCreatePage,
    meta: {
      label: "新建任务",
      searchType: "action",
      searchKeywords: ["job", "任务", "新建", "创建"],
    },
  },
  {
    path: childPath(PATHS.jobs.executions),
    name: routeNames.jobExecutionsList,
    component: JobExecutionsListPage,
    meta: {
      label: "执行记录",
      navGroup: "primary",
      navParent: routeNames.jobsList,
      order: 1,
      searchType: "page",
      searchKeywords: ["execution", "执行", "记录"],
    },
  },
  {
    path: childPath(PATHS.jobs.executionDetail),
    name: routeNames.jobExecutionDetail,
    component: JobExecutionDetailPage,
  },
  {
    path: childPath(PATHS.jobs.detail),
    name: routeNames.jobDetail,
    component: JobDetailPage,
  },
  {
    path: childPath(PATHS.jobs.edit),
    name: routeNames.jobEdit,
    component: JobEditPage,
  },

  // Account Types
  {
    path: childPath(PATHS.accountTypes.list),
    name: routeNames.accountTypesList,
    component: AccountTypesListPage,
    meta: {
      label: "账号类型",
      navGroup: "primary",
      order: 2,
      iconKey: "layers",
      searchType: "page",
      searchKeywords: ["type", "类型", "账号类型"],
    },
  },
  {
    path: childPath(PATHS.accountTypes.create),
    name: routeNames.accountTypeCreate,
    component: AccountTypeCreatePage,
  },
  {
    path: childPath(PATHS.accountTypes.edit),
    name: routeNames.accountTypeEdit,
    component: AccountTypeEditPage,
  },

  // Accounts
  {
    path: childPath(PATHS.accounts.list),
    name: routeNames.accountsList,
    component: AccountsListPage,
    meta: {
      label: "账号管理",
      navGroup: "primary",
      order: 3,
      iconKey: "link",
      searchType: "page",
      searchKeywords: ["account", "账号", "账户"],
      shortcuts: [
        { key: "G then A", label: "前往账号管理", description: "跳转到账号列表" },
      ],
    },
  },
  {
    path: childPath(PATHS.accounts.byType),
    name: routeNames.accountsByType,
    component: AccountsListPage,
  },
  {
    path: childPath(PATHS.accounts.create),
    name: routeNames.accountCreate,
    component: AccountCreatePage,
    meta: {
      label: "新建账号",
      searchType: "action",
      searchKeywords: ["account", "账号", "新建", "创建"],
    },
  },
  {
    path: childPath(PATHS.accounts.detail),
    name: routeNames.accountDetail,
    component: AccountDetailPage,
  },
  {
    path: childPath(PATHS.accounts.edit),
    name: routeNames.accountEdit,
    component: AccountEditPage,
  },

  // Triggers
  {
    path: childPath(PATHS.triggers.list),
    name: routeNames.triggersList,
    component: TriggersListPage,
    meta: {
      label: "触发器",
      navGroup: "primary",
      order: 10,
      iconKey: "thunderbolt",
      searchType: "page",
      searchKeywords: ["trigger", "触发器", "webhook"],
    },
  },
  {
    path: childPath(PATHS.triggers.create),
    name: routeNames.triggerCreate,
    component: TriggerCreatePage,
  },
  {
    path: childPath(PATHS.triggers.edit),
    name: routeNames.triggerEdit,
    component: TriggerEditPage,
  },

  // Email Accounts
  {
    path: childPath(PATHS.emailAccounts.list),
    name: routeNames.emailAccountsList,
    component: EmailAccountsListPage,
    meta: {
      label: "邮箱账号",
      navGroup: "primary",
      order: 4,
      iconKey: "email",
      searchType: "page",
      searchKeywords: ["email", "邮箱", "邮件"],
    },
  },
  {
    path: childPath(PATHS.emailAccounts.create),
    name: routeNames.emailAccountCreate,
    component: EmailAccountCreatePage,
    meta: {
      label: "新建邮箱",
      searchType: "action",
      searchKeywords: ["email", "邮箱", "新建", "创建"],
    },
  },
  {
    path: childPath(PATHS.emailAccounts.edit),
    name: routeNames.emailAccountEdit,
    component: EmailAccountEditPage,
  },
  {
    path: childPath(PATHS.emailAccounts.preview),
    name: routeNames.emailAccountPreview,
    component: EmailAccountPreviewPage,
  },

  // Plugins
  {
    path: childPath(PATHS.plugins.list),
    name: routeNames.pluginsList,
    component: PluginsListPage,
    meta: {
      label: "插件管理",
      navGroup: "primary",
      order: 7,
      iconKey: "apps",
      searchType: "page",
      searchKeywords: ["plugin", "插件"],
    },
  },
  {
    path: childPath(PATHS.plugins.detail),
    name: routeNames.pluginDetail,
    component: PluginDetailPage,
  },

  // Settings and misc
  {
    path: childPath(PATHS.settings.root),
    name: routeNames.settingsRoot,
    component: SettingsPage,
    meta: {
      label: "设置",
      navGroup: "primary",
      order: 9,
      iconKey: "settings",
      searchType: "page",
      searchKeywords: ["settings", "设置", "系统"],
    },
  },
  {
    path: childPath(PATHS.logs.root),
    name: routeNames.logsRoot,
    component: LogsPage,
    meta: {
      label: "系统日志",
      navGroup: "primary",
      order: 8,
      iconKey: "file",
      searchType: "page",
      searchKeywords: ["logs", "日志", "系统日志"],
    },
  },
];

export const routes: AppRouteRecord[] = [
  {
    path: PATHS.oauthCallback,
    name: routeNames.oauthCallback,
    component: OAuthCallbackPage,
  },
  {
    path: "/",
    component: AppShell,
    children: appShellChildren,
  },
];

interface FlatRoute {
  name?: RouteName;
  path: string;
  meta?: AppRouteMeta;
}

const joinPaths = (parentPath: string, childPathValue: string) => {
  if (!parentPath || parentPath === "/") {
    return `/${childPathValue}`.replace(/\/+/g, "/");
  }
  return `${parentPath.replace(/\/$/, "")}/${childPathValue}`.replace(/\/+/g, "/");
};

const flattenRoutes = (items: AppRouteRecord[], parentPath = ""): FlatRoute[] => {
  const result: FlatRoute[] = [];
  for (const route of items) {
    const normalizedPath = route.path?.startsWith("/")
      ? route.path
      : joinPaths(parentPath, route.path ?? "");
    if (route.name || route.meta) {
      result.push({
        name: route.name as RouteName | undefined,
        path: normalizedPath,
        meta: route.meta,
      });
    }
    if (route.children?.length) {
      result.push(...flattenRoutes(route.children as AppRouteRecord[], normalizedPath));
    }
  }
  return result;
};

export const flatRoutes = flattenRoutes(routes);

export interface NavRoute {
  name: RouteName;
  path: string;
  label: string;
  navGroup: NavGroup;
  order: number;
  iconKey?: IconKey;
  navParent?: RouteName;
}

export const navRoutes: NavRoute[] = flatRoutes
  .filter((route): route is FlatRoute & { name: RouteName; meta: AppRouteMeta } =>
    Boolean(route.name && route.meta?.navGroup)
  )
  .map((route) => ({
    name: route.name,
    path: route.path,
    label: route.meta?.label ?? route.name,
    navGroup: route.meta?.navGroup ?? "primary",
    order: route.meta?.order ?? 0,
    iconKey: route.meta?.iconKey,
    navParent: route.meta?.navParent,
  }))
  .sort((a, b) => a.order - b.order);

export interface SearchRoute {
  name: RouteName;
  path: string;
  label: string;
  type: SearchType;
  keywords: string[];
}

export const searchRoutes: SearchRoute[] = flatRoutes
  .filter((route): route is FlatRoute & { name: RouteName; meta: AppRouteMeta } =>
    Boolean(route.name && route.meta?.searchKeywords?.length)
  )
  .map((route) => ({
    name: route.name,
    path: route.path,
    label: route.meta?.label ?? route.name,
    type: route.meta?.searchType ?? "page",
    keywords: route.meta?.searchKeywords ?? [],
  }));

export interface ShortcutRoute extends RouteShortcut {
  name: RouteName;
  path: string;
}

export const shortcutRoutes: ShortcutRoute[] = flatRoutes
  .flatMap((route) =>
    (route.meta?.shortcuts ?? []).map((shortcut) => ({
      ...shortcut,
      name: route.name as RouteName,
      path: route.path,
    }))
  );
