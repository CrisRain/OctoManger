<script setup lang="ts">
import { computed, ref, watch, useSlots, useAttrs, type VNode } from "vue";
import { flattenNodes, cx } from "../utils";

interface Props {
  activeKey?: string | number;
  destroyOnHide?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  activeKey: undefined,
  destroyOnHide: true,
});

const emit = defineEmits<{
  "update:activeKey": [key: string];
  change: [key: string];
}>();

const slots = useSlots();
const attrs = useAttrs();
const internalActive = ref("");

interface ParsedTab {
  key: string;
  title: VNode[] | string;
  content: () => VNode[];
}

const tabs = computed<ParsedTab[]>(() => {
  const source = flattenNodes(slots.default?.() ?? []);
  return source
    .filter((node) => {
      const type = node.type as { name?: string; __name?: string };
      return type?.name === "UiTabPane" || type?.__name === "UiTabPane";
    })
    .map((node, index) => {
      const tabKey = String(node.key ?? index);
      const np = (node.props ?? {}) as Record<string, unknown>;
      const children = node.children as Record<string, (...args: unknown[]) => VNode[]> | null;
      const title = np.title ? String(np.title) : (children?.title?.() ?? tabKey);
      return {
        key: tabKey,
        title,
        content: () => children?.default?.() ?? [],
      } satisfies ParsedTab;
    });
});

watch(
  tabs,
  (nextTabs) => {
    if (!nextTabs.length) { internalActive.value = ""; return; }
    const fallback = nextTabs[0].key;
    const current = props.activeKey != null ? String(props.activeKey) : internalActive.value;
    if (!nextTabs.some((t) => t.key === current)) internalActive.value = fallback;
  },
  { immediate: true },
);

const currentKey = computed(() =>
  props.activeKey != null ? String(props.activeKey) : internalActive.value,
);

function setActive(key: string) {
  if (props.activeKey == null) internalActive.value = key;
  emit("update:activeKey", key);
  emit("change", key);
}
</script>

<template>
  <div v-bind="{ ...attrs, class: undefined }" :class="cx('ui-tabs', attrs.class as string)">
    <div class="ui-tabs-nav flex items-center border-b border-slate-200">
      <div class="ui-tabs-nav-tab-list flex items-center gap-0.5">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          :class="cx(
            'ui-tabs-tab relative px-4 py-2.5 text-sm font-semibold transition-colors focus-visible:outline-none',
            tab.key === (currentKey || tabs[0]?.key)
              ? 'ui-tabs-tab-active text-accent'
              : 'text-slate-500 hover:text-slate-700',
          )"
          @click="setActive(tab.key)"
        >
          <span class="ui-tabs-tab-title">
            <component :is="() => tab.title" />
          </span>
          <span
            v-if="tab.key === (currentKey || tabs[0]?.key)"
            class="absolute bottom-0 left-2 right-2 h-0.5 rounded-full bg-accent"
            style="animation: tab-ink 0.25s cubic-bezier(0.22, 1, 0.36, 1)"
          />
        </button>
      </div>
    </div>

    <div class="ui-tabs-content">
      <div class="ui-tabs-content-list">
        <template v-if="destroyOnHide">
          <section
            v-for="tab in tabs.filter((t) => t.key === (currentKey || tabs[0]?.key))"
            :key="tab.key"
            class="ui-tabs-content-item ui-tabs-pane py-4"
          >
            <component :is="() => tab.content()" />
          </section>
        </template>
        <template v-else>
          <section
            v-for="tab in tabs"
            :key="tab.key"
            :class="cx('ui-tabs-content-item ui-tabs-pane py-4', tab.key === (currentKey || tabs[0]?.key) ? 'block' : 'hidden')"
          >
            <component :is="() => tab.content()" />
          </section>
        </template>
      </div>
    </div>
  </div>
</template>
