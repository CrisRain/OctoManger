<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import {
  IconRobot, IconPlayArrow, IconStop, IconEye,
  IconEdit, IconDelete
} from "@/lib/icons";

import { PageHeader, SmartListBar, RowActionsMenu, DetailDrawer, StatusTag } from "@/components/index";
import { useAgents, useStartAgent, useStopAgent } from "@/composables/useAgents";
import { useMessage, useConfirm, useErrorHandler } from "@/composables";
import { to } from "@/router/registry";

const router = useRouter();
const message = useMessage();
const confirm = useConfirm();
const { withErrorHandler } = useErrorHandler();

const { data: agents, loading, refresh } = useAgents();
const startAgent = useStartAgent();
const stopAgent = useStopAgent();

// 状态筛选
const statusFilter = ref<string>();
const searchKeyword = ref("");

// 过滤 Agent
const filteredAgents = computed(() => {
  let result = agents.value;

  if (statusFilter.value) {
    result = result.filter(item => item.runtime_state === statusFilter.value);
  }

  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase();
    result = result.filter(item =>
      item.name?.toLowerCase().includes(keyword) ||
      item.plugin_key?.toLowerCase().includes(keyword) ||
      item.action?.toLowerCase().includes(keyword)
    );
  }

  return result;
});

// 状态配置
const statusOptions = [
  { label: "全部", value: "" },
  { label: "运行中", value: "running" },
  { label: "已停止", value: "stopped" },
  { label: "停止中", value: "stopping" },
  { label: "错误", value: "error" },
];

// 抽屉状态
const drawerVisible = ref(false);
const currentAgentId = ref<number | null>(null);
const currentAgent = computed(() =>
  agents.value.find((item) => item.id === currentAgentId.value) ?? null
);

// 快速操作
async function handleQuickAction(key: string, agent: any) {
  switch (key) {
    case "view":
      showAgentDetail(agent);
      break;
    case "start":
      await handleStart(agent.id);
      break;
    case "stop":
      await handleStop(agent.id);
      break;
    case "edit":
      router.push(to.agents.edit(agent.id));
      break;
    case "delete":
      await deleteAgent(agent);
      break;
  }
}

// 显示详情抽屉
function showAgentDetail(agent: any) {
  currentAgentId.value = agent.id;
  drawerVisible.value = true;
}

// 启动 Agent
async function handleStart(id: number) {
  await withErrorHandler(
    async () => {
      await startAgent.execute(id);
      message.success("启动指令已发送");
      await refresh();
    },
    { action: "启动 Agent", showSuccess: true }
  );
}

// 停止 Agent
async function handleStop(id: number) {
  await withErrorHandler(
    async () => {
      await stopAgent.execute(id);
      message.success("停止指令已发送");
      await refresh();
    },
    { action: "停止 Agent", showSuccess: true }
  );
}

// 删除 Agent
async function deleteAgent(agent: any) {
  const confirmed = await confirm.confirmDelete(agent.name);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      // TODO: 调用API删除
      message.success(`已删除 Agent: ${agent.name}`);
      await refresh();
    },
    { action: "删除", showSuccess: true }
  );
}

// 批量操作
async function handleBatchStop(items: any[]) {
  const confirmed = await confirm.confirm(`确定要停止选中的 ${items.length} 个 Agent 吗？`);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      // TODO: 调用批量停止API
      message.success(`已向 ${items.length} 个 Agent 发送停止指令`);
      await refresh();
    },
    { action: "批量停止", showSuccess: true }
  );
}

