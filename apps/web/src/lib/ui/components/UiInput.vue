<script setup lang="ts">
import { computed, useAttrs } from "vue";
import { cx } from "../utils";

interface Props {
  modelValue?: string | number | null;
  type?: string;
  placeholder?: string;
  disabled?: boolean;
  allowClear?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: "",
  type: "text",
  placeholder: "",
  disabled: false,
  allowClear: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
  change: [value: string];
  focus: [event: FocusEvent];
  blur: [event: FocusEvent];
  clear: [];
}>();

const attrs = useAttrs();

const wrapperClass = computed(() =>
  cx(
    "ui-input-wrapper flex items-center gap-2 rounded-xl border border-slate-300 bg-white px-3 shadow-input transition-all",
    "focus-within:border-accent focus-within:shadow-input-focus",
    props.disabled && "bg-slate-50 opacity-70",
    attrs.class as string,
  ),
);

const showClear = computed(
  () => props.allowClear && String(props.modelValue ?? "").length > 0 && !props.disabled,
);

function onInput(event: Event) {
  emit("update:modelValue", (event.target as HTMLInputElement).value);
}

function onChange(event: Event) {
  emit("change", (event.target as HTMLInputElement).value);
}

function onClear(event: MouseEvent) {
  event.preventDefault();
  event.stopPropagation();
  emit("update:modelValue", "");
  emit("clear");
}
</script>

<template>
  <div v-bind="{ ...attrs, class: undefined }" :class="wrapperClass">
    <slot name="prefix" />
    <input
      class="ui-input h-10 w-full border-0 bg-transparent px-0 text-sm text-slate-900 outline-none placeholder:text-slate-400"
      :value="modelValue ?? ''"
      :type="type"
      :placeholder="placeholder"
      :disabled="disabled"
      @input="onInput"
      @change="onChange"
      @focus="emit('focus', $event)"
      @blur="emit('blur', $event)"
    />
    <button
      v-if="showClear"
      type="button"
      class="text-slate-400 transition hover:text-slate-600"
      @click="onClear"
    >
      ×
    </button>
    <slot name="suffix" />
  </div>
</template>
