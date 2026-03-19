import { defineStore } from "pinia";
import { ref } from "vue";
import { getConfig, setConfig } from "@/api";
import type { SetConfigRequestBody, SystemConfigValue } from "@/types";

export const useConfigStore = defineStore("config", () => {
  const configs = ref<Record<string, unknown>>({});
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchConfig(key: string): Promise<SystemConfigValue | null> {
    loading.value = true;
    error.value = null;
    try {
      const result = await getConfig(key);
      configs.value = { ...configs.value, [key]: result.value };
      return result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      return null;
    } finally {
      loading.value = false;
    }
  }

  async function updateConfig(
    key: string,
    value: SetConfigRequestBody["value"],
  ): Promise<SystemConfigValue | null> {
    loading.value = true;
    error.value = null;
    try {
      const result = await setConfig(key, value);
      configs.value = { ...configs.value, [key]: result.value };
      return result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      return null;
    } finally {
      loading.value = false;
    }
  }

  return {
    configs,
    loading,
    error,
    fetchConfig,
    updateConfig,
  };
});
