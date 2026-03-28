<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { storeToRefs } from "pinia";
import { useRoute, useRouter } from "vue-router";
import { getSetupStatus, getSystemStatus } from "@/api";
import { useAuthStore } from "@/store";
import { useMessage } from "@/composables";
import { PATHS, to } from "@/router/registry";

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const { adminKey } = storeToRefs(authStore);
const message = useMessage();

const keyDraft = ref(adminKey.value ?? "");
const submitting = ref(false);
const checkingSetup = ref(true);
const setupCheckFailed = ref(false);

const redirectPath = computed(() => {
  const raw = route.query.redirect;
  if (typeof raw === "string" && raw.startsWith("/")) {
    return raw;
  }
  return to.dashboard();
});

async function verifySetupState() {
  checkingSetup.value = true;
  setupCheckFailed.value = false;
  try {
    const status = await getSetupStatus();
    if (status.needs_setup) {
      await router.replace({ path: PATHS.setup, query: { redirect: redirectPath.value } });
      return;
    }
  } catch {
    setupCheckFailed.value = true;
  } finally {
    checkingSetup.value = false;
  }
}

async function handleSubmit() {
  if (!keyDraft.value.trim()) {
    message.warning("请输入 API Key");
    return;
  }

  submitting.value = true;
  authStore.setKey(keyDraft.value);
  try {
    await getSystemStatus();
    await router.replace(redirectPath.value);
  } catch {
    authStore.setKey("");
    message.error("API Key 无效，请重试");
  } finally {
    submitting.value = false;
  }
}

onMounted(() => {
  void verifySetupState();
});
</script>

<template>
  <div class="min-h-full bg-slate-100 px-4 py-10">
    <div class="mx-auto max-w-md">
      <ui-card>
        <template #title>控制台鉴权</template>
        <p class="text-sm leading-6 text-slate-500">
          请输入后端 API Key 继续访问控制台。
        </p>

        <div v-if="checkingSetup" class="mt-4 text-sm text-slate-500">
          正在检查系统初始化状态...
        </div>

        <div v-else-if="setupCheckFailed" class="mt-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700">
          无法连接后端，请确认 API 服务已启动。
        </div>

        <form v-else class="mt-4 flex flex-col gap-3" @submit.prevent="handleSubmit">
          <ui-input
            v-model="keyDraft"
            type="password"
            placeholder="请输入 API Key"
            allow-clear
          />

          <ui-button type="primary" html-type="submit" :loading="submitting" class="w-full justify-center">
            登录
          </ui-button>
        </form>
      </ui-card>
    </div>
  </div>
</template>
