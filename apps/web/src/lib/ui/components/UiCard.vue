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
    "ui-card glass rounded-card border bg-surface-card shadow-card transition-all duration-300 hover:shadow-card-hover overflow-hidden",
    !props.bordered && "border-transparent shadow-none hover:shadow-none",
    attrs.class as string,
  ),
);
</script>

<template>
  <section v-bind="{ ...attrs, class: undefined }" :class="sectionClass">
    <header
      v-if="hasHeader"
      class="ui-card-header flex items-center justify-between gap-3 border-b border-surface-border bg-gradient-to-b from-slate-50/80 to-white/50 px-6 py-5"
    >
      <div class="ui-card-header-title font-display text-[16px] font-bold tracking-[-0.02em] text-text-primary">
        <slot name="title">{{ title }}</slot>
      </div>
      <slot name="extra" />
    </header>
    <div class="ui-card-body p-6">
      <slot />
    </div>
  </section>
</template>
