<script setup lang="ts">
import { ref, computed, watch } from "vue";
import {
  IconSearch, IconFilter, IconCheck, IconClose,
  IconRefresh, IconDownload, IconDelete
} from "@/lib/icons";

interface Props<T = any> {
  data: T[];
  loading?: boolean;
  selectable?: boolean;
  rowKey?: string | ((record: T) => string);
  search?: string;
  filters?: Record<string, any>;
  selectedKeys?: string[];
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
  },
  { deep: true }
);

// 选择状态
const selectedKeys = ref<string[]>(props.selectedKeys ? [...props.selectedKeys] : []);

watch(
  () => props.selectedKeys,
  (value) => {
    if (value !== undefined) {
      selectedKeys.value = [...value];
    }
  },
  { deep: true }
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
    return props.rowKey(record);
  }
  return record[props.rowKey] || String(index);
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

// 清空选择
function clearSelection() {
  selectedKeys.value = [];
  emit("update:selectedKeys", []);
  emit("selection-change", []);
}

// 搜索
watch(searchKeyword, (value) => {
  emit("update:search", value);
});

watch(
  activeFilters,
  (value) => {
    emit("update:filters", { ...value });
  },
  { deep: true }
);

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
  <div class="mb-5">
    <!-- 搜索和操作栏 -->
    <div class="panel-surface flex items-center gap-2 px-3 py-2.5">
      <!-- 搜索框 -->
      <icon-search class="h-4 w-4 flex-shrink-0 text-slate-400" />
      <input
        v-model="searchKeyword"
        type="text"
        placeholder="搜索..."
        class="min-w-0 flex-1 bg-transparent text-[14px] text-slate-800 placeholder:text-slate-400 outline-none"
        data-search-trigger
      />
      <button type="button"
        v-if="searchKeyword"
        class="inline-flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
        @click="searchKeyword = ''"
      >
        <icon-close class="h-3 w-3" />
      </button>

      <div class="mx-1 h-5 w-px bg-slate-200 flex-shrink-0" />

      <!-- 操作按钮组 -->
      <div class="flex flex-shrink-0 items-center gap-1">
        <!-- 选中时的批量操作 -->
        <template v-if="selectedKeys.length > 0">
          <span class="mr-1 text-xs font-medium text-slate-600">{{ selectedKeys.length }} 已选</span>
          <button type="button" class="rounded-md px-2 py-1 text-xs text-slate-500 hover:text-slate-900 transition-colors" @click="clearSelection">清空</button>
          <div class="h-4 w-px bg-slate-200 mx-0.5" />
          <button type="button" class="inline-flex h-7 w-7 cursor-pointer items-center justify-center rounded-lg text-slate-500 transition-all hover:bg-slate-100 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20" title="导出" @click="handleBatchExport">
            <icon-download class="h-4 w-4" />
          </button>
          <button type="button" class="inline-flex h-7 w-7 cursor-pointer items-center justify-center rounded-lg text-red-500 transition-all hover:bg-red-50 hover:text-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400/20" title="删除" @click="handleBatchDelete">
            <icon-delete class="h-4 w-4" />
          </button>
        </template>

        <!-- 常规操作 -->
        <template v-else>
          <button type="button"
            class="inline-flex h-7 w-7 cursor-pointer items-center justify-center rounded-lg text-slate-500 transition-all hover:bg-slate-100 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20"
            :class="{ 'bg-slate-100 text-slate-700': showFilters }"
            :title="showFilters ? '收起筛选' : '展开筛选'"
            @click="showFilters = !showFilters"
          >
            <icon-filter class="h-4 w-4" />
          </button>

          <button type="button" class="inline-flex h-7 w-7 cursor-pointer items-center justify-center rounded-lg text-slate-500 transition-all hover:bg-slate-100 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20" title="刷新" @click="handleRefresh">
            <icon-refresh class="h-4 w-4" />
          </button>

          <slot name="extra-actions" />
        </template>
      </div>
    </div>

    <!-- 筛选面板 -->
    <div v-if="showFilters" class="panel-surface mt-2 px-4 py-3">
      <slot name="filters" :filters="activeFilters" :update-filters="setFilters">
        <div class="flex flex-wrap items-center gap-3">
          <span class="text-sm font-medium text-slate-600">状态：</span>
          <div class="flex flex-wrap items-center gap-1.5">
            <button type="button"
              v-for="option in ['全部', '活跃', '停用']"
              :key="option"
              class="cursor-pointer rounded-lg border border-transparent bg-transparent px-3 py-1.5 text-sm font-medium text-slate-600 transition-all hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20"
              :class="{ 'border-slate-200 bg-white shadow-sm text-slate-900': activeFilters.status === option }"
              @click="patchFilters({ status: option === '全部' ? undefined : option })"
            >
              {{ option }}
            </button>
          </div>
        </div>
      </slot>
    </div>

    <!-- 批量操作栏 (移动端) -->
    <div v-if="selectedKeys.length > 0" class="mt-2 flex items-center justify-between rounded-xl bg-slate-900 px-4 py-3 text-white shadow-md lg:hidden">
      <div class="flex items-center gap-2 text-sm font-medium">
        <icon-check class="h-4 w-4" />
        <span>已选择 {{ selectedKeys.length }} 项</span>
      </div>
      <div class="flex items-center gap-2">
        <button type="button" class="cursor-pointer rounded-lg border-0 bg-white/20 px-3 py-1.5 text-sm font-medium transition-colors hover:bg-white/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20" @click="clearSelection">取消</button>
        <button type="button" class="cursor-pointer rounded-lg border-0 bg-red-500 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20" @click="handleBatchDelete">删除</button>
      </div>
    </div>
  </div>
</template>