// 批量导出
async function handleBatchExport(items: any[]) {
  const data = JSON.stringify(items, null, 2);
  const blob = new Blob([data], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `agents-${Date.now()}.json`;
  link.click();
  URL.revokeObjectURL(url);
  message.success(`已导出 ${items.length} 个 Agent`);
}
</script>

<template>
  <div class="page-shell agents-list-page">
    <PageHeader
      title="Agent 管理"
      subtitle="管理持续运行的 Agent 进程"
      icon-bg="linear-gradient(135deg, rgba(10,132,255,0.12), rgba(10,132,255,0.06))"
      icon-color="var(--icon-purple)"
    >
      <template #icon><icon-robot /></template>
      <template #actions>
        <ui-button type="primary" @click="router.push(to.agents.create())">
          <template #icon><icon-plus /></template>
          创建 Agent
        </ui-button>
      </template>
    </PageHeader>

    <!-- 智能工具栏 -->
    <SmartListBar
      :data="filteredAgents"
      :loading="loading"
      v-model:search="searchKeyword"
      @refresh="refresh"
      @batch-delete="handleBatchStop"
      @batch-export="handleBatchExport"
    >
      <template #filters>
        <!-- 状态筛选 -->
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-xs font-medium text-slate-500">状态：</span>
          <div class="flex flex-wrap items-center gap-1">
            <button type="button"
              v-for="option in statusOptions"
              :key="option.value"
              class="filter-chip"
              :class="{ active: statusFilter === option.value }"
              @click="statusFilter = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </div>
      </template>
    </SmartListBar>

    <!-- 数据表格 -->
    <ui-card class="mb-4 hidden lg:block">
      <ui-table
        :data="filteredAgents"
        :loading="loading"
        :pagination="{
          showTotal: true,
          pageSizeOptions: [10, 20, 50],
          defaultPageSize: 20,
        }"
        :bordered="false"
        row-key="id"
        :row-selection="{ type: 'checkbox' }"
      >
        <template #columns>
          <ui-table-column title="名称" data-index="name">
            <template #cell="{ record }">
              <div class="flex items-center gap-3">
                <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg text-sm text-white" :class="{
                  'bg-emerald-500 shadow-sm shadow-emerald-500/20': record.runtime_state === 'running',
                  'bg-slate-400': record.runtime_state === 'stopped' || !['running', 'error'].includes(record.runtime_state),
                  'bg-red-500 shadow-sm shadow-red-500/20': record.runtime_state === 'error'
                }">
                  <icon-robot />
                </div>
                <div class="flex min-w-0 flex-col gap-0.5">
                  <div class="truncate text-[14px] font-medium text-slate-900">{{ record.name }}</div>
                  <code class="font-mono text-xs text-slate-500">{{ record.plugin_key }}:{{ record.action }}</code>
                </div>
              </div>
            </template>
          </ui-table-column>

          <!-- 运行状态 -->
          <ui-table-column title="运行状态">
            <template #cell="{ record }">
              <StatusTag :status="record.runtime_state" />
              <span v-if="record.desired_state !== record.runtime_state" class="text-xs font-mono text-slate-400 ml-1">
                → {{ record.desired_state }}
              </span>
            </template>
          </ui-table-column>

          <!-- 错误信息 -->
          <ui-table-column title="错误信息" data-index="last_error">
            <template #cell="{ record }">
              <span v-if="record.last_error" class="mt-2 text-sm leading-6 text-red-700">{{ record.last_error }}</span>
              <span v-else class="text-sm text-slate-400">—</span>
            </template>
          </ui-table-column>

          <!-- 快速操作 -->
          <ui-table-column title="操作" align="right">
            <template #cell="{ record }">
              <RowActionsMenu
                :item="record"
                :actions="[
                  { key: 'view', label: '查看详情', icon: 'IconEye' },
                  { key: 'start', label: '启动', icon: 'IconPlayArrow', disabled: record.runtime_state === 'running' },
                  { key: 'stop', label: '停止', icon: 'IconStop', disabled: record.runtime_state === 'stopped', danger: true },
                  { key: 'edit', label: '编辑', icon: 'IconEdit' },
                  { key: 'delete-divider', divider: true },
                  { key: 'delete', label: '删除', icon: 'IconDelete', danger: true },
                ]"
                @action="handleQuickAction"
              />
            </template>
          </ui-table-column>
        </template>

        <!-- 空状态 -->
        <template #empty>
          <ui-empty description="暂无 Agent">
            <ui-button type="primary" @click="router.push(to.agents.create())">
              创建第一个 Agent
            </ui-button>
          </ui-empty>
        </template>
      </ui-table>
    </ui-card>

    <div class="flex flex-col gap-3 lg:hidden">
      <ui-card
        v-for="record in filteredAgents"
        :key="record.id"
        class="rounded-xl border p-5 border-slate-200 bg-white shadow-sm"
      >
        <div class="mb-3 flex items-center gap-3">
          <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg text-sm text-white" :class="{
            'bg-emerald-500 shadow-sm shadow-emerald-500/20': record.runtime_state === 'running',
            'bg-slate-400': record.runtime_state === 'stopped' || !['running', 'error'].includes(record.runtime_state),
            'bg-red-500 shadow-sm shadow-red-500/20': record.runtime_state === 'error'
          }">
            <icon-robot />
          </div>
          <div class="flex min-w-0 flex-1 flex-col gap-0.5">
            <div class="truncate text-sm font-semibold text-slate-900">{{ record.name }}</div>
            <div class="text-xs text-slate-500">
              <code class="font-mono text-xs text-slate-500">{{ record.plugin_key }}:{{ record.action }}</code>
            </div>
          </div>
          <RowActionsMenu
            :item="record"
            :actions="[
              { key: 'view', label: '查看详情', icon: 'IconEye' },
              { key: 'start', label: '启动', icon: 'IconPlayArrow' },
              { key: 'stop', label: '停止', icon: 'IconStop' },
              { key: 'edit', label: '编辑', icon: 'IconEdit' },
              { key: 'delete-divider', divider: true },
              { key: 'delete', label: '删除', icon: 'IconDelete', danger: true },
            ]"
            @action="handleQuickAction"
          />
        </div>

        <div class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">状态</span>
            <div class="flex flex-wrap items-center gap-1">
              <StatusTag :status="record.runtime_state" />
              <span v-if="record.desired_state !== record.runtime_state" class="text-xs font-mono text-slate-400 ml-1">
                → {{ record.desired_state }}
              </span>
            </div>
          </div>

          <div v-if="record.last_error" class="flex items-center justify-between gap-2">
            <span class="w-12 flex-shrink-0 text-xs font-medium text-slate-500">错误</span>
            <div class="flex flex-wrap items-center gap-1">
              <span class="text-xs text-red-600 truncate">{{ record.last_error }}</span>
            </div>
          </div>
        </div>
      </ui-card>

      <ui-card
        v-if="!loading && !filteredAgents.length"
        class="rounded-xl border border-slate-200 bg-white shadow-sm px-5 py-8"
      >
        <ui-empty description="暂无 Agent">
          <ui-button type="primary" @click="router.push(to.agents.create())">
            创建第一个 Agent
          </ui-button>
        </ui-empty>
      </ui-card>
    </div>

    <!-- 详情抽屉 -->
    <DetailDrawer
      v-model:open="drawerVisible"
      :title="currentAgent?.name"
      :loading="loading"
      @refresh="refresh"
      @edit="currentAgent && router.push(to.agents.edit(currentAgent.id))"
      @delete="currentAgent && deleteAgent(currentAgent)"
    >
      <template #detail>
        <div class="rounded-xl border p-4 border-slate-200 bg-white/[56%]">
          <h4 class="mb-3 text-sm font-semibold text-slate-900">基本信息</h4>

          <div class="flex items-start justify-between gap-4 border-b border-slate-100 py-3 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">Agent ID</span>
            <span class="text-sm font-medium text-slate-900">
              <code>{{ currentAgent?.id }}</code>
            </span>
          </div>

          <div class="flex items-start justify-between gap-4 border-b border-slate-100 py-3 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">名称</span>
            <span class="text-sm font-medium text-slate-900">{{ currentAgent?.name }}</span>
          </div>

          <div class="flex items-start justify-between gap-4 border-b border-slate-100 py-3 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">插件</span>
            <span class="text-sm font-medium text-slate-900">
              <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600">{{ currentAgent?.plugin_key }}</code>
            </span>
          </div>

          <div class="flex items-start justify-between gap-4 border-b border-slate-100 py-3 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">动作</span>
            <span class="text-sm font-medium text-slate-900">
              <span class="inline-flex items-center rounded border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-xs font-mono text-slate-600">{{ currentAgent?.action }}</span>
            </span>
          </div>

          <div class="flex items-start justify-between gap-4 border-b border-slate-100 py-3 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">运行状态</span>
            <span class="text-sm font-medium text-slate-900">
              <StatusTag v-if="currentAgent" :status="currentAgent.runtime_state" />
            </span>
          </div>

          <div class="flex items-start justify-between gap-4 border-b border-slate-100 py-3 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">期望状态</span>
            <span class="text-sm font-medium text-slate-900">{{ currentAgent?.desired_state }}</span>
          </div>
        </div>

        <div class="rounded-xl border p-4 border-slate-200 bg-white/[56%]" v-if="currentAgent?.input">
          <h4 class="mb-3 text-sm font-semibold text-slate-900">输入参数</h4>
          <pre class="overflow-auto rounded-xl border border-slate-200 bg-slate-950 p-4 text-xs leading-6 text-slate-300 whitespace-pre-wrap break-all">{{ JSON.stringify(currentAgent.input, null, 2) }}</pre>
        </div>

        <div class="rounded-xl border p-4 border-slate-200 bg-white/[56%]" v-if="currentAgent?.last_error">
          <h4 class="mb-3 text-sm font-semibold text-slate-900">错误信息</h4>
          <div class="mt-2 flex items-start gap-2 rounded-xl border border-red-200 bg-red-50 p-4">
            <icon-close-circle class="h-4 w-4 flex-shrink-0 text-red-500 mt-0.5" />
            <span class="text-sm text-red-600">{{ currentAgent.last_error }}</span>
          </div>
        </div>
      </template>

      <template #footer>
        <div class="flex w-full items-center justify-end gap-2">
          <ui-button
            type="outline"
            :loading="startAgent.loading.value"
            :disabled="currentAgent?.runtime_state === 'running' || currentAgent?.desired_state === 'running'"
            @click="currentAgent && handleStart(currentAgent.id)"
          >
            <template #icon><icon-play-arrow /></template>
            启动
          </ui-button>
          <ui-button
            status="danger"
            :loading="stopAgent.loading.value"
            :disabled="currentAgent?.runtime_state === 'stopped' || currentAgent?.desired_state === 'stopped'"
            @click="currentAgent && handleStop(currentAgent.id)"
          >
            <template #icon><icon-stop /></template>
            停止
          </ui-button>
          <ui-button type="primary" @click="currentAgent && router.push(to.agents.detail(currentAgent.id))">
            查看详情
          </ui-button>
        </div>
      </template>
    </DetailDrawer>
  </div>
</template>
