<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { usePlugins, useSyncPlugins, usePluginSettings, usePluginRuntimeConfig, useExecutePluginAction } from "@/composables/usePlugins";
import { useCreateJobDefinition, useEnqueueJobExecution } from "@/composables/useJobs";
import { useCreateAgent, useStartAgent } from "@/composables/useAgents";
import { useAccounts } from "@/composables/useAccounts";
import { useMessage } from "@/composables";
import { PageHeader, PluginUIButtonForm } from "@/components/index";
import { to } from "@/router/registry";
import type { ExecutePluginActionResult, PluginUIButton, PluginUITab } from "@/types";
import {
  createPluginUIButtonFormState,
  resolvePluginUIButtonInput,
} from "@/utils/pluginUI";

const route = useRoute();
const router = useRouter();
const pluginKey = route.params.id as string;
const message = useMessage();

const { data: plugins, loading, refresh } = usePlugins();
const { data: accounts } = useAccounts();
const plugin = computed(() => plugins.value.find((p) => p.manifest.key === pluginKey));
const createJobDefinition = useCreateJobDefinition();
const enqueueJobExecution = useEnqueueJobExecution();
const createAgent = useCreateAgent();
const startAgent = useStartAgent();

// ── Sync ──────────────────────────────────────────────────────────────────
const sync = useSyncPlugins();

async function handleSync() {
  try {
    const result = await sync.execute();
    await refresh();
    if (result.failed === 0) {
      message.success(`已同步 ${result.synced} 个账号类型`);
    } else {
      const errMsg = result.errors.join("; ");
      message.warning(`部分同步失败：${errMsg}`);
    }
  } catch (e) {
    message.error(e instanceof Error ? e.message : "同步失败");
  }
}

// ── Settings ──────────────────────────────────────────────────────────────
const settings = usePluginSettings(pluginKey);
const runtimeConfig = usePluginRuntimeConfig(pluginKey);
const settingValues = ref<Record<string, string>>({});
const grpcAddress = ref("");
const grpcAddressChanged = computed(() =>
  grpcAddress.value.trim() !== (runtimeConfig.data.value.grpc_address ?? "").trim(),
);

// Load settings once plugin data is available
watch(plugin, async (p) => {
  if (!p) return;
  await Promise.all([settings.load(), runtimeConfig.load()]);
  // Seed form from loaded values
  const vals: Record<string, string> = {};
  for (const s of (p.manifest.settings ?? [])) {
    vals[s.key] = settings.data.value[s.key] != null
      ? String(settings.data.value[s.key])
      : "";
  }
  settingValues.value = vals;
  grpcAddress.value = runtimeConfig.data.value.grpc_address ?? "";
}, { immediate: true });

// ── Action execution ──────────────────────────────────────────────────────
const executeAction = useExecutePluginAction();
const pluginActionTabs = computed<PluginUITab[]>(() =>
  (plugin.value?.manifest.ui?.tabs ?? []).filter((tab) => {
    const context = String(tab.context ?? "").trim().toLowerCase();
    return context === "plugin" || context === "detail";
  })
);
const hasPluginUITabs = computed(() => pluginActionTabs.value.length > 0);
const activePluginUITab = ref<string>("");
const expandedAction = ref<string | null>(null);
const actionParams = ref<Record<string, string>>({});  // keyed by actionKey
const actionResults = ref<Record<string, ExecutePluginActionResult | null>>({});
const buttonFormValues = ref<Record<string, Record<string, unknown>>>({});
const buttonFormErrors = ref<Record<string, Record<string, string>>>({});
const buttonResults = ref<Record<string, ExecutePluginActionResult | null>>({});
const buttonLoading = ref<Record<string, boolean>>({});

watch(
  pluginActionTabs,
  (tabs) => {
    if (!tabs.length) {
      activePluginUITab.value = "";
      return;
    }
    if (!tabs.some((tab) => tab.key === activePluginUITab.value)) {
      activePluginUITab.value = tabs[0].key;
    }
  },
  { immediate: true }
);

function toggleActionPanel(actionKey: string) {
  expandedAction.value = expandedAction.value === actionKey ? null : actionKey;
}

function parseJsonSafe(text: string): Record<string, unknown> | undefined {
  if (!text.trim()) return undefined;
  try {
    const val = JSON.parse(text);
    if (val && typeof val === "object" && !Array.isArray(val)) return val as Record<string, unknown>;
    return undefined;
  } catch {
    return undefined;
  }
}

