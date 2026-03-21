<script setup lang="ts">
import { useRoute } from "vue-router";

interface NavChild {
  to: string;
  label: string;
}

interface NavItem {
  name: string;
  to: string;
  label: string;
  icon: any;
  children?: NavChild[];
}

const props = defineProps<{
  items: NavItem[];
}>();

const emit = defineEmits<{
  (e: "navigate"): void;
}>();

const route = useRoute();

const isActive = (to: string): boolean => {
  return route.path === to || (to !== "/" && route.path.startsWith(`${to}/`));
};

const isChildActive = (to: string): boolean => {
  return route.path === to || route.path.startsWith(`${to}/`);
};
</script>

<template>
  <nav class="flex w-full flex-1 flex-col gap-0.5 dark-scroll">
    <template v-for="item in props.items" :key="item.to">
      <router-link
        :to="item.to"
        class="relative flex items-center gap-3 rounded-lg px-3 py-2 text-[14px] font-medium no-underline transition-all hover:bg-slate-100/80 hover:text-slate-900 text-slate-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/20"
        :class="isActive(item.to)
          ? 'bg-[var(--accent)]/8 text-[var(--accent)] font-semibold'
          : ''"
        @click="emit('navigate')"
      >
        <!-- Active left indicator bar -->
        <span
          v-if="isActive(item.to)"
          class="absolute left-0 top-1/2 h-5 w-0.5 -translate-y-1/2 rounded-full bg-[var(--accent)]"
        />
        <component
          :is="item.icon"
          class="h-[18px] w-[18px] flex-shrink-0 transition-colors"
          :class="isActive(item.to) ? 'text-[var(--accent)]' : 'text-slate-400'"
        />
        <span>{{ item.label }}</span>
      </router-link>
      <div v-if="item.children?.length" class="mt-0.5 flex w-full flex-col gap-0.5">
        <router-link
          v-for="child in item.children"
          :key="child.to"
          :to="child.to"
          class="flex items-center gap-2.5 rounded-lg px-3 py-1.5 text-[13px] font-medium no-underline transition-all hover:bg-slate-100/80 hover:text-slate-900 text-slate-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/20 ml-7"
          :class="isChildActive(child.to) ? 'bg-[var(--accent)]/8 text-[var(--accent)] font-semibold' : ''"
          @click="emit('navigate')"
        >
          {{ child.label }}
        </router-link>
      </div>
    </template>
  </nav>
</template>
