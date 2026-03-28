<script setup lang="ts">
import type { Account, PluginUIFormField } from "@/types";
import {
  filterPluginFieldAccounts,
  formatPluginAccountOption,
  isPluginSecretField,
  pluginFieldLabel,
} from "@/utils/pluginUI";

interface Props {
  fields: PluginUIFormField[];
  modelValue: Record<string, unknown>;
  accounts?: Account[];
  defaultAccountTypeKey?: string;
  errors?: Record<string, string>;
}

const props = withDefaults(defineProps<Props>(), {
  accounts: () => [],
  defaultAccountTypeKey: "",
  errors: () => ({}),
});

const emit = defineEmits<{
  (e: "update:modelValue", value: Record<string, unknown>): void;
}>();

function updateField(name: string, value: unknown) {
  emit("update:modelValue", {
    ...props.modelValue,
    [name]: value,
  });
}

function fieldType(field: PluginUIFormField): string {
  return String(field.type ?? "string").trim().toLowerCase() || "string";
}

function fieldValue(name: string): unknown {
  return props.modelValue?.[name];
}

function fieldAccounts(field: PluginUIFormField): Account[] {
  return filterPluginFieldAccounts(props.accounts, field, props.defaultAccountTypeKey);
}

function fieldPlaceholder(field: PluginUIFormField): string {
  if (field.placeholder) {
    return field.placeholder;
  }

  const type = fieldType(field);
  if (type === "account" || type === "account_ref") {
    return fieldAccounts(field).length ? "选择账号" : "暂无可选账号";
  }
  if (type === "json") {
    return "{}";
  }
  if (type === "textarea") {
    return field.required ? "请输入内容" : "可选";
  }
  return field.required ? "必填" : "可选";
}

function textareaAutoSize(field: PluginUIFormField) {
  const minRows = typeof field.rows === "number" && field.rows > 0 ? field.rows : 4;
  return {
    minRows,
    maxRows: Math.max(minRows, minRows + 4),
  };
}
</script>

<template>
  <ui-form
    layout="vertical"
    class="flex flex-col gap-3"
    :model="modelValue"
  >
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
      <ui-form-item
        v-for="field in fields"
        :key="field.name"
        class="mb-0"
        :validate-status="errors[field.name] ? 'error' : undefined"
        :help="errors[field.name]"
      >
        <template #label>
          <span class="flex flex-wrap items-center gap-2">
            <span>{{ pluginFieldLabel(field) }}</span>
            <ui-tag v-if="field.required" size="small" color="red">必填</ui-tag>
          </span>
        </template>

        <div class="flex flex-col gap-1.5">
          <ui-select
            v-if="field.choices?.length"
            :model-value="String(fieldValue(field.name) ?? '')"
            allow-clear
            :placeholder="fieldPlaceholder(field)"
            class="w-full"
            :popup-container="'body'"
            @update:model-value="updateField(field.name, $event)"
          >
            <ui-option
              v-for="choice in field.choices"
              :key="choice"
              :value="choice"
            >
              {{ choice }}
            </ui-option>
          </ui-select>

          <ui-select
            v-else-if="fieldType(field) === 'account' || fieldType(field) === 'account_ref'"
            :model-value="String(fieldValue(field.name) ?? '')"
            allow-clear
            :placeholder="fieldPlaceholder(field)"
            class="w-full"
            :popup-container="'body'"
            @update:model-value="updateField(field.name, $event)"
          >
            <ui-option
              v-for="account in fieldAccounts(field)"
              :key="account.id"
              :value="String(account.id)"
            >
              {{ formatPluginAccountOption(account) }}
            </ui-option>
          </ui-select>

          <ui-switch
            v-else-if="fieldType(field) === 'boolean' || fieldType(field) === 'switch'"
            :model-value="Boolean(fieldValue(field.name))"
            class="self-start"
            @update:model-value="updateField(field.name, $event)"
          />

          <ui-input-number
            v-else-if="fieldType(field) === 'number' || fieldType(field) === 'integer'"
            :model-value="fieldValue(field.name) as number | undefined"
            :placeholder="fieldPlaceholder(field)"
            :min="field.min"
            :max="field.max"
            :step="field.step"
            class="w-full"
            @update:model-value="updateField(field.name, $event)"
          />

          <ui-textarea
            v-else-if="fieldType(field) === 'textarea' || fieldType(field) === 'json'"
            :model-value="String(fieldValue(field.name) ?? '')"
            :auto-size="textareaAutoSize(field)"
            :placeholder="fieldPlaceholder(field)"
            class="font-mono"
            @update:model-value="updateField(field.name, $event)"
          />

          <ui-input
            v-else-if="isPluginSecretField(field)"
            :model-value="String(fieldValue(field.name) ?? '')"
            type="password"
            allow-clear
            class="w-full"
            :placeholder="fieldPlaceholder(field)"
            @update:model-value="updateField(field.name, $event)"
          />

          <ui-input
            v-else
            :model-value="String(fieldValue(field.name) ?? '')"
            allow-clear
            class="w-full"
            :placeholder="fieldPlaceholder(field)"
            @update:model-value="updateField(field.name, $event)"
          />

          <p
            v-if="field.description"
            class="m-0 text-xs leading-5 text-slate-500"
          >
            {{ field.description }}
          </p>
        </div>
      </ui-form-item>
    </div>
  </ui-form>
</template>
