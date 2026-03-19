<script setup lang="ts">
import { computed, onMounted } from "vue";
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

const { data: statusData, refresh: _refreshStatus } = useSystemStatus();
const message = useMessage();
const { withErrorHandler } = useErrorHandler();

const authStore = useAuthStore();
const { adminKey } = storeToRefs(authStore);
const adminKeySet = computed(() => Boolean(adminKey.value?.trim()));

const {
  knownConfigs,
  configs,
  configEdits,
  configSaving,
  loadConfigs,
  saveConfig,
} = useSystemConfigs();

onMounted(() => { void loadConfigs(); });

function persistAdminKey(value: string) {
  authStore.setKey(value);
}

async function handleSaveConfig(configKey: string) {
  await withErrorHandler(
    async () => {
      await saveConfig(configKey);
      message.success("已保存");
    },
    { action: "保存配置" }
  );
}

// 刷新状态
async function refreshStatus() {
  await withErrorHandler(
    async () => {
      await _refreshStatus();
      message.success("状态已刷新");
    },
    { action: "刷新状态" }
  );
}
</script>

<template>
  <div class="page-container settings-page">
    <PageHeader
      title="设置"
      subtitle="查看系统运行状态，管理配置、认证与访问安全"
      icon-bg="linear-gradient(135deg, rgba(20,184,166,0.16), rgba(45,212,191,0.16))"
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

    <!-- 顶部卡片网格 -->
    <div class="top-grid">
      <!-- 系统状态 -->
      <ui-card class="status-card">
        <template #title>
          <div class="card-title-row">
            <icon-cloud class="card-title-icon card-title-icon--blue" />
            <span>系统状态</span>
          </div>
        </template>

        <div class="info-rows">
          <div class="info-row">
            <span class="detail-label">数据库连接</span>
            <div class="status-value">
              <span
                class="status-dot"
                :class="statusData?.database_ok ? 'status-dot--ok' : 'status-dot--error'"
              ></span>
              <span
                class="status-text"
                :class="statusData?.database_ok ? 'status-text--ok' : 'status-text--error'"
              >
                {{ statusData?.database_ok ? "正常" : "异常" }}
              </span>
            </div>
          </div>
          <div class="info-row">
            <span class="detail-label">已加载插件数</span>
            <span class="detail-value">{{ statusData?.plugin_count ?? "—" }}</span>
          </div>
          <div v-if="statusData?.now" class="info-row">
            <span class="detail-label">服务器时间</span>
            <span class="detail-value mono-text">{{ new Date(statusData.now).toLocaleString("zh-CN") }}</span>
          </div>
        </div>
      </ui-card>

      <!-- 管理员密钥设置 -->
      <ui-card class="status-card">
        <template #title>
          <div class="card-title-row">
            <icon-lock class="card-title-icon card-title-icon--purple" />
            <span>管理员密钥设置</span>
          </div>
        </template>

        <div class="info-rows">
          <div class="info-row info-row--no-border">
            <span class="detail-label">浏览器中的 Admin Key</span>
            <div class="status-value">
              <span
                class="status-dot"
                :class="adminKeySet ? 'status-dot--ok' : 'status-dot--error'"
              ></span>
              <span
                class="status-text"
                :class="adminKeySet ? 'status-text--ok' : 'status-text--error'"
              >
                {{ adminKeySet ? "已配置" : "未配置" }}
              </span>
            </div>
          </div>
        </div>

        <div class="admin-input-block">
          <ui-input
            :model-value="adminKey"
            placeholder="输入 Admin Key..."
            allow-clear
            type="password"
            class="admin-input"
            @input="persistAdminKey($event)"
          />
          <p class="admin-input-caption">
            保存于当前浏览器，填写后会自动在所有请求里附带 <code>X-Admin-Key</code>。
          </p>
        </div>

        <div class="auth-hint">
          <p>您的 Admin Key 已保存在本地浏览器中。</p>
          <p>该值由服务器端的环境变量 <code>ADMIN_KEY</code> 控制。</p>
        </div>
      </ui-card>
    </div>

    <div class="security-grid">
      <ui-card class="security-card">
        <template #title>
          <div class="card-header-with-icon">
            <div class="card-icon-box dark"><icon-safe /></div>
            API 密钥管理
          </div>
        </template>

        <div class="info-section">
          <p class="info-text">
            当前版本使用单个 <strong>Admin Key</strong> 作为 API 访问凭证，服务端通过
            <code class="key-badge highlight-key">ADMIN_KEY</code> 进行配置。
          </p>
          <p class="info-text">
            目前不提供多 API Key 的签发、轮换或撤销列表页面，统一通过管理员密钥进行认证。
          </p>

          <div class="terminal-block">
            <div class="terminal-header">
              <div class="mac-dots">
                <span class="dot red"></span><span class="dot yellow"></span><span class="dot green"></span>
              </div>
              <span class="terminal-title">HTTP Request Header Structure</span>
            </div>
            <div class="terminal-body">
              <code><span class="http-key">X-Admin-Key</span>: <span class="http-val">&lt;your-admin-key&gt;</span></code>
            </div>
          </div>

          <p class="info-text tip">
            <icon-info-circle class="inline-info-icon" />
            <strong>提示：</strong>在本页填写管理员密钥后，浏览器会自动在请求中携带认证信息。
          </p>
        </div>
      </ui-card>

      <ui-card class="security-card">
        <template #title>
          <div class="card-header-with-icon">
            <div class="card-icon-box dark"><icon-lock /></div>
            SSL 安全证书管理
          </div>
        </template>

        <div class="notice notice--info">
          <icon-info-circle class="notice-icon" />
          <p class="notice-title">通过服务器配置管理 TLS</p>
          <p class="notice-body">
            当前版本的 SSL 证书管理通过服务器配置完成，不提供 Web 界面操作。
            请在服务器配置文件或反向代理（如 Nginx、Caddy）中配置 TLS 证书。
          </p>
        </div>
      </ui-card>
    </div>

    <!-- 系统配置 -->
    <ui-card class="config-card">
      <template #title>
        <div class="card-title-row">
          <icon-tool class="card-title-icon card-title-icon--gray" />
          <span>系统配置</span>
        </div>
      </template>

      <template #extra>
        <span class="extra-hint">配置值以 JSON 格式存储</span>
      </template>

      <div class="config-list">
        <div v-for="c in knownConfigs" :key="c.key" class="config-item">
          <div class="config-label-row">
            <span class="config-label">{{ c.label }}</span>
            <code class="key-badge">{{ c.key }}</code>
          </div>
          <p class="config-description">{{ c.description }}</p>
          <div class="config-input-row">
            <ui-input
              :model-value="configEdits[c.key] ?? ''"
              placeholder='例如: "OctoManger" 或 30'
              class="config-input"
              @input="configEdits[c.key] = $event"
            />
            <ui-button
              type="primary"
              class="save-btn"
              :disabled="configSaving[c.key] || configEdits[c.key] === configs[c.key]"
              :loading="configSaving[c.key]"
              @click="handleSaveConfig(c.key)"
            >
              <template #icon><icon-check /></template>
              {{ configSaving[c.key] ? "保存中..." : "保存" }}
            </ui-button>
          </div>
        </div>
      </div>
    </ui-card>
  </div>
</template>
