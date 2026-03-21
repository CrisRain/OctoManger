<script setup lang="ts">
import { useAttrs } from "vue";
import { cx } from "../utils";

interface Props {
  rows?: number;
  lineHeight?: number | string;
  lineSpacing?: number | string;
  widths?: Array<number | string>;
}

const props = withDefaults(defineProps<Props>(), {
  rows: 1,
  lineHeight: "0.875em",
  lineSpacing: "0.5em",
  widths: () => ["100%"],
});

const attrs = useAttrs();
const normalizeLength = (value: number | string) =>
  typeof value === "number" ? `${value / 16}em` : String(value);
</script>

<template>
  <div v-bind="{ ...attrs, class: undefined }" :class="cx('space-y-2', attrs.class as string)">
    <div
      v-for="(_, index) in props.rows"
      :key="index"
      class="ui-skeleton-line-row animate-pulse rounded bg-slate-200"
      :style="{
        blockSize: normalizeLength(props.lineHeight),
        marginTop: index === 0 ? '0' : normalizeLength(props.lineSpacing),
        width: String(props.widths[index] ?? props.widths[props.widths.length - 1] ?? '100%'),
      }"
    />
  </div>
</template>
