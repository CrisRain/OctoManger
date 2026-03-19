<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRoute } from "vue-router";
import { Message } from "@/lib/feedback";
import { usePlugins, useSyncPlugins, usePluginSettings } from "@/composables/usePlugins";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const route = useRoute();
const pluginKey = route.params.id as string;

const { data: plugins, loading, refresh } = usePlugins();
const plugin = computed(() => plugins.value.find((p) => p.manifest.key === pluginKey));

// ── Sync ──────────────────────────────────────────────────────────────────
const sync = useSyncPlugins();

async function handleSync() {
  try {
    const result = await sync.execute();
    await refresh();
    if (result.failed === 0) {
      Message.success(`已同步 ${result.synced} 个账号类型`);
    } else {
      const errMsg = result.errors.join("; ");
      Message.warning(`部分同步失败：${errMsg}`);
    }
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "同步失败");
  }
}

// ── Settings ──────────────────────────────────────────────────────────────
const settings = usePluginSettings(pluginKey);
const settingValues = ref<Record<string, string>>({});

// Load settings once plugin data is available
watch(plugin, async (p) => {
  if (!p) return;
  await settings.load();
  // Seed form from loaded values
  const vals: Record<string, string> = {};
  for (const s of (p.manifest.settings ?? [])) {
    vals[s.key] = settings.data.value[s.key] != null
      ? String(settings.data.value[s.key])
      : "";
  }
  settingValues.value = vals;
}, { immediate: true });

const SECRET_SETTING_KEYS = new Set(["token", "password", "secret", "api_key", "access_token", "refresh_token", "client_secret", "twocaptcha_api_key"]);

function isSecretSetting(key: string, secret: boolean): boolean {
  return secret || SECRET_SETTING_KEYS.has(key.toLowerCase());
}

async function saveSettings() {
  const payload: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(settingValues.value)) {
    if (v !== "") payload[k] = v;
  }
  try {
    await settings.save(payload);
    Message.success("设置已保存");
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "保存失败");
  }
}
</script>

<template>
  <div class="page-container plugin-page">
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
          <code class="key-badge highlight-key">v{{ plugin.manifest.version }}</code>
          &nbsp;·&nbsp;
          <span class="muted-tag">@{{ plugin.manifest.key }}</span>
        </template>
      </template>
    </PageHeader>

    <div v-if="loading" class="center-empty">
      <ui-spin :size="36" />
    </div>
    <div v-else-if="!plugin" class="center-empty">
      <icon-apps class="plugin-empty-icon" />
      <p class="plugin-empty-copy">未找到该插件。</p>
    </div>

    <div v-else class="content-grid">
      <!-- Actions list -->
      <ui-card class="premium-card">
        <template #title>
          <div class="card-header-with-icon">
            <div class="card-icon-box"><icon-thunderbolt /></div>
            可用操作
          </div>
        </template>
        <div class="action-list">
          <div
            v-for="act in plugin.manifest.actions"
            :key="act.key"
            class="action-item"
          >
            <code class="key-badge action-key">{{ act.key }}</code>
            <span class="action-name">{{ act.name }}</span>
            <p v-if="act.description" class="action-desc">{{ act.description }}</p>
          </div>
          <p v-if="!plugin.manifest.actions.length" class="no-actions">
            该插件暂无可用操作。
          </p>
        </div>
      </ui-card>

      <!-- Sidebar info -->
      <div class="side-col">
        <ui-card class="premium-card">
          <template #title>
            <div class="card-header-with-icon">
              <div class="card-icon-box info"><icon-info-circle /></div>
              插件信息
            </div>
          </template>
          <div class="info-rows">
            <div class="info-row">
              <span class="detail-label">健康状态</span>
              <div class="status-cell">
                <span class="status-dot-large" :class="plugin.healthy ? 'online' : 'offline'" />
                <ui-tag :color="plugin.healthy ? 'green' : 'red'" class="status-tag-pill">
                  {{ plugin.healthy ? "healthy" : "degraded" }}
                </ui-tag>
              </div>
            </div>
            <div v-if="plugin.manifest.description" class="info-row align-start">
              <span class="detail-label">描述</span>
              <span class="detail-value text-right description-text">{{ plugin.manifest.description }}</span>
            </div>
            <div class="info-row info-row--column">
              <span class="detail-label m-b-4">权限列表</span>
              <div v-if="plugin.manifest.capabilities.length" class="cap-tags">
                <ui-tag v-for="cap in plugin.manifest.capabilities" :key="cap" size="small" class="cap-pill" color="blue">
                  {{ cap }}
                </ui-tag>
              </div>
              <span v-else class="text-muted italic-tag">无特殊权限</span>
            </div>
          </div>
        </ui-card>

        <ui-card class="premium-card op-card mt-16">
          <template #title>
            <div class="card-header-with-icon">
              <div class="card-icon-box warn"><icon-tool /></div>
              操作
            </div>
          </template>
          <div class="op-rows">
            <div class="op-row">
              <p class="op-label">同步账号类型</p>
              <p class="op-hint">将插件定义的账号类型同步到系统中</p>
              <ui-button
                class="op-btn"
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

        <!-- Settings panel -->
        <ui-card
          v-if="plugin.manifest.settings?.length"
          class="premium-card config-card mt-16"
        >
          <template #title>
            <div class="card-header-with-icon">
              <div class="card-icon-box dark"><icon-settings /></div>
              插件设置
            </div>
          </template>
          <ui-spin :loading="settings.loading.value">
            <ui-form layout="vertical" class="premium-form">
              <ui-form-item
                v-for="s in plugin.manifest.settings"
                :key="s.key"
                class="premium-form-item"
              >
                <template #label>
                  <span class="form-item-label">{{ s.label || s.key }}</span>
                  <ui-tag v-if="s.required" size="small" color="red" class="req-tag">必填</ui-tag>
                </template>
                <div class="field-body">
                  <ui-input
                    v-if="isSecretSetting(s.key, s.secret)"
                    v-model="settingValues[s.key]"
                    type="password"
                    allow-clear
                    class="premium-input"
                    :placeholder="s.required ? '必填' : '可选'"
                  />
                  <ui-input
                    v-else
                    v-model="settingValues[s.key]"
                    allow-clear
                    class="premium-input"
                    :placeholder="s.required ? '必填' : '可选'"
                  />
                  <p v-if="s.description" class="setting-hint">{{ s.description }}</p>
                </div>
              </ui-form-item>
              <ui-button
                type="primary"
                class="save-btn"
                :loading="settings.saving.value"
                @click="saveSettings"
              >
                保存设置
              </ui-button>
            </ui-form>
          </ui-spin>
        </ui-card>
      </div>
    </div>
  </div>
</template>
