import { ref, onMounted, computed } from "vue";
import { storeToRefs } from "pinia";
import { usePluginsStore } from "@/store";
import type { PluginSyncResult } from "@/types";

export function usePlugins() {
  const store = usePluginsStore();
  const { plugins, loading, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchPlugins();
  }

  onMounted(() => { void refresh(); });

  return { data: plugins, loading, error, refresh };
}

export function usePluginSettings(pluginKey: string) {
  const store = usePluginsStore();
  const { pluginSettings, loading, saving, error } = storeToRefs(store);
  const data = computed(() => pluginSettings.value[pluginKey] ?? {});

  async function load() {
    await store.fetchPluginSettings(pluginKey);
  }

  async function save(values: Record<string, unknown>) {
    await store.savePluginSettings(pluginKey, values);
  }

  return { data, loading, saving, error, load, save };
}

export function useSyncPlugins() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = usePluginsStore();

  async function execute(): Promise<PluginSyncResult> {
    loading.value = true;
    error.value = null;
    try {
      const result = await store.syncAllPlugins();
      if (!result) {
        throw new Error("同步失败");
      }
      return result;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}
