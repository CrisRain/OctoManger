<script setup lang="ts">
import { ref, computed } from "vue";
import { useAutoRefresh } from "@/composables/useAutoRefresh";
import { useSystemLogs, useSystemStatus } from "@/composables/useDashboard";
import { useJobExecutions, useJobExecutionStream } from "@/composables/useJobs";
import { useAgents, useAgentEventStream } from "@/composables/useAgents";
import { PageHeader } from "@/components/index";
import LogTerminal from "@/components/LogTerminal.vue";

// ── System status ──────────────────────────────────────────────────────────
const { data: statusData } = useSystemStatus();

// ── System runtime logs ───────────────────────────────────────────────────
const activeTab = ref("system");
const searchKeyword = ref("");
const levelFilter = ref("");
const { data: systemLogs, loading: loadingSystemLogs, error: systemLogsError, refresh: refreshSystemLogs } = useSystemLogs(200);
useAutoRefresh(refreshSystemLogs, { intervalMs: 5000, immediate: false });

const systemLevelOptions = [
  { label: "全部", value: "" },
  { label: "错误", value: "error" },
  { label: "警告", value: "warn" },
  { label: "信息", value: "info" },
  { label: "调试", value: "debug" },
];

const filteredSystemLogs = computed(() => {
  const keyword = searchKeyword.value.trim().toLowerCase();
  return systemLogs.value.filter((item) => {
    if (levelFilter.value && item.level !== levelFilter.value) {
      return false;
    }
    if (!keyword) {
      return true;
    }
    const haystacks = [
      item.message,
      item.source,
      item.logger,
      item.caller,
      JSON.stringify(item.fields ?? {}),
    ];
    return haystacks.some((value) => value?.toLowerCase().includes(keyword));
  });
});

const systemLogLines = computed(() =>
  [...filteredSystemLogs.value]
    .reverse()
    .map((item) =>
      JSON.stringify({
        event_type: item.source || item.logger || "system",
        message: item.message,
        payload: item.fields ?? {},
      })
    )
);

// ── Job executions ─────────────────────────────────────────────────────────
const { data: executions } = useJobExecutions();
const selectedExecId = ref<number | null>(null);

const { lines: jobLines, status: jobStatus } = useJobExecutionStream(selectedExecId);
const jobLive = computed(() => jobStatus.value === "open" || jobStatus.value === "connecting");

function selectExecution(id: number) {
  selectedExecId.value = selectedExecId.value === id ? null : id;
}

// ── Agents ─────────────────────────────────────────────────────────────────
const { data: agents } = useAgents();
const selectedAgentId = ref<number | null>(null);

const { lines: agentLines, status: agentStatus, isRunning: agentRunning } = useAgentEventStream(selectedAgentId);
const agentLive = computed(() => {
  if (selectedAgentId.value) {
    return agentRunning.value;
  }
  return agentStatus.value === "open" || agentStatus.value === "connecting";
});

function selectAgent(id: number) {
  selectedAgentId.value = selectedAgentId.value === id ? null : id;
}

// ── Helpers ────────────────────────────────────────────────────────────────
const execStatusColor: Record<string, string> = {
  pending:   "gray",
  running:   "blue",
  succeeded: "green",
  failed:    "red",
  cancelled: "gray",
};
function execColor(status: string) { return execStatusColor[status] ?? "gray"; }

const agentRuntimeColor: Record<string, string> = {
  running: "green",
  idle:    "blue",
  stopped: "gray",
  error:   "red",
};
function agentColor(state: string | undefined) { return agentRuntimeColor[state ?? ""] ?? "gray"; }

const agentDot: Record<string, string> = {
  running: "running",
  idle:    "online",
  error:   "offline",
};
function dotClass(state: string | undefined) { return agentDot[state ?? ""] ?? "neutral"; }

</script>

