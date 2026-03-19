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
  <div class="page-container form-page">
    <PageHeader
      title="修改 Agent"
      :subtitle="agent ? `正在编辑 ${agent.name}` : ''"
      icon-bg="linear-gradient(135deg, rgba(20,184,166,0.16), rgba(45,212,191,0.16))"
      icon-color="var(--icon-purple)"
      :back-to="to.agents.detail(agentId)"
      back-label="返回 Agent"
    >
      <template #icon><icon-robot /></template>
    </PageHeader>

    <div v-if="loading" class="center-empty"><ui-spin :size="36" /></div>
    <div v-else-if="!agent" class="center-empty"><p class="muted-copy">未找到该 Agent。</p></div>
    <ui-card v-else>
      <div class="inline-info-notice">
        <icon-info-circle class="inline-info-notice-icon" />
        <p class="inline-info-notice-body">暂不支持通过 UI 修改 Agent 配置。请删除后重新创建，或通过 API 修改。</p>
      </div>
    </ui-card>
  </div>
</template>
