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
  <div class="page-shell">
    <PageHeader
      :title="job ? job.name : '任务详情'"
      icon-bg="linear-gradient(135deg, rgba(202,138,4,0.12), rgba(234,179,8,0.12))"
      icon-color="var(--icon-yellow)"
      :back-to="to.jobs.list()"
      back-label="返回任务列表"
    >
      <template #icon><icon-schedule /></template>
      <template #subtitle>
        <code v-if="job" class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600">{{ job.key }}</code>
      </template>
      <template #actions>
        <div v-if="job" class="flex flex-wrap items-center justify-end gap-2">
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
    <div v-if="loading" class="empty-state-block"><ui-spin size="2.25em" /></div>
    <ui-card v-else-if="!job" class="empty-state-block">
      <ui-empty description="未找到该任务">
        <ui-button type="primary" @click="router.push(to.jobs.list())">
          返回任务列表
        </ui-button>
      </ui-empty>
    </ui-card>

    <!-- 任务详情 -->
    <div v-else class="grid grid-cols-1 gap-6 [@media(min-width:1280px)]:grid-cols-2">
      <!-- 基本信息 -->
      <ui-card class="rounded-xl border p-5 md:p-6 border-slate-200 bg-white shadow-sm min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-schedule class="h-5 w-5 text-[var(--accent)]" />
            <span>基本信息</span>
          </div>
        </template>

        <div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">任务ID</span>
            <span class="text-sm font-medium text-slate-900">
              <code>{{ job.id }}</code>
            </span>
          </div>

          <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">标识符</span>
            <span class="text-sm font-medium text-slate-900">
              <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600">{{ job.key }}</code>
            </span>
          </div>

          <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">任务名称</span>
            <span class="text-sm font-medium text-slate-900">{{ job.name }}</span>
          </div>

          <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">插件</span>
            <span class="text-sm font-medium text-slate-900">
              <code class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-mono font-semibold text-slate-700 border-slate-200 bg-white/[64%]">{{ job.plugin_key }}</code>
            </span>
          </div>

          <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">动作</span>
            <span class="text-sm font-medium text-slate-900">
              <span class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold text-[var(--accent)] border-slate-200 bg-slate-50">{{ job.action }}</span>
            </span>
          </div>

          <div class="flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">调度类型</span>
            <span v-if="job.schedule?.cron_expression" class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-semibold border-emerald-200 bg-emerald-50 text-emerald-700">
              <icon-clock-circle />
              定时任务
            </span>
            <span v-else class="inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-semibold border-slate-200 bg-slate-100 text-slate-600">
              手动触发
            </span>
          </div>

          <div class="col-span-full flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]" v-if="job.schedule?.cron_expression">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">Cron 表达式</span>
            <span class="text-sm font-medium text-slate-900">
              <code class="inline-flex items-center gap-1.5 rounded-xl border border-amber-200 bg-amber-50 px-3 py-1.5 text-xs font-mono font-semibold text-amber-700">
                <icon-clock-circle />
                {{ job.schedule.cron_expression }}
              </code>
            </span>
          </div>

          <div class="col-span-full flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-white/[58%]" v-if="job.schedule?.timezone">
            <span class="text-xs font-semibold uppercase tracking-[0.08em] text-slate-500">时区</span>
            <span class="text-sm font-medium text-slate-900">{{ job.schedule.timezone }}</span>
          </div>
        </div>
      </ui-card>

      <!-- 输入参数 -->
      <ui-card class="rounded-xl border p-5 md:p-6 border-slate-200 bg-white shadow-sm min-w-0" v-if="job.input">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-code class="h-5 w-5 text-[var(--accent)]" />
            <span>输入参数</span>
          </div>
        </template>

        <div class="overflow-auto rounded-xl border border-slate-200 bg-slate-50 shadow-sm [&pre]:m-0 [&pre]:whitespace-pre-wrap [&pre]:break-all [&pre]:border-0 [&pre]:bg-transparent [&pre]:p-4 [&pre]:text-xs [&pre]:leading-6 [&pre]:text-[#dbeafe]">
          <pre>{{ JSON.stringify(job.input, null, 2) }}</pre>
        </div>
      </ui-card>

      <!-- 快速操作 -->
      <ui-card class="rounded-xl border p-5 md:p-6 border-slate-200 bg-white shadow-sm min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-thunderbolt class="h-5 w-5 text-[var(--accent)]" />
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

        <div class="mt-5 flex flex-wrap gap-3">
          <button type="button" class="inline-flex items-center gap-2 rounded-full border px-3.5 py-2.5 text-sm font-medium text-slate-700 border-slate-200 bg-slate-50 [transition-property:background-color,_border-color,_transform] hover:border-slate-300 hover:bg-white hover:-translate-y-px [&.danger]:border-red-200 [&.danger]:bg-red-50 [&.danger]:text-red-700 hover:[&.danger:hover]:border-red-300 hover:[&.danger:hover]:bg-red-100" @click="router.push(to.jobs.edit(job.id))">
            <icon-edit />
            编辑任务配置
          </button>
          <button type="button" class="inline-flex items-center gap-2 rounded-full border px-3.5 py-2.5 text-sm font-medium text-slate-700 border-slate-200 bg-slate-50 [transition-property:background-color,_border-color,_transform] hover:border-slate-300 hover:bg-white hover:-translate-y-px [&.danger]:border-red-200 [&.danger]:bg-red-50 [&.danger]:text-red-700 hover:[&.danger:hover]:border-red-300 hover:[&.danger:hover]:bg-red-100" @click="copyToClipboard(job.key)">
            <icon-copy />
            复制任务标识符
          </button>
          <button type="button" class="inline-flex items-center gap-2 rounded-full border px-3.5 py-2.5 text-sm font-medium text-slate-700 border-slate-200 bg-slate-50 [transition-property:background-color,_border-color,_transform] hover:border-slate-300 hover:bg-white hover:-translate-y-px [&.danger]:border-red-200 [&.danger]:bg-red-50 [&.danger]:text-red-700 hover:[&.danger:hover]:border-red-300 hover:[&.danger:hover]:bg-red-100 danger" @click="deleteJob">
            <icon-delete />
            删除此任务
          </button>
        </div>
      </ui-card>

      <!-- 执行记录 -->
      <ui-card class="rounded-xl border p-5 md:p-6 border-slate-200 bg-white shadow-sm min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-history class="h-5 w-5 text-[var(--accent)]" />
            <span>执行记录</span>
          </div>
        </template>

        <div class="py-8">
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
