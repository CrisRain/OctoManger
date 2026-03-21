<script setup lang="ts">
import { useAccountTypes } from "@/composables";
import { navRoutes, routeNames, searchRoutes, shortcutRoutes, to, type IconKey } from "@/router/registry";
import { useCommandPaletteStore } from "@/store/command-palette";

import {
  IconDashboard, IconLayers, IconLink, IconEmail, IconSchedule,
  IconRobot, IconFile, IconApps, IconThunderbolt,
  IconSettings, IconMenu
} from "@/lib/icons";

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

const route = useRoute();
const router = useRouter();
const { data: accountTypes } = useAccountTypes();
const commandPalette = useCommandPaletteStore();

const mobileOpen = ref(false);
const keySequence = ref<string[]>([]);
const keySequenceTimer = ref<number>();

const iconMap: Record<IconKey, any> = {
  dashboard: IconDashboard,
  layers: IconLayers,
  link: IconLink,
  email: IconEmail,
  schedule: IconSchedule,
  robot: IconRobot,
  apps: IconApps,
  file: IconFile,
  settings: IconSettings,
  thunderbolt: IconThunderbolt,
};

function buildNavItems(): NavItem[] {
  const parents = navRoutes.filter((route) => !route.navParent);
  const childrenByParent = new Map<string, NavChild[]>();

  for (const route of navRoutes) {
    if (!route.navParent) continue;
    const children = childrenByParent.get(route.navParent) ?? [];
    children.push({ to: route.path, label: route.label });
    childrenByParent.set(route.navParent, children);
  }

  return parents.map((route) => ({
    name: route.name,
    to: route.path,
    label: route.label,
    icon: iconMap[route.iconKey ?? "settings"] ?? IconSettings,
    children: childrenByParent.get(route.name),
  }));
}

const navItems = computed<NavItem[]>(() => {
  const baseItems = buildNavItems();
  const accountsItem = baseItems.find((item) => item.name === routeNames.accountsList);
  if (accountsItem) {
    const genericTypes = accountTypes.value.filter((item) => item.category === "generic");
    accountsItem.children = [
      { to: to.accounts.list(), label: "全部账号" },
      ...genericTypes.map((item) => ({
        to: to.accounts.byType(item.key),
        label: item.name,
      })),
      ...(accountsItem.children ?? []).filter((child) => child.to !== to.accounts.list()),
    ];
  }
  return baseItems;
});

const currentTitle = computed(() => {
  const path = route.path;
  const topMatch = navItems.value.find(
    (item) => path === item.to || path.startsWith(`${item.to}/`)
  );
  if (!topMatch) return "控制台";
  const childMatch = topMatch.children?.find(
    (child) => path === child.to || path.startsWith(`${child.to}/`)
  );
  return childMatch?.label ?? topMatch.label;
});

function closeMobile() {
  mobileOpen.value = false;
}

const baseCommands = computed(() => [
  {
    id: "open-search",
    label: "打开搜索",
    description: "快速搜索任何资源",
    shortcut: "⌘K",
    action: () => {
      commandPalette.open();
    },
  },
  {
    id: "new-resource",
    label: "新建资源",
    description: "创建新的账号、任务等",
    shortcut: "⌘N",
    action: () => {
      router.push(to.accounts.create());
    },
  },
  {
    id: "refresh",
    label: "刷新当前页面",
    description: "重新加载数据",
    shortcut: "⌘R",
    action: () => router.go(0),
  },
]);

const routeCommands = computed(() =>
  searchRoutes.map((item) => ({
    id: `search-${item.name}`,
    label: item.label,
    description: item.type === "page" ? `前往 ${item.label}` : `打开 ${item.label}`,
    keywords: item.keywords,
    action: () => router.push(item.path),
  }))
);

const commands = computed(() => [...baseCommands.value, ...routeCommands.value]);

function handleCommandExecute() {
  commandPalette.close();
}

