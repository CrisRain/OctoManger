<script setup lang="ts">
import { ref, reactive } from "vue";
import { useRouter } from "vue-router";
import { IconClockCircle, IconPlus } from "@/lib/icons";

import { PageHeader, SmartForm } from "@/components/index";
import { useCreateJobDefinition } from "@/composables/useJobs";
import { useMessage, useErrorHandler } from "@/composables";
import type { FieldConfig } from "@/components/smart-form.types";
import { to } from "@/router/registry";

const router = useRouter();
const message = useMessage();
const { withErrorHandler } = useErrorHandler();
const create = useCreateJobDefinition();

// 表单引用
const formRef = ref<InstanceType<typeof SmartForm>>();

// 表单数据
const formData = reactive({
  key: "",
  name: "",
  plugin_key: "",
  action: "",
  input: "{}",
  cron_expression: "",
  timezone: "UTC",
  schedule_enabled: false,
});

// 表单字段配置
const formFields: FieldConfig[] = [
  {
    name: "key",
    label: "任务标识符",
    type: "text",
    placeholder: "例如: github-verify",
    required: true,
    description: "用于唯一标识此任务定义，只能包含小写字母、数字和连字符",
  },
  {
    name: "name",
    label: "任务名称",
    type: "text",
    placeholder: "例如: GitHub 账号验证",
    required: true,
    description: "任务的显示名称，用于界面展示",
  },
  {
    name: "plugin_key",
    label: "插件标识符",
    type: "text",
    placeholder: "例如: github",
    required: true,
    description: "要调用的插件标识符",
  },
  {
    name: "action",
    label: "动作名称",
    type: "text",
    placeholder: "例如: verify_profile",
    required: true,
    description: "插件中定义的动作名称",
  },
  {
    name: "input",
    label: "输入参数",
    type: "textarea",
    placeholder: '{"username":"octocat"}',
    description: "JSON 格式的输入参数，将传递给插件动作",
    rows: 4,
  },
  {
    name: "schedule_enabled",
    label: "启用定时调度",
    type: "switch",
    description: "开启后需要设置 Cron 表达式，任务将按计划自动执行",
  },
  {
    name: "cron_expression",
    label: "Cron 表达式",
    type: "text",
    placeholder: "0 * * * *",
    description: "格式：分 时 日 月 周。例如: 0 * * * * 表示每小时执行一次",
  },
  {
    name: "timezone",
    label: "时区",
    type: "select",
    defaultValue: "UTC",
    options: [
      { label: "UTC (协调世界时)", value: "UTC" },
      { label: "Asia/Shanghai (上海)", value: "Asia/Shanghai" },
      { label: "Asia/Tokyo (东京)", value: "Asia/Tokyo" },
      { label: "America/New_York (纽约)", value: "America/New_York" },
      { label: "Europe/London (伦敦)", value: "Europe/London" },
    ],
  },
];

// 预设 Cron 表达式选项
const cronPresets = [
  { label: "每分钟", value: "* * * * *" },
  { label: "每小时", value: "0 * * * *" },
  { label: "每天 0 点", value: "0 0 * * *" },
  { label: "每周一 0 点", value: "0 0 * * 1" },
  { label: "每月 1 号 0 点", value: "0 0 1 * *" },
  { label: "工作日 9 点", value: "0 9 * * 1-5" },
];

// 应用预设 Cron 表达式
function applyCronPreset(preset: string) {
  formData.cron_expression = preset;
}

