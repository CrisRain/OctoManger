import { onMounted, onUnmounted } from "vue";
import { storeToRefs } from "pinia";
import { useSystemStore } from "@/store";
import type { DashboardSummary } from "@/types";

export type { DashboardSummary };

export function useSystemStatus() {
  const store = useSystemStore();
  const { systemStatus, loadingStatus, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchSystemStatus();
  }

  let timer: ReturnType<typeof setInterval> | null = null;
  onMounted(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 10000);
  });
  onUnmounted(() => { if (timer) clearInterval(timer); });

  return { data: systemStatus, loading: loadingStatus, error, refresh };
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

  let timer: ReturnType<typeof setInterval> | null = null;
  onMounted(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 10000);
  });
  onUnmounted(() => { if (timer) clearInterval(timer); });

  return { data: dashboardSummary, loading: loadingDashboard, error, refresh };
}
