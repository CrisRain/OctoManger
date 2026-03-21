<script setup lang="ts" generic="T extends Record<string, unknown>">
import EmptyState from "./EmptyState.vue";

interface EmptyConfig {
  type?: "empty" | "error" | "success" | "loading";
  title?: string;
  description?: string;
  actionText?: string;
  hideAction?: boolean;
}

/**
 * DataTable — table container with shared styling and a unified state layer.
 */
defineProps<{
  data: T[];
  loading?: boolean;
  rowKey?: string;
  empty?: EmptyConfig;
  /** How many skeleton rows to show while loading */
  skeletonRows?: number;
}>();

const emit = defineEmits<{
  (e: "empty-action"): void;
}>();
</script>

<template>
  <div class="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
    <!-- Loading skeleton -->
    <div v-if="loading" class="flex flex-col divide-y divide-slate-100 bg-white/30">
      <div v-for="i in (skeletonRows ?? 6)" :key="i" class="px-4 py-3">
        <ui-skeleton :animation="true">
          <ui-skeleton-line :rows="1" line-height="1.125em" :line-spacing="0" :widths="['100%']" />
        </ui-skeleton>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else-if="!data.length" class="bg-white/30 py-16">
      <slot name="empty">
        <EmptyState
          :type="empty?.type"
          :title="empty?.title"
          :description="empty?.description"
          :action-text="empty?.actionText"
          :hide-action="empty?.hideAction"
          @action="emit('empty-action')"
        />
      </slot>
    </div>

    <!-- Table -->
    <ui-table
      v-else
      :data="data"
      :pagination="false"
      :bordered="false"
      :row-key="rowKey ?? 'id'"
      class="premium-table data-table"
    >
      <template v-if="$slots.columns" #columns>
        <slot name="columns" />
      </template>
      <template v-if="$slots.default" #default>
        <slot />
      </template>
    </ui-table>
  </div>
</template>
