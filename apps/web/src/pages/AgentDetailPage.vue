<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Message } from "@/lib/feedback";
import { useAgent, useAgentStatus, useStartAgent, useStopAgent, useAgentStream } from "@/composables/useAgents";
import { PageHeader } from "@/components/index";
import LogTerminal from "@/components/LogTerminal.vue";
import { to } from "@/router/registry";

const route = useRoute();
const router = useRouter();
const agentId = Number(route.params.id);

const { data: agent, loading: loadingAgent } = useAgent(agentId);
const { data: agentStatus } = useAgentStatus(agentId);
const startAgent = useStartAgent();
const stopAgent = useStopAgent();
const { lines, connected, runtimeState: streamRuntimeState } = useAgentStream(agentId || null);

const activeTab = ref("status");

const displayState   = computed(() =>
  (connected.value ? streamRuntimeState.value : null) ??
  agentStatus.value?.runtime_state ??
  agent.value?.runtime_state
);
const displayDesired = computed(() => agentStatus.value?.desired_state  ?? agent.value?.desired_state);
const displayError   = computed(() => agentStatus.value?.last_error     ?? agent.value?.last_error);

const runtimeColor: Record<string, string> = {
  running: "green",
  stopping: "blue",
  stopped: "gray",
  error: "red",
};
function getColor(state: string | undefined): string {
  return runtimeColor[state ?? ""] ?? "gray";
}

const stateDotClass = computed(() => {
  const s = displayState.value;
  if (s === "running") return "running";
  if (s === "error")   return "offline";
  return "neutral";
});

async function handleStart() {
  if (!agent.value) return;
  try {
    await startAgent.execute(agent.value.id);
    Message.success("启动指令已发送");
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "操作失败");
  }
}

async function handleStop() {
  if (!agent.value) return;
  try {
    await stopAgent.execute(agent.value.id);
    Message.success("停止指令已发送");
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "操作失败");
  }
}
</script>

<template>
  <div class="page-container agent-page">
    <PageHeader
      :title="agent ? agent.name : 'Agent 监控台'"
      icon-bg="linear-gradient(135deg, rgba(20,184,166,0.16), rgba(45,212,191,0.16))"
      icon-color="var(--icon-purple)"
      :back-to="to.agents.list()"
      back-label="返回进程池"
    >
      <template #icon><icon-robot /></template>
      <template #subtitle>
        <template v-if="agent">
          <code class="key-badge highlight-key">{{ agent.plugin_key }}</code> &nbsp;·&nbsp;
          <span class="muted-tag">{{ agent.action }}</span>
        </template>
      </template>
      <template #actions>
        <div v-if="agent" class="agent-header-actions">
          <div class="header-state">
            <span class="status-dot-large" :class="stateDotClass" />
            <ui-tag :color="getColor(displayState)" class="status-tag-pill">{{ displayState ?? "—" }}</ui-tag>
            <span class="stream-badge" :class="{ live: connected }">
              <span class="stream-dot" />{{ connected ? "LIVE REPL" : "OFFLINE" }}
            </span>
          </div>
          <ui-button
            type="primary"
            class="action-btn action-btn--start"
            :loading="startAgent.loading.value"
            :disabled="displayState === 'running'"
            @click="handleStart"
          >
            <template #icon><icon-play-arrow /></template>
            唤起进程
          </ui-button>
          <ui-button
            status="danger"
            class="action-btn action-btn--stop"
            :loading="stopAgent.loading.value"
            :disabled="displayState === 'stopped'"
            @click="handleStop"
          >
            <template #icon><icon-pause /></template>
            下发挂起
          </ui-button>
        </div>
      </template>
    </PageHeader>

    <!-- ── Loading / not found ─────────────────────────────── -->
    <div v-if="loadingAgent" class="empty-wrap">
      <ui-spin :size="36" />
    </div>
    <div v-else-if="!agent" class="empty-wrap">
      <icon-robot class="empty-icon" />
      <p class="empty-msg">未找到该 Agent。</p>
      <ui-button @click="router.push(to.agents.list())">返回列表</ui-button>
    </div>

    <!-- ── Tabs ────────────────────────────────────────────── -->
    <ui-tabs
      v-else-if="agent"
      v-model:active-key="activeTab"
      class="detail-tabs premium-tabs"
      :destroy-on-hide="false"
      animation
    >
      <!-- ── Tab 1: Status ─────────────────────────────────── -->
      <ui-tab-pane key="status" title="状态看板">
        <div class="status-grid">
          <ui-card class="state-card status-card">
            <template #title>
              <div class="card-header-with-icon">
                <div class="card-icon-box"><icon-dashboard /></div>
                生命周期状态监视
              </div>
            </template>
            <div class="state-rows">
              <div class="state-row">
                <span class="detail-label">实时状态 (Runtime)</span>
                <div class="state-row-right">
                  <span class="status-dot-large" :class="stateDotClass" />
                  <ui-tag :color="getColor(displayState)" class="status-tag-pill">{{ displayState ?? "—" }}</ui-tag>
                </div>
              </div>
              <div class="state-row">
                <span class="detail-label">期望状态 (Desired)</span>
                <ui-tag color="gray" class="status-tag-pill outline">{{ displayDesired ?? "—" }}</ui-tag>
              </div>
              <div class="state-row">
                <span class="detail-label">日志流通道 (Log Stream)</span>
                <div class="state-row-right">
                  <span class="status-dot-large" :class="connected ? 'online' : 'neutral'" />
                  <ui-tag :color="connected ? 'green' : 'gray'" class="status-tag-pill outline">
                    {{ connected ? "connected" : "disconnected" }}
                  </ui-tag>
                </div>
              </div>
              <div class="state-row">
                <span class="detail-label">执行基座 (Plugin)</span>
                <span class="detail-value mono-text highlight-key">{{ agent.plugin_key }}</span>
              </div>
              <div class="state-row">
                <span class="detail-label">原子动作 (Action)</span>
                <span class="detail-value mono-text highlight-key">{{ agent.action }}</span>
              </div>
              <div class="state-row">
                <span class="detail-label">资源追踪码 (ID)</span>
                <span class="detail-value mono-text">#{{ agent.id }}</span>
              </div>
            </div>

            <div v-if="displayError" class="error-box">
              <icon-close-circle class="error-icon" />
              <span class="error-box-title">错误追踪栈</span>
              <p class="error-text">{{ displayError }}</p>
            </div>
          </ui-card>

          <!-- Input params -->
          <ui-card v-if="agent.input && Object.keys(agent.input).length" class="params-card detail-card">
            <template #title>
              <div class="card-header-with-icon">
                <div class="card-icon-box params-icon"><icon-code-block /></div>
                配置参数装载表 (Input)
              </div>
            </template>
            <div class="state-rows params-rows">
              <div v-for="(val, key) in agent.input" :key="key" class="state-row params-row">
                <span class="detail-label mono-text param-key">{{ key }}</span>
                <span class="detail-value mono-text param-val">{{ val }}</span>
              </div>
            </div>
          </ui-card>
        </div>
      </ui-tab-pane>

      <!-- ── Tab 2: Logs ───────────────────────────────────── -->
      <ui-tab-pane key="logs" title="实时日志">
        <LogTerminal
          class="log-pane"
          :logs="lines"
          :is-live="connected"
          :show-header="false"
          empty-label="等待日志输出…"
        />
      </ui-tab-pane>
    </ui-tabs>
  </div>
</template>
