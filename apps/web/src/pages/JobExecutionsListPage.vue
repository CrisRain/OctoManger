<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { IconHistory, IconRefresh, IconEye } from "@/lib/icons";

import { PageHeader, SmartListBar, RowActionsMenu, DetailDrawer, StatusTag } from "@/components/index";
import { useJobExecutions } from "@/composables/useJobs";
import { useMessage } from "@/composables";
import { to } from "@/router/registry";

const router = useRouter();
const message = useMessage();

const { data: executions, loading, refresh } = useJobExecutions();

// 状态筛选
const statusFilter = ref<string>();
const searchKeyword = ref("");

// 过滤执行记录
const filteredExecutions = computed(() => {
  let result = executions.value;

  if (statusFilter.value) {
    result = result.filter(item => item.status === statusFilter.value);
  }

  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase();
    result = result.filter(item =>
      String(item.id).includes(keyword) ||
      item.definition_name?.toLowerCase().includes(keyword) ||
      item.plugin_key?.toLowerCase().includes(keyword) ||
      item.action?.toLowerCase().includes(keyword)
    );
  }

  return result;
});

// 状态配置
const statusOptions = [
  { label: "全部", value: "" },
  { label: "待执行", value: "pending" },
  { label: "运行中", value: "running" },
  { label: "成功", value: "success" },
  { label: "失败", value: "failed" },
  { label: "已取消", value: "cancelled" },
];

// 抽屉状态
const drawerVisible = ref(false);
const currentExecution = ref<any>(null);

// 快速操作
function handleQuickAction(key: string, execution: any) {
  switch (key) {
    case "view":
      showExecutionDetail(execution);
      break;
    case "retry":
      // TODO: 实现重试逻辑
      message.info("重试功能开发中");
      break;
  }
}

// 显示详情抽屉
function showExecutionDetail(execution: any) {
  currentExecution.value = execution;
  drawerVisible.value = true;
}

// 跳转到详情页（保留原有导航）
function goToDetail(id: number) {
  router.push(to.jobs.executionDetail(id));
}

// 复制ID
async function copyId(id: number) {
  try {
    await navigator.clipboard.writeText(String(id));
    message.success("已复制ID");
  } catch {
    message.error("复制失败");
  }
}
</script>

