<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { PageHeader } from "@/components/index";

const route = useRoute();
const code = computed(() => (route.query.code as string | undefined) ?? "");
</script>

<template>
  <div class="min-h-screen">
    <div class="page-shell oauth-page">
      <PageHeader
        title="OAuth 回调"
        subtitle="把下面的 code 复制回邮箱配置页面，完成 Token Exchange。"
        icon-bg="linear-gradient(135deg, rgba(59,130,246,0.12), rgba(37,99,235,0.12))"
        icon-color="var(--status-info)"
      >
        <template #icon><icon-lock /></template>
      </PageHeader>

      <ui-card title="Authorization Code">
        <ui-textarea
          :model-value="code || 'No code found in query string.'"
          :auto-size="{ minRows: 6 }"
          readonly
          class="font-mono text-sm"
        />
      </ui-card>
    </div>
  </div>
</template>
