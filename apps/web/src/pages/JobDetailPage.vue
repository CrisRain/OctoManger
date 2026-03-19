<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  IconSchedule, IconPlayArrow, IconEdit, IconDelete,
  IconClockCircle, IconCopy
} from "@/lib/icons";

import { useJobDefinitions, useEnqueueJobExecution } from "@/composables/useJobs";
import { useMessage, useConfirm, useErrorHandler } from "@/composables";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const confirm = useConfirm();
const { withErrorHandler } = useErrorHandler();

const jobId = Number(route.params.id);
const { data: definitions, loading, refresh } = useJobDefinitions();
const enqueue = useEnqueueJobExecution();

const job = computed(() => definitions.value.find((j) => j.id === jobId));

// 复制到剪贴板
async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    message.success("已复制到剪贴板");
  } catch {
    message.error("复制失败");
  }
}

// 立即执行任务
async function handleEnqueue() {
  if (!job.value) return;

  await withErrorHandler(
    async () => {
      await enqueue.execute(job.value!.id);
      message.success("已加入执行队列");
    },
    { action: "执行任务", showSuccess: true }
  );
}

// 删除任务
async function deleteJob() {
  if (!job.value) return;

  const confirmed = await confirm.confirmDelete(job.value.name);
  if (!confirmed) return;

  await withErrorHandler(
    async () => {
      // TODO: 调用API删除
      message.success(`已删除任务: ${job.value!.name}`);
      router.push(to.jobs.list());
    },
    { action: "删除", showSuccess: true }
  );
}

</script>

<template>
  <div class="page-container job-detail-page">
    <PageHeader
      :title="job ? job.name : '任务详情'"
      icon-bg="linear-gradient(135deg, rgba(202,138,4,0.12), rgba(234,179,8,0.12))"
      icon-color="var(--icon-yellow)"
      :back-to="to.jobs.list()"
      back-label="返回任务列表"
    >
      <template #icon><icon-schedule /></template>
      <template #subtitle>
        <code v-if="job" class="key-badge">{{ job.key }}</code>
      </template>
      <template #actions>
        <div v-if="job" class="job-header-actions">
          <ui-button
            type="outline"
            :loading="enqueue.loading.value"
            @click="handleEnqueue"
          >
            <template #icon><icon-play-arrow /></template>
            立即执行
          </ui-button>
          <ui-button @click="copyToClipboard(job.key)">
            <template #icon><icon-copy /></template>
            复制 Key
          </ui-button>
          <ui-button type="primary" @click="router.push(to.jobs.edit(job.id))">
            <template #icon><icon-edit /></template>
            编辑
          </ui-button>
        </div>
      </template>
    </PageHeader>

    <!-- 任务未找到 -->
    <div v-if="loading" class="empty-card center-card-loading"><ui-spin :size="36" /></div>
    <ui-card v-else-if="!job" class="empty-card">
      <ui-empty description="未找到该任务">
        <ui-button type="primary" @click="router.push(to.jobs.list())">
          返回任务列表
        </ui-button>
      </ui-empty>
    </ui-card>

    <!-- 任务详情 -->
    <div v-else class="detail-content">
      <!-- 基本信息 -->
      <ui-card class="detail-card">
        <template #title>
          <div class="card-title-row">
            <icon-schedule class="card-title-icon" />
            <span>基本信息</span>
          </div>
        </template>

        <div class="info-grid">
          <div class="info-item">
            <span class="info-label">任务ID</span>
            <span class="info-value">
              <code>{{ job.id }}</code>
            </span>
          </div>

          <div class="info-item">
            <span class="info-label">标识符</span>
            <span class="info-value">
              <code class="key-badge">{{ job.key }}</code>
            </span>
          </div>

          <div class="info-item">
            <span class="info-label">任务名称</span>
            <span class="info-value">{{ job.name }}</span>
          </div>

          <div class="info-item">
            <span class="info-label">插件</span>
            <span class="info-value">
              <code class="plugin-badge">{{ job.plugin_key }}</code>
            </span>
          </div>

          <div class="info-item">
            <span class="info-label">动作</span>
            <span class="info-value">
              <span class="action-badge">{{ job.action }}</span>
            </span>
          </div>

          <div class="info-item" v-if="job.schedule?.cron_expression">
            <span class="info-label">调度类型</span>
            <span class="info-value status-badge status-badge--success">
              <icon-clock-circle />
              定时任务
            </span>
          </div>

          <div class="info-item" v-else>
            <span class="info-label">调度类型</span>
            <span class="info-value status-badge status-badge--manual">
              手动触发
            </span>
          </div>

          <div class="info-item info-item--full" v-if="job.schedule?.cron_expression">
            <span class="info-label">Cron 表达式</span>
            <span class="info-value">
              <code class="cron-badge">
                <icon-clock-circle />
                {{ job.schedule.cron_expression }}
              </code>
            </span>
          </div>

          <div class="info-item info-item--full" v-if="job.schedule?.timezone">
            <span class="info-label">时区</span>
            <span class="info-value">{{ job.schedule.timezone }}</span>
          </div>
        </div>
      </ui-card>

      <!-- 输入参数 -->
      <ui-card class="detail-card" v-if="job.input">
        <template #title>
          <div class="card-title-row">
            <icon-code class="card-title-icon" />
            <span>输入参数</span>
          </div>
        </template>

        <div class="config-box">
          <pre>{{ JSON.stringify(job.input, null, 2) }}</pre>
        </div>
      </ui-card>

      <!-- 快速操作 -->
      <ui-card class="detail-card">
        <template #title>
          <div class="card-title-row">
            <icon-thunderbolt class="card-title-icon" />
            <span>快速操作</span>
          </div>
        </template>

        <ui-button
          type="primary"
          size="large"
          :loading="enqueue.loading.value"
          @click="handleEnqueue"
        >
          <template #icon><icon-play-arrow /></template>
          立即执行此任务
        </ui-button>

        <div class="action-links">
          <button type="button" class="action-link" @click="router.push(to.jobs.edit(job.id))">
            <icon-edit />
            编辑任务配置
          </button>
          <button type="button" class="action-link" @click="copyToClipboard(job.key)">
            <icon-copy />
            复制任务标识符
          </button>
          <button type="button" class="action-link danger" @click="deleteJob">
            <icon-delete />
            删除此任务
          </button>
        </div>
      </ui-card>

      <!-- 执行记录 -->
      <ui-card class="detail-card">
        <template #title>
          <div class="card-title-row">
            <icon-history class="card-title-icon" />
            <span>执行记录</span>
          </div>
        </template>

        <div class="executions-preview">
          <ui-empty description="暂无执行记录">
            <ui-button type="outline" @click="handleEnqueue">
              <template #icon><icon-play-arrow /></template>
              执行一次
            </ui-button>
          </ui-empty>
        </div>

        <template #extra>
          <ui-button type="text" @click="router.push(to.jobs.executions())">
            查看全部
            <icon-right />
          </ui-button>
        </template>
      </ui-card>
    </div>
  </div>
</template>