async function handleExecuteAction(actionKey: string) {
  const paramsText = actionParams.value[actionKey] ?? "";
  const params = parseJsonSafe(paramsText);

  try {
    const result = await executeAction.execute(pluginKey, actionKey, params);
    actionResults.value = { ...actionResults.value, [actionKey]: result };
    message.success(result.message || "执行成功");
  } catch (e) {
    message.error(e instanceof Error ? e.message : "执行失败");
  }
}

function initButtonForm(button: PluginUIButton) {
  if (!buttonFormValues.value[button.action]) {
    buttonFormValues.value[button.action] = createPluginUIButtonFormState(button);
  }
  if (!buttonFormErrors.value[button.action]) {
    buttonFormErrors.value[button.action] = {};
  }
}

function updateButtonForm(action: string, value: Record<string, unknown>) {
  buttonFormValues.value[action] = value;
  if (buttonFormErrors.value[action] && Object.keys(buttonFormErrors.value[action]).length > 0) {
    buttonFormErrors.value[action] = {};
  }
}

function resolveButtonInput(button: PluginUIButton) {
  return resolvePluginUIButtonInput(
    button,
    buttonFormValues.value[button.action] ?? {},
    accounts.value,
    { bindAccountContext: true },
  );
}

function buildPluginActionInput(params: Record<string, unknown>, account?: { id: number; identifier: string }) {
  const input: Record<string, unknown> = {};
  if (Object.keys(params).length > 0) {
    input.params = params;
  }
  if (account) {
    input.account = account;
  }
  return input;
}

async function createAndStartPluginAgent(
  button: PluginUIButton,
  params: Record<string, unknown>,
  account?: { id: number; identifier: string },
) {
  if (!plugin.value) return;

  const agent = await createAgent.execute({
    name: `${plugin.value.manifest.name || plugin.value.manifest.key} · ${button.label}`,
    plugin_key: plugin.value.manifest.key,
    action: button.action,
    input: buildPluginActionInput(params, account),
  });

  try {
    await startAgent.execute(agent.id);
    message.success(`已创建并启动 Agent：${button.label}`);
  } catch (e) {
    const detail = e instanceof Error ? e.message : "请在详情页重试";
    message.warning(`已创建 Agent，但启动失败：${detail}`);
  }

  router.push(to.agents.detail(agent.id));
}

async function executePluginUIButton(button: PluginUIButton) {
  if (!plugin.value) return;

  initButtonForm(button);
  buttonLoading.value[button.action] = true;
  buttonResults.value[button.action] = null;

  try {
    const { params, account } = resolveButtonInput(button);

    if (button.mode === "job") {
      const jobKey = `${plugin.value.manifest.key}-${button.action.toLowerCase()}-${Date.now()}`;
      const jobDefinition = await createJobDefinition.execute({
        key: jobKey,
        name: `${plugin.value.manifest.name || plugin.value.manifest.key} · ${button.label}`,
        plugin_key: plugin.value.manifest.key,
        action: button.action,
        input: buildPluginActionInput(params, account),
      });
      const execution = await enqueueJobExecution.execute(jobDefinition.id);
      message.success(`已提交后台作业：${button.label}`);
      router.push(to.jobs.executionDetail(execution.id));
      return;
    }

    if (button.mode === "agent") {
      await createAndStartPluginAgent(button, params, account);
      return;
    }

    const result = await executeAction.execute(plugin.value.manifest.key, button.action, params, undefined, account);
    buttonResults.value = { ...buttonResults.value, [button.action]: result };
    message.success(result.message || "执行成功");
  } catch (e) {
    message.error(e instanceof Error ? e.message : "执行失败");
  } finally {
    buttonLoading.value[button.action] = false;
  }
}

const SECRET_SETTING_KEYS = new Set(["token", "password", "secret", "api_key", "access_token", "refresh_token", "client_secret", "twocaptcha_api_key"]);

function isSecretSetting(key: string, secret: boolean): boolean {
  return secret || SECRET_SETTING_KEYS.has(key.toLowerCase());
}

function countTabButtons(tab: PluginUITab) {
  return tab.sections.reduce((total, section) => total + section.buttons.length, 0);
}

function describeButtonMode(button: PluginUIButton) {
  if (button.mode === "job") return "作业模式";
  if (button.mode === "agent") return "Agent 模式";
  return "即时模式";
}

