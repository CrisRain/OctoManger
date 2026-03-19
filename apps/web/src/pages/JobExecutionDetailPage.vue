<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { IconThunderbolt, IconLeft, IconCodeBlock, IconRefresh } from "@/lib/icons";

import { useJobExecutions, useJobExecutionStream } from "@/composables/useJobs";
import { PageHeader, StatusTag } from "@/components/index";
import LogTerminal from "@/components/LogTerminal.vue";
import { to } from "@/router/registry";

const route = useRoute();
const router = useRouter();
const executionId = Number(route.params.id);

const { data: executions, loading } = useJobExecutions();
const execution = computed(() => executions.value.find((e) => e.id === executionId));

const stream = useJobExecutionStream(executionId || null);
const isLive = computed(() => stream.status.value === "open" || stream.status.value === "connecting");

// 刷新执行记录
function refreshExecution() {
  router.go(0);
}
</script>

<template>
  <div class="page-container execution-detail-page">
    <PageHeader
      :title="execution ? `执行记录 #${execution.id}` : '执行记录详情'"
      icon-bg="linear-gradient(135deg, rgba(20,184,166,0.16), rgba(45,212,191,0.16))"
      icon-color="var(--icon-purple)"
      :back-to="to.jobs.executions()"
      back-label="返回执行列表"
    >
      <template #icon><icon-thunderbolt /></template>
      <template #subtitle>
        <template v-if="execution">
          <code class="key-badge">{{ execution.definition_name }}</code>
          <span class="separator">·</span>
          <code class="plugin-badge">{{ execution.plugin_key }}</code>
          <span class="separator">:</span>
          <span class="action-tag">{{ execution.action }}</span>
        </template>
      </template>
      <template #actions>
        <div v-if="execution" class="execution-header-actions">
          <ui-button @click="refreshExecution">
            <template #icon><icon-refresh /></template>
            刷新
          </ui-button>
          <ui-button type="primary" @click="router.push(to.jobs.list())">
            返回任务管理
          </ui-button>
        </div>
      </template>
    </PageHeader>

    <!-- 未找到记录 -->
    <div v-if="loading" class="empty-card center-card-loading"><ui-spin :size="36" /></div>
    <ui-card v-else-if="!execution" class="empty-card">
      <ui-empty description="未找到该执行记录">
        <ui-button type="primary" @click="router.push(to.jobs.executions())">
          返回执行列表
        </ui-button>
      </ui-empty>
    </ui-card>

    <!-- 内容区域 -->
    <div v-else class="content-grid">
      <!-- 左侧：详情面板 -->
      <div class="detail-panel">
        <ui-card class="detail-card">
          <template #title>
            <div class="card-title-row">
              <icon-info-circle class="card-title-icon" />
              <span>执行信息</span>
            </div>
          </template>

          <div class="info-rows">
            <div class="info-row">
              <span class="info-label">执行状态</span>
              <StatusTag :status="execution.status" />
            </div>

            <div class="info-row">
              <span class="info-label">任务名称</span>
              <span class="info-value">{{ execution.definition_name }}</span>
            </div>

            <div class="info-row">
              <span class="info-label">插件</span>
              <code class="info-value plugin-badge">{{ execution.plugin_key }}</code>
            </div>

            <div class="info-row">
              <span class="info-label">动作</span>
              <span class="info-value action-badge">{{ execution.action }}</span>
            </div>

            <div class="info-row">
              <span class="info-label">执行节点</span>
              <code v-if="execution.worker_id" class="info-value worker-badge">
                {{ execution.worker_id }}
              </code>
              <span v-else class="info-value text-muted">未分配</span>
            </div>

            <div class="info-row" v-if="execution.input">
              <span class="info-label">输入参数</span>
              <div class="info-value info-value--full">
                <div class="json-box">
                  <pre>{{ JSON.stringify(execution.input, null, 2) }}</pre>
                </div>
              </div>
            </div>
          </div>
        </ui-card>
      </div>

      <!-- 右侧：日志终端 -->
      <div class="log-panel">
        <ui-card class="log-card">
          <template #title>
            <div class="log-title-row">
              <div class="card-title-row">
                <icon-code-block class="card-title-icon" />
                <span>实时日志</span>
              </div>
              <div class="stream-indicator" :class="{ live: isLive }">
                <span class="stream-dot" />
                {{ isLive ? "实时连接" : "离线缓冲" }}
              </div>
            </div>
          </template>

          <div class="log-body">
            <LogTerminal
              :logs="stream.lines.value"
              :is-live="isLive"
              :show-header="false"
              empty-label="等待事件流…"
              height-class="h-full"
            />
          </div>
        </ui-card>
      </div>
    </div>
  </div>
</template>
