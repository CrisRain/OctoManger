<script setup lang="ts">
import { computed, useAttrs } from "vue";
import { cx } from "../utils";

interface Props {
  modelValue?: string | number | null;
  placeholder?: string;
  rows?: number;
  autoSize?: boolean | { minRows?: number; maxRows?: number };
  readonly?: boolean;
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: "",
  placeholder: "",
  rows: 3,
  autoSize: false,
  readonly: false,
  disabled: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
  change: [value: string];
  focus: [event: FocusEvent];
  blur: [event: FocusEvent];
}>();

const attrs = useAttrs();

const minRows = computed(() => {
  if (typeof props.autoSize === "object" && props.autoSize?.minRows) {
    return props.autoSize.minRows;
  }
  return props.rows;
});

const wrapperClass = computed(() =>
  cx(
    "ui-textarea-wrapper rounded-xl border border-slate-300 bg-white px-3 py-2 shadow-input transition-all",
    "focus-within:border-accent focus-within:shadow-input-focus",
    attrs.class as string,
  ),
);
</script>

<template>
  <div v-bind="{ ...attrs, class: undefined }" :class="wrapperClass">
    <textarea
      class="ui-textarea w-full resize-y border-0 bg-transparent text-sm text-slate-900 outline-none placeholder:text-slate-400"
      :value="modelValue ?? ''"
      :placeholder="placeholder"
      :rows="minRows"
      :readonly="readonly"
      :disabled="disabled"
      @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
      @change="emit('change', ($event.target as HTMLTextAreaElement).value)"
      @focus="emit('focus', $event)"
      @blur="emit('blur', $event)"
    />
  </div>
</template>