function handleKeyDown(e: KeyboardEvent) {
  const target = e.target as HTMLElement;
  if (
    target.tagName === "INPUT" ||
    target.tagName === "TEXTAREA" ||
    target.contentEditable === "true"
  ) {
    if (e.key !== "Escape") return;
  }

  const isMac = navigator.platform.toUpperCase().indexOf("MAC") >= 0;
  const modKey = isMac ? e.metaKey : e.ctrlKey;

  if (modKey) {
    switch (e.key.toLowerCase()) {
      case "k":
        e.preventDefault();
        commandPalette.open();
        return;
      case "n":
        e.preventDefault();
        router.push(to.accounts.create());
        return;
      case "/":
        e.preventDefault();
        commandPalette.open();
        return;
      case "r":
        e.preventDefault();
        router.go(0);
        return;
    }
  }

  if (e.key === "Escape") {
    commandPalette.close();
    return;
  }

  clearTimeout(keySequenceTimer.value);
  keySequence.value.push(e.key.toUpperCase());

  keySequenceTimer.value = window.setTimeout(() => {
    const sequence = keySequence.value.join(" then ");
    const shortcut = shortcutRoutes.find((item) => item.key === sequence);
    if (shortcut) {
      router.push(shortcut.path);
    }
    keySequence.value = [];
  }, 500);
}

onMounted(() => {
  window.addEventListener("keydown", handleKeyDown);
});

onUnmounted(() => {
  window.removeEventListener("keydown", handleKeyDown);
  if (keySequenceTimer.value) {
    clearTimeout(keySequenceTimer.value);
  }
});
</script>

<template>
  <div class="flex h-full flex-col">
    <!-- ── Mobile header ─────────────────────────────────── -->
    <header class="sticky top-0 z-40 mt-4 flex h-16 flex-shrink-0 items-center gap-3 rounded-xl border border-slate-200 bg-white/80 px-5 shadow-sm backdrop-blur-md lg:hidden mx-4">
      <button type="button" class="inline-flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 transition-all hover:bg-slate-50 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20" @click="mobileOpen = true">
        <icon-menu />
      </button>
      <span class="flex-1 truncate font-semibold tracking-tight text-slate-900">{{ currentTitle }}</span>
    </header>

    <!-- ── Mobile drawer ─────────────────────────────────── -->
    <ui-drawer
      :visible="mobileOpen"
      placement="left"
      :footer="false"
      :header="false"
      popup-container="body"
      class="[&.ui-drawer]:bg-[var(--sidebar-bg)] [&.ui-drawer-body]:p-0"
      @cancel="closeMobile"
    >
      <div class="flex h-full w-full flex-col overflow-y-auto px-2 py-3">
        <!-- Logo -->
        <div class="mb-5 flex items-center gap-3 px-3 py-3">
          <div class="flex h-8 w-8 items-center justify-center rounded-xl bg-[var(--accent)] text-white shadow-sm">
            <icon-layers class="h-4 w-4" />
          </div>
          <span class="text-[15px] font-bold tracking-tight text-slate-900">OctoManager</span>
        </div>

        <AppNavList
          :items="navItems"
          @navigate="closeMobile"
        />
      </div>
    </ui-drawer>

    <!-- ── Main layout ────────────────────────────────────── -->
    <div class="flex flex-1 overflow-hidden min-h-0">
      <!-- Desktop sidebar -->
      <aside class="hidden w-60 flex-shrink-0 overflow-hidden border-r border-slate-200/80 bg-[var(--sidebar-bg)] p-3 lg:flex">
        <div class="flex h-full w-full flex-col overflow-y-auto px-2 py-3">
          <!-- Logo -->
          <div class="mb-5 flex items-center gap-3 px-3 py-3">
            <div class="flex h-8 w-8 items-center justify-center rounded-xl bg-[var(--accent)] text-white shadow-sm">
              <icon-layers class="h-4 w-4" />
            </div>
            <span class="text-[15px] font-bold tracking-tight text-slate-900">OctoManager</span>
          </div>

          <AppNavList :items="navItems" />
        </div>
      </aside>

      <!-- Content Area with Transition -->
      <main class="flex-1 overflow-y-auto bg-[var(--page-bg,#f8fafc)]">
        <router-view v-slot="{ Component }">
          <transition name="fade-slide" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
    </div>

    <!-- 全局快捷键 -->
    <KeyboardShortcuts
      v-model:open="commandPalette.isOpen"
      v-model:query="commandPalette.query"
      :commands="commands"
      @execute="handleCommandExecute"
    />
  </div>
</template>
