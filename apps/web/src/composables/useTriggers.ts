import { ref, onMounted } from "vue";
import { storeToRefs } from "pinia";
import { useTriggersStore } from "@/store";
import type { TriggerFireInput, TriggerFireResult } from "@/types";

export function useTriggers() {
  const store = useTriggersStore();
  const { triggers, loading, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchTriggers();
  }

  onMounted(() => { void refresh(); });

  return { data: triggers, loading, error, refresh };
}

export function useCreateTrigger() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useTriggersStore();

  async function execute(payload: Parameters<typeof store.create>[0]) {
    loading.value = true;
    error.value = null;
    try {
      const result = await store.create(payload);
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

export function useDeleteTrigger() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useTriggersStore();

  async function execute(id: number) {
    loading.value = true;
    error.value = null;
    try {
      await store.remove(id);
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}

export function useFireTrigger() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const result = ref<unknown>(null);
  const store = useTriggersStore();

  async function execute(id: number, payload?: TriggerFireInput): Promise<TriggerFireResult> {
    loading.value = true;
    error.value = null;
    result.value = null;
    try {
      const res = await store.fire(id, payload);
      result.value = res;
      return res;
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, result, execute };
}
