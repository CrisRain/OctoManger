<script setup lang="ts">
import { ref, computed, watch, useSlots } from "vue";
import {
  IconSearch, IconFilter, IconCheck, IconClose,
  IconRefresh, IconDownload, IconDelete
} from "@/lib/icons";

interface Props<T = any> {
  data: T[];
  loading?: boolean;
  selectable?: boolean;
  rowKey?: string | ((record: T) => string | number);
  search?: string;
  filters?: Record<string, any>;
  selectedKeys?: Array<string | number>;
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  selectable: true,
  rowKey: "id",
});

const emit = defineEmits<{
  (e: "refresh"): void;
  (e: "update:search", value: string): void;
  (e: "update:filters", filters: Record<string, any>): void;
  (e: "update:selectedKeys", selectedKeys: string[]): void;
  (e: "selection-change", selectedKeys: string[]): void;
  (e: "batch-delete", items: any[]): void;
  (e: "batch-export", items: any[]): void;
}>();

const slots = useSlots();

// 搜索和筛选
const searchKeyword = ref(props.search ?? "");
const showFilters = ref(false);
const activeFilters = ref<Record<string, any>>(props.filters ? { ...props.filters } : {});

watch(
  () => props.search,
  (value) => {
    if (value !== undefined && value !== searchKeyword.value) {
      searchKeyword.value = value;
    }
  }
);

watch(
  () => props.filters,
  (value) => {
    if (value !== undefined) {
      activeFilters.value = { ...value };
    }
  }
);

// 选择状态
function normalizeSelectionKeys(keys: Array<string | number> | undefined): string[] {
  return (keys ?? []).map((key) => String(key));
}

const selectedKeys = ref<string[]>(normalizeSelectionKeys(props.selectedKeys));

watch(
  () => props.selectedKeys,
  (value) => {
    if (value !== undefined) {
      selectedKeys.value = normalizeSelectionKeys(value);
    }
  }
);
const isAllSelected = computed(() => {
  return props.data.length > 0 && selectedKeys.value.length === props.data.length;
});
const isIndeterminate = computed(() => {
  return selectedKeys.value.length > 0 && selectedKeys.value.length < props.data.length;
});

// 获取行的唯一键
const getRowKey = (record: any, index: number): string => {
  if (typeof props.rowKey === "function") {
    return String(props.rowKey(record));
  }
  const value = record?.[props.rowKey];
  return value != null ? String(value) : String(index);
};

// 切换全选
function toggleSelectAll() {
  if (isAllSelected.value) {
    selectedKeys.value = [];
  } else {
    selectedKeys.value = props.data.map((item, index) => getRowKey(item, index));
  }
  emit("update:selectedKeys", selectedKeys.value);
  emit("selection-change", selectedKeys.value);
}

// 切换单行选择
function toggleRowSelection(record: any, index: number) {
  const key = getRowKey(record, index);
  const indexInSelected = selectedKeys.value.indexOf(key);

  if (indexInSelected > -1) {
    selectedKeys.value.splice(indexInSelected, 1);
  } else {
    selectedKeys.value.push(key);
  }
  emit("update:selectedKeys", selectedKeys.value);
  emit("selection-change", selectedKeys.value);
}

// 获取选中的项目
const selectedItems = computed(() => {
  return props.data.filter((item, index) =>
    selectedKeys.value.includes(getRowKey(item, index))
  );
});
const hasSelection = computed(() => selectedItems.value.length > 0);
const hasDesktopBatchActions = computed(() => Boolean(slots["batch-actions"]));
const hasMobileBatchActions = computed(() => Boolean(slots["mobile-batch-actions"]));

// 清空选择
function clearSelection() {
  selectedKeys.value = [];
  emit("update:selectedKeys", []);
  emit("selection-change", []);
}

// 搜索（防抖 300ms，避免每次击键都触发父组件过滤计算）
let _searchDebounce: ReturnType<typeof setTimeout> | null = null;
watch(searchKeyword, (value) => {
  if (_searchDebounce) clearTimeout(_searchDebounce);
  _searchDebounce = setTimeout(() => {
    emit("update:search", value);
  }, 300);
});

// activeFilters 通过 setFilters/patchFilters 整体替换，无需 deep
watch(activeFilters, (value) => {
  emit("update:filters", { ...value });
});

function setFilters(next: Record<string, any>) {
  activeFilters.value = { ...next };
  emit("update:filters", { ...activeFilters.value });
}

function patchFilters(patch: Record<string, any>) {
  setFilters({ ...activeFilters.value, ...patch });
}

// 刷新
function handleRefresh() {
  emit("refresh");
}

// 批量删除
async function handleBatchDelete() {
  emit("batch-delete", selectedItems.value);
}

// 批量导出
async function handleBatchExport() {
  emit("batch-export", selectedItems.value);
}

// 暴露方法供父组件调用
defineExpose({
  clearSelection,
  getSelectedItems: () => selectedItems.value,
});
</script>

