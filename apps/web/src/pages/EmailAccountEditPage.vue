<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRoute } from "vue-router";
import { Message } from "@/lib/feedback";
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
      Message.success("配置已保存");
    } catch (e) {
      Message.error(e instanceof Error ? e.message : "保存失败");
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
    Message.error(e instanceof Error ? e.message : "生成授权链接失败");
  }
}

async function handleExchange() {
  try {
    await exchangeCode.execute(accountId, { code: authCode.value });
    authCode.value = "";
    Message.success("Code 交换成功");
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "交换失败");
  }
}
</script>

<template>
  <div class="page-container email-account-edit-page">
    <PageHeader
      title="配置邮箱账号"
      :subtitle="account ? `正在配置 ${account.address}` : ''"
      icon-bg="linear-gradient(135deg, rgba(234,88,12,0.12), rgba(249,115,22,0.12))"
      icon-color="var(--icon-orange)"
      :back-to="to.emailAccounts.list()"
      back-label="返回邮箱账号"
    >
      <template #icon><icon-email /></template>
    </PageHeader>

    <div v-if="loading" class="center-empty"><ui-spin :size="36" /></div>
    <div v-else-if="!account" class="center-empty"><p class="muted-copy">未找到该邮箱账号。</p></div>

    <ui-card v-else>
      <template #title>Outlook 配置与 OAuth</template>

      <div class="config-section">
        <ui-textarea
          v-model="configDraft"
          :auto-size="{ minRows: 8 }"
          class="mono-textarea"
        />
        <div v-if="configError" class="config-error">
          <icon-close-circle />
          {{ configError }}
        </div>
      </div>

      <div class="action-row">
        <ui-button :loading="patch.loading.value" @click="handleSaveConfig">
          <template #icon><icon-save /></template>
          保存配置
        </ui-button>
        <ui-button :loading="buildAuthorize.loading.value" @click="handleBuildAuthorize">
          <template #icon><icon-link /></template>
          生成授权链接
        </ui-button>
      </div>

      <div v-if="authorizeURL" class="authorize-url-box">
        <div class="authorize-url-label">授权链接（点击在新标签页打开）</div>
        <a :href="authorizeURL" target="_blank" rel="noreferrer" class="authorize-url-link">
          {{ authorizeURL }}
        </a>
      </div>

      <div class="exchange-row">
        <ui-input
          v-model="authCode"
          placeholder="粘贴 Authorization code"
          class="auth-code-input"
        />
        <ui-button
          type="primary"
          :disabled="!authCode.trim()"
          :loading="exchangeCode.loading.value"
          @click="handleExchange"
        >交换 Code</ui-button>
      </div>
    </ui-card>
  </div>
</template>