function describeButtonForm(button: PluginUIButton) {
  if (!button.form.length) return "无需输入字段";
  return `${button.form.length} 个输入字段`;
}

async function saveSettings() {
  const payload: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(settingValues.value)) {
    if (v !== "") payload[k] = v;
  }
  try {
    await settings.save(payload);
    message.success("设置已保存");
  } catch (e) {
    message.error(e instanceof Error ? e.message : "保存失败");
  }
}

async function saveRuntimeConfig() {
  try {
    await runtimeConfig.save(grpcAddress.value.trim());
    grpcAddress.value = runtimeConfig.data.value.grpc_address ?? grpcAddress.value.trim();
    message.success("运行配置已保存");
  } catch (e) {
    message.error(e instanceof Error ? e.message : "保存失败");
  }
}
</script>

<template>
  <div class="page-shell">
    <PageHeader
      :title="plugin ? plugin.manifest.name : '插件详情'"
      icon-bg="linear-gradient(135deg, rgba(236,72,153,0.12), rgba(219,39,119,0.12))"
      icon-color="var(--icon-pink)"
      :back-to="to.plugins.list()"
      back-label="返回插件列表"
    >
      <template #icon><icon-apps /></template>
      <template #subtitle>
        <template v-if="plugin">
          <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600 text-[var(--accent)]">v{{ plugin.manifest.version }}</code>
          &nbsp;·&nbsp;
          <span class="inline-flex items-center rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-medium text-slate-600">@{{ plugin.manifest.key }}</span>
        </template>
      </template>
    </PageHeader>

    <div v-if="loading" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/55 shadow-sm">
      <ui-spin size="2.25em" />
    </div>
    <div v-else-if="!plugin" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/55 shadow-sm min-h-[240px]">
      <icon-apps class="h-12 w-12 text-slate-400" />
      <p class="text-base font-semibold text-slate-900">未找到该插件。</p>
    </div>

    <div v-else class="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,_1.15fr)_minmax(16em,_0.85fr)]">
      <div class="flex min-w-0 flex-col gap-6">
        <ui-card
          v-if="hasPluginUITabs"
          class="min-w-0 rounded-xl border overflow-hidden border-slate-200 bg-white shadow"
        >
          <template #title>
            <div class="flex items-center gap-2">
              <icon-thunderbolt class="h-4 w-4 text-[var(--accent)]" />
              插件入口
            </div>
          </template>

          <section class="mb-5 flex flex-col items-start gap-4 rounded-xl border p-5 sm:flex-row border-slate-200 bg-slate-50 shadow-sm">
            <div class="flex flex-1 flex-col gap-2">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--accent)]">Plugin Workbench</span>
              <h3 class="m-0 text-base font-bold text-slate-900">插件级操作面板</h3>
              <p class="text-xs leading-relaxed text-slate-500">
                这里集中放置插件声明的正式入口，可直接执行 sync、job、agent 三类动作，不再依赖 Go 侧预填账号配置。
              </p>
            </div>

            <div class="flex flex-shrink-0 flex-wrap items-center gap-4">
              <article class="flex flex-col items-center gap-0.5 px-3 text-center">
                <span class="text-xs text-slate-500">分组数量</span>
                <span class="text-xl font-bold text-slate-900">{{ pluginActionTabs.length }}</span>
              </article>
              <article class="flex flex-col items-center gap-0.5 px-3 text-center">
                <span class="text-xs text-slate-500">入口总数</span>
                <span class="text-xl font-bold text-slate-900">{{ pluginActionTabs.reduce((total, tab) => total + countTabButtons(tab), 0) }}</span>
              </article>
            </div>
          </section>

          <ui-tabs v-model:active-key="activePluginUITab" type="card" :destroy-on-hide="false" class="ui-tabs">
            <ui-tab-pane
              v-for="tab in pluginActionTabs"
              :key="tab.key"
              :title="tab.label"
            >
              <div class="p-5">
                <div class="flex flex-col gap-4">
                  <section
                    v-for="section in tab.sections"
                    :key="`${tab.key}-${section.title}`"
                    class="rounded-xl border p-4 md:p-5 border-slate-200 bg-slate-50"
                  >
                    <div class="flex items-center justify-between gap-3 border-b border-slate-200 px-4 py-3">
                      <div class="flex flex-col gap-0.5">
                        <p class="text-sm font-semibold text-slate-900">{{ section.title || tab.label }}</p>
                        <p class="text-xs text-slate-500">{{ section.buttons.length }} 个操作</p>
                      </div>
                      <span class="rounded border border-slate-200 bg-white px-2 py-0.5 text-xs font-mono text-slate-500">
                        {{ tab.label }}
                      </span>
                    </div>

                    <div
                      class="grid grid-cols-1 gap-4 p-4"
                      :class="section.buttons.length > 1 ? 'xl:grid-cols-2' : 'xl:grid-cols-1'"
                    >
                      <article
                        v-for="button in section.buttons"
                        :key="button.action"
                        class="flex flex-col gap-4 rounded-lg border p-4 border-slate-200 bg-white shadow-sm"
                      >
                        <div class="flex items-start justify-between gap-3">
                          <div class="flex flex-1 flex-col gap-2">
                            <code class="text-xs font-mono text-[var(--accent)]">{{ button.action }}</code>
                            <h4 class="m-0 text-sm font-semibold text-slate-900">{{ button.label }}</h4>
                            <div class="flex flex-wrap items-center gap-2">
                              <ui-tag size="small" :color="button.mode === 'job' ? 'blue' : button.mode === 'agent' ? 'cyan' : 'green'">
                                {{ describeButtonMode(button) }}
                              </ui-tag>
                              <span class="text-xs text-slate-500">{{ describeButtonForm(button) }}</span>
                            </div>
                          </div>
                          <span class="flex-shrink-0 rounded border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-xs font-mono font-semibold text-slate-500">
                            {{ button.form.length ? "表单" : "直接执行" }}
                          </span>
                        </div>

                        <PluginUIButtonForm
                          v-if="button.form.length"
                          :fields="button.form"
                          :model-value="buttonFormValues[button.action] ?? {}"
                          :errors="buttonFormErrors[button.action] ?? {}"
                          :accounts="accounts"
                          :default-account-type-key="plugin.manifest.key"
                          @vue:mounted="initButtonForm(button)"
                          @update:model-value="updateButtonForm(button.action, $event)"
                        />

                        <div v-else class="rounded-lg border border-dashed border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-500">
                          此操作无需额外参数，可直接执行。
                        </div>

                        <ui-button
                          :type="button.variant === 'primary' ? 'primary' : 'outline'"
                          class="self-start"
                          :loading="buttonLoading[button.action]"
                          @click="executePluginUIButton(button)"
                        >
                          <template #icon><icon-play-arrow /></template>
                          {{ button.label }}
                        </ui-button>

                        <div
                          v-if="buttonResults[button.action]"
                          class="flex flex-col gap-2 rounded-lg border border-emerald-200 bg-emerald-50/60 p-4"
                        >
                          <div class="flex items-center gap-2">
                            <icon-check-circle class="h-4 w-4 text-emerald-600" />
                            <span class="text-sm font-semibold text-emerald-700">{{ buttonResults[button.action]?.message || "执行成功" }}</span>
                          </div>
                          <pre
                            v-if="buttonResults[button.action]?.data"
                            class="overflow-x-auto rounded-lg border border-slate-200 bg-white px-3 py-2 text-xs font-mono text-slate-700"
                          >{{ JSON.stringify(buttonResults[button.action]?.data, null, 2) }}</pre>
                        </div>
                      </article>
                    </div>
                  </section>
                </div>
              </div>
            </ui-tab-pane>
          </ui-tabs>
        </ui-card>

        <ui-card class="min-w-0 rounded-xl border overflow-hidden border-slate-200 bg-white shadow">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-code-block class="h-4 w-4 text-sky-600" />
              {{ hasPluginUITabs ? "动作调试" : "可用操作" }}
            </div>
          </template>
          <div class="flex flex-col gap-3">
            <article
              v-for="act in plugin.manifest.actions"
              :key="act.key"
              class="flex flex-col items-start gap-0 rounded-xl border border-slate-200 bg-slate-50 shadow-sm overflow-hidden"
            >
              <div class="flex w-full items-center gap-3 px-4 py-3">
                <code class="inline-flex items-center rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-mono text-slate-600 flex-shrink-0">{{ act.key }}</code>
                <span class="flex-1 text-sm font-semibold text-slate-900">{{ act.name }}</span>
                <button
                  type="button"
                  class="inline-flex items-center gap-1.5 rounded-lg border border-[var(--accent)]/30 bg-[var(--accent)]/8 px-2.5 py-1 text-xs font-medium text-[var(--accent)] transition-colors hover:bg-[var(--accent)]/15 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]/50"
                  @click="toggleActionPanel(act.key)"
                >
                  <icon-play-arrow class="h-3 w-3" aria-hidden="true" />
                  {{ expandedAction === act.key ? '收起' : '测试执行' }}
                </button>
              </div>

              <p v-if="act.description" class="mx-4 mb-3 -mt-1 text-sm leading-6 text-slate-500">{{ act.description }}</p>

              <div
                v-if="expandedAction === act.key"
                class="w-full border-t border-slate-200 bg-white px-4 py-4 flex flex-col gap-3"
              >
                <div class="flex flex-col gap-1.5">
                  <label class="text-xs font-semibold text-slate-600">params <span class="font-normal text-slate-400">（JSON，可选）</span></label>
                  <ui-textarea
                    v-model="actionParams[act.key]"
                    :auto-size="{ minRows: 3, maxRows: 8 }"
                    placeholder="{}"
                    class="font-mono text-xs"
                  />
                </div>

                <div class="flex items-center justify-between gap-3">
                  <div class="flex items-center gap-2 text-xs text-slate-500">
                    <icon-info-circle class="h-3.5 w-3.5 flex-shrink-0" />
                    直接调用插件 action，不通过任务队列
                  </div>
                  <ui-button
                    type="primary"
                    size="small"
                    :loading="executeAction.loading.value"
                    @click="handleExecuteAction(act.key)"
                  >
                    <template #icon><icon-play-arrow /></template>
                    执行
                  </ui-button>
                </div>

                <div v-if="actionResults[act.key]" class="flex flex-col gap-1.5">
                  <div class="flex items-center gap-2">
                    <icon-check-circle class="h-3.5 w-3.5 text-emerald-600" aria-hidden="true" />
                    <span class="text-xs font-semibold text-emerald-700">{{ actionResults[act.key]?.message || '执行成功' }}</span>
                  </div>
                  <pre v-if="actionResults[act.key]?.data" class="overflow-x-auto rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs font-mono text-slate-700">{{ JSON.stringify(actionResults[act.key]?.data, null, 2) }}</pre>
                </div>
              </div>
            </article>
            <p v-if="!plugin.manifest.actions.length" class="text-sm leading-6 text-slate-500 italic">
              该插件暂无可用操作。
            </p>
          </div>
        </ui-card>
      </div>

      <!-- Sidebar info -->
      <div class="flex flex-col gap-6">
        <ui-card class="min-w-0 rounded-xl border overflow-hidden border-slate-200 bg-white shadow">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-info-circle class="h-4 w-4 text-sky-600" />
              插件信息
            </div>
          </template>
          <div class="flex flex-col">
            <div class="flex items-center justify-between gap-4 border-b border-slate-100 py-4 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start max-md:gap-2">
              <span class="text-xs font-semibold tracking-wider text-slate-500">健康状态</span>
              <div class="inline-flex items-center gap-2">
                <span class="inline-block h-2 w-2 flex-shrink-0 rounded-full" :class="plugin.healthy ? 'bg-emerald-500 animate-pulse' : 'bg-red-500'" />
                <ui-tag :color="plugin.healthy ? 'green' : 'red'">
                  {{ plugin.healthy ? "healthy" : "degraded" }}
                </ui-tag>
              </div>
            </div>
            <div v-if="plugin.manifest.description" class="flex items-center justify-between gap-4 border-b border-slate-100 py-4 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start max-md:gap-2">
              <span class="text-xs font-semibold tracking-wider text-slate-500">描述</span>
              <span class="text-sm font-medium text-slate-900 leading-7 text-slate-700 ml-auto text-left max-md:text-left">{{ plugin.manifest.description }}</span>
            </div>
            <div class="flex items-center justify-between gap-4 border-b border-slate-100 py-4 first:pt-0 last:border-b-0 last:pb-0 max-md:flex-col max-md:items-start max-md:gap-2 flex-col gap-3">
              <span class="text-xs font-semibold tracking-wider text-slate-500 mb-0">权限列表</span>
              <div v-if="plugin.manifest.capabilities.length" class="flex flex-wrap gap-2">
                <ui-tag v-for="cap in plugin.manifest.capabilities" :key="cap" size="small" class="whitespace-nowrap" color="blue">
                  {{ cap }}
                </ui-tag>
              </div>
              <span v-else class="text-sm leading-6 text-slate-500 italic">无特殊权限</span>
            </div>
          </div>
        </ui-card>

        <ui-card class="min-w-0 rounded-xl border overflow-hidden border-slate-200 bg-white shadow">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-tool class="h-4 w-4 text-amber-600" />
              操作
            </div>
          </template>
          <div class="flex flex-col gap-4">
            <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm flex-col">
              <p class="text-sm font-semibold text-slate-900">同步账号类型</p>
              <p class="text-sm leading-6 text-slate-500">将插件定义的账号类型同步到系统中</p>
              <ui-button
                size="small"
                type="outline"
                :loading="sync.loading.value"
                @click="handleSync"
              >
                <template #icon><icon-sync /></template>
                立即同步
              </ui-button>
            </div>
          </div>
        </ui-card>

        <ui-card class="min-w-0 rounded-xl border overflow-hidden border-slate-200 bg-white shadow">
          <template #title>
            <div class="flex items-center gap-2">
              <icon-cloud class="h-4 w-4 text-sky-600" />
              运行配置
            </div>
          </template>
          <ui-spin :loading="runtimeConfig.loading.value">
            <div class="flex flex-col gap-4">
              <div class="flex flex-col gap-2">
                <span class="text-sm font-semibold text-slate-900">gRPC 地址</span>
                <ui-input
                  v-model="grpcAddress"
                  allow-clear
                  class="w-full"
                  placeholder="127.0.0.1:50051"
                />
                <p class="text-sm leading-6 text-slate-500">
                  控制 Worker 连接该插件微服务时使用的地址，已与系统配置完全分离。
                </p>
              </div>

              <div class="flex items-center justify-end gap-3 border-t border-slate-200 pt-4 max-md:flex-col max-md:items-stretch">
                <ui-button
                  class="max-md:w-full max-md:justify-center"
                  :disabled="runtimeConfig.saving.value || !grpcAddressChanged"
                  @click="grpcAddress = runtimeConfig.data.value.grpc_address ?? ''"
                >
                  重置
                </ui-button>
                <ui-button
                  type="primary"
                  class="max-md:w-full max-md:justify-center"
                  :loading="runtimeConfig.saving.value"
                  :disabled="!grpcAddress.trim() || !grpcAddressChanged"
                  @click="saveRuntimeConfig"
                >
                  保存地址
                </ui-button>
              </div>
            </div>
          </ui-spin>
        </ui-card>

        <!-- Settings panel -->
        <ui-card
          v-if="plugin.manifest.settings?.length"
          class="min-w-0 rounded-xl border overflow-hidden border-slate-200 bg-white shadow"
        >
          <template #title>
            <div class="flex items-center gap-2">
              <icon-settings class="h-4 w-4 text-[var(--accent)]" />
              插件设置
            </div>
          </template>
          <ui-spin :loading="settings.loading.value">
            <ui-form layout="vertical" class="flex flex-col gap-4">
              <ui-form-item
                v-for="s in plugin.manifest.settings"
                :key="s.key"
                class="mb-0"
              >
                <template #label>
                  <span class="text-sm font-semibold text-slate-900">{{ s.label || s.key }}</span>
                  <ui-tag v-if="s.required" size="small" color="red" class="ml-2">必填</ui-tag>
                </template>
                <div class="flex flex-col gap-2">
                  <ui-input
                    v-if="isSecretSetting(s.key, s.secret)"
                    v-model="settingValues[s.key]"
                    type="password"
                    allow-clear
                    class="w-full"
                    :placeholder="s.required ? '必填' : '可选'"
                  />
                  <ui-input
                    v-else
                    v-model="settingValues[s.key]"
                    allow-clear
                    class="w-full"
                    :placeholder="s.required ? '必填' : '可选'"
                  />
                  <p v-if="s.description" class="text-sm leading-6 text-slate-500">{{ s.description }}</p>
                </div>
              </ui-form-item>

              <div class="flex items-center justify-end gap-3 border-t border-slate-200 pt-4 max-md:flex-col max-md:items-stretch">
                <ui-button
                  type="primary"
                  class="max-md:w-full max-md:justify-center"
                  :loading="settings.saving.value"
                  @click="saveSettings"
                >
                  保存设置
                </ui-button>
              </div>
            </ui-form>
          </ui-spin>
        </ui-card>
      </div>
    </div>
  </div>
</template>