<template>
  <div class="mb-5 space-y-3">
    <!-- 独立批量操作栏 -->
    <div
      class="panel-surface flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
      :role="hasSelection ? 'toolbar' : undefined"
      aria-label="批量操作"
    >
      <div class="flex items-start gap-3 sm:items-center">
        <div
          class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl shadow-sm"
          :class="hasSelection ? 'bg-slate-900 text-white' : 'bg-slate-100 text-slate-500'"
        >
          <icon-check class="h-4 w-4" aria-hidden="true" />
        </div>
        <div class="min-w-0">
          <div class="text-sm font-semibold text-slate-900">批量操作</div>
          <div v-if="hasSelection" class="text-xs text-slate-500">
            已选择
            <span aria-live="polite">{{ selectedItems.length }}</span>
            项，可继续执行自定义动作、导出或删除。
          </div>
          <div v-else class="text-xs text-slate-500">勾选列表项后，可在这里执行批量处理。</div>
        </div>
      </div>

      <div v-if="hasSelection" class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="inline-flex h-8 items-center rounded-lg px-3 text-sm font-medium text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50"
          @click="clearSelection"
        >
          清空选择
        </button>

        <template v-if="hasDesktopBatchActions">
          <div class="hidden items-center gap-2 lg:flex">
            <slot name="batch-actions" :selected-items="selectedItems" :clear-selection="clearSelection" />
          </div>
        </template>
        <template v-if="hasMobileBatchActions">
          <div class="flex items-center gap-2 lg:hidden">
            <slot name="mobile-batch-actions" :selected-items="selectedItems" :clear-selection="clearSelection" />
          </div>
        </template>
        <template v-else-if="hasDesktopBatchActions">
          <div class="flex items-center gap-2 lg:hidden">
            <slot name="batch-actions" :selected-items="selectedItems" :clear-selection="clearSelection" />
          </div>
        </template>

        <button
          type="button"
          aria-label="导出选中项"
          class="inline-flex h-8 items-center gap-1 rounded-lg px-3 text-sm font-medium text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50"
          @click="handleBatchExport"
        >
          <icon-download class="h-4 w-4" aria-hidden="true" />
          导出
        </button>
        <button
          type="button"
          aria-label="删除选中项"
          class="inline-flex h-8 items-center gap-1 rounded-lg bg-red-50 px-3 text-sm font-medium text-red-600 transition-colors hover:bg-red-100 hover:text-red-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400/50"
          @click="handleBatchDelete"
        >
          <icon-delete class="h-4 w-4" aria-hidden="true" />
          删除
        </button>
      </div>
      <div v-else class="text-xs font-medium text-slate-400">尚未选择项目</div>
    </div>

    <!-- 搜索和操作栏 -->
    <div class="panel-surface flex items-center gap-2 px-3 py-2.5">
      <!-- 搜索框 -->
      <icon-search class="h-4 w-4 flex-shrink-0 text-slate-500" aria-hidden="true" />
      <input
        v-model="searchKeyword"
        type="text"
        placeholder="搜索..."
        aria-label="搜索"
        class="min-w-0 flex-1 bg-transparent text-[14px] text-slate-800 placeholder:text-slate-500 outline-none"
        data-search-trigger
      />
      <!-- 清空搜索：h-5 visual + before: 扩展触控区 -->
      <button type="button"
        v-if="searchKeyword"
        aria-label="清空搜索"
        class="relative inline-flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 before:absolute before:-inset-[10px] before:content-['']"
        @click="searchKeyword = ''"
      >
        <icon-close class="h-3 w-3" aria-hidden="true" />
      </button>

      <div class="mx-1 h-5 w-px bg-slate-200 flex-shrink-0" aria-hidden="true" />

      <!-- 操作按钮组 -->
      <div class="flex flex-shrink-0 items-center gap-1">
        <button type="button"
          :aria-label="showFilters ? '收起筛选' : '展开筛选'"
          :aria-expanded="showFilters"
          aria-controls="smart-list-filter-panel"
          class="relative inline-flex h-7 w-7 cursor-pointer items-center justify-center rounded-lg text-slate-500 transition-all hover:bg-slate-100 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 before:absolute before:-inset-[8px] before:content-['']"
          :class="{ 'bg-slate-100 text-slate-700': showFilters }"
          @click="showFilters = !showFilters"
        >
          <icon-filter class="h-4 w-4" aria-hidden="true" />
        </button>

        <button type="button"
          aria-label="刷新列表"
          class="relative inline-flex h-7 w-7 cursor-pointer items-center justify-center rounded-lg text-slate-500 transition-all hover:bg-slate-100 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 before:absolute before:-inset-[8px] before:content-['']"
          @click="handleRefresh"
        >
          <icon-refresh class="h-4 w-4" aria-hidden="true" />
        </button>

        <slot name="extra-actions" />
      </div>
    </div>

    <!-- 筛选面板：用 max-height 过渡代替 v-if，避免布局位移跳动 -->
    <div
      id="smart-list-filter-panel"
      class="overflow-hidden transition-[max-height,opacity] duration-[240ms] ease-[cubic-bezier(0.4,0,0.2,1)]"
      :style="{ maxHeight: showFilters ? '300px' : '0', opacity: showFilters ? '1' : '0' }"
      :aria-hidden="!showFilters"
    >
      <div class="panel-surface mt-2 px-4 py-3">
        <slot name="filters" :filters="activeFilters" :update-filters="setFilters">
          <div class="flex flex-wrap items-center gap-3">
            <span class="text-sm font-medium text-slate-600">状态：</span>
            <div class="flex flex-wrap items-center gap-2">
              <button type="button"
                v-for="option in ['全部', '活跃', '停用']"
                :key="option"
                class="cursor-pointer rounded-lg border border-transparent bg-transparent px-3 py-1.5 text-sm font-medium text-slate-600 transition-all hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50"
                :class="{ 'border-slate-200 bg-white shadow-sm text-slate-900': activeFilters.status === option }"
                @click="patchFilters({ status: option === '全部' ? undefined : option })"
              >
                {{ option }}
              </button>
            </div>
          </div>
        </slot>
      </div>
    </div>

  </div>
</template>
