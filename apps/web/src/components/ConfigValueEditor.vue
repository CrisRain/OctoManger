<script setup lang="ts">
defineOptions({
  name: "ConfigValueEditor",
});

import { computed } from "vue";
import { IconDelete, IconPlus } from "@/lib/icons";
import {
  CONFIG_VALUE_TYPE_OPTIONS,
  cloneConfigValue,
  configValueTypeOf,
  createConfigValue,
  createUniqueObjectKey,
  isConfigObject,
  type ConfigValue,
  type ConfigValueType,
} from "@/utils/systemConfigEditor";

interface Props {
  modelValue: ConfigValue;
  disabled?: boolean;
  root?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  root: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: ConfigValue];
}>();

const currentType = computed<ConfigValueType>(() => configValueTypeOf(props.modelValue));
const objectEntries = computed(() =>
  isConfigObject(props.modelValue) ? Object.entries(props.modelValue) : [],
);
const arrayEntries = computed(() =>
  Array.isArray(props.modelValue) ? props.modelValue : [],
);

function updateValue(next: ConfigValue) {
  emit("update:modelValue", next);
}

function changeType(next: unknown) {
  const value = String(next) as ConfigValueType;
  updateValue(createConfigValue(value));
}

function updateString(value: string) {
  updateValue(value);
}

function updateNumber(value: number | undefined) {
  updateValue(typeof value === "number" ? value : 0);
}

function updateBoolean(value: boolean) {
  updateValue(value);
}

function renameObjectField(currentKey: string, nextKey: string) {
  if (!isConfigObject(props.modelValue)) {
    return;
  }

  const trimmed = nextKey.trim();
  if (!trimmed || trimmed === currentKey || trimmed in props.modelValue) {
    return;
  }

  const nextValue: Record<string, ConfigValue> = {};
  for (const [key, value] of Object.entries(props.modelValue)) {
    if (key === currentKey) {
      nextValue[trimmed] = cloneConfigValue(value);
      continue;
    }
    nextValue[key] = cloneConfigValue(value);
  }
  updateValue(nextValue);
}

function updateObjectField(key: string, value: ConfigValue) {
  if (!isConfigObject(props.modelValue)) {
    return;
  }

  updateValue({
    ...props.modelValue,
    [key]: cloneConfigValue(value),
  });
}

function removeObjectField(key: string) {
  if (!isConfigObject(props.modelValue)) {
    return;
  }

  const nextValue = { ...props.modelValue };
  delete nextValue[key];
  updateValue(nextValue);
}

function addObjectField() {
  const currentValue = isConfigObject(props.modelValue)
    ? { ...props.modelValue }
    : {};
  const nextKey = createUniqueObjectKey(currentValue);
  currentValue[nextKey] = "";
  updateValue(currentValue);
}

function updateArrayItem(index: number, value: ConfigValue) {
  if (!Array.isArray(props.modelValue)) {
    return;
  }

  const nextValue = props.modelValue.map((item) => cloneConfigValue(item));
  nextValue[index] = cloneConfigValue(value);
  updateValue(nextValue);
}

function removeArrayItem(index: number) {
  if (!Array.isArray(props.modelValue)) {
    return;
  }

  updateValue(
    props.modelValue
      .filter((_, itemIndex) => itemIndex !== index)
      .map((item) => cloneConfigValue(item)),
  );
}

