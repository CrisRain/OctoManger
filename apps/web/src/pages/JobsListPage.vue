<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import {
  IconSchedule, IconClockCircle, IconHistory, IconPlayArrow,
  IconEdit, IconDelete, IconCopy, IconEye
} from "@/lib/icons";

import { PageHeader, SmartListBar, RowActionsMenu, DetailDrawer } from "@/components/index";
import { useJobDefinitions, useEnqueueJobExecution } from "@/composables/useJobs";
import { useMessage, useConfirm, useErrorHandler } from "@/composables";
import { to } from "@/router/registry";

const router = useRouter();
const message = useMessage();
const confirm = useConfirm();
const { withErrorHandler } = useErrorHandler();

const { data: definitions, loading, refresh } = useJobDefinitions();
const enqueue = useEnqueueJobExecution();

// 筛选和搜索
const scheduleFilter = ref<string>();
const searchKeyword = ref("");

// 过滤任务
const filteredJobs = computed(() => {
  let result = definitions.value;

  // 按调度类型筛选
  if (scheduleFilter.value === "scheduled") {
    result = result.filter(item => item.schedule?.cron_expression);
  } else if (scheduleFilter.value === "manual") {
    result = result.filter(item => !item.schedule?.cron_expression);
  }

  // 搜索过滤
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase();
    result = result.filter(item =>
      item.name.toLowerCase().includes(keyword) ||
      item.key.toLowerCase().includes(keyword) ||
      item.plugin_key.toLowerCase().includes(keyword)
    );
  }

  return result;
});

// 抽屉状态
const drawerVisible = ref(false);
const currentJob = ref<any>(null);

// 快速操作
async function handleQuickAction(key: string, job: any) {
  switch (key) {
    case "view":
      showJobDetail(job);
      break;
    case "edit":
      router.push(to.jobs.edit(job.id));
      break;
    case "execute":
      await handleEnqueue(job.id);
      break;
    case "copy":
      await copyToClipboard(job.key);
      break;
    case "delete":
      await deleteJob(job);
      break;
  }
}

// 显示详情抽屉
function showJobDetail(job: any) {
  currentJob.value = job;
  drawerVisible.value = true;
}

// 立即执行任务
async function handleEnqueue(id: number) {
  await withErrorHandler(
    async () => {
      await enqueue.execute(id);
      message.success("已加入执行队列");
      await refresh();
    },
    { action: "执行任务", showSuccess: true }
  );
}

// 复制到剪贴板
async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    message.success("已复制到剪贴板");
  } catch {
    message.error("复制失败");
  }
}

// 删除任务
async function deleteJob(job: any) {
  const confirmed = await confirm.confirmDelete(job.name);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      // TODO: 调用API删除
      message.success(`已删除任务: ${job.name}`);
      await refresh();
    },
    { action: "删除", showSuccess: true }
  );
}

// 批量删除
async function handleBatchDelete(items: any[]) {
  const confirmed = await confirm.confirm(`确定要删除选中的 ${items.length} 个任务吗？`);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      // TODO: 调用批量删除API
      message.success(`已删除 ${items.length} 个任务`);
      await refresh();
    },
    { action: "批量删除", showSuccess: true }
  );
}

