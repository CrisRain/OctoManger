import { defineStore } from "pinia";
import { ref } from "vue";
import { getDashboardSummary, getSystemStatus } from "@/api";
import type { DashboardSummary, SystemStatus } from "@/types";

export const useSystemStore = defineStore("system", () => {
  const systemStatus = ref<SystemStatus | null>(null);
  const dashboardSummary = ref<DashboardSummary | null>(null);
  const loadingStatus = ref(false);
  const loadingDashboard = ref(false);
  const error = ref<string | null>(null);

  async function fetchSystemStatus() {
    loadingStatus.value = true;
    error.value = null;
    try {
      systemStatus.value = await getSystemStatus();
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loadingStatus.value = false;
    }
  }

  async function fetchDashboardSummary() {
    loadingDashboard.value = true;
    error.value = null;
    try {
      dashboardSummary.value = await getDashboardSummary();
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loadingDashboard.value = false;
    }
  }

  return {
    systemStatus,
    dashboardSummary,
    loadingStatus,
    loadingDashboard,
    error,
    fetchSystemStatus,
    fetchDashboardSummary,
  };
});
