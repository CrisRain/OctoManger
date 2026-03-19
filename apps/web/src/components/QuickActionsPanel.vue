<script setup lang="ts">
import { useRouter } from "vue-router";
import {
  IconUser,
  IconEmail,
  IconSync,
  IconRobot,
  IconApps,
  IconRight,
} from "@/lib/icons";

interface QuickAction {
  id: string;
  label: string;
  description: string;
  icon?: any;
  color?: string;
  path?: string;
  shortcut?: string;
  action?: () => void;
}

interface RecentItem {
  id: string;
  name: string;
  type?: "account" | "job" | "agent" | "email" | string;
  path?: string;
  updatedAt?: string;
  icon?: any;
  action?: () => void;
}

const props = withDefaults(defineProps<{
  actions: QuickAction[];
  recentItems?: RecentItem[];
  showSearchHint?: boolean;
}>(), {
  actions: () => [],
  recentItems: () => [],
  showSearchHint: true,
});

const emit = defineEmits<{
  (e: "open-search"): void;
}>();

const router = useRouter();

function handleActionClick(action: QuickAction) {
  if (action.action) {
    action.action();
    return;
  }
  if (action.path) {
    router.push(action.path);
  }
}

function handleRecentClick(item: RecentItem) {
  if (item.action) {
    item.action();
    return;
  }
  if (item.path) {
    router.push(item.path);
  }
}

function resolveRecentIcon(item: RecentItem) {
  if (item.icon) return item.icon;
  switch (item.type) {
    case "account":
      return IconUser;
    case "job":
      return IconSync;
    case "agent":
      return IconRobot;
    case "email":
      return IconEmail;
    default:
      return IconApps;
  }
}
</script>

<template>
  <div class="quick-actions-panel">
    <!-- Quick actions -->
    <div class="quick-actions-panel__section">
      <div class="quick-actions-panel__header">
        <h3 class="quick-actions-panel__title">快捷操作</h3>
        <button type="button"
          v-if="showSearchHint"
          class="quick-actions-panel__search-trigger"
          @click="emit('open-search')"
        >
          <kbd>⌘K</kbd>
          <span>打开搜索</span>
        </button>
      </div>

      <div class="quick-actions-panel__grid">
        <button
          type="button"
          v-for="action in actions"
          :key="action.id"
          class="quick-actions-panel__card"
          :class="action.color ? `quick-actions-panel__card--${action.color}` : ''"
          @click="handleActionClick(action)"
        >
          <div class="quick-actions-panel__card-icon">
            <component :is="action.icon || IconApps" />
          </div>
          <div class="quick-actions-panel__card-label">{{ action.label }}</div>
          <div class="quick-actions-panel__card-desc">{{ action.description }}</div>
          <div class="quick-actions-panel__card-shortcut" v-if="action.shortcut">
            {{ action.shortcut }}
          </div>
        </button>
      </div>
    </div>

    <!-- Recent items -->
    <div class="quick-actions-panel__recent" v-if="recentItems?.length">
      <h3 class="quick-actions-panel__recent-title">最近使用</h3>
      <button
        type="button"
        v-for="item in recentItems"
        :key="item.id"
        class="quick-actions-panel__recent-item"
        @click="handleRecentClick(item)"
      >
        <div class="quick-actions-panel__recent-icon">
          <component :is="resolveRecentIcon(item)" />
        </div>
        <div class="quick-actions-panel__recent-name">{{ item.name }}</div>
        <div class="quick-actions-panel__recent-meta">{{ item.updatedAt }}</div>
        <icon-right class="quick-actions-panel__recent-arrow" />
      </button>
    </div>
  </div>
</template>
