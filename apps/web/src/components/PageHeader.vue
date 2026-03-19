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
    "linear-gradient(135deg, rgba(20, 184, 166, 0.16) 0%, rgba(14, 116, 144, 0.16) 100%)"
);
const resolvedIconColor = computed(() => props.iconColor ?? "var(--accent)");
</script>

<template>
  <div class="page-header page-header--premium page-header--animated">
    <div class="page-header__left">
      <button type="button"
        v-if="backTo"
        class="page-header__back page-header__back-trigger"
        @click="router.push(backTo)"
      >
        <icon-left />
        <span v-if="backLabel" class="page-header__tooltip">{{ backLabel }}</span>
      </button>

      <div class="page-header__titles">
        <h1 class="page-title identifier-title">
          <span
            v-if="$slots.icon"
            class="page-header__icon"
            :style="{ background: resolvedIconBg, color: resolvedIconColor }"
          >
            <slot name="icon" />
          </span>
          {{ title }}
        </h1>
        <p v-if="hasSubtitle" class="page-subtitle">
          <slot name="subtitle">{{ subtitle }}</slot>
        </p>
      </div>
    </div>

    <div v-if="$slots.actions" class="page-header__actions">
      <slot name="actions" />
    </div>
  </div>
</template>
