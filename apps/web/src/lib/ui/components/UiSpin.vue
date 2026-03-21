<script setup lang="ts">
import { computed, useSlots, useAttrs } from "vue";
import { cx } from "../utils";

interface Props {
  loading?: boolean;
  size?: number | string;
  tip?: string;
}

const props = withDefaults(defineProps<Props>(), {
  loading: true,
  size: "1.5em",
  tip: "",
});

const slots = useSlots();
const attrs = useAttrs();
const hasSlot = computed(() => Boolean(slots.default));
const normalizeSize = (value: number | string) =>
  typeof value === "number" ? `${value / 16}em` : String(value);

const spinnerStyle = computed(() => ({
  inlineSize: normalizeSize(props.size),
  blockSize: normalizeSize(props.size),
}));
</script>

<template>
  <!-- Wrap mode: overlay spinner over content -->
  <div v-if="hasSlot" v-bind="{ ...attrs, class: undefined }" :class="cx('relative', attrs.class as string)">
    <slot />
    <div
      v-if="loading"
      class="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-white/70 backdrop-blur-[1px]"
    >
      <span
        class="ui-spin-icon inline-block animate-spin rounded-full border-2 border-slate-200 border-t-[var(--accent)]"
        :style="spinnerStyle"
      />
      <span v-if="tip" class="text-xs text-slate-500">{{ tip }}</span>
    </div>
  </div>

  <!-- Standalone spinner -->
  <div
    v-else-if="loading"
    v-bind="{ ...attrs, class: undefined }"
    :class="cx('ui-spin inline-flex items-center gap-2', attrs.class as string)"
  >
    <span
      class="ui-spin-icon inline-block animate-spin rounded-full border-2 border-slate-200 border-t-[var(--accent)]"
      :style="spinnerStyle"
    />
    <span v-if="tip" class="text-xs text-slate-500">{{ tip }}</span>
  </div>
</template>
