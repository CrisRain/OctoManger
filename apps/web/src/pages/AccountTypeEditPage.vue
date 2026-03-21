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
  <div class="page-shell">
    <PageHeader
      title="编辑账号类型"
      :subtitle="accountType ? `正在编辑 ${accountType.name}` : ''"
      icon-bg="linear-gradient(135deg, rgba(2,132,199,0.12), rgba(14,165,233,0.12))"
      icon-color="#0284c7"
      :back-to="to.accountTypes.list()"
      back-label="返回账号类型列表"
    >
      <template #icon><icon-layers /></template>
    </PageHeader>

    <div v-if="loading" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/[56%] shadow-sm"><ui-spin size="2.25em" /></div>
    <div v-else-if="!accountType" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/[56%] shadow-sm">
      <p class="text-sm leading-6 text-slate-500">未找到该账号类型。</p>
    </div>
    <ui-card v-else>
      <div class="flex items-start gap-3 rounded-xl border border-sky-200 bg-sky-50/70 p-4">
        <icon-info-circle class="mt-0.5 h-4 w-4 flex-shrink-0 text-sky-600" />
        <p class="text-sm leading-6 text-slate-700">
          暂不支持通过 UI 编辑账号类型配置。请删除后重新创建，或通过 API 更新。
        </p>
      </div>
    </ui-card>
  </div>
</template>
