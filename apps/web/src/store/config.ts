import { defineStore } from "pinia";
import { ref } from "vue";
import { getSystemConfig, updateSystemConfig } from "@/api";
import type { SystemConfig } from "@/types";

export const useConfigStore = defineStore("config", () => {
  const config = ref<SystemConfig | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchConfig(): Promise<SystemConfig | null> {
    loading.value = true;
    error.value = null;
    try {
      const result = await getSystemConfig();
      config.value = result;
      return result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      return null;
    } finally {
      loading.value = false;
    }
  }

  async function saveConfig(value: SystemConfig): Promise<SystemConfig | null> {
    loading.value = true;
    error.value = null;
    try {
      const result = await updateSystemConfig(value);
      config.value = result;
      return result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      return null;
    } finally {
      loading.value = false;
    }
  }

  return {
    config,
    loading,
    error,
    fetchConfig,
    saveConfig,
  };
});
