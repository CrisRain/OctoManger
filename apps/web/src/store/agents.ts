import { defineStore } from "pinia";
import { ref } from "vue";
import { createAgent, getAgentStatus, listAgents, startAgent, stopAgent } from "@/api";
import type { Agent, AgentCreateInput, AgentStatus } from "@/types";
import { normalizeListResponse } from "@/utils/normalizeListResponse";

export const useAgentsStore = defineStore("agents", () => {
  const agents = ref<Agent[]>([]);
  const statuses = ref<Record<number, AgentStatus>>({});
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchAgents() {
    loading.value = true;
    error.value = null;
    try {
      agents.value = normalizeListResponse<Agent>(await listAgents());
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loading.value = false;
    }
  }

  async function create(payload: AgentCreateInput) {
    const result = await createAgent(payload);
    agents.value = [result, ...agents.value];
    return result;
  }

  async function start(id: number) {
    await startAgent(id);
  }

  async function stop(id: number) {
    await stopAgent(id);
  }

  async function fetchStatus(id: number) {
    const status = await getAgentStatus(id);
    statuses.value = { ...statuses.value, [id]: status };
    return status;
  }

  return {
    agents,
    statuses,
    loading,
    error,
    fetchAgents,
    create,
    start,
    stop,
    fetchStatus,
  };
});
