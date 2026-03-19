<script setup lang="ts">
import { computed, useAttrs } from "vue";
import { cx } from "../utils";

interface Props {
  size?: string | number;
  direction?: "horizontal" | "vertical";
  wrap?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  size: 8,
  direction: "horizontal",
  wrap: true,
});

const attrs = useAttrs();

const classes = computed(() =>
  cx(
    "ui-space inline-flex",
    props.direction === "vertical" ? "ui-space-vertical flex-col" : "flex-row",
    props.wrap && props.direction !== "vertical" && "flex-wrap",
    attrs.class as string,
  ),
);

const gapStyle = computed(() => ({
  ...(attrs.style as Record<string, unknown>),
  gap: typeof props.size === "number" ? `${props.size}px` : String(props.size),
}));
</script>

<template>
  <div v-bind="{ ...attrs, class: undefined, style: undefined }" :class="classes" :style="gapStyle">
    <slot />
  </div>
</template>
