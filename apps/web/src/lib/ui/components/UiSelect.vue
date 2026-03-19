<script setup lang="ts">
import { computed, useAttrs } from "vue";
import { cx, optionValue } from "../utils";

interface Props {
  modelValue?: unknown;
  placeholder?: string;
  disabled?: boolean;
  allowClear?: boolean;
  multiple?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: undefined,
  placeholder: "",
  disabled: false,
  allowClear: false,
  multiple: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: unknown];
  change: [value: unknown];
  focus: [event: FocusEvent];
  blur: [event: FocusEvent];
  clear: [];
}>();

const attrs = useAttrs();

const wrapperClass = computed(() =>
  cx(
    "ui-select-view relative flex items-center rounded-xl border border-slate-300 bg-white px-3 shadow-input transition-all",
    "focus-within:border-accent focus-within:shadow-input-focus",
    !props.multiple && "ui-select-view-single",
    props.disabled && "bg-slate-50 opacity-70",
    attrs.class as string,
  ),
);

const showClear = computed(() => props.allowClear && !props.multiple && props.modelValue);

function onChange(event: Event) {
  const target = event.target as HTMLSelectElement;
  if (props.multiple) {
    const values = Array.from(target.selectedOptions).map(optionValue);
    emit("update:modelValue", values);
    emit("change", values);
    return;
  }
  const selected = target.options[target.selectedIndex];
  const next = selected ? optionValue(selected) : target.value;
  emit("update:modelValue", next);
  emit("change", next);
}

function onClear(event: MouseEvent) {
  event.preventDefault();
  event.stopPropagation();
  emit("update:modelValue", "");
  emit("change", "");
  emit("clear");
}
</script>

<template>
  <div v-bind="{ ...attrs, class: undefined }" :class="wrapperClass">
    <select
      class="h-10 w-full appearance-none border-0 bg-transparent pr-7 text-sm text-slate-900 outline-none"
      :value="(modelValue ?? '') as never"
      :disabled="disabled"
      :multiple="multiple"
      @change="onChange"
      @focus="emit('focus', $event)"
      @blur="emit('blur', $event)"
    >
      <option
        v-if="placeholder && !multiple"
        value=""
        disabled
        :selected="modelValue === '' || modelValue == null"
      >
        {{ placeholder }}
      </option>
      <slot />
    </select>

    <button
      v-if="showClear"
      type="button"
      class="mr-1 text-slate-400 transition hover:text-slate-600"
      @click="onClear"
    >
      ×
    </button>
    <span v-else class="pointer-events-none absolute right-3 text-slate-400">▾</span>
  </div>
</template>
