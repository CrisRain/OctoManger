<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Message } from "@/lib/feedback";
import { useAccounts } from "@/composables/useAccounts";
import { useExecuteAccount } from "@/composables/useAccounts";
import { usePlugins } from "@/composables/usePlugins";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";
import type {
  AccountExecuteResult,
  PluginUIButton,
  PluginUIFormField,
} from "@/types";

const route = useRoute();
const router = useRouter();
const accountId = Number(route.params.id);

const { data: accounts, loading } = useAccounts();
const { data: plugins } = usePlugins();
const executeAccount = useExecuteAccount();

const account = computed(() => accounts.value.find((a) => a.id === accountId) ?? null);

const plugin = computed(() => {
  if (!account.value?.account_type_key) return null;
  return plugins.value.find((p) => p.manifest.key === account.value!.account_type_key) ?? null;
});

// ── Spec display ──────────────────────────────────────────────────────────
const SECRET_KEYS = new Set(["token", "password", "secret", "access_token", "refresh_token", "client_secret", "api_key"]);

function isSecret(key: string): boolean {
  return SECRET_KEYS.has(key.toLowerCase());
}

function displaySpecValue(key: string, val: unknown): string {
  if (val === null || val === undefined) return "—";
  if (isSecret(key)) return "••••••••";
  if (typeof val === "object") return JSON.stringify(val);
  return String(val);
}

const specEntries = computed((): [string, unknown][] => {
  if (!account.value?.spec) return [];
  return Object.entries(account.value.spec);
});

// ── Tab state ─────────────────────────────────────────────────────────────
const activeTab = ref("info");

// Whether this plugin has custom UI tabs defined
const hasUITabs = computed(() =>
  (plugin.value?.manifest.ui?.tabs?.length ?? 0) > 0
);

// ── Custom UI execution ───────────────────────────────────────────────────
// Active UI sub-tab (within the Actions tab)
const activeUITab = ref<string>("");

const uiTabs = computed(() =>
  (plugin.value?.manifest.ui?.tabs ?? []).filter(tab => tab.context !== "create")
);

watch(
  uiTabs,
  (tabs) => {
    if (!tabs.length) {
      activeUITab.value = "";
      return;
    }
    if (!tabs.some((tab) => tab.key === activeUITab.value)) {
      activeUITab.value = tabs[0].key;
    }
  },
  { immediate: true }
);

// Per-button form state: Map<button type="button"Action, formValues>
const formValues = ref<Record<string, Record<string, string>>>({});
const execResults = ref<Record<string, AccountExecuteResult | null>>({});
const execLoading = ref<Record<string, boolean>>({});

function initForm(button: PluginUIButton) {
  if (!formValues.value[button.action]) {
    const vals: Record<string, string> = {};
    for (const f of button.form) {
      vals[f.name] = "";
    }
    formValues.value[button.action] = vals;
  }
}

function fieldValue(action: string, name: string): string {
  return formValues.value[action]?.[name] ?? "";
}

function setFieldValue(action: string, name: string, val: string) {
  if (!formValues.value[action]) formValues.value[action] = {};
  formValues.value[action][name] = val;
}

async function executeButton(button: PluginUIButton) {
  if (!account.value) return;
  execLoading.value[button.action] = true;
  execResults.value[button.action] = null;
  try {
    const params: Record<string, unknown> = {
      ...button.params,
      ...formValues.value[button.action],
    };
    const res = await executeAccount.execute(accountId, button.action, params);
    execResults.value[button.action] = res;
    if (res.status === "ok") {
      Message.success("执行成功");
    } else {
      Message.warning(`执行失败: ${res.error_message ?? res.error_code}`);
    }
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "执行失败");
  } finally {
    execLoading.value[button.action] = false;
  }
}

function isSecretField(field: PluginUIFormField): boolean {
  return SECRET_KEYS.has(field.name.toLowerCase()) || field.type === "password";
}

