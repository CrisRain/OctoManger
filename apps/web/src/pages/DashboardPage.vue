<script setup lang="ts">
import { computed } from "vue";
import { useRouter } from "vue-router";
import {
  IconApps, IconLayers, IconUser, IconEmail,
  IconSchedule, IconHistory, IconRobot, IconRefresh, IconLoading,
  IconPlus, IconSettings, IconArrowRight, IconThunderbolt
} from "@/lib/icons";

import { useSystemStatus, useDashboardSnapshot } from "@/composables/useDashboard";
import { PageHeader, QuickActionsPanel } from "@/components/index";
import { to } from "@/router/registry";
import type { DashboardSummary } from "@/composables/useDashboard";

const router = useRouter();
const systemStatus = useSystemStatus();
const snapshot = useDashboardSnapshot();

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
    gradientBg: "linear-gradient(135deg, rgba(20,184,166,0.12) 0%, rgba(45,212,191,0.12) 100%)",
    iconColor: "#14b8a6",
    path: to.plugins.list()
  },
  {
    label: "账号类型",
    valueKey: "accountTypeCount",
    icon: IconLayers,
    gradientBg: "linear-gradient(135deg, rgba(2,132,199,0.1) 0%, rgba(14,165,233,0.1) 100%)",
    iconColor: "#0284c7",
    path: to.accountTypes.list()
  },
  {
    label: "通用账号",
    valueKey: "accountCount",
    icon: IconUser,
    gradientBg: "linear-gradient(135deg, rgba(22,163,74,0.1) 0%, rgba(34,197,94,0.1) 100%)",
    iconColor: "#16a34a",
    path: to.accounts.list()
  },
  {
    label: "邮箱账号",
    valueKey: "emailAccountCount",
    icon: IconEmail,
    gradientBg: "linear-gradient(135deg, rgba(234,88,12,0.1) 0%, rgba(249,115,22,0.1) 100%)",
    iconColor: "#ea580c",
    path: to.emailAccounts.list()
  },
  {
    label: "任务定义",
    valueKey: "jobDefinitionCount",
    icon: IconSchedule,
    gradientBg: "linear-gradient(135deg, rgba(202,138,4,0.1) 0%, rgba(234,179,8,0.1) 100%)",
    iconColor: "#ca8a04",
    path: to.jobs.list()
  },
  {
    label: "执行总数",
    valueKey: "jobExecutionCount",
    icon: IconHistory,
    gradientBg: "linear-gradient(135deg, rgba(225,29,72,0.1) 0%, rgba(244,63,94,0.1) 100%)",
    iconColor: "#e11d48",
    path: to.jobs.executions()
  },
  {
    label: "后台任务",
    valueKey: "agentCount",
    icon: IconRobot,
    gradientBg: "linear-gradient(135deg, rgba(14,165,233,0.12) 0%, rgba(56,189,248,0.12) 100%)",
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
  { id: "add-account", label: "添加账号", description: "创建新的通用账号", icon: IconUser, color: "blue", path: to.accounts.create() },
  { id: "add-email", label: "新建邮箱", description: "配置邮箱账号", icon: IconEmail, color: "orange", path: to.emailAccounts.create() },
  { id: "create-job", label: "创建任务", description: "新建定时任务", icon: IconThunderbolt, color: "teal", path: to.jobs.create() },
  { id: "create-agent", label: "创建Agent", description: "启动后台任务", icon: IconRobot, color: "cyan", path: to.agents.create() },
];

// 最近使用（模拟数据，实际应该从后端获取）
interface RecentItem {
  id: string;
  name: string;
  type: "account" | "job" | "agent" | "email";
  path: string;
  updatedAt: string;
}

const recentItems: RecentItem[] = [
  { id: "1", name: "GitHub 账号", type: "account", path: to.accounts.detail(1), updatedAt: "5分钟前" },
  { id: "2", name: "每日备份任务", type: "job", path: to.jobs.detail(1), updatedAt: "1小时前" },
  { id: "3", name: "邮件监控", type: "agent", path: to.agents.detail(1), updatedAt: "2小时前" },
];

