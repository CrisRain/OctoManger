<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, useAttrs } from "vue";
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
const internalVisible = ref(false);

const mergedVisible = computed(() =>
  typeof props.visible === "boolean" ? props.visible : internalVisible.value,
);

function setVisible(next: boolean) {
  if (typeof props.visible !== "boolean") internalVisible.value = next;
  emit("update:visible", next);
  emit("popupVisibleChange", next);
}

function handleDocumentClick(event: MouseEvent) {
  if (!mergedVisible.value) return;
  const target = event.target as Node | null;
  if (target && rootRef.value?.contains(target)) return;
  setVisible(false);
}

onMounted(() => document.addEventListener("click", handleDocumentClick, true));
onBeforeUnmount(() => document.removeEventListener("click", handleDocumentClick, true));
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
    <div
      v-if="mergedVisible"
      class="ui-dropdown-popup absolute right-0 top-full z-popover mt-2 min-w-[180px] rounded-2xl border border-slate-200 bg-white p-2 shadow-xl"
      @click.stop
    >
      <slot name="content" />
    </div>
  </div>
</template>
