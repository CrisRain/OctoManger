<script setup lang="ts">
import {
  IconApps, IconLayers, IconUser, IconEmail,
  IconSchedule, IconHistory, IconRobot, IconRefresh, IconLoading,
  IconPlus, IconSettings, IconArrowRight, IconThunderbolt
} from "@/lib/icons";

import { useSystemStatus, useDashboardSnapshot } from "@/composables/useDashboard";
import { useCommandPaletteStore } from "@/store/command-palette";
import { to } from "@/router/registry";
import type { DashboardSummary } from "@/composables/useDashboard";

const router = useRouter();
const systemStatus = useSystemStatus();
const snapshot = useDashboardSnapshot();
const commandPalette = useCommandPaletteStore();

// 统计卡片配置
interface StatCard {
  label: string;
  valueKey: keyof Omit<DashboardSummary, "recentExecutions">;
  icon: any;
  gradientBg: string;
  iconColor: string;
  path?: string;
}

const statCards: StatCard[] = [
  {
    label: "插件",
    valueKey: "pluginCount",
    icon: IconApps,
    gradientBg: "rgba(20, 184, 166, 0.1)",
    iconColor: "#14b8a6",
    path: to.plugins.list()
  },
  {
    label: "账号类型",
    valueKey: "accountTypeCount",
    icon: IconLayers,
    gradientBg: "rgba(2, 132, 199, 0.1)",
    iconColor: "#0284c7",
    path: to.accountTypes.list()
  },
  {
    label: "账号",
    valueKey: "accountCount",
    icon: IconUser,
    gradientBg: "rgba(22, 163, 74, 0.1)",
    iconColor: "#16a34a",
    path: to.accounts.list()
  },
  {
    label: "邮箱账号",
    valueKey: "emailAccountCount",
    icon: IconEmail,
    gradientBg: "rgba(234, 88, 12, 0.1)",
    iconColor: "#ea580c",
    path: to.emailAccounts.list()
  },
  {
    label: "任务定义",
    valueKey: "jobDefinitionCount",
    icon: IconSchedule,
    gradientBg: "rgba(202, 138, 4, 0.1)",
    iconColor: "#ca8a04",
    path: to.jobs.list()
  },
  {
    label: "执行总数",
    valueKey: "jobExecutionCount",
    icon: IconHistory,
    gradientBg: "rgba(225, 29, 72, 0.1)",
    iconColor: "#e11d48",
    path: to.jobs.executions()
  },
  {
    label: "Agent",
    valueKey: "agentCount",
    icon: IconRobot,
    gradientBg: "rgba(14, 165, 233, 0.1)",
    iconColor: "#0ea5e9",
    path: to.agents.list()
  },
];

// 状态颜色映射
const executionStatusColor: Record<string, string> = {
  pending: "gray",
  running: "blue",
  done: "green",
  success: "green",
  failed: "red",
  cancelled: "gray",
};

function statusColor(status: string): string {
  return executionStatusColor[status] ?? "gray";
}

function executionIcon(status: string) {
  return status === "running" ? IconLoading : IconHistory;
}

// 快速操作配置
interface QuickAction {
  id: string;
  label: string;
  description: string;
  icon: any;
  color: string;
  path: string;
}

const quickActions: QuickAction[] = [
  { id: "add-account", label: "创建账号", description: "创建新的账号", icon: IconUser, color: "blue", path: to.accounts.create() },
  { id: "add-email", label: "创建邮箱账号", description: "创建新的邮箱账号", icon: IconEmail, color: "orange", path: to.emailAccounts.create() },
  { id: "create-job", label: "创建任务", description: "创建新的任务", icon: IconThunderbolt, color: "teal", path: to.jobs.create() },
  { id: "create-agent", label: "创建 Agent", description: "创建新的 Agent", icon: IconRobot, color: "cyan", path: to.agents.create() },
];

function openCommandPalette() {
  commandPalette.open();
}

function refreshDashboard() {
  systemStatus.refresh();
  snapshot.refresh();
}

// 获取图标组件
</script>

