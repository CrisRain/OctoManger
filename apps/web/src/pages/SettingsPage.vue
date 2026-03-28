<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import {
  IconSettings,
  IconCloud,
  IconLock,
  IconTool,
  IconRefresh,
  IconCheck,
  IconSafe,
  IconInfoCircle,
} from "@/lib/icons";

import { useSystemStatus } from "@/composables/useDashboard";
import { useMessage, useErrorHandler, useSystemConfigs } from "@/composables";
import { useAuthStore } from "@/store";
import { storeToRefs } from "pinia";
import { PageHeader } from "@/components/index";

const { data: statusData, refresh: refreshStatusData } = useSystemStatus();
const message = useMessage();
const { withErrorHandler } = useErrorHandler();

const authStore = useAuthStore();
const { adminKey } = storeToRefs(authStore);
const adminKeySet = computed(() => Boolean(adminKey.value?.trim()));
const adminKeyDraft = ref("");
const adminKeyChanged = computed(() => adminKeyDraft.value !== (adminKey.value ?? ""));

const {
  config,
  configLoading,
  configSaving,
  isConfigDirty,
  loadConfigs,
  saveConfig,
  resetConfig,
} = useSystemConfigs();

onMounted(() => {
  void withErrorHandler(
    async () => {
      await loadConfigs();
    },
    { action: "加载配置" },
  );
});

watch(
  adminKey,
  (value) => {
    adminKeyDraft.value = value ?? "";
  },
  { immediate: true },
);

function persistAdminKey(value: string) {
  authStore.setKey(value);
}

function handleSaveAdminKey() {
  persistAdminKey(adminKeyDraft.value);
  message.success(adminKeyDraft.value.trim() ? "管理员密钥已保存" : "管理员密钥已清空");
}

async function handleSaveConfig() {
  await withErrorHandler(
    async () => {
      await saveConfig();
      message.success("系统配置已保存");
    },
    { action: "保存系统配置" },
  );
}

async function refreshStatus() {
  await withErrorHandler(
    async () => {
      await refreshStatusData();
      message.success("状态已刷新");
    },
    { action: "刷新状态" },
  );
}
</script>