// 提交创建
async function handleCreate() {
  // 验证表单
  const isValid = formRef.value?.validate();
  if (!isValid) {
    message.error("请检查表单填写是否正确");
    return;
  }

  // 解析输入参数
  let input: Record<string, unknown> = {};
  try {
    if (formData.input.trim()) {
      input = JSON.parse(formData.input) as Record<string, unknown>;
    }
  } catch (e) {
    message.error("输入参数格式错误，请检查 JSON 格式");
    return;
  }

  await withErrorHandler(
    async () => {
      await create.execute({
        key: formData.key.trim(),
        name: formData.name.trim(),
        plugin_key: formData.plugin_key.trim(),
        action: formData.action.trim(),
        input,
        schedule: formData.schedule_enabled && formData.cron_expression.trim()
          ? {
              cron_expression: formData.cron_expression.trim(),
              timezone: formData.timezone,
              enabled: true,
            }
          : undefined,
      });
      message.success("任务定义已创建");
      router.push(to.jobs.list());
    },
    { action: "创建任务", showSuccess: true }
  );
}

// 取消创建
function handleCancel() {
  router.push(to.jobs.list());
}
</script>

<template>
  <div class="page-container job-create-page">
    <PageHeader
      title="创建任务"
      subtitle="定义一个新的自动化任务，支持定时调度和手动触发"
      icon-bg="linear-gradient(135deg, rgba(202,138,4,0.12), rgba(234,179,8,0.12))"
      icon-color="var(--icon-yellow)"
      :back-to="to.jobs.list()"
      back-label="返回任务列表"
    >
      <template #icon><icon-clock-circle /></template>
    </PageHeader>

    <div class="form-layout">
      <!-- 表单卡片 -->
      <ui-card class="form-card">
        <template #title>
          <div class="card-title-row">
            <icon-plus class="card-title-icon" />
            <span>基本信息</span>
          </div>
        </template>

        <SmartForm
          ref="formRef"
          v-model="formData"
          :fields="formFields"
        />

        <!-- Cron 快捷预设 -->
        <div v-if="formData.schedule_enabled" class="cron-presets">
          <div class="presets-label">快捷预设：</div>
          <div class="presets-list">
            <ui-tag
              v-for="preset in cronPresets"
              :key="preset.value"
              class="preset-tag"
              @click="applyCronPreset(preset.value)"
            >
              {{ preset.label }}
            </ui-tag>
          </div>
        </div>
      </ui-card>

      <!-- 说明卡片 -->
      <ui-card class="info-card">
        <template #title>
          <div class="card-title-row">
            <icon-clock-circle class="card-title-icon" />
            <span>Cron 表达式说明</span>
          </div>
        </template>

        <div class="cron-help">
          <div class="cron-format">
            <code class="format-code">* * * * *</code>
            <div class="format-desc">
              <span>│ │ │ │ │</span>
              <span>│ │ │ │ └─ 星期几 (0-6, 0=周日)</span>
              <span>│ │ │ └─── 月份 (1-12)</span>
              <span>│ │ └───── 日期 (1-31)</span>
              <span>│ └─────── 小时 (0-23)</span>
              <span>└───────── 分钟 (0-59)</span>
            </div>
          </div>

          <div class="cron-examples">
            <h4 class="examples-title">常用示例</h4>
            <div class="example-list">
              <div class="example-item">
                <code>0 * * * *</code>
                <span>每小时执行一次</span>
              </div>
              <div class="example-item">
                <code>0 0 * * *</code>
                <span>每天 0 点执行</span>
              </div>
              <div class="example-item">
                <code>0 9 * * 1-5</code>
                <span>工作日上午 9 点执行</span>
              </div>
              <div class="example-item">
                <code>*/30 * * * *</code>
                <span>每 30 分钟执行一次</span>
              </div>
              <div class="example-item">
                <code>0 0 1 * *</code>
                <span>每月 1 号 0 点执行</span>
              </div>
            </div>
          </div>
        </div>
      </ui-card>
    </div>

    <!-- 底部操作栏 -->
    <div class="form-footer">
      <ui-button size="large" @click="handleCancel">
        取消
      </ui-button>
      <ui-button
        type="primary"
        size="large"
        :loading="create.loading.value"
        @click="handleCreate"
      >
        <template #icon><icon-check /></template>
        {{ create.loading.value ? "创建中..." : "创建任务" }}
      </ui-button>
    </div>
  </div>
</template>
