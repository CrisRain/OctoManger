import { defineStore } from "pinia";
import { ref } from "vue";
import { createJobDefinition, enqueueJobExecution, listJobDefinitions, listJobExecutions } from "@/api";
import type { JobDefinition, JobDefinitionCreateInput, JobExecution } from "@/types";
import { normalizeListResponse } from "@/utils/normalizeListResponse";

export const useJobsStore = defineStore("jobs", () => {
  const jobDefinitions = ref<JobDefinition[]>([]);
  const jobExecutions = ref<JobExecution[]>([]);
  const loadingDefinitions = ref(false);
  const loadingExecutions = ref(false);
  const error = ref<string | null>(null);

  async function fetchJobDefinitions() {
    loadingDefinitions.value = true;
    error.value = null;
    try {
      jobDefinitions.value = normalizeListResponse<JobDefinition>(await listJobDefinitions());
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loadingDefinitions.value = false;
    }
  }

  async function fetchJobExecutions() {
    loadingExecutions.value = true;
    error.value = null;
    try {
      jobExecutions.value = normalizeListResponse<JobExecution>(await listJobExecutions());
    } catch (e) {
      error.value = e instanceof Error ? e.message : "请求失败";
    } finally {
      loadingExecutions.value = false;
    }
  }

  async function createDefinition(payload: JobDefinitionCreateInput) {
    const result = await createJobDefinition(payload);
    jobDefinitions.value = [result, ...jobDefinitions.value];
    return result;
  }

  async function enqueueExecution(id: number) {
    const result = await enqueueJobExecution(id);
    jobExecutions.value = [result, ...jobExecutions.value];
    return result;
  }

  return {
    jobDefinitions,
    jobExecutions,
    loadingDefinitions,
    loadingExecutions,
    error,
    fetchJobDefinitions,
    fetchJobExecutions,
    createDefinition,
    enqueueExecution,
  };
});
