<script setup lang="ts">
import { defineComponent, h, isVNode, type PropType } from "vue";
import StatusTag from "./StatusTag.vue";
import { UiSpace, UiButton } from "@/lib/ui";

interface Column {
  key: string;
  title: string;
  width?: number;
  render?: (record: any) => any;
  format?: "status" | "date" | "datetime" | "relative" | "actions";
}

interface Props {
  data: any[];
  columns: Column[];
  loading?: boolean;
  empty?: {
    type?: "empty" | "error";
    title?: string;
    description?: string;
    actionText?: string;
  };
  showIndex?: boolean;
  rowKey?: string | ((record: any) => string);
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  showIndex: false,
  rowKey: "id",
});

const emit = defineEmits<{
  (e: "refresh"): void;
  (e: "row-action", action: string, record: any): void;
  (e: "row-click", record: any): void;
}>();

const RenderContent = defineComponent({
  name: "RenderContent",
  props: {
    content: {
      type: [String, Number, Boolean, Object] as PropType<unknown>,
      default: "",
    },
  },
  setup(props) {
    return () => {
      if (isVNode(props.content)) {
        return props.content;
      }
      return props.content == null ? "-" : String(props.content);
    };
  },
});

/**
 * 获取行唯一键
 */
const getRowKey = (record: any, index: number): string => {
  if (typeof props.rowKey === "function") {
    return props.rowKey(record);
  }
  return record[props.rowKey] || String(index);
};

/**
 * 格式化单元格值 - 返回 VNode 或字符串
 */
const formatCellValue = (record: any, column: Column) => {
  const value = record[column.key];

  if (column.render) {
    return column.render(record);
  }

  switch (column.format) {
    case "status":
      return h(StatusTag, { status: value });

    case "date":
      return formatDate(value, "date");

    case "datetime":
      return formatDate(value, "full");

    case "relative":
      return formatDate(value, "relative");

    case "actions":
      return h(UiSpace, {}, () => [
        h(UiButton, {
          type: "text",
          size: "small",
          onClick: (event: MouseEvent) => {
            event.stopPropagation();
            emit("row-action", "edit", record);
          },
        }, () => "编辑"),
        h(UiButton, {
          type: "text",
          size: "small",
          status: "danger",
          onClick: (event: MouseEvent) => {
            event.stopPropagation();
            emit("row-action", "delete", record);
          },
        }, () => "删除")
      ]);

    default:
      return value ?? "-";
  }
};

/**
 * 格式化日期
 */
function formatDate(dateStr: string, format: "full" | "date" | "relative") {
  if (!dateStr) return "-";

  const date = new Date(dateStr);
  if (isNaN(date.getTime())) return "-";

  switch (format) {
    case "date":
      return date.toLocaleDateString("zh-CN");
    case "relative": {
      const now = new Date();
      const diff = now.getTime() - date.getTime();
      const seconds = Math.floor(diff / 1000);
      const minutes = Math.floor(seconds / 60);
      const hours = Math.floor(minutes / 60);
      const days = Math.floor(hours / 24);

      if (seconds < 60) return "刚刚";
      if (minutes < 60) return `${minutes}分钟前`;
      if (hours < 24) return `${hours}小时前`;
      if (days < 30) return `${days}天前`;
      return date.toLocaleDateString("zh-CN");
    }
    default:
      return date.toLocaleString("zh-CN");
  }
}
</script>

<template>
  <div class="simple-table-wrapper">
    <!-- 表格工具栏 -->
    <div v-if="$slots.toolbar" class="table-toolbar">
      <slot name="toolbar" />
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="table-loading">
      <ui-spin :loading="true" tip="加载中..." />
    </div>

    <!-- 空状态 -->
    <EmptyState
      v-else-if="!data.length"
      :type="empty?.type || 'empty'"
      :title="empty?.title"
      :description="empty?.description"
      :action-text="empty?.actionText"
      @action="emit('refresh')"
    >
      <template v-if="$slots.empty" #empty>
        <slot name="empty" />
      </template>
    </EmptyState>

    <!-- 表格 -->
    <div v-else class="simple-table-container">
      <table class="simple-table">
        <thead>
          <tr>
            <th v-if="showIndex" class="index-column">#</th>
            <th
              v-for="column in columns"
              :key="column.key"
              :style="{ width: column.width ? `${column.width}px` : undefined }"
            >
              {{ column.title }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(record, index) in data"
            :key="getRowKey(record, index)"
            class="table-row"
            @click="$emit('row-click', record)"
          >
            <td v-if="showIndex" class="index-column">{{ index + 1 }}</td>
            <td
              v-for="column in columns"
              :key="column.key"
              :class="`column-${column.key}`"
            >
              <RenderContent :content="formatCellValue(record, column)" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
