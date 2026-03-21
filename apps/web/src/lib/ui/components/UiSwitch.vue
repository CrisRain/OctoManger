<script setup lang="ts">
import { computed, useAttrs } from "vue";
import { cx } from "../utils";

interface Props {
  modelValue?: boolean;
  disabled?: boolean;
  checkedText?: string;
  uncheckedText?: string;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: false,
  disabled: false,
  checkedText: "",
  uncheckedText: "",
});

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  change: [value: boolean];
}>();

const attrs = useAttrs();

const classes = computed(() =>
  cx(
    "ui-switch inline-flex h-6 min-w-11 items-center rounded-full border px-0.5 transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/30 focus-visible:ring-offset-2 focus-visible:ring-offset-white",
    props.modelValue
      ? "ui-switch-checked border-[var(--accent)] bg-[var(--accent)]"
      : "border-slate-300 bg-slate-200",
    props.disabled && "cursor-not-allowed opacity-60",
    attrs.class as string,
  ),
);

function handleClick(event: MouseEvent) {
  if (props.disabled) {
    event.preventDefault();
    return;
  }
  const next = !props.modelValue;
  emit("update:modelValue", next);
  emit("change", next);
}
</script>

<template>
  <button
    v-bind="{ ...attrs, class: undefined }"
    type="button"
    :class="classes"
    @click="handleClick"
  >
    <span
      :class="cx('h-5 w-5 rounded-full bg-white shadow-sm transition-transform duration-200', modelValue ? 'translate-x-5' : 'translate-x-0')"
    />
    <span
      v-if="checkedText || uncheckedText"
      :class="cx('ml-1 mr-1 text-[11px] font-semibold', modelValue ? 'text-white' : 'text-slate-500')"
    >
      {{ modelValue ? checkedText : uncheckedText }}
    </span>
  </button>
</template>
