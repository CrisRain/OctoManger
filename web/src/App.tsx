import { Suspense, type ReactNode } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import { AppShell } from "@/components/app-shell";
import { useAuthCheck } from "@/hooks/use-auth-check";
import {
  AccountsPage,
  AccountTypesPage,
  ApiKeysPage,
  AuthPage,
  DashboardPage,
  EmailAccountsOutlookPage,
  JobsPage,
  LogsPage,
  OAuthCallbackPage,
  OctoModulesPage,
  SettingsPage,
  SetupPage,
  SSLCertificatePage,
  TriggersPage,
} from "@/lib/page-registry";

function LoadingSpinner({ fullScreen = false, label = "页面加载中..." }: { fullScreen?: boolean; label?: string }) {
  return (
    <div
      className={
        fullScreen
          ? "flex min-h-screen items-center justify-center"
          : "flex min-h-[240px] items-center justify-center rounded-xl border border-dashed bg-card/60"
      }
    >
      <div className="text-center">
        <div className="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-muted-foreground/20 border-t-foreground" />
        {!fullScreen ? <p className="mt-3 text-sm text-muted-foreground">{label}</p> : null}
      </div>
    </div>
  );
}

function withRouteSuspense(element: ReactNode) {
  return <Suspense fallback={<LoadingSpinner />}>{element}</Suspense>;
}

function RequireAuth({ children }: { children: ReactNode }) {
  const location = useLocation();
  const authState = useAuthCheck();

  if (authState === "checking") {
    return <LoadingSpinner fullScreen />;
  }
  if (authState === "unauthenticated") {
    return <Navigate to="/auth" state={{ from: location.pathname }} replace />;
  }
  if (authState === "needs-setup") {
    return <Navigate to="/setup" replace />;
  }
  return <>{children}</>;
}

function RequireSetup({ children }: { children: ReactNode }) {
  const authState = useAuthCheck();

  if (authState === "checking") {
    return <LoadingSpinner fullScreen />;
  }
  if (authState === "needs-setup") {
    return <>{children}</>;
  }
  return <Navigate to={authState === "ok" ? "/dashboard" : "/auth"} replace />;
}

export function App() {
  return (
    <Routes>
      <Route path="/oauth/callback" element={withRouteSuspense(<OAuthCallbackPage />)} />
      <Route path="/setup" element={<RequireSetup>{withRouteSuspense(<SetupPage />)}</RequireSetup>} />
      <Route path="/auth" element={withRouteSuspense(<AuthPage />)} />
      <Route
        element={
          <RequireAuth>
            <AppShell />
          </RequireAuth>
        }
      >
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={withRouteSuspense(<DashboardPage />)} />
        <Route path="/account-types" element={withRouteSuspense(<AccountTypesPage />)} />
        <Route path="/accounts" element={withRouteSuspense(<AccountsPage />)} />
        <Route path="/accounts/:typeKey" element={withRouteSuspense(<AccountsPage />)} />
        <Route path="/email-accounts" element={<Navigate to="/email-accounts/outlook" replace />} />
        <Route path="/email-accounts/outlook" element={withRouteSuspense(<EmailAccountsOutlookPage />)} />
        <Route path="/jobs" element={withRouteSuspense(<JobsPage />)} />
        <Route path="/logs" element={withRouteSuspense(<LogsPage />)} />
        <Route path="/modules" element={withRouteSuspense(<OctoModulesPage />)} />
        <Route path="/api-keys" element={withRouteSuspense(<ApiKeysPage />)} />
        <Route path="/triggers" element={withRouteSuspense(<TriggersPage />)} />
        <Route path="/settings" element={withRouteSuspense(<SettingsPage />)} />
        <Route path="/ssl" element={withRouteSuspense(<SSLCertificatePage />)} />
      </Route>
    </Routes>
  );
}