<template>
  <div class="page-container executions-list-page">
    <PageHeader
      title="执行记录"
      subtitle="查看所有任务的历史执行状态和日志"
      icon-bg="linear-gradient(135deg, rgba(20,184,166,0.16), rgba(45,212,191,0.16))"
      icon-color="var(--icon-purple)"
    >
      <template #icon><icon-history /></template>
      <template #actions>
        <ui-button @click="router.push(to.jobs.list())">返回任务管理</ui-button>
      </template>
    </PageHeader>

    <!-- 智能工具栏 -->
    <SmartListBar
      :data="filteredExecutions"
      :loading="loading"
      v-model:search="searchKeyword"
      @refresh="refresh"
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
        :data="filteredExecutions"
        :loading="loading"
        :pagination="{
          showTotal: true,
          pageSizeOptions: [10, 20, 50],
          defaultPageSize: 20,
        }"
        :bordered="false"
        row-key="id"
      >
        <template #columns>
          <!-- 编号 -->
          <ui-table-column title="编号">
            <template #cell="{ record }">
              <code class="execution-id">#{{ record.id }}</code>
            </template>
          </ui-table-column>

          <!-- 任务名称 -->
          <ui-table-column title="任务名称" data-index="definition_name">
            <template #cell="{ record }">
              <span class="job-name">{{ record.definition_name }}</span>
            </template>
          </ui-table-column>

          <!-- 插件 · 动作 -->
          <ui-table-column title="插件 · 动作">
            <template #cell="{ record }">
              <span class="plugin-action">
                <span class="plugin-key">{{ record.plugin_key }}</span>
                <span class="plugin-dot">·</span>
                <span class="action-tag">{{ record.action }}</span>
              </span>
            </template>
          </ui-table-column>

          <!-- 状态 -->
          <ui-table-column title="状态">
            <template #cell="{ record }">
              <StatusTag :status="record.status" />
            </template>
          </ui-table-column>

          <!-- 节点 -->
          <ui-table-column title="节点">
            <template #cell="{ record }">
              <code v-if="record.worker_id" class="worker-id">{{ record.worker_id }}</code>
              <span v-else class="text-muted">未分配</span>
            </template>
          </ui-table-column>

          <!-- 快速操作 -->
          <ui-table-column title="操作" :width="80" align="right">
            <template #cell="{ record }">
              <RowActionsMenu
                :item="record"
                :actions="[
                  { key: 'view', label: '查看详情', icon: 'IconEye' },
                ]"
                @action="handleQuickAction"
              />
            </template>
          </ui-table-column>
        </template>

        <!-- 空状态 -->
        <template #empty>
          <ui-empty description="暂无执行记录">
            <ui-button type="primary" @click="router.push(to.jobs.list())">
              创建并执行任务
            </ui-button>
          </ui-empty>
        </template>
      </ui-table>
    </ui-card>

    <div class="mobile-list">
      <ui-card
        v-for="record in filteredExecutions"
        :key="record.id"
        class="mobile-list-card"
      >
        <div class="mobile-list-header">
          <div class="icon-box icon-purple">
            <icon-history />
          </div>
          <div class="mobile-list-title-group">
            <div class="mobile-list-title">{{ record.definition_name || `执行记录 #${record.id}` }}</div>
            <div class="mobile-list-subtitle">
              <code class="execution-id">#{{ record.id }}</code>
            </div>
          </div>
          <RowActionsMenu
            :item="record"
            :actions="[
              { key: 'view', label: '查看详情', icon: 'IconEye' },
            ]"
            @action="handleQuickAction"
          />
        </div>

        <div class="mobile-list-meta">
          <div class="mobile-list-meta-row">
            <span class="mobile-list-label">插件</span>
            <div class="mobile-list-value">
              <span class="plugin-action">
                <span class="plugin-key">{{ record.plugin_key }}</span>
                <span class="plugin-dot">·</span>
                <span class="action-tag">{{ record.action }}</span>
              </span>
            </div>
          </div>

          <div class="mobile-list-meta-row">
            <span class="mobile-list-label">状态</span>
            <div class="mobile-list-value">
              <StatusTag :status="record.status" />
            </div>
          </div>

          <div class="mobile-list-meta-row">
            <span class="mobile-list-label">节点</span>
            <div class="mobile-list-value">
              <code v-if="record.worker_id" class="worker-id">{{ record.worker_id }}</code>
              <span v-else class="text-muted">未分配</span>
            </div>
          </div>
        </div>
      </ui-card>

      <ui-card
        v-if="!loading && !filteredExecutions.length"
        class="mobile-list-card mobile-list-empty-card"
      >
        <ui-empty description="暂无执行记录">
          <ui-button type="primary" @click="router.push(to.jobs.list())">
            创建并执行任务
          </ui-button>
        </ui-empty>
      </ui-card>
    </div>

    <!-- 详情抽屉 -->
    <DetailDrawer
      v-model:open="drawerVisible"
      :title="`执行记录 #${currentExecution?.id}`"
      @edit="goToDetail(currentExecution?.id)"
    >
      <template #detail>
        <div class="detail-section">
          <h4 class="detail-section-title">执行信息</h4>

          <div class="detail-row">
            <span class="detail-label">执行ID</span>
            <span class="detail-value">
              <code>{{ currentExecution?.id }}</code>
            </span>
          </div>

          <div class="detail-row">
            <span class="detail-label">任务名称</span>
            <span class="detail-value">{{ currentExecution?.definition_name }}</span>
          </div>

          <div class="detail-row">
            <span class="detail-label">插件</span>
            <span class="detail-value">
              <code class="key-badge">{{ currentExecution?.plugin_key }}</code>
            </span>
          </div>

          <div class="detail-row">
            <span class="detail-label">动作</span>
            <span class="detail-value">
              <span class="action-tag">{{ currentExecution?.action }}</span>
            </span>
          </div>

          <div class="detail-row">
            <span class="detail-label">状态</span>
            <span class="detail-value">
              <StatusTag v-if="currentExecution" :status="currentExecution.status" />
            </span>
          </div>

          <div class="detail-row">
            <span class="detail-label">执行节点</span>
            <span class="detail-value">
              <code v-if="currentExecution?.worker_id">{{ currentExecution.worker_id }}</code>
              <span v-else class="text-muted">未分配</span>
            </span>
          </div>
        </div>

        <div class="detail-section" v-if="currentExecution?.input">
          <h4 class="detail-section-title">输入参数</h4>
          <div class="config-box">
            <pre>{{ JSON.stringify(currentExecution.input, null, 2) }}</pre>
          </div>
        </div>
      </template>

      <template #activity>
        <ui-empty description="暂无活动记录" />
      </template>

      <template #footer>
        <div class="drawer-footer-actions">
          <ui-button @click="copyId(currentExecution?.id)">复制ID</ui-button>
          <ui-button type="primary" @click="goToDetail(currentExecution?.id)">
            查看完整日志
          </ui-button>
        </div>
      </template>
    </DetailDrawer>
  </div>
</template>
