<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { useAccountTypes } from "@/composables/useAccountTypes";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const route = useRoute();
const typeKey = route.params.id as string;

const { data: accountTypes, loading } = useAccountTypes();
const accountType = computed(() => accountTypes.value.find((t) => t.key === typeKey));
</script>

<template>
  <div class="page-container form-page">
    <PageHeader
      title="修改账号类型"
      :subtitle="accountType ? `正在编辑 ${accountType.name}` : ''"
      icon-bg="linear-gradient(135deg, rgba(2,132,199,0.12), rgba(14,165,233,0.12))"
      icon-color="#0284c7"
      :back-to="to.accountTypes.list()"
      back-label="返回账号类型"
    >
      <template #icon><icon-layers /></template>
    </PageHeader>

    <div v-if="loading" class="center-empty"><ui-spin :size="36" /></div>
    <div v-else-if="!accountType" class="center-empty">
      <p class="muted-copy">未找到该账号类型。</p>
    </div>
    <ui-card v-else>
      <div class="inline-info-notice">
        <icon-info-circle class="inline-info-notice-icon" />
        <p class="inline-info-notice-body">
          暂不支持通过 UI 修改账号类型配置。请删除后重新创建，或通过 API 修改。
        </p>
      </div>
    </ui-card>
  </div>
</template>
