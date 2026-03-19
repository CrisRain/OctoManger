<script setup lang="ts">
import { reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { IconEmail, IconPlus } from "@/lib/icons";

import { PageHeader, SmartForm } from "@/components/index";
import { useCreateEmailAccount } from "@/composables/useEmailAccounts";
import { useMessage, useErrorHandler } from "@/composables";
import type { FieldConfig } from "@/components/smart-form.types";
import { to } from "@/router/registry";

const router = useRouter();
const message = useMessage();
const { withErrorHandler } = useErrorHandler();
const create = useCreateEmailAccount();

// 表单引用
const formRef = ref<InstanceType<typeof SmartForm>>();

// 表单数据
const formData = reactive({
  address: "",
  provider: "outlook",
  status: "active",
  // Outlook 配置
  tenant: "common",
  redirect_uri: "http://localhost:5173/oauth/callback",
  mailbox: "Inbox",
});

// 表单字段配置
const formFields: FieldConfig[] = [
  {
    name: "address",
    label: "邮箱地址",
    type: "text",
    placeholder: "robot@example.com",
    required: true,
    description: "完整的邮箱地址",
  },
  {
    name: "provider",
    label: "服务商",
    type: "select",
    required: true,
    options: [
      { label: "Outlook", value: "outlook" },
      { label: "Gmail", value: "gmail" },
      { label: "IMAP", value: "imap" },
    ],
    description: "邮箱服务提供商",
  },
  {
    name: "status",
    label: "状态",
    type: "select",
    defaultValue: "active",
    options: [
      { label: "已激活", value: "active" },
      { label: "待验证", value: "pending" },
      { label: "已停用", value: "inactive" },
    ],
  },
  {
    name: "tenant",
    label: "租户ID (Tenant)",
    type: "text",
    placeholder: "common",
    description: "Outlook 租户ID，通常为 'common' 或组织ID",
  },
  {
    name: "redirect_uri",
    label: "回调地址",
    type: "text",
    placeholder: "http://localhost:5173/oauth/callback",
    description: "OAuth 授权完成后的回调地址",
  },
  {
    name: "mailbox",
    label: "邮箱文件夹",
    type: "text",
    placeholder: "Inbox",
    description: "默认监听的邮箱文件夹",
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

  await withErrorHandler(
    async () => {
      await create.execute({
        address: formData.address.trim(),
        provider: formData.provider,
        status: formData.status,
        config: {
          tenant: formData.tenant,
          redirect_uri: formData.redirect_uri,
          mailbox: formData.mailbox,
          scope: [
            "offline_access",
            "openid",
            "profile",
            "email",
            "https://graph.microsoft.com/Mail.Read",
          ],
        },
      });
      message.success("邮箱账号已创建");
      router.push(to.emailAccounts.list());
    },
    { action: "创建邮箱账号", showSuccess: true }
  );
}

// 取消创建
function handleCancel() {
  router.push(to.emailAccounts.list());
}
</script>

<template>
  <div class="page-container email-account-create-page">
    <PageHeader
      title="添加邮箱账号"
      subtitle="添加一个新的邮箱账号实例"
      icon-bg="linear-gradient(135deg, rgba(234,88,12,0.12), rgba(249,115,22,0.12))"
      icon-color="var(--icon-orange)"
      :back-to="to.emailAccounts.list()"
      back-label="返回邮箱账号"
    >
      <template #icon><icon-email /></template>
    </PageHeader>

    <div class="form-layout">
      <!-- 表单卡片 -->
      <ui-card class="form-card">
        <template #title>
          <div class="card-title-row">
            <icon-email class="card-title-icon" />
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
            <span>配置说明</span>
          </div>
        </template>

        <div class="info-content">
          <h4 class="info-title">支持的邮箱服务商</h4>
          <div class="provider-list">
            <div class="provider-item">
              <div class="provider-icon">O</div>
              <div class="provider-info">
                <div class="provider-name">Outlook</div>
                <div class="provider-desc">Microsoft Outlook / Office 365</div>
              </div>
            </div>
            <div class="provider-item">
              <div class="provider-icon">G</div>
              <div class="provider-info">
                <div class="provider-name">Gmail</div>
                <div class="provider-desc">Google Gmail (即将支持)</div>
              </div>
            </div>
            <div class="provider-item">
              <div class="provider-icon">I</div>
              <div class="provider-info">
                <div class="provider-name">IMAP</div>
                <div class="provider-desc">通用 IMAP 协议 (即将支持)</div>
              </div>
            </div>
          </div>

          <h4 class="info-title">配置步骤</h4>
          <ol class="config-steps">
            <li>填写邮箱地址和选择服务商</li>
            <li>设置初始状态（通常为"待验证"）</li>
            <li>配置 OAuth 回调地址</li>
            <li>保存后进行 OAuth 授权</li>
            <li>授权完成后状态自动更新为"已激活"</li>
          </ol>

          <h4 class="info-title">注意事项</h4>
          <ul class="notes-list">
            <li>回调地址必须与在邮箱服务商处注册的地址一致</li>
            <li>租户ID (Tenant) 通常使用 "common" 即可</li>
            <li>默认邮箱文件夹为 "Inbox"</li>
            <li>需要先在 Azure AD 中注册应用程序</li>
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
        {{ create.loading.value ? "创建中..." : "创建邮箱账号" }}
      </ui-button>
    </div>
  </div>
</template>
