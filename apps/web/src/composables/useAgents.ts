import { ref, onMounted, onUnmounted, computed, unref, type Ref } from "vue";
import { storeToRefs } from "pinia";
import { getAgentEventsUrl } from "@/api";
import { useAgentsStore } from "@/store";
import type { AgentStatus } from "@/types";
import { useEventStream } from "./useEventStream";

export function useAgents() {
  const store = useAgentsStore();
  const { agents, loading, error } = storeToRefs(store);

  async function refresh() {
    await store.fetchAgents();
  }

  let timer: ReturnType<typeof setInterval> | null = null;
  onMounted(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 5000);
  });
  onUnmounted(() => { if (timer) clearInterval(timer); });

  return { data: agents, loading, error, refresh };
}

export function useAgent(id: number) {
  const store = useAgentsStore();
  const { agents, loading, error } = storeToRefs(store);
  const data = computed(() => agents.value.find((a) => a.id === id) ?? null);

  async function refresh() {
    await store.fetchAgents();
  }

  let timer: ReturnType<typeof setInterval> | null = null;
  onMounted(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 5000);
  });
  onUnmounted(() => { if (timer) clearInterval(timer); });

  return { data, loading, error, refresh };
}

export function useCreateAgent() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useAgentsStore();

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

export function useStartAgent() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useAgentsStore();

  async function execute(id: number) {
    loading.value = true;
    error.value = null;
    try {
      await store.start(id);
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}

export function useStopAgent() {
  const loading = ref(false);
  const error = ref<string | null>(null);
  const store = useAgentsStore();

  async function execute(id: number) {
    loading.value = true;
    error.value = null;
    try {
      await store.stop(id);
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return { loading, error, execute };
}

// ── useAgentStatus ────────────────────────────────────────────────────────────
// Polls GET /agents/:id/status every 3 s. This endpoint is served from the
// Redis cache (TTL 5 s) written by the worker on every state transition,
// so it never hits the DB on the hot path.
export function useAgentStatus(agentId: number) {
  const data = ref<AgentStatus | null>(null);
  const error = ref<string | null>(null);
  const store = useAgentsStore();

  async function refresh() {
    try {
      data.value = await store.fetchStatus(agentId);
      error.value = null;
    } catch (e) {
      // status polling is always silent — don't surface transient errors
    }
  }

  let timer: ReturnType<typeof setInterval> | null = null;
  onMounted(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 3000);
  });
  onUnmounted(() => { if (timer) clearInterval(timer); });

  return { data, error, refresh };
}

export interface AgentStreamState {
  runtime_state?: string;
  desired_state?: string;
  last_error?: string;
}

type MaybeRef<T> = T | Ref<T>;

const agentEventNames = ["heartbeat", "log", "progress", "error"];

export function useAgentEventStream(agentId: MaybeRef<number | null>) {
  const streamUrl = computed(() => {
    const id = unref(agentId);
    return id ? getAgentEventsUrl(id) : null;
  });

  const stream = useEventStream(streamUrl, {
    eventNames: agentEventNames,
  });

  const runtimeState = computed(() => {
    const payload = stream.heartbeatPayload.value as { runtime_state?: string } | null;
    return payload?.runtime_state ?? null;
  });

  const isRunning = computed(() => {
    if (runtimeState.value) {
      return runtimeState.value === "running" || runtimeState.value === "idle";
    }
    return stream.status.value === "open" || stream.status.value === "connecting";
  });

  return {
    ...stream,
    runtimeState,
    isRunning,
  };
}

export function useAgentStream(agentId: number | null) {
  const streamUrl = computed(() =>
    agentId ? getAgentEventsUrl(agentId) : null
  );

  const logLines = ref<string[]>([]);
  const connected = ref(false);
  const lastHeartbeatAt = ref<number | null>(null);
  const runtimeState = ref<string | null>(null);

  let source: EventSource | null = null;

  function cleanup() {
    if (source) {
      source.close();
      source = null;
    }
    connected.value = false;
  }

  function connect(url: string) {
    cleanup();
    logLines.value = [];

    const es = new EventSource(url);
    source = es;

    es.onopen = () => { connected.value = true; };
    es.onerror = () => {
      if (es.readyState === EventSource.CLOSED) {
        connected.value = false;
        return;
      }
      if (es.readyState === EventSource.CONNECTING) {
        // Keep UI in live mode while EventSource reconnects automatically.
        connected.value = true;
        return;
      }
      // Ignore SSE payloads using event type "error" (business-level errors).
      connected.value = true;
    };

    const onLog = (e: Event) => {
      logLines.value = [...logLines.value, (e as MessageEvent<string>).data].slice(-200);
    };
    const onProgress = (e: Event) => {
      logLines.value = [...logLines.value, (e as MessageEvent<string>).data].slice(-200);
    };
    const onError = (e: Event) => {
      logLines.value = [...logLines.value, (e as MessageEvent<string>).data].slice(-200);
    };
    const onHeartbeat = (e: Event) => {
      connected.value = true;
      lastHeartbeatAt.value = Date.now();
      try {
        const payload = JSON.parse((e as MessageEvent<string>).data ?? "{}") as { runtime_state?: string };
        runtimeState.value = payload.runtime_state ?? null;
      } catch {
        // ignore malformed heartbeat payloads
      }
    };

    es.addEventListener("log", onLog);
    es.addEventListener("progress", onProgress);
    es.addEventListener("error", onError);
    es.addEventListener("heartbeat", onHeartbeat);
    // "state" events are intentionally ignored — status comes from useAgentStatus
  }

  onMounted(() => {
    if (streamUrl.value) connect(streamUrl.value);
  });

  onUnmounted(cleanup);

  return { lines: logLines, connected, lastHeartbeatAt, runtimeState };
}
