<script setup lang="ts">
import { useRoute } from "vue-router";
import { useAgent } from "@/composables/useAgents";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const route = useRoute();
const agentId = Number(route.params.id);

const { data: agent, loading } = useAgent(agentId);
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="编辑 Agent"
      :subtitle="agent ? `正在编辑 ${agent.name}` : ''"
      icon-bg="linear-gradient(135deg, rgba(10,132,255,0.12), rgba(10,132,255,0.06))"
      icon-color="var(--icon-purple)"
      :back-to="to.agents.detail(agentId)"
      back-label="返回 Agent 详情"
    >
      <template #icon><icon-robot /></template>
    </PageHeader>

    <div v-if="loading" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/[56%] shadow-sm"><ui-spin size="2.25em" /></div>
    <div v-else-if="!agent" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/[56%] shadow-sm"><p class="text-sm leading-6 text-slate-500">未找到该 Agent。</p></div>
    <ui-card v-else>
      <div class="flex items-start gap-3 rounded-xl border border-sky-200 bg-sky-50/70 p-4">
        <icon-info-circle class="mt-0.5 h-4 w-4 flex-shrink-0 text-sky-600" />
        <p class="text-sm leading-6 text-slate-700">暂不支持通过 UI 编辑 Agent 配置。请删除后重新创建，或通过 API 更新。</p>
      </div>
    </ui-card>
  </div>
</template>
