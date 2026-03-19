import { ref, watch, onUnmounted, type Ref } from "vue";

export type EventStreamStatus = "idle" | "connecting" | "open" | "closed" | "error";

export interface UseEventStreamOptions {
  eventNames?: string[];
  closeOn?: string[];
  heartbeatEvents?: string[];
  includeHeartbeatInLines?: boolean;
}

const MAX_LOG_LINES = 200;

export function useEventStream(
  url: Ref<string | null>,
  options: UseEventStreamOptions = {}
) {
  const lines = ref<string[]>([]);
  const status = ref<EventStreamStatus>("idle");
  const lastHeartbeatAt = ref<number | null>(null);
  const heartbeatPayload = ref<unknown>(null);

  let source: EventSource | null = null;
  let closed = false;

  function cleanup() {
    if (source) {
      source.close();
      source = null;
    }
    closed = true;
  }

  function connect(streamUrl: string) {
    cleanup();
    closed = false;
    lines.value = [];
    status.value = "connecting";
    lastHeartbeatAt.value = null;
    heartbeatPayload.value = null;

    const es = new EventSource(streamUrl);
    source = es;

    es.onopen = () => {
      if (!closed) status.value = "open";
    };

    es.onerror = () => {
      if (closed) return;
      if (es.readyState === EventSource.CLOSED) {
        status.value = "error";
        return;
      }
      if (es.readyState === EventSource.CONNECTING) {
        status.value = "connecting";
        return;
      }
      // Keep stream OPEN when receiving SSE payloads whose event type happens to be "error".
      status.value = "open";
    };

    es.onmessage = (event: MessageEvent) => {
      lines.value = [...lines.value, event.data as string].slice(-MAX_LOG_LINES);
    };

    const trackedEvents = options.eventNames ?? [];
    const closeOnSet = new Set(options.closeOn ?? []);
    const heartbeatSet = new Set(options.heartbeatEvents ?? ["heartbeat"]);

    for (const eventName of trackedEvents) {
      es.addEventListener(eventName, (event: Event) => {
        const raw = ((event as MessageEvent).data as string) ?? "";
        const isHeartbeat = heartbeatSet.has(eventName);
        if (isHeartbeat) {
          lastHeartbeatAt.value = Date.now();
          try {
            heartbeatPayload.value = JSON.parse(raw);
          } catch {
            heartbeatPayload.value = raw;
          }
          if (!options.includeHeartbeatInLines) {
            return;
          }
        }
        lines.value = [...lines.value, raw].slice(-MAX_LOG_LINES);
        if (closeOnSet.has(eventName)) {
          status.value = "closed";
          es.close();
          closed = true;
        }
      });
    }
  }

  watch(
    url,
    (newUrl) => {
      cleanup();
      if (newUrl) {
        connect(newUrl);
      } else {
        lines.value = [];
        status.value = "idle";
      }
    },
    { immediate: true }
  );

  onUnmounted(cleanup);

  return { lines, status, lastHeartbeatAt, heartbeatPayload };
}
