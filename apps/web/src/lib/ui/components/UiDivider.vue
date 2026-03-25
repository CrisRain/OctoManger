<script setup lang="ts">
import { computed, useSlots, useAttrs } from "vue";
import { cx } from "../utils";

interface Props {
  orientation?: "left" | "center" | "right";
}

const props = withDefaults(defineProps<Props>(), { orientation: "center" });
const slots = useSlots();
const attrs = useAttrs();

const hasContent = computed(() => Boolean(slots.default?.()?.length));
const alignClass = computed(() => {
  if (props.orientation === "left") return "justify-start";
  if (props.orientation === "right") return "justify-end";
  return "justify-center";
});
</script>

<template>
  <hr
    v-if="!hasContent"
    v-bind="{ ...attrs, class: undefined }"
    :class="cx('ui-divider my-4 border-0 border-t border-slate-200', attrs.class as string)"
  />
  <div
    v-else
    v-bind="{ ...attrs, class: undefined }"
    :class="cx('ui-divider my-4 flex items-center gap-3', alignClass, attrs.class as string)"
  >
    <span class="h-px flex-1 bg-slate-200" />
    <span class="text-xs font-semibold uppercase tracking-wider text-slate-500">
      <slot />
    </span>
    <span class="h-px flex-1 bg-slate-200" />
  </div>
</template>
