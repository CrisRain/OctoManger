<script setup lang="ts">
import { computed, useAttrs } from "vue";
import { cx } from "../utils";

interface Props {
  animation?: boolean;
}

const props = withDefaults(defineProps<Props>(), { animation: true });
const attrs = useAttrs();

const classes = computed(() =>
  cx("ui-skeleton", props.animation && "animate-pulse", attrs.class as string),
);
</script>

<template>
  <div v-bind="{ ...attrs, class: undefined }" :class="classes">
    <slot />
  </div>
</template>
