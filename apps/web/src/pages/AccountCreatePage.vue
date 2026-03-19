<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRouter } from "vue-router";
import { Message } from "@/lib/feedback";
import { useAccountTypes } from "@/composables/useAccountTypes";
import { useCreateAccount } from "@/composables/useAccounts";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const router = useRouter();
const { data: accountTypes } = useAccountTypes();
const create = useCreateAccount();

const accountTypeId = ref(0);
const identifier = ref("");
const status = ref("active");
const tags = ref("");
const spec = ref<Record<string, string>>({});

// Find selected account type
const selectedType = computed(() =>
  accountTypes.value.find((t) => t.id === accountTypeId.value) ?? null
);

// Extract spec field definitions from JSON Schema properties
interface SpecField {
  key: string;
  title: string;
  type: string;
  isSecret: boolean;
  defaultValue: string;
  required: boolean;
}

const SECRET_KEYS = new Set(["token", "password", "secret", "access_token", "refresh_token", "client_secret", "api_key"]);

const specFields = computed((): SpecField[] => {
  const schema = selectedType.value?.schema as Record<string, unknown> | undefined;
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

// Reset spec when account type changes, seeding defaults
watch(selectedType, () => {
  const next: Record<string, string> = {};
  for (const f of specFields.value) {
    next[f.key] = f.defaultValue;
  }
  spec.value = next;
});

async function handleCreate() {
  // Build spec — omit empty-string optional fields
  const builtSpec: Record<string, unknown> = {};
  for (const f of specFields.value) {
    const val = spec.value[f.key] ?? "";
    if (val !== "" || f.required) {
      builtSpec[f.key] = f.type === "integer" ? Number(val) : val;
    }
  }

  try {
    await create.execute({
      account_type_id: accountTypeId.value,
      identifier: identifier.value.trim(),
      status: status.value,
      tags: tags.value.split(",").map((t) => t.trim()).filter(Boolean),
      spec: builtSpec,
    });
    Message.success("账号已创建");
    router.push(to.accounts.list());
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "创建失败");
  }
}
</script>

<template>
  <div class="page-container form-page">
    <PageHeader
      title="创建账号"
      subtitle="添加一个新的账号实例。"
      icon-bg="var(--accent-light)"
      icon-color="var(--accent)"
      :back-to="to.accounts.list()"
      back-label="返回账号列表"
    >
      <template #icon><icon-user /></template>
    </PageHeader>

    <ui-card>
      <ui-form layout="vertical">
        <ui-form-item label="账号类型">
          <ui-select v-model="accountTypeId" placeholder="选择账号类型">
            <ui-option :value="0">选择账号类型</ui-option>
            <ui-option v-for="item in accountTypes" :key="item.id" :value="item.id">
              {{ item.name }} ({{ item.key }})
            </ui-option>
          </ui-select>
        </ui-form-item>

        <ui-form-item label="标识符">
          <ui-input v-model="identifier" placeholder="唯一 ID 或用户名" />
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

        <!-- Dynamic spec fields from schema -->
        <template v-if="specFields.length">
          <ui-divider orientation="left" class="section-divider">
            <span class="divider-label">凭据 / Spec</span>
          </ui-divider>

          <ui-form-item
            v-for="field in specFields"
            :key="field.key"
            :label="field.title"
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
              :placeholder="field.required ? '必填' : '可选'"
            />
            <ui-input
              v-else
              v-model="spec[field.key]"
              allow-clear
              :placeholder="field.defaultValue || (field.required ? '必填' : '可选')"
            />
          </ui-form-item>
        </template>

        <div class="form-actions">
          <ui-button @click="router.push(to.accounts.list())">取消</ui-button>
          <ui-button
            type="primary"
            :disabled="!accountTypeId || !identifier.trim()"
            :loading="create.loading.value"
            @click="handleCreate"
          >
            {{ create.loading.value ? "创建中…" : "创建账号" }}
          </ui-button>
        </div>
      </ui-form>
    </ui-card>
  </div>
</template>
