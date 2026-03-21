<script setup lang="ts">
import { computed, useSlots } from "vue";
import { useRouter } from "vue-router";

const props = defineProps<{
  title: string;
  subtitle?: string;
  /** 用于标题图标背景的 CSS gradient / color 字符串 */
  iconBg?: string;
  /** 用于标题图标前景色 */
  iconColor?: string;
  /** 返回按钮跳转路径，不传则不显示返回按钮 */
  backTo?: string;
  /** 返回按钮 tooltip 文字 */
  backLabel?: string;
}>();

const router = useRouter();
const slots = useSlots();
const hasSubtitle = computed(() => Boolean(props.subtitle || slots.subtitle?.().length));
const resolvedIconBg = computed(
  () =>
    props.iconBg ??
    "linear-gradient(135deg, rgba(10, 132, 255, 0.18) 0%, rgba(94, 92, 230, 0.16) 100%)"
);
const resolvedIconColor = computed(() => props.iconColor ?? "var(--accent)");
</script>

<template>
  <div class="mb-8 flex items-start justify-between gap-4 max-md:flex-col max-md:items-start motion-safe:animate-[slide-up_0.3s_ease-out]">
    <div class="flex min-w-0 flex-1 items-center gap-3">
      <button type="button"
        v-if="backTo"
        class="relative inline-flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-500 shadow-sm transition-all hover:bg-slate-50 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20"
        @click="router.push(backTo)"
      >
        <icon-left />
        <span v-if="backLabel" class="page-header__tooltip pointer-events-none absolute bottom-full left-1/2 mb-2 -translate-x-1/2 translate-y-1.5 whitespace-nowrap rounded-full border border-slate-200 bg-gray-900/90 px-2.5 py-1 text-xs text-white/90 opacity-0 transition-[opacity,transform] duration-[160ms]">{{ backLabel }}</span>
      </button>

      <div class="flex min-w-0 flex-col gap-1">
        <h1 class="flex items-center gap-2 text-2xl font-semibold tracking-tight text-slate-900 max-md:text-xl">
          <span
            v-if="$slots.icon"
            class="inline-flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg border border-slate-200 shadow-sm max-md:h-9 max-md:w-9"
            :style="{ background: resolvedIconBg, color: resolvedIconColor }"
          >
            <slot name="icon" />
          </span>
          {{ title }}
        </h1>
        <p v-if="hasSubtitle" class="max-w-[72ch] text-[14px] leading-relaxed text-slate-600">
          <slot name="subtitle">{{ subtitle }}</slot>
        </p>
      </div>
    </div>

    <div v-if="$slots.actions" class="flex flex-shrink-0 flex-wrap items-center justify-end gap-2.5 max-md:w-full max-md:justify-start">
      <slot name="actions" />
    </div>
  </div>
</template>