// ── Fallback: flat action list (no ui.tabs) ───────────────────────────────
const selectedAction = ref<string | null>(null);
const paramsJSON = ref("{}");
const execResult = ref<AccountExecuteResult | null>(null);
const paramsError = ref("");

function selectAction(key: string) {
  selectedAction.value = key;
  paramsJSON.value = "{}";
  execResult.value = null;
  paramsError.value = "";
}

async function handleExecute() {
  if (!account.value || !selectedAction.value) return;
  paramsError.value = "";
  let params: Record<string, unknown> = {};
  try {
    params = JSON.parse(paramsJSON.value) as Record<string, unknown>;
  } catch {
    paramsError.value = "参数必须是合法 JSON";
    return;
  }
  try {
    const res = await executeAccount.execute(accountId, selectedAction.value, params);
    execResult.value = res;
    if (res.status === "ok") {
      Message.success("执行成功");
    } else {
      Message.warning(`执行失败: ${res.error_message ?? res.error_code}`);
    }
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "执行失败");
  }
}

const statusColor: Record<string, string> = { active: "green", pending: "gray", inactive: "gray" };
const statusDot: Record<string, string>   = { active: "online", pending: "neutral", inactive: "neutral" };
const actionCount = computed(() => plugin.value?.manifest.actions?.length ?? 0);
const tagCount = computed(() => account.value?.tags?.length ?? 0);

const pluginDisplayName = computed(() => {
  if (plugin.value?.manifest.name) return plugin.value.manifest.name;
  return account.value?.account_type_key || "未匹配模型";
});

const overviewDescription = computed(() => {
  if (!account.value) return "";
  if (plugin.value?.manifest.description) return plugin.value.manifest.description;
  if (account.value.account_type_key) {
    return `当前账号绑定到 ${account.value.account_type_key} 模型，下面集中展示运行信息、凭据配置和可执行动作。`;
  }
  return "下面集中展示运行信息、凭据配置和可执行动作。";
});

const overviewTagPreview = computed(() => {
  const tags = account.value?.tags ?? [];
  if (!tags.length) return "未设置标签";
  if (tags.length <= 2) return tags.join(" · ");
  return `${tags.slice(0, 2).join(" · ")} +${tags.length - 2}`;
});

