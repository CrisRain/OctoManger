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
    description: "用于标识此 Agent 的名称",
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
    { action: "创建 Agent", showSuccess: true }
  );
}

// 取消创建
function handleCancel() {
  router.push(to.agents.list());
}
</script>

<template>
  <div class="page-shell agent-create-page">
    <PageHeader
      title="创建 Agent"
      subtitle="创建一个持续运行的 Agent"
      icon-bg="linear-gradient(135deg, rgba(10,132,255,0.12), rgba(10,132,255,0.06))"
      icon-color="var(--icon-purple)"
      :back-to="to.agents.list()"
      back-label="返回 Agent 列表"
    >
      <template #icon><icon-robot /></template>
    </PageHeader>

    <div class="grid grid-cols-1 items-start gap-6 lg:grid-cols-[minmax(0,_1.45fr)_minmax(16em,_0.85fr)]">
      <!-- 表单卡片 -->
      <ui-card class="min-w-0">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-robot class="h-5 w-5 text-[var(--accent)]" />
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
      <ui-card class="min-w-0 lg:sticky lg:top-[var(--space-6)]">
        <template #title>
          <div class="flex items-center gap-2">
            <icon-info-circle class="h-5 w-5 text-[var(--accent)]" />
            <span>关于 Agent</span>
          </div>
        </template>

        <div class="flex flex-col gap-4">
          <h4 class="text-sm font-semibold text-slate-900">什么是 Agent？</h4>
          <p class="text-sm leading-6 text-slate-500">
            Agent 是持续运行的插件进程，可以持续执行特定任务，例如监控数据变化、定期同步信息等。
          </p>

          <h4 class="text-sm font-semibold text-slate-900">任务生命周期</h4>
          <div class="flex flex-col gap-3">
            <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="mt-1.5 h-2.5 w-2.5 flex-shrink-0 rounded-full bg-slate-300"></span>
              <div class="flex flex-col gap-0.5">
                <div class="text-sm font-semibold text-slate-900">已停止 (stopped)</div>
                <div class="text-xs leading-5 text-slate-500">Agent 创建后的初始状态</div>
              </div>
            </div>
            <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="mt-1.5 h-2.5 w-2.5 flex-shrink-0 rounded-full bg-emerald-500"></span>
              <div class="flex flex-col gap-0.5">
                <div class="text-sm font-semibold text-slate-900">运行中 (running)</div>
                <div class="text-xs leading-5 text-slate-500">Agent 正在运行，可持续接收事件</div>
              </div>
            </div>
            <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="mt-1.5 h-2.5 w-2.5 flex-shrink-0 rounded-full bg-sky-500"></span>
              <div class="flex flex-col gap-0.5">
                <div class="text-sm font-semibold text-slate-900">停止中 (stopping)</div>
                <div class="text-xs leading-5 text-slate-500">正在优雅停止 Agent</div>
              </div>
            </div>
            <div class="flex items-start gap-3 rounded-xl border p-4 border-slate-200 bg-slate-50 shadow-sm">
              <span class="mt-1.5 h-2.5 w-2.5 flex-shrink-0 rounded-full bg-red-500"></span>
              <div class="flex flex-col gap-0.5">
                <div class="text-sm font-semibold text-slate-900">错误 (error)</div>
                <div class="text-xs leading-5 text-slate-500">Agent 运行出错，需要检查配置</div>
              </div>
            </div>
          </div>

          <h4 class="text-sm font-semibold text-slate-900">使用场景</h4>
          <ul class="pl-5 text-sm leading-7 text-slate-600">
            <li>持续监控 GitHub 仓库变化</li>
            <li>定期检查邮箱新邮件</li>
            <li>实时同步服务器状态</li>
            <li>定时数据采集和处理</li>
          </ul>
        </div>
      </ui-card>
    </div>

    <!-- 底部操作栏 -->
    <div class="flex items-center justify-end gap-3 rounded-xl border px-5 py-4 sticky bottom-[var(--space-4)] z-10 border-slate-200 bg-slate-50 shadow-sm backdrop-blur-xl backdrop-saturate-150 max-md:flex-col max-md:items-stretch max-md:bottom-[var(--space-3)]">
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
        {{ create.loading.value ? "创建中…" : "创建 Agent" }}
      </ui-button>
    </div>
  </div>
</template>
