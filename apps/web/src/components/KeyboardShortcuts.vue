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
      <div class="command-palette">
        <div class="command-palette__input-wrap">
          <icon-search class="command-palette__search-icon" />
          <input
            v-model="searchQuery"
            type="text"
            class="command-palette__input"
            placeholder="搜索页面、功能..."
            autofocus
          />
          <kbd class="command-palette__shortcut">ESC</kbd>
        </div>

        <div class="command-palette__list">
          <button
            type="button"
            v-for="(command, index) in filteredCommands"
            :key="command.id"
            class="command-palette__item"
            @click="executeCommand(command)"
          >
            <div class="command-palette__item-icon">
              <component :is="command.icon || IconSearch" />
            </div>
            <div class="command-palette__item-content">
              <div class="command-palette__item-label">{{ command.label }}</div>
              <div v-if="command.description" class="command-palette__item-desc">
                {{ command.description }}
              </div>
            </div>
            <div class="command-palette__item-shortcut">
              <span v-if="command.shortcut">{{ command.shortcut }}</span>
              <span v-else-if="index === 0">↵</span>
              <span v-else>{{ index + 1 }}</span>
            </div>
          </button>

          <div v-if="!filteredCommands.length" class="command-palette__empty">
            未找到相关结果
          </div>
        </div>
      </div>
    </ui-modal>

    <!-- Floating trigger -->
    <button type="button"
      class="command-palette__trigger"
      @click="emit('update:open', true)"
      title="快捷键帮助 (⌘K)"
    >
      <icon-command />
    </button>
  </div>
</template>
