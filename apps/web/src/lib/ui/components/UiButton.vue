<script setup lang="ts">
import { computed, useAttrs } from "vue";
import { cx, BUTTON_SIZE_CLASS, BUTTON_SIZE_MARKER_CLASS, TONE_CLASS } from "../utils";

interface Props {
  type?: "primary" | "secondary" | "text" | "outline";
  size?: "mini" | "small" | "medium" | "large";
  status?: "normal" | "danger" | "warning" | "success";
  disabled?: boolean;
  loading?: boolean;
  htmlType?: "button" | "submit" | "reset";
}

const props = withDefaults(defineProps<Props>(), {
  type: "secondary",
  size: "medium",
  status: "normal",
  disabled: false,
  loading: false,
  htmlType: "button",
});

const emit = defineEmits<{ click: [event: MouseEvent] }>();
const attrs = useAttrs();

const isDisabled = computed(() => props.disabled || props.loading);

const tone = computed(() => {
  if (props.status === "danger") return TONE_CLASS.error;
  if (props.status === "warning") return TONE_CLASS.warning;
  if (props.status === "success") return TONE_CLASS.success;
  return TONE_CLASS.default;
});

const typeClass = computed(() => {
  if (props.type === "primary")
    return "ui-btn-primary btn-primary-gradient border-transparent text-white shadow-btn-primary hover:shadow-btn-primary-hover hover:btn-primary-gradient-hover active:translate-y-0 hover:-translate-y-px";
  if (props.type === "text")
    return "border-transparent bg-transparent hover:bg-slate-100";
  if (props.type === "outline")
    return "border-slate-300 bg-white hover:border-accent hover:text-accent";
  return "ui-btn-secondary border-slate-300 bg-white shadow-[0_6px_14px_rgba(15,23,42,0.04)] hover:bg-slate-50 hover:border-accent/25";
});

const classes = computed(() =>
  cx(
    "ui-btn inline-flex items-center justify-center gap-2 rounded-xl border font-semibold transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/30 focus-visible:ring-offset-1",
    BUTTON_SIZE_CLASS[props.size] ?? BUTTON_SIZE_CLASS.medium,
    BUTTON_SIZE_MARKER_CLASS[props.size] ?? "",
    props.type === "primary" ? "" : tone.value,
    typeClass.value,
    isDisabled.value && "cursor-not-allowed opacity-60",
    attrs.class as string,
  ),
);

function handleClick(event: MouseEvent) {
  if (isDisabled.value) {
    event.preventDefault();
    return;
  }
  emit("click", event);
}
</script>

<template>
  <button
    v-bind="{ ...attrs, class: undefined }"
    :type="((attrs.type as string) ?? htmlType) as 'button' | 'submit' | 'reset'"
    :disabled="isDisabled"
    :class="classes"
    @click="handleClick"
  >
    <span
      v-if="loading"
      class="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
    />
    <slot v-else name="icon" />
    <slot />
  </button>
</template>
