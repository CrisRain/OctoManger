<script setup lang="ts">
import { useAttrs } from "vue";
import { cx } from "../utils";

interface Props {
  description?: string;
}

const props = withDefaults(defineProps<Props>(), { description: "暂无数据" });
const attrs = useAttrs();
</script>

<template>
  <div
    v-bind="{ ...attrs, class: undefined }"
    :class="cx('ui-empty flex flex-col items-center gap-3 py-10 text-center', attrs.class as string)"
  >
    <div class="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 text-slate-400">
      <span class="text-xl">∅</span>
    </div>
    <p class="text-sm text-slate-500">{{ description }}</p>
    <slot />
  </div>
</template>
