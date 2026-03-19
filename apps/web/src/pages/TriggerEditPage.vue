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
  <div class="page-container form-page">
    <PageHeader
      title="修改触发器"
      :subtitle="trigger ? `正在编辑 ${trigger.name}` : ''"
      icon-bg="linear-gradient(135deg, rgba(234,179,8,0.12), rgba(202,138,4,0.12))"
      icon-color="var(--icon-yellow)"
      :back-to="to.triggers.list()"
      back-label="返回触发器列表"
    >
      <template #icon><icon-thunderbolt /></template>
    </PageHeader>

    <div v-if="loading" class="center-empty"><ui-spin :size="36" /></div>
    <div v-else-if="!trigger" class="center-empty"><p class="muted-copy">未找到该触发器。</p></div>
    <ui-card v-else>
      <div class="inline-info-notice">
        <icon-info-circle class="inline-info-notice-icon" />
        <p class="inline-info-notice-body">暂不支持通过 UI 修改触发器配置。请删除后重新创建，或通过 API 修改。</p>
      </div>
    </ui-card>
  </div>
</template>
