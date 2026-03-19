<script setup lang="ts">
import { ref, computed, watch } from "vue";
import type { FieldConfig } from "./smart-form.types";
import { IconEye, IconEyeInvisible, IconCheck } from "@/lib/icons";

interface Props {
  fields: FieldConfig[];
  modelValue: Record<string, any>;
  loading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
});

const emit = defineEmits<{
  (e: "update:modelValue", value: Record<string, any>): void;
}>();

const formData = ref<Record<string, any>>({});

// 初始化表单数据
watch(
  () => props.modelValue,
  (val) => {
    formData.value = { ...val };
  },
  { immediate: true, deep: true }
);

// 更新父组件
watch(
  formData,
  (val) => {
    emit("update:modelValue", val);
  },
  { deep: true }
);

// 字段可见性
const visiblePasswords = ref<Record<string, boolean>>({});
const tagInput = ref<Record<string, string>>({});

// 生成智能默认值
function generateAutoValue(field: FieldConfig) {
  if (!field.autoSuggest) return;

  switch (field.autoSuggest.type) {
    case "timestamp":
      formData.value[field.name] = new Date().toISOString();
      break;
    case "uuid":
      formData.value[field.name] = crypto.randomUUID();
      break;
    case "random":
      formData.value[field.name] = Math.random().toString(36).substring(2);
      break;
  }
}

// 切换密码可见性
function togglePasswordVisibility(name: string) {
  visiblePasswords.value[name] = !visiblePasswords.value[name];
}

// 添加标签
function addTag(fieldName: string, tag: string) {
  if (!tag.trim()) return;

  if (!formData.value[fieldName]) {
    formData.value[fieldName] = [];
  }

  if (!formData.value[fieldName].includes(tag)) {
    formData.value[fieldName].push(tag);
  }
}

// 移除标签
function removeTag(fieldName: string, index: number) {
  formData.value[fieldName].splice(index, 1);
}

// 验证表单
const errors = ref<Record<string, string>>({});

function validate(): boolean {
  errors.value = {};
  let isValid = true;

  for (const field of props.fields) {
    if (field.required && !formData.value[field.name]) {
      errors.value[field.name] = `${field.label}为必填项`;
      isValid = false;
    }

    if (field.type === "number" && formData.value[field.name] !== undefined) {
      const value = Number(formData.value[field.name]);
      if (field.min !== undefined && value < field.min) {
        errors.value[field.name] = `不能小于${field.min}`;
        isValid = false;
      }
      if (field.max !== undefined && value > field.max) {
        errors.value[field.name] = `不能大于${field.max}`;
        isValid = false;
      }
    }
  }

  return isValid;
}

// 清除验证错误
function clearError(name: string) {
  delete errors.value[name];
}

// 重置表单
function resetForm() {
  for (const field of props.fields) {
    if (field.type === "tags") {
      formData.value[field.name] = Array.isArray(field.defaultValue)
        ? field.defaultValue
        : [];
      continue;
    }
    formData.value[field.name] = field.defaultValue ?? "";
  }
  errors.value = {};
}

// 获取输入类型
function getInputType(field: FieldConfig): string {
  if (field.type === "password") {
    return visiblePasswords.value[field.name] ? "text" : "password";
  }
  return field.type;
}

// 暴露方法
defineExpose({
  validate,
  resetForm,
  getData: () => formData.value,
});
</script>

<template>
  <ui-form :model="formData" layout="vertical" class="smart-form">
    <ui-form-item
      v-for="field in fields"
      :key="field.name"
      :label="field.label"
      :required="field.required"
      :validate-status="errors[field.name] ? 'error' : undefined"
      :help="errors[field.name]"
      class="smart-form-item"
      :class="{ 'smart-form-item--error': errors[field.name] }"
    >
      <div class="field-body">
        <!-- 文本输入 -->
        <template v-if="field.type === 'text'">
          <ui-input
            v-model="formData[field.name]"
            :placeholder="field.placeholder"
            @focus="clearError(field.name)"
          >
            <template v-if="field.autoSuggest" #suffix>
              <ui-button
                type="text"
                size="small"
                @click="generateAutoValue(field)"
              >
                <icon-check />
              </ui-button>
            </template>
          </ui-input>
        </template>

        <!-- 密码输入 -->
        <template v-else-if="field.type === 'password'">
          <ui-input
            v-model="formData[field.name]"
            :type="getInputType(field)"
            :placeholder="field.placeholder"
            @focus="clearError(field.name)"
          >
            <template #suffix>
              <button
                type="button"
                class="password-toggle"
                @click="togglePasswordVisibility(field.name)"
              >
                <icon-eye v-if="!visiblePasswords[field.name]" />
                <icon-eye-invisible v-else />
              </button>
            </template>
          </ui-input>
        </template>

        <!-- 文本域 -->
        <template v-else-if="field.type === 'textarea'">
          <ui-textarea
            v-model="formData[field.name]"
            :placeholder="field.placeholder"
            :rows="field.rows || 4"
            @focus="clearError(field.name)"
          />
        </template>

        <!-- 数字输入 -->
        <template v-else-if="field.type === 'number'">
          <ui-input-number
            v-model="formData[field.name]"
            :placeholder="field.placeholder"
            :min="field.min"
            :max="field.max"
            class="full-width"
            @focus="clearError(field.name)"
          />
        </template>

        <!-- 选择器 -->
        <template v-else-if="field.type === 'select'">
          <ui-select
            v-model="formData[field.name]"
            :placeholder="field.placeholder"
            class="full-width"
            @focus="clearError(field.name)"
          >
            <ui-option
              v-for="option in field.options"
              :key="option.value"
              :value="option.value"
            >
              {{ option.label }}
            </ui-option>
          </ui-select>
        </template>

        <!-- 开关 -->
        <template v-else-if="field.type === 'switch'">
          <ui-switch
            v-model="formData[field.name]"
            @change="clearError(field.name)"
          />
        </template>

        <!-- 标签输入 -->
        <template v-else-if="field.type === 'tags'">
          <div class="tags-input">
            <div class="tags-list">
              <ui-tag
                v-for="(tag, index) in formData[field.name]"
                :key="index"
                closable
                @close="removeTag(field.name, index as number)"
              >
                {{ tag }}
              </ui-tag>
              <ui-input
                v-model="tagInput[field.name]"
                :placeholder="field.placeholder || '输入后按回车添加'"
                size="small"
                class="tags-input-field"
                @keyup.enter="(e: KeyboardEvent) => { addTag(field.name, (e.target as HTMLInputElement).value); tagInput[field.name] = ''; }"
              />
            </div>
          </div>
        </template>

        <!-- 描述 -->
        <div v-if="field.description" class="field-description">{{ field.description }}</div>
      </div>
    </ui-form-item>

    <!-- 插槽：自定义字段 -->
    <slot name="extra-fields" :formData="formData" :errors="errors" />
  </ui-form>
</template>