<template>
  <div class="page-shell">
    <PageHeader title="控制台" subtitle="查看系统概览并快速访问常用操作">
      <template #actions>
        <ui-button type="secondary" @click="refreshDashboard">
          <template #icon><icon-refresh /></template>
          刷新数据
        </ui-button>
      </template>
    </PageHeader>

    <!-- 系统状态横幅 -->
    <div
      class="mb-6 flex items-center gap-3 rounded-xl border px-5 py-4 text-sm"
      :class="systemStatus.data.value?.database_ok ? 'border-emerald-200 bg-emerald-50/90 text-emerald-800' : 'border-red-200 bg-red-50/90 text-red-700'"
    >
      <span
        class="inline-block h-2.5 w-2.5 flex-shrink-0 rounded-full"
        :class="systemStatus.data.value?.database_ok
          ? 'bg-emerald-500 animate-pulse'
          : 'bg-red-500'"
      />
      <div class="flex flex-1 items-center gap-2 max-md:flex-col max-md:items-start max-md:gap-1.5">
        <span class="font-semibold">
          {{ systemStatus.data.value?.database_ok ? '系统运行正常' : '系统异常，请检查服务状态' }}
        </span>
        <span v-if="systemStatus.data.value?.database_ok" class="text-xs opacity-70">
          最后检测：{{ new Date(systemStatus.data.value.now).toLocaleString("zh-CN") }}
        </span>
      </div>
    </div>

    <!-- 统计卡片 - 可点击跳转 -->
    <div class="mb-7 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7">
      <button
        type="button"
        v-for="card in statCards"
        :key="card.label"
        class="group relative flex flex-col items-center gap-2 rounded-xl border border-slate-200 bg-white p-5 text-center shadow-sm transition-all duration-200"
        :class="card.path ? 'cursor-pointer hover:border-slate-300 hover:shadow-md hover:-translate-y-0.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/20' : 'cursor-default'"
        @click="card.path && router.push(card.path)"
      >
        <div
          class="mb-1 flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl transition-transform duration-200 group-hover:scale-110"
          :style="{ background: card.gradientBg, color: card.iconColor }"
        >
          <component :is="card.icon" class="h-5 w-5" />
        </div>
        <div class="text-[28px] font-semibold leading-none tracking-tighter text-slate-900">
          {{ (snapshot.data.value?.[card.valueKey] as number) ?? 0 }}
        </div>
        <div class="text-xs font-medium text-slate-500">{{ card.label }}</div>
        <icon-arrow-right v-if="card.path" class="absolute right-2.5 top-2.5 h-3.5 w-3.5 text-slate-300 transition-colors group-hover:text-slate-500" />
      </button>
    </div>

    <div class="grid grid-cols-1 gap-6 lg:grid-cols-[2fr_3fr]">
      <!-- 左侧：快捷操作面板 -->
      <QuickActionsPanel
        :actions="quickActions"
        @open-search="openCommandPalette"
      />

      <!-- 右侧：最近执行记录 -->
      <ui-card class="min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-history class="h-4 w-4 text-[var(--accent)]" />
            <span>最近执行</span>
          </div>
        </template>

        <ui-empty
          v-if="!snapshot.data.value?.recentExecutions?.length"
          description="暂无执行记录"
        />

        <div v-else class="flex flex-col gap-3">
          <button
            type="button"
            v-for="record in snapshot.data.value?.recentExecutions ?? []"
            :key="record.id"
            class="flex w-full items-center gap-3.5 rounded-xl border border-slate-200 bg-white px-3.5 py-3.5 text-left transition-all hover:bg-slate-50 hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/20"
            @click="router.push(to.jobs.executionDetail(record.id))"
          >
            <div
              class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg border shadow-sm"
              :class="{
                'border-slate-200 bg-slate-50 text-slate-500': ['pending', 'cancelled'].includes(record.status) || !['pending', 'running', 'done', 'success', 'failed', 'cancelled'].includes(record.status),
                'border-blue-200 bg-blue-50 text-blue-600': record.status === 'running',
                'border-emerald-200 bg-emerald-50 text-emerald-600': ['done', 'success'].includes(record.status),
                'border-red-200 bg-red-50 text-red-600': record.status === 'failed'
              }"
            >
              <component :is="executionIcon(record.status)" />
            </div>

            <div class="min-w-0 flex-1">
              <div class="mb-2 flex items-center gap-2 justify-between">
                <div class="min-w-0 truncate text-sm font-medium text-slate-900">{{ record.definition_name }}</div>
                <ui-tag
                  :color="statusColor(record.status)"
                  size="small"
                  class="flex-shrink-0"
                >
                  {{ record.status }}
                </ui-tag>
              </div>

              <div class="flex flex-wrap items-center gap-1 text-sm text-slate-500">
                <span class="font-medium">{{ record.plugin_key }}</span>
                <span class="text-slate-400">·</span>
                <span class="truncate">{{ record.action }}</span>
                <span class="text-slate-400">·</span>
                <span class="truncate">{{ record.worker_id || "自动分配" }}</span>
              </div>
            </div>

            <icon-arrow-right class="h-4 w-4 flex-shrink-0 text-slate-400" />
          </button>
        </div>
      </ui-card>
    </div>
  </div>
</template>
