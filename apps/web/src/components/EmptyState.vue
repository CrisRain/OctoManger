<script setup lang="ts">
import { computed } from "vue";
import {
  IconFile, IconUser, IconEmail, IconCheckCircle,
  IconCloseCircle, IconRefresh, IconPlus
} from "@/lib/icons";

interface Props {
  type?: "empty" | "error" | "success" | "loading";
  title?: string;
  description?: string;
  icon?: any;
  actionText?: string;
  hideAction?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  type: "empty",
  description: "",
  hideAction: false,
});

const emit = defineEmits<{
  (e: "action"): void;
}>();

/**
 * 默认图标配置
 */
const defaultIcons = {
  empty: IconFile,
  error: IconCloseCircle,
  success: IconCheckCircle,
  loading: IconRefresh,
};

/**
 * 默认标题配置
 */
const defaultTitles = {
  empty: "暂无数据",
  error: "出错了",
  success: "操作成功",
  loading: "加载中...",
};

/**
 * 默认描述配置
 */
const defaultDescriptions = {
  empty: "这里还没有任何内容",
  error: "加载失败，请稍后重试",
  success: "操作已完成",
  loading: "正在努力加载中...",
};

const displayIcon = computed(() => props.icon || defaultIcons[props.type]);
const displayTitle = computed(() => props.title || defaultTitles[props.type]);
const displayDescription = computed(() => props.description || defaultDescriptions[props.type]);
const isLoading = computed(() => props.type === "loading");

function handleAction() {
  emit("action");
}
</script>

<template>
  <div class="empty-state" :class="{ 'empty-state--loading': isLoading }">
    <div class="empty-state-icon" :class="`empty-state-icon--${type}`">
      <component :is="displayIcon" :class="{ 'animate-spin': isLoading }" />
    </div>
    <div class="empty-state-content">
      <h3 class="empty-state-title">{{ displayTitle }}</h3>
      <slot name="description">
        <p v-if="displayDescription" class="empty-state-description">
          {{ displayDescription }}
        </p>
      </slot>
    </div>
    <div v-if="!hideAction" class="empty-state-action">
      <slot name="action">
        <ui-button v-if="actionText" type="primary" @click="handleAction">
          <template #icon>
            <icon-plus />
          </template>
          {{ actionText }}
        </ui-button>
      </slot>
    </div>
  </div>
</template>
