<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { Message } from "@/lib/feedback";
import { useCreateAccountType } from "@/composables/useAccountTypes";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const router = useRouter();
const create = useCreateAccountType();

const key = ref("");
const name = ref("");
const category = ref("generic");

async function handleCreate() {
  try {
    await create.execute({
      key: key.value.trim(),
      name: name.value.trim(),
      category: category.value,
      schema: {},
      capabilities: {},
    });
    Message.success("账号类型已创建");
    router.push(to.accountTypes.list());
  } catch (e) {
    Message.error(e instanceof Error ? e.message : "创建失败");
  }
}
</script>

<template>
  <div class="page-container form-page">
    <PageHeader
      title="新建账号类型"
      subtitle="定义一个新的账号类别。"
      icon-bg="linear-gradient(135deg, rgba(2,132,199,0.12), rgba(14,165,233,0.12))"
      icon-color="#0284c7"
      :back-to="to.accountTypes.list()"
      back-label="返回账号类型"
    >
      <template #icon><icon-layers /></template>
    </PageHeader>

    <ui-card>
      <ui-form layout="vertical">
        <ui-form-item label="键名">
          <ui-input v-model="key" placeholder="github" />
        </ui-form-item>
        <ui-form-item label="显示名称">
          <ui-input v-model="name" placeholder="GitHub" />
        </ui-form-item>
        <ui-form-item label="分类">
          <ui-select v-model="category">
            <ui-option value="generic">generic</ui-option>
            <ui-option value="email">email</ui-option>
            <ui-option value="system">system</ui-option>
          </ui-select>
        </ui-form-item>
        <div class="form-actions">
          <ui-button @click="router.push(to.accountTypes.list())">取消</ui-button>
          <ui-button
            type="primary"
            :disabled="!key.trim() || !name.trim()"
            :loading="create.loading.value"
            @click="handleCreate"
          >
            {{ create.loading.value ? "创建中…" : "创建账号类型" }}
          </ui-button>
        </div>
      </ui-form>
    </ui-card>
  </div>
</template>
