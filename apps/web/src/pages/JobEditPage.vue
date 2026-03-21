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
  <div class="page-shell">
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
    <div v-if="loading" class="empty-state-block"><ui-spin size="2.25em" /></div>
    <ui-card v-else-if="!job" class="empty-state-block">
      <ui-empty description="未找到该任务">
        <ui-button type="primary" @click="router.push(to.jobs.list())">
          返回任务列表
        </ui-button>
      </ui-empty>
    </ui-card>

    <!-- 开发中提示 -->
    <ui-card v-else class="min-w-0 lg:sticky lg:top-[var(--space-6)]">
      <template #title>
        <div class="flex items-center gap-2">
          <icon-info-circle class="h-5 w-5 text-[var(--accent)]" />
          <span>功能说明</span>
        </div>
      </template>

      <div class="mb-5 flex items-start gap-3 rounded-xl border border-sky-200 bg-sky-50/70 p-4">
        <icon-info-circle class="mt-0.5 flex-shrink-0 text-sky-600" />
        <div class="flex flex-col">
          <p class="text-sm font-semibold text-slate-900">任务编辑功能正在开发中</p>
          <p class="mt-1 text-sm leading-6 text-slate-600">
            目前您可以通过以下方式修改任务：
          </p>
          <ul class="mt-1 list-disc pl-4 text-sm leading-6 text-slate-600">
            <li>通过 API 端点直接更新任务定义</li>
            <li>删除现有任务后重新创建</li>
            <li>联系系统管理员进行配置</li>
          </ul>
          <div class="mt-4 flex flex-wrap gap-2">
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
      <div class="mt-2">
        <h3 class="mb-4 text-[15px] font-semibold text-slate-900">当前配置（只读）</h3>

        <div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">任务标识符</span>
            <code class="text-sm font-medium text-slate-900">{{ job.key }}</code>
          </div>

          <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">任务名称</span>
            <span class="text-sm font-medium text-slate-900">{{ job.name }}</span>
          </div>

          <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">插件</span>
            <code class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-mono font-semibold text-slate-700 border-slate-200 bg-white/[64%]">{{ job.plugin_key }}</code>
          </div>

          <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">动作</span>
            <span class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold text-[var(--accent)] border-slate-200 bg-slate-50">{{ job.action }}</span>
          </div>

          <div class="col-span-full flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]" v-if="job.schedule?.cron_expression">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">Cron 表达式</span>
            <code class="inline-flex items-center gap-1.5 rounded-xl border border-amber-200 bg-amber-50 px-3 py-1.5 text-xs font-mono font-semibold text-amber-700">{{ job.schedule.cron_expression }}</code>
          </div>

          <div class="col-span-full flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]" v-if="job.input">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">输入参数</span>
            <pre class="overflow-auto rounded-xl border border-slate-200 bg-slate-950 p-4 text-xs leading-6 text-slate-300 whitespace-pre-wrap break-all">{{ JSON.stringify(job.input, null, 2) }}</pre>
          </div>
        </div>
      </div>
    </ui-card>
  </div>
</template>
