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
  loading?: boolean;
  showActions?: boolean;
  tabs?: DrawerTab[];
  tab?: string;
}

const props = withDefaults(defineProps<Props>(), {
  open: false,
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
    :footer="false"
    :header="false"
    :closable="false"
    placement="right"
    class="detail-drawer"
    @cancel="handleClose"
  >
    <!-- Header -->
    <div class="flex flex-shrink-0 items-center justify-between border-b border-slate-200 bg-slate-50 px-6 py-5">
      <div class="flex min-w-0 flex-1 items-center gap-2.5">
        <button type="button" class="inline-flex h-9 w-9 items-center justify-center rounded-lg text-slate-400 transition-all hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20" @click="handleClose">
          <icon-close />
        </button>
        <div class="flex flex-col gap-0.5">
          <h2 class="m-0 text-[18px] font-semibold tracking-[-0.03em] text-slate-900">{{ title || "详情" }}</h2>
          <p v-if="$slots.subtitle" class="text-xs text-slate-500">
            <slot name="subtitle" />
          </p>
        </div>
      </div>

      <div v-if="showActions" class="flex flex-shrink-0 flex-wrap items-center justify-end gap-2.5 max-md:w-full max-md:justify-start">
        <button type="button" class="inline-flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 transition-all hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20" @click="handleRefresh" :disabled="loading">
          <icon-refresh />
        </button>
        <button type="button" class="inline-flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 transition-all hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20" @click="handleEdit">
          <icon-edit />
        </button>
        <button type="button" class="inline-flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 transition-all hover:bg-red-50 hover:text-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400/20" @click="handleDelete">
          <icon-delete />
        </button>
      </div>
    </div>

    <!-- Tabs -->
    <div v-if="resolvedTabs.length" class="flex flex-shrink-0 items-center gap-1 border-b border-slate-200 px-4 py-2 bg-slate-50/50">
      <button type="button"
        v-for="tab in resolvedTabs"
        :key="tab.key"
        class="flex cursor-pointer items-center gap-1.5 rounded-lg border border-transparent bg-transparent px-4 py-2 text-[14px] font-medium text-slate-500 transition-all hover:bg-slate-100 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/20"
        :class="{ 'bg-white shadow-sm border-slate-200 text-[var(--accent)] font-semibold': currentTab === tab.key }"
        @click="setTab(tab.key)"
      >
        <component v-if="tab.icon" :is="tab.icon" class="h-3.5 w-3.5" />
        {{ tab.label }}
      </button>
    </div>

    <!-- Body -->
    <div class="flex-1 overflow-y-auto p-6">
      <ui-spin v-if="loading" :loading="true" tip="加载中..." class="flex items-center justify-center py-12" />

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
          <div v-else class="py-8 text-center text-sm text-slate-500">暂无内容</div>
        </slot>
      </div>
    </div>

    <!-- Footer -->
    <div v-if="$slots.footer" class="flex flex-shrink-0 items-center justify-end gap-2 border-t border-slate-200 bg-white/[58%] px-5 py-4">
      <slot name="footer" />
    </div>
  </ui-drawer>
</template>