<template>
  <div class="page-shell logs-page">
    <PageHeader
      title="日志与观测"
      subtitle="查看系统运行日志、任务执行日志和 Agent 事件流"
      icon-bg="linear-gradient(135deg, rgba(10,132,255,0.12), rgba(10,132,255,0.06))"
      icon-color="var(--accent)"
    >
      <template #icon><icon-file /></template>
    </PageHeader>

    <!-- ── System status banner ─────────────────────────────────────────── -->
    <div
      class="panel-surface mb-6 flex items-center gap-3 px-5 py-4 text-sm"
      :class="statusData?.database_ok ? 'border-emerald-200 bg-emerald-50/90 text-emerald-800' : 'border-red-200 bg-red-50/90 text-red-700'"
    >
      <span
        class="inline-block h-2.5 w-2.5 flex-shrink-0 rounded-full"
        :class="statusData?.database_ok ? 'bg-emerald-500 animate-pulse' : 'bg-red-500'"
      />
      <span class="text-sm leading-6 text-current">
        <template v-if="statusData?.database_ok">
          数据库连接&nbsp;<strong class="font-semibold">正常</strong>
          &nbsp;<span class="text-slate-400">·</span>&nbsp;
          已加载插件 <strong>{{ statusData.plugin_count }}</strong> 个
          &nbsp;<span class="text-slate-400">·</span>&nbsp;
          当前时间&nbsp;
          <span class="text-xs font-mono">{{ new Date(statusData.now).toLocaleString("zh-CN") }}</span>
        </template>
        <template v-else>数据库连接异常，部分功能可能不可用。</template>
      </span>
    </div>

    <!-- ── Tabs ─────────────────────────────────────────────────────────── -->
    <ui-tabs v-model:active-key="activeTab" class="panel-surface p-4" :destroy-on-hide="false">

      <!-- ── Tab 1: System runtime logs ─────────────────────────────────── -->
      <ui-tab-pane key="system">
        <template #title>
          <div class="inline-flex items-center gap-2"><icon-file /> 系统运行日志</div>
        </template>

        <div class="flex flex-col gap-4">
          <div class="panel-surface flex flex-col gap-3 border border-slate-200 bg-slate-50/80 px-4 py-3 shadow-sm">
            <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <div class="flex min-w-0 flex-1 items-center gap-3 rounded-xl border border-slate-200 bg-white px-3 py-2.5 shadow-sm">
                <icon-search class="h-4 w-4 flex-shrink-0 text-slate-400" />
                <input
                  v-model="searchKeyword"
                  type="text"
                  placeholder="搜索消息、来源、调用点或字段…"
                  class="min-w-0 flex-1 bg-transparent text-sm text-slate-800 outline-none placeholder:text-slate-400"
                />
              </div>
              <div class="flex items-center justify-between gap-3 lg:justify-end">
                <div class="text-xs font-medium text-slate-500">
                  共 {{ filteredSystemLogs.length }} / {{ systemLogs.length }} 条
                </div>
                <ui-button size="small" :loading="loadingSystemLogs" @click="refreshSystemLogs">
                  <template #icon><icon-refresh /></template>
                  刷新
                </ui-button>
              </div>
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <span class="text-xs font-medium text-slate-500">级别：</span>
              <button
                v-for="option in systemLevelOptions"
                :key="option.value"
                type="button"
                class="filter-chip"
                :class="{ active: levelFilter === option.value }"
                @click="levelFilter = option.value"
              >
                {{ option.label }}
              </button>
            </div>
          </div>

          <div
            v-if="systemLogsError"
            class="panel-surface border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800"
          >
            系统日志加载失败：{{ systemLogsError }}
          </div>

          <LogTerminal
            class="min-h-[520px]"
            :logs="systemLogLines"
            :is-live="loadingSystemLogs"
            title="系统运行日志"
            empty-label="暂无系统运行日志"
          />
        </div>
      </ui-tab-pane>

      <!-- ── Tab 2: Job executions ──────────────────────────────────────── -->
      <ui-tab-pane key="jobs">
        <template #title>
          <div class="inline-flex items-center gap-2"><icon-schedule /> 任务执行日志</div>
        </template>
        <div class="grid grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(18em,_1fr)_minmax(0,_1.8fr)]">
          <!-- Left: execution list -->
          <div class="flex min-h-[320px] max-h-[680px] flex-col gap-2 overflow-y-auto rounded-xl border p-3 border-slate-200 bg-slate-50 shadow-sm backdrop-blur-xl backdrop-saturate-[150%]">
            <ui-empty v-if="!executions.length" description="暂无执行记录" class="min-h-[220px]" />
            <button
              type="button"
              v-for="ex in executions"
              :key="ex.id"
              class="rounded-xl border border-transparent p-3.5 text-left bg-slate-50 [transition-property:background-color,_border-color,_box-shadow,_transform] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 hover:border-slate-200 hover:bg-white hover:-translate-y-px"
              :class="{ 'border-blue-500/20 bg-white/85 shadow-sm': selectedExecId === ex.id }"
              @click="selectExecution(ex.id)"
            >
              <div class="flex items-center justify-between gap-3">
                <span class="text-xs font-semibold text-slate-500 mono">#{{ ex.id }}</span>
                <ui-tag :color="execColor(ex.status)" size="small">{{ ex.status }}</ui-tag>
              </div>
              <div class="mt-2 text-sm font-semibold text-slate-900">{{ ex.definition_name ?? "—" }}</div>
              <div class="mt-2 flex items-center justify-between gap-3 text-xs text-slate-500 mono">
                <span>{{ ex.plugin_key }} · {{ ex.action }}</span>
                <span v-if="ex.worker_id" class="text-xs text-slate-400 font-mono">{{ ex.worker_id }}</span>
              </div>
            </button>
          </div>

          <!-- Right: terminal -->
          <div v-if="!selectedExecId" class="flex min-h-[320px] flex-col items-center justify-center gap-3 rounded-xl border border-dashed text-center text-slate-500 border-slate-200 bg-white/55">
            <icon-schedule class="mb-2 h-10 w-10 text-slate-400" />
            <p>选择左侧的执行记录查看日志</p>
          </div>
          <LogTerminal
            v-else
            class="min-h-[320px]"
            :key="selectedExecId"
            :logs="jobLines"
            :is-live="jobLive"
            :title="`执行 #${selectedExecId} 日志`"
            empty-label="等待日志输出…"
          />
        </div>
      </ui-tab-pane>

      <!-- ── Tab 3: Agent events ─────────────────────────────────────────── -->
      <ui-tab-pane key="agents">
        <template #title>
          <div class="inline-flex items-center gap-2"><icon-robot /> Agent 事件流</div>
        </template>
        <div class="grid grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(18em,_1fr)_minmax(0,_1.8fr)]">
          <!-- Left: agent list -->
          <div class="flex min-h-[320px] max-h-[680px] flex-col gap-2 overflow-y-auto rounded-xl border p-3 border-slate-200 bg-slate-50 shadow-sm backdrop-blur-xl backdrop-saturate-[150%]">
            <ui-empty v-if="!agents.length" description="暂无 Agent" class="min-h-[220px]" />
            <button
              type="button"
              v-for="ag in agents"
              :key="ag.id"
              class="rounded-xl border border-transparent p-3.5 text-left bg-slate-50 [transition-property:background-color,_border-color,_box-shadow,_transform] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50 hover:border-slate-200 hover:bg-white hover:-translate-y-px"
              :class="{ 'border-blue-500/20 bg-white/85 shadow-sm': selectedAgentId === ag.id }"
              @click="selectAgent(ag.id)"
            >
              <div class="flex items-center justify-between gap-3">
                <div class="flex items-center gap-2">
                  <span
                    class="inline-block h-2 w-2 flex-shrink-0 rounded-full"
                    :class="{
                      'bg-emerald-500 animate-pulse': ag.runtime_state === 'running',
                      'bg-sky-400': ag.runtime_state === 'idle',
                      'bg-red-500': ag.runtime_state === 'error',
                      'bg-slate-300': !['running', 'idle', 'error'].includes(ag.runtime_state ?? ''),
                    }"
                  />
                  <span class="text-sm font-semibold text-slate-900">{{ ag.name }}</span>
                </div>
                <ui-tag :color="agentColor(ag.runtime_state)" size="small">
                  {{ ag.runtime_state ?? "—" }}
                </ui-tag>
              </div>
              <div class="mt-2 flex items-center justify-between gap-3 text-xs text-slate-500 mono">
                <span>{{ ag.plugin_key }} · {{ ag.action }}</span>
              </div>
              <div v-if="ag.last_error" class="mt-2 text-xs text-red-600 overflow-hidden">{{ ag.last_error }}</div>
            </button>
          </div>

          <!-- Right: terminal -->
          <div v-if="!selectedAgentId" class="flex min-h-[320px] flex-col items-center justify-center gap-3 rounded-xl border border-dashed text-center text-slate-500 border-slate-200 bg-white/55">
            <icon-robot class="mb-2 h-10 w-10 text-slate-400" />
            <p>选择左侧的 Agent 查看实时事件流</p>
          </div>
          <LogTerminal
            v-else
            class="min-h-[320px]"
            :key="selectedAgentId"
            :logs="agentLines"
            :is-live="agentLive"
            :title="`Agent #${selectedAgentId} 事件流`"
            empty-label="等待事件输出…"
          />
        </div>
      </ui-tab-pane>

    </ui-tabs>
  </div>
</template>
