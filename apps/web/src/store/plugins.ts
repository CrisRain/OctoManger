import { defineStore } from "pinia";
import { ref } from "vue";
import { getPluginSettings, listPlugins, syncPlugins, updatePluginSettings } from "@/api";
import type { Plugin, PluginSyncResult } from "@/types";
import { normalizeListResponse } from "@/utils/normalizeListResponse";

export const usePluginsStore = defineStore("plugins", () => {
  const plugins = ref<Plugin[]>([]);
  const pluginSettings = ref<Record<string, Record<string, unknown>>>({});
  const loading = ref(false);
  const saving = ref(false);
  const error = ref<string | null>(null);

  async function fetchPlugins() {
    loading.value = true;
    error.value = null;
    try {
      plugins.value = normalizeListResponse<Plugin>(await listPlugins());
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loading.value = false;
    }
  }

  async function fetchPluginSettings(key: string) {
    loading.value = true;
    error.value = null;
    try {
      const settings = await getPluginSettings(key);
      pluginSettings.value = { ...pluginSettings.value, [key]: settings };
      return settings;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      return null;
    } finally {
      loading.value = false;
    }
  }

  async function savePluginSettings(key: string, values: Record<string, unknown>) {
    saving.value = true;
    error.value = null;
    try {
      await updatePluginSettings(key, values);
      pluginSettings.value = { ...pluginSettings.value, [key]: values };
    } catch (e) {
      error.value = e instanceof Error ? e.message : "保存失败";
      throw e;
    } finally {
      saving.value = false;
    }
  }

  async function syncAllPlugins(): Promise<PluginSyncResult | null> {
    loading.value = true;
    error.value = null;
    try {
      const result = await syncPlugins();
      await fetchPlugins();
      return result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      return null;
    } finally {
      loading.value = false;
    }
  }

  return {
    plugins,
    pluginSettings,
    loading,
    saving,
    error,
    fetchPlugins,
    fetchPluginSettings,
    savePluginSettings,
    syncAllPlugins,
  };
});
