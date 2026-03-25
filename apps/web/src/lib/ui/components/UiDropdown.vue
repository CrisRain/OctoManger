<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useAttrs, watch } from "vue";
import { cx } from "../utils";

interface Props {
  visible?: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  "update:visible": [value: boolean];
  popupVisibleChange: [value: boolean];
  click: [event: MouseEvent];
}>();

const attrs = useAttrs();
const rootRef = ref<HTMLElement | null>(null);
const popupRef = ref<HTMLElement | null>(null);
const internalVisible = ref(false);
const popupStyle = ref<Record<string, string>>({});

const mergedVisible = computed(() =>
  typeof props.visible === "boolean" ? props.visible : internalVisible.value,
);

function setVisible(next: boolean) {
  if (typeof props.visible !== "boolean") internalVisible.value = next;
  emit("update:visible", next);
  emit("popupVisibleChange", next);
}

async function updatePopupPosition() {
  if (!mergedVisible.value || !rootRef.value) return;

  await nextTick();

  const triggerRect = rootRef.value.getBoundingClientRect();
  const popupRect = popupRef.value?.getBoundingClientRect();
  if (!popupRect) {
    requestAnimationFrame(() => {
      void updatePopupPosition();
    });
    return;
  }

  const viewportInset = Math.max(window.innerWidth * 0.015, 12);
  const offset = 8;

  let left = triggerRect.right - popupRect.width;
  if (left < viewportInset) {
    left = viewportInset;
  }
  if (left + popupRect.width > window.innerWidth - viewportInset) {
    left = Math.max(viewportInset, window.innerWidth - popupRect.width - viewportInset);
  }

  let top = triggerRect.bottom + offset;
  if (top + popupRect.height > window.innerHeight - viewportInset) {
    const nextTop = triggerRect.top - popupRect.height - offset;
    top = nextTop >= viewportInset
      ? nextTop
      : Math.max(viewportInset, window.innerHeight - popupRect.height - viewportInset);
  }

  popupStyle.value = {
    left: `${Math.round(left)}px`,
    position: "fixed",
    top: `${Math.round(top)}px`,
    visibility: "visible",
  };
}

function syncPopupPosition() {
  void updatePopupPosition();
}

function bindPopupPositionListeners() {
  window.addEventListener("resize", syncPopupPosition);
  window.addEventListener("scroll", syncPopupPosition, true);
}

function unbindPopupPositionListeners() {
  window.removeEventListener("resize", syncPopupPosition);
  window.removeEventListener("scroll", syncPopupPosition, true);
}

function handleDocumentClick(event: MouseEvent) {
  if (!mergedVisible.value) return;
  const target = event.target as Node | null;
  if (target && (rootRef.value?.contains(target) || popupRef.value?.contains(target))) return;
  setVisible(false);
}

onMounted(() => document.addEventListener("click", handleDocumentClick, true));
onBeforeUnmount(() => {
  document.removeEventListener("click", handleDocumentClick, true);
  unbindPopupPositionListeners();
});

watch(
  mergedVisible,
  (open) => {
    if (open) {
      popupStyle.value = {
        left: "0px",
        position: "fixed",
        top: "0px",
        visibility: "hidden",
      };
      bindPopupPositionListeners();
      void updatePopupPosition();
      return;
    }

    popupStyle.value = {};
    unbindPopupPositionListeners();
  },
  { immediate: true },
);

watch(
  popupRef,
  (popup) => {
    if (mergedVisible.value && popup) {
      void updatePopupPosition();
    }
  },
);
</script>

<template>
  <div
    ref="rootRef"
    v-bind="{ ...attrs, class: undefined }"
    :class="cx('relative inline-flex', mergedVisible && 'ui-dropdown-open', attrs.class as string)"
    @click="emit('click', $event)"
  >
    <div class="inline-flex" @click.stop="setVisible(!mergedVisible)">
      <slot />
    </div>
    <Teleport to="body">
      <div
        v-if="mergedVisible"
        ref="popupRef"
        class="ui-dropdown-popup z-popover min-w-[180px] rounded-xl border border-slate-200 bg-white/95 p-1.5 shadow-md backdrop-blur-md"
        :style="popupStyle"
        @click.stop
      >
        <slot name="content" />
      </div>
    </Teleport>
  </div>
</template>
