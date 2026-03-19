<script setup lang="ts">
import { ref, computed, watch } from "vue";
import {
  IconClose,
  IconEdit,
  IconDelete,
  IconRefresh,
} from "@/lib/icons";

interface DrawerTab {
  key: string;
  label: string;
  icon?: any;
}

interface Props {
  open: boolean;
  title?: string;
  width?: number | string;
  loading?: boolean;
  showActions?: boolean;
  tabs?: DrawerTab[];
  tab?: string;
}

const props = withDefaults(defineProps<Props>(), {
  open: false,
  width: 640,
  loading: false,
  showActions: true,
});

const emit = defineEmits<{
  (e: "update:open", value: boolean): void;
  (e: "update:tab", value: string): void;
  (e: "close"): void;
  (e: "edit"): void;
  (e: "delete"): void;
  (e: "refresh"): void;
}>();

const innerOpen = computed({
  get: () => props.open,
  set: (val) => emit("update:open", val),
});

const defaultTabs: DrawerTab[] = [
  { key: "detail", label: "详细信息" },
  { key: "activity", label: "活动记录" },
  { key: "settings", label: "设置" },
];

const resolvedTabs = computed(() =>
  props.tabs && props.tabs.length ? props.tabs : defaultTabs
);

const currentTab = ref(props.tab ?? resolvedTabs.value[0]?.key ?? "detail");

watch(
  () => props.tab,
  (val) => {
    if (val !== undefined && val !== currentTab.value) {
      currentTab.value = val;
    }
  }
);

watch(resolvedTabs, (tabs) => {
  if (!tabs.length) return;
  if (!tabs.find((tab) => tab.key === currentTab.value)) {
    currentTab.value = tabs[0].key;
    emit("update:tab", currentTab.value);
  }
});

function setTab(key: string) {
  currentTab.value = key;
  emit("update:tab", key);
}

function handleClose() {
  emit("close");
  innerOpen.value = false;
}

function handleEdit() {
  emit("edit");
}

function handleDelete() {
  emit("delete");
}

function handleRefresh() {
  emit("refresh");
}
</script>

<template>
  <ui-drawer
    :visible="innerOpen"
    :width="width"
    :footer="false"
    :header="false"
    :closable="false"
    placement="right"
    class="detail-drawer"
    @cancel="handleClose"
  >
    <!-- Header -->
    <div class="drawer-header">
      <div class="header-left">
        <button type="button" class="icon-btn icon-btn--ghost" @click="handleClose">
          <icon-close />
        </button>
        <div class="header-title">
          <h2>{{ title || "详情" }}</h2>
          <p v-if="$slots.subtitle" class="header-subtitle">
            <slot name="subtitle" />
          </p>
        </div>
      </div>

      <div v-if="showActions" class="header-actions">
        <button type="button" class="icon-btn" @click="handleRefresh" :disabled="loading">
          <icon-refresh />
        </button>
        <button type="button" class="icon-btn" @click="handleEdit">
          <icon-edit />
        </button>
        <button type="button" class="icon-btn icon-btn--danger" @click="handleDelete">
          <icon-delete />
        </button>
      </div>
    </div>

    <!-- Tabs -->
    <div v-if="resolvedTabs.length" class="drawer-tabs">
      <button type="button"
        v-for="tab in resolvedTabs"
        :key="tab.key"
        class="drawer-tab"
        :class="{ 'drawer-tab--active': currentTab === tab.key }"
        @click="setTab(tab.key)"
      >
        <component v-if="tab.icon" :is="tab.icon" class="drawer-tab-icon" />
        {{ tab.label }}
      </button>
    </div>

    <!-- Body -->
    <div class="drawer-body">
      <ui-spin v-if="loading" :loading="true" tip="加载中..." class="drawer-loading" />

      <div
        v-for="tab in resolvedTabs"
        v-show="currentTab === tab.key && !loading"
        :key="tab.key"
        class="drawer-panel"
      >
        <slot :name="tab.key">
          <ui-empty
            v-if="tab.key === 'activity'"
            description="暂无活动记录"
          />
          <ui-empty
            v-else-if="tab.key === 'settings'"
            description="暂无可设置项"
          />
          <div v-else class="drawer-panel-empty">暂无内容</div>
        </slot>
      </div>
    </div>

    <!-- Footer -->
    <div v-if="$slots.footer" class="drawer-footer">
      <slot name="footer" />
    </div>
  </ui-drawer>
</template>