function addArrayItem() {
  if (!Array.isArray(props.modelValue)) {
    updateValue([""]);
    return;
  }

  updateValue([...props.modelValue.map((item) => cloneConfigValue(item)), ""]);
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="flex items-center justify-between gap-3 max-md:flex-col max-md:items-stretch">
      <span class="text-xs font-semibold tracking-wider text-slate-500">
        值类型
      </span>
      <ui-select
        :model-value="currentType"
        class="w-full [@media(min-width:768px)]:max-w-[10rem]"
        :disabled="disabled"
        @update:modelValue="changeType"
      >
        <ui-option
          v-for="option in CONFIG_VALUE_TYPE_OPTIONS"
          :key="option.value"
          :value="option.value"
        >
          {{ option.label }}
        </ui-option>
      </ui-select>
    </div>

    <ui-textarea
      v-if="currentType === 'string' && root"
      :model-value="String(modelValue)"
      :rows="3"
      placeholder="输入文本"
      :disabled="disabled"
      @update:modelValue="updateString"
    />

    <ui-input
      v-else-if="currentType === 'string'"
      :model-value="String(modelValue)"
      placeholder="输入文本"
      :disabled="disabled"
      @update:modelValue="updateString"
    />

    <ui-input-number
      v-else-if="currentType === 'number'"
      :model-value="typeof modelValue === 'number' ? modelValue : 0"
      :disabled="disabled"
      placeholder="输入数字"
      @update:modelValue="updateNumber"
    />

    <div
      v-else-if="currentType === 'boolean'"
      class="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-4 py-3"
    >
      <span class="text-sm text-slate-600">当前开关值</span>
      <ui-switch
        :model-value="Boolean(modelValue)"
        :disabled="disabled"
        checked-text="开"
        unchecked-text="关"
        @update:modelValue="updateBoolean"
      />
    </div>

    <div
      v-else-if="currentType === 'null'"
      class="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-4 py-4 text-sm leading-6 text-slate-500"
    >
      当前值为 <code>null</code>。如果需要填写内容，请先切换为其他类型。
    </div>

    <div v-else-if="currentType === 'object'" class="flex flex-col gap-3">
      <div
        v-if="objectEntries.length === 0"
        class="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-4 py-4 text-sm text-slate-500"
      >
        当前对象还没有字段。
      </div>

      <div
        v-for="[entryKey, entryValue] in objectEntries"
        :key="entryKey"
        class="rounded-xl border border-slate-200 bg-white/80 p-3 shadow-sm"
      >
        <div class="flex items-center gap-3 max-md:flex-col max-md:items-stretch">
          <ui-input
            :model-value="entryKey"
            placeholder="字段名"
            class="w-full [@media(min-width:768px)]:max-w-[16rem]"
            :disabled="disabled"
            @change="renameObjectField(entryKey, $event)"
          />
          <ui-button
            status="danger"
            class="self-start max-md:w-full max-md:justify-center"
            :disabled="disabled"
            @click="removeObjectField(entryKey)"
          >
            <template #icon><icon-delete /></template>
            删除字段
          </ui-button>
        </div>

        <div class="mt-3">
          <ConfigValueEditor
            :model-value="entryValue"
            :disabled="disabled"
            @update:modelValue="updateObjectField(entryKey, $event)"
          />
        </div>
      </div>

      <ui-button
        type="outline"
        class="self-start max-md:w-full max-md:justify-center"
        :disabled="disabled"
        @click="addObjectField"
      >
        <template #icon><icon-plus /></template>
        添加字段
      </ui-button>
    </div>

    <div v-else class="flex flex-col gap-3">
      <div
        v-if="arrayEntries.length === 0"
        class="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-4 py-4 text-sm text-slate-500"
      >
        当前数组还没有项目。
      </div>

      <div
        v-for="(entryValue, index) in arrayEntries"
        :key="index"
        class="rounded-xl border border-slate-200 bg-white/80 p-3 shadow-sm"
      >
        <div class="flex items-center justify-between gap-3">
          <span class="text-xs font-semibold tracking-wider text-slate-500">
            项目 {{ index + 1 }}
          </span>
          <ui-button
            status="danger"
            class="self-start"
            :disabled="disabled"
            @click="removeArrayItem(index)"
          >
            <template #icon><icon-delete /></template>
            删除
          </ui-button>
        </div>

        <div class="mt-3">
          <ConfigValueEditor
            :model-value="entryValue"
            :disabled="disabled"
            @update:modelValue="updateArrayItem(index, $event)"
          />
        </div>
      </div>

      <ui-button
        type="outline"
        class="self-start max-md:w-full max-md:justify-center"
        :disabled="disabled"
        @click="addArrayItem"
      >
        <template #icon><icon-plus /></template>
        添加项目
      </ui-button>
    </div>
  </div>
</template>
