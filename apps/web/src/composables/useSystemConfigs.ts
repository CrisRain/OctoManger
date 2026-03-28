import { computed, ref } from "vue";
import { useConfigStore } from "@/store";
import type { SystemConfig } from "@/types";

export interface SystemConfigForm {
  appName: string;
  jobDefaultTimeoutMinutes: number;
  jobMaxConcurrency: number;
}

const DEFAULT_CONFIG: SystemConfigForm = {
  appName: "OctoManager",
  jobDefaultTimeoutMinutes: 30,
  jobMaxConcurrency: 10,
};

function toForm(value?: Partial<SystemConfig> | null): SystemConfigForm {
  return {
    appName: typeof value?.app_name === "string" && value.app_name.trim()
      ? value.app_name.trim()
      : DEFAULT_CONFIG.appName,
    jobDefaultTimeoutMinutes: typeof value?.job_default_timeout_minutes === "number"
      ? value.job_default_timeout_minutes
      : DEFAULT_CONFIG.jobDefaultTimeoutMinutes,
    jobMaxConcurrency: typeof value?.job_max_concurrency === "number"
      ? value.job_max_concurrency
      : DEFAULT_CONFIG.jobMaxConcurrency,
  };
}

function toPayload(value: SystemConfigForm): SystemConfig {
  return {
    app_name: value.appName.trim(),
    job_default_timeout_minutes: value.jobDefaultTimeoutMinutes,
    job_max_concurrency: value.jobMaxConcurrency,
  };
}

function cloneForm(value: SystemConfigForm): SystemConfigForm {
  return {
    appName: value.appName,
    jobDefaultTimeoutMinutes: value.jobDefaultTimeoutMinutes,
    jobMaxConcurrency: value.jobMaxConcurrency,
  };
}

function sameForm(left: SystemConfigForm, right: SystemConfigForm): boolean {
  return left.appName === right.appName
    && left.jobDefaultTimeoutMinutes === right.jobDefaultTimeoutMinutes
    && left.jobMaxConcurrency === right.jobMaxConcurrency;
}

export function useSystemConfigs() {
  const store = useConfigStore();
  const config = ref<SystemConfigForm>(cloneForm(DEFAULT_CONFIG));
  const savedConfig = ref<SystemConfigForm>(cloneForm(DEFAULT_CONFIG));
  const configLoading = ref(false);
  const configSaving = ref(false);

  const isConfigDirty = computed(() => !sameForm(config.value, savedConfig.value));

  async function loadConfigs() {
    configLoading.value = true;
    try {
      const result = await store.fetchConfig();
      const next = toForm(result);
      config.value = cloneForm(next);
      savedConfig.value = cloneForm(next);
    } finally {
      configLoading.value = false;
    }
  }

  async function saveConfig() {
    configSaving.value = true;
    try {
      const result = await store.saveConfig(toPayload(config.value));
      if (!result) {
        throw new Error(store.error ?? "保存失败");
      }
      const next = toForm(result);
      config.value = cloneForm(next);
      savedConfig.value = cloneForm(next);
      return result;
    } finally {
      configSaving.value = false;
    }
  }

  function resetConfig() {
    config.value = cloneForm(savedConfig.value);
  }

  return {
    config,
    configLoading,
    configSaving,
    isConfigDirty,
    loadConfigs,
    saveConfig,
    resetConfig,
  };
}
