<script setup lang="ts">
import { computed, useAttrs, useSlots } from "vue";
import { cx } from "../utils";

interface Props {
  label?: string;
  required?: boolean;
  validateStatus?: string;
  help?: string;
}

const props = withDefaults(defineProps<Props>(), {
  label: "",
  required: false,
  validateStatus: "",
  help: "",
});

const attrs = useAttrs();
const slots = useSlots();
const hasLabel = computed(() => Boolean(props.label || slots.label));
const helpClass = computed(() =>
  cx("ui-form-item-help text-xs", props.validateStatus === "error" ? "text-red-600" : "text-slate-500"),
);
</script>

<template>
  <div v-bind="{ ...attrs, class: undefined }" :class="cx('ui-form-item space-y-2', attrs.class as string)">
    <label
      v-if="hasLabel"
      class="ui-form-item-label inline-flex items-center gap-1 text-sm font-semibold text-slate-700 w-full"
    >
      <slot name="label">
        {{ label }}
        <span v-if="required" class="text-red-500">*</span>
      </slot>
    </label>
    <div class="ui-form-item-body">
      <slot />
    </div>
    <div v-if="help" :class="helpClass">{{ help }}</div>
  </div>
</template>
