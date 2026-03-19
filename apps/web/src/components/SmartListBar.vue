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
  <div class="smart-list-bar">
    <!-- 搜索和操作栏 -->
    <div class="list-toolbar">
      <!-- 搜索框 -->
      <div class="search-box">
        <icon-search class="search-icon" />
        <input
          v-model="searchKeyword"
          type="text"
          placeholder="搜索..."
          class="search-input"
          data-search-trigger
        />
        <button type="button"
          v-if="searchKeyword"
          class="search-clear"
          @click="searchKeyword = ''"
        >
          <icon-close />
        </button>
      </div>

      <!-- 操作按钮组 -->
      <div class="toolbar-actions">
        <!-- 选中时的批量操作 -->
        <template v-if="selectedKeys.length > 0">
          <div class="selection-info">
            <span class="selection-count">{{ selectedKeys.length }}</span>
            <span class="selection-text">已选中</span>
            <button type="button" class="selection-clear" @click="clearSelection">
              清空
            </button>
          </div>

          <div class="toolbar-divider" />

          <button type="button" class="toolbar-btn" @click="handleBatchExport">
            <icon-download />
            <span>导出</span>
          </button>

          <button type="button" class="toolbar-btn toolbar-btn--danger" @click="handleBatchDelete">
            <icon-delete />
            <span>删除</span>
          </button>
        </template>

        <!-- 常规操作 -->
        <template v-else>
          <button type="button"
            class="toolbar-btn"
            :class="{ 'toolbar-btn--active': showFilters }"
            @click="showFilters = !showFilters"
          >
            <icon-filter />
            <span>筛选</span>
          </button>

          <button type="button" class="toolbar-btn" @click="handleRefresh">
            <icon-refresh />
            <span>刷新</span>
          </button>

          <slot name="extra-actions" />
        </template>
      </div>
    </div>

    <!-- 筛选面板 -->
    <div v-if="showFilters" class="filter-panel">
      <slot name="filters" :filters="activeFilters" :update-filters="setFilters">
        <div class="filter-group">
          <span class="filter-label">状态：</span>
          <div class="filter-options">
            <button type="button"
              v-for="option in ['全部', '活跃', '停用']"
              :key="option"
              class="filter-option"
              :class="{ 'filter-option--active': activeFilters.status === option }"
              @click="patchFilters({ status: option === '全部' ? undefined : option })"
            >
              {{ option }}
            </button>
          </div>
        </div>
      </slot>
    </div>

    <!-- 批量操作栏 (移动端) -->
    <div v-if="selectedKeys.length > 0" class="batch-bar-mobile">
      <div class="batch-info">
        <icon-check class="batch-icon" />
        <span>已选择 {{ selectedKeys.length }} 项</span>
      </div>
      <div class="batch-actions">
        <button type="button" class="batch-btn" @click="clearSelection">取消</button>
        <button type="button" class="batch-btn batch-btn--danger" @click="handleBatchDelete">删除</button>
      </div>
    </div>
  </div>
</template>
