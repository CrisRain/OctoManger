<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { getSetupStatus, initializeSetup } from "@/api";
import { useAuthStore } from "@/store";
import { getAdminKey } from "@/lib/auth";
import { useMessage } from "@/composables";
import { PATHS, to } from "@/router/registry";
import { invalidateSetupStatusCache } from "@/router/guard";

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const message = useMessage();

const checkingStatus = ref(true);
const statusFailed = ref(false);
const submitting = ref(false);
const setupComplete = ref(false);
const generatedKey = ref("");

const form = ref({
  appName: "OctoManager",
  jobDefaultTimeoutMinutes: 30,
  jobMaxConcurrency: 10,
});

const redirectPath = computed(() => {
  const raw = route.query.redirect;
  if (typeof raw === "string" && raw.startsWith("/")) {
    return raw;
  }
  return to.dashboard();
});

async function checkSetupStatus() {
  checkingStatus.value = true;
  statusFailed.value = false;
  try {
    const status = await getSetupStatus();
    if (!status.needs_setup) {
      if (getAdminKey()) {
        await router.replace(redirectPath.value);
      } else {
        await router.replace({ path: PATHS.auth, query: { redirect: redirectPath.value } });
      }
      return;
    }
  } catch {
    statusFailed.value = true;
  } finally {
    checkingStatus.value = false;
  }
}

async function handleInitialize() {
  submitting.value = true;
  try {
    const result = await initializeSetup({
      app_name: form.value.appName,
      job_default_timeout_minutes: Number(form.value.jobDefaultTimeoutMinutes),
      job_max_concurrency: Number(form.value.jobMaxConcurrency),
    });

    generatedKey.value = result.api_key;
    authStore.setKey(result.api_key);
    invalidateSetupStatusCache();
    setupComplete.value = true;
    message.success("初始化完成");
  } catch (error) {
    const text = error instanceof Error ? error.message : "初始化失败";
    if (text.includes("already_initialized")) {
      await router.replace({ path: PATHS.auth, query: { redirect: redirectPath.value } });
      return;
    }
    message.error("初始化失败，请检查输入后重试");
  } finally {
    submitting.value = false;
  }
}

async function copyKey() {
  if (!generatedKey.value) {
    return;
  }

  try {
    await navigator.clipboard.writeText(generatedKey.value);
    message.success("API Key 已复制");
  } catch {
    message.warning("复制失败，请手动复制");
  }
}

async function enterConsole() {
  await router.replace(redirectPath.value);
}

onMounted(() => {
  void checkSetupStatus();
});
</script>

<template>
  <div class="min-h-full bg-slate-100 px-4 py-10">
    <div class="mx-auto max-w-xl">
      <ui-card>
        <template #title>系统初始化</template>
        <p class="text-sm leading-6 text-slate-500">
          首次启动请先写入默认系统配置。后端会自动生成 API Key 并保存至数据库。
        </p>

        <div v-if="checkingStatus" class="mt-4 text-sm text-slate-500">
          正在检查初始化状态...
        </div>

        <div v-else-if="statusFailed" class="mt-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700">
          无法连接后端，请确认 API 服务已启动。
        </div>

        <div v-else-if="setupComplete" class="mt-4 flex flex-col gap-3">
          <div class="rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-700">
            初始化成功，API Key 已自动写入当前会话。
          </div>

          <ui-input :model-value="generatedKey" readonly />

          <div class="flex gap-2">
            <ui-button @click="copyKey">复制 API Key</ui-button>
            <ui-button type="primary" @click="enterConsole">进入控制台</ui-button>
          </div>
        </div>

        <form v-else class="mt-4 flex flex-col gap-4" @submit.prevent="handleInitialize">
          <ui-form-item label="应用名称">
            <ui-input v-model="form.appName" placeholder="OctoManager" />
          </ui-form-item>

          <ui-form-item label="默认任务超时（分钟）">
            <ui-input-number v-model="form.jobDefaultTimeoutMinutes" :min="0" :step="1" />
          </ui-form-item>

          <ui-form-item label="最大并发数">
            <ui-input-number v-model="form.jobMaxConcurrency" :min="0" :step="1" />
          </ui-form-item>

          <ui-button type="primary" html-type="submit" :loading="submitting" class="w-full justify-center">
            执行初始化
          </ui-button>
        </form>
      </ui-card>
    </div>
  </div>
</template>
