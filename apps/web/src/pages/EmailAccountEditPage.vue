<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRoute } from "vue-router";
import { useMessage } from "@/composables";
import {
  useEmailAccounts,
  usePatchEmailAccount,
  useBuildAuthorizeURL,
  useExchangeCode,
} from "@/composables/useEmailAccounts";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const route = useRoute();
const accountId = Number(route.params.id);
const message = useMessage();

const { data: accounts, loading } = useEmailAccounts();
const account = computed(() => accounts.value.find((item) => item.id === accountId));

const patch = usePatchEmailAccount();
const buildAuthorize = useBuildAuthorizeURL();
const exchangeCode = useExchangeCode();

const configDraft = ref("");
const authCode = ref("");
const authorizeURL = ref("");
const configError = ref("");

watch(account, (acc) => {
  if (acc) configDraft.value = JSON.stringify(acc.config ?? {}, null, 2);
}, { immediate: true });

function withParsedConfig(run: (config: Record<string, unknown>) => void) {
  try {
    const parsed = JSON.parse(configDraft.value) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      configError.value = "JSON 必须是一个对象";
      return;
    }
    configError.value = "";
    run(parsed as Record<string, unknown>);
  } catch (e) {
    configError.value = e instanceof Error ? e.message : "JSON 解析失败";
  }
}

async function handleSaveConfig() {
  withParsedConfig(async (config) => {
    try {
      await patch.execute(accountId, { config });
      message.success("配置已保存");
    } catch (e) {
      message.error(e instanceof Error ? e.message : "保存失败");
    }
  });
}

async function handleBuildAuthorize() {
  try {
    const result = await buildAuthorize.execute(accountId);
    if (result && typeof result === "object" && "authorize_url" in result) {
      authorizeURL.value = result.authorize_url as string;
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : "生成授权链接失败");
  }
}

async function handleExchange() {
  try {
    await exchangeCode.execute(accountId, { code: authCode.value });
    authCode.value = "";
    message.success("Code 交换成功");
  } catch (e) {
    message.error(e instanceof Error ? e.message : "交换失败");
  }
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="编辑邮箱账号"
      :subtitle="account ? `正在编辑 ${account.address}` : ''"
      icon-bg="linear-gradient(135deg, rgba(234,88,12,0.12), rgba(249,115,22,0.12))"
      icon-color="var(--icon-orange)"
      :back-to="to.emailAccounts.list()"
      back-label="返回邮箱账号列表"
    >
      <template #icon><icon-email /></template>
    </PageHeader>

    <div v-if="loading" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/[56%] shadow-sm"><ui-spin size="2.25em" /></div>
    <div v-else-if="!account" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/[56%] shadow-sm"><p class="text-sm leading-6 text-slate-500">未找到该邮箱账号。</p></div>

    <ui-card v-else>
      <template #title>邮箱配置与 OAuth</template>

      <div class="mb-4">
        <ui-textarea
          v-model="configDraft"
          :auto-size="{ minRows: 8 }"
          class="font-mono"
        />
        <div v-if="configError" class="mt-2 flex items-center gap-2 text-sm text-red-600">
          <icon-close-circle class="flex-shrink-0" />
          {{ configError }}
        </div>
      </div>

      <div class="flex flex-wrap gap-3">
        <ui-button :loading="patch.loading.value" @click="handleSaveConfig">
          <template #icon><icon-save /></template>
          保存配置
        </ui-button>
        <ui-button :loading="buildAuthorize.loading.value" @click="handleBuildAuthorize">
          <template #icon><icon-link /></template>
          生成授权链接
        </ui-button>
      </div>

      <div v-if="authorizeURL" class="mt-4 rounded-xl border border-sky-200 bg-sky-50/70 p-4">
        <p class="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-500">授权链接（点击在新标签页打开）</p>
        <a :href="authorizeURL" target="_blank" rel="noreferrer" class="break-all text-sm font-medium text-[var(--accent)] underline decoration-dotted hover:decoration-solid">
          {{ authorizeURL }}
        </a>
      </div>

      <div class="mt-4 flex flex-wrap items-start gap-3">
        <ui-input
          v-model="authCode"
          placeholder="粘贴授权码"
          class="flex-1"
        />
        <ui-button
          type="primary"
          :disabled="!authCode.trim()"
          :loading="exchangeCode.loading.value"
          @click="handleExchange"
        >交换授权码</ui-button>
      </div>
    </ui-card>
  </div>
</template>
