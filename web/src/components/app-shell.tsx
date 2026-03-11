import {
  Activity,
  Database,
  KeyRound,
  Layers,
  Link2,
  Mail,
  Menu,
  ScrollText,
  Settings,
  ShieldCheck,
  Shapes,
  Workflow,
  type LucideIcon,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { NavLink, Outlet, useLocation } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";
import { api } from "@/lib/api";
import { coreProtectedRoutePreloads, routePreloads } from "@/lib/page-registry";
import { cn } from "@/lib/utils";

type NavChild = {
  to: string;
  label: string;
  preload?: () => Promise<void>;
};

type NavItem = {
  to: string;
  label: string;
  icon: LucideIcon;
  preload?: () => Promise<void>;
  children?: NavChild[];
};

function warmRoute(preload?: () => Promise<void>) {
  if (preload) {
    void preload();
  }
}

function canWarmRoutes() {
  if (typeof navigator === "undefined") {
    return false;
  }

  const connection = (navigator as Navigator & { connection?: { saveData?: boolean } }).connection;
  return !connection?.saveData;
}

function NavigationList({
  items,
  pathname,
  onNavigate,
}: {
  items: NavItem[];
  pathname: string;
  onNavigate?: () => void;
}) {
  return (
    <nav className="grid gap-1 px-4 text-sm font-medium">
      {items.map((item) => {
        const Icon = item.icon;
        const hasChildren = Boolean(item.children?.length);
        const isAncestor = hasChildren && pathname.startsWith(`${item.to}/`);

        return (
          <div key={item.to} className="grid gap-1">
            <NavLink
              to={item.to}
              end
              onClick={onNavigate}
              onMouseEnter={() => warmRoute(item.preload)}
              onFocus={() => warmRoute(item.preload)}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-3 rounded-xl px-3 py-2 text-sm transition-all",
                  isActive
                    ? "bg-foreground font-medium text-background shadow-sm"
                    : isAncestor
                      ? "font-medium text-foreground hover:bg-muted"
                      : "text-muted-foreground hover:bg-muted hover:text-foreground",
                )
              }
            >
              <Icon className="h-4 w-4" />
              {item.label}
            </NavLink>

            {hasChildren ? (
              <div className="ml-7 grid gap-1">
                {item.children!.map((child) => (
                  <NavLink
                    key={child.to}
                    to={child.to}
                    end
                    onClick={onNavigate}
                    onMouseEnter={() => warmRoute(child.preload)}
                    onFocus={() => warmRoute(child.preload)}
                    className={({ isActive }) =>
                      cn(
                        "flex items-center gap-2 rounded-lg px-3 py-1.5 text-xs transition-all",
                        isActive
                          ? "bg-muted font-medium text-foreground"
                          : "text-muted-foreground hover:bg-muted/60 hover:text-foreground",
                      )
                    }
                  >
                    <span className="h-1.5 w-1.5 rounded-full bg-current opacity-60" />
                    {child.label}
                  </NavLink>
                ))}
              </div>
            ) : null}
          </div>
        );
      })}
    </nav>
  );
}

