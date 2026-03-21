import { storeToRefs } from "pinia";
import { useSystemStore } from "@/store";
import type { DashboardSummary } from "@/types";
import { useAutoRefresh } from "./useAutoRefresh";

export type { DashboardSummary };

export function useSystemStatus() {
  const store = useSystemStore();
  const { systemStatus, loadingStatus, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchSystemStatus();
  }

  const autoRefresh = useAutoRefresh(refresh, {
    intervalMs: 15000,
  });

  return { data: systemStatus, loading: loadingStatus, error, refresh: autoRefresh.refresh };
}

export function useDashboardSnapshot() {
  const store = useSystemStore();
  const { dashboardSummary, loadingDashboard, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchDashboardSummary();
    if (dashboardSummary.value) {
      console.log("[useDashboardSnapshot] response:", dashboardSummary.value);
    }
  }

  const autoRefresh = useAutoRefresh(refresh, {
    intervalMs: 15000,
  });

  return { data: dashboardSummary, loading: loadingDashboard, error, refresh: autoRefresh.refresh };
}
