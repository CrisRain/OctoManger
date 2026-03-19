<script setup lang="ts">
import { ref, computed } from "vue";
import { useSystemStatus } from "@/composables/useDashboard";
import { useJobExecutions, useJobExecutionStream } from "@/composables/useJobs";
import { useAgents, useAgentEventStream } from "@/composables/useAgents";
import { PageHeader } from "@/components/index";
import LogTerminal from "@/components/LogTerminal.vue";

// ── System status ──────────────────────────────────────────────────────────
const { data: statusData } = useSystemStatus();

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
const activeTab = ref("jobs");

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
  <div class="page-container logs-page">
    <PageHeader
      title="实时系统观测与日志 (Logs)"
      subtitle="中央集成的运行状况打点、各路任务执行信标与 Agent 状态事件流观测大屏。"
      icon-bg="linear-gradient(135deg, rgba(20,184,166,0.16), rgba(45,212,191,0.16))"
      icon-color="var(--accent)"
    >
      <template #icon><icon-file /></template>
    </PageHeader>

    <!-- ── System status banner ─────────────────────────────────────────── -->
    <div
      class="status-banner premium-status-banner"
      :class="statusData?.database_ok ? 'status-banner--ok' : 'status-banner--err'"
    >
      <span class="status-dot-large" :class="statusData?.database_ok ? 'online' : 'offline'" />
      <span class="banner-text">
        <template v-if="statusData?.database_ok">
          主线数据库&nbsp;<strong class="fw-bold">心跳正常</strong>
          &nbsp;<span class="divider">·</span>&nbsp;
          集成应用插件池 <strong>{{ statusData.plugin_count }}</strong> 装载
          &nbsp;<span class="divider">·</span>&nbsp;
          基站标准时间&nbsp;
          <span class="banner-time">{{ new Date(statusData.now).toLocaleString("zh-CN") }}</span>
        </template>
        <template v-else>数据库连接断层，指令信道可能受阻。</template>
      </span>
    </div>

    <!-- ── Tabs ─────────────────────────────────────────────────────────── -->
    <ui-tabs v-model:active-key="activeTab" class="main-tabs premium-tabs" :destroy-on-hide="false">

      <!-- ── Tab 1: Job executions ──────────────────────────────────────── -->
      <ui-tab-pane key="jobs">
        <template #title>
          <div class="tab-title"><icon-schedule /> 任务执行观测舱</div>
        </template>
        <div class="split-pane">
          <!-- Left: execution list -->
          <div class="split-list">
            <ui-empty v-if="!executions.length" description="暂无执行记录" class="list-empty" />
            <button
              type="button"
              v-for="ex in executions"
              :key="ex.id"
              class="list-item"
              :class="{ 'list-item--active': selectedExecId === ex.id }"
              @click="selectExecution(ex.id)"
            >
              <div class="item-row">
                <span class="item-id mono">#{{ ex.id }}</span>
                <ui-tag :color="execColor(ex.status)" size="small">{{ ex.status }}</ui-tag>
              </div>
              <div class="item-name">{{ ex.definition_name ?? "—" }}</div>
              <div class="item-meta mono">
                <span>{{ ex.plugin_key }} · {{ ex.action }}</span>
                <span v-if="ex.worker_id" class="item-time">{{ ex.worker_id }}</span>
              </div>
            </button>
          </div>

          <!-- Right: terminal -->
          <div v-if="!selectedExecId" class="split-terminal terminal-placeholder">
            <icon-schedule class="placeholder-icon" />
            <p>选择左侧的执行记录查看日志</p>
          </div>
          <LogTerminal
            v-else
            class="split-terminal"
            :key="selectedExecId"
            :logs="jobLines"
            :is-live="jobLive"
            :title="`执行 #${selectedExecId} 日志`"
            empty-label="等待日志输出…"
          />
        </div>
      </ui-tab-pane>

      <!-- ── Tab 2: Agent events ─────────────────────────────────────────── -->
      <ui-tab-pane key="agents">
        <template #title>
          <div class="tab-title"><icon-robot /> Agent 事件射频流</div>
        </template>
        <div class="split-pane">
          <!-- Left: agent list -->
          <div class="split-list">
            <ui-empty v-if="!agents.length" description="暂无 Agent" class="list-empty" />
            <button
              type="button"
              v-for="ag in agents"
              :key="ag.id"
              class="list-item"
              :class="{ 'list-item--active': selectedAgentId === ag.id }"
              @click="selectAgent(ag.id)"
            >
              <div class="item-row">
                <div class="item-dot-name">
                  <span class="status-dot" :class="dotClass(ag.runtime_state)" />
                  <span class="item-name-inline">{{ ag.name }}</span>
                </div>
                <ui-tag :color="agentColor(ag.runtime_state)" size="small">
                  {{ ag.runtime_state ?? "—" }}
                </ui-tag>
              </div>
              <div class="item-meta mono">
                <span>{{ ag.plugin_key }} · {{ ag.action }}</span>
              </div>
              <div v-if="ag.last_error" class="item-error">{{ ag.last_error }}</div>
            </button>
          </div>

          <!-- Right: terminal -->
          <div v-if="!selectedAgentId" class="split-terminal terminal-placeholder">
            <icon-robot class="placeholder-icon" />
            <p>选择左侧的 Agent 查看实时事件流</p>
          </div>
          <LogTerminal
            v-else
            class="split-terminal"
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
