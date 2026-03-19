<script setup lang="ts">
import { ref, useAttrs, nextTick, onUnmounted } from "vue";
import { cx } from "../utils";
import UiButton from "./UiButton.vue";

interface Props {
  content?: string;
  title?: string;
  okText?: string;
  cancelText?: string;
}

const props = withDefaults(defineProps<Props>(), {
  content: "确认执行该操作？",
  title: "",
  okText: "确定",
  cancelText: "取消",
});

const emit = defineEmits<{ ok: []; cancel: [] }>();
const attrs = useAttrs();

const open = ref(false);
const triggerRef = ref<HTMLElement>();
const popoverRef = ref<HTMLElement>();

function handleClick(event: MouseEvent) {
  event.preventDefault();
  event.stopPropagation();
  open.value = !open.value;
  if (open.value) {
    nextTick(() => positionPopover());
  }
}

function positionPopover() {
  if (!triggerRef.value || !popoverRef.value) return;
  const triggerRect = triggerRef.value.getBoundingClientRect();
  const pop = popoverRef.value;
  const popRect = pop.getBoundingClientRect();

  // Position above the trigger, centered
  let top = triggerRect.top - popRect.height - 8;
  let left = triggerRect.left + triggerRect.width / 2 - popRect.width / 2;

  // If above viewport, flip below
  if (top < 8) {
    top = triggerRect.bottom + 8;
    pop.dataset.placement = "bottom";
  } else {
    pop.dataset.placement = "top";
  }

  // Keep within horizontal bounds
  left = Math.max(8, Math.min(left, window.innerWidth - popRect.width - 8));

  pop.style.top = `${top}px`;
  pop.style.left = `${left}px`;
}

function confirm() {
  open.value = false;
  emit("ok");
}

function cancel() {
  open.value = false;
  emit("cancel");
}

function onClickOutside(event: MouseEvent) {
  if (
    !triggerRef.value?.contains(event.target as Node) &&
    !popoverRef.value?.contains(event.target as Node)
  ) {
    open.value = false;
  }
}

// Use capture to catch clicks before they propagate
if (typeof document !== "undefined") {
  document.addEventListener("mousedown", onClickOutside, true);
}

onUnmounted(() => {
  document.removeEventListener("mousedown", onClickOutside, true);
});
</script>

<template>
  <span
    ref="triggerRef"
    v-bind="{ ...attrs, class: undefined }"
    :class="cx('inline-flex', attrs.class as string)"
    @click="handleClick"
  >
    <slot />
  </span>

  <Teleport to="body">
    <Transition name="popconfirm">
      <div
        v-if="open"
        ref="popoverRef"
        class="fixed z-popover rounded-xl border border-slate-200/80 bg-white px-4 py-3.5 shadow-lg"
        style="min-width: 220px; max-width: 320px"
      >
        <!-- Arrow -->
        <div
          class="absolute w-2 h-2 rotate-45 border bg-white"
          :class="popoverRef?.dataset.placement === 'bottom'
            ? '-top-1 left-1/2 -translate-x-1/2 border-l-slate-200/80 border-t-slate-200/80 border-r-transparent border-b-transparent'
            : '-bottom-1 left-1/2 -translate-x-1/2 border-r-slate-200/80 border-b-slate-200/80 border-l-transparent border-t-transparent'"
        />
        <div class="flex flex-col gap-2">
          <div class="flex items-start gap-2">
            <svg class="w-4 h-4 text-amber-500 flex-shrink-0 mt-0.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
              <line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>
            <div class="flex flex-col gap-0.5">
              <p v-if="title" class="text-sm font-semibold text-slate-900">{{ title }}</p>
              <p class="text-sm text-slate-600 leading-relaxed">{{ content }}</p>
            </div>
          </div>
          <div class="flex items-center justify-end gap-2 pt-1">
            <UiButton size="small" @click="cancel">{{ cancelText }}</UiButton>
            <UiButton size="small" type="primary" @click="confirm">{{ okText }}</UiButton>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
