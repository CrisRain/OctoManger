<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { IconEdit, IconInfoCircle } from "@/lib/icons";

import { useJobDefinitions } from "@/composables/useJobs";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const route = useRoute();
const router = useRouter();
const jobId = Number(route.params.id);

const { data: definitions, loading } = useJobDefinitions();
const job = computed(() => definitions.value.find((j) => j.id === jobId));

function handleCancel() {
  router.push(to.jobs.detail(jobId));
}
</script>

<template>
  <div class="page-container job-edit-page">
    <PageHeader
      title="编辑任务"
      :subtitle="job ? `正在编辑 ${job.name}` : '任务详情加载中...'"
      icon-bg="linear-gradient(135deg, rgba(202,138,4,0.12), rgba(234,179,8,0.12))"
      icon-color="var(--icon-yellow)"
      :back-to="to.jobs.detail(jobId)"
      back-label="返回任务详情"
    >
      <template #icon><icon-edit /></template>
      <template #actions>
        <ui-button @click="handleCancel">取消</ui-button>
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

    <!-- 开发中提示 -->
    <ui-card v-else class="info-card">
      <template #title>
        <div class="card-title-row">
          <icon-info-circle class="card-title-icon" />
          <span>功能说明</span>
        </div>
      </template>

      <div class="notice-notice">
        <div class="notice-icon">
          <icon-info-circle />
        </div>
        <div class="notice-content">
          <p class="notice-title">任务编辑功能正在开发中</p>
          <p class="notice-body">
            目前您可以通过以下方式修改任务：
          </p>
          <ul class="notice-list">
            <li>通过 API 端点直接更新任务定义</li>
            <li>删除现有任务后重新创建</li>
            <li>联系系统管理员进行配置</li>
          </ul>
          <div class="notice-actions">
            <ui-button type="outline" size="small" @click="router.push(to.jobs.detail(jobId))">
              查看任务详情
            </ui-button>
            <ui-button type="primary" size="small" @click="router.push(to.jobs.create())">
              创建新任务
            </ui-button>
          </div>
        </div>
      </div>

      <!-- 只读信息展示 -->
      <div class="job-info-section">
        <h3 class="section-title">当前配置（只读）</h3>

        <div class="info-grid">
          <div class="info-item">
            <span class="info-label">任务标识符</span>
            <code class="info-value">{{ job.key }}</code>
          </div>

          <div class="info-item">
            <span class="info-label">任务名称</span>
            <span class="info-value">{{ job.name }}</span>
          </div>

          <div class="info-item">
            <span class="info-label">插件</span>
            <code class="info-value plugin-badge">{{ job.plugin_key }}</code>
          </div>

          <div class="info-item">
            <span class="info-label">动作</span>
            <span class="info-value action-badge">{{ job.action }}</span>
          </div>

          <div class="info-item info-item--full" v-if="job.schedule?.cron_expression">
            <span class="info-label">Cron 表达式</span>
            <code class="info-value cron-badge">{{ job.schedule.cron_expression }}</code>
          </div>

          <div class="info-item info-item--full" v-if="job.input">
            <span class="info-label">输入参数</span>
            <pre class="info-value json-box">{{ JSON.stringify(job.input, null, 2) }}</pre>
          </div>
        </div>
      </div>
    </ui-card>
  </div>
</template>
