<script setup lang="ts">
/**
 * LogTerminal — fixed-height terminal with internal scrolling.
 */
import { ref, computed, watch, nextTick } from "vue";

interface Props {
  /** Raw log lines to display */
  logs: string[];
  title?: string;
  /** Whether the log stream is currently live (controls the status LED) */
  isLive?: boolean;
  /** Terminal height. Accepts CSS length values like "34rem" or numbers in px. */
  height?: string | number;
  showHeader?: boolean;
  emptyLabel?: string;
}

const props = withDefaults(defineProps<Props>(), {
  title: "运行日志",
  isLive: false,
  height: "34em",
  showHeader: true,
  emptyLabel: "暂无日志",
});

// ── State ──────────────────────────────────────────────────────────────────
const follow = ref(true);
const scrollerRef = ref<HTMLDivElement | null>(null);

// ── Log parsing ────────────────────────────────────────────────────────────
interface ParsedLine {
  /** Stable unique ID — logs are append-only so index is safe */
  id: number;
  source: string;
  level: string;
  message: string;
}

const LEVEL_RE = /^\[(.*?)\]\[(.*?)\]\s*(.*)$/;

function parseLine(raw: string, index: number): ParsedLine {
  const m = LEVEL_RE.exec(raw);
  if (!m) return { id: index, source: "", level: "", message: raw };
  return { id: index, source: m[1] ?? "", level: m[2] ?? "", message: m[3] ?? "" };
}

const parsedItems = computed<ParsedLine[]>(() =>
  props.logs.map((line, i) => parseLine(line, i))
);

// ── Level colors ───────────────────────────────────────────────────────────
const LEVEL_COLOR: Record<string, string> = {
  error: "#fca5a5",
  warn: "#fde68a",
  info: "#5eead4",
  debug: "#cbd5e1",
};

function levelColor(level: string): string {
  return LEVEL_COLOR[level.toLowerCase()] ?? "#cbd5e1";
}

function asParsedLine(item: unknown): ParsedLine {
  return item as ParsedLine;
}

const surfaceStyle = computed(() => ({
  "--terminal-height": typeof props.height === "number" ? `${props.height}px` : props.height,
}));

// ── Auto-scroll to bottom ──────────────────────────────────────────────────
async function scrollToBottom() {
  if (!props.logs.length) return;
  await nextTick();
  const container = scrollerRef.value;
  if (!container) return;
  container.scrollTop = container.scrollHeight;
}

watch(() => props.logs.length, () => {
  if (follow.value) void scrollToBottom();
});

// Re-enable follow → jump to bottom immediately
watch(follow, (on) => {
  if (on) void scrollToBottom();
});
</script>

<template>
  <div
    class="relative flex flex-col overflow-hidden rounded-xl bg-[#0d1117] font-mono text-[12.5px] leading-relaxed shadow-lg ring-1 ring-white/[4%]"
    :style="[surfaceStyle, { height: 'var(--terminal-height)' }]"
  >
    <!-- ── Header bar ────────────────────────────────────────────────────── -->
    <div v-if="showHeader" class="flex flex-shrink-0 items-center gap-3 border-b border-white/[5%] bg-slate-900/60 px-4 py-2.5 backdrop-blur-sm">
      <!-- macOS traffic lights -->
      <div class="flex items-center gap-1.5" aria-hidden="true">
        <span class="h-3 w-3 rounded-full bg-[#ff5f57]" />
        <span class="h-3 w-3 rounded-full bg-[#febc2e]" />
        <span class="h-3 w-3 rounded-full bg-[#28c840]" />
      </div>

      <!-- Title -->
      <span class="flex-1 truncate text-[11px] font-medium text-slate-500">{{ title }}</span>

      <!-- Live badge -->
      <span
        v-if="isLive"
        class="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-emerald-400"
      >
        <span class="h-1.5 w-1.5 animate-[pulse-dot-soft_1.8s_ease-in-out_infinite] rounded-full bg-emerald-400" />
        Live
      </span>

      <!-- Line count -->
      <span class="text-[10px] tabular-nums text-slate-600">{{ logs.length }} lines</span>

      <!-- Follow toggle -->
      <button
        type="button"
        class="rounded border px-2 py-0.5 text-[10px] font-medium transition-colors focus-visible:outline-none"
        :class="follow
          ? 'border-emerald-700/60 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/15'
          : 'border-slate-700 bg-transparent text-slate-500 hover:border-slate-500 hover:text-slate-300'"
        @click="follow = !follow"
      >
        {{ follow ? "↓ Follow" : "Paused" }}
      </button>
    </div>

    <!-- ── Log body ──────────────────────────────────────────────────────── -->
    <div class="relative flex min-h-0 flex-1 flex-col">
      <!-- Empty state -->
      <div v-if="!logs.length" class="flex flex-1 items-center justify-center gap-2 py-12">
        <span class="text-[10px] text-slate-700">▌</span>
        <span class="text-xs text-slate-600">{{ emptyLabel }}</span>
      </div>

      <!-- Scrollable lines -->
      <div v-else ref="scrollerRef" class="flex-1 overflow-y-auto overscroll-contain py-2 dark-scroll">
        <div
          v-for="item in parsedItems"
          :key="item.id"
          class="group flex items-baseline gap-0 hover:bg-white/[3%]"
        >
          <!-- Line number gutter -->
          <span class="w-10 flex-shrink-0 select-none pr-3 text-right text-[10px] text-slate-700 group-hover:text-slate-500">
            {{ item.id + 1 }}
          </span>

          <!-- Level badge -->
          <span
            class="mr-2.5 flex-shrink-0 rounded px-1 py-px text-[9px] font-bold uppercase tracking-wide"
            :style="{ color: levelColor(item.level), background: levelColor(item.level) + '20' }"
          >{{ item.level || "log" }}</span>

          <!-- Source -->
          <span
            v-if="item.source"
            class="mr-3 max-w-[9em] flex-shrink-0 overflow-hidden text-ellipsis whitespace-nowrap text-[10px] text-slate-600"
          >{{ item.source }}</span>

          <!-- Message -->
          <span class="flex-1 break-all pr-4 text-slate-300">{{ item.message }}</span>
        </div>
      </div>
    </div>

    <!-- ── Status bar ────────────────────────────────────────────────────── -->
    <div class="flex flex-shrink-0 items-center justify-between border-t border-white/[4%] bg-slate-900/40 px-4 py-1 text-[10px]">
      <span class="text-slate-700">{{ title }}</span>
      <span
        class="flex items-center gap-1.5 font-medium"
        :class="isLive ? 'text-emerald-600' : 'text-slate-600'"
      >
        <span
          class="h-1.5 w-1.5 rounded-full"
          :class="isLive ? 'bg-emerald-500 animate-[pulse-dot-soft_1.8s_ease-in-out_infinite]' : 'bg-slate-600'"
        />
        {{ isLive ? "streaming" : "idle" }}
      </span>
    </div>

    <!-- CRT scanline overlay -->
    <div
      class="pointer-events-none absolute inset-0 bg-[repeating-linear-gradient(to_bottom,_transparent_0px,_transparent_3px,_rgba(0,_0,_0,_0.025)_3px,_rgba(0,_0,_0,_0.025)_4px)]"
      aria-hidden="true"
    />
  </div>
</template>