<template>
  <div class="page-shell flex flex-col gap-6">
    <PageHeader
      title="设置"
      subtitle="查看系统运行状态，管理配置、认证与访问安全"
      icon-bg="linear-gradient(135deg, rgba(10,132,255,0.12), rgba(10,132,255,0.06))"
      icon-color="var(--accent)"
    >
      <template #icon><icon-settings /></template>
      <template #actions>
        <ui-button @click="refreshStatus">
          <template #icon><icon-refresh /></template>
          刷新状态
        </ui-button>
      </template>
    </PageHeader>

    <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
      <ui-card class="min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-cloud class="h-5 w-5 text-sky-600" />
            <span>系统状态</span>
          </div>
        </template>

        <div class="flex flex-col">
          <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
            <span class="text-xs font-semibold tracking-wider text-slate-500">数据库连接</span>
            <div class="flex items-center gap-2">
              <span
                class="inline-block h-2 w-2 flex-shrink-0 rounded-full transition-colors"
                :class="statusData?.database_ok ? 'bg-emerald-500 animate-pulse' : 'bg-red-500'"
              />
              <span
                class="text-sm font-semibold"
                :class="statusData?.database_ok ? 'text-emerald-700' : 'text-red-700'"
              >
                {{ statusData?.database_ok ? "正常" : "异常" }}
              </span>
            </div>
          </div>
          <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2">
            <span class="text-xs font-semibold tracking-wider text-slate-500">已加载插件数</span>
            <span class="text-sm font-medium text-slate-900">{{ statusData?.plugin_count ?? "—" }}</span>
          </div>
          <div
            v-if="statusData?.now"
            class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2"
          >
            <span class="text-xs font-semibold tracking-wider text-slate-500">服务器时间</span>
            <span class="text-sm font-medium text-slate-900 font-mono">
              {{ new Date(statusData.now).toLocaleString("zh-CN") }}
            </span>
          </div>
        </div>
      </ui-card>

      <ui-card class="min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-lock class="h-5 w-5 text-[var(--accent)]" />
            <span>管理员密钥设置</span>
          </div>
        </template>

        <div class="flex flex-col">
          <div class="flex items-center justify-between gap-4 py-4 max-md:flex-col max-md:items-start max-md:gap-2 border-b-0 pb-0">
            <span class="text-xs font-semibold tracking-wider text-slate-500">浏览器中的 Admin Key</span>
            <div class="flex items-center gap-2">
              <span
                class="inline-block h-2 w-2 flex-shrink-0 rounded-full transition-colors"
                :class="adminKeySet ? 'bg-emerald-500 animate-pulse' : 'bg-red-500'"
              />
              <span
                class="text-sm font-semibold"
                :class="adminKeySet ? 'text-emerald-700' : 'text-red-700'"
              >
                {{ adminKeySet ? "已配置" : "未配置" }}
              </span>
            </div>
          </div>
        </div>

        <form
          class="mt-4 rounded-xl border p-4 border-slate-200 bg-white/55 shadow-sm"
          @submit.prevent="handleSaveAdminKey"
        >
          <div class="flex flex-col gap-3">
            <ui-input
              v-model="adminKeyDraft"
              placeholder="输入 Admin Key..."
              allow-clear
              type="password"
              class="w-full"
            />
            <div class="flex flex-col gap-3 [@media(min-width:768px)]:grid [@media(min-width:768px)]:gap-3 [@media(min-width:768px)]:grid-cols-[minmax(0,_1fr)_auto_auto]">
              <p class="text-sm leading-6 text-slate-500">
                保存后会自动在请求中附带 <code>X-Admin-Key</code>，并持久化到浏览器本地存储。
              </p>
              <ui-button
                class="self-start max-md:w-full max-md:justify-center"
                :disabled="!adminKeyChanged"
                @click="adminKeyDraft = ''"
              >
                清空
              </ui-button>
              <ui-button
                type="primary"
                html-type="submit"
                class="self-start max-md:w-full max-md:justify-center"
                :disabled="!adminKeyChanged"
              >
                <template #icon><icon-check /></template>
                保存
              </ui-button>
            </div>
          </div>
        </form>

        <div class="text-sm leading-6 text-slate-500 mt-4 flex flex-col gap-2 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
          <p>您的 Admin Key 保存在浏览器本地存储中，可在鉴权页或本页更新。</p>
          <p>服务端 API Key 由初始化流程生成并存储在数据库中。</p>
        </div>
      </ui-card>
    </div>

    <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
      <ui-card class="min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-safe class="h-5 w-5 text-[var(--accent)]" />
            API 密钥管理
          </div>
        </template>

        <div class="flex flex-col gap-4">
          <p class="text-sm leading-6 text-slate-500">
            当前版本通过 <strong>Admin Key</strong> 进行 API 访问控制，服务端将密钥哈希保存到数据库。
          </p>
          <p class="text-sm leading-6 text-slate-500">
            初始化后前端需在鉴权页输入密钥，随后请求会自动携带认证头。
          </p>

          <div class="overflow-hidden rounded-xl border border-slate-800 bg-slate-900 shadow-sm">
            <div class="flex items-center justify-between gap-3 border-b border-slate-800 px-4 py-2.5">
              <div class="flex items-center gap-2">
                <span class="h-2.5 w-2.5 rounded-full bg-red-400"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-amber-400"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-emerald-400"></span>
              </div>
              <span class="text-xs font-semibold tracking-wider text-slate-500">HTTP Request Header</span>
            </div>
            <div class="px-4 py-4 font-mono text-sm">
              <code><span class="text-[var(--accent)]/80">X-Admin-Key</span><span class="text-slate-500">: </span><span class="text-slate-200">&lt;your-admin-key&gt;</span></code>
            </div>
          </div>

          <p class="text-sm leading-6 text-slate-500 flex items-start gap-2">
            <icon-info-circle class="mt-0.5 flex-shrink-0 text-sky-500" />
            <strong>提示：</strong>在本页填写管理员密钥后，浏览器会自动在请求中携带认证信息。
          </p>
        </div>
      </ui-card>

      <ui-card class="min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-lock class="h-5 w-5 text-[var(--accent)]" />
            SSL 安全证书管理
          </div>
        </template>

        <div class="rounded-xl border p-5 shadow-sm border-sky-200 bg-sky-50/70">
          <icon-info-circle class="mb-2 text-sky-600" />
          <p class="text-sm font-semibold text-slate-900">通过服务器配置管理 TLS</p>
          <p class="mt-1 text-sm leading-6 text-slate-600">
            当前版本的 SSL 证书管理通过服务器配置完成，不提供 Web 界面操作。
            请在服务器配置文件或反向代理（如 Nginx、Caddy）中配置 TLS 证书。
          </p>
        </div>
      </ui-card>
    </div>

    <ui-card class="min-w-0">
      <template #title>
        <div class="flex items-center gap-2">
          <icon-tool class="h-5 w-5 text-[var(--accent)] text-slate-500" />
          <span>系统配置</span>
        </div>
      </template>

      <template #extra>
        <span class="text-sm leading-6 text-slate-500">系统级参数已改为固定字段，插件配置已迁移到插件详情页</span>
      </template>

      <div class="flex flex-col gap-4">
        <div
          v-if="configLoading"
          class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-6 text-sm text-slate-500 shadow-sm"
        >
          正在加载配置...
        </div>

        <div v-else class="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-semibold text-slate-900">应用名称</p>
                <p class="mt-1 text-sm leading-6 text-slate-500">
                  控制台品牌名称，同时会显示在浏览器标题中。
                </p>
              </div>
              <code class="inline-flex items-center rounded-md border border-slate-200 bg-white px-2 py-0.5 text-xs font-mono text-slate-600">
                app_name
              </code>
            </div>

            <ui-input
              :model-value="config.appName"
              class="mt-4 w-full"
              placeholder="OctoManager"
              allow-clear
              @update:modelValue="config.appName = String($event ?? '')"
            />
          </div>

          <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-semibold text-slate-900">默认任务超时</p>
                <p class="mt-1 text-sm leading-6 text-slate-500">
                  新任务默认执行超时时间，单位为分钟。
                </p>
              </div>
              <code class="inline-flex items-center rounded-md border border-slate-200 bg-white px-2 py-0.5 text-xs font-mono text-slate-600">
                job_default_timeout_minutes
              </code>
            </div>

            <ui-input-number
              :model-value="config.jobDefaultTimeoutMinutes"
              class="mt-4 w-full"
              :min="0"
              :step="1"
              placeholder="30"
              @update:model-value="config.jobDefaultTimeoutMinutes = Number($event ?? 0)"
            />
          </div>

          <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-semibold text-slate-900">最大并发数</p>
                <p class="mt-1 text-sm leading-6 text-slate-500">
                  Worker 同时执行任务的上限。
                </p>
              </div>
              <code class="inline-flex items-center rounded-md border border-slate-200 bg-white px-2 py-0.5 text-xs font-mono text-slate-600">
                job_max_concurrency
              </code>
            </div>

            <ui-input-number
              :model-value="config.jobMaxConcurrency"
              class="mt-4 w-full"
              :min="0"
              :step="1"
              placeholder="10"
              @update:model-value="config.jobMaxConcurrency = Number($event ?? 0)"
            />
          </div>

          <div class="rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
            <p class="text-sm font-semibold text-slate-900">插件配置已独立管理</p>
            <p class="mt-1 text-sm leading-6 text-slate-500">
              插件专属设置和 gRPC 地址不再混在系统配置中，请前往对应插件详情页分别维护。
            </p>
            <div class="mt-4 flex flex-col gap-3 [@media(min-width:768px)]:grid [@media(min-width:768px)]:grid-cols-[minmax(0,_1fr)_auto_auto]">
              <p class="text-sm leading-6 text-slate-500">
                现在系统配置只保留全局字段，编辑更直接，也更便于后续扩展数据库结构。
              </p>
              <ui-button
                class="self-start max-md:w-full max-md:justify-center"
                :disabled="configSaving || !isConfigDirty"
                @click="resetConfig"
              >
                重置
              </ui-button>
              <ui-button
                type="primary"
                class="self-start max-md:w-full max-md:justify-center"
                :disabled="configSaving || !isConfigDirty"
                :loading="configSaving"
                @click="handleSaveConfig"
              >
                <template #icon><icon-check /></template>
                保存系统配置
              </ui-button>
            </div>
          </div>
        </div>
      </div>
    </ui-card>
  </div>
</template>
