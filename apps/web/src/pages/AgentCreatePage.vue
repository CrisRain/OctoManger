<script setup lang="ts">
import { reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { IconRobot, IconPlus } from "@/lib/icons";

import { PageHeader, SmartForm } from "@/components/index";
import { useCreateAgent } from "@/composables/useAgents";
import { useMessage, useErrorHandler } from "@/composables";
import type { FieldConfig } from "@/components/smart-form.types";
import { to } from "@/router/registry";

const router = useRouter();
const message = useMessage();
const { withErrorHandler } = useErrorHandler();
const create = useCreateAgent();

// 表单引用
const formRef = ref<InstanceType<typeof SmartForm>>();

// 表单数据
const formData = reactive({
  name: "",
  plugin_key: "",
  action: "",
  // 预设输入参数
  username: "",
  // 扩展JSON输入
  extra_input: "{}",
});

// 表单字段配置
const formFields: FieldConfig[] = [
  {
    name: "name",
    label: "Agent 名称",
    type: "text",
    placeholder: "例如: GitHub Watcher",
    required: true,
    description: "用于标识此后台任务的名称",
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
    name: "username",
    label: "用户名参数",
    type: "text",
    placeholder: "例如: octocat",
    description: "将作为 input.username 传递给插件",
  },
  {
    name: "extra_input",
    label: "额外输入参数 (JSON)",
    type: "textarea",
    placeholder: '{"key":"value"}',
    description: "额外的JSON格式输入参数，将与基础参数合并",
    rows: 4,
  },
];

// 提交创建
async function handleSubmit() {
  // 验证表单
  const isValid = formRef.value?.validate();
  if (!isValid) {
    message.error("请检查表单填写是否正确");
    return;
  }

  // 构建输入参数
  const input: Record<string, unknown> = {};

  // 添加用户名（如果有）
  if (formData.username.trim()) {
    input.username = formData.username.trim();
  }

  // 合并额外参数
  try {
    if (formData.extra_input.trim()) {
      const extra = JSON.parse(formData.extra_input) as Record<string, unknown>;
      Object.assign(input, extra);
    }
  } catch (e) {
    message.error("额外输入参数格式错误，请检查 JSON 格式");
    return;
  }

  await withErrorHandler(
    async () => {
      await create.execute({
        name: formData.name.trim(),
        plugin_key: formData.plugin_key.trim(),
        action: formData.action.trim(),
        input,
      });
      message.success("Agent 已创建");
      router.push(to.agents.list());
    },
    { action: "创建Agent", showSuccess: true }
  );
}

// 取消创建
function handleCancel() {
  router.push(to.agents.list());
}
</script>

<template>
  <div class="page-container agent-create-page">
    <PageHeader
      title="创建后台任务"
      subtitle="配置一个新的长期运行的插件任务"
      icon-bg="linear-gradient(135deg, rgba(20,184,166,0.16), rgba(45,212,191,0.16))"
      icon-color="var(--icon-purple)"
      :back-to="to.agents.list()"
      back-label="返回后台任务"
    >
      <template #icon><icon-robot /></template>
    </PageHeader>

    <div class="form-layout">
      <!-- 表单卡片 -->
      <ui-card class="form-card">
        <template #title>
          <div class="card-title-row">
            <icon-robot class="card-title-icon" />
            <span>基本信息</span>
          </div>
        </template>

        <SmartForm
          ref="formRef"
          v-model="formData"
          :fields="formFields"
        />
      </ui-card>

      <!-- 说明卡片 -->
      <ui-card class="info-card">
        <template #title>
          <div class="card-title-row">
            <icon-info-circle class="card-title-icon" />
            <span>关于后台任务</span>
          </div>
        </template>

        <div class="info-content">
          <h4 class="info-title">什么是后台任务 (Agent)?</h4>
          <p class="info-text">
            后台任务是长期运行的插件进程，可以持续执行特定任务，如监控数据变化、定期同步信息等。
          </p>

          <h4 class="info-title">任务生命周期</h4>
          <div class="lifecycle">
            <div class="lifecycle-step">
              <span class="step-dot"></span>
              <div class="step-title">已停止 (stopped)</div>
              <div class="step-desc">任务创建后的初始状态</div>
            </div>
            <div class="lifecycle-step">
              <span class="step-dot step-dot--active"></span>
              <div class="step-title">运行中 (running)</div>
              <div class="step-desc">任务正在执行，可以接收事件</div>
            </div>
            <div class="lifecycle-step">
              <span class="step-dot step-dot--transition"></span>
              <div class="step-title">停止中 (stopping)</div>
              <div class="step-desc">正在优雅地停止任务</div>
            </div>
            <div class="lifecycle-step">
              <span class="step-dot step-dot--error"></span>
              <div class="step-title">错误 (error)</div>
              <div class="step-desc">任务执行出错，需要检查配置</div>
            </div>
          </div>

          <h4 class="info-title">使用场景</h4>
          <ul class="use-cases">
            <li>持续监控 GitHub 仓库变化</li>
            <li>定期检查邮箱新邮件</li>
            <li>实时同步服务器状态</li>
            <li>定时数据采集和处理</li>
          </ul>
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
        @click="handleSubmit"
      >
        <template #icon><icon-check /></template>
        {{ create.loading.value ? "创建中..." : "创建任务" }}
      </ui-button>
    </div>
  </div>
</template>
