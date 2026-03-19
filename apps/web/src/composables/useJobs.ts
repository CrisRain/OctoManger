import { ref, onMounted, onUnmounted, computed, unref, type Ref } from "vue";
import { storeToRefs } from "pinia";
import { getJobExecutionEventsUrl } from "@/api";
import { useJobsStore } from "@/store";
import type { JobDefinitionCreateInput } from "@/types";
import { useEventStream } from "./useEventStream";

export function useJobDefinitions() {
  const store = useJobsStore();
  const { jobDefinitions, loadingDefinitions, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchJobDefinitions();
  }

  onMounted(() => { void refresh(); });

  return { data: jobDefinitions, loading: loadingDefinitions, error, refresh };
}

export function useJobExecutions() {
  const store = useJobsStore();
  const { jobExecutions, loadingExecutions, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchJobExecutions();
  }

  let timer: ReturnType<typeof setInterval> | null = null;
  onMounted(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 5000);
  });
  onUnmounted(() => { if (timer) clearInterval(timer); });

  return { data: jobExecutions, loading: loadingExecutions, error, refresh };
}

export function useCreateJobDefinition() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useJobsStore();

  async function execute(payload: JobDefinitionCreateInput) {
    loading.value = true;
    error.value = null;
    try {
      const result = await store.createDefinition(payload);
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

export function useEnqueueJobExecution() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useJobsStore();

  async function execute(id: number) {
    loading.value = true;
    error.value = null;
    try {
      const result = await store.enqueueExecution(id);
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

const executionEventNames = ["log", "progress", "result", "state", "error"];

type MaybeRef<T> = T | Ref<T>;

export function useJobExecutionStream(executionId: MaybeRef<number | null>) {
  const streamUrl = computed(() => {
    const id = unref(executionId);
    return id ? getJobExecutionEventsUrl(id) : null;
  });

  return useEventStream(streamUrl, {
    eventNames: executionEventNames,
    closeOn: ["state"],
  });
}
