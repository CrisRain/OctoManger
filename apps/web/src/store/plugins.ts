import { defineStore } from "pinia";
import { ref } from "vue";
import {
  getPluginRuntimeConfig,
  getPluginSettings,
  listPlugins,
  syncPlugins,
  updatePluginRuntimeConfig,
  updatePluginSettings,
} from "@/api";
import type { Plugin, PluginRuntimeConfig, PluginRuntimeConfigInput, PluginSyncResult } from "@/types";
import { normalizeListResponse } from "@/utils/normalizeListResponse";

export const usePluginsStore = defineStore("plugins", () => {
  const plugins = ref<Plugin[]>([]);
  const pluginSettings = ref<Record<string, Record<string, unknown>>>({});
  const pluginRuntimeConfigs = ref<Record<string, PluginRuntimeConfig>>({});
  const loading = ref(false);
  const loadingSettings = ref(false);
  const savingSettings = ref(false);
  const loadingRuntimeConfig = ref(false);
  const savingRuntimeConfig = ref(false);
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
    loadingSettings.value = true;
    error.value = null;
    try {
      const settings = await getPluginSettings(key);
      pluginSettings.value = { ...pluginSettings.value, [key]: settings };
      return settings;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      return null;
    } finally {
      loadingSettings.value = false;
    }
  }

  async function savePluginSettings(key: string, values: Record<string, unknown>) {
    savingSettings.value = true;
    error.value = null;
    try {
      await updatePluginSettings(key, values);
      pluginSettings.value = { ...pluginSettings.value, [key]: values };
    } catch (e) {
      error.value = e instanceof Error ? e.message : "保存失败";
      throw e;
    } finally {
      savingSettings.value = false;
    }
  }

  async function fetchPluginRuntimeConfigEntry(key: string) {
    loadingRuntimeConfig.value = true;
    error.value = null;
    try {
      const config = await getPluginRuntimeConfig(key);
      pluginRuntimeConfigs.value = { ...pluginRuntimeConfigs.value, [key]: config };
      return config;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      return null;
    } finally {
      loadingRuntimeConfig.value = false;
    }
  }

  async function savePluginRuntimeConfigEntry(
    key: string,
    value: PluginRuntimeConfigInput,
  ) {
    savingRuntimeConfig.value = true;
    error.value = null;
    try {
      const config = await updatePluginRuntimeConfig(key, value);
      pluginRuntimeConfigs.value = { ...pluginRuntimeConfigs.value, [key]: config };
      return config;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "保存失败";
      throw e;
    } finally {
      savingRuntimeConfig.value = false;
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
    pluginRuntimeConfigs,
    loading,
    loadingSettings,
    savingSettings,
    loadingRuntimeConfig,
    savingRuntimeConfig,
    error,
    fetchPlugins,
    fetchPluginSettings,
    savePluginSettings,
    fetchPluginRuntimeConfig: fetchPluginRuntimeConfigEntry,
    savePluginRuntimeConfig: savePluginRuntimeConfigEntry,
    syncAllPlugins,
  };
});
