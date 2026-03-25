import { onMounted, onUnmounted } from "vue";

export interface UseAutoRefreshOptions {
  intervalMs?: number | null;
  immediate?: boolean;
  refreshOnFocus?: boolean;
  refreshOnVisible?: boolean;
}

function hasDOM(): boolean {
  return typeof window !== "undefined" && typeof document !== "undefined";
}

function isDocumentVisible(): boolean {
  return typeof document === "undefined" || document.visibilityState === "visible";
}

export function useAutoRefresh(
  refresh: () => Promise<void>,
  options: UseAutoRefreshOptions = {}
) {
  const {
    intervalMs = null,
    immediate = true,
    refreshOnFocus = true,
    refreshOnVisible = true,
  } = options;

  let timer: ReturnType<typeof setInterval> | null = null;
  let inFlight: Promise<void> | null = null;

  function stopTimer() {
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
  }

  function runRefresh() {
    if (inFlight) {
      return inFlight;
    }
    inFlight = refresh().finally(() => {
      inFlight = null;
    });
    return inFlight;
  }

  function startTimer() {
    stopTimer();
    if (!intervalMs || intervalMs <= 0 || !isDocumentVisible()) {
      return;
    }
    timer = setInterval(() => {
      if (!isDocumentVisible()) {
        return;
      }
      void runRefresh();
    }, intervalMs);
  }

  function handleVisibilityChange() {
    if (!isDocumentVisible()) {
      stopTimer();
      return;
    }
    if (refreshOnVisible) {
      void runRefresh();
    }
    startTimer();
  }

  function handleFocus() {
    if (refreshOnFocus) {
      void runRefresh();
    }
  }

  onMounted(() => {
    if (immediate) {
      void runRefresh();
    }
    startTimer();

    if (!hasDOM()) {
      return;
    }
    if (refreshOnVisible) {
      document.addEventListener("visibilitychange", handleVisibilityChange);
    }
    if (refreshOnFocus) {
      window.addEventListener("focus", handleFocus);
    }
  });

  onUnmounted(() => {
    stopTimer();
    if (!hasDOM()) {
      return;
    }
    if (refreshOnVisible) {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    }
    if (refreshOnFocus) {
      window.removeEventListener("focus", handleFocus);
    }
  });

  return {
    refresh: runRefresh,
  };
}
