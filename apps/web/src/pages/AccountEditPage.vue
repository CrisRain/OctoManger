<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Message } from "@/lib/feedback";
import { useAccounts, usePatchAccount } from "@/composables/useAccounts";
import { useAccountTypes } from "@/composables/useAccountTypes";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const route = useRoute();
const router = useRouter();
const accountId = Number(route.params.id);

const { data: accounts, loading, refresh } = useAccounts();
const { data: accountTypes } = useAccountTypes();
const patch = usePatchAccount();

const account = computed(() => accounts.value.find((a) => a.id === accountId) ?? null);

// Find matching account type for spec fields
const accountType = computed(() =>
  accountTypes.value.find((t) => t.key === account.value?.account_type_key) ?? null
);

// ── Form state ──────────────────────────────────────────────────────────────
const status = ref("active");
const tags = ref("");
const spec = ref<Record<string, string>>({});

const SECRET_KEYS = new Set(["token", "password", "secret", "access_token", "refresh_token", "client_secret", "api_key"]);

interface SpecField {
  key: string;
  title: string;
  type: string;
  isSecret: boolean;
  defaultValue: string;
  required: boolean;
}

const specFields = computed((): SpecField[] => {
  const schema = accountType.value?.schema as Record<string, unknown> | undefined;
  if (!schema) return [];
  const props = schema.properties as Record<string, Record<string, unknown>> | undefined;
  if (!props) return [];
  const required = new Set((schema.required as string[]) ?? []);
  return Object.entries(props).map(([key, def]) => ({
    key,
    title: (def.title as string) || key,
    type: (def.type as string) || "string",
    isSecret: SECRET_KEYS.has(key.toLowerCase()),
    defaultValue: def.default != null ? String(def.default) : "",
    required: required.has(key),
  }));
});

// Seed form when account loads
watch(account, (acc) => {
  if (!acc) return;
  status.value = acc.status;
  tags.value = (acc.tags ?? []).join(", ");
}, { immediate: true });

watch([account, specFields], ([acc, fields]) => {
  if (!acc || !fields.length) return;
  const next: Record<string, string> = {};
  for (const f of fields) {
    const existing = acc.spec?.[f.key];
    next[f.key] = existing != null ? String(existing) : f.defaultValue;
  }
  spec.value = next;
}, { immediate: true });

async function handleSave() {
  const builtSpec: Record<string, unknown> = {};
  for (const f of specFields.value) {
    const val = spec.value[f.key] ?? "";
    if (val !== "" || f.required) {
      builtSpec[f.key] = f.type === "integer" ? Number(val) : val;
    }
  }
  // If no schema fields, pass existing spec unchanged
  const specPayload = specFields.value.length > 0 ? builtSpec : account.value?.spec ?? {};

  try {
    await patch.execute(accountId, {
      status: status.value,
      tags: tags.value.split(",").map((t) => t.trim()).filter(Boolean),
      spec: specPayload,
    });
    await refresh();
    Message.success("账号已更新");
    router.push(to.accounts.detail(accountId));
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "保存失败");
  }
}
</script>

<template>
  <div class="page-container form-page">
    <PageHeader
      title="修改账号"
      :subtitle="account ? `正在编辑 ${account.identifier}` : ''"
      icon-bg="var(--accent-light)"
      icon-color="var(--accent)"
      :back-to="to.accounts.detail(accountId)"
      back-label="返回账号详情"
    >
      <template #icon><icon-user /></template>
    </PageHeader>

    <div v-if="loading" class="center-empty">
      <ui-spin :size="36" />
    </div>
    <div v-else-if="!account" class="center-empty">
      <p class="muted-copy">未找到该账号。</p>
    </div>

    <ui-card v-else-if="account">
      <ui-form layout="vertical">
        <!-- Read-only info -->
        <ui-form-item label="账号类型">
          <code class="key-badge">{{ account.account_type_key || "—" }}</code>
        </ui-form-item>

        <ui-form-item label="标识符">
          <ui-input :model-value="account.identifier" disabled />
        </ui-form-item>

        <ui-form-item label="状态">
          <ui-select v-model="status">
            <ui-option value="active">已激活</ui-option>
            <ui-option value="pending">待验证</ui-option>
            <ui-option value="inactive">已停用</ui-option>
          </ui-select>
        </ui-form-item>

        <ui-form-item label="标签">
          <ui-input v-model="tags" placeholder="用逗号分隔，如 demo, test" />
        </ui-form-item>

        <!-- Dynamic spec fields -->
        <template v-if="specFields.length">
          <ui-divider orientation="left" class="section-divider">
            <span class="divider-label">凭据 / Spec</span>
          </ui-divider>

          <ui-form-item
            v-for="field in specFields"
            :key="field.key"
          >
            <template #label>
              <span>{{ field.title }}</span>
              <ui-tag v-if="field.required" size="small" color="red" class="required-tag">必填</ui-tag>
              <code class="field-key-hint">{{ field.key }}</code>
            </template>

            <ui-input-number
              v-if="field.type === 'integer'"
              v-model="(spec[field.key] as unknown as number)"
              :placeholder="field.defaultValue || field.key"
              class="input-fill"
            />
            <ui-input
              v-else-if="field.isSecret"
              v-model="spec[field.key]"
              type="password"
              allow-clear
              :placeholder="field.required ? '必填' : '可选（留空保持不变）'"
            />
            <ui-input
              v-else
              v-model="spec[field.key]"
              allow-clear
              :placeholder="field.defaultValue || (field.required ? '必填' : '可选')"
            />
          </ui-form-item>
        </template>

        <!-- Fallback: raw spec JSON if no schema -->
        <template v-else>
          <ui-divider orientation="left" class="section-divider">
            <span class="divider-label">凭据 / Spec（JSON）</span>
          </ui-divider>
          <ui-form-item label="Spec（原始 JSON）">
            <ui-textarea
              :model-value="JSON.stringify(account.spec, null, 2)"
              disabled
              :auto-size="{ minRows: 3, maxRows: 10 }"
              class="mono-field"
            />
            <p class="field-hint">该账号类型未找到 Schema，Spec 不可编辑。</p>
          </ui-form-item>
        </template>

        <div class="form-actions">
          <ui-button @click="router.push(to.accounts.detail(accountId))">取消</ui-button>
          <ui-button
            type="primary"
            :loading="patch.loading.value"
            @click="handleSave"
          >
            {{ patch.loading.value ? "保存中…" : "保存" }}
          </ui-button>
        </div>
      </ui-form>
    </ui-card>
  </div>
</template>
