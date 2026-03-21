<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { useTriggers } from "@/composables/useTriggers";
import { PageHeader } from "@/components/index";
import { to } from "@/router/registry";

const route = useRoute();
const triggerId = Number(route.params.id);

const { data: triggers, loading } = useTriggers();
const trigger = computed(() => triggers.value.find((t) => t.id === triggerId));
</script>

<template>
  <div class="page-shell">
    <PageHeader
      title="编辑触发器"
      :subtitle="trigger ? `正在编辑 ${trigger.name}` : ''"
      icon-bg="linear-gradient(135deg, rgba(234,179,8,0.12), rgba(202,138,4,0.12))"
      icon-color="var(--icon-yellow)"
      :back-to="to.triggers.list()"
      back-label="返回触发器列表"
    >
      <template #icon><icon-thunderbolt /></template>
    </PageHeader>

    <div v-if="loading" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/[56%] shadow-sm"><ui-spin size="2.25em" /></div>
    <div v-else-if="!trigger" class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed px-6 py-16 text-center border-slate-200 bg-white/[56%] shadow-sm"><p class="text-sm leading-6 text-slate-500">未找到该触发器。</p></div>
    <ui-card v-else>
      <div class="flex items-start gap-3 rounded-xl border border-sky-200 bg-sky-50/70 p-4">
        <icon-info-circle class="mt-0.5 h-4 w-4 flex-shrink-0 text-sky-600" />
        <p class="text-sm leading-6 text-slate-700">暂不支持通过 UI 编辑触发器配置。请删除后重新创建，或通过 API 更新。</p>
      </div>
    </ui-card>
  </div>
</template>
