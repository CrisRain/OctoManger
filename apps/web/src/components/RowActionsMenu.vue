<script setup lang="ts">
import { ref, computed } from "vue";
import {
  IconMore,
  IconEdit,
  IconDelete,
  IconEye,
  IconCopy,
  IconPlayArrow,
  IconStop,
  IconRefresh,
} from "@/lib/icons";

interface Action {
  key: string;
  label?: string;
  icon?: any;
  danger?: boolean;
  divider?: boolean;
  disabled?: boolean;
}

interface Props {
  actions?: Action[];
  item?: any;
}

const props = withDefaults(defineProps<Props>(), {
  actions: () => [],
});

const emit = defineEmits<{
  (e: "action", key: string, item: any): void;
}>();

const visible = ref(false);

// 默认操作
const defaultActions: Action[] = [
  { key: "view", label: "查看详情", icon: IconEye },
  { key: "edit", label: "编辑", icon: IconEdit },
  { key: "copy", label: "复制", icon: IconCopy },
  { key: "divider-delete", divider: true },
  { key: "delete", label: "删除", icon: IconDelete, danger: true },
];

const iconMap: Record<string, any> = {
  IconEye,
  IconEdit,
  IconDelete,
  IconCopy,
  IconPlayArrow,
  IconStop,
  IconRefresh,
};

function resolveIcon(icon?: any) {
  if (!icon) return undefined;
  if (typeof icon === "string") return iconMap[icon];
  return icon;
}

const allActions = computed(() => {
  const source = props.actions.length > 0 ? props.actions : defaultActions;
  const normalized = source
    .map((action, index) => {
      if (action.divider || action.key === "divider") {
        return { key: `${action.key || "divider"}-${index}`, divider: true } satisfies Action;
      }
      return action;
    })
    .filter((action) => action.divider || action.label);

  return normalized.filter((action, index) => {
    if (!action.divider) return true;
    const prev = normalized[index - 1];
    const next = normalized[index + 1];
    return Boolean(prev && next && !prev.divider && !next.divider);
  });
});

function handleAction(key: string) {
  visible.value = false;
  emit("action", key, props.item);
}

function handleMenuClick(e: Event) {
  e.stopPropagation();
}
</script>

<template>
  <ui-dropdown class="row-actions-dropdown" :visible="visible" @popup-visible-change="visible = $event" @click="handleMenuClick">
    <button type="button" class="row-actions-trigger" @click.stop>
      <icon-more />
    </button>
    <template #content>
      <div class="menu-panel" role="menu">
        <template v-for="action in allActions" :key="action.key">
          <div v-if="action.divider" class="menu-divider" role="separator" />
          <button
            v-else
            type="button"
            class="menu-item"
            :class="{ 'menu-item--danger': action.danger }"
            :disabled="action.disabled"
            @click="handleAction(action.key)"
          >
            <span class="menu-icon-wrap">
              <component v-if="action.icon" :is="resolveIcon(action.icon)" class="menu-icon" />
            </span>
            <span class="menu-label">{{ action.label }}</span>
          </button>
        </template>
      </div>
    </template>
  </ui-dropdown>
</template>
