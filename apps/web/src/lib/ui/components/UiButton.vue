<script setup lang="ts">
import { computed, useAttrs } from "vue";
import { cx, BUTTON_SIZE_CLASS, BUTTON_SIZE_MARKER_CLASS } from "../utils";

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

const typeClass = computed(() => {
  // Status overrides apply to every non-primary button
  if (props.type !== "primary") {
    if (props.status === "danger")
      return "border-rose-600 bg-rose-600 text-white shadow-sm hover:bg-rose-700 hover:border-rose-700 active:scale-[0.98]";
    if (props.status === "warning")
      return "border-amber-500 bg-amber-500 text-white shadow-sm hover:bg-amber-600 hover:border-amber-600 active:scale-[0.98]";
    if (props.status === "success")
      return "border-emerald-600 bg-emerald-600 text-white shadow-sm hover:bg-emerald-700 hover:border-emerald-700 active:scale-[0.98]";
  }

  if (props.type === "primary")
    return "ui-btn-primary border-transparent bg-[var(--accent)] text-white shadow-sm hover:bg-[var(--accent-hover)] active:bg-[var(--accent-active)] active:scale-[0.98]";
  if (props.type === "text")
    return "border-transparent bg-slate-100 text-slate-700 hover:bg-slate-200 hover:text-slate-900 active:scale-[0.98]";
  if (props.type === "outline")
    return "border-[var(--accent)]/40 bg-white text-[var(--accent)] shadow-sm hover:border-[var(--accent)]/70 hover:bg-[var(--accent)]/5 active:scale-[0.98]";
  // secondary (default)
  return "ui-btn-secondary border-slate-200 bg-slate-100 text-slate-800 shadow-sm hover:bg-slate-200 hover:border-slate-300 active:scale-[0.98]";
});

const classes = computed(() =>
  cx(
    "ui-btn inline-flex items-center justify-center gap-2 rounded-lg border font-medium transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/30 focus-visible:ring-offset-1 focus-visible:ring-offset-white",
    BUTTON_SIZE_CLASS[props.size] ?? BUTTON_SIZE_CLASS.medium,
    BUTTON_SIZE_MARKER_CLASS[props.size] ?? "",
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
      class="h-4 w-4 animate-spin rounded-full border-2 border-current/65 border-t-transparent"
    />
    <slot v-else name="icon" />
    <slot />
  </button>
</template>