function describeSpecValue(key: string, val: unknown): string {
  if (isSecret(key)) return "敏感字段已脱敏";
  if (Array.isArray(val)) return `数组 · ${val.length} 项`;
  if (val && typeof val === "object") return "对象结构";
  if (typeof val === "number") return "数值参数";
  if (typeof val === "boolean") return "布尔参数";
  if (typeof val === "string" && /^https?:\/\//i.test(val)) return "URL 地址";
  return "文本参数";
}

function countTabButtons(tab: { sections: Array<{ buttons: PluginUIButton[] }> }) {
  return tab.sections.reduce((total, section) => total + section.buttons.length, 0);
}

function describeButtonMode(button: PluginUIButton) {
  return button.mode === "job" ? "作业模式" : "即时模式";
}

function describeButtonForm(button: PluginUIButton) {
  if (!button.form.length) return "无需输入字段";
  return `${button.form.length} 个输入字段`;
}

const selectedActionMeta = computed(() =>
  plugin.value?.manifest.actions.find((action) => action.key === selectedAction.value) ?? null
);
</script>

<template>
  <div class="page-container account-page">
    <PageHeader
      :title="account ? account.identifier : '账号资源详情'"
      icon-bg="var(--accent-light)"
      icon-color="var(--accent)"
      :back-to="to.accounts.list()"
      back-label="返回列表"
    >
      <template #icon><icon-user /></template>
      <template #subtitle>
        <code v-if="account" class="key-badge">模型: {{ account.account_type_key || "—" }}</code>
      </template>
      <template #actions>
        <div v-if="account" class="account-header-actions">
          <div class="status-cell">
            <span class="status-dot-large" :class="statusDot[account.status] ?? 'neutral'" />
            <ui-tag :color="statusColor[account.status] ?? 'gray'" class="status-tag-pill">{{ account.status }}</ui-tag>
          </div>
          <ui-button type="primary" class="edit-btn" @click="router.push(to.accounts.edit(accountId))">
            <template #icon><icon-edit /></template>
            配置与编辑
          </ui-button>
        </div>
      </template>
    </PageHeader>

    <div v-if="loading" class="empty-wrap">
      <ui-spin :size="36" />
    </div>
    <div v-else-if="!account" class="empty-wrap">
      <icon-user class="empty-icon" />
      <p class="empty-msg">未找到该账号。</p>
      <ui-button @click="router.push(to.accounts.list())">返回列表</ui-button>
    </div>

    <div v-else-if="account" class="account-shell">
      <section class="account-overview">
        <div class="overview-copy">
          <span class="overview-eyebrow">账号资源概览</span>
          <h2 class="overview-title">{{ account.identifier }}</h2>
          <p class="overview-description">{{ overviewDescription }}</p>

          <div class="overview-pills">
            <div class="overview-status-chip">
              <span class="status-dot-large" :class="statusDot[account.status] ?? 'neutral'" />
              <ui-tag :color="statusColor[account.status] ?? 'gray'" class="status-tag-pill">
                {{ account.status }}
              </ui-tag>
            </div>

            <div class="overview-meta-chip">
              <span>类型模型</span>
              <code class="overview-meta-key">{{ account.account_type_key || "—" }}</code>
            </div>

            <div class="overview-meta-chip">
              <span>标签预览</span>
              <span class="overview-meta-value">{{ overviewTagPreview }}</span>
            </div>
          </div>
        </div>

        <div class="overview-stats">
          <div class="overview-stat">
            <span class="overview-stat-label">绑定模型</span>
            <div class="overview-stat-value">
              <span class="overview-stat-primary">{{ pluginDisplayName }}</span>
            </div>
            <span class="overview-stat-sub">{{ account.account_type_key || "未注册模型" }}</span>
          </div>

          <div class="overview-stat">
            <span class="overview-stat-label">凭据字段</span>
            <div class="overview-stat-value">
              <span class="overview-stat-number">{{ specEntries.length }}</span>
            </div>
            <span class="overview-stat-sub">当前 Spec 项数量</span>
          </div>

          <div class="overview-stat">
            <span class="overview-stat-label">可用动作</span>
            <div class="overview-stat-value">
              <span class="overview-stat-number">{{ actionCount }}</span>
            </div>
            <span class="overview-stat-sub">{{ hasUITabs ? "支持分组动作面板" : "使用基础动作面板" }}</span>
          </div>

          <div class="overview-stat">
            <span class="overview-stat-label">标签数量</span>
            <div class="overview-stat-value">
              <span class="overview-stat-number">{{ tagCount }}</span>
            </div>
            <span class="overview-stat-sub">{{ overviewTagPreview }}</span>
          </div>
        </div>
      </section>

      <ui-tabs
        v-model:active-key="activeTab"
        :destroy-on-hide="false"
        class="detail-tabs premium-tabs account-tabs"
      >

        <!-- ── Tab 1: Info ─────────────────────────────────────────────────── -->
        <ui-tab-pane key="info" title="基础运行信息">
          <div class="tab-panel">
            <div class="info-grid">
              <ui-card class="premium-card info-runtime-card">
                <template #title>
                  <div class="card-header-with-icon">
                    <div class="card-icon-box"><icon-info-circle /></div>
                    基本信息与状态
                  </div>
                </template>
                <div class="info-card-body">
                  <section class="panel-hero panel-hero--account">
                    <div class="panel-hero-copy">
                      <span class="panel-kicker">Account Profile</span>
                      <h3 class="panel-title">{{ account.identifier }}</h3>
                      <p class="panel-description">
                        当前账号挂载在 <code class="inline-code-chip">{{ pluginDisplayName }}</code>
                        模型下，可在这里快速查看资源身份、状态与标签概览。
                      </p>
                    </div>

                    <div class="hero-side hero-side--status">
                      <span class="hero-side-label">运行状态</span>
                      <div class="status-cell status-cell--hero">
                        <span class="status-dot-large" :class="statusDot[account.status] ?? 'neutral'" />
                        <ui-tag :color="statusColor[account.status] ?? 'gray'" class="status-tag-pill">{{ account.status }}</ui-tag>
                      </div>
                    </div>
                  </section>

                  <div class="info-stat-grid">
                    <article class="info-stat-card">
                      <span class="info-stat-label">数据库主键</span>
                      <span class="info-stat-value mono-text">#{{ account.id }}</span>
                      <span class="info-stat-sub">系统资源编号</span>
                    </article>

                    <article class="info-stat-card">
                      <span class="info-stat-label">派生应用模型</span>
                      <code class="info-stat-chip">{{ account.account_type_key || "—" }}</code>
                      <span class="info-stat-sub">当前绑定类型</span>
                    </article>

                    <article class="info-stat-card">
                      <span class="info-stat-label">标签数量</span>
                      <span class="info-stat-value">{{ tagCount }}</span>
                      <span class="info-stat-sub">{{ overviewTagPreview }}</span>
                    </article>

                    <article class="info-stat-card">
                      <span class="info-stat-label">可用动作</span>
                      <span class="info-stat-value">{{ actionCount }}</span>
                      <span class="info-stat-sub">{{ hasUITabs ? "分组动作面板" : "基础动作面板" }}</span>
                    </article>
                  </div>

                  <section class="tag-shelf">
                    <div class="tag-shelf-header">
                      <span class="tag-shelf-title">分类标签</span>
                      <span class="tag-shelf-note">Tags / Labels</span>
                    </div>

                    <div v-if="account.tags?.length" class="tags-cell tags-cell--shelf">
                      <ui-tag v-for="tag in account.tags" :key="tag" class="status-tag-pill tag-pill">
                        {{ tag }}
                      </ui-tag>
                    </div>
                    <div v-else class="tag-shelf-empty">
                      当前账号还没有设置任何标签
                    </div>
                  </section>
                </div>
              </ui-card>

              <ui-card v-if="specEntries.length" class="premium-card info-spec-card">
                <template #title>
                  <div class="card-header-with-icon">
                    <div class="card-icon-box orange"><icon-lock /></div>
                    凭据密钥与 Spec 结构
                  </div>
                </template>
                <div class="spec-card-body">
                  <section class="panel-hero panel-hero--spec">
                    <div class="panel-hero-copy">
                      <span class="panel-kicker">Credential Payload</span>
                      <h3 class="panel-title">Spec 结构映射</h3>
                      <p class="panel-description">
                        每个字段拆成独立条目展示，敏感值自动脱敏，方便快速核对当前账号的配置结构。
                      </p>
                    </div>

                    <div class="hero-side hero-side--count">
                      <span class="hero-side-label">字段总数</span>
                      <span class="hero-count">{{ specEntries.length }}</span>
                    </div>
                  </section>

                  <div class="spec-grid">
                    <article
                      v-for="[key, val] in specEntries"
                      :key="key"
                      class="spec-entry"
                      :class="{ 'spec-entry--secret': isSecret(key) }"
                    >
                      <div class="spec-entry-side">
                        <span class="spec-entry-label">字段名</span>
                        <code class="spec-entry-code">{{ key }}</code>
                      </div>

                      <div class="spec-entry-side spec-entry-side--value">
                        <span class="spec-entry-label">{{ describeSpecValue(key, val) }}</span>
                        <span class="spec-entry-value mono-text" :class="{ 'spec-entry-value--secret': isSecret(key) }">
                          {{ displaySpecValue(key, val) }}
                        </span>
                      </div>
                    </article>
                  </div>
                </div>
              </ui-card>

              <ui-card v-else class="premium-card info-spec-card">
                <template #title>
                  <div class="card-header-with-icon">
                    <div class="card-icon-box gray"><icon-lock /></div>
                    凭据密钥与 Spec 结构
                  </div>
                </template>
                <div class="spec-card-empty">
                  <div class="spec-card-empty-badge">0</div>
                  <p class="spec-card-empty-title">当前模型未要求任何凭据字段</p>
                  <p class="spec-card-empty-copy">如果后续插件补充了 Spec 结构，这里会自动展示对应字段。</p>
                </div>
              </ui-card>
            </div>
          </div>
        </ui-tab-pane>

        <!-- ── Tab 2: Actions ─────────────────────────────────────────────── -->
        <ui-tab-pane key="actions" title="操作">
          <div class="tab-panel">
            <div v-if="!plugin" class="empty-wrap tab-empty">
              <icon-apps class="empty-icon" />
              <p class="empty-msg">
                未找到插件 <code class="key-badge">{{ account.account_type_key }}</code>，
                请确认 Worker 已启动并注册该插件。
              </p>
            </div>

            <template v-else>
              <section class="actions-overview-panel">
                <div class="actions-overview-copy">
                  <span class="panel-kicker">Action Workbench</span>
                  <h3 class="actions-overview-title">账号动作工作台</h3>
                  <p class="panel-description">
                    {{ hasUITabs
                      ? "按分组浏览插件暴露的动作能力，带表单的动作会在同一工作台内直接配置和执行。"
                      : "当前插件提供基础动作列表，可在左侧选择动作，并在右侧构造 JSON 参数后执行。"
                    }}
                  </p>
                </div>

                <div class="actions-overview-stats">
                  <article class="actions-mini-stat">
                    <span class="actions-mini-label">总动作数</span>
                    <span class="actions-mini-value">{{ actionCount }}</span>
                  </article>
                  <article class="actions-mini-stat">
                    <span class="actions-mini-label">分组标签</span>
                    <span class="actions-mini-value">{{ hasUITabs ? uiTabs.length : 1 }}</span>
                  </article>
                  <article class="actions-mini-stat">
                    <span class="actions-mini-label">工作模式</span>
                    <span class="actions-mini-text">{{ hasUITabs ? "分组工作台" : "基础执行台" }}</span>
                  </article>
                </div>
              </section>

              <!-- ── Custom UI: ui.tabs ──────────────────────────────────────── -->
              <template v-if="hasUITabs">
                <div class="action-workbench action-workbench--custom">
                  <ui-tabs v-model:active-key="activeUITab" type="card" :destroy-on-hide="false" class="ui-tabs action-tab-switcher">
                    <ui-tab-pane
                      v-for="tab in uiTabs"
                      :key="tab.key"
                      :title="tab.label"
                    >
                      <div class="action-stage">
                        <section class="action-stage-hero">
                          <div class="action-stage-copy">
                            <span class="panel-kicker">Action Group</span>
                            <h3 class="action-stage-title">{{ tab.label }}</h3>
                            <p class="action-stage-description">
                              这个分组下共有 {{ countTabButtons(tab) }} 个动作，可直接在当前页面完成参数填写和执行。
                            </p>
                          </div>

                          <div class="hero-side hero-side--count">
                            <span class="hero-side-label">动作数量</span>
                            <span class="hero-count">{{ countTabButtons(tab) }}</span>
                          </div>
                        </section>

                        <div class="ui-section-stack">
                          <section
                            v-for="section in tab.sections"
                            :key="section.title"
                            class="ui-section-card"
                          >
                            <div class="ui-section-head">
                              <div class="ui-section-copy">
                                <p class="ui-section-title">{{ section.title || tab.label }}</p>
                                <p class="ui-section-subtitle">{{ section.buttons.length }} 个动作待执行</p>
                              </div>
                            </div>

                            <div class="ui-command-grid">
                              <article
                                v-for="button in section.buttons"
                                :key="button.action"
                                class="ui-command-card"
                              >
                                <div class="ui-command-head">
                                  <div class="ui-command-copy">
                                    <code class="ui-command-code">{{ button.action }}</code>
                                    <h4 class="ui-command-title">{{ button.label }}</h4>
                                    <div class="ui-command-meta">
                                      <ui-tag size="small" :color="button.mode === 'job' ? 'twblue' : 'gray'" class="mode-tag-pill">
                                        {{ describeButtonMode(button) }}
                                      </ui-tag>
                                      <span class="ui-command-meta-text">{{ describeButtonForm(button) }}</span>
                                    </div>
                                  </div>
                                  <span class="ui-command-badge">{{ button.form.length ? "Form" : "Direct" }}</span>
                                </div>

                                <ui-form
                                  v-if="button.form.length"
                                  layout="vertical"
                                  class="ui-button-form command-form"
                                  :model="formValues[button.action] ?? {}"
                                  @vue:mounted="initForm(button)"
                                >
                                  <div class="command-field-grid">
                                    <ui-form-item
                                      v-for="field in button.form"
                                      :key="field.name"
                                      class="command-form-item"
                                    >
                                      <template #label>
                                        <span>{{ field.name }}</span>
                                        <ui-tag v-if="field.required" size="small" color="red" class="required-tag">必填</ui-tag>
                                        <span v-if="field.description" class="field-desc-inline">— {{ field.description }}</span>
                                      </template>

                                      <ui-select
                                        v-if="field.choices?.length"
                                        :model-value="fieldValue(button.action, field.name)"
                                        allow-clear
                                        :placeholder="field.required ? '必填' : '可选'"
                                        @change="(v: string) => setFieldValue(button.action, field.name, v)"
                                      >
                                        <ui-option v-for="c in field.choices" :key="c" :value="c">{{ c }}</ui-option>
                                      </ui-select>
                                      <ui-input
                                        v-else-if="isSecretField(field)"
                                        :model-value="fieldValue(button.action, field.name)"
                                        type="password"
                                        allow-clear
                                        :placeholder="field.required ? '必填' : '可选'"
                                        @input="(v: string) => setFieldValue(button.action, field.name, v)"
                                      />
                                      <ui-input
                                        v-else
                                        :model-value="fieldValue(button.action, field.name)"
                                        allow-clear
                                        :placeholder="field.required ? '必填' : '可选'"
                                        @input="(v: string) => setFieldValue(button.action, field.name, v)"
                                      />
                                    </ui-form-item>
                                  </div>

                                  <ui-button
                                    type="primary"
                                    class="command-submit"
                                    :loading="execLoading[button.action]"
                                    @click="executeButton(button)"
                                  >
                                    <template #icon><icon-play-arrow /></template>
                                    {{ button.label }}
                                  </ui-button>
                                </ui-form>

                                <div v-else class="command-direct">
                                  <p class="command-direct-copy">这个动作无需额外输入字段，可以直接发起执行。</p>
                                  <ui-button
                                    :type="button.variant === 'primary' ? 'primary' : 'outline'"
                                    class="command-submit command-submit--inline"
                                    :loading="execLoading[button.action]"
                                    @click="initForm(button); executeButton(button)"
                                  >
                                    <template #icon><icon-play-arrow /></template>
                                    {{ button.label }}
                                  </ui-button>
                                </div>

                                <div
                                  v-if="execResults[button.action]"
                                  class="exec-result"
                                  :class="execResults[button.action]!.status === 'ok' ? 'result-ok' : 'result-err'"
                                >
                                  <div class="result-header">
                                    <icon-check-circle
                                      v-if="execResults[button.action]!.status === 'ok'"
                                      class="result-icon result-icon--ok"
                                    />
                                    <icon-close-circle v-else class="result-icon result-icon--err" />
                                    <span class="result-title">
                                      {{ execResults[button.action]!.status === "ok"
                                        ? "执行成功"
                                        : `错误: ${execResults[button.action]!.error_code}` }}
                                    </span>
                                  </div>
                                  <p v-if="execResults[button.action]!.error_message" class="result-error-msg">
                                    {{ execResults[button.action]!.error_message }}
                                  </p>
                                  <pre v-if="execResults[button.action]!.result" class="result-json">{{
                                    JSON.stringify(execResults[button.action]!.result, null, 2)
                                  }}</pre>
                                </div>
                              </article>
                            </div>
                          </section>
                        </div>
                      </div>
                    </ui-tab-pane>
                  </ui-tabs>
                </div>
              </template>

              <!-- ── Fallback: flat action list ─────────────────────────────── -->
              <div v-else class="action-workbench actions-layout">
                <aside class="action-rail">
                  <div class="action-rail-head">
                    <span class="panel-kicker">Action Directory</span>
                    <h3 class="action-rail-title">动作目录</h3>
                    <p class="action-rail-copy">选择一个原子动作，在右侧构造 JSON 参数并立即执行。</p>
                  </div>

                  <div class="action-list-panel">
                    <button
                      v-for="act in plugin.manifest.actions"
                      :key="act.key"
                      type="button"
                      class="action-item"
                      :class="{ 'action-item--active': selectedAction === act.key }"
                      @click="selectAction(act.key)"
                    >
                      <div class="action-item-top">
                        <code class="key-badge">{{ act.key }}</code>
                        <span class="action-item-badge">Action</span>
                      </div>
                      <p class="action-item-name">{{ act.name }}</p>
                      <p v-if="act.description" class="action-item-desc">{{ act.description }}</p>
                    </button>
                    <ui-empty v-if="!plugin.manifest.actions.length" description="该插件模型未注册任何动作" />
                  </div>
                </aside>

                <div class="action-exec-panel">
                  <div v-if="!selectedAction" class="exec-placeholder action-stage-empty">
                    <icon-thunderbolt class="placeholder-icon" />
                    <p>请在左侧选择要执行的原子动作</p>
                  </div>
                  <template v-else>
                    <ui-card class="premium-card action-stage-card">
                      <template #title>
                        <div class="card-header-with-icon">
                          <div class="card-icon-box purple"><icon-code-block /></div>
                          {{ selectedActionMeta?.name || "运行时调用" }}
                        </div>
                      </template>
                      <div class="action-stage-card-body">
                        <section class="action-stage-summary">
                          <div class="action-stage-summary-copy">
                            <code class="ui-command-code">{{ selectedAction }}</code>
                            <p class="action-stage-summary-text">
                              {{ selectedActionMeta?.description || "为这个原子动作填写 JSON 参数，然后在当前页面直接发起执行。" }}
                            </p>
                          </div>
                          <span class="action-stage-badge">JSON</span>
                        </section>

                        <ui-form layout="vertical" :model="{}">
                          <ui-form-item label="动态挂载参数 (JSON 结构)">
                            <ui-textarea
                              v-model="paramsJSON"
                              :auto-size="{ minRows: 6, maxRows: 14 }"
                              class="mono-field premium-textarea"
                              placeholder="{}"
                            />
                            <p v-if="paramsError" class="field-error">{{ paramsError }}</p>
                          </ui-form-item>
                          <ui-button
                            type="primary"
                            class="execute-btn"
                            :loading="executeAccount.loading.value"
                            @click="handleExecute"
                          >
                            <template #icon><icon-play-arrow /></template>
                            发送执行指令
                          </ui-button>
                        </ui-form>

                        <div v-if="execResult" class="exec-result mt-5" :class="execResult.status === 'ok' ? 'result-ok' : 'result-err'">
                          <div class="result-header">
                            <icon-check-circle v-if="execResult.status === 'ok'" class="result-icon result-icon--ok" />
                            <icon-close-circle v-else class="result-icon result-icon--err" />
                            <span class="result-title">
                              {{ execResult.status === "ok" ? "指令下发并执行成功" : `执行异常: ${execResult.error_code}` }}
                            </span>
                          </div>
                          <p v-if="execResult.error_message" class="result-error-msg">{{ execResult.error_message }}</p>
                          <pre v-if="execResult.result" class="result-json premium-json">{{ JSON.stringify(execResult.result, null, 2) }}</pre>
                        </div>
                      </div>
                    </ui-card>
                  </template>
                </div>
              </div>
            </template>
          </div>
        </ui-tab-pane>

      </ui-tabs>
    </div>
  </div>
</template>