export function AppShell() {
  const location = useLocation();
  const [open, setOpen] = useState(false);
  const { data: accountTypes = [] } = useQuery({
    queryKey: ["account-types"],
    queryFn: api.listAccountTypes,
    staleTime: 30 * 60 * 1000,
  });

  const navItems = useMemo<NavItem[]>(() => {
    const genericTypes = accountTypes.filter((item) => item.category === "generic");
    const accountChildren = [
      { to: "/accounts", label: "全部账号", preload: routePreloads.accounts },
      ...genericTypes.map((item) => ({
        to: `/accounts/${item.key}`,
        label: item.name,
        preload: routePreloads.accounts,
      })),
    ];
    const emailChildren = [
      {
        to: "/email-accounts/outlook",
        label: "Outlook 邮箱管理",
        preload: routePreloads.emailAccounts,
      },
    ];

    return [
      { to: "/dashboard", label: "控制台", icon: Activity, preload: routePreloads.dashboard },
      { to: "/account-types", label: "账号类型", icon: Shapes, preload: routePreloads.accountTypes },
      {
        to: "/accounts",
        label: "账号管理",
        icon: Database,
        preload: routePreloads.accounts,
        children: accountChildren,
      },
      {
        to: "/email-accounts",
        label: "邮箱账号",
        icon: Mail,
        preload: routePreloads.emailAccounts,
        children: emailChildren,
      },
      { to: "/jobs", label: "任务", icon: Workflow, preload: routePreloads.jobs },
      { to: "/logs", label: "日志", icon: ScrollText, preload: routePreloads.logs },
      { to: "/modules", label: "Octo 模块", icon: Layers, preload: routePreloads.modules },
      { to: "/api-keys", label: "API 密钥", icon: KeyRound, preload: routePreloads.apiKeys },
      { to: "/triggers", label: "触发器", icon: Link2, preload: routePreloads.triggers },
      { to: "/ssl", label: "SSL 证书", icon: ShieldCheck, preload: routePreloads.ssl },
      { to: "/settings", label: "设置", icon: Settings, preload: routePreloads.settings },
    ];
  }, [accountTypes]);

  const currentTitle = useMemo(() => {
    const path = location.pathname;
    const topMatch = navItems.find((item) => path === item.to || path.startsWith(`${item.to}/`));

    if (!topMatch) {
      return "控制台";
    }

    const childMatch = topMatch.children?.find((child) => path === child.to);
    return childMatch?.label ?? topMatch.label;
  }, [location.pathname, navItems]);

  useEffect(() => {
    if (!canWarmRoutes()) {
      return;
    }

    const activePath = location.pathname;
    const upcomingRoutes = coreProtectedRoutePreloads.filter((preload) => {
      if (activePath.startsWith("/dashboard")) {
        return preload !== routePreloads.dashboard;
      }
      if (activePath.startsWith("/accounts")) {
        return preload !== routePreloads.accounts;
      }
      if (activePath.startsWith("/jobs")) {
        return preload !== routePreloads.jobs;
      }
      if (activePath.startsWith("/logs")) {
        return preload !== routePreloads.logs;
      }
      if (activePath.startsWith("/triggers")) {
        return preload !== routePreloads.triggers;
      }
      return true;
    });

    const warmUpcomingRoutes = () => {
      upcomingRoutes.slice(0, 3).forEach((preload) => {
        void preload();
      });
    };

    if (typeof window.requestIdleCallback === "function") {
      const idleId = window.requestIdleCallback(warmUpcomingRoutes, { timeout: 1200 });
      return () => {
        window.cancelIdleCallback?.(idleId);
      };
    }

    const timeoutId = window.setTimeout(warmUpcomingRoutes, 400);
    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [location.pathname]);

  return (
    <div className="flex min-h-screen flex-col bg-[radial-gradient(circle_at_top,_rgba(255,255,255,0.9),_transparent_38%),linear-gradient(180deg,_rgba(255,255,255,0.75),_rgba(255,255,255,0))]">
      <header className="sticky top-0 z-50 flex h-14 items-center gap-4 border-b bg-card/95 px-4 backdrop-blur lg:hidden">
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger asChild>
            <Button variant="outline" size="icon" className="shrink-0 rounded-xl">
              <Menu className="h-5 w-5" />
              <span className="sr-only">切换导航菜单</span>
            </Button>
          </SheetTrigger>
          <SheetContent side="left" className="flex w-[280px] flex-col p-0 sm:w-[300px]">
            <div className="flex h-14 items-center border-b px-6 font-semibold">
              <span className="text-lg font-bold">OctoManager</span>
            </div>
            <ScrollArea className="flex-1 py-4">
              <NavigationList items={navItems} pathname={location.pathname} onNavigate={() => setOpen(false)} />
            </ScrollArea>
          </SheetContent>
        </Sheet>

        <div className="flex min-w-0 flex-1 items-center justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">Workspace</p>
            <p className="truncate text-sm font-semibold">{currentTitle}</p>
          </div>
          <div className="flex items-center gap-2 rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-1 text-[11px] font-medium text-emerald-700">
            <span className="h-2 w-2 rounded-full bg-emerald-500" />
            在线
          </div>
        </div>
      </header>

      <div className="flex flex-1">
        <aside className="sticky top-0 hidden h-screen w-[248px] flex-col border-r bg-card/90 backdrop-blur lg:flex">
          <div className="flex h-14 items-center border-b px-6">
            <span className="text-lg font-bold tracking-tight">OctoManager</span>
          </div>

          <ScrollArea className="flex-1 py-4">
            <NavigationList items={navItems} pathname={location.pathname} />
          </ScrollArea>

          <div className="border-t p-4">
            <div className="rounded-2xl border bg-muted/40 p-3">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-full bg-foreground">
                  <Activity className="h-4 w-4 text-background" />
                </div>
                <div>
                  <p className="text-xs font-medium">状态：在线</p>
                  <p className="text-[10px] text-muted-foreground">预加载导航已启用</p>
                </div>
              </div>
            </div>
          </div>
        </aside>

        <main className="flex-1">
          <div className="container mx-auto max-w-7xl space-y-4 p-4 lg:p-8">
            <div className="mb-4 flex items-center justify-between lg:hidden">
              <h1 className="text-lg font-semibold md:text-2xl">{currentTitle}</h1>
            </div>
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
