<script setup lang="ts">
import { computed } from "vue";
import { IconSearch, IconCommand } from "@/lib/icons";

interface CommandItem {
  id: string;
  label: string;
  description?: string;
  icon?: any;
  shortcut?: string;
  keywords?: string[];
  action?: () => void;
}

const props = withDefaults(defineProps<{
  commands: CommandItem[];
  open?: boolean;
  query?: string;
}>(), {
  commands: () => [],
  open: false,
  query: "",
});

const emit = defineEmits<{
  (e: "update:open", value: boolean): void;
  (e: "update:query", value: string): void;
  (e: "execute", command: CommandItem): void;
}>();

const isOpen = computed({
  get: () => props.open ?? false,
  set: (value) => emit("update:open", value),
});

const searchQuery = computed({
  get: () => props.query ?? "",
  set: (value) => emit("update:query", value),
});

const filteredCommands = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase();
  if (!keyword) return props.commands;

  return props.commands.filter((command) => {
    const tokens = [
      command.label,
      command.description ?? "",
      ...(command.keywords ?? []),
    ]
      .join(" ")
      .toLowerCase();
    return tokens.includes(keyword);
  });
});

function executeCommand(command: CommandItem) {
  if (command.action) {
    command.action();
  }
  emit("execute", command);
  emit("update:open", false);
  emit("update:query", "");
}
</script>

<template>
  <div class="keyboard-shortcuts">
    <!-- Command palette -->
    <ui-modal v-model:visible="isOpen" :footer="false" :closable="false" class="search-modal">
      <div class="panel-surface flex flex-col overflow-hidden bg-slate-50">
        <div class="flex items-center gap-3.5 border-b border-slate-200 bg-slate-50/85 px-5 py-4">
          <icon-search class="h-4 w-4 flex-shrink-0" />
          <input
            v-model="searchQuery"
            type="text"
            class="flex-1 bg-transparent text-base font-medium text-slate-900 placeholder:text-slate-400 outline-none"
            placeholder="搜索页面、功能..."
            autofocus
          />
          <kbd class="rounded-full border border-slate-200 bg-white/70 px-2.5 py-1 font-mono text-xs font-medium text-slate-500">ESC</kbd>
        </div>

        <div class="overflow-y-auto px-3 py-3">
          <button
            type="button"
            v-for="(command, index) in filteredCommands"
            :key="command.id"
            class="flex w-full items-center gap-3.5 rounded-xl border border-transparent bg-transparent px-4 py-3 text-left transition-all hover:border-slate-200 hover:bg-white/92 hover:shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/20"
            @click="executeCommand(command)"
          >
            <div class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-slate-50 border border-slate-200 text-slate-500 shadow-sm">
              <component :is="command.icon || IconSearch" class="h-4 w-4" />
            </div>
            <div class="flex min-w-0 flex-1 flex-col gap-0.5">
              <div class="text-sm font-semibold text-slate-900">{{ command.label }}</div>
              <div v-if="command.description" class="text-xs leading-5 text-slate-500">
                {{ command.description }}
              </div>
            </div>
            <div class="flex-shrink-0 rounded-full bg-slate-100 px-2 py-1 text-xs text-slate-500">
              <span v-if="command.shortcut">{{ command.shortcut }}</span>
              <span v-else-if="index === 0">↵</span>
              <span v-else>{{ index + 1 }}</span>
            </div>
          </button>

          <div v-if="!filteredCommands.length" class="py-10 text-center text-sm text-slate-500">
            未找到相关结果
          </div>
        </div>
      </div>
    </ui-modal>

    <!-- Floating trigger -->
    <button type="button"
      class="fixed bottom-[calc(1.25rem_+_env(safe-area-inset-bottom))] right-5 z-50 inline-flex h-11 w-11 items-center justify-center rounded-full border border-slate-200 bg-white/92 text-slate-500 shadow-md backdrop-blur transition-all hover:border-[var(--accent)]/30 hover:text-[var(--accent)] hover:-translate-y-0.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/20"
      @click="emit('update:open', true)"
      title="快捷键帮助 (⌘K)"
    >
      <icon-command />
    </button>
  </div>
</template>
