<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import {
  IconRobot, IconPlayArrow, IconStop, IconEye,
  IconEdit, IconDelete
} from "@/lib/icons";

import { PageHeader, SmartListBar, RowActionsMenu, DetailDrawer } from "@/components/index";
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
const currentAgent = ref<any>(null);

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
  currentAgent.value = agent;
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
    { action: "启动Agent", showSuccess: true }
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
    { action: "停止Agent", showSuccess: true }
  );
}

// 删除 Agent
async function deleteAgent(agent: any) {
  const confirmed = await confirm.confirmDelete(agent.name);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      // TODO: 调用API删除
      message.success(`已删除Agent: ${agent.name}`);
      await refresh();
    },
    { action: "删除", showSuccess: true }
  );
}

// 批量操作
async function handleBatchStop(items: any[]) {
  const confirmed = await confirm.confirm(`确定要停止选中的 ${items.length} 个Agent吗？`);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      // TODO: 调用批量停止API
      message.success(`已发送停止指令到 ${items.length} 个Agent`);
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
  message.success(`已导出 ${items.length} 个Agent`);
}
</script>

<template>
  <div class="page-container agents-list-page">
    <PageHeader
      title="后台任务"
      subtitle="管理持续运行的后台任务进程"
      icon-bg="linear-gradient(135deg, rgba(20,184,166,0.16), rgba(45,212,191,0.16))"
      icon-color="var(--icon-purple)"
    >
      <template #icon><icon-robot /></template>
      <template #actions>
        <ui-button type="primary" @click="router.push(to.agents.create())">
          <template #icon><icon-plus /></template>
          创建任务
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
        <div class="filter-group">
          <span class="filter-label">状态：</span>
          <div class="filter-options">
            <button type="button"
              v-for="option in statusOptions"
              :key="option.value"
              class="filter-option"
              :class="{ 'filter-option--active': statusFilter === option.value }"
              @click="statusFilter = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </div>
      </template>
    </SmartListBar>

    <!-- 数据表格 -->
    <ui-card class="data-grid-card table-desktop-only">
      <ui-table
        class="premium-table"
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
          <!-- 任务名称 -->
          <ui-table-column title="任务名称" data-index="name" :width="280">
            <template #cell="{ record }">
              <div class="identifier-cell">
                <div class="icon-box" :class="`agent-icon--${record.runtime_state}`">
                  <icon-robot />
                </div>
                <div class="grow-content">
                  <div class="identifier-text">{{ record.name }}</div>
                  <code class="sub-text mono">{{ record.plugin_key }}:{{ record.action }}</code>
                </div>
              </div>
            </template>
          </ui-table-column>

          <!-- 运行状态 -->
          <ui-table-column title="运行状态">
            <template #cell="{ record }">
              <div class="status-cell">
                <span
                  class="status-dot"
                  :class="{
                    'status-dot--running': record.runtime_state === 'running',
                    'status-dot--stopped': record.runtime_state === 'stopped',
                    'status-dot--error': record.runtime_state === 'error',
                  }"
                />
                <span
                  class="status-badge"
                  :class="{
                    'status-badge--running': record.runtime_state === 'running',
                    'status-badge--stopped': record.runtime_state === 'stopped',
                    'status-badge--error': record.runtime_state === 'error',
                  }"
                >
                  {{ record.runtime_state }}
                </span>
                <span v-if="record.desired_state !== record.runtime_state" class="status-arrow">
                  → {{ record.desired_state }}
                </span>
              </div>
              <div v-if="record.last_error" class="error-line">
                <icon-close-circle class="error-icon" />
                {{ record.last_error }}
              </div>
            </template>
          </ui-table-column>

          <!-- 错误信息 -->
          <ui-table-column title="错误信息" data-index="last_error" :width="300">
            <template #cell="{ record }">
              <span v-if="record.last_error" class="error-text">{{ record.last_error }}</span>
              <span v-else class="text-muted">—</span>
            </template>
          </ui-table-column>

          <!-- 快速操作 -->
          <ui-table-column title="操作" :width="80" align="right">
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
          <ui-empty description="暂无后台任务">
            <ui-button type="primary" @click="router.push(to.agents.create())">
              创建第一个任务
            </ui-button>
          </ui-empty>
        </template>
      </ui-table>
    </ui-card>

    <div class="mobile-list">
      <ui-card
        v-for="record in filteredAgents"
        :key="record.id"
        class="mobile-list-card"
      >
        <div class="mobile-list-header">
          <div class="icon-box" :class="`agent-icon--${record.runtime_state}`">
            <icon-robot />
          </div>
          <div class="mobile-list-title-group">
            <div class="mobile-list-title">{{ record.name }}</div>
            <div class="mobile-list-subtitle">
              <code class="sub-text mono">{{ record.plugin_key }}:{{ record.action }}</code>
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

        <div class="mobile-list-meta">
          <div class="mobile-list-meta-row">
            <span class="mobile-list-label">状态</span>
            <div class="mobile-list-value">
              <div class="status-cell">
                <span
                  class="status-dot"
                  :class="{
                    'status-dot--running': record.runtime_state === 'running',
                    'status-dot--stopped': record.runtime_state === 'stopped',
                    'status-dot--error': record.runtime_state === 'error',
                  }"
                />
                <span
                  class="status-badge"
                  :class="{
                    'status-badge--running': record.runtime_state === 'running',
                    'status-badge--stopped': record.runtime_state === 'stopped',
                    'status-badge--error': record.runtime_state === 'error',
                  }"
                >
                  {{ record.runtime_state }}
                </span>
                <span v-if="record.desired_state !== record.runtime_state" class="status-arrow">
                  → {{ record.desired_state }}
                </span>
              </div>
            </div>
          </div>

          <div v-if="record.last_error" class="mobile-list-meta-row">
            <span class="mobile-list-label">错误</span>
            <div class="mobile-list-value">
              <span class="error-line">{{ record.last_error }}</span>
            </div>
          </div>
        </div>
      </ui-card>

      <ui-card
        v-if="!loading && !filteredAgents.length"
        class="mobile-list-card mobile-list-empty-card"
      >
        <ui-empty description="暂无后台任务">
          <ui-button type="primary" @click="router.push(to.agents.create())">
            创建第一个任务
          </ui-button>
        </ui-empty>
      </ui-card>
    </div>

    <!-- 详情抽屉 -->
    <DetailDrawer
      v-model:open="drawerVisible"
      :title="currentAgent?.name"
      @edit="currentAgent && router.push(to.agents.edit(currentAgent.id))"
      @delete="currentAgent && deleteAgent(currentAgent)"
    >
      <template #detail>
        <div class="detail-section">
          <h4 class="detail-section-title">基本信息</h4>

          <div class="detail-row">
            <span class="detail-label">Agent ID</span>
            <span class="detail-value">
              <code>{{ currentAgent?.id }}</code>
            </span>
          </div>

          <div class="detail-row">
            <span class="detail-label">任务名称</span>
            <span class="detail-value">{{ currentAgent?.name }}</span>
          </div>

          <div class="detail-row">
            <span class="detail-label">插件</span>
            <span class="detail-value">
              <code class="key-badge">{{ currentAgent?.plugin_key }}</code>
            </span>
          </div>

          <div class="detail-row">
            <span class="detail-label">动作</span>
            <span class="detail-value">
              <span class="action-tag">{{ currentAgent?.action }}</span>
            </span>
          </div>

          <div class="detail-row">
            <span class="detail-label">运行状态</span>
            <span class="detail-value">
              <span
                class="status-badge"
                :class="{
                  'status-badge--running': currentAgent?.runtime_state === 'running',
                  'status-badge--stopped': currentAgent?.runtime_state === 'stopped',
                  'status-badge--error': currentAgent?.runtime_state === 'error',
                }"
              >
                {{ currentAgent?.runtime_state }}
              </span>
            </span>
          </div>

          <div class="detail-row">
            <span class="detail-label">期望状态</span>
            <span class="detail-value">{{ currentAgent?.desired_state }}</span>
          </div>
        </div>

        <div class="detail-section" v-if="currentAgent?.input">
          <h4 class="detail-section-title">输入参数</h4>
          <div class="config-box">
            <pre>{{ JSON.stringify(currentAgent.input, null, 2) }}</pre>
          </div>
        </div>

        <div class="detail-section" v-if="currentAgent?.last_error">
          <h4 class="detail-section-title">错误信息</h4>
          <div class="error-box">
            <icon-close-circle class="error-icon" />
            <span>{{ currentAgent.last_error }}</span>
          </div>
        </div>
      </template>

      <template #footer>
        <div class="drawer-footer-actions">
          <ui-button
            type="outline"
            :loading="startAgent.loading.value"
            :disabled="currentAgent?.runtime_state === 'running'"
            @click="currentAgent && handleStart(currentAgent.id)"
          >
            <template #icon><icon-play-arrow /></template>
            启动
          </ui-button>
          <ui-button
            status="danger"
            :loading="stopAgent.loading.value"
            :disabled="currentAgent?.runtime_state === 'stopped'"
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
