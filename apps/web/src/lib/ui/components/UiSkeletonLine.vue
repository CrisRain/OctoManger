<script setup lang="ts">
import { useAttrs } from "vue";
import { cx } from "../utils";

interface Props {
  rows?: number;
  lineHeight?: number;
  lineSpacing?: number;
  widths?: Array<number | string>;
}

const props = withDefaults(defineProps<Props>(), {
  rows: 1,
  lineHeight: 14,
  lineSpacing: 8,
  widths: () => ["100%"],
});

const attrs = useAttrs();
</script>

<template>
  <div v-bind="{ ...attrs, class: undefined }" :class="cx('space-y-2', attrs.class as string)">
    <div
      v-for="(_, index) in props.rows"
      :key="index"
      class="ui-skeleton-line-row animate-pulse rounded bg-slate-200"
      :style="{
        height: `${props.lineHeight}px`,
        marginTop: index === 0 ? '0px' : `${props.lineSpacing}px`,
        width: String(props.widths[index] ?? props.widths[props.widths.length - 1] ?? '100%'),
      }"
    />
  </div>
</template>