function openCommandPalette() {
  window.dispatchEvent(new CustomEvent("command-palette:open"));
}

function refreshDashboard() {
  systemStatus.refresh();
  snapshot.refresh();
}

// 获取图标组件
</script>

<template>
  <div class="page-container dashboard-page">
    <PageHeader title="控制台" subtitle="系统概览与快捷操作中心">
      <template #actions>
        <ui-button type="secondary" @click="refreshDashboard">
          <template #icon><icon-refresh /></template>
          刷新数据
        </ui-button>
      </template>
    </PageHeader>

    <!-- 系统状态横幅 -->
    <div
      class="dashboard__status"
      :class="systemStatus.data.value?.database_ok ? 'status-banner--ok' : 'status-banner--err'"
    >
      <span
        class="status-dot status-dot--banner dashboard__status-dot"
        :class="systemStatus.data.value?.database_ok ? 'online' : 'offline'"
      />
      <div class="dashboard__status-content">
        <span class="dashboard__status-text">
          {{ systemStatus.data.value?.database_ok ? '系统运行正常' : '系统异常，请检查服务状态' }}
        </span>
        <span v-if="systemStatus.data.value?.database_ok" class="dashboard__status-time">
          最后检测：{{ new Date(systemStatus.data.value.now).toLocaleString("zh-CN") }}
        </span>
      </div>
    </div>

    <!-- 统计卡片 - 可点击跳转 -->
    <div class="dashboard__stats">
      <button
        type="button"
        v-for="card in statCards"
        :key="card.label"
        class="dashboard__stat-card"
        :class="[
          {
            'dashboard__stat-card--clickable': !!card.path,
          },
        ]"
        @click="card.path && router.push(card.path)"
      >
        <div class="dashboard__stat-icon" :style="{ background: card.gradientBg, color: card.iconColor }">
          <component :is="card.icon" class="dashboard__stat-icon-svg" />
        </div>
        <div class="dashboard__stat-value">
          {{ (snapshot.data.value?.[card.valueKey] as number) ?? 0 }}
        </div>
        <div class="dashboard__stat-label">{{ card.label }}</div>
        <icon-arrow-right v-if="card.path" class="dashboard__stat-arrow" />
      </button>
    </div>

    <div class="dashboard__content">
      <!-- 左侧：快捷操作面板 -->
      <QuickActionsPanel
        :actions="quickActions"
        :recent-items="recentItems"
        @open-search="openCommandPalette"
      />

      <!-- 右侧：最近执行记录 -->
      <ui-card class="recent-card">
        <template #title>
          <div class="dashboard__card-title">
            <icon-history class="dashboard__card-title-icon" />
            <span>最近执行</span>
          </div>
        </template>

        <ui-empty
          v-if="!snapshot.data.value?.recentExecutions?.length"
          description="暂无执行记录"
        />

        <div v-else class="dashboard__executions">
          <button
            type="button"
            v-for="record in snapshot.data.value?.recentExecutions ?? []"
            :key="record.id"
            class="dashboard__execution-item"
            @click="router.push(to.jobs.executionDetail(record.id))"
          >
            <ui-tag
              :color="statusColor(record.status)"
              size="small"
              class="dashboard__execution-status"
            >
              <template #icon v-if="record.status === 'running'">
                <icon-loading />
              </template>
              {{ record.status }}
            </ui-tag>
            <div class="dashboard__execution-name">{{ record.definition_name }}</div>
            <div class="dashboard__execution-detail">
              <span class="dashboard__execution-plugin">{{ record.plugin_key }}</span>
              <span class="dashboard__execution-dot">·</span>
              <span class="execution-action">{{ record.action }}</span>
            </div>
            <div class="dashboard__execution-time">
              {{ record.worker_id || "自动分配" }}
            </div>
            <icon-arrow-right class="dashboard__execution-arrow" />
          </button>
        </div>
      </ui-card>
    </div>
  </div>
</template>
