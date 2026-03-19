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
  <nav class="app-nav dark-scroll">
    <template v-for="item in props.items" :key="item.to">
      <router-link
        :to="item.to"
        class="app-nav__item"
        :class="[
          {
            'app-nav__item--active': isActive(item.to),
          },
        ]"
        @click="emit('navigate')"
      >
        <div class="app-nav__item-indicator" v-if="isActive(item.to)" />
        <component :is="item.icon" class="app-nav__icon" />
        <span>{{ item.label }}</span>
      </router-link>
      <div v-if="item.children?.length" class="app-nav__children">
        <router-link
          v-for="child in item.children"
          :key="child.to"
          :to="child.to"
          class="app-nav__child"
          :class="[
            {
              'app-nav__child--active': isChildActive(child.to),
            },
          ]"
          @click="emit('navigate')"
        >
          <span class="app-nav__child-dot" />
          {{ child.label }}
        </router-link>
      </div>
    </template>
  </nav>
</template>