// 批量导出
async function handleBatchExport(items: any[]) {
  const data = JSON.stringify(items, null, 2);
  const blob = new Blob([data], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `jobs-${Date.now()}.json`;
  link.click();
  URL.revokeObjectURL(url);
  message.success(`已导出 ${items.length} 个任务`);
}
</script>

<template>
  <div class="page-container jobs-list-page">
    <PageHeader
      title="定时任务"
      subtitle="管理自动化任务，支持 Cron 定时调度和手动触发"
      icon-bg="linear-gradient(135deg, rgba(202,138,4,0.12), rgba(234,179,8,0.12))"
      icon-color="var(--icon-yellow)"
    >
      <template #icon><icon-schedule /></template>
      <template #actions>
        <ui-button type="primary" @click="router.push(to.jobs.create())">
          <template #icon><icon-plus /></template>
          创建任务
        </ui-button>
      </template>
    </PageHeader>

    <!-- 智能工具栏 -->
    <SmartListBar
      :data="filteredJobs"
      :loading="loading"
      v-model:search="searchKeyword"
      @refresh="refresh"
      @batch-delete="handleBatchDelete"
      @batch-export="handleBatchExport"
    >
      <template #filters>
        <!-- 调度类型筛选 -->
        <div class="filter-group">
          <span class="filter-label">调度：</span>
          <div class="filter-options">
            <button type="button"
              class="filter-option"
              :class="{ 'filter-option--active': !scheduleFilter }"
              @click="scheduleFilter = ''"
            >
              全部
            </button>
            <button type="button"
              class="filter-option"
              :class="{ 'filter-option--active': scheduleFilter === 'scheduled' }"
              @click="scheduleFilter = 'scheduled'"
            >
              定时
            </button>
            <button type="button"
              class="filter-option"
              :class="{ 'filter-option--active': scheduleFilter === 'manual' }"
              @click="scheduleFilter = 'manual'"
            >
              手动
            </button>
          </div>
        </div>
      </template>

      <template #extra-actions>
        <ui-button @click="router.push(to.jobs.executions())">
          <template #icon><icon-history /></template>
          执行记录
        </ui-button>
      </template>
    </SmartListBar>

    <!-- 数据表格 -->
    <ui-card class="data-grid-card table-desktop-only">
      <ui-table
        class="premium-table"
        :data="filteredJobs"
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
          <ui-table-column title="任务名称" data-index="name" :width="240">
            <template #cell="{ record }">
              <div class="identifier-cell">
                <div class="icon-box icon-yellow">
                  <icon-schedule />
                </div>
                <div class="grow-content">
                  <div class="identifier-text">{{ record.name }}</div>
                  <code v-if="record.key" class="sub-text mono">{{ record.key }}</code>
                </div>
              </div>
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

          <!-- 调度计划 -->
          <ui-table-column title="调度计划">
            <template #cell="{ record }">
              <span v-if="record.schedule?.cron_expression" class="cron-badge">
                <icon-clock-circle class="cron-icon" />
                {{ record.schedule.cron_expression }}
              </span>
              <span v-else class="text-muted">手动触发</span>
            </template>
          </ui-table-column>

          <!-- 快速操作 -->
          <ui-table-column title="操作" :width="80" align="right">
            <template #cell="{ record }">
              <RowActionsMenu
                :item="record"
                :actions="[
                  { key: 'view', label: '查看详情', icon: 'IconEye' },
                  { key: 'execute', label: '立即执行', icon: 'IconPlayArrow' },
                  { key: 'edit', label: '编辑', icon: 'IconEdit' },
                  { key: 'copy', label: '复制Key', icon: 'IconCopy' },
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
          <ui-empty description="暂无任务">
            <ui-button type="primary" @click="router.push(to.jobs.create())">
              创建第一个任务
            </ui-button>
          </ui-empty>
        </template>
      </ui-table>
    </ui-card>

    <div class="mobile-list">
      <ui-card
        v-for="record in filteredJobs"
        :key="record.id"
        class="mobile-list-card"
      >
        <div class="mobile-list-header">
          <div class="icon-box icon-yellow">
            <icon-schedule />
          </div>
          <div class="mobile-list-title-group">
            <div class="mobile-list-title">{{ record.name }}</div>
            <div class="mobile-list-subtitle">
              <code v-if="record.key" class="sub-text mono">{{ record.key }}</code>
            </div>
          </div>
          <RowActionsMenu
            :item="record"
            :actions="[
              { key: 'view', label: '查看详情', icon: 'IconEye' },
              { key: 'execute', label: '立即执行', icon: 'IconPlayArrow' },
              { key: 'edit', label: '编辑', icon: 'IconEdit' },
              { key: 'copy', label: '复制Key', icon: 'IconCopy' },
              { key: 'delete-divider', divider: true },
              { key: 'delete', label: '删除', icon: 'IconDelete', danger: true },
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
            <span class="mobile-list-label">调度</span>
            <div class="mobile-list-value">
              <span v-if="record.schedule?.cron_expression" class="cron-badge">
                <icon-clock-circle class="cron-icon" />
                {{ record.schedule.cron_expression }}
              </span>
              <span v-else class="text-muted">手动触发</span>
            </div>
          </div>
        </div>
      </ui-card>

      <ui-card
        v-if="!loading && !filteredJobs.length"
        class="mobile-list-card mobile-list-empty-card"
      >
        <ui-empty description="暂无任务">
          <ui-button type="primary" @click="router.push(to.jobs.create())">
            创建第一个任务
          </ui-button>
        </ui-empty>
      </ui-card>
    </div>

    <!-- 详情抽屉 -->
    <DetailDrawer
      v-model:open="drawerVisible"
      :title="currentJob?.name"
      @edit="currentJob && router.push(to.jobs.edit(currentJob.id))"
      @delete="currentJob && deleteJob(currentJob)"
    >
      <template #detail>
        <div class="detail-section">
          <h4 class="detail-section-title">基本信息</h4>

          <div class="detail-row">
            <span class="detail-label">任务ID</span>
            <span class="detail-value">
              <code>{{ currentJob?.id }}</code>
            </span>
          </div>

          <div class="detail-row">
            <span class="detail-label">标识符</span>
            <span class="detail-value">
              <code class="key-badge">{{ currentJob?.key }}</code>
            </span>
          </div>

          <div class="detail-row">
            <span class="detail-label">插件</span>
            <span class="detail-value">
              <code class="key-badge">{{ currentJob?.plugin_key }}</code>
            </span>
          </div>

          <div class="detail-row">
            <span class="detail-label">动作</span>
            <span class="detail-value">
              <span class="action-tag">{{ currentJob?.action }}</span>
            </span>
          </div>

          <div class="detail-row" v-if="currentJob?.schedule?.cron_expression">
            <span class="detail-label">调度表达式</span>
            <span class="detail-value">
              <span class="cron-badge">
                <icon-clock-circle class="cron-icon" />
                {{ currentJob.schedule.cron_expression }}
              </span>
            </span>
          </div>
        </div>

        <div class="detail-section" v-if="currentJob?.input">
          <h4 class="detail-section-title">输入参数</h4>
          <div class="config-box">
            <pre>{{ JSON.stringify(currentJob.input, null, 2) }}</pre>
          </div>
        </div>
      </template>

      <template #footer>
        <div class="drawer-footer-actions">
          <ui-button @click="drawerVisible = false">关闭</ui-button>
          <ui-button
            type="outline"
            :loading="enqueue.loading.value"
            @click="currentJob && handleEnqueue(currentJob.id)"
          >
            <template #icon><icon-play-arrow /></template>
            立即执行
          </ui-button>
          <ui-button type="primary" @click="currentJob && router.push(to.jobs.edit(currentJob.id))">
            编辑任务
          </ui-button>
        </div>
      </template>
    </DetailDrawer>
  </div>
</template>
