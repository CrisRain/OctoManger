<script setup lang="ts">
/**
 * LogTerminal — Virtual-scrolled terminal component.
 *
 * Uses vue-virtual-scroller DynamicScroller so only the DOM nodes
 * visible in the viewport are rendered, regardless of total log count.
 * Handles variable-height rows (wrapped long messages) via DynamicScrollerItem.
 */
import { ref, computed, watch, nextTick } from "vue";
import { DynamicScroller, DynamicScrollerItem } from "vue-virtual-scroller";
import "vue-virtual-scroller/dist/vue-virtual-scroller.css";

interface Props {
  /** Raw log lines to display */
  logs: string[];
  title?: string;
  /** Whether the log stream is currently live (controls the status LED) */
  isLive?: boolean;
  /** @deprecated Height is now controlled by the parent container via CSS */
  heightClass?: string;
  showHeader?: boolean;
  emptyLabel?: string;
}

const props = withDefaults(defineProps<Props>(), {
  title: "运行日志",
  isLive: false,
  heightClass: "h-full",
  showHeader: true,
  emptyLabel: "暂无日志",
});

// ── State ──────────────────────────────────────────────────────────────────
const follow = ref(true);
const scrollerRef = ref<InstanceType<typeof DynamicScroller> | null>(null);

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

// ── Auto-scroll to bottom ──────────────────────────────────────────────────
/**
 * Scroll to the last item whenever new logs arrive and follow is enabled.
 * scrollToItem() tells the virtual scroller to jump to a specific index,
 * which is efficient regardless of total item count.
 */
async function scrollToBottom() {
  const len = props.logs.length;
  if (len === 0) return;
  await nextTick();
  scrollerRef.value?.scrollToItem(len - 1);
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
  <div class="terminal-surface">
    <!-- ── Optional header bar ──────────────────────────────────────────── -->
    <div v-if="showHeader" class="term-header">
      <div class="term-header-left">
        <span class="status-led" :class="{ live: isLive }" />
        <span class="term-title">{{ title }}</span>
      </div>
      <button type="button"
        class="follow-btn"
        :class="{ 'follow-btn--active': follow }"
        @click="follow = !follow"
      >
        {{ follow ? "Follow" : "Paused" }}
      </button>
    </div>

    <!-- ── Empty state ──────────────────────────────────────────────────── -->
    <div v-if="!logs.length" class="term-empty">{{ emptyLabel }}</div>

    <!-- ── Virtual list ─────────────────────────────────────────────────── -->
    <!--
      DynamicScroller renders only the items currently visible in the
      viewport. DynamicScrollerItem measures each row after mount and
      informs the scroller of its actual height, supporting variable-height
      rows (e.g., long messages that wrap to multiple lines).

      key-field="id"   → uses ParsedLine.id as the item key
      min-item-size=20 → initial height estimate (1 line @ 11px / 1.25rem)
    -->
    <DynamicScroller
      v-else
      ref="scrollerRef"
      :items="parsedItems"
      :min-item-size="20"
      key-field="id"
      class="term-scroller"
    >
      <template #default="{ item, active }">
        <!--
          size-dependencies: when message or source changes, the row is
          re-measured. For append-only logs this rarely triggers, but it
          keeps height estimates accurate if lines are ever mutated.
        -->
        <DynamicScrollerItem
          :item="item"
          :active="active"
          :size-dependencies="[asParsedLine(item).message, asParsedLine(item).source]"
        >
          <div class="log-row">
            <!-- Level badge (fixed 36px wide) -->
            <span class="log-level" :style="{ color: levelColor(asParsedLine(item).level) }">
              {{ asParsedLine(item).level || "log" }}
            </span>
            <!-- Source (fixed 56px wide) -->
            <span class="log-source">{{ asParsedLine(item).source || "—" }}</span>
            <!-- Message (flexible, wraps) -->
            <span class="log-message">{{ asParsedLine(item).message }}</span>
          </div>
        </DynamicScrollerItem>
      </template>
    </DynamicScroller>

    <!-- CRT scanline overlay (purely decorative) -->
    <div class="scanlines" aria-hidden="true" />
  </div>
</template>
