<script setup lang="ts">
import { computed, useSlots, useAttrs } from "vue";
import { cx } from "../utils";

interface Props {
  title?: string;
  bordered?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  title: "",
  bordered: true,
});

const slots = useSlots();
const attrs = useAttrs();

const hasHeader = computed(() => Boolean(props.title || slots.title || slots.extra));

const sectionClass = computed(() =>
  cx(
    "ui-card overflow-hidden rounded-xl border border-slate-200 bg-white/92 shadow-sm backdrop-blur-[8px] transition-all duration-200",
    !props.bordered && "border-transparent shadow-none",
    attrs.class as string,
  ),
);
</script>

<template>
  <section v-bind="{ ...attrs, class: undefined }" :class="sectionClass">
    <header
      v-if="hasHeader"
      class="ui-card-header flex items-center justify-between gap-3 border-b border-slate-200 bg-white/86 px-5 py-4"
    >
      <div class="ui-card-header-title font-display text-[15px] font-semibold text-slate-900">
        <slot name="title">{{ title }}</slot>
      </div>
      <slot name="extra" />
    </header>
    <div class="ui-card-body p-5">
      <slot />
    </div>
  </section>
</template>
